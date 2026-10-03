/// One Unicode code point at a time, over the bytes a `string` is made of.
///
/// Veles text is UTF-8 and is indexed in bytes (D18): `len()` counts bytes,
/// `byteAt(i)` reads one, and `substring` refuses an index that would split
/// a character. That is the right default — it is what makes `indexOf` and
/// `substring` linear and allocation-free — but a scanner that has to look
/// at *characters* (a lexer reading a character literal, a terminal
/// deciding a column width, a validator over bytes off a socket) needs to
/// step one code point at a time, and this module is that step.
///
/// ```veles
/// use utf8
///
/// var i = 0
/// loop (i < s.len()) {
///   val r = utf8.decode(s, i) ?: break      // never null on a string
///   io.println("U+${r.code.toString(radix: 16)}")
///   i += r.size
/// }
///
/// val out: MutableList<u8> = []
/// utf8.encodeTo(out, 0x1F600)               // four bytes appended
/// ```
///
/// ## What is valid
///
/// The decoder is strict in the sense RFC 3629 and the Unicode standard
/// (Table 3-7) mean: a sequence is accepted only in its shortest form, the
/// surrogate range U+D800..U+DFFF is not a code point, and nothing above
/// U+10FFFF exists. An overlong two-byte NUL, a CESU-8 surrogate pair and a
/// five-byte sequence are each rejected, because each is a way to smuggle a
/// character past a filter that decoded more loosely.
///
/// ## Where the nulls are
///
/// A `string` is valid UTF-8 by construction — the only ways to make one
/// are from a literal, from other strings, or through `decodeUtf8`, which
/// checks — so `decode` on a `string` returns null only for an index that
/// is outside it or in the middle of a character. Bytes are the untrusted
/// case, and `decodeBytes` is where the table above earns its keep.
///
/// A caller that must keep going after bad bytes does what every other
/// UTF-8 decoder does: report `replacement` (U+FFFD) and advance one byte.
/// That is one character per bad byte, which is the substitution the
/// Unicode standard recommends and what Go, Rust and browsers all produce,
/// so text recovered this way agrees with theirs byte for byte.

/// The largest code point: U+10FFFF.
public const maxCode: i64 = 0x10FFFF

/// U+FFFD REPLACEMENT CHARACTER — what a decoder substitutes for bytes it
/// cannot read.
public const replacement: i64 = 0xFFFD

/// The most bytes one code point takes: 4.
public const maxSize: i64 = 4

/// A code point and the number of bytes it occupied: `size` is what to add
/// to an index to reach the next one, 1 to 4.
public struct Rune {
  /// The Unicode scalar value.
  public code: i64
  /// Its width in bytes, 1 to 4.
  public size: i64
}

// ---------------------------------------------------------------------------
// code points

/// True when `code` is a Unicode scalar value: 0 to U+10FFFF, and not half
/// of a surrogate pair. These are exactly the values that can be encoded.
public fun isScalar(code: i64): bool = code >= 0 && code <= maxCode && !isSurrogate(code)

/// True in U+D800..U+DFFF — the halves UTF-16 uses to spell a code point
/// above U+FFFF. They are not characters and never appear in UTF-8.
public fun isSurrogate(code: i64): bool = code >= 0xD800 && code <= 0xDFFF

/// How many bytes `code` encodes to, or null when it is not a scalar value.
public fun size(code: i64): i64? {
  if (!isScalar(code)) return null
  if (code < 0x80) return 1
  if (code < 0x800) return 2
  if (code < 0x10000) return 3
  4
}

/// The code point a UTF-16 surrogate pair spells, or null when the halves
/// are not a high one followed by a low one.
///
/// UTF-8 has no surrogates, but the formats that quote a character in hex
/// mostly borrowed UTF-16's escape — a pair of four-digit hex escapes, in
/// JSON, JavaScript and Java alike — so a decoder for one of those needs
/// this on the way in. `std/json` is the caller in this repository.
public fun combineSurrogates(high: i64, low: i64): i64? {
  if (high < 0xD800 || high > 0xDBFF) return null
  if (low < 0xDC00 || low > 0xDFFF) return null
  0x10000 + ((high - 0xD800) << 10) + (low - 0xDC00)
}

// ---------------------------------------------------------------------------
// bytes

/// True for a byte in 0x80..0xBF: the second, third or fourth byte of a
/// sequence, never the first.
public fun isContinuation(b: u8): bool = (b & 0xC0) == 0x80

/// True when `b` may begin a code point — that is, when it is not a
/// continuation byte. Walking back to a character boundary is
/// `loop (!utf8.isStart(s.byteAt(i))) i -= 1`.
public fun isStart(b: u8): bool = !isContinuation(b)

// ---------------------------------------------------------------------------
// decoding

/// The code point at byte offset `at`, or null when `at` is outside the
/// string or lands in the middle of a character.
public fun decode(s: string, at: i64): Rune? {
  val n = s.len()
  if (at < 0 || at >= n) return null
  val rest = n - at
  step(
    s.byteAt(at),
    if (rest > 1) s.byteAt(at + 1) else 0,
    if (rest > 2) s.byteAt(at + 2) else 0,
    if (rest > 3) s.byteAt(at + 3) else 0,
    rest,
  )
}

/// The code point that ends just before byte offset `before` — the step
/// backwards that `decode` is forwards — or null when there is none or the
/// bytes there are not one.
public fun decodeLast(s: string, before: i64): Rune? {
  val end = before.min(s.len())
  if (end <= 0) return null
  // a code point is at most four bytes, so its first byte is within three
  var start = end - 1
  loop (start > 0 && end - start < maxSize && isContinuation(s.byteAt(start))) {
    start -= 1
  }
  val r = decode(s, start) ?: return null
  // a sequence that stops short of `before` means the bytes just before it
  // were not a character either: report nothing rather than a misaligned one
  if (start + r.size != end) return null
  r
}

