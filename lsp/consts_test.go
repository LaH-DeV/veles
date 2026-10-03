package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const constsSrc = `use io

const KB: i64 = 1024
const MB = KB * 1024
const NAMES: List<string> = ["a", "b"]

static assert(MB == 1048576, "a megabyte is 1024 kilobytes")

fun main() {
  io.println("${MB} ${NAMES}")
}
`

// D113: hovering a constant shows what the compiler computed when it is not
// what is written; a module-level `static assert` is not an error.
func TestConstantsInTheEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(constsSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": constsSrc}})
	doc := map[string]any{"uri": uri}
	hover := func(needle string, skip int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": at(t, constsSrc, needle, skip)})
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover("MB}", 0); !strings.Contains(h, "const MB: i64 = KB * 1024  // 1048576") {
		t.Errorf("hover on a use of MB: %s", h)
	}
	if h := hover("KB: i64", 0); !strings.Contains(h, "const KB: i64 = 1024") || strings.Contains(h, "//") {
		t.Errorf("hover on KB, whose value is as written: %s", h)
	}
}
