// Prelude — number formatting (the methods behind `x.toFixed(2)` and
// `n.toString(radix: 16)`). Interpolation stays the way to build strings;
// these shape one number at a time.

extern "C" {
  fun veles_f64_to_fixed(v: f64, digits: i64, out: *raw string)
}

const DIGITS: string = "0123456789abcdefghijklmnopqrstuvwxyz"

fun unsignedToRadix(v: u64, radix: i64): string {
  if (v == 0) return "0"
  var n = v
  var out = ""
  val base = radix as u64
  loop (n > 0) {
    val d = (n % base) as i64
    out = (DIGITS.substring(d, d + 1) ?: "?") + out
    n = n / base
  }
  out
}

fun checkRadix(radix: i64) {
  if (radix < 2 || radix > 36) panic("toString: radix must be between 2 and 36, got $radix")
}

extend i64 {
  /// The number written in `radix` (2 to 36; lower-case digits): `255.toString(radix: 16)` is `"ff"`.
  pub fun toString(radix: i64 = 10): string {
    checkRadix(radix)
    if (self < 0) "-" + unsignedToRadix((0 -% self) as u64, radix) else unsignedToRadix(self as u64, radix)
  }
}

extend u64 {
  /// The number written in `radix` (2 to 36; lower-case digits).
  pub fun toString(radix: i64 = 10): string {
    checkRadix(radix)
    unsignedToRadix(self, radix)
  }
}

extend f64 {
  /// The number with exactly `digits` decimals, rounded: `(2.0 / 3.0).toFixed(2)` is `"0.67"`.
  pub fun toFixed(digits: i64): string {
    var out = ""
    unsafe {
      veles_f64_to_fixed(self, digits, &out)
    }
    out
  }
}

extend f32 {
  /// The number with exactly `digits` decimals, rounded.
  pub fun toFixed(digits: i64): string = (self as f64).toFixed(digits)
}
