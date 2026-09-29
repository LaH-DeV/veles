// Tests of the connection limit (D99): a full server stops accepting, and
// takes the next connection when one closes.

use net

fun limitedOk(): Handler = handler(req => Response.text("ok"))

fun limitedGet(close: bool): string =
  "GET / HTTP/1.1\r\nHost: t\r\n" + (if (close) "Connection: close\r\n" else "") + "\r\n"

// what one read returns, as text ("" at the end of the stream)
fun limitedRead(conn: net.Conn): string throws IoError {
  (try conn.read()).decodeUtf8() ?: "<binary>"
}

test "at the connection limit a new connection waits until one closes" {
  with (listener = try net.listen()) {
    scope {
      val server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
      val port = listener.port()
      with (first = try net.connect("127.0.0.1", port)) {
        try first.writeText(limitedGet(false))
        expect((try limitedRead(first)).startsWith("HTTP/1.1 200"))
        // the first connection is kept alive and holds the only place
        with (second = try net.connect("127.0.0.1", port)) {
          try second.writeText(limitedGet(true))
          val early = withTimeout(Duration.millis(300), () => try limitedRead(second))
          expect(when (early) {
            is Ok  => false
            is Err => true
          })
          // the first one ends: the server sees the end of its requests, closes
          // it, and accepts the second
          try first.shutdownWrite()
          expect((try limitedRead(second)).startsWith("HTTP/1.1 200"))
        }
      }
      server.cancel()
    }
  }
}

test "a limit of zero serves every connection at once" {
  with (listener = try net.listen()) {
    scope {
      val server = async serve(listener, limitedOk(), limits: Limits(connections: 0), log: false)
      val port = listener.port()
      with (first = try net.connect("127.0.0.1", port)) {
        try first.writeText(limitedGet(false))
        expect((try limitedRead(first)).startsWith("HTTP/1.1 200"))
        with (second = try net.connect("127.0.0.1", port)) {
          try second.writeText(limitedGet(true))
          expect((try limitedRead(second)).startsWith("HTTP/1.1 200"))
        }
      }
      server.cancel()
    }
  }
}

test "stopping a full server does not wait for a place" {
  with (listener = try net.listen()) {
    scope {
      val server = async serve(listener, limitedOk(), limits: Limits(connections: 1), log: false)
      val port = listener.port()
      with (first = try net.connect("127.0.0.1", port)) {
        try first.writeText(limitedGet(false))
        expect((try limitedRead(first)).startsWith("HTTP/1.1 200"))
        // the accept loop is now waiting for a place; cancelling ends the wait
        await sleep(Duration.millis(50))
        server.cancel()
      }
    }
  }
}
