// Streaming gzip: a writer that compresses as bytes are written and a reader
// that decompresses as they arrive, each over an io.Stream. Memory stays at
// a window and a buffer whatever the size of the data.
use io

// where compressed bytes go, and where they come from
fun sinkOf(to: io.Stream): sendable fun(List<u8>) suspends throws IoError = bytes => try to.write(bytes)

fun sourceOf(from: io.Stream): sendable fun(): List<u8> suspends throws IoError = () => try from.read()

/// Compresses what is written to it, as gzip, onto another stream:
///
/// ```veles
/// var gz = compress.GzipWriter(to: conn)
/// try gz.write(chunk)
/// try gz.write(more)
/// try gz.finish()        // the trailer; `conn` stays open
/// ```
///
/// Nothing reaches the other stream until enough has been written to fill a
/// block, `flush()` is called or `finish()` ends the stream. Closing `to` is
/// the caller's.
public struct GzipWriter {
  private var inner: Gzipper

  /// A writer onto `to`; `level` is 0 (stored) to 9 (smallest), 6 by default.
  /// A level outside 0–9 panics.
  init(to: io.Stream, level: i64 = 6) {
    this.inner = Gzipper(sink: sinkOf(to), encoder: GzipEncoder(level))
  }

  /// Compresses `bytes`.
  public fun write(bytes: List<u8>) suspends throws IoError {
    try this.inner.write(bytes)
  }

  /// Compresses `text` as UTF-8.
  public fun writeText(text: string) suspends throws IoError {
    try this.inner.write(text.bytes())
  }

  /// Sends everything written so far in a form the reader can decode now
  /// (a sync flush): what a server does between events of a stream. Costs a
  /// little compression ratio.
  public fun flush() suspends throws IoError {
    try this.inner.flush()
  }

  /// Ends the stream: sends what is left, the checksum and the length. Nothing
  /// may be written after it; the other stream is not closed.
  public fun finish() suspends throws IoError {
    try this.inner.finish()
  }
}

/// A gzip compressor with no stream in it: bytes go in, the compressed bytes
/// that are ready come back, and the caller sends them wherever it likes —
/// which is what a writer inside a lock (nothing may suspend there) needs.
///
/// ```veles
/// var encoder = compress.GzipEncoder(level: 6)
/// val first = encoder.push(chunk)    // maybe empty: a block has not filled yet
/// val last = encoder.finish()        // the rest, the checksum and the length
/// ```
public struct GzipEncoder {
  private var deflater: Deflater
  private var level:    i64
  private var crc:      i64 = 0
  private var size:     i64 = 0
  private var started:  bool = false
  private var done:     bool = false

  /// An encoder at `level` 0 (stored) to 9 (smallest); a level outside that panics.
  init(level: i64 = 6) {
    checkLevel("GzipEncoder", level)
    this.deflater = Deflater(level)
    this.level = level
  }

  /// Compresses `bytes`; what comes back is whatever is ready (the first call
  /// also carries the gzip header). `push` after `finish` panics.
  public fun push(bytes: List<u8>): List<u8> {
    if (this.done) panic("compress.GzipEncoder: push after finish")
    this.crc = crc32Update(this.crc, bytes)
    this.size += bytes.len()
    this.deflater.feed(bytes)
    this.deflater.run()
    this.ready()
  }

  /// Everything pushed so far, in a form a reader can decode now (a sync
  /// flush); costs a little compression ratio.
  public fun flush(): List<u8> {
    if (this.done) return []
    this.deflater.sync()
    this.ready()
  }

  /// Ends the stream: the rest of the data, the checksum and the length.
  public fun finish(): List<u8> {
    if (this.done) return []
    this.deflater.finish()
    this.done = true
    val out: MutableList<u8> = []
    out.addAll(this.ready())
    out.addAll(gzipTrailer(this.crc, this.size))
    out.toList()
  }

  // the compressed bytes taken out, the header in front the first time
  private fun ready(): List<u8> {
    val out: MutableList<u8> = []
    if (!this.started) {
      this.started = true
      out.addAll(gzipHeader(this.level))
    }
    out.addAll(this.deflater.take())
    out.toList()
  }
}

// The writer over a function that takes the compressed bytes (the HTTP
// layer hands it the body writer's).
struct Gzipper {
  sink:        sendable fun(List<u8>) suspends throws IoError
  var encoder: GzipEncoder

