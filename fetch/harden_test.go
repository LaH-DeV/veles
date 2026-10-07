package fetch

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

func TestGitAddressesAreChecked(t *testing.T) {
	t.Setenv(AllowLocalGit, "")
	for _, c := range []struct {
		url         string
		fromPackage bool
		want        string // "" = accepted
	}{
		{"https://github.com/veles-db/pg", false, ""},
		{"https://github.com/veles-db/pg", true, ""},
		{"ssh://git@host/x/y.git", true, ""},
		{"git://host/x", true, ""},
		{"git@github.com:acme/x", true, ""},
		{"git@github.com:acme/x", false, ""},
		{"/srv/git/mine", false, ""},
		{"C:/repos/mine", false, ""},
		{"file:///srv/git/mine", false, ""},
		{"http://host/x", false, ""},
		{"", false, "is empty"},
		{"--upload-pack=touch /tmp/x", false, "space or control"},
		{"--upload-pack=x", false, "starts with '-'"},
		{"-x", true, "starts with '-'"},
		{"ext::sh -c touch% /tmp/x", false, "space or control"},
		{"ext::sh", false, "transport helper"},
		{"fd::3", true, "transport helper"},
		{"ftp://host/x", false, "not a repository scheme"},
		{"https://", false, "nothing after"},
		{"https://host/x\nfoo", false, "control character"},
		{"file:///srv/git/mine", true, "may use https, ssh or git"},
		{"http://host/x", true, "may use https, ssh or git"},
		{"/srv/git/mine", true, "a local path, and a fetched package's manifest may not name one"},
		{"../sibling", true, "a local path"},
	} {
		err := checkGitURL(c.url, c.fromPackage)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%q (package=%v) refused: %v", c.url, c.fromPackage, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%q (package=%v): got %v, want %q", c.url, c.fromPackage, err, c.want)
		}
	}
	// the escape hatch for a private mirror on disk
	t.Setenv(AllowLocalGit, "1")
	if err := checkGitURL("/srv/git/mine", true); err != nil {
		t.Errorf("with %s: %v", AllowLocalGit, err)
	}
	if err := checkGitURL("--x", true); err == nil {
		t.Error("the escape hatch must not allow an option")
	}
}

// A fetched package cannot send the reader's git to a path on the reader's disk.
func TestFetchedPackageCannotNameALocalRepository(t *testing.T) {
	home(t)
	t.Setenv(AllowLocalGit, "") // home() set it for the other tests
	inner := newRepo(t)
	inner.release("1.0.0", pkgFiles("inner", "1.0.0", "", "a\n"))
	outer := newRepo(t)
	outer.release("1.0.0", pkgFiles("outer", "1.0.0", "inner = "+gitDep(inner, "1.0.0"), "b\n"))
	dir, man := project(t, "outer = "+gitDep(outer, "1.0.0"))
	// the project's own manifest may name a local repository (outer is one); outer's may not
	_, err := Resolve(dir, man)
	if err == nil || !strings.Contains(err.Error(), "a local path, and a fetched package's manifest may not name one") {
		t.Errorf("got %v", err)
	}
}

func zipWith(t *testing.T, names ...string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, n := range names {
		f, err := w.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte("x"))
	}
	w.Close()
	p := filepath.Join(t.TempDir(), "a.zip")
	os.WriteFile(p, buf.Bytes(), 0o644)
	return p
}

func TestArchivePathsAreChecked(t *testing.T) {
	for _, bad := range [][]string{
		{"lib.vs", "a:stream"},           // an NTFS alternate data stream, or a drive
		{".git/config"},                  // not hashed, so it could hold anything
		{"sub/.GIT/hooks/post-checkout"}, // in any case
		{"a/./b.vs"},
		{"a//b.vs"},
		{"../x"},
		{"/abs"},
		{"a\\b.vs"},
	} {
		tmp := t.TempDir()
		if _, err := unzip(tmp, zipWith(t, bad...)); err == nil || !strings.Contains(err.Error(), "unsafe path") {
			t.Errorf("%q: got %v", bad, err)
		}
		if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
			t.Errorf("%q: left %v behind", bad, entries)
		}
	}
	if _, err := unzip(t.TempDir(), zipWith(t, "A.vs", "a.vs")); err == nil || !strings.Contains(err.Error(), "differ only in case") {
		t.Errorf("case twins: %v", err)
	}
	// a name that merely starts like .git is fine
	if _, err := unzip(t.TempDir(), zipWith(t, ".gitattributes", "lib.vs", "dir/.github/x")); err != nil {
		t.Errorf("harmless names: %v", err)
	}
}

