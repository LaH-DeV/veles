// One connection to PostgreSQL: the login (TLS, SCRAM-SHA-256 or a cleartext
// password over TLS) and the extended query protocol, where the text of a
// statement and the values it is run with are different messages — the reason
// nothing a value holds can change what a query means.
//
// A `Conn` runs one statement at a time. Anything that ends a statement
// abnormally — a timeout, a cancelled task, a broken socket — leaves the
// protocol mid-conversation, so the connection is marked broken and never
// used again (the pool drops it).
use base64
use crypto
use fs
use io
use net
use time
use tls

/// A column of a result.
struct Column {
  name: string
  // the type's OID: what the text of the values means (23 is int4, 25 text, 1184 timestamptz, …)
  oid: i64
}

/// What one statement returned: its columns, its rows as the text the server
/// sent (`null` is SQL NULL), and the command tag (`INSERT 0 1`, `SELECT 3`).
struct Rows {
  columns: List<Column>
  rows:    List<List<string?>>
  tag:     string

  // the number a command tag ends with: rows inserted, updated, deleted or returned
  fun affected(): i64 {
    val last = this.tag.split(" ").last() ?: ""
    last.toInt() ?: 0
  }
}

// a failure of the transport as a DbError
fun transport(e: IoError): DbError =>
  DbError(kind: ErrorKind.Connection, text: e.message())

// An operation in progress on a connection. Closed without `done()` — an
// error, a timeout, a cancellation — it marks the connection broken, since
// the server may still be talking.
struct Busy {
  broken:   Atomic<bool>
  finished: Atomic<bool> = Atomic(value: false)

  fun done() {
    this.finished.store(true)
  }

  implement Closeable {
    fun close() {
      if (!this.finished.load()) this.broken.store(true)
    }
  }
}

struct Conn {
  stream: io.Stream
  broken: Atomic<bool> = Atomic(value: false)
  // true between BEGIN and COMMIT/ROLLBACK, from the server's own ReadyForQuery status
  inTransaction: Atomic<bool> = Atomic(value: false)
  // for spans and logs: host:port/database, never the user or the password
  target: string
  // the server's idea of its version, from the startup (`16.4`)
  version: string = ""

  /// Connects and logs in; `timeout` covers all of it.
  static fun open(target: Target, timeout: Duration): Conn suspends throws DbError {
    when (withTimeout(timeout, () => try connect(target))) {
      is Ok(c)  => c
      is Err(e) => when (e) {
        is Timeout => throw DbError(kind: ErrorKind.Timeout, text: "connecting to ${target.shown()} timed out")
        is DbError => throw e
      }
    }
  }

  fun isUsable(): bool => !this.broken.load()

  fun close() {
    this.broken.store(true)
    this.stream.close()
  }

  fun busy(): Busy => Busy(broken: this.broken)

  /// Runs one statement. `timeout` bounds the whole exchange; past it the
  /// connection is dropped, so a statement that is still running on the
  /// server is cancelled by the disconnect.
  fun run(sql: Sql, timeout: Duration): Rows suspends throws DbError {
    if (this.broken.load()) throw DbError(kind: ErrorKind.Closed, text: "the connection is closed")
    with busy = this.busy()
    when (withTimeout(timeout, () => try this.exchange(sql))) {
      is Ok(rows) => {
        busy.done()
        rows
      }
      is Err(e)   => when (e) {
        is Timeout => throw DbError(kind: ErrorKind.Timeout, text: "the statement ran past ${timeout}")
        is DbError => {
          // the server said no and then ReadyForQuery: the conversation is in order
          if (e.kind == ErrorKind.Server) busy.done()
          throw e
        }
      }
    }
  }

