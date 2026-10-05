package sema

import (
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// `const fun` (D113 part 3): a function the compiler may run while it
// evaluates a constant. It is an ordinary function at run time. What it may
// contain is checked at its declaration (checkConstFuns), so an edit inside it
// cannot silently break a constant elsewhere; what the evaluator runs is the
// same HIR the code generator compiles, so a result equals the run-time one.

// CPtr is a pointer while a constant is evaluated: a method's receiver, `&x`,
// an element's place. It is never part of a result (a pointer is not a
// constant, notConstType).
type CPtr struct {
	T   types.Type
	get func() ConstVal
	set func(ConstVal)
}

func (v *CPtr) Type() types.Type { return v.T }

// constBuiltins are the built-in operations the evaluator runs: collection
// and string operations over values it holds. What is left out reaches the
// outside (tasks, I/O, raw memory, atomics, the test runner) or the layout of
// memory.
var constBuiltins = map[string]bool{
	"string.len": true, "string.byteAt": true, "string.substring": true, "string.bytes": true,
	"string.chars": true, "string.charCount": true, "string.startsWith": true, "string.endsWith": true,
	"string.contains": true, "string.split": true, "string.toInt": true,
	"list.len": true, "list.get": true, "list.ref": true, "list.push": true, "list.pop": true,
	"list.clear": true, "list.copy": true, "list.addAll": true, "list.slice": true, "list.reserve": true,
	"list.touch": true, "list.mods": true, "list.checkMods": true, "list.appendText": true,
	"list.joinText": true, "list.decodeUtf8": true, "list.decodeUtf8Range": true,
	"list.toArray": true, "array.toList": true, "list.sortedNative": true,
	"map.len": true, "map.get": true, "map.contains": true, "map.set": true, "map.remove": true,
	"map.clear": true, "map.reserve": true, "map.copy": true, "map.keys": true, "map.values": true,
	"map.entries": true, "map.mods": true, "map.checkMods": true,
	"set.new": true, "set.add": true,
	"num.toChecked": true, "panic": true,
}

// runsInCompiler reports whether the evaluator may run fn: a `const fun`, or
// the `init` block of a struct, which building the struct runs (D28) and which
// is held to the same rules.
func runsInCompiler(fn *Func) bool {
	return fn.Const || fn.tmpl != nil && fn.tmpl.Name == "$init"
}

// constExterns are the runtime's own entry points the evaluator runs in Go,
// with the runtime's results: std's const functions may call them (and are
// the only code that can name them).
var constExterns = map[string]bool{"veles_string_find": true}

// checkConstFuns checks every `const fun` of the round against the rules of
// D113; each is reported at its declaration.
func (c *Checker) checkConstFuns() {
	for _, fn := range c.funcs {
		if fn.Const {
			c.validateConstFun(fn)
		}
	}
}

// validateConstFun reports, once per function, what a `const fun` may not do,
// and says whether it is fit to run.
func (c *Checker) validateConstFun(fn *Func) bool {
	if ok, done := c.constFunOK[fn]; done {
		return ok
	}
	if c.constFunOK == nil {
		c.constFunOK = map[*Func]bool{}
	}
	c.constFunOK[fn] = false // a recursive function is being validated: it stands
	if fn.Body == nil {
		return false // its own errors are reported where its body is checked
	}
	ok := true
	bad := func(span source.Span, format string, args ...any) {
		ok = false
		if span.File == nil {
			span = fn.Span
		}
		c.errorf(span, "in 'const fun "+fn.Display+"': "+format, args...)
	}
	if fn.Sig.Effects.Throws {
		bad(fn.Span, "a 'const fun' cannot throw yet: a failure in the compiler is a 'panic' (D113)")
	}
	std := fn.tmpl != nil && fn.tmpl.Module != nil && fn.tmpl.Module.Std // std vouches for its own `unsafe`: it calls the runtime entry points constExterns names
	if !std && (fn.tmpl != nil && fn.tmpl.Decl != nil && fn.tmpl.Decl.Unsafe || fn.UsesUnsafe) {
		bad(fn.Span, "a 'const fun' cannot use 'unsafe': the compiler cannot run raw memory or foreign code (D113)")
	}
	if fn.tmpl != nil && fn.tmpl.Extern {
		bad(fn.Span, "an extern function is foreign code, not a 'const fun'")
	}
	fnValue := false // a function value is reported once per function
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Call:
			if n.Fn.Extern && constExterns[n.Fn.Name] {
				break
			}
			if !runsInCompiler(n.Fn) {
				bad(n.Span, "it calls '%s', which is not a 'const fun'; mark that function 'const fun' too if the compiler can run it (D113)", n.Fn.Display)
			}
		case *CallIndirect, *CallVirtual, *Closure, *Box:
			if !fnValue {
				fnValue = true
				bad(spanOf(n.(Expr)), "%s is not supported in a 'const fun' yet: a function value or a trait object cannot be run by the compiler (D113)", describeNode(n.(Expr)))
			}
		case *Launch, *AwaitTask, *ScopeBlock, *Race:
			bad(source.Span{}, "a task cannot run in the compiler: a 'const fun' does not suspend or use 'async' (D113)")
		case *With:
			bad(source.Span{}, "'with' closes a resource, which a 'const fun' does not hold (D113)")
		case *Throw, *Try, *MakeResult:
			bad(source.Span{}, "a 'const fun' cannot throw yet: a failure in the compiler is a 'panic' (D113)")
		case *VarRef:
			if n.Var.IsGlobal {
				bad(source.Span{}, "it reads the module-level '%s', which is computed at run time; a 'const fun' reads its parameters and 'const's only (D113)", n.Var.Name)
			}
		case *Builtin:
			if !constBuiltins[n.Op] {
				bad(n.Span, "'%s' is not available to the compiler: a 'const fun' does no I/O, no tasks, no raw memory and no atomics (D113)", n.Op)
			}
		}
	})
	c.constFunOK[fn] = ok
	return ok
}

