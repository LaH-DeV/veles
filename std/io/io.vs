/// Console input and output.
///
/// Implemented on top of the C runtime (runtime/c/veles_rt.c). Extern calls
/// are unsafe (D44), so each wrapper confines them to one block.

extern "C" {
  fun veles_print(s: string)
  fun veles_eprint(s: string)
  fun veles_println(s: string)
  fun veles_eprintln(s: string)
  fun veles_read_line(max: i64, text: i64, out: *raw string, error: *raw i64): i64
  fun veles_read_all(max: i64, text: i64, out: *raw string, error: *raw i64): i64
}

use bytes

const stdinOK: i64 = 0
const stdinEOF: i64 = 1
const stdinInvalidUTF8: i64 = 2
const stdinTooLong: i64 = 3
const stdinIOError: i64 = 4

/// A bounded read exceeded the ceiling its caller gave. The partial result
/// is discarded and the input remains where overflow was detected. Close a
/// stream after this error; stdin has no handle to close.
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
public trait Stream : Closeable + Sendable {
  /// Up to `max` bytes, as soon as any are available; an empty list means
  /// the end of the stream. Fewer than `max` is normal.
  fun read(max: i64 = bytes.KiB * 64): List<u8> suspends throws IoError

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

fun checkStdinStatus(status: i64, code: i64, max: i64, what: string) throws IoError | TooLong {
  if (status == stdinOK) return
  if (status == stdinInvalidUTF8) {
    throw IoError(path: "stdin", code: 0, detail: "standard input $what is not valid UTF-8", kind: IoKind.InvalidData)
  }
  if (status == stdinTooLong) {
    throw TooLong(message: "standard input $what exceeded its $max-byte limit", limit: max)
  }
  if (status == stdinIOError) {
    throw IoError(path: "stdin", code, detail: "could not read standard input", kind: IoKind.Other)
  }
  throw IoError(path: "stdin", code, detail: "unknown standard-input status $status", kind: IoKind.Other)
}

/// Reads one validated UTF-8 line from standard input without the trailing
/// newline, or `null` at end of input. The default ceiling is 64 KiB.
public fun readLine(max: i64 = bytes.KiB * 64): string? throws IoError | TooLong {
  if (max < 0) panic("io.readLine: max must be nonnegative")
  var line = ""
  var code: i64 = 0
  // SAFETY: stores validated text in `line`, a local that outlives the call
  val status = unsafe {
    veles_read_line(max, 1, &line, &code)
  }
  if (status == stdinEOF) return null
  try checkStdinStatus(status, code, max, "line")
  line
}

/// Reads one raw-byte line, without the trailing newline or a preceding CR.
/// The default ceiling is 64 KiB.
public fun readLineBytes(max: i64 = bytes.KiB * 64): List<u8>? throws IoError | TooLong {
  if (max < 0) panic("io.readLineBytes: max must be nonnegative")
  var data = ""
  var code: i64 = 0
  // SAFETY: stores raw bytes in a temporary buffer consumed immediately below
  val status = unsafe {
    veles_read_line(max, 0, &data, &code)
  }
  if (status == stdinEOF) return null
  try checkStdinStatus(status, code, max, "line")
  data.bytes()
}

/// Reads the rest of standard input as validated UTF-8, up to 1 MiB by default.
public fun readAll(max: i64 = bytes.MiB): string throws IoError | TooLong {
  if (max < 0) panic("io.readAll: max must be nonnegative")
  var text = ""
  var code: i64 = 0
  // SAFETY: stores validated text in `text`, a local that outlives the call
  val status = unsafe {
    veles_read_all(max, 1, &text, &code)
  }
  try checkStdinStatus(status, code, max, "input")
  text
}

/// Reads the rest of standard input as bytes, up to 1 MiB by default.
public fun readAllBytes(max: i64 = bytes.MiB): List<u8> throws IoError | TooLong {
  if (max < 0) panic("io.readAllBytes: max must be nonnegative")
  var data = ""
  var code: i64 = 0
  // SAFETY: stores raw bytes in a temporary buffer consumed immediately below
  val status = unsafe {
    veles_read_all(max, 0, &data, &code)
  }
  try checkStdinStatus(status, code, max, "input")
  data.bytes()
}
