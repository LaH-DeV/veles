// Tests of the protobuf encoder (D126) against bytes worked out by hand, and
// the helpers the other test files read what was exported with.

// ---- a small protobuf reader, for the tests that look inside what was exported ----

struct Fld {
  num:  i64
  wire: i64
  // a varint or a fixed number; the length for a length-delimited field
  value: i64
  data:  List<u8>
}

test fun readVarint(bytes: List<u8>, at: i64): (i64, i64) {
  var result: u64 = 0
  var shift: i64 = 0
  var i = at
  loop {
    val b = bytes.at(i) ?: break
    i += 1
    result = result | ((b & 127).toU64() << shift)
    if (b < 128) break
    shift += 7
  }
  (result.wrapI64(), i)
}

// the fields of one message, in order
test fun fields(bytes: List<u8>): List<Fld> {
  val out: MutableList<Fld> = []
  var i: i64 = 0
  loop (i < bytes.len()) {
    val (key, afterKey) = readVarint(bytes, i)
    i = afterKey
    val num = key / 8
    val wire = key % 8
    if (wire == 0) {
      val (v, next) = readVarint(bytes, i)
      i = next
      out.push(Fld(num, wire, value: v, data: []))
    } else if (wire == 1) {
      var v: u64 = 0
      loop (k in 0..<8) {
        v = v | ((bytes.at(i + k) ?: 0).toU64() << (8 * k))
      }
      out.push(Fld(num, wire, value: v.wrapI64(), data: bytes.slice(i, i + 8)))
      i += 8
    } else if (wire == 2) {
      val (len, next) = readVarint(bytes, i)
      out.push(Fld(num, wire, value: len, data: bytes.slice(next, next + len)))
      i = next + len
    } else if (wire == 5) {
      out.push(Fld(num, wire, value: 0, data: bytes.slice(i, i + 4)))
      i += 4
    } else {
      fail("unknown wire type $wire")
    }
  }
  out.toList()
}

// the fields numbered `num`
test fun all(fs: List<Fld>, num: i64): List<Fld> = fs.filter(f => f.num == num)

