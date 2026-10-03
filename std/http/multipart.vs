// `multipart/form-data` (D97): the body of a form with files, read one part
// at a time off the request's body, so a large upload goes to disk as it
// arrives and nothing but the part being read is ever held.
use fs

// the longest header block a part may have
const maxPartHeaderBytes: i64 = 8192

const crlf: List<u8> = [13, 10]
const dashes: List<u8> = [45, 45]
const space: List<u8> = [32]
const tab: List<u8> = [9]

// where the reader has got to
struct MultipartState {
  // 0 before the first boundary, 1 between parts, 2 inside a part, 3 done
  var phase: i64 = 0
  // the body has no more bytes
  var eof:   bool = false
  var parts: i64 = 0
}

/// The parts of a `multipart/form-data` request, one after the other:
///
/// ```veles
/// val form = try req.multipart(max: 50 * 1024 * 1024)
/// loop {
///   val part = try form.next() ?: break
///   if (val name = part.filename) {
///     val saved = try part.saveTo(path.join(uploads, "upload-1"), max: 10 * 1024 * 1024)
///     io.println("${part.name}: ${name}, $saved bytes")
///   } else {
///     io.println("${part.name} = ${try part.text(max: 4096)}")
///   }
/// }
/// ```
///
/// A part is read from where the last one left off, so it must be read (or
/// left; `next` skips what remains of it) before the next is asked for.
public struct Multipart {
  source: Body
  // CR LF, two dashes and the boundary: what ends a part
  delimiter: List<u8>
  maxParts:  i64
  // bytes taken from the body and not yet used; it starts with a virtual
  // CR LF, so that the first boundary is found like every other
  buffer: Mutex<List<u8>> = Mutex(value: crlf)
  state:  Mutex<MultipartState> = Mutex(value: MultipartState())

  /// The next part, or `null` after the last. Too many parts is a 413, a
  /// body that is not well-formed multipart a 400.
  public fun next(): Part? suspends throws Fail | IoError {
    if (this.state.get().phase == 3) return null
    if (this.state.get().phase == 2) try this.skipPart()
    try this.seekDelimiter()
    // what follows a delimiter: `--` ends the body, a line break starts a part
    try this.need(2)
    if (this.peek(2) == dashes) {
      this.state.withLock(s => {
        s.phase = 3
      })
      return null
    }
    // transport padding (spaces and tabs) may sit before the line break
    loop {
      try this.need(1)
      val pad = this.peek(1)
      if (pad != space && pad != tab) break
      this.skip(1)
    }
    try this.need(2)
    if (this.peek(2) != crlf) throw badRequest("malformed multipart boundary line")
    this.skip(2)
    val count = this.state.withLock(s => {
      s.parts += 1
      s.parts
    })
    if (count > this.maxParts) throw Fail(status: Status.contentTooLarge, text: "too many parts in the form")
    val headers = try this.partHeaders()
    val disposition = headers.get("content-disposition") ?: throw badRequest("a multipart part has no Content-Disposition")
    val (kind, params) = parseParams(disposition)
    if (kind.toLower() != "form-data") throw badRequest("a multipart part is not form-data")
    val name = params.get("name") ?: throw badRequest("a multipart part has no name")
    this.state.withLock(s => {
      s.phase = 2
    })
    Part(name, headers, filename: params.get("filename"), contentType: headers.get("content-type"), form: this)
  }

  fun peek(n: i64): List<u8> = this.buffer.get().take(n)

  fun skip(n: i64) {
    this.buffer.withLock(b => {
      *b = (*b).drop(n)
    })
  }

  fun held(): i64 = this.buffer.get().len()

  // bytes of the current part, up to `max`, never running into the boundary
  // that ends it; empty once it has
  fun readPart(max: i64): List<u8> suspends throws Fail | IoError {
    if (this.state.get().phase != 2) return []
    loop {
      val at = findBytes(this.buffer.get(), this.delimiter, 0)
      if (at == 0) {
        // the boundary is next: the part is over
        this.state.withLock(s => {
          s.phase = 1
        })
        return []
      }
      // up to the boundary when it is in view, else all but the tail that
      // could still turn out to be the start of one
      val safe = if (at > 0) at else this.held() - (this.delimiter.len() - 1)
      if (safe > 0) {
        val piece = this.peek(if (max < safe) max else safe)
        this.skip(piece.len())
        return piece
      }
      if (!(try this.fill())) throw badRequest("the multipart body ends inside a part")
    }
  }

  fun skipPart() suspends throws Fail | IoError {
    loop {
      if (this.state.get().phase != 2) return
      val piece = try this.readPart(65536)
      if (piece.isEmpty()) return
    }
  }

  // Moves past the bytes before the next boundary — the preamble before
  // the first one — and past the boundary itself.
  fun seekDelimiter() suspends throws Fail | IoError {
    loop {
      val at = findBytes(this.buffer.get(), this.delimiter, 0)
      if (at >= 0) {
        this.skip(at + this.delimiter.len())
        this.state.withLock(s => {
          s.phase = 1
        })
        return
      }
      // nothing to keep but a tail that might begin one
      val drop = this.held() - (this.delimiter.len() - 1)
      if (drop > 0) this.skip(drop)
      if (!(try this.fill())) throw badRequest("the multipart body has no closing boundary")
    }
  }

