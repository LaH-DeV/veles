// Prelude — the common face of every error (D4). A type is an error when
// it is declared with `error Name { ... }` (sugar for a struct plus an impl
// of this trait) or given an explicit `impl Error for`. Only errors can be
// thrown; an error union exposes message() directly.

pub trait Error {
  fun message(): string = "$self"
}

/// A failed operating-system call (files, processes, environment): `detail`
/// is the system's description, `path` the file or command involved, and
/// `code` the platform error number.
pub error IoError {
  pub path:   string
  pub code:   i64
  pub detail: string
  fun message(): string = if (self.path.isEmpty()) self.detail else "${self.detail}: ${self.path}"
}
