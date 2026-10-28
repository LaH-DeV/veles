package sema

import (
	"strings"
	"testing"
)

// D197: `cede` makes the function it is in suspend, and every caller
// through ordinary calls, recursion and conditional higher-order inference
// (D116); a plain instance of a generic function stays plain.
func TestCedeEffects(t *testing.T) {
	expectClean(t, prelude+`fun checkpoint() {
  cede
}
fun countdown(n: i64) {
  if (n == 0) return
  cede
  countdown(n - 1)
}
fun each<T>(xs: List<T>, f: fun(T): () suspends) {
  loop (x in xs) f(x)
}
trait Step {
  fun run() suspends
}
struct Slow {
  implement Step {
    fun run() { cede }
  }
}
struct Conn {
  implement Closeable {
    fun close() suspends {
      cede
      io.println("closed")
    }
  }
}
fun main() {
  checkpoint()
  countdown(3)
  each([1, 2], x => { cede })
  val f: fun() suspends = () => { cede }
  f()
  val s: Step = Slow()
  s.run()
  with c = Conn()
  val m = Mutex<i64>(value: 0)
  // a plain instance: nothing passed to each cedes, so it holds the lock
  m.withLock(p => each([1], x => io.println("$x")))
  if (true) cede else io.println("never")
  loop (i in 0..<3) cede
  val t = async { cede }
  cede
  await t
}`)
}

// D197 2: where suspension is refused, `cede` is refused, directly or
// through a helper, with the context's own message.
func TestCedeRefused(t *testing.T) {
	const helper = "fun helper() { cede }\n"
	cases := []struct{ name, src, want string }{
		{"withLock", `fun main() {
  val m = Mutex<i64>(value: 0)
  m.withLock(p => { cede })
}`, "this lambda suspends, so its type must be a suspending function type"},
		{"lock guard", helper + `fun main() {
  val m = Mutex<i64>(value: 0)
  with (n = m.lock()) {
    helper()
  }
}`, "'helper' suspends, and the lock on 'm' is held"},
		{"lock guard direct", `fun main() {
  val m = Mutex<i64>(value: 0)
  with (n = m.lock()) {
    cede
  }
}`, "'cede' suspends, and the lock on 'm' is held"},
		{"rwlock", helper + `fun main() {
  val l = RwLock<i64>(value: 0)
  with (r = l.read()) {
    helper()
  }
}`, "'helper' suspends"},
		{"global", `val g = start()
fun start(): i64 {
  cede
  1
}
fun main() { io.println("$g") }`, "a global initializer cannot suspend"},
		{"const fun", `const fun k(): i64 {
  cede
  1
}
const K: i64 = k()
fun main() { io.println("$K") }`, "'cede' offers a task's executor a turn, and a 'const fun' runs in the compiler"},
		{"plain lambda", `fun main() {
  val f = () => { cede }
  f()
}`, "this lambda suspends, so its type must be a suspending function type"},
		{"thread", `fun main() {
  with th = try Thread.start(name: "t", f: () => { cede }) catch (e) { return }
}`, "this lambda suspends, so its type must be a suspending function type"},
		{"lazy val", `lazy val table: i64 = make()
fun make(): i64 {
  cede
  1
}
fun main() { io.println("$table") }`, "cannot throw or suspend"},
		{"close not declared", `struct Conn {
  implement Closeable {
    fun close() { cede }
  }
}
fun main() { }`, "this 'close()' suspends, so it says so"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectError(t, prelude+tc.src, tc.want)
		})
	}
}

// D197 5: the prelude's yieldNow is gone. Its uses are errors with a fix
// where the rewrite keeps the meaning; the fixed source compiles.
func TestYieldNowRemoved(t *testing.T) {
	cases := []struct{ name, src, want, fixed string }{
		{"statement", "fun main() {\n  yieldNow() // why\n}\n", "write the statement 'cede'", "  cede // why\n"},
		{"unbraced", "fun main() {\n  if (true) yieldNow()\n}\n", "write the statement 'cede'", "  if (true) cede\n"},
		{"fun body", "fun pause() => yieldNow()\nfun main() { pause() }\n", "the body is the block '{ cede }'", "fun pause() { cede }\n"},
		{"lambda body", "fun main() {\n  val f: fun() suspends = () => yieldNow()\n  f()\n}\n", "the body is the block '{ cede }'", "() => { cede }\n"},
		{"async", "fun main() {\n  val t = async yieldNow()\n  cede\n  await t\n}\n", "a task that cedes is 'async { cede }'", "val t = async { cede }\n"},
		{"value", "fun run(f: fun() suspends) { f() }\nfun main() { run(yieldNow) }\n", "pass '() => { cede }'", "run(() => { cede })"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixed := applyFixEdits(t, prelude+tc.src, tc.want)
			if !strings.Contains(fixed, tc.fixed) {
				t.Fatalf("the fix did not give %q:\n%s", tc.fixed, fixed)
			}
			expectClean(t, fixed)
		})
	}
	// no fix where the rewrite would change what runs or drop what an
	// argument computes
	for _, src := range []string{
		"fun main() {\n  val x = yieldNow()\n}\n",
		"fun main() {\n  yieldNow(io.println(\"kept\"))\n}\n",
		"fun main() {\n  val f: fun() = yieldNow\n}\n",
	} {
		for _, d := range checkSource(t, prelude+src).Items {
			if strings.Contains(d.Message, "was removed (D197)") && d.Fix != nil {
				t.Errorf("%q: unexpected fix %q", src, d.Fix.Title)
			}
		}
		expectError(t, prelude+src, "was removed (D197)")
	}
	// a program's own yieldNow is its own: a function, a local, a parameter
	expectClean(t, prelude+`fun yieldNow() { io.println("mine") }
fun main() { yieldNow() }`)
	expectClean(t, prelude+`fun main() {
  val yieldNow = () => io.println("a local")
  yieldNow()
}`)
	expectClean(t, prelude+`fun twice(yieldNow: fun()) {
  yieldNow()
  yieldNow()
}
fun main() { twice(() => io.println("x")) }`)
}
