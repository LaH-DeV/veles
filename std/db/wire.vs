// The PostgreSQL wire protocol, version 3: the messages the client sends and
// the pieces of the ones it reads. Only what a client of the extended query
// protocol needs; every length that comes from the server is checked before
// it is trusted.

use io

// ---- what the server may send (the first byte of a message) ----

const msgAuth: u8 = 82               // R
const msgParameterStatus: u8 = 83    // S
const msgBackendKey: u8 = 75         // K
const msgReady: u8 = 90              // Z
const msgError: u8 = 69              // E
const msgNotice: u8 = 78             // N
const msgParseDone: u8 = 49          // 1
const msgBindDone: u8 = 50           // 2
const msgCloseDone: u8 = 51          // 3
const msgRowDescription: u8 = 84     // T
const msgDataRow: u8 = 68            // D
const msgCommandDone: u8 = 67        // C
const msgNoData: u8 = 110            // n
const msgParamDescription: u8 = 116  // t
const msgEmpty: u8 = 73              // I
const msgSuspended: u8 = 115         // s
const msgNotification: u8 = 65       // A

// the most one message may hold (a row of a few hundred megabytes is not a row)
const maxMessage: i64 = 268435456

/// A message being built: the body is written in pieces and `frame` puts the
/// type byte and the length in front.
struct Out {
  body: MutableList<u8> = []

  fun byte(b: u8) {
    this.body.push(b)
  }

  fun i16(n: i64) {
    this.body.pushU16Be(n.wrapU16())
  }

  fun i32(n: i64) {
    this.body.pushU32Be(n.wrapU32())
  }

  // text followed by the NUL that ends it; a NUL inside would end it early
  fun cstring(s: string) {
    this.body.addAll(s.bytes())
    this.body.push(0)
  }

  fun put(bytes: List<u8>) {
    this.body.addAll(bytes)
  }

  // the message: type, length (counting itself), body
  fun frame(kind: u8): List<u8> {
    val out: MutableList<u8> = []
    out.push(kind)
    out.pushU32Be((this.body.len() + 4).wrapU32())
    out.addAll(this.body.toList())
    out.toList()
  }
}

/// A reader over the body of one message. Reading past the end is the
/// server's fault — a `Protocol` error — never a wrong value.
struct Cursor {
  data:            List<u8>
  private var pos: i64 = 0

  fun remaining(): i64 => this.data.len() - this.pos

  fun byte(): u8 throws DbError {
    val b = this.data.at(this.pos) ?: throw truncated()
    this.pos += 1
    b
  }

  fun i16(): i64 throws DbError {
    val v = this.data.readI16Be(this.pos) ?: throw truncated()
    this.pos += 2
    v.toI64()
  }

  fun i32(): i64 throws DbError {
    val v = this.data.readI32Be(this.pos) ?: throw truncated()
    this.pos += 4
    v.toI64()
  }

  fun take(n: i64): List<u8> throws DbError {
    if (n < 0 || n > this.remaining()) throw truncated()
    val out = this.data.slice(this.pos, this.pos + n)
    this.pos += n
    out
  }

  // text up to a NUL
  fun cstring(): string throws DbError {
    var end = this.pos
    loop {
      val b = this.data.at(end) ?: throw truncated()
      if (b == 0) break
      end += 1
    }
    val text = this.data.slice(this.pos, end).decodeUtf8() ?: throw DbError(kind: ErrorKind.Protocol, text: "the server sent text that is not UTF-8")
    this.pos = end + 1
    text
  }

  fun rest(): List<u8> => this.data.slice(this.pos, this.data.len())
}

fun truncated(): DbError => DbError(kind: ErrorKind.Protocol, text: "the server sent a message that ends too early")

/// A message read from the server.
struct Message {
  kind: u8
  body: List<u8>

  fun cursor(): Cursor => Cursor(data: this.body)
}

/// The next message of `stream`. A connection that ends in the middle of
/// one, or announces one of absurd size, is an error.
fun readMessage(stream: io.Stream): Message suspends throws DbError | IoError {
  val head = try stream.readExact(5)
  if (head.len() < 5) {
    throw DbError(kind: ErrorKind.Closed, text: "the server closed the connection")
  }
  val kind = head.at(0)
  val len = (head.readI32Be(1) ?: 0).toI64()
  if (len < 4 || len - 4 > maxMessage) {
    throw DbError(kind: ErrorKind.Protocol, text: "the server sent a message of $len bytes")
  }
  val body = try stream.readExact(len - 4)
  if (body.len() < len - 4) {
    throw DbError(kind: ErrorKind.Closed, text: "the server closed the connection in the middle of a message")
  }
  Message(kind, body)
}

/// A `ErrorResponse` or `NoticeResponse`: fields, each a code byte and text.
fun readFields(c: *Cursor): Map<u8, string> throws DbError {
  val fields: MutableMap<u8, string> = [:]
  loop {
    val code = try c.byte()
    if (code == 0) break
    fields.set(code, try c.cstring())
  }
  fields.toMap()
}

fun serverError(fields: Map<u8, string>): DbError {
  DbError(
    kind: ErrorKind.Server,
    text: fields.get(77) ?: "the server reported an error",  // M
    code: fields.get(67) ?: "",  // C
    detail: fields.get(68) ?: "",  // D
    hint: fields.get(72) ?: "",  // H
    severity: fields.get(83) ?: "ERROR", // S
  )
}

// the failure an ErrorResponse message reports
fun errorOf(m: Message): DbError throws DbError {
  var c = m.cursor()
  serverError(try readFields(&c))
}
