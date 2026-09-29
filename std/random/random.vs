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
    var x = n.wrapU64()
    var s: MutableList<u64> = []
    loop (_ in 0..<4) {
      x = x +% GOLDEN
      var z = x
      z = (z ^ (z >> 30)) *% MIX1
      z = (z ^ (z >> 27)) *% MIX2
      s.push(z ^ (z >> 31))
    }
    val [s0, s1, s2, s3] = s else panic("random: the loop above pushed four words")
    Rng(s0, s1, s2, s3)
  }

  /// The next 64 random bits.
  public fun nextU64(): u64 {
    val result = rotl(this.s1 *% 5, 7) *% 9
    val t = this.s1 << 17
    this.s2 = this.s2 ^ this.s0
    this.s3 = this.s3 ^ this.s1
    this.s1 = this.s1 ^ this.s2
    this.s0 = this.s0 ^ this.s3
    this.s2 = this.s2 ^ t
    this.s3 = rotl(this.s3, 45)
    result
  }

  /// A number in `lo..<hi`; `hi` must exceed `lo`.
  public fun range(lo: i64, hi: i64): i64 {
    if (hi <= lo) panic("random.range: empty range $lo..<$hi")
    val span = (hi - lo).wrapU64()
    lo + (this.nextU64() % span).wrapI64()
  }

  /// A number in `0.0..<1.0`.
  public fun float(): f64 = ((this.nextU64() >> 11).toF64()) / 9007199254740992.0

  public fun boolean(): bool = (this.nextU64() & 1) == 1

  /// One element of `xs`, or `null` when it is empty.
  public fun pick<T>(xs: List<T>): T? = if (xs.isEmpty()) null else xs.at(this.range(0, xs.len()))

  /// Reorders `xs` in place (Fisher–Yates).
  public fun shuffle<T>(xs: MutableList<T>) {
    var i = xs.len() - 1
    loop (i > 0) {
      val j = this.range(0, i + 1)
      xs.swap(i, j)
      i -= 1
    }
  }
}

fun rotl(x: u64, k: i64): u64 = (x << k) | (x >> (64 - k))

// The module's generator, shared by every task — on whichever thread each
// runs (D66) — so each call takes its lock. A task drawing many numbers
// in a hot loop is faster with an `Rng` of its own.
val shared = Mutex(value: Rng.seeded(time.now().toMicros()))

/// Reseeds the module's shared generator, for a reproducible run.
public fun seed(n: i64) {
  shared.set(Rng.seeded(n))
}

/// A number in `lo..<hi` from the shared generator.
public fun range(lo: i64, hi: i64): i64 = shared.withLock(r => r.range(lo, hi))

/// The next 64 random bits from the shared generator.
public fun nextU64(): u64 = shared.withLock(r => r.nextU64())

/// A number in `0.0..<1.0` from the shared generator.
public fun float(): f64 = shared.withLock(r => r.float())

public fun boolean(): bool = shared.withLock(r => r.boolean())

/// One element of `xs`, or `null` when it is empty.
public fun pick<T>(xs: List<T>): T? = shared.withLock(r => r.pick(xs))

/// Reorders `xs` in place.
public fun shuffle<T>(xs: MutableList<T>) {
  shared.withLock(r => r.shuffle(xs))
}
