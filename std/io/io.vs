/// Console input and output.
///
/// Implemented on top of the C runtime (runtime/c/veles_rt.c). Extern calls
/// are unsafe (D44), so each wrapper confines them to one block.

extern "C" {
  fun veles_print(s: string)
  fun veles_eprint(s: string)
  fun veles_read_line(out: *raw string): bool
}

/// Writes `s` and a newline to standard output.
pub fun println(s: string) {
  unsafe {
    veles_print(s)
    veles_print("\n")
  }
}

/// Writes `s` to standard output, without a newline.
pub fun print(s: string) {
  unsafe {
    veles_print(s)
  }
}

/// Writes `s` and a newline to standard error.
pub fun eprintln(s: string) {
  unsafe {
    veles_eprint(s)
    veles_eprint("\n")
  }
}

/// Reads one line from standard input without the trailing newline, or
/// `null` at end of input.
pub fun readLine(): string? {
  var line = ""
  val ok = unsafe {
    veles_read_line(&line)
  }
  if (ok) line else null
}
