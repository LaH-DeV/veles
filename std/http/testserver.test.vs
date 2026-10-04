// Tests of `http.testServer` (D130): a real listener on a free port for the
// block that holds it.

use net

test "a test server answers on its port and says its url" {
  with srv = try testServer(handler(req => Response.text("hello ${req.path}")))
  expect(srv.url == "http://127.0.0.1:${srv.port()}")
  with conn = try net.connect("127.0.0.1", srv.port())
  try conn.writeText("GET /there HTTP/1.1\r\nHost: t\r\nConnection: close\r\n\r\n")
  var seen = ""
  loop {
    val chunk = try conn.read()
    if (chunk.isEmpty()) break
    seen = seen + (chunk.decodeUtf8() ?: "")
  }
  expect(seen.startsWith("HTTP/1.1 200"))
  expect(seen.endsWith("hello /there"))
}

test "two test servers get two ports, and a closed one refuses" {
  with first = try testServer(handler(req => Response.text("one")))
  var port = 0
  do {
    with second = try testServer(handler(req => Response.text("two")))
    port = second.port()
    expect(port != first.port())
  } catch (e) {
    expect(false)
  }
  expect(net.connect("127.0.0.1", port) is Err)
}
