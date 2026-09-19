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
fun main() { val m = ["a": 1]; val v: i64 = m.get("a") }`, "supply a fallback with '?:'"},
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
		{"D40 function type effects declared", prelude + `
fun g(): i32 throws = 1
fun main() { val h: fun(): i32 throws = g; io.println("${h()}") }`, "error type must be declared"},
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
  val p: string = path.join(path.join("a", "b"), path.dir("c/d"), path.base("e"), path.ext("f.vs"), path.stem("g.vs"))
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
  m.set(Name(text: "ANN"), 2)
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
  m.set("y", 2)
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
		`fun main() { var r: MutableMap<string, i64> = mut [:]; r.set("a", 1); io.println("$r") }`,
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
    val x = i64.parse(parts.atOrPanic(0)) ?: return null
    val y = i64.parse(parts.atOrPanic(1)) ?: return null
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

// #5: `r.ok` / `r.err` are the tag tests of a Result spelled as properties
// and smart-cast like `r is Ok`; `when (val r = e)` names the subject.
func TestResultProperties(t *testing.T) {
	expectClean(t, prelude+`
error Bad { text: string }
fun parse(s: string): i64 throws Bad = s.toInt() ?: throw Bad(text: s)
fun describe(s: string): string = when (val r = parse(s)) {
  is Ok  => "ok ${r * 2}"
  is Err => "err ${r.text}"
}
fun main() {
  val r = parse("42")
  if (r.ok) io.println("${r + 1}") else io.println(r.message())
  if (r.err) io.println(r.text)
  if (!r.ok) io.println(r.text) else io.println("${r - 1}")
  when (val n = "5".toInt()) {
    null   => io.println("none")
    is i64 => io.println("${n + 1}")
  }
  io.println("${describe("1")} ${parse("7").ok}")
}`)
	expectError(t, prelude+`struct P { x: i64 }
fun main() { val p = P(x: 1); io.println("${p.ok}") }`, "has no field 'ok'")
	expectError(t, prelude+`fun main() { when (val n = 1) { else => io.println("$m") } }`, "unknown name 'm'")
}

// Variadic parameters (`parts: string...`) collect the trailing arguments
// into a List; `xs...` passes a list whole.
func TestVariadics(t *testing.T) {
	expectClean(t, prelude+`
fun join(sep: string, parts: string...): string = parts.join(sep)
fun sum(xs: i64...): i64 {
  var t: i64 = 0
  loop (x in xs) { t += x }
  t
}
trait Fmt { fun fmt(args: string...): string }
struct P { x: i64
  impl Fmt { fun fmt(args: string...): string = "${self.x} ${args.len()}" }
  static fun of(xs: i64...): P = P(x: xs.len())
}
fun main() {
  val parts = ["x", "y"]
  io.println("${join("/", "a", "b")} ${join("-")} ${join(",", parts...)} ${join("+", parts: parts)}")
  io.println("${sum()} ${sum(1, 2, 3)} ${sum([4, 5]...)} ${P(x: 1).fmt("a")} ${P.of(1, 2).x}")
}`)
	cases := []struct{ name, src, want string }{
		{"not last", `fun f(xs: i64..., y: i64) { }`, "must be the last one"},
		{"default", `fun f(xs: i64... = [1]) { }`, "cannot have a default"},
		{"spread to plain", `fun f(x: i64) { }
fun main() { f([1]...) }`, "has none here"},
		{"spread plus more", `fun f(xs: i64...) { }
fun main() { f([1]..., 2) }`, "must be the only argument"},
		{"element type", `fun f(xs: i64...) { }
fun main() { f(1, "two") }`, "expected 'i64', found 'string'"},
		{"impl must match", `trait Fmt { fun fmt(args: string...): string }
struct P { x: i64 }
impl Fmt for P { fun fmt(args: List<string>): string = "" }`, "must be variadic exactly as in trait"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+c.src, c.want)
		})
	}
}

// std/time, std/random, byte I/O, io.readAll, number formatting, and the
// bitwise operators the generator is built from.
func TestTimeRandomBytesFormatting(t *testing.T) {
	expectClean(t, `
use io
use fs
use time
use random

fun main() throws IoError {
  val now: i64 = time.now()
  val d: time.DateTime = time.utc(now)
  val l: time.DateTime = time.local(now)
  val sw = time.Stopwatch.start()
  io.println("$d ${d.date()} ${l.time()} ${d.weekday} ${sw.elapsedMillis()} ${sw.elapsedSeconds()} ${time.monotonic()} ${time.monotonicNanos()}")
  random.seed(1)
  val n: i64 = random.range(1, 7)
  val x: f64 = random.float()
  val b: bool = random.boolean()
  val p: string? = random.pick(["a", "b"])
  val xs = mut [1, 2, 3]
  random.shuffle(xs)
  var rng = random.Rng.seeded(7)
  io.println("$n $x $b $p $xs ${rng.range(0, 10)} ${rng.nextU64()} ${random.nextU64()}")
  val bytes: List<u8> = try fs.readBytes("a.bin")
  try fs.writeBytes("a.bin", bytes)
  try fs.appendBytes("a.bin", [1, 2])
  val all: string = io.readAll()
  io.println("${bytes.len()} ${all.len()} ${(2.0 / 3.0).toFixed(2)} ${(255).toString(radix: 16)} ${(7 as u64).toString(radix: 2)} ${(1.5 as f32).toFixed(1)}")
  val flags: u8 = 0b1010
  val nested: List<List<i64>> = [[1]]
  val m: Map<string, List<Set<i64>>> = [:]
  io.println("${flags & 3} ${flags | 1} ${flags ^ 0xFF} ${~flags} ${flags << 2} ${(-8) >> 1} ${(1 as u64) << 63} ${nested.len()} ${m.len()}")
}`)
	cases := []struct{ name, src, want string }{
		{"bitwise on floats", `fun main() { io.println("${1.5 & 2.0}") }`, "only defined for integers"},
		{"not on bool", `fun main() { io.println("${~true}") }`, "only defined for integers"},
		{"shift count", `fun main() { io.println("${1 << 2.0}") }`, "shift count must be an integer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+c.src, c.want)
		})
	}
	// D25: a mutable collection passes where its immutable form is expected
	expectClean(t, prelude+`
