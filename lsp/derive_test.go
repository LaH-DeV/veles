package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Hovering the trait name of an implement the compiler filled in (D58)
// says what it wrote: the header of every implement made from that line,
// the signatures, and — for a Codable derive — the shape the value takes
// on the wire. The bodies are not in the hover; the command that prints
// them is named instead.
func TestDerivedImplementHover(t *testing.T) {
	src := "use io\nuse json\n\nstruct Note {\n  id: i64\n  @key(json: \"userName\") author: string\n  @skip cached: i64 = 0\n  implement Codable\n}\n\nfun main() throws EncodeError {\n  io.println(try json.encode(Note(id: 1, author: \"a\")))\n}\n"
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

	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 7, "character": 14}})
	var h struct {
		Contents struct {
			Value string `json:"value"`
		} `json:"contents"`
	}
	json.Unmarshal(res, &h)
	got := h.Contents.Value
	for _, want := range []string{
		"implement Encodable for Note",
		"implement Decodable for Note",
		"// derived (D58)",
		"fun encode(to: Encoder) throws EncodeError",
		"static fun decode(from: Decoder): Note throws DecodeError",
		"keys      id · author → \"userName\" (json)",
		"skipped   cached (@skip)",
		"--derive Note",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("hover on `implement Codable` is missing %q:\n%s", want, got)
		}
	}
	// the bodies stay out of the hover
	if strings.Contains(got, "try to.beginObject()") {
		t.Errorf("hover should not carry the synthesized bodies:\n%s", got)
	}
}

// On a generic target the header is the one the compiler read, bounds and
// all: the empty implement in `struct Page<T>` is `implement<T: Encodable>
// Encodable for Page<T>` (D58). A derived `Comparable` shows its own.
func TestDerivedImplementHoverGenerics(t *testing.T) {
	src := "use io\n\nstruct Page<T> {\n  items: List<T>\n  total: i64\n  implement Codable\n}\n\nstruct Tag {\n  name: string\n  implement Comparable\n}\n\nfun main() {\n  val a = Tag(name: \"a\")\n  val b = Tag(name: \"b\")\n  io.println(\"${a < b}\")\n}\n"
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
	hover := func(line, ch int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if got := hover(5, 14); !strings.Contains(got, "implement<T: Encodable> Encodable for Page<T>") ||
		!strings.Contains(got, "implement<T: Decodable> Decodable for Page<T>") {
		t.Errorf("generic derive header:\n%s", got)
	}
	if got := hover(10, 14); !strings.Contains(got, "implement Comparable for Tag") ||
		!strings.Contains(got, "fun compareTo(other: Tag): Ordering") {
		t.Errorf("Comparable derive:\n%s", got)
	}
}
