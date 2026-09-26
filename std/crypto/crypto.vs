/// Cryptographic primitives: message digests (SHA-256, SHA-384, SHA-512
/// and SHA-1), HMAC, constant-time comparison, random bytes from the
/// operating system, and UUIDs.
///
/// ```veles
/// use crypto
///
/// val mac = crypto.hmacSha256(secret, body.bytes())
/// if (mac == crypto.Digest.of(try base64.decodeUrl(header))) { … }
///
/// val session = base64.encodeUrl(try crypto.randomBytes(32))
/// val id = crypto.uuidV7()
/// ```
///
/// Two rules hold everywhere in this module:
///
/// - **Comparison is constant time.** `Digest` compares with `==` in time
///   that depends only on the length, never on where the first difference
///   is; `equalBytes` does the same for raw bytes. Comparing a MAC with
///   `a.bytes() == b.bytes()` would short-circuit and tell an attacker how
///   many leading bytes they guessed, so `Digest` does not expose the
///   comparison that way — ask for `bytes()` only when you are about to
///   *send* the digest.
/// - **Randomness comes from the operating system.** `randomBytes` is
///   `BCryptGenRandom` on Windows and `getrandom`/`/dev/urandom`
///   elsewhere; `std/random` is a fast pseudo-random generator and is
///   never a substitute.
///
/// Every digest type is a `Hasher`: feed it with `update` as many times as
/// you like, then `finish()` once. `sha256(data)` is the one-shot form.
use base64, hex

// ---------------------------------------------------------------------------
// digests

/// The output of a hash or a MAC: a short, fixed-length byte string.
///
/// `"$d"` and `toHex()` print lower-case hex, `toBase64Url()` gives the
/// form a URL or a JWT wants, and `==` compares in constant time. Wrap
/// bytes that arrived from outside with `Digest.of` to compare them
/// safely.
public struct Digest {
  private data: List<u8>

  /// Bytes from elsewhere — a signature out of a header, a hash out of a
  /// database — as a `Digest`, so that comparing them uses `==` and is
  /// constant time.
  public static fun of(bytes: List<u8>): Digest = Digest(data: bytes)

  /// The raw bytes. For *sending* the digest; comparing two `bytes()`
  /// results with `==` is not constant time, compare the digests instead.
  public fun bytes(): List<u8> = self.data

  /// Length in bytes (32 for SHA-256, 64 for SHA-512, 20 for SHA-1).
  public fun len(): i64 = self.data.len()

  /// Lower-case hexadecimal, the usual way to write a digest down.
  public fun toHex(): string = hex.encode(self.data)

  /// URL-safe base64 with no padding: a JWT signature, an `ETag`, a
  /// cookie value.
  public fun toBase64Url(): string = base64.encodeUrl(self.data)

  /// The first `n` bytes, as a digest: a shorter tag (an `ETag`, a cache
  /// key). Truncation is the accepted way to shorten a digest; taking
  /// bytes out of the middle is not.
  public fun prefix(n: i64): Digest = Digest(data: self.data.take(n))

  implement Display {
    fun toString(): string = hex.encode(self.data)
  }

  implement Equatable {
    /// Constant time in the contents: every byte is read whatever the
    /// first difference is.
    fun equals(other: Digest): bool = equalBytes(self.data, other.data)
  }

  implement Hashable {
    /// A digest is already uniform, so its first bytes are its hash.
    fun hash(): i64 {
      var h = 0
      var i = 0
      loop (i < 8) {
        val b = self.data.at(i) ?: break
        h = (h << 8) | (b as i64)
        i = i + 1
      }
      h
    }
  }
}

/// A hash function fed in pieces. `start()` opens one, `update` adds bytes
/// (any number of times, any sizes — the result depends only on the
/// concatenation), `finish()` closes it and returns the digest.
///
/// A hasher is finished once: `update` after `finish` panics, and
/// `finish` twice returns the same digest.
public trait Hasher {
  /// A fresh hasher.
  static fun start(): Self

  /// The name the wire formats use: `"SHA-256"`.
  static fun algorithm(): string

  /// The compression block size in bytes — 64 for SHA-1 and SHA-256, 128
  /// for SHA-512. HMAC is defined in terms of it.
  static fun blockSize(): i64

  /// The digest length in bytes.
  static fun digestSize(): i64

  /// Adds bytes to the message.
  fun update(data: List<u8>)

  /// The digest of everything added so far.
  fun finish(): Digest
}

/// The digest of `data` under `H`, in one call:
/// `crypto.digest<Sha256>(bytes)`. `sha256(data)` is the same thing with
/// the algorithm in the name.
public fun digest<H: Hasher>(data: List<u8>): Digest {
  var h = H.start()
  h.update(data)
  h.finish()
}

