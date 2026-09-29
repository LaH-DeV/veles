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

// Everything a module keeps to itself, reached from another: each error
// says how to share it.
func TestPrivateAcrossModulesSaysHow(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("geo/lib.vs", "public struct Box {\n  public shown: i64 = 0\n  kept: i64 = 1\n  fun inner(): i64 = 1\n  static fun make(): Box = Box()\n  static val zero: i64 = 0\n}\npublic fun box(): Box = Box()\nfun helper(): i64 = 1\n")
	write("main.vs", "use geo\nfun main() {\n  val b = geo.box()\n  val _ = b.kept\n  val _ = b.inner()\n  val _ = geo.Box.make()\n  val _ = geo.Box.zero\n  val _ = geo.helper()\n}\n")
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(root, diags)
	if err != nil {
		t.Fatal(err)
	}
	Check(pkg, diags, false)
	got := diags.Render()
	for _, want := range []string{
		"field 'kept' of 'Box' is private to module 'geo'; declare it 'public' there to use it from here (M5)",
		"method 'inner' is private to module 'geo'; declare it 'public' there to use it from here (M5)",
		"static function 'make' is private to module 'geo'; declare it 'public' there to use it from here (M5)",
		"'Box.zero' is private to module 'geo'; declare it 'public' there to use it from here (M5)",
		"'helper' is private to module 'geo'; declare it 'public' there to use it from here (M5)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A function C calls is one symbol in the whole program (D69): two modules
// may not both export the same name, though Veles would keep them apart.
func TestExternCSymbolIsUnique(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("left/lib.vs", "extern \"C\" fun onEvent(n: i64) { }\n")
	write("right/lib.vs", "extern \"C\" fun onEvent(n: i64) { }\n")
	write("main.vs", "use left\nuse right\nfun main() { }\n")
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(root, diags)
	if err != nil {
		t.Fatal(err)
	}
	Check(pkg, diags, false)
	if got := diags.Render(); !strings.Contains(got, "'extern \"C\" fun onEvent' is already defined at") || !strings.Contains(got, "a C symbol is unique in a program") {
		t.Errorf("two modules exporting one C symbol:\n%s", got)
	}
}

// M5 v0.30 across modules: a `private` field with a default is left to it,
// one without a default is given by the call, and only an unmarked field
// keeps the implicit constructor inside its module.
func TestPrivateFieldsAndTheConstructorAcrossModules(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("geo/lib.vs", "public struct Acc {\n  public name: string\n  private items: MutableList<i64> = []\n  private seed: i64\n  public fun count(): i64 = this.items.len() + this.seed\n}\npublic struct Tied {\n  public name: string\n  hidden: i64 = 3\n}\n")
	check := func(main string) string {
		write("main.vs", main)
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(root, diags)
		if err != nil {
			t.Fatal(err)
		}
		if !diags.HasErrors() {
			Check(pkg, diags, false)
		}
		return diags.Render()
	}
	if got := check("use io\nuse geo\nfun main() { val a = geo.Acc(name: \"x\", seed: 2)\n  io.println(\"${a.count()}\") }\n"); strings.Contains(got, "error") {
		t.Errorf("private fields across modules: %s", got)
	}
	if got := check("use geo\nfun main() { val _ = geo.Acc(name: \"x\", seed: 2, items: mut [1]) }\n"); !strings.Contains(got, "field 'items' is private to 'Acc' and has a default") {
		t.Errorf("a private field with a default was settable: %s", got)
	}
	if got := check("use geo\nfun main() { val _ = geo.Tied(name: \"x\") }\n"); !strings.Contains(got, "field 'hidden' belongs to module 'geo'") || !strings.Contains(got, "mark the field 'private'") {
		t.Errorf("an unmarked field: %s", got)
	}
}

// A braced import (D85) binds the module's own symbols: a type, a function
// and an enum written bare mean what the qualified names mean; a private
// name and test code are refused; an unused name is a warning whose fix
// removes it; a local of the same name shadows.
func TestNamedImports(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	check := func(main string) string {
		write("main.vs", main)
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(root, diags)
		if err != nil {
			t.Fatal(err)
		}
		Check(pkg, diags, false)
		return diags.Render()
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("geo/lib.vs", "public struct Point {\n  public x: i64\n  public y: i64\n}\npublic enum Kind { Flat, Round }\npublic fun norm(p: Point): i64 = p.x + p.y\npublic test fun probe(): i64 = 1\nfun secret(): i64 = 1\n")

	ok := check("use geo { Point, Kind, norm as length }\n\nfun main() {\n  val p = Point(x: 1, y: 2)\n  val k: geo.Kind = Kind.Flat\n  val n: i64 = length(p) + geo.norm(p)\n  val shadow = 1\n  val length = shadow\n}\n")
	if strings.Contains(ok, "error") {
		t.Errorf("a braced import should check clean, got:\n%s", ok)
	}
	got := check("use geo { Point, secret, probe, missing }\n\nfun main() {\n  val _ = Point(x: 1, y: 2)\n}\n")
	for _, want := range []string{
		"'secret' is private to module 'geo'; declare it 'public' there to use it from here (M5)",
		"'probe' is test code",
		"module 'geo' has no declaration 'missing'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	unused := check("use geo { Point, Kind }\n\nfun main() {\n  val _ = Point(x: 1, y: 2)\n}\n")
	if !strings.Contains(unused, "'Kind' is imported but never used") {
		t.Errorf("an unused name should warn, got:\n%s", unused)
	}
}
