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

  /// The IEEE 754 binary64 bit pattern of the number, as an integer: sign in
  /// the top bit, then 11 exponent bits, then 52 of fraction. Nothing is
  /// rounded or canonicalised — a NaN keeps its payload, and `-0.0` differs
  /// from `0.0`. `1.5.toBits()` is `0x3FF8000000000000`.
  public fun toBits(): u64 {
    var v = this
    // SAFETY: reads the eight bytes of a local f64 as a u64, the same size
    return unsafe {
      *(&v).cast<*raw u64>()
    }
  }

  /// The number whose bit pattern is `bits`; the inverse of `toBits()`. Every
  /// pattern is a number (some are NaNs), so it cannot fail.
  public static fun fromBits(bits: u64): f64 {
    var b = bits
    // SAFETY: reads the eight bytes of a local u64 as an f64, the same size
    return unsafe {
      *(&b).cast<*raw f64>()
    }
  }
}

extend f32 {
  /// The number with exactly `digits` decimals, rounded.
  public fun toFixed(digits: i64): string = (this.toF64()).toFixed(digits)

  /// The IEEE 754 binary32 bit pattern of the number, as an integer: sign,
  /// 8 exponent bits, 23 of fraction. Nothing is rounded or canonicalised.
  /// For 1.5 it is `0x3FC00000`.
  public fun toBits(): u32 {
    var v = this
    // SAFETY: reads the four bytes of a local f32 as a u32, the same size
    return unsafe {
      *(&v).cast<*raw u32>()
    }
  }

  /// The number whose bit pattern is `bits`; the inverse of `toBits()`.
  public static fun fromBits(bits: u32): f32 {
    var b = bits
    // SAFETY: reads the four bytes of a local u32 as an f32, the same size
    return unsafe {
      *(&b).cast<*raw f32>()
    }
  }
}
