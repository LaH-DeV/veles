/// TCP sockets on the task executor. A `Listener` accepts connections, a
/// `Conn` reads and writes bytes; every call that has to wait — for a
/// client, for data, for room in the send buffer — suspends the task and
/// lets the others run, so one thread serves many connections (D2, D16).
/// Failures are thrown as `IoError`, with the address in `path`.
///
/// ```veles
/// val listener = try net.listen(port: 8080)
/// loop {
///   val conn = try listener.accept()
///   async serve(conn)
/// }
/// ```
///
/// Implemented on top of runtime/c/veles_net.c; every extern call is
/// confined to one `unsafe` block (D44). The sockets are non-blocking: a
/// call that would block returns 1, the task parks with `await ioWait(fd,
/// write)` until the executor's poll sees the socket ready, and retries.
use os

extern "C" {
  fun veles_net_listen(host: string, port: i64, fd: *raw i64): i64
  fun veles_net_port(fd: i64): i64
  fun veles_net_accept(fd: i64, conn: *raw i64): i64
  fun veles_net_connect(host: string, port: i64, fd: *raw i64): i64
  fun veles_net_connect_result(fd: i64): i64
  fun veles_net_recv(fd: i64, max: i64, out: *raw string): i64
  fun veles_net_send(fd: i64, bytes: List<u8>, offset: i64, sent: *raw i64): i64
  fun veles_net_close(fd: i64)
  fun veles_net_shutdown_write(fd: i64): i64
  fun veles_net_peer(fd: i64, out: *raw string)
}

/// The runtime's "would block" answer: wait and retry.
val wouldBlock: i64 = 1

/// A read that would have exceeded the ceiling its caller gave. The bytes
/// read so far are dropped and the connection is left where it stood, so
/// there is nothing sensible to resume from: close it. A server answers
/// the peer first — 431 for headers, 413 for a body — and then closes.
public error TooLong {
  public message: string
  /// The ceiling that was passed, in bytes.
  public limit: i64
}

/// Listens for TCP connections on `host:port`. `host` empty means every
/// interface; `port` 0 lets the system pick one (read it back with
/// `port()`).
public fun listen(host: string = "127.0.0.1", port: i64 = 0): Listener throws IoError {
  var fd: i64 = 0
  val code = unsafe {
    veles_net_listen(host, port, &fd)
  }
  if (code != 0) throw os.ioError(code, "$host:$port")
  Listener(fd, address: "$host:$port")
}

/// Connects to `host:port`; suspends until the connection is up.
public fun connect(host: string, port: i64): Conn suspends throws IoError {
  var fd: i64 = 0
  var code = unsafe {
    veles_net_connect(host, port, &fd)
  }
  if (code == wouldBlock) {
    await ioWait(fd, true)
    code = unsafe {
      veles_net_connect_result(fd)
    }
  }
  if (code != 0) {
    unsafe {
      veles_net_close(fd)
    }
    throw os.ioError(code, "$host:$port")
  }
  Conn(fd, address: "$host:$port")
}

/// A listening socket. Close it with `with` or `close()`.
public struct Listener {
  fd:      i64
  address: string

  /// The port the listener is bound to — the system's choice when `listen`
  /// was asked for port 0.
  public fun port(): i64 = unsafe {
    veles_net_port(self.fd)
  }

  /// The next connection; suspends until a client arrives.
  public fun accept(): Conn suspends throws IoError {
    loop {
      var fd: i64 = 0
      val code = unsafe {
        veles_net_accept(self.fd, &fd)
      }
      if (code == 0) return Conn(fd, address: peerOf(fd))
      if (code != wouldBlock) throw os.ioError(code, self.address)
      await ioWait(self.fd, false)
    }
  }

