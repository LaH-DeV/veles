// DEFLATE encoding (RFC 1951): LZ77 over hash chains, lazy matching at the
// higher levels, and per block the cheapest of stored, fixed and dynamic
// Huffman coding. The encoder is incremental — bytes are fed in pieces and
// the window slides — so a stream compresses in constant memory.

// how hard each level looks for a match: chain length, a length that is good
// enough to stop at, the length above which no lazy second look is taken
// (for the greedy levels: above which the matched text is not hashed), and
// whether the second look happens at all
struct Effort {
  chain:     i64
  nice:      i64
  lazyAbove: i64
  good:      i64
  lazy:      bool
}

fun effortOf(level: i64): Effort => when (level) {
  1    => Effort(chain: 4, nice: 8, lazyAbove: 4, good: 4, lazy: false)
  2    => Effort(chain: 8, nice: 16, lazyAbove: 5, good: 4, lazy: false)
  3    => Effort(chain: 32, nice: 32, lazyAbove: 6, good: 4, lazy: false)
  4    => Effort(chain: 16, nice: 16, lazyAbove: 4, good: 4, lazy: true)
  5    => Effort(chain: 32, nice: 32, lazyAbove: 16, good: 8, lazy: true)
  6    => Effort(chain: 128, nice: 128, lazyAbove: 16, good: 8, lazy: true)
  7    => Effort(chain: 256, nice: 128, lazyAbove: 32, good: 8, lazy: true)
  8    => Effort(chain: 1024, nice: 258, lazyAbove: 128, good: 32, lazy: true)
  else => Effort(chain: 4096, nice: 258, lazyAbove: 258, good: 32, lazy: true)
}

const HASH_SIZE: i64 = 32768
const HASH_MASK: i64 = 32767
const MAX_MATCH: i64 = 258
const MIN_MATCH: i64 = 3
// a match this far back that is only 3 long costs more than its literals
const TOO_FAR: i64 = 4096
// bytes that must be visible beyond a position before it is encoded, unless
// the stream is ending
const LOOKAHEAD: i64 = 262
const BLOCK_TOKENS: i64 = 32768
const NONE: i64 = -1000000000000

fun buildLengthSymbols(): List<i64> {
  val table: MutableList<i64> = MutableList.repeat(0, 259)
  loop (slot in 0..<29) {
    val base = LENGTH_BASE.at(slot)
    val span = 1 << (LENGTH_EXTRA.at(slot))
    loop (n in base..<(base + span)) {
      if (n <= 258) table.set(n, 257 + slot)
    }
  }
  table.set(258, 285)  // 258 has its own symbol, not the last slot of 227's
  table.toList()
}

fun buildDistSymbols(): List<i64> {
  // index d - 1 for d <= 256; for larger d, 256 + ((d - 1) >> 7)
  val table: MutableList<i64> = MutableList.repeat(0, 512)
  loop (slot in 0..<30) {
    val base = DIST_BASE.at(slot)
    val span = 1 << (DIST_EXTRA.at(slot))
    loop (d in base..<(base + span)) {
      if (d <= 256) {
        table.set(d - 1, slot)
      } else if (d <= 32768 && (d - 1) % 128 == 0) {
        table.set(256 + ((d - 1) >> 7), slot)
      }
    }
  }
  // the slots of d > 256 each cover whole multiples of 128, so one entry per
  // multiple is enough; fill the rest of each run
  loop (k in 0..<256) {
    val d = (k << 7) + 1
    if (d > 256) {
      var slot = 0
      loop (s in 0..<30) {
        if (DIST_BASE.at(s) <= d) slot = s
      }
      table.set(256 + k, slot)
    }
  }
  table.toList()
}

val lengthSymbols: List<i64> = buildLengthSymbols()
val distSymbols: List<i64> = buildDistSymbols()

fun distSymbol(d: i64): i64 {
  // SAFETY: d is 1..32768, so both indexes are inside the 512 entries
  unsafe {
    if (d <= 256) distSymbols.atUnchecked(d - 1) else distSymbols.atUnchecked(256 + ((d - 1) >> 7))
  }
}

// ---------------------------------------------------------------------------
// Huffman code construction

