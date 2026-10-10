// Prelude — synchronization (D35, D66). Shared mutable state crosses task
// boundaries only inside these wrappers; both are Sendable by fiat.
//
// Tasks run on several threads (D66), so a Mutex is a real lock: a word in
// the heap taken with one atomic operation when it is free, a short spin
// and then a parked thread when it is not (runtime/c/veles_sync.c). An
// Atomic of a number, a bool, an enum or a pointer needs no lock: its
// operations are single processor instructions (the std-only atomic*
// builtins, D144).
// `withLock` takes a non-suspending function, which is how the compiler
// rejects `await` inside a lock scope — the classic way to deadlock a
// coroutine runtime is to suspend while holding a lock.

extern "C" {
  fun veles_mutex_lock(word: *raw i64)
  fun veles_mutex_unlock(word: *raw i64)
  fun veles_rw_read_lock(words: *raw i64)
  fun veles_rw_read_unlock(words: *raw i64)
  fun veles_rw_write_lock(words: *raw i64)
  fun veles_rw_write_unlock(words: *raw i64)
  fun veles_panic_current(out: *raw string): i64
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
      veles_mutex_lock(word.cast<*raw i64>())
    }
    Held(word)
  }

  implement Closeable {
    fun close() {
      // SAFETY: this Held took the lock in `take`, and `with` closes it once
      unsafe {
        veles_mutex_unlock(this.word.cast<*raw i64>())
      }
    }
  }
}

// What `with n = m.lock()` holds (D107): the taken lock and the value it
// guards. The compiler binds `n` to `value`; the close is the unlock.
struct Locked<T> {
  held:  Held
  value: *T

  implement Closeable {
    fun close() {
      this.held.close()
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
///
/// with n = hits.lock()         // held to the end of the block
/// *n += 1
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
    with held = Held.take(this.word)
    return f(this.cell)
  }

  /// Locks the value to the end of the `with` that holds it:
  /// `with n = notes.lock()` binds `n` to a pointer to the value, and the
  /// block's end — or any way out of it — unlocks it. Usable only as a
  /// `with` value; nothing inside the block may suspend, and locking the
  /// same `Mutex` again before it ends panics (D107).
  public fun lock(): Locked<T> => Locked(held: Held.take(this.word), value: this.cell)

  /// A copy of the value, read under the lock.
  public fun get(): T => this.withLock(p => *p)

  /// Replaces the value under the lock.
  public fun set(value: T) {
    this.withLock(p => {
      *p = value
    })
  }
}

// An RwLock's two words (runtime/c/veles_sync.c): the state — writer bit,
// parked bit, readers — and the writers waiting.
struct RwWords {
  var state:   i64
  var writers: i64
}

// A taken read or write lock on an RwLock, given back when the `with`
// holding it ends (or the function inside panics), as Held is for a Mutex.
struct RwHeld {
  words: *RwWords
  write: bool

  static fun take(words: *RwWords, write: bool): RwHeld {
    // SAFETY: `words` is a heap cell this RwHeld keeps reachable until it unlocks
    unsafe {
      if (write) {
        veles_rw_write_lock(words.cast<*raw i64>())
      } else {
        veles_rw_read_lock(words.cast<*raw i64>())
      }
    }
    RwHeld(words, write)
  }

  implement Closeable {
    fun close() {
      // SAFETY: this RwHeld took the lock in `take`, and `with` closes it once
      unsafe {
        if (this.write) {
          veles_rw_write_unlock(this.words.cast<*raw i64>())
        } else {
          veles_rw_read_unlock(this.words.cast<*raw i64>())
        }
      }
    }
  }
}

// What `with c = lock.read()` and `with w = lock.write()` hold (D146): the
// taken lock and the value, as Locked is for a Mutex (D107).
struct RwLocked<T> {
  held:  RwHeld
  value: *T

