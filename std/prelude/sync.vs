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
pub trait Sendable { }

pub struct Mutex<T> {
  cell: *T

  pub fun withLock<R>(f: fun(*T): R): R = f(self.cell)
  pub fun get(): T = *self.cell
  pub fun set(value: T) {
    *self.cell = value
  }
}

pub fun <T> mutex(value: T): Mutex<T> = Mutex(cell: &value)

pub struct Atomic<T> {
  cell: *T

  pub fun load(): T = *self.cell
  pub fun store(value: T) {
    *self.cell = value
  }
  pub fun swap(value: T): T {
    val old = *self.cell
    *self.cell = value
    old
  }
}

pub fun <T> atomic(value: T): Atomic<T> = Atomic(cell: &value)
