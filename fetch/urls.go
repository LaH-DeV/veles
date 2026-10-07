package fetch

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// A git address in a manifest is handed to git, so it must not be able to
// say anything but "this repository": no option (a leading '-'), no
// transport helper (`ext::sh -c …` runs a command), no whitespace. A
// manifest of a fetched package is written by someone the project's author
// has not met, so it is held to less: only the network schemes, never a path
// on the reader's disk.

// AllowLocalGit is an environment variable that lets a fetched package's
// manifest name local paths and file:// or http:// repositories, as the
// project's own manifest may. It is for a private mirror on a disk, an
// air-gapped build and the tests; leave it unset otherwise.
const AllowLocalGit = "VELES_ALLOW_LOCAL_GIT"

var scpRE = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[^:\s].*$`)

// checkGitURL says why an address may not be fetched, or returns nil.
// fromPackage is true for an address found in the manifest of a fetched
// package rather than in the project's own.
func checkGitURL(u string, fromPackage bool) error {
	if os.Getenv(AllowLocalGit) != "" {
		fromPackage = false // the reader's own mirror or test setup
	}
	if u == "" {
		return fmt.Errorf("the repository address is empty")
	}
	for _, r := range u {
		if r <= ' ' || r == 0x7f {
			return fmt.Errorf("the repository address %q has a space or control character", u)
		}
	}
	if strings.HasPrefix(u, "-") {
		return fmt.Errorf("the repository address %q starts with '-', which git would read as an option", u)
	}
	if i := strings.Index(u, "://"); i >= 0 {
		scheme, rest := u[:i], u[i+3:]
		if rest == "" {
			return fmt.Errorf("the repository address %q has nothing after %s://", u, scheme)
		}
		switch scheme {
		case "https", "ssh", "git":
			return nil
		case "http", "file":
			if !fromPackage {
				return nil
			}
		}
		if fromPackage {
			return fmt.Errorf("the address %q is in a fetched package's manifest, which may use https, ssh or git repositories only (not %s://)", u, scheme)
		}
		return fmt.Errorf("the repository address %q uses %s://, which is not a repository scheme (https, ssh, git, http, file)", u, scheme)
	}
	if strings.Contains(u, "::") {
		return fmt.Errorf("the repository address %q uses a transport helper (name::address), which is not supported", u)
	}
	if scpRE.MatchString(u) {
		return nil // git@github.com:owner/repo
	}
	if fromPackage {
		return fmt.Errorf("the address %q is a local path, and a fetched package's manifest may not name one; it may use https, ssh or git repositories", u)
	}
	return nil // a path on this machine, in the project's own manifest
}
