package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D198: hover over a variant says where it is stored — `indirect` or
// `inline` as written, and "boxed" for a variant of a tree.
func TestHoverBoxedVariant(t *testing.T) {
	src := `use io

sealed trait Expr {
  struct Add {
    left:  *Expr
    right: *Expr
  }
  inline struct Hole { }
}

sealed trait Event {
  indirect struct Upload {
    name: string
  }
}

fun main() {
  val a: Expr = Hole()
  val e: Event = Upload(name: "x")
  io.println("$a $e")
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
		return string(res)
	}
	for _, tc := range []struct {
		line, char int
		want       string
	}{
		{3, 10, "struct Add : Expr — boxed: 'Expr' is a tree"},
		{7, 17, "inline struct Hole : Expr"},
		{11, 19, "indirect struct Upload : Event"},
	} {
		if got := hover(tc.line, tc.char); !strings.Contains(got, tc.want) {
			t.Errorf("hover at %d:%d: %s\nwant it to contain %q", tc.line, tc.char, got, tc.want)
		}
	}
}
