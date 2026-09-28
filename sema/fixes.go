package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Small fixes attached to lints (see lint_impl.go for the larger one).
// Each builds a source.Fix from spans in the file being checked.

// fixReplace replaces one span.
func fixReplace(title string, span source.Span, text string) *source.Fix {
	return &source.Fix{Title: title, Edits: []source.TextEdit{{Span: span, NewText: text}}}
}

// fixAddMembers inserts declarations before the closing brace of the block
// that ends `span` (an implement), one per line, indented one level in
// from the line the block starts on: `implement Shape for Sq { }` and a
// body with methods already in it both come out as a formatted block. Nil
// when the span does not end in a brace.
func fixAddMembers(title string, span source.Span, members []string) *source.Fix {
	if span.File == nil || span.End < 1 || span.End > len(span.File.Content) {
		return nil
	}
	src := span.File.Content
	closing := span.End - 1
	if src[closing] != '}' {
		return nil
	}
	lineStart := strings.LastIndexByte(src[:span.Start], '\n') + 1
	indent := ""
	for i := lineStart; i < len(src) && (src[i] == ' ' || src[i] == '\t'); i++ {
		indent += string(src[i])
	}
	from := closing // back over the blank run before the brace
	for from > span.Start && strings.ContainsRune(" \t\r\n", rune(src[from-1])) {
		from--
	}
	var sb strings.Builder
	for _, m := range members {
		for _, line := range strings.Split(m, "\n") { // a member may be a block
			sb.WriteString("\n" + indent + "  " + line)
		}
	}
	sb.WriteString("\n" + indent)
	return &source.Fix{Title: title, Edits: []source.TextEdit{{Span: source.Span{File: span.File, Start: from, End: closing}, NewText: sb.String()}}}
}

// fixDeleteLine deletes a span together with the rest of its line when
// nothing else is on it: leading indentation and the line break after.
func fixDeleteLine(title string, span source.Span) *source.Fix {
	if span.File == nil {
		return nil
	}
	src := span.File.Content
	start, end := span.Start, span.End
	ls := start
	for ls > 0 && (src[ls-1] == ' ' || src[ls-1] == '\t') {
		ls--
	}
	le := end
	for le < len(src) && (src[le] == ' ' || src[le] == '\t') {
		le++
	}
	if (ls == 0 || src[ls-1] == '\n') && (le == len(src) || src[le] == '\n' || src[le] == '\r') {
		start = ls
		end = le
		if end < len(src) && src[end] == '\r' {
			end++
		}
		if end < len(src) && src[end] == '\n' {
			end++
		}
	}
	return fixReplace(title, source.Span{File: span.File, Start: start, End: end}, "")
}

// fixDropMut removes the `mut` of a collection literal at span (which
// starts at the keyword), with the space after it.
func fixDropMut(span source.Span) *source.Fix {
	if span.File == nil {
		return nil
	}
	src := span.File.Content
	rest := src[span.Start:span.End]
	if !strings.HasPrefix(rest, "mut") {
		return nil
	}
	end := span.Start + 3
	for end < span.End && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return fixReplace("Remove the redundant 'mut'", source.Span{File: span.File, Start: span.Start, End: end}, "")
}

func (f *fnCtx) warnFix(span source.Span, fix *source.Fix, format string, args ...any) {
	if fix == nil {
		f.c.warnf(span, format, args...)
		return
	}
	f.c.warnFix(span, fix, format, args...)
}

// fixVarField inserts `var ` before the declaration of fld in st, when the
// declaration is at hand (a user struct in this compilation).
func (f *fnCtx) fixVarField(st *types.Struct, fld *types.Field) *source.Fix {
	d, ok := st.Decl.(*ast.StructDecl)
	if !ok {
		return nil
	}
	for _, af := range d.Fields {
		if af.Name.Name == fld.Name && af.Name.Pos.File != nil {
			if f.c.varFixes[af] {
				return nil // one insertion per field, however many assignments
			}
			if f.c.varFixes == nil {
				f.c.varFixes = map[*ast.Field]bool{}
			}
			f.c.varFixes[af] = true
			at := af.Name.Pos
			return fixReplace("Declare '"+fld.Name+"' as 'var'", source.Span{File: at.File, Start: at.Start, End: at.Start}, "var ")
		}
	}
	return nil
}
