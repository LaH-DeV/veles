// Tests of std/compress (D124): published-style vectors made by Go's
// compress/flate, round trips at every level, the three ways a stream can be
// bad, the output ceiling, and the streaming writer and reader over a stream
// that hands out a few bytes at a time.

use io

// text with structure, so every block type has something to do
test fun sample(lines: i64): List<u8> {
  val out: MutableList<u8> = []
  loop (i in 0..<lines) {
    out.addAll("line $i: the value is ${i * 7 % 13} and the name is item-${i % 50}\n".bytes())
  }
  out
}

// bytes with no structure: xorshift
test fun noise(n: i64): List<u8> {
  val out: MutableList<u8> = []
  var x: u64 = 88172645463325252
  loop (_ in 0..<n) {
    x = x ^ (x << 13)
    x = x ^ (x >> 7)
    x = x ^ (x << 17)
    out.push((x & 255).wrapU8())
  }
  out
}

test "a gzip file written by Go (level 9) is read" {
  val packed: List<u8> = [31, 139, 8, 0, 0, 0, 0, 0, 2, 255, 202, 72, 205, 201, 201, 87, 192, 32, 185, 0, 1, 0, 0, 255, 255, 0, 136, 89, 11, 24, 0, 0, 0]
  expect(try gunzip(packed) == "hello hello hello hello\n".bytes())
}

test "a Huffman-only gzip file written by Go is read" {
  val packed: List<u8> = [31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 4, 192, 129, 0, 0, 0, 0, 131, 48, 214, 87, 254, 12, 223, 182, 1, 168, 86, 61, 0, 0, 255, 255, 206, 141, 170, 45, 16, 0, 0, 0]
  expect(try gunzip(packed) == "aaaabbbbccccdddd".bytes())
}

test "a raw deflate stream written by Go is read" {
  val packed: List<u8> = [74, 76, 74, 70, 67, 128, 0, 0, 0, 255, 255]
  expect(try inflate(packed) == "abcabcabcabcabcabc".bytes())
}

test "every level round-trips, as gzip and as raw deflate" {
  val inputs: List<List<u8>> = [[], [7], "hello".bytes(), sample(900), noise(5000), noise(70000).concat(sample(300))]
  loop (data in inputs) {
    loop (level in 0..9) {
      expect(try gunzip(gzip(data, level)) == data)
      expect(try inflate(deflate(data, level)) == data)
    }
  }
}

test "better levels do not make text bigger, and text shrinks" {
  val data = sample(3000)
  val fast = gzip(data, 1).len()
  val best = gzip(data, 9).len()
  expect(best <= fast)
  expect(best < data.len() / 4)
  expect(gzip(data, 0).len() > data.len())
}

test "noise is stored, not expanded" {
  val data = noise(100000)
  expect(gzip(data).len() < data.len() + 100)
}

test "a long run and a far match both come back" {
  val far: MutableList<u8> = []
  val key = noise(300)
  far.addAll(key)
  far.addAll(noise(30000))
  far.addAll(key)
  far.addAll(MutableList<u8>.repeat(65, 5000).toList())
  expect(try gunzip(gzip(far.toList(), 9)) == far.toList())
}

test "a damaged stream is Corrupt, a short one Truncated, never a panic" {
  val packed = gzip(sample(500))
  val noMagic: List<u8> = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
  expect(gunzip(noMagic) is Err)
  when (gunzip(noMagic)) {
    is Err(e) => expect(e.kind == CompressKind.Corrupt)
    is Ok(_)  => expect(false)
  }
  // every proper prefix is cut short
  loop (n in [0, 3, 10, 20, packed.len() / 2, packed.len() - 9, packed.len() - 1]) {
    when (gunzip(packed.take(n))) {
      is Err(e) => expect(e.kind == CompressKind.Truncated)
      is Ok(_)  => expect(false)
    }
  }
  // a flipped bit in the data or in the checksum is caught
  val flipped = packed.toMutable()
  flipped.set(30, (packed.at(30) ?: 0) ^ 4)
  expect(gunzip(flipped.toList()) is Err)
  val badCrc = packed.toMutable()
  badCrc.set(packed.len() - 8, (packed.at(packed.len() - 8) ?: 0) ^ 1)
  when (gunzip(badCrc.toList())) {
    is Err(e) => expect(e.kind == CompressKind.Corrupt)
    is Ok(_)  => expect(false)
  }
}