// ensureConstBody checks the body of a function the evaluator is about to run
// when the checker has not reached it yet.
func (c *Checker) ensureConstBody(fn *Func) {
	if fn.checked || fn.Body != nil || fn.tmpl == nil || fn.Extern {
		return
	}
	if c.constChecking[fn] {
		return
	}
	if c.constChecking == nil {
		c.constChecking = map[*Func]bool{}
	}
	c.constChecking[fn] = true
	c.checkBody(fn)
	fn.checked = true
	delete(c.constChecking, fn)
}

// call runs a `const fun`.
func (ev *constEval) call(e *Call) ConstVal {
	fn := e.Fn
	if ev.quiet {
		panic(constFailed{})
	}
	if fn.Extern && constExterns[fn.Name] {
		return ev.extern(e)
	}
	if !runsInCompiler(fn) {
		ev.errorf(e.Span, "a constant cannot call '%s': it is not a 'const fun'; mark it 'const fun' if the compiler can run it, or compute the value at run time with 'val' (D113)", fn.Display)
	}
	args := make([]ConstVal, len(e.Args))
	for i, a := range e.Args {
		args[i] = ev.val(a)
	}
	return ev.invoke(fn, args, e.Span)
}

// invoke runs a `const fun` on argument values (a method's receiver first, as
// a pointer).
func (ev *constEval) invoke(fn *Func, args []ConstVal, at source.Span) ConstVal {
	ev.c.ensureConstBody(fn)
	if fn.Body == nil || ev.c.constChecking[fn] {
		ev.errorf(at, "'%s' is called by a constant while its own body is still being checked; a constant it reads depends on it (D113)", fn.Display)
	}
	if !ev.c.validateConstFun(fn) {
		panic(constFailed{}) // reported at its declaration
	}
	if len(ev.stack) >= constMaxDepth {
		ev.errorf(at, "'%s' recursed more than %d calls deep while the compiler ran it (D113)", fn.Display, constMaxDepth)
	}
	saved := ev.env
	ev.env = map[*Var]*cell{}
	if fn.Receiver != nil && len(args) > 0 {
		ev.bind(fn.Receiver, args[0])
		args = args[1:]
	}
	for i, p := range fn.Params {
		if i < len(args) {
			ev.bind(p, args[i])
		}
	}
	ev.stack = append(ev.stack, constFrame{fn, at})
	c := ev.runBlock(fn.Body)
	ev.stack = ev.stack[:len(ev.stack)-1]
	ev.env = saved
	if c.kind == ctlReturn {
		return c.val
	}
	return &CUnit{}
}

