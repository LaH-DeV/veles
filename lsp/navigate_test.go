package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const navSrc = `use io

/// A shape.
trait Shape {
  fun area(): i64
}

sealed trait Token
struct Word : Token { text: string }
struct Num : Token { value: i64 }

struct Sq {
  side: i64

  implement Shape {
    fun area(): i64 = this.side * this.side
  }
}

struct Rect { w: i64, h: i64 }

implement Shape for Rect {
  fun area(): i64 = this.w * this.h
}

// two lines
// of comment
fun make(): Sq? = Sq(side: 2)

fun main() {
  val sq = make()
  val rects = [Rect(w: 1, h: 2)]
  io.println("${sq?.area() ?: 0} ${rects.size}")
}
`

type navLoc struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

func openNav(t *testing.T) (*client, func(), map[string]any) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(navSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": navSrc}})
	return c, stop, map[string]any{"uri": uri}
}

// lineOf is the 0-based line of the nth occurrence of needle.
func lineOf(t *testing.T, needle string, nth int) int {
	return at(t, navSrc, needle, nth)["line"].(int)
}

func TestTypeDefinition(t *testing.T) {
	c, stop, doc := openNav(t)
	defer stop()
	for _, tc := range []struct {
		needle string
		nth    int
		want   int // the line of the declaration, -1 for none
	}{
		{"sq", 1, lineOf(t, "struct Sq", 0)},      // a `Sq?` variable
		{"make", 1, lineOf(t, "struct Sq", 0)},    // a call: its result's type
		{"rects", 1, lineOf(t, "struct Rect", 0)}, // a List<Rect>
		{"Sq", 1, lineOf(t, "struct Sq", 0)},      // a type name is its own type
		{"side", 1, -1},                           // i64: declared nowhere
	} {
		res, _ := c.call("textDocument/typeDefinition", map[string]any{"textDocument": doc, "position": at(t, navSrc, tc.needle, tc.nth)})
		var loc *navLoc
		json.Unmarshal(res, &loc)
		switch {
		case tc.want < 0 && loc != nil:
			t.Errorf("type of %s: want none, got %s", tc.needle, res)
		case tc.want >= 0 && (loc == nil || loc.Range.Start.Line != tc.want):
			t.Errorf("type of %s: want line %d, got %s", tc.needle, tc.want, res)
		}
	}
}

func TestImplementation(t *testing.T) {
	c, stop, doc := openNav(t)
	defer stop()
	lines := func(needle string, nth int) []int {
		res, _ := c.call("textDocument/implementation", map[string]any{"textDocument": doc, "position": at(t, navSrc, needle, nth)})
		var locs []navLoc
		if err := json.Unmarshal(res, &locs); err != nil {
			t.Fatalf("implementation: %s", res)
		}
		var out []int
		for _, l := range locs {
			out = append(out, l.Range.Start.Line)
		}
		return out
	}
	eq := func(what string, got []int, want ...int) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: got lines %v, want %v", what, got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: got lines %v, want %v", what, got, want)
				return
			}
		}
	}
	// a trait: the implement inside the struct and the one outside it
	eq("Shape", lines("Shape", 0), lineOf(t, "implement Shape {", 0), lineOf(t, "implement Shape for", 0))
	// a trait method, from its declaration and from a call through it
	methods := []int{lineOf(t, "fun area", 1), lineOf(t, "fun area", 2)}
	eq("area", lines("area", 0), methods...)
	eq("area call", lines("area()", 2), methods...)
	// a sealed trait: its variants
	eq("Token", lines("Token", 0), lineOf(t, "struct Word", 0), lineOf(t, "struct Num", 0))
	// a struct implements nothing of its own
	eq("Rect", lines("Rect", 0))
}

func TestFoldingRanges(t *testing.T) {
	c, stop, doc := openNav(t)
	defer stop()
	res, _ := c.call("textDocument/foldingRange", map[string]any{"textDocument": doc})
	var folds []foldingRange
	if err := json.Unmarshal(res, &folds); err != nil {
		t.Fatalf("foldingRange: %s", res)
	}
	has := func(start, end int, kind string) {
		t.Helper()
		for _, f := range folds {
			if f.StartLine == start && f.EndLine == end && f.Kind == kind {
				return
			}
		}
		t.Errorf("no fold %d-%d %q in %+v", start, end, kind, folds)
	}
	sq := lineOf(t, "struct Sq", 0)
	has(sq, lineOf(t, "struct Rect", 0)-3, "")                           // the struct body; its closing line stays visible
	has(lineOf(t, "implement Shape {", 0), lineOf(t, "fun area", 1), "") // the inline implement
	has(lineOf(t, "// two", 0), lineOf(t, "// of", 0), "comment")
	for _, f := range folds {
		if f.StartLine == lineOf(t, "struct Rect", 0) {
			t.Errorf("a one-line body folds: %+v", f)
		}
	}
}

// Tests (D78): the vocabulary hovers like any built-in, and a test is in
// the outline under its sentence.
func TestTestsInTheEditor(t *testing.T) {
	src := "fun half(n: i64): i64? = if (n % 2 == 0) n / 2 else null\n\ntest \"halves even numbers\" {\n  expect(require(half(4)) == 2)\n}\n\nsuite \"odd numbers\" {\n  test \"have no half\" {\n    expect(half(3) == null)\n  }\n}\n\nfun main() { }\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	doc := map[string]any{"uri": uri}
	hover := func(needle string) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": at(t, src, needle, 0)})
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover("expect("); !strings.Contains(h, "fun expect(condition: bool)") || !strings.Contains(h, "the test goes on") {
		t.Errorf("expect: %s", h)
	}
	if h := hover("require("); !strings.Contains(h, "fun require<T>") {
		t.Errorf("require: %s", h)
	}
	res, _ := c.call("textDocument/documentSymbol", map[string]any{"textDocument": doc})
	if !strings.Contains(string(res), `"name":"halves even numbers"`) || !strings.Contains(string(res), `"name":"odd numbers","detail":"suite"`) || !strings.Contains(string(res), `"name":"have no half"`) {
		t.Errorf("outline: %s", res)
	}
}
