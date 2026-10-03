// Arrays (D121): `Array<T, N>` is exactly N elements stored inline — no heap
// object, a value that copies like a struct — with its length part of its
// type. Constants can stand where a type argument goes, and a function can be
// generic over a length.
use crypto, io { println }

// A read-only table in the binary; a constant index is the value itself.
const SQUARES: Array<i64, 6> = [0, 1, 4, 9, 16, 25]

const ROWS = 3

// A struct generic over a length: Board<8> holds eight cells.
struct Board<const N: i64> {
  var cells: Array<u8, N>

  fun size(): i64 = N
}

// One function over every length: N is worked out from the argument.
fun total<const N: i64>(a: Array<i64, N>): i64 = a.fold(0, (sum, x) => sum + x)

fun zeros<const N: i64>(): Array<i64, N> = Array.make(0)

// A table of 4 KiB lives in memory: copied, passed and returned as a block.
fun checksum(page: Array<u8, 4096>): i64 {
  var s = 0
  loop (b in page) {
    s += b.toI64()
  }
  s
}

fun filled(v: u8): Array<u8, 4096> = Array.make(v)

fun main() {
  var a: Array<i64, 4> = [10, 20, 30, 40]
  val snapshot = a
  a.set(0, 7)
  println("$a $snapshot ${total(a)} ${total([1, 2, 3])} ${zeros<3>()}")

  // `at` is a T? — a T where the index is known to be in range
  val far = 9
  println("${a.at(1)} ${a.at(far)} ${SQUARES.at(3)} ${SQUARES.len()}")
  var squares = 0
  loop (i in SQUARES.indices()) {
    squares += SQUARES.at(i)
  }
  loop (&x in a) {
    *x = *x * 2
  }
  println("$squares $a ${[4, 5].toArray<2>()} ${[4, 5, 6].toArray<2>()}")

  var page = filled(1)
  val copy = page
  page.set(0, 250)
  println("${checksum(page)} ${checksum(copy)} ${page == copy}")

  val board = Board<8>(cells: Array.make(0))
  println("${board.size()} ${board.cells}")

  val ids: Array<u8, ROWS * 2> = [1, 2, 3, 4, 5, 6]
  println("${ids.map(x => x.toI64() * 10)} ${ids.any(x => x > 5)} ${ids.len()}")

  // SHA-256 keeps its state, block and schedule in arrays
  println("${crypto.sha256("abc".bytes())}")
}
