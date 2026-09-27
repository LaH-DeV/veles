package sema

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// lintInlinableImpl warns about a top-level `impl Trait for T` that could
// be written inside T's body (D23): T is a struct of the same module and
// the impl adds nothing the inline form cannot say — it uses the struct's
// own type parameters with no bounds beyond the struct's. The warning
// carries the edit that moves it, for editors to offer as a quick fix.
func (c *Checker) lintInlinableImpl(m *Module, f *ast.File, d *ast.ImplDecl, impl *Impl) {
	if d.Inline || d.Extend || d.Trait == nil || len(d.Attrs) > 0 {
		return
	}
	if m.Std && m != c.pkg.Given {
		return // embedded std: not the user's code to fix
	}
	st, ok := impl.Target.(*types.Struct)
	if !ok {
		return
	}
	tmpl := templateOf(st)
	if tmpl.Module != m.prefix() {
		return
	}
	sd, ok := tmpl.Decl.(*ast.StructDecl)
	if !ok || sd.Error || sd.ErrorImpl == d || sd.Extern {
		return
	}
	// the target must be the struct applied to the impl's own parameters,
	// in order: `impl<A, B> Trait for Pair<A, B>`, not `Pair<A, A>` or `Pair<i64, B>`
	if len(impl.TypeParams) != len(tmpl.TypeParams) || len(st.TypeArgs) != len(impl.TypeParams) {
		return
	}
	for i, tp := range impl.TypeParams {
		arg, ok := st.TypeArgs[i].(*types.TypeParam)
		if !ok || arg != tp {
			return
		}
		for _, b := range tp.Bounds {
			if !containsTrait(tmpl.TypeParams[i].Bounds, b) {
				return // a bound the struct does not have: only the top-level form can say it
			}
		}
	}
	// the struct needs a braced body to receive the block
	sf := fileOf(m, sd)
	if sf == nil || sd.Pos.End == 0 || sd.Pos.End > len(sf.Source.Content) || sf.Source.Content[sd.Pos.End-1] != '}' {
		return
	}
	fix := inlineImplFix(f, sf, d, sd, impl.Trait.Name)
	c.warnFix(d.Pos, fix, "implement of '%s' for '%s' can be written inside the body of '%s' as 'implement %s { ... }' (D23)", impl.Trait.Name, tmpl.Name, tmpl.Name, impl.Trait.Name)
}

func containsTrait(ts []*types.Trait, t *types.Trait) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

// fileOf finds the file of m that holds a declaration.
func fileOf(m *Module, d ast.Decl) *ast.File {
	for _, f := range m.Files {
		for _, x := range f.Decls {
			if x == d {
				return f
			}
		}
	}
	return nil
}

// inlineImplFix builds the edits that delete the top-level impl and append
// `impl Trait { body }` to the struct's body, re-indented by one level.
func inlineImplFix(implFile, structFile *ast.File, d *ast.ImplDecl, sd *ast.StructDecl, traitName string) *source.Fix {
	src := implFile.Source.Content
	// the block's body text: between its braces
	open := strings.Index(src[d.Pos.Start:d.Pos.End], "{")
	body := ""
	if open >= 0 {
		body = src[d.Pos.Start+open+1 : d.Pos.End-1]
	}
	body = strings.Trim(body, "\r\n")
	body = strings.TrimRight(body, " \t")
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, "  "+line)
	}
	block := "\n  implement " + traitName + " {\n" + strings.Join(lines, "\n") + "\n  }\n"
	if strings.TrimSpace(body) == "" {
		block = "\n  implement " + traitName + " { }\n"
	}
	// delete the declaration with the line break that follows it, and one
	// blank line before it when there is one
	delStart, delEnd := d.Pos.Start, d.Pos.End
	for delEnd < len(src) && (src[delEnd] == '\r' || src[delEnd] == '\n') {
		delEnd++
		if src[delEnd-1] == '\n' {
			break
		}
	}
	if n := blankLineBefore(src, delStart); n > 0 {
		delStart -= n
	}
	insertAt := sd.Pos.End - 1 // before the closing brace
	ssrc := structFile.Source.Content
	// keep a blank line between the last member and the block when the body
	// is not empty
	if strings.TrimSpace(ssrc[bodyOpen(ssrc, sd)+1:insertAt]) != "" {
		block = "\n" + block
	}
	// the closing brace is on its own line after the block; drop the
	// whitespace that preceded it
	ws := insertAt
	for ws > 0 && (ssrc[ws-1] == ' ' || ssrc[ws-1] == '\t') {
		ws--
	}
	if ws > 0 && ssrc[ws-1] == '\n' {
		insertAt = ws
		block = strings.TrimPrefix(block, "\n")
	}
	return &source.Fix{
		Title: "Move into the body of '" + sd.Name.Name + "'",
		Edits: []source.TextEdit{
			{Span: source.Span{File: implFile.Source, Start: delStart, End: delEnd}},
			{Span: source.Span{File: structFile.Source, Start: insertAt, End: insertAt}, NewText: block},
		},
	}
}