  // the messages of one statement, and the answers up to ReadyForQuery
  fun exchange(sql: Sql): Rows suspends throws DbError {
    val text = sql.text()
    val args = sql.args
    val parse = Out()
    parse.cstring("")
    parse.cstring(text)
    parse.i16(args.len())
    loop (a in args) {
      parse.i32(a.oid)
    }
    val bind = Out()
    bind.cstring("")
    bind.cstring("")
    bind.i16(args.len())
    loop (a in args) {
      bind.i16(if (a.bytes != null) 1 else 0)
    }
    bind.i16(args.len())
    loop (a in args) {
      val bytes = a.bytes
      val value = a.text
      if (bytes != null) {
        bind.i32(bytes.len())
        bind.put(bytes)
      } else if (value != null) {
        val encoded = value.bytes()
        bind.i32(encoded.len())
        bind.put(encoded)
      } else {
        bind.i32(-1)
      }
    }
    bind.i16(0)  // every result column as text
    val describe = Out()
    describe.byte(80)  // P: a portal
    describe.cstring("")
    val execute = Out()
    execute.cstring("")
    execute.i32(0)
    val out: MutableList<u8> = []
    out.addAll(parse.frame(80))
    out.addAll(bind.frame(66))
    out.addAll(describe.frame(68))
    out.addAll(execute.frame(69))
    out.addAll(Out().frame(83))  // Sync
    try this.send(out.toList())
    var columns: List<Column> = []
    val rows: MutableList<List<string?>> = []
    var tag = ""
    var failure: DbError? = null
    loop {
      val m = try this.receive()
      if (m.kind == msgReady) {
        this.inTransaction.store((m.body.at(0) ?: 73) != 73)
        break
      } else if (m.kind == msgError) {
        if (failure == null) failure = try errorOf(m)
      } else if (m.kind == msgRowDescription) {
        var c = m.cursor()
        columns = try readColumns(&c)
      } else if (m.kind == msgDataRow) {
        var c = m.cursor()
        rows.push(try readRow(&c))
      } else if (m.kind == msgCommandDone) {
        tag = try m.cursor().cstring()
      }
    }
    if (failure != null) throw failure
    Rows(columns, rows: rows.toList(), tag)
  }

  fun send(bytes: List<u8>) suspends throws DbError {
    this.stream.write(bytes) catch (e) {
      throw transport(e)
    }
  }

  // the next message that is not a notice or a parameter change
  fun receive(): Message suspends throws DbError {
    loop {
      val m = readMessage(this.stream) catch (e) {
        if (e is DbError) throw e
        throw DbError(kind: ErrorKind.Connection, text: e.message())
      }
      if (m.kind != msgNotice && m.kind != msgParameterStatus && m.kind != msgNotification) return m
    }
  }
}

fun readColumns(c: *Cursor): List<Column> throws DbError {
  val n = try c.i16()
  val out: MutableList<Column> = []
  loop (_ in 0..<n) {
    val name = try c.cstring()
    val _ = try c.i32()  // the table
    val _ = try c.i16()  // the column in it
    val oid = try c.i32()
    val _ = try c.i16()  // size
    val _ = try c.i32()  // modifier
    val _ = try c.i16()  // format
    out.push(Column(name, oid))
  }
  out.toList()
}

fun readRow(c: *Cursor): List<string?> throws DbError {
  val n = try c.i16()
  val out: MutableList<string?> = []
  loop (_ in 0..<n) {
    val len = try c.i32()
    if (len < 0) {
      out.push(null)
    } else {
      val bytes = try c.take(len)
      out.push(bytes.decodeUtf8() ?: throw DbError(kind: ErrorKind.Protocol, text: "the server sent a value that is not UTF-8"))
    }
  }
  out.toList()
}

// ---- the login ----

fun connect(target: Target): Conn suspends throws DbError {
  val socket = net.connect(target.host, target.port) catch (e) {
    throw transport(e)
  }
  var stream: io.Stream = socket
  if (target.tls != Tls.Off) {
    stream = try secure(socket, target)
  }
  val out = Out()
  out.i32(196608)  // protocol 3.0
  out.cstring("user")
  out.cstring(target.user)
  out.cstring("database")
  out.cstring(target.database)
  out.cstring("application_name")
  out.cstring(target.application)
  out.cstring("client_encoding")
  out.cstring("UTF8")
  out.byte(0)
  val startup: MutableList<u8> = []
  startup.pushU32Be((out.body.len() + 4).wrapU32())
  startup.addAll(out.body.toList())
  val conn = Conn(stream, target: target.shown())
  try login(conn, target, startup.toList()) catch (e) {
    conn.close()
    throw e
  }
  conn
}

