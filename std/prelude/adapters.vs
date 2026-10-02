// Prelude — the eager adapters of List and Map (plan B8). Each runs its
// function on every element in order and is generic over what the function
// does: it throws what the function throws (`xs.map(a => try parse(a))` is
// a `List<A> throws ParseError`, a `Result` until `try`), and it suspends
// only when the function does (D116), so `xs.map(x => x * 2)` is a plain
// call that runs under a lock, and `urls.map(u => fetch(u))` waits for each
// fetch in turn (`mapConcurrent` is the concurrent form).

extend<T> List<T> {
  /// What `f` returns for each element, in order: `[1, 2].map(x => x * 10)`
  /// is `[10, 20]`. Eager; `xs.iter().map(f)` is the lazy form (D46).
  public fun map<U, E>(f: fun(T): U suspends throws E): List<U> throws E {
    val out: MutableList<U> = []
    out.reserve(this.len())
    loop (x in this) {
      out.push(try f(x))
    }
    out
  }

  /// The elements `pred` accepts, in order.
  public fun filter<E>(pred: fun(T): bool suspends throws E): List<T> throws E {
    val out: MutableList<T> = []
    loop (x in this) {
      if (try pred(x)) out.push(x)
    }
    out
  }

  /// Runs `f` on each element, in order.
  public fun forEach<E>(f: fun(T): () suspends throws E) throws E {
    loop (x in this) {
      try f(x)
    }
  }

  /// `f` applied to an accumulator and each element in turn, from the left,
  /// starting at `initial`: `f(f(f(initial, x0), x1), x2)`;
  /// `[1, 2, 3].fold(0, (sum, x) => sum + x)` is 6.
  public fun fold<A, E>(initial: A, f: fun(A, T): A suspends throws E): A throws E {
    var acc = initial
    loop (x in this) {
      acc = try f(acc, x)
    }
    acc
  }

  /// Whether `pred` accepts some element; it stops at the first. `false`
  /// for an empty list.
  public fun any<E>(pred: fun(T): bool suspends throws E): bool throws E {
    loop (x in this) {
      if (try pred(x)) return true
    }
    false
  }

  /// Whether `pred` accepts every element; it stops at the first it does
  /// not. `true` for an empty list.
  public fun all<E>(pred: fun(T): bool suspends throws E): bool throws E {
    loop (x in this) {
      if (!(try pred(x))) return false
    }
    true
  }

  /// The first element `pred` accepts, or `null`.
  public fun find<E>(pred: fun(T): bool suspends throws E): T? throws E {
    loop (x in this) {
      if (try pred(x)) return x
    }
    null
  }
}

extend<K, V> Map<K, V> {
  /// Runs `f` on each key and value. The entries are read when the call
  /// starts, so `f` may change the map.
  public fun forEach<E>(f: fun(K, V): () suspends throws E) throws E {
    loop ((k, v) in this.entries()) {
      try f(k, v)
    }
  }

  /// The same keys, each with what `f` returns for its value.
  public fun mapValues<U, E>(f: fun(V): U suspends throws E): Map<K, U> throws E {
    val out: MutableMap<K, U> = [:]
    out.reserve(this.len())
    loop ((k, v) in this.entries()) {
      out.set(k, try f(v))
    }
    out
  }

  /// The entries `pred` accepts.
  public fun filter<E>(pred: fun(K, V): bool suspends throws E): Map<K, V> throws E {
    val out: MutableMap<K, V> = [:]
    loop ((k, v) in this.entries()) {
      if (try pred(k, v)) out.set(k, v)
    }
    out
  }
}

extend<K, V> MutableMap<K, V> {
  /// The value under `key`; when there is none, what `make` returns, stored
  /// under `key` first.
  public fun getOrPut<E>(key: K, make: fun(): V suspends throws E): V throws E {
    if (val found = this.get(key)) return found
    val made = try make()
    this.set(key, made)
    made
  }
}
