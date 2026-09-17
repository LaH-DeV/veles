// Package format is the Veles source formatter (`veles fmt`).
//
// It prints the syntax tree back to source in a canonical style, the way
// gofmt does: spacing, indentation and operator layout are normalised;
// line structure that the author chose is kept where it carries intent —
// whether an argument list or literal was written on one line or several,
// blank lines between statements (at most one), and continuation lines of
// method chains and long conditions. A braced block always breaks onto its
// own lines, as in prettier. Comments are preserved from
// the original text and re-attached by position.
//
// Parentheses are not stored in the tree, so they are regenerated from
// operator precedence: the output has exactly the parentheses the grammar
// needs. Every literal is copied verbatim from the source.
package format

import (
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// Options are the style knobs, read from `[format]` in veles.toml.
type Options struct {
	// Indent is one level of indentation; "  " (two spaces) by default.
	Indent string
	// MaxBlankLines caps consecutive blank lines kept from the source; 1
	// by default.
	MaxBlankLines int
}

// Default is the style used when a package sets nothing.
var Default = Options{Indent: "  ", MaxBlankLines: 1}

func (o Options) fill() Options {
	if o.Indent == "" {
		o.Indent = Default.Indent
	}
	if o.MaxBlankLines <= 0 {
		o.MaxBlankLines = Default.MaxBlankLines
	}
	return o
}

// Source formats one file. When the file does not parse, the diagnostics
// carry the errors and the text comes back unchanged: a formatter must
// never rewrite code it did not understand.
func Source(file *source.File, opts Options) (string, *source.Diagnostics) {
	diags := &source.Diagnostics{}
	f, layout := parser.ParseFileLayout(file, diags)
	if diags.HasErrors() {
		return file.Content, diags
	}
	return Print(f, layout, opts), diags
}

// Print renders a parsed file with the layout the parser collected.
func Print(f *ast.File, layout *parser.Layout, opts Options) string {
	p := &printer{src: f.Source.Content, opts: opts.fill(), comments: layout.Comments, parens: layout.Parens, lastEnd: -1}
	p.file(f)
	return alignColumns(p.out.String(), p.marks)
}

// ---------------------------------------------------------------------------
// output

type printer struct {
	src      string
	opts     Options
	comments []lexer.Comment
	ci       int // next comment to place
	parens   map[[2]int]bool // expressions the author parenthesised
	marks    []mark // alignment columns in the output (see alignColumns)

	out       strings.Builder
	indent    int
	pendingNL int  // newlines owed before the next text
	lineEmpty bool // nothing but indentation on the current output line
	lastEnd   int  // source offset just past the last node or comment printed

	// cont marks that the current statement has broken onto continuation
	// lines: the indent was raised by one and is restored when the
	// statement ends.
	cont bool
}

// w writes text, first paying any newlines owed and indenting the new line.
func (p *printer) w(s string) {
	if s == "" {
		return
	}
	if p.pendingNL > 0 {
		for i := 0; i < p.pendingNL; i++ {
			p.out.WriteByte('\n')
		}
		p.pendingNL = 0
		p.lineEmpty = true
	}
	if p.lineEmpty {
		for i := 0; i < p.indent; i++ {
			p.out.WriteString(p.opts.Indent)
		}
		p.lineEmpty = false
	}
	p.out.WriteString(s)
}

// nl ends the current line; consecutive calls do not add blank lines.
func (p *printer) nl() {
	if p.pendingNL == 0 && p.out.Len() > 0 {
		p.pendingNL = 1
	}
}

// raw copies a span of the source verbatim (literals).
func (p *printer) raw(sp source.Span) {
	p.w(p.src[sp.Start:sp.End])
}

func (p *printer) hasNewline(from, to int) bool {
	if from < 0 || to > len(p.src) || from >= to {
		return false
	}
	return strings.Contains(p.src[from:to], "\n")
}

func (p *printer) newlines(from, to int) int {
	if from < 0 || to > len(p.src) || from >= to {
		return 0
	}
	return strings.Count(p.src[from:to], "\n")
}

// before is called before printing a node that starts at pos: it places
// the comments that precede it and keeps a blank line the author left.
func (p *printer) before(pos int) {
	p.flushComments(pos)
	p.keepBlank(pos)
}

// keepBlank turns a blank line in the source before pos into one in the
// output, when the output is already breaking the line there.
func (p *printer) keepBlank(pos int) {
	if p.lastEnd >= 0 && p.pendingNL >= 1 && p.newlines(p.lastEnd, pos) >= 2 {
		p.pendingNL = p.opts.MaxBlankLines + 1
	}
}

// after records that everything up to pos has been printed.
func (p *printer) after(pos int) {
	if pos > p.lastEnd {
		p.lastEnd = pos
	}
}

// flushComments prints every not-yet-placed comment that starts before
// limit. A comment on the same source line as the previous node trails it;
// any other comment gets its own line(s), keeping a blank line above it.
func (p *printer) flushComments(limit int) {
	for p.ci < len(p.comments) && p.comments[p.ci].Span.Start < limit {
		c := p.comments[p.ci]
		p.ci++
		sameLine := p.lastEnd >= 0 && !p.hasNewline(p.lastEnd, c.Span.Start)
		if sameLine && p.out.Len() > 0 {
			if p.pendingNL > 0 {
				// trailing: sits at the end of the line just finished
				p.mark(alignComment)
				p.out.WriteString("  " + c.Text)
			} else {
				p.w(" " + c.Text)
				if strings.HasPrefix(c.Text, "//") {
					p.nl()
				} else {
					p.out.WriteByte(' ')
				}
			}
			p.after(c.Span.End)
			continue
		}
		p.nl()
		p.keepBlank(c.Span.Start)
		p.commentText(c.Text)
		p.nl()
		p.after(c.Span.End)
	}
}

// commentText writes a comment at the current indentation; a multi-line
// block comment keeps its inner lines as written, minus common leading
// whitespace, so `/* ... */` boxes survive.
func (p *printer) commentText(text string) {
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		p.w(text)
		return
	}
	// re-indent: strip the indentation the block had in the source from
	// every line after the first, keeping relative structure (a ` *`
	// gutter, say)
	min := -1
	for _, l := range lines[1:] {
		t := strings.TrimLeft(l, " \t")
		if t == "" {
			continue
		}
		n := len(l) - len(t)
		if min < 0 || n < min {
			min = n
		}
	}
	p.w(strings.TrimRight(lines[0], " \t\r"))
	for _, l := range lines[1:] {
		p.out.WriteByte('\n')
		l = strings.TrimRight(l, " \t\r")
		if min > 0 && len(l) >= min {
			l = l[min:]
		}
		p.lineEmpty = true
		if l != "" {
			p.w(" " + l)
		}
	}
}

