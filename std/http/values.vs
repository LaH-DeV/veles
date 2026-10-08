// HTTP's named values: statuses, methods and header names (D74). Each is
// an open set with well-known members — a client meets statuses nobody
// registered, a server may be asked for WebDAV's PROPFIND — so each is a
// value type with constants, not an enum.

/// A response status. The constants name the ones RFC 9110 and its
/// companions define; any other code is `Status(code: 499)`. It prints as
/// the status line does: `404 Not Found`.
public struct Status {
  public code: i64

  init {
    if (this.code < 100 || this.code > 999) panic("an HTTP status is three digits, not ${this.code}")
  }

  public static val switchingProtocols = Status(code: 101)
  public static val ok = Status(code: 200)
  public static val created = Status(code: 201)
  public static val accepted = Status(code: 202)
  public static val nonAuthoritativeInformation = Status(code: 203)
  public static val noContent = Status(code: 204)
  public static val resetContent = Status(code: 205)
  public static val partialContent = Status(code: 206)
  public static val multipleChoices = Status(code: 300)
  public static val movedPermanently = Status(code: 301)
  public static val found = Status(code: 302)
  public static val seeOther = Status(code: 303)
  public static val notModified = Status(code: 304)
  public static val temporaryRedirect = Status(code: 307)
  public static val permanentRedirect = Status(code: 308)
  public static val badRequest = Status(code: 400)
  public static val unauthorized = Status(code: 401)
  public static val paymentRequired = Status(code: 402)
  public static val forbidden = Status(code: 403)
  public static val notFound = Status(code: 404)
  public static val methodNotAllowed = Status(code: 405)
  public static val notAcceptable = Status(code: 406)
  public static val proxyAuthenticationRequired = Status(code: 407)
  public static val requestTimeout = Status(code: 408)
  public static val conflict = Status(code: 409)
  public static val gone = Status(code: 410)
  public static val lengthRequired = Status(code: 411)
  public static val preconditionFailed = Status(code: 412)
  public static val contentTooLarge = Status(code: 413)
  public static val uriTooLong = Status(code: 414)
  public static val unsupportedMediaType = Status(code: 415)
  public static val rangeNotSatisfiable = Status(code: 416)
  public static val expectationFailed = Status(code: 417)
  public static val misdirectedRequest = Status(code: 421)
  public static val unprocessableContent = Status(code: 422)
  public static val tooEarly = Status(code: 425)
  public static val upgradeRequired = Status(code: 426)
  public static val preconditionRequired = Status(code: 428)
  public static val tooManyRequests = Status(code: 429)
  public static val requestHeaderFieldsTooLarge = Status(code: 431)
  public static val unavailableForLegalReasons = Status(code: 451)
  public static val internalServerError = Status(code: 500)
  public static val notImplemented = Status(code: 501)
  public static val badGateway = Status(code: 502)
  public static val serviceUnavailable = Status(code: 503)
  public static val gatewayTimeout = Status(code: 504)
  public static val httpVersionNotSupported = Status(code: 505)

  /// The standard reason phrase (RFC 9110 §15), or `""` for a code it
  /// does not name.
  public fun reason(): string => when (this.code) {
    100  => "Continue"
    101  => "Switching Protocols"
    200  => "OK"
    201  => "Created"
    202  => "Accepted"
    203  => "Non-Authoritative Information"
    204  => "No Content"
    205  => "Reset Content"
    206  => "Partial Content"
    300  => "Multiple Choices"
    301  => "Moved Permanently"
    302  => "Found"
    303  => "See Other"
    304  => "Not Modified"
    307  => "Temporary Redirect"
    308  => "Permanent Redirect"
    400  => "Bad Request"
    401  => "Unauthorized"
    402  => "Payment Required"
    403  => "Forbidden"
    404  => "Not Found"
    405  => "Method Not Allowed"
    406  => "Not Acceptable"
    407  => "Proxy Authentication Required"
    408  => "Request Timeout"
    409  => "Conflict"
    410  => "Gone"
    411  => "Length Required"
    412  => "Precondition Failed"
    413  => "Content Too Large"
    414  => "URI Too Long"
    415  => "Unsupported Media Type"
    416  => "Range Not Satisfiable"
    417  => "Expectation Failed"
    421  => "Misdirected Request"
    422  => "Unprocessable Content"
    425  => "Too Early"
    426  => "Upgrade Required"
    428  => "Precondition Required"
    429  => "Too Many Requests"
    431  => "Request Header Fields Too Large"
    451  => "Unavailable For Legal Reasons"
    500  => "Internal Server Error"
    501  => "Not Implemented"
    502  => "Bad Gateway"
    503  => "Service Unavailable"
    504  => "Gateway Timeout"
    505  => "HTTP Version Not Supported"
    else => ""
  }

  /// 1xx: the request goes on.
  public fun isInformational(): bool => this.code < 200
  /// 2xx.
  public fun isSuccess(): bool => this.code >= 200 && this.code < 300
  /// 3xx.
  public fun isRedirect(): bool => this.code >= 300 && this.code < 400
  /// 4xx: the client's mistake.
  public fun isClientError(): bool => this.code >= 400 && this.code < 500
  /// 5xx: the server's.
  public fun isServerError(): bool => this.code >= 500

  implement Display {
    fun toString(): string {
      val reason = this.reason()
      if (reason.isEmpty()) "${this.code}" else "${this.code} $reason"
    }
  }
}

/// A request method. The constants are RFC 9110's eight and PATCH
/// (RFC 5789); any other token is `Method(name: "PROPFIND")`. Methods are
/// case-sensitive, so `Method(name: "get")` is not `Method.get`.
public struct Method {
  public name: string

