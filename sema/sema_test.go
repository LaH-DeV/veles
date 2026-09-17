package sema

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
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
error E { }
fun f(): i32 throws E = 1
fun main() { f() }`, "unused Result"},
		{"D4 throw needs throws", prelude + `
error E { }
fun f(): i32 { throw E() }
fun main() { }`, "not declared 'throws'"},
		{"D4 throw not in declared union", prelude + `
error E1 { }
error E2 { }
fun f(): i32 throws E1 { throw E2() }
fun main() { }`, "E2"},
		{"hint: io not imported", `fun main() { io.println("x") }`, "add 'use io'"},
		{"hint: unqualified std function", `fun main() { println("x") }`, "did you mean 'io.println'"},
		{"hint: nullable used as value", prelude + `
fun main() { val m = ["a": 1]; val v: i64 = m["a"] }`, "supply a fallback with '?:'"},
		{"hint: string plus number", prelude + `
fun main() { val s = "a" + 1 }`, "interpolation"},
		{"hint: unknown struct field", prelude + `
struct P { x: i64 }
fun main() { val p = P(x: 1, y: 2) }`, "no field named 'y'"},
		{"D4 try needs throws", prelude + `
error E { }
fun f(): i32 throws E = 1
fun main() { val x = try f() }`, "not declared 'throws'"},
		{"D45 error not in declared union", prelude + `
error E1 { }
error E2 { }
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
error E1 { }
error E2 { }
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
error E { code: i32 }
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
		"throw as sugar for Err": prelude + `
error E { n: i32 }
fun f(x: i32): i32 throws { if (x < 0) throw E(n: x); x }
fun g(x: i32?): i32 throws E = x ?: throw E(n: 0)
fun main() { when (f(1)) { is Ok(v) => { }; is Err(e) => { } } }`,
		"labeled loops": prelude + `
fun main() { loop :outer (i in 0..3) { loop (j in 0..3) { if (j == 1) continue outer; if (i == 2) break outer } } }`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) { expectClean(t, src) })
	}
}

func TestConcurrencyRules(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"D3 async outside scope", prelude + `
fun work() { }
fun main() { async work() }`, "lexically inside a 'scope'"},
		{"D35 non-Sendable argument", prelude + `
fun work(xs: MutableList<i32>) { }
fun main() { var xs: MutableList<i32> = []; scope { async work(xs) } }`, "not Sendable"},
		{"D16 recv must be awaited", prelude + `
fun main() { val ch = Channel<i32>(); val x = ch.recv() }`, "must be awaited"},
		{"D35 no await inside a lock", prelude + `
fun main() { val m = mutex(1); m.withLock(p => { await sleep(1); 0 }) }`, "lambda suspends"},
		{"D40 impl must declare suspends", prelude + `
trait T { fun f(): i32 }
struct A { }
impl T for A { fun f(): i32 { await sleep(1); 1 } }
fun main() { }`, "declares it non-suspending"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { expectError(t, c.src, c.want) })
	}
}

func TestConcurrencyAccepted(t *testing.T) {
	expectClean(t, prelude+`
fun tick(n: i32): i32 { await sleep(1); n }
fun main() throws {
  val ch = Channel<i32>(capacity: 2)
  scope {
    val t = async tick(1)
    ch.send(await t)
    val v = await ch.recv()
    val (a, b) = gather { async tick(2); async tick(3) }
    val x = try a
    val y = try b
    val w = race { val m = ch.recv() => if (m == null) 1 else 2; sleep(10) => 2 }
    io.println("$v $x $y $w")
  }
}`)
}

