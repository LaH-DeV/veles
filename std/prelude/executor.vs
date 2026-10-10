// Prelude — where tasks and threads run (D143): `Executor` (a pool of
// threads of its own, or one thread), `scope(on: e)` and `e.run(f)` to
// place tasks on it, `blocking(f)` for a call known to block, and `Thread`
// for plain code on an OS thread of its own. The runtime side is
// runtime/c/veles_task.c ("executors", "Thread") and veles_sync.c (the OS
// settings).

extern "C" {
  fun veles_exec_new(threads: i64, single: bool, name: string, priority: i64, cpus: *raw i64, ncpus: i64, err: *raw string): *raw u8
  fun veles_exec_close(e: *raw u8)
  fun veles_exec_blocking(): *raw u8
  fun veles_exec_default_threads(): i64
  fun veles_thread_start(name: string, priority: i64, cpus: *raw i64, ncpus: i64, stackSize: i64, body: *raw u8, err: *raw string): *raw u8
  fun veles_thread_join(t: *raw u8)
}

/// How the OS schedules a thread against the others (D143). `Normal` leaves
/// it as the system made it. Raising it can need a privilege: on Linux
/// `High` and `Realtime` need `CAP_SYS_NICE`, and without it the thread is
/// refused with a `ThreadError`, never started at a priority it did not ask
/// for.
public enum Priority {
  Low
  Normal
  High
  Realtime
}

/// The OS would not start a thread as asked: no thread could be made, or it
/// refused the priority or the CPUs (D143). Nothing was left running.
public error ThreadError {
  /// What the system refused, in its words where it gave them.
  public reason: string
  fun message(): string => this.reason
}

/// Threads of its own that tasks are placed on (D143). Without one, every
/// task runs on the default pool — one thread per core, or `[runtime]
/// threads` in `veles.toml`. An executor keeps work apart: CPU-heavy tasks
/// that must not take the threads a server answers on, a library that must
/// always be called from one thread, a thread pinned to a core at a raised
/// priority.
///
/// ```veles
/// with cpu = try Executor.pool(threads: 4, name: "cpu")
/// scope(on: cpu) {
///   loop (f in files) async compress(f)   // every child runs on cpu's threads
/// }
/// with gl = try Executor.thread(name: "gl")
/// val tex = gl.run(() => upload(image))    // on gl's one thread; this task waits
/// ```
///
/// A task placed on an executor stays on it across every suspension. Values
/// cross to it as between any two tasks: `Sendable` (D35). Closing it stops
/// its threads; the tasks placed on it have finished by then, since the
/// scopes that placed them joined them. Copies are the same executor.
public struct Executor {
  private handle: *raw u8

  /// A pool of `threads` threads, named `name-0`, `name-1`, … for debuggers
  /// and profilers, each started at `priority` and kept to `cpus` (logical
  /// CPUs numbered from 0; empty: any). Throws `ThreadError` when the OS
  /// refuses one of them.
  public static fun pool(threads: i64, name: string, priority: Priority = Priority.Normal, cpus: List<i64> = []): Executor throws ThreadError {
    if (threads < 1) panic("Executor.pool: threads must be at least 1, got $threads")
    try Executor.make(threads, false, name, priority, cpus)
  }

  /// One thread, named `name`, that runs every task placed on it: a task
  /// that must always be on the same OS thread — a GL context, COM, a GUI's
  /// main loop — is placed here.
  public static fun thread(name: string, priority: Priority = Priority.Normal, cpus: List<i64> = []): Executor throws ThreadError {
    try Executor.make(1, true, name, priority, cpus)
  }

  static fun make(threads: i64, single: bool, name: string, priority: Priority, cpus: List<i64>): Executor throws ThreadError {
    var reason = ""
    val at = listRawData(cpus)
    // SAFETY: the runtime copies the name and the CPUs (`at`, alive with
    // `cpus` for the call); on a refusal it stores a fresh string in
    // `reason`, a local that outlives the call
    val h = unsafe {
      veles_exec_new(threads, single, name, priority.value, at, cpus.len(), &reason)
    }
    if (!reason.isEmpty()) throw ThreadError(reason)
    Executor(handle: h)
  }

