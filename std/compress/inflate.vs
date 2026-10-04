// DEFLATE decoding (RFC 1951), resumable: input is fed in pieces and the
// decoder stops where it runs out, picking up at the last whole symbol (or
// the start of the block header) when more arrives. A whole buffer is the
// one-piece case; the streaming reader is the many-piece one.

// length symbols 257..285: base length and extra bits
const LENGTH_BASE: Array<i64, 29> = [
  3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31,
  35, 43, 51, 59, 67, 83, 99, 115, 131, 163, 195, 227, 258,
]
const LENGTH_EXTRA: Array<i64, 29> = [
  0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2,
  3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0,
]
// distance symbols 0..29
const DIST_BASE: Array<i64, 30> = [
  1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193,
  257, 385, 513, 769, 1025, 1537, 2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577,
]
const DIST_EXTRA: Array<i64, 30> = [
  0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6,
  7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13,
]
// the order the code-length code's lengths are sent in
const CLEN_ORDER: Array<i64, 19> = [16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15]

/// What `Inflater.run` stopped for.
const NEED_INPUT: i64 = 0
const FINISHED: i64 = 1
const FULL: i64 = 2

// block states
const AT_HEADER: i64 = 0
const IN_STORED: i64 = 1
const IN_CODES: i64 = 2
const ALL_DONE: i64 = 3

// the window DEFLATE may reach back into
const WINDOW: i64 = 32768

fun corrupt(why: string): CompressError =
  CompressError(message: "compressed data is corrupt: $why", kind: CompressKind.Corrupt)

// Fills `table` (2^maxLen entries) from `n` code lengths starting at `from`;
// the result is maxLen. An entry is `symbol << 4 | length`, indexed by the
// next maxLen input bits (DEFLATE sends codes starting from their top bit,
// so the index is the code reversed); 0 marks a code nobody has.
fun buildTable(lengths: MutableList<i64>, from: i64, n: i64, table: MutableList<i64>): i64 throws CompressError {
  val counts: MutableList<i64> = MutableList.repeat(0, 16)
  var maxLen = 0
  loop (i in 0..<n) {
    val len = lengths.at(from + i) ?: 0
    counts.set(len, (counts.at(len) ?: 0) + 1)
    if (len > maxLen) maxLen = len
  }
  var left = 1
  loop (len in 1..15) {
    left = (left << 1) - (counts.at(len) ?: 0)
    if (left < 0) throw corrupt("a Huffman code over-subscribes its lengths")
  }
  counts.set(0, 0)
  val next: MutableList<i64> = MutableList.repeat(0, 17)
  var code = 0
  loop (len in 1..15) {
    code = (code + (counts.at(len - 1) ?: 0)) << 1
    next.set(len, code)
  }
  if (maxLen == 0) {
    maxLen = 1  // no codes at all (no distances used): every entry invalid
  }
  val size = 1 << maxLen
  table.clear()
  table.reserve(size)
  loop (_ in 0..<size) {
    table.push(0)
  }
  loop (sym in 0..<n) {
    val len = lengths.at(from + sym) ?: 0
    if (len == 0) continue
    val canonical = next.at(len) ?: 0
    next.set(len, canonical + 1)
    // reverse the code's bits: the stream delivers its top bit first
    var reversed = 0
    var rest = canonical
    loop (_ in 0..<len) {
      reversed = (reversed << 1) | (rest & 1)
      rest = rest >> 1
    }
    val entry = (sym << 4) | len
    var at = reversed
    loop (at < size) {
      table.set(at, entry)
      at += 1 << len
    }
  }
  maxLen
}

/// Decodes a DEFLATE stream a piece at a time.
struct Inflater {
  private var input: MutableList<u8> = []
  private var pos:   i64 = 0
  private var bits:  i64 = 0
  private var count: i64 = 0
  private var ended: bool = false

  // the output: `dropped` bytes were handed out and discarded before this
  // list starts; `delivered` of the list were handed out and kept as window
  private var out:       MutableList<u8> = []
  private var dropped:   i64 = 0
  private var delivered: i64 = 0
  private var limit:     i64

  private var state:      i64 = 0
  private var last:       bool = false
  private var storedLeft: i64 = 0

  private var lengths:   MutableList<i64> = MutableList.repeat(0, 320)
  private var litTable:  MutableList<i64> = []
  private var litBits:   i64 = 0
  private var distTable: MutableList<i64> = []
  private var distBits:  i64 = 0