// extern runs a runtime entry point in Go.
func (ev *constEval) extern(e *Call) ConstVal {
	switch e.Fn.Name {
	case "veles_string_find": // (s, part, from): the first index at or after from, or -1
		s, part := ev.expr(e.Args[0]).(*CString).V, ev.expr(e.Args[1]).(*CString).V
		from := ev.expr(e.Args[2]).(*CInt).V
		start := clampIndex(from, len(s)+1) // a start past the end finds nothing, except the empty text at the end itself
		if from.Cmp(big.NewInt(int64(len(s)))) > 0 {
			return i64v(-1)
		}
		if part == "" {
			return i64v(int64(start))
		}
		if i := strings.Index(s[start:], part); i >= 0 {
			return i64v(int64(start + i))
		}
		return i64v(-1)
	}
	ev.errorf(e.Span, "'%s' is not available to the compiler (D113)", e.Fn.Display)
	return nil
}

// ---------------------------------------------------------------------------
// the built-in operations

func (ev *constEval) intArg(e *Builtin, i int) *big.Int {
	v, ok := ev.expr(e.Args[i]).(*CInt)
	if !ok {
		ev.errorf(e.Span, "%s is not a constant expression (D113)", describeNode(e))
	}
	return v.V
}

func i64v(n int64) ConstVal { return &CInt{types.TI64, big.NewInt(n)} }

func (ev *constEval) elems(e *Builtin, v ConstVal) []ConstVal {
	switch l := v.(type) {
	case *CList:
		return l.Elems
	case *CArray:
		return l.Elems
	}
	ev.errorf(e.Span, "%s is not a constant expression (D113)", describeNode(e))
	return nil
}

func (ev *constEval) mutList(e *Builtin) *CList {
	l, ok := ev.expr(e.Args[0]).(*CList)
	if !ok {
		ev.errorf(e.Span, "%s is not a constant expression (D113)", describeNode(e))
	}
	return l
}

func (ev *constEval) str(e *Builtin, i int) string {
	s, ok := ev.expr(e.Args[i]).(*CString)
	if !ok {
		ev.errorf(e.Span, "%s is not a constant expression (D113)", describeNode(e))
	}
	return s.V
}

func isBoundary(s string, i int) bool {
	return i == 0 || i == len(s) || s[i]&0xC0 != 0x80
}