// startsWith reports whether the source at pos begins with the keyword.
func (p *printer) startsWith(pos int, kw string) bool {
	if pos < 0 || pos+len(kw) > len(p.src) || p.src[pos:pos+len(kw)] != kw {
		return false
	}
	if pos+len(kw) == len(p.src) {
		return true
	}
	c := p.src[pos+len(kw)]
	return !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9')
}

// breakCont starts a continuation line for the current statement.
func (p *printer) breakCont() {
	if !p.cont {
		p.cont = true
		p.indent++
	}
	p.nl()
}

// nested runs f as a statement-level unit of its own: a continuation
// indent it opens is closed afterwards, and one the enclosing statement
// opened does not leak into it.
func (p *printer) nested(f func()) {
	cont, ind := p.cont, p.indent
	p.cont = false
	f()
	p.cont, p.indent = cont, ind
}

// ---------------------------------------------------------------------------
// file and declarations

func (p *printer) file(f *ast.File) {
	// `error Name { }` desugars to a struct plus an impl the parser appends
	// to the file; the impl is printed as part of the error declaration
	synth := map[*ast.ImplDecl]bool{}
	for _, d := range f.Decls {
		if sd, ok := d.(*ast.StructDecl); ok && sd.ErrorImpl != nil {
			synth[sd.ErrorImpl] = true
		}
	}
	for _, d := range f.Decls {
		if impl, ok := d.(*ast.ImplDecl); ok && synth[impl] {
			continue
		}
		p.before(declStart(d))
		p.nested(func() { p.decl(d) })
		p.after(d.Span().End)
		p.nl()
	}
	p.flushComments(len(p.src) + 1)
	// exactly one newline at the end; trailing blank lines are dropped
	if p.out.Len() > 0 {
		p.out.WriteByte('\n')
	}
	p.pendingNL = 0
}

// declStart is where a declaration begins in the source, attributes
// included (their span is not part of the declaration's).
func declStart(d ast.Node) int {
	var attrs []*ast.Attribute
	switch d := d.(type) {
	case *ast.FunDecl:
		attrs = d.Attrs
	case *ast.StructDecl:
		attrs = d.Attrs
	case *ast.TraitDecl:
		attrs = d.Attrs
	case *ast.ImplDecl:
		attrs = d.Attrs
	case *ast.ValDecl:
		attrs = d.Attrs
	case *ast.ErrorAliasDecl:
		attrs = d.Attrs
	}
	if len(attrs) > 0 {
		return attrs[0].Pos.Start
	}
	return d.Span().Start
}

func (p *printer) attrs(attrs []*ast.Attribute) {
	for _, a := range attrs {
		p.before(a.Pos.Start)
		p.w("@" + a.Name.Name)
		if a.Args != nil {
			p.w("(")
			p.args(a.Args, a.Pos)
			p.w(")")
		}
		p.after(a.Pos.End)
		p.nl()
	}
}

func (p *printer) decl(d ast.Decl) {
	switch d := d.(type) {
	case *ast.UseDecl:
		p.useDecl(d)
	case *ast.FunDecl:
		p.fun(d)
	case *ast.StructDecl:
		p.structDecl(d)
	case *ast.TraitDecl:
		p.traitDecl(d)
	case *ast.ImplDecl:
		p.implDecl(d)
	case *ast.ValDecl:
		p.attrs(d.Attrs)
		if d.Pub {
			p.w("pub ")
		}
		p.w(d.Kind.String() + " " + d.Name.Name)
		if d.Type != nil {
			p.w(": ")
			p.typ(d.Type)
		}
		if d.Value != nil {
			p.w(" = ")
			p.initExpr(d.Value)
		}
	case *ast.ExternBlock:
		p.w("extern \"" + d.ABI + "\" ")
		p.members(p.openBrace(d.Pos.Start, d.Pos.End), d.Pos.End, len(d.Funs) == 0, func(i int) bool { return i < len(d.Funs) }, func(i int) {
			fn := d.Funs[i]
			p.before(fn.Pos.Start)
			p.fun(fn)
			p.after(fn.Pos.End)
		})
	case *ast.ErrorAliasDecl:
		p.attrs(d.Attrs)
		if d.Pub {
			p.w("pub ")
		}
		p.w("error " + d.Name.Name + " = ")
		p.typ(d.Members)
	case *ast.BadDecl:
		p.raw(d.Pos)
	}
}

func (p *printer) useDecl(d *ast.UseDecl) {
	p.w("use ")
	for i, seg := range d.Path {
		if i > 0 {
			p.w(".")
		}
		p.w(seg.Name)
	}
	if d.Items != nil {
		p.w(".{ ")
		for i, it := range d.Items {
			if i > 0 {
				p.w(", ")
			}
			p.w(it.Name.Name)
			if it.Alias != nil {
				p.w(" as " + it.Alias.Name)
			}
		}
		p.w(" }")
	}
	if d.Alias != nil {
		p.w(" as " + d.Alias.Name)
	}
}

// modifiers prints a function's modifiers in the order the author used.
func (p *printer) modifiers(fn *ast.FunDecl) {
	head := p.src[fn.Pos.Start:fn.Name.Pos.Start]
	for _, kw := range strings.Fields(head) {
		switch kw {
		case "pub", "mut", "override", "unsafe":
			p.w(kw + " ")
		}
	}
}

