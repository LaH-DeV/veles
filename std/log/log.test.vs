// Tests of std/log's formatting, which the terminal-versus-pipe choice keeps
// out of reach of a program run from a test: both line formats are built
// here directly.

use otel

// the level is the process's: the tests that change it or depend on it take turns
val levelGate = Semaphore(permits: 1)

test "a text line reads as timestamp, level, message and fields" {
  val line = textLine("2026-09-29T10:15:03.123Z", Level.Info, "served", [field("path", "/x"), field("ms", 3)])
  expect(line == "2026-09-29T10:15:03.123Z INFO  served path=/x ms=3")
}

test "a text field with a space or an equals sign is quoted" {
  val line = textLine("t", Level.Warn, "m", [field("who", "a b"), field("eq", "x=y"), field("empty", "")])
  expect(line == "t WARN  m who=\"a b\" eq=\"x=y\" empty=\"\"")
}

test "a JSON line keeps a value's type and escapes the message" {
  val line = jsonLine("t", Level.Error, "say \"hi\"", [field("n", 3), field("ok", true), field("s", "x")])
  expect(line == "{\"time\":\"t\",\"level\":\"error\",\"msg\":\"say \\\"hi\\\"\",\"n\":3,\"ok\":true,\"s\":\"x\"}")
}

test "level names read from the environment are case-insensitive" {
  expect(levelFromName("DEBUG") == Level.Debug)
  expect(levelFromName("Off") == Level.Off)
  expect(levelFromName("loud") == null)
}

test "a level is enabled at or above the threshold" {
  with permit = levelGate.acquire()
  setLevel(Level.Warn)
  expect(!enabled(Level.Info))
  expect(enabled(Level.Warn))
  expect(enabled(Level.Error))
  setLevel(Level.Off)
  expect(!enabled(Level.Error))
  setLevel(Level.Info)
}

// ---- OpenTelemetry (D126): a line is also a log record, in the trace it was written in ----

test "severity numbers are OpenTelemetry's" {
  expect(severity(Level.Debug) == 5)
  expect(severity(Level.Info) == 9)
  expect(severity(Level.Warn) == 13)
  expect(severity(Level.Error) == 17)
}

test "a field becomes an attribute of its own type" {
  expect(attrOf(field("s", "text")).value is otel.StringValue)
  expect(attrOf(field("n", 7)).value is otel.IntValue)
  expect(attrOf(field("b", true)).value is otel.BoolValue)
  expect(attrOf(field("x", 1.5)).value is otel.DoubleValue)
  expect(attrOf(field("n", 7)).key == "n")
}

struct Seen {
  bodies: Mutex<MutableList<List<u8>>> = newBodies()

  implement otel.Exporter {
    fun export(signal: otel.Signal, body: List<u8>) suspends throws otel.ExportError {
      if (signal == otel.Signal.Logs) {
        this.bodies.withLock(q => {
          q.push(body)
        })
      }
    }
  }
}

test fun newBodies(): Mutex<MutableList<List<u8>>> {
  val empty: MutableList<List<u8>> = []
  Mutex(value: empty)
}

// whether `part` occurs in `bytes`
test fun holds(bytes: List<u8>, part: string): bool {
  val needle = part.bytes()
  loop (i in 0..<bytes.len()) {
    if (bytes.slice(i, i + needle.len()) == needle) return true
  }
  false
}

test "with otel running, a log line is exported as a record carrying its fields" {
  with permit = levelGate.acquire()
  val seen = Seen()
  with tel = try otel.start(service: "logged", exporter: seen, interval: Duration.seconds(3600))
  info("served the request", field("path", "/x"), field("ms", 3))
  expect(tel.flush())
  val bodies = seen.bodies.withLock(q => q.toList())
  expect(bodies.len() == 1)
  val body = bodies.at(0) ?: []
  expect(holds(body, "served the request"))
  expect(holds(body, "path"))
  expect(holds(body, "INFO"))
}
