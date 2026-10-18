package sema

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Lint rules written in Veles (D177). A package turns a rule of a dependency
// on in its manifest, `[lint] acme-lints.no_sleep = "warn"`; the dependency
// lists its rules in a `public const fun rules(): List<lint.Rule>`
// (std/lint). After the package is checked, the compiler's evaluator (D113)
// runs each rule: `create` makes a Visitor for a file, and every syntax node
// the visitor has a callback for is handed to it, as the Veles syntax tree
// (selfhost/ast, written out in std/lint), with a Context to report through.
// A report becomes a diagnostic at the level the manifest gave, with the
// rule's fix.
//
// The tree a rule sees is built from the Go front end's, node by node and only
// when a callback asks for it: the types are those std/lint declares, the
// fields are matched by name, and a shared node is built once.

// libraryLintSteps is the step budget of one rule on one file; a package
// raises it with `--const-steps`.
const libraryLintSteps = 1_000_000_000

// ruleGoName maps a Go syntax node to the Veles struct that shows it, where
// the two are not spelled the same.
var ruleGoName = map[string]string{"File": "SourceFile", "SelfExpr": "ThisExpr"}

// ruleHost converts the Go syntax tree to evaluator values for one file.
type ruleHost struct {
	c      *Checker
	lint   *Module
	file   *source.File
	files  map[*source.File]ConstVal
	back   map[*CStruct]*source.File
	memo   map[ruleKey]ConstVal
	failed string // the first syntax a rule cannot be shown
	where  string // the struct field being built, for the message
}

type ruleKey struct {
	p uintptr
	t types.Type
}

func (h *ruleHost) structT(name string) *types.Struct {
	if sym := h.lint.Scope.symbols[name]; sym != nil {
		if s, ok := sym.Type.(*types.Struct); ok {
			return s
		}
	}
	panic("std/lint declares no struct " + name)
}

func ruleField(t *types.Struct, name string) int {
	for i, f := range t.Fields {
		if f.Name == name {
			return i
		}
	}
	panic("std/lint: " + t.Name + " has no field " + name)
}

func i64Val(n int) ConstVal { return &CInt{types.TI64, big.NewInt(int64(n))} }

// fileVal is a source file as a rule sees it: path, text and line starts.
func (h *ruleHost) fileVal(f *source.File) ConstVal {
	if v, ok := h.files[f]; ok {
		return v
	}
	t := h.structT("File")
	starts := []ConstVal{i64Val(0)}
	for i := 0; i < len(f.Content); i++ {
		if f.Content[i] == '\n' {
			starts = append(starts, i64Val(i+1))
		}
	}
	v := &CStruct{T: t, Fields: make([]ConstVal, len(t.Fields))}
	v.Fields[ruleField(t, "path")] = &CString{f.Path}
	v.Fields[ruleField(t, "content")] = &CString{f.Content}
	v.Fields[ruleField(t, "lineStarts")] = &CList{T: t.Fields[ruleField(t, "lineStarts")].Type.(*types.List), Elems: starts}
	h.files[f] = v
	h.back[v] = f
	return v
}

func (h *ruleHost) spanVal(s source.Span) ConstVal {
	t := h.structT("Span")
	v := &CStruct{T: t, Fields: make([]ConstVal, len(t.Fields))}
	fileT := t.Fields[ruleField(t, "file")].Type.(*types.Nullable)
	if s.File == nil {
		v.Fields[ruleField(t, "file")] = &CNull{fileT}
	} else {
		v.Fields[ruleField(t, "file")] = &CSome{fileT, h.fileVal(s.File)}
	}
	v.Fields[ruleField(t, "start")] = i64Val(s.Start)
	v.Fields[ruleField(t, "end")] = i64Val(s.End)
	return v
}

