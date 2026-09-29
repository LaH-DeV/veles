// Tests of std/log's formatting, which the terminal-versus-pipe choice keeps
// out of reach of a program run from a test: both line formats are built
// here directly.

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
  setLevel(Level.Warn)
  expect(!enabled(Level.Info))
  expect(enabled(Level.Warn))
  expect(enabled(Level.Error))
  setLevel(Level.Off)
  expect(!enabled(Level.Error))
  setLevel(Level.Info)
}