// the one field numbered `num`, as a message
test fun child(fs: List<Fld>, num: i64): List<Fld> {
  val found = all(fs, num)
  expect(found.len() == 1)
  fields((found.at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data)
}

test fun text(fs: List<Fld>, num: i64): string {
  val found = all(fs, num)
  (found.at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.decodeUtf8() ?: "<binary>"
}

test fun number(fs: List<Fld>, num: i64): i64 = (all(fs, num).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).value

// ---- the encoder ----

test fun sixteen(): List<u8> = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16]

test fun one(meter: string, name: string): MetricData {
  val point = PointData(attrs: [], intValue: 1, realValue: 0.0, count: 0, buckets: [], min: 0.0, max: 0.0)
  MetricData(meter, name, description: "", unit: "", kind: MetricKind.Counter, integer: true, bounds: [], startNanos: 0, points: [point])
}

test "a varint is seven bits at a time, least significant group first" {
  val cases: List<(u64, List<u8>)> = [
    (0, [0]),
    (1, [1]),
    (127, [127]),
    (128, [128, 1]),
    (300, [172, 2]),
    (16384, [128, 128, 1]),
    (18446744073709551615, [255, 255, 255, 255, 255, 255, 255, 255, 255, 1]),
  ]
  loop ((n, want) in cases) {
    val pb = Pb()
    pb.varint(n)
    expect(pb.done() == want)
  }
}

test "the examples of the protobuf documentation, byte for byte" {
  // field 1, varint 150 → 08 96 01
  val a = Pb()
  a.int(1, 150)
  expect(a.done() == [8, 150, 1])
  // field 2, string "testing" → 12 07 74 65 73 74 69 6e 67
  val b = Pb()
  b.string(2, "testing")
  expect(b.done() == [18, 7, 116, 101, 115, 116, 105, 110, 103])
  // field 3 holding the message above-style: 1a 03 08 96 01
  val c = Pb()
  c.message(3, [8, 150, 1])
  expect(c.done() == [26, 3, 8, 150, 1])
}

test "a negative int64 is ten bytes, and zero is left out" {
  val neg = Pb()
  neg.int(1, -1)
  expect(neg.done() == [8, 255, 255, 255, 255, 255, 255, 255, 255, 255, 1])
  val zero = Pb()
  zero.int(1, 0)
  zero.string(2, "")
  zero.bool(3, false)
  zero.bytes(4, [])
  expect(zero.done().isEmpty())
}

test "fixed-width numbers are little-endian" {
  val a = Pb()
  a.fixed64(1, 1)
  expect(a.done() == [9, 1, 0, 0, 0, 0, 0, 0, 0])
  val b = Pb()
  b.double(2, 1.0)
  // 1.0 is 0x3FF0000000000000
  expect(b.done() == [17, 0, 0, 0, 0, 0, 0, 240, 63])
  val c = Pb()
  c.fixed32(3, 258)
  expect(c.done() == [29, 2, 1, 0, 0])
}

test "a repeated number is packed into one length-delimited run" {
  val pb = Pb()
  pb.packedFixed64(6, [1, 2])
  expect(pb.done() == [50, 16, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0])
  val none = Pb()
  none.packedDouble(7, [])
  expect(none.done().isEmpty())
}

test "an attribute is a key and a typed value" {
  // key "a" (0a 01 61), value message (12 len …) holding int_value 5 (18 05)
  expect(keyValue(attr("a", 5)) == [10, 1, 97, 18, 2, 24, 5])
  // string_value is field 1, written even when empty
  expect(keyValue(attr("a", "")) == [10, 1, 97, 18, 2, 10, 0])
  expect(keyValue(attr("a", "b")) == [10, 1, 97, 18, 3, 10, 1, 98])
  // bool_value is field 2
  expect(keyValue(attr("a", true)) == [10, 1, 97, 18, 2, 16, 1])
  expect(keyValue(attr("a", false)) == [10, 1, 97, 18, 2, 16, 0])
  // int_value is written even when zero
  expect(keyValue(attr("a", 0)) == [10, 1, 97, 18, 2, 24, 0])
  // double_value is field 4
  expect(keyValue(attr("a", 1.0)) == [10, 1, 97, 18, 9, 33, 0, 0, 0, 0, 0, 0, 240, 63])
  // array_value is field 5: 2a, its length 10, then two AnyValues of 5 bytes each
  val list: List<string> = ["x", "y"]
  expect(keyValue(attr("a", list)) == [10, 1, 97, 18, 12, 42, 10, 10, 3, 10, 1, 120, 10, 3, 10, 1, 121])
}

test "a span's fields have the numbers opentelemetry-proto gives them" {
  val s = SpanData(
    traceId: sixteen(),
    spanId: [1, 2, 3, 4, 5, 6, 7, 8],
    parentId: [8, 7, 6, 5, 4, 3, 2, 1],
    name: "load",
    kind: SpanKind.Server,
    startNanos: 1000,
    endNanos: 2000,
    attrs: [attr("k", "v")],
    events: [SpanEvent(timeNanos: 1500, name: "hit", attrs: [])],
    status: SpanStatus.Error,
    statusMessage: "boom",
  )
  val f = fields(spanBytes(s))
  expect(all(f, 1).len() == 1)
  expect((all(f, 1).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.len() == 16)
  expect((all(f, 2).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.len() == 8)
  expect((all(f, 4).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data == [8, 7, 6, 5, 4, 3, 2, 1])
  expect(text(f, 5) == "load")
  expect(number(f, 6) == 2)
  expect(number(f, 7) == 1000)
  expect(number(f, 8) == 2000)
  expect(text(child(f, 9), 1) == "k")
  val event = child(f, 11)
  expect(number(event, 1) == 1500)
  expect(text(event, 2) == "hit")
  val status = child(f, 15)
  expect(text(status, 2) == "boom")
  expect(number(status, 3) == 2)
}

test "a span with no status writes none, and no parent writes none" {
  val s = SpanData(traceId: sixteen(), spanId: [1, 2, 3, 4, 5, 6, 7, 8], parentId: [], name: "x", kind: SpanKind.Internal, startNanos: 1, endNanos: 2, attrs: [], events: [], status: SpanStatus.Unset, statusMessage: "")
  val f = fields(spanBytes(s))
  expect(all(f, 15).isEmpty())
  expect(all(f, 4).isEmpty())
}

test "a request nests resource, scope and the records" {
  val s = SpanData(traceId: sixteen(), spanId: [1, 2, 3, 4, 5, 6, 7, 8], parentId: [], name: "x", kind: SpanKind.Internal, startNanos: 1, endNanos: 2, attrs: [], events: [], status: SpanStatus.Unset, statusMessage: "")
  val request = fields(traceRequest([attr("service.name", "notes")], [s, s]))
  val resourceSpans = child(request, 1)
  expect(text(child(child(resourceSpans, 1), 1), 1) == "service.name")
  val scopeSpans = child(resourceSpans, 2)
  expect(text(child(scopeSpans, 1), 1) == "veles")
  expect(all(scopeSpans, 2).len() == 2)
}

test "metrics: a counter is a monotonic cumulative sum, a histogram carries its buckets" {
  val point = PointData(attrs: [attr("route", "/x")], intValue: 7, realValue: 0.0, count: 0, buckets: [], min: 0.0, max: 0.0)
  val counter = MetricData(meter: "m", name: "hits", description: "d", unit: "1", kind: MetricKind.Counter, integer: true, bounds: [], startNanos: 5, points: [point])
  val c = fields(metricBytes(counter, 5, 9))
  expect(text(c, 1) == "hits")
  expect(text(c, 2) == "d")
  expect(text(c, 3) == "1")
  val sum = child(c, 7)
  expect(number(sum, 2) == 2)
  expect(number(sum, 3) == 1)
  val p = child(sum, 1)
  expect(number(p, 2) == 5)
  expect(number(p, 3) == 9)
  expect(number(p, 6) == 7)

  val hist = PointData(attrs: [], intValue: 0, realValue: 4.5, count: 3, buckets: [1, 2, 0], min: 0.5, max: 3.0)
  val h = MetricData(meter: "m", name: "lat", description: "", unit: "s", kind: MetricKind.Histogram, integer: false, bounds: [1.0, 2.0], startNanos: 5, points: [hist])
  val hf = fields(metricBytes(h, 5, 9))
  val data = child(hf, 9)
  expect(number(data, 2) == 2)
  val hp = child(data, 1)
  expect(number(hp, 4) == 3)
  // 3 buckets of 8 bytes, packed; 2 bounds of 8 bytes, packed
  expect((all(hp, 6).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.len() == 24)
  expect((all(hp, 7).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.len() == 16)
}

test "a gauge is a gauge, an up-down counter is a non-monotonic sum" {
  val point = PointData(attrs: [], intValue: -2, realValue: 1.5, count: 0, buckets: [], min: 0.0, max: 0.0)
  val g = fields(metricBytes(MetricData(meter: "m", name: "g", description: "", unit: "", kind: MetricKind.Gauge, integer: false, bounds: [], startNanos: 0, points: [point]), 0, 1))
  expect(all(g, 5).len() == 1)
  val u = fields(metricBytes(MetricData(meter: "m", name: "u", description: "", unit: "", kind: MetricKind.UpDownCounter, integer: true, bounds: [], startNanos: 0, points: [point]), 0, 1))
  val sum = child(u, 7)
  // not monotonic: the flag is left out
  expect(all(sum, 3).isEmpty())
  expect(number(child(sum, 1), 6) == -2)
}

test "metrics are grouped by meter name" {
  val request = fields(metricsRequest([], [one("a", "x"), one("b", "y"), one("a", "z")], 1))
  val scopes = all(child(request, 1), 2)
  expect(scopes.len() == 2)
}

test "a log record carries its severity, body and trace" {
  val r = LogData(timeNanos: 77, severity: 17, severityText: "ERROR", body: "boom", attrs: [attr("k", 1)], traceId: sixteen(), spanId: [1, 2, 3, 4, 5, 6, 7, 8])
  val f = fields(logBytes(r))
  expect(number(f, 1) == 77)
  expect(number(f, 2) == 17)
  expect(text(f, 3) == "ERROR")
  expect(text(child(f, 5), 1) == "boom")
  expect((all(f, 9).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.len() == 16)
  expect((all(f, 10).at(0) ?: Fld(num: 0, wire: 0, value: 0, data: [])).data.len() == 8)
  expect(number(f, 11) == 77)
}