/// SHA-256 of `data` — the default choice when something needs hashing.
public fun sha256(data: List<u8>): Digest = digest<Sha256>(data)

/// SHA-512 of `data`. Faster than SHA-256 on 64-bit machines, and twice
/// as long.
public fun sha512(data: List<u8>): Digest = digest<Sha512>(data)

/// SHA-384 of `data` — SHA-512 cut to 48 bytes, with its own starting
/// state. Asked for by `ES384`/`HS384` and by some government profiles.
public fun sha384(data: List<u8>): Digest = digest<Sha384>(data)

/// SHA-1 of `data`. **Broken for signatures** — collisions are practical
/// since 2017. It is here for the protocols that specify it anyway (the
/// WebSocket handshake, Git object names); never pick it for anything new.
public fun sha1Legacy(data: List<u8>): Digest = digest<Sha1>(data)

// ---------------------------------------------------------------------------
// comparison

/// True when the two byte strings are equal, in time that depends only on
/// their length. Use it wherever a mismatch is a *secret*: a MAC, a
/// session token, a password hash, an API key.
///
/// Lengths are compared first, and a difference in length answers at once
/// — the length of a MAC is public knowledge, its contents are not.
public fun equalBytes(a: List<u8>, b: List<u8>): bool {
  if (a.len() != b.len()) return false
  var diff: u8 = 0
  var i = 0
  val n = a.len()
  loop (i < n) {
    diff = diff | (a.at(i) ^ (b.at(i) ?: panic("equalBytes: the lengths were compared first")))
    i = i + 1
  }
  diff == 0
}

// ---------------------------------------------------------------------------
// bit helpers, shared by the digest implementations

/// `x` rotated right by `n` bits (`n` in 1..31).
fun rotr32(x: u32, n: i64): u32 = (x >> n) | (x << (32 - n))

/// `x` rotated left by `n` bits (`n` in 1..31).
fun rotl32(x: u32, n: i64): u32 = (x << n) | (x >> (32 - n))

/// `x` rotated right by `n` bits (`n` in 1..63).
fun rotr64(x: u64, n: i64): u64 = (x >> n) | (x << (64 - n))

/// Byte `i` of a block. The digests read a word only where the block holds
/// all of its bytes: whole 64- or 128-byte blocks, at offsets inside them.
fun blockByte(data: List<u8>, i: i64): u8 = data.at(i) ?: panic("crypto: a word is read only where the block holds all of its bytes")

/// The big-endian 32-bit word at `at`.
fun beU32(data: List<u8>, at: i64): u32 =
  ((blockByte(data, at) as u32) << 24) |
  ((blockByte(data, at + 1) as u32) << 16) |
  ((blockByte(data, at + 2) as u32) << 8) |
  (blockByte(data, at + 3) as u32)

/// The big-endian 64-bit word at `at`.
fun beU64(data: List<u8>, at: i64): u64 {
  var v: u64 = 0
  var i = 0
  loop (i < 8) {
    v = (v << 8) | (blockByte(data, at + i) as u64)
    i = i + 1
  }
  v
}

/// Appends `w` as four big-endian bytes.
fun pushU32(out: MutableList<u8>, w: u32) {
  out.push((w >> 24) as u8)
  out.push((w >> 16) as u8)
  out.push((w >> 8) as u8)
  out.push(w as u8)
}

/// Appends `w` as eight big-endian bytes.
fun pushU64(out: MutableList<u8>, w: u64) {
  var s = 56
  loop (s >= 0) {
    out.push((w >> s) as u8)
    s = s - 8
  }
}

/// The Merkle–Damgård tail every SHA function shares: the byte `0x80`,
/// zeros up to `lengthBytes` short of a block boundary, then the message
/// length in bits, big-endian, in `lengthBytes` bytes.
///
/// `buffered` is how many bytes of the current block are already held.
fun padding(buffered: i64, blockSize: i64, lengthBytes: i64, totalBytes: i64): List<u8> {
  val out: MutableList<u8> = []
  out.push(128)
  var len = buffered + 1
  val mark = blockSize - lengthBytes
  loop (len % blockSize != mark) {
    out.push(0)
    len = len + 1
  }
  // the high bytes of the length: a message long enough to use them
  // cannot be held in memory, so they are zero
  var i = lengthBytes - 8
  loop (i > 0) {
    out.push(0)
    i = i - 1
  }
  pushU64(out, (totalBytes * 8) as u64)
  out.toList()  // a copy: `out` was handed to pushU64 (D63)
}