  /// Runs `f` as a task on this executor and waits for it: its result, or
  /// the error it throws. The one-call form of `scope(on: e) { async f() }`.
  public fun run<R: Sendable, E>(f: sendable fun(): R suspends throws E): R throws E {
    scope(on: this) {
      return await async f()
    }
  }

  /// How many threads the default pool has: `VELES_THREADS`, `[runtime]
  /// threads`, or one per core.
  public static fun defaultThreads(): i64 => unsafe {
    // SAFETY: reads the runtime's configuration
    veles_exec_default_threads()
  }

  implement Closeable {
    /// Stops the threads and waits for them to end. Closing it again — or
    /// a copy — does nothing; placing a task on it afterwards panics.
    fun close() {
      // SAFETY: the runtime stops the executor once; a later call is a no-op
      unsafe {
        veles_exec_close(this.handle)
      }
    }
  }
}

/// Runs `f`, a call known to block its thread — a C library's read, a
/// legacy synchronous API — on a thread of the blocking pool, and waits for
/// it without holding up the tasks of the default pool (D143). The pool
/// starts threads as calls need them, up to 128, and a call beyond that
/// waits for one; a thread idle for 10 seconds ends.
///
/// ```veles
/// val data = try blocking(() => try legacy.readAll(path))
/// ```
///
/// A blocking call made without it is still detected and its thread's
/// queue handed on (D66); `blocking` skips the detection.
public fun blocking<R: Sendable, E>(f: sendable fun(): R throws E): R throws E {
  try blockingPool.run(() => try f())
}

// the blocking pool: it starts threads as calls need them, and lives as
// long as the program
val blockingPool: Executor = Executor(handle: unsafe {
  // SAFETY: the runtime makes the pool once; it is never closed
  veles_exec_blocking()
})

/// An OS thread running plain code of its own, outside every executor
/// (D143): an audio loop at a raised priority, pinned to a core.
///
/// ```veles
/// with audio = try Thread.start(name: "audio", priority: Priority.High, cpus: [3], f: () => audioLoop())
/// ```
///
/// `f` does not suspend. The `with` that holds the thread waits for it to
/// end when its block does — the waiting thread blocks, as `join` does in
/// Rust — and a panic in `f` is raised again there.
public struct Thread {
  private handle: *raw u8

  /// Starts `f` on a new thread named `name`, at `priority`, kept to `cpus`
  /// (empty: any), with a stack of `stackSize` bytes (0: the size Veles
  /// gives its own threads). Throws `ThreadError` when the OS refuses.
  public static fun start(name: string, priority: Priority = Priority.Normal, cpus: List<i64> = [], stackSize: i64 = 0, f: sendable fun()): Thread throws ThreadError {
    if (stackSize < 0) panic("Thread.start: stackSize must not be negative, got $stackSize")
    var body = f
    var reason = ""
    // SAFETY: `body` is a heap cell (its address is taken) that the runtime
    // keeps reachable until the join, and calls as a plain function; the
    // name and CPUs are copied; on a refusal `reason` gets a fresh string
    val h = unsafe {
      val cell: *raw sendable fun() = &body
      veles_thread_start(name, priority.value, listRawData(cpus), cpus.len(), stackSize, cell.cast<*raw u8>(), &reason)
    }
    if (!reason.isEmpty()) throw ThreadError(reason)
    Thread(handle: h)
  }

  implement Closeable {
    /// Waits for the thread to end, and raises a panic of its function
    /// here. Joining again — or a copy — does nothing.
    fun close() {
      // SAFETY: the runtime joins once and keeps the record until then
      unsafe {
        veles_thread_join(this.handle)
      }
    }
  }
}
