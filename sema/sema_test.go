package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// checkSource writes a one-module package to a temp dir and checks it.
func checkSource(t *testing.T, src string) *source.Diagnostics {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	if !diags.HasErrors() {
		Check(pkg, diags, false)
	}
	return diags
}

func expectError(t *testing.T, src, want string) {
	t.Helper()
	diags := checkSource(t, src)
	for _, d := range diags.Items {
		if d.Severity == source.Error && strings.Contains(d.Message, want) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got:\n%s", want, diags.Render())
}

func expectClean(t *testing.T, src string) {
	t.Helper()
	diags := checkSource(t, src)
	if diags.HasErrors() {
		t.Errorf("unexpected errors:\n%s", diags.Render())
	}
}

const prelude = "use io\n"

func TestSpecRules(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"D11 val is immutable", prelude + `fun main() { val x = 1; x = 2 }`, "it is a 'val'"},
		{"D22 mut fun on val", prelude + `
struct C { n: i32 = 0; mut fun bump() { self.n += 1 } }
fun main() { val c = C(); c.bump() }`, "it is a 'val'"},
		{"D22 mutation in non-mut method", prelude + `
struct C { n: i32 = 0; fun bump() { self.n += 1 } }
fun main() { }`, "non-'mut' method"},
		{"D12 variant in other module only", prelude + `
struct X : Missing { }
fun main() { }`, "unknown type"},
		{"D13 exhaustiveness", prelude + `
sealed trait S
struct A : S { }
struct B : S { }
fun f(s: S): i32 = when (s) { is A => 1 }
fun main() { }`, "not exhaustive"},
		{"D13 guard does not count", prelude + `
sealed trait S
struct A : S { n: i32 }
fun f(s: S): i32 = when (s) { is A(n) if n > 0 => 1 }
fun main() { }`, "not exhaustive"},
		{"D5 nullable member access", prelude + `
struct U { name: string }
fun f(u: U?): string = u.name
fun main() { }`, "may be null"},
		{"D4 unhandled Result", prelude + `
struct E { }
fun f(): i32 throws E = 1
fun main() { f() }`, "unused Result"},
		{"D4 try needs throws", prelude + `
struct E { }
fun f(): i32 throws E = 1
fun main() { val x = try f() }`, "not declared 'throws'"},
		{"D45 error not in declared union", prelude + `
struct E1 { }
struct E2 { }
fun f(): i32 throws E1 = 1
fun g(): i32 throws E2 = try f()
fun main() { }`, "not in the declared 'throws"},
		{"D31 infinite size", prelude + `
sealed trait T
struct Leaf : T { }
struct Node : T { left: T }
fun main() { }`, "infinite size"},
		{"D17 coherence", prelude + `
trait Show { fun show(): string }
struct A { }
impl Show for A { fun show(): string = "a" }
impl Show for A { fun show(): string = "b" }
fun main() { }`, "conflicting impl"},
		{"D28 impl parameter names", prelude + `
trait T { fun f(x: i32): i32 }
struct A { }
impl T for A { fun f(y: i32): i32 = y }
fun main() { }`, "parameter must be named"},
		{"D53 override required", prelude + `
trait T { fun f(): i32 = 1 }
struct A { }
impl T for A { fun f(): i32 = 2 }
fun main() { }`, "must be marked 'override'"},
		{"D53 override without default", prelude + `
trait T { fun f(): i32 }
struct A { }
impl T for A { override fun f(): i32 = 2 }
fun main() { }`, "only allowed when trait"},
		{"D44 extern needs unsafe", prelude + `
extern "C" { fun puts(s: *raw u8): i32 }
fun main() { val p: *raw u8 = null }`, "must be nullable"},
		{"D44 raw deref needs unsafe", prelude + `
struct P { n: i32 }
extern "C" { fun get(): *raw P }
fun main() { val p = unsafe { get() }; val n = p.n }`, "requires an 'unsafe' block"},
		{"M5 private function", prelude + `
fun main() { io.veles_print("x") }`, "private to module"},
		{"D26 ambiguity", prelude + `
trait A { fun f(): i32 }
trait B { fun f(): i32 }
struct S { }
impl A for S { fun f(): i32 = 1 }
impl B for S { fun f(): i32 = 2 }
fun main() { val s = S(); s.f() }`, "ambiguous method"},
		{"D25 push into List", prelude + `
fun main() { val xs = [1, 2]; xs.push(3) }`, "immutable List"},
		{"D25 empty list needs a type", prelude + `
fun main() { val xs = [] }`, "cannot infer the element type"},
		{"D30 elvis needs nullable", prelude + `
fun main() { val x = 1 ?: 2 }`, "needs a nullable left operand"},
		{"D40 trait effects declared", prelude + `
trait T { fun f(): i32 throws }
fun main() { }`, "error type must be declared"},
		{"D28 generic inference", prelude + `
struct Box<T> { value: T }
fun main() { val b = Box(value: 1); val s: string = b.value }`, "type mismatch"},
		{"main signature", prelude + `fun main(): i32 = 1`, "'main' must take no parameters"},
		{"unreachable", prelude + `fun main() { return; io.println("x") }`, "unreachable code"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { expectError(t, c.src, c.want) })
	}
}

func TestAccepted(t *testing.T) {
	cases := map[string]string{
		"smart cast": prelude + `
fun f(s: string?): i64 { if (s == null) return 0; s.len() }
fun main() { }`,
		"smart cast via early return with elvis": prelude + `
fun f(s: string?): i64 { val v = s ?: return -1; v.len() }
fun main() { }`,
		"inferred expression body": prelude + `
fun add(a: i32, b: i32) = a + b
fun main() { val x: i32 = add(1, 2) }`,
		"inferred error union": prelude + `
struct E1 { }
struct E2 { }
fun f(): i32 throws E1 = 1
fun g(): i32 throws E2 = 2
fun h(): i32 throws { try f() + try g() }
fun k(): i32 throws E1 | E2 = try h()
fun main() { }`,
		"nested nullable exhaustive": prelude + `
fun f(x: i32??): i32 = when (x) {
  null => 0
  Some(null) => 1
  Some(Some(v)) => v
}
fun main() { }`,
		"trait default and generic bound": prelude + `
trait Show { fun show(): string; fun twice(): string = self.show() + self.show() }
struct A { }
impl Show for A { fun show(): string = "a" }
fun <T: Show> p(x: T): string = x.twice()
fun main() { io.println(p(A())) }`,
		"struct methods on generic": prelude + `
struct Stack<T> { items: MutableList<T> = []; fun push(x: T) { self.items.push(x) }; fun len(): i64 = self.items.len() }
fun main() { val s = Stack<i32>(); s.push(1); io.println("${s.len()}") }`,
		"mutation through pointer on val": prelude + `
struct C { n: i32 }
fun main() { val c = C(n: 1); val p = &c; p.n = 2 }`,
		"Result value matched": prelude + `
struct E { code: i32 }
fun f(): i32 throws E = Err(E(code: 1))
fun main() {
  val r = f()
  val v = when (r) { is Ok(value) => value; is Err(error) => error.code }
}`,
		"pointer scrutinee (D39)": prelude + `
sealed trait T
struct A : T { n: i32 }
struct B : T { }
fun f(p: *T): i32 = when (p) { is A(n) => n; is B => 0 }
fun main() { }`,
		"labeled loops": prelude + `
fun main() { outer: loop (i in 0..3) { loop (j in 0..3) { if (j == 1) continue outer; if (i == 2) break outer } } }`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) { expectClean(t, src) })
	}
}
