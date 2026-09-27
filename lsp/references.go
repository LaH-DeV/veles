package lsp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// Find-references, document highlights and rename all start from the
// checker's index: every resolved name records its declaration, so the
// uses of a declaration are the references that group under it (sema.Ref.Key).
// Rename then proves itself: the edited package is checked again, and the
// rename is refused if any name would resolve differently — a new name that
// shadows, or is captured by, another declaration changes what the program
// means, which a rename must never do.

// errRequestFailed is LSP's RequestFailed: the request was understood but
// cannot be carried out; the editor shows the message.
const errRequestFailed = -32803

// spanKey identifies a span across two checks of the same files.
type spanKey struct {
	path       string
	start, end int
}

func keyOf(sp source.Span) spanKey {
	return spanKey{sema.OverlayKey(sp.File.Path), sp.Start, sp.End}
}

// occurrence is one place a declaration's name is written.
type occurrence struct {
	span source.Span
	// pun: the name is a D28 pun (`Point(x)`), both the field and a variable
	pun bool
	// decl: this is the declaration itself
	decl bool
}

// occurrencesOf returns every place the declaration target names is
// written, one per span, ordered by file and offset. A reference with an
// empty span (a member the compiler wrote, D58) is not a place in the text.
func occurrencesOf(ix *sema.Index, target *sema.Ref) []occurrence {
	want := target.Key()
	if !want.IsValid() {
		return nil
	}
	wantKey := keyOf(want)
	puns := map[spanKey]bool{}
	for i := range ix.Refs {
		if r := &ix.Refs[i]; r.Pun {
			puns[keyOf(r.Span)] = true
		}
	}
	seen := map[spanKey]int{}
	var out []occurrence
	for i := range ix.Refs {
		r := &ix.Refs[i]
		// the name check keeps `this` (declared at its method's name) out
		// of the method's references
		if !r.Span.IsValid() || r.Span.Start == r.Span.End || r.Name != target.Name {
			continue
		}
		if k := r.Key(); !k.IsValid() || keyOf(k) != wantKey {
			continue
		}
		k := keyOf(r.Span)
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = len(out)
		out = append(out, occurrence{span: r.Span, pun: puns[k], decl: r.Span == r.Def})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].span, out[j].span
		if a.File.Path != b.File.Path {
			return a.File.Path < b.File.Path
		}
		return a.Start < b.Start
	})
	return out
}

// shownPath reports whether a location in path can be handed to the
// editor: the embedded standard library only when it has been materialised.
func (s *Server) shownPath(path string) bool {
	return !strings.HasPrefix(path, "std/") || s.stdDir != ""
}

func (s *Server) references(params json.RawMessage) any {
	var p struct {
		Context struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}
	json.Unmarshal(params, &p)
	a, ref, _ := s.targetAt(params)
	out := []map[string]any{}
	if ref == nil {
		return out
	}
	for _, o := range occurrencesOf(a.index, ref) {
		if (o.decl && !p.Context.IncludeDeclaration) || !s.shownPath(o.span.File.Path) {
			continue
		}
		out = append(out, map[string]any{"uri": s.uriFor(o.span.File.Path), "range": spanToRange(o.span)})
	}
	return out
}

// documentHighlight marks the other uses of the name under the cursor in
// the same file; the declaration is marked as a write.
func (s *Server) documentHighlight(params json.RawMessage) any {
	a, ref, f := s.targetAt(params)
	out := []map[string]any{}
	if ref == nil {
		return out
	}
	for _, o := range occurrencesOf(a.index, ref) {
		if o.span.File != f {
			continue
		}
		kind := 2 // read
		if o.decl {
			kind = 3 // write
		}
		out = append(out, map[string]any{"range": spanToRange(o.span), "kind": kind})
	}
	return out
}

// targetAt is refAt that also returns the analysis the reference is in.
func (s *Server) targetAt(params json.RawMessage) (*analysis, *sema.Ref, *source.File) {
	var p positionParams
	json.Unmarshal(params, &p)
	a, f := s.analysisFor(p.TextDocument.URI)
	if a == nil || a.index == nil {
		return nil, nil, nil
	}
	return a, a.index.RefAt(f, positionToOffset(f, p.Position)), f
}

