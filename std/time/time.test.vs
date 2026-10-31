test "offset of rejects extreme hours before multiplying" {
  expect(Offset.of(18)?.totalMinutes() == 1080)
  expect(Offset.of(-18)?.totalMinutes() == -1080)
  expect(Offset.of(19) == null)
  expect(Offset.of(-19) == null)
  expect(Offset.of(18, 1) == null)
  expect(Offset.of(-18, 1) == null)
  expect(Offset.of(9223372036854775807) == null)
  expect(Offset.of(-9223372036854775807 - 1) == null)
}
