package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Hover on what has no declaration of its own: an inline loop's name (one
// field per copy of the body, D174) and a built-in generic in a type
// position (`Array<u8, 4>`, `List<i64>`).
func TestHoverHandleAndBuiltinTypes(t *testing.T) {
	src := `use io

struct Pixel {
  r: u8 = 0
  g: u8 = 0
  implement Show
}

trait Show {
  fun show(): string {
    var out = ""
    inline loop (f in This.fields) {
      out += f.name
    }
    out
  }
}

fun main() {
  val bytes: Array<u8, 4> = Array<u8, 4>.make(0)
  val xs: List<i64> = [1]
  io.println("${Pixel().show()} ${bytes.len()} ${xs.len()}")
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
	hover := func(line, char int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": char}})
		var h struct {
			Contents struct{ Value string } `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	for _, tc := range []struct {
		line, char int
		want       string
	}{
		{11, 18, "inline loop (f in Pixel.fields)"}, // the binding
		{12, 13, "stands for one field of `Pixel`"}, // a use
		{19, 14, "struct Array<T, const N: i64>"},
		{20, 10, "struct List<T>"},
	} {
		if got := hover(tc.line, tc.char); !strings.Contains(got, tc.want) {
			t.Errorf("hover at %d:%d: %s\nwant it to contain %q", tc.line, tc.char, got, tc.want)
		}
	}
}
