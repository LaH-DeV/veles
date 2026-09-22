package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// A `static fun` (D23) has no receiver and is called on the type:
// `Point.origin()`, `i64.parse(s)`, or `T.parse(s)` inside generic code.
// It is looked up where methods are — the struct body, extend blocks and
// trait impls — and instantiated like any other template.

// typeNamed resolves a name used as a call target to the type it denotes:
// a declared or built-in type, or a type parameter (concrete inside a
// stencil). nil when the name is not a type.
func (f *fnCtx) typeNamed(n *ast.NameExpr) types.Type {
	if sym := f.lookup(n.Name); sym != nil {
		if sym.Kind != SymType {
			return nil
		}
		f.c.refSym(n.Pos, sym)
		if len(n.TypeArgs) > 0 {
			// `Stack<i64>.empty()`: the type arguments name the instance
			path := []ast.Ident{{Name: n.Name, Pos: n.Pos}}
			return f.resolve(&ast.NamedType{Path: path, Args: n.TypeArgs, Pos: n.Pos})
		}
		if t := f.c.symType(sym); t != nil {
			return t
		}
		return sym.Type
	}
	if _, ok := f.env.tps[n.Name]; ok {
		return f.resolve(&ast.NamedType{Path: []ast.Ident{{Name: n.Name, Pos: n.Pos}}, Pos: n.Pos})
	}
	switch n.Name {
	case "List", "MutableList", "Map", "MutableMap", "Set", "MutableSet":
		// `MutableList<bool>.repeat(false, n)`: statics the prelude adds to a
		// built-in generic type with `extend`; the type arguments are required.
		if len(n.TypeArgs) == 0 {
			f.errorf(n.Pos, "'%s' is generic; write the type arguments, e.g. '%s<T>.f(...)'", n.Name, n.Name)
			return types.TInvalid
		}
		path := []ast.Ident{{Name: n.Name, Pos: n.Pos}}
		return f.resolve(&ast.NamedType{Path: path, Args: n.TypeArgs, Pos: n.Pos})
	}
	return nil
}

