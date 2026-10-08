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
use io
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
  fun veles_io_closing(fd: i64)
  fun veles_io_closed(fd: i64)
}

/// The runtime's "would block" answer: wait and retry.
const wouldBlock: i64 = 1

// A host holding a NUL byte would reach the resolver cut short at it:
// `evil.example\0.trusted.example` passes an `endsWith` allow-list check
// and then connects to `evil.example`. Refused instead.
fun checkHost(host: string) throws IoError {
  if (host.contains("\u{0}")) {
    throw IoError(path: host.replace("\u{0}", "\\0"), code: 22, detail: "a host name cannot hold a NUL byte", kind: IoKind.InvalidInput)
  }
}

/// Listens for TCP connections on `host:port`. `host` empty means every
/// interface; `port` 0 lets the system pick one (read it back with
/// `port()`).
public fun listen(host: string = "127.0.0.1", port: i64 = 0): Listener throws IoError {
  try checkHost(host)
  var fd: i64 = 0
  // SAFETY: takes `host` by value and stores the new socket's number in `fd`,
  // a local that outlives the call
  val code = unsafe {
    veles_net_listen(host, port, &fd)
  }
  if (code != 0) throw os.ioError(code, "$host:$port")
  Listener(fd: Socket(fd), address: "$host:$port")
}

/// Connects to `host:port`; suspends until the connection is up.
public fun connect(host: string, port: i64): Conn suspends throws IoError {
  try checkHost(host)
  var fd: i64 = 0
  // SAFETY: takes `host` by value and stores the new socket's number in `fd`,
  // a local that outlives the call
  var code = unsafe {
    veles_net_connect(host, port, &fd)
  }
  if (code == wouldBlock) {
    await ioWait(fd, true)
    // SAFETY: `fd` is the socket connect just opened, which nothing else has seen
    code = unsafe {
      veles_net_connect_result(fd)
    }
  }
  if (code != 0) {
    // SAFETY: `fd` never left this function, so it is closed exactly once
    unsafe {
      veles_net_close(fd)
    }
    throw os.ioError(code, "$host:$port")
  }
  Conn(fd: Socket(fd), address: "$host:$port")
}

// A socket's descriptor, shared by every copy of the Listener or Conn that
// owns it, with a count of the operations using it (Go's fdMutex).
//
// Systems hand a freed number straight to the next socket they open, so
// the descriptor may be closed only when nothing is about to use its
// number: a read that had loaded it on one thread would otherwise read
// the next connection's bytes. So every operation holds the socket for as
// long as it touches it — its system calls and its waits — with
// `with held = this.fd.using()`. `close()` marks the socket closing and
// wakes the operations parked on it; each retries, sees -1 from
// `held.fd()`, fails and lets go; and the last to let go (or `close`,
// when none held it) closes the descriptor, exactly once.
struct Socket {
  number: i64
  // (operations holding the socket) * 2, plus 1 once it is closing
  private state: Atomic<i64>
  // the descriptor has been closed; set by whoever closed it
  private released: Atomic<bool>

  init(fd: i64) {
    this.number = fd
    this.state = Atomic(value: 0)
    this.released = Atomic(value: false)
  }

  // an operation's hold, given back when the `with` ends — by a return, a
  // throw or the task's cancellation alike
  fun using(): SocketUse {
    this.state.update(s => s + 2)
    SocketUse(socket: this)
  }

  fun closing(): bool => this.state.load() % 2 == 1

  fun letGo() {
    if (this.state.update(s => s - 2) == 1) this.release()
  }

  fun close() {
    val s = this.state.update(s => if (s % 2 == 1) s else s + 1)
    // SAFETY: a runtime call on the number alone; it wakes the tasks parked
    // on it and keeps new ones from parking until the descriptor is closed
    unsafe {
      veles_io_closing(this.number)
    }
    if (s == 1) this.release()
  }

  fun release() {
    if (this.released.swap(true)) return
    // SAFETY: the swap lets exactly one caller through, and only once no
    // operation holds the socket, so nothing can use the number after this
    unsafe {
      veles_net_close(this.number)
      veles_io_closed(this.number)
    }
  }
}

// One operation's hold on its socket (see Socket).
struct SocketUse {
  socket: Socket

  // the descriptor to call with now: -1 once the socket is closing, which
  // every system call refuses — so a retry after `close` woke the
  // operation fails instead of waiting again
  fun fd(): i64 => if (this.socket.closing()) -1 else this.socket.number

  implement Closeable {
    fun close() {
      this.socket.letGo()
    }
  }
}

/// A listening socket. Close it with `with` or `close()`.
public struct Listener {
  fd:      Socket
  address: string

  /// The port the listener is bound to — the system's choice when `listen`
  /// was asked for port 0.
  public fun port(): i64 {
    with held = this.fd.using()
    // SAFETY: a query on the held socket; on a closed one the number is -1
    // and the call fails
    return unsafe {
      veles_net_port(held.fd())
    }
  }

  /// The next connection; suspends until a client arrives.
  public fun accept(): Conn suspends throws IoError {
    with held = this.fd.using()
    loop {
      var fd: i64 = 0
      // SAFETY: stores the accepted socket's number in `fd`, a local that outlives
      // the call; on a closed listener the number is -1 and the call fails
      val code = unsafe {
        veles_net_accept(held.fd(), &fd)
      }
      if (code == 0) return Conn(fd: Socket(fd), address: peerOf(fd))
      if (code != wouldBlock) throw os.ioError(code, this.address)
      await ioWait(held.fd(), false)
    }
  }

