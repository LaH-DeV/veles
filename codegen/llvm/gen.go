// Package llvm emits textual LLVM IR from the HIR (spec I1/I2). The output
// is handed to clang together with the C runtime.
package llvm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/types"
)

type gen struct {
	prog *sema.Program

	typeDecls map[string]string
	typeOrder []string
	strs      map[string]string
	strOrder  []string
	out       strings.Builder // function definitions
	helpers   strings.Builder // generated show/eq helpers
	showFns   map[string]string
	eqFns     map[string]string
	pending   []func() // helper bodies to generate after the current function

	// per-function state
	fn       *sema.Func
	body     strings.Builder
	allocas  strings.Builder
	tmp      int
	label    int
	term     bool
	storage  map[*sema.Var]string
	loops    map[*sema.Loop]loopLabels
	fnResult types.Type // Result<T, E> sealed when the function throws
	inHelper bool
}

type loopLabels struct {
	cond, post, end string
}

// Generate returns the LLVM IR module for a checked program.
func Generate(prog *sema.Program) string {
	g := &gen{
		prog:      prog,
		typeDecls: map[string]string{},
		strs:      map[string]string{},
		showFns:   map[string]string{},
		eqFns:     map[string]string{},
	}
	g.typeDecls[strType] = "{ ptr, i64 }"
	g.typeOrder = append(g.typeOrder, strType)

	for _, fn := range prog.Funcs {
		if fn.Extern {
			continue
		}
		g.function(fn)
		g.flushPending()
	}
	g.globalsInit()
	g.entryPoint()
	g.flushPending()

	var sb strings.Builder
	sb.WriteString("; Veles bootstrap compiler output\n")
	sb.WriteString("target triple = \"" + targetTriple() + "\"\n\n")
	for _, name := range g.typeOrder {
		fmt.Fprintf(&sb, "%s = type %s\n", name, g.typeDecls[name])
	}
	sb.WriteString("\n")
	for _, name := range g.strOrder {
		sb.WriteString(g.strs[name])
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	for _, gl := range prog.Globals {
		fmt.Fprintf(&sb, "@%s = global %s zeroinitializer\n", mangleGlobal(gl), g.llType(gl.Type))
	}
	sb.WriteString("\n")
	sb.WriteString(runtimeDecls)
	for _, fn := range prog.Funcs {
		if fn.Extern {
			sb.WriteString(g.externDecl(fn))
		}
	}
	sb.WriteString("\n")
	sb.WriteString(g.out.String())
	sb.WriteString(g.helpers.String())
	return sb.String()
}

func mangleGlobal(gl *sema.Global) string {
	return "g_" + strings.NewReplacer(".", "_", "/", "_").Replace(gl.Name)
}

const runtimeDecls = `declare void @veles_rt_init(i32, ptr)
declare ptr @veles_alloc(i64)
declare void @veles_panic(ptr, i64)
declare void @veles_report_error(ptr, i64)
declare void @veles_string_concat(ptr, ptr, i64, ptr, i64)
declare i1 @veles_string_eq(ptr, i64, ptr, i64)
declare i32 @veles_string_cmp(ptr, i64, ptr, i64)
declare i1 @veles_string_starts_with(ptr, i64, ptr, i64)
declare i1 @veles_string_ends_with(ptr, i64, ptr, i64)
declare i1 @veles_string_contains(ptr, i64, ptr, i64)
declare i1 @veles_string_substring(ptr, ptr, i64, i64, i64)
declare i1 @veles_string_to_int(ptr, i64, ptr)
declare void @veles_i64_to_string(ptr, i64)
declare void @veles_u64_to_string(ptr, i64)
declare void @veles_f64_to_string(ptr, double)
declare void @veles_bool_to_string(ptr, i1)
declare ptr @veles_list_new(i64, i64)
declare i64 @veles_list_len(ptr)
declare void @veles_list_push(ptr, ptr)
declare ptr @veles_list_ref(ptr, i64)
declare i1 @veles_list_pop(ptr, ptr)
declare ptr @veles_list_copy(ptr)
declare void @veles_list_clear(ptr)
declare void @llvm.memcpy.p0.p0.i64(ptr, ptr, i64, i1)
declare { i8, i1 } @llvm.sadd.with.overflow.i8(i8, i8)
declare { i16, i1 } @llvm.sadd.with.overflow.i16(i16, i16)
declare { i32, i1 } @llvm.sadd.with.overflow.i32(i32, i32)
declare { i64, i1 } @llvm.sadd.with.overflow.i64(i64, i64)
declare { i8, i1 } @llvm.ssub.with.overflow.i8(i8, i8)
declare { i16, i1 } @llvm.ssub.with.overflow.i16(i16, i16)
declare { i32, i1 } @llvm.ssub.with.overflow.i32(i32, i32)
declare { i64, i1 } @llvm.ssub.with.overflow.i64(i64, i64)
declare { i8, i1 } @llvm.smul.with.overflow.i8(i8, i8)
declare { i16, i1 } @llvm.smul.with.overflow.i16(i16, i16)
declare { i32, i1 } @llvm.smul.with.overflow.i32(i32, i32)
declare { i64, i1 } @llvm.smul.with.overflow.i64(i64, i64)
declare { i8, i1 } @llvm.uadd.with.overflow.i8(i8, i8)
declare { i16, i1 } @llvm.uadd.with.overflow.i16(i16, i16)
declare { i32, i1 } @llvm.uadd.with.overflow.i32(i32, i32)
declare { i64, i1 } @llvm.uadd.with.overflow.i64(i64, i64)
declare { i8, i1 } @llvm.usub.with.overflow.i8(i8, i8)
declare { i16, i1 } @llvm.usub.with.overflow.i16(i16, i16)
declare { i32, i1 } @llvm.usub.with.overflow.i32(i32, i32)
declare { i64, i1 } @llvm.usub.with.overflow.i64(i64, i64)
declare { i8, i1 } @llvm.umul.with.overflow.i8(i8, i8)
declare { i16, i1 } @llvm.umul.with.overflow.i16(i16, i16)
declare { i32, i1 } @llvm.umul.with.overflow.i32(i32, i32)
declare { i64, i1 } @llvm.umul.with.overflow.i64(i64, i64)
`

// ---------------------------------------------------------------------------
// builder helpers

func (g *gen) newTmp() string {
	g.tmp++
	return fmt.Sprintf("%%t%d", g.tmp)
}

func (g *gen) newLabel(hint string) string {
	g.label++
	return fmt.Sprintf("%s.%d", hint, g.label)
}

// emit writes one instruction. After a terminator, a fresh (unreachable)
// block is opened so the IR stays well-formed.
func (g *gen) emit(format string, args ...any) {
	if g.term {
		g.placeLabel(g.newLabel("dead"))
	}
	g.body.WriteString("  ")
	fmt.Fprintf(&g.body, format, args...)
	g.body.WriteString("\n")
}

func (g *gen) emitTerm(format string, args ...any) {
	g.emit(format, args...)
	g.term = true
}

func (g *gen) placeLabel(name string) {
	if !g.term {
		g.body.WriteString("  br label %" + name + "\n")
	}
	g.body.WriteString(name + ":\n")
	g.term = false
}

func (g *gen) alloca(llt string) string {
	g.tmp++
	name := fmt.Sprintf("%%a%d", g.tmp)
	fmt.Fprintf(&g.allocas, "  %s = alloca %s\n", name, llt)
	return name
}

// stringConst interns a string literal and returns a `%str` constant.
func (g *gen) stringConst(s string) string {
	name, ok := g.strs[s]
	if !ok {
		name = fmt.Sprintf("@.str.%d", len(g.strOrder))
		var enc strings.Builder
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 0x20 && c < 0x7f && c != '"' && c != '\\' {
				enc.WriteByte(c)
			} else {
				fmt.Fprintf(&enc, "\\%02X", c)
			}
		}
		decl := fmt.Sprintf("%s = private unnamed_addr constant [%d x i8] c\"%s\\00\"", name, len(s)+1, enc.String())
		g.strs[s] = name
		g.strOrder = append(g.strOrder, name)
		// store decl under the name key for output ordering
		g.strs[name] = decl
	}
	return fmt.Sprintf("{ ptr %s, i64 %d }", name, len(s))
}

