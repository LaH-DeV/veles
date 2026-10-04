/// DEFLATE and gzip (RFC 1951, RFC 1952), written in Veles.
///
/// ```veles
/// use compress
///
/// val packed = compress.gzip(body)                       // level 6
/// val again = try compress.gunzip(packed)                // up to 64 MiB out
/// val small = compress.gzip(body, level: 9)
/// ```
///
/// What comes out of `gunzip` is bounded: output past `max` (64 MiB unless
/// the caller says otherwise) is a `TooLarge`, so a few kilobytes of input
/// cannot be turned into gigabytes. Corrupt or cut-short input is a
/// `CompressError`, never a panic.

/// Why a stream could not be decoded.
public enum CompressKind {
  /// Not a valid stream: bad header, bad code, bad checksum.
  Corrupt
  /// The input ends before the stream does.
  Truncated
  /// The output went past the `max` the caller allowed.
  TooLarge
}

/// A stream that could not be decoded.
public error CompressError {
  public message: string
  public kind:    CompressKind
}

/// The default ceiling on decompressed output: 64 MiB.
const DEFAULT_MAX: i64 = 64 * 1024 * 1024

/// The bytes of a raw DEFLATE stream, decompressed. Output past `max` bytes
/// throws `TooLarge`.
public fun inflate(bytes: List<u8>, max: i64 = DEFAULT_MAX): List<u8> throws CompressError {
  val decoder = Inflater(limit: max)
  decoder.feed(bytes)
  decoder.end()
  val status = try decoder.run(max.max(1) + 1)
  if (status != FINISHED) throw corrupt("the output is larger than expected")
  decoder.all()
}

/// The bytes of a gzip file (RFC 1952), decompressed. Several members one
/// after another are joined, as `gzip -d` does. The checksum and the length
/// are verified.
public fun gunzip(bytes: List<u8>, max: i64 = DEFAULT_MAX): List<u8> throws CompressError {
  val result: MutableList<u8> = []
  var at = 0
  loop {
    val member = try gunzipMember(bytes, at, max - result.len())
    result.addAll(member.data)
    at = member.next
    if (at >= bytes.len()) break
  }
  result
}

struct Member {
  data: List<u8>
  next: i64
}

fun truncated(what: string): CompressError =
  CompressError(message: "gzip data ends inside $what", kind: CompressKind.Truncated)

fun byteAt(bytes: List<u8>, i: i64, what: string): i64 throws CompressError {
  val b = bytes.at(i) ?: throw truncated(what)
  b.toI64()
}

// The length of the gzip header at the start of bytes[from...], or -1 when
// the bytes end inside it (more may come).
fun gzipHeaderLength(bytes: List<u8>, from: i64): i64 throws CompressError {
  val have = bytes.len() - from
  if (have < 10) {
    // what is there must still be the start of a header
    if (have >= 1 && bytes.at(from) != 0x1f) throw corrupt("not a gzip stream (the magic bytes are missing)")
    if (have >= 2 && bytes.at(from + 1) != 0x8b) throw corrupt("not a gzip stream (the magic bytes are missing)")
    return -1
  }
  if (bytes.at(from) != 0x1f || bytes.at(from + 1) != 0x8b) throw corrupt("not a gzip stream (the magic bytes are missing)")
  if (bytes.at(from + 2) != 8) throw corrupt("a gzip compression method other than deflate")
  val flags = (bytes.at(from + 3) ?: 0).toI64()
  if (flags & 0xe0 != 0) throw corrupt("reserved gzip flags are set")
  var at = from + 10
  if (flags & 4 != 0) {
    if (at + 2 > bytes.len()) return -1
    val extra = (bytes.at(at) ?: 0).toI64() | ((bytes.at(at + 1) ?: 0).toI64() << 8)
    at += 2 + extra
  }
  if (flags & 8 != 0) {
    loop (at < bytes.len() && bytes.at(at) != 0) {
      at += 1
    }
    at += 1
  }
  if (flags & 16 != 0) {
    loop (at < bytes.len() && bytes.at(at) != 0) {
      at += 1
    }
    at += 1
  }
  if (flags & 2 != 0) at += 2
  if (at > bytes.len()) return -1
  at - from
}

fun gunzipMember(bytes: List<u8>, start: i64, max: i64): Member throws CompressError {
  val header = try gzipHeaderLength(bytes, start)
  if (header < 0) throw truncated("the header")
  val decoder = Inflater(limit: max)
  decoder.feed(bytes.drop(start + header))
  decoder.end()
  val status = try decoder.run(max.max(1) + 1)
  if (status != FINISHED) throw corrupt("the output is larger than expected")
  val data = decoder.all()
  val rest = decoder.unread()
  if (rest.len() < 8) throw truncated("the trailer")
  val crc = try le32(rest, 0)
  val size = try le32(rest, 4)
  if (crc != crc32(data)) throw corrupt("the checksum does not match")
  if (size != data.len() & 0xffffffff) throw corrupt("the length does not match")
  Member(data, next: bytes.len() - rest.len() + 8)
}