fun total(xs: List<i64>): i64 = xs.fold(0, (a, b) => a + b)
fun first<T>(xs: List<T>): T? = xs.first()
fun main() {
  val xs = mut [1, 2]
  val m: MutableMap<string, i64> = [:]
  val ro: Map<string, i64> = m
  io.println("${total(xs)} ${first(xs)} ${ro.len()}")
}`)
}

// D25 (v0.24): brackets are collection literals only. The old index forms
// are errors that carry a mechanical fix; the methods that replace them
// type as documented; reads are copies and `refOrPanic` is the pointer.
func TestIndexingIsMethodsOnly(t *testing.T) {
	cases := []struct{ src, msg, fix string }{
		{`fun main() { val xs = [1]; io.println("${xs[0]}") }`, "'xs[0]' is not indexing", "xs.atOrPanic(0)"},
		{`fun main() { val m = ["a": 1]; io.println("${m["a"]}") }`, "'m[\"a\"]' is not indexing", `m.get("a")`},
		{`fun main() { var xs: MutableList<i64> = [1]; xs[0] = 2; io.println("$xs") }`, "is not index assignment", "xs.set(0, 2)"},
		{`fun main() { var xs: MutableList<i64> = [1]; xs[0] += 2; io.println("$xs") }`, "is not index assignment", "xs.set(0, xs.atOrPanic(0) + 2)"},
		{`fun main() { var m: MutableMap<string, i64> = [:]; m["k"] = 2; io.println("$m") }`, "is not index assignment", `m.set("k", 2)`},
	}
	for _, c := range cases {
		diags := checkSource(t, prelude+c.src)
		found := false
		for _, d := range diags.Items {
			if d.Severity == source.Error && strings.Contains(d.Message, c.msg) {
				found = true
				if d.Fix == nil || len(d.Fix.Edits) != 1 || d.Fix.Edits[0].NewText != c.fix {
					t.Errorf("%q: expected a fix to %q, got %+v", c.src, c.fix, d.Fix)
				}
			}
		}
		if !found {
			t.Errorf("%q: expected %q, got:\n%s", c.src, c.msg, diags.Render())
		}
	}
	expectClean(t, prelude+`
struct C { n: i64 = 0
  mut fun bump() { self.n += 1 } }
fun main() {
  val xs = [1, 2]
  val a: i64? = xs.at(5)
  val b: i64 = xs.atOrPanic(-1)
  val c: i64 = xs.atOrDefault(9, 0)
  var ys: MutableList<i64> = [1]
  ys.set(0, 3)
  val cs: MutableList<C> = [C()]
  cs.refOrPanic(0).bump()
  cs.refOrPanic(0).n = 4
  val p = cs.refOrPanic(0)
  p.n += 1
  var copy = cs.atOrPanic(0)
  copy.n = 9
  val m = ["k": 1]
  val d: i64? = m.get("k")
  val e: i64 = m.getOrPanic("k")
  val f: i64 = m.getOrDefault("z", 0)
  var mm: MutableMap<string, i64> = [:]
  mm.set("k", 1)
  io.println("$a $b $c $ys ${cs.atOrPanic(0).n} $d $e $f $mm")
}`)
	expectError(t, prelude+`fun main() { val xs = [1]; xs.set(0, 2) }`, "immutable List")
	expectError(t, prelude+`fun main() { val m = ["a": 1]; m.set("a", 2) }`, "immutable Map")
	expectError(t, prelude+`fun main() { val xs = [1]; val n: i64 = xs.at(0); io.println("$n") }`, "supply a fallback with '?:'")
	expectError(t, prelude+`fun main() { val s = "abc"; io.println("${s[0]}") }`, "strings are not indexable")
}
// The sealed-`else` lint (D13, scoped 2026-09-18): an `else` standing for
// exactly one missing variant warns (an enumeration a new variant would
// fall into); an `else` with every variant covered is dead and carries a
// removal fix; one-variant extraction with several variants left is silent.
func TestSealedElseLint(t *testing.T) {
	src := prelude + `
sealed trait S
struct A : S { }
struct B : S { }
struct C : S { }
fun f(s: S): i64 = when (s) {
  is A => 1
  is B => 2
  else => 3
}
fun g(s: S): i64 = when (s) {
  is A => 1
  is B => 2
  is C => 3
  else => 4
}
fun h(s: S): i64? = when (s) {
  is A => 1
  else => null
}
fun main() { io.println("${f(A())} ${g(B())} ${h(C())}") }`
	diags := checkSource(t, src)
	var fixes []bool
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "D13 lint") {
			fixes = append(fixes, d.Fix != nil)
		}
	}
	if diags.HasErrors() || len(fixes) != 2 || fixes[0] || !fixes[1] {
		t.Errorf("expected two warnings (f without a fix, g with one) and none for h; got:\n%s", diags.Render())
	}
}

// A collection literal passed to a generic constructor or function takes
// its mutability from the still-generic parameter type, so the literal
// needs no `mut` and the element type still infers the type parameter.
func TestLiteralMutabilityFromGenericParam(t *testing.T) {
	expectClean(t, prelude+`
struct Stack<T> { items: MutableList<T>
  mut fun push(x: T) { self.items.push(x) } }
fun fill<T>(xs: MutableList<T>, x: T): i64 { xs.push(x); xs.len() }
fun keyed<K, V>(m: MutableMap<K, V>, k: K, v: V): i64 { m.set(k, v); m.len() }
fun main() {
  var s = Stack(items: [1, 2])
  s.push(3)
  io.println("${s.items.len()} ${fill(["a"], "b")} ${keyed(["k": 1], "j", 2)}")
}`)
}

// `xs.ref(i)?.m()` reaches the element in place through the nullable
// pointer, so a `mut fun` sticks; `at`/`first`/`last` return copies, on
// which a value-returning method is fine and a unit `mut fun` an error.
func TestSafeCallThroughRef(t *testing.T) {
	expectClean(t, prelude+`
struct C { n: i64 = 0
  mut fun bump() { self.n += 1 }
  fun show(): string = "${self.n}" }
