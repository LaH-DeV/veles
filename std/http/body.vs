// Request bodies (D97). A request's body is not read with its head: the
// handler asks for it — `req.bytes()`, `req.text()`, `req.form<T>()`,
// `req.stream(max:)` — and what it asks for is read then, off the wire,
// under a ceiling it can see. A handler that never asks costs the server no
// read (and a client that sent `Expect: 100-continue` is never told to go
// ahead).
use net

// How a body is framed on the wire, and where a read has got to. It sits in a
// Mutex so that a Request stays Sendable and every copy of it — a middleware
// makes one with `withHeader` — sees the same place in the same body.
struct BodyState {
  // 0 no body, 1 Content-Length, 2 chunked, 3 held in memory
  var framing: i64 = 0
  // bytes left of the length, or of the chunk being read
  var remaining: i64 = 0
  var memory:    List<u8> = []
  // bytes handed out so far, against the ceiling
  var total: i64 = 0
  // Content-Length, or the memory's length; -1 when the body is chunked
  var declared: i64 = -1
  var done:     bool = false
  // a read failed half way: where the next request begins is lost
  var broken: bool = false
  // the client sent `Expect: 100-continue` and has not been answered
  var waiting: bool = false
  var cached:  List<u8>? = null
}

/// The unread body of a request: `req.stream(max: 100 * 1024 * 1024)`.
/// `read` hands out what has arrived, a piece at a time, so a body larger
/// than memory can go to a file or a parser as it comes.
public struct Body {
  conn:  net.Conn?
  state: Mutex<BodyState>
  // the longest silence between two reads
  timeout: Duration
  // what `bytes()` and `text()` collect when told no other ceiling
  defaultMax: i64
  // the ceiling of this handle: reading past it is a 413
  limit: i64

  static fun none(limits: Limits): Body = Body.hold([], limits)

  // a body already in memory (`http.call`, a test)
  static fun hold(bytes: List<u8>, limits: Limits): Body {
    val done = bytes.isEmpty()
    Body(
      conn: null,
      state: Mutex(value: BodyState(framing: if (done) 0 else 3, memory: bytes, declared: bytes.len(), done)),
      timeout: limits.bodyTimeout,
      defaultMax: limits.bodyBytes,
      limit: limits.bodyBytes,
    )
  }

  // a body on the wire: `length` bytes (framing 1), or chunked (framing 2)
  static fun wire(conn: net.Conn, framing: i64, length: i64, waiting: bool, limits: Limits): Body =
    Body(
      conn,
      state: Mutex(value: BodyState(framing, remaining: length, declared: if (framing == 1) length else -1, waiting)),
      timeout: limits.bodyTimeout,
      defaultMax: limits.bodyBytes,
      limit: limits.bodyBytes,
    )

  fun withLimit(max: i64): Body =
    Body(conn: this.conn, state: this.state, timeout: this.timeout, defaultMax: this.defaultMax, limit: max)

  /// How many bytes the body has, when the request said: its
  /// `Content-Length`. `null` for a chunked body, whose end is not known
  /// until it comes.
  public fun length(): i64? {
    val declared = this.state.get().declared
    if (declared < 0) null else declared
  }

  /// Up to `max` bytes, as soon as any have arrived; empty at the end of the
  /// body. Reading past this handle's ceiling is a 413, a silence longer
  /// than `Limits.bodyTimeout` a 408, and a body that ends short or is
  /// framed wrongly a 400.
  public fun read(max: i64 = 65536): List<u8> suspends throws Fail | IoError {
    val st = this.state.get()
    if (st.done || max <= 0) return []
    if (st.broken) throw badRequest("the request body could not be read")
    // a declared length over the ceiling is refused before a byte is asked
    // for, and before the client is told to send it
    if (st.declared >= 0 && st.declared > this.limit) {
      // the client may be sending it: the connection cannot be reused
      this.markBroken()
      throw tooLarge()
    }
    if (st.framing == 3) return try this.readMemory(st, max)
    val c = this.conn ?: return []
    if (st.waiting) {
      this.state.withLock(s => {
        s.waiting = false
      })
      when (c.write("HTTP/1.1 100 Continue\r\n\r\n".bytes())) {
        is Ok(_)  => { }
        is Err(e) => {
          this.markBroken()
          throw e
        }
      }
    }
    val outcome = if (st.framing == 1) this.pullLength(c, max) else this.pullChunked(c, max)
    when (outcome) {
      is Ok(chunk) => chunk
      is Err(e)    => {
        this.markBroken()
        when (e) {
          is Fail    => throw e
          is IoError => throw e
        }
      }
    }
  }

  /// The rest of the body, at most as much as this handle's ceiling allows.
  public fun readAll(): List<u8> suspends throws Fail | IoError {
    val out: MutableList<u8> = []
    loop {
      val chunk = try this.read(65536)
      if (chunk.isEmpty()) break
      out.addAll(chunk)
    }
    out.toList()
  }

  // what `Request.bytes` collected, kept for the next call that wants it
  fun cached(): List<u8>? = this.state.get().cached

  fun remember(bytes: List<u8>) {
    this.state.withLock(s => {
      s.cached = bytes
    })
  }

  fun markBroken() {
    this.state.withLock(s => {
      s.broken = true
    })
  }