  // the header lines of a part, up to the blank line, as a map of lower-case
  // names
  fun partHeaders(): Map<string, string> suspends throws Fail | IoError {
    val out: MutableMap<string, string> = [:]
    loop {
      // a line of the block, ended by CR LF
      var end = -1
      loop {
        end = findBytes(this.buffer.get(), crlf, 0)
        if (end >= 0) break
        if (this.held() > maxPartHeaderBytes) throw badRequest("multipart part headers too large")
        if (!(try this.fill())) throw badRequest("the multipart body ends inside a part's headers")
      }
      if (end > maxPartHeaderBytes) throw badRequest("multipart part headers too large")
      val line = this.peek(end)
      this.skip(end + 2)
      if (line.isEmpty()) break
      val text = line.decodeUtf8() ?: throw badRequest("multipart part header is not valid UTF-8")
      val (name, value) = text.splitOnce(":") ?: throw badRequest("malformed multipart part header")
      if (!isToken(name)) throw badRequest("malformed multipart part header name")
      out.set(name.toLower(), value.trim())
    }
    out.toMap()
  }

  // at least `n` bytes in the buffer, or the body's end
  fun need(n: i64) suspends throws Fail | IoError {
    loop (this.held() < n) {
      if (!(try this.fill())) throw badRequest("the multipart body ends inside its framing")
    }
  }

  // one piece of the body into the buffer; false at its end
  fun fill(): bool suspends throws Fail | IoError {
    if (this.state.get().eof) return false
    val chunk = try this.source.read(65536)
    if (chunk.isEmpty()) {
      this.state.withLock(s => {
        s.eof = true
      })
      return false
    }
    this.buffer.withLock(b => {
      *b = (*b).concat(chunk)
    })
    true
  }
}

/// One part of a multipart form: a field, or an uploaded file.
public struct Part {
  /// The field name.
  public name: string
  /// The file name the client gave, only for a file; it is the client's
  /// word, so it never names a path — `saveTo` takes the path.
  public filename: string?
  /// The part's own `Content-Type`, when it sent one (a file usually does).
  public contentType: string?
  /// The part's headers, names in lower case.
  public headers: Map<string, string>
  form:           Multipart

  /// Up to `max` bytes of the part, as they arrive; empty at its end.
  public fun read(max: i64 = 65536): List<u8> suspends throws Fail | IoError = try this.form.readPart(max)

  /// The rest of the part, at most `max` bytes: a longer one is a 413.
  public fun bytes(max: i64): List<u8> suspends throws Fail | IoError {
    val out: MutableList<u8> = []
    loop {
      val piece = try this.read(65536)
      if (piece.isEmpty()) break
      if (out.len() + piece.len() > max) throw Fail(status: Status.contentTooLarge, text: "the part '${this.name}' is larger than $max bytes")
      out.addAll(piece)
    }
    out.toList()
  }

  /// The rest of the part as text, at most `max` bytes; text that is not
  /// UTF-8 is a 400.
  public fun text(max: i64): string suspends throws Fail | IoError =
    try (try this.bytes(max)).decodeUtf8() ?! badRequest("the part '${this.name}' is not valid UTF-8")

  /// Writes the rest of the part to `path`, replacing what is there, and
  /// returns how many bytes it was. A part longer than `max` is a 413 and
  /// leaves no file behind. The path is the caller's: build it from
  /// something the caller chose, never from `filename`.
  public fun saveTo(path: string, max: i64): i64 suspends throws Fail | IoError {
    var written: i64 = 0
    var tooLong = false
    with (f = try fs.open(path, fs.FileMode.Write)) {
      loop {
        val piece = try this.read(65536)
        if (piece.isEmpty()) break
        written += piece.len()
        if (written > max) {
          tooLong = true
          break
        }
        try f.write(piece)
      }
    }
    if (tooLong) {
      val _ = fs.remove(path)
      throw Fail(status: Status.contentTooLarge, text: "the part '${this.name}' is larger than $max bytes")
    }
    written
  }
}

// index of the first `needle` in `xs` at or after `from`, or -1
fun findBytes(xs: List<u8>, needle: List<u8>, from: i64): i64 {
  val last = xs.len() - needle.len()
  loop (i in from..last) {
    var same = true
    loop (j in 0..<needle.len()) {
      if (xs.at(i + j) != needle.at(j)) {
        same = false
        break
      }
    }
    if (same) return i
  }
  -1
}

// `form-data; name="a"; filename="b c.png"`: the first word, and the
// parameters, quoted values unquoted. A `;` inside quotes stays in the value,
// and a backslash is an ordinary byte (RFC 7578 §4.2: browsers send a file
// name as it is, `C:\dir\a.txt` included). Names are lower-cased.
fun parseParams(value: string): (string, Map<string, string>) {
  val bytes = value.bytes()
  val params: MutableMap<string, string> = [:]
  var first = ""
  var current: MutableList<u8> = []
  val fields: MutableList<string> = []
  var quoted = false
  loop (b in bytes) {
    if (quoted) {
      if (b == '"') quoted = false
      current.push(b)
    } else if (b == '"') {
      quoted = true
      current.push(b)
    } else if (b == ';') {
      fields.push(current.toList().decodeUtf8() ?: "")
      current = []
    } else {
      current.push(b)
    }
  }
  fields.push(current.toList().decodeUtf8() ?: "")
  loop (k in 0..<fields.len()) {
    val field = (fields.at(k) ?: "").trim()
    if (k == 0) {
      first = field
      continue
    }
    val (name, unquoted) = field.splitOnce("=") ?: continue
    var v = unquoted.trim()
    if (v.len() >= 2 && v.startsWith("\"") && v.endsWith("\"")) v = v.substring(1, v.len() - 1) ?: v
    params.set(name.trim().toLower(), v)
  }
  (first, params.toMap())
}
