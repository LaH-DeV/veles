package llvm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// Plain instances (D116). A conditionally suspending function — one that
// suspends only through its suspend-parameters — is emitted as a coroutine
// under its own name and, when some call binds none of those parameters to
// a function that suspends, also as an ordinary function, `<name>.plain`.
// An instance carries its view (sema.Func.Plain): the suspend-parameters
// it knows do not suspend. Calls inside it ask sema what they do under that
// view, and every lambda it creates is emitted under it too, so a lambda
// that calls a captured parameter calls it the way the instance has it.
//
// A plain instance is a copy of the function's header over the same body;
// it differs in its name, in not being a coroutine, and in its view.

type plainKey struct {
	fn    *sema.Func
	env   uintptr // the view's identity: one map per instance
	plain bool    // the copy is an ordinary function, not a coroutine
}

// instanceOf returns fn emitted under env, as an ordinary function when
// plain, queuing its emission the first time it is asked for.
func (g *gen) instanceOf(fn *sema.Func, env map[*sema.Var]bool, plain bool) *sema.Func {
	if env == nil && (!plain || !fn.Suspends) {
		// the function as sema made it; a lambda's is emitted only when a
		// value of it is made, so one only ever called plain has no
		// coroutine copy in the output
		if fn.IsClosure {
			key := plainKey{fn: fn}
			if _, done := g.instances[key]; !done {
				g.instances[key] = fn
				g.pending = append(g.pending, func() { g.function(fn) })
			}
		}
		return fn
	}
	key := plainKey{fn, reflect.ValueOf(env).Pointer(), plain}
	if inst, ok := g.instances[key]; ok {
		return inst
	}
	inst := *fn
	inst.Plain = env
	g.instanceCount[fn]++
	suffix := ".plain"
	if !plain {
		suffix = ".in"
	}
	if n := g.instanceCount[fn]; n > 1 {
		suffix += fmt.Sprint(n)
	}
	inst.Name = fn.Name + suffix
	if plain {
		inst.Suspends, inst.Conditional = false, false
		sig := *fn.Sig
		sig.Effects.Suspends = false
		inst.Sig = &sig
	}
	g.instances[key] = &inst
	g.pending = append(g.pending, func() { g.function(&inst) })
	return &inst
}

// plainEnv is a conditional function's own plain view, one map per function.
func (g *gen) plainEnv(fn *sema.Func) map[*sema.Var]bool {
	env, ok := g.plainEnvs[fn]
	if !ok {
		env = sema.PlainEnv(fn)
		g.plainEnvs[fn] = env
	}
	return env
}

// callee picks the instance a call runs: the plain one of a conditional
// function when nothing it is given suspends under the caller's view. The
// lambdas given for its suspend-parameters are then emitted plain too;
// the caller clears plainLambdas once the arguments are emitted.
func (g *gen) callee(e *sema.Call) *sema.Func {
	fn := e.Fn
	if !fn.Conditional || sema.CallSuspendsIn(e, g.view()) {
		return fn
	}
	off := 0
	if fn.Receiver != nil {
		off = 1
	}
	for i, p := range fn.Params {
		if i+off >= len(e.Args) || !sema.IsSuspendParam(p) {
			continue
		}
		if cl := lambdaArg(e.Args[i+off]); cl != nil {
			g.plainLambdas[cl] = true
		}
	}
	return g.instanceOf(fn, g.plainEnv(fn), true)
}

// lambdaArg is the lambda an argument is, through what passes a function
// value on unchanged.
func lambdaArg(a sema.Expr) *sema.Closure {
	cl, _ := sema.FuncValueOf(a).(*sema.Closure)
	return cl
}

// lambdaFn is the function a lambda's value points to in this instance.
func (g *gen) lambdaFn(e *sema.Closure) *sema.Func {
	return g.instanceOf(e.Fn, g.view(), g.plainLambdas[e])
}

// view is the instance being emitted's view; nil outside a function (a
// global's initializer) and in an instance sema made.
func (g *gen) view() map[*sema.Var]bool {
	if g.fn == nil {
		return nil
	}
	return g.fn.Plain
}

// referencedHeld is the text of the held coroutine copies something refers
// to — the rest of the output, or a copy already kept — found to a fixpoint.
func (g *gen) referencedHeld(rest string) string {
	var kept strings.Builder
	keep := make([]bool, len(g.held))
	for changed := true; changed; {
		changed = false
		for i, h := range g.held {
			if !keep[i] && (refersTo(rest, h.name) || refersTo(kept.String(), h.name)) {
				keep[i], changed = true, true
				kept.WriteString(h.text)
			}
		}
	}
	return kept.String()
}

// refersTo reports whether IR text names the global @name itself (not a
// longer name it is a prefix of, such as `name.plain`).
func refersTo(text, name string) bool {
	at := "@" + name
	for i := strings.Index(text, at); i >= 0; {
		end := i + len(at)
		if end == len(text) || !isNameByte(text[end]) {
			return true
		}
		next := strings.Index(text[end:], at)
		if next < 0 {
			return false
		}
		i = end + next
	}
	return false
}

func isNameByte(b byte) bool {
	return b == '.' || b == '_' || b == '$' || b == '-' || '0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}
