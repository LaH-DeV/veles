// Health endpoints (D133): the two probes an orchestrator or a load balancer
// asks of a service, as middleware.
use json as js, log as logs { field }

// one named check; `run` answers null when it passed and why it did not
// otherwise, and neither throws nor panics
struct Check {
  name: string
  run:  sendable fun(): string? suspends
}

/// What a probe-driven deployment needs of a service, in two questions.
/// *Live*: is the process able to answer at all — if not, restart it.
/// *Ready*: can it do its work now — if not, send the traffic elsewhere and
/// leave the process alone. They are different questions, so they are two
/// paths: a slow database must take the instance out of rotation, not get
/// it killed.
///
/// ```veles
/// val health = http.Health()
/// health.check("db", () => try pool.ping())
/// health.check("cache", () => try cache.ping(), timeout: Duration.millis(500))
/// app.wrap(health.endpoints())
/// http.serve(listener, app.handler(), health: health, stop: () => os.shutdownSignal())
/// ```
public struct Health {
  private checks:  Mutex<MutableList<Check>> = newChecks()
  private stopped: Atomic<bool> = Atomic(value: false)

  /// Adds a readiness check: `f` passes by returning and fails by throwing,
  /// or by taking longer than `timeout`. It may suspend. A check that panics
  /// fails too — the probe still answers.
  public fun check<E>(name: string, f: sendable fun() suspends throws E, timeout: Duration = Duration.seconds(2)) {
    val run: sendable fun(): string? suspends = () => when (withTimeout(timeout, f)) {
      is Ok(_)  => null
      is Err(e) => when (e) {
        is Timeout => "timed out after $timeout"
        else       => e.message()
      }
    }
    this.checks.withLock(list => {
      list.push(Check(name, run))
    })
  }

  /// Marks the service as stopping: readiness answers 503 from now on, and
  /// runs no checks. `serve` does this itself when it is given this `Health`
  /// and its `stop:` fires, before it closes anything, so the load balancer
  /// stops sending requests while the ones in flight finish.
  public fun stopping() {
    this.stopped.store(true)
  }

  /// Middleware that answers `live` and `ready` itself, before the router, so
  /// an access log or an authentication layer *inside* it never sees a probe.
  ///
  /// - `live` (`/healthz`): `200 ok`, always — it runs no check.
  /// - `ready` (`/readyz`): every check at once, each under its own timeout.
  ///   `200` with `{"status": "ok", "checks": {"db": "ok"}}` when all pass,
  ///   `503` naming the ones that failed otherwise, and `503` without
  ///   running anything once `stopping()` has been called. Why a check failed
  ///   goes to the log, not into the answer: it may describe infrastructure.
  ///
  /// `GET` and `HEAD` are answered; any other method on these paths is a 405.
  /// Probes are left out of the request log (`logging()` and `serve`'s own).
  /// Panics when a path does not start with `/` or the two are the same.
  public fun endpoints(live: string = "/healthz", ready: string = "/readyz"): Middleware {
    if (!live.startsWith("/") || !ready.startsWith("/")) panic("health: paths start with '/', got '$live' and '$ready'")
    if (live == ready) panic("health: liveness and readiness need different paths, both are '$live'")
    val health = this
    next => req => {
      if (req.path != live && req.path != ready) return next(req)
      if (req.method != Method.get && req.method != Method.head) {
        return Response(status: Status.methodNotAllowed, headers: [Header.allow: "GET, HEAD"], quiet: true)
      }
      if (req.path == live) return probeAnswer(Status.ok, "ok", MediaType.text)
      health.readiness()
    }
  }

  // the readiness answer: the checks run all at once, one task each
  fun readiness(): Response suspends {
    if (this.stopped.load()) {
      return probeAnswer(Status.serviceUnavailable, "{\"status\": \"stopping\", \"checks\": {}}", MediaType.json)
    }
    val checks = this.checks.withLock(list => list.toList())
    val failures: List<string?> = runAll(checks)
    var failed = false
    val parts = StringBuilder()
    loop ((i, c) in checks.enumerate()) {
      val why = failures.at(i) ?: null
      if (why != null) {
        failed = true
        logs.warn("health check failed", field("check", c.name), field("error", why))
      }
      if (i > 0) parts.append(", ")
      parts.append(jsonString(c.name))
      parts.append(if (why == null) ": \"ok\"" else ": \"failed\"")
    }
    val body = "{\"status\": ${if (failed) "\"unavailable\"" else "\"ok\""}, \"checks\": {${parts.toString()}}}"
    probeAnswer(if (failed) Status.serviceUnavailable else Status.ok, body, MediaType.json)
  }
}

fun newChecks(): Mutex<MutableList<Check>> {
  val empty: MutableList<Check> = []
  Mutex(value: empty)
}

// every check, concurrently, each in a task of its own that cannot fail the
// others: the outcome of each, in order
fun runAll(checks: List<Check>): List<string?> suspends {
  val outcomes: MutableList<string?> = []
  scope {
    val tasks: MutableList<Task<string?>> = []
    loop (c in checks) {
      tasks.push(async probe(c))
    }
    loop (t in tasks) {
      outcomes.push(await t)
    }
  }
  outcomes.toList()
}

// one check behind the same boundary a handler has: a panic is a failure
fun probe(c: Check): string? suspends {
  val run = c.run
  when (gather {
    async run()
  }) {
    is Ok(why) => why
    is Err(p)  => "panicked: ${p.message()}"
  }
}

fun probeAnswer(status: Status, body: string, kind: MediaType): Response =
  Response(status, headers: ["content-type": kind.name, Header.cacheControl: "no-store"], body: body.bytes(), quiet: true)

fun jsonString(text: string): string = js.encode(text) ?? "\"?\""