  implement Closeable {
    fun close() {
      this.fd.close()
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
  Mutex(value: empty)
}

fun peerOf(fd: i64): string {
  var out = ""
  // SAFETY: `fd` was just accepted; the address goes into `out`, a local that
  // outlives the call
  unsafe {
    veles_net_peer(fd, &out)
  }
  out
}

/// One TCP connection. Reads hand out whatever has arrived; `readLine`
/// and `readExact` buffer on top of that.
public struct Conn {
  fd:      Socket
  address: string
  buffer:  Mutex<MutableList<u8>> = newBuffer()

  /// The peer's address, `host:port`.
  public fun peer(): string => this.address

  // the refusal, with the address so a log says which peer it was
  fun tooLong(what: string, max: i64): io.TooLong =>
    io.TooLong(message: "$what from ${this.address} is longer than $max bytes", limit: max)

  implement io.Stream {
    /// Up to `max` bytes, as soon as any are available; an empty list means
    /// the peer closed its side.
    fun read(max: i64 = 65536): List<u8> suspends throws IoError {
      val buffered = this.take(max)
      if (!buffered.isEmpty()) return buffered
      with held = this.fd.using()
      loop {
        var data = ""
        // SAFETY: stores what arrived in `data`, a local that outlives the call; on a
        // closed connection the number is -1 and the call fails
        val code = unsafe {
          veles_net_recv(held.fd(), max, &data)
        }
        if (code == 0) return data.bytes()
        if (code != wouldBlock) throw os.ioError(code, this.address)
        await ioWait(held.fd(), false)
      }
    }

    /// Exactly `n` bytes, or fewer when the peer closes first. `n` is the
    /// ceiling as well as the count, so a caller that takes it from the peer
    /// — a `Content-Length` header, a length prefix — must check it against
    /// its own limit before calling, or the peer chooses the allocation.
    fun readExact(n: i64): List<u8> suspends throws IoError {
      loop (this.buffered() < n) {
        val chunk = try this.fetch()
        if (chunk.isEmpty()) break
      }
      this.take(n)
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
    fun readLine(max: i64): string? suspends throws IoError | io.TooLong {
      var scanned: i64 = 0
      loop {
        val nl = this.buffer.withLock(b => findByte(b, 10, from: scanned))
        if (nl >= 0) {
          if (nl > max) throw this.tooLong("line", max)
          val line = this.buffer.withLock(b => {
            var end = nl
            if (end > 0 && b.at(end - 1) == 13) end -= 1
            val line = b.take(end)
            b.removePrefix(nl + 1)
            line
          })
          return line.decodeUtf8() ?: throw IoError(path: this.address, code: 0, detail: "line is not valid UTF-8", kind: IoKind.InvalidData)
        }
        scanned = this.buffered()
        // no newline in what has arrived: stop before asking for more, or a
        // peer that never sends one decides how much we allocate
        if (scanned > max) throw this.tooLong("line", max)
        val chunk = try this.fetch()
        if (chunk.isEmpty()) {
          val rest = this.take(scanned)
          if (rest.isEmpty()) return null
          if (rest.len() > max) throw this.tooLong("line", max)
          return rest.decodeUtf8() ?: throw IoError(path: this.address, code: 0, detail: "line is not valid UTF-8", kind: IoKind.InvalidData)
        }
      }
    }

    /// Sends all of `bytes`; suspends while the peer catches up.
    fun write(bytes: List<u8>) suspends throws IoError {
      var offset: i64 = 0
      with held = this.fd.using()
      loop (offset < bytes.len()) {
        var sent: i64 = 0
        // SAFETY: reads `bytes` from `offset` within its length and stores the count in
        // `sent`, a local; on a closed connection the number is -1 and it fails
        val code = unsafe {
          veles_net_send(held.fd(), bytes, offset, &sent)
        }
        if (code == wouldBlock) {
          await ioWait(held.fd(), true)
          continue
        }
        if (code != 0) throw os.ioError(code, this.address)
        offset += sent
      }
    }

    /// Sends `text` as UTF-8.
    fun writeText(text: string) suspends throws IoError {
      try this.write(text.bytes())
    }

    /// Tells the peer that nothing more will be sent (its reads see the end
    /// of the stream) while this side keeps reading — how a client marks the
    /// end of a request when the protocol has no other way to say so.
    fun shutdownWrite() throws IoError {
      with held = this.fd.using()
      // SAFETY: on a closed connection the number is -1 and the call fails
      val code = unsafe {
        veles_net_shutdown_write(held.fd())
      }
      if (code != 0) throw os.ioError(code, this.address)
    }
  }

  implement Closeable {
    fun close() {
      this.fd.close()
    }
  }

  // one recv into the buffer; the chunk (empty at end of stream)
  fun fetch(): List<u8> suspends throws IoError {
    with held = this.fd.using()
    loop {
      var data = ""
      // SAFETY: stores what arrived in `data`, a local that outlives the call; on a
      // closed connection the number is -1 and the call fails
      val code = unsafe {
        veles_net_recv(held.fd(), 65536, &data)
      }
      if (code == 0) {
        val chunk = data.bytes()
        this.buffer.withLock(b => b.addAll(chunk))
        return chunk
      }
      if (code != wouldBlock) throw os.ioError(code, this.address)
      await ioWait(held.fd(), false)
    }
  }

  // how many bytes are buffered
  fun buffered(): i64 => this.buffer.withLock(b => b.len())

  // takes up to n buffered bytes
  fun take(n: i64): List<u8> => this.buffer.withLock(b => {
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
    val rest = this.drop(n)
    this.clear()
    this.addAll(rest)
  }
}