test "no input makes the decoder panic" {
  // short prefixes of noise, as raw deflate
  loop (seed in 0..<40) {
    val junk = noise(10 + seed * 7)
    val _ = inflate(junk, 100000)
    val _ = gunzip(junk, 100000)
  }
}

test "output past max is TooLarge, whatever the ratio" {
  val bomb = gzip(MutableList<u8>.repeat(0, 3000000).toList(), 9)
  expect(bomb.len() < 5000)
  when (gunzip(bomb, 100000)) {
    is Err(e) => expect(e.kind == CompressKind.TooLarge)
    is Ok(_)  => expect(false)
  }
  expect((try gunzip(bomb)).len() == 3000000)
  when (inflate(deflate(sample(4000)), 1000)) {
    is Err(e) => expect(e.kind == CompressKind.TooLarge)
    is Ok(_)  => expect(false)
  }
}

test "members one after another read as one" {
  val joined = gzip("first ".bytes()).concat(gzip("second".bytes()))
  expect(try gunzip(joined) == "first second".bytes())
}

// A stream over memory that gives out at most `step` bytes per read, so a
// decoder is cut off in the middle of everything.
struct Pipe {
  incoming: Mutex<MutableList<u8>>
  outgoing: Mutex<MutableList<u8>>
  step:     i64

  static fun of(data: List<u8>, step: i64): Pipe {
    val inbound = data.toMutable()
    val outbound: MutableList<u8> = []
    Pipe(incoming: Mutex(value: inbound), outgoing: Mutex(value: outbound), step)
  }

  fun written(): List<u8> = this.outgoing.withLock(out => out.toList())

  implement io.Stream {
    fun read(max: i64 = 65536): List<u8> {
      this.incoming.withLock(inbound => {
        val n = max.min(this.step).min(inbound.len())
        val chunk = inbound.take(n)
        val rest = inbound.drop(n)
        inbound.clear()
        inbound.addAll(rest)
        chunk
      })
    }
    fun readExact(n: i64): List<u8> = try this.read(n)
    fun readLine(max: i64): string? = null
    fun write(bytes: List<u8>) {
      this.outgoing.withLock(out => out.addAll(bytes))
    }
    fun writeText(text: string) {
      try this.write(text.bytes())
    }
    fun shutdownWrite() { }
  }

  implement Closeable {
    fun close() { }
  }
}

test "the streaming reader decodes input that arrives a few bytes at a time" {
  val data = sample(2500).concat(noise(9000)).concat(sample(40))
  loop (step in [1, 2, 7, 100, 65536]) {
    loop (level in [0, 1, 6, 9]) {
      val pipe = Pipe.of(gzip(data, level), step)
      var reader = GzipReader(from: pipe)
      val out: MutableList<u8> = []
      loop {
        val chunk = try reader.read(500)
        if (chunk.isEmpty()) break
        expect(chunk.len() <= 500)
        out.addAll(chunk)
      }
      expect(out.toList() == data)
    }
  }
}

test "the streaming reader reports a cut stream and a ceiling" {
  val packed = gzip(sample(800))
  var cut = GzipReader(from: Pipe.of(packed.take(packed.len() - 12), 64))
  when (cut.readAll()) {
    is Err(e) => expect(e is CompressError)
    is Ok(_)  => expect(false)
  }
  var small = GzipReader(from: Pipe.of(packed, 4096), max: 1000)
  when (small.readAll()) {
    is Err(e) => expect(e is CompressError)
    is Ok(_)  => expect(false)
  }
  var joined = GzipReader(from: Pipe.of(gzip("ab".bytes()).concat(gzip("cd".bytes())), 3))
  expect(try joined.readAll() == "abcd".bytes())
}

test "the streaming writer's output is a gzip file, with or without flushes" {
  val data = sample(4000).concat(noise(5000))
  loop (level in [0, 1, 6, 9]) {
    val pipe = Pipe.of([], 1)
    var writer = GzipWriter(to: pipe, level)
    var at = 0
    loop (at < data.len()) {
      try writer.write(data.slice(at, at + 3001))
      at += 3001
      if (at % 6002 == 0) try writer.flush()
    }
    try writer.finish()
    expect(try gunzip(pipe.written()) == data)
  }
}

test "a level outside 0-9 panics at the caller" {
  expectPanics(() => gzip([1], 10))
  expectPanics(() => deflate([1], -1))
  expectPanics(() => GzipWriter(to: Pipe.of([], 1), level: 12))
}