  fun readMemory(st: BodyState, max: i64): List<u8> throws Fail {
    val n = if (max < st.memory.len()) max else st.memory.len()
    if (st.total + n > this.limit) throw tooLarge()
    val chunk = st.memory.take(n)
    this.state.withLock(s => {
      s.memory = s.memory.drop(n)
      s.total += n
      if (s.memory.isEmpty()) s.done = true
    })
    chunk
  }

  fun pullLength(c: net.Conn, max: i64): List<u8> suspends throws Fail | IoError {
    val remaining = this.state.get().remaining
    val chunk = try this.recv(c, if (max < remaining) max else remaining)
    if (chunk.isEmpty()) throw badRequest("body shorter than content-length")
    this.state.withLock(s => {
      s.remaining -= chunk.len()
      s.total += chunk.len()
      if (s.remaining == 0) s.done = true
    })
    chunk
  }

  // one piece of a chunked body: the chunk header when none is open, then
  // what has arrived of the chunk's data
  fun pullChunked(c: net.Conn, max: i64): List<u8> suspends throws Fail | IoError {
    var remaining = this.state.get().remaining
    if (remaining == 0) {
      val header = try this.recvLine(c, 1024)
      val size = parseChunkSize(header) ?: throw badRequest("malformed chunk size")
      if (size == 0) {
        try this.skipTrailers(c)
        this.state.withLock(s => {
          s.done = true
        })
        return []
      }
      // a chunk that cannot fit under the ceiling is refused before its data
      if (this.state.get().total + size > this.limit) throw tooLarge()
      remaining = size
    }
    val chunk = try this.recv(c, if (max < remaining) max else remaining)
    if (chunk.isEmpty()) throw badRequest("body ended inside a chunk")
    remaining -= chunk.len()
    // the line break that ends a chunk's data
    if (remaining == 0 && !(try this.recvLine(c, 2).isEmpty())) throw badRequest("malformed chunk end")
    this.state.withLock(s => {
      s.remaining = remaining
      s.total += chunk.len()
    })
    chunk
  }

  // trailer fields follow the last chunk; they are read and dropped, since
  // nothing a sender puts there may change how the request was framed or
  // decided (RFC 9110 §6.5.1)
  fun skipTrailers(c: net.Conn) suspends throws Fail | IoError {
    var lines: i64 = 0
    var bytes: i64 = 0
    loop {
      val line = try this.recvLine(c, 8192)
      if (line.isEmpty()) return
      lines += 1
      bytes += line.len()
      if (lines > 32 || bytes > 8192) throw badRequest("trailer fields too large")
    }
  }

  fun recv(c: net.Conn, n: i64): List<u8> suspends throws Fail | IoError {
    when (withTimeout(this.timeout, () => try c.read(max: n))) {
      is Ok(bytes) => bytes
      is Err(e)    => when (e) {
        is Timeout => throw Fail(status: Status.requestTimeout, text: "request body timeout")
        is IoError => throw e
      }
    }
  }

  fun recvLine(c: net.Conn, max: i64): string suspends throws Fail | IoError {
    when (withTimeout(this.timeout, () => try c.readLine(max: max))) {
      is Ok(line) => line ?: throw badRequest("body ended inside the framing")
      is Err(e)   => when (e) {
        is Timeout     => throw Fail(status: Status.requestTimeout, text: "request body timeout")
        is net.TooLong => throw badRequest("body framing line too long")
        is IoError     => try rethrow(e)
      }
    }
  }

  // Whether the connection can carry another request after what the handler
  // left unread is thrown away: not when a read failed, not when the client
  // is still waiting for `100 Continue` (its body was never sent), not when
  // more than `cap` bytes are left — a chunked body's rest is not known, so
  // it is tried.
  fun reusable(cap: i64): bool {
    val st = this.state.get()
    if (st.done || st.framing == 0 || st.framing == 3) return true
    if (st.broken || st.waiting) return false
    st.declared < 0 || st.declared - st.total <= cap
  }

  // Reads and throws away the rest of the body, up to `cap` bytes; false
  // when it could not, and the connection must close.
  fun drain(cap: i64): bool suspends {
    val st = this.state.get()
    if (st.done || st.framing == 0 || st.framing == 3) return true
    if (!this.reusable(cap)) return false
    val rest = this.withLimit(st.total + cap)
    loop {
      when (rest.read(65536)) {
        is Ok(chunk) => if (chunk.isEmpty()) return true
        is Err(_)    => return false
      }
    }
  }
}

fun tooLarge(): Fail = Fail(status: Status.contentTooLarge, text: "payload too large")

// The size on a chunk header line: hexadecimal, then any `;extension`, which
// is ignored. Sixteen digits would not fit an i64 and no body is that large.
fun parseChunkSize(line: string): i64? {
  val (size, _) = line.splitOnce(";") ?: (line, "")
  val digits = size.trim()
  if (digits.isEmpty() || digits.len() > 15) return null
  var n: i64 = 0
  loop (b in digits.bytes()) {
    val v = hexValue(b)
    if (v < 0) return null
    n = n * 16 + v
  }
  n
}

// How much an unread body may be to keep the connection: more is closed on
// rather than read for nothing
val drainCap: i64 = 65536
