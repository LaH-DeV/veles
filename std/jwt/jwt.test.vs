use crypto

test "verify accepts i64 max expiration with positive leeway" {
  val maximum: i64 = 9223372036854775807
  val key = Secret.of("0123456789abcdef0123456789abcdef".bytes())
  val token = try sign(Claims(expiresAt: maximum), key)
  val verified = try verify(token, key, Options(now: 0, leeway: 1))
  expect(verified.expiresAt == maximum)
}

test "verify handles max time claims and now with positive leeway" {
  val maximum: i64 = 9223372036854775807
  val key = Secret.of("0123456789abcdef0123456789abcdef".bytes())
  val token = try sign(Claims(expiresAt: maximum, notBefore: maximum), key)
  val verified = try verify(token, key, Options(now: maximum, leeway: 1))
  expect(verified.expiresAt == maximum)
  expect(verified.notBefore == maximum)
}

test "time checks at i64 limits do not overflow" {
  val maximum: i64 = 9223372036854775807
  val minimum: i64 = -9223372036854775807 - 1
  expect(checkTime(Claims(expiresAt: maximum), Options(now: 0, leeway: 1)) is Ok)
  expect(checkTime(Claims(notBefore: maximum), Options(now: maximum, leeway: 1, requireExpiry: false)) is Ok)
  expect(checkTime(Claims(expiresAt: minimum), Options(now: minimum, leeway: minimum)) is Err)
  expect(checkTime(Claims(notBefore: 0), Options(now: minimum, leeway: minimum, requireExpiry: false)) is Err)
  expect(checkTime(Claims(notBefore: maximum), Options(now: 1, leeway: maximum, requireExpiry: false)) is Ok)
}

test "extreme time rejection messages do not overflow" {
  val maximum: i64 = 9223372036854775807
  val minimum: i64 = -9223372036854775807 - 1
  val expired = checkTime(Claims(expiresAt: minimum), Options(now: maximum))
  expect(when (expired) {
    is Err(e) => e.reason == Reason.Expired
    is Ok(_)  => false
  })
  val future = checkTime(Claims(notBefore: maximum), Options(now: minimum, requireExpiry: false))
  expect(when (future) {
    is Err(e) => e.reason == Reason.NotYetValid
    is Ok(_)  => false
  })
}