// builtinMore runs the operations a `const fun` uses beyond what a constant
// expression needs; the second result says the operation is not one of them.
func (ev *constEval) builtinMore(e *Builtin) (ConstVal, bool) {
	switch e.Op {
	case "panic":
		ev.errorf(e.Span, "panic in a constant: %s", ev.str(e, 0))
	case "string.byteAt":
		s, i := ev.str(e, 0), ev.intArg(e, 1)
		if i.Sign() < 0 || i.Cmp(big.NewInt(int64(len(s)))) >= 0 {
			ev.errorf(e.Span, "byte index %s is out of range for a string of length %d (D113)", i, len(s))
		}
		return &CInt{types.TU8, big.NewInt(int64(s[i.Int64()]))}, true
	case "string.substring":
		s, lo, hi := ev.str(e, 0), ev.intArg(e, 1), ev.intArg(e, 2)
		nt := e.Type().(*types.Nullable)
		if lo.Sign() < 0 || hi.Cmp(big.NewInt(int64(len(s)))) > 0 || lo.Cmp(hi) > 0 {
			return &CNull{nt}, true
		}
		l, h := int(lo.Int64()), int(hi.Int64())
		if !isBoundary(s, l) || !isBoundary(s, h) {
			return &CNull{nt}, true
		}
		return &CSome{nt, &CString{s[l:h]}}, true
	case "string.bytes":
		s := ev.str(e, 0)
		out := &CList{T: e.Type().(*types.List)}
		for i := 0; i < len(s); i++ {
			out.Elems = append(out.Elems, &CInt{types.TU8, big.NewInt(int64(s[i]))})
		}
		return out, true
	case "string.chars":
		s := ev.str(e, 0)
		out := &CList{T: e.Type().(*types.List)}
		for i := 0; i < len(s); {
			j := i + 1
			for j < len(s) && s[j]&0xC0 == 0x80 {
				j++
			}
			out.Elems = append(out.Elems, &CString{s[i:j]})
			i = j
		}
		return out, true
	case "string.charCount":
		s := ev.str(e, 0)
		n := 0
		for i := 0; i < len(s); i++ {
			if s[i]&0xC0 != 0x80 {
				n++
			}
		}
		return i64v(int64(n)), true
	case "string.startsWith":
		return &CBool{strings.HasPrefix(ev.str(e, 0), ev.str(e, 1))}, true
	case "string.endsWith":
		return &CBool{strings.HasSuffix(ev.str(e, 0), ev.str(e, 1))}, true
	case "string.contains":
		return &CBool{strings.Contains(ev.str(e, 0), ev.str(e, 1))}, true
	case "string.split":
		s, sep := ev.str(e, 0), ev.str(e, 1)
		out := &CList{T: e.Type().(*types.List)}
		for _, p := range strings.Split(s, sep) {
			out.Elems = append(out.Elems, &CString{p})
		}
		return out, true
	case "string.toInt":
		s := ev.str(e, 0)
		nt := e.Type().(*types.Nullable)
		// the runtime reads an optional sign and decimal digits only, in 20 bytes at most
		if len(s) == 0 || len(s) > 20 || strings.ContainsAny(s[1:], "+-") || s == "-" || s == "+" {
			return &CNull{nt}, true
		}
		for i := 0; i < len(s); i++ {
			if (s[i] < '0' || s[i] > '9') && !(i == 0 && (s[i] == '-' || s[i] == '+')) {
				return &CNull{nt}, true
			}
		}
		n, err := strconv.ParseInt(strings.TrimPrefix(s, "+"), 10, 64)
		if err != nil {
			return &CNull{nt}, true
		}
		return &CSome{nt, i64v(n)}, true
	case "list.ref":
		return ev.elemPlace(e).get(), true
	case "list.push":
		l := ev.mutList(e)
		l.Elems = append(l.Elems, ev.val(e.Args[1]))
		return &CUnit{}, true
	case "list.pop":
		l := ev.mutList(e)
		nt := e.Type().(*types.Nullable)
		if len(l.Elems) == 0 {
			return &CNull{nt}, true
		}
		last := l.Elems[len(l.Elems)-1]
		l.Elems = l.Elems[:len(l.Elems)-1]
		return &CSome{nt, last}, true
	case "list.clear":
		ev.mutList(e).Elems = nil
		return &CUnit{}, true
	case "list.copy":
		src := ev.elems(e, ev.expr(e.Args[0]))
		out := &CList{T: e.Type().(*types.List), Elems: make([]ConstVal, len(src))}
		for i, x := range src {
			out.Elems[i] = copyVal(x)
		}
		return out, true
	case "list.addAll":
		l := ev.mutList(e)
		for _, x := range ev.elems(e, ev.expr(e.Args[1])) {
			l.Elems = append(l.Elems, copyVal(x))
		}
		return &CUnit{}, true
	case "list.slice":
		src := ev.elems(e, ev.expr(e.Args[0]))
		from, to := ev.intArg(e, 1), ev.intArg(e, 2)
		lo, hi := clampIndex(from, len(src)), clampIndex(to, len(src))
		if lo > hi {
			lo = hi
		}
		out := &CList{T: e.Type().(*types.List), Elems: make([]ConstVal, hi-lo)}
		for i := lo; i < hi; i++ {
			out.Elems[i-lo] = copyVal(src[i])
		}
		return out, true
	case "list.reserve":
		ev.expr(e.Args[0])
		ev.expr(e.Args[1])
		return &CUnit{}, true
	case "list.touch", "list.checkMods", "map.checkMods":
		for _, a := range e.Args {
			ev.expr(a)
		}
		return &CUnit{}, true // no other code runs while the compiler evaluates: nothing can change under a loop
	case "list.mods", "map.mods":
		ev.expr(e.Args[0])
		return i64v(0), true
	case "list.appendText":
		l := ev.mutList(e)
		s := ev.str(e, 1)
		for i := 0; i < len(s); i++ {
			l.Elems = append(l.Elems, &CInt{types.TU8, big.NewInt(int64(s[i]))})
		}
		return &CUnit{}, true
	case "list.joinText":
		src := ev.elems(e, ev.expr(e.Args[0]))
		sep := ev.str(e, 1)
		parts := make([]string, len(src))
		for i, x := range src {
			parts[i] = x.(*CString).V
		}
		return &CString{strings.Join(parts, sep)}, true
	case "list.decodeUtf8", "list.decodeUtf8Range":
		src := ev.elems(e, ev.expr(e.Args[0]))
		lo, hi := 0, len(src)
		nt := e.Type().(*types.Nullable)
		if e.Op == "list.decodeUtf8Range" {
			from, to := ev.intArg(e, 1), ev.intArg(e, 2)
			if from.Sign() < 0 || to.Cmp(from) < 0 || to.Cmp(big.NewInt(int64(len(src)))) > 0 {
				return &CNull{nt}, true
			}
			lo, hi = int(from.Int64()), int(to.Int64())
		}
		buf := make([]byte, hi-lo)
		for i := lo; i < hi; i++ {
			buf[i-lo] = byte(src[i].(*CInt).V.Uint64())
		}
		if !utf8.Valid(buf) {
			return &CNull{nt}, true
		}
		return &CSome{nt, &CString{string(buf)}}, true
	case "list.sortedNative":
		src := ev.elems(e, ev.expr(e.Args[0]))
		out := &CList{T: e.Type().(*types.List), Elems: append([]ConstVal(nil), src...)}
		sort.SliceStable(out.Elems, func(i, j int) bool {
			c, _ := constCompare(out.Elems[i], out.Elems[j])
			return c < 0
		})
		return out, true
	case "list.toArray":
		src := ev.elems(e, ev.expr(e.Args[0]))
		at := e.Type().(*types.Array)
		out := &CArray{T: at, Elems: make([]ConstVal, len(src))}
		for i, x := range src {
			out.Elems[i] = copyVal(x)
		}
		return out, true
	case "array.toList":
		src := ev.elems(e, ev.expr(e.Args[0]))
		out := &CList{T: e.Type().(*types.List), Elems: make([]ConstVal, len(src))}
		for i, x := range src {
			out.Elems[i] = copyVal(x)
		}
		return out, true
	case "map.set":
		m := ev.expr(e.Args[0]).(*CMap)
		mapPut(m, ev.val(e.Args[1]), ev.val(e.Args[2]))
		return &CUnit{}, true
	case "map.remove":
		k := ev.expr(e.Args[1])
		switch m := ev.expr(e.Args[0]).(type) {
		case *CMap:
			if i := constIndex(m.Keys, k); i >= 0 {
				m.Keys = append(m.Keys[:i:i], m.Keys[i+1:]...)
				m.Vals = append(m.Vals[:i:i], m.Vals[i+1:]...)
				return &CBool{true}, true
			}
			return &CBool{false}, true
		case *CSet:
			if i := constIndex(m.Elems, k); i >= 0 {
				m.Elems = append(m.Elems[:i:i], m.Elems[i+1:]...)
				return &CBool{true}, true
			}
			return &CBool{false}, true
		}
	case "map.clear":
		switch m := ev.expr(e.Args[0]).(type) {
		case *CMap:
			m.Keys, m.Vals = nil, nil
		case *CSet:
			m.Elems = nil
		}
		return &CUnit{}, true
	case "map.reserve":
		ev.expr(e.Args[0])
		ev.expr(e.Args[1])
		return &CUnit{}, true
	case "map.copy":
		switch m := ev.expr(e.Args[0]).(type) {
		case *CMap:
			out := &CMap{T: e.Type().(*types.Map)}
			for i, k := range m.Keys {
				out.Keys = append(out.Keys, copyVal(k))
				out.Vals = append(out.Vals, copyVal(m.Vals[i]))
			}
			return out, true
		case *CSet:
			out := &CSet{T: e.Type().(*types.Set)}
			for _, k := range m.Elems {
				out.Elems = append(out.Elems, copyVal(k))
			}
			return out, true
		}
	case "map.keys", "map.values":
		switch m := ev.expr(e.Args[0]).(type) {
		case *CMap:
			src := m.Keys
			if e.Op == "map.values" {
				src = m.Vals
			}
			out := &CList{T: e.Type().(*types.List)}
			for _, x := range src {
				out.Elems = append(out.Elems, copyVal(x))
			}
			return out, true
		case *CSet:
			out := &CList{T: e.Type().(*types.List)}
			for _, x := range m.Elems {
				out.Elems = append(out.Elems, copyVal(x))
			}
			return out, true
		}
	case "map.entries":
		m := ev.expr(e.Args[0]).(*CMap)
		lt := e.Type().(*types.List)
		out := &CList{T: lt}
		for i, k := range m.Keys {
			out.Elems = append(out.Elems, &CTuple{T: lt.Elem.(*types.Tuple), Elems: []ConstVal{copyVal(k), copyVal(m.Vals[i])}})
		}
		return out, true
	}
	return nil, false
}

// clampIndex clamps a slice bound into 0..n, as the runtime does.
func clampIndex(v *big.Int, n int) int {
	if v.Sign() < 0 {
		return 0
	}
	if v.Cmp(big.NewInt(int64(n))) > 0 {
		return n
	}
	return int(v.Int64())
}