func TestUnusedBindings(t *testing.T) {
	src := prelude + `
error E { code: i32 }
fun may(): i32 throws E { 1 }
fun takes(r: Result<i32, E>) {}
fun main() {
  val r = may()
  val s = may()
  if (s is Ok) io.println("ok")
  takes(may())
  val _ = may()
  val unusedInt = 5
  var counter = 0
  counter = 1
  val (a, b) = (1, 2)
  io.println("$a")
  val cap = 7
  val f = () => cap + 1
  loop (i in 0..3) { }
  when (s) {
    is Ok(v) => io.println("v")
    is Err(e) => io.println("e")
  }
}`
	diags := checkSource(t, src)
	out := diags.Render()
	for _, want := range []string{
		"unused Result 'r'",
		"'unusedInt' is never used",
		"'counter' is never used",
		"'b' is never used",
		"'f' is never used",
		"'i' is never used",
		"'v' is never used",
		"'e' is never used",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, bad := range []string{"'s'", "'a'", "'cap'", "'_'"} {
		if strings.Contains(out, bad+" is never used") || strings.Contains(out, "unused Result "+bad) {
			t.Errorf("false positive on %s in:\n%s", bad, out)
		}
	}
}

func TestLoopNeverRepeatsWarning(t *testing.T) {
	diags := checkSource(t, prelude+`
fun main() {
  loop { break }
  var i = 0
  loop { i += 1; if (i > 3) break }
  loop (x in [1, 2]) { if (x == 1) continue; break }
}`)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors:\n%s", diags.Render())
	}
	warnings := 0
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "never repeats") {
			warnings++
		}
	}
	if warnings != 1 {
		t.Errorf("want exactly one 'never repeats' warning, got %d:\n%s", warnings, diags.Render())
	}
}

func TestResultSmartCast(t *testing.T) {
	// `is Ok` / `is Err` read through to the payload, like `!= null` on T?
	expectClean(t, prelude+`
error E { code: i32 }
fun may(n: i32): i32 throws E = if (n > 0) n else throw E(code: n)
fun main() {
  val r = may(1)
  if (r is Ok) {
    val v: i32 = r + 1
    io.println("$v")
  } else {
    val c: i32 = r.code
    io.println("$c")
  }
  if (r !is Ok) io.println("${r.code}")
  when (r) {
    is Ok => io.println("${r + 1}")
    is Err => io.println("${r.code}")
  }
  when (r) {
    is Ok => io.println("$r")
    is Err => when (r) { is E => io.println("${r.code}") }
  }
  if (r is Err) { if (r is E) io.println("${r.code}") }
  val o: Option<i32> = Some(value: 5)
  if (o is Some) io.println("${o + 1}")
}`)
	// outside the test the subject is still the Result
	expectError(t, prelude+`
error E { code: i32 }
fun may(n: i32): i32 throws E = if (n > 0) n else throw E(code: n)
fun main() {
  val r = may(1)
  if (r is Ok) io.println("ok")
  val v: i32 = r
}`, "type mismatch")
}

func TestTwoVariantSealedElseNarrows(t *testing.T) {
	expectClean(t, prelude+`
sealed trait Shape
struct Circle : Shape { r: f64 }
struct Rect : Shape { w: f64, h: f64 }
fun area(s: Shape): f64 = if (s is Circle) s.r * s.r * 3.0 else s.w * s.h
fun main() { io.println("${area(Circle(r: 1.0))}") }`)
}

func TestErrorTrait(t *testing.T) {
	// every error-position struct implements Error: synthesized default,
	// explicit override, and dispatch on a union without a `when`
	expectClean(t, prelude+`
error ParseError { text: string }
error RangeError { value: i64; fun message(): string = "out of range: ${self.value}" }
fun parse(s: string): i64 throws ParseError | RangeError {
  val n = s.toInt() ?: throw ParseError(text: s)
  if (n > 10) throw RangeError(value: n)
  n
}
fun load(s: string): i64 throws = try parse(s)
fun main() {
  val r = load("x")
  when (r) {
    is Ok => io.println("$r")
    is Err => io.println(r.message())
  }
  if (r is Err) {
    val m: string = r.message()
    io.println(m)
    if (r is ParseError) io.println(r.message())
  }
  val p: Panic = Panic(message: "boom")
  io.println(p.message())
}`)
	expectError(t, prelude+`
fun f(): i64 throws string = 1
fun main() { }`, "cannot be an error")
	expectError(t, prelude+`
fun f(): i64 throws { throw 5 }
fun main() { }`, "cannot be an error")
	expectError(t, prelude+`
fun <X> f(x: X): i64 throws X = throw x
fun main() { val r = f("s"); if (r is Err) io.println("err") }`, "cannot be an error")
	expectClean(t, prelude+`
error E { }
fun <X: Error> f(x: X): i64 throws X = throw x
fun main() { val r = f(E()); if (r is Err) io.println(r.message()) }`)
}

