/// JSON Web Tokens signed with HMAC (JWS compact serialization, RFC 7515
/// and RFC 7519): `header.payload.signature`, each part URL-safe base64.
///
/// ```veles
/// use jwt
///
/// val key = Secret.of(crypto.randomBytes(32))   // keep this; it is the whole secret
///
/// val token = try jwt.sign(jwt.Claims(
///   subject: "user-42",
///   issuer: "notes.example",
///   expiresAt: jwt.now() + 3600,
/// ), key)
///
/// when (val claims = jwt.verify(token, key, jwt.Options(issuer: "notes.example"))) {
///   is Ok  => io.println("hello ${claims.subject}")
///   is Err => io.println("rejected: ${claims.message} (${claims.reason})")
/// }
/// ```
///
/// ## What this module refuses
///
/// A JWT library is mostly a list of refusals, and the famous
/// vulnerabilities are all things a permissive one accepts:
///
/// - **The algorithm comes from you, not from the token.** `Options`
///   carries the expected `algorithm`; the header's `alg` must equal it.
///   Trusting the header is how `alg: none` and "verify an RS256 token as
///   HS256 with the public key as the secret" work.
/// - `crit` in the header is rejected: it means "you must understand this
///   extension", and this module understands none.
/// - The signature is compared in constant time (`crypto.Digest`).
/// - Base64url is strict (`std/base64`): no whitespace, no non-canonical
///   trailing bits, so one token has one encoding.
/// - `exp` is **required** by default; a token that never expires is
///   almost never what a server wants. `Options(requireExpiry: false)`
///   opts out.
/// - An HMAC key shorter than the digest is refused (RFC 7518 §3.2): a
///   password is not a key.
///
/// What it does not do: RSA and ECDSA signatures (`RS256`, `ES256`) need
/// bignum arithmetic or a native binding, and encryption (JWE) is a
/// different specification. `HS256` covers a server that issues its own
/// tokens.
use base64
use codec
use crypto
use json
use time

/// The HMAC algorithms of RFC 7518. The name on the wire is the member
/// name: `"HS256"`.
public enum Algorithm {
  HS256
  HS384
  HS512
}

/// Why a token was rejected. Worth branching on: `Expired` usually means
/// "refresh", everything else means "sign in again".
public enum Reason {
  /// Not three base64url parts, or the parts are not JSON objects.
  Malformed
  /// The header's `alg` is not one this module implements.
  UnsupportedAlgorithm
  /// The header's `alg` is not the one the caller expects — including
  /// `none`.
  AlgorithmMismatch
  /// The header asks for an extension (`crit`) that is not understood.
  UnsupportedExtension
  /// The signature does not match. The token was altered, or signed with
  /// another key.
  BadSignature
  /// `exp` has passed.
  Expired
  /// `nbf` has not arrived yet.
  NotYetValid
  /// `aud` does not contain the expected audience.
  WrongAudience
  /// `iss` is not the expected issuer.
  WrongIssuer
  /// A claim the caller requires is missing, or has the wrong type.
  MissingClaim
}

/// A token that will not be honoured.
public error Invalid {
  public message: string
  public reason:  Reason
}

/// The registered claims of RFC 7519 §4.1, plus everything else the token
/// carries in `extra`.
///
/// `expiresAt`, `notBefore` and `issuedAt` are **Unix seconds** (the
/// specification's NumericDate), not the milliseconds `std/time` works in
/// — `jwt.now()` is the reading to add to.
public struct Claims {
  /// `iss`
  public issuer: string? = null
  /// `sub` — who the token is about.
  public subject: string? = null
  /// `aud`. One string on the wire when there is exactly one.
  public audience: List<string> = []
  /// `exp`, in Unix seconds.
  public expiresAt: i64? = null
  /// `nbf`, in Unix seconds.
  public notBefore: i64? = null
  /// `iat`, in Unix seconds.
  public issuedAt: i64? = null
  /// `jti` — a unique id, for a replay list.
  public id: string? = null
  /// Every claim that is not one of the above.
  public extra: Map<string, codec.Value> = [:]

  /// A private claim by name: `claims.claim("role")?.asString()`.
  public fun claim(name: string): codec.Value? => this.extra.get(name)

