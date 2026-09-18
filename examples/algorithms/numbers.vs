// Number theory and classic recursions.

/// Euclid's algorithm.
pub fun gcd(a: i64, b: i64): i64 {
  var x = a.abs()
  var y = b.abs()
  loop (y != 0) (x, y) = (y, x % y)
  x
}

pub fun lcm(a: i64, b: i64): i64 = if (a == 0 || b == 0) 0 else (a / gcd(a, b) * b).abs()

/// Trial division up to sqrt(n).
pub fun isPrime(n: i64): bool {
  if (n < 2) return false
  if (n < 4) return true
  if (n % 2 == 0) return false
  var d = 3
  loop (d * d <= n) {
    if (n % d == 0) return false
    d += 2
  }
  true
}

/// Sieve of Eratosthenes: all primes up to and including limit.
pub fun sieve(limit: i64): List<i64> {
  if (limit < 2) return []
  var composite: MutableList<bool> = []
  loop (_ in 0..limit) composite.push(false)
  var p = 2
  loop (p * p <= limit) {
    if (!composite.atOrPanic(p)) {
      loop (m in (p * p..limit).step(p)) composite.set(m, true)
      // var m = p * p
      // loop (m <= limit) {
      //   composite.set(m, true)
      //   m += p
      // }
    }
    p += 1
  }
  var primes: MutableList<i64> = []
  loop (i in 2..limit) {
    if (!composite.atOrPanic(i)) primes.push(i)
  }
  primes.toList()
}

/// Iterative Fibonacci: O(n), no recursion, no memo table needed.
pub fun fib(n: i64): i64 {
  var a = 0
  var b = 1
  loop (_ in 0..<n) (a, b) = (b, a + b)
  a
}

/// Memoised recursive Fibonacci — the shape to use when the recursion is
/// not a simple loop; the table is threaded through explicitly.
pub fun fibMemo(n: i64, memo: MutableMap<i64, i64>): i64 {
  if (n < 2) return n
  val hit = memo.get(n)
  if (hit != null) return hit
  val v = fibMemo(n - 1, memo) + fibMemo(n - 2, memo)
  memo.set(n, v)
  v
}

/// Fast exponentiation by squaring: O(log e).
pub fun power(base: i64, exp: i64): i64 {
  var result = 1
  var b = base
  var e = exp
  loop (e > 0) {
    if (e % 2 == 1) result *= b
    b *= b
    e /= 2
  }
  result
}

/// Modular exponentiation, the same loop with a reduction each step.
pub fun powMod(base: i64, exp: i64, mod: i64): i64 {
  var result = 1 % mod
  var b = base % mod
  var e = exp
  loop (e > 0) {
    if (e % 2 == 1) result = result * b % mod
    b = b * b % mod
    e /= 2
  }
  result
}

pub fun factorial(n: i64): i64 = if (n <= 1) 1 else n * factorial(n - 1)

/// Decimal digits of a non-negative number, most significant first.
pub fun digits(n: i64): List<i64> {
  if (n == 0) return [0]
  var out: MutableList<i64> = []
  var m = n
  loop (m > 0) {
    out.push(m % 10)
    m /= 10
  }
  out.toList().reversed()
}

/// Integer square root: largest r with r * r <= n (Newton's method).
pub fun isqrt(n: i64): i64 {
  if (n < 2) return n
  var x = n
  var y = (x + 1) / 2
  loop (y < x) {
    x = y
    y = (x + n / x) / 2
  }
  x
}

/// Prime factorisation as (prime, exponent) pairs.
pub fun factorize(n: i64): List<(i64, i64)> {
  var out: MutableList<(i64, i64)> = []
  var m = n
  var p = 2
  loop (p * p <= m) {
    var count = 0
    loop (m % p == 0) {
      m /= p
      count += 1
    }
    if (count > 0) out.push((p, count))
    p += 1
  }
  if (m > 1) out.push((m, 1))
  out.toList()
}
