package sema

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Dumps are stable text renderings of the checker's output (plan S10): the
// oracle a checker written in Veles is gated against, phase by phase, as the
// self-hosted parser is against `veles parse`. They are deterministic: every
// list is sorted by a name that does not depend on map order, pointers are
// printed as the names they stand for, and spans are left out (byte offsets
// follow the source, not the checking).
//
//	names  the symbols of the program: functions, globals, structs, sealed types
//	types  their types: signatures with effects, field lists, variants
//	hir    every function's lowered body, each expression with its type
const (
	DumpNames = "names"
	DumpTypes = "types"
	DumpHIR   = "hir"
)

// DumpKinds lists the dump names `veles check --dump` accepts.
var DumpKinds = []string{DumpNames, DumpTypes, DumpHIR}

// DumpProgram writes the dump of kind (see DumpKinds) for a checked program.
// The standard library's symbols are left out unless withStd: they are the
// same in every program and would bury the package's own (a std generic
// instantiated for the package is std's symbol too).
func DumpProgram(w io.Writer, p *Program, kind string, withStd bool) error {
	if !withStd {
		p = withoutStd(p)
	}
	switch kind {
	case DumpNames:
		dumpNames(w, p)
	case DumpTypes:
		dumpTypes(w, p)
	case DumpHIR:
		dumpHIR(w, p)
	default:
		return errors.New("unknown dump " + strconv.Quote(kind) + " (one of " + strings.Join(DumpKinds, ", ") + ")")
	}
	return nil
}

// fromStd reports whether a symbol was declared in the embedded std sources,
// or has no source at all (the compiler made it for a std type).
func fromStd(sp source.Span) bool {
	return sp.File == nil || sp.File.Embedded
}

// withoutStd is a view of p holding only the package's own symbols.
func withoutStd(p *Program) *Program {
	q := *p
	q.Funcs, q.Globals, q.Structs, q.Sealeds, q.Tests = nil, nil, nil, nil, nil
	for _, f := range p.Funcs {
		if !fromStd(f.Span) {
			q.Funcs = append(q.Funcs, f)
		}
	}
	for _, g := range p.Globals {
		if !fromStd(g.Span) {
			q.Globals = append(q.Globals, g)
		}
	}
	for _, s := range p.Structs {
		if !strings.HasPrefix(s.Module, "std") {
			q.Structs = append(q.Structs, s)
		}
	}
	for _, s := range p.Sealeds {
		if !strings.HasPrefix(s.Module, "std") {
			q.Sealeds = append(q.Sealeds, s)
		}
	}
	for _, t := range p.Tests {
		if !fromStd(t.Span) {
			q.Tests = append(q.Tests, t)
		}
	}
	return &q
}

func sortedFuncs(p *Program) []*Func {
	fs := append([]*Func(nil), p.Funcs...)
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].Name < fs[j].Name })
	return fs
}

func sortedGlobals(p *Program) []*Global {
	gs := append([]*Global(nil), p.Globals...)
	sort.SliceStable(gs, func(i, j int) bool { return gs[i].Name < gs[j].Name })
	return gs
}

func dumpNames(w io.Writer, p *Program) {
	for _, f := range sortedFuncs(p) {
		kind := "fun"
		switch {
		case f.Extern:
			kind = "extern"
		case f.IsClosure:
			kind = "closure"
		}
		fmt.Fprintf(w, "%s %s  (%s)\n", kind, f.Name, f.Display)
	}
	for _, g := range sortedGlobals(p) {
		fmt.Fprintf(w, "global %s  (%s)\n", g.Name, g.Display)
	}
	var structs []string
	for _, s := range p.Structs {
		structs = append(structs, s.String())
	}
	sort.Strings(structs)
	for _, s := range structs {
		fmt.Fprintf(w, "struct %s\n", s)
	}
	var sealeds []string
	for _, s := range p.Sealeds {
		sealeds = append(sealeds, s.String())
	}
	sort.Strings(sealeds)
	for _, s := range sealeds {
		fmt.Fprintf(w, "sealed %s\n", s)
	}
	for _, t := range p.Tests {
		fmt.Fprintf(w, "test %s\n", t.Name)
	}
}

