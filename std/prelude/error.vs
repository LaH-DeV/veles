// Prelude — the common face of every error (D4). A type is an error when
// it is declared with `error Name { ... }` (sugar for a struct plus an impl
// of this trait) or given an explicit `impl Error for`. Only errors can be
// thrown; an error union exposes message() directly.

pub trait Error {
  fun message(): string = "$self"
}
