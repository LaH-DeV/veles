package fetch

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

// --- helpers -------------------------------------------------------------

func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// repo is a local git repository standing in for a remote one.
type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	needGit(t)
	r := &repo{t: t, dir: t.TempDir()}
	run(t, r.dir, "git", "init", "-q")
	return r
}

// url is how a manifest names the repository.
func (r *repo) url() string { return filepath.ToSlash(r.dir) }

// release replaces the files, commits and tags v<version>; it returns the commit.
func (r *repo) release(version string, files map[string]string) string {
	r.t.Helper()
	entries, _ := os.ReadDir(r.dir)
	for _, e := range entries {
		if e.Name() != ".git" {
			os.RemoveAll(filepath.Join(r.dir, e.Name()))
		}
	}
	writeFiles(r.t, r.dir, files)
	run(r.t, r.dir, "git", "add", "-A")
	run(r.t, r.dir, "git", "commit", "-q", "-m", "release "+version)
	if version != "" {
		run(r.t, r.dir, "git", "tag", "-f", "v"+version)
	}
	return run(r.t, r.dir, "git", "rev-parse", "HEAD")
}

func pkgFiles(name, version, requires, code string) map[string]string {
	toml := fmt.Sprintf("[package]\nname = %q\nversion = %q\n", name, version)
	if requires != "" {
		toml += "[dependencies]\n" + requires + "\n"
	}
	return map[string]string{"veles.toml": toml, "lib.vs": code}
}

// project writes a root package and returns its directory and manifest.
func project(t *testing.T, requires string) (string, *sema.Manifest) {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"veles.toml": "[package]\nname = \"app\"\n[dependencies]\n" + requires + "\n",
		"main.vs":    "fun main() {}\n",
	})
	man, err := sema.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, man
}

func home(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("VELES_HOME", h)
	t.Setenv(AllowLocalGit, "1") // the tests' repositories are local
	return h
}

func gitDep(r *repo, version string) string {
	return fmt.Sprintf("{ git = %q, version = %q }", r.url(), version)
}

