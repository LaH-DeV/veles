package source

import (
	"strings"
	"testing"
)

// The caret under a diagnostic stands under the span whatever precedes it
// on the line — non-ASCII characters (one space each, not one per byte) and
// tabs (copied, so a terminal moves both lines alike) — and is one `^` per
// character of the span.
func TestCaretCountsCharacters(t *testing.T) {
	cases := []struct {
		line       string
		start, end int // the span, in bytes of the line
		want       string
	}{
		{"val x = 1", 4, 5, "    ^"},
		{"val é = \"\\q\"", 9, 11, "        ^^"},
		{"\t\tval s = é", 10, 12, "\t\t        ^"},
		{"// €€€ here", 3, 9, "   ^^"},
		{"abc", 3, 3, "   ^"},
	}
	for _, c := range cases {
		f := NewFile("t.vs", c.line+"\n")
		var d Diagnostics
		d.Errorf(Span{File: f, Start: c.start, End: c.end}, "x")
		lines := strings.Split(d.Render(), "\n")
		if got := strings.TrimPrefix(lines[2], "  "); got != c.want {
			t.Errorf("%q [%d,%d): caret %q, want %q", c.line, c.start, c.end, got, c.want)
		}
	}
}