// code lengths for `freqs` (symbols 0..<n), none longer than `maxBits`;
// a symbol with frequency 0 gets length 0
fun huffmanLengths(freqs: MutableList<i64>, n: i64, maxBits: i64): MutableList<i64> {
  val lens: MutableList<i64> = MutableList.repeat(0, n)
  val keyed: MutableList<i64> = []
  loop (sym in 0..<n) {
    val f = freqs.at(sym) ?: 0
    if (f > 0) keyed.push((f << 10) | sym)
  }
  val order = keyed.sorted()  // ascending frequency
  val m = order.len()
  if (m == 0) return lens
  if (m == 1) {
    lens.set(order.at(0) & 1023, 1)
    return lens
  }
  // the two-queue construction: leaves in order, internal nodes made in
  // order of weight, so each queue stays sorted
  val weight: MutableList<i64> = MutableList.repeat(0, 2 * m)
  val parent: MutableList<i64> = MutableList.repeat(0, 2 * m)
  loop (i in 0..<m) {
    weight.set(i, order.at(i) >> 10)
  }
  var leaf = 0
  var made = m
  var open = m  // first internal node not yet merged
  loop (_ in 0..<(m - 1)) {
    var pair = 0
    loop (_ in 0..<2) {
      var pick = 0
      if (leaf < m && (open >= made || (weight.at(leaf) ?: 0) <= (weight.at(open) ?: 0))) {
        pick = leaf
        leaf += 1
      } else {
        pick = open
        open += 1
      }
      parent.set(pick, made)
      pair += weight.at(pick) ?: 0
    }
    weight.set(made, pair)
    made += 1
  }
  // depths, from the root down (a parent always comes after its children)
  val depth: MutableList<i64> = MutableList.repeat(0, 2 * m)
  var node = made - 2
  loop (node >= 0) {
    depth.set(node, (depth.at(parent.at(node) ?: 0) ?: 0) + 1)
    node -= 1
  }
  // how many leaves per length, the long ones folded into maxBits
  val counts: MutableList<i64> = MutableList.repeat(0, maxBits + 1)
  loop (i in 0..<m) {
    val d = (depth.at(i) ?: 1).min(maxBits)
    counts.set(d, (counts.at(d) ?: 0) + 1)
  }
  // folding made the code over-full; lengthen shorter codes until it is not
  var kraft = 0
  loop (len in 1..maxBits) {
    kraft += (counts.at(len) ?: 0) << (maxBits - len)
  }
  loop (kraft > (1 << maxBits)) {
    counts.set(maxBits, (counts.at(maxBits) ?: 0) - 1)
    var len = maxBits - 1
    loop (len >= 1) {
      if ((counts.at(len) ?: 0) > 0) {
        counts.set(len, (counts.at(len) ?: 0) - 1)
        counts.set(len + 1, (counts.at(len + 1) ?: 0) + 2)
        break
      }
      len -= 1
    }
    kraft -= 1
  }
  // the rarest symbols take the longest codes
  var next = 0
  loop (j in 0..<maxBits) {
    val len = maxBits - j
    loop (_ in 0..<(counts.at(len) ?: 0)) {
      lens.set((order.at(next) ?: 0) & 1023, len)
      next += 1
    }
  }
  lens
}

// the canonical codes of `lens`, each with its bits reversed (the stream
// carries a code's top bit first, into the low end of the byte)
fun canonicalCodes(lens: MutableList<i64>, n: i64): MutableList<i64> {
  val counts: MutableList<i64> = MutableList.repeat(0, 17)
  loop (sym in 0..<n) {
    val len = lens.at(sym) ?: 0
    counts.set(len, (counts.at(len) ?: 0) + 1)
  }
  counts.set(0, 0)
  val next: MutableList<i64> = MutableList.repeat(0, 17)
  var code = 0
  loop (len in 1..15) {
    code = (code + (counts.at(len - 1) ?: 0)) << 1
    next.set(len, code)
  }
  val codes: MutableList<i64> = MutableList.repeat(0, n)
  loop (sym in 0..<n) {
    val len = lens.at(sym) ?: 0
    if (len == 0) continue
    val canonical = next.at(len) ?: 0
    next.set(len, canonical + 1)
    var reversed = 0
    var rest = canonical
    loop (_ in 0..<len) {
      reversed = (reversed << 1) | (rest & 1)
      rest = rest >> 1
    }
    codes.set(sym, reversed)
  }
  codes
}

