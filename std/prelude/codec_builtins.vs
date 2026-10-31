// ---------------------------------------------------------------------------
// the built-in types

implement Encodable for i8 {
  fun encode(to: Encoder) throws EncodeError => try to.writeI64(this.toI64())
}
implement Encodable for i16 {
  fun encode(to: Encoder) throws EncodeError => try to.writeI64(this.toI64())
}
implement Encodable for i32 {
  fun encode(to: Encoder) throws EncodeError => try to.writeI64(this.toI64())
}
implement Encodable for i64 {
  fun encode(to: Encoder) throws EncodeError => try to.writeI64(this)
}
implement Encodable for isize {
  fun encode(to: Encoder) throws EncodeError => try to.writeI64(this.toI64())
}
implement Encodable for u8 {
  fun encode(to: Encoder) throws EncodeError => try to.writeU64(this.toU64())
}
implement Encodable for u16 {
  fun encode(to: Encoder) throws EncodeError => try to.writeU64(this.toU64())
}
implement Encodable for u32 {
  fun encode(to: Encoder) throws EncodeError => try to.writeU64(this.toU64())
}
implement Encodable for u64 {
  fun encode(to: Encoder) throws EncodeError => try to.writeU64(this)
}
implement Encodable for usize {
  fun encode(to: Encoder) throws EncodeError => try to.writeU64(this.toU64())
}
implement Encodable for f32 {
  fun encode(to: Encoder) throws EncodeError => try to.writeF64(this.toF64())
}
implement Encodable for f64 {
  fun encode(to: Encoder) throws EncodeError => try to.writeF64(this)
}
implement Encodable for bool {
  fun encode(to: Encoder) throws EncodeError => try to.writeBool(this)
}
implement Encodable for string {
  fun encode(to: Encoder) throws EncodeError => try to.writeString(this)
}

fun intSchema(what: string): Schema => Schema.leaf(SchemaKind.Int, what)

implement Decodable for i8 {
  static fun decode(from: Decoder): i8 throws DecodeError => try narrowI64(from, -128, 127).wrapI8()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer from -128 to 127")
}
implement Decodable for i16 {
  static fun decode(from: Decoder): i16 throws DecodeError => try narrowI64(from, -32768, 32767).wrapI16()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer from -32768 to 32767")
}
implement Decodable for i32 {
  static fun decode(from: Decoder): i32 throws DecodeError => try narrowI64(from, -2147483648, 2147483647).wrapI32()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer from -2147483648 to 2147483647")
}
implement Decodable for i64 {
  static fun decode(from: Decoder): i64 throws DecodeError => try from.readI64()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer")
}
implement Decodable for isize {
  static fun decode(from: Decoder): isize throws DecodeError => try from.readI64().toIsize()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer")
}
implement Decodable for u8 {
  static fun decode(from: Decoder): u8 throws DecodeError => try narrowU64(from, 255).wrapU8()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer from 0 to 255")
}
implement Decodable for u16 {
  static fun decode(from: Decoder): u16 throws DecodeError => try narrowU64(from, 65535).wrapU16()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer from 0 to 65535")
}
implement Decodable for u32 {
  static fun decode(from: Decoder): u32 throws DecodeError => try narrowU64(from, 4294967295).wrapU32()
  override static fun schema(format: string, keys: Case): Schema => intSchema("an integer from 0 to 4294967295")
}
implement Decodable for u64 {
  static fun decode(from: Decoder): u64 throws DecodeError => try from.readU64()
  override static fun schema(format: string, keys: Case): Schema => intSchema("a non-negative integer")
}
implement Decodable for usize {
  static fun decode(from: Decoder): usize throws DecodeError => try from.readU64().toUsize()
  override static fun schema(format: string, keys: Case): Schema => intSchema("a non-negative integer")
}
implement Decodable for f32 {
  static fun decode(from: Decoder): f32 throws DecodeError => try from.readF64().toF32()
  override static fun schema(format: string, keys: Case): Schema => Schema.leaf(SchemaKind.Float, "a number")
}
implement Decodable for f64 {
  static fun decode(from: Decoder): f64 throws DecodeError => try from.readF64()
  override static fun schema(format: string, keys: Case): Schema => Schema.leaf(SchemaKind.Float, "a number")
}
implement Decodable for bool {
  static fun decode(from: Decoder): bool throws DecodeError => try from.readBool()
  override static fun schema(format: string, keys: Case): Schema => Schema.leaf(SchemaKind.Bool, "true or false")
}
implement Decodable for string {
  static fun decode(from: Decoder): string throws DecodeError => try from.readString()
  override static fun schema(format: string, keys: Case): Schema => Schema.leaf(SchemaKind.Text, "text")
}

