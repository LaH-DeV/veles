/// The process: arguments, environment, exit, and running other programs.
///
/// Implemented on top of runtime/c/veles_os.c; every extern call is confined
/// to one `unsafe` block (D44). Failures are thrown as `IoError`.

extern "C" {
  fun veles_os_argc(): i64
  fun veles_os_arg(i: i64, out: *raw string)
  fun veles_os_getenv(name: string, out: *raw string): bool
  fun veles_os_exit(code: i64): Never
  fun veles_os_run(argz: string, input: string, mode: i64, out: *raw string, errout: *raw string, err: *raw i64): i64
  fun veles_os_strerror(code: i64, out: *raw string)
  fun veles_io_kind(code: i64): i64
  fun veles_os_pid(): i64
  fun veles_os_hostname(out: *raw string): i64
  fun veles_os_temp_dir(out: *raw string)
  fun veles_signal_watch()
  fun veles_signal_take(): i64
  fun veles_signal_raise(sig: i64)
}

/// The command-line arguments, without the program name.
public fun args(): List<string> {
  var out: MutableList<string> = []
  // SAFETY: reads the count the runtime saved at start-up
  val n = unsafe {
    veles_os_argc()
  }
  loop (i in 1..<n) {
    var s = ""
    // SAFETY: `i` is below the count; the argument goes into `s`, a local that
    // outlives the call
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
  // SAFETY: argument 0 always exists; it goes into `s`, a local that outlives
  // the call
  unsafe {
    veles_os_arg(0, &s)
  }
  s
}

/// The value of environment variable `name`, or `null` when it is not set.
public fun env(name: string): string? {
  if (name.contains("\u{0}")) return null  // C would read a shorter name
  var s = ""
  // SAFETY: takes `name` by value and stores the value in `s`, a local that
  // outlives the call
  val ok = unsafe {
    veles_os_getenv(name, &s)
  }
  if (ok) s else null
}

/// This process's id, as the operating system numbers it.
public fun pid(): i64 = unsafe {
  // SAFETY: a query with no arguments
  veles_os_pid()
}

/// The host's name, as the network knows it (`gethostname`, or the DNS
/// host name on Windows).
public fun hostname(): string throws IoError {
  var s = ""
  // SAFETY: stores the name in `s`, a local that outlives the call
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
  // SAFETY: stores the directory in `s`, a local that outlives the call
  unsafe {
    veles_os_temp_dir(&s)
  }
  s
}

/// Ends the process with `code` after flushing output.
public fun exit(code: i64): Never {
  // SAFETY: flushes output and ends the process; nothing runs after it
  unsafe {
    veles_os_exit(code)
  }
}

/// Why the process is being asked to stop. The values are the POSIX
/// signal numbers.
public enum Signal {
  /// SIGINT; Ctrl+C in a terminal or console.
  Interrupt = 2
  /// SIGTERM (what `kill`, systemd and Docker send); on Windows also
  /// Ctrl+Break, closing the console window, log-off and system shutdown.
  Terminate = 15
}

/// Waits until the process is asked to stop, and says how (D68):
///
/// ```veles
/// http.serve(listener, app.handler(), stop: () => os.shutdownSignal())
/// ```
///
/// Nothing is intercepted until the first call, so a program that never
/// asks keeps the default: the signal ends it. Each call takes one signal;
/// a second one before the next call has the default effect — a shutdown
/// that hangs can still be stopped with a second Ctrl+C. A signal that
/// arrives while nobody is waiting is kept for the next call. On Windows,
/// closing the console gives the program about five seconds before the
/// system ends it.
public fun shutdownSignal(): Signal {
  // SAFETY: installs the runtime's handlers once; they only record the signal
  unsafe {
    veles_signal_watch()
  }
  loop {
    // SAFETY: takes the recorded signal atomically; no arguments
    val sig = unsafe {
      veles_signal_take()
    }
    if (sig != 0) return Signal.fromValue(sig) ?: Signal.Terminate
    // A signal handler may not touch the executor, so the waiting task
    // looks for the recorded signal; a tenth of a second is well inside
    // any shutdown budget and costs nothing measurable.
    await sleep(Duration.millis(100))
  }
}

/// Delivers `sig` to this process as if it came from outside: a waiting
/// `shutdownSignal()` returns it, and with nobody watching the process
/// ends as the signal would end it. For testing a shutdown path.
public fun raiseSignal(sig: Signal) {
  // SAFETY: `sig.value` is a signal number the enum lists
  unsafe {
    veles_signal_raise(sig.value)
  }
}

/// What a finished program produced.
public struct Output {
  public code:   i64
  public stdout: string
  /// Its standard error, when `run` captured it apart (the default);
  /// empty with `Stderr.Inherit` or `Stderr.Merge`.
  public stderr: string = ""
  public fun ok(): bool = this.code == 0
}

/// What `run` does with the program's standard error (D82).
public enum Stderr {
  /// Kept apart, in `Output.stderr`.
  Capture = 0
  /// Written to this process's standard error as the program writes it.
  Inherit = 1
  /// Into `Output.stdout`, in the order the program wrote the two.
  Merge = 2
}

/// Runs `program` with `args`, waits for it, and captures its standard
/// output and — unless `stderr` says otherwise — its standard error apart.
/// `input` is written to its standard input, which is then closed; with
/// none, the program reads the end of its input at once. Throws when the
/// program cannot be started; a non-zero exit is reported in
/// `Output.code`, not thrown.
///
/// ```veles
/// val r = try os.run("git", ["apply", "-"], input: patch)
/// if (!r.ok()) io.eprintln(r.stderr)
/// ```
///
/// No shell is involved: each argument reaches the program as exactly one
/// argument, so `;`, `|`, `$(...)`, `*` and quotes in it are plain text —
/// passing text from a user cannot run a second command. `program` is
/// looked up on `PATH`. For shell features, run the shell yourself
/// (`os.run("sh", ["-c", script])`) and own what the script contains. On
/// Windows a `.bat` or `.cmd` file is refused: `cmd.exe` would re-read its
/// arguments with rules no quoting can make safe.
public fun run(program: string, args: List<string> = [], input: string = "", stderr: Stderr = Stderr.Capture): Output throws IoError {
  // the runtime takes the program and its arguments as one text, each
  // ended by a NUL — which is why none of them may contain one
  var argz = StringBuilder()
  argz.append(program)
  loop (a in args) {
    if (a.contains("\u{0}")) throw invalidArgument("an argument holds a NUL byte", program)
    argz.append("\u{0}")
    argz.append(a)
  }
  if (program.isEmpty() || program.contains("\u{0}")) throw invalidArgument("not a program name", program)
  var out = ""
  var errText = ""
  var err: i64 = 0
  // SAFETY: takes the texts by value — the runtime reads `input` only
  // until the call returns — and stores the outputs and the error number
  // in `out`, `errText` and `err`, locals that outlive the call
  val code = unsafe {
    veles_os_run(argz.toString(), input, stderr.value, &out, &errText, &err)
  }
  if (code == -2) throw invalidArgument("a batch file runs through cmd.exe, which re-reads its arguments; run 'cmd' with '/c' yourself", program)
  if (code < 0) throw ioError(err, program)
  Output(code, stdout: out, stderr: errText)
}

fun invalidArgument(why: string, program: string): IoError =
  IoError(path: program, code: 22, detail: why, kind: IoKind.InvalidInput)

/// Builds an `IoError` for a platform error number: its description and
/// its portable `kind`.
public fun ioError(code: i64, path: string): IoError {
  var detail = ""
  // SAFETY: stores the description in `detail`, a local that outlives the
  // call; veles_io_kind only maps the number
  val kind = unsafe {
    veles_os_strerror(code, &detail)
    veles_io_kind(code)
  }
  IoError(path, code, detail, kind: IoKind.fromValue(kind) ?: IoKind.Other)
}