// renameTarget returns the reference under the cursor if its declaration
// can be renamed, or why not.
func (s *Server) renameTarget(params json.RawMessage) (*analysis, *sema.Ref, error) {
	var p positionParams
	json.Unmarshal(params, &p)
	a, f := s.analysisFor(p.TextDocument.URI)
	if a == nil {
		return nil, nil, fmt.Errorf("this file is not part of an analysed package")
	}
	if a.index == nil {
		return nil, nil, fmt.Errorf("the package does not parse; fix the syntax errors first")
	}
	ref := a.index.RefAt(f, positionToOffset(f, p.Position))
	switch {
	case ref == nil:
		return nil, nil, fmt.Errorf("there is no name here to rename")
	case ref.Kind == "module":
		return nil, nil, fmt.Errorf("a module is named by its file or directory; rename that instead")
	case ref.Name == "this":
		return nil, nil, fmt.Errorf("'this' is the receiver's keyword, not a name")
	case !ref.Key().IsValid():
		return nil, nil, fmt.Errorf("'%s' is built into the language", ref.Name)
	case a.files[sema.OverlayKey(ref.Key().File.Path)] == nil:
		return nil, nil, fmt.Errorf("'%s' is declared in the standard library, which cannot be renamed from here", ref.Name)
	}
	if by, ok := a.index.Wire[ref.Key()]; ok {
		// the program would still compile, and read back nothing it wrote before
		return nil, nil, fmt.Errorf("'%s' is written on the wire by the derived '%s'; renaming it changes the encoded form. Write @key(\"%s\") on it first to keep the old name", ref.Name, by, ref.Name)
	}
	return a, ref, nil
}

func (s *Server) prepareRename(params json.RawMessage) (any, error) {
	_, ref, err := s.renameTarget(params)
	if err != nil {
		return nil, err
	}
	return map[string]any{"range": spanToRange(ref.Span), "placeholder": ref.Name}, nil
}

// validName reports whether name lexes as exactly one identifier.
func validName(name string) bool {
	diags := &source.Diagnostics{}
	toks := lexer.Tokenize(source.NewFile("rename", name), diags)
	if diags.HasErrors() || len(toks) == 0 || toks[0].Kind != lexer.Ident || toks[0].Text != name {
		return false
	}
	for _, t := range toks[1:] {
		if t.Kind != lexer.EOF && t.Kind != lexer.Semi {
			return false
		}
	}
	return true
}

type fileEdit struct {
	span    source.Span
	newText string
	pun     bool // a pun spelled out: the checks after it see other spans
}

func (s *Server) rename(params json.RawMessage) (any, error) {
	var p struct {
		positionParams
		NewName string `json:"newName"`
	}
	json.Unmarshal(params, &p)
	a, ref, err := s.renameTarget(params)
	if err != nil {
		return nil, err
	}
	if !validName(p.NewName) {
		return nil, fmt.Errorf("'%s' is not an identifier (or is a keyword)", p.NewName)
	}
	occs := occurrencesOf(a.index, ref)
	byFile := map[*source.File][]fileEdit{}
	for _, o := range occs {
		if a.files[sema.OverlayKey(o.span.File.Path)] == nil {
			return nil, fmt.Errorf("'%s' is used in the standard library, which cannot be renamed from here", ref.Name)
		}
		e := fileEdit{span: o.span, newText: p.NewName}
		if o.pun {
			// `Point(x)` names both the field and the variable: keep the
			// other one's name and spell the pair out
			e.pun = true
			if ref.Kind == "field" {
				e.newText = p.NewName + ": " + ref.Name
			} else {
				e.newText = ref.Name + ": " + p.NewName
			}
		}
		byFile[o.span.File] = append(byFile[o.span.File], e)
	}
	if p.NewName == ref.Name {
		return map[string]any{"changes": map[string]any{}}, nil
	}
	if err := s.verifyRename(a, p.TextDocument.URI, byFile, p.NewName); err != nil {
		return nil, err
	}
	changes := map[string][]lspTextEdit{}
	for f, edits := range byFile {
		uri := s.uriFor(f.Path)
		for _, e := range edits {
			changes[uri] = append(changes[uri], lspTextEdit{Range: spanToRange(e.span), NewText: e.newText})
		}
	}
	return map[string]any{"changes": changes}, nil
}

