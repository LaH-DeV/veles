// Forms and query strings (D94): the `name=value&...` text of a urlencoded
// body or of the part of a URL after `?`, kept with its repeats, and read
// into a struct through the codec layer (D58) the way `json.decode` reads a
// document.
use codec

/// The fields of a urlencoded text, in order, repeats kept: `a=1&b=2&a=3`
/// has two names and three values. Read one with `get`, all of a name with
/// `all`; or hand the whole thing to `Request.form<T>()` / `query<T>()`.
public struct Fields {
  public pairs: List<(string, string)>

  /// The fields of `text`: `name=value` pairs joined by `&`, `+` a space,
  /// `%XX` escapes decoded; a name without `=` has the empty value.
  public static fun parse(text: string): Fields {
    val pairs: MutableList<(string, string)> = []
    if (!text.isEmpty()) {
      loop (pair in text.split("&")) {
        if (pair.isEmpty()) continue
        val (k, v) = pair.splitOnce("=") ?: (pair, "")
        pairs.push((percentDecode(k, plusIsSpace: true), percentDecode(v, plusIsSpace: true)))
      }
    }
    Fields(pairs: pairs.toList())
  }

  /// The first value of `name`, or null.
  public fun get(name: string): string? {
    loop ((k, v) in this.pairs) {
      if (k == name) return v
    }
    null
  }

  /// Every value of `name`, in order (empty when it is absent).
  public fun all(name: string): List<string> {
    val out: MutableList<string> = []
    loop ((k, v) in this.pairs) {
      if (k == name) out.push(v)
    }
    out.toList()
  }

  /// The distinct names, in the order they first appear.
  public fun names(): List<string> {
    val seen: MutableSet<string> = []
    val out: MutableList<string> = []
    loop ((k, _) in this.pairs) {
      if (seen.contains(k)) continue
      seen.add(k)
      out.push(k)
    }
    out.toList()
  }
}

// ---------------------------------------------------------------------------
// reading a struct from fields

/// A `T` read from `fields`: `decodeFields<Signup>(fields)`. Every problem is
/// reported together, each at its field's name.
fun decodeFields<T: Decodable>(fields: Fields, keys: codec.KeyStyle): T throws DecodeError {
  val dec = FormDecoder.of(fields, keys)
  val value = try T.decode(dec)
  try codec.finish(dec, value)
}

/// Where a `FormDecoder` is; behind a pointer, because a decoder is handed
/// around as a trait object, which boxes a copy of the struct (D9).
struct FormState {
  var stage:    i64 = 0      // 0: nothing read yet, 1: the root object is open, 2: it closed
  var next:     i64 = 0      // the index of the next name to hand out
  var name:     string = ""  // the member being read
  var values:   List<string> = []
  var pending:  bool = false  // its value has not been read yet
  var inList:   bool = false
  var listAt:   i64 = 0
  var nested:   i64 = 0  // objects opened inside the member: forms have none, so they are skipped
  var skipList: i64 = 0  // the same for lists inside a list
}

/// The `codec.Decoder` over `Fields`: one flat object whose members are
/// the field names. A member with one value reads as a scalar and, when a
/// list is asked for, as a list of one; one with several values reads only
/// as a list. Text is turned into what the struct asks for — an integer, a
/// number, a boolean (`true`/`false`, `on`/`off`, `yes`/`no`, `1`/`0`, which
/// is what a checkbox sends), a string, an enum by name — and a value that
/// does not fit is a problem at that field. An empty value is `null` where
/// the field is optional, and an empty list where it is a list; nothing
/// nested exists, and asking for it is a problem.
public struct FormDecoder {
  private fields:   Fields
  private names:    List<string>
  private state:    *FormState
  private recorded: codec.Problems = codec.Problems()
  private keyStyle: codec.KeyStyle = codec.KeyStyle.AsWritten

  public static fun of(fields: Fields, keys: codec.KeyStyle = codec.KeyStyle.AsWritten): FormDecoder =>
    FormDecoder(fields, names: fields.names(), state: &FormState(), keyStyle: keys)

