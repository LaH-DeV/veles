package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D120: C layout. `@packed` (an extern struct or union: no padding,
// alignment 1), `@align(n)` (any struct, or a field of an extern struct:
// n a power of two not below the natural alignment), `extern union`
// (fields of C types, all at offset 0, touched only in `unsafe`) and
// `@transparent` (a struct of one field, laid out and passed to C as that
// field). The layout itself is types.Layout, shared with the code
// generator.

// layoutAttrs records a struct declaration's layout attributes.
func (c *Checker) layoutAttrs(s *types.Struct, d *ast.StructDecl, attrs map[string]*ast.Attribute) {
	if a, ok := attrs["packed"]; ok {
		if d.Extern {
			s.Packed = true
		} else {
			c.errorf(a.Pos, "@packed applies to an extern struct or extern union, whose layout is C's; a Veles struct's layout is the compiler's (D120)")
		}
	}
	if a, ok := attrs["align"]; ok {
		s.Align = alignArg(c, a)
	}
	if a, ok := attrs["transparent"]; ok {
		if d.Extern || d.Variant != nil || d.Error {
			c.errorf(a.Pos, "@transparent applies to a plain struct of one field, which C then sees as that field (D120)")
		} else {
			s.Transparent = true
		}
	}
}

// checkLayouts checks what needs every struct's fields: run once they are
// resolved.
func (c *Checker) checkLayouts() {
	lay := &types.Layout{}
	for _, s := range c.structs {
		if len(s.TypeParams) > 0 {
			continue // checked as its instances are laid out
		}
		d, _ := s.Decl.(*ast.StructDecl)
		if d == nil {
			continue
		}
		if s.Transparent && len(s.Fields) != 1 {
			c.errorf(d.Name.Pos, "a @transparent struct has exactly one field, which C sees in its place; '%s' has %d (D120)", s.Name, len(s.Fields))
		}
		if s.Union {
			if len(s.Fields) == 0 {
				c.errorf(d.Name.Pos, "an extern union needs a field (D120)")
			}
			for i, f := range s.Fields {
				if !cLayout(f.Type) {
					c.errorf(d.Fields[i].Name.Pos, "a field of an extern union is a C type — a number, bool, raw pointer, extern struct or union, or 'extern fun' — and '%s' is not one (D120)", f.Type)
				}
			}
		}
		for i, f := range s.Fields {
			if f.Align == 0 {
				continue
			}
			if _, natural := lay.Of(f.Type); f.Align < natural {
				c.errorf(d.Fields[i].Name.Pos, "@align(%d) is below the natural alignment of '%s', %d; an alignment only raises it (D120)", f.Align, f.Type, natural)
			}
		}
		if s.Align > 0 {
			plain := *s
			plain.Align = 0
			if _, natural := lay.Of(&plain); s.Align < natural {
				c.errorf(d.Name.Pos, "@align(%d) is below the natural alignment of '%s', %d; an alignment only raises it (D120)", s.Align, s.Name, natural)
			}
		}
	}
}

// unionLit builds an extern union from exactly one of its fields (D120):
// which one is live is C's business afterwards, but a value starts as one.
func (f *fnCtx) unionLit(st *types.Struct, bound []ast.Expr, span source.Span) Expr {
	lit := &StructLit{exprBase{st}, st, make([]Expr, len(st.Fields))}
	given := -1
	for i, b := range bound {
		if i >= len(st.Fields) || b == nil {
			continue
		}
		if given >= 0 {
			f.errorf(b.Span(), "a union is built from one field: '%s' and '%s' would lie on the same bytes (D120)", st.Fields[given].Name, st.Fields[i].Name)
			return bad()
		}
		given = i
		lit.Fields[i] = f.checkExprTo(b, st.Fields[i].Type)
	}
	if given < 0 {
		f.errorf(span, "a union is built from one of its fields: '%s(%s: …)' (D120)", st.Name, st.Fields[0].Name)
		return bad()
	}
	return lit
}

// unionInside is an extern union t holds by value where structural
// printing would reach it, or nil; a struct with its own Display stops
// the search.
func (f *fnCtx) unionInside(t types.Type, seen map[types.Type]bool) *types.Struct {
	switch t := t.(type) {
	case *types.Struct:
		if seen[t] {
			return nil
		}
		seen[t] = true
		if t.Union {
			return t
		}
		if f.c.implementsPrelude(t, "Display") {
			return nil
		}
		for _, fld := range t.Fields {
			if u := f.unionInside(fld.Type, seen); u != nil {
				return u
			}
		}
	case *types.Nullable:
		return f.unionInside(t.Elem, seen)
	case *types.List:
		return f.unionInside(t.Elem, seen)
	case *types.Set:
		return f.unionInside(t.Elem, seen)
	case *types.Map:
		if u := f.unionInside(t.Key, seen); u != nil {
			return u
		}
		return f.unionInside(t.Value, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if u := f.unionInside(e, seen); u != nil {
				return u
			}
		}
	}
	return nil
}

// inPacked: e is a field lying inside a @packed struct, so a pointer typed
// as e's type could be misaligned (the code generator reads and writes such
// fields through the whole packed struct instead).
func inPacked(e Expr) bool {
	for {
		switch x := e.(type) {
		case *FieldGet:
			if st, ok := x.X.Type().(*types.Struct); ok && st.Packed {
				return true
			}
			e = x.X
		case *TupleGet:
			e = x.X
		default:
			return false
		}
	}
}
