package fetch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

func TestVersionsAndLatest(t *testing.T) {
	home(t)
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("lib", "1.0.0", "", "a\n"))
	lib.release("1.4.0", pkgFiles("lib", "1.4.0", "", "b\n"))
	lib.release("2.0.0-beta.1", pkgFiles("lib", "2.0.0-beta.1", "", "c\n"))
	run(t, lib.dir, "git", "tag", "nightly")
	f := NewFetcher(nil)
	vs, err := f.Versions("git", lib.url())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range vs {
		got = append(got, v.String())
	}
	if strings.Join(got, " ") != "1.0.0 1.4.0 2.0.0-beta.1" {
		t.Errorf("versions: %v (a tag that is not a version is ignored)", got)
	}
	if v, ok := Latest(vs, ""); !ok || v.String() != "1.4.0" {
		t.Errorf("latest: %v %v (a pre-release is never picked on its own)", v, ok)
	}
	if v, ok := Latest(vs, "1"); !ok || v.String() != "1.4.0" {
		t.Errorf("latest of major 1: %v %v", v, ok)
	}
	if _, ok := Latest(vs, "3"); ok {
		t.Error("there is no major 3")
	}
	f.Offline = true
	if _, err := f.Versions("git", lib.url()); err == nil {
		t.Error("offline must not list")
	}
}

func TestProxyVersionList(t *testing.T) {
	home(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/acme/m/@v/list" {
			w.Write([]byte("v1.0.0\n1.2.0\nnot-a-version\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	f := NewFetcher(nil)
	f.Proxy = srv.URL
	vs, err := f.Versions("registry", "acme/m")
	if err != nil || len(vs) != 2 || vs[1].String() != "1.2.0" {
		t.Errorf("%v %v", vs, err)
	}
	if _, err := f.Versions("registry", "acme/none"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("got %v", err)
	}
	f.Proxy = ""
	if _, err := f.Versions("registry", "acme/m"); err == nil || !strings.Contains(err.Error(), "needs a proxy") {
		t.Errorf("got %v", err)
	}
}

// The tree shows what selection chose, marks a package met twice, and --why
// lists every way into one.
func TestTreeAndWhy(t *testing.T) {
	home(t)
	util := newRepo(t)
	util.release("1.0.0", pkgFiles("util", "1.0.0", "", "a\n"))
	util.release("1.3.0", pkgFiles("util", "1.3.0", "", "b\n"))
	mid := newRepo(t)
	mid.release("1.0.0", pkgFiles("mid", "1.0.0", "util = "+gitDep(util, "1.3.0"), "c\n"))
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"veles.toml":          "[package]\nname = \"app\"\n[dependencies]\nutil = " + gitDep(util, "1.0.0") + "\nmid = " + gitDep(mid, "1.0.0") + "\nlocal = \"../local\"\n",
		"main.vs":             "fun main() {}\n",
		"../local/veles.toml": "[package]\nname = \"local\"\n",
	})
	man, err := sema.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	res := resolveOK(t, dir, man)
	tree := res.Tree()
	for _, want := range []string{
		"app\n",
		"+- local  path ../local\n",
		"+- mid 1.0.0  (git " + mid.url() + ")\n",
		"|  `- util 1.3.0  (git " + util.url() + ")\n",
		"`- util 1.0.0 -> 1.3.0  (git " + util.url() + ")  (*)\n",
	} {
		if !strings.Contains(tree, want) {
			t.Errorf("missing %q in:\n%s", want, tree)
		}
	}
	why, ok := res.Why("util")
	if !ok || strings.Count(why, "app") != 2 || !strings.Contains(why, "mid 1.0.0") || !strings.Contains(why, "util 1.3.0") {
		t.Errorf("why util (%v):\n%s", ok, why)
	}
	if _, ok := res.Why("nothing"); ok {
		t.Error("nothing matches")
	}
	if why, ok := res.Why("local"); !ok || !strings.Contains(why, "local  path ../local") {
		t.Errorf("why local: %s", why)
	}
}

func TestVendorBuildsOffline(t *testing.T) {
	h := home(t)
	util := newRepo(t)
	util.release("1.0.0", pkgFiles("util", "1.0.0", "", "a\n"))
	util.release("1.3.0", pkgFiles("util", "1.3.0", "", "b\n"))
	mid := newRepo(t)
	mid.release("1.0.0", pkgFiles("mid", "1.0.0", "util = "+gitDep(util, "1.3.0"), "c\n"))
	dir, man := project(t, "util = "+gitDep(util, "1.0.0")+"\nmid = "+gitDep(mid, "1.0.0"))

	n, err := Vendor(dir, man)
	if err != nil || n != 3 { // util 1.0.0, util 1.3.0 (its manifest is read by selection) and mid
		t.Fatalf("vendored %d: %v", n, err)
	}
	// nothing but vendor/ is used now: no cache, no repositories
	os.RemoveAll(filepath.Join(h, "pkg"))
	os.RemoveAll(util.dir)
	os.RemoveAll(mid.dir)
	res := resolveOK(t, dir, man)
	got := depDir(res, dir, "util")
	if !strings.HasPrefix(got, filepath.Join(dir, "vendor")) {
		t.Errorf("util comes from %s, not vendor/", got)
	}
	if _, err := os.Stat(filepath.Join(h, "pkg")); err == nil {
		t.Error("the module cache was used")
	}

	// vendor/ is checked against veles.sum like the cache: edit a file
	os.WriteFile(filepath.Join(got, "lib.vs"), []byte("tampered\n"), 0o644)
	if _, err := Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "was modified after it was fetched") {
		t.Errorf("tampered vendor: got %v", err)
	}
	// and a package vendor/ lacks says how to fix that
	dir2, man2 := project(t, "util = "+gitDep(util, "1.0.0"))
	os.MkdirAll(filepath.Join(dir2, "vendor"), 0o755)
	if _, err := Resolve(dir2, man2); err == nil || !strings.Contains(err.Error(), "run `veles vendor`") {
		t.Errorf("missing from vendor: got %v", err)
	}
}