  implement Closeable {
    fun close() {
      this.held.close()
    }
  }
}

/// A value that many tasks read at once and one at a time changes (D146).
/// Copies of an `RwLock` are the same lock and the same value.
///
/// ```veles
/// val config = RwLock(value: loadConfig())
/// with c = config.read()      // any number of readers at once
/// io.println(c.name)
///
/// with w = config.write()     // one writer, and no readers meanwhile
/// w.name = "new"
/// ```
///
/// A writer that waits stops new readers from entering, so readers cannot
/// starve it. As with a `Mutex`, nothing inside a held region may suspend,
/// and taking the same `RwLock` again before the region ends — a read inside
/// a read or a write, a write inside either — panics instead of hanging.
public struct RwLock<T> {
  private cell:  *T
  private words: *RwWords = &RwWords(state: 0, writers: 0)

  init(value: T) {
    this.cell = &value
  }

  /// Takes the lock to read, to the end of the `with` that holds it:
  /// `with c = config.read()` binds `c` to a pointer to the value. Usable
  /// only as a `with` value.
  public fun read(): RwLocked<T> => RwLocked(held: RwHeld.take(this.words, false), value: this.cell)

  /// Takes the lock to change the value, to the end of the `with` that
  /// holds it: `with w = config.write()` binds `w` to a pointer to the
  /// value. Usable only as a `with` value.
  public fun write(): RwLocked<T> => RwLocked(held: RwHeld.take(this.words, true), value: this.cell)

  /// Runs `f` with the value locked for reading — other readers may run at
  /// the same time — and returns what it returns. `f` cannot suspend.
  public fun withRead<R>(f: fun(*T): R): R {
    with held = RwHeld.take(this.words, false)
    return f(this.cell)
  }

  /// Runs `f` with the value locked for writing — no other reader or
  /// writer at the same time — and returns what it returns. `f` cannot
  /// suspend.
  public fun withWrite<R>(f: fun(*T): R): R {
    with held = RwHeld.take(this.words, true)
    return f(this.cell)
  }

  /// A copy of the value, read under a read lock.
  public fun get(): T => this.withRead(p => *p)

  /// Replaces the value under the write lock.
  public fun set(value: T) {
    this.withWrite(p => {
      *p = value
    })
  }
}

/// How an atomic operation is ordered against the memory reads and writes
/// around it (D144). `SeqCst`, the default, is what a program that never
/// names an order gets: every task sees every atomic operation happen in
/// one order. The weaker ones are for lock-free code that pairs them:
///
/// - `Relaxed` — the operation itself is atomic, nothing more (a counter
///   only ever read at the end).
/// - `Acquire` — on a load: what the task that stored the value with
///   `Release` wrote before that store is visible after this load.
/// - `Release` — on a store: what this task wrote before it is visible to a
///   task that loads the value with `Acquire`.
/// - `AcqRel` — both, on an operation that reads and writes (`swap`, `add`,
///   `compareAndSet`).
///
/// A load takes `Relaxed`, `Acquire` or `SeqCst`; a store `Relaxed`,
/// `Release` or `SeqCst`; an order written for an operation that cannot take
/// it is a compile error.
public enum MemoryOrder {
  Relaxed
  Acquire
  Release
  AcqRel
  SeqCst
}

/// A value that tasks read and replace whole, each operation indivisible.
/// For a change that reads the value first, use `compareAndSet`, `update`
/// or, on an integer, `add`: a `load` followed by a `store` can lose another
/// task's store in between.
///
/// ```veles
/// val hits = Atomic(value: 0)
/// hits.add(1)                                  // from any number of tasks
/// if (state.compareAndSet(Phase.Idle, Phase.Running)) { … }
/// ```
///
/// An `Atomic` of a number, a `bool`, an enum or a pointer (nullable or
/// not) takes no lock: each operation is one processor instruction. Any
/// other value is guarded by a lock word, and its operations act as
/// `SeqCst` whatever `order:` says. Every operation takes an optional `order:` (MemoryOrder,
/// D144).
public struct Atomic<T> {
  private cell: *T
  private word: *i64 = newWord()

