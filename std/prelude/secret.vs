/// A password, a key, a connection string: text or bytes that must not
/// end up in a log line, a JSON body or a crash report (D112).
///
/// ```veles
/// struct Config {
///   port: i64 = 8080
///   databaseUrl: Secret<string>
///   implement Decodable
/// }
/// println("db: ${cfg.databaseUrl}")              // db: [redacted]
/// val conn = try pg.connect(cfg.databaseUrl.expose())
/// ```
///
/// - `expose()` is the only way to the value: a fresh copy each time, so
///   every use of the secret is one grep away.
/// - Printing, interpolation and `expect` show `[redacted]`.
/// - Not `Encodable`: a `Secret` cannot be written to JSON or a log field
///   by accident. It is `Decodable`, so a config file or the environment
///   fills it.
/// - `==` takes the same time whatever the contents (only the lengths may
///   answer early); a `Secret` is not `Hashable` or `Comparable`.
/// - Wiped twice: `close()` zeroes the bytes at once (and `expose()` then
///   panics), and the collector zeroes them when it frees them.
///
/// Honest limit: what `expose()` returns, and the text a `Secret` was made
/// from, are ordinary memory — keep them short-lived.
public struct Secret<T> {
  private bytes:  List<u8>
  private closed: Atomic<bool> = Atomic(value: false)

  /// A `Secret` holding a private copy of `value` (a `string` or a
  /// `List<u8>`).
  public static fun of(value: T): Secret<T> = Secret(bytes: secretOf(value))

  /// A fresh copy of the value. Panics once the secret is closed.
  @caller_location
  public fun expose(): T {
    if (this.closed.load()) panic("Secret.expose: the secret was closed")
    secretExpose<T>(this.bytes)
  }

  /// The length in bytes, which is not secret.
  public fun len(): i64 = this.bytes.len()

  implement Display {
    fun toString(): string = "[redacted]"
  }

  implement Equatable {
    /// Constant time in the contents: every byte is read whatever the
    /// first difference is; different lengths answer at once.
    fun equals(other: Secret<T>): bool {
      if (this.bytes.len() != other.bytes.len()) return false
      var diff: u8 = 0
      loop ((a, b) in this.bytes.zip(other.bytes)) {
        diff = diff | (a ^ b)
      }
      diff == 0
    }
  }

  implement Closeable {
    /// Zeroes the bytes now; `expose()` panics from here on. A second
    /// close does nothing.
    fun close() {
      if (!this.closed.swap(true)) secretWipe(this.bytes)
    }
  }
}

implement<T: Decodable> Decodable for Secret<T> {
  static fun decode(from: Decoder): Secret<T> throws DecodeError = Secret.of(try T.decode(from))
}
