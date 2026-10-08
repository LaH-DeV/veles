// Guarding a router (D99): one hook that can answer a request instead of the
// handlers behind it, and the two HTTP schemes almost every API uses.

use base64
use log as logs { field }

/// A check in front of everything the router answers. `check` sees the
/// request first: `null` lets it through to the handlers, a `Response`
/// answers it instead, and a thrown `Fail` is the answer too (any other error
/// is a 500, as in a handler).
///
/// ```veles
/// app.wrap(http.guard(req =>
///   if (req.path.startsWith("/admin") && req.header("x-admin") == null) http.Response.text("no", status: http.Status.forbidden) else null
/// ))
/// ```
///
/// A guard wraps the whole router, its 404s included; a check that only
/// concerns some routes tests the path itself (route groups will make that
/// a wrapper on the group).
public fun guard<E>(check: sendable fun(Request): Response? suspends throws E | Fail): Middleware =>
  screen(req => Screen(req, answer: try check(req)))

/// HTTP Basic authentication: every request needs `Authorization: Basic
/// <user:password>`, and `verify(user, password)` says whether that pair is
/// good. A request without one, or with one `verify` refuses, is answered 401
/// with the `WWW-Authenticate` challenge that makes a browser ask for it.
///
/// A request that passes carries the user name in `x-remote-user`
/// (`Header.remoteUser`), replacing whatever the client sent under that name,
/// so a handler can trust it. Compare secrets with `crypto.equalBytes`, not
/// `==`: the time `==` takes says how many leading bytes matched.
///
/// ```veles
/// app.wrap(http.basicAuth("admin", (user, pass) =>
///   user == "root" && crypto.equalBytes(pass.bytes(), secret.bytes())
/// ))
/// ```
///
/// Basic sends the password with every request, so serve it over TLS only.
/// A `realm` with a quote or a control character is a panic at the caller.
@caller_location
public fun basicAuth<E>(realm: string, verify: sendable fun(string, string): bool suspends throws E | Fail): Middleware {
  checkRealm(realm)
  val challenge = "Basic realm=\"$realm\", charset=\"UTF-8\""
  screen(req => {
    val (user, pass) = basicCredentials(req) ?: return Screen(req, answer: refuse(challenge))
    if (!(try verify(user, pass))) return Screen(req, answer: refuse(challenge))
    Screen(req: req.withHeader(Header.remoteUser, user), answer: null)
  })
}

/// Bearer-token authentication (API keys, JWTs): every request needs
/// `Authorization: Bearer <token>`, and `verify(token)` says whether the token
/// is good. Without a token the answer is 401 with `WWW-Authenticate: Bearer`;
/// with one `verify` refuses it says `error="invalid_token"` too (RFC 6750).
///
/// ```veles
/// app.wrap(http.bearer(token => crypto.equalBytes(token.bytes(), apiKey.bytes())))
/// ```
///
/// Compare secrets with `crypto.equalBytes`, not `==`. A handler that needs
/// the token's claims reads the `Authorization` header itself, or is written
/// with `guard`.
@caller_location
public fun bearer<E>(verify: sendable fun(string): bool suspends throws E | Fail, realm: string? = null): Middleware {
  val named = if (realm == null) "" else " realm=\"${realm}\""
  if (realm != null) checkRealm(realm)
  val challenge = "Bearer$named"
  val invalid = if (realm == null) "Bearer error=\"invalid_token\"" else "Bearer realm=\"${realm}\", error=\"invalid_token\""
  screen(req => {
    val token = bearerToken(req) ?: return Screen(req, answer: refuse(challenge))
    if (!(try verify(token))) return Screen(req, answer: refuse(invalid))
    Screen(req, answer: null)
  })
}

// What a check decides: the request to pass on (a check may add to it), or the
// answer that replaces the handlers.
struct Screen {
  req:    Request
  answer: Response?
}

fun screen<E>(check: sendable fun(Request): Screen suspends throws E | Fail): Middleware => (next => req => when (check(req)) {
  is Ok(s)  => s.answer ?: next(s.req)
  is Err(e) => when (e) {
    is Fail => Response.text(e.text, status: e.status)
    else    => {
      logs.error("guard failed", field("method", "${req.method}"), field("path", req.path), field("error", e.message()))
      val status = Status.internalServerError
      Response.text(body: status.reason(), status: status)
    }
  }
})

fun refuse(challenge: string): Response =>
  Response.text("unauthorized", status: Status.unauthorized).withHeader(Header.wwwAuthenticate, challenge)

@caller_location
fun checkRealm(realm: string) {
  if (realm.bytes().any(b => b < 32 || b == 127 || b == '"' || b == '\\')) panic("realm '$realm' cannot hold a quote, a backslash or a control character")
}

// the credentials of `Authorization: <scheme> <rest>`, when the scheme is `scheme`
fun credentialsOf(req: Request, scheme: string): string? {
  val header = req.header(Header.authorization) ?: return null
  val (given, rest) = header.trim().splitOnce(" ") ?: return null
  if (given.toLower() != scheme) return null
  val value = rest.trim()
  if (value.isEmpty()) null else value
}

fun basicCredentials(req: Request): (string, string)? {
  val encoded = credentialsOf(req, "basic") ?: return null
  val decoded = base64.decode(encoded) catch {
    return null
  }
  val text = decoded.decodeUtf8() ?: return null
  text.splitOnce(":")
}

fun bearerToken(req: Request): string? {
  val token = credentialsOf(req, "bearer") ?: return null
  if (token.contains(" ")) null else token
}
