// Password hashing (D130): Argon2id in the PHC string format, with the
// parameters OWASP recommends for a login. A stored hash carries its own
// parameters and salt, so raising the cost later does not invalidate the
// hashes already stored.
use base64

// OWASP's first choice for Argon2id: 19 MiB, two passes, one lane
const passwordMemoryKiB: i64 = 19456
const passwordPasses: i64 = 2
const passwordLanes: i64 = 1
const passwordSaltBytes: i64 = 16
const passwordTagBytes: i64 = 32

// what `verifyPassword` accepts in a hash it is given: a hash from a database is trusted
// less than code, and an absurd cost in one would make a login stall or run out of memory
const maxMemoryKiB: i64 = 1048576
const maxPasses: i64 = 64
const maxLanes: i64 = 64

/// A hash of `password` to store: `$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>`,
/// with a fresh random salt. It takes about 50 ms and 19 MiB of memory — the
/// point of Argon2 is that guessing passwords costs the same to an attacker. Store the
/// whole string; `verifyPassword` reads everything it needs from it.
///
/// ```veles
/// val stored = crypto.hashPassword(Secret.of(form.password))
/// // later, at login:
/// if (!crypto.verifyPassword(Secret.of(attempt), stored)) return Response.text("no", status: Status.unauthorized)
/// ```
public fun hashPassword(password: Secret<string>): string {
  val salt = randomBytes(passwordSaltBytes)
  val tag = argon2(Argon2Type.Id, password.expose().bytes(), salt, passwordMemoryKiB, passwordPasses, passwordLanes, passwordTagBytes)
  "\$argon2id\$v=19\$m=$passwordMemoryKiB,t=$passwordPasses,p=$passwordLanes\$${base64.encode(salt).replace("=", "")}\$${base64.encode(tag).replace("=", "")}"
}

/// Whether `password` is the one `hash` was made from. The comparison takes the
/// same time wherever the two differ. A `hash` that is not an Argon2id string in
/// the form `hashPassword` writes — or that asks for more than a login should
/// cost (over 1 GiB, 64 passes or 64 lanes) — is `false`, never a panic.
public fun verifyPassword(password: Secret<string>, hash: string): bool {
  val parts = hash.split("$")
  // "", "argon2id", "v=19", "m=…,t=…,p=…", salt, tag
  if (parts.len() != 6 || !parts.at(0).isEmpty() || parts.at(1) != "argon2id" || parts.at(2) != "v=19") return false
  var memory: i64 = 0
  var passes: i64 = 0
  var lanes: i64 = 0
  loop (field in parts.at(3).split(",")) {
    val n = (field.substring(2, field.len()) ?: "").toInt() ?: return false
    if (field.startsWith("m=")) memory = n else if (field.startsWith("t=")) passes = n else if (field.startsWith("p=")) lanes = n else return false
  }
  if (lanes < 1 || lanes > maxLanes || passes < 1 || passes > maxPasses) return false
  if (memory < 8 * lanes || memory > maxMemoryKiB) return false
  val salt = unpadded(parts.at(4)) ?: return false
  val tag = unpadded(parts.at(5)) ?: return false
  if (salt.len() < 8 || salt.len() > 1024 || tag.len() < 4 || tag.len() > 1024) return false
  val got = argon2(Argon2Type.Id, password.expose().bytes(), salt, memory, passes, lanes, tag.len())
  Digest.of(got) == Digest.of(tag)
}

// base64 without its '=' padding, as the PHC format writes it
fun unpadded(text: string): List<u8>? {
  val padded = text + ("===".substring(0, (4 - text.len() % 4) % 4) ?: "")
  base64.decode(padded) catch (e) {
    return null
  }
}