/// An integer that must fit a narrower type: out of range is a problem.
fun narrowI64(from: Decoder, min: i64, max: i64): i64 throws DecodeError {
  val v = try from.readI64()
  if (v < min || v > max) {
    from.problem("$v is out of range ($min to $max)")
    return 0
  }
  v
}

fun narrowU64(from: Decoder, max: u64): u64 throws DecodeError {
  val v = try from.readU64()
  if (v > max) {
    from.problem("$v is out of range (0 to $max)")
    return 0
  }
  v
}

implement<T: Encodable> Encodable for T? {
  fun encode(to: Encoder) throws EncodeError {
    if (this == null) try to.writeNull() else try this.encode(to)
  }
}
implement<T: Decodable> Decodable for T? {
  static fun decode(from: Decoder): T? throws DecodeError {
    if (try from.peek() == Kind.Null) {
      try from.readNull()
      return null
    }
    try T.decode(from)
  }
  override static fun schema(format: string, keys: Case): Schema => T.schema(format, keys)
}

implement<T: Encodable> Encodable for List<T> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginList()
    loop (x in this) try x.encode(to)
    try to.endList()
  }
}
implement<T: Encodable> Encodable for MutableList<T> {
  fun encode(to: Encoder) throws EncodeError => try this.toList().encode(to)
}
implement<T: Decodable> Decodable for List<T> {
  static fun decode(from: Decoder): List<T> throws DecodeError => try MutableList<T>.decode(from).toList()
  override static fun schema(format: string, keys: Case): Schema => Schema.list(T.schema(format, keys))
}
implement<T: Decodable> Decodable for MutableList<T> {
  static fun decode(from: Decoder): MutableList<T> throws DecodeError {
    val out: MutableList<T> = []
    try from.beginList()
    loop (try from.hasNext()) out.push(try T.decode(from))
    try from.endList()
    out
  }
  override static fun schema(format: string, keys: Case): Schema => Schema.list(T.schema(format, keys))
}

// An array travels as a list; reading one needs exactly N elements (D121).
implement<T: Encodable, const N: i64> Encodable for Array<T, N> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginList()
    loop (x in this) try x.encode(to)
    try to.endList()
  }
}
implement<T: Decodable, const N: i64> Decodable for Array<T, N> {
  static fun decode(from: Decoder): Array<T, N> throws DecodeError {
    val items = try List<T>.decode(from)
    val a = items.toArray<N>()
    if (a != null) return a
    from.problem("expected $N elements, found ${items.len()}")
    throw DecodeError(problems: from.problems())
  }
  override static fun schema(format: string, keys: Case): Schema => Schema.list(T.schema(format, keys))
}

implement<V: Encodable> Encodable for Map<string, V> {
  fun encode(to: Encoder) throws EncodeError {
    try to.beginObject()
    loop ((k, v) in this.entries()) {
      try to.key(k)
      try v.encode(to)
    }
    try to.endObject()
  }
}
implement<V: Encodable> Encodable for MutableMap<string, V> {
  fun encode(to: Encoder) throws EncodeError => try this.toMap().encode(to)
}
implement<V: Decodable> Decodable for Map<string, V> {
  static fun decode(from: Decoder): Map<string, V> throws DecodeError => try MutableMap<string, V>.decode(from).toMap()
  override static fun schema(format: string, keys: Case): Schema => Schema.unsupported("a Map")
}
implement<V: Decodable> Decodable for MutableMap<string, V> {
  static fun decode(from: Decoder): MutableMap<string, V> throws DecodeError {
    val out: MutableMap<string, V> = [:]
    try from.beginObject()
    loop (val k = try from.nextKey()) {
      out.set(k, try V.decode(from))
    }
    try from.endObject()
    out
  }
  override static fun schema(format: string, keys: Case): Schema => Schema.unsupported("a Map")
}
