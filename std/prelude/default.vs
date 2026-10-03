// Prelude — `Default` (D119): a value to start from, for generic code that
// has to make one. A struct opts in with an empty `implement Default`: each
// field takes its declared default, or else its type's `default()`.

/// A type's starting value: `T.default()`. Numbers are 0, `bool` is
/// `false`, `string` is `""`, collections are empty, `T?` is `null`, a
/// tuple is its elements' defaults and `Duration` is `Duration.zero`. A
/// struct gets one from an empty `implement Default` when every field has a
/// default or a type that is `Default`; a sealed trait, an enum, a `Secret`
/// or a value holding a task writes `default()` by hand.
public trait Default {
  static fun default(): Self
}

implement Default for i8 {
  static fun default(): i8 = 0
}
implement Default for i16 {
  static fun default(): i16 = 0
}
implement Default for i32 {
  static fun default(): i32 = 0
}
implement Default for i64 {
  static fun default(): i64 = 0
}
implement Default for isize {
  static fun default(): isize = 0
}
implement Default for u8 {
  static fun default(): u8 = 0
}
implement Default for u16 {
  static fun default(): u16 = 0
}
implement Default for u32 {
  static fun default(): u32 = 0
}
implement Default for u64 {
  static fun default(): u64 = 0
}
implement Default for usize {
  static fun default(): usize = 0
}
implement Default for f32 {
  static fun default(): f32 = 0.0
}
implement Default for f64 {
  static fun default(): f64 = 0.0
}
implement Default for bool {
  static fun default(): bool = false
}
implement Default for string {
  static fun default(): string = ""
}

implement<T> Default for T? {
  static fun default(): T? = null
}

implement<T> Default for List<T> {
  static fun default(): List<T> = []
}
implement<T> Default for MutableList<T> {
  static fun default(): MutableList<T> = []
}
implement<K, V> Default for Map<K, V> {
  static fun default(): Map<K, V> = [:]
}
implement<K, V> Default for MutableMap<K, V> {
  static fun default(): MutableMap<K, V> = [:]
}
implement<T> Default for Set<T> {
  static fun default(): Set<T> = Set<T>()
}
implement<T> Default for MutableSet<T> {
  static fun default(): MutableSet<T> = MutableSet<T>()
}

implement<A: Default, B: Default> Default for (A, B) {
  static fun default(): (A, B) = (A.default(), B.default())
}
implement<A: Default, B: Default, C: Default> Default for (A, B, C) {
  static fun default(): (A, B, C) = (A.default(), B.default(), C.default())
}
implement<A: Default, B: Default, C: Default, D: Default> Default for (A, B, C, D) {
  static fun default(): (A, B, C, D) = (A.default(), B.default(), C.default(), D.default())
}
implement<A: Default, B: Default, C: Default, D: Default, E: Default> Default for (A, B, C, D, E) {
  static fun default(): (A, B, C, D, E) = (A.default(), B.default(), C.default(), D.default(), E.default())
}
implement<A: Default, B: Default, C: Default, D: Default, E: Default, F: Default> Default for (A, B, C, D, E, F) {
  static fun default(): (A, B, C, D, E, F) = (A.default(), B.default(), C.default(), D.default(), E.default(), F.default())
}
implement<A: Default, B: Default, C: Default, D: Default, E: Default, F: Default, G: Default> Default for (A, B, C, D, E, F, G) {
  static fun default(): (A, B, C, D, E, F, G) = (A.default(), B.default(), C.default(), D.default(), E.default(), F.default(), G.default())
}
implement<A: Default, B: Default, C: Default, D: Default, E: Default, F: Default, G: Default, H: Default> Default for (A, B, C, D, E, F, G, H) {
  static fun default(): (A, B, C, D, E, F, G, H) = (A.default(), B.default(), C.default(), D.default(), E.default(), F.default(), G.default(), H.default())
}

// An array of `N` copies of the element's default (D121).
implement<T: Default, const N: i64> Default for Array<T, N> {
  static fun default(): Array<T, N> = Array.make(T.default())
}
