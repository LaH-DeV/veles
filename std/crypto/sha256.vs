// SHA-256 (FIPS 180-4). 32-bit words, 64-byte blocks, a 32-byte digest.

/// The round constants: the first 32 bits of the fractional parts of the
/// cube roots of the first 64 primes.
val K256: List<u32> = [
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
val IV256: List<u32> = [
  0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
]

/// SHA-256, fed in pieces: `var h = crypto.Sha256.start()`, `h.update(…)`,
/// `h.finish()`. `crypto.sha256(data)` is the one-shot form.
public struct Sha256 {
  private state:      MutableList<u32>
  private buffer:     MutableList<u8>
  private scratch:    MutableList<u32>
  private var total:  i64 = 0
  private var result: Digest? = null

  implement Hasher {
    static fun start(): Sha256 = Sha256(
      state: IV256.toMutable(),
      buffer: [],
      scratch: MutableList<u32>.repeat(0, 64),
    )

    static fun algorithm(): string = "SHA-256"
    static fun blockSize(): i64 = 64
    static fun digestSize(): i64 = 32

    fun update(data: List<u8>) {
      if (self.result != null) panic("crypto.Sha256: update after finish")
      self.total = self.total + data.len()
      self.absorb(data)
    }

    fun finish(): Digest {
      val done = self.result
      if (done != null) return done
      self.absorb(padding(self.buffer.len(), 64, 8, self.total))
      val out: MutableList<u8> = []
      loop (w in self.state) {
        pushU32(out, w)
      }
      val d = Digest.of(out.toList())
      self.result = d
      d
    }
  }

  /// Feeds bytes through the compression function, keeping whatever does
  /// not fill a block. Full blocks in `data` are read where they lie, so a
  /// large `update` copies nothing.
  private fun absorb(data: List<u8>) {
    val n = data.len()
    var i = 0
    if (self.buffer.len() > 0) {
      loop (self.buffer.len() < 64 && i < n) {
        self.buffer.push(data.atOrPanic(i))
        i = i + 1
      }
      if (self.buffer.len() < 64) return
      compress256(self.state, self.buffer, 0, self.scratch)
      self.buffer.clear()
    }
    loop (i + 64 <= n) {
      compress256(self.state, data, i, self.scratch)
      i = i + 64
    }
    loop (i < n) {
      self.buffer.push(data.atOrPanic(i))
      i = i + 1
    }
  }
}

/// One 64-byte block into the eight-word state. `w` is scratch space the
/// caller owns, so a long message allocates nothing per block.
fun compress256(state: MutableList<u32>, block: List<u8>, at: i64, w: MutableList<u32>) {
  var i = 0
  loop (i < 16) {
    w.set(i, beU32(block, at + i * 4))
    i = i + 1
  }
  loop (i < 64) {
    val a = w.atOrPanic(i - 15)
    val b = w.atOrPanic(i - 2)
    val s0 = rotr32(a, 7) ^ rotr32(a, 18) ^ (a >> 3)
    val s1 = rotr32(b, 17) ^ rotr32(b, 19) ^ (b >> 10)
    w.set(i, w.atOrPanic(i - 16) +% s0 +% w.atOrPanic(i - 7) +% s1)
    i = i + 1
  }

  var a = state.atOrPanic(0)
  var b = state.atOrPanic(1)
  var c = state.atOrPanic(2)
  var d = state.atOrPanic(3)
  var e = state.atOrPanic(4)
  var f = state.atOrPanic(5)
  var g = state.atOrPanic(6)
  var h = state.atOrPanic(7)

  loop (r in 0..<64) {
    val sum1 = rotr32(e, 6) ^ rotr32(e, 11) ^ rotr32(e, 25)
    val choose = (e & f) ^ ((~e) & g)
    val t1 = h +% sum1 +% choose +% K256.atOrPanic(r) +% w.atOrPanic(r)
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

  state.set(0, state.atOrPanic(0) +% a)
  state.set(1, state.atOrPanic(1) +% b)
  state.set(2, state.atOrPanic(2) +% c)
  state.set(3, state.atOrPanic(3) +% d)
  state.set(4, state.atOrPanic(4) +% e)
  state.set(5, state.atOrPanic(5) +% f)
  state.set(6, state.atOrPanic(6) +% g)
  state.set(7, state.atOrPanic(7) +% h)
}
