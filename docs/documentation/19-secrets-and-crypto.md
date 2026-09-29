# 19. Secrets, hashes and tokens

A server that holds a session, checks a webhook or issues an API key needs
four things: a hash, a keyed hash, unpredictable bytes, and a way to write
bytes down as text. Veles puts them in four modules.

| Module | What it is for |
|---|---|
| `crypto` | SHA-256/384/512 and SHA-1, HMAC, constant-time comparison, random bytes, UUIDs |
| `hex` | bytes as two digits each — digests, fingerprints, `ETag`s |
| `base64` | bytes as text, standard or URL-safe — tokens, cookies, JWTs |
| `jwt` | signed JSON Web Tokens (`HS256`, `HS384`, `HS512`) |

`hex` and `base64` are not in `crypto` on purpose: base64 is an
*encoding*, not encryption, and putting it behind a name that suggests
otherwise is how `base64.encode(password)` gets written.

## Hashing

`crypto.sha256(bytes)` answers with a `Digest`:

```veles
use crypto, io

fun main() {
  val d = crypto.sha256("abc".bytes())
  io.println("$d")
  io.println("${d.len()} bytes, ${d.toBase64Url()}")
  io.println("${d == crypto.sha256("abc".bytes())}")
}
```

Output:
```text
ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
32 bytes, ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0
true
```

A `Digest` prints as lower-case hex, so `"$d"` is the form you write in a
log or a header. `toBase64Url()` is the shorter form for a URL or an
`ETag`, `bytes()` gives the raw bytes, and `prefix(n)` truncates — the
one safe way to shorten a digest.

`sha512`, `sha384` and `sha1Legacy` are the other three. The name of the
last one is a warning: SHA-1 collisions have been practical since 2017, so
it is there for the protocols that specify it (the WebSocket handshake,
Git object names) and for nothing you design yourself.

### Hashing something large

A hash does not need the whole message at once. `Sha256.start()` opens a
hasher, `update` adds bytes as many times as you like, `finish()` closes
it — which is how a file or an upload is hashed without being held in
memory:

```veles
use crypto, io

fun main() {
  var h = crypto.Sha256.start()
  h.update("a".bytes())
  h.update("bc".bytes())
  io.println("${h.finish() == crypto.sha256("abc".bytes())}")
}
```

Output:
```text
true
```

The result depends only on the concatenation, never on where the calls
fell. A hasher is finished once: `update` after `finish` panics, and
`finish` twice returns the same digest.

Every digest type implements the `Hasher` trait, so code can be written
against the trait and told the algorithm later: `crypto.digest<Sha512>(bytes)`.

## MACs: a hash with a key

A hash says "these are the bytes". It does not say "the holder of the
secret wrote them" — anyone can hash anything. That is what HMAC is for:

```veles
use crypto, hex, io

fun main() throws hex.Invalid {
  val secret = "shhh".bytes()
  val body = "{\"event\":\"push\"}"

  // the sender computes this and puts it in a header
  val header = crypto.hmacSha256(secret, body.bytes()).toHex()

  // the receiver recomputes it over the body it actually read
  val sent = crypto.Digest.of(try hex.decode(header))
  io.println("genuine  ${crypto.hmacSha256(secret, body.bytes()) == sent}")
  io.println("altered  ${crypto.hmacSha256(secret, "{}".bytes()) == sent}")
}
```

Output:
```text
genuine  true
altered  false
```

`hmacSha384`, `hmacSha512` and `hmacSha1Legacy` are the same shape, and
`Hmac<Sha256>.start(key)` is the streaming form for a body you do not want
to buffer.

### Why `==` and not `bytes()`

`==` on a `Digest` is **constant time**: it reads every byte whatever the
first difference is. Comparing the raw bytes instead would short-circuit,
and how long the comparison took tells an attacker how many leading bytes
they guessed — which turns a 32-byte MAC into 32 guesses of one byte.

So the safe comparison is the easy one. `Digest.of(bytes)` exists to wrap
bytes that arrived from outside, so a signature out of a header can be
compared the same way; `crypto.equalBytes(a, b)` is the same guarantee for
raw bytes that are not digests (an API key, a session id).

## Random bytes

Anything an attacker must not guess comes from `crypto.randomBytes`:

```veles
// fragment
val session = base64.encodeUrl(crypto.randomBytes(32))
val key = crypto.randomBytes(32)   // an HMAC-SHA-256 key
```

This is the operating system's generator — `BCryptGenRandom` on Windows,
`getrandom(2)` on Linux, `arc4random_buf` on macOS. There is no seed and
no state.

`std/random` is a different thing: a fast pseudo-random generator seeded
from the clock, for shuffling a list or picking a colour. Its output is
predictable from a handful of samples, so a session token built from it is
not a token. The modules are separate so the choice is visible in the
`use` line.

16 bytes is the floor for anything unguessable; 32 is the usual answer and
costs nothing.

`randomBytes` **panics** rather than throwing if the kernel refuses: there
is no weaker source worth falling back to and nothing for a caller to
decide. A server still survives it — a panic is caught at the request
boundary.

## UUIDs