fun make(f: fun(i64): i64): List<C> = [C(n: f(1))]
fun main() {
  val cs: MutableList<C> = [C()]
  cs.ref(0)?.bump()
  cs.ref(-1)?.bump()
  cs.ref(9)?.bump()
  val s: string? = cs.at(0)?.show()
  val m = ["k": C()]
  val t: string? = m.get("k")?.show()
  val u: string? = make(x => x + 1).at(0)?.show()
  io.println("${cs.atOrPanic(0).n} $s $t $u")
}`)
	expectError(t, prelude+`fun main() { val xs = [1]; xs.at(0).abs() }`, "may be null; use '?.'")
	for _, c := range []struct{ src, want string }{
		{`fun main() { val cs: MutableList<C> = [C()]; cs.at(0)?.bump() }`, "reach the element itself with 'cs.ref(0)'"},
		{`fun main() { val cs: MutableList<C> = [C()]; cs.first()?.bump() }`, "would change a temporary copy"},
		{`fun main() { val cs: MutableList<C> = [C()]; cs.atOrPanic(0).bump() }`, "reach the element itself with 'cs.refOrPanic(0)'"},
		{"fun make(): C = C()\nfun main() { make().bump() }", "would change a temporary copy of 'C' that is then discarded"},
		{`fun main() { val cs = [C()]; cs.ref(0)?.bump() }`, "'ref' needs a MutableList"},
		{`fun main() { val cs: MutableList<C> = [C()]; val p = &cs.atOrPanic(0); p.n = 1 }`, "address of a copy of the element"},
	} {
		expectError(t, prelude+"struct C { n: i64 = 0\n  mut fun bump() { self.n += 1 } }\n"+c.src, c.want)
	}
}

// D5/D25 (v0.27): `?.` reaches places. `m.refOrPanic(k)` and `m.ref(k)?.`
// point at the map entry itself (reads are copies); a nullable variable or
// field is written through after its null test (`if (p != null) p.n = 5`);
// and assignment through `?.` (`x?.f = v`, `x?.f op= v`) writes only when
// the receiver is present. Immutable collections and `val` structs stay
// closed; a write into a copy is an error.
func TestPlacesThroughNullables(t *testing.T) {
	expectClean(t, prelude+`
struct C { n: i64 = 0
  mut fun bump() { self.n += 1 } }
struct Box { c: C? = null }
fun main() {
  val m: MutableMap<string, C> = ["a": C()]
  m.ref("a")?.bump()
  m.ref("a")?.n += 1
  m.ref("zz")?.n = 9
  m.refOrPanic("a").bump()
  m.refOrPanic("a").n = 3
  val p = m.refOrPanic("a")
  p.n += 1
  val xs: MutableList<C> = [C()]
  xs.ref(0)?.n = 7
  xs.ref(0)?.n *= 2
  xs.ref(-1)?.bump()
  loop (&c in xs) c.bump()
  loop ((_, &c) in m) c.n += 1
  var q: C? = C()
  if (q != null) {
    q.n = 5
    q.bump()
  }
  q?.n += 1
  var b = Box(c: C())
  b.c?.n = 8
  b.c?.bump()
  var r: (*C)? = xs.refOrPanic(0)
  r?.n = 1
  val q2 = xs.ref(0)
  q2?.n = 2
  io.println("$m $xs $q ${b.c}")
}`)
	for _, c := range []struct{ src, want string }{
		{`fun main() { val f = ["b": C()]; f.ref("b")?.bump() }`, "'ref' needs a MutableMap"},
		{`fun main() { val f: MutableMap<string, C> = ["b": C()]; f.get("b")?.n = 1 }`, "reach the element itself with 'f.ref(\"b\")'"},
		{`fun main() { val f: MutableMap<string, C> = ["b": C()]; f.getOrPanic("b").bump() }`, "reach the element itself with 'f.refOrPanic(\"b\")'"},
		{`fun main() { val xs = [C()]; loop (&c in xs) c.bump() }`, "needs a MutableList"},
		{`fun main() { val m = ["a": C()]; loop ((k, &v) in m) v.bump() }`, "needs a MutableMap"},
		{`fun main() { val m: MutableMap<string, C> = ["a": C()]; loop ((&k, v) in m) v.bump() }`, "'&' goes on the value"},
		{`fun main() { val xs: MutableList<C> = [C()]; xs.atOrPanic(0).n = 1 }`, "reach the element itself with 'xs.refOrPanic(0)'"},
		{`fun main() { val ns: MutableList<i64> = [1]; loop (&n in ns) n += 1 }`, "write '*n' to change the value it points to"},
		{`fun main() { val ns: MutableList<i64> = [1]; ns.atOrPanic(0) += 1 }`, "assign through '*ns.refOrPanic(0)'"},
		{`fun make(): C? = C()
fun main() { make()?.n = 1 }`, "into a temporary value of type 'C' has no effect"},
		{`fun main() { val p: C? = C(); p?.n = 1 }`, "it is a 'val'"},
		{`fun main() { var p: C = C(); p?.n = 1 }`, "'?.' on a non-nullable value"},
	} {
		expectError(t, prelude+"struct C { n: i64 = 0\n  mut fun bump() { self.n += 1 } }\n"+c.src, c.want)
	}
}

// D40 (v0.24): a bare `throws` on a trait method leaves the error to each
// impl (an implicit associated `Error`), inferred from impl bodies; an
// impl that cannot fail has `Never`. Projections are spelled with a dot.
func TestTraitImplDefinedError(t *testing.T) {
	src := prelude + `
error HttpError { status: i64 }
error Missing { name: string }
trait Fetcher {
  fun fetch(url: string): string throws
}
struct Http { impl Fetcher {
  fun fetch(url: string): string throws {
    if (url.startsWith("bad")) throw HttpError(status: 500)
    "http:$url"
  } } }
struct Memory { data: Map<string, string>
  impl Fetcher {
    fun fetch(url: string): string throws Missing = self.data.get(url) ?: throw Missing(name: url) } }
struct Always { impl Fetcher { fun fetch(url: string): string throws = url } }
struct Pinned { impl Fetcher {
  type Error = HttpError
  fun fetch(url: string): string throws = url } }
