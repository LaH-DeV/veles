package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const arraysSrc = `use io

fun sum<const N: i64>(a: Array<i64, N>): i64 = a.fold(0, (s, x) => s + x)

fun main() {
  var a: Array<i64, 3> = [1, 2, 3]
  a.set(0, 9)
  io.println("${sum(a)} ${a.at(1)} ${a.len()}")
}
`

// D121: an array's methods hover with the catalogue's signature for the
// element type, and a generic function over a constant length is not an
// error in the editor.
func TestArraysInTheEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(arraysSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": arraysSrc}})
	doc := map[string]any{"uri": uri}
	hover := func(needle string, skip int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": at(t, arraysSrc, needle, skip)})
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover("set(0", 0); !strings.Contains(h, "Array<i64, 3>.set(i: i64, x: i64)") {
		t.Errorf("hover on set: %s", h)
	}
	if h := hover("at(1", 0); !strings.Contains(h, "Array<i64, 3>.at(i: i64): i64?") {
		t.Errorf("hover on at: %s", h)
	}
	if h := hover("sum(a)", 0); !strings.Contains(h, "fun sum<const N: i64>") {
		t.Errorf("hover on sum: %s", h)
	}
}