func (p *printer) fun(fn *ast.FunDecl) {
	p.attrs(fn.Attrs)
	p.modifiers(fn)
	p.w("fun ")
	// `fun <T> name` and `fun name<T>` are both accepted; keep the author's
	head := p.src[fn.Pos.Start:fn.Name.Pos.Start]
	prefixTP := strings.Contains(head, "<")
	if prefixTP {
		p.typeParams(fn.TypeParams)
		p.w(" ")
	}
	p.w(fn.Name.Name)
	if !prefixTP {
		p.typeParams(fn.TypeParams)
	}
	p.params(fn.Params, fn.Name.Pos.End, fn)
	if fn.Ret != nil {
		p.w(": ")
		p.typ(fn.Ret)
	}
	p.effects(fn.Effects)
	switch {
	case fn.Body != nil:
		p.w(" ")
		p.block(fn.Body)
	case fn.ExprBody != nil:
		p.w(" =")
		if p.hasNewline(exprBodyEq(p, fn), fn.ExprBody.Span().Start) {
			p.breakCont()
		} else {
			p.w(" ")
		}
		p.expr(fn.ExprBody, 0)
	}
}

// exprBodyEq finds the `=` that introduces an expression body.
func exprBodyEq(p *printer, fn *ast.FunDecl) int {
	i := fn.ExprBody.Span().Start
	for i > fn.Name.Pos.End && p.src[i] != '=' {
		i--
	}
	return i
}

func (p *printer) effects(e ast.Effects) {
	if e.Suspends {
		p.w(" suspends")
	}
	if e.Throws {
		p.w(" throws")
		if e.Error != nil {
			p.w(" ")
			p.typ(e.Error)
		}
	}
}

func (p *printer) typeParams(tps []ast.TypeParam) {
	if len(tps) == 0 {
		return
	}
	p.w("<")
	for i, tp := range tps {
		if i > 0 {
			p.w(", ")
		}
		p.w(tp.Name.Name)
		p.bounds(tp.Bounds)
	}
	p.w(">")
}

func (p *printer) bounds(bs []ast.Type) {
	for i, b := range bs {
		if i == 0 {
			p.w(": ")
		} else {
			p.w(" + ")
		}
		p.typ(b)
	}
}

// params prints a parameter list, one per line when the author broke it.
func (p *printer) params(params []ast.Param, open int, owner ast.Node) {
	p.w("(")
	if len(params) == 0 {
		p.w(")")
		return
	}
	multi := p.listBroken(open, params[0].Pos.Start, func(i int) (int, int) {
		return params[i].Pos.End, params[i+1].Pos.Start
	}, len(params))
	if multi {
		p.after(open + 1)
	}
	for i, prm := range params {
		if multi {
			if i == 0 {
				p.indent++
			}
			p.nl()
			p.before(prm.Pos.Start)
		} else if i > 0 {
			p.w(", ")
		}
		p.w(prm.Name.Name)
		if prm.Type != nil {
			p.w(": ")
			p.typ(prm.Type)
		}
		if prm.Default != nil {
			p.w(" = ")
			p.expr(prm.Default, 0)
		}
		if multi {
			p.w(",")
			p.after(prm.Pos.End)
		}
	}
	if multi {
		p.indent--
		p.nl()
		p.flushComments(owner.Span().End)
	}
	p.w(")")
}

// listBroken reports whether a bracketed list was written across lines:
// a newline between the opening bracket and the first element, or between
// two elements (a newline inside an element, such as a lambda body, does
// not count).
func (p *printer) listBroken(open, first int, gap func(i int) (int, int), n int) bool {
	if p.hasNewline(open, first) {
		return true
	}
	for i := 0; i < n-1; i++ {
		a, b := gap(i)
		if p.hasNewline(a, b) {
			return true
		}
	}
	return false
}

// members prints a `{ ... }` body of declarations, one per line with blank
// lines kept; an empty body is `{ }`.
func (p *printer) members(open, close int, empty bool, has func(i int) bool, member func(i int)) {
	if open < 0 {
		return
	}
	if empty {
		p.emptyBody(close)
		return
	}
	p.w("{")
	p.after(open + 1)
	p.indent++
	for i := 0; has(i); i++ {
		p.nl()
		p.nested(func() { member(i) })
		p.nl()
	}
	p.indent--
	p.flushComments(close - 1)
	p.nl()
	p.w("}")
}

// emptyBody prints `{ }`, or a brace pair around the comments the body
// holds; close is the offset just past the closing brace.
func (p *printer) emptyBody(close int) {
	p.w("{")
	p.indent++
	p.flushComments(close - 1)
	p.indent--
	if p.pendingNL > 0 {
		p.nl()
		p.w("}")
		return
	}
	p.w(" }")
}

// openBrace finds the `{` that opens a body, searching from `from`.
func (p *printer) openBrace(from, to int) int {
	if from < 0 || to > len(p.src) || from > to {
		return -1
	}
	i := strings.Index(p.src[from:to], "{")
	if i < 0 {
		return -1
	}
	return from + i
}

type memberRef struct {
	pos   int
	print func()
}

func (p *printer) sortedMembers(ms []memberRef) []memberRef {
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].pos < ms[j].pos })
	return ms
}