  /// A private claim that should be text, or `null` when it is missing or
  /// is something else.
  public fun text(name: string): string? => this.extra.get(name)?.asString()

  /// A private claim that should be a whole number.
  public fun number(name: string): i64? => this.extra.get(name)?.asI64()

  /// A private claim that should be a boolean.
  public fun flag(name: string): bool? => this.extra.get(name)?.asBool()
}

/// What `verify` insists on. The defaults are the strict ones: HS256, an
/// `exp` that has not passed, no clock slack.
public struct Options {
  /// The algorithm the token must be signed with. Never read from the
  /// token itself.
  public algorithm: Algorithm = Algorithm.HS256
  /// When set, `aud` must contain it.
  public audience: string? = null
  /// When set, `iss` must equal it.
  public issuer: string? = null
  /// Seconds of tolerance on `exp` and `nbf`, for clocks that disagree.
  /// Sixty is a common setting; the default is none.
  public leeway: i64 = 0
  /// Whether a token without `exp` is acceptable.
  public requireExpiry: bool = true
  /// The moment to judge the token at, in Unix seconds. `null` is the
  /// clock; a fixed value is for tests.
  public now: i64? = null
}

/// The current time in Unix **seconds**, the unit `exp`, `nbf` and `iat`
/// are written in.
public fun now(): i64 => time.now().toSeconds()

// ---------------------------------------------------------------------------
// signing

/// A signed token for `claims`.
///
/// `keyId` goes into the header as `kid`, so that a verifier with several
/// keys can pick one (`readHeader` reads it back without trusting
/// anything else).
///
/// The key must be at least as long as the digest — 32 bytes for HS256, 48
/// for HS384, 64 for HS512 (RFC 7518 §3.2). A shorter one is a panic, not
/// an error: it is a configuration mistake, and no request should be
/// served with it.
public fun sign(
  claims: Claims,
  key: Secret<List<u8>>,
  algorithm: Algorithm = Algorithm.HS256,
  keyId: string? = null,
): string throws EncodeError {
  checkKey(key, algorithm, "jwt.sign")

  val header: MutableMap<string, codec.Value> = [:]
  header.set("alg", codec.VString(value: "$algorithm"))
  header.set("typ", codec.VString(value: "JWT"))
  val kid = keyId
  if (kid != null) header.set("kid", codec.VString(value: kid))

  val payload = claimsToObject(claims)
  val signing = base64.encodeUrl((try json.encode(codec.VObject(fields: header.toMap()))).bytes()) + "." +
    base64.encodeUrl(try json.encode(payload).bytes())
  signing + "." + mac(algorithm, key, signing).toBase64Url()
}

fun claimsToObject(claims: Claims): codec.Value {
  val out: MutableMap<string, codec.Value> = [:]
  loop ((name, v) in claims.extra) {
    out.set(name, v)
  }
  val iss = claims.issuer
  if (iss != null) out.set("iss", codec.VString(value: iss))
  val sub = claims.subject
  if (sub != null) out.set("sub", codec.VString(value: sub))
  // one audience is written as a string, several as a list (RFC 7519 4.1.3)
  when (claims.audience) {
    []    => { }
    [one] => out.set("aud", codec.VString(value: one))
    else  => out.set("aud", codec.VList(items: claims.audience.map(a => codec.VString(value: a))))
  }
  val exp = claims.expiresAt
  if (exp != null) out.set("exp", codec.VInt(value: exp))
  val nbf = claims.notBefore
  if (nbf != null) out.set("nbf", codec.VInt(value: nbf))
  val iat = claims.issuedAt
  if (iat != null) out.set("iat", codec.VInt(value: iat))
  val jti = claims.id
  if (jti != null) out.set("jti", codec.VString(value: jti))
  codec.VObject(fields: out.toMap())
}

// ---------------------------------------------------------------------------
// verifying

