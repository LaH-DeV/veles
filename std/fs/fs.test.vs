// Tests of std/fs's open files (D97): reading a piece at a time, writing as
// data arrives, and what a closed or wrongly opened file says.

use io, os, path, time

fun scratch(name: string): string {
  val dir = path.join(os.tempDir(), "veles-fs-test")
  when (mkdir(dir)) {
    is Err(e) => panic("cannot make the scratch directory: ${e.message()}")
    is Ok(_)  => path.join(dir, name)
  }
}

test "a file is read from where the last read ended, and at any offset" {
  val p = scratch("read.bin")
  try writeFile(p, "0123456789")
  with (f = try open(p)) {
    expect(try f.size() == 10)
    expect((try f.read(4)).decodeUtf8() == "0123")
    expect((try f.read(4)).decodeUtf8() == "4567")
    // readAt does not move the place `read` continues from
    expect((try f.readAt(1, 3)).decodeUtf8() == "123")
    expect((try f.read(4)).decodeUtf8() == "89")
    expect(try f.read(4).isEmpty())
    expect(try f.readAt(10, 4).isEmpty())
    expect(try f.readAt(50, 4).isEmpty())
    expect((try f.readAt(8, 100)).decodeUtf8() == "89")
  }
  try remove(p)
}

test "a file opened to write is emptied, an appended one is kept" {
  val p = scratch("write.txt")
  with (f = try open(p, FileMode.Write)) {
    try f.write("abc".bytes())
    try f.write("def".bytes())
  }
  expect(try readFile(p) == "abcdef")
  with (f = try open(p, FileMode.Append)) {
    try f.write("!".bytes())
  }
  expect(try readFile(p) == "abcdef!")
  with (f = try open(p, FileMode.Write)) {
    try f.write("x".bytes())
  }
  expect(try readFile(p) == "x")
  try remove(p)
}

test "bytes that are not text survive a round trip" {
  val p = scratch("raw.bin")
  val data: List<u8> = [0, 255, 10, 13, 0, 128]
  with (f = try open(p, FileMode.Write)) {
    try f.write(data)
  }
  with (f = try open(p)) {
    expect(try f.read(100) == data)
  }
  try remove(p)
}

test "opening what is not there, or a directory to read, fails" {
  expect(when (open(scratch("nothing-here"))) {
    is Err(e) => e.kind == IoKind.NotFound
    is Ok(_)  => false
  })
  expect(when (open(path.join(os.tempDir(), "veles-fs-test"))) {
    is Err(_) => true
    is Ok(f)  => {
      f.close()
      false
    }
  })
}

test "a closed file refuses, and closing twice is fine" {
  val p = scratch("closed.txt")
  try writeFile(p, "hi")
  val f = try open(p)
  f.close()
  f.close()
  expect(when (f.read(1)) {
    is Err(e) => e.message().contains("closed")
    is Ok(_)  => false
  })
  try remove(p)
}

test "a file opened one way refuses the other" {
  val p = scratch("mode.txt")
  try writeFile(p, "hi")
  with (r = try open(p)) {
    expect(when (r.write("x".bytes())) {
      is Err(_) => true
      is Ok(_)  => false
    })
  }
  with (w = try open(p, FileMode.Append)) {
    expect(when (w.read(1)) {
      is Err(_) => true
      is Ok(_)  => false
    })
  }
  try remove(p)
}

test "stat reports size, kind and a recent write time" {
  val p = scratch("stat.txt")
  try writeFile(p, "12345")
  val st = try stat(p)
  expect(st.size == 5 && st.isFile() && !st.isDir)
  val now = time.now().toSeconds()
  expect(st.modified.toSeconds() > now - 3600 && st.modified.toSeconds() < now + 3600)
  expect((try stat(path.join(os.tempDir(), "veles-fs-test"))).isDir)
  try remove(p)
}

// io.Stream over a file (D128): the same code a connection runs through

// everything a stream holds, read through the trait object
fun drain(s: io.Stream): List<u8> throws IoError {
  val out: MutableList<u8> = []
  loop {
    val chunk = try s.read(3)
    if (chunk.isEmpty()) break
    out.addAll(chunk)
  }
  out
}

fun lines<S: io.Stream>(s: S, max: i64): List<string> throws IoError | io.TooLong {
  val out: MutableList<string> = []
  loop {
    val line = try s.readLine(max)
    if (line == null) break
    out.push(line)
  }
  out
}

test "a file is a stream: reads, lines and exact reads" {
  val p = scratch("lines.txt")
  try writeFile(p, "one\r\ntwo\n\nlast")
  with (f = try open(p)) {
    val s: io.Stream = f
    expect((try drain(s)).decodeUtf8() == "one\r\ntwo\n\nlast")
  }
  with (f = try open(p)) {
    expect(try lines(f, 16) == ["one", "two", "", "last"])
  }
  with (f = try open(p)) {
    expect((try f.readExact(5)).decodeUtf8() == "one\r\n")
    expect(try f.readLine(16) == "two")
    expect((try f.readExact(100)).decodeUtf8() == "\nlast")
    expect(try f.readLine(16) == null)
  }
  try remove(p)
}

test "a line past the ceiling is TooLong, on a file" {
  val p = scratch("long.txt")
  try writeFile(p, "short\n" + "x".repeat(9000) + "\nafter\n")
  with (f = try open(p)) {
    expect(try f.readLine(10) == "short")
    val r = f.readLine(100)
    when (r) {
      is Err(e) => expect(e is io.TooLong)
      is Ok(_)  => expect(false)
    }
  }
  try remove(p)
}

test "a file's write and shutdownWrite through a stream" {
  val p = scratch("out.txt")
  with (f = try open(p, FileMode.Write)) {
    val s: io.Stream = f
    try s.writeText("a")
    try s.write("bc".bytes())
    try s.shutdownWrite()
    try s.writeText("d")
  }
  expect(try readFile(p) == "abcd")
  try remove(p)
}