  init(value: T) {
    this.cell = &value
  }

  public fun load(order: MemoryOrder = MemoryOrder.SeqCst): T {
    if (atomicLockFree(this.cell)) {
      return atomicLoad(this.cell, order)
    }
    with held = Held.take(this.word)
    return *this.cell
  }

  public fun store(value: T, order: MemoryOrder = MemoryOrder.SeqCst) {
    if (atomicLockFree(this.cell)) {
      atomicStore(this.cell, value, order)
      return
    }
    with held = Held.take(this.word)
    *this.cell = value
  }

  /// Stores `value` and returns the value it replaced.
  public fun swap(value: T, order: MemoryOrder = MemoryOrder.SeqCst): T {
    if (atomicLockFree(this.cell)) {
      return atomicSwap(this.cell, value, order)
    }
    with held = Held.take(this.word)
    val old = *this.cell
    *this.cell = value
    return old
  }

  /// Stores `new` if the value is `expected`, as one step, and says whether
  /// it did. A float compares by its bits (so a NaN can be replaced); any
  /// other value as `==` does — a pointer by identity. `failure:` orders
  /// the load of a failed attempt: `Relaxed`, `Acquire` or `SeqCst`, no
  /// stronger than `order:` (by default the strongest load `order:` allows).
  public fun compareAndSet(expected: T, new: T, order: MemoryOrder = MemoryOrder.SeqCst, failure: MemoryOrder? = null): bool {
    if (atomicLockFree(this.cell)) {
      return atomicCompareAndSwap(this.cell, expected, new, order, failure ?: failureOrder(order))
    }
    with held = Held.take(this.word)
    if (*this.cell != expected) {
      return false
    }
    *this.cell = new
    return true
  }

  /// `compareAndSet` that returns the value it found: `expected` (as
  /// `compareAndSet` compares) when it stored `new`, the newer value when it
  /// did not — which is the next attempt's `expected` in a loop.
  public fun compareExchange(expected: T, new: T, order: MemoryOrder = MemoryOrder.SeqCst, failure: MemoryOrder? = null): T {
    if (atomicLockFree(this.cell)) {
      return atomicCompareExchange(this.cell, expected, new, order, failure ?: failureOrder(order))
    }
    with held = Held.take(this.word)
    val found = *this.cell
    if (found == expected) {
      *this.cell = new
    }
    return found
  }

  /// Replaces the value with `f` of it, as one step, and returns the new
  /// value: `counter.update(n => n * 2)`. When tasks update at the same
  /// moment, `f` may run more than once — with the newer value each time —
  /// so it should only compute, not print or change other state.
  public fun update(f: fun(T): T, order: MemoryOrder = MemoryOrder.SeqCst): T {
    if (atomicLockFree(this.cell)) {
      val failure = failureOrder(order)
      loop {
        val old = atomicLoad(this.cell, failure)
        val next = f(old)
        if (atomicCompareAndSwap(this.cell, old, next, order, failure)) {
          return next
        }
      }
    }
    with held = Held.take(this.word)
    val next = f(*this.cell)
    *this.cell = next
    return next
  }
}

// The integer operations (D144): `add` and `sub` return the new value and
// wrap on overflow, as an atomic add does on every machine; `fetchAnd`,
// `fetchOr` and `fetchXor` return the old one, to test the bits they changed.

extend Atomic<i8> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: i8, order: MemoryOrder = MemoryOrder.SeqCst): i8 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: i8, order: MemoryOrder = MemoryOrder.SeqCst): i8 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: i8, order: MemoryOrder = MemoryOrder.SeqCst): i8 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: i8, order: MemoryOrder = MemoryOrder.SeqCst): i8 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: i8, order: MemoryOrder = MemoryOrder.SeqCst): i8 => atomicXor(this.cell, mask, order)
}

