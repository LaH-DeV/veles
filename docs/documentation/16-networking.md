# 16. Networking

The `net` module gives you TCP: a `Listener` that accepts connections and
a `Conn` that reads and writes bytes. Everything that has to wait — for a
client, for data, for the peer to catch up — suspends the task and lets
the other tasks run, so a server is one task per connection on one
thread, written as if each connection had the thread to itself. Every
failing call throws `IoError`, with the address in `path`.

## A server and its client

Programs in this chapter play both sides in one process: a server task
and a client task under one `scope`. Nothing about the code changes when
the client is another machine.

```veles
use io, net

fun serve(listener: net.Listener) throws IoError | io.TooLong {
  with conn = try listener.accept()
  loop {
    val line = try conn.readLine(max: 4096) ?: break
    try conn.writeText("echo: $line\n")
  }
}

fun client(port: i64) throws IoError | io.TooLong {
  with conn = try net.connect("127.0.0.1", port)
  loop (word in ["one", "two"]) {
    try conn.writeText("$word\n")
    io.println("client got: ${try conn.readLine(max: 4096)}")
  }
}

fun main() throws IoError | io.TooLong {
  with (listener = try net.listen()) {
    scope {
      async serve(listener)
      async client(listener.port())
    }
  }
  io.println("done")
}
```

Output:
```text
client got: echo: one
client got: echo: two
done
```

Reading it:

- `net.listen(host, port)` binds a socket. Both arguments have defaults:
  the loopback address, and port 0, which asks the system for a free
  port — `listener.port()` reads it back. A real server passes
  `host: ""` (every interface) and its port.
- `listener.accept()` suspends until a client connects and returns the
  `Conn`. `net.connect(host, port)` is the other end.
- `readLine(max:)` returns the next line without its `\n` (a `\r` before
  it is dropped too), or `null` when the peer closed the connection and
  nothing is left — the `?: break` above is the usual way to run a
  connection to its end. `max` is how many bytes of one line you are
  willing to hold, and it has no default: the peer decides where the
  newline goes, so a line is only as long as the reader allows. Reaching
  the ceiling throws `io.TooLong`, and the connection is spent — close
  it.
- `with` closes the connection and the listener on every way out,
  including a thrown error or the task being cancelled
  ([chapter 13](13-memory-and-ffi.md)).
- `close()` may come from another task, on another thread, while a
  read, write or `accept` is waiting on the same socket: that call fails
  with an `IoError` at once rather than waiting for data that can no
  longer come. The socket itself is released only when the last call
  using it has returned, so a call never lands on a newer socket that
  happens to get the same number from the system. Closing twice is
  harmless.
- `serve` and `client` are ordinary functions; the `scope` waits for
  both. Neither says `suspends`: it is inferred from the calls inside.

The `Conn` API, in full:

| call | what it does |
|---|---|
| `read(max = 65536): List<u8>` | up to `max` bytes, as soon as any arrive; empty at end of stream |
| `readExact(n): List<u8>` | exactly `n` bytes (fewer only if the peer closes first) — a body with a known length. `n` is the ceiling as well as the count, so check a length that came from the peer against your own limit first |
| `readLine(max): string?` | the next text line, `null` at end of stream; more than `max` bytes without a newline throws `io.TooLong` |
| `write(bytes)`, `writeText(text)` | send everything, suspending while the peer catches up |
| `shutdownWrite()` | "nothing more from me": the peer's reads see end of stream while this side keeps reading |
| `peer()` | the other end's address as `host:port` |

A `Conn` is an `io.Stream`, as an `fs.File` is: code that only reads and
writes bytes takes the trait (`fun summarize(s: io.Stream)`) and works on
a socket, a file and whatever implements it next. The calls through it
suspend the task like the direct ones do (`examples/streams`).

`read` hands out whatever has arrived, which for TCP means any split of
the bytes sent; `readLine` and `readExact` buffer on top of it, and the
three can be mixed on one connection.

## One task per connection

A server accepts in a loop and hands each connection to its own task.
The handler owns the connection from then on; the accept loop is back to
`accept()` immediately, so a slow client never delays the next one:

