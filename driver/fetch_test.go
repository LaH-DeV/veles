package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRelease makes (or updates) a local repository standing in for a remote
// one, commits the files and tags the version.
func gitRelease(t *testing.T, dir, version string, files map[string]string) {
	t.Helper()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		os.MkdirAll(dir, 0o755)
		git("init", "-q")
	}
	for name, text := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", "-A")
	git("commit", "-q", "-m", "release "+version)
	git("tag", "-f", "v"+version)
}

// D138 end to end: an app builds and runs against git dependencies — one
// shared by two packages at the highest version asked for, and a second major
// of it under another name — from a module cache, with veles.sum written.
func TestGitDependenciesBuildAndRun(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	t.Setenv("VELES_HOME", t.TempDir())
	t.Setenv("VELES_ALLOW_LOCAL_GIT", "1") // the tests' repositories are local
	root := t.TempDir()
	util := filepath.Join(root, "remote", "util")
	mid := filepath.Join(root, "remote", "mid")
	url := func(d string) string { return filepath.ToSlash(d) }
	pkg := func(name, version, deps string) string {
		s := fmt.Sprintf("[package]\nname = %q\nversion = %q\n", name, version)
		if deps != "" {
			s += "[dependencies]\n" + deps + "\n"
		}
		return s
	}
	gitRelease(t, util, "1.0.0", map[string]string{"veles.toml": pkg("util", "1.0.0", ""), "lib.vs": "public fun version(): i64 => 100\n"})
	gitRelease(t, util, "1.3.0", map[string]string{"veles.toml": pkg("util", "1.3.0", ""), "lib.vs": "public fun version(): i64 => 130\n"})
	gitRelease(t, util, "2.0.0", map[string]string{"veles.toml": pkg("util", "2.0.0", ""), "lib.vs": "public fun version(): i64 => 200\n"})
	gitRelease(t, mid, "1.0.0", map[string]string{
		"veles.toml": pkg("mid", "1.0.0", fmt.Sprintf("util = { git = %q, version = \"1.3.0\" }", url(util))),
		"lib.vs":     "use util\n\npublic fun fromMid(): i64 => util.version()\n",
	})

	app := filepath.Join(root, "app")
	os.MkdirAll(app, 0o755)
	os.WriteFile(filepath.Join(app, "veles.toml"), []byte(pkg("app", "0.1.0",
		fmt.Sprintf("util = { git = %q, version = \"1.0.0\" }\nmid = { git = %q, version = \"1.0.0\" }\nutil2 = { git = %q, version = \"2.0.0\" }", url(util), url(mid), url(util)))), 0o644)
	os.WriteFile(filepath.Join(app, "main.vs"), []byte("use io\nuse util\nuse mid\nuse util2\n\nfun main() {\n  io.println(\"${util.version()} ${mid.fromMid()} ${util2.version()}\")\n}\n"), 0o644)

	exe := filepath.Join(root, "app.exe")
	if code := Run(Options{Path: app, Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	if got := strings.TrimSpace(runExe(t, exe)); got != "130 130 200" {
		t.Errorf("got %q, want %q (util 1.3.0 for the app and mid, util 2.0.0 apart)", got, "130 130 200")
	}
	sum, err := os.ReadFile(filepath.Join(app, "veles.sum"))
	if err != nil || strings.Count(string(sum), "h1:") != 4 { // util 1.0.0, 1.3.0, 2.0.0 and mid
		t.Errorf("veles.sum (%v):\n%s", err, sum)
	}

	// `veles fetch` is the same resolution on its own, and a second build
	// does not need the repositories at all
	if code := Fetch(app); code != 0 {
		t.Errorf("fetch exit %d", code)
	}
	os.RemoveAll(filepath.Join(root, "remote"))
	if code := Run(Options{Path: app, Mode: "check"}); code != 0 {
		t.Errorf("check from the cache alone: exit %d", code)
	}
}
