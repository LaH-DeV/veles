package parser

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
)

// D174: `annotation Name(params) on targets for Readers` is a struct whose
// fields are the parameters; `@mod.Name(args)` names an annotation through
// a module; `inline loop` is a loop that is unrolled while compiling.
func TestAnnotationDecl(t *testing.T) {
	cases := map[string]string{
		"annotation Min(value: i64) on field":                             "(annotation Min on field\n  (field value: i64))",
		"public annotation Hidden on field":                               "(annotation public Hidden on field)",
		"annotation Short(letter: string, long: string = \"\") on field":  "(annotation Short on field\n  (field letter: string)\n  (field long: string = (str \"\")))",
		"annotation Route(path: string) on fun, struct for Router, Other": "(annotation Route on fun struct for Router, Other\n  (field path: string))",
		"annotation Tag on enumMember, variant":                           "(annotation Tag on enumMember variant)",
	}
	for src, want := range cases {
		f, diags := parse(t, src+"\n")
		if diags.HasErrors() {
			t.Errorf("%q: %s", src, diags.Render())
			continue
		}
		if got := strings.TrimSpace(ast.Dump(f)); got != want {
			t.Errorf("%q:\n got %s\nwant %s", src, got, want)
		}
	}
}

func TestAnnotationDeclErrors(t *testing.T) {
	cases := map[string]string{
		"annotation Min(value) on field\n":          "needs a type",
		"annotation Min(value: i64)\n":              "says where it may stand",
		"annotation Min(values: i64...) on field\n": "plain 'name: T'",
		"annotation Min on 7\n":                     "where the annotation may stand",
	}
	for src, want := range cases {
		_, diags := parse(t, src)
		if !diags.HasErrors() || !strings.Contains(diags.Render(), want) {
			t.Errorf("%q: want an error containing %q, got:\n%s", src, want, diags.Render())
		}
	}
}

// `annotation` is a word only at declaration position: elsewhere it is a name.
func TestAnnotationIsContextual(t *testing.T) {
	f, diags := parse(t, "fun f(annotation: i64): i64 => annotation + 1\nval annotation = 2\n")
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	if len(f.Decls) != 2 {
		t.Fatalf("got %d declarations", len(f.Decls))
	}
}

// A declaration ending in a keyword (`on fun`) ends at the line: the lexer
// inserts no newline after a keyword.
func TestAnnotationEndsInKeyword(t *testing.T) {
	f, diags := parse(t, "annotation A on fun\nannotation B on struct\nfun f() { }\n")
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	if len(f.Decls) != 3 {
		t.Fatalf("got %d declarations, want 3", len(f.Decls))
	}
}

func TestQualifiedAnnotationUse(t *testing.T) {
	f, diags := parse(t, "struct S {\n  @validate.Min(1)\n  @Hidden\n  n: i64\n}\n")
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	sd := f.Decls[0].(*ast.StructDecl)
	attrs := sd.Fields[0].Attrs
	if len(attrs) != 2 || attrs[0].Qual.Name != "validate" || attrs[0].Name.Name != "Min" || attrs[1].Qual.Name != "" || attrs[1].Name.Name != "Hidden" {
		t.Fatalf("attributes: %+v %+v", attrs[0], attrs[1])
	}
}

func TestInlineLoop(t *testing.T) {
	f, diags := parse(t, "fun f() {\n  inline loop (x in T.fields) { }\n  loop (y in ys) { }\n  val inline = 1\n}\n")
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	dump := ast.Dump(f)
	if !strings.Contains(dump, "(inline-loop (x in") || !strings.Contains(dump, "(loop (y in") {
		t.Fatalf("dump:\n%s", dump)
	}
}

func TestSelfAsReceiver(t *testing.T) {
	f, diags := parse(t, "trait Z {\n  static fun zero(): This => This.default()\n  fun n(): i64 => This.fields.len()\n}\n")
	if diags.HasErrors() {
		t.Fatalf("%s", diags.Render())
	}
	dump := ast.Dump(f)
	if !strings.Contains(dump, "(call (. This default))") || !strings.Contains(dump, "(. This fields)") {
		t.Fatalf("dump:\n%s", dump)
	}
}

// D175: the implementing type is `This`; `Self` still reads as it, with an
// error whose fix writes the new word.
func TestOldSelfTypeSpelling(t *testing.T) {
	f, diags := parse(t, "trait Z {\n  static fun zero(): Self => Self.default()\n}\n")
	if len(diags.Items) != 2 {
		t.Fatalf("want two errors, got:\n%s", diags.Render())
	}
	for _, d := range diags.Items {
		if !strings.Contains(d.Message, "spelled 'This'") || d.Fix == nil || d.Fix.Edits[0].NewText != "This" {
			t.Errorf("diagnostic %q has no fix to 'This'", d.Message)
		}
	}
	if dump := ast.Dump(f); !strings.Contains(dump, "(. This default)") {
		t.Errorf("the old spelling should read as the type:\n%s", dump)
	}
}