func TestErrorDeclarations(t *testing.T) {
	// message field convention, override-free message(), named error sets,
	// a cause field holding a union
	expectClean(t, prelude+`
error Plain
error Tagged { message: string }
error Parse { text: string; fun message(): string = "parse: ${self.text}" }
error Bounds { value: i64; fun bound(): i64 = 65535 }
error PortErrors = Parse | Bounds | Tagged | Plain
error More = PortErrors | Panic
error Wrapped {
  key: string
  cause: PortErrors
  fun message(): string = "${self.key}: ${self.cause.message()}"
}
fun parse(s: string): i64 throws PortErrors {
  if (s == "t") throw Tagged(message: "tagged")
  if (s == "p") throw Plain()
  s.toInt() ?: throw Parse(text: s)
}
fun wrap(s: string): i64 throws Wrapped {
  val r = parse(s)
  if (r is Err) throw Wrapped(key: "k", cause: r)
  r
}
fun both(s: string): i64 throws More = try parse(s)
fun main() {
  val r = parse("x")
  when (r) {
    is Ok => io.println("$r")
    is Err => when (r) {
      is Parse => io.println(r.message())
      is Bounds => io.println("${r.bound()}")
      else => io.println(r.message())
    }
  }
  val w = wrap("x")
  if (w is Err) io.println("${w.message()} ${w.cause.message()}")
  val m = both("x")
  if (m is Err) io.println(m.message())
  val p = Panic(message: "boom")
  io.println(p.message())
}`)
	for _, c := range []struct{ src, want string }{
		{`struct Dog { }
error Errs = Dog
fun main() { }`, "'Dog' is not an error"},
		{`error A
error Errs = A
struct Holder { e: Errs }
fun main() { }`, "names an error set"},
		{`error A
error Errs = A
fun main() { val x: Errs = A() }`, "names an error set"},
		{`error A
error S1 = A | S2
error S2 = S1
fun main() { }`, "refers to itself"},
		{`error A { n: i64; override fun helper(): i64 = 1 }
fun main() { }`, "only 'message' can be overridden"},
		{`error A { message: i64 }
fun main() { val a = A(message: 1); io.println(a.message()) }`, "A(message: 1)"}, // not a string field: default rendering, no error; see below
	} {
		if c.want == "A(message: 1)" {
			expectClean(t, prelude+c.src)
			continue
		}
		expectError(t, prelude+c.src, c.want)
	}
}

func TestFieldPathSmartCasts(t *testing.T) {
	// D5: a chain of direct struct fields from a local (or value self) is a
	// stable place; tests on it narrow the path itself
	expectClean(t, prelude+`
error ParseError { text: string }
error RangeError { value: i64 }
error PortErrors = ParseError | RangeError
error ConfigError { key: string, cause: PortErrors }
struct Address { city: string }
struct User {
  name: string
  address: Address?
  fun city(): string = if (self.address != null) self.address.city else "?"
}
struct Box { user: User }
fun load(): i64 throws ConfigError = throw ConfigError(key: "k", cause: RangeError(value: 7))
fun main() {
  val c = load()
  if (c is Err) {
    when (c.cause) {
      is RangeError => io.println("${c.cause.value}")
      is ParseError => io.println(c.cause.text)
    }
    if (c.cause is RangeError) io.println("${c.cause.value}")
  }
  val b = Box(user: User(name: "a", address: Address(city: "x")))
  if (b.user.address != null) io.println(b.user.address.city)
  var u = b.user
  if (u.address != null) { io.println(u.address.city); u.name = "n"; io.println(u.address.city) }
  val f = () => if (b.user.address != null) b.user.address.city else "?"
  io.println(f() + u.city())
}`)
	for _, c := range []struct{ name, src string }{
		{"field assigned", `
  if (u.address != null) { u.address = null; io.println(u.address.city) }`},
		{"root reassigned", `
  if (u.address != null) { u = User(name: "b", address: null); io.println(u.address.city) }`},
		{"mut method on root", `
  if (u.address != null) { u.clear(); io.println(u.address.city) }`},
		{"address taken", `
  if (u.address != null) { val p = &u; other(p); io.println(u.address.city) }`},
		{"through a pointer", `
  val p = &u
  if (p.address != null) io.println(p.address.city)`},
		{"assigned in loop", `
  if (u.address != null) { loop (i in 0..1) { u.address = null }; io.println(u.address.city) }`},
		{"mut self", ``},
	} {
		src := prelude + `
struct Address { city: string }
struct User { name: string, address: Address?; mut fun clear() { self.address = null }
  mut fun city(): string = if (self.address != null) self.address.city else "?" }
fun other(u: *User) { u.address = null }
fun main() {
  var u = User(name: "a", address: Address(city: "x"))` + c.src + `
}`
		t.Run(c.name, func(t *testing.T) { expectError(t, src, "may be null") })
	}
}

