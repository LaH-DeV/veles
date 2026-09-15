// Package sema resolves names, checks types and effects, and lowers the
// syntax tree to the HIR consumed by code generation.
package sema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

type Checker struct {
	pkg   *Package
	diags *source.Diagnostics
	// roundDiags collects diagnostics of the current inference round; only
	// the final round's are kept (D45 fixpoint).
	roundDiags *source.Diagnostics
	seen       map[string]bool

	universe *Scope
	prog     *Program
	release  bool
	testMode bool

	// declaration tables
	structs        []*types.Struct // templates
	sealeds        []*types.Sealed
	traits         []*types.Trait
	templates      []*FuncTemplate
	impls          map[*types.Trait][]*Impl
	methods        map[*types.Struct]map[string]*FuncTemplate // inherent, by template
	globals        map[*Global]*ast.ValDecl
	globalMod      map[*Global]*Module
	globalFile     map[*Global]*ast.File
	structDecl     map[*types.Struct]*declCtx
	sealedDecl     map[*types.Sealed]*declCtx
	traitDecl      map[*types.Trait]*declCtx
	resultTmpl     *types.Sealed
	traitMethodTPs map[string][]*types.TypeParam
	tests          []*FuncTemplate
	optionTmpl     *types.Sealed

	// per-round state
	queue          []*Func
	instances      map[string]*Func
	funcs          []*Func
	checkedGlobals map[*Global]bool
	changed        bool
	nextVar        int
	nextLoop       int
	nextTmp        int
	nextLambda     int
	concrete       map[string]*types.Struct
	concreteSealed map[string]*types.Sealed
}

type declCtx struct {
	module    *Module
	file      *ast.File
	decl      ast.Decl
	tps       map[string]*types.TypeParam
	resolved  bool
	resolving bool
}

// Check runs the whole analysis over a loaded package.
// CheckTests is Check for `veles test`: no main is required and @test
// functions become the entry point.
func CheckTests(pkg *Package, diags *source.Diagnostics, release bool) *Program {
	return check(pkg, diags, release, true)
}

func Check(pkg *Package, diags *source.Diagnostics, release bool) *Program {
	return check(pkg, diags, release, false)
}

func check(pkg *Package, diags *source.Diagnostics, release bool, testMode bool) *Program {
	c := &Checker{
		pkg:            pkg,
		diags:          diags,
		seen:           map[string]bool{},
		impls:          map[*types.Trait][]*Impl{},
		methods:        map[*types.Struct]map[string]*FuncTemplate{},
		globals:        map[*Global]*ast.ValDecl{},
		globalMod:      map[*Global]*Module{},
		globalFile:     map[*Global]*ast.File{},
		structDecl:     map[*types.Struct]*declCtx{},
		sealedDecl:     map[*types.Sealed]*declCtx{},
		traitDecl:      map[*types.Trait]*declCtx{},
		traitMethodTPs: map[string][]*types.TypeParam{},
		release:        release,
		testMode:       testMode,
	}
	types.AssocResolver = c.resolveAssoc
	types.StructInstantiator = func(tmpl *types.Struct, args []types.Type) types.Type {
		return c.instantiateStruct(tmpl, args, source.Span{})
	}
	types.SealedInstantiator = func(tmpl *types.Sealed, args []types.Type) types.Type {
		return c.instantiateSealed(tmpl, args, source.Span{})
	}
	c.roundDiags = diags
	c.buildUniverse()
	c.collect()
	if diags.HasErrors() {
		return nil
	}

	// Effect inference to a fixpoint (D4/D45): inferred error unions grow
	// as bodies are checked; rerun until stable.
	var prog *Program
	for round := 0; round < 6; round++ {
		c.roundDiags = &source.Diagnostics{}
		c.seen = map[string]bool{}
		prog = c.runRound()
		if !c.changed {
			break
		}
	}
	if prog != nil && !c.roundDiags.HasErrors() {
		c.inferSuspension(prog)
	}
	diags.Items = append(diags.Items, c.roundDiags.Items...)
	if diags.HasErrors() {
		return nil
	}
	return prog
}

func (c *Checker) errorf(span source.Span, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	key := span.String() + msg
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.roundDiags.Errorf(span, "%s", msg)
}

func (c *Checker) warnf(span source.Span, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	key := "w" + span.String() + msg
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.roundDiags.Warnf(span, "%s", msg)
}

// ---------------------------------------------------------------------------
// universe / prelude (D24)

func (c *Checker) buildUniverse() {
	u := NewScope(nil)
	for name, t := range types.Primitives {
		u.Insert(&Symbol{Name: name, Kind: SymType, Type: t, Pub: true})
	}
	// Result<T, E> and Option<T> as builtin sealed templates (D4/D5).
	c.resultTmpl = c.builtinSealed("Result", []string{"T", "E"}, [][2]string{{"Ok", "value"}, {"Err", "error"}})
	c.optionTmpl = c.builtinSealed("Option", []string{"T"}, [][2]string{{"Some", "value"}, {"None", ""}})
	u.Insert(&Symbol{Name: "Result", Kind: SymType, Type: c.resultTmpl, Pub: true})
	u.Insert(&Symbol{Name: "Option", Kind: SymType, Type: c.optionTmpl, Pub: true})
	for _, ctor := range []string{"Some", "None", "Ok", "Err"} {
		u.Insert(&Symbol{Name: ctor, Kind: SymVariantCtor, Pub: true})
	}
	c.universe = u
}

func (c *Checker) builtinSealed(name string, params []string, variants [][2]string) *types.Sealed {
	s := &types.Sealed{Name: name, Module: "<prelude>", Pub: true, Instances: map[string]*types.Sealed{}}
	for i, p := range params {
		s.TypeParams = append(s.TypeParams, &types.TypeParam{Name: p, Index: i, Owner: name})
	}
	for i, v := range variants {
		vs := &types.Struct{Name: v[0], Module: "<prelude>", Pub: true, Sealed: s, Tag: i, Instances: map[string]*types.Struct{}}
		vs.TypeParams = s.TypeParams
		if v[1] != "" {
			var ft types.Type = s.TypeParams[0]
			if v[0] == "Err" {
				ft = s.TypeParams[1]
			}
			vs.Fields = []*types.Field{{Name: v[1], Type: ft, Pub: true, Index: 0}}
		}
		s.Variants = append(s.Variants, vs)
	}
	return s
}

// ResultType returns the concrete Result<ok, err> sealed type.
func (c *Checker) ResultType(ok, errT types.Type) *types.Sealed {
	if errT == nil {
		errT = types.TNever
	}
	return c.instantiateSealed(c.resultTmpl, []types.Type{ok, errT}, source.Span{})
}

// ---------------------------------------------------------------------------
// collection: module scopes and declarations

func (c *Checker) collect() {
	mods := c.pkg.SortedModules()
	// 1. declare every module-level name
	for _, m := range mods {
		m.Scope = NewScope(c.universe)
		m.Scope.module = m
		m.Imports = map[*ast.File]*Scope{}
		for _, f := range m.Files {
			m.Imports[f] = NewScope(m.Scope)
			for _, d := range f.Decls {
				c.declare(m, f, d)
			}
		}
	}
	// 1b. the prelude is in scope everywhere (D24)
	if prelude, ok := c.pkg.Modules["std/prelude"]; ok {
		for _, sym := range prelude.Scope.symbols {
			if sym.Pub {
				c.universe.Insert(sym)
			}
		}
	}
	// 2. wire imports (per file)
	for _, m := range mods {
		for _, f := range m.Files {
			for _, d := range f.Decls {
				if u, ok := d.(*ast.UseDecl); ok {
					c.declareUse(m, f, u)
				}
			}
		}
	}
	// 3. resolve signatures, fields, variants, traits, impls
	for _, s := range c.structs {
		c.resolveStruct(s)
	}
	for _, s := range c.sealeds {
		c.resolveSealed(s)
	}
	for _, t := range c.traits {
		c.resolveTrait(t)
	}
	for _, m := range mods {
		for _, f := range m.Files {
			for _, d := range f.Decls {
				if impl, ok := d.(*ast.ImplDecl); ok {
					c.declareImpl(m, f, impl)
				}
			}
		}
	}
	for _, t := range c.templates {
		c.resolveSignature(t)
	}
	c.checkCoherence()
}