func (p *printer) structDecl(d *ast.StructDecl) {
	p.attrs(d.Attrs)
	if d.Pub {
		p.w("pub ")
	}
	switch {
	case d.Error:
		p.w("error ")
	case d.Extern:
		p.w("extern struct ")
	default:
		p.w("struct ")
	}
	p.w(d.Name.Name)
	p.typeParams(d.TypeParams)
	if d.Variant != nil {
		p.w(" : ")
		p.typ(d.Variant)
	}
	var ms []memberRef
	for _, f := range d.Fields {
		f := f
		ms = append(ms, memberRef{f.Pos.Start, func() { p.field(f) }})
	}
	methods := d.Methods
	if d.ErrorImpl != nil {
		// `message` lives in the desugared impl; a forwarding method the
		// parser made from a `message: string` field is not source
		for _, m := range d.ErrorImpl.Methods {
			synthetic := false
			for _, f := range d.Fields {
				if m.Pos.Start == f.Name.Pos.Start {
					synthetic = true
				}
			}
			if !synthetic {
				methods = append(methods, m)
			}
		}
	}
	for _, m := range methods {
		m := m
		ms = append(ms, memberRef{declStart(m), func() { p.fun(m) }})
	}
	ms = p.sortedMembers(ms)
	hdr := d.Name.Pos.End
	if d.Variant != nil {
		hdr = d.Variant.Span().End
	}
	open := p.openBrace(hdr, d.Pos.End)
	if open < 0 {
		return // `struct Unit` with no body
	}
	p.w(" ")
	p.members(open, d.Pos.End, len(ms) == 0, func(i int) bool { return i < len(ms) }, func(i int) {
		p.before(ms[i].pos)
		ms[i].print()
		p.after(memberEnd(d, ms[i].pos))
	})
}

// memberEnd finds the end of the member starting at pos within a struct.
func memberEnd(d *ast.StructDecl, pos int) int {
	for _, f := range d.Fields {
		if f.Pos.Start == pos {
			return f.Pos.End
		}
	}
	for _, m := range d.Methods {
		if declStart(m) == pos {
			return m.Pos.End
		}
	}
	if d.ErrorImpl != nil {
		for _, m := range d.ErrorImpl.Methods {
			if declStart(m) == pos {
				return m.Pos.End
			}
		}
	}
	return pos
}

func (p *printer) field(f *ast.Field) {
	if f.Pub {
		p.w("pub ")
	}
	p.w(f.Name.Name + ":")
	p.mark(alignField)
	p.w(" ")
	p.typ(f.Type)
	if f.Default != nil {
		p.w(" = ")
		p.expr(f.Default, 0)
	}
}

func (p *printer) traitDecl(d *ast.TraitDecl) {
	p.attrs(d.Attrs)
	if d.Pub {
		p.w("pub ")
	}
	if d.Sealed {
		p.w("sealed ")
	}
	p.w("trait " + d.Name.Name)
	p.typeParams(d.TypeParams)
	for i, s := range d.Supers {
		if i == 0 {
			p.w(" : ")
		} else {
			p.w(" + ")
		}
		p.typ(s)
	}
	var ms []memberRef
	ends := map[int]int{}
	for _, at := range d.AssocTypes {
		at := at
		ends[at.Pos.Start] = at.Pos.End
		ms = append(ms, memberRef{at.Pos.Start, func() {
			p.w("type " + at.Name.Name)
			p.bounds(at.Bounds)
		}})
	}
	for _, m := range d.Methods {
		m := m
		ends[declStart(m)] = m.Pos.End
		ms = append(ms, memberRef{declStart(m), func() { p.fun(m) }})
	}
	ms = p.sortedMembers(ms)
	hdr := d.Name.Pos.End
	if n := len(d.Supers); n > 0 {
		hdr = d.Supers[n-1].Span().End
	}
	open := p.openBrace(hdr, d.Pos.End)
	if open < 0 {
		return
	}
	p.w(" ")
	p.members(open, d.Pos.End, len(ms) == 0, func(i int) bool { return i < len(ms) }, func(i int) {
		p.before(ms[i].pos)
		ms[i].print()
		p.after(ends[ms[i].pos])
	})
}

func (p *printer) implDecl(d *ast.ImplDecl) {
	p.attrs(d.Attrs)
	if d.Extend {
		p.w("extend")
		p.typeParams(d.TypeParams)
		p.w(" ")
		p.typ(d.Target)
	} else {
		p.w("impl")
		p.typeParams(d.TypeParams)
		p.w(" ")
		p.typ(d.Trait)
		p.w(" for ")
		p.typ(d.Target)
	}
	var ms []memberRef
	ends := map[int]int{}
	for _, b := range d.AssocTypes {
		b := b
		// the binding has no span of its own: locate `type Name` in the body
		pos := strings.Index(p.src[d.Pos.Start:d.Pos.End], "type "+b.Name.Name)
		if pos < 0 {
			pos = 0
		}
		start := d.Pos.Start + pos
		ends[start] = b.Type.Span().End
		ms = append(ms, memberRef{start, func() {
			p.w("type " + b.Name.Name + " = ")
			p.typ(b.Type)
		}})
	}
	for _, m := range d.Methods {
		m := m
		ends[declStart(m)] = m.Pos.End
		ms = append(ms, memberRef{declStart(m), func() { p.fun(m) }})
	}
	ms = p.sortedMembers(ms)
	p.w(" ")
	p.members(p.openBrace(d.Target.Span().End, d.Pos.End), d.Pos.End, len(ms) == 0, func(i int) bool { return i < len(ms) }, func(i int) {
		p.before(ms[i].pos)
		ms[i].print()
		p.after(ends[ms[i].pos])
	})
}

// ---------------------------------------------------------------------------
// types

func (p *printer) typ(t ast.Type) {
	switch t := t.(type) {
	case *ast.NamedType:
		for i, seg := range t.Path {
			if i > 0 {
				p.w(".")
			}
			p.w(seg.Name)
		}
		if len(t.Args) > 0 {
			p.w("<")
			p.typeList(t.Args)
			p.w(">")
		}
	case *ast.NullableType:
		// `(*T)?` is a nullable pointer; `*T?` would be a pointer to a nullable
		if _, ptr := t.Elem.(*ast.PointerType); ptr {
			p.w("(")
			p.typ(t.Elem)
			p.w(")")
		} else {
			p.typ(t.Elem)
		}
		p.w("?")
	case *ast.PointerType:
		p.w("*")
		if t.Raw {
			p.w("raw ")
		}
		p.typ(t.Elem)
	case *ast.TupleType:
		p.w("(")
		p.typeList(t.Elems)
		if len(t.Elems) == 1 {
			p.w(",")
		}
		p.w(")")
	case *ast.FunType:
		p.w("fun(")
		p.typeList(t.Params)
		p.w(")")
		if t.Ret != nil {
			p.w(": ")
			p.typ(t.Ret)
		}
		p.effects(t.Effects)
	case *ast.SelfType:
		p.w("Self")
	case *ast.AssocType:
		p.typ(t.Base)
		p.w("::" + t.Name.Name)
	case *ast.ErrorUnionType:
		for i, m := range t.Members {
			if i > 0 {
				p.w(" | ")
			}
			p.typ(m)
		}
	}
}