func (g *gen) strPtrLen(v string) (string, string) {
	p := g.newTmp()
	g.emit("%s = extractvalue %s %s, 0", p, strType, v)
	l := g.newTmp()
	g.emit("%s = extractvalue %s %s, 1", l, strType, v)
	return p, l
}

// panic emits a call to veles_panic with a message and terminates.
func (g *gen) panicMsg(msg string) {
	c := g.stringConst(msg)
	p, l := g.strPtrLen(c)
	g.emit("call void @veles_panic(ptr %s, i64 %s)", p, l)
	g.emitTerm("unreachable")
}

// ---------------------------------------------------------------------------
// functions

func (g *gen) resetFn(fn *sema.Func) {
	g.fn = fn
	g.body.Reset()
	g.allocas.Reset()
	g.tmp = 0
	g.label = 0
	g.term = false
	g.storage = map[*sema.Var]string{}
	g.loops = map[*sema.Loop]loopLabels{}
	g.fnResult = nil
}

// retLL returns the LLVM return type of a function.
func (g *gen) retLL(fn *sema.Func) string {
	if fn.Sig.Effects.Throws {
		return g.llType(g.resultOf(fn))
	}
	if types.IsUnit(fn.Sig.Ret) || types.IsNever(fn.Sig.Ret) {
		return "void"
	}
	return g.llType(fn.Sig.Ret)
}

