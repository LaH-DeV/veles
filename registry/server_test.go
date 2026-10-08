package registry_test

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
	"time"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/registry"
	"github.com/LaH-DeV/veles/sema"
)

// --- helpers -----------------------------------------------------------------

func write(t *testing.T, dir string, files map[string]string) {
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

type env struct {
	t      *testing.T
	srv    *registry.Server
	http   *httptest.Server
	pubKey string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	t.Setenv("VELES_HOME", t.TempDir())
	t.Setenv("VELES_PROXY", "")
	t.Setenv("VELES_TOKEN", "")
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	srv := registry.New(priv, map[string]string{"tok-acme": "acme", "tok-bob": "bob"})
	srv.Now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
	h := httptest.NewServer(srv)
	t.Cleanup(h.Close)
	return &env{t: t, srv: srv, http: h, pubKey: srv.PublicKey()}
}

// pkgDir writes a publishable package.
func pkgDir(t *testing.T, name, registryName, version, code string) (string, *sema.Manifest) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"veles.toml": "[package]\nname = \"" + name + "\"\nversion = \"" + version + "\"\nregistry = \"" + registryName + "\"\n",
		"lib.vs":     code,
	})
	man, err := sema.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, man
}

func (e *env) publish(registryName, version, code string) string {
	e.t.Helper()
	dir, man := pkgDir(e.t, "httputil", registryName, version, code)
	owner := strings.Split(registryName, "/")[0]
	if _, _, err := fetch.Publish(dir, man, e.http.URL, "tok-"+owner); err != nil {
		e.t.Fatalf("publish %s %s: %v", registryName, version, err)
	}
	return dir
}

// app writes a project depending on acme/httputil 1.0.0 (plus extra manifest text).
func (e *env) app(version, extra string) (string, *sema.Manifest) {
	e.t.Helper()
	dir := e.t.TempDir()
	write(e.t, dir, map[string]string{
		"veles.toml": "[package]\nname = \"app\"\n[dependencies]\nhttputil = { registry = \"acme/httputil\", version = \"" + version + "\" }\n[registry]\nurl = \"" + e.http.URL + "\"\nkey = \"" + e.pubKey + "\"\n" + extra,
		"main.vs":    "fun main() {}\n",
	})
	man, err := sema.ReadManifest(dir)
	if err != nil {
		e.t.Fatal(err)
	}
	return dir, man
}

// --- the life cycle ------------------------------------------------------------

func TestPublishFetchAndTier(t *testing.T) {
	e := newEnv(t)
	e.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 1\n")
	e.publish("acme/httputil", "0.9.0", "public fun get(): i64 => 0\n")

	// versions are immutable, and a token writes only below its owner
	dir, man := pkgDir(t, "httputil", "acme/httputil", "1.0.0", "public fun get(): i64 => 2\n")
	if _, _, err := fetch.Publish(dir, man, e.http.URL, "tok-acme"); err == nil || !strings.Contains(err.Error(), "(409)") || !strings.Contains(err.Error(), "immutable") {
		t.Errorf("republish: %v", err)
	}
	if _, _, err := fetch.Publish(dir, man, e.http.URL, "tok-bob"); err == nil || !strings.Contains(err.Error(), "(403)") {
		t.Errorf("another owner: %v", err)
	}
	if _, _, err := fetch.Publish(dir, man, e.http.URL, ""); err == nil || !strings.Contains(err.Error(), "needs a token") {
		t.Errorf("no token: %v", err)
	}
	noReg, nrMan := pkgDir(t, "x", "acme/x", "1.0.0", "a\n")
	nrMan.RegistryName = ""
	if _, _, err := fetch.Publish(noReg, nrMan, e.http.URL, "tok-acme"); err == nil || !strings.Contains(err.Error(), "registry = \"owner/name\"") {
		t.Errorf("no registry name: %v", err)
	}

	// the client lists, reads signed metadata, and learns the tier
	f := fetch.NewClient(dirOf(t, e, "1.0.0"))
	vs, err := f.Versions("registry", "acme/httputil")
	if err != nil || len(vs) != 2 || vs[0].String() != "0.9.0" || vs[1].String() != "1.0.0" {
		t.Fatalf("versions: %v %v", vs, err)
	}
	for version, tier := range map[string]string{"1.0.0": "listed", "0.9.0": "unreviewed"} {
		appDir, appMan := e.app(version, "[policy]\ndeny = [\"unlisted\"]\n")
		_, err := fetch.Resolve(appDir, appMan)
		if tier == "listed" && err != nil {
			t.Errorf("%s is listed but was refused: %v", version, err)
		}
		if tier == "unreviewed" && (err == nil || !strings.Contains(err.Error(), "httputil 0.9.0 is unlisted (tier: unreviewed (the registry has not listed this version))")) {
			t.Errorf("%s is unreviewed and should be refused: %v", version, err)
		}
	}
	// path dependencies are never policed, and an allow exempts
	appDir, appMan := e.app("0.9.0", "[policy]\ndeny = [\"unlisted\"]\nallow = { unlisted = [\"acme/httputil\"] }\n")
	if _, err := fetch.Resolve(appDir, appMan); err != nil {
		t.Errorf("allowed: %v", err)
	}
}

