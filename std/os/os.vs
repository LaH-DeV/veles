/// The process: arguments, environment, exit, and running other programs.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44). Failures are thrown as `IoError`.

extern "C" {
  fun veles_os_argc(): i64
  fun veles_os_arg(i: i64, out: *raw string)
  fun veles_os_getenv(name: string, out: *raw string): bool
  fun veles_os_exit(code: i64): Never
  fun veles_os_run(cmd: string, out: *raw string, err: *raw i64): i64
  fun veles_os_strerror(code: i64, out: *raw string)
  fun veles_os_pid(): i64
  fun veles_os_hostname(out: *raw string): i64
  fun veles_os_temp_dir(out: *raw string)
}

/// The command-line arguments, without the program name.
public fun args(): List<string> {
  var out: MutableList<string> = []
  val n = unsafe {
    veles_os_argc()
  }
  loop (i in 1..<n) {
    var s = ""
    unsafe {
      veles_os_arg(i, &s)
    }
    out.push(s)
  }
  out.toList()
}

/// The program's own path, as it was invoked.
public fun program(): string {
  var s = ""
  unsafe {
    veles_os_arg(0, &s)
  }
  s
}

/// The value of environment variable `name`, or `null` when it is not set.
public fun env(name: string): string? {
  var s = ""
  val ok = unsafe {
    veles_os_getenv(name, &s)
  }
  if (ok) s else null
}

/// This process's id, as the operating system numbers it.
public fun pid(): i64 = unsafe {
  veles_os_pid()
}

/// The host's name, as the network knows it (`gethostname`, or the DNS
/// host name on Windows).
public fun hostname(): string throws IoError {
  var s = ""
  val code = unsafe {
    veles_os_hostname(&s)
  }
  if (code != 0) throw ioError(code, "hostname")
  s
}

/// The directory for temporary files: `TMPDIR` or `/tmp` on POSIX, what
/// `TMP`/`TEMP` name on Windows. Without a trailing separator, so
/// `path.join(os.tempDir(), name)` reads as it should.
public fun tempDir(): string {
  var s = ""
  unsafe {
    veles_os_temp_dir(&s)
  }
  s
}

/// Ends the process with `code` after flushing output.
public fun exit(code: i64): Never {
  unsafe {
    veles_os_exit(code)
  }
}

/// What a finished program produced.
public struct Output {
  public code:   i64
  public stdout: string
  public fun ok(): bool = self.code == 0
}

/// Runs `program` with `args`, waits for it, and captures its standard
/// output; standard error passes through, or is captured into the same
/// text with `mergeStderr`. Throws when the program cannot be started; a
/// non-zero exit is reported in `Output.code`, not thrown.
public fun run(program: string, args: List<string> = [], mergeStderr: bool = false): Output throws IoError {
  var cmd = quote(program)
  loop (a in args) {
    cmd = cmd + " " + quote(a)
  }
  if (mergeStderr) cmd = cmd + " 2>&1"
  var out = ""
  var err: i64 = 0
  val code = unsafe {
    veles_os_run(cmd, &out, &err)
  }
  if (code < 0) throw ioError(err, program)
  Output(code, stdout: out)
}

/// Builds an `IoError` for a platform error number.
public fun ioError(code: i64, path: string): IoError {
  var detail = ""
  unsafe {
    veles_os_strerror(code, &detail)
  }
  IoError(path, code, detail)
}

fun quote(s: string): string {
  if (!s.isEmpty() && !s.contains(" ") && !s.contains("\"")) return s
  "\"" + s.replace("\"", "\\\"") + "\""
}