func (c *Checker) insert(m *Module, sym *Symbol) {
	if old := m.Scope.Insert(sym); old != nil {
		c.errorf(sym.Span, "'%s' is already declared in this module (at %s)", sym.Name, old.Span)
	}
}

func (c *Checker) declare(m *Module, f *ast.File, d ast.Decl) {
	switch d := d.(type) {
	case *ast.FunDecl:
		t := c.newTemplate(m, f, d, nil, nil)
		c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymFunc, Pub: d.Pub, Module: m, Span: d.Name.Pos, Func: t})
	case *ast.ExternBlock:
		for _, fn := range d.Funs {
			t := c.newTemplate(m, f, fn, nil, nil)
			t.Extern = true
			t.Mangled = fn.Name.Name // C symbol
			c.insert(m, &Symbol{Name: fn.Name.Name, Kind: SymFunc, Pub: fn.Pub, Module: m, Span: fn.Name.Pos, Func: t})
		}
	case *ast.StructDecl:
		c.attrsOf(d.Attrs, "struct")
		s := &types.Struct{Name: d.Name.Name, Module: m.prefix(), Pub: d.Pub, Extern: d.Extern, Decl: d, Instances: map[string]*types.Struct{}}
		ctx := &declCtx{module: m, file: f, decl: d, tps: map[string]*types.TypeParam{}}
		for i, tp := range d.TypeParams {
			p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: d.Name.Name}
			s.TypeParams = append(s.TypeParams, p)
			ctx.tps[tp.Name.Name] = p
		}
		c.structDecl[s] = ctx
		c.structs = append(c.structs, s)
		c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymType, Pub: d.Pub, Module: m, Span: d.Name.Pos, Type: s})
		c.methods[s] = map[string]*FuncTemplate{}
		for _, md := range d.Methods {
			t := c.newTemplate(m, f, md, s, ctx.tps)
			if _, dup := c.methods[s][md.Name.Name]; dup {
				c.errorf(md.Name.Pos, "duplicate method '%s' on '%s'", md.Name.Name, s.Name)
				continue
			}
			c.methods[s][md.Name.Name] = t
		}
	case *ast.TraitDecl:
		c.attrsOf(d.Attrs, "trait")
		if d.Sealed {
			s := &types.Sealed{Name: d.Name.Name, Module: m.prefix(), Pub: d.Pub, Decl: d, Instances: map[string]*types.Sealed{}}
			ctx := &declCtx{module: m, file: f, decl: d, tps: map[string]*types.TypeParam{}}
			for i, tp := range d.TypeParams {
				p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: d.Name.Name}
				s.TypeParams = append(s.TypeParams, p)
				ctx.tps[tp.Name.Name] = p
			}
			c.sealedDecl[s] = ctx
			c.sealeds = append(c.sealeds, s)
			c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymType, Pub: d.Pub, Module: m, Span: d.Name.Pos, Type: s})
			if len(d.AssocTypes) > 0 {
				c.errorf(d.Name.Pos, "sealed traits cannot declare associated types")
			}
			if len(d.Methods) > 0 {
				// the trait half: methods implemented per variant, dispatched by tag
				t := &types.Trait{Name: d.Name.Name, Module: m.prefix(), Pub: d.Pub, Decl: d, Methods: map[string]*types.Func{}}
				s.Trait = t
				c.traitDecl[t] = ctx
				c.traits = append(c.traits, t)
			}
			return
		}
		t := &types.Trait{Name: d.Name.Name, Module: m.prefix(), Pub: d.Pub, Decl: d, Methods: map[string]*types.Func{}}
		for _, at := range d.AssocTypes {
			t.AssocTypes = append(t.AssocTypes, at.Name.Name)
		}
		ctx := &declCtx{module: m, file: f, decl: d, tps: map[string]*types.TypeParam{}}
		for i, tp := range d.TypeParams {
			p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: d.Name.Name}
			t.TypeParams = append(t.TypeParams, p)
			ctx.tps[tp.Name.Name] = p
		}
		c.traitDecl[t] = ctx
		c.traits = append(c.traits, t)
		c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymType, Pub: d.Pub, Module: m, Span: d.Name.Pos, Type: t})
	case *ast.ValDecl:
		c.attrsOf(d.Attrs, "value")
		g := &Global{Name: m.prefix() + "." + d.Name.Name, Display: d.Name.Name, Mutable: d.Kind == ast.BindVar, Span: d.Name.Pos}
		c.globals[g] = d
		c.globalMod[g] = m
		c.globalFile[g] = f
		c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymGlobal, Pub: d.Pub, Module: m, Span: d.Name.Pos, Global: g})
	case *ast.ImplDecl, *ast.UseDecl, *ast.BadDecl:
		// handled elsewhere
	}
}

func (c *Checker) newTemplate(m *Module, f *ast.File, d *ast.FunDecl, owner *types.Struct, outer map[string]*types.TypeParam) *FuncTemplate {
	t := &FuncTemplate{Name: d.Name.Name, Module: m, File: f, Decl: d, Pub: d.Pub, Owner: owner, Extern: d.Extern}
	t.Attrs = c.attrsOf(d.Attrs, "function")
	if _, isTest := t.Attrs["test"]; isTest {
		if len(d.Params) > 0 || d.Ret != nil || owner != nil {
			c.errorf(d.Name.Pos, " functions take no parameters and return nothing")
		}
		c.tests = append(c.tests, t)
	}
	t.Instances = map[string]*Func{}
	t.Mangled = m.prefix() + "." + d.Name.Name
	if owner != nil {
		t.Mangled = m.prefix() + "." + owner.Name + "." + d.Name.Name
	}
	for i, tp := range d.TypeParams {
		t.TypeParams = append(t.TypeParams, &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: d.Name.Name})
	}
	c.templates = append(c.templates, t)
	return t
}

func (c *Checker) declareUse(m *Module, f *ast.File, u *ast.UseDecl) {
	var path []string
	for _, seg := range u.Path {
		path = append(path, seg.Name)
	}
	key := strings.Join(path, "/")
	dep := m.Uses[u]
	if dep == nil {
		return // loader already reported
	}
	scope := m.Imports[f]
	if u.Items != nil {
		for _, it := range u.Items {
			sym := dep.Scope.LookupLocal(it.Name.Name)
			if sym == nil {
				c.errorf(it.Name.Pos, "module '%s' has no declaration '%s'", key, it.Name.Name)
				continue
			}
			if !sym.Pub {
				c.errorf(it.Name.Pos, "'%s' is private to module '%s' (M5: add 'pub' to share it)", it.Name.Name, key)
				continue
			}
			name := it.Name.Name
			if it.Alias != nil {
				name = it.Alias.Name
			}
			alias := *sym
			alias.Name = name
			if old := scope.Insert(&alias); old != nil {
				c.errorf(it.Name.Pos, "'%s' is imported twice", name)
			}
		}
		return
	}
	name := path[len(path)-1]
	if u.Alias != nil {
		name = u.Alias.Name
	}
	if old := scope.Insert(&Symbol{Name: name, Kind: SymModule, Mod: dep, Span: u.Pos}); old != nil {
		c.errorf(u.Pos, "'%s' is already imported in this file", name)
	}
}

