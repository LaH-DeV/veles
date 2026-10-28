package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// Hover on a keyword explains it and links its chapter; a contextual word
// is explained only where it is that word (`inline` before `loop`, `on` in
// an annotation's declaration), never on a name spelled the same.
func TestKeywordHover(t *testing.T) {
	src := "annotation Min(value: i64) on field for Check\n" +
		"trait Check {\n" +
		"  fun count(): i64 {\n" +
		"    var on = 0\n" +
		"    inline loop (f in This.fields) {\n" +
		"      if (f.has<Min>()) on += 1\n" +
		"    }\n" +
		"    on\n" +
		"  }\n" +
		"}\n"
	f := source.NewFile("main.vs", src)
	at := func(line int, word string, nth int) int {
		lines := strings.SplitAfter(src, "\n")
		off := 0
		for i := 0; i < line; i++ {
			off += len(lines[i])
		}
		// the nth whole-word occurrence (`on` is also inside `annotation`)
		hits := regexp.MustCompile(`\b`+word+`\b`).FindAllStringIndex(lines[line], -1)
		return off + hits[nth][0]
	}
	text := func(h any) string {
		if h == nil {
			return ""
		}
		return h.(map[string]any)["contents"].(map[string]any)["value"].(string)
	}
	for _, c := range []struct {
		line int
		word string
		nth  int
		want string // "" for no hover
	}{
		{0, "annotation", 0, "Declares an annotation"},
		{0, "on", 0, "Where an annotation may stand"},
		{0, "for", 0, "the trait that reads the annotation"},
		{2, "fun", 0, "Declares a function"},
		{3, "on", 0, ""}, // a variable named `on`
		{4, "inline", 0, "unrolled while compiling"},
		{4, "loop", 0, "The one loop keyword"},
		{5, "on", 0, ""},
	} {
		got := text(keywordHover(f, at(c.line, c.word, c.nth)))
		switch {
		case c.want == "" && got != "":
			t.Errorf("line %d %q: want no hover, got %q", c.line+1, c.word, got)
		case c.want != "" && !strings.Contains(got, c.want):
			t.Errorf("line %d %q: want %q in %q", c.line+1, c.word, c.want, got)
		}
	}
	// D197's hover, word for word; a longer name is not the keyword
	ceded := source.NewFile("c.vs", "fun f() {\n  cede\n  ceded()\n}\n")
	if got := text(keywordHover(ceded, strings.Index(ceded.Content, "cede"))); !strings.Contains(got, "— Yields execution to the current executor's scheduler. May resume immediately. Cancellation is checked unless cleanup is shielded.") {
		t.Errorf("cede: %q", got)
	}
	if got := text(keywordHover(ceded, strings.Index(ceded.Content, "ceded"))); got != "" {
		t.Errorf("ceded: want no hover, got %q", got)
	}
	// `test fun` is a helper, `test "…"` a test; `implements` asks about a type
	more := source.NewFile("t.vs", "test fun sample(): i64 => 1\ntest \"one\" { }\nfun f<T>(x: T): bool => T implements Display\n")
	for _, c := range []struct {
		off  int
		want string
	}{
		{0, "is a helper for tests"},
		{len("test fun sample(): i64 => 1\n"), "is a test, run by `veles test`"},
		{strings.Index(more.Content, "implements"), "asks, while compiling, whether a type parameter"},
	} {
		if got := text(keywordHover(more, c.off)); !strings.Contains(got, c.want) {
			t.Errorf("offset %d: want %q in %q", c.off, c.want, got)
		}
	}
}

// Every keyword's link names a chapter that exists and a heading in it.
func TestKeywordDocLinks(t *testing.T) {
	var docs []keywordDoc
	for _, d := range keywordDocs {
		docs = append(docs, d)
	}
	for _, cs := range contextualDocs {
		for _, c := range cs {
			docs = append(docs, c.keywordDoc)
		}
	}
	heading := regexp.MustCompile(`(?m)^#{1,4} (.+)$`)
	for _, d := range docs {
		file, anchor, _ := strings.Cut(d.doc, "#")
		data, err := os.ReadFile(filepath.Join("..", "docs", "documentation", file))
		if err != nil {
			t.Errorf("%s: %v", d.doc, err)
			continue
		}
		if anchor == "" {
			continue
		}
		found := false
		for _, m := range heading.FindAllStringSubmatch(string(data), -1) {
			if githubSlug(m[1]) == anchor {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no heading has the anchor %q", file, anchor)
		}
	}
}

// githubSlug is the anchor GitHub gives a heading: lower case, spaces as
// hyphens, punctuation dropped.
func githubSlug(h string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(h)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}