  fun write(bytes: List<u8>) suspends throws IoError {
    val packed = this.encoder.push(bytes)
    if (!packed.isEmpty()) try this.sink(packed)
  }

  fun flush() suspends throws IoError {
    val packed = this.encoder.flush()
    if (!packed.isEmpty()) try this.sink(packed)
  }

  fun finish() suspends throws IoError {
    val packed = this.encoder.finish()
    if (!packed.isEmpty()) try this.sink(packed)
  }
}

/// Decompresses gzip from another stream as it is read. Several members one
/// after another are read as one stream, and the checksum and length of each
/// are verified when it ends (the last bytes of a corrupt member are not
/// handed out before the check fails only if they fit one `read`).
///
/// ```veles
/// var gz = compress.GzipReader(from: conn, max: 1024 * 1024)
/// loop {
///   val chunk = try gz.read()
///   if (chunk.isEmpty()) break
///   process(chunk)
/// }
/// ```
public struct GzipReader {
  private var inner: Gunzipper

  /// A reader of the gzip data on `from`. More than `max` bytes of output
  /// throws a `TooLarge` (64 MiB unless said otherwise).
  init(from: io.Stream, max: i64 = DEFAULT_MAX) {
    this.inner = Gunzipper(pull: sourceOf(from), max)
  }

  /// Up to `max` bytes of decompressed data; empty at the end of the stream.
  public fun read(max: i64 = 65536): List<u8> suspends throws IoError | CompressError {
    try this.inner.read(max)
  }

  /// The rest of the data, in one list (within the reader's `max`).
  public fun readAll(): List<u8> suspends throws IoError | CompressError {
    val out: MutableList<u8> = []
    loop {
      val chunk = try this.inner.read(65536)
      if (chunk.isEmpty()) break
      out.addAll(chunk)
    }
    out
  }
}

// header, body, trailer, member after member
const AT_MEMBER: i64 = 0
const IN_MEMBER: i64 = 1

struct Gunzipper {
  pull:         sendable fun(): List<u8> suspends throws IoError
  max:          i64
  var decoder:  Inflater = Inflater(limit: 0)
  var state:    i64 = 0
  var head:     MutableList<u8> = []
  var crc:      i64 = 0
  var size:     i64 = 0
  var produced: i64 = 0
  var members:  i64 = 0
  var eof:      bool = false

  fun read(max: i64): List<u8> suspends throws IoError | CompressError {
    loop {
      if (this.decoder.pending() > 0) return this.handOut(this.decoder.takeUpTo(max))
      if (this.eof) return []
      if (this.state == AT_MEMBER) {
        val length = try gzipHeaderLength(this.head.toList(), 0)
        if (length < 0) {
          val chunk = try this.pull()
          if (chunk.isEmpty()) {
            if (this.head.isEmpty() && this.members > 0) {
              this.eof = true
              return []
            }
            throw truncated("the header")
          }
          this.head.addAll(chunk)
          continue
        }
        this.decoder = Inflater(limit: this.max - this.produced)
        this.decoder.feed(this.head.drop(length))
        this.head.clear()
        this.state = IN_MEMBER
        this.crc = 0
        this.size = 0
        continue
      }
      val status = try this.decoder.run(max)
      if (status == NEED_INPUT) {
        val chunk = try this.pull()
        if (chunk.isEmpty()) this.decoder.end() else this.decoder.feed(chunk)
      } else if (status == FINISHED && this.decoder.pending() == 0) {
        try this.trailer()
      }
    }
  }

  fun handOut(chunk: List<u8>): List<u8> {
    this.crc = crc32Update(this.crc, chunk)
    this.size += chunk.len()
    this.produced += chunk.len()
    chunk
  }

  // the checksum and the length after the last block, then on to the next member
  fun trailer() suspends throws IoError | CompressError {
    val tail: MutableList<u8> = []
    tail.addAll(this.decoder.unread())
    loop (tail.len() < 8) {
      val chunk = try this.pull()
      if (chunk.isEmpty()) throw truncated("the trailer")
      tail.addAll(chunk)
    }
    val fixed = tail.toList()
    val crc = try le32(fixed, 0)
    val size = try le32(fixed, 4)
    if (crc != this.crc) throw corrupt("the checksum does not match")
    if (size != this.size & 0xffffffff) throw corrupt("the length does not match")
    this.head.addAll(fixed.drop(8))
    this.members += 1
    this.state = AT_MEMBER
    this.decoder = Inflater(limit: 0)
  }
}
