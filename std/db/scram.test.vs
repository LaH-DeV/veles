// SCRAM-SHA-256 against the worked example of RFC 7677 §3.

test "the RFC 7677 exchange produces the RFC's proof and accepts the RFC's server signature" {
  val scram = Scram(user: "user", nonce: "rOprNGfwEbeRWgbNEkqO", password: Secret.of("pencil"))
  expect(scram.first() == "n,,n=user,r=rOprNGfwEbeRWgbNEkqO")
  val serverFirst = "r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF\$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"
  val (final, signature) = scram.answer(serverFirst) ?: fail("a valid challenge")
  expect(final == "c=biws,r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF\$k0,p=dHzbZapWIk4jUhN+Ute9ytag9zjfMHgsqmmiz7AndVQ=")
  expect(scram.verify("v=6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G4=", signature))
  expect(!scram.verify("v=6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G5=", signature))
  expect(!scram.verify("e=invalid-proof", signature))
}

test "a challenge whose nonce does not extend ours is refused" {
  val scram = Scram(user: "", nonce: "abc", password: Secret.of("pw"))
  val _ = scram.first()
  expect(scram.answer("r=xyz123,s=c2FsdA==,i=4096") == null)
  expect(scram.answer("r=abc,s=c2FsdA==,i=4096") == null)
}

test "a challenge with a silly iteration count or a bad salt is refused" {
  val scram = Scram(user: "", nonce: "abc", password: Secret.of("pw"))
  val _ = scram.first()
  expect(scram.answer("r=abcdef,s=c2FsdA==,i=0") == null)
  expect(scram.answer("r=abcdef,s=c2FsdA==,i=999999999") == null)
  expect(scram.answer("r=abcdef,s=!!!,i=4096") == null)
  expect(scram.answer("r=abcdef,i=4096") == null)
}