func TestVendorWithNothingToVendor(t *testing.T) {
	home(t)
	dir, man := project(t, "")
	if _, err := Vendor(dir, man); err == nil || !strings.Contains(err.Error(), "no registry or git dependencies to vendor") {
		t.Errorf("got %v", err)
	}
}

func TestPruneSum(t *testing.T) {
	home(t)
	a := newRepo(t)
	a.release("1.0.0", pkgFiles("a", "1.0.0", "", "a\n"))
	a.release("1.1.0", pkgFiles("a", "1.1.0", "", "b\n"))
	dir, man := project(t, "a = "+gitDep(a, "1.0.0"))
	resolveOK(t, dir, man)
	// the author moves to 1.1.0: the old line is now unused
	man2 := rewrite(t, dir, "a = "+gitDep(a, "1.1.0"))
	res := resolveOK(t, dir, man2)
	n, err := res.PruneSum()
	if err != nil || n != 1 {
		t.Fatalf("pruned %d: %v", n, err)
	}
	if sum := readSum(t, dir); strings.Contains(sum, "1.0.0") || !strings.Contains(sum, "1.1.0") {
		t.Errorf("veles.sum:\n%s", sum)
	}
	// no remote dependency left: the whole file goes
	rewrite(t, dir, "")
	if n, err := PruneAllSum(dir); err != nil || n != 1 {
		t.Errorf("PruneAllSum: %d %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "veles.sum")); err == nil {
		t.Error("veles.sum should be gone")
	}
}

func rewrite(t *testing.T, dir, requires string) *sema.Manifest {
	t.Helper()
	writeFiles(t, dir, map[string]string{"veles.toml": "[package]\nname = \"app\"\n[dependencies]\n" + requires + "\n"})
	man, err := sema.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	return man
}