fun fixedLitLengths(): MutableList<i64> {
  val lens: MutableList<i64> = MutableList.repeat(8, 288)
  loop (i in 144..<256) {
    lens.set(i, 9)
  }
  loop (i in 256..<280) {
    lens.set(i, 7)
  }
  lens
}

val fixedLitLens: List<i64> = fixedLitLengths().toList()
val fixedLitCodes: List<i64> = canonicalCodes(fixedLitLengths(), 288).toList()
val fixedDistCodes: List<i64> = canonicalCodes(MutableList.repeat(5, 30), 30).toList()

// ---------------------------------------------------------------------------

/// Encodes a DEFLATE stream a piece at a time.
struct Deflater {
  private var level:  i64
  private var effort: Effort

  // input not yet forgotten: buf[0] is absolute position `base`
  private var buf:         MutableList<u8> = []
  private var base:        i64 = 0
  private var pos:         i64 = 0
  private var head:        MutableList<i64> = MutableList.repeat(NONE, HASH_SIZE)
  private var prev:        MutableList<i64> = MutableList.repeat(NONE, HASH_SIZE)
  private var havePending: bool = false
  private var pendingLen:  i64 = 0
  private var pendingDist: i64 = 0

  // the block being built
  private var tokens:     MutableList<i64> = []
  private var blockStart: i64 = 0
  private var covered:    i64 = 0

  // output
  private var out:      MutableList<u8> = []
  private var acc:      i64 = 0
  private var nbits:    i64 = 0
  private var finished: bool = false

  init(level: i64) {
    this.level = level
    this.effort = effortOf(level)
  }

  /// More input.
  fun feed(data: List<u8>) {
    // forget what is further back than a match can reach
    val keepFrom = this.pos - 32768 - 1
    if (keepFrom - this.base > 65536) {
      val drop = keepFrom - this.base
      this.buf = this.buf.drop(drop).toMutable()
      this.base += drop
    }
    this.buf.addAll(data)
  }

  /// The bytes produced so far, handed over; whole bytes only, until `finish`.
  fun take(): List<u8> {
    val bytes = this.out.toList()
    this.out.clear()
    bytes
  }

  /// Encodes what has been fed. Until `finish`, the last few bytes wait for
  /// more input to match against.
  fun run() {
    if (this.level == 0) {
      this.storeAll()
      return
    }
    val end = this.base + this.buf.len()
    this.encode(end - LOOKAHEAD)
  }

  /// Ends the stream: everything fed is encoded and the final block written.
  fun finish() {
    if (this.finished) return
    if (this.level == 0) {
      this.storeAll()
      this.storedBlock(this.buf.len() - (this.pos - this.base), true)
    } else {
      this.encode(this.base + this.buf.len())
      this.flush(true)
    }
    this.align()
    this.finished = true
  }

  /// Ends the current block and pads to a byte boundary, so a reader can
  /// take everything so far (a sync flush: an empty stored block).
  fun sync() {
    if (this.level == 0) {
      this.storeAll()
    } else {
      this.encode(this.base + this.buf.len())
      this.flush(false)
    }
    this.putBits(0, 3)
    this.align()
    this.putBits(0, 16)
    this.putBits(65535, 16)
  }

  private fun putBits(value: i64, n: i64) {
    this.acc = this.acc | (value << this.nbits)
    this.nbits += n
    loop (this.nbits >= 8) {
      this.out.push((this.acc & 255).wrapU8())
      this.acc = this.acc >> 8
      this.nbits -= 8
    }
  }

  private fun align() {
    if (this.nbits > 0) this.putBits(0, 8 - this.nbits)
  }

  private fun hashAt(p: i64): i64 {
    val at = p - this.base
    // SAFETY: callers hash only positions with two more bytes after them
    val v = unsafe {
      this.buf.atUnchecked(at).toI64() | (this.buf.atUnchecked(at + 1).toI64() << 8) | (this.buf.atUnchecked(at + 2).toI64() << 16)
    }
    ((v * 2654435761) >> 15) & HASH_MASK
  }

