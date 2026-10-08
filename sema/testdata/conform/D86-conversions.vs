// D86: numeric conversions are methods — `toT()` (total where nothing can be
// lost, a `T?` where something can) and `wrapT()` (integers, keeps the low
// bits); `p.cast<*raw U>()` reinterprets a raw pointer. `as` only renames.
use io

fun widen(b: u8): i64 => b.toI64()
fun widenTotal(x: i32): f64 => x.toF64()
fun narrowChecked(n: i64): u8? => n.toU8()
fun narrowWrapped(n: i64): u8 => n.wrapU8()
fun floatToInt(f: f64): i64? => f.toI64()
fun floatNarrow(f: f64): f32 => f.toF32()
fun wrongType(n: i64): u8 => n.toU8() // error: type mismatch: expected 'u8', found 'u8?'
fun literalFits(): u8? => 200.toU8()
fun literalTooBig(): u8? => 300.toU8() // error: the literal does not fit 'u8' (D86); use '.wrapU8()'
fun literalWrap(): u8 => 300.wrapU8()
fun wrapAFloat(f: f64): i64 => f.wrapI64() // error: 'wrapI64' keeps the low bits of an integer
fun wrapIntoFloat(n: i64): f64 => n.wrapF64() // error: 'wrapF64' keeps the low bits of an integer
fun conversionTakesNothing(n: i64): u8? => n.toU8(1) // error: 'toU8' takes 0 arguments
fun oldNumeric(n: i64): i32 => n as i32 // error: 'as' no longer converts (D86; it only renames): write '.wrapI32()'
fun oldWidening(b: u8): i64 => b as i64 // error: 'as' no longer converts (D86; it only renames): write '.toI64()'
fun oldFloat(f: f64): i64 => f as i64 // error: 'as' no longer converts (D86; it only renames): write '.toI64()' (a 'i64?': null when it does not fit)
fun oldIdentity(n: i64): i64 => n as i64 // error: 'as' no longer converts (D86; it only renames): drop the cast
fun callAType(n: i64): i64 => i64(n) // error: 'i64' is not callable; convert with a method: 'x.toI64()'
fun freeze(xs: MutableList<i64>): List<i64> => xs as List<i64> // error: 'as' no longer converts (D86; it only renames): numeric conversions are methods
fun addressOutsideUnsafe(x: i64): *raw u8 => (&x).cast<*raw u8>() // error: handing out a raw address requires an 'unsafe' block (D44)
fun wrapInto<T>(n: i64): T => n.wrapTo<T>()
fun wrapIntoByte(n: i64): u8 => n.wrapTo<u8>()
fun wrapIntoText(n: i64): string => n.wrapTo<string>() // error: 'wrapTo' keeps the low bits of an integer
fun wrapToNoType(n: i64): u8 => n.wrapTo() // error: 'wrapTo' takes the integer type to wrap into
fun wrapFloatTo(f: f64): i64 => f.wrapTo<i64>() // error: 'wrapTo' keeps the low bits of an integer
unsafe fun castRaw(p: *raw u8): *raw i64 => p.cast<*raw i64>()
fun castOutsideUnsafe(p: *raw u8): *raw i64 => p.cast<*raw i64>() // error: casting a raw pointer requires an 'unsafe' block (D44)
unsafe fun castNoType(p: *raw u8): *raw i64 => p.cast() // error: 'cast' takes the raw pointer type to reinterpret as
unsafe fun oldRawCast(p: *raw u8): *raw i64 => p as *raw i64 // error: 'as' no longer converts (D86; it only renames): write '.cast<*raw i64>()'

fun main() {
  io.println("${widen(200)} ${narrowChecked(300)} ${narrowWrapped(300)}")
}