// ---------------------------------------------------------------------------
// type resolution

type typeEnv struct {
	module *Module
	file   *ast.File
	tps    map[string]*types.TypeParam
	self   types.Type
	// trait is set inside a trait body so bare associated names resolve;
	// implAssoc holds an impl's `type X = T` bindings.
	trait     *types.Trait
	implAssoc map[string]types.Type
}

func (c *Checker) lookupTypeName(env *typeEnv, path []ast.Ident) (*Symbol, *types.Sealed) {
	scope := env.module.Imports[env.file]
	if scope == nil {
		scope = env.module.Scope
	}
	sym := scope.Lookup(path[0].Name)
	if sym == nil {
		return nil, nil
	}
	for i := 1; i < len(path); i++ {
		switch sym.Kind {
		case SymModule:
			next := sym.Mod.Scope.LookupLocal(path[i].Name)
			if next == nil {
				c.errorf(path[i].Pos, "module '%s' has no declaration '%s'", sym.Mod.Path, path[i].Name)
				return nil, nil
			}
			if !next.Pub {
				c.errorf(path[i].Pos, "'%s' is private to module '%s' (M5)", path[i].Name, sym.Mod.Path)
				return nil, nil
			}
			sym = next
		case SymType:
			// Sealed.Variant
			if s, ok := sym.Type.(*types.Sealed); ok {
				v := s.VariantByName(path[i].Name)
				if v == nil {
					c.errorf(path[i].Pos, "'%s' has no variant '%s'", s.Name, path[i].Name)
					return nil, nil
				}
				return &Symbol{Name: v.Name, Kind: SymType, Type: v, Pub: true}, s
			}
			c.errorf(path[i].Pos, "'%s' is not a module or sealed trait", path[i-1].Name)
			return nil, nil
		default:
			c.errorf(path[i-1].Pos, "'%s' is not a module", path[i-1].Name)
			return nil, nil
		}
	}
	return sym, nil
}

func (c *Checker) resolveType(env *typeEnv, t ast.Type) types.Type {
	switch t := t.(type) {
	case nil:
		return types.TUnit
	case *ast.NamedType:
		if len(t.Path) == 1 {
			name := t.Path[0].Name
			// bare associated names inside trait and impl bodies
			if env.implAssoc != nil {
				if bt, ok := env.implAssoc[name]; ok && len(t.Args) == 0 {
					return bt
				}
			}
			if env.trait != nil && containsString(env.trait.AssocTypes, name) && len(t.Args) == 0 {
				return &types.Assoc{Base: selfParamOf(env.trait), Trait: env.trait, Name: name}
			}
			if tp, ok := env.tps[name]; ok {
				if len(t.Args) > 0 {
					c.errorf(t.Pos, "type parameter '%s' cannot take type arguments", name)
				}
				return tp
			}
			switch name {
			case "List", "MutableList":
				if len(t.Args) != 1 {
					c.errorf(t.Pos, "%s takes exactly one type argument", name)
					return types.TInvalid
				}
				return &types.List{Elem: c.resolveType(env, t.Args[0]), Mutable: name == "MutableList"}
			case "Range":
				if len(t.Args) != 1 {
					c.errorf(t.Pos, "Range takes exactly one type argument")
					return types.TInvalid
				}
				return &types.Range{Elem: c.resolveType(env, t.Args[0])}
			case "Option":
				if len(t.Args) != 1 {
					c.errorf(t.Pos, "Option takes exactly one type argument")
					return types.TInvalid
				}
				return &types.Nullable{Elem: c.resolveType(env, t.Args[0])}
			case "Map", "MutableMap":
				if len(t.Args) != 2 {
					c.errorf(t.Pos, "%s takes two type arguments", name)
					return types.TInvalid
				}
				k := c.resolveType(env, t.Args[0])
				c.checkHashable(k, t.Args[0].Span())
				return &types.Map{Key: k, Value: c.resolveType(env, t.Args[1]), Mutable: name == "MutableMap"}
			case "Channel":
				if len(t.Args) != 1 {
					c.errorf(t.Pos, "Channel takes one type argument")
					return types.TInvalid
				}
				return &types.Channel{Elem: c.resolveType(env, t.Args[0])}
			case "Task":
				if len(t.Args) != 1 {
					c.errorf(t.Pos, "Task takes one type argument")
					return types.TInvalid
				}
				return &types.Task{Result: c.resolveType(env, t.Args[0])}
			case "Set", "MutableSet":
				if len(t.Args) != 1 {
					c.errorf(t.Pos, "%s takes one type argument", name)
					return types.TInvalid
				}
				k := c.resolveType(env, t.Args[0])
				c.checkHashable(k, t.Args[0].Span())
				return &types.Set{Elem: k, Mutable: name == "MutableSet"}
			}
		}
		sym, _ := c.lookupTypeName(env, t.Path)
		if sym == nil {
			c.errorf(t.Pos, "unknown type '%s'", pathString(t.Path))
			return types.TInvalid
		}
		if sym.Kind != SymType {
			c.errorf(t.Pos, "'%s' is not a type", pathString(t.Path))
			return types.TInvalid
		}
		var args []types.Type
		for _, a := range t.Args {
			args = append(args, c.resolveType(env, a))
		}
		switch st := sym.Type.(type) {
		case *types.Struct:
			return c.applyStructArgs(st, args, t.Pos)
		case *types.Sealed:
			return c.applySealedArgs(st, args, t.Pos)
		case *types.Trait:
			if len(args) > 0 {
				c.errorf(t.Pos, "generic traits as types are not supported")
			}
			return st
		}
		if len(args) > 0 {
			c.errorf(t.Pos, "'%s' is not generic", sym.Name)
		}
		return sym.Type
	case *ast.NullableType:
		return &types.Nullable{Elem: c.resolveType(env, t.Elem)}
	case *ast.PointerType:
		return &types.Pointer{Elem: c.resolveType(env, t.Elem), Raw: t.Raw}
	case *ast.TupleType:
		if len(t.Elems) == 0 {
			return types.TUnit
		}
		tt := &types.Tuple{}
		for _, e := range t.Elems {
			tt.Elems = append(tt.Elems, c.resolveType(env, e))
		}
		return tt
	case *ast.FunType:
		ft := &types.Func{Ret: c.resolveType(env, t.Ret)}
		for _, p := range t.Params {
			ft.Params = append(ft.Params, types.Param{Type: c.resolveType(env, p)})
		}
		ft.Effects = c.resolveEffects(env, t.Effects, true)
		return ft
	case *ast.SelfType:
		if env.self == nil {
			c.errorf(t.Pos, "'Self' is only meaningful inside a struct, trait or impl")
			return types.TInvalid
		}
		return env.self
	case *ast.AssocType:
		base := c.resolveType(env, t.Base)
		if types.IsInvalid(base) {
			return base
		}
		return c.projectAssoc(env, base, t.Name.Name, t.Pos)
	case *ast.ErrorUnionType:
		var members []types.Type
		for _, m := range t.Members {
			members = append(members, c.resolveType(env, m))
		}
		return types.MakeErrorUnion(members...)
	}
	c.errorf(t.Span(), "unsupported type syntax")
	return types.TInvalid
}