fun load<F: Fetcher>(f: F, url: string): string throws F.Error = try f.fetch(url)
fun main() {
  val h: Result<string, HttpError> = load(Http(), "x")
  val m: Result<string, Missing> = load(Memory(data: [:]), "k")
  val a = load(Always(), "y")
  val p: Result<string, HttpError> = load(Pinned(), "z")
  io.println("$h $m $a $p")
}`
	expectClean(t, src)
	expectError(t, prelude+`
trait Fetcher { fun fetch(url: string): string throws }
struct A { impl Fetcher { fun fetch(url: string): string throws = url } }
fun main() { val f: Fetcher = A(); io.println("${f.fetch("x")}") }`, "bare 'throws'")
	expectError(t, prelude+`
trait Iter2 { type Item
  fun next(): Item? }
fun f<I: Iter2>(i: I): I::Item? = i.next()`, "'::' is not Veles")
	expectClean(t, prelude+`
trait Iter2 { type Item
  fun next(): Item? }
fun f<I: Iter2>(i: I): I.Item? = i.next()
struct Ones { impl Iter2 { type Item = i64
  fun next(): Self.Item? = 1 } }
fun main() { io.println("${f(Ones())}") }`)
}

// Found by the calculator program: an arm body may be an assignment
// (`cond => self.pos += 1`), a loop body may be a bare statement like an
// `if` body, and `x = Node(child: &x)` is warned about (the pointer would
// name the variable being assigned, making the value contain itself).
func TestCalculatorFindings(t *testing.T) {
	expectClean(t, prelude+`
struct Cursor { pos: i64 = 0
  mut fun step(b: u8) {
    when {
      b == 32 => self.pos += 1
      else => self.pos = 0
    }
    loop (self.pos < 3) self.pos += 1
  } }
fun main() { var c = Cursor(); c.step(32); io.println("${c.pos}") }`)
	diags := checkSource(t, prelude+`
sealed trait T
struct Leaf : T { }
struct Node : T { child: *T }
fun main() {
  var t: T = Leaf()
  t = Node(child: &t)
  val old = t
  t = Node(child: &old)
  io.println("${t is Node}")
}`)
	n := 0
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "would contain itself") {
			n++
		}
	}
	if diags.HasErrors() || n != 1 {
		t.Errorf("expected exactly one self-address warning, got:\n%s", diags.Render())
	}
}

// D46 (v0.24): an eager collection operation given a throwing function is
// itself fallible — `Result<T, E>` — and stops at the first failure;
// `sortedBy` and `getOrPut` still refuse (their function runs outside a
// loop the adapter controls). `'x'` is a u8 byte literal (D18).
func TestThrowingAdaptersAndByteLiterals(t *testing.T) {
	expectClean(t, prelude+`
error Bad { n: i64 }
fun check(x: i64): i64 throws Bad = if (x < 0) throw Bad(n: x) else x * 10
fun all(xs: List<i64>): List<i64> throws Bad = try xs.map(x => try check(x))
fun main() {
  val r: Result<List<i64>, Bad> = [1, -2].map(x => try check(x))
  val k: Result<List<i64>, Bad> = [1, 2].filter(x => try check(x) > 15)
  val s: Result<i64, Bad> = [1, 2].fold(0, (a, x) => a + (try check(x)))
  val f: Result<i64?, Bad> = [1, 2].find(x => try check(x) == 20)
  val e: Result<(), Bad> = [1].forEach(x => { val _ = try check(x) })
  val mv: Result<Map<string, i64>, Bad> = ["a": 1].mapValues(v => try check(v))
  val plain: List<i64> = [1, 2].map(x => x + 1)
  val q: u8 = '"'
  val d = 'a' + 1
  io.println("${all([1])} ${r.ok} ${k.ok} ${s.ok} ${f.ok} ${e.ok} ${mv.ok} $plain $q $d ${'0' == 48}")
}`)
	expectError(t, prelude+`
error Bad { n: i64 }
fun key(x: i64): i64 throws Bad = x
fun main() { io.println("${[2, 1].sortedBy(x => try key(x))}") }`, "cannot be passed here")
	expectError(t, prelude+`fun main() { val b = 'é'; io.println("$b") }`, "one ASCII character")
}

// `(a, b) = (b, a)`: D37 destructuring as an assignment. The right side is
// evaluated once, then each place is stored in order.
func TestTupleAssignment(t *testing.T) {
	expectClean(t, prelude+`
struct P { x: i64; y: i64 }
fun pair(): (i64, i64) = (1, 2)
fun main() {
  var a = 1
  var b = 2
  (a, b) = (b, a)
  (a, b) = pair()
  var xs = mut [1, 2, 3]
  (*xs.refOrPanic(0), *xs.refOrPanic(2)) = (xs.atOrPanic(2), xs.atOrPanic(0))
  var p = P(x: 1, y: 2)
  (p.x, p.y) = (p.y, p.x)
  var s: string? = null
  var n = 0
  (s, n) = ("hi", 5)
  io.println("$a $b $xs $p ${s.len()} $n")
}`)
	expectError(t, prelude+`fun main() { var a = 1; var b = 2; (a, b) = (1, 2, 3); io.println("$a $b") }`, "tuple has 3 elements but 2 places")
	expectError(t, prelude+`fun main() { var a = 1; val c = 2; (a, c) = (2, 3); io.println("$a $c") }`, "it is a 'val'")
	expectError(t, prelude+`fun main() { var a = 1; var b = 2; (a, b) += (1, 1); io.println("$a $b") }`, "compound assignment cannot target a tuple")
	expectError(t, prelude+`fun main() { var a = 1; var b = 2; (a, b) = 5; io.println("$a $b") }`, "only tuples destructure positionally")
	expectError(t, prelude+`fun main() { var a = 1; var b = 2; (a, b) = ("x", 1); io.println("$a $b") }`, "expected 'i64', found 'string'")
}

// Statics the prelude adds to a built-in generic type (`extend<T>
// MutableList<T> { static fun repeat ... }`) are called with the type
// arguments written, like a generic struct's statics.
func TestBuiltinStatics(t *testing.T) {
	expectClean(t, prelude+`
fun main() {
  val flags = MutableList<bool>.repeat(false, 3)
  flags.fill(true)
  val slots = MutableList<i64?>.repeat(null, 2)
  val rows = MutableList<MutableList<i64>>.make(2, _ => [])
  rows.atOrPanic(0).push(1)
  flags.swap(0, 2)
  io.println("$flags $slots $rows")
}`)
	expectError(t, prelude+`fun main() { val xs = MutableList.repeat(0, 3); io.println("$xs") }`, "'MutableList' is generic; write the type arguments")
	expectError(t, prelude+`fun main() { val xs = MutableList<i64>.nope(3); io.println("$xs") }`, "no static function 'nope'")
	// `repeat` duplicates its value, so the element type must be Sendable (D35):
	// a mutable collection, a pointer or a closure would be shared by every slot
	expectError(t, prelude+`fun main() { val xs = MutableList<MutableList<i64>>.repeat(mut [1], 2); io.println("$xs") }`, "requires 'MutableList<i64>' to implement 'Sendable'")
	expectError(t, prelude+`struct N { v: i64 }
fun main() { val xs = MutableList<*N>.repeat(&N(v: 1), 2); io.println("$xs") }`, "to implement 'Sendable'")
	expectClean(t, prelude+`struct P { x: i64; y: i64 }
fun main() { val xs = MutableList<P>.repeat(P(x: 1, y: 2), 2); val ys = MutableList<List<i64>>.repeat([1], 2); io.println("$xs $ys") }`)
	expectError(t, prelude+`struct N { v: i64 }
impl Sendable for N { }
fun main() { io.println("x") }`, "cannot be implemented by hand")
}

