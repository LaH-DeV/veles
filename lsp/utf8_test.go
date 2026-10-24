package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// D199: an editor opens a file that is not UTF-8 with U+FFFD in place of
// each byte it cannot decode, and sends that text. While the buffer is the
// file on disk, the server analyses the disk's bytes, so the editor shows
// the compiler's errors where they are; once edited, the buffer is what
// counts.
func TestNotUTF8OnDisk(t *testing.T) {
	disk := "use io\n\n// caf\xe9\nfun main() {\n  io.println(\"\xff\")\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(disk), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	errorsAfter := func(method string, params map[string]any) []lspDiagnostic {
		c.notify(method, params)
		_, notes := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 0}})
		var diags []lspDiagnostic
		for _, n := range notes {
			var env struct {
				Method string `json:"method"`
				Params struct {
					Diagnostics []lspDiagnostic `json:"diagnostics"`
				} `json:"params"`
			}
			json.Unmarshal(n, &env)
			if env.Method == "textDocument/publishDiagnostics" {
				diags = nil
				for _, d := range env.Params.Diagnostics {
					if d.Severity == 1 {
						diags = append(diags, d)
					}
				}
			}
		}
		return diags
	}
	shown := strings.ToValidUTF8(disk, string(utf8.RuneError))
	diags := errorsAfter("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": shown}})
	if len(diags) != 2 || !strings.Contains(diags[0].Message, "not valid UTF-8 here (byte 0xE9)") || !strings.Contains(diags[1].Message, "(byte 0xFF)") {
		t.Fatalf("expected the two D199 errors, got %+v", diags)
	}
	if r := diags[0].Range; r.Start.Line != 2 || r.Start.Character != 6 || r.End.Character != 7 {
		t.Errorf("first error at %+v, want line 2, characters 6–7", r)
	}
	if diags[0].Code != "syntax" {
		t.Errorf("code %q, want syntax", diags[0].Code)
	}
	// the user retypes the text: the buffer is valid, and what is analysed
	fixed := strings.ReplaceAll(strings.ReplaceAll(shown, "caf"+string(utf8.RuneError), "café"), string(utf8.RuneError), "x")
	diags = errorsAfter("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": fixed}}})
	if len(diags) != 0 {
		t.Errorf("after the edit: %+v", diags)
	}
}
