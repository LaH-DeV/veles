/// TLS, over any `io.Stream` (D128). `connect` opens a TCP connection and
/// runs the handshake; the `Conn` it returns is an `io.Stream` like a
/// `net.Conn`, so code written for one — `http` included — runs over the
/// other.
///
/// ```veles
/// use tls
///
/// with conn = try tls.connect("example.com", 443)
/// try conn.writeText("GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")
/// val head = try conn.readLine(8192)
/// ```
///
/// The certificate is checked against the system's trusted roots and the
/// host name, always; there is no way to turn it off but the option that
/// says what it is, `dangerouslyAcceptAnyCertificate`. TLS 1.2 is the
/// minimum, 1.3 is used where the system has it.
///
/// Implemented on top of runtime/c/veles_tlsio.c: SChannel on Windows,
/// OpenSSL elsewhere. The C side is a state machine that never touches a
/// socket — it is given the bytes that arrived and hands back the bytes to
/// send — so every call there is short and the waiting happens here, where
/// a task can be parked.
use fs, io, log as logs { field }, net, os

extern "C" {
  fun veles_tls_client(host: string, roots: string, alpn: string, insecure: i64, handle: *raw i64, err: *raw string): i64
  fun veles_tls_feed(h: i64, bytes: List<u8>)
  fun veles_tls_take(h: i64, out: *raw string)
  fun veles_tls_pending(h: i64): i64
  fun veles_tls_handshake(h: i64): i64
  fun veles_tls_read(h: i64, max: i64, out: *raw string): i64
  fun veles_tls_write(h: i64, bytes: List<u8>, offset: i64): i64
  fun veles_tls_close_notify(h: i64)
  fun veles_tls_error(h: i64, out: *raw string)
  fun veles_tls_alpn(h: i64, out: *raw string)
  fun veles_tls_free(h: i64)
  fun veles_tls_server_creds(chain: string, key: string, alpn: string, handle: *raw i64, err: *raw string): i64
  fun veles_tls_server_creds_free(h: i64)
  fun veles_tls_accept(creds: i64, handle: *raw i64, err: *raw string): i64
}

// the engine's answers (veles_tlsio.c)
const done: i64 = 0
const needInput: i64 = 1
const peerClosed: i64 = 2

/// How a connection is made. Everything has a safe default.
public struct Options {
  /// PEM text of the certificates to trust *instead of* the system's: a
  /// private authority, or a server's own certificate in a test. Empty
  /// means the system's trusted roots.
  public roots: string = ""
  /// The name the certificate must be valid for, and the one sent for SNI.
  /// Empty means the host that was dialled.
  public serverName: string = ""
  /// The application protocols to offer (ALPN), most preferred first.
  public alpn: List<string> = []
  /// Skips every check of the server's certificate: anyone on the path can
  /// then read and change the traffic. Only for talking to a throwaway
  /// local server.
  public dangerouslyAcceptAnyCertificate: bool = false
}

// a TLS failure as the IoError every stream call throws; `detail` is the
// engine's reason (a certificate that does not match, an alert, ...)
fun failure(address: string, detail: string): IoError =
  IoError(path: address, code: 0, detail: "tls: $detail", kind: IoKind.Other)

/// Dials `host:port`, runs the handshake and returns the secured connection;
/// suspends until it is up. A certificate that is not valid for `host`, a
/// protocol the server does not speak or a connection cut during the
/// handshake throw `IoError`.
public fun connect(host: string, port: i64, options: Options = Options()): Conn suspends throws IoError {
  val socket = try net.connect(host, port)
  val name = if (options.serverName.isEmpty()) host else options.serverName
  try client(socket, name, options, address: "$host:$port")
}

/// Runs the client handshake over a stream that is already connected — a
/// proxy tunnel, a test pipe — and returns the secured connection, which
/// owns `stream` from here on: closing the `Conn` closes it, and so does a
/// failed handshake.
public fun client(stream: io.Stream, serverName: string, options: Options = Options(), address: string = ""): Conn suspends throws IoError {
  val label = if (address.isEmpty()) serverName else address
  var handle: i64 = 0
  var reason = ""
  val alpn = options.alpn.join(",")
  // SAFETY: the strings are read within their lengths and not kept; the handle and the
  // reason go into locals that outlive the call
  val code = unsafe {
    veles_tls_client(serverName, options.roots, alpn, if (options.dangerouslyAcceptAnyCertificate) 1 else 0, &handle, &reason)
  }
  if (code != done) {
    stream.close()
    throw failure(label, reason)
  }
  val conn = Conn(inner: stream, session: Session(number: handle), address: label)
  try conn.handshake() catch (e) {
    conn.close()
    throw e
  }
  conn
}