func (p *printer) typeList(ts []ast.Type) {
	for i, t := range ts {
		if i > 0 {
			p.w(", ")
		}
		p.typ(t)
	}
}

// ---------------------------------------------------------------------------
// statements

// block prints a `{ }` body, one statement per line (a block always breaks,
// as in prettier; only an empty one is `{ }`). A body written without braces
// (a single statement after `if` or `=>`) stays without them.
func (p *printer) block(b *ast.Block) {
	if !p.braced(b) {
		if len(b.Stmts) == 1 {
			p.stmt(b.Stmts[0])
		}
		return
	}
	if len(b.Stmts) == 0 {
		p.emptyBody(b.Pos.End)
		return
	}
	p.w("{")
	p.after(b.Pos.Start + 1)
	p.indent++
	for _, s := range b.Stmts {
		p.nl()
		p.before(s.Span().Start)
		p.nested(func() { p.stmt(s) })
		p.after(s.Span().End)
		p.nl()
	}
	p.flushComments(b.Pos.End - 1)
	p.indent--
	p.nl()
	p.w("}")
}

// braced reports whether a block was written with `{ }` (the parser wraps
// a bare statement body in a block with the statement's span).
func (p *printer) braced(b *ast.Block) bool {
	return b.Pos.Start < len(p.src) && p.src[b.Pos.Start] == '{'
}

func (p *printer) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.ValStmt:
		p.w(s.Kind.String() + " ")
		p.binding(s.Binding)
		if s.Value != nil {
			p.w(" = ")
			p.initExpr(s.Value)
		}
	case *ast.ExprStmt:
		p.expr(s.X, 0)
	case *ast.AssignStmt:
		p.expr(s.Target, 0)
		p.w(" " + s.Op.String() + " ")
		p.initExpr(s.Value)
	case *ast.ReturnStmt:
		p.w("return")
		if s.Value != nil {
			p.w(" ")
			p.expr(s.Value, 0)
		}
	case *ast.ThrowStmt:
		p.w("throw ")
		p.expr(s.Value, 0)
	case *ast.BreakStmt:
		p.w("break")
		if s.Label != nil {
			p.w(" " + s.Label.Name)
		}
	case *ast.ContinueStmt:
		p.w("continue")
		if s.Label != nil {
			p.w(" " + s.Label.Name)
		}
	case *ast.LoopStmt:
		p.w("loop")
		if s.Label != nil {
			p.w(" :" + s.Label.Name)
		}
		switch {
		case s.Var != nil:
			p.w(" (")
			p.binding(*s.Var)
			p.w(" in ")
			p.expr(s.Iter, 0)
			p.w(")")
		case s.Cond != nil:
			p.w(" (")
			p.expr(s.Cond, 0)
			p.w(")")
		}
		p.w(" ")
		p.block(s.Body)
	case *ast.WithStmt:
		p.w("with (")
		for i, b := range s.Bindings {
			if i > 0 {
				p.w(", ")
			}
			p.w(b.Name.Name + " = ")
			p.expr(b.Value, 0)
		}
		p.w(") ")
		p.block(s.Body)
	case *ast.ScopeStmt:
		p.w("scope ")
		p.block(s.Body)
	case *ast.FunStmt:
		p.fun(s.Fun)
	case *ast.Block:
		p.block(s)
	case *ast.BadStmt:
		p.raw(s.Pos)
	}
}

// initExpr prints the right side of `=`, keeping a line break the author
// put after the `=`.
func (p *printer) initExpr(e ast.Expr) {
	start := e.Span().Start
	eq := start
	for eq > 0 && p.src[eq] != '=' {
		eq--
	}
	if p.hasNewline(eq, start) {
		p.breakCont()
	}
	p.expr(e, 0)
}

func (p *printer) binding(b ast.Binding) {
	if b.Name != nil {
		p.w(b.Name.Name)
		if b.Type != nil {
			p.w(": ")
			p.typ(b.Type)
		}
		return
	}
	p.w("(")
	for i, e := range b.Tuple {
		if i > 0 {
			p.w(", ")
		}
		p.binding(e)
	}
	p.w(")")
}

// ---------------------------------------------------------------------------
// expressions

// Binding powers mirror parser/expr.go; bpPostfix is above every operator
// so that `(-x).abs()` and `(a as f64).floor()` keep their parentheses.
const (
	bpNone = iota
	bpOr
	bpAnd
	bpEq
	bpCmp
	bpNamed
	bpElvis
	bpRange
	bpAdd
	bpMul
	bpCast
	bpUnary
	bpPostfix
)

func infixBp(k lexer.TokenKind) int {
	switch k {
	case lexer.OrOr:
		return bpOr
	case lexer.AndAnd:
		return bpAnd
	case lexer.Eq, lexer.NotEq:
		return bpEq
	case lexer.Lt, lexer.LtEq, lexer.Gt, lexer.GtEq:
		return bpCmp
	case lexer.Plus, lexer.Minus, lexer.WrapPlus, lexer.WrapMinus:
		return bpAdd
	case lexer.Star, lexer.Slash, lexer.Percent, lexer.WrapStar:
		return bpMul
	}
	return bpNone
}