func dirOf(t *testing.T, e *env, version string) (string, *sema.Manifest) {
	d, m := e.app(version, "")
	return d, m
}

// A pinned key makes the registry prove itself; a different key, or metadata
// the archive does not match, stops the fetch before anything is cached.
func TestRegistryIntegrity(t *testing.T) {
	e := newEnv(t)
	e.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 1\n")

	// not the pinned key
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	dir, man := e.app("1.0.0", "")
	man.Registry.Key = fetch.EncodePublic(other.Public().(ed25519.PublicKey))
	os.WriteFile(filepath.Join(dir, "veles.toml"), []byte(strings.Replace(mustRead(t, dir), e.pubKey, man.Registry.Key, 1)), 0o644)
	man, _ = sema.ReadManifest(dir)
	if _, err := fetch.Resolve(dir, man); err == nil || !strings.Contains(err.Error(), "not the key this project pins") || !strings.Contains(err.Error(), "nothing was fetched") {
		t.Errorf("wrong key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "veles.sum")); err == nil {
		t.Error("veles.sum written for a refused registry")
	}

	// the right key, but the archive is not what the metadata says
	liar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".zip") {
			w.Write(zipBytes(t, map[string]string{"veles.toml": "[package]\nname = \"httputil\"\n", "lib.vs": "public fun get(): i64 => 666\n"}))
			return
		}
		e.srv.ServeHTTP(w, r)
	}))
	defer liar.Close()
	dir2, man2 := e.app("1.0.0", "")
	man2.Registry.URL = liar.URL
	if _, err := fetch.Resolve(dir2, man2); err == nil || !strings.Contains(err.Error(), "the registry serves two different packages for one version") {
		t.Errorf("archive vs metadata: %v", err)
	}

	// without a pinned key the registry is trusted on first use, and the sum then holds it
	dir3 := t.TempDir()
	write(t, dir3, map[string]string{
		"veles.toml": "[package]\nname = \"app\"\n[dependencies]\nhttputil = { registry = \"acme/httputil\", version = \"1.0.0\" }\n[registry]\nurl = \"" + e.http.URL + "\"\n",
		"main.vs":    "fun main() {}\n",
	})
	man3, _ := sema.ReadManifest(dir3)
	if _, err := fetch.Resolve(dir3, man3); err != nil {
		t.Errorf("unpinned: %v", err)
	}
}