func TestMathBuiltins(t *testing.T) {
	expectClean(t, prelude+`
fun main() {
  val x = 2.0
  val a: f64 = x.sqrt() + x.abs() + x.floor() + x.ceil() + x.round() + x.trunc() + x.pow(2.0) + x.min(1.0) + x.max(3.0)
  val b: bool = x.isNaN() || x.isFinite() || x.isInfinite()
  val f: f32 = (9.0 as f32).sqrt()
  val n = -7
  val c: i64 = n.abs() + n.min(3) + n.max(3) + n.mod(3) + n.clamp(0, 5) + n.sign()
  val d: f64 = x.mod(1.5) + x.clamp(0.0, 1.0) + x.sign()
  val u: u8 = (200 as u8).min(3 as u8)
  io.println("$a $b $f $c $u")
}`)
	expectError(t, prelude+`fun main() { val s = "x".sqrt() }`, "no method 'sqrt'")
	expectError(t, prelude+`fun main() { val s = 2.0.pow() }`, "takes 1 argument")
}

func TestNumberAndCollectionBuiltins(t *testing.T) {
	expectClean(t, prelude+`
fun main() {
  val x = 2.0
  val a: f64 = x.log() + x.log2() + x.log10() + x.exp() + x.sin() + x.cos() + x.tan() + x.atan2(1.0) + x.hypot(3.0)
  val n = 7
  val b: i64 = n.pow(2) + n.wrappingAdd(1) + n.wrappingSub(1) + n.wrappingMul(2) + n.saturatingAdd(1) + n.saturatingSub(1)
  val c: i64? = n.checkedAdd(1)
  val d: i64? = n.checkedSub(1)
  val e: i64? = n.checkedMul(2)
  val f: i64 = n.countOnes() + n.leadingZeros() + n.trailingZeros()
  val g: u8 = (3 as u8).pow(2 as u8).saturatingAdd(1 as u8)
  val m: MutableMap<string, i64> = [:]
  val got: i64 = m.getOrPut("k", () => 1)
  val ages = ["ann": 41]
  ages.forEach((k, v) => io.println("$k $v"))
  val next: Map<string, i64> = ages.mapValues(v => v + 1)
  val adults: Map<string, i64> = ages.filter((k, v) => v >= 18)
  val s = MutableSet<i64>()
  val u: Set<i64> = s.union(s)
  val i: Set<i64> = s.intersect(s)
  val df: Set<i64> = s.difference(s)
  val sub: bool = s.isSubsetOf(s)
  io.println("$a $b $c $d $e $f $g $got ${next.len()} ${adults.len()} ${u.len()} ${i.len()} ${df.len()} $sub")
}`)
	expectError(t, prelude+`
fun main() { val m = ["a": 1]; m.getOrPut("b", () => 2) }`, "changes the map")
	expectError(t, prelude+`
fun main() { val s = Set<i64>(); val t = Set<string>(); s.union(t) }`, "needs a set of 'i64'")
}

