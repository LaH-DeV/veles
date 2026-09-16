package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// A package is the program: `main` lives in the root module, and pointing
// the tools at a sub-module still loads the package (M1).
func TestPackageIsTheProgram(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("main.vs", "use io\nuse geometry\nfun main() { io.println(\"${geometry.twice(2)}\") }\n")
	write("geometry/lib.vs", "pub fun twice(n: i64): i64 = n * 2\n")
	write("unused/lib.vs", "pub fun broken(): i64 = \"not a number\"\n")

	load := func(path string, needMain bool) (*Package, *source.Diagnostics) {
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(path, diags)
		if err != nil {
			t.Fatal(err)
		}
		pkg.NeedMain = needMain
		if !diags.HasErrors() {
			Check(pkg, diags, false)
		}
		return pkg, diags
	}
	// from a sub-module: the root module is the entry and its main is found
	pkg, diags := load(filepath.Join(root, "geometry"), true)
	if diags.HasErrors() {
		t.Errorf("run from a sub-module: %s", diags.Render())
	}
	if pkg.Entry == nil || pkg.Entry.Path != "" || pkg.Given == nil || pkg.Given.Path != "geometry" {
		t.Errorf("entry/given = %v / %v", pkg.Entry, pkg.Given)
	}
	// a module nothing imports is still loaded and checked when pointed at
	_, diags = load(filepath.Join(root, "unused"), false)
	if !strings.Contains(diags.Render(), "type mismatch") {
		t.Errorf("unimported module not checked: %s", diags.Render())
	}
	// a library root: check passes, build explains
	write("main.vs", "use io\nfun helper() { io.println(\"x\") }\n")
	if _, diags = load(root, false); diags.HasErrors() {
		t.Errorf("check of a library: %s", diags.Render())
	}
	if _, diags = load(root, true); !strings.Contains(diags.Render(), "can be used as a library but not run") {
		t.Errorf("build of a library: %s", diags.Render())
	}
	if _, diags = load(filepath.Join(root, "geometry"), true); !strings.Contains(diags.Render(), "is a module of the package") {
		t.Errorf("build from a module of a library: %s", diags.Render())
	}
}