func readSum(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "veles.sum"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func resolveOK(t *testing.T, dir string, man *sema.Manifest) *Result {
	t.Helper()
	res, err := Resolve(dir, man)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func depDir(res *Result, root, name string) string {
	return res.Dirs[sema.DirKey(root)][name]
}

// --- tests ---------------------------------------------------------------

func TestGitTagIsFetchedHashedAndCached(t *testing.T) {
	home(t)
	lib := newRepo(t)
	lib.release("1.2.0", pkgFiles("lib", "1.2.0", "", "public fun one(): i64 = 1\n"))
	dir, man := project(t, "lib = "+gitDep(lib, "1.2.0"))

	res := resolveOK(t, dir, man)
	got := depDir(res, dir, "lib")
	if got == "" {
		t.Fatalf("no directory for lib: %+v", res.Dirs)
	}
	if _, err := os.Stat(filepath.Join(got, "lib.vs")); err != nil {
		t.Error("the package files are not in the cache directory")
	}
	if _, err := os.Stat(filepath.Join(got, ".git")); err == nil {
		t.Error(".git was kept in the cache")
	}
	sum := readSum(t, dir)
	if !strings.HasPrefix(sum, "git "+lib.url()+" 1.2.0 h1:") || strings.Count(sum, "\n") != 1 {
		t.Errorf("veles.sum:\n%s", sum)
	}
	if len(res.Selected) != 1 || !strings.Contains(res.Selected[0], "1.2.0") {
		t.Errorf("selected: %v", res.Selected)
	}

	// the second resolution needs neither git nor the network
	Offline = true
	defer func() { Offline = false }()
	os.RemoveAll(lib.dir)
	res = resolveOK(t, dir, man)
	if depDir(res, dir, "lib") != got {
		t.Error("the cached copy was not used")
	}
}

// Minimal version selection: `mid` needs util 1.3.0 while the app asks 1.0.0,
// so 1.3.0 is built; util 2.0.0 is another major, a separate package.
func TestMinimalVersionSelectionPerMajor(t *testing.T) {
	home(t)
	util := newRepo(t)
	util.release("1.0.0", pkgFiles("util", "1.0.0", "", "public fun v(): i64 = 100\n"))
	util.release("1.3.0", pkgFiles("util", "1.3.0", "", "public fun v(): i64 = 130\n"))
	util.release("2.0.0", pkgFiles("util", "2.0.0", "", "public fun v(): i64 = 200\n"))
	mid := newRepo(t)
	mid.release("1.0.0", pkgFiles("mid", "1.0.0", "util = "+gitDep(util, "1.3.0"), "use util\npublic fun m(): i64 = util.v()\n"))
	dir, man := project(t, "util = "+gitDep(util, "1.0.0")+"\nmid = "+gitDep(mid, "1.0.0")+"\nutil2 = "+gitDep(util, "2.0.0"))

	res := resolveOK(t, dir, man)
	read := func(d string) string {
		data, err := os.ReadFile(filepath.Join(d, "lib.vs"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if !strings.Contains(read(depDir(res, dir, "util")), "130") {
		t.Error("the app's util should be 1.3.0, the highest asked for in major 1")
	}
	midDir := depDir(res, dir, "mid")
	if depDir(res, midDir, "util") != depDir(res, dir, "util") {
		t.Error("mid and the app must share one util")
	}
	if !strings.Contains(read(depDir(res, dir, "util2")), "200") {
		t.Error("util2 is major 2")
	}
	if len(res.Selected) != 3 {
		t.Errorf("selected: %v", res.Selected)
	}
	// every version mentioned is in veles.sum, the unselected 1.0.0 too
	if sum := readSum(t, dir); strings.Count(sum, "\n") != 4 {
		t.Errorf("veles.sum:\n%s", sum)
	}
}

func TestChecksumMismatchIsAHardError(t *testing.T) {
	h := home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 1\n"))
	dir, man := project(t, "lib = "+gitDep(lib, "1.0.0"))
	resolveOK(t, dir, man)

	// the tag is moved to other contents, and the cache is empty again
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 666\n"))
	os.RemoveAll(filepath.Join(h, "pkg"))
	_, err := Resolve(dir, man)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch for git "+lib.url()+" 1.0.0") || !strings.Contains(err.Error(), "veles.sum records h1:") {
		t.Fatalf("got %v", err)
	}
	// and the moved tag was not kept
	if entries, _ := os.ReadDir(filepath.Join(h, "pkg", "git")); len(entries) != 0 {
		t.Errorf("a package that failed its checksum is in the cache: %v", entries)
	}
}

func TestTamperedCacheIsAHardError(t *testing.T) {
	home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 1\n"))
	dir, man := project(t, "lib = "+gitDep(lib, "1.0.0"))
	res := resolveOK(t, dir, man)
	os.WriteFile(filepath.Join(depDir(res, dir, "lib"), "lib.vs"), []byte("public fun one(): i64 = 2\n"), 0o644)
	_, err := Resolve(dir, man)
	if err == nil || !strings.Contains(err.Error(), "was modified after it was fetched") {
		t.Fatalf("got %v", err)
	}
	// a damaged sum line is caught as well: the cache is right, veles.sum is not
	os.WriteFile(filepath.Join(depDir(res, dir, "lib"), "lib.vs"), []byte("public fun one(): i64 = 1\n"), 0o644)
	sum := readSum(t, dir)
	os.WriteFile(filepath.Join(dir, "veles.sum"), []byte(strings.Replace(sum, "h1:", "h1:0", 1)), 0o644)
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("a wrong veles.sum line: got %v", err)
	}
}

// The hash is of the committed bytes: a .gitattributes that asks for CRLF, or
// autocrlf, must not change them (and so the hash) on any platform.
func TestCheckoutIsByteExact(t *testing.T) {
	home(t)
	lib := newRepo(t)
	files := pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 1\n")
	files[".gitattributes"] = "* text eol=crlf\n"
	lib.release("1.0.0", files)
	dir, man := project(t, "lib = "+gitDep(lib, "1.0.0"))
	res := resolveOK(t, dir, man)
	data, _ := os.ReadFile(filepath.Join(depDir(res, dir, "lib"), "lib.vs"))
	if bytes.Contains(data, []byte("\r")) {
		t.Errorf("line endings were converted: %q", data)
	}
}

func TestCommitPins(t *testing.T) {
	home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 1\n"))
	c2 := lib.release("", pkgFiles("lib", "1.1.0", "", "public fun one(): i64 = 11\n"))
	dir, man := project(t, fmt.Sprintf("lib = { git = %q, commit = %q }", lib.url(), c2[:8]))
	resolveOK(t, dir, man)
	if sum := readSum(t, dir); !strings.Contains(sum, "commit:"+c2+" h1:") {
		t.Errorf("veles.sum should hold the full commit:\n%s", sum)
	}

	// a tag and a commit of one (repository, major) are one package: say one
	dir, man = project(t, "lib = "+gitDep(lib, "1.0.0")+"\n"+fmt.Sprintf("pin = { git = %q, commit = %q }", lib.url(), c2[:8]))
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "pinned by tag 1.0.0 and by commit "+c2[:10]) {
		t.Errorf("tag and commit: got %v", err)
	}

	// two commits of one repository
	c1 := run(t, lib.dir, "git", "rev-parse", "v1.0.0")
	dir, man = project(t, fmt.Sprintf("a = { git = %q, commit = %q }\nb = { git = %q, commit = %q }", lib.url(), c1[:8], lib.url(), c2[:8]))
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "is pinned to two commits") {
		t.Errorf("two commits: got %v", err)
	}
}

// A package others build on that pins a commit cannot take part in version
// selection; the program using it is told.
func TestLibraryCommitPinWarns(t *testing.T) {
	home(t)
	base := newRepo(t)
	c := base.release("1.0.0", pkgFiles("base", "1.0.0", "", "public fun b(): i64 = 1\n"))
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", fmt.Sprintf("base = { git = %q, commit = %q }", base.url(), c[:8]), "use base\npublic fun l(): i64 = base.b()\n"))
	dir, man := project(t, "lib = "+gitDep(lib, "1.0.0"))
	res := resolveOK(t, dir, man)
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "package 'lib'") || !strings.Contains(res.Warnings[0], "at commit "+c[:8]) {
		t.Errorf("warnings: %v", res.Warnings)
	}
}

