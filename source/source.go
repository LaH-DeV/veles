// Package source holds source files, positions and diagnostics shared by
// every compiler phase.
package source

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// File is one `.vs` source file loaded into memory.
type File struct {
	Path    string
	Content string
	// Embedded marks a file compiled into the tool (the std sources) rather
	// than read from disk; fixes are never written to one.
	Embedded bool
	lines    []int // byte offset of the start of each line
}

func NewFile(path, content string) *File {
	f := &File{Path: path, Content: content}
	f.lines = append(f.lines, 0)
	for i, c := range content {
		if c == '\n' {
			f.lines = append(f.lines, i+1)
		}
	}
	return f
}

// Position converts a byte offset into a 1-based line/column pair.
func (f *File) Position(offset int) (line, col int) {
	if f == nil {
		return 0, 0
	}
	i := sort.Search(len(f.lines), func(i int) bool { return f.lines[i] > offset }) - 1
	if i < 0 {
		i = 0
	}
	return i + 1, offset - f.lines[i] + 1
}

// Line returns the text of the given 1-based line without its newline.
func (f *File) Line(n int) string {
	if n < 1 || n > len(f.lines) {
		return ""
	}
	start := f.lines[n-1]
	end := len(f.Content)
	if n < len(f.lines) {
		end = f.lines[n] - 1
	}
	return strings.TrimRight(f.Content[start:end], "\r")
}

// Span is a half-open byte range inside a file.
type Span struct {
	File       *File
	Start, End int
}

func (s Span) String() string {
	if s.File == nil {
		return "<builtin>"
	}
	line, col := s.File.Position(s.Start)
	return fmt.Sprintf("%s:%d:%d", s.File.Path, line, col)
}

func (s Span) IsValid() bool { return s.File != nil }

// To returns a span covering from s to the end of other.
func (s Span) To(other Span) Span {
	if !s.IsValid() {
		return other
	}
	if !other.IsValid() {
		return s
	}
	return Span{File: s.File, Start: s.Start, End: other.End}
}

type Severity int

const (
	Error Severity = iota
	Warning
	Note
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "note"
	}
}

type Diagnostic struct {
	Severity Severity
	Span     Span
	Message  string
	// Fix is an optional automatic correction (a lint's autofix); tools
	// that can apply edits — the language server — offer it.
	Fix *Fix
}

// Fix is a set of text edits, possibly across files, that resolves a
// diagnostic. Edits are applied to the text as it was when the diagnostic
// was produced; an empty span inserts.
type Fix struct {
	Title string
	Edits []TextEdit
	// Guess marks a likely correction rather than a certain one (a typo's
	// nearest name): the editor offers it, `veles check --fix` leaves it.
	Guess bool
}

type TextEdit struct {
	Span    Span
	NewText string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s: %s: %s", d.Span, d.Severity, d.Message)
}

// Diagnostics collects reports from every phase. Phases never panic on user
// errors; they record a diagnostic and keep going where they can.
type Diagnostics struct {
	Items []Diagnostic
}

func (d *Diagnostics) Errorf(span Span, format string, args ...any) {
	d.Items = append(d.Items, Diagnostic{Severity: Error, Span: span, Message: fmt.Sprintf(format, args...)})
}

func (d *Diagnostics) Warnf(span Span, format string, args ...any) {
	d.Items = append(d.Items, Diagnostic{Severity: Warning, Span: span, Message: fmt.Sprintf(format, args...)})
}

func (d *Diagnostics) HasErrors() bool {
	for _, it := range d.Items {
		if it.Severity == Error {
			return true
		}
	}
	return false
}

func (d *Diagnostics) ErrorCount() int {
	n := 0
	for _, it := range d.Items {
		if it.Severity == Error {
			n++
		}
	}
	return n
}

// caret is the line under a source line that marks `width` bytes from byte
// `at`: it lines up by characters, not bytes — a space for each character
// before the span and a tab for a tab, so it sits under the span however a
// terminal draws `é` or a tab — and is one `^` per character of the span
// on this line, at least one. (It once counted bytes, and stood to the right
// of, and wider than, anything after a non-ASCII character.)
func caret(text string, at, width int) string {
	if at > len(text) {
		at = len(text)
	}
	var sb strings.Builder
	for _, r := range text[:at] {
		if r == '\t' {
			sb.WriteByte('\t')
		} else {
			sb.WriteByte(' ')
		}
	}
	end := min(at+max(width, 0), len(text))
	n := utf8.RuneCountInString(text[at:end])
	sb.WriteString(strings.Repeat("^", max(n, 1)))
	return sb.String()
}

// Render formats every diagnostic with a source excerpt and caret.
func (d *Diagnostics) Render() string {
	var sb strings.Builder
	for _, it := range d.Items {
		sb.WriteString(it.String())
		sb.WriteByte('\n')
		if it.Span.IsValid() {
			line, col := it.Span.File.Position(it.Span.Start)
			text := it.Span.File.Line(line)
			sb.WriteString("  " + text + "\n")
			sb.WriteString("  " + caret(text, col-1, it.Span.End-it.Span.Start) + "\n")
		}
	}
	// where to read more (D79): once per family, in the order they came up
	seen := map[string]bool{}
	for _, it := range d.Items {
		if f := FamilyOf(it.Message); f != "" && !seen[f] {
			seen[f] = true
			sb.WriteString("see: veles explain " + f + "\n")
		}
	}
	return sb.String()
}
