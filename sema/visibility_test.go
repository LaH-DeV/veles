package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// writeTree writes a package's files and returns the rendered diagnostics
// of loading and checking it.
func checkTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, src := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(root, diags)
	if err != nil {
		return err.Error()
	}
	if !diags.HasErrors() {
		Check(pkg, diags, false)
	}
	return diags.Render()
}

// D142: unmarked is the package — another module of it sees the item —
// while `internal` stops at the module.
func TestPackageLevelBetweenModules(t *testing.T) {
	const manifest = "[package]\nname = \"app\"\nversion = \"0.1.0\"\n"
	shared := checkTree(t, map[string]string{
		"veles.toml": manifest,
		"geo/lib.vs": "struct Box {\n  w: i64\n  fun area(): i64 => this.w * this.w\n}\nfun make(): Box => Box(w: 2)\n",
		"main.vs":    "use io { println }\nuse geo\nfun main() {\n  val b = geo.make()\n  println(\"${b.w} ${b.area()} ${geo.Box(w: 3).area()}\")\n}\n",
	})
	if strings.Contains(shared, "error") {
		t.Errorf("an unmarked item should be visible in the package:\n%s", shared)
	}
	for use, want := range map[string]string{
		"fun main() {\n  val _ = geo.make()\n}\n": "'make' is private to module 'geo'",
		"fun f(b: geo.Box) { }\nfun main() { }\n": "'Box' is private to module 'geo'",
	} {
		hidden := checkTree(t, map[string]string{
			"veles.toml": manifest,
			"geo/lib.vs": "internal struct Box {\n  w: i64\n}\ninternal fun make(): i64 => 1\n",
			"main.vs":    "use geo\n" + use,
		})
		if !strings.Contains(hidden, want) {
			t.Errorf("missing %q in:\n%s", want, hidden)
		}
	}
}

// D142: `public` in a program guards nothing; the warning's fix removes it.
func TestPublicInProgramWarns(t *testing.T) {
	out := checkTree(t, map[string]string{
		"veles.toml": "[package]\nname = \"app\"\nversion = \"0.1.0\"\n",
		"main.vs":    "public struct P {\n  public x: i64\n  public fun get(): i64 => x\n}\nfun main() { }\n",
	})
	for _, name := range []string{"'P'", "'x'", "'get'"} {
		if !strings.Contains(out, "warning: 'public' on "+name+" does nothing in a program") {
			t.Errorf("no warning for %s in:\n%s", name, out)
		}
	}
	lib := checkTree(t, map[string]string{
		"veles.toml": "[package]\nname = \"lib\"\nversion = \"0.1.0\"\n",
		"lib.vs":     "public struct P {\n  public x: i64\n}\n",
	})
	if strings.Contains(lib, "does nothing") {
		t.Errorf("a library's public is its API:\n%s", lib)
	}
}

// D142 (Q23): a public signature names only types other packages can see.
func TestPublicSignatureNamesVisibleTypes(t *testing.T) {
	const manifest = "[package]\nname = \"lib\"\nversion = \"0.1.0\"\n"
	out := checkTree(t, map[string]string{
		"veles.toml": manifest,
		"lib.vs":     "struct Token {\n  text: string\n}\npublic fun parse(s: string): Token => Token(text: s)\npublic fun show(t: Token): string => t.text\npublic struct Holder {\n  public t: Token\n}\n",
	})
	for _, want := range []string{
		"'parse' is public but its result type 'Token' is not visible outside the package: make 'Token' public, or drop 'public' from 'parse'",
		"'show' is public but its parameter 't''s type",
		"field 't' is public but its type 'Token' is not visible outside the package",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// a facade's export counts as visible
	ok := checkTree(t, map[string]string{
		"veles.toml":     manifest,
		"lib.vs":         "public use lexer { Token }\npublic use lexer\n",
		"lexer/lexer.vs": "struct Token {\n  text: string\n}\npublic fun parse(s: string): Token => Token(text: s)\n",
	})
	if strings.Contains(ok, "not visible outside") {
		t.Errorf("a type a facade exports is visible:\n%s", ok)
	}
}