  // records position p, returning the one that held its hash before
  private fun insert(p: i64): i64 {
    val h = this.hashAt(p)
    // SAFETY: h is masked to the table size, and so is the ring index
    unsafe {
      val candidate = this.head.atUnchecked(h)
      this.prev.setUnchecked(p & HASH_MASK, candidate)
      this.head.setUnchecked(h, p)
      candidate
    }
  }

  // the best match for p among the chain starting at `candidate`, longer
  // than `atLeast`: length << 16 | distance, or 0
  private fun longest(p: i64, first: i64, atLeast: i64): i64 {
    val avail = this.base + this.buf.len() - p
    val limit = avail.min(MAX_MATCH)
    if (limit < MIN_MATCH) return 0
    var best = atLeast
    var bestDist = 0
    var chain = this.effort.chain
    if (atLeast >= this.effort.good) chain = chain >> 2
    var candidate = first
    val at = p - this.base
    loop (chain > 0 && candidate >= this.base && p - candidate <= 32768 && candidate < p) {
      val c = candidate - this.base
      // SAFETY: c + best < at + best < len (best < limit <= avail), and every
      // compared index is below `limit` bytes from c and from at
      unsafe {
        if (best >= limit || this.buf.atUnchecked(c + best) == this.buf.atUnchecked(at + best)) {
          var n = 0
          loop (n < limit && this.buf.atUnchecked(c + n) == this.buf.atUnchecked(at + n)) {
            n += 1
          }
          if (n > best) {
            best = n
            bestDist = p - candidate
            if (n >= this.effort.nice || n >= limit) break
          }
        }
      }
      // SAFETY: the ring index is masked to the table size
      val next = unsafe {
        this.prev.atUnchecked(candidate & HASH_MASK)
      }
      if (next >= candidate) break  // the ring entry was overwritten
      candidate = next
      chain -= 1
    }
    if (bestDist == 0) return 0
    if (best == MIN_MATCH && bestDist > TOO_FAR) return 0
    (best << 16) | bestDist
  }

  private fun literal(at: i64) {
    // SAFETY: at is a position inside the buffer
    this.tokens.push(unsafe {
      this.buf.atUnchecked(at - this.base).toI64()
    })
    this.covered += 1
    if (this.tokens.len() >= BLOCK_TOKENS) this.flush(false)
  }

  private fun emitMatch(len: i64, dist: i64) {
    this.tokens.push((len << 16) | dist)
    this.covered += len
    if (this.tokens.len() >= BLOCK_TOKENS) this.flush(false)
  }

  // encodes positions up to (not including) `stop`; with a pending lazy
  // match, the last position is settled too
  private fun encode(stop: i64) {
    val end = this.base + this.buf.len()
    val lazy = this.effort.lazy
    loop (this.pos < stop || (this.havePending && stop >= end)) {
      val here = this.pos
      var found = 0
      if (here + 2 < end && here < stop) {
        val candidate = this.insert(here)
        found = this.longest(here, candidate, if (this.havePending) this.pendingLen else 2)
      }
      val len = found >> 16
      val dist = found & 65535
      if (this.havePending) {
        if (len > this.pendingLen) {
          // the match one byte later is longer: the byte before it is a literal
          this.literal(here - 1)
          this.pendingLen = len
          this.pendingDist = dist
          this.pos = here + 1
        } else {
          this.emitMatch(this.pendingLen, this.pendingDist)
          // the match covers here - 1 ..< here - 1 + pendingLen; here is hashed
          val until = here - 1 + this.pendingLen
          var p = here + 1
          loop (p < until) {
            if (p + 2 < end) this.insert(p)
            p += 1
          }
          this.pos = until
          this.havePending = false
        }
      } else if (len >= MIN_MATCH) {
        if (lazy && len < this.effort.lazyAbove) {
          this.havePending = true
          this.pendingLen = len
          this.pendingDist = dist
          this.pos = here + 1
        } else {
          this.emitMatch(len, dist)
          if (len <= this.effort.lazyAbove || lazy) {
            var p = here + 1
            loop (p < here + len) {
              if (p + 2 < end) this.insert(p)
              p += 1
            }
          }
          this.pos = here + len
        }
      } else {
        this.literal(here)
        this.pos = here + 1
      }
    }
  }

  // level 0: whole blocks of 65535 bytes as they fill; the rest waits for finish
  private fun storeAll() {
    loop (this.buf.len() - (this.pos - this.base) >= 65535) {
      this.storedBlock(65535, false)
    }
  }