// The engine's handle, shared by every copy of the Conn. Closing frees it
// exactly once; a call that races the close finds the handle gone and
// fails (the table never reuses a handle, so a stale one cannot reach
// another session).
struct Session {
  number:        i64
  private freed: Atomic<bool>

  init(number: i64) {
    this.number = number
    this.freed = Atomic(value: false)
  }

  fun free() {
    if (this.freed.swap(true)) return
    // SAFETY: the swap lets exactly one caller through; the engine refuses a handle it no longer has
    unsafe {
      veles_tls_free(this.number)
    }
  }
}

// index of the first `b` at or after `from`, or -1
fun findByte(xs: *MutableList<u8>, b: u8, from: i64): i64 {
  loop (i in from..<xs.len()) {
    if (xs.at(i) == b) return i
  }
  -1
}

fun newBuffer(): Mutex<MutableList<u8>> {
  val empty: MutableList<u8> = []
  Mutex(value: empty)
}

/// A TLS connection. Reads hand out whatever has arrived and been decrypted;
/// `readLine` and `readExact` buffer on top of that. One task may read while
/// another writes.
public struct Conn {
  inner:   io.Stream
  session: Session
  address: string
  // plaintext read ahead for readLine/readExact, as in net.Conn
  buffer: Mutex<MutableList<u8>> = newBuffer()
  // one task at a time turns bytes into records and sends them, or the
  // records could reach the peer out of order
  sendLock: Semaphore = Semaphore(permits: 1)
  // a server's connection runs its handshake on first use, in the task that
  // serves it (see Listener); `gate` makes the tasks that arrive together wait for it
  pending: Atomic<bool> = Atomic(value: false)
  gate:    Semaphore = Semaphore(permits: 1)

  /// The application protocol the two sides agreed on (ALPN), or `null`
  /// when none was offered or chosen.
  public fun protocol(): string? {
    var name = ""
    // SAFETY: stores the name in `name`, a local that outlives the call
    unsafe {
      veles_tls_alpn(this.session.number, &name)
    }
    if (name.isEmpty()) null else name
  }

  /// The peer's address, `host:port`.
  public fun peer(): string = this.address

  fun reason(): IoError {
    var text = ""
    // SAFETY: stores the reason in `text`, a local that outlives the call
    unsafe {
      veles_tls_error(this.session.number, &text)
    }
    failure(this.address, text)
  }

  // sends the records the engine has queued; always called after a step
  // that may have produced some
  fun flush() suspends throws IoError {
    with permit = this.sendLock.acquire()
    var out = ""
    // SAFETY: stores the queued bytes in `out`, a local that outlives the call
    unsafe {
      veles_tls_take(this.session.number, &out)
    }
    if (!out.isEmpty()) try this.inner.write(out.bytes())
  }

  fun flushIfPending() suspends throws IoError {
    // SAFETY: a query on the handle; a freed one answers 0
    val pending = unsafe {
      veles_tls_pending(this.session.number)
    }
    if (pending != 0) try this.flush()
  }

  // the bytes that arrived from the peer go to the engine
  fun feed() suspends throws IoError {
    val chunk = try this.inner.read()
    if (chunk.isEmpty()) {
      throw failure(this.address, "the connection was closed without a close_notify, so the data may be cut short")
    }
    // SAFETY: reads `chunk` within its length and keeps nothing
    unsafe {
      veles_tls_feed(this.session.number, chunk)
    }
  }

  // the handshake of a connection that has not had one yet
  fun ensureHandshake() suspends throws IoError {
    if (!this.pending.load()) return
    with permit = this.gate.acquire()
    if (!this.pending.load()) return
    try this.handshake()
    this.pending.store(false)
  }

  fun handshake() suspends throws IoError {
    loop {
      // SAFETY: a step on the handle; a freed one fails
      val r = unsafe {
        veles_tls_handshake(this.session.number)
      }
      try this.flush()
      if (r == done) return
      if (r != needInput) throw this.reason()
      try this.feed() catch (e) {
        throw failure(this.address, "the connection was closed during the handshake (${e.message()})")
      }
    }
  }

