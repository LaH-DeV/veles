// Tests of guard, basicAuth and bearer (D99).

use base64

error AuthBoom {
  message: string
}

test fun authApp(m: Middleware): Handler {
  val r = Router()
  r.get("/who", req => Response.text(req.header(Header.remoteUser) ?: "-"))
  r.wrap(m)
  r.handler()
}

test fun basic(user: string, pass: string): Map<string, string> =>
  ["Authorization": "Basic " + base64.encode("$user:$pass".bytes())]

test fun forged(user: string, pass: string, remote: string): Map<string, string> {
  val h = basic(user, pass).toMutable()
  h.set("X-Remote-User", remote)
  h.toMap()
}

test fun authText(r: Response): string => r.body.decodeUtf8() ?: "<binary>"

test fun rootOnly(): Middleware => basicAuth("admin", (user, pass) => user == "root" && pass == "s3cret")

test "a guard lets a request through on null and answers with a Response" {
  val app = authApp(guard(req => if (req.path == "/who") null else Response.text("no", status: Status.forbidden)))
  expect(call(app, Method.get, "/who").status == Status.ok)
  expect(call(app, Method.get, "/other").status == Status.forbidden)
}

test "a guard's Fail answers with its status, any other error with a 500" {
  val fails = authApp(guard(req => throw forbidden("closed today")))
  val r = call(fails, Method.get, "/who")
  expect(r.status == Status.forbidden)
  expect(authText(r) == "closed today")
  val broken = authApp(guard(req => throw AuthBoom(message: "database is down")))
  val b = call(broken, Method.get, "/who")
  expect(b.status == Status.internalServerError)
  expect(!authText(b).contains("database"))
}

test "basic: no credentials, or wrong ones, get a 401 and a challenge" {
  val app = authApp(rootOnly())
  val none = call(app, Method.get, "/who")
  expect(none.status == Status.unauthorized)
  expect(none.headers.get("www-authenticate") == "Basic realm=\"admin\", charset=\"UTF-8\"")
  expect(call(app, Method.get, "/who", headers: basic("root", "wrong")).status == Status.unauthorized)
  expect(call(app, Method.get, "/who", headers: basic("nobody", "s3cret")).status == Status.unauthorized)
  // a 401 for a path that has no route too: the guard is in front of the 404
  expect(call(app, Method.get, "/nowhere").status == Status.unauthorized)
}

test "basic: good credentials pass, and the handler sees the user" {
  val app = authApp(rootOnly())
  val r = call(app, Method.get, "/who", headers: basic("root", "s3cret"))
  expect(r.status == Status.ok)
  expect(authText(r) == "root")
}

test "basic: a user name the client forged is never trusted" {
  val app = authApp(rootOnly())
  val refused = call(app, Method.get, "/who", headers: ["X-Remote-User": "root"])
  expect(refused.status == Status.unauthorized)
  val replaced = call(app, Method.get, "/who", headers: forged("root", "s3cret", "someone-else"))
  expect(authText(replaced) == "root")
}

test "basic: a password may hold colons, the scheme is any case, junk is a 401" {
  val app = authApp(basicAuth("r", (user, pass) => pass == "a:b:c"))
  expect(call(app, Method.get, "/who", headers: basic("u", "a:b:c")).status == Status.ok)
  val upper = ["Authorization": "BASIC " + base64.encode("u:a:b:c".bytes())]
  expect(call(app, Method.get, "/who", headers: upper).status == Status.ok)
  expect(call(app, Method.get, "/who", headers: ["Authorization": "Basic !!!not base64"]).status == Status.unauthorized)
  expect(call(app, Method.get, "/who", headers: ["Authorization": "Basic " + base64.encode("no colon".bytes())]).status == Status.unauthorized)
  expect(call(app, Method.get, "/who", headers: ["Authorization": "Basic"]).status == Status.unauthorized)
  expect(call(app, Method.get, "/who", headers: ["Authorization": "Bearer abc"]).status == Status.unauthorized)
}

test "bearer: no token, a wrong token, a good token" {
  val app = authApp(bearer(token => token == "k-123"))
  val none = call(app, Method.get, "/who")
  expect(none.status == Status.unauthorized)
  expect(none.headers.get("www-authenticate") == "Bearer")
  val bad = call(app, Method.get, "/who", headers: ["Authorization": "Bearer nope"])
  expect(bad.status == Status.unauthorized)
  expect(bad.headers.get("www-authenticate") == "Bearer error=\"invalid_token\"")
  expect(call(app, Method.get, "/who", headers: ["Authorization": "Bearer k-123"]).status == Status.ok)
  expect(call(app, Method.get, "/who", headers: ["authorization": "bearer k-123"]).status == Status.ok)
  expect(call(app, Method.get, "/who", headers: ["Authorization": "Bearer k-123 extra"]).status == Status.unauthorized)
}

test "bearer: a realm is part of both challenges" {
  val app = authApp(bearer(token => false, realm: "api"))
  expect(call(app, Method.get, "/who").headers.get("www-authenticate") == "Bearer realm=\"api\"")
  val bad = call(app, Method.get, "/who", headers: ["Authorization": "Bearer x"])
  expect(bad.headers.get("www-authenticate") == "Bearer realm=\"api\", error=\"invalid_token\"")
}

test "a verify that throws a Fail answers with it" {
  val app = authApp(bearer(token => throw Fail(status: Status.serviceUnavailable, text: "auth backend is away")))
  val r = call(app, Method.get, "/who", headers: ["Authorization": "Bearer x"])
  expect(r.status == Status.serviceUnavailable)
}

test "a realm that could break the header is a panic" {
  expectPanics(() => checkRealm("say \"hi\""))
  expectPanics(() => checkRealm("line\r\nbreak"))
  expectPanics(() => checkRealm("back\\slash"))
  checkRealm("Staff area")
}
