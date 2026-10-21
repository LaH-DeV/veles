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

test "slice is what substring gives where substring succeeds" {
  val text = "slice and substring agree"
  expect(text.slice(3, 9) == (text.substring(3, 9) ?: ""))
}