func (c *Checker) resolveEffects(env *typeEnv, e ast.Effects, mustDeclare bool) types.Effects {
	eff := types.Effects{Suspends: e.Suspends, Throws: e.Throws}
	if e.Throws && e.Error != nil {
		eff.Error = c.resolveType(env, e.Error)
		if _, isUnion := eff.Error.(*types.ErrorUnion); !isUnion {
			eff.Error = types.MakeErrorUnion(eff.Error)
		}
	} else if e.Throws && mustDeclare {
		c.errorf(e.ThrowsSpan, "error type must be declared here: 'throws E' (D40 — effects are declared wherever dispatch is dynamic)")
	}
	return eff
}

func pathString(path []ast.Ident) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = p.Name
	}
	return strings.Join(parts, ".")
}

func (c *Checker) applyStructArgs(st *types.Struct, args []types.Type, span source.Span) types.Type {
	if len(args) != len(st.TypeParams) {
		if len(st.TypeParams) == 0 {
			c.errorf(span, "'%s' is not generic", st.Name)
		} else {
			c.errorf(span, "'%s' expects %d type arguments, got %d", st.Name, len(st.TypeParams), len(args))
		}
		return types.TInvalid
	}
	if len(args) == 0 {
		return st
	}
	return c.instantiateStruct(st, args, span)
}

func (c *Checker) applySealedArgs(st *types.Sealed, args []types.Type, span source.Span) types.Type {
	if len(args) != len(st.TypeParams) {
		if len(st.TypeParams) == 0 {
			c.errorf(span, "'%s' is not generic", st.Name)
		} else {
			c.errorf(span, "'%s' expects %d type arguments, got %d", st.Name, len(st.TypeParams), len(args))
		}
		return types.TInvalid
	}
	if len(args) == 0 {
		return st
	}
	return c.instantiateSealed(st, args, span)
}

func (c *Checker) envFor(ctx *declCtx, self types.Type) *typeEnv {
	return &typeEnv{module: ctx.module, file: ctx.file, tps: ctx.tps, self: self}
}

// resolveStruct fills in field types of a struct template.
func (c *Checker) resolveStruct(s *types.Struct) {
	ctx := c.structDecl[s]
	if ctx == nil || ctx.resolved {
		return
	}
	if ctx.resolving {
		return
	}
	ctx.resolving = true
	d := ctx.decl.(*ast.StructDecl)
	env := c.envFor(ctx, s)
	for i, tp := range d.TypeParams {
		for _, b := range tp.Bounds {
			if tr, ok := c.resolveType(env, b).(*types.Trait); ok {
				s.TypeParams[i].Bounds = append(s.TypeParams[i].Bounds, tr)
			}
		}
	}
	seen := map[string]bool{}
	for i, f := range d.Fields {
		if seen[f.Name.Name] {
			c.errorf(f.Name.Pos, "duplicate field '%s'", f.Name.Name)
			continue
		}
		seen[f.Name.Name] = true
		ft := c.resolveType(env, f.Type)
		s.Fields = append(s.Fields, &types.Field{Name: f.Name.Name, Type: ft, Pub: f.Pub, HasDefault: f.Default != nil, Index: i})
	}
	if d.Variant != nil {
		vt := c.resolveType(env, d.Variant)
		parent, ok := vt.(*types.Sealed)
		if !ok {
			if !types.IsInvalid(vt) {
				c.errorf(d.Variant.Span(), "'%s' is not a sealed trait; 'struct X : T' declares variant membership (D23), trait conformance uses 'impl T for X'", vt)
			}
		} else {
			tmpl := parent
			if parent.Template != nil {
				tmpl = parent.Template
			}
			if tmpl.Module != s.Module {
				c.errorf(d.Variant.Span(), "variants of sealed trait '%s' must be declared in the same module as the trait (D12)", tmpl.Name)
			} else if len(tmpl.TypeParams) != len(s.TypeParams) {
				c.errorf(d.Variant.Span(), "variant '%s' must have the same type parameters as '%s'", s.Name, tmpl.Name)
			} else {
				s.Sealed = tmpl
				s.Tag = len(tmpl.Variants)
				tmpl.Variants = append(tmpl.Variants, s)
			}
		}
	}
	ctx.resolved = true
	ctx.resolving = false
	// D31: a value-typed field of the struct's own sealed parent or itself
	// has infinite size.
	for _, f := range s.Fields {
		if c.containsInline(f.Type, s) {
			c.errorf(d.Name.Pos, "struct '%s' has infinite size: field '%s' contains it by value; recursive types must go through a pointer, e.g. '*%s' (D31)", s.Name, f.Name, f.Type)
		}
	}
}

func (c *Checker) containsInline(t types.Type, target *types.Struct) bool {
	switch t := t.(type) {
	case *types.Struct:
		if t == target || t.Template == target {
			return true
		}
		for _, f := range t.Fields {
			if c.containsInline(f.Type, target) {
				return true
			}
		}
	case *types.Sealed:
		tmpl := t
		if t.Template != nil {
			tmpl = t.Template
		}
		for _, v := range tmpl.Variants {
			if v == target || c.containsInline(v, target) {
				return true
			}
		}
	case *types.Nullable:
		return c.containsInline(t.Elem, target)
	case *types.Tuple:
		for _, e := range t.Elems {
			if c.containsInline(e, target) {
				return true
			}
		}
	}
	return false
}

func (c *Checker) resolveSealed(s *types.Sealed) {
	// variants attach themselves in resolveStruct; nothing else to do
}

func (c *Checker) resolveTrait(t *types.Trait) {
	ctx := c.traitDecl[t]
	d := ctx.decl.(*ast.TraitDecl)
	self := selfParamOf(t)
	self.Bounds = []*types.Trait{t}
	env := c.envFor(ctx, self)
	env.trait = t
	t.AssocBounds = map[string][]*types.Trait{}
	for _, at := range d.AssocTypes {
		for _, b := range at.Bounds {
			bt := c.resolveType(env, b)
			if tr, ok := bt.(*types.Trait); ok {
				t.AssocBounds[at.Name.Name] = append(t.AssocBounds[at.Name.Name], tr)
			} else if !types.IsInvalid(bt) {
				c.errorf(b.Span(), "bound '%s' is not a trait", bt)
			}
		}
	}
	for _, m := range d.Methods {
		if _, dup := t.Methods[m.Name.Name]; dup {
			c.errorf(m.Name.Pos, "duplicate trait method '%s'", m.Name.Name)
			continue
		}
		// A generic trait method's parameters are shared between the trait
		// signature and the default-body template so impls compare equal.
		menv := *env
		menv.tps = map[string]*types.TypeParam{}
		for k, v := range env.tps {
			menv.tps[k] = v
		}
		var mtps []*types.TypeParam
		for i, tp := range m.TypeParams {
			p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: m.Name.Name}
			mtps = append(mtps, p)
			menv.tps[tp.Name.Name] = p
		}
		for i, tp := range m.TypeParams {
			for _, b := range tp.Bounds {
				if tr, ok := c.resolveType(&menv, b).(*types.Trait); ok {
					mtps[i].Bounds = append(mtps[i].Bounds, tr)
				}
			}
		}
		sig := c.signatureOf(&menv, m, true)
		t.Methods[m.Name.Name] = sig
		t.MethodList = append(t.MethodList, m.Name.Name)
		c.traitMethodTPs[t.Name+"."+m.Name.Name] = mtps
		if m.Body != nil || m.ExprBody != nil {
			tmpl := c.newTemplate(ctx.module, ctx.file, m, nil, ctx.tps)
			tmpl.Trait = t
			tmpl.TypeParams = mtps
			tmpl.Mangled = ctx.module.prefix() + "." + t.Name + "." + m.Name.Name
		}
	}
	if len(d.Supers) > 0 {
		c.errorf(d.Name.Pos, "supertraits are not supported yet in the bootstrap compiler")
	}
}

