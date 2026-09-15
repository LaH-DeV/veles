// Prelude — resources (D43). `with (r = expr) { }` closes r on every exit
// path; cleanup is implicitly non-cancellable (D47).

pub trait Closeable {
  mut fun close()
}
