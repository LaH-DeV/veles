// Package llvm emits textual LLVM IR from the HIR (spec I1/I2). The output
// is handed to clang together with the C runtime.
package llvm

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
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
	thunks    map[string]bool
	hashFns   map[string]string
	vtables   map[string]bool
	typeIDs   map[string]int // type identity of a method table's type, by its info global (D117, D135)
	typeInfos []string       // the info globals, in id order
	// plain and viewed instances of functions (D116, plain.go)
	instances     map[plainKey]*sema.Func
	instanceCount map[*sema.Func]int
	plainEnvs     map[*sema.Func]map[*sema.Var]bool
	plainLambdas  map[*sema.Closure]bool
	held          []heldFn // coroutine copies of conditional functions, until referenced
	descs         map[string]string
	memo          map[string]bool // isMem, by type (mem.go)
	descNames     map[string]bool
	descOut       strings.Builder
	consts        map[sema.ConstVal]string // constant tables (consts.go)
	lay           *types.Layout            // sizes, alignments, offsets (D120)
	fieldMap      map[string][]int         // struct field → LLVM index, for padded struct types
	overAlign     map[string]int           // LLVM struct type → alignment above LLVM's own
	constOut      strings.Builder
	eqPtrFns      map[string]string
	pending       []func() // helper bodies to generate after the current function

	// per-function state
	fn               *sema.Func
	body             strings.Builder
	allocas          strings.Builder
	tmp              int
	label            int
	term             bool
	storage          map[*sema.Var]string
	loops            map[*sema.Loop]loopLabels
	fnResult         types.Type  // Result<T, E> sealed when the function throws
	cleanups         []sema.Expr // active `with` close calls, innermost last
	loopCleanupDepth map[*sema.Loop]int
	coro             *coroState
	scopeSlots       map[*sema.ScopeBlock]string
	// bodyScopes are the fail-fast scopes whose body is being emitted,
	// innermost last: a suspension point inside checks them and, when a
	// child has failed, abandons the body for the scope's join (D34).
	bodyScopes []bodyScope
	// abandonSlots map the "scope.abandon" cleanup entries to their scope
	// slot: leaving a scope body early (return, throw, cancellation) cancels
	// the children and joins them, so nothing outlives the block (D34).
	abandonSlots map[*sema.Builtin]string
	// receivingSlots map the "receiving.restore" cleanup entries (D111) to
	// the slot holding the bindings to put back
	receivingSlots map[*sema.Builtin]string
	// cleanupRecs are the frame slots holding each active cleanup's entry
	// in the task's cleanup list: pushing a `with` allocates nothing
	cleanupRecs map[sema.Expr]string
	closeThunks map[*sema.With]string // panic-path close functions, per `with`
	thunkSeq    int
	// inCleanup is set while cleanups are emitted: a suspension point in a
	// cleanup (the join of an abandoned scope) is non-cancellable (D47) and
	// does not re-enter the cleanups
	inCleanup   int
	launchSlots map[*sema.Launch]string
	ramps       map[*sema.Func]*sema.Func
	envSlot     string // closure environment pointer slot
}

type loopLabels struct {
	cond, post, end string
}

// Generate returns the LLVM IR module for a checked program.
func Generate(prog *sema.Program) string {
	g := &gen{
		prog:          prog,
		typeDecls:     map[string]string{},
		strs:          map[string]string{},
		showFns:       map[string]string{},
		eqFns:         map[string]string{},
		thunks:        map[string]bool{},
		hashFns:       map[string]string{},
		vtables:       map[string]bool{},
		typeIDs:       map[string]int{},
		instances:     map[plainKey]*sema.Func{},
		instanceCount: map[*sema.Func]int{},
		plainEnvs:     map[*sema.Func]map[*sema.Var]bool{},
		plainLambdas:  map[*sema.Closure]bool{},
		descs:         map[string]string{},
		ramps:         map[*sema.Func]*sema.Func{},
		closeThunks:   map[*sema.With]string{},
		descNames:     map[string]bool{},
		eqPtrFns:      map[string]string{},
		consts:        map[sema.ConstVal]string{},
	}
	g.typeDecls[strType] = "{ ptr, i64 }"
	g.typeOrder = append(g.typeOrder, strType)

	for _, fn := range prog.Funcs {
		if fn.Extern || fn.IsClosure {
			continue // a lambda is emitted when a value of it is made (plain.go)
		}
		g.function(fn)
		if fn.ExportC != "" {
			g.exportWrapper(fn)
		}
		g.flushPending()
	}
	g.globalsInit()
	g.entryPoint()
	g.flushPending()
	g.traitTables()
	g.flushPending()
	g.typeInfoGlobals()

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
		llt := g.llType(gl.Type)
		if a := g.overAlign[llt]; a > 0 {
			fmt.Fprintf(&sb, "@%s = global %s zeroinitializer, align %d\n", mangleGlobal(gl), llt, a) // @align(n) (D120)
			continue
		}
		fmt.Fprintf(&sb, "@%s = global %s zeroinitializer\n", mangleGlobal(gl), llt)
	}
	sb.WriteString("\n")
	sb.WriteString(runtimeDecls)
	sb.WriteString(mapDecls)
	sb.WriteString(gcDecls)
	sb.WriteString(coroDecls)
	sb.WriteString(g.descOut.String())
	sb.WriteString(g.constOut.String())
	declared := map[string]bool{}
	for _, m := range declRe.FindAllStringSubmatch(runtimeDecls+mapDecls+gcDecls+coroDecls, -1) {
		declared[m[1]] = true
	}
	for _, fn := range prog.Funcs {
		if fn.Extern && !declared[fn.Name] {
			declared[fn.Name] = true
			sb.WriteString(g.externDecl(fn))
		}
	}
	sb.WriteString("\n")
	out, helpers := g.out.String(), g.helpers.String()
	sb.WriteString(out)
	sb.WriteString(helpers)
	sb.WriteString(g.referencedHeld(out + helpers))
	return sb.String()
}

