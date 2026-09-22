/// Pseudo-random numbers: a xoshiro256** generator seeded from the clock
/// (call `seed(n)` for a reproducible sequence). Fast, well distributed,
/// and for shuffling a list or picking a sample — never for a secret.
///
/// **Not for cryptography.** A handful of outputs is enough to recover the
/// state and predict every value before and after, so a session token, an
/// API key, a password-reset link or a nonce built from here is guessable.
/// `crypto.randomBytes(n)` is the operating system's generator and is what
/// those want; `crypto.uuidV4()` and `crypto.uuidV7()` use it.
use time

// splitmix64 constants; u64 literals need a typed binding to be read as u64
const GOLDEN: u64 = 11400714819323198485
const MIX1: u64 = 13787848793156543929
const MIX2: u64 = 10723151780598845931

public struct Rng {
  var s0: u64
  var s1: u64
  var s2: u64
  var s3: u64

  /// A generator with its own state, seeded from `n`.
  public static fun seeded(n: i64): Rng {
    // splitmix64 expands one seed into four words
    var x = n as u64
    var s: MutableList<u64> = []
    loop (_ in 0..<4) {
      x = x +% GOLDEN
      var z = x
      z = (z ^ (z >> 30)) *% MIX1
      z = (z ^ (z >> 27)) *% MIX2
      s.push(z ^ (z >> 31))
    }
    Rng(s0: s.atOrPanic(0), s1: s.atOrPanic(1), s2: s.atOrPanic(2), s3: s.atOrPanic(3))
  }

  /// The next 64 random bits.
  public fun nextU64(): u64 {
    val result = rotl(self.s1 *% 5, 7) *% 9
    val t = self.s1 << 17
    self.s2 = self.s2 ^ self.s0
    self.s3 = self.s3 ^ self.s1
    self.s1 = self.s1 ^ self.s2
    self.s0 = self.s0 ^ self.s3
    self.s2 = self.s2 ^ t
    self.s3 = rotl(self.s3, 45)
    result
  }

  /// A number in `lo..<hi`; `hi` must exceed `lo`.
  public fun range(lo: i64, hi: i64): i64 {
    if (hi <= lo) panic("random.range: empty range $lo..<$hi")
    val span = (hi - lo) as u64
    lo + (self.nextU64() % span) as i64
  }

  /// A number in `0.0..<1.0`.
  public fun float(): f64 = ((self.nextU64() >> 11) as f64) / 9007199254740992.0

  public fun boolean(): bool = (self.nextU64() & 1) == 1

  /// One element of `xs`, or `null` when it is empty.
  public fun pick<T>(xs: List<T>): T? = if (xs.isEmpty()) null else xs.atOrPanic(self.range(0, xs.len()))

  /// Reorders `xs` in place (Fisher–Yates).
  public fun shuffle<T>(xs: MutableList<T>) {
    var i = xs.len() - 1
    loop (i > 0) {
      val j = self.range(0, i + 1)
      val tmp = xs.atOrPanic(i)
      xs.set(i, xs.atOrPanic(j))
      xs.set(j, tmp)
      i -= 1
    }
  }
}

fun rotl(x: u64, k: i64): u64 = (x << k) | (x >> (64 - k))

var shared = Rng.seeded(time.now())

/// Reseeds the module's shared generator, for a reproducible run.
public fun seed(n: i64) {
  shared = Rng.seeded(n)
}

/// A number in `lo..<hi` from the shared generator.
public fun range(lo: i64, hi: i64): i64 = shared.range(lo, hi)

/// The next 64 random bits from the shared generator.
public fun nextU64(): u64 = shared.nextU64()

/// A number in `0.0..<1.0` from the shared generator.
public fun float(): f64 = shared.float()

public fun boolean(): bool = shared.boolean()

/// One element of `xs`, or `null` when it is empty.
public fun pick<T>(xs: List<T>): T? = shared.pick(xs)

/// Reorders `xs` in place.
public fun shuffle<T>(xs: MutableList<T>) {
  shared.shuffle(xs)
}