func (c *Checker) signatureOf(env *typeEnv, d *ast.FunDecl, isTrait bool) *types.Func {
	sig := &types.Func{Ret: c.resolveType(env, d.Ret)}
	seen := map[string]bool{}
	for _, p := range d.Params {
		if seen[p.Name.Name] {
			c.errorf(p.Name.Pos, "duplicate parameter '%s'", p.Name.Name)
		}
		seen[p.Name.Name] = true
		pt := types.Type(types.TInvalid)
		if p.Type != nil {
			pt = c.resolveType(env, p.Type)
		} else {
			c.errorf(p.Name.Pos, "parameter '%s' needs a type", p.Name.Name)
		}
		sig.Params = append(sig.Params, types.Param{Name: p.Name.Name, Type: pt, HasDefault: p.Default != nil})
	}
	sig.Effects = c.resolveEffects(env, d.Effects, isTrait || d.Extern)
	return sig
}

func (c *Checker) resolveSignature(t *FuncTemplate) {
	if t.Sig != nil {
		return
	}
	env := &typeEnv{module: t.Module, file: t.File, tps: map[string]*types.TypeParam{}}
	if t.Owner != nil {
		ctx := c.structDecl[t.Owner]
		for k, v := range ctx.tps {
			env.tps[k] = v
		}
		env.self = t.Owner
	}
	if t.Impl != nil {
		for _, tp := range t.Impl.TypeParams {
			env.tps[tp.Name] = tp
		}
		env.self = t.Impl.Target
	}
	if t.Trait != nil {
		ctx := c.traitDecl[t.Trait]
		for k, v := range ctx.tps {
			env.tps[k] = v
		}
		env.self = selfParamOf(t.Trait)
		env.trait = t.Trait
	}
	if t.Impl != nil {
		env.implAssoc = t.Impl.AssocTypes
	}
	for _, tp := range t.TypeParams {
		if _, dup := env.tps[tp.Name]; dup {
			c.errorf(t.Decl.Name.Pos, "type parameter '%s' shadows an outer type parameter", tp.Name)
		}
		env.tps[tp.Name] = tp
	}
	for i, tp := range t.Decl.TypeParams {
		if t.Trait != nil {
			break // bounds were resolved with the trait signature
		}
		for _, b := range tp.Bounds {
			bt := c.resolveType(env, b)
			if tr, ok := bt.(*types.Trait); ok {
				t.TypeParams[i].Bounds = append(t.TypeParams[i].Bounds, tr)
			} else if !types.IsInvalid(bt) {
				c.errorf(b.Span(), "bound '%s' is not a trait", bt)
			}
		}
	}
	t.Sig = c.signatureOf(env, t.Decl, false)
	if t.Decl.Ret == nil && t.Decl.ExprBody != nil {
		t.InferRet = true
		t.Sig.Ret = nil
	}
	if t.Extern {
		for _, p := range t.Sig.Params {
			c.checkExternType(p.Type, t.Decl.Name.Pos)
		}
		c.checkExternType(t.Sig.Ret, t.Decl.Name.Pos)
	}
	if t.Decl.Mut && t.Owner == nil && t.Impl == nil && t.Trait == nil {
		c.errorf(t.Decl.Name.Pos, "'mut fun' is only meaningful for methods (D22)")
	}
	if t.Decl.Override && t.Impl == nil {
		c.errorf(t.Decl.Name.Pos, "'override' is only meaningful inside an impl block (D53)")
	}
	if t.Sig.Effects.Throws && t.Sig.Effects.Error == nil && !t.InferDone {
		t.InferError = types.MakeErrorUnion() // nil: nothing yet
	}
}

func (c *Checker) checkExternType(t types.Type, span source.Span) {
	switch t := t.(type) {
	case *types.Basic:
		return
	case *types.Pointer:
		if !t.Raw {
			c.errorf(span, "extern \"C\" functions cannot take GC-managed pointers; use '*raw T' (D44/D50)")
		}
	case *types.Nullable:
		if _, ok := t.Elem.(*types.Basic); !ok {
			c.checkExternType(t.Elem, span)
		}
	case *types.Struct:
		if !t.Extern {
			c.errorf(span, "extern \"C\" functions can only pass 'extern struct' types by value")
		}
	default:
		c.errorf(span, "type '%s' cannot cross the C ABI", t)
	}
}

func (c *Checker) declareImpl(m *Module, f *ast.File, d *ast.ImplDecl) {
	ctx := &declCtx{module: m, file: f, decl: d, tps: map[string]*types.TypeParam{}}
	impl := &Impl{Module: m, Decl: d, Methods: map[string]*FuncTemplate{}}
	for i, tp := range d.TypeParams {
		p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: "impl"}
		impl.TypeParams = append(impl.TypeParams, p)
		ctx.tps[tp.Name.Name] = p
	}
	env := c.envFor(ctx, nil)
	for i, tp := range d.TypeParams {
		for _, b := range tp.Bounds {
			if tr, ok := c.resolveType(env, b).(*types.Trait); ok {
				impl.TypeParams[i].Bounds = append(impl.TypeParams[i].Bounds, tr)
			}
		}
	}
	tt := c.resolveType(env, d.Trait)
	var sealedFor *types.Sealed
	if s, isSealed := tt.(*types.Sealed); isSealed {
		if s.Trait == nil {
			c.errorf(d.Trait.Span(), "sealed trait '%s' declares no methods to implement", s.Name)
			return
		}
		sealedFor = sealedTemplate(s)
		tt = sealedTemplate(s).Trait
	}
	trait, ok := tt.(*types.Trait)
	if !ok {
		if !types.IsInvalid(tt) {
			c.errorf(d.Trait.Span(), "'%s' is not a trait", tt)
		}
		return
	}
	impl.Trait = trait
	impl.Target = c.resolveType(env, d.Target)
	env.self = impl.Target
	if types.IsInvalid(impl.Target) {
		return
	}
	if sealedFor != nil {
		st, isStruct := impl.Target.(*types.Struct)
		if !isStruct || templateOf(st).Sealed != sealedFor {
			c.errorf(d.Target.Span(), "'%s' is not a variant of sealed trait '%s'; sealed trait methods are implemented per variant (D12)", impl.Target, sealedFor.Name)
			return
		}
	}
	impl.AssocTypes = map[string]types.Type{}
	for _, b := range d.AssocTypes {
		if !containsString(trait.AssocTypes, b.Name.Name) {
			c.errorf(b.Name.Pos, "trait '%s' has no associated type '%s'", trait.Name, b.Name.Name)
			continue
		}
		bt := c.resolveType(env, b.Type)
		impl.AssocTypes[b.Name.Name] = bt
		for _, bound := range trait.AssocBounds[b.Name.Name] {
			if !types.ContainsTypeParam(bt) && c.findImplFor(bt, bound) == nil {
				c.errorf(b.Type.Span(), "'%s' does not implement '%s', required by 'type %s: %s' in trait '%s'", bt, bound.Name, b.Name.Name, bound.Name, trait.Name)
			}
		}
	}
	for _, name := range trait.AssocTypes {
		if _, ok := impl.AssocTypes[name]; !ok {
			c.errorf(d.Pos, "impl of '%s' for '%s' must bind associated type '%s': 'type %s = ...'", trait.Name, impl.Target, name, name)
		}
	}
	env.implAssoc = impl.AssocTypes
	c.impls[trait] = append(c.impls[trait], impl)
	for _, md := range d.Methods {
		sig, declared := trait.Methods[md.Name.Name]
		if !declared {
			c.errorf(md.Name.Pos, "trait '%s' has no method '%s'", trait.Name, md.Name.Name)
			continue
		}
		t := c.newTemplate(m, f, md, nil, ctx.tps)
		t.Impl = impl
		t.Mangled = m.prefix() + "." + trait.Name + "." + typeMangle(impl.Target) + "." + md.Name.Name
		impl.Methods[md.Name.Name] = t
		c.resolveSignature(t)
		// D28: impls inherit the trait's parameter names; D53: override only
		// when the trait supplies a default.
		hasDefault := c.traitDefault(trait, md.Name.Name) != nil
		if md.Override && !hasDefault {
			c.errorf(md.Name.Pos, "'override' is only allowed when trait '%s' supplies a default body for '%s' (D53)", trait.Name, md.Name.Name)
		}
		if !md.Override && hasDefault {
			c.errorf(md.Name.Pos, "method '%s' overrides a default body in trait '%s' and must be marked 'override' (D53)", md.Name.Name, trait.Name)
		}
		c.checkImplSignature(t, sig, trait, impl, md)
	}
	for _, name := range trait.MethodList {
		if _, ok := impl.Methods[name]; !ok && c.traitDefault(trait, name) == nil {
			c.errorf(d.Pos, "impl of '%s' for '%s' is missing method '%s'", trait.Name, impl.Target, name)
		}
	}
}