```veles
use crypto, io

fun main() {
  val a = crypto.uuidV7()
  val b = crypto.uuidV7()
  io.println("${a.version()} ${a < b} ${a.timestamp() != null}")
  io.println("${crypto.Uuid.parse("$a") == a}")
  io.println("${crypto.Uuid.parse("nope") == null}")
}
```

Output:
```text
7 true true
true
true
```

`uuidV4()` is 122 random bits. `uuidV7()` puts the millisecond first and a
counter after it, so successive ids are **strictly increasing** — as a
primary key it appends to the index instead of scattering writes across
it. That is almost always the one to use when the id reaches a database.

`parse` accepts the canonical `8-4-4-4-12` form and the same digits
without dashes, and refuses everything else — no braces, no `urn:uuid:`.

## hex and base64

```veles
use base64, hex, io

fun main() {
  io.println(hex.encode("abc".bytes()))
  io.println(base64.encode("foobar".bytes()))
  io.println(base64.encodeUrl([251, 255, 254]))
  when (val bytes = base64.decode("Zm9vYmE")) {
    is Ok  => io.println("${bytes.decodeUtf8()}")
    is Err => io.println(bytes.message)
  }
  when (val bytes = base64.decode("Zm 9v")) {
    is Ok  => io.println("${bytes.len()}")
    is Err => io.println(bytes.message)
  }
}
```

Output:
```text
616263
Zm9vYmFy
-__-
fooba
base64: whitespace at 2; strip line breaks before decoding
```

`encode`/`decode` are the standard alphabet with `+`, `/` and `=`;
`encodeUrl`/`decodeUrl` are the URL-safe one (`-`, `_`) with no padding,
which is what a query parameter, a cookie and a JWT want. Both decoders
take input with or without padding.

Everything else they refuse, with the position in the message: a character
from the other alphabet, an `=` in the middle, a line break, a length that
cannot be a whole number of bytes, and — the one that matters for
security — a last character with bits set that the byte count does not
use. That last case is a *non-canonical* encoding: without the check, two
different texts decode to the same signature, and a replay filter keyed on
the text never notices.

## JSON Web Tokens

`jwt` signs and verifies HMAC tokens. The interesting half is what it
refuses.

```veles
use codec, crypto, io, jwt

fun main() throws EncodeError {
  val key = crypto.randomBytes(32)
  val at = 1700000000

  val token = try jwt.sign(jwt.Claims(
    subject: "user-42",
    issuer: "notes.example",
    audience: ["web"],
    expiresAt: at + 3600,
    extra: ["role": codec.VString(value: "admin")],
  ), key)

  when (val claims = jwt.verify(token, key, jwt.Options(now: at, issuer: "notes.example"))) {
    is Ok  => io.println("${claims.subject} ${claims.text("role")}")
    is Err => io.println("${claims.reason}")
  }

  // the same token, judged an hour and a second later
  when (val claims = jwt.verify(token, key, jwt.Options(now: at + 3601))) {
    is Ok  => io.println("still good")
    is Err => io.println("${claims.reason}")
  }
}
```

Output:
```text
user-42 admin
Expired
```

`Claims` carries the registered claims under readable names — `issuer`,
`subject`, `audience`, `expiresAt`, `notBefore`, `issuedAt`, `id` — and
everything else in `extra`, as the `Value` it was. The three times are
Unix **seconds** (the specification's NumericDate), not the milliseconds
`std/time` works in; `jwt.now()` is the reading to add to.

`Reason` is an enum, because the answers differ: `Expired` usually means
"refresh", everything else means "sign in again".

### The refusals

- **The algorithm comes from you.** `Options.algorithm` says what the
  token must be signed with, and the header's `alg` must equal it. A
  library that trusts the header instead is how `alg: none` and "verify
  this RS256 token as HS256, with the public key as the secret" work.
- `exp` is required unless you pass `Options(requireExpiry: false)`.
- `crit` in the header is rejected: it means "you must understand this
  extension", and this module understands none.
- An HMAC key shorter than the digest is refused outright (RFC 7518
  §3.2) — 32 bytes for HS256. A password is not a key; `crypto.randomBytes(32)`
  is.
- The signature is compared through `crypto.Digest`, so in constant time.

`Options` also holds `audience`, `issuer` and `leeway` (seconds of
tolerance for clocks that disagree; 60 is a common setting).

`jwt.readHeader(token)` reads the header *without* verifying anything, for
a verifier that keeps several keys and needs the `kid` before it can pick
one. Nothing in it is trustworthy — it is unsigned text from whoever called
your API. Look a key up with it, then `verify`.

RSA and ECDSA (`RS256`, `ES256`) are not here: they need bignum
arithmetic or a native binding. `HS256` covers a server that issues its
own tokens, which is the common case.

## A worked example

`examples/crypto` runs every vector in this chapter — FIPS 180-4 for the
hashes, RFC 4231 and RFC 2202 for the MACs, RFC 4648 for base64, and the
token from RFC 7515 Appendix A.1, which is how you know the JWT code
agrees with something other than itself.

[Codable and JSON](18-codable-and-json.md) covers the wire shape of your
own types; this chapter covers the bytes that protect them.

Next: [Time](20-time.md), or back to the [index](index.md).
