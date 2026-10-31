test "IEC byte units are powers of 1024" {
  expect(B == 1)
  expect(KiB == 1024)
  expect(MiB == KiB * 1024)
  expect(GiB == MiB * 1024)
  expect(TiB == GiB * 1024)
  expect(PiB == TiB * 1024)
  expect(EiB == PiB * 1024)
}