func (c *Checker) traitDefault(trait *types.Trait, name string) *FuncTemplate {
	for _, t := range c.templates {
		if t.Trait == trait && t.Name == name {
			return t
		}
	}
	return nil
}

func (c *Checker) checkImplSignature(t *FuncTemplate, traitSig *types.Func, trait *types.Trait, impl *Impl, md *ast.FunDecl) {
	if len(t.Sig.Params) != len(traitSig.Params) {
		c.errorf(md.Name.Pos, "method '%s' takes %d parameters but trait '%s' declares %d", md.Name.Name, len(t.Sig.Params), trait.Name, len(traitSig.Params))
		return
	}
	subst := map[*types.TypeParam]types.Type{selfParamOf(trait): impl.Target}
	if mtps := c.traitMethodTPs[trait.Name+"."+md.Name.Name]; len(mtps) == len(t.TypeParams) {
		for i, tp := range mtps {
			subst[tp] = t.TypeParams[i]
		}
	} else {
		c.errorf(md.Name.Pos, "method '%s' must declare the same type parameters as in trait '%s'", md.Name.Name, trait.Name)
	}
	for i, p := range t.Sig.Params {
		want := types.Subst(traitSig.Params[i].Type, subst)
		if p.Name != traitSig.Params[i].Name {
			c.errorf(md.Params[i].Name.Pos, "parameter must be named '%s' as in trait '%s' (D28: impls inherit the trait's parameter names)", traitSig.Params[i].Name, trait.Name)
		}
		if !types.Identical(p.Type, want) {
			c.errorf(md.Params[i].Pos, "parameter '%s' has type '%s' but trait '%s' declares '%s'", p.Name, p.Type, trait.Name, want)
		}
	}
	wantRet := types.Subst(traitSig.Ret, subst)
	if !types.Identical(t.Sig.Ret, wantRet) {
		c.errorf(md.Name.Pos, "method '%s' returns '%s' but trait '%s' declares '%s'", md.Name.Name, t.Sig.Ret, trait.Name, wantRet)
	}
	if t.Sig.Effects.Throws && !traitSig.Effects.Throws {
		c.errorf(md.Name.Pos, "method '%s' throws but trait '%s' declares it as non-throwing (D40)", md.Name.Name, trait.Name)
	}
	if t.Sig.Effects.Throws && traitSig.Effects.Throws && t.Sig.Effects.Error == nil {
		// inherit the declared error
		t.Sig.Effects.Error = traitSig.Effects.Error
	}
}

// substSelf replaces the trait's Self with the impl target, resolving
// associated-type projections through the impl.
func (c *Checker) substSelf(t types.Type, trait *types.Trait, target types.Type) types.Type {
	return types.Subst(t, map[*types.TypeParam]types.Type{selfParamOf(trait): target})
}

// checkCoherence enforces D17: at most one impl per (trait, type) pair.
func (c *Checker) checkCoherence() {
	for trait, impls := range c.impls {
		byKey := map[string]*Impl{}
		for _, impl := range impls {
			k := types.Key(impl.Target)
			if other, dup := byKey[k]; dup {
				c.errorf(impl.Decl.Pos, "conflicting impl of '%s' for '%s'; another impl exists at %s (D17: one impl per trait/type pair program-wide)", trait.Name, impl.Target, other.Decl.Pos)
				continue
			}
			byKey[k] = impl
		}
	}
}

func typeMangle(t types.Type) string {
	s := t.String()
	r := strings.NewReplacer("<", "_", ">", "", ", ", "_", "*", "ptr_", "?", "_opt", "(", "tup_", ")", "", " ", "", "|", "_or_", "!", "never")
	return r.Replace(s)
}

// ---------------------------------------------------------------------------
// generic instantiation

func argsKey(args []types.Type) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = types.Key(a)
	}
	return strings.Join(parts, ",")
}

func (c *Checker) instantiateStruct(tmpl *types.Struct, args []types.Type, span source.Span) *types.Struct {
	c.resolveStruct(tmpl)
	key := argsKey(args)
	if inst, ok := tmpl.Instances[key]; ok {
		return inst
	}
	m := map[*types.TypeParam]types.Type{}
	for i, tp := range tmpl.TypeParams {
		m[tp] = args[i]
	}
	inst := &types.Struct{Name: tmpl.Name, Module: tmpl.Module, Pub: tmpl.Pub, Extern: tmpl.Extern, TypeArgs: args, Template: tmpl, Decl: tmpl.Decl, Tag: tmpl.Tag}
	tmpl.Instances[key] = inst
	for _, f := range tmpl.Fields {
		nf := *f
		nf.Type = types.Subst(f.Type, m)
		inst.Fields = append(inst.Fields, &nf)
	}
	if tmpl.Sealed != nil {
		inst.Sealed = c.instantiateSealed(tmpl.Sealed, args, span)
	}
	return inst
}

func (c *Checker) instantiateSealed(tmpl *types.Sealed, args []types.Type, span source.Span) *types.Sealed {
	key := argsKey(args)
	if inst, ok := tmpl.Instances[key]; ok {
		return inst
	}
	inst := &types.Sealed{Name: tmpl.Name, Module: tmpl.Module, Pub: tmpl.Pub, TypeArgs: args, Template: tmpl, Decl: tmpl.Decl}
	tmpl.Instances[key] = inst
	for _, v := range tmpl.Variants {
		vi := c.instantiateStruct(v, args, span)
		vi.Sealed = inst
		inst.Variants = append(inst.Variants, vi)
	}
	return inst
}

// templateOf returns the declaration template of a struct.
func templateOf(s *types.Struct) *types.Struct {
	if s.Template != nil {
		return s.Template
	}
	return s
}