// Deque and PriorityQueue (prelude/collections.vs) are reference types: a
// `val` binding can grow them and a callee shares the caller's one.
func TestPreludeCollections(t *testing.T) {
	expectClean(t, prelude+`
struct Job {
  cost: i64
  impl Comparable {
    fun compareTo(other: Job): i64 = self.cost.compareTo(other.cost)
  }
}
fun drain(q: Deque<i64>): i64 {
  var n = 0
  loop {
    q.removeFirst() ?: break
    n += 1
  }
  n
}
fun main() {
  val q = deque<i64>()
  q.addLast(1)
  q.addFirst(0)
  val typed: Deque<string> = deque()
  typed.addLast("x")
  loop (x in q) io.println("$x")
  io.println("$q ${q.first()} ${q.last()} ${q.at(-1)} ${q.len()} ${drain(q)} ${q.isEmpty()} $typed")
  val jobs = priorityQueue<Job>()
  jobs.push(Job(cost: 3))
  val words = priorityQueueBy<string>((a, b) => b.compareTo(a))
  words.push("a")
  val holes = deque<i64?>()
  holes.addLast(null)
  io.println("${holes.len()} ${holes.removeFirst()}")
  io.println("${jobs.pop()?.cost} ${jobs.peek()} ${words.pop()} ${words.len()}")
}`)
}

// A struct that contains itself by value is a D31 error; the predicates
// that walk fields (hashable, comparable, sendable) must not recurse
// forever on it — this fuzz-found shape names the struct `string` so a
// `Map<string, ...>` field elsewhere resolves the key to the struct.
func TestRecursiveStructDoesNotHang(t *testing.T) {
	expectError(t, prelude+`
struct string { op: string }
struct Env { vars: Map<string, f64> }
fun main() { }
`, "infinite size")
	expectError(t, prelude+`
struct S { s: S; n: i64 }
fun main() {
  val a: S? = null
  val b: S? = null
  io.println("${a == b}")
  scope { async work(a) }
}
fun work(s: S?) { }
`, "infinite size")
}

// A compound assignment reads and writes one place: the index, key or
// pointer that locates it is evaluated once (hoistPlace).
func TestCompoundAssignmentEvaluatesPlaceOnce(t *testing.T) {
	prog := checkProgram(t, prelude+`
var calls = 0
fun idx(): i64 { calls += 1; 0 }
fun key(): string { calls += 1; "a" }
fun main() {
  var xs = mut [1, 2]
  *xs.refOrPanic(idx()) += 1
  var m: MutableMap<string, i64> = ["a": 1]
  *m.refOrPanic(key()) *= 2
  io.println("$calls $xs $m")
}`)
	stmts := prog.Main.Body.Stmts
	// each compound assignment is preceded by the temporaries for its
	// collection and index/key, and the Assign itself names no call
	var assigns int
	for _, s := range stmts {
		if a, ok := s.(*Assign); ok {
			assigns++
			if containsCall(a.Target) {
				t.Errorf("compound target still evaluates a call: %T", a.Target)
			}
		}
	}
	if assigns != 2 {
		t.Fatalf("want 2 assignments, got %d", assigns)
	}
}

// checkProgram checks src and returns the lowered program.
func checkProgram(t *testing.T, src string) *Program {
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
	prog := Check(pkg, diags, false)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors:\n%s", diags.Render())
	}
	return prog
}

func containsCall(e Expr) bool {
	switch e := e.(type) {
	case *Call:
		return true
	case *Deref:
		return containsCall(e.X)
	case *Builtin:
		for _, a := range e.Args {
			if containsCall(a) {
				return true
			}
		}
	case *BlockExpr:
		return true
	}
	return false
}

// A field behind a pointer that a call returns is writable (D11: through
// a pointer, mutability is not gated by a binding); a field of a value a
// call returns is a temporary.
func TestFieldWriteThroughCallResult(t *testing.T) {
	expectClean(t, prelude+`
struct Acc { n: i64 = 0 }
struct H { p: *Acc }
fun ptrOf(h: H): *Acc = h.p
fun main() {
  val h = H(p: &Acc())
  ptrOf(h).n = 5
  ptrOf(h).n += 1
  io.println("${h.p.n}")
}`)
	expectError(t, prelude+`
struct Acc { n: i64 = 0 }
fun make(): Acc = Acc()
fun main() { make().n = 1 }`, "field of a temporary value")
}

