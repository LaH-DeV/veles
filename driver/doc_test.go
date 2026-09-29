package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `veles doc` writes one Markdown file per module a reader outside the
// package can reach: the declaration as hover spells it, its `///`
// comment, its documented members — and nothing private, and no module
// the root module does not re-export (D89).
func TestDoc(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(src), 0o644)
	}
	write("veles.toml", "[package]\nname = \"shapes\"\nversion = \"0.1.0\"\n")
	write("lib.vs", "/// Shapes.\n\n/// A circle.\npublic struct Circle {\n  /// Edge to centre.\n  public radius: f64\n  private cache: i64 = 0\n}\n\nfun helper(): i64 = 1\n\npublic use geo\n")
	write("geo/geo.vs", "/// Twice `x`.\npublic fun twice(x: i64): i64 = x * 2\n")
	write("internal/x.vs", "public fun hidden(): i64 = 1\n")
	out := filepath.Join(root, "api")
	if code := Doc(root, out); code != 0 {
		t.Fatalf("doc: exit %d", code)
	}
	lib, _ := os.ReadFile(filepath.Join(out, "shapes.md"))
	geo, _ := os.ReadFile(filepath.Join(out, "geo.md"))
	for _, want := range []string{"# Module `shapes`", "Shapes.", "## `struct Circle`", "public val radius: f64", "A circle.", "- `radius` — Edge to centre."} {
		if !strings.Contains(string(lib), want) {
			t.Errorf("shapes.md lacks %q:\n%s", want, lib)
		}
	}
	if strings.Contains(string(lib), "cache") || strings.Contains(string(lib), "helper") || strings.Contains(string(lib), "not visible") {
		t.Errorf("shapes.md shows what an outsider cannot use:\n%s", lib)
	}
	if !strings.Contains(string(geo), "public fun twice(x: i64): i64") || !strings.Contains(string(geo), "Twice `x`.") {
		t.Errorf("geo.md:\n%s", geo)
	}
	if _, err := os.Stat(filepath.Join(out, "internal.md")); err == nil {
		t.Errorf("a module the manifest does not export was documented")
	}
}
