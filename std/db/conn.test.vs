// Tests against a real PostgreSQL. They run when VELES_TEST_PG_URL names a
// database (postgres://user@127.0.0.1:5432/postgres); without it they pass
// without doing anything, so the suite stays green on a machine with no server.
use os

test fun testUrl(): Secret<string>? {
  val text = os.env("VELES_TEST_PG_URL") ?: return null
  Secret.of(text)
}

test "a connection logs in and answers a query" {
  val url = testUrl() ?: return
  val target = try parseTarget(url, "veles-test")
  val conn = try Conn.open(target, Duration.seconds(10))
  val none: string? = null
  val rows = try conn.run(sql"select ${41 + 1}::int as answer, ${"héllo"}::text as greeting, ${none}::text as nothing", Duration.seconds(5))
  expect(rows.columns.len() == 3)
  expect(rows.columns.at(0)?.name == "answer")
  expect(rows.rows.len() == 1)
  expect(rows.rows.at(0)?.at(0) == "42")
  expect(rows.rows.at(0)?.at(1) == "héllo")
  expect((rows.rows.at(0) ?: []).filter(c => c == null).len() == 1)
  expect(rows.tag == "SELECT 1")
  conn.close()
}

// a server whose role logs in with SCRAM-SHA-256: VELES_TEST_PG_SCRAM_URL names it, with the password
test "SCRAM-SHA-256 logs in with the right password and refuses the wrong one" {
  val text = os.env("VELES_TEST_PG_SCRAM_URL") ?: return
  val target = try parseTarget(Secret.of(text), "veles-test")
  val conn = try Conn.open(target, Duration.seconds(10))
  val rows = try conn.run(sql"select current_user::text", Duration.seconds(5))
  expect(rows.rows.at(0)?.at(0) == target.user)
  conn.close()
  val wrong = Target(host: target.host, port: target.port, user: target.user, password: Secret.of("not the password"), database: target.database, tls: target.tls, rootCert: target.rootCert, application: "veles-test")
  when (Conn.open(wrong, Duration.seconds(10))) {
    is Ok(c)  => {
      c.close()
      fail("a wrong password must not log in")
    }
    is Err(e) => {
      // PostgreSQL on Windows sometimes resets the connection after its error, and a reset
      // discards the unread message: the refusal then shows as a lost connection
      expect(e.kind == ErrorKind.Auth || e.kind == ErrorKind.Connection)
      if (e.kind == ErrorKind.Auth) expect(e.code == "28P01")
    }
  }
}

// TLS: VELES_TEST_PG_TLS_URL names a server with TLS on; its certificate is in the file VELES_TEST_PG_ROOT
test "TLS is verified, and the escape hatch is named" {
  val text = os.env("VELES_TEST_PG_TLS_URL") ?: return
  val root = os.env("VELES_TEST_PG_ROOT") ?: ""
  val verified = try parseTarget(Secret.of(text + "?sslmode=verify-full&sslrootcert=" + root), "veles-test")
  val conn = try Conn.open(verified, Duration.seconds(10))
  expect((try conn.run(sql"select 1", Duration.seconds(5))).rows.len() == 1)
  conn.close()
  // no root given: the certificate is not signed by anyone the system trusts
  val untrusted = try parseTarget(Secret.of(text + "?sslmode=verify-full"), "veles-test")
  when (Conn.open(untrusted, Duration.seconds(10))) {
    is Ok(c)  => {
      c.close()
      fail("an unknown authority must be refused")
    }
    is Err(e) => expect(e.kind == ErrorKind.Connection)
  }
  val insecure = try parseTarget(Secret.of(text + "?sslmode=insecure"), "veles-test")
  val c2 = try Conn.open(insecure, Duration.seconds(10))
  c2.close()
}

test "the URL: escapes, defaults, and what it refuses" {
  val t = try parseTarget(Secret.of("postgres://app:p%40ss%3Aw%2Frd@db.example.com:6543/shop?application_name=api"), "x")
  expect(t.host == "db.example.com")
  expect(t.port == 6543)
  expect(t.user == "app")
  expect(t.password.expose() == "p@ss:w/rd")
  expect(t.database == "shop")
  expect(t.application == "api")
  expect(t.tls == Tls.Verified)
  expect(t.shown() == "db.example.com:6543/shop")
  val local = try parseTarget(Secret.of("postgresql://me@localhost/"), "x")
  expect(local.tls == Tls.Off)
  expect(local.port == 5432)
  expect(local.database == "me")
  val v6 = try parseTarget(Secret.of("postgres://u@[::1]:5433/d"), "x")
  expect(v6.host == "::1")
  expect(v6.port == 5433)
  expect(parseTarget(Secret.of("mysql://u@h/d"), "x") is Err)
  expect(parseTarget(Secret.of("postgres://@h/d"), "x") is Err)
  expect(parseTarget(Secret.of("postgres://u@h:abc/d"), "x") is Err)
  expect(parseTarget(Secret.of("postgres://u@h/d?sslmode=prefer"), "x") is Err)
  expect(parseTarget(Secret.of("postgres://u@h/d?surprise=1"), "x") is Err)
  expect(parseTarget(Secret.of("postgres://u:%zz@h/d"), "x") is Err)
}
