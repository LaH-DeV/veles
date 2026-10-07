package driver

import (
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/registry"
	"github.com/LaH-DeV/veles/sema"
)

// D139 through the commands, against the reference server: publish (and the
// refusal of a package that does not check), add from the registry, reviews
// signed, pushed, verified and required by a policy, and yank.
func TestRegistryCommands(t *testing.T) {
	t.Setenv("VELES_HOME", t.TempDir())
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	srv := registry.New(priv, map[string]string{"tok-acme": "acme"})
	hs := httptest.NewServer(srv)
	defer hs.Close()
	t.Setenv("VELES_PROXY", hs.URL)
	t.Setenv("VELES_TOKEN", "tok-acme")
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// publish: a package that does not check is not uploaded
	write("lib/veles.toml", "[package]\nname = \"lib\"\nversion = \"1.0.0\"\nregistry = \"acme/lib\"\n")
	write("lib/lib.vs", "public fun f(): i64 = \"not a number\"\n")
	lib := filepath.Join(root, "lib")
	if code, _, errs := capture(t, func() int { return Publish(PkgOptions{Dir: lib}) }); code != 1 || !strings.Contains(errs, "does not check, so it was not published") {
		t.Errorf("publish a broken package: exit %d\n%s", code, errs)
	}
	write("lib/lib.vs", "public fun f(): i64 = 1\n")
	code, out, errs := capture(t, func() int { return Publish(PkgOptions{Dir: lib}) })
	if code != 0 || !strings.Contains(out, "published acme/lib 1.0.0") {
		t.Fatalf("publish: exit %d\n%s\n%s", code, out, errs)
	}
	if code, _, errs := capture(t, func() int { return Publish(PkgOptions{Dir: lib}) }); code != 1 || !strings.Contains(errs, "versions are immutable") {
		t.Errorf("publish twice: exit %d\n%s", code, errs)
	}

	// add from the registry: the newest release, with its tier said
	write("app/veles.toml", "[package]\nname = \"app\"\n\n[registry]\nurl = \""+hs.URL+"\"\nkey = \""+srv.PublicKey()+"\"\n")
	write("app/main.vs", "use io\nuse lib\n\nfun main() {\n  io.println(\"${lib.f()}\")\n}\n")
	app := filepath.Join(root, "app")
	code, out, errs = capture(t, func() int { return Add(PkgOptions{Dir: app, Args: []string{"acme/lib"}}) })
	if code != 0 || !strings.Contains(out, "added lib = acme/lib 1.0.0") || !strings.Contains(out, "tier: listed") {
		t.Fatalf("add: exit %d\n%s\n%s", code, out, errs)
	}
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 0 {
		t.Fatalf("check: exit %d\n%s", code, errs)
	}

	// reviews: a key, a signed statement kept in the project and pushed to the registry
	code, out, _ = capture(t, func() int { return Attest(PkgOptions{Dir: app, Args: []string{"keygen", "alice"}}) })
	if code != 0 || !strings.Contains(out, `alice = "ed25519:`) {
		t.Fatalf("keygen: exit %d\n%s", code, out)
	}
	pub := out[strings.Index(out, `"ed25519:`)+1:]
	pub = pub[:strings.Index(pub, `"`)]
	if code, _, errs := capture(t, func() int { return Attest(PkgOptions{Dir: app, Args: []string{"sign", "lib"}}) }); code != 1 || !strings.Contains(errs, "say whose review this is") {
		t.Errorf("sign without a key: exit %d\n%s", code, errs)
	}
	code, out, errs = capture(t, func() int {
		return Attest(PkgOptions{Dir: app, Args: []string{"sign", "lib"}, Key: "alice", Push: true})
	})
	if code != 0 || !strings.Contains(out, `signed "reviewed" for acme/lib 1.0.0`) || !strings.Contains(out, "pushed to the registry") {
		t.Fatalf("sign: exit %d\n%s\n%s", code, out, errs)
	}
	code, out, _ = capture(t, func() int { return Attest(PkgOptions{Dir: app, Args: []string{"verify"}}) })
	if code != 0 || !strings.Contains(out, "ok") || !strings.Contains(out, `"reviewed"`) {
		t.Errorf("verify: exit %d\n%s", code, out)
	}

	// a policy that requires the review: held by the project, and then by the registry alone
	manifest := filepath.Join(app, "veles.toml")
	text, _ := os.ReadFile(manifest)
	os.WriteFile(manifest, []byte(string(text)+"\n[policy]\ntrust = { alice = \""+pub+"\" }\nrequire = { reviewed = 1 }\n"), 0o644)
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 0 {
		t.Errorf("a reviewed dependency under require: exit %d\n%s", code, errs)
	}
	code, out, _ = capture(t, func() int { return Audit(PkgOptions{Dir: app, Detail: true}) })
	if code != 0 || !strings.Contains(out, "listed") || !strings.Contains(out, "reviewed by alice") {
		t.Errorf("audit: exit %d\n%s", code, out)
	}
	os.RemoveAll(filepath.Join(app, "attestations"))
	if code, _, errs := capture(t, func() int { return Run(Options{Path: app, Mode: "check"}) }); code != 0 {
		t.Errorf("the registry holds the review: exit %d\n%s", code, errs) // pushed above
	}

	// import: a file of statements, one good and one tampered
	rep, _ := fetch.ParseStatements("x", []byte(srvAttest(t, hs.URL)))
	if len(rep) != 1 {
		t.Fatalf("the registry should serve one statement, has %d", len(rep))
	}
	good := rep[0].String()
	bad := strings.Replace(good, " reviewed ", " audited ", 1)
	write("shared.attest", good+"\n"+bad+"\n")
	code, out, errs = capture(t, func() int {
		return Attest(PkgOptions{Dir: app, Args: []string{"import", filepath.Join(root, "shared.attest")}})
	})
	if code != 1 || !strings.Contains(out, "imported 1 new statement(s)") || !strings.Contains(errs, "not imported:") || !strings.Contains(errs, "signature does not match") {
		t.Errorf("import: exit %d\n%s\n%s", code, out, errs)
	}

	// yank: a new project cannot add it any more
	if code, _, errs := capture(t, func() int {
		return Yank(PkgOptions{Dir: app, Args: []string{"acme/lib@1.0.0"}, Reason: "broken"})
	}); code != 0 {
		t.Fatalf("yank: exit %d\n%s", code, errs)
	}
	write("other/veles.toml", "[package]\nname = \"other\"\n\n[registry]\nurl = \""+hs.URL+"\"\n")
	other := filepath.Join(root, "other")
	code, _, errs = capture(t, func() int { return Add(PkgOptions{Dir: other, Args: []string{"acme/lib@1.0.0"}}) })
	if code != 1 || !strings.Contains(errs, "was yanked from the registry: broken") {
		t.Errorf("add a yanked version: exit %d\n%s", code, errs)
	}
	if code, _, errs := capture(t, func() int {
		return Yank(PkgOptions{Dir: app, Args: []string{"acme/lib@1.0.0"}, Undo: true})
	}); code != 0 {
		t.Errorf("undo: exit %d\n%s", code, errs)
	}
}