// bp is how tightly an expression holds together: a child printed where
// the context binds tighter than this needs parentheses.
func bp(e ast.Expr) int {
	switch e := e.(type) {
	case *ast.BinaryExpr:
		return infixBp(e.Op)
	case *ast.ElvisExpr:
		return bpElvis
	case *ast.RangeExpr:
		return bpRange
	case *ast.IsExpr:
		return bpNamed
	case *ast.CastExpr:
		return bpCast
	case *ast.UnaryExpr, *ast.TryExpr, *ast.AwaitExpr:
		return bpUnary
	case *ast.CallExpr:
		if e.Async {
			return bpUnary
		}
	case *ast.LambdaExpr, *ast.ControlExpr:
		// open-ended: the body runs to the end of the expression
		return bpNone
	case *ast.IfExpr:
		return bpNone
	}
	return bpPostfix
}

// expr prints e, parenthesised when it binds more loosely than minBp or
// when the author parenthesised it.
func (p *printer) expr(e ast.Expr, minBp int) {
	if bp(e) < minBp || p.authored(e) {
		p.w("(")
		p.exprInner(e)
		p.w(")")
		return
	}
	p.exprInner(e)
}

// exprRight prints a right operand: an open-ended expression (a lambda,
// `if`, `throw`, ...) there runs to the end of the enclosing expression
// exactly as the parser read it, so it needs no parentheses.
func (p *printer) exprRight(e ast.Expr, minBp int) {
	if bp(e) == bpNone && !p.authored(e) {
		p.exprInner(e)
		return
	}
	p.expr(e, minBp)
}

// authored reports whether the author wrote parentheses around e.
func (p *printer) authored(e ast.Expr) bool {
	sp := e.Span()
	return p.parens[[2]int{sp.Start, sp.End}]
}

func (p *printer) exprInner(e ast.Expr) {
	switch e := e.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.CharLit:
		p.raw(e.Span())
	case *ast.BoolLit:
		if e.Value {
			p.w("true")
		} else {
			p.w("false")
		}
	case *ast.NullLit:
		p.w("null")
	case *ast.SelfExpr:
		p.w("self")
	case *ast.NameExpr:
		p.w(e.Name)
	case *ast.MemberExpr:
		if e.X != nil {
			p.expr(e.X, bpPostfix)
			// a chain the author broke onto lines stays broken
			if p.hasNewline(e.X.Span().End, e.Name.Pos.Start) {
				p.breakCont()
			}
		}
		if e.Safe {
			p.w("?.")
		} else {
			p.w(".")
		}
		p.w(e.Name.Name)
	case *ast.IndexExpr:
		p.expr(e.X, bpPostfix)
		p.w("[")
		p.expr(e.Index, 0)
		p.w("]")
	case *ast.CallExpr:
		if e.Async {
			p.w("async ")
		}
		p.expr(e.Fun, bpPostfix)
		if len(e.TypeArgs) > 0 {
			p.w("<")
			p.typeList(e.TypeArgs)
			p.w(">")
		}
		p.w("(")
		p.args(e.Args, e.Pos)
		p.w(")")
	case *ast.UnaryExpr:
		p.w(e.Op.String())
		if _, twice := e.X.(*ast.UnaryExpr); twice {
			// `-(-a)`, never `--a`
			p.w("(")
			p.exprInner(e.X)
			p.w(")")
		} else {
			p.expr(e.X, bpUnary)
		}
	case *ast.TryExpr:
		p.w("try ")
		p.exprRight(e.X, bpUnary)
	case *ast.AwaitExpr:
		p.w("await ")
		p.exprRight(e.X, bpUnary)
	case *ast.BinaryExpr:
		b := infixBp(e.Op)
		p.expr(e.L, b)
		p.operator(e.L.Span().End, e.R.Span().Start, e.Op.String())
		p.exprRight(e.R, b+1)
	case *ast.ElvisExpr:
		// right-associative
		p.expr(e.L, bpElvis+1)
		p.operator(e.L.Span().End, e.R.Span().Start, "?:")
		p.exprRight(e.R, bpElvis)
	case *ast.RangeExpr:
		p.expr(e.Lo, bpRange)
		if e.Inclusive {
			p.w("..")
		} else {
			p.w("..<")
		}
		p.exprRight(e.Hi, bpRange+1)
	case *ast.IsExpr:
		p.expr(e.X, bpNamed)
		if e.Not {
			p.w(" !is ")
		} else {
			p.w(" is ")
		}
		p.typePat(e.Pat, false)
	case *ast.CastExpr:
		p.expr(e.X, bpCast)
		p.w(" as ")
		p.typ(e.Type)
	case *ast.LambdaExpr:
		p.lambda(e)
	case *ast.TupleExpr:
		p.w("(")
		p.exprList(e.Elems, e.Pos, "(")
		if len(e.Elems) == 1 {
			p.w(",")
		}
		p.w(")")
	case *ast.ListLit:
		if e.Mut {
			p.w("mut ")
		}
		p.w("[")
		p.exprList(e.Elems, e.Pos, "[")
		p.w("]")
	case *ast.MapLit:
		if e.Mut {
			p.w("mut ")
		}
		if len(e.Entries) == 0 {
			p.w("[:]")
			return
		}
		p.w("[")
		open := strings.Index(p.src[e.Pos.Start:e.Pos.End], "[") + e.Pos.Start
		multi := p.listBroken(open, e.Entries[0].Key.Span().Start, func(i int) (int, int) {
			return e.Entries[i].Value.Span().End, e.Entries[i+1].Key.Span().Start
		}, len(e.Entries))
		p.brokenList(multi, len(e.Entries), open, e.Pos.End-1, func(i int) (int, int) {
			en := e.Entries[i]
			return en.Key.Span().Start, en.Value.Span().End
		}, func(i int) {
			en := e.Entries[i]
			p.expr(en.Key, 0)
			p.w(": ")
			p.expr(en.Value, 0)
		})
		p.w("]")
	case *ast.IfExpr:
		p.ifExpr(e)
	case *ast.WhenExpr:
		p.whenExpr(e)
	case *ast.BlockExpr:
		p.block(e.Block)
	case *ast.GatherExpr:
		p.w("gather ")
		p.block(e.Body)
	case *ast.UnsafeExpr:
		p.w("unsafe ")
		p.block(e.Body)
	case *ast.RaceExpr:
		p.raceExpr(e)
	case *ast.ControlExpr:
		p.stmt(e.Stmt)
	case *ast.BadExpr:
		p.raw(e.Pos)
	}
}

