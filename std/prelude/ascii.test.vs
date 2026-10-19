test classes "the ASCII classes of a byte" {
  expect('7'.isAsciiDigit())
  expect(!'a'.isAsciiDigit())
  expect('a'.isAsciiLower() && !'A'.isAsciiLower())
  expect('Z'.isAsciiUpper() && !'z'.isAsciiUpper())
  expect('q'.isAsciiLetter() && 'Q'.isAsciiLetter() && !'1'.isAsciiLetter())
  expect('1'.isAsciiAlphanumeric() && 'x'.isAsciiAlphanumeric() && !'_'.isAsciiAlphanumeric())
  expect(' '.isAsciiSpace() && '\t'.isAsciiSpace() && '\n'.isAsciiSpace() && '\r'.isAsciiSpace())
  expect(!'a'.isAsciiSpace())
  expect('/'.isAsciiDigit() == false && ':'.isAsciiDigit() == false)
}

test digits "the value of a digit in a radix" {
  expect('7'.digitValue() == 7)
  expect('0'.digitValue() == 0)
  expect('f'.digitValue(16) == 15)
  expect('F'.digitValue(16) == 15)
  expect('g'.digitValue(16) == null)
  expect('z'.digitValue(36) == 35)
  expect('9'.digitValue(8) == null)
  expect('1'.digitValue(2) == 1)
  expect('2'.digitValue(2) == null)
  expect(' '.digitValue() == null)
  expect('a'.isAsciiHexDigit() && 'F'.isAsciiHexDigit() && '9'.isAsciiHexDigit() && !'g'.isAsciiHexDigit())
}

test cases "the ASCII case of a byte" {
  expect('A'.toAsciiLower() == 'a' && 'Z'.toAsciiLower() == 'z' && 'm'.toAsciiLower() == 'm')
  expect('a'.toAsciiUpper() == 'A' && 'z'.toAsciiUpper() == 'Z' && 'M'.toAsciiUpper() == 'M')
  expect('1'.toAsciiLower() == '1' && '@'.toAsciiUpper() == '@' && '['.toAsciiLower() == '[')
}

test boundaries "bytes next to the ranges" {
  expect(!'/'.isAsciiDigit() && !':'.isAsciiDigit())
  expect(!'@'.isAsciiLetter() && !'['.isAsciiLetter())
  expect(!'`'.isAsciiLetter() && !'{'.isAsciiLetter())
  expect(!128.wrapU8().isAsciiLetter())
}