  init(limit: i64) {
    this.limit = limit
  }

  /// More compressed bytes. Whatever the last call left unread is kept.
  fun feed(data: List<u8>) {
    if (this.pos > 0 && this.pos == this.input.len()) {
      this.input.clear()
      this.pos = 0
    }
    this.input.addAll(data)
  }

  /// No more input will come: running out now is a truncation.
  fun end() {
    this.ended = true
  }

  /// Input bytes the decoder has not consumed (whole bytes after the last
  /// block, once it has finished; the gzip trailer starts there).
  fun unread(): List<u8> {
    // bytes already pulled into the bit buffer belong to the stream
    // unless they lie beyond the final block's last byte
    val back = this.count / 8
    this.pos -= back
    this.count -= back * 8
    this.bits = this.bits & ((1 << this.count) - 1)
    this.input.drop(this.pos)
  }

  /// Total bytes produced so far.
  fun total(): i64 = this.dropped + this.out.len()

  /// Output produced and not yet taken.
  fun pending(): i64 = this.out.len() - this.delivered

  /// The output not yet taken, at most `n` bytes; the last 32 KiB taken stay as
  /// the window a later match may reach into.
  fun takeUpTo(n: i64): List<u8> {
    val upto = (this.delivered + n).min(this.out.len())
    val fresh = this.out.slice(this.delivered, upto)
    this.delivered = upto
    val cut = this.delivered - WINDOW
    if (cut > WINDOW) {
      this.out = this.out.drop(cut).toMutable()
      this.dropped += cut
      this.delivered -= cut
    }
    fresh
  }

  /// All the output not yet taken.
  fun take(): List<u8> = this.takeUpTo(this.pending())

  /// Everything produced, for a stream decoded in one go: nothing was taken.
  fun all(): List<u8> = this.out.toList()

  private fun need(n: i64): bool {
    loop (this.count < n) {
      if (this.pos >= this.input.len()) return false
      // SAFETY: pos < len was just checked
      val b = unsafe {
        this.input.atUnchecked(this.pos)
      }
      this.bits = this.bits | (b.toI64() << this.count)
      this.pos += 1
      this.count += 8
    }
    true
  }

  private fun eat(n: i64): i64 {
    val v = this.bits & ((1 << n) - 1)
    this.bits = this.bits >> n
    this.count -= n
    v
  }

  // the next symbol of a table, -1 when the input ends inside it
  private fun symbol(table: MutableList<i64>, width: i64): i64 throws CompressError {
    val enough = this.need(width)
    // SAFETY: the index has `width` bits and the table has 2^width entries
    val entry = unsafe {
      table.atUnchecked(this.bits & ((1 << width) - 1))
    }
    val len = entry & 15
    if (len == 0) {
      if (!enough) return -1
      throw corrupt("a code that no symbol has")
    }
    if (len > this.count) return -1
    this.bits = this.bits >> len
    this.count -= len
    entry >> 4
  }

  /// Decodes as much as the input allows, stopping early with `FULL` once
  /// `room` bytes of output wait to be taken. `NEED_INPUT` means feed more
  /// (or, after `end()`, that the stream was cut short — thrown here).
  fun run(room: i64): i64 throws CompressError {
    loop {
      if (this.state == ALL_DONE) return FINISHED
      if (this.state == AT_HEADER) {
        val progressed = try this.header()
        if (!progressed) return try this.starved()
      } else if (this.state == IN_STORED) {
        val got = try this.stored(room)
        if (got == NEED_INPUT) return try this.starved()
        if (got == FULL) return FULL
      } else {
        val got = try this.codes(room)
        if (got == NEED_INPUT) return try this.starved()
        if (got == FULL) return FULL
      }
    }
  }

  private fun starved(): i64 throws CompressError {
    if (this.ended) {
      throw CompressError(message: "compressed data ends before its last block does", kind: CompressKind.Truncated)
    }
    NEED_INPUT
  }

  private fun finishBlock() {
    this.state = if (this.last) ALL_DONE else AT_HEADER
  }

  // a block header, or false (nothing consumed) when the input ends in it
  private fun header(): bool throws CompressError {
    val savedPos = this.pos
    val savedBits = this.bits
    val savedCount = this.count
    val done = try this.parseHeader()
    if (!done) {
      this.pos = savedPos
      this.bits = savedBits
      this.count = savedCount
    }
    done
  }

