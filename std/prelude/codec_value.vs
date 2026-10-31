// ---------------------------------------------------------------------------
// Value: a document as a tree

/// A decoded document with no type in mind — what `json.parse` returns and
/// what a derived sealed-trait decoder buffers an object into before it
/// knows the variant. It is `Codable` itself, so it passes through any
/// format.
public sealed trait Value {
  struct VNull { }
  struct VBool {
    public value: bool
  }
  struct VInt {
    public value: i64
  }
  struct VFloat {
    public value: f64
  }
  struct VString {
    public value: string
  }
  struct VList {
    public items: List<Value>
  }
  struct VObject {
    public fields: Map<string, Value>
  }
}

extend Value {
  /// The member `key` of an object, or `null`.
  public fun get(key: string): Value? => when (this) {
    is VObject => this.fields.get(key)
    else       => null
  }

  /// Element `i` of a list, or `null`.
  public fun at(i: i64): Value? => when (this) {
    is VList => this.items.at(i)
    else     => null
  }

  public fun asString(): string? => when (this) {
    is VString => this.value
    else       => null
  }

  public fun asI64(): i64? => when (this) {
    is VInt   => this.value
    is VFloat => if (this.value == this.value.trunc()) this.value.toI64() else null
    else      => null
  }

  public fun asF64(): f64? => when (this) {
    is VInt   => this.value.toF64()
    is VFloat => this.value
    else      => null
  }

  public fun asBool(): bool? => when (this) {
    is VBool => this.value
    else     => null
  }

  public fun isNull(): bool => this is VNull
}

implement Encodable for Value {
  fun encode(to: Encoder) throws EncodeError => when (this) {
    is VNull   => try to.writeNull()
    is VBool   => try to.writeBool(this.value)
    is VInt    => try to.writeI64(this.value)
    is VFloat  => try to.writeF64(this.value)
    is VString => try to.writeString(this.value)
    is VList   => try this.items.encode(to)
    is VObject => try this.fields.encode(to)
  }
}

implement Decodable for Value {
  static fun decode(from: Decoder): Value throws DecodeError => when (try from.peek()) {
    Kind.Null   => {
      try from.readNull()
      VNull()
    }
    Kind.Bool   => VBool(value: try from.readBool())
    Kind.Int    => VInt(value: try from.readI64())
    Kind.Float  => VFloat(value: try from.readF64())
    Kind.String => VString(value: try from.readString())
    Kind.List   => VList(items: try List<Value>.decode(from))
    Kind.Object => VObject(fields: try Map<string, Value>.decode(from))
  }
  override static fun schema(format: string, keys: Case): Schema => Schema.unsupported("a free-form Value")
}

// ---------------------------------------------------------------------------
// decoding from a Value

/// One open container while walking a tree: an object's remaining keys or
/// a list's remaining items, and the value the next `read` consumes.
struct Frame {
  var path:    string
  var keys:    List<string> = []
  var values:  List<Value> = []
  var next:    i64 = 0
  var pending: Value? = null  // the value of the key just handed out
  isList:      bool
}

/// A `Decoder` over a `Value`: what a derived decoder uses to read a
/// variant out of an object it buffered, and what `json.decodeValue<T>`
/// uses to type a tree after the fact.
public struct ValueDecoder {
  private root:          Value
  private stack:         MutableList<Frame> = []
  private recorded:      Problems = Problems()
  private started:       bool = false
  private name:          string = "json"
  private style:         EnumStyle = EnumStyle.Name
  private keyStyle:      Case = Case.AsWritten
  private durationStyle: DurationStyle = DurationStyle.Seconds
  private maxDepth:      i64 = maxRecursionDepth

  public static fun of(v: Value, format: string = "json", enums: EnumStyle = EnumStyle.Name, keys: Case = Case.AsWritten, durations: DurationStyle = DurationStyle.Seconds, maxDepth: i64 = maxRecursionDepth): ValueDecoder =>
    ValueDecoder(root: v, name: format, style: enums, keyStyle: keys, durationStyle: durations, maxDepth)

  /// The value about to be read.
  private fun current(): Value {
    val top = this.stack.ref(-1) ?: return this.root
    if (top.isList) return top.values.at(top.next) ?: VNull()
    top.pending ?: VNull()
  }

