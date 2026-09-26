package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// explainSource checks a one-file package and returns what the compiler
// derived in it, rendered as Veles (D58).
func explainSource(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	out := ExplainDerived(pkg, diags, "")
	if diags.HasErrors() {
		t.Fatalf("unexpected errors:\n%s", diags.Render())
	}
	return strings.Join(out, "\n\n")
}

// The derived bodies print as Veles a programmer could read: keywords,
// two-space indent, braced bodies broken, literals re-escaped. The
// formatter cannot do this (it copies text out of the source and derived
// nodes have none), so this pins the separate printer.
func TestDumpDerivedIsVeles(t *testing.T) {
	src := "use io\nuse json\n\nstruct Note {\n" +
		"  id: i64\n" +
		"  @key(json: \"userName\") author: string\n" +
		"  tags: List<string> = []\n" +
		"  @skip cached: i64 = 0\n" +
		"  implement Codable\n" +
		"}\n\n" +
		"fun main() throws EncodeError {\n  io.println(try json.encode(Note(id: 1, author: \"a\")))\n}\n"
	got := explainSource(t, src)
	for _, want := range []string{
		"implement Encodable for Note",
		"implement Decodable for Note",
		"fun encode(to: Encoder) throws EncodeError {",
		"  try to.beginObject()",
		"  try to.key(styleKey(\"id\", to.keys()))",
		"  try this.id.encode(to)",
		// a per-format @key is a `when` on the encoder's format
		"  try to.key(when (to.format()) {",
		"    \"json\" => \"userName\"",
		"    else => styleKey(\"author\", to.keys())",
		"  })",
		"static fun decode(from: Decoder): Note throws DecodeError {",
		"    val k = try from.nextKey() ?: break",
		"      when (i64.decode(from)) {",
		"        is Ok($ok) => {",
		// the field default has no spelling of its own
		"<default of Note.tags>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("derived source is missing %q:\n%s", want, got)
		}
	}
	// nothing fell through to a Go type name
	if strings.Contains(got, "<*ast.") {
		t.Errorf("a node printed as its Go type:\n%s", got)
	}
	// the skipped field is nowhere on the wire
	if strings.Contains(got, "\"cached\"") {
		t.Errorf("@skip field reached the derived code:\n%s", got)
	}
}

// A generic target's implement carries the bounds the compiler inferred.
func TestDumpDerivedInfersBounds(t *testing.T) {
	src := "use io\n\nstruct Page<T> {\n  items: List<T>\n  total: i64\n  implement Codable\n}\n\nfun main() { io.println(\"ok\") }\n"
	got := explainSource(t, src)
	for _, want := range []string{
		"implement<T: Encodable> Encodable for Page<T>",
		"implement<T: Decodable> Decodable for Page<T>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
}
