// D121: an array is `[N x T]` in an alloca, a field or a constant table; an
// index into it is a getelementptr; a constant index into a constant is the
// value itself; and a value of 128 bytes or more that holds one lives in
// memory — copied with memmove, passed by address, returned through sret.
use io { println }

const K: Array<u32, 4> = [10, 20, 30, 40]

struct Page {
  var bytes: Array<u8, 256>
  used: i64
}

fun sum<const N: i64>(a: Array<i64, N>): i64 {
  var s = 0
  loop (i in a.indices()) {
    s += a.at(i)
  }
  s
}

fun filled(v: u8): Array<u8, 256> = Array.make(v)

fun weight(p: Page): i64 {
  var s = 0
  loop (b in p.bytes) {
    s += b.toI64()
  }
  s + p.used
}

fun main() {
  var small: Array<i64, 4> = [1, 2, 3, 4]
  small.set(1, 20)
  val far = 7
  println("${sum(small)} ${small.at(far)} ${K.at(2)} ${K.at(small.len() - 1)}")
  var page = Page(bytes: filled(1), used: 3)
  val copy = page
  page.bytes.set(0, 9)
  println("${weight(page)} ${weight(copy)}")
}
