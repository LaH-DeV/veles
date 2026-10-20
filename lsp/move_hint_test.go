package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D188: where a mutable collection becomes its immutable form by a move, the
// editor says "moved immutable" after it — and says nothing where the code
// copies with .toList().
func TestMoveHint(t *testing.T) {
	src := `use io

struct Acc {
  items: MutableList<i64> = []
}

fun fill(xs: MutableList<i64>) {
  xs.push(1)
}

fun built(): List<i64> {
  val out: MutableList<i64> = []
  fill(out)
  out
}

fun field(): List<i64> {
  val acc = Acc()
  acc.items.push(2)
  acc.items
}

fun copied(xs: MutableList<i64>): List<i64> {
  xs.toList()
}

fun main() {
  io.println("${built()} ${field()} ${copied([3])}")
}
`
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
	write("main.vs", src)
	uri := pathToURI(filepath.Join(dir, "main.vs"))
	cl, stop := newClient(t)
	defer stop()
	cl.call("initialize", map[string]any{})
	cl.notify("initialized", map[string]any{})
	cl.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	res, _ := cl.call("textDocument/inlayHint", map[string]any{"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	got := string(res)
	if n := strings.Count(got, "moved immutable"); n != 2 {
		t.Errorf("want two hints (the local and the field), got %d in %s", n, got)
	}
	// after `out` (line 14) and after `acc.items` (line 20), 1-based, nowhere else
	for _, want := range []string{`"line":13,"character":5`, `"line":19,"character":11`} {
		if !strings.Contains(got, want) {
			t.Errorf("want a hint at %s, got %s", want, got)
		}
	}
}
