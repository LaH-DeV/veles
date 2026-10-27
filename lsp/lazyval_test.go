package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const lazySrc = `use io

lazy val keywords = ["fun", "val"].toSet()

struct Tz {
  name: string

  static lazy val utc = Tz(name: "UTC")
}

fun main() {
  io.println("${keywords.contains("fun")} ${Tz.utc.name}")
}
`

// D210: a lazy value hovers as declared, with its type and when it is built.
func TestLazyValHover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(lazySrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": lazySrc}})
	hover := func(needle string, skip int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": at(t, lazySrc, needle, skip)})
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover("keywords.contains", 0); !strings.Contains(h, "lazy val keywords: Set<string>") || !strings.Contains(h, "lazy: built on first read") {
		t.Errorf("hover on a read of keywords: %s", h)
	}
	if h := hover("utc.name", 0); !strings.Contains(h, "static lazy val utc: Tz") {
		t.Errorf("hover on Tz.utc: %s", h)
	}
}
