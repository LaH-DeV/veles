/// Hexadecimal text: two digits per byte, lower case on the way out,
/// either case on the way in. This is an *encoding*, not encryption —
/// anyone can read it back.
///
/// ```veles
/// use crypto
/// use hex
///
/// val etag = hex.encode(crypto.sha256(body).bytes())
/// val key  = try hex.decode("00ff10")
/// ```
///
/// `decode` is strict: an odd number of digits, a non-digit character or
/// any whitespace is an `Invalid` naming the position, never a silently
/// shorter result.

/// Text that is not hexadecimal. `position` is the byte offset of the
/// character at fault, or the length of the input when it ends too early.
public error Invalid {
  public message: string
  /// Byte offset into the input.
  public position: i64
}

/// The bytes as lower-case hex: `encode([0, 255])` is `"00ff"`.
public const fun encode(bytes: List<u8>): string => write(bytes, upper: false)

/// The bytes as upper-case hex: `encodeUpper([0, 255])` is `"00FF"`.
public const fun encodeUpper(bytes: List<u8>): string => write(bytes, upper: true)

const fun write(bytes: List<u8>, upper: bool): string {
  val out: MutableList<u8> = []
  loop (b in bytes) {
    out.push(digit(b >> 4, upper))
    out.push(digit(b & 15, upper))
  }
  out.decodeUtf8() ?: panic("hex.encode: the digits are ASCII")
}

/// One nibble as its digit byte. 48 is `'0'`, 87 + 10 is `'a'`, 55 + 10 is `'A'`.
const fun digit(nibble: u8, upper: bool): u8 =>
  if (nibble < 10) 48 +% nibble else (if (upper) 55 else 87) +% nibble

/// The bytes the text spells. Upper and lower case may be mixed; nothing
/// else is accepted — not `0x`, not whitespace, not a separator.
public fun decode(text: string): List<u8> throws Invalid {
  val n = text.len()
  if (n % 2 != 0) {
    throw Invalid(message: "hex: an odd number of digits ($n); a byte is two", position: n)
  }
  val out: MutableList<u8> = []
  var i = 0
  loop (i < n) {
    val hi = try nibbleAt(text, i)
    val lo = try nibbleAt(text, i + 1)
    out.push((hi << 4) | lo)
    i = i + 2
  }
  out
}

fun nibbleAt(text: string, at: i64): u8 throws Invalid {
  val b = text.byteAt(at)
  if (b >= 48 && b <= 57) return b -% 48   // '0'..'9'
  if (b >= 97 && b <= 102) return b -% 87  // 'a'..'f'
  if (b >= 65 && b <= 70) return b -% 55   // 'A'..'F'
  throw Invalid(message: "hex: ${describe(b)} at $at is not a hex digit", position: at)
}

/// A byte as it should read in a message: the character when it is
/// printable ASCII, `0x0a` when it is not.
fun describe(b: u8): string {
  if (b >= 32 && b < 127) {
    val one: MutableList<u8> = [b]
    return "'${one.decodeUtf8() ?: "?"}'"
  }
  val d = "0123456789abcdef"
  val hi = (b >> 4).toI64()
  val lo = (b & 15).toI64()
  "0x${d.substring(hi, hi + 1) ?: ""}${d.substring(lo, lo + 1) ?: ""}"
}
