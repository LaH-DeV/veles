// Tests of the health endpoints (D133).

use net
use time

error Down {
  detail: string
  fun message(): string => this.detail
}

// the application behind the probes: one route, so passing through is visible
test fun app(health: Health): Handler {
  val router = Router()
  router.get("/hello", req => Response.text("hello"))
  health.endpoints()(router.handler())
}

test "liveness answers 200 ok and runs no check" {
  val health = Health()
  val ran: Atomic<i64> = Atomic(value: 0)
  health.check("db", () => {
    val _ = ran.update(n => n + 1)
    throw Down(detail: "down")
  })
  val r = call(app(health), Method.get, "/healthz")
  expect(r.status == Status.ok)
  expect(text(r) == "ok")
  expect(r.headers.get("cache-control") == "no-store")
  expect(ran.load() == 0)
}

test "readiness with nothing to check is ok" {
  val r = call(app(Health()), Method.get, "/readyz")
  expect(r.status == Status.ok)
  expect(text(r) == "{\"status\": \"ok\", \"checks\": {}}")
  expect(r.headers.get("content-type") == "application/json")
}

test "readiness lists every check, in the order they were added" {
  val health = Health()
  health.check("db", () => { })
  health.check("cache", () => { })
  val r = call(app(health), Method.get, "/readyz")
  expect(r.status == Status.ok)
  expect(text(r) == "{\"status\": \"ok\", \"checks\": {\"db\": \"ok\", \"cache\": \"ok\"}}")
}

test "a failing check is a 503 that names it and does not say why" {
  val health = Health()
  health.check("db", () => { })
  health.check("cache", () => {
    throw Down(detail: "redis at 10.0.0.7 refused the connection")
  })
  val r = call(app(health), Method.get, "/readyz")
  expect(r.status == Status.serviceUnavailable)
  expect(text(r) == "{\"status\": \"unavailable\", \"checks\": {\"db\": \"ok\", \"cache\": \"failed\"}}")
  expect(!text(r).contains("redis"))
  expect(!text(r).contains("10.0.0.7"))
}

test "a check that takes too long fails at its own timeout" {
  val health = Health()
  health.check("slow", () => {
    await sleep(Duration.seconds(30))
  }, timeout: Duration.millis(100))
  val sw = time.Stopwatch.start()
  val r = call(app(health), Method.get, "/readyz")
  expect(r.status == Status.serviceUnavailable)
  expect(text(r).contains("\"slow\": \"failed\""))
  expect(sw.elapsed() < Duration.seconds(5))
}

test "a check that panics fails, and the probe still answers" {
  val health = Health()
  health.check("odd", () => {
    val xs: List<i64> = []
    val _ = xs.at(3) ?: panic("no such element")
  })
  health.check("fine", () => { })
  val r = call(app(health), Method.get, "/readyz")
  expect(r.status == Status.serviceUnavailable)
  expect(text(r).contains("\"odd\": \"failed\""))
  expect(text(r).contains("\"fine\": \"ok\""))
}

test "the checks run at once, not one after another" {
  val health = Health()
  health.check("a", () => {
    await sleep(Duration.millis(400))
  })
  health.check("b", () => {
    await sleep(Duration.millis(400))
  })
  health.check("c", () => {
    await sleep(Duration.millis(400))
  })
  val sw = time.Stopwatch.start()
  val r = call(app(health), Method.get, "/readyz")
  expect(r.status == Status.ok)
  expect(sw.elapsed() < Duration.millis(1000))
}

test "a check added after the middleware was made still counts" {
  val health = Health()
  val handler = app(health)
  expect(call(handler, Method.get, "/readyz").status == Status.ok)
  health.check("late", () => {
    throw Down(detail: "x")
  })
  expect(call(handler, Method.get, "/readyz").status == Status.serviceUnavailable)
}

test "once stopping, readiness is 503 and runs nothing; liveness stays 200" {
  val health = Health()
  val ran: Atomic<i64> = Atomic(value: 0)
  health.check("db", () => {
    val _ = ran.update(n => n + 1)
  })
  val handler = app(health)
  expect(call(handler, Method.get, "/readyz").status == Status.ok)
  expect(ran.load() == 1)
  health.stopping()
  val r = call(handler, Method.get, "/readyz")
  expect(r.status == Status.serviceUnavailable)
  expect(text(r).contains("stopping"))
  expect(ran.load() == 1)
  expect(call(handler, Method.get, "/healthz").status == Status.ok)
}

test "serve marks the Health as stopping when its stop fires" {
  val health = Health()
  with listener = try net.listen()
  serve(listener, app(health), log: false, health: health, stop: () => {
    await sleep(Duration.millis(50))
  }, grace: Duration.millis(200))
  expect(call(app(health), Method.get, "/readyz").status == Status.serviceUnavailable)
}

test "other paths and methods" {
  val health = Health()
  val handler = app(health)
  expect(text(call(handler, Method.get, "/hello")) == "hello")
  expect(call(handler, Method.get, "/nowhere").status == Status.notFound)
  val post = call(handler, Method.post, "/healthz")
  expect(post.status == Status.methodNotAllowed)
  expect(post.headers.get("allow") == "GET, HEAD")
  expect(call(handler, Method.head, "/readyz").status == Status.ok)
  expect(call(handler, Method.get, "/healthz/").status == Status.notFound)
}

test "the paths can be chosen" {
  val health = Health()
  val router = Router()
  val handler = health.endpoints(live: "/live", ready: "/ready")(router.handler())
  expect(call(handler, Method.get, "/live").status == Status.ok)
  expect(call(handler, Method.get, "/ready").status == Status.ok)
  expect(call(handler, Method.get, "/healthz").status == Status.notFound)
}

test "bad paths are refused at once" {
  val health = Health()
  expectPanics(() => health.endpoints(live: "healthz"))
  expectPanics(() => health.endpoints(live: "/p", ready: "/p"))
}

test "a probe's answer is kept out of the request log, through every wrapper" {
  val health = Health()
  val r = call(app(health), Method.get, "/readyz")
  expect(r.quiet)
  expect(r.withHeader("x-a", "b").quiet)
  expect(r.withCookie(Cookie(name: "a", value: "b")).quiet)
  expect(!call(app(health), Method.get, "/hello").quiet)
}
