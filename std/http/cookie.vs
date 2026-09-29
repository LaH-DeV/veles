// Cookies (D94): what a `Response` sets and what a `Request` carries back.
// A cookie's value is percent-encoded on the way out and decoded on the way
// in, so any string round-trips and none can end the header early; the
// parts of a cookie that cannot be encoded (its name, path and domain) are
// checked, and a bad one is a panic at the line that wrote it.

/// Which requests a browser attaches a cookie to.
public enum SameSite {
  /// Sent on ordinary navigation to the site, not on cross-site sub-requests
  /// (images, forms posted from elsewhere). The default.
  Lax
  /// Sent only when the request starts on the same site.
  Strict
  /// Sent everywhere; browsers accept it only with `secure: true`.
  None
}

/// One cookie to set. The defaults are the safe ones: the whole site (`/`),
/// hidden from scripts (`httpOnly`), not sent cross-site (`SameSite=Lax`).
/// `secure` is off so a program served over plain http while it is being
/// written still works; switch it on for anything deployed.
///
/// ```veles
/// resp.withCookie(http.Cookie(name: "sid", value: id, maxAge: Duration.days(7), secure: true))
/// ```
public struct Cookie {
  /// A token: letters, digits and ``!#$%&'*+-.^_`|~``.
  public name: string
  /// Any text; it is percent-encoded when written and decoded by `Request.cookie`.
  public value: string
  public path:  string = "/"
  /// The domain it is for, and its subdomains; null: this host only.
  public domain: string? = null
  /// How long the browser keeps it; null: until the browser closes.
  public maxAge: Duration? = null
  public secure: bool = false
  /// Not readable by scripts on the page (`document.cookie`).
  public httpOnly: bool = true
  /// null leaves the attribute out and the browser to decide.
  public sameSite: SameSite? = SameSite.Lax
}

// ---------------------------------------------------------------------------
// writing

// A cookie the browsers would refuse, or one that could not be written
// safely, is a programmer's mistake, so it panics — at the call of the
// function that took it (D88), not in here.
@caller_location
fun checkCookie(c: Cookie) {
  if (!isToken(c.name)) panic("cookie name '${c.name}' must be a token: letters, digits and !#\$%&'*+-.^_`|~")
  if (!c.path.startsWith("/") || c.path.bytes().any(b => b < 32 || b == 127 || b == ';')) panic("cookie path '${c.path}' must start with '/' and hold no ';' or control characters")
  if (val domain = c.domain && (domain.isEmpty() || !domain.bytes().all(b => (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '.' || b == '-'))) {
    panic("cookie domain '$domain' must be a host name: letters, digits, '.' and '-'")
  }
  if (val age = c.maxAge && age.isNegative()) panic("cookie '${c.name}' has a negative maxAge; withoutCookie removes a cookie")
  if (c.sameSite == SameSite.None && !c.secure) panic("cookie '${c.name}': SameSite=None needs secure: true, or browsers drop it")
  if (c.name.startsWith("__Host-") && (!c.secure || c.path != "/" || c.domain != null)) {
    panic("cookie '${c.name}': a __Host- cookie must be secure, have path '/' and no domain")
  }
  if (c.name.startsWith("__Secure-") && !c.secure) panic("cookie '${c.name}': a __Secure- cookie must be secure")
}

/// The value of a `Set-Cookie` header for `c`.
fun setCookieLine(c: Cookie): string {
  val out = StringBuilder()
  out.append(c.name)
  out.append("=")
  out.append(encodeCookieValue(c.value))
  if (val age = c.maxAge) out.append("; Max-Age=${age.toSeconds()}")
  if (val domain = c.domain) out.append("; Domain=$domain")
  out.append("; Path=${c.path}")
  if (c.secure) out.append("; Secure")
  if (c.httpOnly) out.append("; HttpOnly")
  if (val same = c.sameSite) out.append("; SameSite=$same")
  out.toString()
}

// Every byte but the unreserved ones (letters, digits, `-._~`) as `%XX`,
// which is also what keeps `;`, `,`, spaces, quotes and line breaks out.
fun encodeCookieValue(v: string): string {
  val out = StringBuilder()
  val hex = "0123456789ABCDEF"
  loop (b in v.bytes()) {
    if ((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '.' || b == '_' || b == '~') {
      out.appendByte(b)
    } else {
      out.append("%")
      out.append(hex.substring((b / 16).toI64(), (b / 16).toI64() + 1) ?: "0")
      out.append(hex.substring((b % 16).toI64(), (b % 16).toI64() + 1) ?: "0")
    }
  }
  out.toString()
}

// ---------------------------------------------------------------------------
// reading

/// The cookies of a `Cookie` header: `name=value` pairs separated by `;`.
/// A value may be quoted, and is percent-decoded when it decodes to text
/// (one written by another system as it came stays as it came). When a name
/// repeats the first counts, as browsers list the most specific path first.
fun parseCookies(header: string): Map<string, string> {
  val out: MutableMap<string, string> = [:]
  loop (part in header.split(";")) {
    val (rawName, rawValue) = part.trim().splitOnce("=") ?: continue
    val name = rawName.trim()
    if (name.isEmpty() || out.get(name) != null) continue
    var value = rawValue.trim()
    if (value.len() >= 2 && value.startsWith("\"") && value.endsWith("\"")) value = value.substring(1, value.len() - 1) ?: value
    out.set(name, percentDecode(value, plusIsSpace: false))
  }
  out.toMap()
}
