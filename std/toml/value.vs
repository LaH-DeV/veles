/// What a value is. A sealed family, so a reader of a configuration matches
/// on it:
///
/// ```veles
/// when (value.data) {
///   is Text(text)   => use(text)
///   is Integer(n)   => use(n)
///   is Items(items) => items.each(...)
///   else            => throw wrong(value, "a string, a number or a list")
/// }
/// ```
public sealed trait Data {
  /// How a message names this kind of value: "a string", "a list", …
  public fun describe(): string => when (this) {
    is Text             => "a string"
    is Integer          => "a number"
    is Real             => "a number"
    is Flag             => "true or false"
    is Moment(shape, _) => describeShape(shape)
    is Items            => "a list"
    is Nested           => "a table"
  }

  struct Text {
    public value: string
  }
  struct Integer {
    public value: i64
  }
  struct Real {
    public value: f64
  }
  struct Flag {
    public value: bool
  }
  /// A date, a time or both, as written (`1979-05-27T07:32:00Z`), checked for
  /// its form and range.
  struct Moment {
    public shape: MomentShape
    public text:  string
  }
  /// A list. `ofTables` is set for one made by `[[name]]` headers, which the
  /// next such header extends; an inline `[ … ]` is closed.
  struct Items {
    public items:    List<Value>
    public ofTables: bool = false
  }
  struct Nested {
    public table: *Table
  }
}

/// The four shapes TOML gives a date or a time.
public enum MomentShape {
  OffsetDateTime
  LocalDateTime
  LocalDate
  LocalTime
}

/// How a message names a date or a time of this shape.
fun describeShape(shape: MomentShape): string => when (shape) {
  MomentShape.OffsetDateTime => "a date and time"
  MomentShape.LocalDateTime  => "a date and time"
  MomentShape.LocalDate      => "a date"
  MomentShape.LocalTime      => "a time"
}

/// A value and the line it was written on.
public struct Value {
  public data: Data
  public line: i64

  /// The text of a string.
  public fun asString(): string? => when (this.data) {
    is Text(text) => text
    else          => null
  }

  public fun asInt(): i64? => when (this.data) {
    is Integer(n) => n
    else          => null
  }

  /// A float, or an integer read as one.
  public fun asFloat(): f64? => when (this.data) {
    is Real(x)    => x
    is Integer(n) => n.toF64()
    else          => null
  }

  public fun asBool(): bool? => when (this.data) {
    is Flag(flag) => flag
    else          => null
  }

  public fun asList(): List<Value>? => when (this.data) {
    is Items(items) => items
    else            => null
  }

  public fun asTable(): (*Table)? => when (this.data) {
    is Nested(table) => table
    else             => null
  }

  /// The elements of a list made by `[[name]]` headers; null for any other value.
  fun asArrayOfTables(): List<Value>? => when (this.data) {
    is Items(items, ofTables) => if (ofTables) items else null
    else                      => null
  }

  /// The date or time, with its shape.
  public fun asMoment(): Moment? => when (this.data) {
    is Moment(shape, text) => Moment(shape, text)
    else                   => null
  }
}

/// Where a table came from. A `[header]` or an inline table *defines* it; one
/// only on the way to a header (`[a.b]` makes `a`), or made by a dotted key
/// (`a.b = 1`), can still be defined later.
enum Origin {
  Implied
  Dotted
  Header
  Inline
}

/// A table of keys to values.
public struct Table {
  entries:     MutableMap<string, Value> = [:]
  public line: i64
  var origin:  Origin = Origin.Implied

  /// The value under `key`.
  public fun get(key: string): Value? => this.entries.get(key)

  public fun has(key: string): bool => this.entries.containsKey(key)

  /// The keys, in a fixed (sorted) order.
  public fun keys(): List<string> => this.entries.keys().sorted()

  /// True when a `[header]` or an inline table named it, not when it is only
  /// the parent of one.
  public fun defined(): bool => this.origin in [Origin.Header, Origin.Inline]
}
