package driver

import (
	"strings"
	"testing"
)

// D113 part 3: what the compiler computes by running a `const fun` is what the
// program computes by calling it. Each function below is run twice — once while
// the compiler evaluates a constant, once at run time — and the two results are
// compared, in a debug and a release build: integer wrapping and checks, floats
// (f32 rounding), strings (byte indexes, code points, number parsing), lists
// (clamped slices, negative indexes), maps and sets (insertion order), structs
// and arrays as values, methods that change their receiver, labelled loops, early
// returns, generics.
func TestConstFunMatchesRunTime(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	for _, release := range []bool{false, true} {
		out, code := runTestFiles(t, map[string]string{"fns.vs": constFunSrc, "cf.test.vs": constFunTests}, Options{Release: release})
		if code != 0 || !strings.Contains(out, "20 passed") {
			t.Fatalf("release=%v: exit %d, output:\n%s", release, code, out)
		}
	}
}

const constFunSrc = `struct P {
  var x: i64
  var y: i64
}

enum Color { Red, Green, Blue }

const fun ints(): List<i64> {
  val out: MutableList<i64> = []
  val big: i64 = 9223372036854775807
  out.push(big +% 1)
  out.push(-7 / 2)
  out.push(-7 % 3)
  out.push(7 % -3)
  out.push(-8 >> 1)
  out.push(1 << 62)
  out.push(0xFF.wrapU8().toI64())
  out.push(300.wrapU8().toI64())
  out.push((-1).wrapU32().toI64())
  out.push(~5)
  out.push(6 & 3 | 8 ^ 1)
  out.toList()
}
const INTS: List<i64> = ints()

const fun fnv(n: i64): u32 {
  var h: u32 = 2166136261
  loop (i in 0..<n) {
    h = (h ^ i.wrapU32()) *% 16777619
  }
  h
}
const FNV: u32 = fnv(1000)

const fun floats(): List<f64> {
  val out: MutableList<f64> = []
  var s: f32 = 0.0
  var d: f64 = 0.0
  loop (_ in 0..<10) {
    s += 0.1
    d += 0.1
  }
  out.push(s.toF64())
  out.push(d)
  out.push(1.0 / 3.0)
  out.push(2.5 - 1.0)
  out.toList()
}
const FLOATS: List<f64> = floats()

const fun texts(): List<string> {
  val s = "héllo, wörld"
  val out: MutableList<string> = []
  out.push("${s.len()} ${s.charCount()}")
  out.push(s.substring(0, 2) ?: "null")
  out.push(s.substring(0, 3) ?: "null")
  out.push(s.substring(5, 2) ?: "null")
  out.push(s.substring(-1, 2) ?: "null")
  out.push(s.substring(0, 99) ?: "null")
  out.push("${"12".toInt() ?: -1} ${"-9223372036854775808".toInt() ?: -1} ${"+5".toInt() ?: -1} ${"1_0".toInt() ?: -1} ${"".toInt() ?: -1} ${"-".toInt() ?: -1} ${"9223372036854775808".toInt() ?: -1}")
  var joined = ""
  loop (c in s.chars()) {
    joined = c + "|" + joined
  }
  out.push(joined)
  loop (part in "a,b,,c".split(",")) {
    out.push("[" + part + "]")
  }
  out.push("${s.startsWith("hé")} ${s.endsWith("ld")} ${s.contains("lo,")} ${s.contains("xyz")}")
  out.push("${s.byteAt(1)} ${s.byteAt(2)}")
  val t = "  Key=Va=lue \r\nsecond\n\nlast  "
  out.push("[" + t.trim() + "]")
  out.push("${t.indexOf("=")} ${t.indexOf("=", from: 5)} ${t.indexOf("zz")} ${t.indexOf("", from: 3)} ${t.indexOf("", from: 99)} ${t.lastIndexOf("a")}")
  out.push("${(t.splitOnce("=") ?: ("", "")).1} ${t.splitOnce("#") == null}")
  loop (line in t.lines()) {
    out.push("<" + line + ">")
  }
  out.push("a-b-c".replace("-", "+") + "|" + "abc".replace("", "x") + "|" + "xx".replace("x", ""))
  out.push("ab".repeat(3) + "|" + "Hello".toUpper() + "|" + "Hello".toLower() + "|" + "hello".capitalize() + "|" + "7".padStart(3, "0") + "|" + "7".padEnd(3, "."))
  out.toList()
}
const TEXTS: List<string> = texts()

const fun lists(): List<i64> {
  val m: MutableList<i64> = [1, 2, 3]
  m.addAll(m.toList())
  val popped = m.pop() ?: -1
  val out: MutableList<i64> = []
  out.push(popped)
  out.push(m.len().toI64())
  out.addAll([1, 2, 3, 4, 5].slice(-5, 10))
  out.addAll([1, 2, 3, 4, 5].slice(3, 1))
  out.addAll([1, 2, 3, 4, 5].slice(1, 3))
  out.push(m.at(-1) ?: -9)
  out.push(m.at(99) ?: -9)
  out.toList()
}
const LISTS: List<i64> = lists()

const fun maps(): List<string> {
  val m: MutableMap<string, i64> = [:]
  m.set("a", 1)
  m.set("b", 2)
  m.set("a", 3)
  m.set("c", 4)
  m.remove("b")
  m.set("b", 5)
  val out: MutableList<string> = []
  loop (k in m.keys()) {
    out.push("$k=${m.get(k) ?: -1}")
  }
  out.push("${m.len()} ${m.containsKey("c")} ${m.containsKey("z")}")
  val s: MutableSet<i64> = []
  out.push("${s.add(3)} ${s.add(3)} ${s.add(4)} ${s.len()}")
  out.toList()
}
const MAPS: List<string> = maps()

const fun structs(): List<i64> {
  val a = P(x: 1, y: 2)
  var b = a
  b.x = 10
  val xs: MutableList<P> = [a, b]
  var c = xs.at(0) ?: a
  c.x = 99
  loop (&p in xs) {
    p.y += 100
  }
  val t = (a, b)
  var u = t
  u.0.x = 7
  [a.x.toI64(), b.x.toI64(), c.x.toI64(), (xs.at(0) ?: a).y.toI64(), (xs.at(1) ?: a).y.toI64(), t.0.x.toI64(), u.0.x.toI64()].toList()
}
const STRUCTS: List<i64> = structs()

const fun flow(): List<i64> {
  val out: MutableList<i64> = []
  loop :outer (i in 0..<6) {
    loop (j in 0..<6) {
      if (j == 3) continue outer
      if (i == 4) break outer
      out.push(i * 10 + j)
    }
  }
  out.toList()
}
const FLOW: List<i64> = flow()

const fun firstBig(xs: List<i64>): i64 {
  loop (x in xs) {
    loop (y in 0..<x) {
      if (y * y > 50) return y
    }
  }
  -1
}
const BIG1: i64 = firstBig([1, 3, 20, 30])
const BIG2: i64 = firstBig([1, 2])

const fun describe(n: i64): string = when (n) {
  0 => "zero"
  1, 2 => "small"
  else => if (n < 0) "negative" else "large"
}
const fun describeAll(): List<string> = [describe(0), describe(2), describe(-4), describe(99)]
const DESC: List<string> = describeAll()

const fun colorName(c: Color): string = when (c) {
  Color.Red => "r"
  Color.Green => "g"
  Color.Blue => "b"
}
const COLORS: string = colorName(Color.Green) + colorName(Color.Blue)

const fun ack(m: i64, n: i64): i64 = if (m == 0) n + 1 else if (n == 0) ack(m - 1, 1) else ack(m - 1, ack(m, n - 1))
const ACK: i64 = ack(2, 3)

const fun gcd(a: i64, b: i64): i64 = if (b == 0) a else gcd(b, a % b)
const GCD: i64 = gcd(1071, 462)

const fun builder(n: i64): string {
  val sb = StringBuilder()
  loop (i in 0..<n) {
    sb.append("$i,")
  }
  sb.appendLine("end")
  sb.appendByte(33)
  sb.toString()
}
const BUILT: string = builder(5)

const fun nulls(n: i64?): i64 {
  val m = n ?: return -1
  m + 1
}
const N1: i64 = nulls(null)
const N2: i64 = nulls(4)

const fun arrays(): List<i64> {
  val a: Array<i64, 4> = [1, 2, 3, 4]
  var b = a
  b.set(0, 100)
  [a.at(0), b.at(0), a.len().toI64()]
}
const ARRAYS: List<i64> = arrays()


const fun firstOr<T>(xs: List<T>, d: T): T = xs.at(0) ?: d
const G1: i64 = firstOr([7, 8], 0)
const G2: string = firstOr([], "none")

struct Inner {
  var x: i64
}

struct Outer {
  var inner: Inner
  var tag: string

  const fun bump(by: i64) {
    this.inner.x += by
  }

  const fun summary(): string = "${this.tag}:${this.inner.x}"

  static const fun make(tag: string, x: i64): Outer = Outer(inner: Inner(x), tag)
}

const fun nested(): List<string> {
  val o = Outer.make("a", 1)
  var copy = o
  copy.bump(10)
  o.bump(1)
  copy.inner.x += 5
  val out: MutableList<string> = [o.summary(), copy.summary()]
  out.toList()
}
const NESTED: List<string> = nested()

const fun sorting(): List<string> {
  val nums = [5, 3, 9, 1, 3].sorted()
  val words = ["pear", "apple", "fig"].sorted()
  val out: MutableList<string> = []
  loop (n in nums) {
    out.push("$n")
  }
  loop (w in words) {
    out.push(w)
  }
  out.toList()
}
const SORTED: List<string> = sorting()

const fun withDefault(a: i64, b: i64 = 10): i64 = a * b
const DEF: List<i64> = [withDefault(2), withDefault(2, 3)]

const fun mapLoop(): List<string> {
  val m: MutableMap<string, i64> = ["x": 1, "y": 2]
  val out: MutableList<string> = []
  loop ((k, v) in m) {
    out.push("$k$v")
  }
  out.toList()
}

const fun intOps(): List<string> {
  val out: MutableList<string> = []
  val a: i8 = -128
  val b: i8 = 100
  out.push("${b.abs()} ${(-5).abs()} ${(-5).sign()} ${0.sign()} ${5.sign()} ${b.min(a)} ${b.max(a)}")
  out.push("${(-7).mod(3)} ${7.mod(-3)} ${(-7).mod(-3)} ${15.clamp(0, 10)} ${(-3).clamp(0, 10)}")
  out.push("${2.pow(10)} ${(-2).pow(3)} ${7.pow(0)} ${0.pow(0)} ${(-1).pow(63)}")
  out.push("${b.wrappingAdd(100)} ${b.saturatingAdd(100)} ${a.saturatingSub(1)} ${a.wrappingSub(1)} ${b.wrappingMul(3)} ${b.saturatingMul(3)} ${a.saturatingMul(-1)}")
  out.push("${b.checkedAdd(100) ?: -1} ${b.checkedAdd(10) ?: -1} ${b.checkedMul(2) ?: -1} ${a.checkedSub(1) ?: -1}")
  val u: u32 = 0xF0F00001
  out.push("${u.countOnes()} ${u.leadingZeros()} ${u.trailingZeros()} ${u.swapBytes()} ${u.reverseBits()}")
  out.push("${u.rotateLeft(8)} ${u.rotateRight(4)} ${u.rotateLeft(-4)} ${u.rotateLeft(36)} ${u.rotateLeft(0)}")
  val z: u64 = 0
  val w: u64 = 0x8000000000000001
  out.push("${z.leadingZeros()} ${z.trailingZeros()} ${w.countOnes()} ${w.rotateLeft(1)} ${w.swapBytes()} ${w.reverseBits()} ${w.leadingZeros()}")
  val n: i16 = -2
  val m: i8 = -128
  out.push("${n.countOnes()} ${n.leadingZeros()} ${n.trailingZeros()} ${n.swapBytes()} ${n.reverseBits()} ${n.rotateLeft(3)} ${m.rotateRight(1)} ${m.swapBytes()} ${m.reverseBits()}")
  val big: u64 = 18446744073709551615
  out.push("${big.checkedAdd(1) ?: 0} ${big.wrappingAdd(2)} ${big.saturatingMul(2)} ${big.min(5)} ${big.mod(1000)}")
  out.toList()
}
const INTOPS: List<string> = intOps()

const fun floatOps(): List<string> {
  val out: MutableList<string> = []
  val x: f64 = -2.5
  out.push("${x.abs()} ${x.floor()} ${x.ceil()} ${x.round()} ${x.trunc()} ${x.sign()} ${(2.0).sqrt()} ${x.min(1.0)} ${x.max(1.0)}")
  out.push("${x.mod(2.0)} ${(5.5).mod(-2.0)} ${x.clamp(-1.0, 1.0)} ${x.copySign(1.0)} ${x.isSignNegative()} ${x.isNaN()} ${x.isFinite()}")
  val y: f32 = 2.5
  out.push("${y.round()} ${(0.5).round()} ${(-0.5).round()} ${(1.0 / 3.0).sqrt()} ${y.sqrt()} ${(-y).abs()} ${y.mod(0.7)}")
  val nan = 0.0 / 0.0
  val inf = 1.0 / 0.0
  out.push("${nan.isNaN()} ${nan.min(1.0)} ${nan.max(1.0)} ${nan.sign()} ${inf.isInfinite()} ${inf.isFinite()} ${(-inf).clamp(-5.0, 5.0)} ${(-0.0).isSignNegative()} ${(-0.0).sign()}")
  out.toList()
}
const FLOATOPS: List<string> = floatOps()

const fun maxOf<T: Comparable>(a: T, b: T): T = if (a.compareTo(b) == Ordering.Less) b else a
const fun radix(): List<string> = [255.toString(radix: 16), (-255).toString(radix: 2), 0.toString(), 12345.toString()]
const MAXES: List<string> = ["${maxOf(3, 9)}", maxOf("pear", "apple"), "${maxOf(2.5, -1.0)}"]
const RADIX: List<string> = radix()
`

