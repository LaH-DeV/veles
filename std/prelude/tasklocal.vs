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
/// val requestId = TaskLocal(fallback: "-")
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
/// `get` returns the `fallback` it was made with.
public struct TaskLocal<T: Sendable> {
  private key:      i64
  private fallback: T

  init(fallback: T) {
    // SAFETY: hands out a fresh number; no arguments
    this.key = unsafe {
      veles_local_key()
    }
    this.fallback = fallback
  }

  /// The value bound innermost in the current task, or the fallback.
  public fun get(): T {
    // SAFETY: a lookup in the current task's bindings; nothing is changed
    val cell = unsafe {
      veles_local_find(this.key)
    } ?: return this.fallback
    // SAFETY: a cell bound to this key was bound by `withValue` of this
    // TaskLocal<T>, so it holds a T; the binding keeps it reachable
    unsafe {
      *(cell.cast<*raw T>())
    }
  }

  /// Runs `f` with this bound to `value` — there and in every task started
  /// inside it — and returns what `f` returns. The binding ends with `f`,
  /// also when `f` throws, panics or its task is cancelled.
  public fun withValue<R, E>(value: T, f: fun(): R suspends throws E): R throws E {
    var held = value
    // SAFETY: `held` is a heap cell (its address is taken); the binding node
    // that holds it is GC memory reachable from every task that can see it
    val cell: *raw u8 = unsafe {
      (&held).cast<*raw u8>()
    }
    with binding = LocalBinding.take(this.key, cell)
    return try f()
  }
}

// One binding in effect: ends — the previous bindings back — when the
// `with` holding it does.
struct LocalBinding {
  previous: (*raw u8)?

  static fun take(key: i64, cell: *raw u8): LocalBinding {
    // SAFETY: `cell` holds a T for this key (see withValue); the node is GC
    // memory, so the cell lives as long as a task can find it
    val previous = unsafe {
      veles_local_bind(key, cell)
    }
    LocalBinding(previous)
  }

  implement Closeable {
    fun close() {
      // SAFETY: puts back the head `take` saw; the list is never changed in
      // place, so tasks started meanwhile keep the bindings they had
      unsafe {
        veles_local_restore(this.previous)
      }
    }
  }
}