extend Atomic<i16> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: i16, order: MemoryOrder = MemoryOrder.SeqCst): i16 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: i16, order: MemoryOrder = MemoryOrder.SeqCst): i16 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: i16, order: MemoryOrder = MemoryOrder.SeqCst): i16 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: i16, order: MemoryOrder = MemoryOrder.SeqCst): i16 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: i16, order: MemoryOrder = MemoryOrder.SeqCst): i16 => atomicXor(this.cell, mask, order)
}

extend Atomic<i32> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: i32, order: MemoryOrder = MemoryOrder.SeqCst): i32 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: i32, order: MemoryOrder = MemoryOrder.SeqCst): i32 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: i32, order: MemoryOrder = MemoryOrder.SeqCst): i32 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: i32, order: MemoryOrder = MemoryOrder.SeqCst): i32 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: i32, order: MemoryOrder = MemoryOrder.SeqCst): i32 => atomicXor(this.cell, mask, order)
}

extend Atomic<i64> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: i64, order: MemoryOrder = MemoryOrder.SeqCst): i64 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: i64, order: MemoryOrder = MemoryOrder.SeqCst): i64 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: i64, order: MemoryOrder = MemoryOrder.SeqCst): i64 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: i64, order: MemoryOrder = MemoryOrder.SeqCst): i64 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: i64, order: MemoryOrder = MemoryOrder.SeqCst): i64 => atomicXor(this.cell, mask, order)
}

extend Atomic<isize> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: isize, order: MemoryOrder = MemoryOrder.SeqCst): isize => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: isize, order: MemoryOrder = MemoryOrder.SeqCst): isize => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: isize, order: MemoryOrder = MemoryOrder.SeqCst): isize => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: isize, order: MemoryOrder = MemoryOrder.SeqCst): isize => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: isize, order: MemoryOrder = MemoryOrder.SeqCst): isize => atomicXor(this.cell, mask, order)
}

extend Atomic<u8> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: u8, order: MemoryOrder = MemoryOrder.SeqCst): u8 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: u8, order: MemoryOrder = MemoryOrder.SeqCst): u8 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: u8, order: MemoryOrder = MemoryOrder.SeqCst): u8 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: u8, order: MemoryOrder = MemoryOrder.SeqCst): u8 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: u8, order: MemoryOrder = MemoryOrder.SeqCst): u8 => atomicXor(this.cell, mask, order)
}

extend Atomic<u16> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: u16, order: MemoryOrder = MemoryOrder.SeqCst): u16 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: u16, order: MemoryOrder = MemoryOrder.SeqCst): u16 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: u16, order: MemoryOrder = MemoryOrder.SeqCst): u16 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: u16, order: MemoryOrder = MemoryOrder.SeqCst): u16 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: u16, order: MemoryOrder = MemoryOrder.SeqCst): u16 => atomicXor(this.cell, mask, order)
}

extend Atomic<u32> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: u32, order: MemoryOrder = MemoryOrder.SeqCst): u32 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: u32, order: MemoryOrder = MemoryOrder.SeqCst): u32 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: u32, order: MemoryOrder = MemoryOrder.SeqCst): u32 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: u32, order: MemoryOrder = MemoryOrder.SeqCst): u32 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: u32, order: MemoryOrder = MemoryOrder.SeqCst): u32 => atomicXor(this.cell, mask, order)
}

extend Atomic<u64> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: u64, order: MemoryOrder = MemoryOrder.SeqCst): u64 => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: u64, order: MemoryOrder = MemoryOrder.SeqCst): u64 => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: u64, order: MemoryOrder = MemoryOrder.SeqCst): u64 => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: u64, order: MemoryOrder = MemoryOrder.SeqCst): u64 => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: u64, order: MemoryOrder = MemoryOrder.SeqCst): u64 => atomicXor(this.cell, mask, order)
}

