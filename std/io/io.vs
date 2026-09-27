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
