// Prelude — the common face of every error (D4). A type is an error when
// it is declared with `error Name { ... }` (sugar for a struct plus an impl
// of this trait) or given an explicit `implement Error for`. Only errors can be
// thrown; an error union exposes message() directly.

/// Which I/O failure an `IoError` is, portably (D76): the runtime maps
/// errno, and the Winsock codes on Windows, to these. `Other` is anything
/// they do not name; `code` still has the number.
public enum IoKind {
  Other = 0
  /// No such file, directory or host.
  NotFound
  /// The system refused: permissions, or an operation not allowed.
  PermissionDenied
  /// Creating what is already there.
  AlreadyExists
  /// A path component is a file where a directory was needed.
  NotADirectory
  /// A directory where a file was needed.
  IsADirectory
  /// Removing a directory that still has entries.
  DirectoryNotEmpty
  /// Nobody is listening at the address.
  ConnectionRefused
  /// The peer reset the connection.
  ConnectionReset
  /// The connection was aborted on this side.
  ConnectionAborted
  /// The operation timed out.
  TimedOut
  /// The address is already bound.
  AddressInUse
  /// The address is not this machine's.
  AddressNotAvailable
  /// Writing to a connection or pipe the other end closed.
  BrokenPipe
  /// A signal interrupted the call.
  Interrupted
  /// An argument the system call refused.
  InvalidInput
  /// Data that is not in the expected form (text that is not UTF-8).
  InvalidData
}

public trait Error {
  fun message(): string => "$this"
}

/// A failed operating-system call (files, processes, environment, network):
/// `kind` says which failure, the same on every platform (D76); `detail` is
/// the system's description, `path` the file, command or address involved,
/// and `code` the platform's own error number, for logs.
///
/// ```veles
/// val text = fs.readFile(path) catch (e) {
///   if (e.kind == IoKind.NotFound) return "{}"
///   throw e
/// }
/// ```
public error IoError {
  public path:   string
  public code:   i64
  public detail: string
  public kind:   IoKind = IoKind.Other
  fun message(): string => if (this.path.isEmpty()) this.detail else "${this.detail}: ${this.path}"
}
