package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// Template literals (D129): `sql"select * from t where id = ${id}"` hands a
// function — one marked `@template` — the literal's text pieces and its
// values, instead of a string with the values already pasted in. What the
// function returns is the literal's type; a `Sql` made this way cannot be
// confused with text a program built, because the only way to make one is
// through the function.
//
// The literal is checked as the call it stands for:
//
//	sql"a ${x} b ${y}"   =   sql(["a ", " b ", ""], [x, y])
//
// so the values convert to the function's element type as list elements do
// (a trait object boxes each), and every diagnostic of a call applies. The
// pieces are the text written in the source, escapes processed, and always
// one more than the values.

// checkTemplateSignature reports a `@template` whose parameters are not
// `(parts: List<string>, values: List<V>)`.
func (c *Checker) checkTemplateSignature(d *ast.FunDecl, method bool) {
	const shape = "a @template function takes the literal's pieces and values: 'fun name(parts: List<string>, values: List<V>)' (D129)"
	if method || d.Static {
		c.errorf(d.Name.Pos, "@template applies to a module-level function, not a method: %s", shape)
		return
	}
	if len(d.Params) != 2 {
		c.errorf(d.Name.Pos, "'%s' is a @template with %d parameters; %s", d.Name.Name, len(d.Params), shape)
		return
	}
	if !isListOf(d.Params[0].Type, "string") {
		c.errorf(d.Params[0].Pos, "the first parameter of a @template is the pieces, 'List<string>'; %s", shape)
	}
	if !isListOf(d.Params[1].Type, "") {
		c.errorf(d.Params[1].Pos, "the second parameter of a @template is the values, a 'List<V>'; %s", shape)
	}
}

// isListOf reports whether t is written `List<elem>` (any element when elem
// is empty).
func isListOf(t ast.Type, elem string) bool {
	n, ok := t.(*ast.NamedType)
	if !ok || len(n.Path) != 1 || n.Path[0].Name != "List" || len(n.Args) != 1 {
		return false
	}
	if elem == "" {
		return true
	}
	e, ok := n.Args[0].(*ast.NamedType)
	return ok && len(e.Path) == 1 && e.Path[0].Name == elem && len(e.Args) == 0
}

// templateTarget finds the function a tag names, or nil when it names
// nothing the call check would not report better.
func (f *fnCtx) templateTarget(tag ast.Expr) *FuncTemplate {
	switch t := tag.(type) {
	case *ast.NameExpr:
		if sym := f.lookup(t.Name); sym != nil && sym.Kind == SymFunc {
			return sym.Func
		}
	case *ast.MemberExpr:
		n, ok := t.X.(*ast.NameExpr)
		if !ok {
			return nil
		}
		if sym := f.lookup(n.Name); sym != nil && sym.Kind == SymModule {
			if member := sym.Mod.Scope.LookupLocal(t.Name.Name); member != nil && member.Pub && member.Kind == SymFunc {
				return member.Func
			}
		}
	}
	return nil
}

// templateLit checks `tag"…"` as the call `tag(pieces, values)`.
func (f *fnCtx) templateLit(e *ast.TemplateExpr, want types.Type) Expr {
	target := f.templateTarget(e.Tag)
	refuse := func() Expr {
		for _, p := range e.Lit.Parts {
			if p.Expr != nil {
				f.checkExpr(p.Expr, nil)
			}
		}
		return bad()
	}
	if target == nil {
		// an unknown name, a module without that member, a value: the
		// ordinary diagnostics for the tag say it best
		before := len(f.c.roundDiags.Items)
		f.checkExpr(e.Tag, nil)
		if len(f.c.roundDiags.Items) == before {
			f.errorf(e.Tag.Span(), "a template tag is the name of a function marked '@template' (D129); this is not a function")
		}
		return refuse()
	}
	if _, ok := target.Attrs["template"]; !ok {
		f.errorf(e.Tag.Span(), "'%s' is not a template: a string written right after a name is handed to a function marked '@template' (D129); mark '%s' @template, or call it with parentheses", target.Name, target.Name)
		return refuse()
	}
	var pieces, values []ast.Expr
	var text string
	piece := func() {
		pieces = append(pieces, &ast.StringLit{Parts: []ast.StringPart{{Text: text}}, Pos: e.Lit.Pos})
		text = ""
	}
	for _, p := range e.Lit.Parts {
		if p.Expr == nil {
			text += p.Text
			continue
		}
		piece()
		values = append(values, p.Expr)
	}
	piece()
	call := &ast.CallExpr{
		Fun: e.Tag,
		Args: []ast.Arg{
			{Value: &ast.ListLit{Elems: pieces, Pos: e.Lit.Pos}},
			{Value: &ast.ListLit{Elems: values, Pos: e.Lit.Pos}},
		},
		Pos: e.Pos,
	}
	return f.checkExpr(call, want)
}