  // one decrypt: the bytes, or an empty list at the peer's close_notify
  fun fetch(max: i64): List<u8> suspends throws IoError {
    loop {
      var data = ""
      // SAFETY: stores the plaintext in `data`, a local that outlives the call
      val r = unsafe {
        veles_tls_read(this.session.number, max, &data)
      }
      if (r == done) {
        try this.flushIfPending()
        return data.bytes()
      }
      if (r == peerClosed) {
        try this.flushIfPending()
        return []
      }
      if (r != needInput) throw this.reason()
      try this.flushIfPending()
      try this.feed()
    }
  }

  fun tooLong(what: string, max: i64): io.TooLong =
    io.TooLong(message: "$what from ${this.address} is longer than $max bytes", limit: max)

  implement io.Stream {
    /// Up to `max` bytes, as soon as any are available; an empty list means
    /// the peer ended the session (close_notify). A connection that is cut
    /// without it throws, so a truncated body is never taken for a whole one.
    fun read(max: i64 = 65536): List<u8> suspends throws IoError {
      try this.ensureHandshake()
      val buffered = this.take(max)
      if (!buffered.isEmpty()) return buffered
      try this.fetch(max)
    }

    /// Exactly `n` bytes, or fewer when the peer ends the session first.
    /// `n` is the ceiling as well as the count: check a length the peer sent
    /// against your own limit before calling.
    fun readExact(n: i64): List<u8> suspends throws IoError {
      try this.ensureHandshake()
      loop (this.buffered() < n) {
        val chunk = try this.fetch(65536)
        if (chunk.isEmpty()) break
        this.buffer.withLock(b => b.addAll(chunk))
      }
      this.take(n)
    }

    /// The next line as text, without its `\n` (and a `\r` before it), or
    /// `null` when the peer ended the session with nothing left. `max` is how
    /// many bytes of one line the caller will hold; a longer line throws
    /// `TooLong` and the connection is spent.
    fun readLine(max: i64): string? suspends throws IoError | io.TooLong {
      try this.ensureHandshake()
      var scanned: i64 = 0
      loop {
        val nl = this.buffer.withLock(b => findByte(b, 10, from: scanned))
        if (nl >= 0) {
          if (nl > max) throw this.tooLong("line", max)
          val line = this.buffer.withLock(b => {
            var end = nl
            if (end > 0 && b.at(end - 1) == 13) end -= 1
            val line = b.take(end)
            b.dropFront(nl + 1)
            line
          })
          return line.decodeUtf8() ?: throw IoError(path: this.address, code: 0, detail: "line is not valid UTF-8", kind: IoKind.InvalidData)
        }
        scanned = this.buffered()
        if (scanned > max) throw this.tooLong("line", max)
        val chunk = try this.fetch(65536)
        if (chunk.isEmpty()) {
          val rest = this.take(scanned)
          if (rest.isEmpty()) return null
          if (rest.len() > max) throw this.tooLong("line", max)
          return rest.decodeUtf8() ?: throw IoError(path: this.address, code: 0, detail: "line is not valid UTF-8", kind: IoKind.InvalidData)
        }
        this.buffer.withLock(b => b.addAll(chunk))
      }
    }

    /// Encrypts and sends all of `bytes`; suspends while the peer catches up.
    fun write(bytes: List<u8>) suspends throws IoError {
      try this.ensureHandshake()
      with permit = this.sendLock.acquire()
      // SAFETY: reads `bytes` within its length and keeps nothing
      val code = unsafe {
        veles_tls_write(this.session.number, bytes, 0)
      }
      if (code != done) throw this.reason()
      var out = ""
      // SAFETY: stores the records in `out`, a local that outlives the call
      unsafe {
        veles_tls_take(this.session.number, &out)
      }
      if (!out.isEmpty()) try this.inner.write(out.bytes())
    }

    /// Sends `text` as UTF-8.
    fun writeText(text: string) suspends throws IoError {
      try this.write(text.bytes())
    }

    /// Sends close_notify and half-closes the underlying stream: the peer's
    /// reads end cleanly, and this side keeps reading.
    fun shutdownWrite() suspends throws IoError {
      try this.ensureHandshake()
      // SAFETY: queues the alert on the handle; a freed one does nothing
      unsafe {
        veles_tls_close_notify(this.session.number)
      }
      try this.flush()
      try this.inner.shutdownWrite()
    }
  }

