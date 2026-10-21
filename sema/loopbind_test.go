package sema

import "testing"

// D190: a loop head may bind like an `if` head: it runs before every
// iteration and ends the loop on null.
func TestLoopHeadBinding(t *testing.T) {
	expectClean(t, prelude+`
fun first(xs: List<i64>, i: i64): i64? => xs.at(i)
fun main() {
  val xs = [1, 2, 3]
  var i = 0
  var total = 0
  loop (val x = first(xs, i)) {
    total += x
    i += 1
  }
  loop (val y = first(xs, i - 1) && y > 100) {
    total += y
  }
  io.println("$total")
}`)
	for _, c := range []struct{ name, body, want string }{
		{"a value that cannot be null", `loop (val x = 3) { io.println("$x") }`, "needs a value that can be null"},
		{"joined by ||", `var n: i64? = 1
  loop (val x = n || true) { io.println("$x") }`, "joined to the rest by '&&'"},
		{"outside an if or a loop", `val y = (val z = 1)`, "only allowed in the condition of an 'if' or a 'loop'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+"fun main() {\n  "+c.body+"\n}", c.want)
		})
	}
}

// D189: `in` and `!in`.
func TestMembershipOperator(t *testing.T) {
	expectClean(t, prelude+`
fun kind(c: u8): string {
  if (c in '0'..'9') return "digit"
  if (c in ['a', 'e', 'i', 'o', 'u']) return "vowel"
  if (c in " \t\r\n") return "space"
  return "other"
}
fun main() {
  val stops: Set<string> = ["a"]
  val heads: Map<string, i64> = ["host": 1]
  val xs = [1, 2, 3]
  val url = "https://x.org"
  val b: u8 = 'x'
  io.println("${kind(b)} ${"a" in stops} ${"b" !in stops} ${"host" in heads} ${2 in xs} ${7 !in xs} ${3 in 1..<3} ${"://" in url} ${b in url} ${b in "xyz"}")
}`)
	for _, c := range []struct{ name, body, want string }{
		{"a nullable left operand", `val n: i64? = 1
  val xs = [1, 2]
  val r = n in xs`, "type mismatch"},
		{"a container without contains", `val r = 1 in 5`, "no method 'contains'"},
		{"a wrong element type", `val xs = ["a"]
  val r = 1 in xs`, "type mismatch"},
		{"a literal set of the wrong type", `val s = "a"
  val r = s in [1, 2]`, "type mismatch"},
	} {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+"fun main() {\n  "+c.body+"\n}", c.want)
		})
	}
}

// D194: an enum gains extension functions (and only those).
func TestEnumExtensionFunctions(t *testing.T) {
	expectClean(t, prelude+`
enum Level { Debug, Info, Warn, Error }
extend Level {
  fun label(): string => when (this) {
    Level.Debug => "dbg"
    Level.Info => "inf"
    Level.Warn => "wrn"
    Level.Error => "err"
  }
  fun isSevere(): bool => this >= Level.Warn
}
extend Level with fun next(): Level? => Level.fromValue(this.value + 1)
fun main() {
  io.println("${Level.Warn.label()} ${Level.Warn.isSevere()} ${Level.Debug.next()?.label()}")
}`)
	expectError(t, prelude+`
enum Level { Debug, Warn }
extend Level { fun label(): string => "x" }
implement Display for Level { fun toString(): string => "L" }
fun main() { }`, "cannot implement 'Display' for enum 'Level'")
}

// D193: structs declared in the body of a sealed trait are its variants.
func TestNestedVariants(t *testing.T) {
	expectClean(t, prelude+`
sealed trait Shape {
  struct Circle { r: i64 }
  struct Rect {
    w: i64
    h: i64
  }
  fun sides(): i64 => 0
}
sealed trait Option<T> {
  struct Some<T> : Option<T> { value: T }
  struct None<T> : Option<T> { }
}
fun area(s: Shape): i64 => when (s) {
  is Circle => s.r * s.r * 3
  is Rect => s.w * s.h
}
fun unwrap(o: Option<i64>): i64 => when (o) {
  is Some => o.value
  is None => 0
}
fun main() { io.println("${area(Circle(r: 2))} ${area(Rect(w: 2, h: 3))} ${unwrap(Some(value: 7))}") }`)
	for _, c := range []struct{ name, src, want string }{
		{"a trait that is not sealed", `trait T {
  struct V { x: i64 }
}
fun main() { }`, "only a sealed trait declares its variants"},
		{"a generic trait without the clause", `sealed trait Option<T> {
  struct Some<T> { value: T }
}
fun main() { }`, "names it with its arguments"},
		{"a missing arm", `sealed trait Shape {
  struct Circle { r: i64 }
  struct Rect { w: i64 }
}
fun area(s: Shape): i64 => when (s) {
  is Circle => 1
}
fun main() { }`, "Rect"},
	} {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+c.src, c.want)
		})
	}
}