// verifyRename checks the package again with the edits applied and refuses
// the rename if it adds an error or if any name in the package's own files
// resolves to a different declaration than before.
func (s *Server) verifyRename(a *analysis, uri string, byFile map[*source.File][]fileEdit, newName string) error {
	overlay := map[string]string{}
	for k, v := range s.overlay {
		overlay[k] = v
	}
	for f, edits := range byFile {
		overlay[sema.OverlayKey(f.Path)] = applyEdits(f.Content, edits)
	}
	d := s.docs[uri]
	if d == nil {
		return fmt.Errorf("the document is not open")
	}
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackageOverlay(d.path, diags, overlay)
	if err != nil {
		return err
	}
	if diags.HasErrors() {
		return fmt.Errorf("renaming to '%s' would not parse: %s", newName, firstError(diags))
	}
	ix := sema.CheckIndex(pkg, diags)
	if n := countErrors(diags); n > a.errors {
		return fmt.Errorf("renaming to '%s' would break the program: %s", newName, firstNewError(diags, a.errorMsgs))
	}

	// where each old offset lands after the edits
	editsByPath := map[string][]fileEdit{}
	for f, edits := range byFile {
		sort.Slice(edits, func(i, j int) bool { return edits[i].span.Start < edits[j].span.Start })
		editsByPath[sema.OverlayKey(f.Path)] = edits
	}
	shift := func(k spanKey) (spanKey, bool) {
		start, end := k.start, k.end
		for _, e := range editsByPath[k.path] {
			delta := len(e.newText) - (e.span.End - e.span.Start)
			if e.pun && e.span.Start <= k.start && k.end <= e.span.End {
				return k, false // a spelled-out pun: its two names moved apart
			}
			if e.span.End <= k.start {
				start += delta
			}
			if e.span.End <= k.end {
				end += delta
			}
		}
		return spanKey{k.path, start, end}, true
	}
	type pair struct{ use, def spanKey }
	after := map[pair]bool{}
	for i := range ix.Refs {
		r := &ix.Refs[i]
		if r.Span.IsValid() && r.Def.IsValid() {
			after[pair{keyOf(r.Span), keyOf(r.Def)}] = true
		}
	}
	for i := range a.index.Refs {
		r := &a.index.Refs[i]
		if !r.Span.IsValid() || !r.Def.IsValid() || r.Span.Start == r.Span.End || a.files[sema.OverlayKey(r.Span.File.Path)] == nil {
			continue
		}
		use, ok1 := shift(keyOf(r.Span))
		def, ok2 := shift(keyOf(r.Def))
		if !ok1 || !ok2 {
			continue
		}
		if !after[pair{use, def}] {
			line, col := r.Span.File.Position(r.Span.Start)
			return fmt.Errorf("renaming to '%s' would change what '%s' at %s:%d:%d refers to", newName, r.Name, displayPath(r.Span.File.Path), line, col)
		}
	}
	return nil
}

func applyEdits(text string, edits []fileEdit) string {
	sorted := append([]fileEdit(nil), edits...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].span.Start < sorted[j].span.Start })
	var sb strings.Builder
	at := 0
	for _, e := range sorted {
		sb.WriteString(text[at:e.span.Start])
		sb.WriteString(e.newText)
		at = e.span.End
	}
	sb.WriteString(text[at:])
	return sb.String()
}

func countErrors(diags *source.Diagnostics) int {
	n := 0
	for _, it := range diags.Items {
		if it.Severity == source.Error {
			n++
		}
	}
	return n
}

func firstError(diags *source.Diagnostics) string {
	for _, it := range diags.Items {
		if it.Severity == source.Error {
			return it.Message
		}
	}
	return ""
}

// firstNewError returns the first error message that was not reported
// before the edit (by count, so a repeated message is still new).
func firstNewError(diags *source.Diagnostics, before []string) string {
	left := map[string]int{}
	for _, m := range before {
		left[m]++
	}
	for _, it := range diags.Items {
		if it.Severity != source.Error {
			continue
		}
		if left[it.Message] > 0 {
			left[it.Message]--
			continue
		}
		return it.Message
	}
	return firstError(diags)
}

func displayPath(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}

// workspaceSymbols answers Ctrl+T: the declarations of every analysed
// package (not the standard library) whose name contains the query's
// letters in order, case-insensitively — `nf` finds `notFound`.
func (s *Server) workspaceSymbols(params json.RawMessage) any {
	var p struct {
		Query string `json:"query"`
	}
	json.Unmarshal(params, &p)
	out := []map[string]any{}
	seen := map[string]bool{}
	var add func(sym docSymbol, container, uri string)
	add = func(sym docSymbol, container, uri string) {
		if subsequence(strings.ToLower(p.Query), strings.ToLower(sym.Name)) {
			key := uri + "#" + fmt.Sprint(sym.SelectionRange.Start)
			if !seen[key] {
				seen[key] = true
				item := map[string]any{"name": sym.Name, "kind": sym.Kind,
					"location": map[string]any{"uri": uri, "range": sym.SelectionRange}}
				if container != "" {
					item["containerName"] = container
				}
				out = append(out, item)
			}
		}
		for _, child := range sym.Children {
			add(child, sym.Name, uri)
		}
	}
	for _, a := range s.analyses {
		for _, m := range a.pkg.Modules {
			if m.Std && m != a.pkg.Given {
				continue
			}
			for _, f := range m.Files {
				uri := s.uriFor(f.Source.Path)
				for _, decl := range f.Decls {
					if sym, ok := declSymbol(decl); ok {
						add(sym, m.Name(), uri)
					}
				}
			}
		}
	}
	return out
}

// subsequence reports whether the letters of q appear in s in order.
func subsequence(q, s string) bool {
	for _, r := range q {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i+len(string(r)):]
	}
	return true
}