func (g *gen) resultOf(fn *sema.Func) types.Type {
	if g.fnResult != nil && g.fn == fn {
		return g.fnResult
	}
	// The checker instantiates Result<T, E> for every throwing call; find
	// the same instance through the prelude template.
	return g.prog.ResultType(fn.Sig.Ret, fn.Sig.Effects.Error)
}

func (g *gen) externDecl(fn *sema.Func) string {
	var params []string
	for _, p := range fn.Sig.Params {
		params = append(params, g.externParamTypes(p.Type)...)
	}
	ret := "void"
	if !types.IsUnit(fn.Sig.Ret) {
		ret = g.llType(fn.Sig.Ret)
	}
	return fmt.Sprintf("declare %s @%s(%s)\n", ret, fn.Name, strings.Join(params, ", "))
}

// externParamTypes lowers a parameter for the C ABI: strings become a
// (ptr, len) pair so no aggregate crosses the boundary by value.
func (g *gen) externParamTypes(t types.Type) []string {
	if types.IsString(t) {
		return []string{"ptr", "i64"}
	}
	return []string{g.llType(t)}
}

func (g *gen) function(fn *sema.Func) {
	g.resetFn(fn)
	if fn.Sig.Effects.Throws {
		g.fnResult = g.prog.ResultType(fn.Sig.Ret, fn.Sig.Effects.Error)
	}
	var params []string
	var prologue []string
	bind := func(v *sema.Var, i int) {
		llt := g.llType(v.Type)
		params = append(params, fmt.Sprintf("%s %%p%d", llt, i))
		st := g.varStorage(v, true)
		prologue = append(prologue, fmt.Sprintf("  store %s %%p%d, ptr %s", llt, i, st))
	}
	if fn.Receiver != nil {
		bind(fn.Receiver, 0)
	}
	for i, p := range fn.Params {
		bind(p, i+1)
	}
	if fn.Body != nil {
		g.block(fn.Body)
	}
	if !g.term {
		// falling off the end of a unit function
		if fn.Sig.Effects.Throws {
			g.retValue(nil)
		} else if types.IsUnit(fn.Sig.Ret) {
			g.emitTerm("ret void")
		} else {
			g.emitTerm("unreachable")
		}
	}
	fmt.Fprintf(&g.out, "define %s @%s(%s) {\nentry:\n", g.retLL(fn), fn.Name, strings.Join(params, ", "))
	g.out.WriteString(g.allocas.String())
	for _, p := range prologue {
		g.out.WriteString(p + "\n")
	}
	g.out.WriteString(g.body.String())
	g.out.WriteString("}\n\n")
}

