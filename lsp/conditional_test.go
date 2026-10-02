package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D116: a function that suspends only through its `suspends` parameter
// says so in its hover — "suspends if `f` does" — and is not marked
// `suspends` itself; one that also suspends by itself is, even though a
// parameter's type already spells the word.
func TestHoverShowsConditionalSuspension(t *testing.T) {
	src := "fun each(xs: List<i64>, f: fun(i64): () suspends) {\n" +
		"  loop (x in xs) {\n" +
		"    f(x)\n" +
		"  }\n" +
		"}\n" +
		"\n" +
		"fun eachSlowly(xs: List<i64>, f: fun(i64): () suspends) {\n" +
		"  await sleep(Duration.millis(1))\n" +
		"  each(xs, f)\n" +
		"}\n" +
		"\n" +
		"fun main() {\n" +
		"  each([1], x => { })\n" +
		"  eachSlowly([1], x => { })\n" +
		"}\n"
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
	hover := func(line, col int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": col}})
		return string(res)
	}
	conditional := "suspends if `f` does"
	for _, tc := range []struct {
		what      string
		line, col int
		want      string
		not       string
	}{
		{"conditional declaration", 0, 5, conditional, "fun(i64) suspends) suspends"},
		{"conditional call", 12, 3, conditional, "fun(i64) suspends) suspends"},
		{"suspends by itself", 6, 5, "fun(i64) suspends) suspends", conditional},
		{"its call", 13, 3, "fun(i64) suspends) suspends", conditional},
	} {
		got := hover(tc.line, tc.col)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: hover has no %q: %s", tc.what, tc.want, got)
		}
		if strings.Contains(got, tc.not) {
			t.Errorf("%s: hover has %q: %s", tc.what, tc.not, got)
		}
	}
}