  /// A read consumed the current value.
  private fun advance() {
    val top = this.stack.ref(-1) ?: return
    if (top.isList) top.next += 1 else top.pending = null
  }

  private fun here(): string {
    val top = this.stack.ref(-1) ?: return ""
    if (top.isList) return indexPath(top.path, top.next)
    childPath(top.path, top.keys.at(top.next - 1) ?: "")
  }

  private fun wrong(expected: string): Value {
    val v = this.current()
    this.recorded.record(this.here(), "expected $expected, found ${kindName(v)}")
    this.advance()
    v
  }

  /// A tree nests as deep as whoever built it wanted, and walking it is
  /// recursion, so it is bounded like every other walk over input someone
  /// else produced (§2). Unlike a wrong type, this is not a problem to
  /// record and carry on from: the frames are already on the stack.
  private fun checkDepth() throws DecodeError {
    if (this.stack.len() < this.maxDepth) return
    throw DecodeError(problems: this.recorded.list().concat([Problem(path: this.here(), message: tooDeepMessage(this.maxDepth))]))
  }

  implement Decoder {
    fun format(): string => this.name
    override fun enums(): EnumStyle => this.style
    override fun durations(): DurationStyle => this.durationStyle
    override fun keys(): Case => this.keyStyle

    fun peek(): Kind throws DecodeError => when (this.current()) {
      is VNull   => Kind.Null
      is VBool   => Kind.Bool
      is VInt    => Kind.Int
      is VFloat  => Kind.Float
      is VString => Kind.String
      is VList   => Kind.List
      is VObject => Kind.Object
    }

    fun beginObject() throws DecodeError {
      try this.checkDepth()
      val v = this.current()
      when (v) {
        is VObject => {
          this.stack.push(Frame(path: this.here(), keys: v.fields.keys(), values: v.fields.values(), isList: false))
        }
        else       => {
          val _ = this.wrong("an object")
          this.stack.push(Frame(path: this.here(), isList: false))
        }
      }
    }

    fun nextKey(): string? throws DecodeError {
      val top = this.stack.ref(-1) ?: panic(openFrame)
      val k = top.keys.at(top.next) ?: return null
      top.pending = top.values.at(top.next)
      top.next += 1
      k
    }

    fun endObject() throws DecodeError {
      val _ = this.stack.removeAt(this.stack.lastIndex())
      this.advance()
    }

    fun beginList() throws DecodeError {
      try this.checkDepth()
      val v = this.current()
      when (v) {
        is VList => this.stack.push(Frame(path: this.here(), values: v.items, isList: true))
        else     => {
          val _ = this.wrong("a list")
          this.stack.push(Frame(path: this.here(), isList: true))
        }
      }
    }

    fun hasNext(): bool throws DecodeError {
      val top = this.stack.ref(-1) ?: panic(openFrame)
      top.next < top.values.len()
    }

    fun endList() throws DecodeError {
      val _ = this.stack.removeAt(this.stack.lastIndex())
      this.advance()
    }

    fun readI64(): i64 throws DecodeError {
      val v = this.current()
      val n = v.asI64()
      if (n == null) {
        val _ = this.wrong("an integer")
        return 0
      }
      this.advance()
      n
    }

    fun readU64(): u64 throws DecodeError {
      val n = try this.readI64()
      if (n < 0) {
        this.recorded.record(this.here(), "$n is negative")
        return 0
      }
      n.wrapU64()
    }

    fun readF64(): f64 throws DecodeError {
      val v = this.current()
      val n = v.asF64()
      if (n == null) {
        val _ = this.wrong("a number")
        return 0.0
      }
      this.advance()
      n
    }

    fun readBool(): bool throws DecodeError {
      val v = this.current()
      val b = v.asBool()
      if (b == null) {
        val _ = this.wrong("a boolean")
        return false
      }
      this.advance()
      b
    }

    fun readString(): string throws DecodeError {
      val v = this.current()
      val s = v.asString()
      if (s == null) {
        val _ = this.wrong("a string")
        return ""
      }
      this.advance()
      s
    }

    fun readNull() throws DecodeError {
      if (!this.current().isNull()) {
        val _ = this.wrong("null")
        return
      }
      this.advance()
    }

    fun skip() throws DecodeError => this.advance()

    fun path(): string => this.here()

    fun problemAt(path: string, message: string) => this.recorded.record(path, message)

    fun problems(): List<Problem> => this.recorded.list()
  }
}