// operator prints a binary operator, on a continuation line when the
// author started the right operand on a new line (`a\n  ?: b`, `x\n  && y`).
func (p *printer) operator(lend, rstart int, op string) {
	if p.hasNewline(lend, rstart) {
		p.breakCont()
		p.w(op + " ")
		return
	}
	p.w(" " + op + " ")
}

func (p *printer) args(args []ast.Arg, owner source.Span) {
	if len(args) == 0 {
		return
	}
	open := strings.LastIndex(p.src[owner.Start:args[0].Value.Span().Start], "(")
	if args[0].Name != nil {
		open = strings.LastIndex(p.src[owner.Start:args[0].Name.Pos.Start], "(")
	}
	if open < 0 {
		open = 0
	}
	open += owner.Start
	first := args[0].Value.Span().Start
	if args[0].Name != nil {
		first = args[0].Name.Pos.Start
	}
	multi := p.listBroken(open, first, func(i int) (int, int) {
		next := args[i+1].Value.Span().Start
		if args[i+1].Name != nil {
			next = args[i+1].Name.Pos.Start
		}
		return args[i].Value.Span().End, next
	}, len(args))
	p.brokenList(multi, len(args), open, owner.End-1, func(i int) (int, int) {
		s := args[i].Value.Span().Start
		if args[i].Name != nil {
			s = args[i].Name.Pos.Start
		}
		return s, args[i].Value.Span().End
	}, func(i int) {
		if args[i].Name != nil {
			p.w(args[i].Name.Name + ": ")
		}
		p.expr(args[i].Value, 0)
	})
}

func (p *printer) exprList(elems []ast.Expr, owner source.Span, bracket string) {
	if len(elems) == 0 {
		return
	}
	open := strings.Index(p.src[owner.Start:elems[0].Span().Start], bracket) + owner.Start
	multi := p.listBroken(open, elems[0].Span().Start, func(i int) (int, int) {
		return elems[i].Span().End, elems[i+1].Span().Start
	}, len(elems))
	p.brokenList(multi, len(elems), open, owner.End-1, func(i int) (int, int) {
		return elems[i].Span().Start, elems[i].Span().End
	}, func(i int) { p.expr(elems[i], 0) })
}

// brokenList prints n elements either separated by `, ` or, when the author
// broke the list, one per line with a trailing comma and the closing
// bracket on its own line.
func (p *printer) brokenList(multi bool, n, open, close int, span func(i int) (int, int), elem func(i int)) {
	if !multi {
		for i := 0; i < n; i++ {
			if i > 0 {
				p.w(", ")
			}
			elem(i)
		}
		return
	}
	p.after(open + 1)
	p.indent++
	for i := 0; i < n; i++ {
		s, e := span(i)
		p.nl()
		p.before(s)
		p.nested(func() { elem(i) })
		p.w(",")
		p.after(e)
	}
	p.indent--
	p.flushComments(close)
	p.nl()
}

func (p *printer) lambda(e *ast.LambdaExpr) {
	single := len(e.Params) == 1 && e.Params[0].Type == nil && e.Ret == nil && p.src[e.Pos.Start] != '('
	if single {
		p.w(e.Params[0].Name.Name)
	} else {
		p.w("(")
		for i, prm := range e.Params {
			if i > 0 {
				p.w(", ")
			}
			p.w(prm.Name.Name)
			if prm.Type != nil {
				p.w(": ")
				p.typ(prm.Type)
			}
		}
		p.w(")")
		if e.Ret != nil {
			p.w(": ")
			p.typ(e.Ret)
		}
	}
	p.w(" => ")
	if b, ok := e.Body.(*ast.BlockExpr); ok && !p.braced(b.Block) {
		// `x => total += x`: the parser wrapped an assignment body
		if len(b.Block.Stmts) == 1 {
			p.stmt(b.Block.Stmts[0])
		}
		return
	}
	p.expr(e.Body, 0)
}

func (p *printer) ifExpr(e *ast.IfExpr) {
	p.w("if (")
	p.expr(e.Cond, 0)
	p.w(") ")
	p.block(e.Then)
	if e.Else == nil {
		return
	}
	// `else if` is a block holding one IfExpr that starts with `if`
	if len(e.Else.Stmts) == 1 && !p.braced(e.Else) {
		if es, ok := e.Else.Stmts[0].(*ast.ExprStmt); ok {
			if inner, ok := es.X.(*ast.IfExpr); ok && p.startsWith(inner.Pos.Start, "if") {
				p.elseKeyword(e)
				p.ifExpr(inner)
				return
			}
		}
	}
	p.elseKeyword(e)
	p.block(e.Else)
}

// elseKeyword prints ` else `, on a new line when the then-branch had no
// braces and the author put `else` on the next line.
func (p *printer) elseKeyword(e *ast.IfExpr) {
	if !p.braced(e.Then) && p.hasNewline(e.Then.Pos.End, e.Else.Pos.Start) {
		p.nl()
		p.w("else ")
		return
	}
	p.w(" else ")
}

func (p *printer) whenExpr(e *ast.WhenExpr) {
	p.w("when ")
	if e.Subject != nil {
		p.w("(")
		p.expr(e.Subject, 0)
		p.w(") ")
	}
	hdr := e.Pos.Start
	if e.Subject != nil {
		hdr = e.Subject.Span().End
	}
	p.members(p.openBrace(hdr, e.Pos.End), e.Pos.End, len(e.Arms) == 0, func(i int) bool { return i < len(e.Arms) }, func(i int) {
		arm := e.Arms[i]
		p.before(arm.Pos.Start)
		p.whenArm(arm)
		p.after(arm.Pos.End)
	})
}

