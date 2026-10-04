/// Console input and output.
///
/// Implemented on top of the C runtime (runtime/c/veles_rt.c). Extern calls
/// are unsafe (D44), so each wrapper confines them to one block.

extern "C" {
  fun veles_print(s: string)
  fun veles_eprint(s: string)
  fun veles_println(s: string)
  fun veles_eprintln(s: string)
  fun veles_read_line(out: *raw string): bool
  fun veles_read_all(out: *raw string): i64
}

/// A read that would have exceeded the ceiling its caller gave. The bytes
/// read so far are dropped and the stream is left where it stood, so
/// there is nothing sensible to resume from: close it. A server answers
/// the peer first — 431 for headers, 413 for a body — and then closes.
public error TooLong {
  public message: string
  /// The ceiling that was passed, in bytes.
  public limit: i64
}

/// A sequence of bytes that is read and written: a TCP connection
/// (`net.Conn`), a file (`fs.File`), later a TLS connection and the
/// compressors that wrap one. Code that takes a `Stream` works on all of
/// them — `http` serves and calls over one — and `with s = ...` closes it.
///
/// The effects are declared (D40), so every implement may suspend: a
/// file's calls never do, a socket's park the task until the peer is ready.
///
/// ```veles
/// fun copyAll(from: io.Stream, to: io.Stream) suspends throws IoError {
///   loop {
///     val chunk = try from.read()
///     if (chunk.isEmpty()) break
///     try to.write(chunk)
///   }
/// }
/// ```
public trait Stream : Closeable {
  /// Up to `max` bytes, as soon as any are available; an empty list means
  /// the end of the stream. Fewer than `max` is normal.
  fun read(max: i64 = 65536): List<u8> suspends throws IoError

  /// Exactly `n` bytes, or fewer when the stream ends first. `n` is the
  /// ceiling as well as the count, so a caller that takes it from the peer
  /// must check it against its own limit before calling.
  fun readExact(n: i64): List<u8> suspends throws IoError

  /// The next line as text, without its `\n` (and a `\r` before it), or
  /// `null` at the end with nothing left; a line that is not valid UTF-8 is
  /// an error. `max` has no default on purpose: the peer decides where the
  /// newline goes, so a program that never says allows a stranger to fill
  /// its memory. A line that reaches the ceiling throws `TooLong`.
  fun readLine(max: i64): string? suspends throws IoError | TooLong

  /// All of `bytes`.
  fun write(bytes: List<u8>) suspends throws IoError

  /// `text` as UTF-8.
  fun writeText(text: string) suspends throws IoError

  /// Tells the other side that nothing more will be sent while this side
  /// keeps reading. Does nothing on a file.
  fun shutdownWrite() suspends throws IoError
}

/// Writes `s` and a newline to standard output.
public fun println(s: string) {
  // SAFETY: writes `s` within its length and keeps nothing; the stream is locked
  unsafe {
    veles_println(s)
  }
}

/// Writes `s` to standard output, without a newline.
public fun print(s: string) {
  // SAFETY: writes `s` within its length and keeps nothing; the stream is locked
  unsafe {
    veles_print(s)
  }
}

/// Writes `s` and a newline to standard error.
public fun eprintln(s: string) {
  // SAFETY: writes `s` within its length and keeps nothing; the stream is locked
  unsafe {
    veles_eprintln(s)
  }
}

/// Reads one line from standard input without the trailing newline, or
/// `null` at end of input.
public fun readLine(): string? {
  var line = ""
  // SAFETY: stores the line in `line`, a local that outlives the call
  val ok = unsafe {
    veles_read_line(&line)
  }
  if (ok) line else null
}

/// Reads standard input to its end (the rest of it, after any `readLine`).
public fun readAll(): string {
  var text = ""
  // SAFETY: stores the input in `text`, a local that outlives the call
  unsafe {
    veles_read_all(&text)
  }
  text
}
