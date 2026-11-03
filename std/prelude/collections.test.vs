test "deque grows with non-Sendable elements" {
  val queue = Deque<List<i64>>()
  loop (i in 0..<10) {
    val item: List<i64> = []
    item.push(i)
    queue.addLast(item)
  }
  expect(queue.len() == 10)
  loop (i in 0..<10) {
    val item = queue.removeFirst() ?: panic("the deque holds ten values")
    expect((item.get(0) ?: -1) == i)
  }
}

test "deque at reaches the element in place, across the ring's wrap" {
  val queue = Deque<List<i64>>()
  queue.reserve(4)
  queue.addLast([1])
  queue.addLast([2])
  val dropped = queue.removeFirst()
  queue.addLast([3])
  queue.addLast([4])
  queue.addLast([5])
  queue.at(-1)?.push(50)
  queue.at(0)?.push(20)
  expect(dropped == [1])
  expect(queue.toList() == [[2, 20], [3], [4], [5, 50]])
  expect(queue.at(4) == null)
  expect(queue.at(-5) == null)
  val copy = queue.get(1) ?: panic("four elements")
  copy.push(30)
  expect(queue.get(1) == [3])
}

test "deque at reaches a stored null" {
  val maybe = Deque<i64?>()
  maybe.addLast(null)
  val slot = maybe.at(0) ?: panic("one element")
  *slot = 9
  expect(maybe.toList() == [9])
}

error AdapterStop { }

test "atOrPut creates a missing value once and keeps existing values" {
  val values: Map<string, i64> = [:]
  var calls = 0
  val first = values.atOrPut("count", () => {
    calls += 1
    7
  })
  val second = values.atOrPut("count", () => {
    calls += 1
    99
  })
  expect(*first == 7 && *second == 7)
  expect(calls == 1)
  expect(values.get("count") == 7)
}

test "atOrPut propagates a factory error without inserting a value" {
  val values: Map<string, i64> = ["kept": 3]
  val failed = values.atOrPut("missing", () => throw AdapterStop())
  expect(failed is Err)
  expect(!values.containsKey("missing"))
  expect(values.get("kept") == 3)
  expect(*values.atOrPut("missing", () => 11) == 11)
}

test "atOrPut gives the stored value to change in place" {
  val groups: Map<string, List<string>> = [:]
  groups.atOrPut("a", () => []).push("x")
  groups.atOrPut("a", () => []).push("y")
  groups.atOrPut("b", () => []).push("z")
  expect(groups.get("a") == ["x", "y"])
  expect(groups.get("b") == ["z"])
}

test "map adapters preserve keys and leave the input unchanged" {
  val original: Map<string, i64> = ["tea": 3, "coffee": 5, "water": 0]
  val labels = original.mapValues(value => "stock-$value")
  expect(labels.len() == 3)
  expect(labels.get("tea") == "stock-3")
  expect(labels.get("coffee") == "stock-5")
  expect(labels.get("water") == "stock-0")
  val stocked = original.filter((name, value) => name != "water" && value > 0)
  expect(stocked.len() == 2)
  expect(!stocked.containsKey("water"))
  expect(original.len() == 3 && original.get("water") == 0)
  var total = 0
  original.forEach((name, value) => total += value)
  expect(total == 8)
}

test "map adapters handle empty inputs without invoking callbacks" {
  val empty: Map<string, i64> = [:]
  var calls = 0
  expect(empty.mapValues(value => {
    calls += 1
    value * 2
  }).isEmpty())
  expect(empty.filter((name, value) => {
    calls += 1
    true
  }).isEmpty())
  empty.forEach((name, value) => calls += 1)
  expect(calls == 0)
}

test "map adapters propagate callback errors" {
  val values: Map<string, i64> = ["only": 1]
  val mapped = values.mapValues(value => {
    if (value == 1) throw AdapterStop()
    value
  })
  val filtered = values.filter((name, value) => throw AdapterStop())
  val visited = values.forEach((name, value) => throw AdapterStop())
  expect(mapped is Err)
  expect(filtered is Err)
  expect(visited is Err)
  expect(values.get("only") == 1)
}