func sealedTemplate(s *types.Sealed) *types.Sealed {
	if s.Template != nil {
		return s.Template
	}
	return s
}

// substOf returns the type-parameter mapping of an instantiated struct.
func substOf(s *types.Struct) map[*types.TypeParam]types.Type {
	m := map[*types.TypeParam]types.Type{}
	tmpl := templateOf(s)
	for i, tp := range tmpl.TypeParams {
		if i < len(s.TypeArgs) {
			m[tp] = s.TypeArgs[i]
		}
	}
	return m
}

// unify matches a pattern type containing type parameters against a
// concrete type, extending m. It reports whether they are compatible.
func unify(pattern, concrete types.Type, m map[*types.TypeParam]types.Type) bool {
	switch p := pattern.(type) {
	case *types.TypeParam:
		if bound, ok := m[p]; ok {
			return types.Identical(bound, concrete)
		}
		m[p] = concrete
		return true
	case *types.Pointer:
		cc, ok := concrete.(*types.Pointer)
		return ok && p.Raw == cc.Raw && unify(p.Elem, cc.Elem, m)
	case *types.Nullable:
		cc, ok := concrete.(*types.Nullable)
		return ok && unify(p.Elem, cc.Elem, m)
	case *types.List:
		cc, ok := concrete.(*types.List)
		return ok && p.Mutable == cc.Mutable && unify(p.Elem, cc.Elem, m)
	case *types.Range:
		cc, ok := concrete.(*types.Range)
		return ok && unify(p.Elem, cc.Elem, m)
	case *types.Map:
		cc, ok := concrete.(*types.Map)
		return ok && p.Mutable == cc.Mutable && unify(p.Key, cc.Key, m) && unify(p.Value, cc.Value, m)
	case *types.Set:
		cc, ok := concrete.(*types.Set)
		return ok && p.Mutable == cc.Mutable && unify(p.Elem, cc.Elem, m)
	case *types.Channel:
		cc, ok := concrete.(*types.Channel)
		return ok && unify(p.Elem, cc.Elem, m)
	case *types.Task:
		cc, ok := concrete.(*types.Task)
		return ok && unify(p.Result, cc.Result, m)
	case *types.Tuple:
		cc, ok := concrete.(*types.Tuple)
		if !ok || len(cc.Elems) != len(p.Elems) {
			return false
		}
		for i := range p.Elems {
			if !unify(p.Elems[i], cc.Elems[i], m) {
				return false
			}
		}
		return true
	case *types.Struct:
		cc, ok := concrete.(*types.Struct)
		if !ok || templateOf(cc) != templateOf(p) {
			return false
		}
		if len(p.TypeArgs) == 0 && len(p.TypeParams) > 0 {
			// bare template used as pattern: bind its params to args
			for i, tp := range p.TypeParams {
				if !unify(tp, cc.TypeArgs[i], m) {
					return false
				}
			}
			return true
		}
		for i := range p.TypeArgs {
			if !unify(p.TypeArgs[i], cc.TypeArgs[i], m) {
				return false
			}
		}
		return true
	case *types.Sealed:
		cc, ok := concrete.(*types.Sealed)
		if !ok || sealedTemplate(cc) != sealedTemplate(p) {
			return false
		}
		if len(p.TypeArgs) == 0 && len(p.TypeParams) > 0 {
			for i, tp := range p.TypeParams {
				if !unify(tp, cc.TypeArgs[i], m) {
					return false
				}
			}
			return true
		}
		for i := range p.TypeArgs {
			if !unify(p.TypeArgs[i], cc.TypeArgs[i], m) {
				return false
			}
		}
		return true
	case *types.Func:
		cc, ok := concrete.(*types.Func)
		if !ok || len(cc.Params) != len(p.Params) {
			return false
		}
		for i := range p.Params {
			if !unify(p.Params[i].Type, cc.Params[i].Type, m) {
				return false
			}
		}
		return unify(p.Ret, cc.Ret, m)
	}
	return types.Identical(pattern, concrete)
}

// ---------------------------------------------------------------------------
// rounds

func (c *Checker) runRound() *Program {
	c.changed = false
	c.prog = &Program{Release: c.release}
	c.prog.ResultType = func(ok, err types.Type) types.Type { return c.ResultType(ok, err) }
	c.queue = nil
	c.instances = map[string]*Func{}
	c.funcs = nil
	c.checkedGlobals = map[*Global]bool{}
	c.nextVar, c.nextLoop, c.nextTmp = 0, 0, 0
	for _, t := range c.templates {
		t.Instances = map[string]*Func{}
	}
	for _, s := range c.structs {
		// keep instantiations: they are pure type data
		_ = s
	}
	// Globals first (they may be referenced by any function).
	var globals []*Global
	for g := range c.globals {
		globals = append(globals, g)
	}
	sort.Slice(globals, func(i, j int) bool { return globals[i].Name < globals[j].Name })
	// dependency order: module order, then source order
	sort.SliceStable(globals, func(i, j int) bool {
		mi, mj := c.globalMod[globals[i]], c.globalMod[globals[j]]
		if mi != mj {
			return modIndex(c.pkg, mi) < modIndex(c.pkg, mj)
		}
		return c.globals[globals[i]].Pos.Start < c.globals[globals[j]].Pos.Start
	})
	for _, g := range globals {
		c.checkGlobal(g)
		c.prog.Globals = append(c.prog.Globals, g)
	}
	// Instantiate every non-generic function and method so that unused code
	// is still checked.
	for _, t := range c.templates {
		if t.Extern || len(t.TypeParams) > 0 || t.Trait != nil {
			continue
		}
		if t.Owner != nil && len(t.Owner.TypeParams) > 0 {
			continue
		}
		if t.Impl != nil && (len(t.Impl.TypeParams) > 0 || types.ContainsTypeParam(t.Impl.Target)) {
			continue
		}
		c.instantiate(t, nil, nil, t.Decl.Name.Pos)
	}
	for len(c.queue) > 0 {
		fn := c.queue[0]
		c.queue = c.queue[1:]
		if fn.checked {
			continue
		}
		c.checkBody(fn)
	}
	c.prog.TestMode = c.testMode
	c.prog.PanicType = c.panicType()
	for _, t := range c.tests {
		if inst, ok := t.Instances[""]; ok {
			c.prog.Tests = append(c.prog.Tests, inst)
		}
	}
	// entry point
	entry := c.pkg.Entry
	if entry != nil && !c.testMode {
		if sym := entry.Scope.LookupLocal("main"); sym != nil && sym.Kind == SymFunc {
			t := sym.Func
			if len(t.Sig.Params) != 0 || !types.IsUnit(t.Sig.Ret) {
				c.errorf(t.Decl.Name.Pos, "'main' must take no parameters and return nothing")
			}
			if inst, ok := t.Instances[""]; ok {
				c.prog.Main = inst
			}
		} else {
			c.errorf(source.Span{}, "no 'fun main()' in module %s", entry.Dir)
		}
	}
	// Finalise inferred error types and detect change.
	for _, t := range c.templates {
		if t.Sig == nil || !t.Sig.Effects.Throws || t.Decl.Effects.Error != nil {
			continue
		}
		var collected []types.Type
		for _, inst := range t.Instances {
			collected = append(collected, inst.inferredErrors...)
		}
		newErr := types.MakeErrorUnion(collected...)
		if !types.Identical(newErr, t.InferError) && !(newErr == nil && t.InferError == nil) {
			t.InferError = newErr
			c.changed = true
		}
	}
	c.prog.Funcs = c.funcs
	c.prog.Structs = nil
	for _, s := range c.structs {
		if len(s.TypeParams) == 0 {
			c.prog.Structs = append(c.prog.Structs, s)
		}
		for _, inst := range sortedStructInstances(s) {
			c.prog.Structs = append(c.prog.Structs, inst)
		}
	}
	return c.prog
}

