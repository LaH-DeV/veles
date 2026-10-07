package fetch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitTimeout = 5 * time.Minute

// git runs one git command in dir (or the current directory) and returns its
// combined output. It never prompts: a repository that wants a password is
// an error, not a hang.
func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{
		"-c", "core.autocrlf=false", "-c", "core.symlinks=false", "-c", "advice.detachedHead=false",
	}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=file:http:https:ssh:git") // never ext:: or fd::
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			return "", fmt.Errorf("git dependencies need git, and it is not on the PATH")
		}
		return text, fmt.Errorf("git %s: %v: %s", args[0], err, text)
	}
	return text, nil
}

// gitCheckout fetches one tag or commit of a repository into a fresh
// directory below tmpRoot and returns it, without `.git`, with the commit it
// is. The files are exactly the committed bytes: `.gitattributes` and
// `core.autocrlf` cannot convert them (info/attributes outranks both), so
// the hash is the same on every platform.
func gitCheckout(tmpRoot, url string, tags []string, commit string) (dir, full string, err error) {
	tmp, err := os.MkdirTemp(tmpRoot, "git-")
	if err != nil {
		return "", "", err
	}
	dir = tmp
	defer func() { // dir is a result and is blanked by a failing return: clean up by the local
		if err != nil {
			removeAll(tmp)
		}
	}()
	if _, err = git(dir, "init", "-q"); err != nil {
		return "", "", err
	}
	if err = os.MkdirAll(filepath.Join(dir, ".git", "info"), 0o755); err != nil {
		return "", "", err
	}
	if err = os.WriteFile(filepath.Join(dir, ".git", "info", "attributes"), []byte("* -text -filter -ident -working-tree-encoding\n"), 0o644); err != nil {
		return "", "", err
	}
	if _, err = git(dir, "remote", "add", "origin", url); err != nil {
		return "", "", err
	}
	if commit == "" {
		var last error
		for _, tag := range tags {
			if _, last = git(dir, "fetch", "-q", "--depth", "1", "origin", "refs/tags/"+tag); last == nil {
				break
			}
		}
		if last != nil {
			return "", "", fmt.Errorf("%s has no tag %s: %v", url, strings.Join(tags, " or "), last)
		}
		_, err = git(dir, "checkout", "-q", "FETCH_HEAD")
	} else {
		// a server may refuse a fetch by hash; fetch everything then
		if _, e := git(dir, "fetch", "-q", "--depth", "1", "origin", commit); e != nil {
			if _, err = git(dir, "fetch", "-q", "origin", "+refs/heads/*:refs/remotes/origin/*", "+refs/tags/*:refs/tags/*"); err != nil {
				return "", "", fmt.Errorf("cannot fetch %s: %v", url, err)
			}
			_, err = git(dir, "checkout", "-q", commit)
		} else {
			_, err = git(dir, "checkout", "-q", "FETCH_HEAD")
		}
	}
	if err != nil {
		return "", "", fmt.Errorf("%s: %v", url, err)
	}
	full, err = git(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	if commit != "" && !strings.HasPrefix(full, strings.ToLower(commit)) {
		err = fmt.Errorf("%s: commit %s resolved to %s", url, commit, full)
		return "", "", err
	}
	if err = removeAll(filepath.Join(dir, ".git")); err != nil {
		return "", "", err
	}
	return dir, full, nil
}