  // a stored block of the next n bytes from pos
  private fun storedBlock(n: i64, final: bool) {
    this.putBits(if (final) 1 else 0, 1)
    this.putBits(0, 2)
    this.align()
    this.putBits(n, 16)
    this.putBits(n ^ 65535, 16)
    loop (k in 0..<n) {
      // SAFETY: the callers pass n no larger than what is left in the buffer
      this.out.push(unsafe {
        this.buf.atUnchecked(this.pos - this.base + k)
      })
    }
    this.pos += n
  }

  // writes the tokens gathered so far as one block (an empty final block when
  // there are none and `final`)
  private fun flush(final: bool) {
    if (this.tokens.isEmpty() && !final) return
    val litFreq: MutableList<i64> = MutableList.repeat(0, 286)
    val distFreq: MutableList<i64> = MutableList.repeat(0, 30)
    var extraBits = 0
    loop (token in this.tokens) {
      if (token < 256) {
        litFreq.set(token, (litFreq.at(token) ?: 0) + 1)
      } else {
        val len = token >> 16
        val dist = token & 65535
        val ls = lengthSymbols.at(len) ?: 257
        val ds = distSymbol(dist)
        litFreq.set(ls, (litFreq.at(ls) ?: 0) + 1)
        distFreq.set(ds, (distFreq.at(ds) ?: 0) + 1)
        extraBits += (LENGTH_EXTRA.at(ls - 257) ?: 0) + (DIST_EXTRA.at(ds) ?: 0)
      }
    }
    litFreq.set(256, 1)
    val litLens = huffmanLengths(litFreq, 286, 15)
    val distLens = huffmanLengths(distFreq, 30, 15)
    var distUsed = false
    loop (d in distLens) {
      if (d > 0) distUsed = true
    }
    if (!distUsed) distLens.set(0, 1)

    // the cost of each way to write it
    var litBits = 0
    loop (sym in 0..<286) {
      litBits += (litFreq.at(sym) ?: 0) * (litLens.at(sym) ?: 0)
    }
    var distBits = 0
    loop (sym in 0..<30) {
      distBits += (distFreq.at(sym) ?: 0) * (distLens.at(sym) ?: 0)
    }
    var fixedBits = 3 + extraBits
    loop (sym in 0..<286) {
      fixedBits += (litFreq.at(sym) ?: 0) * (fixedLitLens.at(sym) ?: 8)
    }
    loop (sym in 0..<30) {
      fixedBits += (distFreq.at(sym) ?: 0) * 5
    }
    // the header of a dynamic block: lengths run-length coded, themselves Huffman coded
    var hlit = 286
    loop (hlit > 257 && (litLens.at(hlit - 1) ?: 0) == 0) {
      hlit -= 1
    }
    var hdist = 30
    loop (hdist > 1 && (distLens.at(hdist - 1) ?: 0) == 0) {
      hdist -= 1
    }
    val sequence: MutableList<i64> = []
    loop (i in 0..<hlit) {
      sequence.push(litLens.at(i) ?: 0)
    }
    loop (i in 0..<hdist) {
      sequence.push(distLens.at(i) ?: 0)
    }
    // symbols of the code-length alphabet, with the extra-bit values after 16-18
    val coded: MutableList<i64> = []
    var i = 0
    loop (i < sequence.len()) {
      val value = sequence.at(i) ?: 0
      var run = 1
      loop (i + run < sequence.len() && (sequence.at(i + run) ?: -1) == value) {
        run += 1
      }
      if (value == 0 && run >= 3) {
        val take = run.min(138)
        if (take >= 11) {
          coded.push(18 | ((take - 11) << 8))
        } else {
          coded.push(17 | ((take - 3) << 8))
        }
        i += take
      } else if (value != 0 && run >= 4) {
        coded.push(value)
        var left = run - 1
        loop (left >= 3) {
          val take = left.min(6)
          coded.push(16 | ((take - 3) << 8))
          left -= take
        }
        loop (left > 0) {
          coded.push(value)
          left -= 1
        }
        i += run
      } else {
        coded.push(value)
        i += 1
      }
    }
    val clenFreq: MutableList<i64> = MutableList.repeat(0, 19)
    loop (c in coded) {
      val sym = c & 255
      clenFreq.set(sym, (clenFreq.at(sym) ?: 0) + 1)
    }
    val clenLens = huffmanLengths(clenFreq, 19, 7)
    var hclen = 19
    loop (hclen > 4 && (clenLens.at(CLEN_ORDER.at(hclen - 1) ?: 0) ?: 0) == 0) {
      hclen -= 1
    }
    var headerBits = 3 + 14 + 3 * hclen
    loop (c in coded) {
      val sym = c & 255
      headerBits += (clenLens.at(sym) ?: 0) + (if (sym == 16) 2 else if (sym == 17) 3 else if (sym == 18) 7 else 0)
    }
    val dynamicBits = headerBits + litBits + distBits + extraBits

    // a stored block, when the bytes are still here and it is smaller
    val rawLen = this.covered - this.blockStart
    val haveRaw = this.blockStart >= this.base
    val storedBits = rawLen * 8 + 40 * ((rawLen + 65534) / 65535) + 7
    if (haveRaw && rawLen > 0 && storedBits < dynamicBits.min(fixedBits)) {
      this.writeStoredRange(this.blockStart, rawLen, final)
    } else if (fixedBits <= dynamicBits) {
      this.putBits(if (final) 1 else 0, 1)
      this.putBits(1, 2)
      this.writeTokens(fixedLitCodes.toMutable(), fixedLitLens.toMutable(), fixedDistCodes.toMutable(), MutableList.repeat(5, 30))
    } else {
      this.putBits(if (final) 1 else 0, 1)
      this.putBits(2, 2)
      this.putBits(hlit - 257, 5)
      this.putBits(hdist - 1, 5)
      this.putBits(hclen - 4, 4)
      loop (k in 0..<hclen) {
        this.putBits(clenLens.at(CLEN_ORDER.at(k) ?: 0) ?: 0, 3)
      }
      val clenCodes = canonicalCodes(clenLens, 19)
      loop (c in coded) {
        val sym = c & 255
        this.putBits(clenCodes.at(sym) ?: 0, clenLens.at(sym) ?: 0)
        if (sym == 16) this.putBits(c >> 8, 2)
        if (sym == 17) this.putBits(c >> 8, 3)
        if (sym == 18) this.putBits(c >> 8, 7)
      }
      this.writeTokens(canonicalCodes(litLens, 286), litLens, canonicalCodes(distLens, 30), distLens)
    }
    this.tokens.clear()
    this.blockStart = this.covered
  }