extend Atomic<usize> {
  /// Adds `n` and returns the new value, wrapping on overflow.
  public fun add(n: usize, order: MemoryOrder = MemoryOrder.SeqCst): usize => atomicAdd(this.cell, n, order) +% n
  /// Subtracts `n` and returns the new value, wrapping on overflow.
  public fun sub(n: usize, order: MemoryOrder = MemoryOrder.SeqCst): usize => atomicSub(this.cell, n, order) -% n
  /// Keeps only the bits also set in `mask`; returns the old value.
  public fun fetchAnd(mask: usize, order: MemoryOrder = MemoryOrder.SeqCst): usize => atomicAnd(this.cell, mask, order)
  /// Sets the bits set in `mask`; returns the old value.
  public fun fetchOr(mask: usize, order: MemoryOrder = MemoryOrder.SeqCst): usize => atomicOr(this.cell, mask, order)
  /// Flips the bits set in `mask`; returns the old value.
  public fun fetchXor(mask: usize, order: MemoryOrder = MemoryOrder.SeqCst): usize => atomicXor(this.cell, mask, order)
}

// The load a failed compare-and-set makes for `order` when `failure:` is
// not given: the strongest a load may be without exceeding it.
fun failureOrder(order: MemoryOrder): MemoryOrder => when (order) {
  MemoryOrder.Release => MemoryOrder.Relaxed
  MemoryOrder.AcqRel  => MemoryOrder.Acquire
  else                => order
}

/// A value built once, on first use, by whichever task asks first (D146):
/// a table, a compiled pattern, a client — often a module-level `val`.
///
/// ```veles
/// val table = Lazy(init: () => buildTable())
/// // anywhere, from any task:
/// val t = table.get()
/// ```
///
/// Tasks that ask while it is being built wait for it. A panic in `init`
/// is the panic of the `get` that ran it, and every later `get` panics
/// with the same message. `init` cannot suspend; it runs under the
/// `Lazy`'s lock, so calling `get` on the same `Lazy` inside it panics.
public struct Lazy<T> {
  private build: sendable fun(): T
  private cell:  *LazyCell<T> = &LazyCell(value: null, failure: null, done: false)
  private word:  *i64 = newWord()
  private ready: Atomic<bool> = Atomic(value: false)

  init(init: sendable fun(): T) {
    this.build = init
  }

  /// The value, built by `init` on the first call.
  @caller_location
  public fun get(): T {
    if (!this.ready.load(order: MemoryOrder.Acquire)) {
      this.fill()
    }
    this.cell.value ?: panic("Lazy: no value after init")
  }

  /// Whether the value has been built.
  public fun isReady(): bool => this.ready.load()

  @caller_location
  fun fill() {
    with held = Held.take(this.word)
    val failure = this.cell.failure
    if (failure != null) panic(failure)
    if (this.ready.load(order: MemoryOrder.Relaxed)) return
    with building = LazyBuild(cell: this.cell)
    this.cell.value = this.build()
    this.cell.done = true
    this.ready.store(true, order: MemoryOrder.Release)
  }
}

struct LazyCell<T> {
  var value:   T?
  var failure: string?  // the message of init's panic
  var done:    bool
}

// While init runs: a panic in it leaves through this close, which keeps the
// message for every later get.
struct LazyBuild<T> {
  cell: *LazyCell<T>

  implement Closeable {
    fun close() {
      if (this.cell.done) return
      var text = ""
      var panicking: i64 = 0
      // SAFETY: veles_panic_current writes a string the collector owns into
      // the local it is given, or nothing
      unsafe {
        panicking = veles_panic_current(&text)
      }
      if (panicking != 0) this.cell.failure = text
    }
  }
}

// a lock word of its own, free
fun newWord(): *i64 {
  var word: i64 = 0
  &word
}
