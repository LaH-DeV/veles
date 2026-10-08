// Tests of the connection limit (D99): a full server stops accepting, and
// takes the next connection when one closes.

use net

test fun limitedOk(): Handler => handler(req => Response.text("ok"))

test fun limitedGet(close: bool): string =>
  "GET / HTTP/1.1\r\nHost: t\r\n" + (if (close) "Connection: close\r\n" else "") + "\r\n"

// what one read returns, as text ("" at the end of the stream)
test fun limitedRead(conn: net.Conn): string throws IoError {
  (try conn.read()).decodeUtf8() ?: "<binary>"
}

test "at the connection limit a new connection waits until one closes" {
  with srv = try testServer(limitedOk(), Limits(connections: 1))
  val port = srv.port()
  with first = try net.connect("127.0.0.1", port)
  try first.writeText(limitedGet(false))
  expect(try limitedRead(first).startsWith("HTTP/1.1 200"))
  // the first connection is kept alive and holds the only place
  with second = try net.connect("127.0.0.1", port)
  try second.writeText(limitedGet(true))
  expect(withTimeout(Duration.millis(300), () => try limitedRead(second)) is Err)
  // the first one ends: the server sees the end of its requests, closes
  // it, and accepts the second
  try first.shutdownWrite()
  expect(try limitedRead(second).startsWith("HTTP/1.1 200"))
}

test "a limit of zero serves every connection at once" {
  with srv = try testServer(limitedOk(), Limits(connections: 0))
  val port = srv.port()
  with first = try net.connect("127.0.0.1", port)
  try first.writeText(limitedGet(false))
  expect(try limitedRead(first).startsWith("HTTP/1.1 200"))
  with second = try net.connect("127.0.0.1", port)
  try second.writeText(limitedGet(true))
  expect(try limitedRead(second).startsWith("HTTP/1.1 200"))
}

test "stopping a full server does not wait for a place" {
  with srv = try testServer(limitedOk(), Limits(connections: 1))
  val port = srv.port()
  with first = try net.connect("127.0.0.1", port)
  try first.writeText(limitedGet(false))
  expect(try limitedRead(first).startsWith("HTTP/1.1 200"))
  // the accept loop is now waiting for a place; cancelling it while `first`
  // still holds the place must end the wait (were it to wait on, it would
  // take the place `first` gives up when it closes, and serve for ever)
  await sleep(Duration.millis(50))
  srv.server.cancel()
}
