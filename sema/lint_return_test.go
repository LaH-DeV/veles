package sema

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// `[lint] implicit_return` decides where a `{ }` body may end in its value:
// "full" (the default) everywhere, "lambda" only in lambdas, "expr" nowhere.
// Each body that breaks the setting is one error with a fix that writes
// `return`; bodies without braces and `if`/`when` blocks are never touched.
func TestImplicitReturnLint(t *testing.T) {
	const src = `use io

struct Point {
  x: i64

  fun double(): i64 {
    this.x * 2
  }
}

fun pick(c: bool): string {
  if (c) "yes" else "no"
}

fun short(n: i64): i64 => n + 1

fun already(n: i64): i64 {
  return n
}

fun fails(): i64 {
  panic("never returns")
}

fun nothing() {
  io.println("a statement")
}

fun main() {
  val f = (n: i64): i64 => {
    val m = n * 3
    m + 1
  }
  val g = (n: i64) => n - 1
  val label = when (short(1)) {
    2 => {
      io.println("two")
      "two"
    }
    else => "other"
  }
  nothing()
  io.println("${Point(x: 2).double()} ${pick(true)} ${f(1)} ${g(1)} ${already(3)} $label")
}
`
	check := func(lint string) *source.Diagnostics {
		root := t.TempDir()
		write := func(name, text string) {
			if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		write("veles.toml", "[package]\nname = \"app\"\n"+lint)
		write("main.vs", src)
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(root, diags)
		if err != nil {
			t.Fatal(err)
		}
		Check(pkg, diags, false)
		return diags
	}

	// the lines reported under each setting; every report carries a fix that
	// writes `return ` before the value
	reported := func(setting string) []int {
		diags := check("\n[lint]\nimplicit_return = \"" + setting + "\"\n")
		var lines []int
		for _, d := range diags.Items {
			if !strings.Contains(d.Message, "[lint] implicit_return = \""+setting+"\"") {
				t.Errorf("%s: unexpected: %s", setting, d)
				continue
			}
			if d.Fix == nil || len(d.Fix.Edits) != 1 || d.Fix.Edits[0].NewText != "return " || d.Fix.Edits[0].Span.Start != d.Span.Start {
				t.Errorf("%s: the fix does not insert 'return ' before the value: %+v", d, d.Fix)
			}
			line, _ := d.Span.File.Position(d.Span.Start)
			lines = append(lines, line)
		}
		sort.Ints(lines)
		return lines
	}

	if diags := check(""); diags.HasErrors() {
		t.Fatalf("without a setting (\"full\"):\n%s", diags.Render())
	}
	if got := reported("full"); len(got) != 0 {
		t.Errorf("\"full\" reports %v", got)
	}
	// "lambda": the method and the `if` body; the braced lambda keeps its value
	if got, want := reported("lambda"), []int{7, 12}; !slices.Equal(got, want) {
		t.Errorf("\"lambda\" reports %v, want %v", got, want)
	}
	// "expr": the braced lambda too; never the `=> expr` body, the written
	// return, the panic, the unit function, the lambda without braces or the
	// `when` arm's block
	if got, want := reported("expr"), []int{7, 12, 32}; !slices.Equal(got, want) {
		t.Errorf("\"expr\" reports %v, want %v", got, want)
	}
}

func TestManifestLintAndImportsKeys(t *testing.T) {
	m := parseOK(t, "[package]\nname = \"p\"\n\n[format]\nimports = \"merged\"\n\n[lint]\nimplicit_return = \"lambda\"\n")
	if !m.Format.MergeImports || m.Lint.ImplicitReturn != ImplicitLambda {
		t.Errorf("format %+v, lint %+v", m.Format, m.Lint)
	}
	if m := parseOK(t, "[package]\nname = \"p\"\n"); m.Format.MergeImports || m.Lint.ImplicitReturn != ImplicitFull {
		t.Errorf("defaults: format %+v, lint %+v", m.Format, m.Lint)
	}
	pkg := "[package]\nname = \"p\"\n"
	for _, c := range []struct{ text, want string }{
		{pkg + "[format]\nimports = \"grouped\"\n", "[format] imports is \"lines\""},
		{pkg + "[lint]\nimplicit_return = \"none\"\n", "[lint] implicit_return is \"full\""},
		{pkg + "[lint]\nimplicit_return = false\n", "[lint] implicit_return is \"full\""},
		{pkg + "[lint]\nexplicit_return = true\n", "unknown [lint] key \"explicit_return\" (implicit_return)"},
	} {
		if _, err := parseManifest("veles.toml", ".", c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q:\n got %v\nwant %q", c.text, err, c.want)
		}
	}
}
