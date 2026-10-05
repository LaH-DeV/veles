// SCRAM-SHA-256 (RFC 5802, RFC 7677): how PostgreSQL logs a client in without
// the password crossing the wire. The client proves it knows the password,
// and checks that the server does too. The password is used as written —
// SASLprep (normalisation of non-ASCII) is not applied (checklist §11).
use base64, crypto

/// The client's side of a SCRAM exchange. `first()` is the message that opens it,
/// `answer(...)` turns the server's challenge into the proof and the signature to
/// expect back, and `verify(...)` checks the server's own proof. A value with no
/// state of its own: the pieces are recomputed, so a copy is as good as the original.
struct Scram {
  user:     string
  nonce:    string
  password: Secret<string>

  fun bare(): string = "n=${this.user},r=${this.nonce}"

  // "n,," says: no channel binding (the TLS layer, when there is one, is not bound to)
  fun first(): string = "n,," + this.bare()

  // The server's first message `r=<nonce>,s=<salt>,i=<count>` → the final message with the proof.
  // Null when the challenge is not one: a nonce that does not extend ours, a salt that is not
  // base64, or an iteration count outside what is sane. The second part is the signature the
  // server's last message must carry.
  fun answer(serverFirst: string): (string, List<u8>)? {
    var nonce = ""
    var salt: List<u8> = []
    var iterations: i64 = 0
    loop (field in serverFirst.split(",")) {
      if (field.startsWith("r=")) {
        nonce = field.substring(2, field.len()) ?: ""
      } else if (field.startsWith("s=")) {
        salt = base64.decode(field.substring(2, field.len()) ?: "") catch (e) {
          return null
        }
      } else if (field.startsWith("i=")) {
        iterations = (field.substring(2, field.len()) ?: "").toInt() ?: 0
      }
    }
    if (!nonce.startsWith(this.nonce) || nonce.len() == this.nonce.len() || salt.isEmpty()) return null
    // PostgreSQL asks for 4096; a server that asks for far more is a way to stall a client
    if (iterations < 1 || iterations > 10000000) return null
    val withoutProof = "c=biws,r=$nonce"
    val authMessage = "${this.bare()},$serverFirst,$withoutProof".bytes()
    val salted = pbkdf2(this.password.expose().bytes(), salt, iterations)
    val clientKey = hmac(salted, "Client Key".bytes())
    val storedKey = crypto.sha256(clientKey).bytes()
    val signature = hmac(storedKey, authMessage)
    val proof: MutableList<u8> = []
    loop ((i, k) in clientKey.enumerate()) {
      proof.push(k ^ (signature.at(i) ?: 0))
    }
    val serverKey = hmac(salted, "Server Key".bytes())
    ("$withoutProof,p=${base64.encode(proof.toList())}", hmac(serverKey, authMessage))
  }

  // The server's last message `v=<signature>`: true when it is the one only a server that knows
  // the password could make.
  fun verify(serverFinal: string, expected: List<u8>): bool {
    if (!serverFinal.startsWith("v=")) return false
    val got = base64.decode(serverFinal.substring(2, serverFinal.len()) ?: "") catch (e) {
      return false
    }
    crypto.Digest.of(got) == crypto.Digest.of(expected)
  }
}

fun hmac(key: List<u8>, message: List<u8>): List<u8> {
  var m = crypto.Hmac<crypto.Sha256>.start(key)
  m.update(message)
  m.finish().bytes()
}

// PBKDF2-HMAC-SHA-256 with one block of output (RFC 2898; 32 bytes are all SCRAM wants).
fun pbkdf2(password: List<u8>, salt: List<u8>, iterations: i64): List<u8> {
  val first: MutableList<u8> = salt.toMutable()
  first.push(0)
  first.push(0)
  first.push(0)
  first.push(1)
  var u = hmac(password, first.toList())
  val result = u.toMutable()
  var n: i64 = 1
  loop (n < iterations) {
    u = hmac(password, u)
    loop (i in 0..<result.len()) {
      result.set(i, result.at(i) ^ (u.at(i) ?: 0))
    }
    n += 1
  }
  result.toList()
}
