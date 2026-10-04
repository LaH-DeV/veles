// Tests of Conn as an io.Stream (D128): a suspending call through the
// trait object, on a real connection.

use io

// serves one line, then echoes what it reads until the peer is done
test fun echo(listener: Listener) suspends throws IoError | io.TooLong {
  with c = try listener.accept()
  val s: io.Stream = c
  try s.writeText("hello\n")
  loop {
    val line = try s.readLine(64)
    if (line == null) break
    try s.writeText("echo ${line}\n")
  }
}

test "a connection is a stream: a suspending call through the object" {
  with listener = try listen()
  scope {
    val server = async echo(listener)
    with conn = try connect("127.0.0.1", listener.port())
    val s: io.Stream = conn
    expect(try s.readLine(64) == "hello")
    try s.writeText("ping\n")
    expect(try s.readLine(64) == "echo ping")
    try s.shutdownWrite()
    expect(try s.readLine(64) == null)
    try await server
  }
}