  private fun parseHeader(): bool throws CompressError {
    if (!this.need(3)) return false
    this.last = this.eat(1) == 1
    val kind = this.eat(2)
    if (kind == 0) {
      this.eat(this.count % 8)  // to the byte boundary
      if (!this.need(32)) return false
      val len = this.eat(16)
      val inverse = this.eat(16)
      if (len != (inverse ^ 65535)) throw corrupt("a stored block's length does not match its complement")
      this.storedLeft = len
      this.state = IN_STORED
      return true
    }
    if (kind == 1) {
      loop (i in 0..<144) {
        this.lengths.set(i, 8)
      }
      loop (i in 144..<256) {
        this.lengths.set(i, 9)
      }
      loop (i in 256..<280) {
        this.lengths.set(i, 7)
      }
      loop (i in 280..<288) {
        this.lengths.set(i, 8)
      }
      loop (i in 0..<30) {
        this.lengths.set(288 + i, 5)
      }
      this.litBits = try buildTable(this.lengths, 0, 288, this.litTable)
      this.distBits = try buildTable(this.lengths, 288, 30, this.distTable)
      this.state = IN_CODES
      return true
    }
    if (kind == 3) throw corrupt("a block of the reserved type 3")
    if (!this.need(14)) return false
    val literals = this.eat(5) + 257
    val distances = this.eat(5) + 1
    val clens = this.eat(4) + 4
    if (literals > 286) throw corrupt("more than 286 literal/length codes")
    if (distances > 30) throw corrupt("more than 30 distance codes")
    loop (i in 0..<19) {
      this.lengths.set(i, 0)
    }
    loop (i in 0..<clens) {
      if (!this.need(3)) return false
      this.lengths.set(CLEN_ORDER.at(i) ?: 0, this.eat(3))
    }
    // the code-length code lives in the tail of `lengths`, past the others
    val shifted: MutableList<i64> = MutableList.repeat(0, 19)
    loop (i in 0..<19) {
      shifted.set(i, this.lengths.at(i) ?: 0)
    }
    val clenTable: MutableList<i64> = []
    val clenBits = try buildTable(shifted, 0, 19, clenTable)
    val total = literals + distances
    var i = 0
    var previous = 0
    loop (i < total) {
      val sym = try this.symbol(clenTable, clenBits)
      if (sym == -1) return false
      if (sym < 16) {
        this.lengths.set(i, sym)
        previous = sym
        i += 1
        continue
      }
      var repeat = 0
      var value = 0
      if (sym == 16) {
        if (i == 0) throw corrupt("a length repeat with nothing before it")
        if (!this.need(2)) return false
        repeat = 3 + this.eat(2)
        value = previous
      } else if (sym == 17) {
        if (!this.need(3)) return false
        repeat = 3 + this.eat(3)
      } else {
        if (!this.need(7)) return false
        repeat = 11 + this.eat(7)
      }
      if (i + repeat > total) throw corrupt("a length repeat runs past the code lengths")
      loop (_ in 0..<repeat) {
        this.lengths.set(i, value)
        i += 1
      }
      previous = value
    }
    if ((this.lengths.at(256) ?: 0) == 0) throw corrupt("a block without an end-of-block code")
    this.litBits = try buildTable(this.lengths, 0, literals, this.litTable)
    this.distBits = try buildTable(this.lengths, literals, distances, this.distTable)
    this.state = IN_CODES
    true
  }

  private fun stored(room: i64): i64 throws CompressError {
    loop (this.storedLeft > 0) {
      if (this.pending() >= room) return FULL
      // whole bytes still sitting in the bit buffer come first
      if (this.count >= 8) {
        this.push(this.eat(8).wrapU8())
        this.storedLeft -= 1
        continue
      }
      val available = this.input.len() - this.pos
      if (available == 0) return NEED_INPUT
      val n = available.min(this.storedLeft).min((room - this.pending()).max(1))
      loop (k in 0..<n) {
        // SAFETY: pos + k < len, as n <= available
        this.out.push(unsafe {
          this.input.atUnchecked(this.pos + k)
        })
      }
      this.pos += n
      this.storedLeft -= n
      try this.checkLimit()
    }
    this.finishBlock()
    CONTINUE
  }

