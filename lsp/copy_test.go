package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The generated `copy` (D164) hovers with its signature as the reader would
// write it: the fields this call site may change, each defaulting to the
// receiver's — not a field `init` assigns.
func TestHoverGeneratedCopy(t *testing.T) {
	src := `use io

struct Rect {
  w: i64
  h: i64 = 1
  area: i64
  init {
    this.area = this.w * this.h
  }
}

fun main() {
  val r = Rect(w: 2)
  val wide = r.copy(w: 10)
  io.println("${wide.area}")
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
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 13, "character": 16}})
	var h struct {
		Contents struct {
			Value string `json:"value"`
		} `json:"contents"`
	}
	json.Unmarshal(res, &h)
	want := "fun Rect.copy(w: i64 = this.w, h: i64 = this.h): Rect"
	if !strings.Contains(h.Contents.Value, want) {
		t.Errorf("hover over copy:\n%s\nwant it to contain %q", h.Contents.Value, want)
	}
}