func TestExtendBlocks(t *testing.T) {
	expectClean(t, prelude+`
struct Point { x: i64, y: i64 }
struct Box<T> { value: T }
trait Show { fun show(): string }
impl Show for Point { fun show(): string = "p" }
extend Point {
  pub fun sum(): i64 = self.x + self.y
  mut fun bump() { self.x += 1 }
}
extend<T: Show> Box<T> { fun label(): string = self.value.show() }
extend<T> Box<T> { fun get(): T = self.value }
fun main() {
  var p = Point(x: 1, y: 2)
  p.bump()
  val q = &p
  val b = Box(value: p)
  io.println("${p.sum()} ${q.sum()} ${b.get().x} ${b.label()}")
  // the prelude's extend blocks
  val words: List<string> = " a b ".trim().split(" ")
  val xs = mut [3, 1, 2]
  xs.insert(0, 9)
  io.println("${words.len()} ${xs.take(2)} ${xs.sum()} ${xs.at(-1)} ${(1..4).len()} ${"abc".toUpper()}")
  val bad: MutableList<u8> = [255]
  val text: string? = bad.decodeUtf8()
  io.println("$text ${"hi".byteAt(0)} ${"hi".bytes()}")
  val v: i64 = xs.at(-1) ?: panic("empty")
  io.println("$v")
}`)
	cases := []struct{ name, src, want string }{
		{"builtin outside std", `extend string { fun shout(): string = self }`, "outside the standard library"},
		{"foreign struct", `extend Panic { fun why(): string = "" }`, "declared outside this package"},
		{"pointer target", `struct P { x: i64 }
extend *P { fun z() {} }`, "only named types can be extended"},
		{"type param target", `extend<T> T { fun z() {} }`, "only named types can be extended"},
		{"struct body collision", `struct P { x: i64
  fun sum(): i64 = self.x }
extend P { fun sum(): i64 = 1 }`, "already declared in the body"},
		{"duplicate in block", `struct P { x: i64 }
extend P { fun a(): i64 = 1
  fun a(): i64 = 2 }`, "duplicate method 'a'"},
		{"overlapping blocks", `struct P { x: i64 }
extend P { fun a(): i64 = 1 }
extend P { fun a(): i64 = 2 }`, "already provided for 'P'"},
		{"override", `struct P { x: i64 }
extend P { override fun a(): i64 = 1 }`, "only meaningful inside an impl block"},
		{"for keyword", `struct P { x: i64 }
extend Show for P { }`, "names the type being extended"},
		{"bound not met", `struct P { x: i64 }
trait Show { fun show(): string }
struct Box<T> { value: T }
extend<T: Show> Box<T> { fun label(): string = self.value.show() }
fun main() { val b = Box(value: P(x: 1)); b.label() }`, "requires 'P' to implement 'Show'"},
		{"private across modules is still M5", `struct P { x: i64 }
extend P { fun a(): i64 = 1 }
fun main() { val p = P(x: 1); p.a() }`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.want == "" {
				expectClean(t, prelude+c.src)
				return
			}
			expectError(t, prelude+c.src, c.want)
		})
	}
}

// The compiler's own std sources (`<dir>/std/<module>`) are checked as that
// std module, replacing the embedded copy: editing the prelude reports
// against the edited files, and its extend blocks on built-in types pass.
func TestStdSourceTreeIsStd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "std", "prelude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(std.FS, "prelude")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, _ := fs.ReadFile(std.FS, "prelude/"+e.Name())
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	extra := "extend string {\n  pub fun shout(): string = self + \"!\"\n}\nfun wrong(): i64 = \"x\"\n"
	if err := os.WriteFile(filepath.Join(dir, "zz_extra.vs"), []byte(extra), 0o644); err != nil {
		t.Fatal(err)
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	if !pkg.Given.Std || pkg.Given.Path != "std/prelude" {
		t.Fatalf("given module = %q std=%v", pkg.Given.Path, pkg.Given.Std)
	}
	Check(pkg, diags, false)
	var sawMismatch bool
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "outside the standard library") {
			t.Errorf("std source tree treated as a user package: %s", d.Message)
		}
		if strings.Contains(d.Message, "type mismatch") && strings.Contains(d.Span.File.Path, "zz_extra.vs") {
			sawMismatch = true
		}
	}
	if !sawMismatch {
		t.Errorf("the on-disk prelude was not the one checked:\n%s", diags.Render())
	}
}

