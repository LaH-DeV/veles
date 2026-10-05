// BLAKE2b against RFC 7693, Argon2 against the three test vectors of RFC 9106 §5.
use hex

test "blake2b: the RFC 7693 vector for abc, the empty message, and a key" {
  expect(hex.encode(blake2b("abc".bytes(), 64)) == "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923")
  expect(hex.encode(blake2b([], 64)) == "786a02f742015903c6c6fd852552d272912f4740e15847618a86e217f71f5419d25e1031afee585313896444934eb04b903a685b1448b755d56f701afe9be2ce")
  expect(hex.encode(blake2b("abc".bytes(), 32)).len() == 64)
  // a message of exactly one block, and one of two blocks, exercise the last-block rule
  val block: MutableList<u8> = MutableList<u8>.repeat(7, 128)
  expect(blake2b(block.toList(), 64).len() == 64)
  block.addAll(block.toList())
  expect(blake2b(block.toList(), 20).len() == 20)
}

test fun rfcInputs(kind: Argon2Type): List<u8> {
  val password = MutableList<u8>.repeat(1, 32).toList()
  val salt = MutableList<u8>.repeat(2, 16).toList()
  val secret = MutableList<u8>.repeat(3, 8).toList()
  val data = MutableList<u8>.repeat(4, 12).toList()
  argon2(kind, password, salt, 32, 3, 4, 32, secret, data)
}

test "argon2d matches RFC 9106" {
  expect(hex.encode(rfcInputs(Argon2Type.D)) == "512b391b6f1162975371d30919734294f868e3be3984f3c1a13a4db9fabe4acb")
}

test "argon2i matches RFC 9106" {
  expect(hex.encode(rfcInputs(Argon2Type.I)) == "c814d9d1dc7f37aa13f0d77f2494bda1c8de6b016dd388d29952a4c4672b6ce8")
}

test "argon2id matches RFC 9106" {
  expect(hex.encode(rfcInputs(Argon2Type.Id)) == "0d640df58d78766c08c037a34a8b53c9d01ef0452d75b65eb52520e96b01e659")
}

test "argon2 refuses what the RFC forbids" {
  val salt = MutableList<u8>.repeat(2, 16).toList()
  expectPanics(() => argon2(Argon2Type.Id, "p".bytes(), "short".bytes(), 32, 1, 1, 32))
  expectPanics(() => argon2(Argon2Type.Id, "p".bytes(), salt, 4, 1, 1, 32))
  expectPanics(() => argon2(Argon2Type.Id, "p".bytes(), salt, 32, 0, 1, 32))
  expectPanics(() => argon2(Argon2Type.Id, "p".bytes(), salt, 32, 1, 1, 3))
}

test "a password hashes to a PHC string that verifies, and only for that password" {
  val stored = hashPassword(Secret.of("correct horse battery staple"))
  expect(stored.startsWith("\$argon2id\$v=19\$m=19456,t=2,p=1\$"))
  // salt and hash are written without base64's padding
  val parts = stored.split("$")
  expect(parts.len() == 6)
  expect(!(parts.at(4) ?: "=").contains("="))
  expect(!(parts.at(5) ?: "=").contains("="))
  expect(verifyPassword(Secret.of("correct horse battery staple"), stored))
  expect(!verifyPassword(Secret.of("correct horse battery stapl"), stored))
  expect(!verifyPassword(Secret.of(""), stored))
  // a fresh salt each time
  expect(hashPassword(Secret.of("same")) != hashPassword(Secret.of("same")))
}

test "a hash made elsewhere verifies (argon2-cffi's documented example)" {
  val stored = "\$argon2id\$v=19\$m=65536,t=3,p=4\$MIIRqgvgQbgj220jfp0MPA\$YfwJSVjtjSU0zzV/P3S9nnQ/USre2wvJMjfCIjrTQbg"
  expect(verifyPassword(Secret.of("correct horse battery staple"), stored))
  expect(!verifyPassword(Secret.of("wrong"), stored))
}

test "a hash that is not one, or asks too much, is false and never a panic" {
  val pw = Secret.of("x")
  expect(!verifyPassword(pw, ""))
  expect(!verifyPassword(pw, "plain text"))
  expect(!verifyPassword(pw, "\$argon2i\$v=19\$m=32,t=1,p=1\$c29tZXNhbHQ\$AAAAAAAAAAAAAAAAAAAAAA"))
  expect(!verifyPassword(pw, "\$argon2id\$v=16\$m=32,t=1,p=1\$c29tZXNhbHQ\$AAAAAAAAAAAAAAAAAAAAAA"))
  expect(!verifyPassword(pw, "\$argon2id\$v=19\$m=99999999,t=1,p=1\$c29tZXNhbHQ\$AAAAAAAAAAAAAAAAAAAAAA"))
  expect(!verifyPassword(pw, "\$argon2id\$v=19\$m=32,t=0,p=1\$c29tZXNhbHQ\$AAAAAAAAAAAAAAAAAAAAAA"))
  expect(!verifyPassword(pw, "\$argon2id\$v=19\$m=32,t=1,p=1\$!!\$AAAAAAAAAAAAAAAAAAAAAA"))
  expect(!verifyPassword(pw, "\$argon2id\$v=19\$m=32,t=1,p=1\$c29tZXNhbHQ"))
}
