// Package sema resolves names, checks types and effects, and lowers the
// syntax tree to the HIR consumed by code generation.
package sema

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

type Checker struct {
	namedImports []namedImport // every bound name of a braced import (D85)
	pkg          *Package
	diags        *source.Diagnostics
	// roundDiags collects diagnostics of the current inference round; only
	// the final round's are kept (D45 fixpoint).
	roundDiags *source.Diagnostics
	seen       map[string]bool
	// exportedC: the `extern "C" fun`s by C symbol (D69), by declaration so
	// a later inference round sees the same one again
	exportedC map[string]*ast.FunDecl
	varFixes  map[*ast.Field]bool            // fields already offered the `var` insertion (fixes.go)
	initDecl  map[*types.Struct]*ast.FunDecl // the synthetic `$init` method of structs with an `init { }` block

	universe *Scope
	prog     *Program
	index    *Index // reference recording for editor tooling; nil when compiling
	release  bool
	testMode bool

	// declaration tables
	structs        []*types.Struct // templates
	sealeds        []*types.Sealed
	enums          []*types.Enum
	enumDecl       map[*types.Enum]*declCtx
	ordering       *types.Enum // the prelude's Ordering, once looked up (enum.go)
	traits         []*types.Trait
	templates      []*FuncTemplate
	impls          map[*types.Trait][]*Impl
	extends        []*Impl                                    // `extend Type { }` blocks (Trait == nil)
	methods        map[*types.Struct]map[string]*FuncTemplate // inherent, by template
	staticVals     map[*types.Struct]map[string]*Symbol       // `static val` members, as globals in the type's namespace
	staticOwner    map[*Global]*types.Struct                  // the struct a `static val` global belongs to
	globals        map[*Global]*ast.ValDecl
	globalMod      map[*Global]*Module
	globalFile     map[*Global]*ast.File
	structDecl     map[*types.Struct]*declCtx
	sealedDecl     map[*types.Sealed]*declCtx
	traitDecl      map[*types.Trait]*declCtx
	resultTmpl     *types.Sealed
	traitMethodTPs map[string][]*types.TypeParam
	traitStatic    map[string]bool  // "Trait.method" declared `static fun`
	globalVars     map[*Global]*Var // the Var standing for each module-level binding
	hooks          *types.Hooks     // what types.Subst calls back into
	// error-position types seen before the impls were collected, checked
	// at the end of collection; collected marks that point
	pendingErrorChecks []pendingErrorCheck
	pendingBoundChecks []func()
	pathReported       bool                  // lookupTypeName reported why a qualified type path failed
	stdFiles           map[*source.File]bool // stdSpan
	pendingAssocChecks []func()              // associated-type bounds, run once all impls exist
	pendingDerives     []func()              // derived impl bodies and supertrait impls, once all impls exist (D58)
	deriveFailed       map[*Impl]bool
	debugDerive        bool                 // VELES_DEBUG_DERIVE: print every synthesized declaration
	syntheticSpans     map[source.Span]bool // spans that derived code carries; no hover there
	collected          bool
	collectRefs        int // index refs recorded by collect(); rounds reset only past this
	tests              []*FuncTemplate
	testNames          map[*Module]map[string]source.Span // each module's qualified test and suite names, for duplicates (D78)
	suites             int                                // suites declared, for unique helper symbols
	suiteHelpers       map[string]string                  // a suite helper's name -> its suite, for "unknown function" (D78)
	optionTmpl         *types.Sealed

	// per-round state
	queue          []*Func
	instances      map[string]*Func
	funcs          []*Func
	checkedGlobals map[*Global]bool
	changed        bool
	nextVar        int
	tupleCmp       map[string]*Func // synthesized tuple comparisons by type key (tuple_order.go)
	enumFns        map[string]*Func // synthesized enum functions by type key and name (enum.go)
	nextLoop       int
	nextTmp        int
	nextLambda     int
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
// CheckTests is Check for `veles test`: no main is required and the tests
// (`test "..." { }`, D78) become the entry point.
func CheckTests(pkg *Package, diags *source.Diagnostics, release bool) *Program {
	return check(pkg, diags, release, true)
}

func Check(pkg *Package, diags *source.Diagnostics, release bool) *Program {
	if pkg.NeedMain {
		dropTestFiles(pkg) // a build or a run never sees *.test.vs (D78)
	}
	return check(pkg, diags, release, false)
}

func check(pkg *Package, diags *source.Diagnostics, release bool, testMode bool) *Program {
	return checkWith(pkg, diags, release, testMode, nil)
}

func checkWith(pkg *Package, diags *source.Diagnostics, release bool, testMode bool, index *Index) *Program {
	prog, _ := checkCollect(pkg, diags, release, testMode, index)
	return prog
}

// checkCollect is checkWith, handing the checker back so a caller that
// needs more than the HIR — `veles explain --derive` wants the synthesized
// implements — can read it.
func checkCollect(pkg *Package, diags *source.Diagnostics, release bool, testMode bool, index *Index) (*Program, *Checker) {
	c := &Checker{
		deriveFailed:   map[*Impl]bool{},
		syntheticSpans: map[source.Span]bool{},
		debugDerive:    os.Getenv("VELES_DEBUG_DERIVE") != "",
		globalVars:     map[*Global]*Var{},
		index:          index,
		pkg:            pkg,
		diags:          diags,
		seen:           map[string]bool{},
		impls:          map[*types.Trait][]*Impl{},
		methods:        map[*types.Struct]map[string]*FuncTemplate{},
		staticVals:     map[*types.Struct]map[string]*Symbol{},
		initDecl:       map[*types.Struct]*ast.FunDecl{},
		staticOwner:    map[*Global]*types.Struct{},
		globals:        map[*Global]*ast.ValDecl{},
		globalMod:      map[*Global]*Module{},
		globalFile:     map[*Global]*ast.File{},
		structDecl:     map[*types.Struct]*declCtx{},
		sealedDecl:     map[*types.Sealed]*declCtx{},
		enumDecl:       map[*types.Enum]*declCtx{},
		traitDecl:      map[*types.Trait]*declCtx{},
		traitMethodTPs: map[string][]*types.TypeParam{},
		traitStatic:    map[string]bool{},
		release:        release,
		testMode:       testMode,
	}
	c.hooks = &types.Hooks{
		AssocResolver: c.resolveAssoc,
		StructInstantiator: func(tmpl *types.Struct, args []types.Type) types.Type {
			return c.instantiateStruct(tmpl, args, source.Span{})
		},
		SealedInstantiator: func(tmpl *types.Sealed, args []types.Type) types.Type {
			return c.instantiateSealed(tmpl, args, source.Span{})
		},
	}
	c.roundDiags = diags
	c.buildUniverse()
	c.collect()
	if c.index != nil {
		c.collectRefs = len(c.index.Refs)
	}
	if diags.HasErrors() {
		return nil, c
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
		c.receiverPass(prog)
		c.inferSuspension(prog)
		c.indexInferred(prog)
	}
	diags.Items = append(diags.Items, c.roundDiags.Items...)
	c.dropSyntheticRefs()
	c.indexImpls()
	if diags.HasErrors() {
		return nil, c
	}
	return prog, c
}

// dropSyntheticRefs removes the references derived code recorded (D58):
// every node of a synthesized body carries the span of the declaration
// that asked for it, which is not where any of them is.
func (c *Checker) dropSyntheticRefs() {
	if c.index == nil || len(c.syntheticSpans) == 0 {
		return
	}
	kept := c.index.Refs[:0]
	for _, r := range c.index.Refs {
		if !c.syntheticSpans[r.Span] {
			kept = append(kept, r)
		}
	}
	c.index.Refs = kept
}

// noteDiagnostic, when a test sets it, is told the format of every
// diagnostic the checker reports: the conformance suite's coverage
// (conform_test.go). Set once, before any checking starts.
var noteDiagnostic func(format string)

func (c *Checker) errorf(span source.Span, format string, args ...any) {
	if noteDiagnostic != nil {
		noteDiagnostic(format)
	}
	msg := fmt.Sprintf(format, args...)
	key := span.String() + msg
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.roundDiags.Errorf(span, "%s", msg)
}

func (c *Checker) warnf(span source.Span, format string, args ...any) {
	if noteDiagnostic != nil {
		noteDiagnostic(format)
	}
	msg := fmt.Sprintf(format, args...)
	key := "w" + span.String() + msg
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.roundDiags.Warnf(span, "%s", msg)
}

// warnFix is warnf with an automatic correction attached (a lint).
func (c *Checker) warnFix(span source.Span, fix *source.Fix, format string, args ...any) {
	if noteDiagnostic != nil {
		noteDiagnostic(format)
	}
	msg := fmt.Sprintf(format, args...)
	key := "w" + span.String() + msg
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.roundDiags.Items = append(c.roundDiags.Items, source.Diagnostic{Severity: source.Warning, Span: span, Message: msg, Fix: fix})
}

// errorFix is errorf with an automatic correction attached: for a form
// that was removed from the language and has a mechanical replacement.
func (c *Checker) errorFix(span source.Span, fix *source.Fix, format string, args ...any) {
	if noteDiagnostic != nil {
		noteDiagnostic(format)
	}
	msg := fmt.Sprintf(format, args...)
	key := span.String() + msg
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.roundDiags.Items = append(c.roundDiags.Items, source.Diagnostic{Severity: source.Error, Span: span, Message: msg, Fix: fix})
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
			m.Imports[f].used = map[string]bool{}
			for _, d := range f.Decls {
				c.declare(m, f, d)
			}
		}
	}
	// 1b. the prelude is in scope everywhere (D24), except the names that
	// live in a module of their own (D75): `codec.Value`, `recursion.Depth`
	if prelude, ok := c.pkg.Modules["std/prelude"]; ok {
		for _, sym := range prelude.Scope.symbols {
			if !sym.Pub {
				continue
			}
			if home := PreludeHome(sym.Name); home != "" {
				if m, ok := c.pkg.Modules["std/"+home]; ok {
					m.Scope.Insert(sym)
				}
				continue
			}
			c.universe.Insert(sym)
		}
	}
	// 2. wire imports (per file)
	for _, m := range mods {
		for _, f := range m.Files {
			for _, d := range f.Decls {
				if u, ok := d.(*ast.UseDecl); ok {
					for _, s := range u.Specs {
						c.declareUse(m, f, s)
					}
				}
			}
		}
	}
	// 3. resolve signatures, fields, variants, traits, impls
	for _, e := range c.enums {
		c.resolveEnum(e)
	}
	for _, t := range c.traits {
		c.resolveSupers(t) // before any bound is read: bounds carry supertraits
	}
	c.checkSuperCycles()
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
					if impl.Extend {
						c.declareExtend(m, f, impl)
					} else {
						c.declareImpl(m, f, impl)
					}
				}
			}
		}
	}
	c.checkSealedVariants()
	c.deriveEnumCodecs()
	// derived bodies, supertrait impls and variant impls, in the order
	// they were asked for; each may ask for more (D58)
	for len(c.pendingDerives) > 0 {
		next := c.pendingDerives[0]
		c.pendingDerives = c.pendingDerives[1:]
		next()
	}
	for _, check := range c.pendingAssocChecks {
		check()
	}
	c.pendingAssocChecks = nil
	for _, t := range c.templates {
		c.resolveSignature(t)
	}
	c.checkCoherence()
	// every error set is resolved (and its members checked) even if unused
	for _, m := range mods {
		for _, sym := range m.Scope.symbols {
			if sym.Alias != nil {
				c.resolveAlias(sym)
			}
			if sym.TypeAlias != nil {
				c.aliasBody(sym)
			}
		}
	}
	// error sets and error fields were resolved before the impls existed
	for _, pc := range c.pendingErrorChecks {
		c.checkErrorType(pc.t, pc.span)
	}
	c.pendingErrorChecks = nil
	c.collected = true // the bound checks below need impls, and may defer no further
	for _, check := range c.pendingBoundChecks {
		check()
	}
	c.pendingBoundChecks = nil
	c.collected = true
	// the declaration sites themselves: a hover on `struct Notes {` or on a
	// field where it is declared shows the same spelled-out form as a use
	if c.index != nil {
		for s, ctx := range c.structDecl {
			d, ok := ctx.decl.(*ast.StructDecl)
			if !ok || s.Template != nil {
				continue
			}
			c.resolveStruct(s)
			c.refType(d.Name.Pos, s.Name, s, d.Name.Pos)
			for _, fld := range s.Fields {
				for _, af := range d.Fields {
					if af.Name.Name == fld.Name {
						c.refField(af.Name.Pos, s, fld)
					}
				}
			}
		}
		for tr, ctx := range c.traitDecl {
			if d, ok := ctx.decl.(*ast.TraitDecl); ok && !d.Sealed {
				c.refType(d.Name.Pos, tr.Name, tr, d.Name.Pos)
				for _, m := range d.Methods {
					if m.Body == nil && m.ExprBody == nil {
						c.refTraitMethod(m.Name.Pos, tr, m.Name.Name)
					}
				}
			}
		}
		for s, ctx := range c.sealedDecl {
			if d, ok := ctx.decl.(*ast.TraitDecl); ok && s.Template == nil {
				c.refType(d.Name.Pos, s.Name, s, d.Name.Pos)
			}
		}
		// every function and method at its declaration, checked or not
		for _, t := range c.templates {
			if t.Decl != nil && t.Decl.Name.Pos.IsValid() {
				c.refFunc(t.Decl.Name.Pos, t)
			}
		}
		c.indexDerived(mods)
	}
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
	case *ast.TestDecl:
		c.declareTest(m, f, d, fileSuite(f), nil)
	case *ast.SuiteDecl:
		c.declareSuite(m, f, d, fileSuite(f), nil)
	case *ast.ExternBlock:
		for _, fn := range d.Funs {
			t := c.newTemplate(m, f, fn, nil, nil)
			t.Extern = true
			t.Mangled = fn.Name.Name // C symbol
			c.insert(m, &Symbol{Name: fn.Name.Name, Kind: SymFunc, Pub: fn.Pub, Module: m, Span: fn.Name.Pos, Func: t})
		}
	case *ast.StructDecl:
		if d.Variant != nil {
			c.attrsOf(d.Attrs, "variant")
		} else {
			c.attrsOf(d.Attrs, "struct")
		}
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
		if d.Init != nil {
			// `init { }` is checked and compiled as a hidden method the
			// constructor calls on the freshly built value; `$` keeps it out
			// of reach of any name a user can write (D28 v0.30)
			decl := &ast.FunDecl{Doc: "", Private: true, Name: ast.Ident{Name: "$init", Pos: d.InitPos}, Params: d.InitParams, Body: d.Init, Pos: d.InitPos.To(d.Init.Pos)}
			c.methods[s]["$init"] = c.newTemplate(m, f, decl, s, ctx.tps)
			c.initDecl[s] = decl
		}
		// `static val`: a module global that lives in the type's namespace
		// (`Status.ok`), initialised with the other globals
		for _, sv := range d.Statics {
			if len(d.TypeParams) > 0 {
				c.errorf(sv.Name.Pos, "a generic struct cannot have a 'static val' (it would need one value per instantiation); use a 'static fun'")
				continue
			}
			if _, dup := c.methods[s][sv.Name.Name]; dup {
				c.errorf(sv.Name.Pos, "'%s' is already a static function of '%s'", sv.Name.Name, s.Name)
				continue
			}
			if c.staticVals[s] == nil {
				c.staticVals[s] = map[string]*Symbol{}
			}
			if _, dup := c.staticVals[s][sv.Name.Name]; dup {
				c.errorf(sv.Name.Pos, "duplicate static '%s' on '%s'", sv.Name.Name, s.Name)
				continue
			}
			c.attrsOf(sv.Attrs, "value")
			g := &Global{Name: m.prefix() + "." + d.Name.Name + "." + sv.Name.Name, Display: d.Name.Name + "." + sv.Name.Name, Span: sv.Name.Pos}
			c.globals[g] = sv
			c.globalMod[g] = m
			c.globalFile[g] = f
			c.staticOwner[g] = s
			c.staticVals[s][sv.Name.Name] = &Symbol{Name: sv.Name.Name, Kind: SymGlobal, Pub: sv.Pub, Module: m, Span: sv.Name.Pos, Global: g}
		}
	case *ast.ErrorAliasDecl:
		c.attrsOf(d.Attrs, "error")
		c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymType, Pub: d.Pub, Module: m, Span: d.Name.Pos,
			Alias: &errorAlias{decl: d, module: m, file: f}})
	case *ast.TypeAliasDecl:
		c.attrsOf(d.Attrs, "type")
		c.insert(m, &Symbol{Name: d.Name.Name, Kind: SymType, Pub: d.Pub, Module: m, Span: d.Name.Pos,
			TypeAlias: &typeAlias{decl: d, module: m, file: f}})
	case *ast.EnumDecl:
		c.declareEnum(m, f, d)
	case *ast.TraitDecl:
		if d.Sealed {
			c.attrsOf(d.Attrs, "sealed")
		} else {
			c.attrsOf(d.Attrs, "trait")
		}
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
			if len(d.Supers) > 0 {
				// here too: a sealed trait without methods never reaches
				// resolveSupers, and its supertraits were silently dropped
				c.errorf(d.Supers[0].Span(), "a sealed trait has no supertraits: its variants implement traits one by one (D12)")
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
		// a bare `throws` on a method means "each impl decides": the trait
		// gets an implicit associated type `Error` (D40, v0.24)
		for _, md := range d.Methods {
			if md.Effects.Throws && md.Effects.Error == nil && !containsString(t.AssocTypes, "Error") {
				t.AssocTypes = append(t.AssocTypes, "Error")
				t.ImplicitError = true
				break
			}
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
	t.TestCode = d.Test || isTestFile(f)
	if a, isTest := t.Attrs["test"]; isTest {
		c.oldTestSpelling(a, d)
		if len(d.Params) > 0 || d.Ret != nil || owner != nil {
			c.errorf(d.Name.Pos, "a test takes no parameters and returns nothing")
		}
		t.TestCode = true
		c.tests = append(c.tests, t)
	}
	if d.Test && owner != nil {
		c.errorf(d.Name.Pos, "a 'test fun' is a top-level helper, not a method")
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

func (c *Checker) declareUse(m *Module, f *ast.File, u *ast.UseSpec) {
	dep := m.Uses[u]
	if dep == nil {
		return // loader already reported
	}
	scope := m.Imports[f]
	name := u.Path[len(u.Path)-1].Name
	if u.Alias != nil {
		name = u.Alias.Name
	}
	if old := scope.Insert(&Symbol{Name: name, Kind: SymModule, Mod: dep, Span: u.Pos}); old != nil {
		c.errorf(u.Pos, "'%s' is already imported in this file", name)
	}
	// the module name in the `use` line hovers like any other reference
	c.refSym(u.Path[len(u.Path)-1].Pos, &Symbol{Name: name, Kind: SymModule, Mod: dep, Span: u.Pos})
	c.declareUseNames(m, f, u, dep, scope)
	if u.Pub {
		c.reexport(m, u, dep) // D89
	}
}

// declareUseNames binds the names of `use m { f, T as U }` (D85) in the
// file's import scope. The bound symbol is the module's own, so a call or a
// type written bare means exactly what the qualified one would; a rename
// gets a copy under its new name.
func (c *Checker) declareUseNames(m *Module, f *ast.File, u *ast.UseSpec, dep *Module, scope *Scope) {
	mod := pathString(u.Path)
	for _, n := range u.Names {
		member := dep.Scope.LookupLocal(n.Name.Name)
		if member == nil {
			c.noNamedMember(n, mod, dep)
			continue
		}
		if !member.Pub {
			c.errorf(n.Name.Pos, "'%s' is private to module '%s'%s (M5)", n.Name.Name, mod, privateHint(dep.Path))
			continue
		}
		if member.Kind == SymFunc && member.Func.TestCode && !isTestFile(f) {
			c.errorf(n.Name.Pos, "'%s' is test code (a 'test fun', or declared in a *.test.vs file); only tests can use it, and a build leaves it out (D78)", n.Name.Name)
			continue
		}
		bound := member
		name := n.Name.Name
		if n.Alias != nil {
			name = n.Alias.Name
			cp := *member
			cp.Name = name
			bound = &cp
		}
		if own := m.Scope.LookupLocal(name); own != nil {
			c.errorf(n.Pos, "'%s' is already declared in this module (at %s): import it under another name, '%s { %s as … }'", name, own.Span, mod, n.Name.Name)
			continue
		}
		if old := scope.Insert(bound); old != nil {
			c.errorf(n.Pos, "'%s' is already imported in this file: import one of them under another name, '%s { %s as … }'", name, mod, n.Name.Name)
			continue
		}
		c.refSym(n.Name.Pos, member)
		c.namedImports = append(c.namedImports, namedImport{u, n, scope, name})
	}
}

// namedImport is a bound name of a braced import, kept for the unused check.
type namedImport struct {
	spec  *ast.UseSpec
	name  *ast.UseName
	scope *Scope
	bound string
}

// finishImportRefs makes the index tell a braced import's names apart from the
// module's own (D85). A bare use of a renamed name becomes a reference to the
// alias — its own declaration, at the alias in the `use` line — so renaming it
// leaves the member alone, and renaming the member leaves the alias uses
// alone; a bare use of an unrenamed name says which module it came from.
func (c *Checker) finishImportRefs() {
	if c.index == nil {
		return
	}
	for _, ni := range c.namedImports {
		file := ni.name.Name.Pos.File
		var member *Ref
		for i := range c.index.Refs {
			if r := &c.index.Refs[i]; r.Span == ni.name.Name.Pos {
				member = r
			}
		}
		if file == nil || member == nil {
			continue
		}
		def := member.Key()
		from := "module " + pathString(ni.spec.Path)
		for i := range c.index.Refs {
			r := &c.index.Refs[i]
			if r.Span.File != file || r.Span == ni.name.Name.Pos || r.Key() != def {
				continue
			}
			if r.Span.Start > 0 && file.Content[r.Span.Start-1] == '.' {
				continue // written through the module
			}
			if file.Content[r.Span.Start:r.Span.End] != ni.bound {
				continue
			}
			if ni.name.Alias != nil {
				r.Name = ni.bound
				r.Def = ni.name.Alias.Pos
				r.Family = source.Span{}
				r.Where = "alias of " + pathString(ni.spec.Path) + "." + ni.name.Name.Name
			} else if r.Where == "" {
				r.Where = from
			}
		}
		if ni.name.Alias != nil {
			decl := *member
			decl.Span, decl.Def, decl.Name = ni.name.Alias.Pos, ni.name.Alias.Pos, ni.bound
			decl.Family = source.Span{}
			decl.Where = "alias of " + pathString(ni.spec.Path) + "." + ni.name.Name.Name
			c.index.Refs = append(c.index.Refs, decl)
		}
	}
}

// lintUnusedNames warns about each imported name no lookup in its file
// found, with the edit that removes it (D85).
func (c *Checker) lintUnusedNames() {
	for _, ni := range c.namedImports {
		if ni.scope.used[ni.bound] || ni.spec.Pub {
			continue // a re-exported name is used by whoever imports this module (D89)
		}
		c.warnFix(ni.name.Pos, unusedNameFix(ni.spec, ni.name), "'%s' is imported but never used", ni.bound)
	}
}

// unusedNameFix removes one name from a braced import — with its comma —
// or, when it is the only one, the whole brace group.
func unusedNameFix(u *ast.UseSpec, n *ast.UseName) *source.Fix {
	title := "Remove '" + n.Name.Name + "' from the import"
	at := -1
	for i, x := range u.Names {
		if x == n {
			at = i
		}
	}
	span := func(from, to int) source.Span { return source.Span{File: n.Pos.File, Start: from, End: to} }
	switch {
	case at < 0:
		return nil
	case len(u.Names) == 1:
		end := u.Path[len(u.Path)-1].Pos.End
		if u.Alias != nil {
			end = u.Alias.Pos.End
		}
		return fixReplace(title, span(end, u.Pos.End), "")
	case at+1 < len(u.Names):
		return fixReplace(title, span(n.Pos.Start, u.Names[at+1].Pos.Start), "")
	default:
		return fixReplace(title, span(u.Names[at-1].Pos.End, n.Pos.End), "")
	}
}

// noNamedMember reports a name a braced import asks of a module that does
// not have it, with the closest member as the fix.
func (c *Checker) noNamedMember(n *ast.UseName, mod string, dep *Module) {
	name := n.Name.Name
	if sym := c.universe.LookupLocal(name); sym != nil && sym.Pub {
		c.errorf(n.Name.Pos, "module '%s' has no declaration '%s'; '%s' is global (the prelude): it needs no import", mod, name, name)
		return
	}
	if home := PreludeHome(name); home != "" && home != mod {
		c.errorf(n.Name.Pos, "module '%s' has no declaration '%s'; it is '%s.%s': import it from '%s'", mod, name, home, name, home)
		return
	}
	if hit := didYouMean(name, moduleMembers(dep)); hit != "" {
		c.errorFix(n.Name.Pos, typoFix(n.Name.Pos, hit), "module '%s' has no declaration '%s'; did you mean '%s'?", mod, name, hit)
		return
	}
	c.errorf(n.Name.Pos, "module '%s' has no declaration '%s'", mod, name)
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
	// errorPos is set while resolving a type in error position (after
	// `throws`, a union member, an error's field): the only places a named
	// error set may appear (D45).
	errorPos bool
}

// errorAlias is an `error Name = A | B` declaration awaiting resolution.
type errorAlias struct {
	decl      *ast.ErrorAliasDecl
	module    *Module
	file      *ast.File
	resolving bool
}

// typeAlias is a `type Name<T> = Type` declaration (D55) awaiting resolution.
type typeAlias struct {
	decl      *ast.TypeAliasDecl
	module    *Module
	file      *ast.File
	tps       []*types.TypeParam
	body      types.Type
	resolving bool
	done      bool
}

// aliasBody resolves an alias's right-hand side once, in an environment
// holding its own type parameters.
func (c *Checker) aliasBody(sym *Symbol) types.Type {
	a := sym.TypeAlias
	if a.done {
		return a.body
	}
	if a.resolving {
		c.errorf(a.decl.Name.Pos, "type alias '%s' refers to itself; a recursive type is a sealed trait or a struct (D55)", sym.Name)
		return types.TInvalid
	}
	a.resolving = true
	env := &typeEnv{module: a.module, file: a.file, tps: map[string]*types.TypeParam{}}
	for i, tp := range a.decl.TypeParams {
		if len(tp.Bounds) > 0 {
			c.errorf(tp.Name.Pos, "a type alias parameter takes no bounds; state '%s: ...' where the alias is used (D55)", tp.Name.Name)
		}
		p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: sym.Name}
		a.tps = append(a.tps, p)
		env.tps[tp.Name.Name] = p
	}
	if a.decl.Type == nil {
		a.body = types.TInvalid
	} else {
		a.body = c.resolveType(env, a.decl.Type)
	}
	a.resolving = false
	a.done = true
	return a.body
}

// resolveTypeAlias is an alias used as a type: its body with the
// arguments substituted, displayed under the alias's name. The result is
// the underlying type itself (a display name on a structural type, D55);
// a named type keeps its own name.
func (c *Checker) resolveTypeAlias(sym *Symbol, args []types.Type, span source.Span) types.Type {
	a := sym.TypeAlias
	body := c.aliasBody(sym)
	if types.IsInvalid(body) {
		return body
	}
	if len(args) != len(a.tps) {
		if len(a.tps) == 0 {
			c.errorf(span, "'%s' is not generic: write it without '<...>'", sym.Name)
		} else {
			c.errorf(span, "'%s' expects %d type arguments, got %d", sym.Name, len(a.tps), len(args))
		}
		return types.TInvalid
	}
	display := sym.Name
	t := body
	if len(a.tps) > 0 {
		m := map[*types.TypeParam]types.Type{}
		parts := make([]string, len(args))
		for i, tp := range a.tps {
			m[tp] = args[i]
			parts[i] = args[i].String()
		}
		t = c.hooks.Subst(body, m)
		display += "<" + strings.Join(parts, ", ") + ">"
	}
	return types.Aliased(t, display)
}

// aliasDetail is the hover line of an alias: its name, its definition as
// written, and the full expansion when that differs.
func aliasDetail(sym *Symbol, t types.Type) string {
	a := sym.TypeAlias
	head := "type " + sym.Name
	if len(a.tps) > 0 {
		head += typeParamList(a.tps)
	}
	one := types.Unaliased(a.body, false).String()
	full := types.Unaliased(a.body, true).String()
	s := head + " = " + one
	if full != one {
		s += "  (= " + full + ")"
	}
	return s
}

// symType is the type a SymType symbol names, resolving a non-generic
// alias on the way; generic aliases need arguments and yield nil here.
func (c *Checker) symType(sym *Symbol) types.Type {
	if sym.TypeAlias != nil {
		if len(sym.TypeAlias.decl.TypeParams) > 0 {
			c.aliasBody(sym)
			return nil
		}
		return c.resolveTypeAlias(sym, nil, sym.Span)
	}
	return sym.Type
}

// resolveAlias resolves a named error set on first use.
func (c *Checker) resolveAlias(sym *Symbol) types.Type {
	if sym.Type != nil {
		return sym.Type
	}
	a := sym.Alias
	if a.resolving {
		c.errorf(a.decl.Name.Pos, "error set '%s' refers to itself", sym.Name)
		return types.TInvalid
	}
	a.resolving = true
	env := &typeEnv{module: a.module, file: a.file, tps: map[string]*types.TypeParam{}, errorPos: true}
	var syntax []ast.Type
	if u, ok := a.decl.Members.(*ast.ErrorUnionType); ok {
		syntax = u.Members
	} else {
		syntax = []ast.Type{a.decl.Members}
	}
	var members []types.Type
	for _, m := range syntax {
		mt := c.resolveType(env, m)
		members = append(members, mt)
		// a member that is itself a set has already checked its own members
		if _, nested := mt.(*types.ErrorUnion); !nested && !types.IsInvalid(mt) {
			c.deferErrorCheck(mt, m.Span())
		}
	}
	a.resolving = false
	var t types.Type = types.MakeErrorUnion(members...)
	for _, m := range members {
		if types.IsInvalid(m) {
			t = types.TInvalid
		}
	}
	if _, isUnion := t.(*types.ErrorUnion); isUnion {
		t = types.Aliased(t, sym.Name) // hover and diagnostics say `GetErrors`, not the members
	}
	sym.Type = t
	return t
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
				c.noMember(path[i-1].Pos, path[i].Pos, path[i-1].Name, path[i].Name, sym.Mod)
				c.pathReported = true
				return nil, nil
			}
			if !next.Pub {
				c.errorf(path[i].Pos, "'%s' is private to module '%s'%s (M5)", path[i].Name, sym.Mod.Path, privateHint(sym.Mod.Path))
				c.pathReported = true
				return nil, nil
			}
			sym = next
		case SymType:
			// Sealed.Variant
			if s, ok := c.symType(sym).(*types.Sealed); ok {
				v := s.VariantByName(path[i].Name)
				if v == nil {
					c.errorf(path[i].Pos, "'%s' has no variant '%s'", s.Name, path[i].Name)
					c.pathReported = true
					return nil, nil
				}
				return &Symbol{Name: v.Name, Kind: SymType, Type: v, Pub: true}, s
			}
			c.errorf(path[i].Pos, "'%s' is not a module or sealed trait", path[i-1].Name)
			c.pathReported = true
			return nil, nil
		default:
			c.errorf(path[i-1].Pos, "'%s' is not a module", path[i-1].Name)
			c.pathReported = true
			return nil, nil
		}
	}
	return sym, nil
}