func TestOsFsPathModules(t *testing.T) {
	expectClean(t, `
use io
use os
use fs
use path

fun main() throws IoError {
  val args: List<string> = os.args()
  val home: string? = os.env("HOME")
  val r: os.Output = try os.run("clang", ["--version"])
  io.println("${r.code} ${r.stdout} ${r.ok()} ${os.program()} $args $home")
  val text: string = try fs.readFile("a.txt")
  try fs.writeFile("a.txt", text)
  try fs.appendFile("a.txt", text)
  val names: List<string> = try fs.listDir(".")
  try fs.mkdir("x/y")
  try fs.rename("a.txt", "b.txt")
  try fs.remove("b.txt")
  val here: string = try fs.cwd()
  io.println("${fs.exists("a")} ${fs.isFile("a")} ${fs.isDir("a")} $names $here")
  val p: string = path.joinAll([path.join("a", "b"), path.dir("c/d"), path.base("e"), path.ext("f.vs"), path.stem("g.vs")])
  io.println("$p ${path.isAbsolute(p)}")
  val sb = stringBuilder()
  sb.append("a")
  sb.appendLine()
  sb.clear()
  io.println("${sb.toString()} ${sb.len()} ${sb.isEmpty()}")
  val e: IoError = os.ioError(2, "p")
  io.println("${e.message()} ${e.detail} ${e.code} ${e.path}")
  os.exit(0)
}`)
	// a failing call must be handled (D4), and the error is the documented one
	expectError(t, "use fs\nfun main() { fs.readFile(\"a\") }", "unused Result")
	expectError(t, "use fs\nfun main() throws Panic { try fs.readFile(\"a\") }", "IoError")
}

// The prelude's Comparable, Equatable, Hashable and Display replace the
// structural behaviour of the ordering operators, `==`, map hashing and
// interpolation for a struct or sealed type; the numbers and strings
// implement Comparable so generic code can order them.
func TestOperatorTraits(t *testing.T) {
	expectClean(t, prelude+`
struct Version { major: i64, minor: i64 }
impl Comparable for Version {
  fun compareTo(other: Version): i64 = if (self.major != other.major) self.major.compareTo(other.major) else self.minor.compareTo(other.minor)
}
impl Display for Version { fun toString(): string = "v${self.major}.${self.minor}" }
struct Name { text: string }
impl Equatable for Name { fun equals(other: Name): bool = self.text.toLower() == other.text.toLower() }
impl Hashable for Name { fun hash(): i64 = self.text.toLower().len() }
struct Pair<T> { a: T, b: T }
impl<T: Display> Display for Pair<T> { fun toString(): string = "<${self.a}, ${self.b}>" }
sealed trait Shape
struct Circle : Shape { r: f64 }
struct Square : Shape { side: f64 }
impl Comparable for Shape { fun compareTo(other: Shape): i64 = area(self).compareTo(area(other)) }
fun area(s: Shape): f64 = when (s) {
  is Circle => 3.14 * s.r * s.r
  is Square => s.side * s.side
}
fun maxOf<T: Comparable>(a: T, b: T): T = if (a.compareTo(b) >= 0) a else b
fun main() {
  val a = Version(major: 1, minor: 10)
  val b = Version(major: 1, minor: 9)
  io.println("$a ${a > b} ${a <= b} ${a == b} ${[a, b].sorted()} ${[a, b].min()} ${[a, b].sortedBy(v => v)}")
  var m = mut [Name(text: "Ann"): 1]
  m[Name(text: "ANN")] = 2
  io.println("${m.len()} ${Name(text: "a") == Name(text: "A")} ${[Name(text: "x")].contains(Name(text: "X"))}")
  val shapes: List<Shape> = [Square(side: 2.0), Circle(r: 1.0)]
  io.println("${shapes.max()} ${Pair(a: a, b: b)} ${maxOf(3, 9)} ${maxOf("b", "a")} ${(5).compareTo(7)}")
}`)
	cases := []struct{ name, src, want string }{
		{"ordering needs Comparable", `struct P { x: i64 }
fun main() { val p = P(x: 1); io.println("${p < p}") }`, "implement 'Comparable' to order it"},
		{"min needs Comparable at the call", `struct P { x: i64 }
fun main() { io.println("${[P(x: 1)].min()}") }`, "'min' on 'List<P>' requires 'P' to implement 'Comparable'"},
		{"sorted needs Comparable", `struct P { x: i64 }
fun main() { io.println("${[P(x: 1)].sorted()}") }`, "implement 'Comparable'"},
		{"Equatable key needs Hashable", `struct K { s: string }
impl Equatable for K { fun equals(other: K): bool = true }
fun main() { val m = [K(s: "a"): 1]; io.println("${m.len()}") }`, "implements Equatable but not Hashable"},
		{"Hashable alone is fine", `struct K { s: string }
impl Hashable for K { fun hash(): i64 = 1 }
fun main() { val m = [K(s: "a"): 1]; io.println("${m.len()}") }`, ""},
		{"compareTo must match the trait", `struct P { x: i64 }
impl Comparable for P { fun compareTo(other: P): bool = true }`, "Comparable"},
		{"function fields cannot compare, Equatable makes them", `struct H { f: fun(i64): i64 }
impl Equatable for H { fun equals(other: H): bool = true }
fun main() { val h = H(f: x => x); io.println("${h == h}") }`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.want == "" {
				expectClean(t, prelude+c.src)
				return
			}
			expectError(t, prelude+c.src, c.want)
		})
	}
}