func sortedStructInstances(s *types.Struct) []*types.Struct {
	keys := make([]string, 0, len(s.Instances))
	for k := range s.Instances {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*types.Struct, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.Instances[k])
	}
	return out
}

func modIndex(p *Package, m *Module) int {
	for i, x := range p.SortedModules() {
		if x == m {
			return i
		}
	}
	return 0
}

// instantiate creates (or finds) the concrete function for a template with
// the given type arguments. ownerSubst maps the owner struct's parameters.
func (c *Checker) instantiate(t *FuncTemplate, ownerSubst map[*types.TypeParam]types.Type, typeArgs []types.Type, span source.Span) *Func {
	c.resolveSignature(t)
	m := map[*types.TypeParam]types.Type{}
	for k, v := range ownerSubst {
		m[k] = v
	}
	var keyParts []string
	for i, tp := range t.TypeParams {
		if i < len(typeArgs) {
			m[tp] = typeArgs[i]
			keyParts = append(keyParts, types.Key(typeArgs[i]))
		}
	}
	// include owner args in the key
	var ownerKeys []string
	for _, tp := range sortedParams(ownerSubst) {
		ownerKeys = append(ownerKeys, tp.Name+"="+types.Key(ownerSubst[tp]))
	}
	key := strings.Join(ownerKeys, ";") + "|" + strings.Join(keyParts, ",")
	if len(ownerKeys) == 0 && len(keyParts) == 0 {
		key = ""
	}
	if inst, ok := t.Instances[key]; ok {
		if inst.inferring {
			c.errorf(span, "cannot infer the return type of recursive function '%s'; declare it explicitly", t.Name)
			inst.Sig.Ret = types.TInvalid
		}
		return inst
	}
	sig := types.Subst(t.Sig, m).(*types.Func)
	if sig == t.Sig {
		cp := *t.Sig
		sig = &cp
	}
	if sig.Effects.Throws && sig.Effects.Error == nil {
		sig.Effects.Error = t.InferError
	}
	name := t.Mangled
	if key != "" {
		name += "<" + key + ">"
	}
	fn := &Func{Name: mangleName(name), Display: t.Name, Sig: sig, Extern: t.Extern, Mut: t.Decl.Mut, Span: t.Decl.Name.Pos}
	if _, ok := t.Attrs["inline"]; ok {
		fn.Inline = 1
	}
	if _, ok := t.Attrs["noinline"]; ok {
		fn.Inline = -1
	}
	if t.Extern {
		fn.Name = t.Mangled // the C symbol
	}
	fn.tmpl = t
	fn.subst = m
	t.Instances[key] = fn
	c.instances[fn.Name] = fn
	c.funcs = append(c.funcs, fn)
	if t.InferRet {
		// Infer the return type now so callers can use it.
		fn.inferring = true
		c.checkBody(fn)
		fn.inferring = false
		fn.checked = true
		return fn
	}
	if !t.Extern {
		c.queue = append(c.queue, fn)
	}
	return fn
}

func sortedParams(m map[*types.TypeParam]types.Type) []*types.TypeParam {
	var out []*types.TypeParam
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func mangleName(s string) string {
	r := strings.NewReplacer("<", "_", ">", "", ", ", "_", ",", "_", "*", "ptr_", "?", "_opt", "(", "tup_", ")", "", " ", "", "|", "_", "!", "never", "=", "_", ";", "_", ":", "_")
	return "v_" + r.Replace(s)
}

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// findImplFor finds the impl of trait for a concrete type, if any.
func (c *Checker) findImplFor(t types.Type, trait *types.Trait) *Impl {
	for _, impl := range c.impls[trait] {
		m := map[*types.TypeParam]types.Type{}
		if unify(impl.Target, t, m) {
			return impl
		}
	}
	return nil
}

// resolveAssoc implements types.AssocResolver: the binding of an
// associated type for a concrete implementing type.
func (c *Checker) resolveAssoc(base types.Type, trait *types.Trait, name string) types.Type {
	for _, impl := range c.impls[trait] {
		m := map[*types.TypeParam]types.Type{}
		if !unify(impl.Target, base, m) {
			continue
		}
		bt, ok := impl.AssocTypes[name]
		if !ok {
			return nil
		}
		return types.Subst(bt, m)
	}
	return nil
}

// projectAssoc resolves `Base::Name`.
func (c *Checker) projectAssoc(env *typeEnv, base types.Type, name string, span source.Span) types.Type {
	if env.implAssoc != nil && env.self != nil && types.Identical(base, env.self) {
		if bt, ok := env.implAssoc[name]; ok {
			return bt
		}
	}
	if tp, ok := base.(*types.TypeParam); ok {
		for _, bound := range tp.Bounds {
			if containsString(bound.AssocTypes, name) {
				return &types.Assoc{Base: tp, Trait: bound, Name: name}
			}
		}
		c.errorf(span, "type parameter '%s' has no bound declaring an associated type '%s'", tp.Name, name)
		return types.TInvalid
	}
	if types.ContainsTypeParam(base) {
		// e.g. MapIter<I, U>::Item — resolve through the impl once concrete
		for _, tr := range c.traits {
			if containsString(tr.AssocTypes, name) {
				if impl := c.findImplFor(base, tr); impl != nil {
					return &types.Assoc{Base: base, Trait: tr, Name: name}
				}
			}
		}
		c.errorf(span, "cannot resolve '%s::%s'", base, name)
		return types.TInvalid
	}
	var found types.Type
	var foundTrait *types.Trait
	for _, tr := range c.traits {
		if !containsString(tr.AssocTypes, name) {
			continue
		}
		if r := c.resolveAssoc(base, tr, name); r != nil {
			if found != nil {
				c.errorf(span, "'%s::%s' is ambiguous: declared by traits '%s' and '%s'", base, name, foundTrait.Name, tr.Name)
				return types.TInvalid
			}
			found, foundTrait = r, tr
		}
	}
	if found == nil {
		c.errorf(span, "'%s' has no associated type '%s'", base, name)
		return types.TInvalid
	}
	return found
}

// checkHashable reports whether a type can be a map key or set element:
// anything with structural equality except mutable collections and
// function values.
func (c *Checker) checkHashable(t types.Type, span source.Span) bool {
	if types.ContainsTypeParam(t) || types.IsInvalid(t) {
		return true
	}
	if !hashable(t) {
		c.errorf(span, "'%s' cannot be a map key or set element: it has no structural equality", t)
		return false
	}
	return true
}

func hashable(t types.Type) bool {
	switch t := t.(type) {
	case *types.Basic:
		return t.Kind != types.Unit && t.Kind != types.Never && t.Kind != types.Invalid
	case *types.Pointer:
		return true
	case *types.Nullable:
		return hashable(t.Elem)
	case *types.Tuple:
		for _, e := range t.Elems {
			if !hashable(e) {
				return false
			}
		}
		return true
	case *types.Struct:
		for _, f := range t.Fields {
			if !hashable(f.Type) {
				return false
			}
		}
		return true
	case *types.Sealed:
		for _, v := range t.Variants {
			if !hashable(v) {
				return false
			}
		}
		return true
	case *types.List:
		return !t.Mutable && hashable(t.Elem)
	}
	return false
}