  implement Closeable {
    fun close() {
      unsafe {
        veles_net_close(self.fd)
      }
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

// The read-ahead buffer of a connection. It sits in a Mutex so that a
// Conn is Sendable — `withTimeout(ms, () => try conn.readLine())` hands
// the connection to another task — and every access to it is a short,
// non-suspending critical section; the socket reads happen outside.
fun newBuffer(): Mutex<MutableList<u8>> {
  val empty: MutableList<u8> = []
  mutex(empty)
}

fun peerOf(fd: i64): string {
  var out = ""
  unsafe {
    veles_net_peer(fd, &out)
  }
  out
}

/// One TCP connection. Reads hand out whatever has arrived; `readLine`
/// and `readExact` buffer on top of that.
public struct Conn {
  fd:      i64
  address: string
  buffer:  Mutex<MutableList<u8>> = newBuffer()

  /// The peer's address, `host:port`.
  public fun peer(): string = self.address

  /// Up to `max` bytes, as soon as any are available; an empty list means
  /// the peer closed its side.
  public fun read(max: i64 = 65536): List<u8> suspends throws IoError {
    val buffered = self.take(max)
    if (!buffered.isEmpty()) return buffered
    loop {
      var data = ""
      val code = unsafe {
        veles_net_recv(self.fd, max, &data)
      }
      if (code == 0) return data.bytes()
      if (code != wouldBlock) throw os.ioError(code, self.address)
      await ioWait(self.fd, false)
    }
  }

  /// Exactly `n` bytes, or fewer when the peer closes first. `n` is the
  /// ceiling as well as the count, so a caller that takes it from the peer
  /// — a `Content-Length` header, a length prefix — must check it against
  /// its own limit before calling, or the peer chooses the allocation.
  public fun readExact(n: i64): List<u8> suspends throws IoError {
    loop (self.buffered() < n) {
      val chunk = try self.fetch()
      if (chunk.isEmpty()) break
    }
    self.take(n)
  }

  /// The next line as text, without its `\n` (and a `\r` before it), or
  /// `null` when the peer closed with nothing left; a line that is not
  /// valid UTF-8 is an error.
  ///
  /// `max` is how many bytes of one line this caller is willing to hold.
  /// It has no default on purpose: the peer decides where the newline
  /// goes, so a line is only as long as the reader allows, and a program
  /// that never says allows a stranger to fill its memory. A line that
  /// reaches the ceiling throws `TooLong` and the connection is spent.
  public fun readLine(max: i64): string? suspends throws IoError | TooLong {
    var scanned: i64 = 0
    loop {
      val nl = self.buffer.withLock(b => findByte(b, 10, from: scanned))
      if (nl >= 0) {
        if (nl > max) throw self.tooLong("line", max)
        val line = self.buffer.withLock(b => {
          var end = nl
          if (end > 0 && b.at(end - 1) == 13) end -= 1
          val line = b.take(end)
          b.removePrefix(nl + 1)
          line
        })
        return line.decodeUtf8() ?: throw IoError(path: self.address, code: 0, detail: "line is not valid UTF-8")
      }
      scanned = self.buffered()
      // no newline in what has arrived: stop before asking for more, or a
      // peer that never sends one decides how much we allocate
      if (scanned > max) throw self.tooLong("line", max)
      val chunk = try self.fetch()
      if (chunk.isEmpty()) {
        val rest = self.take(scanned)
        if (rest.isEmpty()) return null
        if (rest.len() > max) throw self.tooLong("line", max)
        return rest.decodeUtf8() ?: throw IoError(path: self.address, code: 0, detail: "line is not valid UTF-8")
      }
    }
  }

  // the refusal, with the address so a log says which peer it was
  fun tooLong(what: string, max: i64): TooLong =
    TooLong(message: "$what from ${self.address} is longer than $max bytes", limit: max)

  /// Sends all of `bytes`; suspends while the peer catches up.
  public fun write(bytes: List<u8>) suspends throws IoError {
    var offset: i64 = 0
    loop (offset < bytes.len()) {
      var sent: i64 = 0
      val code = unsafe {
        veles_net_send(self.fd, bytes, offset, &sent)
      }
      if (code == wouldBlock) {
        await ioWait(self.fd, true)
        continue
      }
      if (code != 0) throw os.ioError(code, self.address)
      offset += sent
    }
  }

  /// Sends `text` as UTF-8.
  public fun writeText(text: string) suspends throws IoError {
    try self.write(text.bytes())
  }

  /// Tells the peer that nothing more will be sent (its reads see the end
  /// of the stream) while this side keeps reading — how a client marks the
  /// end of a request when the protocol has no other way to say so.
  public fun shutdownWrite() throws IoError {
    val code = unsafe {
      veles_net_shutdown_write(self.fd)
    }
    if (code != 0) throw os.ioError(code, self.address)
  }

  implement Closeable {
    fun close() {
      unsafe {
        veles_net_close(self.fd)
      }
    }
  }

  // one recv into the buffer; the chunk (empty at end of stream)
  fun fetch(): List<u8> suspends throws IoError {
    loop {
      var data = ""
      val code = unsafe {
        veles_net_recv(self.fd, 65536, &data)
      }
      if (code == 0) {
        val chunk = data.bytes()
        self.buffer.withLock(b => b.addAll(chunk))
        return chunk
      }
      if (code != wouldBlock) throw os.ioError(code, self.address)
      await ioWait(self.fd, false)
    }
  }

  // how many bytes are buffered
  fun buffered(): i64 = self.buffer.withLock(b => b.len())

  // takes up to n buffered bytes
  fun take(n: i64): List<u8> = self.buffer.withLock(b => {
    val got = n.min(b.len())
    val out = b.take(got)
    b.removePrefix(got)
    out
  })
}

extend<T> MutableList<T> {
  // drops the first n elements
  fun removePrefix(n: i64) {
    if (n <= 0) return
    val rest = self.drop(n)
    self.clear()
    self.addAll(rest)
  }
}