// #8: the expected type decides a literal's mutability; `mut` is for
// untyped literals, and a list literal where a Set is expected builds one.
func TestLiteralMutabilityInference(t *testing.T) {
	expectClean(t, prelude+`
fun fill(xs: MutableList<i64>) { xs.push(4) }
struct Bag { items: MutableList<string> = [], tags: MutableMap<string, i64> = [:] }
fun make(): MutableList<i64> = [1, 2]
fun main() {
  val a: MutableList<i64> = [1, 2, 3]
  a.push(9)
  val m: MutableMap<string, i64> = [:]
  m["y"] = 2
  val s: Set<i64> = [1, 2, 2]
  val ms: MutableSet<string> = ["a"]
  ms.add("b")
  fill([1, 2])
  val b = Bag()
  b.items.push("t")
  val untyped = mut [1]
  untyped.push(2)
  io.println("$a $m ${s.len()} ${ms.len()} ${b.items} ${make()} $untyped")
}`)
	for _, src := range []string{
		`fun main() { var r: MutableList<i64> = mut []; r.push(1); io.println("$r") }`,
		`fun main() { var r: MutableMap<string, i64> = mut [:]; r["a"] = 1; io.println("$r") }`,
	} {
		diags := checkSource(t, prelude+src)
		found := false
		for _, d := range diags.Items {
			if d.Severity == source.Warning && strings.Contains(d.Message, "redundant 'mut'") {
				found = true
			}
		}
		if !found || diags.HasErrors() {
			t.Errorf("expected a redundant-mut warning and no errors for %q, got:\n%s", src, diags.Render())
		}
	}
	expectError(t, prelude+`fun main() { val xs = []; io.println("$xs") }`, "val xs: List<i32> = []")
	expectError(t, prelude+`struct P { f: fun(): i64 }
fun main() { val s: Set<P> = [P(f: () => 1)]; io.println("${s.len()}") }`, "cannot be a map key or set element")
}

