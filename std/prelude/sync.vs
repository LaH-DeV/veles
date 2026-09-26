// Prelude — synchronization (D35). Shared mutable state crosses task
// boundaries only inside these wrappers; both are Sendable by fiat.
//
// The executor is single-threaded (build plan §5), so a lock never
// contends; the types exist so that programs are written against the
// isolation rules from day one. `withLock` takes a non-suspending
// function, which is how the compiler rejects `await` inside a lock scope.

/// The marker for values that can be shared without aliasing mutable state:
/// numbers, strings, immutable collections of Sendable elements, and structs
/// whose fields are all Sendable; never a mutable collection, a pointer or a
/// closure. It is derived from a type's shape and cannot be implemented by
/// hand. A value crossing a task boundary must be Sendable (D35/D54), and
/// so must one that a function duplicates, such as `MutableList<T>.fill`.
public trait Sendable { }

public struct Mutex<T> {
  cell: *T

  public fun withLock<R>(f: fun(*T): R): R = f(this.cell)
  public fun get(): T = *this.cell
  public fun set(value: T) {
    *this.cell = value
  }
}

public fun mutex<T>(value: T): Mutex<T> = Mutex(cell: &value)

public struct Atomic<T> {
  cell: *T

  public fun load(): T = *this.cell
  public fun store(value: T) {
    *this.cell = value
  }
  public fun swap(value: T): T {
    val old = *this.cell
    *this.cell = value
    old
  }
}

public fun atomic<T>(value: T): Atomic<T> = Atomic(cell: &value)