func mustRead(t *testing.T, dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "veles.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, files)
	data, err := fetch.ZipPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Cargo's rule: a new resolution refuses a yanked version and `add`/`update`
// skip it; a project whose veles.sum already holds it carries on, warned.
func TestYank(t *testing.T) {
	e := newEnv(t)
	e.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 1\n")
	e.publish("acme/httputil", "1.1.0", "public fun get(): i64 => 2\n")

	// a project builds against 1.1.0 and records it
	known, knownMan := e.app("1.1.0", "")
	if _, err := fetch.Resolve(known, knownMan); err != nil {
		t.Fatal(err)
	}

	if err := fetch.Yank(e.http.URL, "acme/httputil", "1.1.0", "it deletes files", "tok-bob", false); err == nil || !strings.Contains(err.Error(), "(403)") {
		t.Errorf("another owner yanks: %v", err)
	}
	if err := fetch.Yank(e.http.URL, "acme/httputil", "1.1.0", "it deletes files", "tok-acme", false); err != nil {
		t.Fatal(err)
	}

	// a fresh project (no sum line) is refused, with the reason
	fresh, freshMan := e.app("1.1.0", "")
	if _, err := fetch.Resolve(fresh, freshMan); err == nil || !strings.Contains(err.Error(), "was yanked from the registry: it deletes files") {
		t.Errorf("fresh resolution: %v", err)
	}
	// the project that already holds it goes on, with a warning (the cache has the
	// old metadata, so the warning appears once the version is fetched again)
	os.RemoveAll(filepath.Join(os.Getenv("VELES_HOME"), "pkg"))
	res, err := fetch.Resolve(known, knownMan)
	if err != nil {
		t.Fatalf("a build that holds the yanked version: %v", err)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "veles.sum already holds it") {
		t.Errorf("warnings: %v", res.Warnings)
	}

	// the newest usable version skips the yanked one
	f := fetch.NewClient(known, knownMan)
	vs, _ := f.Versions("registry", "acme/httputil")
	if v, ok := f.LatestUsable("registry", "acme/httputil", vs, ""); !ok || v.String() != "1.0.0" {
		t.Errorf("latest usable: %v %v", v, ok)
	}
	if err := fetch.Yank(e.http.URL, "acme/httputil", "1.1.0", "", "tok-acme", true); err != nil {
		t.Fatal(err)
	}
	if v, _ := f.LatestUsable("registry", "acme/httputil", vs, ""); v.String() != "1.1.0" {
		t.Errorf("after undo: %v", v)
	}
}

// Reviews: signed, portable, counted only from trusted keys and only for the
// exact bytes; held in the project or served by the registry.
func TestReviews(t *testing.T) {
	e := newEnv(t)
	e.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 1\n")
	_, aliceKey, _ := ed25519.GenerateKey(rand.Reader)
	_, malloryKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := fetch.EncodePublic(aliceKey.Public().(ed25519.PublicKey))
	policy := "[policy]\ntrust = { alice = \"" + alice + "\" }\nrequire = { reviewed = 1 }\n"

	dir, man := e.app("1.0.0", policy)
	_, err := fetch.Resolve(dir, man)
	if err == nil || !strings.Contains(err.Error(), "has 0 of the 1 trusted \"reviewed\" review(s)") || !strings.Contains(err.Error(), "veles attest sign") {
		t.Fatalf("unreviewed: %v", err)
	}

	// the subject: the build's package, as veles.sum knows it
	res, err := fetch.ResolveWith(dir, man, fetch.ResolveOptions{SkipPolicy: true})
	if err != nil {
		t.Fatal(err)
	}
	report, _ := res.Report()
	p := report[0]
	st := func(key ed25519.PrivateKey, hash, claim string) fetch.Statement {
		return fetch.Statement{Kind: "registry", Source: "acme/httputil", Ref: "1.0.0", Hash: hash, Claim: claim, Time: "2026-10-07T12:00:00Z"}.Sign(key)
	}
	if err := st(aliceKey, p.Hash, "reviewed").Verify(); err != nil {
		t.Fatal(err)
	}

	// an untrusted key, a review of other bytes, another claim: none counts
	for name, s := range map[string]fetch.Statement{
		"untrusted key": st(malloryKey, p.Hash, "reviewed"),
		"other bytes":   st(aliceKey, "h1:"+strings.Repeat("0", 64), "reviewed"),
		"other claim":   st(aliceKey, p.Hash, "audited"),
	} {
		if _, err := fetch.AddAttestations(dir, "x", []fetch.Statement{s}); err != nil {
			t.Fatal(err)
		}
		if _, err := fetch.Resolve(dir, man); err == nil {
			t.Errorf("%s was counted", name)
		}
	}

	// a tampered statement is not accepted into the project
	bad := st(aliceKey, p.Hash, "reviewed")
	line := strings.Replace(bad.String(), " reviewed ", " audited ", 1)
	if _, errs := fetch.ParseStatements("x", []byte(line)); len(errs) != 1 || !strings.Contains(errs[0].Error(), "signature does not match") {
		t.Errorf("tampered: %v", errs)
	}

	// a trusted review committed in the project counts
	if n, err := fetch.AddAttestations(dir, "alice", []fetch.Statement{st(aliceKey, p.Hash, "reviewed")}); err != nil || n != 1 {
		t.Fatalf("add: %d %v", n, err)
	}
	if n, _ := fetch.AddAttestations(dir, "alice", []fetch.Statement{st(aliceKey, p.Hash, "reviewed")}); n != 0 {
		t.Error("the same statement was added twice")
	}
	if _, err := fetch.Resolve(dir, man); err != nil {
		t.Errorf("reviewed in the project: %v", err)
	}

	// …and so does one only the registry holds
	dir2, man2 := e.app("1.0.0", policy)
	if _, err := fetch.Resolve(dir2, man2); err == nil {
		t.Fatal("dir2 has no reviews yet")
	}
	if err := fetch.PushStatement(e.http.URL, st(aliceKey, p.Hash, "reviewed"), "tok-bob"); err != nil {
		t.Fatal(err)
	}
	if _, err := fetch.Resolve(dir2, man2); err != nil {
		t.Errorf("reviewed at the registry: %v", err)
	}
	// the registry refuses a review of other bytes, and a push without a token
	if err := fetch.PushStatement(e.http.URL, st(aliceKey, "h1:"+strings.Repeat("1", 64), "reviewed"), "tok-bob"); err == nil || !strings.Contains(err.Error(), "(422)") {
		t.Errorf("wrong bytes: %v", err)
	}
	if err := fetch.PushStatement(e.http.URL, st(aliceKey, p.Hash, "reviewed"), ""); err == nil || !strings.Contains(err.Error(), "needs a token") {
		t.Errorf("no token: %v", err)
	}
	// an allow exempts a package from the requirement
	dir3, man3 := e.app("1.0.0", "[policy]\ntrust = { alice = \""+alice+"\" }\nrequire = { reviewed = 1 }\nallow = { reviewed = [\"acme/httputil\"] }\n")
	if _, err := fetch.Resolve(dir3, man3); err != nil {
		t.Errorf("exempt: %v", err)
	}
}

func TestKeysAndStatementsRoundTrip(t *testing.T) {
	t.Setenv("VELES_HOME", t.TempDir())
	pub, path, err := fetch.GenerateKey("alice")
	if err != nil || !strings.HasPrefix(pub, "ed25519:") {
		t.Fatalf("%q %v", pub, err)
	}
	if _, _, err := fetch.GenerateKey("alice"); err == nil || !strings.Contains(err.Error(), "never overwritten") {
		t.Errorf("overwrite: %v", err)
	}
	if _, _, err := fetch.GenerateKey("../evil"); err == nil {
		t.Error("a key name with a path was accepted")
	}
	if info, err := os.Stat(path); err != nil || (info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/') {
		t.Errorf("key file mode: %v %v", info, err)
	}
	priv, err := fetch.LoadKey("alice")
	if err != nil {
		t.Fatal(err)
	}
	if fetch.EncodePublic(priv.Public().(ed25519.PublicKey)) != pub {
		t.Error("the stored key is not the one generated")
	}
	if _, err := fetch.LoadKey("nobody"); err == nil || !strings.Contains(err.Error(), "veles attest keygen nobody") {
		t.Errorf("missing key: %v", err)
	}
	s := fetch.Statement{Kind: "git", Source: "https://x/y", Ref: "commit:abc", Hash: "h1:00", Claim: "reviewed", Time: "2026-10-07T00:00:00Z"}.Sign(priv)
	back, err := fetch.ParseStatement(s.String())
	if err != nil || back != s {
		t.Errorf("round trip: %v %+v", err, back)
	}
	for _, bad := range []string{"", "attest v2 git a b c d e f g", "attest v1 svn a b h1:00 reviewed t k s"} {
		if _, err := fetch.ParseStatement(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
}

func TestInfoSignature(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	key := fetch.EncodePublic(priv.Public().(ed25519.PublicKey))
	i := fetch.SignInfo(priv, "acme/x", fetch.Info{Version: "1.0.0", Time: "t", Hash: "h1:00", Tier: "listed"})
	if err := fetch.VerifyInfo(key, "acme/x", i); err != nil {
		t.Fatal(err)
	}
	// every field the client acts on is covered
	for name, mutate := range map[string]func(*fetch.Info){
		"hash":   func(i *fetch.Info) { i.Hash = "h1:01" },
		"tier":   func(i *fetch.Info) { i.Tier = "unreviewed" },
		"yanked": func(i *fetch.Info) { i.Yanked = true },
		"time":   func(i *fetch.Info) { i.Time = "u" },
	} {
		c := i
		mutate(&c)
		if err := fetch.VerifyInfo(key, "acme/x", c); err == nil {
			t.Errorf("a changed %s verified", name)
		}
	}
	if err := fetch.VerifyInfo(key, "acme/other", i); err == nil {
		t.Error("metadata of another package verified")
	}
}

// Two registries may both have an acme/httputil 1.0.0 that are different
// packages; the module cache keeps them apart, so one registry's package is
// never served to a project that uses the other (dependency confusion).
func TestCacheIsPerRegistry(t *testing.T) {
	a := newEnv(t)
	b := newEnv(t) // a second server; newEnv sets the same variables, VELES_HOME is shared below
	home := t.TempDir()
	t.Setenv("VELES_HOME", home)
	a.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 1\n")
	b.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 2\n")
	read := func(e *env) string {
		dir, man := e.app("1.0.0", "")
		res, err := fetch.Resolve(dir, man)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(res.Dirs[sema.DirKey(dir)]["httputil"], "lib.vs"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if got := read(a); !strings.Contains(got, "=> 1") {
		t.Errorf("registry a served %q", got)
	}
	if got := read(b); !strings.Contains(got, "=> 2") {
		t.Errorf("registry b was served registry a's package from the cache: %q", got)
	}
}

// Metadata cached without a signature check is not believed by a project that
// pins the registry's key: the tier and the yank are asked again (online), or
// the package counts as unreviewed (offline).
func TestCachedTierNeedsTheSignature(t *testing.T) {
	e := newEnv(t)
	e.publish("acme/httputil", "1.0.0", "public fun get(): i64 => 1\n")
	policy := "[policy]\ndeny = [\"unlisted\"]\n"

	// a project with no pinned key fetches it: the cache remembers "listed", unsigned
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"veles.toml": "[package]\nname = \"loose\"\n[dependencies]\nhttputil = { registry = \"acme/httputil\", version = \"1.0.0\" }\n[registry]\nurl = \"" + e.http.URL + "\"\n",
		"main.vs":    "fun main() {}\n",
	})
	lman, _ := sema.ReadManifest(dir)
	if _, err := fetch.Resolve(dir, lman); err != nil {
		t.Fatal(err)
	}

	// a project that pins the key: offline it does not believe the cached tier
	pinned, pman := e.app("1.0.0", policy)
	fetch.Offline = true
	_, err := fetch.Resolve(pinned, pman)
	fetch.Offline = false
	if err == nil || !strings.Contains(err.Error(), "is unlisted") {
		t.Errorf("offline, unverified tier: %v", err)
	}
	// online it asks, verifies, and the package is listed
	if _, err := fetch.Resolve(pinned, pman); err != nil {
		t.Errorf("online: %v", err)
	}
	// and now that the cache holds a signed answer, offline works too
	fetch.Offline = true
	defer func() { fetch.Offline = false }()
	pinned2, pman2 := e.app("1.0.0", policy)
	if _, err := fetch.Resolve(pinned2, pman2); err != nil {
		t.Errorf("offline with a verified cache: %v", err)
	}
}

func TestServerRefusesOddNames(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/Acme/Lib/@v/list", "/acme/li b/@v/list", "/acme/../@v/list", "/acme/lib/@v/../x", "/acme/lib/other/list", "/"} {
		resp, err := http.Get(e.http.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode < 400 || resp.StatusCode >= 500 {
			t.Errorf("%s answered %d", path, resp.StatusCode)
		}
	}
	// a body is read before the registry takes its lock: a request with a body
	// that never finishes must not hold up another
	slow, fast := make(chan struct{}), make(chan error, 1)
	go func() {
		pr, pw := io.Pipe()
		req, _ := http.NewRequest("PUT", e.http.URL+"/acme/lib/@v/1.0.0.zip", pr)
		req.Header.Set("Authorization", "Bearer tok-acme")
		go func() { pw.Write([]byte("PK")); <-slow; pw.Close() }()
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
	go func() {
		resp, err := http.Get(e.http.URL + "/acme/lib/@v/list")
		if err == nil {
			resp.Body.Close()
		}
		fast <- err
	}()
	select {
	case err := <-fast:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(5 * time.Second):
		t.Error("a request waited behind an unfinished upload")
	}
	close(slow)
}
