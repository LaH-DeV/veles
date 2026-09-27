package lsp

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// location is a span as an LSP Location, or nil when the editor cannot
// open it (the standard library before it is materialised).
func (s *Server) location(sp source.Span) map[string]any {
	if !sp.IsValid() || (strings.HasPrefix(sp.File.Path, "std/") && s.stdDir == "") {
		return nil
	}
	return map[string]any{"uri": s.uriFor(sp.File.Path), "range": spanToRange(sp)}
}

// typeDefinition goes from a value to the declaration of its type: a
// variable of type `Note?` to `struct Note`, a call to its result's type,
// a `List<Tag>` to `Tag`.
func (s *Server) typeDefinition(params json.RawMessage) any {
	ref, _ := s.refAt(params)
	if ref == nil {
		return nil
	}
	t := ref.Type
	if fn, ok := t.(*types.Func); ok {
		t = fn.Ret
	}
	sp := sema.TypeDecl(t)
	if !sp.IsValid() {
		switch ref.Kind {
		case "struct", "trait", "sealed", "enum", "type":
			sp = ref.Def // a type name: its type is itself
		}
	}
	if loc := s.location(sp); loc != nil {
		return loc
	}
	return nil
}

// implementation goes from a trait to every implement of it (and a
// sealed trait to its variants), and from a trait method to each method
// implementing it.
func (s *Server) implementation(params json.RawMessage) any {
	var p positionParams
	json.Unmarshal(params, &p)
	a, f := s.analysisFor(p.TextDocument.URI)
	if a == nil || a.index == nil {
		return nil
	}
	ref := a.index.RefAt(f, positionToOffset(f, p.Position))
	if ref == nil {
		return nil
	}
	spans := a.index.Impls[ref.Def]
	if len(spans) == 0 && ref.Family.IsValid() {
		spans = a.index.Impls[ref.Family]
	}
	out := []map[string]any{}
	for _, sp := range spans {
		if loc := s.location(sp); loc != nil {
			out = append(out, loc)
		}
	}
	return out
}

type foldingRange struct {
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Kind      string `json:"kind,omitempty"`
}

// foldingRanges folds what spans lines: bracketed bodies and argument
// lists, comment blocks, and a run of `use` lines. It reads the tokens
// alone, so folding works while the file does not parse.
func (s *Server) foldingRanges(params json.RawMessage) any {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	if d == nil {
		return []foldingRange{}
	}
	return foldingRangesOf(source.NewFile(d.path, d.text))
}

func foldingRangesOf(f *source.File) []foldingRange {
	toks, _, comments := lexer.TokenizeAll(f, &source.Diagnostics{})
	line := func(off int) int { return offsetToPosition(f, off).Line }
	byStart := map[int]foldingRange{}
	put := func(r foldingRange) {
		if r.EndLine <= r.StartLine {
			return
		}
		// one range per start line, the widest: `} else {` closes one
		// block and opens the next on the same line
		if old, ok := byStart[r.StartLine]; !ok || r.EndLine > old.EndLine {
			byStart[r.StartLine] = r
		}
	}

	var open []lexer.Token
	for _, t := range toks {
		switch t.Kind {
		case lexer.LBrace, lexer.LBracket, lexer.LParen:
			open = append(open, t)
		case lexer.RBrace, lexer.RBracket, lexer.RParen:
			if len(open) == 0 {
				continue
			}
			o := open[len(open)-1]
			open = open[:len(open)-1]
			// the closing line stays visible
			put(foldingRange{StartLine: line(o.Span.Start), EndLine: line(t.Span.Start) - 1})
		}
	}

	// consecutive `use` lines
	first, last := -1, -1
	for i, t := range toks {
		if t.Kind != lexer.KwUse || (i > 0 && toks[i-1].Kind != lexer.Semi) {
			continue
		}
		l := line(t.Span.Start)
		if first >= 0 && l == last+1 {
			last = l
			continue
		}
		if first >= 0 {
			put(foldingRange{StartLine: first, EndLine: last, Kind: "imports"})
		}
		first, last = l, l
	}
	if first >= 0 {
		put(foldingRange{StartLine: first, EndLine: last, Kind: "imports"})
	}

	// comment blocks: one multi-line comment, or a run of line comments
	start, end := -1, -1
	flush := func() {
		if start >= 0 {
			put(foldingRange{StartLine: start, EndLine: end, Kind: "comment"})
		}
	}
	for _, c := range comments {
		if !startsLine(f.Content, c.Span.Start) {
			continue // a comment after code folds with nothing
		}
		a, b := line(c.Span.Start), line(c.Span.End)
		if strings.HasPrefix(c.Text, "/*") {
			flush()
			start = -1
			put(foldingRange{StartLine: a, EndLine: b, Kind: "comment"})
			continue
		}
		if start >= 0 && a == end+1 {
			end = a
			continue
		}
		flush()
		start, end = a, a
	}
	flush()

	out := make([]foldingRange, 0, len(byStart))
	for _, r := range byStart {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartLine < out[j].StartLine })
	return out
}

// startsLine reports whether only blanks precede off on its line.
func startsLine(src string, off int) bool {
	for i := off - 1; i >= 0 && src[i] != '\n'; i-- {
		if src[i] != ' ' && src[i] != '\t' {
			return false
		}
	}
	return true
}