// bodyOpen finds the `{` that opens a struct body.
func bodyOpen(src string, sd *ast.StructDecl) int {
	i := strings.Index(src[sd.Name.Pos.End:sd.Pos.End], "{")
	if i < 0 {
		return sd.Pos.End - 1
	}
	return sd.Name.Pos.End + i
}

// blankLineBefore returns how many bytes of whitespace form one blank line
// (and the line break ending the previous line) directly before pos.
func blankLineBefore(src string, pos int) int {
	i := pos
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	if i == 0 || src[i-1] != '\n' {
		return 0
	}
	j := i - 1
	if j > 0 && src[j-1] == '\r' {
		j--
	}
	for j > 0 && (src[j-1] == ' ' || src[j-1] == '\t') {
		j--
	}
	if j > 0 && src[j-1] == '\n' {
		return pos - j
	}
	return 0
}

// implementHint finishes a "does not implement" message for a struct the
// program declares: where the implement goes. Inside the struct's body is
// the form the inline-implement lint asks for, so that is what it names.
// Nothing for std's types, whose implements are not the reader's to write.
func implementHint(t types.Type, trait *types.Trait) string {
	st, ok := t.(*types.Struct)
	if !ok {
		return ""
	}
	if st.Template != nil {
		st = st.Template
	}
	if st.Module == "std" || strings.HasPrefix(st.Module, "std.") || strings.HasPrefix(st.Module, "<") {
		return ""
	}
	if trait.Module == "std.prelude" && (trait.Name == "Encodable" || trait.Name == "Decodable" || trait.Name == "Comparable") {
		// the compiler writes the body (D58)
		return fmt.Sprintf("; add 'implement %s' inside 'struct %s' — the compiler derives it", trait.Name, st.Name)
	}
	return fmt.Sprintf("; add 'implement %s { ... }' inside 'struct %s'", trait.Name, st.Name)
}

// implementFix answers "type 'P' does not implement trait 'Encodable'"
// with the one line that makes it true when the compiler can write the
// rest (D58): `implement Encodable` inside the struct's body, derived.
// Only for a struct of the user's own source and a derivable trait; any
// other trait needs methods only the author can write.
func (c *Checker) implementFix(t types.Type, trait *types.Trait) *source.Fix {
	st, ok := t.(*types.Struct)
	if !ok || c.derivable(trait) == "" {
		return nil
	}
	d, _ := templateOf(st).Decl.(*ast.StructDecl)
	if d == nil || !d.Pos.IsValid() || d.Pos.File.Embedded || d.Pos.End < 1 {
		return nil
	}
	src := d.Pos.File.Content
	brace := d.Pos.End - 1
	if brace >= len(src) || src[brace] != '}' {
		return nil
	}
	line := strings.LastIndexByte(src[:brace], '\n') + 1
	at, text := brace, "\n  implement "+trait.Name+"\n"
	if strings.TrimSpace(src[line:brace]) == "" {
		at, text = line, "  implement "+trait.Name+"\n" // the brace has a line of its own
	}
	fix := fixReplace("Add 'implement "+trait.Name+"' to 'struct "+st.Name+"' (derived)", source.Span{File: d.Pos.File, Start: at, End: at}, text)
	return fix
}