```veles
use io, net

fun handle(conn: net.Conn, store: Mutex<MutableMap<string, string>>) throws IoError | io.TooLong {
  with c = conn
  loop {
    val line = try c.readLine(max: 4096) ?: break
    val reply = when (line.split(" ")) {
      ["SET", key, value] => {
        store.withLock(m => m.set(key, value))
        "OK"
      }
      ["GET", key]        => store.withLock(m => m.get(key)) ?: "(none)"
      else                => "ERR unknown command"
    }
    try c.writeText("$reply\n")
  }
}

fun server(listener: net.Listener, connections: i64) throws IoError | io.TooLong {
  val store = Mutex(value: MutableMap<string, string>())
  scope {
    loop (_ in 0..<connections) {
      val conn = try listener.accept()
      async handle(conn, store)
    }
  }
}

fun ask(port: i64, commands: List<string>): List<string> throws IoError | io.TooLong {
  with conn = try net.connect("127.0.0.1", port)
  var replies: MutableList<string> = []
  loop (cmd in commands) {
    try conn.writeText("$cmd\n")
    replies.push(try conn.readLine(max: 4096) ?: "closed")
  }
  return replies.toList()
}

// two clients, one after the other: the second sees what the first stored
fun clients(port: i64): string throws IoError | io.TooLong {
  val first = try ask(port, ["SET lang veles", "GET lang"])
  val second = try ask(port, ["GET lang", "DEL lang"])
  "$first $second"
}

fun main() throws IoError {
  with listener = try net.listen()
  val port = listener.port()
  val results = gather {
    async server(listener, 2)
    async clients(port)
  }
  io.println(results.1.getOrDefault("failed"))
}
```

Output:
```text
[OK, veles] [veles, ERR unknown command]
```

Two things carry over to any server you write. The connection is passed
to `async handle(conn, ...)`, so `Conn` had to be Sendable — it is, which
is why its read-ahead buffer sits behind a `Mutex` inside. And shared
state between handlers (the `store`) is a `Mutex` too; the compiler would
refuse a bare `MutableMap` at the `async` ([chapter 12](12-concurrency.md)).
This server stops after a fixed number of connections so the program
ends; a real one loops forever and is stopped by cancelling its task.

## Timeouts

Size and time are two different ceilings. `max` bounds what one read may
hold; it does nothing about a client that connects and then says nothing,
which would hold a `readLine` open forever. 
`withTimeout` from the prelude puts a limit on any suspending call; on timeout the read is cancelled, the connection's `with` — if the
read was inside one — has run, and `Timeout` is thrown:

```veles
use io, net

fun greetOrDrop(conn: net.Conn): string throws IoError | io.TooLong | Timeout {
  with c = conn
  return try withTimeout(Duration.millis(50), () => try c.readLine(max: 4096)) ?: "closed"
}

fun silent(port: i64) throws IoError {
  with conn = try net.connect("127.0.0.1", port)
  await sleep(Duration.millis(500))
}

fun main() throws IoError {
  with listener = try net.listen()
  scope {
    async silent(listener.port())
    val conn = try listener.accept()
    when (greetOrDrop(conn)) {
      is Ok(line) => io.println("got $line")
      is Err(e)   => io.println("dropped: ${e.message()}")
    }
  }
}
```

Output:
```text
dropped: timed out after 50ms
```

The function passed to `withTimeout` runs in a task of its own, so it must
be sendable: it may capture `val`s of Sendable types — `c` here — and
nothing mutable. `E | Timeout` in the signature is the lambda's own error
type joined with the timeout; write `throws IoError | Timeout`, or let a
`when` handle both.

## What happens underneath

Sockets are non-blocking. When a read finds nothing to read, the `Conn`
parks the task on that socket and returns to the executor, which polls
every parked socket alongside its timers and resumes the task when the
socket is ready; the read is then retried. No thread ever blocks inside a
socket call, which is what lets one thread hold thousands of idle
connections — the same shape as Node's event loop or Go's netpoller, with
the difference that the waiting is explicit in the types: `accept`,
`read` and `write` are `suspends` functions, and a function that calls
them becomes one.

The cost of that model is the one it has everywhere: a task that computes
for a long time without suspending stalls every other connection. Break
long work with `await sleep(Duration.zero)`, or keep it out of the serving tasks.

Next: [An HTTP server](17-http.md), which is this module used in anger, or
back to [Concurrency](12-concurrency.md) for what `scope`, `gather` and
cancellation do in detail.
