test "Duration.parse rejects integer overflow without panicking" {
  expect(Duration.parse("9223372036854775807ns") == Duration.nanos(9223372036854775807))
  expect(Duration.parse("9223372036854775808ns") == null)
}

test "Duration.parse rejects fractional overflow without panicking" {
  expect(Duration.parse("9223372036.854775807s") == Duration.nanos(9223372036854775807))
  expect(Duration.parse("9223372036.854775808s") == null)
}

test "Duration.toString round trips both i64 bounds" {
  val min = Duration.nanos(-9223372036854775807 - 1)
  val max = Duration.nanos(9223372036854775807)
  expect(Duration.parse("-9223372036854775808ns") == min)
  expect(min.toString() == "-106751d23h47m16.854775808s")
  expect(Duration.parse(min.toString()) == min)
  expect(max.toString() == "106751d23h47m16.854775807s")
  expect(Duration.parse(max.toString()) == max)
}
