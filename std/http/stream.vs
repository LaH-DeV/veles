// Streamed response bodies (D97): the writer a producer is handed, and the
// boundary it runs behind.
use compress as gz
use io
use log as logs { field }

/// Where a streamed response's body goes (see `Response.stream`).
public struct BodyWriter {
  conn: io.Stream?
  // framed as chunks, or as the bare bytes of an HTTP/1.0 response
  chunked: bool
  // `http.call` has no socket: the bytes are collected here
  memory: Mutex<MutableList<u8>> = newMemory()
  // bytes still owed when the body has a declared length, else -1
  left: Atomic<i64> = Atomic(value: -1)
  // set by `http.compress()`: what is written is compressed on its way out
  gzip: Mutex<gz.GzipEncoder>? = null

  static fun toConn(conn: io.Stream, chunked: bool, length: i64?): BodyWriter => BodyWriter(conn, chunked, left: Atomic(value: length ?: -1))

  static fun toMemory(length: i64?): BodyWriter => BodyWriter(conn: null, chunked: false, left: Atomic(value: length ?: -1))

  // bytes the declared length still asks for; 0 when there is none
  fun missing(): i64 {
    val n = this.left.load()
    if (n < 0) 0 else n
  }

  // this writer, compressing what is written to it (the length, if any, was
  // dropped by the caller)
  fun gzipped(level: i64): BodyWriter =>
    BodyWriter(conn: this.conn, chunked: this.chunked, memory: this.memory, left: this.left, gzip: Mutex(value: gz.GzipEncoder(level)))

  // the end of a compressed body: what the encoder still holds, the checksum
  fun finishGzip() suspends throws IoError {
    val g = this.gzip ?: return
    val tail = g.withLock(e => e.finish())
    try this.send(tail)
  }

  /// Sends `bytes` now. An empty list sends nothing (an empty chunk would
  /// end the body). Throws `IoError` when the client has gone.
  public fun write(bytes: List<u8>) suspends throws IoError {
    if (bytes.isEmpty()) return
    if (val g = this.gzip) {
      // the lock covers the compression only; the send, which may wait, is outside
      try this.send(g.withLock(e => e.push(bytes)))
      return
    }
    try this.send(bytes)
  }

  fun send(bytes: List<u8>) suspends throws IoError {
    if (bytes.isEmpty()) return
    // a body longer than the length announced would run into the next response
    if (this.left.load() >= 0) {
      if (bytes.len() > this.left.load()) throw IoError(path: "response", code: 0, detail: "more bytes than the declared length", kind: IoKind.InvalidInput)
      val _ = this.left.update(l => l - bytes.len())
    }
    if (val c = this.conn) {
      if (this.chunked) {
        try c.write("${hexLength(bytes.len())}\r\n".bytes().concat(bytes).concat("\r\n".bytes()))
      } else {
        try c.write(bytes)
      }
    } else {
      this.memory.withLock(m => {
        m.addAll(bytes)
      })
    }
  }

  /// Sends `text` as UTF-8.
  public fun writeText(text: string) suspends throws IoError {
    try this.write(text.bytes())
  }

  fun collected(): List<u8> => this.memory.withLock(m => m.toList())
}

fun newMemory(): Mutex<MutableList<u8>> {
  val empty: MutableList<u8> = []
  Mutex(value: empty)
}

// a chunk's size line: hexadecimal, lower-case
fun hexLength(n: i64): string {
  if (n == 0) return "0"
  val digits = "0123456789abcdef"
  var out = ""
  var rest = n
  loop (rest > 0) {
    out = (digits.substring(rest % 16, rest % 16 + 1) ?: "0") + out
    rest /= 16
  }
  out
}

// Runs a response's producer behind the same boundary a handler has (D56): a
// panic is logged and is a failure, not a crash. True when it returned.
fun produce(producer: sendable fun(BodyWriter) suspends throws IoError, out: BodyWriter): bool suspends {
  val outcome = gather {
    async runProducer(producer, out)
  }
  when (outcome) {
    is Ok(done) => done
    is Err(p)   => {
      logs.error("response stream panicked", field("panic", p.message()))
      false
    }
  }
}

fun runProducer(producer: sendable fun(BodyWriter) suspends throws IoError, out: BodyWriter): bool suspends =>
  when (producer(out)) {
    is Ok(_)  => true
    is Err(_) => false
  }

// `resp` with a streamed body collected into `body`: what `call` returns
fun collect(resp: Response): Response suspends {
  val producer = resp.stream ?: return resp
  val out = BodyWriter.toMemory(resp.streamLength)
  val done = produce(producer, out) && out.missing() == 0
  Response(status: if (done) resp.status else Status.internalServerError, headers: resp.headers, body: out.collected(), cookies: resp.cookies, quiet: resp.quiet)
}
