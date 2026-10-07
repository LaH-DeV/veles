package driver

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/registry"
	"github.com/LaH-DeV/veles/sema"
)

// E7 end to end (D138, D139): one program built from every kind of
// dependency at once — a path, git tags with minimal version selection and
// two majors of one repository, a commit pin, and a package published to a
// registry — then the guarantees: the build is reproducible from the module
// cache alone, a moved tag is a hard error before anything is built, and a
// yanked version is refused to a new project.
func TestPackagesEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	home := t.TempDir()
	t.Setenv("VELES_HOME", home)
	t.Setenv("VELES_ALLOW_LOCAL_GIT", "1") // the tests' repositories are local
	t.Setenv("VELES_PROXY", "")
	t.Setenv("VELES_TOKEN", "")
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := func(name, version, deps string) string {
		s := fmt.Sprintf("[package]\nname = %q\nversion = %q\n", name, version)
		if deps != "" {
			s += "[dependencies]\n" + deps + "\n"
		}
		return s
	}

	// a registry with one published package
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	srv := registry.New(priv, map[string]string{"tok": "acme"})
	hs := httptest.NewServer(srv)
	defer hs.Close()
	pub := filepath.Join(root, "published")
	os.MkdirAll(pub, 0o755)
	write("published/veles.toml", "[package]\nname = \"clock\"\nversion = \"1.2.0\"\nregistry = \"acme/clock\"\n")
	write("published/lib.vs", "public fun tick(): i64 = 12\n")
	pm, _ := sema.ReadManifest(pub)
	if _, _, err := fetch.Publish(pub, pm, hs.URL, "tok"); err != nil {
		t.Fatal(err)
	}

	// git: util 1.0.0 and 1.3.0 and 2.0.0; mid needs util 1.3.0; a library at an untagged commit
	util := filepath.Join(root, "remote", "util")
	url := func(d string) string { return fileURL(d) }
	gitRelease(t, util, "1.0.0", map[string]string{"veles.toml": manifest("util", "1.0.0", ""), "lib.vs": "public fun v(): i64 = 100\n"})
	gitRelease(t, util, "1.3.0", map[string]string{"veles.toml": manifest("util", "1.3.0", ""), "lib.vs": "public fun v(): i64 = 130\n"})
	gitRelease(t, util, "2.0.0", map[string]string{"veles.toml": manifest("util", "2.0.0", ""), "lib.vs": "public fun v(): i64 = 200\n"})
	mid := filepath.Join(root, "remote", "mid")
	gitRelease(t, mid, "1.0.0", map[string]string{
		"veles.toml": manifest("mid", "1.0.0", fmt.Sprintf("util = { git = %q, version = \"1.3.0\" }", url(util))),
		"lib.vs":     "use util\n\npublic fun m(): i64 = util.v()\n",
	})
	edge := filepath.Join(root, "remote", "edge")
	gitRelease(t, edge, "0.1.0", map[string]string{"veles.toml": manifest("edge", "0.1.0", ""), "lib.vs": "public fun e(): i64 = 1\n"})
	commit := gitHead(t, edge)

	// a path dependency, and the app that uses everything
	write("local/veles.toml", manifest("local", "0.1.0", ""))
	write("local/lib.vs", "public fun l(): i64 = 7\n")
	write("app/veles.toml", manifest("app", "0.1.0", strings.Join([]string{
		`local = "../local"`,
		fmt.Sprintf("util = { git = %q, version = \"1.0.0\" }", url(util)),
		fmt.Sprintf("mid = { git = %q, version = \"1.0.0\" }", url(mid)),
		fmt.Sprintf("util2 = { git = %q, version = \"2.0.0\" }", url(util)),
		fmt.Sprintf("edge = { git = %q, commit = %q }", url(edge), commit[:10]),
		`clock = { registry = "acme/clock", version = "1.2.0" }`,
	}, "\n"))+"\n[registry]\nurl = \""+hs.URL+"\"\nkey = \""+srv.PublicKey()+"\"\n")
	write("app/main.vs", "use io\nuse local\nuse util\nuse mid\nuse util2\nuse edge\nuse clock\n\nfun main() {\n  io.println(\"${local.l()} ${util.v()} ${mid.m()} ${util2.v()} ${edge.e()} ${clock.tick()}\")\n}\n")
	app := filepath.Join(root, "app")

	exe := filepath.Join(root, "app.exe")
	if code := Run(Options{Path: app, Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build: exit %d", code)
	}
	if got := strings.TrimSpace(runExe(t, exe)); got != "7 130 130 200 1 12" {
		t.Errorf("got %q: util 1.3.0 for the app and mid, util 2.0.0 apart, the commit pin and the registry package built in", got)
	}
	sum, _ := os.ReadFile(filepath.Join(app, "veles.sum"))
	for _, want := range []string{
		"git " + url(util) + " 1.0.0 h1:", "git " + url(util) + " 1.3.0 h1:", "git " + url(util) + " 2.0.0 h1:",
		"git " + url(mid) + " 1.0.0 h1:", "git " + url(edge) + " commit:" + commit + " h1:", "registry acme/clock 1.2.0 h1:",
	} {
		if !strings.Contains(string(sum), want) {
			t.Errorf("veles.sum lacks %q:\n%s", want, sum)
		}
	}

	// the tree and the audit see all of it
	if code, out, errs := capture(t, func() int { return Deps(PkgOptions{Dir: app}) }); code != 0 || !strings.Contains(out, "util 1.0.0 -> 1.3.0") || !strings.Contains(out, "clock 1.2.0") || !strings.Contains(out, "local  path ../local") {
		t.Errorf("deps: exit %d\n%s\n%s", code, out, errs)
	}
	if code, out, errs := capture(t, func() int { return Audit(PkgOptions{Dir: app}) }); code != 0 || !strings.Contains(out, "listed") || !strings.Contains(out, "unreviewed") {
		t.Errorf("audit: exit %d\n%s\n%s", code, out, errs)
	}

	// reproducible from the cache alone: the repositories and the registry are gone
	hs.Close()
	os.RemoveAll(filepath.Join(root, "remote"))
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 0 {
		t.Errorf("check from the module cache alone: exit %d\n%s", code, errs)
	}
}