  private fun push(b: u8) {
    this.out.push(b)
  }

  private fun checkLimit() throws CompressError {
    if (this.dropped + this.out.len() > this.limit) {
      throw CompressError(message: "decompressed data is larger than the ${this.limit} bytes allowed", kind: CompressKind.TooLarge)
    }
  }

  // The block's symbols. The hot loop: the bit buffer lives in locals, is
  // topped up to 48 bits at each symbol (so a whole length/distance pair is
  // there unless the input has ended), and a symbol is committed only when
  // all of it was available.
  private fun codes(room: i64): i64 throws CompressError {
    var bits = this.bits
    var count = this.count
    var pos = this.pos
    val input = this.input
    val end = input.len()
    val out = this.out
    val lit = this.litTable
    val dist = this.distTable
    val litMask = (1 << this.litBits) - 1
    val distMask = (1 << this.distBits) - 1
    val stopAt = this.delivered + room
    loop {
      if (out.len() >= stopAt) {
        this.bits = bits
        this.count = count
        this.pos = pos
        return FULL
      }
      loop (count <= 48 && pos < end) {
        // SAFETY: pos < end was just checked
        bits = bits | (unsafe {
          input.atUnchecked(pos)
        }.toI64() << count)
        pos += 1
        count += 8
      }
      // SAFETY: each index is masked to its table's size, and the tables hold
      // 2^width entries (buildTable)
      val entry = unsafe {
        lit.atUnchecked(bits & litMask)
      }
      val len = entry & 15
      var b = bits >> len
      var c = count - len
      if (len == 0 || c < 0) {
        this.bits = bits
        this.count = count
        this.pos = pos
        if (len == 0 && count >= this.litBits) throw corrupt("a code that no symbol has")
        return NEED_INPUT
      }
      val sym = entry >> 4
      if (sym < 256) {
        out.push(sym.wrapU8())
        bits = b
        count = c
        continue
      }
      if (sym == 256) {
        this.bits = b
        this.count = c
        this.pos = pos
        this.finishBlock()
        return CONTINUE
      }
      if (sym > 285) throw corrupt("a length symbol past 285")
      val slot = sym - 257
      // SAFETY: slot is 0..28 (sym was checked to be at most 285)
      val extra = unsafe {
        LENGTH_EXTRA.atUnchecked(slot)
      }
      if (c < extra) {
        this.bits = bits
        this.count = count
        this.pos = pos
        return NEED_INPUT
      }
      // SAFETY: as above
      val length = unsafe {
        LENGTH_BASE.atUnchecked(slot)
      } + (b & ((1 << extra) - 1))
      b = b >> extra
      c -= extra
      // SAFETY: masked index, as for lit
      val dentry = unsafe {
        dist.atUnchecked(b & distMask)
      }
      val dlen = dentry & 15
      if (dlen == 0 || c < dlen) {
        this.bits = bits
        this.count = count
        this.pos = pos
        if (dlen == 0 && c >= this.distBits) throw corrupt("a distance code that no symbol has")
        return NEED_INPUT
      }
      val dsym = dentry >> 4
      b = b >> dlen
      c -= dlen
      if (dsym > 29) throw corrupt("a distance symbol past 29")
      // SAFETY: dsym is 0..29
      val dextra = unsafe {
        DIST_EXTRA.atUnchecked(dsym)
      }
      if (c < dextra) {
        this.bits = bits
        this.count = count
        this.pos = pos
        return NEED_INPUT
      }
      // SAFETY: as above
      val distance = unsafe {
        DIST_BASE.atUnchecked(dsym)
      } + (b & ((1 << dextra) - 1))
      b = b >> dextra
      c -= dextra
      if (distance > out.len()) throw corrupt("a match reaches back before the start of the data")
      bits = b
      count = c
      var from = out.len() - distance
      loop (_ in 0..<length) {
        // SAFETY: from < len: distance >= 1, and every push grows the list
        out.push(unsafe {
          out.atUnchecked(from)
        })
        from += 1
      }
      if (this.dropped + out.len() > this.limit) {
        this.bits = bits
        this.count = count
        this.pos = pos
        try this.checkLimit()
      }
    }
  }
}

// `codes` and `stored` return this for "block finished, keep going"
const CONTINUE: i64 = 3
