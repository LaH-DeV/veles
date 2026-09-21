// Prelude — the traits behind the operators. Every type gets structural
// behaviour for free: `==` compares field by field, map keys hash the same
// way, and interpolation prints `Name(field: value, ...)`. A struct or
// sealed trait may replace any of these by implementing the trait; the
// built-in types (numbers, strings, collections) keep theirs, and the
// numbers and strings implement Comparable so generic code can order them.

/// The result of a comparison: `a.compareTo(b)` is `Less` when `a` sorts before
/// `b`, `Equal` when neither sorts first, `Greater` otherwise. The values
/// are -1, 0 and 1, so `a.compareTo(b) < 0` reads as it always has.
public enum Ordering {
  Less = -1
  Equal
  Greater
}

/// Ordering for `<`, `<=`, `>`, `>=`, `sorted()`, `min()` and `max()`.
/// `compareTo` returns `Ordering.Less` when `self` sorts before `other`,
/// `Ordering.Equal` when they are equal, and `Ordering.Greater` otherwise.
public trait Comparable {
  fun compareTo(other: Self): Ordering
}

/// Custom equality for `==` and `!=`. A type that implements Equatable and
/// is used as a map key or set element must implement Hashable too, so that
/// equal values hash alike.
public trait Equatable {
  fun equals(other: Self): bool
}

/// Custom hashing for map keys and set elements. Equal values (by `==`)
/// must return the same hash.
public trait Hashable {
  fun hash(): i64
}

/// Custom text for interpolation: `"$x"` calls `x.toString()` when the type
/// implements Display, and prints the fields otherwise.
public trait Display {
  fun toString(): string
}

impl Comparable for i8 {
  fun compareTo(other: i8): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for i16 {
  fun compareTo(other: i16): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for i32 {
  fun compareTo(other: i32): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for i64 {
  fun compareTo(other: i64): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for isize {
  fun compareTo(other: isize): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for u8 {
  fun compareTo(other: u8): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for u16 {
  fun compareTo(other: u16): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for u32 {
  fun compareTo(other: u32): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for u64 {
  fun compareTo(other: u64): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for usize {
  fun compareTo(other: usize): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for f32 {
  fun compareTo(other: f32): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for f64 {
  fun compareTo(other: f64): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

impl Comparable for string {
  fun compareTo(other: string): Ordering =
    if (self < other) Ordering.Less else if (self > other) Ordering.Greater else Ordering.Equal
}

/// Construction from text, the inverse of Display: `i64.parse("42")`, or
/// `T.parse(s)` in generic code. `null` when the text is not a value of
/// the type.
public trait Parsable {
  static fun parse(s: string): Self?
}

impl Parsable for i64 {
  static fun parse(s: string): i64? = s.toInt()
}

impl Parsable for f64 {
  static fun parse(s: string): f64? = s.toF64()
}

impl Parsable for bool {
  static fun parse(s: string): bool? = when (s) {
    "true"  => true
    "false" => false
    else    => null
  }
}

impl Parsable for string {
  static fun parse(s: string): string? = s
}
