// What can go wrong talking to a database, in one error: `kind` says where
// the failure is, and for the ones the server reports — a constraint, a
// syntax error, a deadlock — `code` is PostgreSQL's SQLSTATE, the stable
// five characters to branch on.

/// Where a failure is.
public enum ErrorKind {
  /// The connection could not be made, or broke: a refused port, a reset, a failed TLS handshake.
  Connection
  /// The server closed the connection, or the pool was closed.
  Closed
  /// The statement or the connection ran past its time. The connection is dropped.
  Timeout
  /// The login was refused, or the server could not prove who it is.
  Auth
  /// The server refused the statement: see `code`.
  Server
  /// The server sent something that is not the protocol.
  Protocol
  /// The URL, or an argument, is not one the driver accepts.
  Config
}

/// A failed database call.
///
/// ```veles
/// val inserted = pool.exec(sql"insert into users (email) values (${email})") catch (e) {
///   if (e.isUniqueViolation()) return Response.text("taken", status: Status.conflict)
///   throw e
/// }
/// ```
public error DbError {
  public kind: ErrorKind
  public text: string
  /// The SQLSTATE of a `Server` failure (`23505`, `42P01`, …), or empty.
  public code:     string = ""
  public detail:   string = ""
  public hint:     string = ""
  public severity: string = ""

  fun message(): string {
    val head = if (this.code.isEmpty()) this.text else "${this.text} (${this.code})"
    if (this.detail.isEmpty()) head else "$head: ${this.detail}"
  }

  /// A unique or primary key constraint refused the row (23505).
  public fun isUniqueViolation(): bool = this.code == "23505"

  /// A foreign key refused it (23503).
  public fun isForeignKeyViolation(): bool = this.code == "23503"

  /// Any constraint: not null, check, unique, foreign key, exclusion (class 23).
  public fun isConstraintViolation(): bool = this.code.startsWith("23")

  /// A deadlock (40P01) or a serialization failure (40001): the transaction
  /// rolled back and may be tried again from the start.
  public fun isRetryable(): bool = this.code == "40001" || this.code == "40P01"

  /// The statement ran past its `statement_timeout` or was cancelled (57014).
  public fun isCancelled(): bool = this.code == "57014"
}
