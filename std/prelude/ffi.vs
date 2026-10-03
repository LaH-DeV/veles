// Prelude — lending a list to C for the length of a call (D69). The rest of
// the boundary is `std/ffi`.

extend<T: CLayout> List<T> {
  /// Hands `f` a pointer to this list's own elements — no copy — for C
  /// to read (or, for a `MutableList`, write) during `f`:
  ///
  /// ```veles
  /// val sent = bytes.withRaw(p => unsafe { send(sock, p, bytes.len(), 0) })
  /// ```
  ///
  /// The pointer is valid only while `f` runs: the list may move to new
  /// storage when it grows, and nothing keeps it alive afterwards. C must
  /// not keep it, and `f` must not change the list's length. The type
  /// cannot say so, which is why what `f` does with it is `unsafe`.
  public fun withRaw<R, E>(f: fun(*raw T): R throws E): R throws E {
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