  // Where the decoder is: the member being read, or, between members and
  // after the last one, the object itself (a derived decoder names a missing
  // field under it).
  private fun here(): string {
    val s = this.state
    if (!s.pending && !s.inList) return ""
    val base = codec.childPath("", s.name)
    if (s.inList) codec.indexPath(base, s.listAt) else base
  }

  /// The next scalar's text, consuming it; null (and a problem) when the
  /// member has several values and a scalar was asked for.
  private fun take(): string? {
    val s = this.state
    if (s.inList) {
      val v = s.values.at(s.listAt)
      s.listAt += 1
      return v
    }
    val at = this.here()
    s.pending = false
    if (s.values.len() != 1) {
      this.recorded.record(at, "expected one value, found ${s.values.len()}")
      return null
    }
    s.values.at(0)
  }

  implement codec.Decoder {
    fun format(): string => "form"
    override fun keys(): codec.KeyStyle => this.keyStyle

    fun peek(): codec.Kind throws DecodeError {
      val s = this.state
      if (s.stage == 0) return codec.Kind.Object
      if (!s.inList && s.values.len() > 1) return codec.Kind.List
      val v = (if (s.inList) s.values.at(s.listAt) else s.values.at(0)) ?: return codec.Kind.Null
      if (v.isEmpty()) codec.Kind.Null else codec.Kind.String
    }

    fun beginObject() throws DecodeError {
      val s = this.state
      if (s.stage == 0) {
        s.stage = 1
        return
      }
      this.recorded.record(this.here(), "a form has no nested values")
      s.pending = false
      s.nested += 1
    }

    fun nextKey(): string? throws DecodeError {
      val s = this.state
      if (s.nested > 0) return null
      val k = this.names.at(s.next) ?: return null
      s.next += 1
      s.name = k
      s.values = this.fields.all(k)
      s.pending = true
      s.inList = false
      k
    }

    fun endObject() throws DecodeError {
      val s = this.state
      if (s.nested > 0) {
        s.nested -= 1
        return
      }
      s.stage = 2
      s.pending = false
    }

    fun beginList() throws DecodeError {
      val s = this.state
      if (s.inList) {
        this.recorded.record(this.here(), "a form has no nested lists")
        s.skipList += 1
        return
      }
      s.inList = true
      s.listAt = 0
      // `tags=` alone is an empty list, not a list of one empty string
      if (s.values.len() == 1 && (s.values.at(0) ?: "").isEmpty()) s.values = []
    }

    fun hasNext(): bool throws DecodeError {
      val s = this.state
      s.skipList == 0 && s.listAt < s.values.len()
    }

    fun endList() throws DecodeError {
      val s = this.state
      if (s.skipList > 0) {
        s.skipList -= 1
        return
      }
      s.inList = false
      s.pending = false
    }

    fun readString(): string throws DecodeError => this.take() ?: ""

    fun readI64(): i64 throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return 0
      val n = text.trim().toInt()
      if (n == null) {
        this.recorded.record(at, "expected an integer, found \"$text\"")
        return 0
      }
      n
    }

    fun readU64(): u64 throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return 0
      val n = text.trim().toInt()
      if (n == null || n < 0) {
        this.recorded.record(at, "expected a whole number, not negative, found \"$text\"")
        return 0
      }
      n.wrapU64()
    }

    fun readF64(): f64 throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return 0.0
      val x = text.trim().toF64()
      if (x == null) {
        this.recorded.record(at, "expected a number, found \"$text\"")
        return 0.0
      }
      x
    }

    fun readBool(): bool throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return false
      when (text.trim().toLower()) {
        "true", "on", "yes", "1"  => true
        "false", "off", "no", "0" => false
        else                      => {
          this.recorded.record(at, "expected true or false (or on/off, yes/no, 1/0), found \"$text\"")
          false
        }
      }
    }

    fun readNull() throws DecodeError {
      val _ = this.take()
    }

    fun skip() throws DecodeError {
      val s = this.state
      if (s.inList) s.listAt += 1 else s.pending = false
    }

    fun path(): string => this.here()

    fun problemAt(path: string, message: string) => this.recorded.record(path, message)

    fun problems(): List<codec.Problem> => this.recorded.list()
  }
}