  public static val get = Method(name: "GET")
  public static val head = Method(name: "HEAD")
  public static val post = Method(name: "POST")
  public static val put = Method(name: "PUT")
  public static val delete = Method(name: "DELETE")
  public static val connect = Method(name: "CONNECT")
  public static val options = Method(name: "OPTIONS")
  public static val trace = Method(name: "TRACE")
  public static val patch = Method(name: "PATCH")

  implement Display {
    fun toString(): string => this.name
  }
}

/// Header names, lower-cased as `Request.header` and `withHeader` store
/// them: `req.header(http.Header.authorization)`.
public struct Header {
  public static val accept: string = "accept"
  public static val acceptEncoding: string = "accept-encoding"
  public static val allow: string = "allow"
  public static val authorization: string = "authorization"
  public static val cacheControl: string = "cache-control"
  public static val connection: string = "connection"
  public static val contentEncoding: string = "content-encoding"
  public static val contentLength: string = "content-length"
  public static val contentType: string = "content-type"
  public static val cookie: string = "cookie"
  public static val date: string = "date"
  public static val etag: string = "etag"
  public static val host: string = "host"
  public static val ifModifiedSince: string = "if-modified-since"
  public static val ifNoneMatch: string = "if-none-match"
  public static val lastModified: string = "last-modified"
  public static val location: string = "location"
  public static val origin: string = "origin"
  public static val remoteUser: string = "x-remote-user"
  public static val requestId: string = "x-request-id"
  public static val retryAfter: string = "retry-after"
  public static val setCookie: string = "set-cookie"
  public static val transferEncoding: string = "transfer-encoding"
  public static val userAgent: string = "user-agent"
  public static val wwwAuthenticate: string = "www-authenticate"
}

/// A media type, as a `Content-Type` says it: `text/html; charset=utf-8`.
/// The constants are what a server most often sends; any other is
/// `MediaType(name: "application/vnd.api+json")`. `Response.bytes` and
/// `Response.stream` take one, or a plain string.
public struct MediaType {
  public name: string

  public static val text = MediaType(name: "text/plain; charset=utf-8")
  public static val html = MediaType(name: "text/html; charset=utf-8")
  public static val css = MediaType(name: "text/css; charset=utf-8")
  public static val javascript = MediaType(name: "text/javascript; charset=utf-8")
  public static val csv = MediaType(name: "text/csv; charset=utf-8")
  public static val eventStream = MediaType(name: "text/event-stream")
  public static val json = MediaType(name: "application/json")
  public static val xml = MediaType(name: "application/xml")
  public static val form = MediaType(name: "application/x-www-form-urlencoded")
  public static val pdf = MediaType(name: "application/pdf")
  public static val zip = MediaType(name: "application/zip")
  public static val wasm = MediaType(name: "application/wasm")
  public static val octetStream = MediaType(name: "application/octet-stream")
  public static val svg = MediaType(name: "image/svg+xml")
  public static val png = MediaType(name: "image/png")
  public static val jpeg = MediaType(name: "image/jpeg")
  public static val gif = MediaType(name: "image/gif")
  public static val webp = MediaType(name: "image/webp")
  public static val avif = MediaType(name: "image/avif")
  public static val icon = MediaType(name: "image/x-icon")
  public static val woff = MediaType(name: "font/woff")
  public static val woff2 = MediaType(name: "font/woff2")
  public static val mp3 = MediaType(name: "audio/mpeg")
  public static val wav = MediaType(name: "audio/wav")
  public static val ogg = MediaType(name: "audio/ogg")
  public static val mp4 = MediaType(name: "video/mp4")
  public static val webm = MediaType(name: "video/webm")

  /// The type for a file extension, with or without the dot and in any
  /// case (`".PNG"`, `"png"`); `octetStream` when it is not known.
  public static fun ofExtension(ext: string): MediaType {
    val e = ext.toLower()
    when (if (e.startsWith(".")) e.substring(1, e.len()) ?: "" else e) {
      "html", "htm" => MediaType.html
      "css"         => MediaType.css
      "js", "mjs"   => MediaType.javascript
      "json"        => MediaType.json
      "txt", "md"   => MediaType.text
      "csv"         => MediaType.csv
      "xml"         => MediaType.xml
      "svg"         => MediaType.svg
      "png"         => MediaType.png
      "jpg", "jpeg" => MediaType.jpeg
      "gif"         => MediaType.gif
      "webp"        => MediaType.webp
      "avif"        => MediaType.avif
      "ico"         => MediaType.icon
      "woff"        => MediaType.woff
      "woff2"       => MediaType.woff2
      "mp3"         => MediaType.mp3
      "wav"         => MediaType.wav
      "ogg"         => MediaType.ogg
      "mp4"         => MediaType.mp4
      "webm"        => MediaType.webm
      "wasm"        => MediaType.wasm
      "pdf"         => MediaType.pdf
      "zip"         => MediaType.zip
      else          => MediaType.octetStream
    }
  }

  /// The type and subtype alone, lower-cased, without parameters:
  /// `text/html; charset=utf-8` → `text/html`. What to compare.
  public fun essence(): string => (this.name.split(";").at(0) ?: "").trim().toLower()

  implement Display {
    fun toString(): string => this.name
  }
  implement AsMediaType {
    fun mediaType(): MediaType => this
  }
}

/// What names a media type: a `MediaType`, or a string that already spells
/// one (`"application/vnd.api+json"`).
public trait AsMediaType {
  fun mediaType(): MediaType
}

implement AsMediaType for string {
  fun mediaType(): MediaType => MediaType(name: this)
}