// sourceSpan reads a Span value back.
func (h *ruleHost) sourceSpan(v ConstVal) source.Span {
	s, ok := v.(*CStruct)
	if !ok {
		return source.Span{File: h.file}
	}
	t := s.T
	out := source.Span{File: h.file}
	if some, ok := s.Fields[ruleField(t, "file")].(*CSome); ok {
		if fs, ok := some.V.(*CStruct); ok && h.back[fs] != nil {
			out.File = h.back[fs]
		}
	}
	out.Start = int(s.Fields[ruleField(t, "start")].(*CInt).V.Int64())
	out.End = int(s.Fields[ruleField(t, "end")].(*CInt).V.Int64())
	return out
}

// zero is the value of a field the Go tree has no counterpart for.
func (h *ruleHost) zero(t types.Type) ConstVal {
	switch t := t.(type) {
	case *types.Basic:
		switch {
		case t == types.TString:
			return &CString{}
		case t == types.TBool:
			return &CBool{}
		case types.IsFloat(t):
			return &CFloat{t, 0}
		}
		return &CInt{t, new(big.Int)}
	case *types.Enum:
		return &CInt{t, new(big.Int)}
	case *types.Nullable:
		return &CNull{t}
	case *types.List:
		return &CList{T: t}
	case *types.Struct:
		if t.Name == "Span" {
			return h.spanVal(source.Span{})
		}
		v := &CStruct{T: t, Fields: make([]ConstVal, len(t.Fields))}
		for i, f := range t.Fields {
			v.Fields[i] = h.zero(f.Type)
		}
		return v
	}
	h.fail(fmt.Sprintf("a value of type '%s' (field %s)", t, h.where))
	return &CUnit{}
}

func (h *ruleHost) fail(what string) {
	if h.failed == "" {
		h.failed = what
	}
}

func isNilValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	}
	return false
}

func derefAll(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

// ruleFieldAlias names the Go field of a Veles field that is spelled around a
// keyword or reads as a question (selfhost/ast: `typ`, `isPrivate`, `callee`).
var ruleFieldAlias = map[string]string{
	"typ": "Type", "callee": "Fun", "defaultValue": "Default", "suspending": "Suspends",
	"throwing": "Throws", "orElse": "Else",
	// the same Veles name for different Go ones, per node
	"AnnotationInfo.targets": "On", "AnnotationInfo.readers": "For", "AnnotationInfo.readersPos": "ForPos",
	"ImplDecl.traitType": "Trait", "ImplDecl.withFun": "With", "LoopStmt.bound": "Var", "FunStmt.decl": "Fun",
	"BinaryExpr.left": "L", "BinaryExpr.right": "R", "OrFailExpr.left": "L", "OrFailExpr.right": "R",
	"CoalesceExpr.left": "L", "CoalesceExpr.right": "R", "ElvisExpr.left": "L", "ElvisExpr.right": "R",
	"ImplementsExpr.traitType": "Trait", "RaceArm.name": "As",
}

// ruleFieldNames are the names a Veles field may have in the Go tree.
func ruleFieldNames(node, name string) []string {
	names := []string{name}
	if a, ok := ruleFieldAlias[node+"."+name]; ok {
		names = append(names, a)
	}
	if a, ok := ruleFieldAlias[name]; ok {
		names = append(names, a)
	}
	if len(name) > 2 && strings.HasPrefix(name, "is") && name[2] >= 'A' && name[2] <= 'Z' {
		names = append(names, name[2:])
	}
	return names
}

type ruleShape struct {
	goT reflect.Type
	t   *types.Struct
}

// ruleShapes remembers, per Go node type and Veles struct, which Go field each
// Veles field shows (-1: none), so the names are matched once.
var ruleShapes sync.Map

// goFieldIndexes finds the Go field each field of t shows: the same name in any
// case, or the name without its `is`, or the alias the Veles side spells around a
// keyword.
func goFieldIndexes(goT reflect.Type, t *types.Struct) []int {
	key := ruleShape{goT, t}
	if at, ok := ruleShapes.Load(key); ok {
		return at.([]int)
	}
	at := make([]int, len(t.Fields))
	for i, f := range t.Fields {
		at[i] = -1
	find:
		for _, want := range ruleFieldNames(t.Name, f.Name) {
			for j := 0; j < goT.NumField(); j++ {
				if goT.Field(j).IsExported() && strings.EqualFold(goT.Field(j).Name, want) {
					at[i] = j
					break find
				}
			}
		}
	}
	ruleShapes.Store(key, at)
	return at
}

// conv builds the value of type t that the Go value v shows.
func (h *ruleHost) conv(v reflect.Value, t types.Type) ConstVal {
	switch t := t.(type) {
	case *types.Nullable:
		if isNilValue(v) || !v.IsValid() {
			return &CNull{t}
		}
		return &CSome{t, h.conv(v, t.Elem)}
	case *types.Pointer:
		inner := &cell{h.conv(v, t.Elem)}
		return &CPtr{T: t, get: func() ConstVal { return inner.v }, set: func(x ConstVal) { inner.v = x }}
	case *types.List:
		out := &CList{T: t}
		v = derefAll(v)
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				out.Elems = append(out.Elems, h.conv(v.Index(i), t.Elem))
			}
		}
		return out
	case *types.Enum:
		v = derefAll(v)
		return &CInt{t, big.NewInt(v.Int())}
	case *types.Basic:
		v = derefAll(v)
		switch {
		case t == types.TString:
			return &CString{v.String()}
		case t == types.TBool:
			return &CBool{v.Bool()}
		case types.IsFloat(t):
			return &CFloat{t, v.Float()}
		}
		switch v.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return &CInt{t, new(big.Int).SetUint64(v.Uint())}
		}
		return &CInt{t, big.NewInt(v.Int())}
	case *types.Sealed:
		return h.sealed(v, t)
	case *types.Struct:
		return h.structVal(v, t)
	}
	h.fail(fmt.Sprintf("a value of type '%s'", t))
	return &CUnit{}
}

