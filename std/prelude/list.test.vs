// Tests of list growth through `addAll` / `concat`: a list of elements wider
// than a byte must get room for them all. (The runtime once sized the grown
// buffer in elements rather than bytes, so appending to a list of structs
// wrote past its end and corrupted whatever was allocated next — seen as a
// crash in the collector, far from the append.)

struct Wide {
  name:  string
  count: i64
  more:  i64
}

struct Cell {
  v: i64
}

test fun wide(n: i64): Wide => Wide(name: "item-$n", count: n, more: n * 3)

test "addAll grows a list of structs to hold every element, and neighbours are untouched" {
  val cells: MutableList<Cell> = []
  var wrong: i64 = 0
  loop (round in 0..<3000) {
    val out: MutableList<Wide> = []
    out.addAll([wide(1), wide(2), wide(3), wide(4), wide(5)])
    out.addAll([wide(6), wide(7)])
    // small objects allocated around the grown buffer
    loop (k in 0..<4) {
      cells.push(Cell(v: round.toI64() * 10 + k))
    }
    if (out.len() != 7) wrong += 1
    val last = out.at(6) ?: wide(0)
    if (last.name != "item-7" || last.more != 21) wrong += 1
  }
  loop (i in 0..<cells.len()) {
    val c = cells.at(i) ?: Cell(v: -1)
    if (c.v != (i / 4).toI64() * 10 + (i % 4).toI64()) wrong += 1
  }
  expect(wrong == 0)
}

test "concat of lists of wide elements keeps every element" {
  val base: List<Wide> = [wide(1), wide(2), wide(3), wide(4), wide(5)]
  val joined = base.concat([wide(6), wide(7), wide(8)]).concat([])
  expect(joined.len() == 8)
  expect((joined.at(7) ?: wide(0)).name == "item-8")
  expect((joined.at(0) ?: wide(0)).count == 1)
}

test "addAll of eight-byte elements past the capacity" {
  val out: MutableList<i64> = []
  loop (_ in 0..<2000) {
    out.addAll([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
  }
  expect(out.len() == 20000)
  expect(out.sum() == 110000)
}
