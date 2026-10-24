// std/build (D160, D204): the target and the program are constants, so a
// test can be skipped on them; a member's predicate (D206) is a constant
// too, and so is a `Case.Any` parse (D205), which is how the compiler's
// words become the enums.

@SkipIf(target.os.isWindows(), "not on Windows")
test "a test skipped on Windows sees another OS" {
  expect(!target.os.isWindows())
}

@SkipIf(!target.os.isWindows(), "only on Windows")
test "a test kept only on Windows sees Windows" {
  expect(target.os.isWindows())
}

test "the compiler's words read as the enums" {
  expect(Os.parse("macos", case: Case.Any) == Os.MacOS)
  expect(Os.parse("linux", case: Case.Any) == Os.Linux)
  expect(Arch.parse("arm64", case: Case.Any) == Arch.Arm64)
  expect(Arch.parse("x64", case: Case.Any) == Arch.X64)
}