  implement Closeable {
    /// Ends the connection at once: the session is dropped and the stream
    /// underneath closed. It sends no close_notify, because closing cannot
    /// wait for the peer — call `shutdownWrite()` first for a clean end.
    fun close() {
      this.session.free()
      this.inner.close()
    }
  }

  // takes up to n buffered bytes
  fun take(n: i64): List<u8> = this.buffer.withLock(b => {
    val got = n.min(b.len())
    val out = b.take(got)
    b.dropFront(got)
    out
  })

  fun buffered(): i64 = this.buffer.withLock(b => b.len())
}

// ---------------------------------------------------------------------------
// serving

// The engine's credentials for a certificate and its key. A Certificate can
// be replaced while connections are accepted (see CertificateReloader), so
// the number lives in an Atomic shared by every copy; each new connection
// takes whatever is current, and one already accepted keeps the credentials
// it started with.
struct Credentials {
  number: Atomic<i64>
}

/// A server's certificate chain and its private key, ready to accept
/// connections with. Load it once and give it to `listen`, or keep it
/// current with `reloading`. Closing it releases the key; connections
/// accepted before keep working.
///
/// ```veles
/// with cert = try tls.Certificate.load("server.pem", "server.key")
/// with listener = try tls.listen(cert: cert, port: 8443)
/// ```
public struct Certificate {
  private creds: Credentials

  /// Reads the chain (the server's certificate first, then the authorities
  /// that vouch for it) and the private key from PEM files. The key is read
  /// into a `Secret`. `alpn` lists the application protocols the server
  /// speaks, most preferred first.
  public static fun load(certPath: string, keyPath: string, alpn: List<string> = []): Certificate throws IoError {
    val chain = try fs.readFile(certPath)
    val key = Secret.of(try fs.readFile(keyPath))
    try Certificate.fromPem(chain, key, alpn)
  }

  /// A certificate from the PEM text of its chain and of its key. RSA keys
  /// and ECDSA keys on P-256, P-384 and P-521 are accepted, in PKCS#8,
  /// PKCS#1 or SEC1 form; an encrypted key is not.
  public static fun fromPem(chain: string, key: Secret<string>, alpn: List<string> = []): Certificate throws IoError {
    val number = try makeCredentials(chain, key, alpn)
    Certificate(creds: Credentials(number: Atomic(value: number)))
  }

  /// Replaces the chain and key for the connections accepted from now on;
  /// the ones already open are not touched. A failure leaves the old
  /// certificate in place.
  public fun replace(chain: string, key: Secret<string>, alpn: List<string> = []) throws IoError {
    val number = try makeCredentials(chain, key, alpn)
    freeCredentials(this.creds.number.swap(number))
  }

  // the engine's handle of the credentials now in force; 0 once closed
  fun current(): i64 = this.creds.number.load()

  implement Closeable {
    /// Releases the key. A listener that still uses this certificate
    /// refuses connections from now on.
    fun close() {
      freeCredentials(this.creds.number.swap(0))
    }
  }
}

fun makeCredentials(chain: string, key: Secret<string>, alpn: List<string>): i64 throws IoError {
  var handle: i64 = 0
  var reason = ""
  val protocols = alpn.join(",")
  // SAFETY: the strings are read within their lengths and not kept; the handle and the
  // reason go into locals that outlive the call
  val code = unsafe {
    veles_tls_server_creds(chain, key.expose(), protocols, &handle, &reason)
  }
  if (code != done) throw failure("", reason)
  handle
}

fun freeCredentials(number: i64) {
  if (number == 0) return
  // SAFETY: the number came out of the swap, so exactly one caller frees it
  unsafe {
    veles_tls_server_creds_free(number)
  }
}

/// Accepts TCP connections and secures each with a `Certificate`. The TLS
/// handshake happens on the connection's first read or write, in the task that
/// serves it — one client that is slow to start it cannot hold up the accept
/// loop.
public struct Listener {
  inner: net.Listener
  cert:  Certificate

  /// The port the listener is bound to — the system's choice when `listen`
  /// was asked for port 0.
  public fun port(): i64 = this.inner.port()