// the SSLRequest: eight bytes, answered with one — S to go on with TLS, N for no
fun secure(socket: net.Conn, target: Target): io.Stream suspends throws DbError {
  val request: List<u8> = [0, 0, 0, 8, 4, 210, 22, 47]
  socket.write(request) catch (e) {
    throw transport(e)
  }
  val answer = socket.readExact(1) catch (e) {
    throw transport(e)
  }
  if (answer.at(0) != 83) {
    socket.close()
    throw DbError(kind: ErrorKind.Connection, text: "the server does not offer TLS; connect with ?sslmode=disable if the network is trusted")
  }
  var roots = ""
  if (!target.rootCert.isEmpty()) {
    roots = fs.readFile(target.rootCert) catch (e) {
      throw DbError(kind: ErrorKind.Config, text: "cannot read sslrootcert: ${e.message()}")
    }
  }
  val options = tls.Options(roots, dangerouslyAcceptAnyCertificate: target.tls == Tls.Insecure)
  tls.client(socket, target.host, options, address: target.shown()) catch (e) {
    throw transport(e)
  }
}

fun login(conn: Conn, target: Target, startup: List<u8>) suspends throws DbError {
  try conn.send(startup)
  var scram: Scram? = null
  var expected: List<u8> = []
  loop {
    val m = try conn.receive()
    if (m.kind == msgError) {
      val e = try errorOf(m)
      throw DbError(kind: if (e.code.startsWith("28")) ErrorKind.Auth else ErrorKind.Server, text: e.text, code: e.code, detail: e.detail, hint: e.hint, severity: e.severity)
    } else if (m.kind == msgAuth) {
      var c = m.cursor()
      val code = try c.i32()
      if (code == 0) {
        // authenticated; what follows is parameter status and the key, up to ReadyForQuery
      } else if (code == 3) {
        if (target.tls == Tls.Off && !isLoopback(target.host)) {
          throw DbError(kind: ErrorKind.Auth, text: "the server asks for the password in clear text over a connection that is not encrypted")
        }
        val reply = Out()
        reply.cstring(target.password.expose())
        try conn.send(reply.frame(112))
      } else if (code == 10) {
        // the mechanisms the server offers, a list of names ended by an empty one
        var offered = false
        loop {
          val name = try c.cstring()
          if (name.isEmpty()) break
          if (name == "SCRAM-SHA-256") offered = true
        }
        if (!offered) throw DbError(kind: ErrorKind.Auth, text: "the server offers no SASL mechanism this driver speaks (SCRAM-SHA-256)")
        val nonce = base64.encode(crypto.randomBytes(18))
        val s = Scram(user: "", nonce, password: target.password)
        val first = s.first().bytes()
        scram = s
        val reply = Out()
        reply.cstring("SCRAM-SHA-256")
        reply.i32(first.len())
        reply.put(first)
        try conn.send(reply.frame(112))
      } else if (code == 11) {
        val s = scram ?: throw DbError(kind: ErrorKind.Protocol, text: "the server continued a SASL exchange that was not begun")
        val challenge = c.rest().decodeUtf8() ?: ""
        val (answer, signature) = s.answer(challenge) ?: throw DbError(kind: ErrorKind.Auth, text: "the server's SCRAM challenge is not valid")
        expected = signature
        val reply = Out()
        reply.put(answer.bytes())
        try conn.send(reply.frame(112))
      } else if (code == 12) {
        val s = scram ?: throw DbError(kind: ErrorKind.Protocol, text: "the server finished a SASL exchange that was not begun")
        if (!s.verify(c.rest().decodeUtf8() ?: "", expected)) {
          throw DbError(kind: ErrorKind.Auth, text: "the server could not prove it knows the password (its SCRAM signature is wrong)")
        }
      } else if (code == 5) {
        throw DbError(kind: ErrorKind.Auth, text: "the server asks for an MD5 password; use scram-sha-256 (password_encryption) on the server")
      } else {
        throw DbError(kind: ErrorKind.Auth, text: "the server asks for an authentication method this driver does not speak (code $code)")
      }
    } else if (m.kind == msgReady) {
      conn.inTransaction.store((m.body.at(0) ?: 73) != 73)
      return
    }
  }
}
