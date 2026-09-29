// crypto: the primitives a server needs before it can be trusted with a
// password, a cookie or a webhook — hashes, MACs, random bytes, UUIDs,
// and the two encodings that carry them. Every digest printed here is a
// published test vector: FIPS 180-4 for the hashes, RFC 4231 and RFC 2202
// for the MACs, RFC 4648 for base64, RFC 7515 for the token.
use base64, codec, crypto, hex, io { println }, jwt

fun bytes(b: u8, n: i64): List<u8> = MutableList<u8>.repeat(b, n).toList()

/// FIPS 180-4, the three documented messages for each hash.
fun digests() {
  println("-- digests (FIPS 180-4)")
  val long = "abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq"
  println("sha256 ''    ${crypto.sha256("".bytes())}")
  println("sha256 'abc' ${crypto.sha256("abc".bytes())}")
  println("sha256 long  ${crypto.sha256(long.bytes())}")
  println("sha384 'abc' ${crypto.sha384("abc".bytes())}")
  println("sha512 'abc' ${crypto.sha512("abc".bytes())}")
  println("sha1   'abc' ${crypto.sha1Legacy("abc".bytes())}")

  // fed in pieces, the result depends only on the concatenation — this is
  // how a large upload or a file is hashed without holding it in memory
  var h = crypto.Sha256.start()
  h.update("a".bytes())
  h.update("bc".bytes())
  println("streamed     ${h.finish()}  same=${h.finish() == crypto.sha256("abc".bytes())}")

  // 100 blocks of ten bytes: the buffering crosses block boundaries
  var big = crypto.Sha256.start()
  loop (_ in 0..<100) big.update("0123456789".bytes())
  println("1000 bytes   ${big.finish()}")

  println("names        ${crypto.Sha256.algorithm()} block=${crypto.Sha256.blockSize()} digest=${crypto.Sha256.digestSize()}")
  println("etag         ${crypto.sha256("hello".bytes()).prefix(8)}")
}

/// RFC 4231 (SHA-2) and RFC 2202 (SHA-1). A MAC is a hash with a key: it
/// answers "did the holder of the secret write this", which a bare hash
/// cannot.
fun macs() {
  println("-- MACs (RFC 4231, RFC 2202)")
  println("hs256 t1 ${crypto.hmacSha256(bytes(0x0b, 20), "Hi There".bytes())}")
  println("hs256 t2 ${crypto.hmacSha256("Jefe".bytes(), "what do ya want for nothing?".bytes())}")
  println("hs256 t3 ${crypto.hmacSha256(bytes(0xaa, 20), bytes(0xdd, 50))}")
  println("hs256 t6 ${crypto.hmacSha256(bytes(0xaa, 131), "Test Using Larger Than Block-Size Key - Hash Key First".bytes())}")
  println("hs384 t1 ${crypto.hmacSha384(bytes(0x0b, 20), "Hi There".bytes())}")
  println("hs512 t1 ${crypto.hmacSha512(bytes(0x0b, 20), "Hi There".bytes())}")
  println("hs1   t1 ${crypto.hmacSha1Legacy(bytes(0x0b, 20), "Hi There".bytes())}")

  var m = crypto.Hmac<crypto.Sha256>.start("Jefe".bytes())
  m.update("what do ya ".bytes())
  m.update("want for nothing?".bytes())
  println("streamed same=${m.finish() == crypto.hmacSha256("Jefe".bytes(), "what do ya want for nothing?".bytes())}")
  println("name     ${crypto.Hmac<crypto.Sha256>.algorithm()}")
}

/// Verifying a webhook: recompute the MAC over the body and compare the
/// two digests. `==` on a `Digest` is constant time, so a wrong signature
/// takes as long as a right one and tells the sender nothing about how
/// close they were.
fun webhook() throws hex.Invalid {
  println("-- verifying a signature")
  val secret = "shhh".bytes()
  val body = "{\"event\":\"push\"}"
  val header = crypto.hmacSha256(secret, body.bytes()).toHex()
  println("header   sha256=$header")

  val sent = crypto.Digest.of(try hex.decode(header))
  println("accepted ${crypto.hmacSha256(secret, body.bytes()) == sent}")
  println("altered  ${crypto.hmacSha256(secret, "{\"event\":\"pull\"}".bytes()) == sent}")
  println("raw      ${crypto.equalBytes("abc".bytes(), "abc".bytes())} ${crypto.equalBytes("abc".bytes(), "abd".bytes())} ${crypto.equalBytes("abc".bytes(), "ab".bytes())}")
}