fun le32(bytes: List<u8>, at: i64): i64 throws CompressError {
  try byteAt(bytes, at, "the trailer") | (try byteAt(bytes, at + 1, "the trailer") << 8) | (try byteAt(bytes, at + 2, "the trailer") << 16) | (try byteAt(bytes, at + 3, "the trailer") << 24)
}

// ---------------------------------------------------------------------------
// CRC-32 (the gzip polynomial, reflected)

// eight tables of 256: table k holds the CRC of a byte followed by k zero
// bytes, so eight bytes are folded with eight independent lookups
fun buildCrcTable(): List<i64> {
  val table: MutableList<i64> = []
  loop (n in 0..<256) {
    var c = n
    loop (_ in 0..<8) {
      c = if (c & 1 == 1) 0xedb88320 ^ (c >> 1) else c >> 1
    }
    table.push(c)
  }
  loop (k in 1..<8) {
    loop (n in 0..<256) {
      val before = table.at((k - 1) * 256 + n) ?: 0
      table.push((table.at(before & 255) ?: 0) ^ (before >> 8))
    }
  }
  table.toList()
}

val crcTable: List<i64> = buildCrcTable()

/// The CRC-32 of `bytes`, as gzip stores it.
fun crc32(bytes: List<u8>): i64 = crc32Update(0, bytes)

// continues a CRC over more bytes; `crc` is the value so far (0 to begin)
fun crc32Update(crc: i64, bytes: List<u8>): i64 {
  var c = crc ^ 0xffffffff
  val n = bytes.len()
  var i = 0
  // SAFETY: every table index is below 8 * 256 (a byte plus a multiple of
  // 256), and i + 7 < n guards the eight reads
  unsafe {
    loop (i + 8 <= n) {
      val low = c ^ (bytes.atUnchecked(i).toI64() | (bytes.atUnchecked(i + 1).toI64() << 8) | (bytes.atUnchecked(i + 2).toI64() << 16) | (bytes.atUnchecked(i + 3).toI64() << 24))
      c = crcTable.atUnchecked(1792 + (low & 255)) ^ crcTable.atUnchecked(1536 + ((low >> 8) & 255)) ^ crcTable.atUnchecked(1280 + ((low >> 16) & 255)) ^ crcTable.atUnchecked(1024 + ((low >> 24) & 255)) ^ crcTable.atUnchecked(768 + bytes.atUnchecked(i + 4).toI64()) ^ crcTable.atUnchecked(512 + bytes.atUnchecked(i + 5).toI64()) ^ crcTable.atUnchecked(256 + bytes.atUnchecked(i + 6).toI64()) ^ crcTable.atUnchecked(bytes.atUnchecked(i + 7).toI64())
      i += 8
    }
    loop (i < n) {
      c = crcTable.atUnchecked((c ^ bytes.atUnchecked(i).toI64()) & 255) ^ (c >> 8)
      i += 1
    }
  }
  c ^ 0xffffffff
}

// ---------------------------------------------------------------------------
// compression

fun checkLevel(name: string, level: i64) {
  if (level < 0 || level > 9) panic("compress.$name: level is $level; it is 0 (stored) to 9 (smallest)")
}

/// `bytes` as a raw DEFLATE stream. `level` is 0 (stored, no compression) to
/// 9 (smallest, slowest); 6 is the usual trade. A level outside 0–9 panics.
public fun deflate(bytes: List<u8>, level: i64 = 6): List<u8> {
  checkLevel("deflate", level)
  val encoder = Deflater(level)
  encoder.feed(bytes)
  encoder.run()
  encoder.finish()
  encoder.take()
}

/// `bytes` as a gzip file (RFC 1952), the format of `.gz` files, `gzip` and
/// `Content-Encoding: gzip`. A level outside 0–9 panics.
public fun gzip(bytes: List<u8>, level: i64 = 6): List<u8> {
  checkLevel("gzip", level)
  val out: MutableList<u8> = []
  out.addAll(gzipHeader(level))
  out.addAll(deflate(bytes, level))
  out.addAll(gzipTrailer(crc32(bytes), bytes.len()))
  out
}

// magic, deflate, no flags, no time, the speed hint, "unknown OS"
fun gzipHeader(level: i64): List<u8> =
  [0x1f, 0x8b, 8, 0, 0, 0, 0, 0, if (level == 9) 2 else if (level == 1) 4 else 0, 255]

fun gzipTrailer(crc: i64, size: i64): List<u8> {
  val out: MutableList<u8> = []
  loop (k in 0..<4) {
    out.push(((crc >> (8 * k)) & 255).wrapU8())
  }
  loop (k in 0..<4) {
    out.push(((size >> (8 * k)) & 255).wrapU8())
  }
  out
}
