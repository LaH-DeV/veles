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
	write("geometry/lib.vs", "public fun twice(n: i64): i64 = n * 2\n")
	write("unused/lib.vs", "public fun broken(): i64 = \"not a number\"\n")

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

// A script (`.vss`) is a one-file package: its siblings are not part of it,
// standard modules are importable, and local directories are not.
func TestScriptIsItsOwnPackage(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("main.vs", "use io\nfun main() { io.println(\"module\") }\n")
	write("one.vss", "use io\nfun main() { io.println(\"one\") }\n")
	write("two.vss", "use io, os\nfun main() { io.println(\"two ${os.args().len()}\") }\n")
	write("local.vss", "use io, geometry\nfun main() { io.println(\"${geometry.twice(2)}\") }\n")
	write("nomain.vss", "fun helper(): i64 = 1\n")
	write("geometry/lib.vs", "public fun twice(n: i64): i64 = n * 2\n")

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
	// the directory package still sees only .vs files
	pkg, diags := load(root, true)
	if diags.HasErrors() {
		t.Errorf("directory package next to scripts: %s", diags.Render())
	}
	if n := len(pkg.Entry.Files); n != 1 {
		t.Errorf("root module has %d files, want 1", n)
	}
	// each script is a program of its own, despite the sibling mains
	for _, name := range []string{"one.vss", "two.vss"} {
		pkg, diags := load(filepath.Join(root, name), true)
		if diags.HasErrors() {
			t.Errorf("%s: %s", name, diags.Render())
		}
		if pkg.Script == "" || pkg.Entry != pkg.Given || len(pkg.Entry.Files) != 1 {
			t.Errorf("%s: script=%q entry=%v given=%v", name, pkg.Script, pkg.Entry, pkg.Given)
		}
		if pkg.Manifest != nil {
			t.Errorf("%s: a script has no manifest", name)
		}
	}
	// a script has no modules of its own
	if _, diags := load(filepath.Join(root, "local.vss"), true); !strings.Contains(diags.Render(), "a script imports only standard modules") {
		t.Errorf("local import from a script: %s", diags.Render())
	}
	// and it must have a main to run
	if _, diags := load(filepath.Join(root, "nomain.vss"), true); !strings.Contains(diags.Render(), "script") || !strings.Contains(diags.Render(), "no 'fun main()'") {
		t.Errorf("script without main: %s", diags.Render())
	}
	if _, diags := load(filepath.Join(root, "nomain.vss"), false); diags.HasErrors() {
		t.Errorf("check of a script without main: %s", diags.Render())
	}
}