func TestFetchedPackageCannotDependOnADirectory(t *testing.T) {
	home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", `other = "../other"`, "public fun l(): i64 = 1\n"))
	dir, man := project(t, "lib = "+gitDep(lib, "1.0.0"))
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "cannot depend on a directory") {
		t.Fatalf("got %v", err)
	}
}

func TestMissingTagAndOffline(t *testing.T) {
	h := home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 1\n"))
	dir, man := project(t, "lib = "+gitDep(lib, "9.9.9"))
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "has no tag v9.9.9 or 9.9.9") || !strings.Contains(err.Error(), "veles.toml:4: dependency 'lib'") {
		t.Errorf("missing tag: got %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(h, "tmp")); len(entries) != 0 {
		t.Errorf("a failed fetch left temporary directories: %v", entries)
	}
	dir, man = project(t, "lib = "+gitDep(lib, "1.0.0"))
	Offline = true
	defer func() { Offline = false }()
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "not in the module cache; run `veles fetch`") {
		t.Errorf("offline: got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "veles.sum")); err == nil {
		t.Error("a failed resolution wrote veles.sum")
	}
}

func TestNoRemoteDependencyTouchesNothing(t *testing.T) {
	h := home(t)
	dir, man := project(t, "")
	res := resolveOK(t, dir, man)
	if len(res.Dirs) != 0 || len(res.Selected) != 0 {
		t.Errorf("%+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "veles.sum")); err == nil {
		t.Error("veles.sum written for a package with no remote dependency")
	}
	if _, err := os.Stat(h + "/pkg"); err == nil {
		t.Error("the cache was created")
	}
}

// In a workspace the sum file is the root's, shared by the members.
func TestWorkspaceSharesOneSum(t *testing.T) {
	home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "public fun one(): i64 = 1\n"))
	ws := t.TempDir()
	writeFiles(t, ws, map[string]string{
		"veles.toml":        "[workspace]\nmembers = [\"apps/a\"]\n",
		"apps/a/veles.toml": "[package]\nname = \"a\"\n[dependencies]\nlib = " + gitDep(lib, "1.0.0") + "\n",
		"apps/a/main.vs":    "fun main() {}\n",
	})
	member := filepath.Join(ws, "apps", "a")
	man, err := sema.ReadManifest(member)
	if err != nil {
		t.Fatal(err)
	}
	resolveOK(t, member, man)
	if !strings.Contains(readSum(t, ws), "git "+lib.url()+" 1.0.0 h1:") {
		t.Error("the workspace root's veles.sum has no entry")
	}
	if _, err := os.Stat(filepath.Join(member, "veles.sum")); err == nil {
		t.Error("the member has a veles.sum of its own")
	}
}

// --- the proxy -----------------------------------------------------------

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, text := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(text))
	}
	w.Close()
	return buf.Bytes()
}

