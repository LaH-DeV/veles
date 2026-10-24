// string.slice (D192): a substring that cannot fail.

test "slice takes the bytes in from..<to" {
  expect("hello".slice(1, 3) == "el")
  expect("hello".slice(0, 5) == "hello")
  expect("hello".slice(2, 2) == "")
}

test "slice clamps both ends to the string" {
  expect("hello".slice(-5, 2) == "he")
  expect("hello".slice(3, 99) == "lo")
  expect("hello".slice(-9, 99) == "hello")
  expect("hello".slice(99, 120) == "")
}

test "slice of a reversed or empty range is empty" {
  expect("hello".slice(4, 2) == "")
  expect("".slice(0, 3) == "")
}

test "slice moves a bound inside a character outward to its edge" {
  // "héllo": h, then é as two bytes (indexes 1 and 2), then l, l, o
  expect("héllo".slice(2, 3) == "é")   // the start is inside é
  expect("héllo".slice(0, 2) == "hé")  // the end is inside é
  expect("héllo".slice(2, 2) == "")    // empty stays empty
  expect("héllo".slice(1, 3) == "é")   // on edges already
  expect("日本語".slice(1, 2) == "日")     // three-byte characters
  expect("日本語".slice(4, 8) == "本語")
}

// toLower, toUpper and replace: one pass, one copy (or none)

test "case mapping changes ASCII letters only" {
  expect("Hello, World".toLower() == "hello, world")
  expect("Hello, World".toUpper() == "HELLO, WORLD")
  expect("ÉtÉ Ab".toLower() == "ÉtÉ ab")  // non-ASCII bytes are kept
  expect("already".toLower() == "already")
  expect("".toUpper() == "")
}

test "replace takes every occurrence, left to right" {
  expect("a b c".replace(" ", "-") == "a-b-c")
  expect("aXa".replace("a", "bb") == "bbXbb")
  expect("aaa".replace("aa", "b") == "ba")
  expect("abc".replace("x", "y") == "abc")
  expect("aa".replace("aa", "") == "")
  expect("abc".replace("", "z") == "abc")
}

const SLUG: string = "  Hello World ".trim().toLower().replace(" ", "-")

test "the same in a constant" {
  expect(SLUG == "hello-world")
}

// string.startsWith(prefix, startIndex) (D202)

test "startsWith at a byte index compares from there" {
  expect("hello".startsWith("ll", 2))
  expect("hello".startsWith("lo", startIndex: 3))
  expect(!"hello".startsWith("ll", 3))
  expect("hello".startsWith(prefix: "he", startIndex: 0))
  expect(!"hello".startsWith("lox", 3))  // runs past the end
}

test "startsWith answers false for a startIndex out of range" {
  expect(!"hello".startsWith("h", -1))
  expect(!"hello".startsWith("", 6))
  expect("hello".startsWith("", 5))  // the empty prefix at the end
  expect("".startsWith("", 0))
}

const ARROW_AT_1 = "a=>b".startsWith("=>", 1)

test "startsWith at a byte index works in a constant" {
  expect(ARROW_AT_1)
}

test "slice is what substring gives where substring succeeds" {
  val text = "slice and substring agree"
  expect(text.slice(3, 9) == (text.substring(3, 9) ?: ""))
}