func (h *ruleHost) structVal(v reflect.Value, t *types.Struct) ConstVal {
	if t.Name == "Span" && t.Module == h.lint.Path {
		if s, ok := derefAll(v).Interface().(source.Span); ok {
			return h.spanVal(s)
		}
	}
	if t.Name == "File" && t.Module == h.lint.Path {
		if f, ok := v.Interface().(*source.File); ok && f != nil {
			return h.fileVal(f)
		}
	}
	var key ruleKey
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		key = ruleKey{v.Pointer(), t}
		if m, ok := h.memo[key]; ok {
			return m
		}
	}
	v = derefAll(v)
	out := &CStruct{T: t, Fields: make([]ConstVal, len(t.Fields))}
	var at []int
	if v.Kind() == reflect.Struct {
		at = goFieldIndexes(v.Type(), t)
	}
	for i, f := range t.Fields {
		if at != nil && at[i] >= 0 {
			out.Fields[i] = h.conv(v.Field(at[i]), f.Type)
			continue
		}
		h.where = t.Name + "." + f.Name
		out.Fields[i] = h.zero(f.Type)
	}
	if key.p != 0 {
		h.memo[key] = out
	}
	return out
}

// sealed builds a value of a sealed type from a Go interface value: the
// variant with the dynamic type's name, or the one that wraps it (`FunDecl` is
// shown as a Decl by `FunDeclNode`, a `Block` as a Stmt by `BlockStmt`).
func (h *ruleHost) sealed(v reflect.Value, t *types.Sealed) ConstVal {
	inner := v
	for inner.Kind() == reflect.Interface && !inner.IsNil() {
		inner = inner.Elem()
	}
	name := derefAll(inner).Type().Name()
	if inner.Kind() == reflect.Invalid || isNilValue(inner) {
		h.fail("a missing node")
		return &CUnit{}
	}
	if n, ok := ruleGoName[name]; ok {
		name = n
	}
	find := func(n string) *types.Struct {
		for _, vs := range t.Variants {
			if vs.Name == n {
				return vs
			}
		}
		return nil
	}
	if vs := find(name); vs != nil {
		return &CVariant{t, vs, h.structVal(inner, vs)}
	}
	for _, n := range []string{name + "Node", name + t.Name} {
		if vs := find(n); vs != nil && len(vs.Fields) == 1 {
			payload := &CStruct{T: vs, Fields: []ConstVal{h.conv(inner, vs.Fields[0].Type)}}
			return &CVariant{t, vs, payload}
		}
	}
	// syntax the checker added to the tree (a resolved type, a derived
	// default): a rule sees it as the family's Bad node
	if vs := find("Bad" + t.Name); vs != nil {
		var pos source.Span
		if n, ok := inner.Interface().(ast.Node); ok {
			pos = n.Span()
		}
		payload := &CStruct{T: vs, Fields: make([]ConstVal, len(vs.Fields))}
		for i, f := range vs.Fields {
			if f.Name == "pos" {
				payload.Fields[i] = h.spanVal(pos)
			} else {
				payload.Fields[i] = h.zero(f.Type)
			}
		}
		return &CVariant{t, vs, payload}
	}
	h.fail("the syntax '" + name + "' (" + t.Name + ")")
	return &CUnit{}
}