// varStorage returns the pointer to a variable's storage, creating it on
// first use. Address-taken variables live on the heap (D10).
func (g *gen) varStorage(v *sema.Var, param bool) string {
	if v.IsGlobal {
		return "@" + mangleGlobal(v.Global)
	}
	if st, ok := g.storage[v]; ok {
		return st
	}
	llt := g.llType(v.Type)
	if v.AddrTaken && !param {
		slot := g.alloca("ptr")
		g.storage[v] = slot
		return slot
	}
	st := g.alloca(llt)
	g.storage[v] = st
	return st
}

// varPtr returns a pointer to the variable's current storage.
func (g *gen) varPtr(v *sema.Var) string {
	st := g.varStorage(v, false)
	if v.AddrTaken && !v.IsGlobal {
		p := g.newTmp()
		g.emit("%s = load ptr, ptr %s", p, st)
		return p
	}
	return st
}

// declareVar allocates fresh storage for a declaration (a new heap cell
// for address-taken variables on every execution).
func (g *gen) declareVar(v *sema.Var) string {
	st := g.varStorage(v, false)
	if v.AddrTaken && !v.IsGlobal {
		size, _ := g.layout(v.Type)
		cell := g.newTmp()
		g.emit("%s = call ptr @veles_alloc(i64 %d)", cell, size)
		g.emit("store ptr %s, ptr %s", cell, st)
		return cell
	}
	return st
}

// ---------------------------------------------------------------------------
// statements

func (g *gen) block(b *sema.Block) string {
	for _, s := range b.Stmts {
		g.stmt(s)
		if g.term {
			// the rest is unreachable; the checker already diagnosed it
		}
	}
	if b.Value != nil {
		return g.expr(b.Value)
	}
	return zeroOf(b.Type)
}

func zeroOf(t types.Type) string {
	return "zeroinitializer"
}

func (g *gen) stmt(s sema.Stmt) {
	switch s := s.(type) {
	case *sema.Block:
		g.block(s)
	case *sema.VarDecl:
		if s.Init != nil {
			v := g.expr(s.Init)
			st := g.declareVar(s.Var)
			if !types.IsNever(s.Init.Type()) {
				g.emit("store %s %s, ptr %s", g.llType(s.Var.Type), v, st)
			}
		} else {
			st := g.declareVar(s.Var)
			g.emit("store %s zeroinitializer, ptr %s", g.llType(s.Var.Type), st)
		}
	case *sema.Assign:
		v := g.expr(s.Value)
		if types.IsNever(s.Value.Type()) {
			return
		}
		p := g.place(s.Target)
		g.emit("store %s %s, ptr %s", g.llType(s.Target.Type()), v, p)
	case *sema.ExprStmt:
		g.expr(s.X)
	case *sema.Return:
		g.retValue(s.Value)
	case *sema.Loop:
		g.loop(s)
	case *sema.Break:
		g.emitTerm("br label %%%s", g.loops[s.Loop].end)
	case *sema.Continue:
		g.emitTerm("br label %%%s", g.loops[s.Loop].post)
	default:
		panic(fmt.Sprintf("codegen: unsupported statement %T", s))
	}
}

// retValue returns from the current function, wrapping into Result when
// the function throws.
func (g *gen) retValue(value sema.Expr) {
	fn := g.fn
	var v string
	if value != nil {
		v = g.expr(value)
		if types.IsNever(value.Type()) {
			return
		}
	}
	if fn.Sig.Effects.Throws {
		rs := g.fnResult.(*types.Sealed)
		if value == nil || types.IsUnit(fn.Sig.Ret) {
			v = "zeroinitializer"
		}
		payload := g.buildStruct(rs.Variants[0], []string{v})
		r := g.makeTagged(g.llType(rs), 0, g.llType(rs.Variants[0]), payload)
		g.emitTerm("ret %s %s", g.llType(rs), r)
		return
	}
	if value == nil || types.IsUnit(fn.Sig.Ret) {
		g.emitTerm("ret void")
		return
	}
	g.emitTerm("ret %s %s", g.llType(fn.Sig.Ret), v)
}