// `static fun` (D23): no receiver, called on the type — in struct bodies,
// extend blocks and traits (where generic code writes `T.parse(s)`).
func TestStaticFunctions(t *testing.T) {
	expectClean(t, prelude+`
struct Point {
  x: i64
  y: i64
  static fun origin(): Point = Point(x: 0, y: 0)
  static fun fromText(s: string): Point? {
    val parts = s.split(",")
    if (parts.len() != 2) return null
    val x = i64.parse(parts[0]) ?: return null
    val y = i64.parse(parts[1]) ?: return null
    Point(x: x, y: y)
  }
  impl Parsable {
    static fun parse(s: string): Point? = Point.fromText(s)
  }
}
extend Point {
  static fun unit(): Point = Point(x: 1, y: 1)
}
struct Stack<T> {
  items: MutableList<T> = []
  static fun of(x: T): Stack<T> {
    val s = Stack<T>()
    s.items.push(x)
    s
  }
}
fun parseAll<T: Parsable>(xs: List<string>): List<T?> = xs.map(x => T.parse(x))
fun main() {
  val p: Point? = Point.parse("3,4")
  val ns: List<i64?> = parseAll(["1", "x"])
  io.println("${Point.origin()} ${Point.unit()} $p $ns ${bool.parse("true")} ${Stack<string>.of("x").items}")
}`)
	cases := []struct{ name, src, want string }{
		{"top level", `static fun f() { }`, "a top-level function needs no marker"},
		{"self in static", `struct P { x: i64
  static fun make(): P = P(x: self.x) }`, "'self' is not available in a static function"},
		{"static called on a value", `struct P { x: i64
  static fun make(): P = P(x: 1) }
fun main() { val p = P(x: 1); p.make() }`, "call it on the type: 'P.make(...)'"},
		{"method called on the type", `struct P { x: i64
  fun m(): i64 = 1 }
fun main() { P.m() }`, "call it on a value, not on the type"},
		{"unknown static", `struct P { x: i64 }
fun main() { P.nothing() }`, "no static function 'nothing' on type 'P'"},
		{"impl must say static", `trait F { static fun make(): Self }
struct P { x: i64 }
impl F for P { fun make(): P = P(x: 1) }`, "declare it 'static fun'"},
		{"impl must not say static", `trait F { fun m(): i64 }
struct P { x: i64 }
impl F for P { static fun m(): i64 = 1 }`, "cannot be 'static'"},
		{"sealed trait", `sealed trait S { static fun z(): i64 = 1 }
struct A : S { }`, "a sealed trait cannot declare a static function"},
		{"not object safe", `trait F { static fun make(): Self }
struct P { x: i64 }
impl F for P { static fun make(): P = P(x: 1) }
fun main() { val f: F = P(x: 1); io.println("$f") }`, "function 'make' is static"},
		{"generic needs type args", `struct S<T> { x: T
  static fun z(): i64 = 0 }
fun main() { io.println("${S.z()}") }`, "write the type arguments"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+c.src, c.want)
		})
	}
}

// A top-level impl for a struct of the same module that the inline form
// can express is a lint with an autofix (D23); other cases stay silent.
func TestInlinableImplLint(t *testing.T) {
	hasLint := func(src string) (bool, *source.Fix) {
		diags := checkSource(t, prelude+src)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors:\n%s", diags.Render())
		}
		for _, d := range diags.Items {
			if strings.Contains(d.Message, "can be written inside the body") {
				return true, d.Fix
			}
		}
		return false, nil
	}
	if ok, fix := hasLint(`trait Show { fun show(): string }
struct P { x: i64 }
impl Show for P { fun show(): string = "p" }`); !ok || fix == nil || len(fix.Edits) != 2 || fix.Title != "Move into the body of 'P'" {
		t.Errorf("expected the lint with a two-edit fix, got %v %+v", ok, fix)
	}
	if ok, _ := hasLint(`trait Show { fun show(): string }
struct Box<T> { x: T }
impl<T> Show for Box<T> { fun show(): string = "box" }`); !ok {
		t.Errorf("a generic impl with the struct's own parameters is inlinable")
	}
	for name, src := range map[string]string{
		"extra bound": `trait Show { fun show(): string }
struct Box<T> { x: T }
impl<T: Show> Show for Box<T> { fun show(): string = self.x.show() }`,
		"foreign type": `trait Show { fun show(): string }
impl Show for i64 { fun show(): string = "n" }`,
		"already inline": `trait Show { fun show(): string }
struct P { x: i64; impl Show { fun show(): string = "p" } }`,
		"specific instance": `trait Show { fun show(): string }
struct Box<T> { x: T }
impl Show for Box<i64> { fun show(): string = "box" }`,
		"error declaration": `error E { code: i64 }`,
	} {
		if ok, _ := hasLint(src); ok {
			t.Errorf("%s: the lint should not fire", name)
		}
	}
}