/// The claims of `token`, once its signature and its time claims have been
/// checked. Everything this refuses is listed at the top of the module.
public fun verify(token: string, key: Secret<List<u8>>, options: Options = Options()): Claims throws Invalid {
  checkKey(key, options.algorithm, "jwt.verify")

  val parts = token.split(".")
  val [headerPart, payloadPart, signaturePart] = parts else throw Invalid(
    message: "jwt: a compact token has three parts, found ${parts.len()}",
    reason: Reason.Malformed,
  )
  val headerText = try decodePart(headerPart, "header")
  val payloadText = try decodePart(payloadPart, "payload")
  val signature = try base64.decodeUrl(signaturePart) ?! Invalid(
    message: "jwt: the signature is not URL-safe base64",
    reason: Reason.Malformed,
  )

  // the header decides nothing except which key id was used
  val header = try parseObject(headerText, "header")
  if (header.get("crit") != null) {
    throw Invalid(
      message: "jwt: the header asks for an extension this module does not implement (crit)",
      reason: Reason.UnsupportedExtension,
    )
  }
  val alg = header.get("alg")?.asString() ?: throw Invalid(
    message: "jwt: the header has no 'alg'",
    reason: Reason.Malformed,
  )
  val expected = "${options.algorithm}"
  if (alg != expected) {
    val known = Algorithm.parse(alg)
    if (known == null) {
      throw Invalid(
        message: "jwt: 'alg' is '$alg'; this module implements HS256, HS384 and HS512",
        reason: Reason.UnsupportedAlgorithm,
      )
    }
    throw Invalid(message: "jwt: 'alg' is '$alg' but '$expected' was expected", reason: Reason.AlgorithmMismatch)
  }

  // the signature covers the first two parts exactly as they were written
  val signing = headerPart + "." + payloadPart
  if (mac(options.algorithm, key, signing) != crypto.Digest.of(signature)) {
    throw Invalid(message: "jwt: the signature does not match", reason: Reason.BadSignature)
  }

  val payload = try parseObject(payloadText, "payload")
  val claims = try readClaims(payload)
  try checkTime(claims, options)
  try checkParties(claims, options)
  claims
}

/// The header of `token`, without checking anything: for a verifier that
/// keeps several keys and needs the `kid` before it can pick one.
///
/// Nothing in here is trustworthy — it is unsigned text from the caller of
/// your API. Use it to look a key up, then `verify`.
public fun readHeader(token: string): codec.Value throws Invalid {
  val parts = token.split(".")
  val [headerPart, _, _] = parts else throw Invalid(
    message: "jwt: a compact token has three parts, found ${parts.len()}",
    reason: Reason.Malformed,
  )
  try parseObject(try decodePart(headerPart, "header"), "header")
}

fun decodePart(part: string, what: string): string throws Invalid {
  val bytes = try base64.decodeUrl(part) ?! Invalid(
    message: "jwt: the $what is not URL-safe base64",
    reason: Reason.Malformed,
  )
  bytes.decodeUtf8() ?: throw Invalid(
    message: "jwt: the $what is not UTF-8 text",
    reason: Reason.Malformed,
  )
}

fun parseObject(text: string, what: string): codec.Value throws Invalid {
  val v = try json.parse(text) ?! Invalid(message: "jwt: the $what is not JSON", reason: Reason.Malformed)
  if (v !is codec.VObject) {
    throw Invalid(message: "jwt: the $what is not a JSON object", reason: Reason.Malformed)
  }
  v
}

/// The registered claims out of a payload object; everything else lands in
/// `extra` untouched, as the `codec.Value` it was.
fun readClaims(payload: codec.Value): Claims throws Invalid {
  val fields = when (payload) {
    is codec.VObject => payload.fields
    else             => throw Invalid(
      message: "jwt: the payload is not a JSON object",
      reason: Reason.Malformed,
    )
  }
  var issuer: string? = null
  var subject: string? = null
  var id: string? = null
  var expiresAt: i64? = null
  var notBefore: i64? = null
  var issuedAt: i64? = null
  var who: List<string> = []
  val extra: MutableMap<string, codec.Value> = [:]
  loop ((name, v) in fields) {
    when (name) {
      "iss" => issuer = try text(v, "iss")
      "sub" => subject = try text(v, "sub")
      "jti" => id = try text(v, "jti")
      "exp" => expiresAt = try seconds(v, "exp")
      "nbf" => notBefore = try seconds(v, "nbf")
      "iat" => issuedAt = try seconds(v, "iat")
      "aud" => who = try audience(v)
      else  => extra.set(name, v)
    }
  }
  Claims(issuer, subject, audience: who, expiresAt, notBefore, issuedAt, id, extra: extra.toMap())
}