// D193: an `implement Trait { }` in the body of a variant declared inside its
// trait is an implement for that variant, as in a top-level struct.
func TestNestedVariantImplement(t *testing.T) {
	expectClean(t, prelude+`
sealed trait Shape {
  struct Circle {
    r: i64

    implement Shape {
      fun area(): i64 => this.r * this.r * 3
    }
  }
  fun area(): i64
}
fun main() { io.println("${Circle(r: 2).area()}") }`)
}

// D195: a method in a variant's body that has the name of a method of its
// sealed trait implements it; `implement Shape { }` is not needed around it.
func TestVariantMethodsImplementTheTrait(t *testing.T) {
	expectClean(t, prelude+`
sealed trait Shape {
  struct Circle {
    r: i64
    fun area(): i64 => this.r * this.r * 3
    override fun describe(): string => "circle"
    fun diameter(): i64 => this.r * 2
  }
  struct Rect {
    w: i64
    h: i64
    fun area(): i64 => this.w * this.h
  }
  fun area(): i64
  fun describe(): string => "shape"
}
sealed trait Animal {
  fun sound(): string
}
struct Dog : Animal {
  fun sound(): string => "woof"
}
struct Cat : Animal {
  implement Animal {
    fun sound(): string => "meow"
  }
}
fun main() {
  val shapes: List<Shape> = [Circle(r: 2), Rect(w: 2, h: 3)]
  loop (s in shapes) io.println("${s.describe()} ${s.area()}")
}`)
	for _, c := range []struct{ name, src, want string }{
		{"a default body needs override", `sealed trait Shape {
  struct Circle {
    fun describe(): string => "c"
  }
  fun describe(): string => "shape"
}
fun main() { }`, "must be marked 'override'"},
		{"a wrong signature", `sealed trait Shape {
  struct Circle {
    fun area(): f64 => 1.0
  }
  fun area(): i64
}
fun main() { }`, "area"},
		{"a missing method", `sealed trait Shape {
  struct Circle {
    r: i64
  }
  fun area(): i64
}
fun main() { }`, "write 'fun area(): i64' in its body"},
	} {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, prelude+c.src, c.want)
		})
	}
}

func TestRedundantImplementLint(t *testing.T) {
	const marker = "around the methods of its variant"
	if got := lintMessages(t, prelude+`
sealed trait Shape {
  struct Circle {
    r: i64
    implement Shape {
      fun area(): i64 => this.r * 3
    }
  }
  fun area(): i64
}
fun main() { io.println("${Circle(r: 1).area()}") }`, marker); len(got) != 1 {
		t.Errorf("want one warning for an implement of the variant's own sealed trait, got %v", got)
	}
	if got := lintMessages(t, prelude+`
sealed trait Shape {
  struct Circle {
    r: i64
    fun area(): i64 => this.r * 3
  }
  fun area(): i64
}
fun main() { io.println("${Circle(r: 1).area()}") }`, marker); len(got) != 0 {
		t.Errorf("methods in the body are the form asked for, got %v", got)
	}
	if got := lintMessages(t, prelude+`
trait Named { fun name(): string }
sealed trait Shape {
  struct Circle {
    r: i64
    fun area(): i64 => this.r * 3
    implement Named {
      fun name(): string => "circle"
    }
  }
  fun area(): i64
}
fun main() { io.println("${Circle(r: 1).area()}") }`, marker); len(got) != 0 {
		t.Errorf("an implement of another trait stays, got %v", got)
	}
}