  /// The next connection; suspends until a client arrives.
  public fun accept(): Conn suspends throws IoError {
    val socket = try this.inner.accept()
    try serverConn(socket, this.cert)
  }

  implement Closeable {
    /// Stops listening; the certificate stays open.
    fun close() {
      this.inner.close()
    }
  }
}

/// Listens for TLS connections on `host:port`. `host` empty means every
/// interface; `port` 0 lets the system pick one.
public fun listen(cert: Certificate, host: string = "127.0.0.1", port: i64 = 0): Listener throws IoError {
  Listener(inner: try net.listen(host, port), cert)
}

/// Secures the connections of a listener that already exists.
public fun wrap(listener: net.Listener, cert: Certificate): Listener = Listener(inner: listener, cert)

/// Secures a connection that was accepted elsewhere — what `http.serve(tls:)`
/// does with each one. The handshake runs on its first read or write; the
/// `Conn` owns `stream`, and a failure here closes it.
public fun accept(stream: io.Stream, cert: Certificate, address: string = ""): Conn throws IoError = try serverConn(stream, cert, address)

// a session for a connection the listener accepted; its handshake is still to run
fun serverConn(stream: io.Stream, cert: Certificate, address: string = ""): Conn throws IoError {
  loop (_ in 0..<3) {
    val number = cert.current()
    var handle: i64 = 0
    var reason = ""
    // SAFETY: the handle and the reason go into locals that outlive the call
    val code = unsafe {
      veles_tls_accept(number, &handle, &reason)
    }
    if (code == done) {
      return Conn(inner: stream, session: Session(number: handle), address, pending: Atomic(value: true))
    }
    // the certificate may have been replaced between the load and the call
    if (cert.current() != number) continue
    stream.close()
    throw failure(address, reason)
  }
  stream.close()
  throw failure(address, "the certificate keeps changing under the listener")
}

/// Keeps a `Certificate` current: every `every` it reads the two files again
/// and, if they changed, replaces the certificate for new connections. A read
/// that fails (a file half written by a renewal) keeps the old one and logs.
/// It holds a task, so it is `with`-bound or returned (D111); closing it stops
/// the task and closes the certificate.
///
/// ```veles
/// with certs = try tls.reloading("server.pem", "server.key", every: Duration.minutes(10))
/// with listener = try tls.listen(cert: certs.certificate(), port: 8443)
/// ```
public struct CertificateReloader {
  cert: Certificate
  task: Task<()>

  /// The certificate this reloader keeps current; give it to `listen`.
  public fun certificate(): Certificate = this.cert

  implement Closeable {
    fun close() {
      this.cert.close()
    }
  }
}

/// Loads the certificate and starts keeping it current; see
/// `CertificateReloader`. A first load that fails throws.
public fun reloading(certPath: string, keyPath: string, every: Duration, alpn: List<string> = []): CertificateReloader throws IoError {
  val chain = try fs.readFile(certPath)
  val key = try fs.readFile(keyPath)
  val cert = try Certificate.fromPem(chain, Secret.of(key), alpn)
  CertificateReloader(cert, task: async watch(cert, certPath, keyPath, chain + "\u{0}" + key, every, alpn))
}

fun watch(cert: Certificate, certPath: string, keyPath: string, seen: string, every: Duration, alpn: List<string>) suspends {
  var last = seen
  loop {
    await sleep(every)
    val chain = fs.readFile(certPath) catch (e) {
      logs.warn("certificate reload: cannot read", field("path", certPath), field("error", e.message()))
      continue
    }
    val key = fs.readFile(keyPath) catch (e) {
      logs.warn("certificate reload: cannot read", field("path", keyPath), field("error", e.message()))
      continue
    }
    val now = chain + "\u{0}" + key
    if (now == last) continue
    cert.replace(chain, Secret.of(key), alpn) catch (e) {
      logs.warn("certificate reload: the new files were refused, keeping the old certificate", field("error", e.message()))
      continue
    }
    last = now
    logs.info("certificate reloaded", field("path", certPath))
  }
}

extend<T> MutableList<T> {
  // drops the first n elements
  fun dropFront(n: i64) {
    if (n <= 0) return
    val rest = this.drop(n)
    this.clear()
    this.addAll(rest)
  }
}
