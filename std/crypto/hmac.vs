// HMAC (RFC 2104): a keyed digest. `H(opad || H(ipad || message))`, where
// the two pads are the key stretched to one block and xored with 0x36 and
// 0x5c. A plain hash of `key || message` is not a MAC — HMAC exists so
// that length-extension cannot forge one.

/// A keyed digest, fed in pieces: `var m = crypto.Hmac<Sha256>.start(key)`,
/// `m.update(…)`, `m.finish()`. `crypto.hmacSha256(key, message)` is the
/// one-shot form.
///
/// The key may be any length. A key longer than the hash's block size is
/// hashed first (RFC 2104), a shorter one is padded with zeros, so a key
/// of 32 random bytes is the sensible choice for SHA-256.
public struct Hmac<H: Hasher> {
  private inner:      H
  private outerPad:   List<u8>
  private var result: Digest? = null

  /// A MAC under `key`, ready for `update`.
  public static fun start(key: List<u8>): Hmac<H> {
    val block = H.blockSize()
    // RFC 2104: a key longer than one block is replaced by its own digest
    val k = if (key.len() > block) digest<H>(key).bytes() else key
    val ipad: MutableList<u8> = []
    val opad: MutableList<u8> = []
    var i = 0
    loop (i < block) {
      val b: u8 = if (i < k.len()) k.at(i) else 0
      ipad.push(b ^ 0x36)
      opad.push(b ^ 0x5c)
      i = i + 1
    }
    var h = H.start()
    h.update(ipad)
    Hmac(inner: h, outerPad: opad.toList())
  }

  /// The name for a wire format: `"HMAC-SHA-256"`.
  public static fun algorithm(): string => "HMAC-" + H.algorithm()

  /// The MAC length in bytes, the same as the hash's.
  public static fun digestSize(): i64 => H.digestSize()

  /// Adds bytes to the message.
  public fun update(data: List<u8>) {
    if (this.result != null) panic("crypto.Hmac: update after finish")
    this.inner.update(data)
  }

  /// The MAC of everything added so far. Compare it with `==`, never by
  /// its bytes — see `equalBytes`.
  public fun finish(): Digest {
    val done = this.result
    if (done != null) return done
    val innerDigest = this.inner.finish()
    var outer = H.start()
    outer.update(this.outerPad)
    outer.update(innerDigest.bytes())
    val d = outer.finish()
    this.result = d
    d
  }
}

/// The MAC of `message` under `key` and `H`, in one call:
/// `crypto.hmac<Sha256>(key, message)`. The key is a `Secret` (D112), so it
/// never reaches a log line; `Hmac.start` takes the raw bytes when a program
/// manages them itself.
public fun hmac<H: Hasher>(key: Secret<List<u8>>, message: List<u8>): Digest {
  var m = Hmac<H>.start(key.expose())
  m.update(message)
  m.finish()
}

/// HMAC-SHA-256 — the default MAC: signed cookies, webhook signatures,
/// JWT `HS256`.
public fun hmacSha256(key: Secret<List<u8>>, message: List<u8>): Digest => hmac<Sha256>(key, message)

/// HMAC-SHA-512, for `HS512` and for keys longer than 256 bits.
public fun hmacSha512(key: Secret<List<u8>>, message: List<u8>): Digest => hmac<Sha512>(key, message)

/// HMAC-SHA-1. Unlike plain SHA-1 this is not broken — collisions do not
/// help an attacker without the key — but nothing new should ask for it.
public fun hmacSha1Legacy(key: Secret<List<u8>>, message: List<u8>): Digest => hmac<Sha1>(key, message)

/// HMAC-SHA-384, for `HS384`.
public fun hmacSha384(key: Secret<List<u8>>, message: List<u8>): Digest => hmac<Sha384>(key, message)
