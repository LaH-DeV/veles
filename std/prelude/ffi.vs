// Prelude — lending a list to C for the length of a call (D69). The rest of
// the boundary is `std/ffi`.

extend<T: CLayout> List<T> {
  /// Hands `f` a pointer to this list's elements — no copy — for C to
  /// **read** during `f`:
  ///
  /// ```veles
  /// val sent = bytes.withRaw(p => unsafe { send(sock, p, bytes.len(), 0) })
  /// ```
  ///
  /// The storage may be shared with copies of the list (D231), so C must
  /// not write through the pointer; `withRawMut` is for that. The pointer
  /// is valid only while `f` runs: the list may move to new storage when it
  /// grows, and nothing keeps it alive afterwards. C must not keep it, and
  /// `f` must not change the list's length. The type cannot say so, which
  /// is why what `f` does with it is `unsafe`.
  public fun withRaw<R, E>(f: fun(*raw T): R throws E): R throws E {
    val p = listReadData(this)
    try f(p)
  }

  /// Hands `f` a pointer to this list's own elements for C to **write**
  /// during `f` (`memset`, `qsort`, a `read` into a buffer). The list is
  /// made the only holder of its storage first — a list that was shared is
  /// copied then, once — so the writes reach this list and no copy of it
  /// (D231). The rest is as for `withRaw`.
  public fun withRawMut<R, E>(f: fun(*raw T): R throws E): R throws E {
    val p = listRawData(this)
    try f(p)
  }
}

extend<T: CLayout, const N: i64> Array<T, N> {
  /// Hands `f` a pointer to this array's own elements — no copy — for C to
  /// read during `f`, or to write when the array is a `var`. The pointer is
  /// valid only while `f` runs; what `f` does with it is `unsafe`.
  public fun withRaw<R, E>(f: fun(*raw T): R throws E): R throws E {
    val p = arrayRawData(this)
    try f(p)
  }
}