// srvAttest reads the statements the registry serves for acme/lib 1.0.0.
func srvAttest(t *testing.T, base string) string {
	t.Helper()
	resp, err := http.Get(base + "/acme/lib/@v/1.0.0.attest")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data)
}

// `veles yank` needs no package: the environment names the registry.
func TestYankFromAnywhere(t *testing.T) {
	t.Setenv("VELES_HOME", t.TempDir())
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	srv := registry.New(priv, map[string]string{"tok": "acme"})
	hs := httptest.NewServer(srv)
	defer hs.Close()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "veles.toml"), []byte("[package]\nname = \"lib\"\nversion = \"1.0.0\"\nregistry = \"acme/lib\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "lib.vs"), []byte("public fun f(): i64 = 1\n"), 0o644)
	man, _ := sema.ReadManifestIn(dir)
	if _, _, err := fetch.Publish(dir, man, hs.URL, "tok"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VELES_PROXY", hs.URL)
	t.Setenv("VELES_TOKEN", "tok")
	elsewhere := t.TempDir()
	if code, _, errs := capture(t, func() int { return Yank(PkgOptions{Dir: elsewhere, Args: []string{"acme/lib@1.0.0"}, Reason: "x"}) }); code != 0 {
		t.Errorf("yank from a directory with no package: exit %d\n%s", code, errs)
	}
	t.Setenv("VELES_PROXY", "")
	if code, _, errs := capture(t, func() int { return Yank(PkgOptions{Dir: elsewhere, Args: []string{"acme/lib@1.0.0"}}) }); code != 1 || !strings.Contains(errs, "has no veles.toml") {
		t.Errorf("yank with no registry anywhere: exit %d\n%s", code, errs)
	}
}
