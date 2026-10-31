test "deque grows with non-Sendable elements" {
  val queue = Deque<MutableList<i64>>()
  loop (i in 0..<10) {
    val item: MutableList<i64> = []
    item.push(i)
    queue.addLast(item)
  }
  expect(queue.len() == 10)
  loop (i in 0..<10) {
    val item = queue.removeFirst() ?: panic("the deque holds ten values")
    expect((item.at(0) ?: -1) == i)
  }
}

error AdapterStop { }

test "getOrPut creates a missing value once and keeps existing values" {
  val values: MutableMap<string, i64> = [:]
  var calls = 0
  val first = values.getOrPut("count", () => {
    calls += 1
    7
  })
  val second = values.getOrPut("count", () => {
    calls += 1
    99
  })
  expect(first == 7 && second == 7)
  expect(calls == 1)
  expect(values.get("count") == 7)
}

test "getOrPut propagates a factory error without inserting a value" {
  val values: MutableMap<string, i64> = ["kept": 3]
  val failed = values.getOrPut("missing", () => throw AdapterStop())
  expect(failed is Err)
  expect(!values.containsKey("missing"))
  expect(values.get("kept") == 3)
  expect(values.getOrPut("missing", () => 11) == 11)
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