// staticCall checks `Type.name(args)`.
func (f *fnCtx) staticCall(rt types.Type, callee *ast.MemberExpr, typeArgs []types.Type, e *ast.CallExpr, want types.Type) Expr {
	name := callee.Name.Name
	if types.IsInvalid(rt) {
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if en, ok := rt.(*types.Enum); ok && (name == "values" || name == "fromValue" || name == "parse") {
		return f.enumStaticCall(en, callee, e) // D57; `decode` comes from the derived impl (D58)
	}
	t, subst, ok := f.findStatic(rt, name, callee.Name.Pos)
	if !ok {
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	if t == nil {
		if m, _, _ := f.findMethod(rt, name); m != nil {
			f.errorf(callee.Name.Pos, "'%s' is a method of '%s'; call it on a value, not on the type", name, rt)
		} else if en, isEnum := rt.(*types.Enum); isEnum {
			if m := en.MemberByName(name); m != nil {
				f.errorf(callee.Name.Pos, "'%s.%s' is a value, not a function", en.Name, name)
			} else {
				f.errorf(callee.Name.Pos, "enum '%s' has no function '%s'; an enum has 'values()', 'fromValue(n)', 'parse(s)' and 'decode(from)' (D57)", en.Name, name)
			}
		} else {
			f.errorf(callee.Name.Pos, "no static function '%s' on type '%s'", name, rt)
		}
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	f.c.refFunc(callee.Name.Pos, t)
	inherent := t.Owner != nil || (t.Impl != nil && t.Impl.Trait == nil)
	if inherent && !t.Pub && t.Module != f.module {
		f.errorf(callee.Name.Pos, "static function '%s' is private to module '%s' (M5)", name, t.Module.Path)
	}
	return f.callTemplateRecv(t, subst, typeArgs, nil, e.Args, e.Pos, want)
}

// findStatic finds the static function `name` for a type in the struct
// body, the extend blocks and the trait impls; ok is false when an error
// was reported (an unmet extend bound, an ambiguity).
func (f *fnCtx) findStatic(rt types.Type, name string, pos source.Span) (*FuncTemplate, map[*types.TypeParam]types.Type, bool) {
	t, subst, found := f.findMethod(rt, name)
	if found != "" {
		f.errorf(pos, "%s", found)
		return nil, nil, false
	}
	if t == nil || !t.Decl.Static {
		return nil, nil, true
	}
	return t, subst, true
}

// findMethod locates a method or static function by name on a type without
// any receiver handling: the struct body first, then extend blocks (whose
// bounds must hold), then trait impls (ambiguity is an error, D26). The
// third result is an error message, or "".
func (f *fnCtx) findMethod(rt types.Type, name string) (*FuncTemplate, map[*types.TypeParam]types.Type, string) {
	if st, ok := rt.(*types.Struct); ok {
		if t, ok := f.c.methods[templateOf(st)][name]; ok {
			return t, substOf(st), ""
		}
	}
	for _, view := range receiverViews(rt) {
		for _, ext := range f.c.extends {
			t, ok := ext.Methods[name]
			if !ok {
				continue
			}
			m := map[*types.TypeParam]types.Type{}
			if !unify(ext.Target, view, m) {
				continue
			}
			for _, tp := range ext.TypeParams {
				for _, bound := range tp.Bounds {
					if bt, ok := m[tp]; ok && !f.implements(bt, bound) {
						return nil, nil, "'" + name + "' on '" + rt.String() + "' requires '" + bt.String() + "' to implement '" + bound.Name + "' (extend<" + tp.Name + ": " + bound.Name + "> " + ext.Target.String() + ")" + sendableHint(bound)
					}
				}
			}
			return t, m, ""
		}
	}
	var found []*FuncTemplate
	var foundSubst []map[*types.TypeParam]types.Type
	var traits []string
	unmet := ""
	for trait, impls := range f.c.impls {
		if _, has := trait.Methods[name]; !has {
			continue
		}
		for _, impl := range impls {
			m := map[*types.TypeParam]types.Type{}
			if !unify(impl.Target, rt, m) {
				continue
			}
			// the impl's own bounds: `impl<T: Encodable> Encodable for List<T>`
			// serves `List<H>` only when `H` is Encodable
			if why := f.unmetImplBound(impl, m, name, rt); why != "" {
				unmet = why
				continue
			}
			if t, ok := impl.Methods[name]; ok {
				found, foundSubst, traits = append(found, t), append(foundSubst, m), append(traits, trait.Name)
			} else if dt := f.c.traitDefault(trait, name); dt != nil {
				m[selfParamOf(trait)] = rt
				found, foundSubst, traits = append(found, dt), append(foundSubst, m), append(traits, trait.Name)
			}
		}
	}
	if len(found) > 1 {
		return nil, nil, "ambiguous '" + name + "' on '" + rt.String() + "': provided by traits " + strings.Join(traits, " and ") + " (D26)"
	}
	if len(found) == 1 {
		return found[0], foundSubst[0], ""
	}
	return nil, nil, unmet
}

// unmetImplBound says why an impl does not serve the receiver: a type
// argument that misses a bound of the impl's type parameters; "" when all
// hold (or are not yet known).
func (f *fnCtx) unmetImplBound(impl *Impl, m map[*types.TypeParam]types.Type, name string, rt types.Type) string {
	for _, tp := range impl.TypeParams {
		bt, ok := m[tp]
		if !ok || types.ContainsTypeParam(bt) {
			continue
		}
		for _, bound := range tp.Bounds {
			if !f.implements(bt, bound) {
				return "'" + name + "' on '" + rt.String() + "' requires '" + bt.String() + "' to implement '" + bound.Name + "' (impl<" + tp.Name + ": " + bound.Name + "> " + impl.Trait.Name + " for " + impl.Target.String() + ")" + sendableHint(bound)
			}
		}
	}
	return ""
}

// moduleTypeNamed resolves `module.Type` used as a call target to the type,
// or nil when x is not that shape.
func (f *fnCtx) moduleTypeNamed(x ast.Expr) types.Type {
	m, ok := x.(*ast.MemberExpr)
	if !ok || m.Safe {
		return nil
	}
	n, ok := m.X.(*ast.NameExpr)
	if !ok {
		return nil
	}
	sym := f.lookup(n.Name)
	if sym == nil || sym.Kind != SymModule {
		return nil
	}
	member := sym.Mod.Scope.LookupLocal(m.Name.Name)
	if member == nil || member.Kind != SymType {
		return nil
	}
	if !member.Pub {
		f.errorf(m.Name.Pos, "'%s' is private to module '%s' (M5)", m.Name.Name, n.Name)
		return types.TInvalid
	}
	f.c.refSym(n.Pos, sym)
	f.c.refSym(m.Name.Pos, member)
	if len(m.TypeArgs) > 0 {
		// `crypto.Hmac<Sha256>.start(key)`: the type arguments name the instance
		path := []ast.Ident{{Name: n.Name, Pos: n.Pos}, {Name: m.Name.Name, Pos: m.Name.Pos}}
		return f.resolve(&ast.NamedType{Path: path, Args: m.TypeArgs, Pos: m.Pos})
	}
	if t := f.c.symType(member); t != nil {
		return t
	}
	return member.Type
}

// sendableHint explains a failed `Sendable` bound: the value would be
// shared, which is what the bound exists to refuse.
func sendableHint(bound *types.Trait) string {
	if !isSendableTrait(bound) {
		return ""
	}
	return "; it holds shared mutable state (a mutable collection, a pointer or a closure), so every slot would alias one value; build one per slot with make(n, i => ...)"
}
