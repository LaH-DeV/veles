// The operating system's cryptographically secure generator. `std/random`
// is xoshiro256** seeded from the clock: fast, reproducible, and useless
// against an adversary who watches the output. Anything an attacker must
// not guess — a session id, a password-reset token, a nonce, an HMAC key,
// a UUID — comes from here.
//
// Implemented on top of runtime/c/veles_os.c; every extern call is
// confined to one `unsafe` block (D44).
use os

extern "C" {
  fun veles_random_bytes(n: i64, out: *raw string): i64
}

/// `n` unpredictable bytes from the operating system: `BCryptGenRandom` on
/// Windows, `getrandom(2)` on Linux, `arc4random_buf` on macOS and the
/// BSDs. There is no seed and no state to manage.
///
/// ```veles
/// val token = base64.encodeUrl(crypto.randomBytes(32))
/// val key = crypto.randomBytes(32)   // an HMAC-SHA-256 key
/// ```
///
/// 16 bytes (128 bits) is the floor for anything that must not be guessed;
/// 32 is the usual answer and costs nothing.
///
/// **This panics instead of throwing.** A kernel that will not produce
/// randomness cannot be worked around — there is no weaker source worth
/// falling back to, and a caller has nothing to decide — so the failure is
/// a panic (which a server still catches at the request boundary, D56)
/// rather than an error every call site must thread through. It has never
/// been observed on a booted system.
public fun randomBytes(n: i64): List<u8> {
  if (n < 0) panic("crypto.randomBytes: $n bytes")
  if (n == 0) return []
  var buf = ""
  // SAFETY: the runtime fills a fresh string of exactly `n` bytes and stores it
  // in `buf`, a local that outlives the call; it keeps no pointer
  val code = unsafe {
    veles_random_bytes(n, &buf)
  }
  if (code != 0) panic("crypto.randomBytes: the system has no randomness (${os.ioError(code, "random")})")
  buf.bytes()
}

/// 64 unpredictable bits. For a random *number* in a range use
/// `std/random`; this is for keys, nonces and identifiers.
public fun randomU64(): u64 {
  var v: u64 = 0
  loop (b in randomBytes(8)) {
    v = (v << 8) | (b as u64)
  }
  v
}
