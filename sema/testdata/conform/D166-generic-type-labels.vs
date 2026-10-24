// D166: one instance of a generic type serves every labelling of its type
// arguments and keeps none, so labels written there are refused rather than
// dropped where `b.value.lo` would then fail. Built-in collections keep them.
struct Box<T> {
  value: T
}

fun boxed(b: Box<(lo: i64, hi: i64)>): i64 => b.value.0 // error: 'Box' keeps no tuple labels in its type arguments — one instance serves every labelling (D166); write '(i64, i64)', or make the tuple a struct

fun listed(xs: List<(lo: i64, hi: i64)>): i64 => xs.first()?.hi ?: 0

fun main() { }