func TestStatementFieldsHaveNoWhitespace(t *testing.T) {
	_, priv := testKey(t)
	s := Statement{Kind: "git", Source: "https://x/y z", Ref: "1.0.0", Hash: "h1:00", Claim: "reviewed", Time: "t"}.Sign(priv)
	if err := s.Verify(); err == nil || !strings.Contains(err.Error(), "no whitespace") {
		t.Errorf("source with a space: %v", err)
	}
	s = Statement{Kind: "git", Source: "https://x/y", Ref: "1.0.0", Hash: "h1:00", Claim: "reviewed"}.Sign(priv)
	if err := s.Verify(); err == nil {
		t.Error("a statement without a time verified")
	}
}

func TestTokensGoOverHTTPSOrLocalhost(t *testing.T) {
	for url, ok := range map[string]bool{
		"https://registry.example.com": true,
		"http://localhost:8080":        true,
		"http://127.0.0.1:9000":        true,
		"http://[::1]:9000":            true,
		"http://registry.example.com":  false,
		"http://10.1.2.3":              false,
		"ftp://registry.example.com":   false,
	} {
		err := checkTokenTransport(url, "secret")
		if ok != (err == nil) {
			t.Errorf("%s: %v", url, err)
		}
		if err := checkTokenTransport(url, ""); err != nil {
			t.Errorf("%s: no token to protect, yet %v", url, err)
		}
	}
	if _, _, err := Publish(t.TempDir(), &sema.Manifest{Name: "p", Version: "1.0.0", RegistryName: "a/b"}, "http://registry.example.com", "tok"); err == nil || !strings.Contains(err.Error(), "refusing to send a token") {
		t.Errorf("publish over http: %v", err)
	}
}

func testKey(t *testing.T) (string, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return EncodePublic(pub), priv
}

// In a git repository `veles publish` uploads what git would: a compiled
// program or scratch file the repository's .gitignore keeps out stays out.
func TestZipPackageHonoursGitignore(t *testing.T) {
	needGit(t)
	dir := t.TempDir()
	run(t, dir, "git", "init", "-q")
	writeFiles(t, dir, map[string]string{
		"veles.toml":   "[package]\nname = \"p\"\n",
		"lib.vs":       "a\n",
		".gitignore":   "*.exe\nscratch/\n",
		"p.exe":        "binary",
		"scratch/x":    "tmp",
		"untracked.vs": "new file, not ignored\n",
	})
	run(t, dir, "git", "add", "veles.toml", "lib.vs", ".gitignore")
	data, err := ZipPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	zp := filepath.Join(t.TempDir(), "a.zip")
	os.WriteFile(zp, data, 0o644)
	r, err := zip.OpenReader(zp)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	if got := strings.Join(names, " "); got != "lib.vs untracked.vs veles.toml" {
		t.Errorf("archive holds %q (hidden files, ignored files and the binary stay out)", got)
	}
	// outside a repository the directory is archived as it is, minus the usual exclusions
	plain := t.TempDir()
	writeFiles(t, plain, map[string]string{"veles.toml": "x", "lib.vs": "y", "p.exe": "z", "vendor/a/lib.vs": "v", "veles.sum": "s", ".hidden": "h"})
	data, _ = ZipPackage(plain)
	os.WriteFile(zp, data, 0o644)
	r2, _ := zip.OpenReader(zp)
	defer r2.Close()
	names = nil
	for _, f := range r2.File {
		names = append(names, f.Name)
	}
	if got := strings.Join(names, " "); got != "lib.vs p.exe veles.toml" {
		t.Errorf("plain directory archive holds %q", got)
	}
}
