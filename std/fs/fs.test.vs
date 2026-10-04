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

fun readLines<S: io.Stream>(s: S, max: i64): List<string> throws IoError | io.TooLong {
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
    expect(try readLines(f, 16) == ["one", "two", "", "last"])
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

test "seek moves where read and write continue" {
  val p = scratch("seek.bin")
  try writeFile(p, "0123456789")
  with (f = try open(p)) {
    try f.seek(6)
    expect((try f.read(2)).decodeUtf8() == "67")
    try f.seek(1)
    expect((try f.read(2)).decodeUtf8() == "12")
    try f.seek(50)
    expect(try f.read(4).isEmpty())
  }
  with (w = try open(p, FileMode.Write)) {
    try w.write("abcdef".bytes())
    try w.seek(2)
    try w.write("XY".bytes())
    try w.sync()
  }
  expect(try readFile(p) == "abXYef")
  expectPanics(() => seekBelowZero(p))
  try remove(p)
}

fun seekBelowZero(p: string) {
  val opened = open(p)
  when (opened) {
    is Ok(f)  => {
      val _ = f.seek(-1)
    }
    is Err(_) => { }
  }
}

test "an exclusive lock keeps a second taker out until it is let go" {
  val p = scratch("lock.pid")
  try writeFile(p, "")
  with (a = try open(p, FileMode.Append)) {
    with (b = try open(p, FileMode.Append)) {
      with (held = try a.lock()) {
        expect(try b.tryLock() == null)
      }
      // let go with the block: now b can take it, and a cannot
      with (second = (try b.tryLock()) ?: panic("the lock was free")) {
        expect(try a.tryLock() == null)
      }
      with (third = try a.lock()) {
        expect(try b.tryLock() == null)
      }
    }
  }
  try remove(p)
}

test "closing the file lets its lock go" {
  val p = scratch("lock2.pid")
  try writeFile(p, "")
  with (b = try open(p, FileMode.Append)) {
    val a = try open(p, FileMode.Append)
    val held = try a.lock()
    expect(try b.tryLock() == null)
    held.close()
    held.close()
    with (now = (try b.tryLock()) ?: panic("closing the lock did not let it go")) {
      expect(try a.tryLock() == null)
    }
    // the file's own close lets go as well
    val again = try a.lock()
    expect(try b.tryLock() == null)
    a.close()
    with (last = (try b.tryLock()) ?: panic("closing the file did not let the lock go")) {
      again.close()
    }
  }
  try remove(p)
}

test "writeAtomic replaces the file whole and leaves no temporary file behind" {
  val dir = scratch("atomic")
  try mkdir(dir)
  val p = path.join(dir, "state.bin")
  try writeAtomic(p, "first".bytes())
  expect(try readFile(p) == "first")
  try writeAtomic(p, "second, longer".bytes())
  expect(try readFile(p) == "second, longer")
  try writeAtomic(p, [])
  expect(try readBytes(p).isEmpty())
  expect((try listDir(dir)) == ["state.bin"])
  try remove(p)
  try remove(dir)
}

test "writeAtomic into a missing directory fails and leaves nothing" {
  val dir = scratch("nowhere")
  val p = path.join(dir, "inside", "x")
  expect(writeAtomic(p, [1]) is Err)
  expect(!exists(dir))
}

test "copy reproduces a file bigger than one piece, and replaces the target" {
  val from = scratch("copy-from.bin")
  val to = scratch("copy-to.bin")
  val data: MutableList<u8> = []
  loop (i in 0..<200000) {
    data.push((i % 251).wrapU8())
  }
  try writeBytes(from, data.toList())
  try writeFile(to, "old contents that are much shorter than the new")
  try copy(from, to)
  expect(try readBytes(to) == data.toList())
  try copy(from, from)
  expect(try readBytes(from) == data.toList())
  expect(copy(scratch("missing.bin"), to) is Err)
  try remove(from)
  try remove(to)
}

test "lines hands out a file one line at a time" {
  val p = scratch("lines.txt")
  try writeFile(p, "one\r\ntwo\n\nlast")
  val got: MutableList<string> = []
  loop (line in try lines(p, max: 64)) {
    got.push(try line)
  }
  expect(got.toList() == ["one", "two", "", "last"])
  // the adapters work on it, and an empty file has no lines
  expect((try lines(p, max: 64)).map(r => r catch (e) {
    "?"
  }).toList() == ["one", "two", "", "last"])
  try writeFile(p, "")
  expect((try lines(p, max: 64)).toList().isEmpty())
  try remove(p)
}

test "a line past max is an error item and ends the lines" {
  val p = scratch("lines-long.txt")
  try writeFile(p, "ok\n" + "x".repeat(100) + "\nnever seen\n")
  val seen: MutableList<string> = []
  var failure = ""
  loop (line in try lines(p, max: 10)) {
    when (line) {
      is Ok(text) => seen.push(text)
      is Err(e)   => failure = e.message()
    }
  }
  expect(seen.toList() == ["ok"])
  expect(failure.contains("longer than 10 bytes"))
  try remove(p)
}

test "lines closes the file when asked to stop early, and a missing file throws" {
  val p = scratch("lines-early.txt")
  try writeFile(p, "a\nb\nc\n")
  with (stream = try lines(p, max: 8)) {
    val first = stream.next()
    expect(first != null)
  }
  expect(lines(scratch("lines-missing.txt"), max: 8) is Err)
  expectPanics(() => lines(p, max: 0))
  try remove(p)
}
