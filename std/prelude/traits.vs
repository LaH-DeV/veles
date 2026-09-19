// Prelude — the traits behind the operators. Every type gets structural
// behaviour for free: `==` compares field by field, map keys hash the same
// way, and interpolation prints `Name(field: value, ...)`. A struct or
// sealed trait may replace any of these by implementing the trait; the
// built-in types (numbers, strings, collections) keep theirs, and the
// numbers and strings implement Comparable so generic code can order them.

/// Ordering for `<`, `<=`, `>`, `>=`, `sorted()`, `min()` and `max()`.
/// `compareTo` returns a negative number when `self` sorts before `other`,
/// zero when they are equal, and a positive number otherwise.
public trait Comparable {
  fun compareTo(other: Self): i64
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
  fun compareTo(other: i8): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for i16 {
  fun compareTo(other: i16): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for i32 {
  fun compareTo(other: i32): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for i64 {
  fun compareTo(other: i64): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for isize {
  fun compareTo(other: isize): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for u8 {
  fun compareTo(other: u8): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for u16 {
  fun compareTo(other: u16): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for u32 {
  fun compareTo(other: u32): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for u64 {
  fun compareTo(other: u64): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for usize {
  fun compareTo(other: usize): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for f32 {
  fun compareTo(other: f32): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for f64 {
  fun compareTo(other: f64): i64 = if (self < other) -1 else if (self > other) 1 else 0
}

impl Comparable for string {
  fun compareTo(other: string): i64 = if (self < other) -1 else if (self > other) 1 else 0
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
