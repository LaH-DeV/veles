/// Pseudo-random numbers: a xoshiro256** generator seeded from the clock
/// (call `seed(n)` for a reproducible sequence). Not for cryptography.
use time

// splitmix64 constants; u64 literals need a typed binding to be read as u64
const GOLDEN: u64 = 11400714819323198485
const MIX1: u64 = 13787848793156543929
const MIX2: u64 = 10723151780598845931

pub struct Rng {
  s0: u64
  s1: u64
  s2: u64
  s3: u64

  /// A generator with its own state, seeded from `n`.
  pub static fun seeded(n: i64): Rng {
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
  pub mut fun nextU64(): u64 {
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
  pub mut fun range(lo: i64, hi: i64): i64 {
    if (hi <= lo) panic("random.range: empty range $lo..<$hi")
    val span = (hi - lo) as u64
    lo + (self.nextU64() % span) as i64
  }

  /// A number in `0.0..<1.0`.
  pub mut fun float(): f64 = ((self.nextU64() >> 11) as f64) / 9007199254740992.0

  pub mut fun boolean(): bool = (self.nextU64() & 1) == 1

  /// One element of `xs`, or `null` when it is empty.
  pub mut fun pick<T>(xs: List<T>): T? = if (xs.isEmpty()) null else xs.atOrPanic(self.range(0, xs.len()))

  /// Reorders `xs` in place (Fisher–Yates).
  pub mut fun shuffle<T>(xs: MutableList<T>) {
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
pub fun seed(n: i64) {
  shared = Rng.seeded(n)
}

/// A number in `lo..<hi` from the shared generator.
pub fun range(lo: i64, hi: i64): i64 = shared.range(lo, hi)

/// The next 64 random bits from the shared generator.
pub fun nextU64(): u64 = shared.nextU64()

/// A number in `0.0..<1.0` from the shared generator.
pub fun float(): f64 = shared.float()

pub fun boolean(): bool = shared.boolean()

/// One element of `xs`, or `null` when it is empty.
pub fun pick<T>(xs: List<T>): T? = shared.pick(xs)

/// Reorders `xs` in place.
pub fun shuffle<T>(xs: MutableList<T>) {
  shared.shuffle(xs)
}
