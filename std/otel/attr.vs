// Attributes: the named values spans, metrics and log records carry.

/// The value of an attribute: the types OpenTelemetry defines. Make attributes
/// with `attr`, which converts the common Veles types.
public sealed trait AnyValue
public struct StringValue : AnyValue {
  public value: string
}
public struct BoolValue : AnyValue {
  public value: bool
}
public struct IntValue : AnyValue {
  public value: i64
}
public struct DoubleValue : AnyValue {
  public value: f64
}
public struct StringsValue : AnyValue {
  public values: List<string>
}

/// What can be an attribute's value: text, numbers, booleans and lists of
/// text.
public trait ToAnyValue {
  fun anyValue(): AnyValue
}

implement ToAnyValue for string {
  fun anyValue(): AnyValue => StringValue(value: this)
}

implement ToAnyValue for bool {
  fun anyValue(): AnyValue => BoolValue(value: this)
}

implement ToAnyValue for i64 {
  fun anyValue(): AnyValue => IntValue(value: this)
}

implement ToAnyValue for i32 {
  fun anyValue(): AnyValue => IntValue(value: this.toI64())
}

implement ToAnyValue for f64 {
  fun anyValue(): AnyValue => DoubleValue(value: this)
}

implement ToAnyValue for List<string> {
  fun anyValue(): AnyValue => StringsValue(values: this)
}

/// A named value attached to a span, a metric data point or a log record.
public struct Attr {
  public key:   string
  public value: AnyValue
}

/// An attribute: `otel.attr("note.id", id)`. Names follow the OpenTelemetry
/// semantic conventions where one exists (`http.request.method`,
/// `db.system`); a value is text, a number, a boolean or a list of text.
public fun attr<T: ToAnyValue>(key: string, value: T): Attr => Attr(key, value: value.anyValue())

// the attribute set as one string, the key a metric series is found by:
// order-independent, so the same set in any order is one series
fun seriesKey(attrs: List<Attr>): string {
  if (attrs.isEmpty()) return ""
  val parts = attrs.map(a => a.key + "=" + valueText(a.value)).sorted()
  parts.join("\u{1f}")
}

fun valueText(v: AnyValue): string => when (v) {
  is StringValue  => v.value
  is BoolValue    => "${v.value}"
  is IntValue     => "${v.value}"
  is DoubleValue  => "${v.value}"
  is StringsValue => v.values.join(",")
}
