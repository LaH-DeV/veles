// Prelude — resources (D43). `with (r = expr) { }` closes r on every exit
// path; cleanup is implicitly non-cancellable (D47).

public trait Closeable {
  mut fun close()
}
