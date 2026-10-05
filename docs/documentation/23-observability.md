# 23. Observability

A service in production is a program nobody is watching. `otel` is how it
tells the people who run it what it is doing — in OpenTelemetry's format,
which every collector and dashboard reads (D126). Three kinds of evidence:

| | Answers | Looks like |
|---|---|---|
| **Traces** | where did this one request spend its time? | `POST /notes` took 84 ms: 70 of them in `db.insert` |
| **Metrics** | how is the service doing over time? | requests per second, p99 latency, jobs queued |
| **Logs** | what happened, in context? | the `std/log` lines, each with its trace's id |

They connect through the **trace id**: the slow request, its log lines and
its failure are one click apart.

## Spans

A span is one timed piece of work with a name, attributes, and a parent. `with`
holds one — it ends with the block, however the block ends — and the span you
start while another is running becomes its child:

```veles
use io { println }, otel

struct Tally {
  spans: Atomic<i64> = Atomic(value: 0)

  implement otel.Exporter {
    fun export(signal: otel.Signal, body: List<u8>) suspends throws otel.ExportError {
      if (signal == otel.Signal.Traces) {
        val _ = this.spans.update(n => n + 1)
      }
    }
  }
}

fun charge(cents: i64) {
  with span = otel.span("charge card", attrs: [otel.attr("amount", cents)])
  span.event("authorised")
  println("charging $cents inside ${otel.current() != null}")
}

fun checkout() {
  with span = otel.span("checkout")
  charge(1999)
  charge(500)
}

fun main() throws otel.StartError {
  val tally = Tally()
  with tel = try otel.start(service: "shop", exporter: tally, interval: Duration.seconds(3600))
  checkout()
  println("after the spans: ${otel.current() != null}")
  val sent = tel.shutdown()
  println("exported: $sent")
}
```

Output:
```text
charging 1999 inside true
charging 500 inside true
after the spans: false
exported: true
```

- `otel.span(name, attrs:, kind:)` starts a span and makes it the **current**
  one for the rest of the block — in this task and in every task started
  inside it, so work handed to `async` stays in the trace. When the block
  ends, the previous span is current again.
- `span.set(otel.attr("note.id", id))` adds an attribute; `span.event("cache
  miss")` records a moment; `span.ok()` marks success. Attribute names follow
  OpenTelemetry's conventions where one exists (`http.request.method`,
  `db.system`); a value is text, a number, a boolean or a list of text.
- A span that **failed** says so: `span.fail(error)` or `span.failWith("why")`
  mark it (status error, plus an event with the message) and backends show it
  red. A `with` cannot see an error that is on its way out, so for work that
  can fail use `inSpan`, which does it for you and lets the error go on:

```veles
// fragment
val note = try otel.inSpan("load note", () => try db.find(id), attrs: [otel.attr("note.id", id)])
```

- Before `otel.start` — and in your tests — every span, instrument and log
  record is a no-op that costs one load and a branch, so a library can be
  instrumented without asking whether anyone is listening.

## Across services

A request often crosses services. `http.fetch` sends the current trace in a
`traceparent` header (the W3C standard), and `http.serve` reads one from an
incoming request and continues the trace — or starts a new one. So once
`otel` is running:

- every request the server answers is a **server span**, named after the route
  pattern (`GET /users/{id}`, never `GET /users/1042` — a span name is what a
  backend groups by) with the method, path, status code and client address;
  a 5xx marks it failed; **health probes are left out** (chapter 17);
- every call `http.fetch` makes is a **client span** (method, `server.address`,
  `url.full` without its query string, status code) and carries the header,
  so the service you call joins the same trace;
- a header you set yourself is left alone, and the query string, where tokens
  and personal data go, is never recorded.

`otel.traceparent()` gives the header for a request you send some other way,
and `otel.SpanContext.parse(text)` reads one — it refuses anything that is
not exactly the W3C form rather than half-continuing a trace.

## Metrics

```veles
// fragment
val notes = otel.meter("notes")
val created = notes.counter("notes.created", unit: "1")
val open = notes.upDownCounter("notes.open")
val level = notes.gauge("queue.level")
val latency = notes.histogram("notes.save.duration", unit: "s")

created.add(1, [otel.attr("kind", "text")])
open.add(-1)
level.set(0.75)
latency.record(sw.elapsed())            // a Duration is recorded in seconds
```

- A `counter` only goes up (a negative `add` is a panic — a count that went
  down is an up-down counter); a `gauge` keeps the last value set; a
  `histogram` counts values into buckets (`buckets: [...]`, default 5 ms to
  10 s) and keeps their sum, least and greatest.
- Asking for the same name twice gives the same instrument; asking for it as a
  different kind panics.
- Each set of attributes is its own series, cumulative since the instrument was
  made. Keep the sets few: a label taken from user input makes a series per
  value, so past 2000 series an instrument folds the rest into one
  `otel.metric.overflow` series instead of growing without bound.

## Logs

Once `otel` runs, every line `std/log` writes — at or above the log level —
is also an OTLP log record with its fields as attributes and the ids of the
span it was written in. The terminal and JSON output are unchanged except for
one field: a line written inside a span carries `trace_id`.

```text
{"time":"2026-10-04T22:38:19Z","level":"info","msg":"note read","id":"7","trace_id":"2f61cd7b28b3a765742cf6655ac14855"}
```

## Sending it somewhere

`otel.start(service:, exporter:, interval: 10s, resource: [], sampleRatio: 1.0,
maxQueue: 2048)` starts the pipeline: finished spans and log records queue,
and every `interval` a background task sends them with the metrics' current
totals. `exporter` is where it goes; `http.otlp` is the one for a collector
that speaks OTLP over HTTP (port 4318):

```veles
// fragment
val exporter = http.otlp(endpoint: "http://localhost:4318", headers: ["authorization": Secret.of(token)])
with tel = try otel.start(service: "notes", exporter: exporter, resource: [otel.attr("deployment.environment", "prod")])
// ... run ...
tel.shutdown()
```

- The bodies are Protocol Buffers, gzipped, posted to `/v1/traces`,
  `/v1/metrics` and `/v1/logs`; `headers` are `Secret`s, so a key never reaches
  a log or an error message.
- A busy collector (429, 502, 503, 504) or a lost connection is retried — up
  to four tries, 200, 400 and 800 ms apart; a collector that refuses the data
  is asked once. After that the batch is **dropped and counted**: the queues
  are bounded (`maxQueue`), so a collector that is down never costs the
  service memory, and the count is exported as the metric `otel.dropped` and
  read with `otel.dropped()`. Telemetry is never allowed to take the service
  down with it.
- `sampleRatio: 0.1` records a tenth of the new traces. The choice is made
  once, from the trace id, and travels with the trace: every service in it
  agrees, and a request that arrives already decided keeps that decision.
- `resource` describes the program (`service.version`,
  `deployment.environment`); `service.name` and a random
  `service.instance.id` are always set.
- `https://` endpoints are verified against the system's trusted roots;
  `http.otlp(tlsOptions: tls.Options(roots: pem))` trusts a private authority instead.

**Call `shutdown()` before the program ends.** It sends what has gathered
since the last interval and stops recording. A `with` block cannot do it for
you, because closing cannot wait on the network; so if a program ends with
telemetry still queued, closing the `with` prints one line to standard error
saying how much was lost.

`examples/telemetry` runs a service and a client in one program with all of
this on, and shows the trace crossing the call. Not in the module yet: runtime
metrics (garbage collection, tasks, threads), a span per database query
(`std/db`), and `https` export.

Next: back to the [index](index.md).
