// Prelude — synchronization (D35, D66). Shared mutable state crosses task
// boundaries only inside these wrappers; both are Sendable by fiat.
//
// Tasks run on several threads (D66), so a Mutex is a real lock: a word in
// the heap taken with one atomic operation when it is free, a short spin
// and then a parked thread when it is not (runtime/c/veles_sync.c). An
// Atomic of a number or a bool needs no lock: its operations are single
// processor instructions (the std-only atomic* builtins).
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
public trait Sendable

// A taken lock, given back when the `with` holding it ends — also when the
// function inside panics, so a failed task does not leave the value locked.
struct Held {
  word: *i64

  static fun take(word: *i64): Held {
    // SAFETY: `word` is a heap cell this Held keeps reachable until it unlocks
    unsafe {
      veles_mutex_lock(word as *raw i64)
    }
    Held(word)
  }

  implement Closeable {
    fun close() {
      // SAFETY: this Held took the lock in `take`, and `with` closes it once
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
/// val hits = Mutex(value: 0)
/// hits.withLock(n => *n += 1)  // from any number of tasks
/// io.println(hits.get())
/// ```
public struct Mutex<T> {
  private cell: *T
  private word: *i64 = newWord()

  init(value: T) {
    this.cell = &value
  }

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

/// A value that tasks read and replace whole, each operation indivisible.
/// For a change that reads the value first, use `update`: a `load`
/// followed by a `store` can lose another task's store in between.
///
/// An `Atomic` of a number or a `bool` takes no lock: each operation is
/// one processor instruction. Any other value is guarded by a lock word.
public struct Atomic<T> {
  private cell: *T
  private word: *i64 = newWord()

  init(value: T) {
    this.cell = &value
  }

  public fun load(): T {
    if (atomicLockFree(this.cell)) {
      return atomicLoad(this.cell)
    }
    with (held = Held.take(this.word)) {
      return *this.cell
    }
  }

  public fun store(value: T) {
    if (atomicLockFree(this.cell)) {
      atomicStore(this.cell, value)
      return
    }
    with (held = Held.take(this.word)) {
      *this.cell = value
    }
  }

  /// Stores `value` and returns the value it replaced.
  public fun swap(value: T): T {
    if (atomicLockFree(this.cell)) {
      return atomicSwap(this.cell, value)
    }
    with (held = Held.take(this.word)) {
      val old = *this.cell
      *this.cell = value
      return old
    }
  }

  /// Replaces the value with `f` of it, as one step, and returns the new
  /// value: `counter.update(n => n + 1)`. When tasks update at the same
  /// moment, `f` may run more than once — with the newer value each time —
  /// so it should only compute, not print or change other state.
  public fun update(f: fun(T): T): T {
    if (atomicLockFree(this.cell)) {
      loop {
        val old = atomicLoad(this.cell)
        val next = f(old)
        if (atomicCompareAndSwap(this.cell, old, next)) {
          return next
        }
      }
    }
    with (held = Held.take(this.word)) {
      val next = f(*this.cell)
      *this.cell = next
      return next
    }
  }
}

// a lock word of its own, free
fun newWord(): *i64 {
  var word: i64 = 0
  &word
}
