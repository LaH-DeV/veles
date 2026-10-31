test "xoshiro256 star star follows the reference state transition" {
  var rng = Rng(s0: 1, s1: 2, s2: 3, s3: 4)
  val expected: List<u64> = [11520, 0, 1509978240]
  loop (value in expected) expect(rng.nextU64() == value)
}

test "equal seeds reproduce independent sequences including signed bounds" {
  loop (seed in [0, 1, -1, 7, -9223372036854775807 - 1, 9223372036854775807]) {
    var first = Rng.seeded(seed)
    var second = Rng.seeded(seed)
    loop (_ in 0..<64) expect(first.nextU64() == second.nextU64())
  }
}

test "boolean and float consume one draw with the documented bit mapping" {
  var rng = Rng.seeded(42)
  var reference = Rng.seeded(42)
  loop (_ in 0..<128) {
    expect(rng.boolean() == ((reference.nextU64() & 1) == 1))
    val value = rng.float()
    expect(value == (reference.nextU64() >> 11).toF64() / 9007199254740992.0)
    expect(value >= 0.0 && value < 1.0)
  }
}

test "range respects exclusive upper bounds for positive negative and crossing intervals" {
  var rng = Rng.seeded(7)
  val lower = [0, -10, -5, 9223372036854775806, -9223372036854775807 - 1]
  val upper = [10, -1, 5, 9223372036854775807, -9223372036854775807]
  loop ((lo, hi) in lower.zip(upper)) {
    loop (_ in 0..<64) {
      val value = rng.range(lo, hi)
      expect(value >= lo && value < hi)
    }
  }
}

test "range supports intervals wider than the signed integer maximum" {
  val minimum: i64 = -9223372036854775807 - 1
  val maximum: i64 = 9223372036854775807
  var rng = Rng.seeded(7)
  loop ((lo, hi) in [minimum, minimum, -1].zip([maximum, 0, maximum])) {
    loop (_ in 0..<64) {
      val value = rng.range(lo, hi)
      expect(value >= lo && value < hi)
    }
  }
}

test "range refuses empty and reversed intervals" {
  expectPanics(() => Rng.seeded(1).range(0, 0))
  expectPanics(() => Rng.seeded(1).range(1, -1))
  expectPanics(() => Rng.seeded(1).range(9223372036854775807, -9223372036854775807 - 1))
}

test "empty picks and trivial shuffles do not advance the generator" {
  var rng = Rng.seeded(5)
  var reference = Rng.seeded(5)
  val empty: List<i64> = []
  expect(rng.pick(empty) == null)
  val emptyDeck: MutableList<i64> = []
  val singleton: MutableList<i64> = [42]
  rng.shuffle(emptyDeck)
  rng.shuffle(singleton)
  expect(emptyDeck.isEmpty())
  expect(singleton.at(0) == 42)
  expect(rng.nextU64() == reference.nextU64())
}

test "picks always belong to their input including nullable elements" {
  var rng = Rng.seeded(11)
  loop (_ in 0..<64) {
    expect(rng.pick([42]) == 42)
    val picked = rng.pick([10, 20, 30])
    expect(picked == 10 || picked == 20 || picked == 30)
  }
  val absent: List<i64?> = [null]
  val selected = require(rng.pick(absent))
  expect(selected == null)
}

test "seeded shuffles reproduce a permutation including duplicate elements" {
  val original: List<i64> = [0, 1, 1, 2, 3, 5, 8, 13]
  val first: MutableList<i64> = []
  val second: MutableList<i64> = []
  loop (value in original) {
    first.push(value)
    second.push(value)
  }
  var rng = Rng.seeded(17)
  var reference = Rng.seeded(17)
  rng.shuffle(first)
  reference.shuffle(second)
  expect(first.toList() == second.toList())
  expect(first.toList().sorted() == original)
}

test "module seed resets all shared generator adapters" {
  seed(23)
  var reference = Rng.seeded(23)
  loop (_ in 0..<32) {
    expect(nextU64() == reference.nextU64())
    expect(range(-10, 10) == reference.range(-10, 10))
    expect(float() == reference.float())
    expect(boolean() == reference.boolean())
    expect(pick([1, 2, 3]) == reference.pick([1, 2, 3]))
  }
  val first: MutableList<i64> = [1, 2, 3, 4]
  val second: MutableList<i64> = [1, 2, 3, 4]
  shuffle(first)
  reference.shuffle(second)
  expect(first.toList() == second.toList())
  seed(23)
  expect(nextU64() == Rng.seeded(23).nextU64())
}
