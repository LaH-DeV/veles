// Tests of the pieces of tracing that need no pipeline: the W3C trace
// context, identifiers, and the sampler.

test "a traceparent is read and written in exactly the W3C form" {
  val text = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
  val ctx = SpanContext.parse(text) ?: fail("a valid traceparent was refused")
  expect(ctx.traceIdHex() == "4bf92f3577b34da6a3ce929d0e0e4736")
  expect(ctx.spanIdHex() == "00f067aa0ba902b7")
  expect(ctx.sampled)
  expect(ctx.traceparent() == text)
  val unsampled = SpanContext.parse("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00") ?: fail("refused")
  expect(!unsampled.sampled)
  expect(unsampled.traceparent().endsWith("-00"))
}

test "the flags carry more than the sampled bit, and only that bit counts" {
  val ctx = SpanContext.parse("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-03") ?: fail("refused")
  expect(ctx.sampled)
  val other = SpanContext.parse("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-02") ?: fail("refused")
  expect(!other.sampled)
}

test "a later version may add fields; version 00 may not" {
  expect(SpanContext.parse("01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra") != null)
  expect(SpanContext.parse("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra") == null)
  expect(SpanContext.parse("ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01") == null)
}

test "a traceparent that is not the form is refused, not half read" {
  val bad = [
    "",
    "garbage",
    "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7",
    "00-4bf92f3577b34da6a3ce929d0e0e47-00f067aa0ba902b7-01",
    "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba9-01",
    "00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01",
    "00-4bf92f3577b34da6a3ce929d0e0e4736-00F067AA0BA902B7-01",
    "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
    "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
    "00-4bf92f3577b34da6a3ce929d0e0e473g-00f067aa0ba902b7-01",
    "0-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
    "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1",
    " 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  ]
  loop (text in bad) {
    expect(SpanContext.parse(text) == null)
  }
}

test "identifiers have their lengths and are never all zero" {
  loop (_ in 0..<200) {
    val trace = randomBytes(16)
    val span = randomBytes(8)
    expect(trace.len() == 16)
    expect(span.len() == 8)
    expect(trace.any(b => b != 0))
    expect(span.any(b => b != 0))
  }
  expect(randomBytes(16) != randomBytes(16))
}

test "hex goes both ways, lower case only" {
  expect(hexOf([0, 1, 171, 255]) == "0001abff")
  expect(bytesOfHex("0001abff", 8) == [0, 1, 171, 255])
  expect(bytesOfHex("0001ABFF", 8) == null)
  expect(bytesOfHex("0001abf", 8) == null)
  expect(bytesOfHex("000g", 4) == null)
}

test "a ratio samples that share of traces, the same trace the same way" {
  expect(sampleTrace(randomBytes(16), 1.0))
  expect(!sampleTrace(randomBytes(16), 0.0))
  var kept: i64 = 0
  loop (_ in 0..<2000) {
    val id = randomBytes(16)
    val first = sampleTrace(id, 0.25)
    expect(first == sampleTrace(id, 0.25))
    if (first) kept += 1
  }
  // 2000 draws at 0.25: 500 expected, far from 300 and 700
  expect(kept > 380 && kept < 620)
}
