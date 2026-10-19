package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// checkExtTree checks a package written as path → source (D173 needs more than
// one module). "main.vs" is the root module.
func checkExtTree(t *testing.T, files map[string]string) *source.Diagnostics {
	t.Helper()
	dir := t.TempDir()
	files["veles.toml"] = "[package]\nname = \"app\"\nversion = \"0.1.0\"\n"
	for path, src := range files {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	if !diags.HasErrors() {
		Check(pkg, diags, false)
	}
	return diags
}

func wantDiag(t *testing.T, diags *source.Diagnostics, sev source.Severity, want string) {
	t.Helper()
	for _, d := range diags.Items {
		if d.Severity == sev && strings.Contains(d.Message, want) {
			return
		}
	}
	t.Errorf("expected a diagnostic containing %q, got:\n%s", want, diags.Render())
}

// D173: `extend T with fun …` and `extend T { }` on a type the package does
// not own are extension functions, visible where their module is imported.
func TestExtensionFunctions(t *testing.T) {
	const strx = `extend string with fun shout(): string => this.toUpper() + "!"

public extend string {
  fun slug(): string => this.toLower().replace(" ", "-")
}

extend<T: Comparable> List<T> with fun isSorted(): bool => this.sorted() == this

extend i64 with fun clamped(lo: i64, hi: i64): i64 => if (this < lo) lo else if (this > hi) hi else this
`
	t.Run("imported module's extensions chain", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"strx/strx.vs": strx,
			"main.vs": `use io
use strx extend
fun main() {
  io.println(" Hi There ".trim().slug().shout())
  io.println("${[1, 2].isSorted()} ${150.clamped(0, 100)}")
}`,
		})
		if diags.HasErrors() {
			t.Errorf("unexpected errors:\n%s", diags.Render())
		}
	})
	t.Run("not imported", func(t *testing.T) {
		// the scope is the file: a sibling file that imports the module does not help
		diags := checkExtTree(t, map[string]string{
			"strx/strx.vs": strx,
			"other.vs":     "use strx extend\nfun g(): string => \"a\".shout()",
			"main.vs":      "fun f(): string => \"a\".shout()",
		})
		wantDiag(t, diags, source.Error, "'shout' is an extension function in module 'strx': import it with 'use strx extend { shout }'")
	})
	t.Run("a use brings no extension unless it says extend (D185)", func(t *testing.T) {
		withHello := strx + "\npublic fun hello(): string => \"hello\"\n"
		cases := []struct{ name, use, want string }{
			{"plain use", "use strx", "'shout' is an extension function in module 'strx'"},
			{"braced names", "use strx { hello }", "'shout' is an extension function in module 'strx'"},
			{"extend names another", "use strx { hello } extend { slug }", "'shout' is an extension function in module 'strx'"},
			{"extend names it", "use strx { hello } extend { shout }", ""},
			{"extend alone is all", "use strx extend", ""},
			{"braced and extend alone", "use strx { hello } extend", ""},
			{"extend names without braced names", "use strx extend { shout, slug }", ""},
			{"extend an unknown name", "use strx extend { shoot }", "module 'strx' has no extension function 'shoot'; did you mean 'shout'?"},
			{"extend of a module with none", "use plain extend", "module 'plain' has no extension functions: drop 'extend'"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				diags := checkExtTree(t, map[string]string{
					"strx/strx.vs":   withHello,
					"plain/plain.vs": "public fun nothing(): i64 => 1",
					"main.vs":        tc.use + "\nfun f(): string => \"a\".shout()",
				})
				if tc.want == "" {
					if diags.HasErrors() {
						t.Errorf("unexpected errors:\n%s", diags.Render())
					}
					return
				}
				wantDiag(t, diags, source.Error, tc.want)
			})
		}
	})
	t.Run("a std module's extensions on built-in types are scoped like any other", func(t *testing.T) {
		// importing regex's extensions in one module gives them to no other file
		diags := checkExtTree(t, map[string]string{
			"x/x.vs":  "use regex { Regex } extend\npublic fun make(): Regex => Regex.compile(\"a\") ?? panic(\"x\")",
			"main.vs": "use x\nfun g(): bool => \"a\".findAll(x.make()).isEmpty()",
		})
		wantDiag(t, diags, source.Error, "'findAll' is an extension function in module 'regex'")
	})
	t.Run("an inherent method wins and the extension warns", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"main.vs": "extend string with fun trim(): string => \"x\"\nfun main() { }",
		})
		wantDiag(t, diags, source.Warning, "extension 'trim' on 'string' is never called")
	})
	t.Run("two imported extensions of one name are ambiguous", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"a/a.vs":  "extend string with fun twice(): string => this + this",
			"b/b.vs":  "extend string with fun twice(): string => this + \"2\"",
			"main.vs": "use a extend\nuse b extend\nfun f(): string => \"a\".twice()",
		})
		wantDiag(t, diags, source.Error, "ambiguous method 'twice' on 'string'")
	})
	t.Run("two modules of one name are fine when only one is imported", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"a/a.vs":  "extend string with fun twice(): string => this + this",
			"b/b.vs":  "extend string with fun twice(): string => this + \"2\"",
			"main.vs": "use a extend\nfun f(): string => \"a\".twice()",
		})
		if diags.HasErrors() {
			t.Errorf("unexpected errors:\n%s", diags.Render())
		}
	})
	t.Run("a trait method of the same name is ambiguous", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"a/a.vs": "extend string with fun twice(): string => this + this",
			"main.vs": `use a extend
trait Twice { fun twice(): string }
implement Twice for string { fun twice(): string => "t" }
fun f(): string => "a".twice()`,
		})
		wantDiag(t, diags, source.Error, "provided by trait Twice and the extension in module 'a'")
	})
	t.Run("bounds are checked", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"a/a.vs": "extend<T: Comparable> List<T> with fun top(): i64 => this.len()",
			"main.vs": `use a extend
struct P { x: i64 }
fun f(): i64 => [P(x: 1)].top()`,
		})
		wantDiag(t, diags, source.Error, "requires 'P' to implement 'Comparable'")
	})
	t.Run("no static functions or private marker", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"main.vs": "extend string with static fun make(): string => \"\"\nextend string with private fun hid(): string => \"\"\nfun main() { }",
		})
		wantDiag(t, diags, source.Error, "cannot declare a static function")
		wantDiag(t, diags, source.Error, "'private' belongs to the members of a struct")
	})
	t.Run("not part of the interface", func(t *testing.T) {
		// an extension never satisfies a trait bound
		diags := checkExtTree(t, map[string]string{
			"a/a.vs": "extend string with fun shout(): string => this",
			"main.vs": `use a extend
trait Shouts { fun shout(): string }
fun needs<T: Shouts>(x: T): string => x.shout()
fun f(): string => needs("a")`,
		})
		wantDiag(t, diags, source.Error, "does not implement")
	})
	t.Run("a Result or Option cannot be extended", func(t *testing.T) {
		diags := checkExtTree(t, map[string]string{
			"main.vs": "extend<T, E> Result<T, E> with fun f(): i64 => 1\nfun main() { }",
		})
		wantDiag(t, diags, source.Error, "built in")
	})
}

// An extension sees a type the way any other package does (D173): the private
// fields of a std struct stay private to it.
func TestExtensionSeesPublicApiOnly(t *testing.T) {
	diags := checkExtTree(t, map[string]string{
		"main.vs": "extend Duration with fun nanosRaw(): i64 => this.ns\nfun main() { }",
	})
	wantDiag(t, diags, source.Error, "ns")
	diags = checkExtTree(t, map[string]string{
		"main.vs": "extend Duration with fun half(): Duration => this.dividedBy(2)\nfun main() { }",
	})
	if diags.HasErrors() {
		t.Errorf("unexpected errors:\n%s", diags.Render())
	}
}