fun kindName(v: Value): string => when (v) {
  is VNull   => "null"
  is VBool   => "a boolean"
  is VInt    => "a number"
  is VFloat  => "a number"
  is VString => "a string"
  is VList   => "a list"
  is VObject => "an object"
}

// ---------------------------------------------------------------------------
// encoding into a Value

/// An open container being built.
struct Building {
  var keys:   MutableList<string> = []
  var values: MutableList<Value> = []
  var key:    string = ""
  isList:     bool
}

/// The state of a ValueEncoder, behind a pointer: an encoder is handed
/// around as a trait object, which boxes a copy of the struct (D9), so
/// what it builds must live where every copy can reach it.
struct ValueEncoderState {
  var result: Value? = null
}

/// An `Encoder` that builds a `Value`: `json.toValue(x)`, and the way a
/// value is re-encoded through another format without a document in
/// between.
public struct ValueEncoder {
  private stack:         MutableList<Building> = []
  private state:         *ValueEncoderState
  private name:          string = "json"
  private style:         EnumStyle = EnumStyle.Name
  private keyStyle:      Case = Case.AsWritten
  private durationStyle: DurationStyle = DurationStyle.Seconds
  private maxDepth:      i64 = maxRecursionDepth

  public static fun of(format: string = "json", enums: EnumStyle = EnumStyle.Name, keys: Case = Case.AsWritten, durations: DurationStyle = DurationStyle.Seconds, maxDepth: i64 = maxRecursionDepth): ValueEncoder =>
    ValueEncoder(state: &ValueEncoderState(), name: format, style: enums, keyStyle: keys, durationStyle: durations, maxDepth)

  /// The value built, once one whole value has been written.
  public fun value(): Value => this.state.result ?: VNull()

  /// The same bound the text encoders keep (§2): a value nested deeper than
  /// this is refused rather than walked, whichever direction it is going.
  private fun checkDepth() throws EncodeError {
    if (this.stack.len() >= this.maxDepth) throw EncodeError(message: tooDeepMessage(this.maxDepth))
  }

  private fun put(v: Value) {
    val top = this.stack.ref(-1)
    if (top == null) {
      this.state.result = v
      return
    }
    if (!top.isList) top.keys.push(top.key)
    top.values.push(v)
  }

  implement Encoder {
    fun format(): string => this.name
    override fun enums(): EnumStyle => this.style
    override fun durations(): DurationStyle => this.durationStyle
    override fun keys(): Case => this.keyStyle

    fun beginObject() throws EncodeError {
      try this.checkDepth()
      this.stack.push(Building(isList: false))
    }

    fun key(name: string) throws EncodeError {
      val top = this.stack.ref(-1) ?: panic(openFrame)
      top.key = name
    }

    fun endObject() throws EncodeError {
      val top = this.stack.removeAt(this.stack.lastIndex())
      val fields: MutableMap<string, Value> = [:]
      var i: i64 = 0
      loop (k in top.keys) {
        fields.set(k, top.values.at(i) ?: panic("codec: an object's keys and values are pushed in pairs"))
        i += 1
      }
      this.put(VObject(fields))
    }

    fun beginList() throws EncodeError {
      try this.checkDepth()
      this.stack.push(Building(isList: true))
    }

    fun endList() throws EncodeError {
      val top = this.stack.removeAt(this.stack.lastIndex())
      this.put(VList(items: top.values.toList()))
    }

    fun writeI64(v: i64) throws EncodeError => this.put(VInt(value: v))
    fun writeU64(v: u64) throws EncodeError => this.put(VInt(value: v.wrapI64()))
    fun writeF64(v: f64) throws EncodeError => this.put(VFloat(value: v))
    fun writeBool(v: bool) throws EncodeError => this.put(VBool(value: v))
    fun writeString(v: string) throws EncodeError => this.put(VString(value: v))
    fun writeNull() throws EncodeError => this.put(VNull())
  }
}
