use io

struct ProbeStream {
  reads:      Atomic<i64> = Atomic(value: 0)
  writes:     Atomic<i64> = Atomic(value: 0)
  closes:     Atomic<i64> = Atomic(value: 0)
  failWrites: bool = false

  implement io.Stream {
    fun read(max: i64 = 65536): List<u8> {
      this.reads.add(1)
      []
    }
    fun readExact(n: i64): List<u8> => []
    fun readLine(max: i64): string? => null
    fun write(bytes: List<u8>) throws IoError {
      this.writes.add(1)
      if (this.failWrites) {
        throw IoError(path: "probe", code: 5, detail: "probe write failed", kind: IoKind.Other)
      }
    }
    fun writeText(text: string) throws IoError => try this.write(text.bytes())
    fun shutdownWrite() { }
  }

  implement Closeable {
    fun close() {
      this.closes.add(1)
    }
  }
}

test "TLS defaults keep certificate verification enabled" {
  val options = Options()
  expect(options.roots.isEmpty())
  expect(options.serverName.isEmpty())
  expect(options.alpn.isEmpty())
  expect(!options.dangerouslyAcceptAnyCertificate)
}

test "TLS failures retain their address and diagnostic prefix" {
  val error = failure("localhost:443", "certificate mismatch")
  expect(error.path == "localhost:443")
  expect(error.kind == IoKind.Other)
  expect(error.code == 0)
  expect(error.detail == "tls: certificate mismatch")
}

test "NUL server names close the owned stream before handshake IO" {
  loop (name in ["\u{0}", "localhost\u{0}evil", "\u{E9}\u{0}host"]) {
    val stream = ProbeStream()
    expect(when (client(stream, name)) {
      is Err(error) => error.kind == IoKind.InvalidInput && error.code == 22 &&
        error.path == name.replace("\u{0}", "\\0") &&
        error.detail == "tls: server name cannot contain a NUL byte"
      is Ok(conn)   => {
        conn.close()
        false
      }
    })
    expect(stream.reads.load() == 0)
    expect(stream.writes.load() == 0)
    expect(stream.closes.load() == 1)
  }
}

test "NUL rejection retains and sanitizes an explicit diagnostic address" {
  val stream = ProbeStream()
  expect(when (client(stream, "localhost\u{0}evil", address: "proxy\u{0}:443")) {
    is Err(error) => error.path == "proxy\\0:443" && error.kind == IoKind.InvalidInput
    is Ok(conn)   => {
      conn.close()
      false
    }
  })
  expect(stream.closes.load() == 1)
  expect(stream.writes.load() == 0)
}

test "EOF during the handshake closes the owned stream" {
  val stream = ProbeStream()
  expect(when (client(stream, "localhost", Options(dangerouslyAcceptAnyCertificate: true))) {
    is Err(error) => {
      if (stream.reads.load() == 0) fail(error.message())
      error.path == "localhost" && error.detail.startsWith("tls:")
    }
    is Ok(conn)   => {
      conn.close()
      false
    }
  })
  expect(stream.reads.load() > 0)
  expect(stream.closes.load() == 1)
}

test "handshake write failure is propagated and closes the stream" {
  val stream = ProbeStream(failWrites: true)
  expect(when (client(stream, "localhost", Options(dangerouslyAcceptAnyCertificate: true))) {
    is Err(error) => error.path == "probe" && error.code == 5 && error.detail == "probe write failed"
    is Ok(conn)   => {
      conn.close()
      false
    }
  })
  expect(stream.writes.load() == 1)
  expect(stream.reads.load() == 0)
  expect(stream.closes.load() == 1)
}

test "parallel initialization reaches every clients transport" {
  val reached = Atomic<i64>(value: 0)
  scope {
    loop (_ in 0..<32) {
      async {
        val stream = ProbeStream()
        val failed = client(stream, "localhost", Options(dangerouslyAcceptAnyCertificate: true))
        when (failed) {
          is Err(_)   => {
            if (stream.reads.load() > 0 && stream.closes.load() == 1) reached.add(1)
          }
          is Ok(conn) => conn.close()
        }
      }
    }
  }
  expect(reached.load() == 32)
}

test "malformed certificate material returns a typed TLS failure" {
  expect(when (Certificate.fromPem("not a certificate", Secret.of("not a private key"))) {
    is Err(error) => error.kind == IoKind.Other && error.detail.startsWith("tls:")
    is Ok(cert)   => {
      cert.close()
      false
    }
  })
}

test "plaintext buffer takes preserve order and leave the remainder" {
  with conn = Conn(inner: ProbeStream(), session: Session(number: 0), address: "probe")
  expect(conn.buffered() == 0)
  expect(conn.take(10).isEmpty())
  conn.buffer.withLock(buffer => buffer.addAll([0, 255, 13, 10]))
  expect(conn.take(0).isEmpty())
  expect(conn.buffered() == 4)
  expect(conn.take(2) == [0, 255])
  expect(conn.buffered() == 2)
  expect(conn.take(10) == [13, 10])
  expect(conn.buffered() == 0)
}
