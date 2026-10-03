package driver

import (
	"strings"
	"testing"
)

// D121: `Array<T, N>` — inline, a value, its length part of its type — and
// the constants that go where a type argument does. One program, in a debug
// and a release build: what an array is and does, const generic parameters,
// and the memory class, in which a value of 128 bytes or more that holds an
// array is never loaded into registers but copied, passed and returned
// through memory.
func TestArrays(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	for _, release := range []bool{false, true} {
		out, code := runTests(t, arraysSrc, Options{Release: release})
		if code != 0 || !strings.Contains(out, "12 passed, 0 failed") {
			t.Fatalf("release=%v: exit %d, output:\n%s", release, code, out)
		}
	}
}

const arraysSrc = `use json

const K: Array<u32, 4> = [10, 20, 30, 40]
const NAMES: Array<string, 2> = ["a", "b"]
const GRID: Array<Array<u8, 2>, 2> = [[1, 2], [3, 4]]
const WIDTH = 4

struct Sha {
  var state: Array<u32, 8> = [1, 2, 3, 4, 5, 6, 7, 8]
}

struct Buf<const N: i64> {
  var data: Array<u8, N>

  fun capacity(): i64 = N
}

struct Packet {
  id: i64
  words: Array<u32, 3>
  grid: Array<Array<u8, 2>, 2>

  implement Codable
}

struct Defaults {
  words: Array<u32, 4>
  name: string

  implement Default
}

fun sum<const N: i64>(a: Array<i64, N>): i64 = a.fold(0, (s, x) => s + x)

fun zeros<const N: i64>(): Array<i64, N> = Array.make(0)

fun lengthOf<T, const N: i64>(a: Array<T, N>): i64 = N

fun total(a: Array<i64, 4>): i64 {
  var s = 0
  loop (x in a) {
    s += x
  }
  s
}

test "an array is a value" {
  var a: Array<i64, 4> = [10, 20, 30, 40]
  val b = a
  a.set(0, 7)
  a.set(-1, 41)
  expect(a == [7, 20, 30, 41])
  expect(b == [10, 20, 30, 40])
  expect(total(a) == 98 && total(b) == 100)
  var inner = Sha()
  val before = inner
  inner.state.set(2, 33)
  expect(inner.state.at(2) == 33 && before.state.at(2) == 3)
  expect("$a" == "[7, 20, 30, 41]" && a.len() == 4 && !a.isEmpty())
}

test "reading: at, first, last, indices" {
  val a: Array<i64, 4> = [10, 20, 30, 40]
  val far = 9
  val farBack = -9
  expect(a.at(1) == 20 && a.at(-1) == 40 && a.at(far) == null && a.at(farBack) == null)
  expect(a.first() == 10 && a.last() == 40)
  var s = 0
  loop (i in a.indices()) {
    s += a.at(i)
  }
  loop (i in 0..<4) {
    s += a.at(i)
  }
  var j = 0
  loop (j < 4) {
    s += a.at(j)
    j += 1
  }
  expect(s == 300)
  expect(K.at(3) == 40 && K.at(-1) == 40)
  var t: u32 = 0
  loop (r in K.indices()) {
    t += K.at(r)
  }
  expect(t == 100)
  expect(K.len() == 4 && NAMES.at(1) == "b" && GRID.at(1) == [3, 4])
}

test "loops: by value over a copy, by reference in place" {
  var a: Array<i64, 3> = [1, 2, 3]
  var seen = 0
  loop (x in a) {
    a.set(0, 100)
    seen += x
  }
  expect(seen == 6 && a.at(0) == 100)
  loop (&x in a) {
    *x = *x + 1
  }
  expect(a == [101, 3, 4])
}

test "constants as type arguments, and const generic parameters" {
  expect(sum([5, 6]) == 11 && sum<3>([1, 2, 3]) == 6 && sum(zeros<5>()) == 0)
  expect(lengthOf(zeros<7>()) == 7 && lengthOf(["a", "b"]) == 2)
  val q: Array<i64, WIDTH> = [1, 2, 3, 4]
  val r: Array<u8, 4 * 16> = Array.make(0)
  expect(q.len() == 4 && r.len() == 64)
  val b = Buf<8>(data: Array.make(1))
  expect(b.capacity() == 8 && b.data.len() == 8 && b.data.at(7) == 1)
  val c: MutableMap<string, Array<u8, 200>> = [:]
  c.set("x", Array.make(3))
  expect(c.get("x")?.at(199) == 3)
}

test "list methods work on an array, over a copy" {
  val a: Array<i64, 4> = [3, 1, 2, 5]
  expect(a.map(x => x * 2) == [6, 2, 4, 10])
  expect(a.filter(x => x > 1) == [3, 2, 5])
  expect(a.contains(2) && a.indexOf(5) == 3 && !a.contains(9))
  expect(a.sorted() == [1, 2, 3, 5] && a.toList() == [3, 1, 2, 5])
  expect(a.any(x => x == 5) && a.all(x => x > 0) && a.count(x => x > 1) == 3)
  expect(a.toMutable().len() == 4 && a.sum() == 11)
  expect(["a", "b"].toArray<2>() == ["a", "b"] && ["a"].toArray<2>() == null)
  val words: Array<string, 3> = ["x", "y", "z"]
  expect(words.join("-") == "x-y-z")
}

test "equality, hashing, text and defaults" {
  val a: Array<i64, 2> = [1, 2]
  val m: MutableMap<Array<i64, 2>, string> = [:]
  m.set(a, "one-two")
  expect(m.get([1, 2]) == "one-two" && m.get([2, 1]) == null)
  expect(a != [1, 3] && a == [1, 2])
  val d = Defaults.default()
  expect("${d}" == "Defaults(words: [0, 0, 0, 0], name: )")
  expect(Array<i64, 3>.default() == [0, 0, 0])
}

test "an array is codable as a list of exactly N" {
  val p = Packet(id: 1, words: [7, 8, 9], grid: [[1, 2], [3, 4]])
  val text = json.encode(p).getOrNull() ?: "?"
  expect(text == "{\"id\":1,\"words\":[7,8,9],\"grid\":[[1,2],[3,4]]}")
  val back = json.decode<Packet>(text)
  expect(back is Ok)
  val short = json.decode<Packet>("{\"id\":2,\"words\":[1,2],\"grid\":[[1,2],[3,4]]}")
  expect(short is Err)
}

// ---------------------------------------------------------------------------
// the memory class: 128 bytes and more

struct Block {
  var data: Array<u8, 200>
  tag: i64
}

error TooBig {
  message: string
}

fun make(v: u8): Array<u8, 200> = Array.make(v)

fun checked(v: u8): Array<u8, 200> throws TooBig {
  if (v > 100) throw TooBig(message: "too big")
  Array.make(v)
}

fun weight(a: Array<u8, 200>): i64 {
  var s = 0
  loop (x in a) {
    s += x.toI64()
  }
  s
}

fun bump(a: Array<u8, 200>): Array<u8, 200> {
  var b = a
  b.set(0, 200)
  b
}

trait Shape {
  fun weigh(extra: Array<u8, 200>): i64
}

struct Heavy {
  w: Array<u8, 200>

  implement Shape {
    fun weigh(extra: Array<u8, 200>): i64 = weight(this.w) + weight(extra)
  }
}

fun twice(a: Array<u8, 200>, f: fun(Array<u8, 200>): Array<u8, 200>): Array<u8, 200> = f(f(a))

fun feed(ch: Channel<Array<u8, 200>>, a: Array<u8, 200>) suspends {
  ch.send(a)
  ch.send(make(8))
}

fun slowMake(v: u8): Array<u8, 200> suspends {
  await sleep(Duration.millis(1))
  Array.make(v)
}

test "a large array is copied, passed and returned through memory" {
  val a = make(2)
  var b = a
  b.set(5, 9)
  expect(weight(a) == 400 && weight(b) == 407 && a != b)
  val c = bump(b)
  expect(c.at(0) == 200 && b.at(0) == 2 && weight(c) == 605)
  val big = Block(data: a, tag: 1)
  var big2 = big
  big2.data.set(1, 77)
  expect(big.data.at(1) == 2 && big2.data.at(1) == 77 && big != big2 && big == Block(data: a, tag: 1))
  expect("${big.tag} ${big.data.at(0)}" == "1 2")
}

test "a large array in a nullable, a tuple, a branch and a list" {
  val a = make(2)
  val o: Array<u8, 200>? = a
  val none: Array<u8, 200>? = null
  expect(o?.at(5) == 2 && none?.at(5) == null && o != none)
  val pair = (a, 7)
  val pair2 = pair
  expect(weight(pair.0) == 400 && pair2.1 == 7 && pair == pair2)
  val pick = if (weight(a) > 100) make(9) else make(1)
  val w = when {
    weight(a) > 1000 => make(5)
    else => a
  }
  expect(pick.at(0) == 9 && w.at(1) == 2)
  val l = [a, pick]
  expect(l.len() == 2 && l.at(1).at(0) == 9 && l.at(0) == a)
  val m: MutableMap<string, Array<u8, 200>> = [:]
  m.set("a", a)
  m.set("b", pick)
  expect(m.get("b")?.at(0) == 9 && m.get("z") == null)
}

test "a large array through errors, closures and trait objects" {
  expect(weight(checked(3).getOrNull() ?: make(0)) == 600)
  expect(checked(200) is Err)
  val inc = twice(make(2), x => {
    var y = x
    y.set(0, y.at(0) + 1)
    y
  })
  expect(inc.at(0) == 4)
  val s: Shape = Heavy(w: make(1))
  expect(s.weigh(make(2)) == 600)
}

test "a large array through tasks and channels" {
  val a = make(2)
  val ch = Channel<Array<u8, 200>>(capacity: 4)
  scope {
    async feed(ch, a)
    val x = await ch.recv()
    val y = await ch.recv()
    expect(x?.at(0) == 2 && y?.at(0) == 8)
  }
  expect(weight(slowMake(4)) == 800)
}

test "a megabyte array lives on the stack and copies like memcpy" {
  val big: Array<u8, 1048576> = Array.make(2)
  var big2 = big
  big2.set(5, 100)
  expect(big.at(5) == 2 && big2.at(5) == 100 && big.len() == 1048576)
}
`
