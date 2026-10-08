package sema

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// ReadManifestIn answers for exactly the directory it is given. ReadManifest
// walks up to the enclosing package, which is right for "which package is this
// file in" and wrong for "what is this dependency": a dependency directory
// without a manifest must not be taken for the package around it.
func TestReadManifestInDoesNotWalkUp(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "veles.toml"), []byte("[package]\nname = \"outer\"\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "inner"), 0o755)
	m, err := ReadManifestIn(filepath.Join(root, "inner"))
	if err != nil || m != nil {
		t.Errorf("a directory with no manifest: %v %v", m, err)
	}
	m, err = ReadManifestIn(root)
	if err != nil || m == nil || m.Name != "outer" {
		t.Errorf("the package itself: %v %v", m, err)
	}
	if up, _ := ReadManifest(filepath.Join(root, "inner")); up == nil || up.Name != "outer" {
		t.Error("ReadManifest is expected to find the enclosing package")
	}
}

// A failed resolution is one error, reported once; the rest of the package is
// still loaded (so an editor keeps its diagnostics), and an import of a remote
// dependency is not reported again as if it were a mistake of its own.
func TestFailedResolutionIsReportedOnce(t *testing.T) {
	saved := ResolveRemote
	defer func() { ResolveRemote = saved }()
	ResolveRemote = func(root string, man *Manifest) (*Remote, error) {
		return nil, errors.New("the registry is down")
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "veles.toml"), []byte("[package]\nname = \"app\"\n[dependencies]\nhttputil = { registry = \"acme/httputil\", version = \"1.4.2\" }\n"), 0o644)
	os.WriteFile(filepath.Join(root, "main.vs"), []byte("use io\nuse httputil\nfun main() { io.println(\"x\") }\nfun broken(): i64 => \"not a number\"\n"), 0o644)
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(root, diags)
	if err != nil {
		t.Fatalf("LoadPackage returned %v; resolution failures are diagnostics", err)
	}
	if pkg.Entry == nil {
		t.Fatal("the package was not loaded after the failure")
	}
	if out := diags.Render(); strings.Count(out, "the registry is down") != 1 || strings.Contains(out, "was not fetched") {
		t.Errorf("diagnostics:\n%s", out)
	}
}

func TestNativePkgConfigNameIsNotAnOption(t *testing.T) {
	_, err := parseManifest("veles.toml", ".", "[package]\nname = \"p\"\n[native]\npkg-config = [\"libpq\", \"--define-variable=prefix=/x\"]\n")
	if err == nil || !strings.Contains(err.Error(), "is a package name, not an option") {
		t.Errorf("got %v", err)
	}
}