var declRe = regexp.MustCompile(`declare [^@]+@([A-Za-z0-9_.]+)[(]`)

func mangleGlobal(gl *sema.Global) string {
	return "g_" + strings.NewReplacer(".", "_", "/", "_").Replace(gl.Name)
}

const runtimeDecls = `declare void @veles_rt_init(i32, ptr)
declare void @veles_print(ptr, i64)
declare ptr @veles_alloc(i64)
declare void @veles_panic(ptr, i64)
declare void @veles_panic_at(ptr, i64, ptr, i64)
declare void @veles_test_fail(ptr, i64, ptr, i64, i64)
declare i64 @veles_test_enter(ptr, i64)
declare void @veles_test_leave(i64)
declare void @veles_cleanup_push(ptr, ptr, ptr)
declare void @veles_cleanup_pop(ptr)
declare i64 @veles_ffi_enter()
declare void @veles_ffi_leave(i64)
declare void @veles_blocking_enter()
declare void @veles_blocking_leave()
declare void @veles_gc_park()
@veles_stop_requested = external global i32
declare void @veles_report_error(ptr, i64)
declare void @veles_string_concat(ptr, ptr, i64, ptr, i64)
declare i1 @veles_string_eq(ptr, i64, ptr, i64)
declare i32 @veles_string_cmp(ptr, i64, ptr, i64)
declare i1 @veles_string_starts_with(ptr, i64, ptr, i64)
declare i1 @veles_string_ends_with(ptr, i64, ptr, i64)
declare i1 @veles_string_contains(ptr, i64, ptr, i64)
declare i1 @veles_string_substring(ptr, ptr, i64, i64, i64)
declare i1 @veles_string_to_int(ptr, i64, ptr)
declare i64 @veles_string_char_count(ptr, i64)
declare ptr @veles_string_chars(ptr, ptr, i64)
declare ptr @veles_string_split(ptr, ptr, i64, ptr, i64)
declare i8 @veles_string_byte_at(ptr, i64, i64)
declare ptr @veles_string_bytes(ptr, ptr, i64)
declare i1 @veles_bytes_decode_utf8(ptr, ptr)
declare i1 @veles_bytes_decode_utf8_range(ptr, ptr, i64, i64)
declare void @veles_i64_to_string(ptr, i64)
declare void @veles_u64_to_string(ptr, i64)
declare i64 @veles_i64_format(ptr, i64)
declare i64 @veles_u64_format(ptr, i64)
declare void @veles_string_concat_n(ptr, ptr, i64)
declare void @veles_list_append_bytes(ptr, ptr, i64)
declare void @veles_string_join(ptr, ptr, ptr, i64)
declare void @veles_list_sort_native(ptr, i32)
declare void @veles_f64_to_string(ptr, double)
declare void @veles_f32_to_string(ptr, float)
declare void @veles_bool_to_string(ptr, i1 zeroext)
declare ptr @veles_list_new(ptr, i64)
declare i64 @veles_list_len(ptr)
declare void @veles_list_push(ptr, ptr)
declare void @veles_list_reserve(ptr, i64)
declare ptr @veles_list_ref(ptr, i64)
declare void @veles_list_index_panic(ptr, i64, ptr, i64)
declare void @veles_array_index_panic(i64, i64, ptr, i64)
declare void @veles_unchecked_index_panic(i64, i64, i64, ptr, i64)
declare ptr @veles_secret_bytes(ptr, i64)
declare void @veles_secret_wipe(ptr)
declare void @veles_secret_text(ptr, ptr)
declare ptr @veles_secret_copy(ptr, ptr)
declare i1 @veles_list_pop(ptr, ptr)
declare ptr @veles_list_copy(ptr)
declare ptr @veles_list_slice(ptr, i64, i64)
declare void @veles_list_clear(ptr)
declare void @llvm.memcpy.p0.p0.i64(ptr, ptr, i64, i1)
declare void @llvm.memmove.p0.p0.i64(ptr, ptr, i64, i1)
declare void @llvm.memset.p0.i64(ptr, i8, i64, i1)
declare double @llvm.sqrt.f64(double)
declare double @llvm.fabs.f64(double)
declare double @llvm.floor.f64(double)
declare double @llvm.ceil.f64(double)
declare double @llvm.round.f64(double)
declare double @llvm.trunc.f64(double)
declare double @llvm.pow.f64(double, double)
declare double @llvm.minnum.f64(double, double)
declare double @llvm.maxnum.f64(double, double)
declare float @llvm.sqrt.f32(float)
declare float @llvm.fabs.f32(float)
declare float @llvm.floor.f32(float)
declare float @llvm.ceil.f32(float)
declare float @llvm.round.f32(float)
declare float @llvm.trunc.f32(float)
declare float @llvm.pow.f32(float, float)
declare float @llvm.minnum.f32(float, float)
declare float @llvm.maxnum.f32(float, float)
declare double @llvm.log.f64(double)
declare double @llvm.log2.f64(double)
declare double @llvm.log10.f64(double)
declare double @llvm.exp.f64(double)
declare double @llvm.sin.f64(double)
declare double @llvm.cos.f64(double)
declare float @llvm.log.f32(float)
declare float @llvm.log2.f32(float)
declare float @llvm.log10.f32(float)
declare float @llvm.exp.f32(float)
declare float @llvm.sin.f32(float)
declare float @llvm.cos.f32(float)
declare double @tan(double)
declare double @atan2(double, double)
declare double @hypot(double, double)
declare float @tanf(float)
declare float @atan2f(float, float)
declare float @hypotf(float, float)
declare i64 @veles_int_pow(i64, i64, i64, i1 zeroext, ptr, i64)
declare i8 @llvm.ctpop.i8(i8)
declare i16 @llvm.ctpop.i16(i16)
declare i32 @llvm.ctpop.i32(i32)
declare i64 @llvm.ctpop.i64(i64)
declare i8 @llvm.ctlz.i8(i8, i1)
declare i16 @llvm.ctlz.i16(i16, i1)
declare i32 @llvm.ctlz.i32(i32, i1)
declare i64 @llvm.ctlz.i64(i64, i1)
declare i8 @llvm.cttz.i8(i8, i1)
declare i16 @llvm.cttz.i16(i16, i1)
declare i32 @llvm.cttz.i32(i32, i1)
declare i64 @llvm.cttz.i64(i64, i1)
declare i8 @llvm.fshl.i8(i8, i8, i8)
declare i16 @llvm.fshl.i16(i16, i16, i16)
declare i32 @llvm.fshl.i32(i32, i32, i32)
declare i64 @llvm.fshl.i64(i64, i64, i64)
declare i8 @llvm.fshr.i8(i8, i8, i8)
declare i16 @llvm.fshr.i16(i16, i16, i16)
declare i32 @llvm.fshr.i32(i32, i32, i32)
declare i64 @llvm.fshr.i64(i64, i64, i64)
declare i16 @llvm.bswap.i16(i16)
declare i32 @llvm.bswap.i32(i32)
declare i64 @llvm.bswap.i64(i64)
declare i8 @llvm.bitreverse.i8(i8)
declare i16 @llvm.bitreverse.i16(i16)
declare i32 @llvm.bitreverse.i32(i32)
declare i64 @llvm.bitreverse.i64(i64)
declare double @llvm.copysign.f64(double, double)
declare float @llvm.copysign.f32(float, float)
declare i8 @llvm.sadd.sat.i8(i8, i8)
declare i16 @llvm.sadd.sat.i16(i16, i16)
declare i32 @llvm.sadd.sat.i32(i32, i32)
declare i64 @llvm.sadd.sat.i64(i64, i64)
declare i8 @llvm.ssub.sat.i8(i8, i8)
declare i16 @llvm.ssub.sat.i16(i16, i16)
declare i32 @llvm.ssub.sat.i32(i32, i32)
declare i64 @llvm.ssub.sat.i64(i64, i64)
declare i8 @llvm.uadd.sat.i8(i8, i8)
declare i16 @llvm.uadd.sat.i16(i16, i16)
declare i32 @llvm.uadd.sat.i32(i32, i32)
declare i64 @llvm.uadd.sat.i64(i64, i64)
declare i8 @llvm.usub.sat.i8(i8, i8)
declare i16 @llvm.usub.sat.i16(i16, i16)
declare i32 @llvm.usub.sat.i32(i32, i32)
declare i64 @llvm.usub.sat.i64(i64, i64)
declare i8 @llvm.abs.i8(i8, i1)
declare i16 @llvm.abs.i16(i16, i1)
declare i32 @llvm.abs.i32(i32, i1)
declare i64 @llvm.abs.i64(i64, i1)
declare i8 @llvm.smin.i8(i8, i8)
declare i16 @llvm.smin.i16(i16, i16)
declare i32 @llvm.smin.i32(i32, i32)
declare i64 @llvm.smin.i64(i64, i64)
declare i8 @llvm.smax.i8(i8, i8)
declare i16 @llvm.smax.i16(i16, i16)
declare i32 @llvm.smax.i32(i32, i32)
declare i64 @llvm.smax.i64(i64, i64)
declare i8 @llvm.umin.i8(i8, i8)
declare i16 @llvm.umin.i16(i16, i16)
declare i32 @llvm.umin.i32(i32, i32)
declare i64 @llvm.umin.i64(i64, i64)
declare i8 @llvm.umax.i8(i8, i8)
declare i16 @llvm.umax.i16(i16, i16)
declare i32 @llvm.umax.i32(i32, i32)
declare i64 @llvm.umax.i64(i64, i64)
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
declare i8 @llvm.fptosi.sat.i8.f64(double)
declare i16 @llvm.fptosi.sat.i16.f64(double)
declare i32 @llvm.fptosi.sat.i32.f64(double)
declare i64 @llvm.fptosi.sat.i64.f64(double)
declare i8 @llvm.fptoui.sat.i8.f64(double)
declare i16 @llvm.fptoui.sat.i16.f64(double)
declare i32 @llvm.fptoui.sat.i32.f64(double)
declare i64 @llvm.fptoui.sat.i64.f64(double)
declare i8 @llvm.fptosi.sat.i8.f32(float)
declare i16 @llvm.fptosi.sat.i16.f32(float)
declare i32 @llvm.fptosi.sat.i32.f32(float)
declare i64 @llvm.fptosi.sat.i64.f32(float)
declare i8 @llvm.fptoui.sat.i8.f32(float)
declare i16 @llvm.fptoui.sat.i16.f32(float)
declare i32 @llvm.fptoui.sat.i32.f32(float)
declare i64 @llvm.fptoui.sat.i64.f32(float)
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
	if a := g.overAlign[llt]; a > 0 {
		fmt.Fprintf(&g.allocas, "  %s = alloca %s, align %d\n", name, llt, a) // @align(n) (D120)
		return name
	}
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

// panicAt emits a panic with a constant message and the location `where`
// (from g.where; empty when there is none), and terminates.
func (g *gen) panicAt(msg, where string) {
	g.panicValueAt(g.stringConst(msg), where)
}

// panicValueAt emits a panic whose message is the string value msg.
func (g *gen) panicValueAt(msg, where string) {
	p, l := g.strPtrLen(msg)
	wp, wl := g.strPtrLen(g.stringConst(where))
	g.emit("call void @veles_panic_at(ptr %s, i64 %s, ptr %s, i64 %s)", p, l, wp, wl)
	g.emitTerm("unreachable")
}

// where is the `file:line:col` a panic at span reports (D64): relative to
// the package root with forward slashes, so a binary does not carry the
// build machine's directories; std files keep their `std/...` path.
func (g *gen) where(span source.Span) string {
	if span.File == nil {
		return ""
	}
	path := span.File.Path
	if !span.File.Embedded && g.prog.Root != "" && filepath.IsAbs(path) {
		if rel, err := filepath.Rel(g.prog.Root, path); err == nil && !strings.HasPrefix(rel, "..") {
			path = rel
		}
	}
	line, col := span.File.Position(span.Start)
	return fmt.Sprintf("%s:%d:%d", filepath.ToSlash(path), line, col)
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
	g.cleanups = nil
	g.loopCleanupDepth = map[*sema.Loop]int{}
	g.coro = nil
	g.scopeSlots = map[*sema.ScopeBlock]string{}
	g.abandonSlots = map[*sema.Builtin]string{}
	g.receivingSlots = map[*sema.Builtin]string{}
	g.cleanupRecs = map[sema.Expr]string{}
	g.inCleanup = 0
	g.launchSlots = map[*sema.Launch]string{}
}

// retLL returns the LLVM return type of a function: void when the result is
// written through an `sret` pointer (the memory class, mem.go).
func (g *gen) retLL(fn *sema.Func) string {
	if fn.Suspends {
		return "ptr" // coroutine handle
	}
	rt := g.fnRet(fn)
	if rt == nil || g.isMem(rt) {
		return "void"
	}
	return g.llType(rt)
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
	sig := g.cSignature(paramTypes(fn.Sig.Params), fn.Sig.Ret)
	var params []string
	if !sig.void && sig.ret.pass == cIndirect {
		params = append(params, sig.sretDecl())
	}
	for i, p := range fn.Sig.Params {
		if isCStruct(p.Type) {
			params = append(params, sig.params[i].decl(cArgExt(p.Type)))
			continue
		}
		params = append(params, g.externParamTypes(p.Type)...)
	}
	if fn.Sig.CVariadic {
		params = append(params, "...") // D123
	}
	ret := "void"
	if !types.IsUnit(fn.Sig.Ret) {
		ret = sig.retDecl(g.llType(fn.Sig.Ret), cExt(fn.Sig.Ret))
	}
	return fmt.Sprintf("declare %s @%s(%s)\n", ret, fn.Name, strings.Join(params, ", "))
}

// cVariadicCallee is how a call names a C-variadic function: LLVM needs its
// type written out, `i32 (ptr, ...) @printf`, to lay the extra arguments
// out by the platform's variadic convention (D123).
func (g *gen) cVariadicCallee(fn *sema.Func, ret string) string {
	var ps []string
	for _, p := range fn.Sig.Params {
		for _, t := range g.externParamTypes(p.Type) {
			ps = append(ps, strings.Fields(t)[0]) // the type, without its extension attribute
		}
	}
	ps = append(ps, "...")
	return fmt.Sprintf("%s (%s) @%s", ret, strings.Join(ps, ", "), fn.Name)
}

func paramTypes(ps []types.Param) []types.Type {
	var out []types.Type
	for _, p := range ps {
		out = append(out, p.Type)
	}
	return out
}

// externParamTypes lowers a parameter for the C ABI: strings become a
// (ptr, len) pair so no aggregate crosses the boundary by value.
func (g *gen) externParamTypes(t types.Type) []string {
	if types.IsString(t) {
		return []string{"ptr", "i64"}
	}
	return []string{g.llType(t) + cArgExt(t)}
}

// cExt is the attribute the C ABI puts on a small integer crossing it, as
// a prefix of the type: clang passes a bool or an unsigned char/short
// zero-extended and a signed one sign-extended, and the other side reads
// the whole register. Without it the bits above an i1 are undefined — a C
// function taking `bool` would see garbage.
func cExt(t types.Type) string {
	if st, ok := t.(*types.Struct); ok && st.Transparent && len(st.Fields) == 1 {
		return cExt(st.Fields[0].Type) // C sees the field (D120)
	}
	b, ok := t.(*types.Basic)
	if !ok {
		return ""
	}
	switch b.Kind {
	case types.Bool, types.U8, types.U16:
		return "zeroext "
	case types.I8, types.I16:
		return "signext "
	}
	return ""
}

// cArgExt is cExt as a suffix of an argument's type: `i1 zeroext %x`.
func cArgExt(t types.Type) string {
	if e := cExt(t); e != "" {
		return " " + strings.TrimSpace(e)
	}
	return ""
}

func (g *gen) function(fn *sema.Func) {
	source.SetWhere("generating code for", fn.Display, fn.Span)
	g.resetFn(fn)
	if fn.Sig.Effects.Throws {
		g.fnResult = g.prog.ResultType(fn.Sig.Ret, fn.Sig.Effects.Error)
	}
	var params []string
	var prologue []string
	if rt := g.fnRet(fn); !fn.Suspends && rt != nil && g.isMem(rt) {
		params = append(params, g.sretParam(rt)+" %sret") // the result is written here
	}
	bind := func(v *sema.Var, i int) {
		llt := g.vt(v.Type)
		params = append(params, fmt.Sprintf("%s %%p%d", llt, i))
		if g.isMem(v.Type) {
			// a memory-class argument is the address of a copy made for this call:
			// the variable's own storage, unless it is a heap cell or the function
			// is a coroutine, whose frame keeps its own
			size := g.memSize(v.Type)
			move := func(dst string) {
				prologue = append(prologue, fmt.Sprintf("  call void @llvm.memmove.p0.p0.i64(ptr %s, ptr %%p%d, i64 %d, i1 false)", dst, i, size))
			}
			switch {
			case v.AddrTaken:
				slot := g.varStorage(v, false)
				prologue = append(prologue, fmt.Sprintf("  %%cell%d = call ptr @veles_gc_alloc(ptr %s, i64 %d)", i, g.descOf(v.Type), size))
				prologue = append(prologue, fmt.Sprintf("  store ptr %%cell%d, ptr %s", i, slot))
				move(fmt.Sprintf("%%cell%d", i))
			case fn.Suspends:
				move(g.varStorage(v, true))
			default:
				g.storage[v] = fmt.Sprintf("%%p%d", i)
			}
			return
		}
		if v.AddrTaken {
			// captured or address-taken parameter: copy into a heap cell
			size, _ := g.layout(v.Type)
			slot := g.varStorage(v, false)
			prologue = append(prologue, fmt.Sprintf("  %%cell%d = call ptr @veles_gc_alloc(ptr %s, i64 %d)", i, g.descOf(v.Type), size))
			prologue = append(prologue, fmt.Sprintf("  store ptr %%cell%d, ptr %s", i, slot))
			prologue = append(prologue, fmt.Sprintf("  store %s %%p%d, ptr %%cell%d", llt, i, i))
			return
		}
		st := g.varStorage(v, true)
		prologue = append(prologue, fmt.Sprintf("  store %s %%p%d, ptr %s", llt, i, st))
	}
	if fn.Suspends {
		params = append(params, "ptr %task")
	}
	if fn.IsClosure {
		params = append(params, "ptr %env")
		g.envSlot = g.alloca("ptr")
		prologue = append(prologue, fmt.Sprintf("  store ptr %%env, ptr %s", g.envSlot))
	}
	if fn.Receiver != nil {
		bind(fn.Receiver, 0)
	}
	for i, p := range fn.Params {
		bind(p, i+1)
	}
	if fn.CallerLoc {
		params = append(params, strType+" %callerloc") // D88: the site this call was made from
	}
	if fn.Suspends {
		g.coroPrologue()
	}
	if fn.Body != nil {
		g.block(fn.Body)
	}
	if !g.term {
		// falling off the end of a unit function
		if fn.Sig.Effects.Throws || fn.Suspends {
			g.retValue(nil)
		} else if types.IsUnit(fn.Sig.Ret) {
			g.emitTerm("ret void")
		} else {
			g.emitTerm("unreachable")
		}
	}
	attrs := ""
	if fn.Suspends {
		g.coroEpilogue()
		attrs = " presplitcoroutine"
	}
	if fn.Inline > 0 && !fn.Suspends {
		attrs += " alwaysinline"
	} else if fn.Inline < 0 {
		attrs += " noinline"
	}
	out := &g.out
	var held strings.Builder
	if fn.Conditional && fn.Plain == nil && !fn.IsClosure && fn.ExportC == "" {
		// the coroutine copy of a conditional function (D116): kept only if
		// something calls it, since every plain call uses `.plain`
		out = &held
	}
	fmt.Fprintf(out, "define %s @%s(%s)%s {\nentry:\n", g.retLL(fn), fn.Name, strings.Join(params, ", "), attrs)
	out.WriteString(g.allocas.String())
	for _, p := range prologue {
		out.WriteString(p + "\n")
	}
	out.WriteString(g.body.String())
	out.WriteString("}\n\n")
	if out == &held {
		g.held = append(g.held, heldFn{fn.Name, held.String()})
	}
}

// heldFn is a function's text kept out of the output until something is
// found to refer to it (plain.go).
type heldFn struct {
	name, text string
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
	if v.Captured {
		// the environment holds the address of the shared heap cell
		env := g.newTmp()
		g.emit("%s = load ptr, ptr %s", env, g.envSlot)
		slot := g.newTmp()
		g.emit("%s = getelementptr ptr, ptr %s, i64 %d", slot, env, v.CapIndex)
		cell := g.newTmp()
		g.emit("%s = load ptr, ptr %s", cell, slot)
		return cell
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
	if v.AddrTaken && !v.IsGlobal && !v.Captured {
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
	if v.AddrTaken && !v.IsGlobal && !v.Captured {
		cell := g.gcAlloc(v.Type)
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
				g.storeVal(s.Var.Type, v, st)
			}
		} else {
			st := g.declareVar(s.Var)
			if g.isMem(s.Var.Type) {
				g.emit("call void @llvm.memset.p0.i64(ptr %s, i8 0, i64 %d, i1 false)", st, g.memSize(s.Var.Type))
			} else {
				g.emit("store %s zeroinitializer, ptr %s", g.llType(s.Var.Type), st)
			}
		}
	case *sema.Assign:
		v := g.expr(s.Value)
		if types.IsNever(s.Value.Type()) {
			return
		}
		g.assign(s.Target, v)
	case *sema.ExprStmt:
		g.expr(s.X)
	case *sema.Return:
		g.retValue(s.Value)
	case *sema.Loop:
		g.loop(s)
	case *sema.Break:
		g.runCleanups(g.loopCleanupDepth[s.Loop])
		g.emitTerm("br label %%%s", g.loops[s.Loop].end)
	case *sema.Continue:
		g.runCleanups(g.loopCleanupDepth[s.Loop])
		g.emitTerm("br label %%%s", g.loops[s.Loop].post)
	case *sema.ScopeBlock:
		g.scopeBlock(s)
	case *sema.With:
		v := g.expr(s.Init)
		st := g.declareVar(s.Var)
		g.storeVal(s.Var.Type, v, st)
		// registered with the task as well, so that a panic inside the
		// body closes the resource before the task is abandoned (D49)
		g.pushCleanup(s.Close, "@"+g.closeThunk(s), st)
		g.block(s.Body)
		g.cleanups = g.cleanups[:len(g.cleanups)-1]
		if !g.term {
			g.popCleanup(s.Close)
			g.expr(s.Close)
		}
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
	g.runCleanups(0)
	if fn.Sig.Effects.Throws {
		rs := g.fnResult.(*types.Sealed)
		if value == nil || types.IsUnit(fn.Sig.Ret) {
			v = "zeroinitializer"
		}
		payload := g.buildStruct(rs.Variants[0], []string{v})
		r := g.makeTaggedT(rs, 0, rs.Variants[0], payload)
		g.emitRet(rs, r, true)
		return
	}
	if value == nil || types.IsUnit(fn.Sig.Ret) {
		g.emitRet(nil, "", false)
		return
	}
	g.emitRet(fn.Sig.Ret, v, false)
}

func (g *gen) loop(l *sema.Loop) {
	labels := loopLabels{cond: g.newLabel("loop.cond"), post: g.newLabel("loop.post"), end: g.newLabel("loop.end")}
	body := g.newLabel("loop.body")
	g.loops[l] = labels
	g.loopCleanupDepth[l] = len(g.cleanups)
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
	g.safepointPoll()
	g.emitTerm("br label %%%s", labels.cond)
	g.placeLabel(labels.end)
}

// safepointPoll is a loop's back edge (D66): one load and a branch that is
// never taken unless a collection is waiting for this thread, so a loop
// that allocates nothing cannot hold the other threads stopped.
func (g *gen) safepointPoll() {
	flag := g.newTmp()
	g.emit("%s = load volatile i32, ptr @veles_stop_requested, align 4", flag)
	stop := g.newTmp()
	g.emit("%s = icmp ne i32 %s, 0", stop, flag)
	park, on := g.newLabel("safepoint"), g.newLabel("safepoint.on")
	g.emitTerm("br i1 %s, label %%%s, label %%%s, !prof !{!\"branch_weights\", i32 1, i32 100000}", stop, park, on)
	g.placeLabel(park)
	g.emit("call void @veles_gc_park()")
	g.emitTerm("br label %%%s", on)
	g.placeLabel(on)
}

// ---------------------------------------------------------------------------
// globals and entry point

func (g *gen) globalsInit() {
	g.resetFn(&sema.Func{Name: "veles_init_globals", Sig: &types.Func{Ret: types.TUnit}})
	for _, gl := range g.prog.Globals {
		if len(g.pointerOffsets(gl.Type)) > 0 {
			g.emit("call void @veles_gc_root(ptr @%s, ptr %s)", mangleGlobal(gl), g.descOf(gl.Type))
		}
	}
	for _, gl := range g.prog.Globals {
		if gl.Init == nil {
			continue
		}
		v := g.expr(gl.Init)
		g.storeVal(gl.Type, v, "@"+mangleGlobal(gl))
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
	if g.prog.TestMode {
		g.testRunner()
	} else if main == nil {
		g.emitTerm("ret i32 0")
	} else if main.Sig.Effects.Throws {
		rs := g.prog.ResultType(main.Sig.Ret, main.Sig.Effects.Error).(*types.Sealed)
		r := g.newTmp()
		if main.Suspends {
			root := g.runRoot(main)
			rp := g.newTmp()
			g.emit("%s = call ptr @veles_task_result(ptr %s)", rp, root)
			if g.isMem(rs) {
				r = rp
			} else {
				g.emit("%s = load %s, ptr %s", r, g.llType(rs), rp)
			}
		} else if g.isMem(rs) {
			r = g.callRet(rs, "@"+main.Name, nil)
		} else {
			g.emit("%s = call %s @%s()", r, g.llType(rs), main.Name)
		}
		tag := g.tagOf(rs, r)
		isErr := g.newTmp()
		g.emit("%s = icmp eq i32 %s, 1", isErr, tag)
		errL, okL := g.newLabel("main.err"), g.newLabel("main.ok")
		g.emitTerm("br i1 %s, label %%%s, label %%%s", isErr, errL, okL)
		g.placeLabel(errL)
		errVariant := rs.Variants[1]
		payload := g.extractTaggedT(rs, r, errVariant)
		errVal := g.part(errVariant, payload, 0, errVariant.Fields[0].Type)
		errT := errVariant.Fields[0].Type
		var msg string
		if rep := g.prog.MainReport; rep != nil {
			msg = g.newTmp()
			g.emit("%s = call %s @%s(%s %s)", msg, g.llType(types.TString), rep.Name, g.llType(errT), errVal)
		} else {
			msg = g.show(errT, errVal)
		}
		p, l := g.strPtrLen(msg)
		g.emit("call void @veles_report_error(ptr %s, i64 %s)", p, l)
		g.emitTerm("ret i32 1")
		g.placeLabel(okL)
		g.emitTerm("ret i32 0")
	} else if main.Suspends {
		g.runRoot(main)
		g.emitTerm("ret i32 0")
	} else {
		if !g.prog.Release {
			g.emit("call void @veles_call_base(ptr null, ptr %s)", g.chainRecord("", main))
		}
		g.emit("call void @%s()", main.Name)
		g.emitTerm("ret i32 0")
	}
	if g.prog.TestMode {
		g.out.WriteString("declare void @veles_test_setup(i64, i64, i64)\ndeclare ptr @veles_test_new(ptr, i64)\ndeclare ptr @veles_test_bind(ptr)\ndeclare void @veles_test_unbind(ptr)\ndeclare void @veles_test_queue(ptr, ptr, ptr)\ndeclare i64 @veles_test_take(ptr, ptr, ptr, i64)\n\n")
	}
	g.out.WriteString("define i32 @veles_main(i32 %argc, ptr %argv) {\nentry:\n")
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

// pushCleanup registers a cleanup with the running task — fn(env) is what
// a panic calls to run it — and makes it the innermost of g.cleanups. The
// list entry lives in the function's frame (the coroutine frame, for a
// suspending function), so entering a `with` costs no allocation.
func (g *gen) pushCleanup(c sema.Expr, fn, env string) {
	rec := g.alloca("{ ptr, ptr, ptr }")
	g.cleanupRecs[c] = rec
	g.emit("call void @veles_cleanup_push(ptr %s, ptr %s, ptr %s)", rec, fn, env)
	g.cleanups = append(g.cleanups, c)
}

// popCleanup unregisters c, the innermost cleanup, from the task.
func (g *gen) popCleanup(c sema.Expr) {
	g.emit("call void @veles_cleanup_pop(ptr %s)", g.cleanupRecs[c])
}

// runCleanups emits the close calls of every `with` entered since depth,
// innermost first, without popping them (the code after the jump still
// belongs to those blocks).
func (g *gen) runCleanups(depth int) {
	saved := g.cleanups
	g.inCleanup++
	for i := len(saved) - 1; i >= depth; i-- {
		g.cleanups = saved[:i]
		g.popCleanup(saved[i]) // this path runs it itself
		g.expr(saved[i])
	}
	g.inCleanup--
	g.cleanups = saved
}

// closeThunk defines (once per `with`) the function a panic calls to close
// the binding: it receives the address of the resource and runs the same
// close call the normal exit runs.
func (g *gen) closeThunk(s *sema.With) string {
	if name, ok := g.closeThunks[s]; ok {
		return name
	}
	g.thunkSeq++
	name := fmt.Sprintf("with.close.%d", g.thunkSeq)
	g.closeThunks[s] = name
	v := s.Var
	g.pending = append(g.pending, func() {
		g.defineHelperEx(name, "void", []string{"ptr %env"}, "", func() {
			if v.AddrTaken && !v.IsGlobal && !v.Captured {
				// varPtr loads the heap cell's address from the slot
				slot := g.alloca("ptr")
				g.emit("store ptr %%env, ptr %s", slot)
				g.storage[v] = slot
			} else {
				g.storage[v] = "%env"
			}
			g.expr(s.Close)
			g.emitTerm("ret void")
		})
	})
	return name
}

// recvOperand renders v (a value of type t) as the receiver argument of
// fn: methods take a pointer to the receiver's place (D22 v0.30), so a
// value is spilled — to the stack when the method keeps no pointer to its
// receiver, to the heap when it might (sema's receiver pass decides).
func (g *gen) recvOperand(fn *sema.Func, t types.Type, v string) string {
	llt := g.llType(t)
	if fn.Receiver == nil {
		return g.vt(t) + " " + v
	}
	var cell string
	if fn.SelfEscapes {
		cell = g.gcAlloc(t)
	} else {
		cell = g.alloca(llt)
	}
	g.storeVal(t, v, cell)
	return "ptr " + cell
}

// exportWrapper emits the function C calls for an `extern "C" fun` (D69):
// under the C name, with the C calling convention, it marks the runtime as
// inside a callback — a panic there must end the process rather than
// unwind through C's frames — and calls the Veles body.
func (g *gen) exportWrapper(fn *sema.Func) {
	var pts []types.Type
	for _, p := range fn.Params {
		pts = append(pts, p.Type)
	}
	sig := g.cSignature(pts, fn.Sig.Ret)
	var params, args []string
	var body strings.Builder
	if !sig.void && sig.ret.pass == cIndirect {
		params = append(params, sig.sretDecl()+" %sret")
	}
	for i, p := range fn.Params {
		llt := g.llType(p.Type)
		cv := sig.params[i]
		params = append(params, fmt.Sprintf("%s %%a%d", cv.decl(cArgExt(p.Type)), i))
		switch cv.pass {
		case cCoerce:
			// what C passed, back to the struct through memory
			fmt.Fprintf(&body, "  %%c%d = alloca %s\n  store %s %%a%d, ptr %%c%d\n  %%v%d = load %s, ptr %%c%d\n", i, cv.ty, cv.ty, i, i, i, llt, i)
			args = append(args, fmt.Sprintf("%s %%v%d", llt, i))
		case cIndirect, cByval:
			if g.isMem(p.Type) {
				args = append(args, fmt.Sprintf("ptr %%a%d", i)) // the copy C made is the callee's
				break
			}
			fmt.Fprintf(&body, "  %%v%d = load %s, ptr %%a%d\n", i, llt, i)
			args = append(args, fmt.Sprintf("%s %%v%d", llt, i))
		default:
			args = append(args, fmt.Sprintf("%s %%a%d", llt, i))
		}
	}
	ret := g.retLL(fn)
	retDecl := "void"
	if ret != "void" {
		retDecl = sig.retDecl(ret, cExt(fn.Sig.Ret))
	}
	fmt.Fprintf(&g.out, "define %s @%s(%s) {\nentry:\n", retDecl, fn.ExportC, strings.Join(params, ", "))
	g.out.WriteString(body.String())
	g.out.WriteString("  %saved = call i64 @veles_ffi_enter()\n")
	if ret == "void" {
		if rt := g.fnRet(fn); rt != nil && g.isMem(rt) {
			args = append([]string{g.sretParam(rt) + " %sret"}, args...) // C's own sret pointer is where it is written
		}
		fmt.Fprintf(&g.out, "  call void @%s(%s)\n  call void @veles_ffi_leave(i64 %%saved)\n  ret void\n}\n\n", fn.Name, strings.Join(args, ", "))
		return
	}
	fmt.Fprintf(&g.out, "  %%r = call %s @%s(%s)\n  call void @veles_ffi_leave(i64 %%saved)\n", ret, fn.Name, strings.Join(args, ", "))
	switch sig.ret.pass {
	case cIndirect:
		fmt.Fprintf(&g.out, "  store %s %%r, ptr %%sret\n  ret void\n}\n\n", ret)
	case cCoerce:
		fmt.Fprintf(&g.out, "  %%rc = alloca %s\n  store %s %%r, ptr %%rc\n  %%rv = load %s, ptr %%rc\n  ret %s %%rv\n}\n\n", sig.ret.ty, ret, sig.ret.ty, sig.ret.ty)
	default:
		fmt.Fprintf(&g.out, "  ret %s %%r\n}\n\n", ret)
	}
}

// callerLocArg is the constant or value a call of a `@caller_location`
// function passes as its site (D88): where the call is written, or — from
// inside another such function — the site that one was itself given, so a
// misuse reaches the first caller that is not std's own. A call the compiler
// made has no site: the empty string, which the callee answers with its own.
func (g *gen) callerLocArg(span source.Span) string {
	if g.fn != nil && g.fn.CallerLoc {
		return "%callerloc"
	}
	if span.File == nil {
		return g.stringConst("")
	}
	return g.stringConst(g.where(span))
}

// panicAtCaller emits a panic in a `@caller_location` function: it reports
// the site the function was called from (D88), or its own `where` when it
// was given none.
func (g *gen) panicAtCaller(msg, where string) {
	p, l := g.strPtrLen(msg)
	own := g.stringConst(where)
	op, ol := g.strPtrLen(own)
	cp, cl := g.strPtrLen("%callerloc")
	empty := g.newTmp()
	g.emit("%s = icmp eq i64 %s, 0", empty, cl)
	wp, wl := g.newTmp(), g.newTmp()
	g.emit("%s = select i1 %s, ptr %s, ptr %s", wp, empty, op, cp)
	g.emit("%s = select i1 %s, i64 %s, i64 %s", wl, empty, ol, cl)
	g.emit("call void @veles_panic_at(ptr %s, i64 %s, ptr %s, i64 %s)", p, l, wp, wl)
	g.emitTerm("unreachable")
}