/// RFC 4648. base64 and hex are encodings, not encryption: they carry
/// bytes through something that expects text.
fun encodings() throws base64.Invalid | hex.Invalid {
  println("-- base64 and hex (RFC 4648)")
  loop (s in ["", "f", "fo", "foo", "foob", "fooba", "foobar"]) {
    println("'${s.padEnd(6)}' std=${base64.encode(s.bytes()).padEnd(8)} url=${base64.encodeUrl(s.bytes())}")
  }
  val edge = [251 as u8, 255 as u8, 254 as u8]
  println("alphabets  std=${base64.encode(edge)} url=${base64.encodeUrl(edge)}")
  println("hex        ${hex.encode(edge)} ${hex.encodeUpper(edge)}")
  println("round trip ${(try base64.decode("Zm9vYmFy")).decodeUtf8()} ${(try hex.decode("666f6f")).decodeUtf8()}")
  println("lengths    ${base64.encodedLen(32)} ${base64.encodedLen(32, pad: false)}")

  // every decoder refusal names the position, so a log line locates the
  // bad character in a long token
  loop (bad in ["Zg=", "Zm9vYmF", "-_8", "Zm 9v", "Z"]) {
    when (val r = base64.decode(bad)) {
      is Ok  => println("base64 '$bad' -> ${r.len()} bytes")
      is Err => println("base64 '$bad' -> ${r.message}")
    }
  }
  loop (bad in ["0", "0g", "0x00"]) {
    when (val r = hex.decode(bad)) {
      is Ok  => println("hex '$bad' -> ${r.len()} bytes")
      is Err => println("hex '$bad' -> ${r.message}")
    }
  }
}

/// Random bytes come from the operating system, never from `std/random`:
/// a session token an attacker can predict is not a token.
fun identifiers() {
  println("-- randomness and ids")
  val token = base64.encodeUrl(crypto.randomBytes(32))
  println("token    len=${token.len()} distinct=${token != base64.encodeUrl(crypto.randomBytes(32))}")

  val v4 = crypto.uuidV4()
  val v7 = crypto.uuidV7()
  println("v4       version=${v4.version()} text=${"$v4".len()} timestamp=${v4.timestamp()}")
  println("v7       version=${v7.version()} ordered=${v7 < crypto.uuidV7()}")
  println("zero     ${crypto.Uuid.zero()} isZero=${crypto.Uuid.zero().isZero()}")

  val text = "$v7"
  println("parse    canonical=${crypto.Uuid.parse(text) == v7} bare=${crypto.Uuid.parse(text.replace("-", "")) == v7}")
  println("refuse   ${crypto.Uuid.parse("not-a-uuid") == null} ${crypto.Uuid.parse("0190d3e1-7c00-7000-8000-9a5b1c2d3e4g") == null}")
}

/// RFC 7515 Appendix A.1 gives a token and its key, so a library can be
/// checked against the specification rather than against itself.
fun tokens() throws base64.Invalid | EncodeError {
  println("-- JSON Web Tokens (RFC 7515)")
  val token = "eyJ0eXAiOiJKV1QiLA0KICJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJqb2UiLA0KICJleHAiOjEzMDA4MTkzODAsDQogImh0dHA6Ly9leGFtcGxlLmNvbS9pc19yb290Ijp0cnVlfQ.dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
  val key = try base64.decodeUrl("AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow")

  // the token expired in 2011, so the tests pin the moment
  report("rfc token   ", jwt.verify(token, key, jwt.Options(now: 1300819300)))
  report("expired     ", jwt.verify(token, key, jwt.Options(now: 1400000000)))
  report("with leeway ", jwt.verify(token, key, jwt.Options(now: 1300819400, leeway: 100)))
  report("wrong issuer", jwt.verify(token, key, jwt.Options(now: 1300819300, issuer: "jane")))
  report("as HS512    ", jwt.verify(token, key, jwt.Options(now: 1300819300, algorithm: jwt.Algorithm.HS512)))
  report("wrong key   ", jwt.verify(token, crypto.randomBytes(32), jwt.Options(now: 1300819300)))
  report("two parts   ", jwt.verify("header.payload", key, jwt.Options()))

  // one signed here: the header carries a key id, the payload a private
  // claim, and `verify` is the only way back in
  val own = crypto.randomBytes(32)
  val at = 1700000000
  val mine = try jwt.sign(jwt.Claims(
    subject: "user-42",
    issuer: "notes.example",
    audience: ["api", "web"],
    expiresAt: at + 3600,
    issuedAt: at,
    extra: ["role": codec.VString(value: "admin")],
  ), own, keyId: "2026-09")
  report("mine        ", jwt.verify(mine, own, jwt.Options(now: at, audience: "web", issuer: "notes.example")))
  report("wrong aud   ", jwt.verify(mine, own, jwt.Options(now: at, audience: "admin")))
  report("no exp      ", jwt.verify(try jwt.sign(jwt.Claims(subject: "s"), own), own, jwt.Options()))

  when (val header = jwt.readHeader(mine)) {
    is Ok  => println("header       kid=${header.get("kid")?.asString()} alg=${header.get("alg")?.asString()}")
    is Err => println("header       ${header.message}")
  }
  when (val claims = jwt.verify(mine, own, jwt.Options(now: at))) {
    is Ok  => println("claims       role=${claims.text("role")} extra=${claims.extra.len()} iat=${claims.issuedAt}")
    is Err => println("claims       ${claims.message}")
  }
}

fun report(label: string, r: Result<jwt.Claims, jwt.Invalid>) {
  when (r) {
    is Ok  => println("$label ok sub=${r.subject ?: "-"} iss=${r.issuer ?: "-"} aud=${if (r.audience.isEmpty()) "-" else r.audience.join("|")}")
    is Err => println("$label ${r.reason} — ${r.message}")
  }
}

fun main() throws base64.Invalid | hex.Invalid | EncodeError {
  digests()
  macs()
  try webhook()
  try encodings()
  identifiers()
  try tokens()
}
