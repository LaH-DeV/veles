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
  val base = radix.wrapU64()
  loop (n > 0) {
    val d = (n % base).wrapI64()
    out = (DIGITS.substring(d, d + 1) ?: "?") + out
    n = n / base
  }
  out
}

@caller_location
fun checkRadix(radix: i64) {
  if (radix < 2 || radix > 36) panic("toString: radix must be between 2 and 36, got $radix")
}

extend i64 {
  /// The number written in `radix` (2 to 36; lower-case digits): `255.toString(radix: 16)` is `"ff"`.
  @caller_location
  public fun toString(radix: i64 = 10): string {
    checkRadix(radix)
    if (this < 0) "-" + unsignedToRadix((0 -% this).wrapU64(), radix) else unsignedToRadix(this.wrapU64(), radix)
  }
}

extend u64 {
  /// The number written in `radix` (2 to 36; lower-case digits).
  @caller_location
  public fun toString(radix: i64 = 10): string {
    checkRadix(radix)
    unsignedToRadix(this, radix)
  }
}

extend f64 {
  /// The number with exactly `digits` decimals, rounded: `(2.0 / 3.0).toFixed(2)` is `"0.67"`.
  public fun toFixed(digits: i64): string {
    var out = ""
    // SAFETY: stores the text in `out`, a local that outlives the call
    unsafe {
      veles_f64_to_fixed(this, digits, &out)
    }
    out
  }
}

extend f32 {
  /// The number with exactly `digits` decimals, rounded.
  public fun toFixed(digits: i64): string = (this.toF64()).toFixed(digits)
}
