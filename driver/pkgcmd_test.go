package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// capture runs f with standard output and error redirected to files.
func capture(t *testing.T, f func() int) (int, string, string) {
	t.Helper()
	dir := t.TempDir()
	out, err := os.Create(filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	errf, _ := os.Create(filepath.Join(dir, "err"))
	so, se := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = out, errf
	code := f()
	os.Stdout, os.Stderr = so, se
	out.Close()
	errf.Close()
	a, _ := os.ReadFile(out.Name())
	b, _ := os.ReadFile(errf.Name())
	return code, string(a), string(b)
}

// D132/D138: add, update, remove, deps and vendor against local git
// repositories, with the manifest edited in place and put back on failure.
func TestDependencyCommands(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("VELES_HOME", t.TempDir())
	t.Setenv("VELES_ALLOW_LOCAL_GIT", "1") // the tests' repositories are local
	root := t.TempDir()
	util := filepath.Join(root, "remote", "util")
	// a repository on disk is named by a file:// URL: a bare path means a
	// directory dependency to `veles add`
	url := "file://" + filepath.ToSlash(util)
	if !strings.HasPrefix(filepath.ToSlash(util), "/") {
		url = "file:///" + filepath.ToSlash(util)
	}
	pkg := func(version string) map[string]string {
		return map[string]string{
			"veles.toml": fmt.Sprintf("[package]\nname = \"util\"\nversion = %q\n", version),
			"lib.vs":     "public fun v(): i64 => 1\n// " + version + "\n",
		}
	}
	gitRelease(t, util, "1.0.0", pkg("1.0.0"))
	gitRelease(t, util, "1.4.0", pkg("1.4.0"))
	gitRelease(t, util, "2.0.0-beta.1", pkg("2.0.0-beta.1"))

	app := filepath.Join(root, "app")
	os.MkdirAll(app, 0o755)
	manifest := filepath.Join(app, "veles.toml")
	original := "# my app\n[package]\nname = \"app\"\n\n[dependencies]\n# nothing yet\n"
	os.WriteFile(manifest, []byte(original), 0o644)
	os.WriteFile(filepath.Join(app, "main.vs"), []byte("fun main() {}\n"), 0o644)
	read := func() string { b, _ := os.ReadFile(manifest); return string(b) }

	// add: the newest stable release (the beta is not picked), the comment kept
	code, out, errs := capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{url}}) })
	if code != 0 || !strings.Contains(out, "added util = "+url+" 1.4.0 to [dependencies]") {
		t.Fatalf("add: exit %d\n%s\n%s", code, out, errs)
	}
	if got, want := read(), original+"util = { git = \""+url+"\", version = \"1.4.0\" }\n"; got != want {
		t.Errorf("manifest:\n%s\nwant:\n%s", got, want)
	}
	if sum, err := os.ReadFile(filepath.Join(app, "veles.sum")); err != nil || !strings.Contains(string(sum), "git "+url+" 1.4.0 h1:") {
		t.Errorf("veles.sum (%v): %s", err, sum)
	}

	// adding it again, or a version that is not there, changes nothing
	before := read()
	code, _, errs = capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{url + "@1.0.0"}}) })
	if code != 1 || !strings.Contains(errs, "already depends on 'util'") || read() != before {
		t.Errorf("add twice: exit %d, %s", code, errs)
	}
	code, _, errs = capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{url + "@9.9.9"}, As: "other"}) })
	if code != 1 || !strings.Contains(errs, "has no tag v9.9.9 or 9.9.9") || !strings.Contains(errs, "veles.toml was not changed") || read() != before {
		t.Errorf("add missing version: exit %d, %s\nmanifest:\n%s", code, errs, read())
	}

	// a second name for 1.0.0, then update: util goes to the newest 1.x, old goes to the
	// newest 1.x as well, and the beta major is not offered
	code, out, errs = capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{url + "@1.0.0"}, As: "old"}) })
	if code != 0 {
		t.Fatalf("add old: %d %s %s", code, out, errs)
	}
	code, out, errs = capture(t, func() int { return Update(PkgOptions{Dir: app, All: true}) })
	if code != 0 || !strings.Contains(out, "old 1.0.0 -> 1.4.0") || !strings.Contains(out, "util 1.4.0 is up to date") || strings.Contains(out, "newer major") {
		t.Errorf("update --all: exit %d\n%s\n%s", code, out, errs)
	}
	if strings.Contains(read(), "1.0.0") {
		t.Errorf("old was not updated:\n%s", read())
	}
	if sum, _ := os.ReadFile(filepath.Join(app, "veles.sum")); strings.Contains(string(sum), "1.0.0") {
		t.Errorf("the unused 1.0.0 line was not pruned:\n%s", sum)
	}

	// a stable 2.0.0 appears: update stays in major 1 and says so; naming the version takes it
	gitRelease(t, util, "2.0.0", pkg("2.0.0"))
	code, out, errs = capture(t, func() int { return Update(PkgOptions{Dir: app, Args: []string{"util"}}) })
	if code != 0 || !strings.Contains(out, "util 1.4.0 is up to date") || !strings.Contains(out, "a newer major, 2.0.0, is available; `veles update util@2.0.0` takes it") {
		t.Errorf("update util: exit %d\n%s\n%s", code, out, errs)
	}
	code, out, _ = capture(t, func() int { return Update(PkgOptions{Dir: app, Args: []string{"util@2.0.0"}}) })
	if code != 0 || !strings.Contains(out, "util 1.4.0 -> 2.0.0") || !strings.Contains(read(), `version = "2.0.0"`) {
		t.Errorf("update util@2.0.0: exit %d\n%s", code, out)
	}
	code, _, errs = capture(t, func() int { return Update(PkgOptions{Dir: app, Args: []string{"nope"}}) })
	if code != 1 || !strings.Contains(errs, "has no dependency 'nope' (it has: old, util)") {
		t.Errorf("update nope: %d %s", code, errs)
	}

	// deps shows the build; --why answers; vendor then builds without the cache
	code, out, errs = capture(t, func() int { return Deps(PkgOptions{Dir: app}) })
	if code != 0 || !strings.Contains(out, "app\n") || !strings.Contains(out, "old 1.4.0") || !strings.Contains(out, "util 2.0.0") {
		t.Errorf("deps: exit %d\n%s\n%s", code, out, errs)
	}
	code, out, _ = capture(t, func() int { return Deps(PkgOptions{Dir: app, Why: "old"}) })
	if code != 0 || !strings.Contains(out, "app\n  -> old 1.4.0") {
		t.Errorf("deps --why: exit %d\n%s", code, out)
	}
	code, out, errs = capture(t, func() int { return Vendor(PkgOptions{Dir: app}) })
	if code != 0 || !strings.Contains(out, "vendored 2 packages") {
		t.Fatalf("vendor: exit %d\n%s\n%s", code, out, errs)
	}
	os.RemoveAll(os.Getenv("VELES_HOME"))
	os.RemoveAll(filepath.Join(root, "remote"))
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 0 {
		t.Errorf("check from vendor/ alone: exit %d\n%s", code, errs)
	}

	// remove: the lines go, and with the last dependency veles.sum goes too
	code, out, errs = capture(t, func() int { return Remove(PkgOptions{Dir: app, Args: []string{"old", "util"}}) })
	if code != 0 || !strings.Contains(out, "removed old, util") || !strings.Contains(out, "pruned 2 unused lines") {
		t.Errorf("remove: exit %d\n%s\n%s", code, out, errs)
	}
	if read() != original {
		t.Errorf("manifest after removing everything:\n%s", read())
	}
	if _, err := os.Stat(filepath.Join(app, "veles.sum")); err == nil {
		t.Error("veles.sum should be gone with the last dependency")
	}
}

func TestAddPathDependencyAndWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "libs", "my-math"), 0o755)
	os.WriteFile(filepath.Join(root, "libs", "my-math", "veles.toml"), []byte("[package]\nname = \"mathlib\"\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "app"), 0o755)
	os.WriteFile(filepath.Join(root, "app", "veles.toml"), []byte("[package]\nname = \"app\"\n"), 0o644)
	os.WriteFile(filepath.Join(root, "veles.toml"), []byte("[workspace]\nmembers = [\"app\", \"libs/my-math\"]\n"), 0o644)

	// at the workspace root there is no package to change
	code, _, errs := capture(t, func() int { return Add(PkgOptions{Dir: root, Args: []string{"./libs/my-math"}}) })
	if code != 1 || !strings.Contains(errs, "workspace root with no [package] of its own; name a member with --dir, e.g. --dir app") {
		t.Errorf("at the root: exit %d, %s", code, errs)
	}
	// in the member: the local name is the package's own, the path as given
	code, out, errs := capture(t, func() int { return Add(PkgOptions{Dir: filepath.Join(root, "app"), Args: []string{"../libs/my-math"}}) })
	if code != 0 || !strings.Contains(out, "added mathlib = ../libs/my-math") {
		t.Fatalf("add path: exit %d\n%s\n%s", code, out, errs)
	}
	got, _ := os.ReadFile(filepath.Join(root, "app", "veles.toml"))
	if string(got) != "[package]\nname = \"app\"\n\n[dependencies]\nmathlib = \"../libs/my-math\"\n" {
		t.Errorf("manifest:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "app", "veles.sum")); err == nil {
		t.Error("a path dependency writes no veles.sum")
	}
	code, out, _ = capture(t, func() int { return Deps(PkgOptions{Dir: filepath.Join(root, "app")}) })
	if code != 0 || !strings.Contains(out, "`- mathlib  path ../libs/my-math") {
		t.Errorf("deps: exit %d\n%s", code, out)
	}
	code, _, errs = capture(t, func() int { return Update(PkgOptions{Dir: filepath.Join(root, "app"), Args: []string{"mathlib"}}) })
	if code != 1 || !strings.Contains(errs, "'mathlib' is a path, which has no version to update") {
		t.Errorf("update a path: exit %d, %s", code, errs)
	}
}

func fileURL(dir string) string {
	p := filepath.ToSlash(dir)
	if strings.HasPrefix(p, "/") {
		return "file://" + p
	}
	return "file:///" + p
}

// D138 capabilities: `add` says what the new package can do, `audit` lists
// every package's, and a [policy] makes a build refuse what it denies — and
// `add` leaves the manifest alone when the policy refuses the addition.
func TestCapabilitiesAuditAndPolicy(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("VELES_HOME", t.TempDir())
	t.Setenv("VELES_ALLOW_LOCAL_GIT", "1") // the tests' repositories are local
	root := t.TempDir()
	netlib := filepath.Join(root, "remote", "netlib")
	gitRelease(t, netlib, "1.0.0", map[string]string{
		"veles.toml": "[package]\nname = \"netlib\"\nversion = \"1.0.0\"\n",
		"lib.vs":     "use net\n\npublic fun f() {}\n",
	})
	quiet := filepath.Join(root, "remote", "quiet")
	gitRelease(t, quiet, "1.0.0", map[string]string{
		"veles.toml": "[package]\nname = \"quiet\"\nversion = \"1.0.0\"\n",
		"lib.vs":     "public fun f(): i64 => 1\n",
	})
	app := filepath.Join(root, "app")
	os.MkdirAll(app, 0o755)
	manifest := filepath.Join(app, "veles.toml")
	os.WriteFile(manifest, []byte("[package]\nname = \"app\"\n\n[policy]\ndeny = [\"net\"]\n"), 0o644)
	os.WriteFile(filepath.Join(app, "main.vs"), []byte("fun main() {}\n"), 0o644)
	read := func() string { b, _ := os.ReadFile(manifest); return string(b) }

	code, out, errs := capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{fileURL(quiet)}}) })
	if code != 0 || !strings.Contains(out, "quiet can use no unsafe code, C, network, files or processes") {
		t.Fatalf("add quiet: exit %d\n%s\n%s", code, out, errs)
	}
	before := read()
	code, _, errs = capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{fileURL(netlib)}}) })
	if code != 1 || !strings.Contains(errs, "[policy] refuses 1 use(s)") || !strings.Contains(errs, "uses net (lib.vs:1 use net)") || read() != before {
		t.Errorf("add netlib under deny net: exit %d\n%s\nmanifest:\n%s", code, errs, read())
	}

	// without the policy it is added, with the capability said
	os.WriteFile(manifest, []byte("[package]\nname = \"app\"\n"), 0o644)
	code, out, errs = capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{fileURL(netlib)}}) })
	if code != 0 || !strings.Contains(out, "netlib can use: net (with its dependencies)") {
		t.Fatalf("add netlib: exit %d\n%s\n%s", code, out, errs)
	}
	code, out, _ = capture(t, func() int { return Audit(PkgOptions{Dir: app, Detail: true}) })
	if code != 0 || !strings.Contains(out, "no [policy] in veles.toml") || !strings.Contains(out, fileURL(netlib)+" 1.0.0") || !strings.Contains(out, "net    lib.vs:1 use net") {
		t.Errorf("audit: exit %d\n%s", code, out)
	}

	// a policy that denies it: audit reports and exits 1, a build refuses
	os.WriteFile(manifest, []byte(read()+"\n[policy]\ndeny = [\"net\"]\n"), 0o644)
	code, out, _ = capture(t, func() int { return Audit(PkgOptions{Dir: app}) })
	if code != 1 || !strings.Contains(out, "[policy] denies net, and 1 use(s) break it") {
		t.Errorf("audit under policy: exit %d\n%s", code, out)
	}
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 1 || !strings.Contains(errs, "[policy] refuses") {
		t.Errorf("check under policy: exit %d\n%s", code, errs)
	}
	os.WriteFile(manifest, []byte(strings.Replace(read(), "deny = [\"net\"]", "deny = [\"net\"]\nallow = { net = [\"netlib\"] }", 1)), 0o644)
	if code, out, errs := capture(t, func() int { return Audit(PkgOptions{Dir: app}) }); code != 0 || !strings.Contains(out, "no dependency breaks it") {
		t.Errorf("audit with allow: exit %d\n%s\n%s", code, out, errs)
	}
}
