// Number theory and classic recursions.

/// Euclid's algorithm; tuple assignment steps both values at once.
public fun gcd(a: i64, b: i64): i64 {
  var x = a.abs()
  var y = b.abs()
  loop (y != 0) (x, y) = (y, x % y)
  x
}

public fun lcm(a: i64, b: i64): i64 => if (a == 0 || b == 0) 0 else (a / gcd(a, b) * b).abs()

/// Trial division by the odd numbers up to sqrt(n).
public fun isPrime(n: i64): bool {
  if (n < 2) return false
  if (n < 4) return true
  if (n % 2 == 0) return false
  loop (divisor in (3..isqrt(n)).step(2)) {
    if (n % divisor == 0) return false
  }
  true
}

/// Sieve of Eratosthenes: all primes up to and including limit.
public fun sieve(limit: i64): List<i64> {
  if (limit < 2) return []
  val composite = MutableList<bool>.repeat(false, limit + 1)
  loop (p in 2..isqrt(limit)) {
    if (composite.at(p) ?: panic("sieve: p <= isqrt(limit) <= limit")) continue
    loop (multiple in (p * p..limit).step(p)) composite.set(multiple, true)
  }
  (2..limit).iter().filter(n => !(composite.at(n) ?: panic("sieve: n <= limit"))).toList()
}

/// Iterative Fibonacci: O(n), no recursion, no memo table needed.
public fun fib(n: i64): i64 {
  var a = 0
  var b = 1
  loop (_ in 0..<n) (a, b) = (b, a + b)
  a
}

/// Memoised recursive Fibonacci — the shape to use when the recursion is
/// not a simple loop; the table is threaded through explicitly.
public fun fibMemo(n: i64, memo: MutableMap<i64, i64>): i64 {
  if (n < 2) return n
  val known = memo.get(n)
  if (known != null) return known
  val v = fibMemo(n - 1, memo) + fibMemo(n - 2, memo)
  memo.set(n, v)
  v
}

/// Fast exponentiation by squaring: O(log e).
public fun power(base: i64, exp: i64): i64 {
  var result = 1
  var factor = base
  var remaining = exp
  loop (remaining > 0) {
    if (remaining % 2 == 1) result *= factor
    factor *= factor
    remaining /= 2
  }
  result
}

/// Modular exponentiation, the same loop with a reduction each step.
public fun powMod(base: i64, exp: i64, mod: i64): i64 {
  var result = 1 % mod
  var factor = base % mod
  var remaining = exp
  loop (remaining > 0) {
    if (remaining % 2 == 1) result = result * factor % mod
    factor = factor * factor % mod
    remaining /= 2
  }
  result
}

public fun factorial(n: i64): i64 => if (n <= 1) 1 else n * factorial(n - 1)

/// Decimal digits of a non-negative number, most significant first.
public fun digits(n: i64): List<i64> {
  if (n == 0) return [0]
  val out: MutableList<i64> = []
  var rest = n
  loop (rest > 0) {
    out.push(rest % 10)
    rest /= 10
  }
  out.toList().reversed()
}

/// Integer square root: largest r with r * r <= n (Newton's method).
public fun isqrt(n: i64): i64 {
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
public fun factorize(n: i64): List<(i64, i64)> {
  val out: MutableList<(i64, i64)> = []
  var rest = n
  var p = 2
  loop (p * p <= rest) {
    var count = 0
    loop (rest % p == 0) {
      rest /= p
      count += 1
    }
    if (count > 0) out.push((p, count))
    p += 1
  }
  if (rest > 1) out.push((rest, 1))
  out.toList()
}