/// The code point at index `at` of a byte buffer, or null when `at` is
/// outside it or the bytes there are not a well-formed, shortest-form
/// sequence.
///
/// This is the one to reach for on bytes that came from outside — a file, a
/// socket, an FFI buffer — since, unlike a `string`, they carry no promise.
public fun decodeBytes(bytes: List<u8>, at: i64): Rune? {
  val n = bytes.len()
  if (at < 0 || at >= n) return null
  step(
    bytes.at(at),
    bytes.atOrDefault(at + 1, 0),
    bytes.atOrDefault(at + 2, 0),
    bytes.atOrDefault(at + 3, 0),
    n - at,
  )
}

/// The decoder itself, over the four bytes at the cursor and how many of
/// them are really there.
///
/// The ranges are Table 3-7 of the Unicode standard, which is the shortest
/// statement of well-formed UTF-8 there is: the first byte fixes the length
/// and the *range of the second*, and it is that range — not a check of the
/// value afterwards — that rejects both the overlong forms and the
/// surrogates. `E0` may only be followed by `A0..BF` (below that the code
/// point would fit in two bytes), `ED` only by `80..9F` (above that is a
/// surrogate), `F0` only by `90..BF`, and `F4` only by `80..8F` (above that
/// is past U+10FFFF).
fun step(b0: u8, b1: u8, b2: u8, b3: u8, rest: i64): Rune? {
  if (b0 < 0x80) return Rune(code: b0.toI64(), size: 1)
  // 0x80..0xC1 is a stray continuation byte or an overlong two-byte lead
  if (b0 < 0xC2 || b0 > 0xF4) return null

  val size = if (b0 < 0xE0) 2 else if (b0 < 0xF0) 3 else 4
  if (rest < size) return null

  val lo: u8 = when {
    b0 == 0xE0 => 0xA0
    b0 == 0xF0 => 0x90
    else       => 0x80
  }
  val hi: u8 = when {
    b0 == 0xED => 0x9F
    b0 == 0xF4 => 0x8F
    else       => 0xBF
  }
  if (b1 < lo || b1 > hi) return null

  val mask: u8 = when {
    size == 2 => 0x1F
    size == 3 => 0x0F
    else      => 0x07
  }
  var code = (((b0 & mask).toI64()) << 6) | ((b1 & 0x3F).toI64())
  if (size > 2) {
    if (!isContinuation(b2)) return null
    code = (code << 6) | ((b2 & 0x3F).toI64())
  }
  if (size > 3) {
    if (!isContinuation(b3)) return null
    code = (code << 6) | ((b3 & 0x3F).toI64())
  }
  Rune(code, size)
}

// ---------------------------------------------------------------------------
// encoding

/// Appends the UTF-8 bytes of `code` to `out` and reports how many there
/// were, 1 to 4.
///
/// A `code` that is not a scalar value — negative, above U+10FFFF, or a
/// surrogate — writes U+FFFD instead of failing, which is what Go's
/// `utf8.EncodeRune` and Rust's lossy conversions do: a text pipeline that
/// stops dead on one bad code point is a denial of service, and one that
/// emits invalid UTF-8 is worse than either. Check with `isScalar` where
/// the difference matters.
public fun encodeTo(out: MutableList<u8>, code: i64): i64 {
  val cp = if (isScalar(code)) code else replacement
  when {
    cp < 0x80    => {
      out.push(cp.wrapU8())
      1
    }
    cp < 0x800   => {
      out.push((0xC0 | (cp >> 6)).wrapU8())
      out.push((0x80 | (cp & 0x3F)).wrapU8())
      2
    }
    cp < 0x10000 => {
      out.push((0xE0 | (cp >> 12)).wrapU8())
      out.push((0x80 | ((cp >> 6) & 0x3F)).wrapU8())
      out.push((0x80 | (cp & 0x3F)).wrapU8())
      3
    }
    else         => {
      out.push((0xF0 | (cp >> 18)).wrapU8())
      out.push((0x80 | ((cp >> 12) & 0x3F)).wrapU8())
      out.push((0x80 | ((cp >> 6) & 0x3F)).wrapU8())
      out.push((0x80 | (cp & 0x3F)).wrapU8())
      4
    }
  }
}

/// The UTF-8 bytes of `code`, as their own list. `encodeTo` is the one to
/// use in a loop; this one is for a single character.
public fun encode(code: i64): List<u8> {
  val out: MutableList<u8> = []
  encodeTo(out, code)
  out.toList()
}

/// `code` as a one-character string — U+FFFD when it is not a scalar value,
/// on the same reasoning as `encodeTo`.
public fun char(code: i64): string =
  encode(code).decodeUtf8() ?: panic("utf8.char: an encoded scalar value is valid UTF-8")

// ---------------------------------------------------------------------------
// whole buffers

/// True when every byte of `bytes` is part of a well-formed sequence — the
/// same question `decodeUtf8` answers, without building the string.
public fun isValid(bytes: List<u8>): bool {
  var i: i64 = 0
  val n = bytes.len()
  loop (i < n) {
    val r = decodeBytes(bytes, i) ?: return false
    i += r.size
  }
  true
}

/// How many code points `bytes` holds, counting each byte that is not part
/// of a well-formed sequence as one — so the count matches the number of
/// characters a decoder that substitutes U+FFFD would produce, and never
/// depends on the buffer being valid.
public fun count(bytes: List<u8>): i64 {
  var i: i64 = 0
  var n: i64 = 0
  loop (i < bytes.len()) {
    i += decodeBytes(bytes, i)?.size ?: 1
    n += 1
  }
  n
}
