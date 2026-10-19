// Prelude — ASCII classes of a byte (D184). What every scanner, parser and
// protocol reader asks of the bytes it walks: is this a digit, a letter, a
// blank, and what number does this digit stand for. A `u8` is a byte, not a
// code point, so the names say ASCII: a Unicode question is asked of a code
// point (`utf8`), never of a byte.

extend u8 {
  /// True for `0`–`9`.
  public const fun isAsciiDigit(): bool => this >= '0' && this <= '9'

  /// True for `a`–`z`.
  public const fun isAsciiLower(): bool => this >= 'a' && this <= 'z'

  /// True for `A`–`Z`.
  public const fun isAsciiUpper(): bool => this >= 'A' && this <= 'Z'

  /// True for `a`–`z` and `A`–`Z`.
  public const fun isAsciiLetter(): bool => this.isAsciiLower() || this.isAsciiUpper()

  /// True for an ASCII letter or digit.
  public const fun isAsciiAlphanumeric(): bool => this.isAsciiLetter() || this.isAsciiDigit()

  /// True for `0`–`9`, `a`–`f` and `A`–`F`.
  public const fun isAsciiHexDigit(): bool => this.digitValue(16) != null

  /// True for space, tab, newline, carriage return, form feed and vertical tab.
  public const fun isAsciiSpace(): bool => this == ' ' || (this >= 9 && this <= 13)

  /// The byte with `A`–`Z` made `a`–`z`; any other byte as it is.
  public const fun toAsciiLower(): u8 => if (this.isAsciiUpper()) this +% 32 else this

  /// The byte with `a`–`z` made `A`–`Z`; any other byte as it is.
  public const fun toAsciiUpper(): u8 => if (this.isAsciiLower()) this -% 32 else this

  /// The number this digit stands for in `radix` (2 to 36): `0`–`9` are 0 to 9,
  /// `a`–`z` and `A`–`Z` are 10 to 35. Null where the byte is not a digit of that
  /// radix. A radix outside 2 to 36 is a bug in the caller and panics.
  ///
  /// ```veles
  /// '7'.digitValue()        // 7
  /// 'f'.digitValue(16)      // 15
  /// 'g'.digitValue(16)      // null
  /// ```
  public const fun digitValue(radix: i64 = 10): i64? {
    if (radix < 2 || radix > 36) panic("u8.digitValue: the radix is 2 to 36, not $radix")
    val value: i64 = when {
      this.isAsciiDigit() => (this - '0').toI64()
      this.isAsciiLower() => (this - 'a').toI64() + 10
      this.isAsciiUpper() => (this - 'A').toI64() + 10
      else                => return null
    }
    if (value < radix) value else null
  }
}