func TestRegistryPackageThroughAProxy(t *testing.T) {
	home(t)
	archive := zipOf(t, pkgFiles("httputil", "1.4.2", "", "public fun get(): i64 = 1\n"))
	hits := 0
	unpacked := t.TempDir()
	zp := filepath.Join(unpacked, "a.zip")
	os.WriteFile(zp, archive, 0o644)
	pdir, err := unzip(unpacked, zp)
	if err != nil {
		t.Fatal(err)
	}
	phash, _ := TreeHash(pdir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/acme/httputil/@v/1.4.2.zip":
			hits++
			w.Write(archive)
		case "/acme/httputil/@v/1.4.2.info":
			fmt.Fprintf(w, `{"version":"1.4.2","time":"2026-10-07T00:00:00Z","hash":%q,"tier":"unreviewed"}`, phash)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("VELES_PROXY", srv.URL+"/")
	dir, man := project(t, `httputil = { registry = "acme/httputil", version = "1.4.2" }`)
	res := resolveOK(t, dir, man)
	if _, err := os.Stat(filepath.Join(depDir(res, dir, "httputil"), "lib.vs")); err != nil {
		t.Error("the archive was not unpacked")
	}
	if sum := readSum(t, dir); !strings.HasPrefix(sum, "registry acme/httputil 1.4.2 h1:") {
		t.Errorf("veles.sum:\n%s", sum)
	}
	resolveOK(t, dir, man)
	if hits != 1 {
		t.Errorf("the proxy was asked %d times; the cache should serve the second", hits)
	}

	// the same files from a git repository hash the same: a registry and a
	// repository serving one release agree
	h1, _ := TreeHash(depDir(res, dir, "httputil"))
	lib := newRepo(t)
	lib.release("1.4.2", pkgFiles("httputil", "1.4.2", "", "public fun get(): i64 = 1\n"))
	dir2, man2 := project(t, "x = "+gitDep(lib, "1.4.2"))
	res2 := resolveOK(t, dir2, man2)
	if h2, _ := TreeHash(depDir(res2, dir2, "x")); h1 != h2 {
		t.Errorf("registry %s, git %s", h1, h2)
	}
}

func TestProxyErrors(t *testing.T) {
	home(t)
	dir, man := project(t, `m = { registry = "acme/m", version = "1.0.0" }`)
	t.Setenv("VELES_PROXY", "")
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "needs a proxy") || !strings.Contains(err.Error(), "VELES_PROXY") {
		t.Errorf("no proxy: got %v", err)
	}
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	t.Setenv("VELES_PROXY", srv.URL)
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Errorf("404: got %v", err)
	}
}

func TestUnsafeArchivesAreRefused(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"parent path": {"veles.toml": "x", "../evil.vs": "x"},
		"absolute":    {"/etc/evil": "x"},
		"backslash":   {"a\\..\\evil": "x"},
		"drive":       {"C:/evil": "x"},
	} {
		zp := filepath.Join(t.TempDir(), "a.zip")
		os.WriteFile(zp, zipOf(t, files), 0o644)
		tmp := t.TempDir()
		if _, err := unzip(tmp, zp); err == nil || !strings.Contains(err.Error(), "unsafe path") {
			t.Errorf("%s: got %v", name, err)
		}
		if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
			t.Errorf("%s: left %v behind", name, entries)
		}
	}
}

func TestSumFileFormat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "veles.sum")
	os.WriteFile(p, []byte("# comment\r\ngit u 1.0.0 h1:aa\r\n\r\nregistry a/b 2.0.0 h1:bb\r\n"), 0o644)
	s, err := LoadSum(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Check("git u 1.0.0", "h1:aa"); err != nil {
		t.Error(err)
	}
	s.Check("git a 0.1.0", "h1:cc")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	if string(data) != "git a 0.1.0 h1:cc\ngit u 1.0.0 h1:aa\nregistry a/b 2.0.0 h1:bb\n" {
		t.Errorf("saved:\n%s", data)
	}
	for _, bad := range []string{"git u 1.0.0\n", "svn u 1.0.0 h1:a\n", "git u 1.0.0 sha:a\n", "git u 1.0.0 h1:a\ngit u 1.0.0 h1:b\n"} {
		os.WriteFile(p, []byte(bad), 0o644)
		if _, err := LoadSum(p); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}

func TestTreeHashIsAboutBytesOnly(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeFiles(t, a, map[string]string{"x/1.vs": "one", "2.vs": "two"})
	writeFiles(t, b, map[string]string{"2.vs": "two", "x/1.vs": "one", ".git/config": "ignored", markerFile: "ignored"})
	ha, _ := TreeHash(a)
	hb, _ := TreeHash(b)
	if ha != hb || !strings.HasPrefix(ha, "h1:") {
		t.Errorf("%s vs %s", ha, hb)
	}
	writeFiles(t, b, map[string]string{"2.vs": "two\r\n"})
	if hc, _ := TreeHash(b); hc == ha {
		t.Error("a CRLF is a different byte and must change the hash")
	}
}