// ---------------------------------------------------------------------------
// running

var astPkgPath = reflect.TypeOf(ast.File{}).PkgPath()

// ruleSet is one rule made ready for a file.
type ruleSet struct {
	lintName string // package.rule, as the manifest names it
	level    LintLevel
	calls    map[string]ruleCall // Go node type name -> callback
}

type ruleCall struct {
	fn    *CFunc
	param types.Type // the node struct the callback takes
}

// walk calls every callback of every rule on the nodes of v, parents before
// children.
func (h *ruleHost) walk(ev *constEval, v reflect.Value, sets []*ruleSet, ctx ConstVal, reports *CList, onReport func(*ruleSet, ConstVal)) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			h.walk(ev, v.Elem(), sets, ctx, reports, onReport)
		}
		return
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		if v.Elem().Kind() == reflect.Struct && v.Elem().Type().PkgPath() == astPkgPath {
			h.visit(ev, v, v.Elem(), sets, ctx, reports, onReport)
		}
		return
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			h.walk(ev, v.Index(i), sets, ctx, reports, onReport)
		}
		return
	case reflect.Struct:
		if v.Type().PkgPath() == astPkgPath {
			h.visit(ev, v, v, sets, ctx, reports, onReport)
		}
	}
}

func (h *ruleHost) visit(ev *constEval, node, st reflect.Value, sets []*ruleSet, ctx ConstVal, reports *CList, onReport func(*ruleSet, ConstVal)) {
	name := st.Type().Name()
	if n, ok := ruleGoName[name]; ok {
		name = n
	}
	var arg ConstVal
	for _, set := range sets {
		call, ok := set.calls[name]
		if !ok {
			continue
		}
		if arg == nil {
			arg = h.conv(node, call.param)
			if h.failed != "" {
				return
			}
		}
		before := len(reports.Elems)
		ev.invokeWith(call.fn.Fn, []ConstVal{arg, ctx}, source.Span{}, call.fn)
		for _, r := range reports.Elems[before:] {
			onReport(set, r)
		}
	}
	for i := 0; i < st.NumField(); i++ {
		if st.Type().Field(i).IsExported() {
			h.walk(ev, st.Field(i), sets, ctx, reports, onReport)
		}
	}
}

// libraryLints runs the rules of other packages that the entry packages'
// manifests turned on.
func (c *Checker) libraryLintsRun() []source.Diagnostic {
	if c.roundDiags.HasErrors() {
		return nil
	}
	if c.pkg == nil {
		return nil
	}
	lintMod := c.pkg.Modules["std/lint"]
	keys := make([]string, 0, len(c.pkg.Modules))
	for k := range c.pkg.Modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []source.Diagnostic
	for _, k := range keys {
		m := c.pkg.Modules[k]
		if m.Std || m.Pkg == nil || m.Pkg.Manifest == nil || len(m.Pkg.Manifest.Lint.Library) == 0 {
			continue
		}
		if lintMod == nil {
			continue
		}
		out = append(out, c.runLibraryLints(m, lintMod)...)
	}
	return out
}