// A tag that is moved to other contents after veles.sum recorded it stops the
// build before anything is compiled — on a fresh machine, whose cache is empty.
func TestMovedTagStopsTheBuild(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	home := t.TempDir()
	t.Setenv("VELES_HOME", home)
	t.Setenv("VELES_ALLOW_LOCAL_GIT", "1") // the tests' repositories are local
	root := t.TempDir()
	lib := filepath.Join(root, "remote", "lib")
	gitRelease(t, lib, "1.0.0", map[string]string{"veles.toml": "[package]\nname = \"lib\"\nversion = \"1.0.0\"\n", "lib.vs": "public fun v(): i64 = 1\n"})
	app := filepath.Join(root, "app")
	os.MkdirAll(app, 0o755)
	os.WriteFile(filepath.Join(app, "veles.toml"), []byte(fmt.Sprintf("[package]\nname = \"app\"\n[dependencies]\nlib = { git = %q, version = \"1.0.0\" }\n", fileURL(lib))), 0o644)
	os.WriteFile(filepath.Join(app, "main.vs"), []byte("use io\nuse lib\n\nfun main() {\n  io.println(\"${lib.v()}\")\n}\n"), 0o644)
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 0 {
		t.Fatalf("first check: exit %d\n%s", code, errs)
	}

	// someone force-moves the tag to different code; a clean machine has no cache
	os.WriteFile(filepath.Join(lib, "lib.vs"), []byte("public fun v(): i64 = 666\n"), 0o644)
	gitRelease(t, lib, "1.0.0", nil)
	os.RemoveAll(filepath.Join(home, "pkg"))
	code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "build", Output: filepath.Join(root, "x.exe")}) })
	if code != 1 || !strings.Contains(errs, "checksum mismatch for git "+fileURL(lib)+" 1.0.0") || !strings.Contains(errs, "Nothing was built") {
		t.Errorf("moved tag: exit %d\n%s", code, errs)
	}
	if _, err := os.Stat(filepath.Join(root, "x.exe")); err == nil {
		t.Error("an executable was built from a package that failed its checksum")
	}
}

func gitHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
