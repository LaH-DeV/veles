test "common categories and scripts" {
  expect(isLetter('A'.toI64()) && isLetter(0x03B1))
  expect(!isLetter('7'.toI64()) && !isLetter(0xD800))
  expect(isDecimalDigit('7'.toI64()) && isDecimalDigit(0x0661))
  expect(!isDecimalDigit(0x00B2))
  expect(isWhiteSpace(' ') && isWhiteSpace(0x00A0))
  expect(!isWhiteSpace(0x200B))
  expect(hasProperty("Greek", 0x03B1) == true)
  expect(hasProperty("Lu", 'A'.toI64()) == true)
  expect(hasProperty("Greek", 'A'.toI64()) == false)
  expect(hasProperty("not-a-property", 'A'.toI64()) == null)
  expect(hasProperty("Cs", 0xD800) == false)
}

test "printable characters match the compiler's Unicode policy" {
  expect(isPrintable(' '.toI64()) && isPrintable('A'.toI64()) && isPrintable(0x1F600))
  expect(!isPrintable('\n') && !isPrintable(0x7F) && !isPrintable(0x00A0))
  expect(!isPrintable(0x110000) && !isPrintable(0xD800))
}

test "identifier predicates match UAX 31 and the Veles underscore rule" {
  expect(isIdentifierStart('_'.toI64()) && isIdentifierStart('A'.toI64()) && isIdentifierStart(0x03B1))
  expect(!isIdentifierStart('7'.toI64()) && !isIdentifierStart(0x200B))
  expect(isIdentifierContinue('_'.toI64()) && isIdentifierContinue('7'.toI64()) && isIdentifierContinue(0x0301))
  expect(!isIdentifierContinue(0x200B) && !isIdentifierContinue(0xD800))
}