// runLibraryLints runs the rules a module's package names on the module's
// files.
func (c *Checker) runLibraryLints(m *Module, lintMod *Module) (out []source.Diagnostic) {
	man := m.Pkg.Manifest
	names := make([]string, 0, len(man.Lint.Library))
	for k := range man.Lint.Library {
		names = append(names, k)
	}
	sort.Strings(names)
	budget := int64(libraryLintSteps)
	if c.pkg.ConstSteps > 0 {
		budget = c.pkg.ConstSteps
	}
	// group by package: its `rules()` runs once
	byPkg := map[string][]string{}
	var pkgs []string
	for _, n := range names {
		i := strings.LastIndexByte(n, '.')
		if _, ok := byPkg[n[:i]]; !ok {
			pkgs = append(pkgs, n[:i])
		}
		byPkg[n[:i]] = append(byPkg[n[:i]], n[i+1:])
	}
	type made struct {
		rule  *CStruct
		level LintLevel
		name  string
	}
	var chosen []made
	for _, pkg := range pkgs {
		root := m.Pkg.lintMods[pkg]
		if root == nil {
			continue // not loaded: reported where it was resolved
		}
		list, at, ok := c.libraryRules(root)
		if !ok {
			continue
		}
		have := map[string]*CStruct{}
		for _, r := range list {
			have[string(r.Fields[0].(*CString).V)] = r
		}
		for _, rn := range byPkg[pkg] {
			full := pkg + "." + rn
			level := man.Lint.Library[full]
			r := have[rn]
			if r == nil {
				var known []string
				for k := range have {
					known = append(known, k)
				}
				sort.Strings(known)
				msg := fmt.Sprintf("[lint] %s: package '%s' has no rule '%s'", full, pkg, rn)
				if len(known) > 0 {
					msg += "; its rules are " + strings.Join(known, ", ")
				}
				if s := didYouMean(rn, known); s != "" {
					msg += " (did you mean '" + s + "'?)"
				}
				out = append(out, source.Diagnostic{Severity: source.Error, Span: at, Message: msg})
				continue
			}
			if level == LintAllow {
				continue
			}
			chosen = append(chosen, made{r, level, full})
		}
	}
	if len(chosen) == 0 {
		return out
	}
	for _, f := range m.Files {
		if f.Source == nil {
			continue
		}
		h := &ruleHost{c: c, lint: lintMod, file: f.Source, files: map[*source.File]ConstVal{}, back: map[*CStruct]*source.File{}, memo: map[ruleKey]ConstVal{}}
		ev := c.newConstEval(source.Span{}, false)
		ev.budget = budget
		reports := &CList{T: &types.List{Elem: h.structT("Report")}, Elems: nil}
		ctxT := h.structT("Context")
		ctx := &CStruct{T: ctxT, Fields: make([]ConstVal, len(ctxT.Fields))}
		ctx.Fields[ruleField(ctxT, "file")] = h.fileVal(f.Source)
		ctx.Fields[ruleField(ctxT, "reports")] = reports
		ruleT := h.structT("Rule")
		visT := h.structT("Visitor")
		var sets []*ruleSet
		c.runConstGuarded(ev, func() {
			for _, ch := range chosen {
				create, ok := ch.rule.Fields[ruleField(ruleT, "create")].(*CFunc)
				if !ok {
					continue
				}
				vis, ok := ev.invokeWith(create.Fn, []ConstVal{ctx}, source.Span{}, create).(*CStruct)
				if !ok {
					continue
				}
				set := &ruleSet{lintName: ch.name, level: ch.level, calls: map[string]ruleCall{}}
				for i, vf := range visT.Fields {
					some, ok := vis.Fields[i].(*CSome)
					if !ok {
						continue
					}
					fn, ok := some.V.(*CFunc)
					if !ok {
						continue
					}
					ft := vf.Type.(*types.Nullable).Elem.(*types.Func)
					set.calls[strings.TrimPrefix(vf.Name, "on")] = ruleCall{fn, ft.Params[0].Type}
				}
				sets = append(sets, set)
			}
			var found []source.Diagnostic
			h.walk(ev, reflect.ValueOf(f), sets, ctx, reports, func(set *ruleSet, r ConstVal) {
				found = append(found, h.diagnostic(set, r.(*CStruct)))
			})
			if h.failed != "" {
				ev.errorf(source.Span{}, "a lint rule was shown %s, which std/lint cannot show yet", h.failed)
			}
			out = append(out, found...)
		})
	}
	return out
}

