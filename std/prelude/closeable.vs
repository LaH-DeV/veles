// Prelude — resources (D43). `with r = expr` closes r when its block ends
// and `with (r = expr) { }` at that `}` — on every exit path either way
// (D100); cleanup is implicitly non-cancellable (D47).

public trait Closeable {
  fun close()
}