// Collections compare by content (D25, v0.26) whenever their elements do,
// across the mutable and immutable views; an immutable collection of
// hashable elements is a key, a mutable one is not.
func TestCollectionEquality(t *testing.T) {
	expectClean(t, prelude+`
struct P { tags: List<string> }
fun main() {
  val xs: MutableList<i64> = [1]
  val ys = [1]
  val m: MutableMap<string, List<i64>> = [:]
  var keyed: MutableMap<List<i64>, string> = [:]
  keyed.set([1], "one")
  val s: Set<Set<i64>> = [[1]]
  io.println("${xs == ys} ${ys == xs} ${m == ["a": [1]]} ${P(tags: []) == P(tags: ["a"])} ${keyed.get([1])} ${s.contains([1])}")
}`)
	expectError(t, prelude+`
fun main() {
  var m: MutableMap<MutableList<i64>, i64> = [:]
  io.println("${m.len()}")
}`, "MutableList can change after it is stored")
	expectError(t, prelude+`
fun main() {
  val a = [() => 1]
  io.println("${a == a}")
}`, "cannot be compared")
}

// `?.` on a value that is nullable twice (`xs.at(i)` on a `List<T?>`)
// flattens the levels: null at either level is null (D30).
func TestSafeAccessOnNestedNullable(t *testing.T) {
	expectClean(t, prelude+`
struct W { inner: i64?; fun show(): string = "w" }
fun main() {
  val ws: List<W?> = [null, W(inner: 3)]
  val m: Map<string, W?> = ["a": null]
  val deep: ((i64?)?)? = 4
  io.println("${ws.at(1)?.inner} ${ws.at(0)?.show()} ${m.get("a")?.inner} ${deep?.abs()}")
}`)
}

// A reference into a collection goes stale when the collection is grown
// or rearranged (D25); the lint warns when such a reference is still used
// afterwards, and when a `loop (&x in xs)` body changes `xs`.
func TestStaleReferenceLint(t *testing.T) {
	src := prelude + `
struct C { n: i64 = 0
  mut fun bump() { self.n += 1 } }
fun main() {
  val xs: MutableList<C> = [C()]
  val p = xs.refOrPanic(0)
  xs.push(C())
  p.n = 1
  val q = xs.refOrPanic(0)
  q.n = 2
  xs.push(C())
  val s = xs.refOrPanic(0)
  xs.fill(C())
  s.n = 4
  loop (&c in xs) { if (c.n > 100) xs.push(C()) }
  loop (&c in xs) c.bump()
  val m: MutableMap<string, C> = ["a": C()]
  loop ((k, &c) in m) { if (k == "a") m.remove("b"); c.bump() }
  val ys: MutableList<C> = [C()]
  val u = xs.refOrPanic(0)
  ys.push(C())
  u.n = 9
  io.println("${xs.len()} ${ys.len()}")
}`
	diags := checkSource(t, src)
	var got []string
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "may move or reorder") || strings.Contains(d.Message, "walks it by reference") {
			line, _ := d.Span.File.Position(d.Span.Start)
			got = append(got, itoa(line))
		}
	}
	if want := "8,16,19"; strings.Join(got, ",") != want {
		t.Errorf("stale-reference warnings on lines %v, want %s:\n%s", got, want, diags.Render())
	}
}

// A declaration named like a built-in generic (`struct Task`) shadows it in
// type position as it already did at call sites; the built-in stays
// reachable from other modules (D24: names in scope win over the prelude).
func TestUserTypeShadowsBuiltinGeneric(t *testing.T) {
	expectClean(t, prelude+`
struct Task { text: string }
struct List { n: i64 }
fun first(xs: MutableList<Task>): Task? = xs.first()
fun main() {
  val xs: MutableList<Task> = [Task(text: "a")]
  val l = List(n: 1)
  io.println("${first(xs)?.text} ${l.n} ${xs.len()}")
}`)
	expectError(t, prelude+`
fun main() { val t: Task<i64> = 1 }`, "expected 'Task<i64>', found 'i64'")
}

// `try f().m()` applies `try` to the whole chain; the message says how to
// unwrap first instead of reporting a missing method on the Result.
func TestTryCoversChainHint(t *testing.T) {
	expectError(t, prelude+`
error E { message: string }
fun f(): string throws E = "a b"
fun main() throws E {
  val n = try f().split(" ").len()
  io.println("$n")
}`, "write '(try f()).split(...)' to unwrap first")
}

// One `use` may list several imports (M6 addendum); each behaves as its
// own import, including the "already imported" check.
func TestUseList(t *testing.T) {
	expectClean(t, `use io, os,
  path as p
fun main() {
  io.println("${p.base(os.program())}")
}`)
	expectError(t, `use io, os, io
fun main() { io.println("x") }`, "'io' is already imported in this file")
}

// Construction is by field name; a bare identifier is a pun for a field of
// the same name, anything else bare is an error with a fix, and `x: x` is
// a warning with a fix (D28 addendum).
func TestConstructorPuns(t *testing.T) {
	expectClean(t, prelude+`
struct H { file: string; size: i64 = 0 }
fun main() {
  val file = "a"
  val size = 2
  io.println("${H(file, size)} ${H(size, file: "b")} ${H(file)}")
}`)
	expectError(t, prelude+`
struct H { file: string; size: i64 }
fun main() { io.println("${H("a", 1)}") }`, "construct 'H' by field name: 'file: \"a\"'")
	diags := checkSource(t, prelude+`
struct H { file: string }
fun main() { val file = "a"; io.println("${H(file: file)}") }`)
	found := false
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "'file: file' can be written 'file'") && d.Fix != nil {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the pun warning with a fix, got:\n%s", diags.Render())
	}
}

