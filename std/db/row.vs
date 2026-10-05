// Rows read into structs: a `codec.Decoder` over one row of a result, so a
// type's derived `Decodable` (D58) is all `query<User>` needs. A struct is the
// row — its fields' names, in snake_case, are the columns' (`@key(db:
// "user_id")` says one outright) — and a scalar target (`query<i64>`) reads the first column. The server
// sends every value as text, so reading is parsing: a number that is not
// one, or a column of the wrong kind, is a problem at that column and the row
// is not built.
use codec, time

// the OIDs of the types the decoder knows by name
const oidBool: i64 = 16
const oidBytea: i64 = 17
const oidInt8: i64 = 20
const oidInt2: i64 = 21
const oidInt4: i64 = 23
const oidOid: i64 = 26
const oidFloat4: i64 = 700
const oidFloat8: i64 = 701
const oidDate: i64 = 1082
const oidTimestamp: i64 = 1114
const oidTimestamptz: i64 = 1184
const oidNumeric: i64 = 1700

/// Where a `RowDecoder` is, behind a pointer because a decoder is handed
/// around as a trait object, which boxes a copy of the struct (D9).
struct RowState {
  var stage:   i64 = 0             // 0: nothing read, 1: the row's object is open, 2: closed
  var next:    i64 = 0             // the column the next key names
  var column:  i64 = -1            // the column being read, -1 when none
  var pending: bool = false        // its value has not been read yet
  var items:   List<string?> = []  // the elements of an array being read
  var inList:  bool = false
  var listAt:  i64 = 0
  var depth:   i64 = 0  // objects or lists opened inside a column: a row has none, so they are skipped
}

struct RowDecoder {
  columns:  List<Column>
  row:      List<string?>
  state:    *RowState
  recorded: codec.Problems = codec.Problems()

  static fun of(columns: List<Column>, row: List<string?>): RowDecoder =
    RowDecoder(columns, row, state: &RowState())

  // the text of the column being read, or of the first one for a scalar target
  fun cell(): string? {
    val s = this.state
    if (s.inList) return s.items.at(s.listAt) ?: null
    val i = if (s.column < 0) 0 else s.column
    this.row.at(i) ?: null
  }

  fun oid(): i64 {
    val s = this.state
    val i = if (s.column < 0) 0 else s.column
    (this.columns.at(i)?.oid) ?: 0
  }

  fun here(): string {
    val s = this.state
    if (s.column < 0 || !s.pending && !s.inList) return ""
    val name = this.columns.at(s.column)?.name ?: ""
    if (s.inList) codec.indexPath(name, s.listAt) else name
  }

  // the value being read, consumed; null (and no problem) for SQL NULL
  fun take(): string? {
    val s = this.state
    val v = this.cell()
    if (s.inList) s.listAt += 1 else s.pending = false
    v
  }

  fun bad(at: string, want: string, got: string) {
    this.recorded.record(at, "expected $want, found \"$got\"")
  }

  implement codec.Decoder {
    fun format(): string = "db"
    override fun keys(): codec.KeyStyle = codec.KeyStyle.SnakeCase

    fun peek(): codec.Kind throws DecodeError {
      val s = this.state
      // nothing read yet: a struct target would not ask; a nullable scalar one asks whether column 0 is NULL
      if (s.stage == 0 && s.column < 0) return if (this.cell() == null) codec.Kind.Null else codec.Kind.Object
      if (this.cell() == null) return codec.Kind.Null
      if (s.inList) return codec.Kind.String
      val oid = this.oid()
      if (oid == oidBool) return codec.Kind.Bool
      if (oid == oidInt2 || oid == oidInt4 || oid == oidInt8 || oid == oidOid) return codec.Kind.Int
      if (oid == oidFloat4 || oid == oidFloat8 || oid == oidNumeric) return codec.Kind.Float
      codec.Kind.String
    }

    fun beginObject() throws DecodeError {
      val s = this.state
      if (s.stage == 0 && s.column < 0) {
        s.stage = 1
        return
      }
      this.recorded.record(this.here(), "a column does not hold an object; read it as text and decode it")
      s.pending = false
      s.depth += 1
    }

    fun nextKey(): string? throws DecodeError {
      val s = this.state
      if (s.depth > 0) return null
      val c = this.columns.at(s.next) ?: return null
      s.column = s.next
      s.next += 1
      s.pending = true
      c.name
    }

    fun endObject() throws DecodeError {
      val s = this.state
      if (s.depth > 0) {
        s.depth -= 1
        return
      }
      s.stage = 2
      s.pending = false
    }

    fun beginList() throws DecodeError {
      val s = this.state
      if (s.inList) {
        this.recorded.record(this.here(), "a column does not hold a list of lists")
        s.depth += 1
        return
      }
      val at = this.here()
      val taken = this.take()
      if (taken == null) {
        s.items = []
        s.inList = true
        s.listAt = 0
        return
      }
      val text = taken
      val parsed = if (this.oid() == oidBytea) byteaItems(text) else arrayItems(text)
      if (parsed == null) {
        this.recorded.record(at, "expected an array or bytea, found \"$text\"")
        s.items = []
      } else {
        s.items = parsed
      }
      s.inList = true
      s.listAt = 0
    }

    fun hasNext(): bool throws DecodeError {
      val s = this.state
      s.depth == 0 && s.listAt < s.items.len()
    }

    fun endList() throws DecodeError {
      val s = this.state
      if (s.depth > 0) {
        s.depth -= 1
        return
      }
      s.inList = false
      s.items = []
      s.pending = false
    }

    fun readString(): string throws DecodeError {
      val oid = this.oid()
      val text = this.take() ?: return ""
      if (oid == oidTimestamp || oid == oidTimestamptz) return rfc3339(text, oid == oidTimestamptz)
      text
    }

    fun readI64(): i64 throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return 0
      val n = text.toInt()
      if (n == null) {
        this.bad(at, "an integer", text)
        return 0
      }
      n
    }