// runConstGuarded runs body, which may fail as a constant does; the failure is
// reported through the checker, which is how a rule that panics or runs past
// its budget is told to its author.
func (c *Checker) runConstGuarded(ev *constEval, body func()) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(constFailed); !ok {
				panic(r)
			}
		}
	}()
	body()
}

// libraryRules runs a package's `rules()`.
func (c *Checker) libraryRules(root *Module) (rules []*CStruct, at source.Span, ok bool) {
	sym := root.Scope.symbols["rules"]
	if sym == nil || sym.Kind != SymFunc || !sym.Pub {
		c.errorf(source.Span{}, "package '%s' has no 'public const fun rules(): List<lint.Rule>', so it has no lint rules (D177)", root.Name())
		return nil, source.Span{}, false
	}
	fn := c.instantiate(sym.Func, nil, nil, sym.Span)
	at = sym.Span
	ev := c.newConstEval(at, false)
	ev.budget = libraryLintSteps
	c.runConstGuarded(ev, func() {
		list, isList := ev.invoke(fn, nil, at).(*CList)
		if !isList {
			return
		}
		for _, e := range list.Elems {
			if s, isStruct := e.(*CStruct); isStruct {
				rules = append(rules, s)
			}
		}
		ok = true
	})
	return rules, at, ok
}

// diagnostic turns a rule's Report into a diagnostic at the level the
// manifest gave it.
func (h *ruleHost) diagnostic(set *ruleSet, r *CStruct) source.Diagnostic {
	t := r.T
	d := source.Diagnostic{
		Severity: source.Warning,
		Span:     h.sourceSpan(r.Fields[ruleField(t, "span")]),
		Message:  r.Fields[ruleField(t, "message")].(*CString).V,
		Lint:     "",
	}
	if set.level == LintDeny {
		d.Severity = source.Error
	}
	d.Message += " [" + set.lintName + "]"
	if some, ok := r.Fields[ruleField(t, "fix")].(*CSome); ok {
		fix := some.V.(*CStruct)
		ft := fix.T
		out := &source.Fix{Title: fix.Fields[ruleField(ft, "title")].(*CString).V}
		for _, e := range fix.Fields[ruleField(ft, "edits")].(*CList).Elems {
			edit := e.(*CStruct)
			et := edit.T
			out.Edits = append(out.Edits, source.TextEdit{
				Span:    h.sourceSpan(edit.Fields[ruleField(et, "span")]),
				NewText: edit.Fields[ruleField(et, "text")].(*CString).V,
			})
		}
		if out.Title == "" {
			out.Title = "apply the fix of " + set.lintName
		}
		d.Fix = out
	}
	return d
}

// libraryLints runs the rules; what the evaluator reports on the way (a rule
// that panics, one that ran past its budget) is returned with their findings,
// since the round's own diagnostics were taken already.
func (c *Checker) libraryLints() []source.Diagnostic {
	if c.pkg == nil || c.roundDiags.HasErrors() {
		return nil
	}
	before := len(c.roundDiags.Items)
	out := c.libraryLintsRun()
	out = append(out, c.roundDiags.Items[before:]...)
	c.roundDiags.Items = c.roundDiags.Items[:before]
	return out
}