// D35 (v0.28): a function value is sendable when it is a named function or
// a lambda over vals of Sendable types; `sendable fun` parameters demand
// it, plain function parameters accept it.
func TestSendableFunctions(t *testing.T) {
	expectClean(t, prelude+`
fun run(f: sendable fun(i64): i64, x: i64): i64 = f(x)
fun plain(f: fun(i64): i64, x: i64): i64 = f(x)
fun twice(n: i64): i64 = n * 2
fun main() {
  val k = 3
  val g = (n: i64) => n + k
  scope {
    val a = async run(n => n * k, 1)
    val b = async run(twice, 2)
    val c = async run(g, 3)
    io.println("${await a} ${await b} ${await c} ${plain(g, 4)}")
  }
}`)
	expectError(t, prelude+`
fun run(f: sendable fun(i64): i64, x: i64): i64 = f(x)
fun main() { var m = 1; io.println("${run(n => n * m, 1)}") }`, "captures 'm', a 'var'")
	expectError(t, prelude+`
fun run(f: sendable fun(i64): i64, x: i64): i64 = f(x)
fun main() { val xs: MutableList<i64> = []; io.println("${run(n => { xs.push(n); n }, 1)}") }`, "captures 'xs', of type 'MutableList<i64>'")
	expectError(t, prelude+`
fun run(f: sendable fun(i64): i64, x: i64): i64 = f(x)
fun main() { val h: fun(i64): i64 = n => n; io.println("${run(h, 1)}") }`, "expected 'sendable fun(i64): i64', found 'fun(i64): i64'")
}

// D46 (v0.28): a generic `f: fun(T): R throws E` infers E from the lambda;
// a non-throwing lambda makes the instance an ordinary function.
func TestErrorPolymorphicHigherOrder(t *testing.T) {
	expectClean(t, prelude+`
error Bad { n: i64 }
fun <T, R, E> apply(x: T, f: fun(T): R throws E): R throws E = try f(x)
fun main() {
  val plain = apply(3, (n: i64) => n + 1)
  when (val r = apply(7, (n: i64) => if (n > 5) throw Bad(n) else n)) {
    is Ok  => io.println("ok $r $plain")
    is Err => io.println("err ${r.n}")
  }
  val xs = [1, 2]
  val doubled = xs.mapConcurrent(n => n * 2)
  when (val r = xs.mapConcurrent(n => if (n > 1) throw Bad(n) else n)) {
    is Ok  => io.println("$doubled $r")
    is Err => io.println("${r.n}")
  }
}`)
	// a throwing lambda where the instance is used without `try`: the
	// caller sees the Result like any other fallible call
	expectError(t, prelude+`
error Bad { n: i64 }
fun <T, R, E> apply(x: T, f: fun(T): R throws E): R throws E = try f(x)
fun main() { val v: i64 = apply(1, (n: i64) => if (n > 0) throw Bad(n) else n); io.println("$v") }`, "type mismatch")
}

// D55: type aliases are transparent names; diagnostics print the alias;
// recursion and bounds are errors; an alias of a struct constructs.
func TestTypeAliases(t *testing.T) {
	expectClean(t, prelude+`
type Index = i64
type Key = (Index, u64)
type StrMap<V> = Map<string, V>
type Handler = fun(string): string
struct P { x: i64; static fun origin(): P = P(x: 0) }
pub type Pt = P
fun apply(h: Handler, s: string): string = h(s)
fun main() {
  val k: Key = (1, 2)
  val n: Index = 3
  val m: i64 = n
  val names: StrMap<i64> = ["a": 1]
  val p = Pt(x: 1)
  io.println("${k.0 + m} ${names.get("a")} ${apply(s => s, "x")} ${Pt.origin()} ${p is Pt}")
}`)
	expectError(t, prelude+`
type Key = (i64, u64)
fun main() { val k: Key = 1; io.println("$k") }`, "expected 'Key', found 'i64'")
	expectError(t, prelude+`
type Loop = List<Loop>
fun main() { }`, "refers to itself")
	expectError(t, prelude+`
type Sorted<T: Comparable> = List<T>
fun main() { }`, "takes no bounds")
	expectError(t, prelude+`
type One = i64
fun main() { val x: One<i64> = 1; io.println("$x") }`, "'One' is not generic")
	expectError(t, prelude+`
type Pair<T> = (T, T)
fun main() { val x: Pair = (1, 2); io.println("$x") }`, "'Pair' expects 1 type arguments, got 0")
}

// `Never` is nameable: os.exit and a `fun f(): Never` end control flow, so a
// `when` whose arms all diverge makes what follows unreachable.
func TestNeverType(t *testing.T) {
	expectError(t, prelude+`use os
fun bail(): Never {
  io.println("bye")
  os.exit(2)
}
fun main() {
  val xs: List<i64> = []
  when (xs.first()) {
    null => bail()
    else => os.exit(0)
  }
  io.println("after")
}`, "unreachable code")
	expectClean(t, prelude+`use os
fun bail(): Never = os.exit(2)
fun main() {
  val xs: List<i64> = [1]
  val n: i64 = xs.first() ?: bail()
  io.println("$n")
}`)
	expectError(t, prelude+`
fun bad(): Never { io.println("x") }
fun main() { bad() }`, "expected 'Never', found '()'")
}

// Result accessors and the List<Result> helpers live in the prelude.
func TestResultHelpers(t *testing.T) {
	expectClean(t, prelude+`
error Odd { n: i64 }
fun check(n: i64): i64 throws Odd = if (n % 2 == 1) throw Odd(n) else n
fun main() {
  val rs = [1, 2, 3].map(n => check(n))
  val oks: List<i64> = rs.oks()
  val errs: List<Odd> = rs.errors()
  val (small, big) = [1, 5].partition(n => n < 3)
  io.println("$oks $errs ${rs.atOrPanic(0).getOrNull()} ${rs.atOrPanic(1).getOrDefault(0)} $small $big ${["1", "x"].mapNotNull(s => s.toInt())}")
}`)
}

// D48 (v0.28): tuples of ordered elements order lexicographically, through
// `<`, `sorted`, `sortedBy` with a tuple key, `min`, bounds and `compareTo`.
func TestTupleOrdering(t *testing.T) {
	expectClean(t, prelude+`
struct E { size: i64; name: string }
fun <T: Comparable> smallest(xs: List<T>): T? = xs.min()
fun main() {
  val xs = [(2, "b"), (1, "z"), (2, "a")]
  val es = [E(size: 5, name: "b"), E(size: 9, name: "a")]
  io.println("${xs.sorted()} ${(1, "a") < (1, "b")} ${(1, "a").compareTo((2, "a"))} ${es.sortedBy(e => (-e.size, e.name)).len()} ${smallest(xs)} ${[((1, 2), "x")].sorted().len()}")
}`)
	expectError(t, prelude+`
struct P { n: i64 }
fun main() { io.println("${(1, P(n: 1)) < (2, P(n: 2))}") }`, "operator '<' is not defined for '(i64, P)'")
}

