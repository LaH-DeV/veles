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

test "incrementAt updates an element and checks its index" {
  val counts: MutableList<i64> = [0, 2, 0]
  counts.incrementAt(1)
  counts.incrementAt(-1)
  expect(counts.toList() == [0, 3, 1])
  expectPanics(() => MutableList<i64>.repeat(0, 0).incrementAt(0))
  expectPanics(() => MutableList<i64>.repeat(0, 0).incrementAt(-1))
}

test "updateAt transforms once, stores and returns the new value" {
  val values: MutableList<i64> = [3, 5]
  var calls = 0
  val updated = values.updateAt(-1, value => {
    calls += 1
    value * 2
  })
  expect(updated == 10)
  expect(calls == 1)
  expect(values.toList() == [3, 10])

  val words: MutableList<string> = ["veles"]
  expect(words.updateAt(0, word => word.toUpper()) == "VELES")
  expect(words.toList() == ["VELES"])
  expectPanics(() => MutableList<i64>.repeat(0, 0).updateAt(0, value => value + 1))
}

test "eager list adapters preserve order and stop after a callback error" {
  val source = [1, 2, 3]
  expect(source.map(value => value * 10) == [10, 20, 30])
  expect(source.filter(value => value % 2 == 1) == [1, 3])
  expect(source.fold("", (text, value) => "$text$value") == "123")
  val visited: MutableList<i64> = []
  val failed = source.map(value => {
    visited.push(value)
    if (value == 2) throw AdapterStop()
    value * 10
  })
  expect(failed is Err)
  expect(visited.toList() == [1, 2])
  expect(source == [1, 2, 3])
}

test "list predicates short-circuit and have explicit empty behavior" {
  val source = [1, 2, 3]
  var calls = 0
  expect(source.any(value => {
    calls += 1
    value == 2
  }))
  expect(calls == 2)
  calls = 0
  expect(!source.all(value => {
    calls += 1
    value < 2
  }))
  expect(calls == 2)
  expect(source.find(value => value > 1) == 2)
  val empty: List<i64> = []
  calls = 0
  expect(!empty.any(value => {
    calls += 1
    true
  }))
  expect(empty.all(value => {
    calls += 1
    false
  }))
  expect(empty.find(value => {
    calls += 1
    true
  }) == null)
  expect(empty.fold(17, (total, value) => total + value) == 17)
  expect(calls == 0)
}

test "list composition handles uneven and empty inputs" {
  val source = [1, 2, 2, 3]
  expect(source.distinct() == [1, 2, 3])
  expect(source.zip(["a", "b"]) == [(1, "a"), (2, "b")])
  expect(source.chunked(3) == [[1, 2, 2], [3]])
  expect(source.windowed(3) == [[1, 2, 2], [2, 2, 3]])
  expect(source.windowed(5).isEmpty())
  expect([1, 2].flatMap(value => [value, -value]) == [1, -1, 2, -2])
  val empty: List<i64> = []
  expect(empty.distinct().isEmpty())
  expect(empty.chunked(2).isEmpty())
}
