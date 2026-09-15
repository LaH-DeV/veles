// Prelude — synchronization (D35). Shared mutable state crosses task
// boundaries only inside these wrappers; both are Sendable by fiat.
//
// The executor is single-threaded (build plan §5), so a lock never
// contends; the types exist so that programs are written against the
// isolation rules from day one. `withLock` takes a non-suspending
// function, which is how the compiler rejects `await` inside a lock scope.

pub struct Mutex<T> {
  cell: *T

  pub fun withLock<R>(f: fun(*T): R): R = f(self.cell)
  pub fun get(): T = *self.cell
  pub fun set(value: T) { *self.cell = value }
}

pub fun <T> mutex(value: T): Mutex<T> = Mutex(cell: &value)

pub struct Atomic<T> {
  cell: *T

  pub fun load(): T = *self.cell
  pub fun store(value: T) { *self.cell = value }
  pub fun swap(value: T): T {
    val old = *self.cell
    *self.cell = value
    old
  }
}

pub fun <T> atomic(value: T): Atomic<T> = Atomic(cell: &value)
