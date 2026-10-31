// ---------------------------------------------------------------------------
// errors

/// One thing wrong with a document being decoded: where (`user.address[2].zip`)
/// and what.
public struct Problem {
  public path:    string
  public message: string

  /// The path as an RFC 6901 JSON pointer: `/user/address/2/zip`.
  public fun pointer(): string {
    val out = StringBuilder()
    loop (part in splitPath(this.path)) {
      out.append("/")
      out.append(part.replace("~", "~0").replace("/", "~1"))
    }
    out.toString()
  }

  public fun toString(): string => if (this.path.isEmpty()) this.message else "${this.path}: ${this.message}"
}

// What a panic on an empty frame stack says: derived bodies call `key`,
// `nextKey` and `hasNext` only between a `begin…` and its `end…`.
const openFrame = "codec: a derived body calls this only between a begin and its end, so a frame is open"

/// The parts of a path: `a.b[2].c` is `a`, `b`, `2`, `c`; a quoted key
/// `a["x.y"]` is one part.
fun splitPath(path: string): List<string> {
  val parts: MutableList<string> = []
  val cur = StringBuilder()
  var i = 0
  val n = path.len()
  loop (i < n) {
    val b = path.byteAt(i)
    when {
      b == '.' => {
        if (!cur.isEmpty()) {
          parts.push(cur.toString())
          cur.clear()
        }
        i += 1
      }
      b == '[' => {
        if (!cur.isEmpty()) {
          parts.push(cur.toString())
          cur.clear()
        }
        i += 1
        if (i < n && path.byteAt(i) == '"') {
          i += 1
          loop (i < n && path.byteAt(i) != '"') {
            val b = path.byteAt(i)
            if (b == '\\' && i + 1 < n && path.byteAt(i + 1) == '"') {
              cur.appendByte('"')
              i += 2
            } else {
              cur.appendByte(b)
              i += 1
            }
          }
          i += 1  // the closing quote
        } else {
          loop (i < n && path.byteAt(i) != ']') {
            cur.appendByte(path.byteAt(i))
            i += 1
          }
        }
        parts.push(cur.toString())
        cur.clear()
        i += 1  // the ]
      }
      else     => {
        cur.appendByte(b)
        i += 1
      }
    }
  }
  if (!cur.isEmpty()) parts.push(cur.toString())
  parts
}

/// Encoding failed: the sink refused a write, or a value cannot be
/// represented (a NaN in JSON, a nested object in a row).
public error EncodeError {
  public message: string
  public path:    string = ""
  fun message(): string => if (this.path.isEmpty()) this.message else "${this.path}: ${this.message}"
}

/// Decoding failed. Every problem found is listed — a wrong type here, a
/// missing field there — so a client fixes its document in one round trip;
/// a malformed document is one problem, since nothing follows it.
public error DecodeError {
  public problems: List<Problem>
  fun message(): string => this.problems.map(p => p.toString()).join("\n")
}
