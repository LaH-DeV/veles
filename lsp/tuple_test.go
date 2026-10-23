package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A labelled tuple (D166) hovers with its labels, and completion after the
// dot offers each label beside its position.
func TestLabelledTupleHoverAndCompletion(t *testing.T) {
	src := `use io

fun divide(a: i64, b: i64): (quotient: i64, remainder: i64) => (a / b, a % b)

fun main() {
  val r = divide(7, 2)
  io.println("${r.quotient}")
}
`
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
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 5, "character": 6}})
	if want := "val r: (quotient: i64, remainder: i64)"; !strings.Contains(string(res), want) {
		t.Errorf("hover over r: %s\nwant it to contain %q", res, want)
	}
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 6, "character": 18}})
	for _, want := range []string{`"label":"quotient"`, `"label":"remainder"`, `"label":"0"`} {
		if !strings.Contains(string(res), want) {
			t.Errorf("completion after 'r.' is missing %s: %s", want, res)
		}
	}
}