// D37 (v0.28): tuple patterns nest in vals, loop heads and lambda parameters.
func TestNestedTuplePatterns(t *testing.T) {
	expectClean(t, prelude+`
fun main() {
  val groups = [((3, 7), ["a"]), ((1, 2), ["b", "c"])]
  val n = groups.map(((size, hash), files) => size * files.len() + hash)
  val byName = groups.sortedWith((((sa, _), _), ((sb, _), _)) => sa - sb)
  val ((s, h), fs) = groups.atOrPanic(0)
  var ((a, b), c) = ((1, 2), 3)
  a += 10
  loop (((size, _), files) in groups) io.println("$size ${files.len()}")
  io.println("$n ${byName.len()} $s $h $fs $a $b $c")
}`)
	expectError(t, prelude+`
fun main() { val xs = [(1, 2)]; io.println("${xs.map(((a, b), c) => a)}") }`, "cannot destructure a 'i64' into 2 names")
}

// D13 (v0.28): filterIs narrows a list of a sealed type by variant.
func TestFilterIs(t *testing.T) {
	expectClean(t, prelude+`
sealed trait Shape
struct Circle : Shape { r: f64 }
struct Square : Shape { side: f64 }
fun main() {
  val shapes: List<Shape> = [Circle(r: 1.0), Square(side: 2.0)]
  val circles: List<Circle> = shapes.filterIs<Circle>()
  val maybe: List<i64?> = [1, null, 3]
  val present: List<i64> = maybe.filterNotNull()
  io.println("${circles.map(c => c.r)} $present")
}`)
	expectError(t, prelude+`
sealed trait Shape
struct Circle : Shape { r: f64 }
fun main() { val xs: List<i64> = [1]; io.println("${xs.filterIs<Circle>()}") }`, "'filterIs' works on a list of a sealed type")
	expectError(t, prelude+`
sealed trait Shape
struct Circle : Shape { r: f64 }
struct Other { n: i64 }
fun main() { val xs: List<Shape> = [Circle(r: 1.0)]; io.println("${xs.filterIs<Other>()}") }`, "'Other' is not a variant of 'Shape'")
}

func TestTaskCancel(t *testing.T) {
	expectClean(t, prelude+`
fun slow(): i64 {
  await sleep(100)
  1
}
fun main() {
  scope {
    val t = async slow()
    t.cancel()
  }
  io.println("done")
}`)
	expectError(t, prelude+`
fun slow(): i64 = 1
fun main() {
  scope {
    val t = async slow()
    t.cancel(1)
  }
}`, "'cancel' takes no arguments")
}

func TestIoWaitIsStdOnly(t *testing.T) {
	// the socket wait exists for std/net; user code has no such name
	expectError(t, prelude+`
fun main() {
  await ioWait(3, false)
}`, "unknown function 'ioWait'")
}

func TestRaceNarrowingJoins(t *testing.T) {
	// an assignment in one race arm does not smart-cast the variable after
	// the race: exactly one arm runs, and the other may not have assigned
	expectError(t, prelude+`
fun slow(): i64 {
  await sleep(50)
  1
}
fun main() {
  var out: i64? = null
  scope {
    val t = async slow()
    race {
      val r = await t => out = r
      sleep(1) => io.println("late")
    }
  }
  val n: i64 = out
  io.println("$n")
}`, "type mismatch")
	// every arm returning makes the race — and the scope — diverge
	expectClean(t, prelude+`
fun slow(): i64 {
  await sleep(50)
  1
}
fun pick(): i64 {
  scope {
    val t = async slow()
    race {
      val r = await t => return r
      sleep(1) => return -1
    }
  }
}
fun main() { io.println("${pick()}") }`)
}

func TestScopeBodyDiverges(t *testing.T) {
	expectClean(t, prelude+`
fun work(): i64 {
  await sleep(1)
  1
}
fun early(): i64 {
  scope {
    async work()
    return 1
  }
}
error Boom { }
fun thrower(): i64 throws Boom {
  scope {
    async work()
    throw Boom()
  }
}
fun main() { io.println("${early()} ${thrower()}") }`)
}

func TestErrorUnionWithTypeParam(t *testing.T) {
	// `throws E | Timeout`: with E bound to Never the union is just Timeout;
	// with E bound to an error it is the two-member union
	expectClean(t, prelude+`
error Timeout { }
error Late { }
fun <R, E> guarded(flag: bool, f: fun(): R throws E): R throws E | Timeout {
  if (flag) throw Timeout()
  try f()
}
fun failing(): i64 throws Late = throw Late()
fun onlyTimeout(): i64 throws Timeout = try guarded(false, () => 5)
fun both(): i64 throws Late | Timeout = try guarded(false, () => try failing())
fun main() { io.println("${onlyTimeout()} ${both()}") }`)
	expectError(t, prelude+`
error Timeout { }
error Late { }
fun <R, E> guarded(flag: bool, f: fun(): R throws E): R throws E | Timeout {
  if (flag) throw Timeout()
  try f()
}
fun failing(): i64 throws Late = throw Late()
fun narrow(): i64 throws Timeout = try guarded(false, () => try failing())
fun main() { io.println("${narrow()}") }`, "not in the declared 'throws Timeout'")
}

func TestWithTimeoutTypes(t *testing.T) {
	expectClean(t, prelude+`
error Late { }
fun slow(): i64 {
  await sleep(1)
  1
}
fun failing(): string? throws Late {
  await sleep(1)
  throw Late()
}
fun a(): i64 throws Timeout = try withTimeout(100, () => slow())
fun b(): string? throws Late | Timeout = try withTimeout(100, () => try failing())
fun main() { io.println("${a()} ${b()}") }`)
	expectError(t, prelude+`
fun main() {
  var n = 0
  io.println("${withTimeout(10, () => { n += 1; n })}")
}`, "cannot cross a task boundary")
}