const constFunTests = `// const fun results are what the same function computes at run time (D113).
test "integer arithmetic" {
  expect(INTS == ints())
}
test "wrapping hash" {
  expect(FNV == fnv(1000))
}
test "floats" {
  expect(FLOATS == floats())
}
test "text" {
  expect(TEXTS == texts())
}
test "lists" {
  expect(LISTS == lists())
}
test "maps and sets" {
  expect(MAPS == maps())
}
test "structs are values" {
  expect(STRUCTS == structs())
}
test "labelled loops" {
  expect(FLOW == flow())
}
test "return from nested loops" {
  expect(BIG1 == firstBig([1, 3, 20, 30]) && BIG2 == firstBig([1, 2]))
}
test "when" {
  expect(DESC == describeAll())
  expect(COLORS == colorName(Color.Green) + colorName(Color.Blue))
}
test "recursion" {
  expect(ACK == ack(2, 3) && GCD == gcd(1071, 462))
}
test "string builder" {
  expect(BUILT == builder(5))
}
test "nullables" {
  expect(N1 == nulls(null) && N2 == nulls(4))
}
test "arrays are values" {
  expect(ARRAYS == arrays())
}
test "generics, methods and statics" {
  expect(G1 == firstOr([7, 8], 0) && G2 == firstOr([], "none"))
  expect(NESTED == nested())
  expect(NESTED == ["a:2", "a:16"])
}
test "sorting and defaults" {
  expect(SORTED == sorting())
  expect(DEF == [withDefault(2), withDefault(2, 3)])
}
test "iterating a map" {
  expect(mapLoop() == ["x1", "y2"])
}
test "integer built-ins" {
  expect(INTOPS == intOps())
}
test "float built-ins" {
  expect(FLOATOPS == floatOps())
}
test "generic comparison and number text" {
  expect(MAXES == ["${maxOf(3, 9)}", maxOf("pear", "apple"), "${maxOf(2.5, -1.0)}"])
  expect(RADIX == radix())
  expect(RADIX == ["ff", "-11111111", "0", "12345"])
}
`
