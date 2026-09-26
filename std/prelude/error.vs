// Prelude — the common face of every error (D4). A type is an error when
// it is declared with `error Name { ... }` (sugar for a struct plus an impl
// of this trait) or given an explicit `implement Error for`. Only errors can be
// thrown; an error union exposes message() directly.

public trait Error {
  fun message(): string = "$this"
}

/// A failed operating-system call (files, processes, environment): `detail`
/// is the system's description, `path` the file or command involved, and
/// `code` the platform error number.
public error IoError {
  public path:   string
  public code:   i64
  public detail: string
  fun message(): string = if (this.path.isEmpty()) this.detail else "${this.detail}: ${this.path}"
}