    fun readU64(): u64 throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return 0
      val n = text.toInt()
      if (n == null || n < 0) {
        this.bad(at, "a whole number, not negative", text)
        return 0
      }
      n.wrapU64()
    }

    fun readF64(): f64 throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return 0.0
      val x = when (text) {
        "NaN"       => 0.0 / 0.0
        "Infinity"  => 1.0 / 0.0
        "-Infinity" => -1.0 / 0.0
        else        => text.toF64()
      }
      if (x == null) {
        this.bad(at, "a number", text)
        return 0.0
      }
      x
    }

    fun readBool(): bool throws DecodeError {
      val at = this.here()
      val text = this.take() ?: return false
      when (text) {
        "t", "true"  => true
        "f", "false" => false
        else         => {
          this.bad(at, "t or f", text)
          false
        }
      }
    }

    fun readNull() throws DecodeError {
      val _ = this.take()
    }

    fun skip() throws DecodeError {
      val _ = this.take()
    }

    fun path(): string = this.here()

    fun problemAt(path: string, message: string) = this.recorded.record(path, message)

    fun problems(): List<codec.Problem> = this.recorded.list()
  }
}

/// `T` read from each row of `rows`: every problem of a row is reported together, at its column.
fun decodeRows<T: Decodable>(rows: Rows): List<T> throws DecodeError {
  val out: MutableList<T> = []
  loop (row in rows.rows) {
    val dec = RowDecoder.of(rows.columns, row)
    val value = try T.decode(dec)
    out.push(try codec.finish(dec, value))
  }
  out.toList()
}

// A timestamp as the server writes it (`2026-10-05 12:34:56.789+02`, or without a zone for
// `timestamp`, read as UTC) in RFC 3339 (`2026-10-05T12:34:56.789+02:00`).
fun rfc3339(text: string, zoned: bool): string {
  val t = text.replace(" ", "T")
  if (!t.contains("T")) return t  // infinity, -infinity
  // the zone, if any, starts at the first + or - after the time's 'T'
  val at = t.indexOf("T")
  var zone = -1
  loop (i in (at + 1)..<t.len()) {
    val b = t.byteAt(i)
    if (b == 43 || b == 45) {
      zone = i
      break
    }
  }
  if (zone < 0) return t + "Z"
  val head = t.substring(0, zone) ?: t
  val offset = t.substring(zone, t.len()) ?: ""
  // +02 → +02:00, +0530 → +05:30
  if (offset.len() == 3) return head + offset + ":00"
  if (offset.len() == 5) return head + (offset.substring(0, 3) ?: "") + ":" + (offset.substring(3, 5) ?: "")
  head + offset
}

// `\x0a0b` (bytea in its hex output form) as the decimal text of each byte
fun byteaItems(text: string): List<string?>? {
  if (!text.startsWith("\\x")) return null
  val out: MutableList<string?> = []
  val bytes = text.bytes()
  var i = 2
  loop (i + 1 < bytes.len()) {
    val hi = hexValue(bytes.at(i) ?: 0)
    val lo = hexValue(bytes.at(i + 1) ?: 0)
    if (hi < 0 || lo < 0) return null
    out.push("${hi * 16 + lo}")
    i += 2
  }
  if (i != bytes.len()) return null
  out.toList()
}

// An array in its text form, `{1,2,3}` or `{"a b",NULL,c}`, as its elements. One
// dimension of scalars; a nested array is not a list of scalars and is refused.
fun arrayItems(text: string): List<string?>? {
  val bytes = text.bytes()
  if (bytes.len() < 2 || bytes.at(0) != 123 || bytes.at(bytes.len() - 1) != 125) return null
  val out: MutableList<string?> = []
  var i = 1
  val end = bytes.len() - 1
  if (end == 1) return out.toList()
  loop {
    val first = bytes.at(i)
    if (first == 123) return null
    val item: MutableList<u8> = []
    var quoted = false
    if (first == 34) {
      quoted = true
      i += 1
      loop {
        if (i >= end) return null
        val b = bytes.at(i) ?: 0
        if (b == 92) {
          if (i + 1 >= end) return null
          item.push(bytes.at(i + 1) ?: 0)
          i += 2
        } else if (b == 34) {
          i += 1
          break
        } else {
          item.push(b)
          i += 1
        }
      }
    } else {
      loop {
        if (i >= end) break
        val b = bytes.at(i) ?: 0
        if (b == 44) break
        item.push(b)
        i += 1
      }
    }
    val s = item.decodeUtf8() ?: return null
    out.push(if (!quoted && s == "NULL") null else s)
    if (i >= end) break
    if (bytes.at(i) != 44) return null
    i += 1
  }
  out.toList()
}