fun text(v: codec.Value, name: string): string throws Invalid =>
  v.asString() ?: throw Invalid(message: "jwt: '$name' is not a string", reason: Reason.MissingClaim)

/// A NumericDate: seconds since the epoch, which the specification allows
/// to be fractional — the fraction is dropped.
fun seconds(v: codec.Value, name: string): i64 throws Invalid {
  val whole = v.asI64()
  if (whole != null) return whole
  val fraction = v.asF64() ?: throw Invalid(
    message: "jwt: '$name' is not a number of seconds",
    reason: Reason.MissingClaim,
  )
  fraction.floor().toI64() ?: throw Invalid(
    message: "jwt: '$name' is out of range",
    reason: Reason.MissingClaim,
  )
}

/// `aud` is one string, or a list of them (RFC 7519 §4.1.3).
fun audience(v: codec.Value): List<string> throws Invalid {
  val one = v.asString()
  if (one != null) return [one]
  when (v) {
    is codec.VList => {
      val out: MutableList<string> = []
      loop (item in v.items) {
        out.push(item.asString() ?: throw Invalid(
          message: "jwt: 'aud' holds something that is not a string",
          reason: Reason.MissingClaim,
        ))
      }
      out.toList()
    }
    else           => throw Invalid(
      message: "jwt: 'aud' is neither a string nor a list of strings",
      reason: Reason.MissingClaim,
    )
  }
}

fun checkTime(claims: Claims, options: Options) throws Invalid {
  val at = options.now ?: now()
  val exp = claims.expiresAt
  if (exp == null) {
    if (options.requireExpiry) {
      throw Invalid(
        message: "jwt: the token has no 'exp'; pass Options(requireExpiry: false) to accept one that never expires",
        reason: Reason.MissingClaim,
      )
    }
  } else if (at >= exp + options.leeway) {
    throw Invalid(message: "jwt: expired ${at - exp} seconds ago", reason: Reason.Expired)
  }
  val nbf = claims.notBefore
  if (nbf != null && at + options.leeway < nbf) {
    throw Invalid(message: "jwt: not valid for another ${nbf - at} seconds", reason: Reason.NotYetValid)
  }
}

fun checkParties(claims: Claims, options: Options) throws Invalid {
  val audience = options.audience
  if (audience != null && !claims.audience.contains(audience)) {
    val found = if (claims.audience.isEmpty()) "none" else claims.audience.join(", ")
    throw Invalid(message: "jwt: 'aud' is $found, not '$audience'", reason: Reason.WrongAudience)
  }
  val issuer = options.issuer
  if (issuer != null && claims.issuer != issuer) {
    throw Invalid(message: "jwt: 'iss' is ${claims.issuer ?: "missing"}, not '$issuer'", reason: Reason.WrongIssuer)
  }
}

// ---------------------------------------------------------------------------
// keys and MACs

fun mac(algorithm: Algorithm, key: Secret<List<u8>>, signing: string): crypto.Digest => when (algorithm) {
  Algorithm.HS256 => crypto.hmacSha256(key, signing.bytes())
  Algorithm.HS384 => crypto.hmacSha384(key, signing.bytes())
  Algorithm.HS512 => crypto.hmacSha512(key, signing.bytes())
}

/// RFC 7518 §3.2: an HMAC key must be at least as long as the digest. A
/// shorter one is a configuration mistake, and serving requests with it
/// would be worse than stopping.
fun checkKey(key: Secret<List<u8>>, algorithm: Algorithm, who: string) {
  val least = keyBytes(algorithm)
  if (key.len() < least) {
    panic("$who: an $algorithm key must be at least $least bytes (RFC 7518 §3.2), got ${key.len()} — use Secret.of(crypto.randomBytes($least)), not a password")
  }
}

fun keyBytes(algorithm: Algorithm): i64 => when (algorithm) {
  Algorithm.HS256 => 32
  Algorithm.HS384 => 48
  Algorithm.HS512 => 64
}
