package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// D89: `public use` in the root module is what a package shows other
// packages — whole modules (`public use geometry`, under an alias if asked)
// and flattened items (`public use shapes { area, Circle as Round }`);
// nothing else escapes, a module can pass on what it re-exports, and only the
// package's own modules can be re-exported.
func TestPublicUse(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lib/veles.toml", "[package]\nname = \"mathlib\"\nversion = \"0.1.0\"\n")
	write("lib/lib.vs", "public use geometry\npublic use shapes { area, Circle as Round }\npublic use util as tools\n")
	write("lib/geometry/lib.vs", "public use deep\npublic fun twice(n: i64): i64 => n * 2\n")
	write("lib/deep/lib.vs", "public fun depth(): i64 => 3\n")
	write("lib/shapes/lib.vs", "public struct Circle {\n  public r: f64\n}\npublic fun area(c: Circle): f64 => c.r * c.r * 3.0\n")
	write("lib/util/lib.vs", "public fun one(): i64 => 1\n")
	write("lib/hidden/lib.vs", "public fun secret(): i64 => 7\n")
	write("app/veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n\n[dependencies]\nmathlib = \"../lib\"\n")

	check := func(dir string) string {
		t.Helper()
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(filepath.Join(root, dir), diags)
		if err != nil {
			return err.Error()
		}
		if !diags.HasErrors() {
			Check(pkg, diags, false)
		}
		return diags.Render()
	}
	app := func(src string) string {
		t.Helper()
		write("app/main.vs", src)
		return check("app")
	}

	// what the root re-exports is reachable: a module, a module under its
	// alias, flattened items (renamed one too), through `use` and qualified
	good := "use io { println }\nuse mathlib { area, Round }, mathlib.geometry, mathlib.tools\n\n" +
		"fun main() {\n  println(\"${geometry.twice(2)} ${tools.one()} ${area(Round(r: 1.0))} ${mathlib.area(mathlib.Round(r: 2.0))}\")\n}\n"
	if out := app(good); strings.Contains(out, "error") {
		t.Errorf("a re-exported surface should check:\n%s", out)
	}
	// a module the root does not re-export
	if out := app("use mathlib.hidden\nfun main() {}\n"); !strings.Contains(out, "module 'hidden' of package 'mathlib' is not re-exported; add 'public use hidden' to its root module") {
		t.Errorf("hidden module:\n%s", out)
	}
	// a module whose items are flattened is not itself reachable
	if out := app("use mathlib.shapes\nfun main() {}\n"); !strings.Contains(out, "module 'shapes' of package 'mathlib' is not re-exported") {
		t.Errorf("flattened module:\n%s", out)
	}
	// a name only the alias carries
	if out := app("use mathlib.util\nfun main() {}\n"); !strings.Contains(out, "module 'util' of package 'mathlib' is not re-exported") {
		t.Errorf("aliased module under its own name:\n%s", out)
	}
	// a module passes on what it re-exports: `geometry` shows `deep`
	if out := app("use io { println }\nuse mathlib.geometry.deep\nfun main() { println(\"${deep.depth()}\") }\n"); strings.Contains(out, "error") {
		t.Errorf("a chain of re-exports:\n%s", out)
	}
	if out := app("use mathlib.deep\nfun main() {}\n"); !strings.Contains(out, "module 'deep' of package 'mathlib' is not re-exported; add 'public use deep' to its root module") {
		t.Errorf("a module only a sub-module re-exports, from the root path:\n%s", out)
	}
	// hover and definition follow a flattened item to its declaration: the
	// module scope holds the module's own symbol
	write("app/main.vs", good)
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(filepath.Join(root, "app"), diags)
	if err != nil {
		t.Fatal(err)
	}
	Check(pkg, diags, false)
	lib := pkg.Modules["dep/mathlib"]
	if lib == nil {
		t.Fatalf("no dependency root module among %v", len(pkg.Modules))
	}
	if sym := lib.Scope.LookupLocal("area"); sym == nil || sym.Module == nil || sym.Module.Path != "dep/mathlib/shapes" {
		t.Errorf("area in the root's scope is %+v, want the shapes module's symbol", sym)
	}
	if sym := lib.Scope.LookupLocal("Round"); sym == nil || sym.Name != "Round" {
		t.Errorf("Round in the root's scope is %+v", sym)
	}

	// only the package's own modules: std and dependencies are refused
	write("solo/veles.toml", "[package]\nname = \"solo\"\nversion = \"0.1.0\"\n")
	write("solo/lib.vs", "public use io\n")
	if out := check("solo"); !strings.Contains(out, "'public use io' re-exports a module of another package") {
		t.Errorf("re-exporting std:\n%s", out)
	}
	write("solo/lib.vs", "public use mathlib\n")
	write("solo/veles.toml", "[package]\nname = \"solo\"\nversion = \"0.1.0\"\n\n[dependencies]\nmathlib = \"../lib\"\n")
	if out := check("solo"); !strings.Contains(out, "'public use mathlib' re-exports a module of another package") {
		t.Errorf("re-exporting a dependency:\n%s", out)
	}

	// a re-exported name may not collide with a declaration of the module
	write("clash/veles.toml", "[package]\nname = \"clash\"\nversion = \"0.1.0\"\n")
	write("clash/lib.vs", "public use geometry { twice }\npublic fun twice(n: i64): i64 => n\n")
	write("clash/geometry/lib.vs", "public fun twice(n: i64): i64 => n * 2\n")
	if out := check("clash"); !strings.Contains(out, "'twice' is already declared in this module") || !strings.Contains(out, "re-export it under another name, 'public use geometry { twice as … }'") {
		t.Errorf("colliding item:\n%s", out)
	}
	write("clash/lib.vs", "public use geometry\npublic fun geometry(): i64 => 1\n")
	if out := check("clash"); !strings.Contains(out, "'geometry' is already declared in this module") || !strings.Contains(out, "re-export the module under another name, 'public use geometry as …'") {
		t.Errorf("colliding module:\n%s", out)
	}

	// the manifest's `exports` is gone, and the error says what replaces it
	write("old/veles.toml", "[package]\nname = \"old\"\nversion = \"0.1.0\"\nexports = [\"geometry\"]\n")
	write("old/lib.vs", "public fun f(): i64 => 1\n")
	if out := check("old"); !strings.Contains(out, "'exports' was removed (D89)") || !strings.Contains(out, "public use geometry") {
		t.Errorf("manifest exports:\n%s", out)
	}
}
