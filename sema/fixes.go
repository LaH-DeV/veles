package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/source"
)

// Small fixes attached to lints (see lint_impl.go for the larger one).
// Each builds a source.Fix from spans in the file being checked.

// fixReplace replaces one span.
func fixReplace(title string, span source.Span, text string) *source.Fix {
	return &source.Fix{Title: title, Edits: []source.TextEdit{{Span: span, NewText: text}}}
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
