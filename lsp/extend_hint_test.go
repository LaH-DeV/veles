package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D185: the hint at the end of a `use` line names the types of the extension
// functions its `extend` clause brings — and there is none where the use
// brings none.
func TestExtensionHintFollowsTheExtendClause(t *testing.T) {
	strx := "public extend string with fun shout(): string => this.toUpper()\n\npublic extend<T> List<T> with fun second(): T? => this.at(1)\n"
	cases := []struct {
		use  string
		want string // "" when there is no hint at all
	}{
		{"use strx", ""},
		{"use strx extend", "extends string, List\\u003cT\\u003e"},
		{"use strx extend { shout }", "extends string"},
		{"use strx extend { second }", "extends List\\u003cT\\u003e"},
	}
	for _, c := range cases {
		t.Run(c.use, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "strx"), 0o755); err != nil {
				t.Fatal(err)
			}
			write := func(name, text string) {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			write(filepath.Join("strx", "strx.vs"), strx)
			write("veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n")
			src := "use io\n" + c.use + "\n\nfun main() {\n  io.println(\"hi\")\n}\n"
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
			if c.want == "" {
				if strings.Contains(got, "extends") {
					t.Errorf("a use without extend has no hint, got %s", got)
				}
				return
			}
			if !strings.Contains(got, c.want) {
				t.Errorf("want a hint %q, got %s", c.want, got)
			}
			if c.want == "extends string" && strings.Contains(got, "List") {
				t.Errorf("the hint names a type the clause does not bring: %s", got)
			}
		})
	}
}
