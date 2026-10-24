package sema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/source"
)

// D208: a test the compiler can run runs there too. A `const fun` runs on two
// engines — compiled code, and the checker's interpreter with its own
// built-ins — and a test that runs on both checks that they agree. Which
// tests: the ones whose body calls only `const fun`s and test helpers the
// compiler can run, does no I/O, starts no task and does not throw (the
// `const fun` rules, inferred: nothing is written). The checker runs them
// after the last round and leaves the outcome on the test; the program's
// runner reports it beside its own (`… ok (also in the compiler)`, or a
// failure on one side only).

// testRun is what one test recorded while the compiler ran it.
type testRun struct {
	fails      []string // `  file:line:col: message` lines, as the runtime writes them
	skipped    bool
	skipWhy    string
	outOfSteps bool // over the step budget: it ran in the program only
}

// testStopped ends a test the compiler runs: `require`, `skip`, a panic, or
// anything the program would stop at too.
type testStopped struct{}

// debugD208 prints why a test does not run in the compiler
// (VELES_COMPILER_TESTS=why when running `veles test`).
var debugD208 = os.Getenv("VELES_COMPILER_TESTS") == "why"

// compilerCanRun reports whether the compiler can run fn: a test, a test
// helper, or what they call. Nothing is reported: a test it cannot run runs
// in the program only, as every test did before D208.
func (c *Checker) compilerCanRun(fn *Func) bool {
	c.ensureConstBody(fn) // asked early, by a constant (`@SkipIf`), before the checker reached it
	if runsInCompiler(fn) {
		return c.validateConstFunQuiet(fn)
	}
	if fn.tmpl == nil || !fn.tmpl.TestCode {
		if debugD208 {
			fmt.Fprintf(os.Stderr, "D208: %s is neither a const fun nor test code\n", fn.Display)
		}
		return false
	}
	if ok, done := c.testRunnable[fn]; done {
		return ok
	}
	if c.testRunnable == nil {
		c.testRunnable = map[*Func]bool{}
	}
	c.testRunnable[fn] = false // recursion: it stands while it is decided
	if fn.Body == nil || fn.Suspends || fn.UsesUnsafe { // a test is declared throwing: its body says whether it does
		if debugD208 {
			fmt.Fprintf(os.Stderr, "D208: %s: body %v, throws %v, suspends %v, unsafe %v\n", fn.Display, fn.Body != nil, fn.Sig.Effects.Throws, fn.Suspends, fn.UsesUnsafe)
		}
		return false
	}
	ok := true
	no := func(why string, args ...any) {
		if ok && debugD208 {
			fmt.Fprintf(os.Stderr, "D208: %s: %s\n", fn.Display, fmt.Sprintf(why, args...))
		}
		ok = false
	}
	var visit func(n any)
	visit = func(n any) {
		if !ok {
			return
		}
		switch n := n.(type) {
		case *Call:
			if n.Fn.Extern && constExterns[n.Fn.Name] {
				break
			}
			if !c.compilerCanRun(n.Fn) {
				no("calls %s", n.Fn.Display)
			}
		case *Closure:
			if n.Fn.Body != nil {
				walkBlock(n.Fn.Body, visit)
			}
		case *FuncRef:
			if !c.compilerCanRun(n.Fn) {
				no("uses %s as a value", n.Fn.Display)
			}
		case *CallVirtual, *Box, *Launch, *AwaitTask, *ScopeBlock, *Race, *With, *Throw, *Try, *MakeResult:
			no("has a %T", n)
		case *VarRef:
			if n.Var.IsGlobal {
				no("reads the global %s", n.Var.Name)
			}
		case *Builtin:
			if !constBuiltins[n.Op] {
				no("uses %s", n.Op)
			}
		}
	}
	walkBlock(fn.Body, visit)
	c.testRunnable[fn] = ok
	if ok {
		fn.compilerRunnable = true // a constant may call it (`@SkipIf(helper())`)
	}
	return ok
}

// validateConstFunQuiet is validateConstFun's answer without its reports:
// a `const fun` that breaks the rules is reported where it is declared, by
// checkConstFuns; here it only keeps the test in the program.
func (c *Checker) validateConstFunQuiet(fn *Func) bool {
	if ok, done := c.constFunOK[fn]; done {
		return ok
	}
	saved := c.roundDiags
	c.roundDiags = &source.Diagnostics{}
	ok := c.validateConstFun(fn)
	c.roundDiags = saved
	return ok
}

// runTestsInCompiler runs every test the compiler can run, in the compiler,
// and leaves the outcome on it for the runner.
func (c *Checker) runTestsInCompiler(prog *Program) {
	for _, t := range prog.Tests {
		if t.Skipped || len(t.Gates) > 0 || !c.compilerCanRun(t) {
			continue
		}
		run := c.runTestInCompiler(t)
		if run.outOfSteps {
			continue // too long for the compiler: the program's run decides
		}
		t.CompilerRan = true
		t.CompilerSkipped = run.skipped
		t.CompilerFails = strings.Join(run.fails, "")
	}
}

func (c *Checker) runTestInCompiler(t *Func) (run *testRun) {
	run = &testRun{}
	ev := c.newConstEval(t.Span, false)
	ev.testing = run
	ev.name = "test '" + t.Display + "'"
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case testStopped, constFailed:
			default:
				panic(r)
			}
		}
	}()
	ev.invoke(t, nil, t.Span)
	return run
}

// testFail records one failure line, as veles_test_fail writes it.
func (ev *constEval) testFail(span source.Span, msg string) {
	ev.testing.fails = append(ev.testing.fails, "  "+ev.c.whereText(span)+": "+msg+"\n")
}

// whereText is `path:line:col` as the program's runner prints it: relative
// to the package root.
func (c *Checker) whereText(span source.Span) string {
	if span.File == nil {
		return ""
	}
	path := span.File.Path
	if !span.File.Embedded && c.pkg != nil && c.pkg.Root != "" && filepath.IsAbs(path) {
		if rel, err := filepath.Rel(c.pkg.Root, path); err == nil && !strings.HasPrefix(rel, "..") {
			path = rel
		}
	}
	line, col := span.File.Position(span.Start)
	return fmt.Sprintf("%s:%d:%d", filepath.ToSlash(path), line, col)
}

// SkewReplaceForTest makes the compiler's `string.replace` answer wrongly
// (an extra "?"), so a test can show the two engines disagreeing (D208). Only
// tests set it.
var SkewReplaceForTest bool