func (g *gen) loop(l *sema.Loop) {
	labels := loopLabels{cond: g.newLabel("loop.cond"), post: g.newLabel("loop.post"), end: g.newLabel("loop.end")}
	body := g.newLabel("loop.body")
	g.loops[l] = labels
	g.placeLabel(labels.cond)
	if l.Cond != nil {
		c := g.expr(l.Cond)
		g.emitTerm("br i1 %s, label %%%s, label %%%s", c, body, labels.end)
	} else {
		g.emitTerm("br label %%%s", body)
	}
	g.placeLabel(body)
	g.block(l.Body)
	if !g.term {
		g.emitTerm("br label %%%s", labels.post)
	}
	g.placeLabel(labels.post)
	for _, s := range l.Post {
		g.stmt(s)
	}
	g.emitTerm("br label %%%s", labels.cond)
	g.placeLabel(labels.end)
}

// ---------------------------------------------------------------------------
// globals and entry point

func (g *gen) globalsInit() {
	g.resetFn(&sema.Func{Name: "veles_init_globals", Sig: &types.Func{Ret: types.TUnit}})
	for _, gl := range g.prog.Globals {
		if gl.Init == nil {
			continue
		}
		v := g.expr(gl.Init)
		g.emit("store %s %s, ptr @%s", g.llType(gl.Type), v, mangleGlobal(gl))
	}
	g.emitTerm("ret void")
	g.out.WriteString("define void @veles_init_globals() {\nentry:\n")
	g.out.WriteString(g.allocas.String())
	g.out.WriteString(g.body.String())
	g.out.WriteString("}\n\n")
}

func (g *gen) entryPoint() {
	main := g.prog.Main
	g.resetFn(&sema.Func{Name: "main", Sig: &types.Func{Ret: types.TUnit}})
	g.emit("call void @veles_rt_init(i32 %%argc, ptr %%argv)")
	g.emit("call void @veles_init_globals()")
	if main == nil {
		g.emitTerm("ret i32 0")
	} else if main.Sig.Effects.Throws {
		rs := g.prog.ResultType(main.Sig.Ret, main.Sig.Effects.Error).(*types.Sealed)
		r := g.newTmp()
		g.emit("%s = call %s @%s()", r, g.llType(rs), main.Name)
		tag := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", tag, g.llType(rs), r)
		isErr := g.newTmp()
		g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
		errL, okL := g.newLabel("main.err"), g.newLabel("main.ok")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
		g.placeLabel(errL)
		errVariant := rs.Variants[1]
		payload := g.extractTagged(g.llType(rs), r, g.llType(errVariant))
		errVal := g.newTmp()
		g.emit("%s = extractvalue %s %s, 0", errVal, g.llType(errVariant), payload)
		msg := g.show(errVariant.Fields[0].Type, errVal)
		p, l := g.strPtrLen(msg)
		g.emit("call void @veles_report_error(ptr %s, i64 %s)", p, l)
		g.emitTerm("ret i32 1")
		g.placeLabel(okL)
		g.emitTerm("ret i32 0")
	} else {
		g.emit("call void @%s()", main.Name)
		g.emitTerm("ret i32 0")
	}
	g.out.WriteString("define i32 @main(i32 %argc, ptr %argv) {\nentry:\n")
	g.out.WriteString(g.allocas.String())
	g.out.WriteString(g.body.String())
	g.out.WriteString("}\n\n")
}

func (g *gen) flushPending() {
	for len(g.pending) > 0 {
		p := g.pending[0]
		g.pending = g.pending[1:]
		p()
	}
}

// sortedKeys is used for deterministic output.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