func dumpTypes(w io.Writer, p *Program) {
	for _, f := range sortedFuncs(p) {
		fmt.Fprintf(w, "fun %s: %s%s\n", f.Name, funcSig(f), funcEffects(f))
		vars := varNames{}
		if f.Receiver != nil {
			fmt.Fprintf(w, "  this: %s\n", f.Receiver.Type)
		}
		for _, v := range f.Params {
			fmt.Fprintf(w, "  param %s: %s\n", v.Name, v.Type)
		}
		for _, v := range f.Locals {
			fmt.Fprintf(w, "  local %s: %s\n", vars.ref(v), v.Type)
		}
	}
	for _, g := range sortedGlobals(p) {
		fmt.Fprintf(w, "global %s: %s\n", g.Name, g.Type)
	}
	structs := append([]*types.Struct(nil), p.Structs...)
	sort.SliceStable(structs, func(i, j int) bool { return structs[i].String() < structs[j].String() })
	for _, s := range structs {
		fmt.Fprintf(w, "struct %s\n", s)
		for _, f := range s.Fields {
			fmt.Fprintf(w, "  %s: %s\n", f.Name, f.Type)
		}
	}
	sealeds := append([]*types.Sealed(nil), p.Sealeds...)
	sort.SliceStable(sealeds, func(i, j int) bool { return sealeds[i].String() < sealeds[j].String() })
	for _, s := range sealeds {
		fmt.Fprintf(w, "sealed %s\n", s)
		for _, v := range s.Variants {
			fmt.Fprintf(w, "  variant %s\n", v)
		}
	}
}

func funcSig(f *Func) string {
	if f.Sig == nil {
		return "?"
	}
	return f.Sig.String()
}

func funcEffects(f *Func) string {
	var s string
	if f.Suspends {
		s += " suspends"
	}
	if f.Conditional {
		s += " conditional"
	}
	if f.SelfEscapes {
		s += " self-escapes"
	}
	if f.WritesSelf {
		s += " writes-self"
	}
	return s
}

// varNames numbers the variables of one function in the order a dump meets
// them, so the text does not change when something else in the program
// (std, another function) allocates variables.
type varNames map[*Var]int

func (n varNames) ref(v *Var) string {
	id, ok := n[v]
	if !ok {
		id = len(n)
		n[v] = id
	}
	return fmt.Sprintf("%s#%d", v.Name, id)
}

func dumpHIR(w io.Writer, p *Program) {
	d := &hirDumper{w: w, seen: map[uintptr]bool{}, vars: varNames{}}
	for _, f := range sortedFuncs(p) {
		d.top(f)
	}
	for _, g := range sortedGlobals(p) {
		fmt.Fprintf(w, "global %s: %s\n", g.Name, g.Type)
		if g.Init != nil {
			d.vars = varNames{}
			d.line(1, "init:")
			d.value(2, reflect.ValueOf(g.Init))
		}
	}
}

type hirDumper struct {
	w    io.Writer
	seen map[uintptr]bool
	vars varNames
}

func (d *hirDumper) line(depth int, format string, args ...any) {
	fmt.Fprint(d.w, strings.Repeat("  ", depth))
	fmt.Fprintf(d.w, format, args...)
	fmt.Fprintln(d.w)
}

// top prints one function: a header with its signature and effects, then its
// body. Fields are written by reflection, so a new HIR node is dumped
// without being listed here.
func (d *hirDumper) top(f *Func) {
	d.vars = varNames{}
	fmt.Fprintf(d.w, "fun %s: %s%s\n", f.Name, funcSig(f), funcEffects(f))
	if f.Display != "" && f.Display != f.Name {
		d.line(1, "display: %s", f.Display)
	}
	if f.Receiver != nil {
		d.line(1, "this: %s", d.vars.ref(f.Receiver))
	}
	for _, v := range f.Params {
		d.line(1, "param: %s %s", d.vars.ref(v), v.Type)
	}
	for _, v := range f.CapVars {
		d.line(1, "capture: %s %s", d.vars.ref(v), v.Type)
	}
	switch {
	case f.Extern:
		d.line(1, "extern")
	case f.Body == nil:
		d.line(1, "no body")
	default:
		d.line(1, "body:")
		d.value(2, reflect.ValueOf(f.Body))
	}
}

