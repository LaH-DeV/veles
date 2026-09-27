// Prelude — synchronization (D35, D66). Shared mutable state crosses task
// boundaries only inside these wrappers; both are Sendable by fiat.
//
// Tasks run on several threads (D66), so both are real locks: a word in
// the heap taken with one atomic operation when it is free, a short spin
// and then a parked thread when it is not (runtime/c/veles_sync.c).
// `withLock` takes a non-suspending function, which is how the compiler
// rejects `await` inside a lock scope — the classic way to deadlock a
// coroutine runtime is to suspend while holding a lock.

extern "C" {
  fun veles_mutex_lock(word: *raw i64)
  fun veles_mutex_unlock(word: *raw i64)
}

/// The marker for values that can be shared without aliasing mutable state:
/// numbers, strings, immutable collections of Sendable elements, and structs
/// whose fields are all Sendable; never a mutable collection, a pointer or a
/// closure. It is derived from a type's shape and cannot be implemented by
/// hand. A value crossing a task boundary must be Sendable (D35/D54), and
/// so must one that a function duplicates, such as `MutableList<T>.fill`.
public trait Sendable { }

// A taken lock, given back when the `with` holding it ends — also when the
// function inside panics, so a failed task does not leave the value locked.
struct Held {
  word: *i64

  static fun take(word: *i64): Held {
    unsafe {
      veles_mutex_lock(word as *raw i64)
    }
    Held(word)
  }

  implement Closeable {
    fun close() {
      unsafe {
        veles_mutex_unlock(this.word as *raw i64)
      }
    }
  }
}

/// A value that tasks share and change, one at a time. Copies of a
/// `Mutex` are the same lock and the same value.
///
/// ```veles
/// val hits = mutex(0)
/// hits.withLock(n => *n += 1)  // from any number of tasks
/// io.println(hits.get())
/// ```
public struct Mutex<T> {
  cell: *T
  word: *i64

  /// Runs `f` with the value locked and returns what it returns. `f`
  /// cannot suspend; locking the same `Mutex` again inside it panics.
  public fun withLock<R>(f: fun(*T): R): R {
    with (held = Held.take(this.word)) {
      return f(this.cell)
    }
  }

  /// A copy of the value, read under the lock.
  public fun get(): T = this.withLock(p => *p)

  /// Replaces the value under the lock.
  public fun set(value: T) {
    this.withLock(p => {
      *p = value
    })
  }
}

public fun mutex<T>(value: T): Mutex<T> = Mutex(cell: &value, word: newWord())

/// A value that tasks read and replace whole, each operation indivisible.
/// For a change that reads the value first, use `update`: a `load`
/// followed by a `store` can lose another task's store in between.
public struct Atomic<T> {
  cell: *T
  word: *i64

  public fun load(): T {
    with (held = Held.take(this.word)) {
      return *this.cell
    }
  }

  public fun store(value: T) {
    with (held = Held.take(this.word)) {
      *this.cell = value
    }
  }

  /// Stores `value` and returns the value it replaced.
  public fun swap(value: T): T {
    with (held = Held.take(this.word)) {
      val old = *this.cell
      *this.cell = value
      return old
    }
  }

  /// Replaces the value with `f` of it, as one step, and returns the new
  /// value: `counter.update(n => n + 1)`.
  public fun update(f: fun(T): T): T {
    with (held = Held.take(this.word)) {
      val next = f(*this.cell)
      *this.cell = next
      return next
    }
  }
}

public fun atomic<T>(value: T): Atomic<T> = Atomic(cell: &value, word: newWord())

// a lock word of its own, free
fun newWord(): *i64 {
  var word: i64 = 0
  &word
}