func (p *printer) whenArm(arm *ast.WhenArm) {
	switch {
	case arm.Else:
		p.w("else")
	case arm.Cond != nil:
		p.expr(arm.Cond, 0)
	default:
		for i, pat := range arm.Patterns {
			if i > 0 {
				p.w(", ")
			}
			p.pattern(pat)
		}
		if arm.Guard != nil {
			p.w(" if ")
			p.expr(arm.Guard, 0)
		}
	}
	p.mark(alignArrow)
	p.w(" => ")
	p.armBody(arm.Body)
}

// armBody prints the right side of `=>`, keeping a break the author put
// after the arrow.
func (p *printer) armBody(body ast.Expr) {
	start := body.Span().Start
	arrow := strings.LastIndex(p.src[:start], "=>")
	if arrow >= 0 && p.hasNewline(arrow, start) {
		p.breakCont()
	}
	p.expr(body, 0)
}

func (p *printer) raceExpr(e *ast.RaceExpr) {
	p.w("race ")
	p.members(p.openBrace(e.Pos.Start, e.Pos.End), e.Pos.End, len(e.Arms) == 0, func(i int) bool { return i < len(e.Arms) }, func(i int) {
		arm := e.Arms[i]
		p.before(arm.Pos.Start)
		if arm.Binding != nil {
			p.w("val ")
			p.binding(*arm.Binding)
			p.w(" = ")
		}
		p.expr(arm.Source, 0)
		p.mark(alignArrow)
		p.w(" => ")
		p.armBody(arm.Body)
		p.after(arm.Pos.End)
	})
}

// ---------------------------------------------------------------------------
// patterns

func (p *printer) pattern(pat ast.Pattern) {
	switch pat := pat.(type) {
	case *ast.WildcardPat:
		p.w("_")
	case *ast.BindPat:
		p.w(pat.Name.Name)
	case *ast.LiteralPat:
		p.expr(pat.Value, 0)
	case *ast.RangePat:
		p.w("in ")
		p.expr(pat.Range, 0)
	case *ast.TypePat:
		p.typePat(pat, true)
	case *ast.TuplePat:
		p.w("(")
		for i, e := range pat.Elems {
			if i > 0 {
				p.w(", ")
			}
			p.pattern(e)
		}
		p.w(")")
	}
}

// typePat prints `is T(fields)`; the `is` is written only when the author
// did (inside a destructuring, `Some(x)` needs none) and mayIs is set.
func (p *printer) typePat(pat *ast.TypePat, mayIs bool) {
	if mayIs && p.startsWith(pat.Pos.Start, "is") {
		p.w("is ")
	}
	if nt, ok := pat.Type.(*ast.NamedType); ok && nt.Pos.Start < len(p.src) && p.src[nt.Pos.Start] == '.' {
		p.w(".") // leading-dot variant: `.none`, `.some(x)`
	}
	p.typ(pat.Type)
	if !pat.HasArg {
		return
	}
	p.w("(")
	for i, f := range pat.Fields {
		if i > 0 {
			p.w(", ")
		}
		if f.Name.Name != "" {
			p.w(f.Name.Name)
			if f.Pat != nil {
				p.w(": ")
			}
		}
		if f.Pat != nil {
			p.pattern(f.Pat)
		}
	}
	p.w(")")
}


// ---------------------------------------------------------------------------
// column alignment

// Runs of consecutive lines that share a structure are aligned on it, as
// gofmt does with a tabwriter: trailing comments, the `=>` of `when` and
// `race` arms, and the types of struct fields.
type alignKind int

const (
	alignField   alignKind = iota // after `name:` in a struct field
	alignArrow                    // before ` => `
	alignComment                  // before a trailing comment
)

type mark struct {
	off  int // byte offset in the output
	kind alignKind
}

// mark records that the current output position is an alignment column.
func (p *printer) mark(kind alignKind) {
	p.marks = append(p.marks, mark{p.out.Len(), kind})
}

// maxAlignSpread caps how far a run may push its column: a run whose
// widest cell exceeds its narrowest by more is left unaligned rather than
// pushed across the page.
const maxAlignSpread = 24

func alignColumns(out string, marks []mark) string {
	if len(marks) == 0 {
		return out
	}
	lines := strings.Split(out, "\n")
	// per line, the column (byte offset) of each kind, or -1
	cols := make([][3]int, len(lines))
	for i := range cols {
		cols[i] = [3]int{-1, -1, -1}
	}
	off, mi := 0, 0
	for i, l := range lines {
		for mi < len(marks) && marks[mi].off <= off+len(l) {
			if marks[mi].off >= off {
				cols[i][marks[mi].kind] = marks[mi].off - off
			}
			mi++
		}
		off += len(l) + 1
	}
	for _, kind := range []alignKind{alignField, alignArrow, alignComment} {
		i := 0
		for i < len(lines) {
			if cols[i][kind] < 0 {
				i++
				continue
			}
			j := i
			width, narrow := 0, -1
			for j < len(lines) && cols[j][kind] >= 0 {
				w := displayWidth(lines[j][:cols[j][kind]])
				if w > width {
					width = w
				}
				if narrow < 0 || w < narrow {
					narrow = w
				}
				j++
			}
			if j-i >= 2 && width-narrow <= maxAlignSpread {
				for k := i; k < j; k++ {
					c := cols[k][kind]
					pad := width - displayWidth(lines[k][:c])
					if pad == 0 {
						continue
					}
					lines[k] = lines[k][:c] + strings.Repeat(" ", pad) + lines[k][c:]
					for other := range cols[k] {
						if cols[k][other] > c {
							cols[k][other] += pad
						}
					}
				}
			}
			i = j
		}
	}
	return strings.Join(lines, "\n")
}

// displayWidth counts characters (not bytes) so that non-ASCII code lines
// align.
func displayWidth(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
