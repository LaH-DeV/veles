// The connection URL: `postgres://user:password@host:5432/database?sslmode=verify-full`.
// It arrives as a `Secret` (it holds the password), is read here and never
// printed: the pieces that are shown — host, database — leave the password
// behind.

/// How the connection is secured.
enum Tls {
  // plain TCP: only allowed by default for the machine itself
  Off
  // TLS, certificate and name verified against the system's roots or `sslrootcert`
  Verified
  // TLS without checking the certificate: encrypted, but anyone can be the server
  Insecure
}

/// Where and as whom to connect.
struct Target {
  host:        string
  port:        i64
  user:        string
  password:    Secret<string>
  database:    string
  tls:         Tls
  rootCert:    string
  application: string

  // what a log may say about the target: no user, no password
  fun shown(): string => "${this.host}:${this.port}/${this.database}"
}

fun configError(text: string): DbError => DbError(kind: ErrorKind.Config, text)

fun isLoopback(host: string): bool => host == "localhost" || host == "127.0.0.1" || host == "::1" || host.startsWith("127.")

// %XX escapes and '+' are not special in the userinfo and path, only %XX is
fun unescape(text: string): string? {
  if (!text.contains("%")) return text
  val out: MutableList<u8> = []
  val bytes = text.bytes()
  var i = 0
  loop (i < bytes.len()) {
    val b = bytes.at(i)
    if (b == 37) {
      val hi = hexValue(bytes.at(i + 1) ?: 0)
      val lo = hexValue(bytes.at(i + 2) ?: 0)
      if (hi < 0 || lo < 0) return null
      out.push((hi * 16 + lo).wrapU8())
      i += 3
    } else {
      out.push(b)
      i += 1
    }
  }
  out.decodeUtf8()
}

fun hexValue(b: u8): i64 {
  if (b >= 48 && b <= 57) return (b - 48).toI64()
  if (b >= 97 && b <= 102) return (b - 87).toI64()
  if (b >= 65 && b <= 70) return (b - 55).toI64()
  -1
}

fun parseTarget(url: Secret<string>, application: string): Target throws DbError {
  val text = url.expose()
  val (scheme, rest) = text.splitOnce("://") else throw configError("the database URL must start with postgres://")
  if (scheme != "postgres" && scheme != "postgresql") throw configError("the database URL must start with postgres:// (got '$scheme://')")
  val (beforeQuery, query) = if (rest.contains("?")) (rest.splitOnce("?") ?: (rest, "")) else (rest, "")
  val (authority, path) = if (beforeQuery.contains("/")) (beforeQuery.splitOnce("/") ?: (beforeQuery, "")) else (beforeQuery, "")
  var user = ""
  var password = ""
  var hostPort = authority
  if (authority.contains("@")) {
    val at = authority.lastIndexOf("@")
    val info = authority.substring(0, at) ?: ""
    hostPort = authority.substring(at + 1, authority.len()) ?: ""
    if (info.contains(":")) {
      val (u, p) = info.splitOnce(":") ?: (info, "")
      user = u
      password = p
    } else {
      user = info
    }
  }
  var host = hostPort
  var port: i64 = 5432
  if (hostPort.startsWith("[")) {
    val close = hostPort.indexOf("]")
    if (close < 0) throw configError("the database URL has an unclosed [ in its host")
    host = hostPort.substring(1, close) ?: ""
    val after = hostPort.substring(close + 1, hostPort.len()) ?: ""
    if (after.startsWith(":")) port = (after.substring(1, after.len()) ?: "").toInt() ?: throw configError("the database URL has a port that is not a number")
  } else if (hostPort.contains(":")) {
    val (h, p) = hostPort.splitOnce(":") ?: (hostPort, "")
    host = h
    port = p.toInt() ?: throw configError("the database URL has a port that is not a number")
  }
  if (port < 1 || port > 65535) throw configError("the database URL has a port outside 1..65535")
  if (host.isEmpty()) host = "localhost"
  val userText = unescape(user) ?: throw configError("the user in the database URL has a bad %-escape")
  val passText = unescape(password) ?: throw configError("the password in the database URL has a bad %-escape")
  val database = unescape(path) ?: throw configError("the database name in the URL has a bad %-escape")
  var mode: string? = null
  var rootCert = ""
  var app = application
  loop (pair in query.split("&")) {
    if (pair.isEmpty()) continue
    val (k, v) = pair.splitOnce("=") ?: (pair, "")
    val value = unescape(v) ?: throw configError("the database URL has a bad %-escape in '$k'")
    when (k) {
      "sslmode"          => mode = value
      "sslrootcert"      => rootCert = value
      "application_name" => app = value
      else               => throw configError("the database URL has a parameter this driver does not know: '$k' (sslmode, sslrootcert and application_name are accepted)")
    }
  }
  val tls = when (mode) {
    null          => if (isLoopback(host)) Tls.Off else Tls.Verified
    "disable"     => Tls.Off
    "verify-full" => Tls.Verified
    "insecure"    => Tls.Insecure
    else          => throw configError("sslmode '${mode}' is not one of disable, verify-full, insecure")
  }
  if (userText.isEmpty()) throw configError("the database URL names no user")
  Target(host, port, user: userText, password: Secret.of(passText), database: if (database.isEmpty()) userText else database, tls, rootCert, application: app)
}
