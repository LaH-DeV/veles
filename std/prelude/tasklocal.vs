// Prelude — task-local values (D72). A value bound for a stretch of a
// task's work that every task started inside that stretch sees too: a
// request id, a trace context, a logger. Bindings live on the task
// (runtime/c/veles_task.c), as a list nothing changes in place, so a child
// shares its parent's without a lock and keeps what was bound when it
// started.

extern "C" {
  fun veles_local_key(): i64
  fun veles_local_find(key: i64): (*raw u8)?
  fun veles_local_bind(key: i64, cell: *raw u8): (*raw u8)?
  fun veles_local_restore(head: (*raw u8)?)
}

/// A value that follows a task and the tasks it starts, without being
/// passed as a parameter. Declare one at module level, bind it around a
/// piece of work with `withValue`, and read it anywhere inside with `get`:
///
/// ```veles
/// val requestId = taskLocal("-")
///
/// fun handle(req: Request) {
///   requestId.withValue(req.id, () => process(req))
/// }
///
/// fun log(msg: string) {
///   io.println("[${requestId.get()}] $msg")   // the id of the request being handled
/// }
/// ```
///
/// A binding cannot change while it is in effect; a nested `withValue`
/// shadows it until it ends. A task started inside sees the values bound
/// where it was started, for as long as it runs. Outside every binding,
/// `get` returns the value `taskLocal` was given.
public struct TaskLocal<T: Sendable> {
  key:      i64
  fallback: T

  /// The value bound innermost in the current task, or the fallback.
  public fun get(): T {
    val cell = unsafe {
      veles_local_find(this.key)
    } ?: return this.fallback
    unsafe {
      *(cell as *raw T)
    }
  }

  /// Runs `f` with this bound to `value` — there and in every task started
  /// inside it — and returns what `f` returns. The binding ends with `f`,
  /// also when `f` throws, panics or its task is cancelled.
  public fun withValue<R, E>(value: T, f: fun(): R suspends throws E): R throws E {
    var held = value
    val cell: *raw u8 = unsafe {
      &held as *raw u8
    }
    with (binding = LocalBinding.take(this.key, cell)) {
      return try f()
    }
  }
}

/// A task-local value whose `get` returns `fallback` outside any binding.
public fun taskLocal<T: Sendable>(fallback: T): TaskLocal<T> {
  val key = unsafe {
    veles_local_key()
  }
  TaskLocal(key, fallback)
}

// One binding in effect: ends — the previous bindings back — when the
// `with` holding it does.
struct LocalBinding {
  previous: (*raw u8)?

  static fun take(key: i64, cell: *raw u8): LocalBinding {
    val previous = unsafe {
      veles_local_bind(key, cell)
    }
    LocalBinding(previous)
  }

  implement Closeable {
    fun close() {
      unsafe {
        veles_local_restore(this.previous)
      }
    }
  }
}