  private fun writeStoredRange(start: i64, length: i64, final: bool) {
    var at = start
    var left = length
    loop (left > 0) {
      val n = left.min(65535)
      this.putBits(if (final && n == left) 1 else 0, 1)
      this.putBits(0, 2)
      this.align()
      this.putBits(n, 16)
      this.putBits(n ^ 65535, 16)
      loop (k in 0..<n) {
        // SAFETY: the range was checked to lie inside the buffer by the caller
        this.out.push(unsafe {
          this.buf.atUnchecked(at - this.base + k)
        })
      }
      at += n
      left -= n
    }
  }

  private fun writeTokens(litCodes: MutableList<i64>, litLens: MutableList<i64>, distCodes: MutableList<i64>, distLens: MutableList<i64>) {
    loop (token in this.tokens) {
      if (token < 256) {
        this.putBits(litCodes.at(token) ?: 0, litLens.at(token) ?: 0)
        continue
      }
      val len = token >> 16
      val dist = token & 65535
      val ls = lengthSymbols.at(len) ?: 257
      this.putBits(litCodes.at(ls) ?: 0, litLens.at(ls) ?: 0)
      val lextra = LENGTH_EXTRA.at(ls - 257) ?: 0
      if (lextra > 0) this.putBits(len - (LENGTH_BASE.at(ls - 257) ?: 3), lextra)
      val ds = distSymbol(dist)
      this.putBits(distCodes.at(ds) ?: 0, distLens.at(ds) ?: 0)
      val dextra = DIST_EXTRA.at(ds) ?: 0
      if (dextra > 0) this.putBits(dist - (DIST_BASE.at(ds) ?: 1), dextra)
    }
    this.putBits(litCodes.at(256) ?: 0, litLens.at(256) ?: 0)
  }
}
