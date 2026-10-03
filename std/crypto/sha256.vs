// SHA-256 (FIPS 180-4). 32-bit words, 64-byte blocks, a 32-byte digest.

/// The round constants: the first 32 bits of the fractional parts of the
/// cube roots of the first 64 primes.
const K256: Array<u32, 64> = [
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
]

/// The initial state: the first 32 bits of the fractional parts of the
/// square roots of the first 8 primes.
const IV256: Array<u32, 8> = [
  0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
]

/// SHA-256, fed in pieces: `var h = crypto.Sha256.start()`, `h.update(…)`,
/// `h.finish()`. `crypto.sha256(data)` is the one-shot form.
public struct Sha256 {
  private var state:  Array<u32, 8> = IV256
  private var block:  Array<u8, 64> = Array.make(0)
  private var filled: i64 = 0
  private var total:  i64 = 0
  private var result: Digest? = null

  implement Hasher {
    static fun start(): Sha256 = Sha256()

    static fun algorithm(): string = "SHA-256"
    static fun blockSize(): i64 = 64
    static fun digestSize(): i64 = 32

    fun update(data: List<u8>) {
      if (this.result != null) panic("crypto.Sha256: update after finish")
      this.total = this.total + data.len()
      this.absorb(data)
    }

    fun finish(): Digest {
      val done = this.result
      if (done != null) return done
      this.absorb(padding(this.filled, 64, 8, this.total))
      val out: MutableList<u8> = []
      loop (w in this.state) {
        pushU32(out, w)
      }
      val d = Digest.of(out.toList())
      this.result = d
      d
    }
  }

  /// Feeds bytes through the compression function, keeping whatever does
  /// not fill a block. Full blocks in `data` are read where they lie, so a
  /// large `update` copies nothing.
  private fun absorb(data: List<u8>) {
    val n = data.len()
    var i = 0
    if (this.filled > 0) {
      loop (this.filled < 64 && i < n) {
        this.block.set(this.filled, data.at(i))
        this.filled = this.filled + 1
        i = i + 1
      }
      if (this.filled < 64) return
      this.compressBuffered()
      this.filled = 0
    }
    loop (i + 64 <= n) {
      this.compressAt(data, i)
      i = i + 64
    }
    loop (i < n) {
      this.block.set(this.filled, data.at(i))
      this.filled = this.filled + 1
      i = i + 1
    }
  }

  /// The block at `at` in `data` into the state.
  private fun compressAt(data: List<u8>, at: i64) {
    var w: Array<u32, 64> = Array.make(0)
    loop (i in 0..<16) {
      w.set(i, beU32(data, at + i * 4))
    }
    this.rounds(w)
  }

  /// The buffered block into the state.
  private fun compressBuffered() {
    var w: Array<u32, 64> = Array.make(0)
    loop (i in 0..<16) {
      w.set(i, wordOf(this.block, i * 4))
    }
    this.rounds(w)
  }

  /// The message schedule from the block's sixteen words, then the sixty-four
  /// rounds over the eight-word state.
  private fun rounds(first: Array<u32, 64>) {
    var w = first
    // SAFETY: i runs over 16..<64, so every index below is in 0..<64
    unsafe {
      loop (i in 16..<64) {
        val a = w.atUnchecked(i - 15)
        val b = w.atUnchecked(i - 2)
        val s0 = rotr32(a, 7) ^ rotr32(a, 18) ^ (a >> 3)
        val s1 = rotr32(b, 17) ^ rotr32(b, 19) ^ (b >> 10)
        w.setUnchecked(i, w.atUnchecked(i - 16) +% s0 +% w.atUnchecked(i - 7) +% s1)
      }
    }

    val a0 = this.state.at(0)
    val b0 = this.state.at(1)
    val c0 = this.state.at(2)
    val d0 = this.state.at(3)
    val e0 = this.state.at(4)
    val f0 = this.state.at(5)
    val g0 = this.state.at(6)
    val h0 = this.state.at(7)
    var a = a0
    var b = b0
    var c = c0
    var d = d0
    var e = e0
    var f = f0
    var g = g0
    var h = h0

    loop (r in 0..<64) {
      val sum1 = rotr32(e, 6) ^ rotr32(e, 11) ^ rotr32(e, 25)
      val choose = (e & f) ^ ((~e) & g)
      val t1 = h +% sum1 +% choose +% K256.at(r) +% w.at(r)
      val sum0 = rotr32(a, 2) ^ rotr32(a, 13) ^ rotr32(a, 22)
      val major = (a & b) ^ (a & c) ^ (b & c)
      val t2 = sum0 +% major
      h = g
      g = f
      f = e
      e = d +% t1
      d = c
      c = b
      b = a
      a = t1 +% t2
    }

    this.state.set(0, a0 +% a)
    this.state.set(1, b0 +% b)
    this.state.set(2, c0 +% c)
    this.state.set(3, d0 +% d)
    this.state.set(4, e0 +% e)
    this.state.set(5, f0 +% f)
    this.state.set(6, g0 +% g)
    this.state.set(7, h0 +% h)
  }
}

/// The big-endian word at `at` in a block held in an array. Called only
/// where the block holds all four bytes: offsets 0, 4, … 60.
fun wordOf(block: Array<u8, 64>, at: i64): u32 =
  ((block.at(at) ?: panic("sha256: a word is read inside the block")).toU32() << 24) |
  ((block.at(at + 1) ?: panic("sha256: a word is read inside the block")).toU32() << 16) |
  ((block.at(at + 2) ?: panic("sha256: a word is read inside the block")).toU32() << 8) |
  (block.at(at + 3) ?: panic("sha256: a word is read inside the block")).toU32()
