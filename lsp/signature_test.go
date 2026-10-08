package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

const sigSrc = `use io

struct Point { x: i64, y: i64 = 0 }

/// Multiplies.
fun scale(value: i64, factor: i64 = 2): i64 => value * factor

fun main() {
  val p = Point(x: 1)
  io.println("${scale(p.x, 3)}")
}
`

func TestSignatureHelp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(sigSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": sigSrc}})

	version := 1
	help := func(src, marker string) (string, int) {
		t.Helper()
		off := strings.Index(src, marker)
		if off < 0 {
			t.Fatalf("no %q", marker)
		}
		src = strings.Replace(src, marker, "", 1)
		version++
		c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": []map[string]any{{"text": src}}})
		pos := offsetToPosition(source.NewFile("x", src), off)
		res, _ := c.call("textDocument/signatureHelp", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": pos})
		var h struct {
			Signatures []signatureInfo `json:"signatures"`
			Active     int             `json:"activeParameter"`
		}
		json.Unmarshal(res, &h)
		if len(h.Signatures) != 1 {
			return string(res), -1
		}
		return h.Signatures[0].Label, h.Active
	}

	// a complete call: the second argument
	marked := strings.Replace(sigSrc, "scale(p.x, 3)", "scale(p.x, |3)", 1)
	label, active := help(marked, "|")
	if label != "scale(value: i64, factor: i64 = …): i64" || active != 1 {
		t.Errorf("complete call: %q active %d", label, active)
	}

	// mid-typing: the buffer does not parse, the last good index answers
	typing := strings.Replace(sigSrc, "  io.println(\"${scale(p.x, 3)}\")\n", "  val q = scale(|\n", 1)
	label, active = help(typing, "|")
	if label != "scale(value: i64, factor: i64 = …): i64" || active != 0 {
		t.Errorf("mid-typing: %q active %d", label, active)
	}

	// a named argument selects its parameter whatever its position
	named := strings.Replace(sigSrc, "Point(x: 1)", "Point(y: |", 1)
	label, active = help(named, "|")
	if label != "Point(x: i64, y: i64 = …)" || active != 1 {
		t.Errorf("constructor by name: %q active %d", label, active)
	}

	// nested: the inner call wins; after it closes, the outer one
	nested := strings.Replace(sigSrc, "scale(p.x, 3)", "scale(scale(1, |), 3)", 1)
	if _, active = help(nested, "|"); active != 1 {
		t.Errorf("inner call: active %d", active)
	}
	outer := strings.Replace(sigSrc, "scale(p.x, 3)", "scale(scale(1, 2), |3)", 1)
	if _, active = help(outer, "|"); active != 1 {
		t.Errorf("outer call: active %d", active)
	}

	// a parenthesised expression is not a call
	paren := strings.Replace(sigSrc, "val p = Point(x: 1)", "val n = (1 + |", 1)
	if res, active := help(paren, "|"); active != -1 {
		t.Errorf("parentheses offered %s", res)
	}
}