func (c *Checker) resolveType(env *typeEnv, t ast.Type) types.Type {
	switch t := t.(type) {
	case nil:
		return types.TUnit
	case *ast.ResolvedType:
		return t.T.(types.Type) // synthesized by the compiler (D58)
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
			// the built-in generics are not in any scope: a declaration or
			// import of the same name (a user's `struct Task`) shadows them,
			// as it does at a call site (`f.lookup` in checkCall)
			if sym, _ := c.lookupTypeName(env, t.Path); sym != nil && sym.Kind == SymType && c.universe.LookupLocal(name) != sym {
				name = ""
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
		if tp, ok := env.tps[t.Path[0].Name]; ok && len(t.Path) > 1 {
			// `I.Item`: an associated type projected from a type parameter
			// (the dot is a path in the grammar; a type parameter in scope
			// cannot also name a module)
			var base types.Type = tp
			for i := 1; i < len(t.Path); i++ {
				seg := t.Path[i]
				// `T.Trait.Item`: the associated type of one bound, named
				// through it (D84) — how two bounds declaring `Item` are told apart
				if btp, isTP := base.(*types.TypeParam); isTP && i+1 < len(t.Path) {
					if tr := boundNamed(btp, seg.Name); tr != nil {
						next := t.Path[i+1].Name
						if !containsString(tr.AssocTypes, next) {
							c.errorf(t.Pos, "trait '%s' has no associated type '%s'", tr.Name, next)
							return types.TInvalid
						}
						base = &types.Assoc{Base: btp, Trait: tr, Name: next}
						i++
						continue
					}
				}
				base = c.projectAssoc(env, base, seg.Name, t.Pos)
				if types.IsInvalid(base) {
					return base
				}
			}
			if len(t.Args) > 0 {
				c.errorf(t.Pos, "'%s' cannot take type arguments", pathString(t.Path))
			}
			return base
		}
		sym, _ := c.lookupTypeName(env, t.Path)
		if sym == nil && c.pathReported {
			c.pathReported = false // the path already said what is wrong with it
			return types.TInvalid
		}
		if sym == nil {
			hint := ""
			var fix *source.Fix
			if len(t.Path) == 1 && env.module != nil {
				hint = c.suggestUnknown(env.module, t.Path[0].Name)
				fix = c.unknownFix(env.file, env.module, t.Path[0].Pos, t.Path[0].Name)
			}
			c.errorFix(t.Pos, fix, "unknown type '%s'%s", pathString(t.Path), hint)
			return types.TInvalid
		}
		if sym.Kind != SymType {
			c.errorf(t.Pos, "'%s' is not a type", pathString(t.Path))
			return types.TInvalid
		}
		if sym.TypeAlias != nil {
			var args []types.Type
			for _, a := range t.Args {
				args = append(args, c.resolveType(env, a))
			}
			u := c.resolveTypeAlias(sym, args, t.Pos)
			if c.index != nil && !types.IsInvalid(u) {
				c.index.Refs = append(c.index.Refs, Ref{Span: t.Path[len(t.Path)-1].Pos, Def: sym.Span, Kind: "type", Name: sym.Name, Type: u,
					Detail: aliasDetail(sym, u), Doc: sym.TypeAlias.decl.Doc, Shape: c.shapeOf(u)})
			}
			return u
		}
		if sym.Alias != nil {
			u := c.resolveAlias(sym)
			if c.index != nil {
				c.refType(t.Path[len(t.Path)-1].Pos, sym.Name, u, sym.Span)
				c.index.Refs[len(c.index.Refs)-1].Unfold = c.unfoldOf(sym)
				c.index.Refs[len(c.index.Refs)-1].Doc = sym.Alias.decl.Doc
			}
			if len(t.Args) > 0 {
				c.errorf(t.Pos, "'%s' is not generic: write it without '<...>'", sym.Name)
			}
			if !env.errorPos && !types.IsInvalid(u) {
				c.errorf(t.Pos, "'%s' names an error set; it can only appear after 'throws', in another error set, or as the type of an error's field (D45)", sym.Name)
				return types.TInvalid
			}
			return u
		}
		if c.index != nil {
			last := t.Path[len(t.Path)-1]
			def := sym.Span
			if v, ok := sym.Type.(*types.Struct); ok && !def.IsValid() {
				def = variantDefSpan(v)
			}
			c.refType(last.Pos, sym.Name, sym.Type, def)
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
			c.errorf(t.Pos, "'%s' is not generic: write it without '<...>'", sym.Name)
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
		ft := &types.Func{Ret: c.resolveType(env, t.Ret), Sendable: t.Sendable, C: t.C}
		for _, p := range t.Params {
			ft.Params = append(ft.Params, types.Param{Type: c.resolveType(env, p)})
		}
		ft.Effects = c.resolveEffects(env, t.Effects, true)
		if ft.C && (ft.Effects.Suspends || ft.Effects.Throws) {
			c.errorf(t.Pos, "a C function pointer cannot suspend or throw; C has no way to wait for it or to receive the error (D69)")
		}
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
		saved := env.errorPos
		env.errorPos = true
		var members []types.Type
		for _, m := range t.Members {
			members = append(members, c.resolveType(env, m))
		}
		env.errorPos = saved
		return types.MakeErrorUnion(members...)
	}
	c.errorf(t.Span(), "unsupported type syntax")
	return types.TInvalid
}

func (c *Checker) resolveEffects(env *typeEnv, e ast.Effects, mustDeclare bool) types.Effects {
	eff := types.Effects{Suspends: e.Suspends, Throws: e.Throws}
	if e.Throws && e.Error != nil {
		saved := env.errorPos
		env.errorPos = true
		eff.Error = c.resolveType(env, e.Error)
		env.errorPos = saved
		if _, isUnion := eff.Error.(*types.ErrorUnion); !isUnion {
			eff.Error = types.MakeErrorUnion(eff.Error)
		}
	} else if e.Throws && mustDeclare {
		if env.trait != nil && env.trait.ImplicitError {
			// the trait's implicit `Error`: whatever the impl throws
			eff.Error = types.MakeErrorUnion(&types.Assoc{Base: selfParamOf(env.trait), Trait: env.trait, Name: "Error"})
		} else {
			c.errorf(e.ThrowsSpan, "error type must be declared here: 'throws E' (D40 — effects are declared wherever dispatch is dynamic)")
		}
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
			c.errorf(span, "'%s' is not generic: write it without '<...>'", st.Name)
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
			c.errorf(span, "'%s' is not generic: write it without '<...>'", st.Name)
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
				s.TypeParams[i].Bounds = c.withSupers(s.TypeParams[i].Bounds, tr)
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
		c.checkFieldAttrs(f)
		env.errorPos = d.Error // an error's field may hold a cause: a union (D45)
		ft := c.resolveType(env, f.Type)
		env.errorPos = false
		if u, isUnion := ft.(*types.ErrorUnion); isUnion && !isAliasRef(f.Type) { // an alias checks its own members
			c.deferErrorCheck(u, f.Type.Span())
		}
		s.Fields = append(s.Fields, &types.Field{Name: f.Name.Name, Type: ft, Pub: f.Pub, Private: f.Private, Var: f.Var, Protected: f.Protected, HasDefault: f.Default != nil, Index: i})
	}
	if d.Init != nil {
		// a field without a default that the block assigns is the block's
		// to initialise: the constructor call does not take it (D28)
		for _, name := range initAssigned(d.Init) {
			for _, fld := range s.Fields {
				if fld.Name == name && !fld.HasDefault {
					fld.Init = true
				}
			}
		}
		// `init(value: T)`: its parameters join the constructor's (D73), so
		// one may not be named like a field the call can also pass
		for _, prm := range d.InitParams {
			for _, fld := range s.Fields {
				if fld.Name == prm.Name.Name && !fld.Init {
					c.errorf(prm.Name.Pos, "'init' parameter '%s' has the name of a field the constructor already takes; rename one of them (D73)", prm.Name.Name)
				}
			}
		}
	}
	if d.Variant != nil {
		vt := c.resolveType(env, d.Variant)
		parent, ok := vt.(*types.Sealed)
		if !ok {
			if !types.IsInvalid(vt) {
				c.errorf(d.Variant.Span(), "'%s' is not a sealed trait; 'struct X : T' declares variant membership (D23), trait conformance uses 'implement T for X'", vt)
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
	return c.containsInlineSeen(t, target, map[*types.Struct]bool{})
}

// containsInlineSeen is containsInline with the structs already entered,
// so a cycle that does not pass through target (B holds C holds B) ends;
// that cycle is reported when B itself is resolved.
func (c *Checker) containsInlineSeen(t types.Type, target *types.Struct, seen map[*types.Struct]bool) bool {
	switch t := t.(type) {
	case *types.Struct:
		if t == target || t.Template == target {
			return true
		}
		if seen[t] {
			return false
		}
		seen[t] = true
		for _, f := range t.Fields {
			if c.containsInlineSeen(f.Type, target, seen) {
				return true
			}
		}
	case *types.Sealed:
		tmpl := t
		if t.Template != nil {
			tmpl = t.Template
		}
		for _, v := range tmpl.Variants {
			if v == target || c.containsInlineSeen(v, target, seen) {
				return true
			}
		}
	case *types.Nullable:
		return c.containsInlineSeen(t.Elem, target, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if c.containsInlineSeen(e, target, seen) {
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
	self.Bounds = c.withSupers(nil, t) // Self has the supertraits' methods too
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
					mtps[i].Bounds = c.withSupers(mtps[i].Bounds, tr)
				}
			}
		}
		sig := c.signatureOf(&menv, m, true)
		t.Methods[m.Name.Name] = sig
		t.MethodList = append(t.MethodList, m.Name.Name)
		c.traitMethodTPs[t.Name+"."+m.Name.Name] = mtps
		if m.Static {
			if d.Sealed {
				c.errorf(m.Name.Pos, "a sealed trait cannot declare a static function; write a free function (D12)")
			}
			c.traitStatic[t.Name+"."+m.Name.Name] = true
		}
		if m.Body != nil || m.ExprBody != nil {
			tmpl := c.newTemplate(ctx.module, ctx.file, m, nil, ctx.tps)
			tmpl.Trait = t
			tmpl.TypeParams = mtps
			tmpl.Mangled = ctx.module.prefix() + "." + t.Name + "." + m.Name.Name
		}
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
		if p.Variadic {
			// `parts: string...` is a List<string> inside; only the last
			// parameter may be variadic, and an extern cannot be
			if i := len(sig.Params); i != len(d.Params)-1 {
				c.errorf(p.Name.Pos, "the variadic parameter '%s' must be the last one", p.Name.Name)
			}
			if d.Extern {
				c.errorf(p.Name.Pos, "extern \"C\" functions cannot be variadic")
			}
			if !types.IsInvalid(pt) {
				pt = &types.List{Elem: pt}
			}
		}
		if p.Lazy {
			c.checkLazyParam(env, p, pt, d)
		}
		sig.Params = append(sig.Params, types.Param{Name: p.Name.Name, Type: pt, HasDefault: p.Default != nil, Variadic: p.Variadic, Lazy: p.Lazy})
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
			c.errorf(t.Decl.Name.Pos, "type parameter '%s' shadows an outer type parameter; give it another name", tp.Name)
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
				t.TypeParams[i].Bounds = c.withSupers(t.TypeParams[i].Bounds, tr)
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
			c.checkExternType(p.Type, t.Decl.Name.Pos, t.Module.Std)
		}
		c.checkExternType(t.Sig.Ret, t.Decl.Name.Pos, t.Module.Std)
	}
	if t.Decl.ExportC {
		c.checkExportC(t)
	}

	if t.Decl.Override && (t.Impl == nil || t.Impl.Trait == nil) {
		c.errorf(t.Decl.Name.Pos, "'override' is only meaningful inside an implement block (D53)")
	}
	if t.Sig.Effects.Throws && t.Sig.Effects.Error == nil && !t.InferDone {
		t.InferError = types.MakeErrorUnion() // nil: nothing yet
	}
}

// checkExportC validates an `extern "C" fun` (D69): a module-level,
// non-generic function whose parameters and result C can read as they lie
// in memory, and which cannot throw (suspending is refused once calls are
// known, in suspend.go). Its name is a C symbol, so it is unique in the
// program.
func (c *Checker) checkExportC(t *FuncTemplate) {
	pos := t.Decl.Name.Pos
	if t.Owner != nil || t.Impl != nil || t.Trait != nil {
		c.errorf(pos, "only a module-level function can be 'extern \"C\"'; C calls it by name (D69)")
		return
	}
	if len(t.TypeParams) > 0 {
		c.errorf(pos, "an 'extern \"C\" fun' cannot be generic: C calls one function with one signature (D69)")
		return
	}
	bad := func(ty types.Type) {
		c.errorf(pos, "'%s' cannot cross into C; an 'extern \"C\" fun' takes and returns numbers, bool, raw pointers, extern structs and 'extern fun' pointers (D69)", ty)
	}
	for _, p := range t.Sig.Params {
		if !cLayout(p.Type) {
			bad(p.Type)
		}
	}
	if t.Sig.Ret != nil && !types.IsUnit(t.Sig.Ret) && !cLayout(t.Sig.Ret) {
		bad(t.Sig.Ret)
	}
	if t.Sig.Effects.Throws {
		c.errorf(pos, "an 'extern \"C\" fun' cannot throw: C has no way to receive the error; return a status code instead (D69)")
	}
	if prev, dup := c.exportedC[t.Name]; dup && prev != t.Decl {
		c.errorf(pos, "'extern \"C\" fun %s' is already defined at %s; a C symbol is unique in a program", t.Name, prev.Name.Pos)
		return
	}
	if c.exportedC == nil {
		c.exportedC = map[string]*ast.FunDecl{}
	}
	c.exportedC[t.Name] = t.Decl
}

func (c *Checker) checkExternType(t types.Type, span source.Span, std bool) {
	switch t := t.(type) {
	case *types.Basic:
		return
	case *types.Func:
		if !t.C {
			c.errorf(span, "a Veles function value cannot be handed to C; declare the callback 'extern \"C\" fun name(...)' and pass '&name' (D69)")
		}
		return
	case *types.Pointer:
		if !t.Raw {
			c.errorf(span, "extern \"C\" functions cannot take GC-managed pointers; use '*raw T' (D44/D50)")
		}
	case *types.Nullable:
		if _, ok := t.Elem.(*types.Basic); !ok {
			c.checkExternType(t.Elem, span, std)
		}
	case *types.Struct:
		if !t.Extern {
			c.errorf(span, "extern \"C\" functions can only pass 'extern struct' types by value")
		}
	case *types.List:
		// the runtime's own list layout: `List<u8>` is passed as a pointer to
		// it, for the standard library's byte I/O
		if !std || !types.Identical(t.Elem, types.TU8) {
			c.errorf(span, "type '%s' cannot cross the C ABI", t)
		}
	default:
		c.errorf(span, "type '%s' cannot cross the C ABI", t)
	}
}

// newImpl starts an Impl for an `impl` or `extend` block: its type
// parameters with their bounds, and the environment its types resolve in.
func (c *Checker) newImpl(m *Module, f *ast.File, d *ast.ImplDecl) (*Impl, *declCtx, *typeEnv) {
	ctx := &declCtx{module: m, file: f, decl: d, tps: map[string]*types.TypeParam{}}
	impl := &Impl{Module: m, File: f, Decl: d, Methods: map[string]*FuncTemplate{}}
	for i, tp := range d.TypeParams {
		p := &types.TypeParam{Name: tp.Name.Name, Index: i, Owner: "impl"}
		impl.TypeParams = append(impl.TypeParams, p)
		ctx.tps[tp.Name.Name] = p
	}
	env := c.envFor(ctx, nil)
	for i, tp := range d.TypeParams {
		for _, b := range tp.Bounds {
			if tr, ok := c.resolveType(env, b).(*types.Trait); ok {
				impl.TypeParams[i].Bounds = c.withSupers(impl.TypeParams[i].Bounds, tr)
			}
		}
	}
	return impl, ctx, env
}

// declareExtend records `extend<T> Type { methods }` (D23 addendum): inherent
// methods for a type the package declares. The built-in types — string,
// numbers, the collections, ranges, channels — are declared by std, so only
// the prelude may extend them; everyone else adds behaviour to foreign types
// through a trait.
func (c *Checker) declareExtend(m *Module, f *ast.File, d *ast.ImplDecl) {
	impl, ctx, env := c.newImpl(m, f, d)
	impl.Target = c.resolveType(env, d.Target)
	env.self = impl.Target
	if types.IsInvalid(impl.Target) {
		return
	}
	if en, ok := impl.Target.(*types.Enum); ok {
		c.errorf(d.Target.Span(), "cannot extend enum '%s': an enum is a set of values with no methods of its own; write functions that take one (D57)", en.Name)
		return
	}
	if !c.ownsType(m, impl.Target) {
		switch impl.Target.(type) {
		case *types.Struct, *types.Sealed:
			c.errorf(d.Target.Span(), "cannot extend '%s': it is declared outside this package; declare a trait and implement it for '%s' instead (D23)", impl.Target, impl.Target)
		case *types.Basic, *types.List, *types.Map, *types.Set, *types.Range, *types.Channel:
			c.errorf(d.Target.Span(), "cannot extend built-in type '%s' outside the standard library; declare a trait and implement it for '%s' instead (D23)", impl.Target, impl.Target)
		default:
			c.errorf(d.Target.Span(), "cannot extend '%s': only named types can be extended", impl.Target)
		}
		return
	}
	c.extends = append(c.extends, impl)
	for _, md := range d.Methods {
		name := md.Name.Name
		if prev, dup := impl.Methods[name]; dup {
			c.errorf(md.Name.Pos, "duplicate method '%s' in extend block; first declared at %s", name, prev.Decl.Name.Pos)
			continue
		}
		if st, ok := impl.Target.(*types.Struct); ok {
			if _, has := c.methods[templateOf(st)][name]; has {
				c.errorf(md.Name.Pos, "method '%s' is already declared in the body of '%s'", name, st.Name)
				continue
			}
		}
		if lookupBuiltinDoc(builtinFamily(impl.Target), name) != nil {
			c.errorf(md.Name.Pos, "method '%s' is a compiler built-in on '%s' and cannot be redeclared", name, impl.Target)
			continue
		}
		t := c.newTemplate(m, f, md, nil, ctx.tps)
		t.Impl = impl
		t.Mangled = m.prefix() + ".extend." + typeMangle(impl.Target) + "." + name
		impl.Methods[name] = t
	}
}

// ownsType reports whether m's package declares the head type of t; the
// built-in types belong to the standard library.
func (c *Checker) ownsType(m *Module, t types.Type) bool {
	var prefix string
	switch t := t.(type) {
	case *types.Struct:
		prefix = t.Module
	case *types.Sealed:
		if t.Module == "<prelude>" {
			return m.Std // Result and Option are built in; std extends them
		}
		prefix = t.Module
	case *types.Basic, *types.List, *types.Map, *types.Set, *types.Range, *types.Channel:
		return m.Std
	default:
		return false
	}
	for _, om := range c.pkg.Modules {
		if om.prefix() == prefix {
			return om.Pkg == m.Pkg && om.Std == m.Std
		}
	}
	return false
}

func (c *Checker) declareImpl(m *Module, f *ast.File, d *ast.ImplDecl) {
	impl, ctx, env := c.newImpl(m, f, d)
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
	if isSendableTrait(trait) {
		c.errorf(d.Trait.Span(), "Sendable is derived from a type's fields and cannot be implemented by hand (D35)")
		return
	}
	if isCLayoutTrait(trait) {
		c.errorf(d.Trait.Span(), "CLayout is derived from a type's shape (numbers, bool, raw pointers, extern structs) and cannot be implemented by hand (D69)")
		return
	}
	impl.Trait = trait
	impl.Target = c.resolveType(env, d.Target)
	env.self = impl.Target
	if types.IsInvalid(impl.Target) {
		return
	}
	if en, ok := impl.Target.(*types.Enum); ok && !d.Derived {
		c.errorf(d.Target.Span(), "cannot implement '%s' for enum '%s': an enum is a set of values with no methods of its own; it already compares, hashes, orders, prints and encodes by itself (D57, D58)", trait.Name, en.Name)
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
			if !types.ContainsTypeParam(bt) {
				// checked once every impl is declared: the impl that satisfies
				// the bound may come later in the source (or inside another
				// struct's body)
				b, bound := b, bound
				c.pendingAssocChecks = append(c.pendingAssocChecks, func() {
					if c.findImplFor(bt, bound) == nil {
						c.errorf(b.Type.Span(), "'%s' does not implement '%s', required by 'type %s: %s' in trait '%s'", bt, bound.Name, b.Name.Name, bound.Name, trait.Name)
					}
				})
			}
		}
	}
	for _, name := range trait.AssocTypes {
		if _, ok := impl.AssocTypes[name]; !ok {
			if name == "Error" && trait.ImplicitError {
				impl.ImplicitError = true // defined by what the methods throw (resolveAssoc)
				continue
			}
			if t := c.assocFromMethods(env, d, trait, name); t != nil {
				impl.AssocTypes[name] = t // `fun plus(other: Duration): Duration` binds Rhs and Out (D71)
				continue
			}
			c.errorf(d.Pos, "implement of '%s' for '%s' must bind associated type '%s': 'type %s = ...'", trait.Name, impl.Target, name, name)
		}
	}
	env.implAssoc = impl.AssocTypes
	c.impls[trait] = append(c.impls[trait], impl)
	if !d.Derived {
		c.lintInlinableImpl(m, f, d, impl)
	}
	// D58: methods written for a supertrait are declared with that super's
	// impl; the supers the type lacks are derived once every impl exists
	routed := c.routeSuperMethods(d, trait)
	for _, md := range d.Methods {
		c.declareImplMethod(m, f, d, impl, trait, ctx, md)
	}
	if c.derivable(trait) != "" || len(trait.Supers) > 0 {
		c.pendingDerives = append(c.pendingDerives, func() {
			c.deriveMissing(d, impl, trait)
			for _, md := range d.Methods {
				if _, done := impl.Methods[md.Name.Name]; !done {
					c.declareImplMethod(m, f, d, impl, trait, ctx, md)
				}
			}
			c.checkImplComplete(d, impl, trait)
			if len(trait.Supers) > 0 {
				c.deriveSupers(superImplReq{module: m, file: f, decl: d, impl: impl, trait: trait, methods: routed})
			}
		})
		return
	}
	c.checkImplComplete(d, impl, trait)
}

// declareImplMethod declares one method of a trait impl.
func (c *Checker) declareImplMethod(m *Module, f *ast.File, d *ast.ImplDecl, impl *Impl, trait *types.Trait, ctx *declCtx, md *ast.FunDecl) {
	sig, declared := trait.Methods[md.Name.Name]
	if !declared {
		names := make([]string, 0, len(trait.Methods))
		for n := range trait.Methods {
			names = append(names, n)
		}
		hint := "; a method of the type's own belongs in its body or an 'extend' block"
		if hit := didYouMean(md.Name.Name, names); hit != "" {
			hint = "; did you mean '" + hit + "'?"
		}
		c.errorf(md.Name.Pos, "trait '%s' has no method '%s'%s", trait.Name, md.Name.Name, hint)
		return
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

// checkImplComplete reports the trait methods an impl neither writes nor
// derives.
// checkSealedVariants reports, at its declaration, a variant with no
// `implement` of its sealed trait while the trait has methods without a
// default (D12): a call on the sealed value dispatches to every variant, so
// each needs the body. A variant with an implement that lacks a method is
// checkImplComplete's to report.
func (c *Checker) checkSealedVariants() {
	for _, s := range c.sealeds {
		if s.Template != nil || len(s.TypeParams) > 0 || s.Trait == nil {
			continue
		}
		var required []string
		for _, name := range s.Trait.MethodList {
			if c.traitDefault(s.Trait, name) == nil {
				required = append(required, name)
			}
		}
		if len(required) == 0 {
			continue
		}
		for _, v := range s.Variants {
			sd, ok := v.Decl.(*ast.StructDecl)
			if !ok || c.findImplFor(v, s.Trait) != nil {
				continue
			}
			hint := fmt.Sprintf("; add 'implement %s { %s }' to its body", s.Name, traitMethodText(s.Trait, required[0]))
			for _, md := range sd.Methods {
				if md.Name.Name == required[0] {
					// the method is there, one level too far out
					hint = fmt.Sprintf("; '%s' is declared in its body, outside any implement: move it inside 'implement %s { }'", md.Name.Name, s.Name)
				}
			}
			// the block, with a stub per method, added to the variant's body —
			// scaffolding, as for an implement missing methods
			block := "implement " + s.Name + " {"
			for _, name := range required {
				block += "\n  " + traitMethodText(s.Trait, name) + " = panic(\"'" + name + "' is not written yet\")"
			}
			fix := fixAddMembers(fmt.Sprintf("Implement '%s' in '%s'", s.Name, v.Name), sd.Pos, []string{block + "\n}"})
			if fix != nil {
				fix.Guess = true
			}
			c.errorFix(sd.Name.Pos, fix, "variant '%s' of '%s' does not implement '%s'%s", v.Name, s.Name, strings.Join(required, "', '"), hint)
		}
	}
}

func (c *Checker) checkImplComplete(d *ast.ImplDecl, impl *Impl, trait *types.Trait) {
	if c.deriveFailed[impl] {
		return // the reason was reported
	}
	var missing, stubs []string
	for _, name := range trait.MethodList {
		if _, ok := impl.Methods[name]; !ok && c.traitDefault(trait, name) == nil {
			missing = append(missing, name)
			if sig := traitMethodText(trait, name); sig != "" {
				stubs = append(stubs, sig+" = panic(\"'"+name+"' is not written yet\")")
			}
		}
	}
	// one quick fix, on every one of the errors: add all the missing methods,
	// each a stub that panics until it is written — scaffolding, so the
	// editor offers it and `check --fix` does not apply it
	var fix *source.Fix
	if len(stubs) == len(missing) && !d.Braceless && !d.Derived {
		fix = fixAddMembers(fmt.Sprintf("Add the missing methods of '%s'", trait.Name), d.Pos, stubs)
		if fix != nil {
			fix.Guess = true
		}
	}
	for _, name := range missing {
		msg := fmt.Sprintf("implement of '%s' for '%s' is missing method '%s'", trait.Name, impl.Target, name)
		if sig := traitMethodText(trait, name); sig != "" {
			msg += "; add: " + sig
		}
		c.errorFix(d.Pos, fix, "%s", msg)
	}
}

// traitMethodText is a required trait method's declaration as written in
// the trait, on one line — what an implement has to add.
func traitMethodText(trait *types.Trait, name string) string {
	td, ok := trait.Decl.(*ast.TraitDecl)
	if !ok {
		return ""
	}
	for _, md := range td.Methods {
		if md.Name.Name == name && md.Body == nil && md.ExprBody == nil {
			text := strings.Join(strings.Fields(spanText(md.Pos)), " ")
			return strings.NewReplacer("( ", "(", " )", ")", ", )", ")", ",)", ")").Replace(text)
		}
	}
	return ""
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
	if isStatic := c.traitStatic[trait.Name+"."+md.Name.Name]; isStatic != md.Static {
		if isStatic {
			c.errorf(md.Name.Pos, "'%s' is a static function in trait '%s'; declare it 'static fun'", md.Name.Name, trait.Name)
		} else {
			c.errorf(md.Name.Pos, "'%s' is a method in trait '%s'; it cannot be 'static'", md.Name.Name, trait.Name)
		}
	}
	subst := map[*types.TypeParam]types.Type{selfParamOf(trait): impl.Target}
	if mtps := c.traitMethodTPs[trait.Name+"."+md.Name.Name]; len(mtps) == len(t.TypeParams) {
		for i, tp := range mtps {
			subst[tp] = t.TypeParams[i]
		}
	} else {
		// the trait's own type parameters have no counterpart to compare
		// against: its parameter and result types would all "differ"
		c.errorf(md.Name.Pos, "method '%s' must declare the same type parameters as in trait '%s'", md.Name.Name, trait.Name)
		return
	}
	for i, p := range t.Sig.Params {
		want := c.hooks.Subst(traitSig.Params[i].Type, subst)
		if p.Name != traitSig.Params[i].Name {
			c.errorf(md.Params[i].Name.Pos, "parameter must be named '%s' as in trait '%s' (D28: impls inherit the trait's parameter names)", traitSig.Params[i].Name, trait.Name)
		}
		if !types.Identical(p.Type, want) && !types.IsInvalid(p.Type) {
			c.errorf(md.Params[i].Pos, "parameter '%s' has type '%s' but trait '%s' declares '%s'", p.Name, p.Type, trait.Name, want)
		} else if p.Variadic != traitSig.Params[i].Variadic {
			c.errorf(md.Params[i].Pos, "parameter '%s' must be variadic exactly as in trait '%s'", p.Name, trait.Name)
		}
	}
	wantRet := c.hooks.Subst(traitSig.Ret, subst)
	if t.InferRet {
		// `fun encode(to: Encoder) = try to.writeI64(self)`: an impl method
		// with an expression body and no return type takes the trait's
		t.InferRet = false
		t.Sig.Ret = wantRet
	}
	if !types.Identical(t.Sig.Ret, wantRet) {
		c.errorf(md.Name.Pos, "method '%s' returns '%s' but trait '%s' declares '%s'", md.Name.Name, t.Sig.Ret, trait.Name, wantRet)
	}
	if t.Sig.Effects.Throws && !traitSig.Effects.Throws {
		c.errorf(md.Name.Pos, "method '%s' throws but trait '%s' declares it as non-throwing (D40)", md.Name.Name, trait.Name)
	}
	if t.Sig.Effects.Throws && traitSig.Effects.Throws && t.Sig.Effects.Error == nil && !impl.ImplicitError {
		// inherit the declared error; with an implicit `Error` the method's
		// own error is inferred from its body (D45) and defines the impl's
		t.Sig.Effects.Error = c.hooks.Subst(traitSig.Effects.Error, subst)
	}
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
	// extend blocks: a method name may be provided once per receiver type;
	// two blocks overlap when either target matches the other
	for i, a := range c.extends {
		for _, b := range c.extends[:i] {
			if !unify(a.Target, b.Target, map[*types.TypeParam]types.Type{}) && !unify(b.Target, a.Target, map[*types.TypeParam]types.Type{}) {
				continue
			}
			for name, t := range a.Methods {
				if other, dup := b.Methods[name]; dup {
					c.errorf(t.Decl.Name.Pos, "method '%s' is already provided for '%s' by the extend block at %s", name, b.Target, other.Decl.Name.Pos)
				}
			}
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
	c.checkTypeArgBounds(tmpl.Name, tmpl.TypeParams, args, span)
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
		nf.Type = c.hooks.Subst(f.Type, m)
		inst.Fields = append(inst.Fields, &nf)
	}
	if tmpl.Sealed != nil {
		inst.Sealed = c.instantiateSealed(tmpl.Sealed, args, span)
	}
	return inst
}

func (c *Checker) instantiateSealed(tmpl *types.Sealed, args []types.Type, span source.Span) *types.Sealed {
	c.checkTypeArgBounds(tmpl.Name, tmpl.TypeParams, args, span)
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
	case *types.ErrorUnion:
		// `throws E | Fail` against what a function actually throws: the
		// concrete members named in the pattern are accounted for, E is
		// bound to the rest (Never when nothing is left), so a handler that
		// throws `IoError | Fail` gives E = IoError and one that throws
		// only `Fail` gives E = Never
		var param *types.TypeParam
		var fixed []types.Type
		for _, mem := range p.Members {
			if tp, ok := mem.(*types.TypeParam); ok {
				if param != nil {
					return types.Identical(pattern, concrete)
				}
				param = tp
			} else {
				fixed = append(fixed, mem)
			}
		}
		if param == nil {
			return types.Identical(pattern, concrete)
		}
		var rest []types.Type
		for _, cm := range types.UnionMembers(concrete) {
			if types.IsNever(cm) {
				continue
			}
			covered := false
			for _, fm := range fixed {
				if types.Identical(fm, cm) {
					covered = true
				}
			}
			if !covered {
				rest = append(rest, cm)
			}
		}
		var restT types.Type = types.TNever
		if len(rest) > 0 {
			restT = types.MakeErrorUnion(rest...)
		}
		return unify(param, restT, m)
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
		if p.Effects.Throws && p.Effects.Error != nil && types.ContainsTypeParam(p.Effects.Error) {
			// `throws E`: E is what the concrete function throws — nothing
			// (Never) when it does not throw, which makes the instance an
			// ordinary function (Subst drops `throws Never`)
			var concreteErr types.Type = types.TNever
			if cc.Effects.Throws && cc.Effects.Error != nil {
				concreteErr = cc.Effects.Error
			}
			if !unify(p.Effects.Error, concreteErr, m) {
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
	if c.index != nil {
		c.index.Refs = c.index.Refs[:c.collectRefs] // keep signature refs recorded during collection
	}
	c.prog = &Program{Release: c.release}
	if c.pkg != nil {
		c.prog.Root = c.pkg.Root
	}
	c.prog.ResultType = func(ok, err types.Type) types.Type { return c.ResultType(ok, err) }
	c.queue = nil
	c.instances = map[string]*Func{}
	c.funcs = nil
	c.tupleCmp, c.enumFns = nil, nil // synthesized per round, like every other function
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
	// every value at its declaration, now that its type is known
	if c.index != nil {
		for _, g := range globals {
			c.refGlobal(g.Span, g.Display[strings.LastIndex(g.Display, ".")+1:], g)
		}
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
	c.drainQueue()
	c.resolveAllCustom()
	c.prog.TestMode = c.testMode
	c.prog.PanicType = c.panicType()
	for _, t := range c.tests {
		if inst, ok := t.Instances[""]; ok {
			c.prog.Tests = append(c.prog.Tests, inst)
		}
	}
	// tests outside any suite first, so the report does not print one
	// between two suites' groups (D78); otherwise declaration order
	sort.SliceStable(c.prog.Tests, func(i, j int) bool {
		return len(c.prog.Tests[i].Suite) == 0 && len(c.prog.Tests[j].Suite) > 0
	})
	// entry point: a package is a program when its root module has `main`
	entry := c.pkg.Entry
	if !c.testMode && (c.pkg.NeedMain || entry != nil) {
		var sym *Symbol
		if entry != nil {
			sym = entry.Scope.LookupLocal("main")
		}
		if sym != nil && sym.Kind == SymFunc {
			t := sym.Func
			if len(t.Sig.Params) != 0 || !types.IsUnit(t.Sig.Ret) {
				c.errorf(t.Decl.Name.Pos, "'main' must take no parameters and return nothing")
			}
			if inst, ok := t.Instances[""]; ok {
				c.prog.Main = inst
				if eff := inst.Sig.Effects; eff.Throws && eff.Error != nil && !types.IsNever(eff.Error) {
					c.prog.MainReport = c.synthReporter(eff.Error, t)
					c.drainQueue() // the message() instances it calls
				}
			}
		} else if c.pkg.NeedMain {
			c.errorf(source.Span{}, "%s", c.noMainMessage())
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
	c.lintNeedlessThrows()
	c.lintUnusedNames()
	c.finishImportRefs()
	c.prog.Funcs = c.funcs
	c.prog.Structs = nil
	for _, s := range c.structs {
		if len(s.TypeParams) == 0 {
			c.prog.Structs = append(c.prog.Structs, s)
		}
		c.prog.Structs = append(c.prog.Structs, sortedStructInstances(s)...)
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
	sig := c.hooks.Subst(t.Sig, m).(*types.Func)
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
	fn := &Func{Name: mangleName(name), Display: t.Name, Suite: t.Suite, Sig: sig, Extern: t.Extern, Span: t.Decl.Name.Pos}
	if _, ok := t.Attrs["inline"]; ok {
		fn.Inline = 1
	}
	if _, ok := t.Attrs["noinline"]; ok {
		fn.Inline = -1
	}
	if a, ok := t.Attrs["caller_location"]; ok {
		fn.CallerLoc = true
		switch {
		case t.Module == nil || !t.Module.Std:
			c.errorf(a.Pos, "'@caller_location' is reserved for the standard library for now (D88)")
		case t.Impl != nil && t.Impl.Trait != nil, t.Extern:
			c.errorf(a.Pos, "'@caller_location' applies to plain functions and inherent methods, not to trait implementations or extern functions (D88)")
		}
	}
	if t.Extern {
		fn.Name = t.Mangled // the C symbol
		// std's externs are the runtime's own entry points; any other is
		// foreign code that may block (D66)
		fn.Foreign = t.Module == nil || !t.Module.Std
	}
	if t.Decl != nil && t.Decl.ExportC {
		fn.ExportC = t.Name // the wrapper C calls; fn.Name stays the Veles body's
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
			if name == "Error" && impl.ImplicitError {
				return c.hooks.Subst(c.implErrorType(impl), m)
			}
			return nil
		}
		return c.hooks.Subst(bt, m)
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
		var hits []*types.Trait
		for _, bound := range tp.Bounds {
			if containsString(bound.AssocTypes, name) {
				hits = append(hits, bound)
			}
		}
		switch len(hits) {
		case 0:
			c.errorf(span, "type parameter '%s' has no bound declaring an associated type '%s'", tp.Name, name)
			return types.TInvalid
		case 1:
			return &types.Assoc{Base: tp, Trait: hits[0], Name: name}
		}
		// two bounds declare it: which one is meant is the writer's to say,
		// through the trait (D84) — the first was taken silently before
		c.errorf(span, "'%s.%s' is ambiguous: bounds '%s' and '%s' both declare '%s'; name the one meant: '%s.%s.%s' (D84)",
			tp.Name, name, hits[0].Name, hits[1].Name, name, tp.Name, hits[0].Name, name)
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
		c.errorf(span, "cannot resolve '%s.%s'", base, name)
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
				c.errorf(span, "'%s.%s' is ambiguous: declared by traits '%s' and '%s'", base, name, foundTrait.Name, tr.Name)
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
	if reason := c.unhashable(t); reason != "" {
		c.errorf(span, "'%s' cannot be a map key or set element: %s", t, reason)
		return false
	}
	return true
}

// unhashable explains why t cannot be hashed, or returns "" when it can:
// structurally (every part has structural equality) or through the
// prelude's Hashable. A custom Equatable without a matching Hashable is
// refused, because equal values would not hash alike.
func (c *Checker) unhashable(t types.Type) string {
	return c.unhashableIn(t, map[types.Type]bool{})
}

// unhashableIn is unhashable with the structs and sealed types already on
// the path in seen: a type that contains itself by value (a D31 error,
// reported by the size check) must not recurse forever here.
func (c *Checker) unhashableIn(t types.Type, seen map[types.Type]bool) string {
	const structural = "it has no structural equality"
	switch t := t.(type) {
	case *types.Basic:
		if t.Kind == types.Unit || t.Kind == types.Never || t.Kind == types.Invalid {
			return structural
		}
		return ""
	case *types.Pointer, *types.Enum:
		return ""
	case *types.Nullable:
		return c.unhashableIn(t.Elem, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if r := c.unhashableIn(e, seen); r != "" {
				return r
			}
		}
		return ""
	case *types.Struct, *types.Sealed:
		if seen[t] {
			return ""
		}
		seen[t] = true
		hasHash := c.implementsPrelude(t, "Hashable")
		if c.implementsPrelude(t, "Equatable") && !hasHash {
			return fmt.Sprintf("'%s' implements Equatable but not Hashable, so equal values might hash differently", t)
		}
		if hasHash {
			return ""
		}
		if st, ok := t.(*types.Struct); ok {
			for _, f := range st.Fields {
				if r := c.unhashableIn(f.Type, seen); r != "" {
					return r
				}
			}
			return ""
		}
		for _, v := range t.(*types.Sealed).Variants {
			if r := c.unhashableIn(v, seen); r != "" {
				return r
			}
		}
		return ""
	case *types.List:
		if t.Mutable {
			return "a MutableList can change after it is stored; use an immutable List"
		}
		return c.unhashableIn(t.Elem, seen)
	case *types.Set:
		if t.Mutable {
			return "a MutableSet can change after it is stored; use an immutable Set"
		}
		return c.unhashableIn(t.Elem, seen)
	case *types.Map:
		if t.Mutable {
			return "a MutableMap can change after it is stored; use an immutable Map"
		}
		if r := c.unhashableIn(t.Key, seen); r != "" {
			return r
		}
		return c.unhashableIn(t.Value, seen)
	}
	return structural
}

// checkErrorType reports a type in error position that is not an error:
// every member must implement Error (D4) — declared with `error Name { }`
// or an explicit `impl Error for` — or be a type parameter bounded by it.
func (c *Checker) checkErrorType(errT types.Type, span source.Span) {
	errTrait := c.traitNamed("Error")
	if errTrait == nil {
		return
	}
	var members []types.Type
	if u, ok := errT.(*types.ErrorUnion); ok {
		members = u.Members
	} else {
		members = []types.Type{errT}
	}
	for _, m := range members {
		if types.IsInvalid(m) || types.IsNever(m) || c.findImplFor(m, errTrait) != nil {
			continue
		}
		switch t := m.(type) {
		case *types.TypeParam:
			ok := false
			for _, b := range t.Bounds {
				if b == errTrait {
					ok = true
				}
			}
			if ok {
				continue
			}
			c.errorf(span, "'%s' cannot be an error without the bound '%s: Error' (D4)", t.Name, t.Name)
		case *types.Struct:
			c.errorf(span, "'%s' is not an error: declare it with 'error %s { ... }' instead of 'struct', or 'implement Error for %s' (D4)", t.Name, t.Name, t.Name)
		default:
			c.errorf(span, "'%s' cannot be an error: only types declared with 'error' (or an 'implement Error for' them) can be thrown (D4)", m)
		}
	}
}

// synthReporter builds the function the entry point calls to render an
// error escaping `main() throws`: `fun (e: E): string = e.message()`,
// where E is main's (possibly inferred) error type (D4).
func (c *Checker) synthReporter(errT types.Type, t *FuncTemplate) *Func {
	pos := t.Decl.Name.Pos
	fn := &Func{Name: "veles.main.report", Display: "main.report", Span: pos,
		Sig: &types.Func{Params: []types.Param{{Name: "e", Type: errT}}, Ret: types.TString}}
	env := &typeEnv{module: t.Module, file: t.File, tps: map[string]*types.TypeParam{}}
	f := c.newFnCtx(fn, t.Module, t.File, env, nil)
	f.retType = types.TString
	v := f.newVar("$e", errT, false, source.Span{})
	fn.Params = []*Var{v}
	f.declareLocal("$e", v, source.Span{})
	call := &ast.CallExpr{Fun: &ast.MemberExpr{X: &ast.NameExpr{Name: "$e", Pos: pos}, Name: ast.Ident{Name: "message", Pos: pos}, Pos: pos}, Pos: pos}
	x := f.checkExprTo(call, types.TString)
	fn.Body = &Block{Stmts: []Stmt{&Return{Value: x}}, Type: types.TNever}
	fn.checked = true
	c.funcs = append(c.funcs, fn)
	return fn
}

// drainQueue checks the body of every instantiation queued so far,
// including those queued while checking.
func (c *Checker) drainQueue() {
	for len(c.queue) > 0 {
		fn := c.queue[0]
		c.queue = c.queue[1:]
		if fn.checked {
			continue
		}
		c.timeBody(fn, func() { c.checkBody(fn) })
	}
}

type pendingErrorCheck struct {
	t    types.Type
	span source.Span
}

// deferErrorCheck runs checkErrorType once collection is complete; during
// collection the impls that make a type an error may not exist yet.
func (c *Checker) deferErrorCheck(t types.Type, span source.Span) {
	if c.collected {
		c.checkErrorType(t, span)
		return
	}
	c.pendingErrorChecks = append(c.pendingErrorChecks, pendingErrorCheck{t, span})
}

// isAliasRef reports whether a type expression is a bare name (which, if
// it resolved to a union, must be an error-set alias).
func isAliasRef(t ast.Type) bool {
	_, ok := t.(*ast.NamedType)
	return ok
}

// noMainMessage explains a missing entry point in terms of the package
// layout: `main` lives in the root module, and a module is not a program.
func (c *Checker) noMainMessage() string {
	root := c.pkg.Root
	if c.pkg.Script != "" {
		return fmt.Sprintf("script %s has no 'fun main()'", c.pkg.Script)
	}
	if c.pkg.Given != nil && c.pkg.Given != c.pkg.Entry {
		return fmt.Sprintf("'%s' is a module of the package at %s, not a program; a package runs from 'fun main()' in its root module, and this root has none (build a program from %s, or use the module as a library)",
			c.pkg.GivenDir, root, root)
	}
	if c.pkg.Manifest == nil {
		return fmt.Sprintf("no 'fun main()' in package %s (no veles.toml above it, so this directory is its own package root); if it is a module of a larger package, add a veles.toml at that package's root and build from there", root)
	}
	return fmt.Sprintf("package %s has no 'fun main()' in its root module; it can be used as a library but not run", root)
}

// implErrorType is the implicit `Error` of an impl whose trait declares a
// method with a bare `throws`: the union of everything the impl's throwing
// methods declare or, when they too say only `throws`, infer from their
// bodies (D45). Nothing thrown is Never, so a fallible-by-signature method
// whose impl cannot fail costs its callers nothing.
func (c *Checker) implErrorType(impl *Impl) types.Type {
	var members []types.Type
	for _, name := range impl.Trait.MethodList {
		t := impl.Methods[name]
		if t == nil {
			continue
		}
		c.resolveSignature(t)
		if t.Sig == nil || !t.Sig.Effects.Throws {
			continue
		}
		e := t.Sig.Effects.Error
		if e == nil {
			e = t.InferError
		}
		if e != nil {
			members = append(members, e)
		}
	}
	if u := types.MakeErrorUnion(members...); u != nil {
		return u
	}
	return types.TNever
}

// initAssigned lists the fields an `init { }` block assigns as `this.f = ...`
// (or `this.f op= ...`), anywhere in it, in first-assignment order.
func initAssigned(b *ast.Block) []string {
	var names []string
	seen := map[string]bool{}
	walkAST(b, func(n any) bool {
		if s, ok := n.(*ast.AssignStmt); ok {
			if m, ok := s.Target.(*ast.MemberExpr); ok && !m.Safe {
				if _, isSelf := m.X.(*ast.SelfExpr); isSelf && !seen[m.Name.Name] {
					seen[m.Name.Name] = true
					names = append(names, m.Name.Name)
				}
			}
		}
		return true
	})
	return names
}

// assocFromMethods infers an associated type an implement does not bind
// from a method it writes: when the trait declares the type as a
// parameter's or the result's type outright (`fun plus(other: Rhs): Out`),
// the implement's own spelling at that position is the binding. Nil when no
// written method pins it down.
func (c *Checker) assocFromMethods(env *typeEnv, d *ast.ImplDecl, trait *types.Trait, name string) types.Type {
	names := func(t types.Type) bool {
		a, ok := t.(*types.Assoc)
		return ok && a.Name == name && a.Trait == trait
	}
	for _, md := range d.Methods {
		sig := trait.Methods[md.Name.Name]
		if sig == nil {
			continue
		}
		for i, p := range sig.Params {
			if i < len(md.Params) && md.Params[i].Type != nil && names(p.Type) {
				return c.resolveType(env, md.Params[i].Type)
			}
		}
		if md.Ret != nil && names(sig.Ret) {
			return c.resolveType(env, md.Ret)
		}
	}
	return nil
}

// checkTypeArgBounds refuses a generic struct or sealed type instantiated
// with an argument that does not meet its parameter's bounds —
// `Box<P>` for `struct Box<T: Comparable>` — at the place that wrote or
// implied it. An argument that is itself a type parameter is checked
// where the enclosing generic is instantiated, and an instantiation with
// no place (a substitution inside the checker) was checked at its source,
// and so was one inside the standard library, which only passes on the
// arguments its caller chose (reporting it there would say it twice).
// During collection the impls may not exist yet, so the check waits.
func (c *Checker) checkTypeArgBounds(name string, tps []*types.TypeParam, args []types.Type, span source.Span) {
	if !span.IsValid() || len(tps) != len(args) || c.stdSpan(span) {
		return
	}
	check := func() {
		f := &fnCtx{c: c}
		for i, tp := range tps {
			a := args[i]
			if types.ContainsTypeParam(a) || types.IsInvalid(a) {
				continue
			}
			for _, bound := range tp.Bounds {
				if !f.implements(a, bound) {
					c.errorFix(span, c.implementFix(a, bound), "type '%s' does not implement trait '%s' required by parameter '%s' of '%s'%s", a, bound.Name, tp.Name, name, implementHint(a, bound))
				}
			}
		}
	}
	if !c.collected {
		c.pendingBoundChecks = append(c.pendingBoundChecks, check)
		return
	}
	check()
}

// stdSpan reports whether a span is in the standard library's source.
func (c *Checker) stdSpan(span source.Span) bool {
	if c.stdFiles == nil {
		c.stdFiles = map[*source.File]bool{}
		for _, m := range c.pkg.Modules {
			if m.Std {
				for _, f := range m.Files {
					c.stdFiles[f.Source] = true
				}
			}
		}
	}
	return c.stdFiles[span.File]
}

// boundNamed is the bound of tp called name, or nil.
func boundNamed(tp *types.TypeParam, name string) *types.Trait {
	for _, b := range tp.Bounds {
		if b.Name == name {
			return b
		}
	}
	return nil
}