var (
	dumpSpanType     = reflect.TypeOf(source.Span{})
	dumpTypeIface    = reflect.TypeOf((*types.Type)(nil)).Elem()
	dumpStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

// value writes v, a statement, expression or a part of one.
func (d *hirDumper) value(depth int, v reflect.Value) {
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		if v.Type().Implements(dumpTypeIface) && v.Type() == dumpTypeIface {
			d.line(depth, "%s", v.Interface().(types.Type).String())
			return
		}
		v = v.Elem()
	}
	switch x := v.Interface().(type) {
	case *Var:
		d.line(depth, "var %s", d.vars.ref(x))
		return
	case *Func:
		d.line(depth, "func @%s", x.Name)
		return
	case *Global:
		d.line(depth, "global @%s", x.Name)
		return
	case *Executor:
		d.line(depth, "executor %s", x.Name)
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		if v.Type().Implements(dumpStringerType) && v.Type().Elem().PkgPath() == "github.com/LaH-DeV/veles/types" {
			d.line(depth, "%s", v.Interface().(fmt.Stringer).String())
			return
		}
		if d.seen[v.Pointer()] {
			d.line(depth, "<again %s>", v.Type().Elem().Name())
			return
		}
		d.seen[v.Pointer()] = true
		defer delete(d.seen, v.Pointer())
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		d.structValue(depth, v)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			d.value(depth, v.Index(i))
		}
	case reflect.Map:
		d.mapValue(depth, v)
	default:
		d.line(depth, "%s", scalar(v))
	}
}

func scalar(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return fmt.Sprintf("%q", v.String())
	}
	if v.CanInterface() {
		if s, ok := v.Interface().(fmt.Stringer); ok {
			return s.String()
		}
	}
	return fmt.Sprint(v.Interface())
}

func (d *hirDumper) mapValue(depth int, v reflect.Value) {
	type kv struct {
		key string
		val reflect.Value
	}
	var kvs []kv
	for it := v.MapRange(); it.Next(); {
		kvs = append(kvs, kv{d.keyText(it.Key()), it.Value()})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].key < kvs[j].key })
	for _, e := range kvs {
		d.line(depth, "%s =>", e.key)
		d.value(depth+1, e.val)
	}
}

func (d *hirDumper) keyText(k reflect.Value) string {
	if k.Kind() == reflect.Interface {
		k = k.Elem()
	}
	switch x := k.Interface().(type) {
	case *Var:
		return d.vars.ref(x)
	case *Func:
		return "@" + x.Name
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(k.Interface())
}

// structValue prints a node as its type name and then its non-zero exported
// fields, a type-valued field on the node's own line as `: T`.
func (d *hirDumper) structValue(depth int, v reflect.Value) {
	t := v.Type()
	if t == dumpSpanType {
		return
	}
	name := t.Name()
	typ := ""
	// exprBase holds the node's type; print it beside the name
	if e, ok := v.Interface().(Expr); ok {
		if et := e.Type(); et != nil {
			typ = ": " + et.String()
		}
	}
	d.line(depth, "%s%s", name, typ)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" || sf.Type == dumpSpanType {
			continue
		}
		fv := v.Field(i)
		// a number is always written: 0 is a real operator or index (BinAdd)
		switch fv.Kind() {
		case reflect.Bool, reflect.String, reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
			if fv.IsZero() || (fv.Kind() != reflect.Pointer && fv.Kind() != reflect.Interface && fv.Kind() != reflect.Bool && fv.Kind() != reflect.String && fv.Len() == 0) {
				continue
			}
		}
		if isLeaf(fv) {
			d.line(depth+1, "%s: %s", sf.Name, d.leafText(fv))
			continue
		}
		d.line(depth+1, "%s:", sf.Name)
		d.value(depth+2, fv)
	}
}

// isLeaf reports whether a field is written on one line: a scalar, a type, a
// variable, a function.
func isLeaf(v reflect.Value) bool {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		if v.Type() == dumpTypeIface {
			return true
		}
		v = v.Elem()
	}
	switch v.Interface().(type) {
	case *Var, *Func, *Global:
		return true
	}
	if v.Kind() == reflect.Pointer && v.Type().Implements(dumpStringerType) &&
		v.Type().Elem().PkgPath() == "github.com/LaH-DeV/veles/types" {
		return true
	}
	switch v.Kind() {
	case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func (d *hirDumper) leafText(v reflect.Value) string {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "nil"
		}
		v = v.Elem()
	}
	switch x := v.Interface().(type) {
	case *Var:
		return d.vars.ref(x)
	case *Func:
		return "@" + x.Name
	case *Global:
		return "@" + x.Name
	case types.Type:
		return x.String()
	}
	return scalar(v)
}
