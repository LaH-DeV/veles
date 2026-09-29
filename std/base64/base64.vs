/// Base64 (RFC 4648): bytes as text that survives anything expecting
/// characters. `encode`/`decode` use the standard alphabet with `+`, `/`
/// and `=` padding; `encodeUrl`/`decodeUrl` use the URL-safe alphabet
/// (`-`, `_`) with no padding, which is what JWTs, cookies and query
/// parameters want.
///
/// This is an *encoding*, not encryption — base64 hides nothing.
///
/// ```veles
/// use base64
///
/// val text = base64.encodeUrl(crypto.randomBytes(32))   // no '=' to escape
/// val bytes = try base64.decodeUrl(text)
/// ```
///
/// Both decoders take input with or without padding, and both are strict
/// about everything else: a character from the other alphabet, an `=` in
/// the middle, a line break, a length that cannot be a whole number of
/// bytes, or bits set in the last character that the byte count does not
/// use (a *non-canonical* encoding, the kind that lets two different
/// texts mean one signature) is an `Invalid` naming the position.

const STANDARD: string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
const URL_SAFE: string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

const PAD: u8 = 61    // '='
const PLUS: u8 = 43   // '+'
const SLASH: u8 = 47  // '/'
const MINUS: u8 = 45  // '-'
const UNDER: u8 = 95  // '_'

/// Text that is not base64. `position` is the byte offset of the character
/// at fault, or the length of the input when it ends too early.
public error Invalid {
  public message: string
  /// Byte offset into the input.
  public position: i64
}

/// The bytes in standard base64, padded to a multiple of four characters.
public fun encode(bytes: List<u8>): string = write(bytes, STANDARD, pad: true)

/// The bytes in URL-safe base64 (RFC 4648 §5) with no padding: safe in a
/// path segment, a query parameter, a cookie value or a JWT.
public fun encodeUrl(bytes: List<u8>): string = write(bytes, URL_SAFE, pad: false)

/// The bytes the standard base64 text spells; padding is optional.
public fun decode(text: string): List<u8> throws Invalid = try read(text, url: false)

/// The bytes the URL-safe base64 text spells; padding is optional.
public fun decodeUrl(text: string): List<u8> throws Invalid = try read(text, url: true)

/// How many characters `encode` produces for `n` bytes (`encodeUrl` drops
/// the padding): useful for a size ceiling before you encode.
public fun encodedLen(n: i64, pad: bool = true): i64 {
  if (pad) return ((n + 2) / 3) * 4
  val tail = when (n % 3) {
    0    => 0
    1    => 2
    else => 3
  }
  (n / 3) * 4 + tail
}

// ---------------------------------------------------------------------------
// encoding

/// Byte `i` of the input, widened. `write` reads below `n` only: the loop
/// runs while `i + 3 <= n`, and the tail reads just the `left` bytes after it.
fun input(bytes: List<u8>, i: i64): i64 = (bytes.at(i) ?: panic("base64.encode: every read is below the input length")).toI64()

fun write(bytes: List<u8>, alphabet: string, pad: bool): string {
  val out: MutableList<u8> = []
  val n = bytes.len()
  var i = 0
  loop (i + 3 <= n) {
    val v = ((input(bytes, i)) << 16) |
      ((input(bytes, i + 1)) << 8) |
      (input(bytes, i + 2))
    out.push(alphabet.byteAt((v >> 18) & 63))
    out.push(alphabet.byteAt((v >> 12) & 63))
    out.push(alphabet.byteAt((v >> 6) & 63))
    out.push(alphabet.byteAt(v & 63))
    i = i + 3
  }
  val left = n - i
  if (left == 1) {
    val a = input(bytes, i)
    out.push(alphabet.byteAt(a >> 2))
    out.push(alphabet.byteAt((a << 4) & 63))
    if (pad) {
      out.push(PAD)
      out.push(PAD)
    }
  } else if (left == 2) {
    val a = input(bytes, i)
    val b = input(bytes, i + 1)
    out.push(alphabet.byteAt(a >> 2))
    out.push(alphabet.byteAt(((a << 4) | (b >> 4)) & 63))
    out.push(alphabet.byteAt((b << 2) & 63))
    if (pad) out.push(PAD)
  }
  out.decodeUtf8() ?: panic("base64.encode: the alphabet is ASCII")
}

// ---------------------------------------------------------------------------
// decoding

fun read(text: string, url: bool): List<u8> throws Invalid {
  val n = text.len()
  // trailing '=' first: it is not data, and it fixes the expected length
  var end = n
  var padding = 0
  loop (end > 0 && text.byteAt(end - 1) == PAD) {
    padding = padding + 1
    end = end - 1
  }
  if (padding > 2) {
    throw Invalid(message: "base64: $padding padding characters at the end; at most two", position: end)
  }
  if (padding > 0 && n % 4 != 0) {
    throw Invalid(message: "base64: padded input must be a multiple of four characters, got $n", position: n)
  }

  val out: MutableList<u8> = []
  var acc = 0
  var bits = 0
  var i = 0
  loop (i < end) {
    acc = (acc << 6) | (try sextet(text.byteAt(i), url, i))
    bits = bits + 6
    if (bits >= 8) {
      bits = bits - 8
      out.push(((acc >> bits) & 255).wrapU8())
    }
    i = i + 1
  }

  if (bits >= 6) {
    throw Invalid(message: "base64: the input ends inside a byte — one character is left over", position: end)
  }
  if (bits > 0 && (acc & ((1 << bits) - 1)) != 0) {
    throw Invalid(message: "base64: the last character sets $bits bits the byte count does not use (not a canonical encoding)", position: end - 1)
  }
  val want = when (bits) {
    0    => 0
    2    => 1
    else => 2
  }
  if (padding > 0 && padding != want) {
    throw Invalid(message: "base64: $padding padding characters, but the data needs $want", position: end)
  }
  out
}

fun sextet(b: u8, url: bool, at: i64): i64 throws Invalid {
  if (b >= 65 && b <= 90) return (b -% 65).toI64()        // 'A'..'Z' → 0..25
  if (b >= 97 && b <= 122) return (b -% 97).toI64() + 26  // 'a'..'z' → 26..51
  if (b >= 48 && b <= 57) return (b -% 48).toI64() + 52   // '0'..'9' → 52..61
  if (url) {
    if (b == MINUS) return 62
    if (b == UNDER) return 63
    if (b == PLUS || b == SLASH) {
      throw Invalid(message: "base64: ${describe(b)} at $at is standard base64; use decode, not decodeUrl", position: at)
    }
  } else {
    if (b == PLUS) return 62
    if (b == SLASH) return 63
    if (b == MINUS || b == UNDER) {
      throw Invalid(message: "base64: ${describe(b)} at $at is URL-safe base64; use decodeUrl, not decode", position: at)
    }
  }
  if (b == PAD) {
    throw Invalid(message: "base64: '=' at $at, before the end of the input", position: at)
  }
  if (b == 9 || b == 10 || b == 13 || b == 32) {
    throw Invalid(message: "base64: whitespace at $at; strip line breaks before decoding", position: at)
  }
  throw Invalid(message: "base64: ${describe(b)} at $at is not a base64 character", position: at)
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
