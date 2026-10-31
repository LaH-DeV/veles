test "codec child paths preserve special keys in JSON pointers" {
  expect(Problem(path: childPath("", "a\"b"), message: "").pointer() == "/a\"b")
  expect(Problem(path: childPath("", "a\\b"), message: "").pointer() == "/a\\b")
  expect(Problem(path: childPath("", "a.b"), message: "").pointer() == "/a.b")
  expect(Problem(path: childPath("", "a[b]"), message: "").pointer() == "/a[b]")
  expect(Problem(path: childPath("root", "a\"b"), message: "").pointer() == "/root/a\"b")
}

struct CodecFixture {
  name:    string
  counts:  List<i64>
  enabled: bool
  implement Codable
}

test "derived codecs round-trip through the shared value tree" {
  val original = CodecFixture(name: "café", counts: [1, -2, 3], enabled: true)
  val encoder = ValueEncoder.of()
  try original.encode(encoder)
  val decoder = ValueDecoder.of(encoder.value())
  val decoded = try CodecFixture.decode(decoder)
  try finish(decoder, decoded)
  expect(decoded.name == original.name)
  expect(decoded.counts == original.counts)
  expect(decoded.enabled == original.enabled)
}

test "decoder problems retain paths and respect the configured cap" {
  val problems = Problems.capped(2)
  problems.record("user.name", "required")
  problems.record("user.counts[1]", "expected integer")
  problems.record("user.extra", "ignored after cap")
  val kept = problems.list()
  expect(kept.len() == 2)
  expect(kept.at(0)?.pointer() == "/user/name")
  expect(kept.at(1)?.pointer() == "/user/counts/1")
  val none = Problems.capped(0)
  none.record("user", "not stored")
  expect(none.isEmpty())
}
