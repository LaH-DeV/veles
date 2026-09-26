// SHA-512 (FIPS 180-4). 64-bit words, 128-byte blocks, a 64-byte digest;
// on a 64-bit machine it hashes a long message faster than SHA-256.

/// The round constants: the first 64 bits of the fractional parts of the
/// cube roots of the first 80 primes.
val K512: List<u64> = [
  0x428a2f98d728ae22, 0x7137449123ef65cd, 0xb5c0fbcfec4d3b2f, 0xe9b5dba58189dbbc,
  0x3956c25bf348b538, 0x59f111f1b605d019, 0x923f82a4af194f9b, 0xab1c5ed5da6d8118,
  0xd807aa98a3030242, 0x12835b0145706fbe, 0x243185be4ee4b28c, 0x550c7dc3d5ffb4e2,
  0x72be5d74f27b896f, 0x80deb1fe3b1696b1, 0x9bdc06a725c71235, 0xc19bf174cf692694,
  0xe49b69c19ef14ad2, 0xefbe4786384f25e3, 0x0fc19dc68b8cd5b5, 0x240ca1cc77ac9c65,
  0x2de92c6f592b0275, 0x4a7484aa6ea6e483, 0x5cb0a9dcbd41fbd4, 0x76f988da831153b5,
  0x983e5152ee66dfab, 0xa831c66d2db43210, 0xb00327c898fb213f, 0xbf597fc7beef0ee4,
  0xc6e00bf33da88fc2, 0xd5a79147930aa725, 0x06ca6351e003826f, 0x142929670a0e6e70,
  0x27b70a8546d22ffc, 0x2e1b21385c26c926, 0x4d2c6dfc5ac42aed, 0x53380d139d95b3df,
  0x650a73548baf63de, 0x766a0abb3c77b2a8, 0x81c2c92e47edaee6, 0x92722c851482353b,
  0xa2bfe8a14cf10364, 0xa81a664bbc423001, 0xc24b8b70d0f89791, 0xc76c51a30654be30,
  0xd192e819d6ef5218, 0xd69906245565a910, 0xf40e35855771202a, 0x106aa07032bbd1b8,
  0x19a4c116b8d2d0c8, 0x1e376c085141ab53, 0x2748774cdf8eeb99, 0x34b0bcb5e19b48a8,
  0x391c0cb3c5c95a63, 0x4ed8aa4ae3418acb, 0x5b9cca4f7763e373, 0x682e6ff3d6b2b8a3,
  0x748f82ee5defb2fc, 0x78a5636f43172f60, 0x84c87814a1f0ab72, 0x8cc702081a6439ec,
  0x90befffa23631e28, 0xa4506cebde82bde9, 0xbef9a3f7b2c67915, 0xc67178f2e372532b,
  0xca273eceea26619c, 0xd186b8c721c0c207, 0xeada7dd6cde0eb1e, 0xf57d4f7fee6ed178,
  0x06f067aa72176fba, 0x0a637dc5a2c898a6, 0x113f9804bef90dae, 0x1b710b35131c471b,
  0x28db77f523047d84, 0x32caab7b40c72493, 0x3c9ebe0a15c9bebc, 0x431d67c49c100d4c,
  0x4cc5d4becb3e42b6, 0x597f299cfc657e2a, 0x5fcb6fab3ad6faec, 0x6c44198c4a475817,
]

/// The initial state: the first 64 bits of the fractional parts of the
/// square roots of the first 8 primes.
val IV512: List<u64> = [
  0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
  0x510e527fade682d1, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
]

/// SHA-512, fed in pieces: `var h = crypto.Sha512.start()`, `h.update(…)`,
/// `h.finish()`. `crypto.sha512(data)` is the one-shot form.
public struct Sha512 {
  private state:      MutableList<u64>
  private buffer:     MutableList<u8>
  private scratch:    MutableList<u64>
  private var total:  i64 = 0
  private var result: Digest? = null

  implement Hasher {
    static fun start(): Sha512 = Sha512(
      state: IV512.toMutable(),
      buffer: [],
      scratch: MutableList<u64>.repeat(0, 80),
    )

    static fun algorithm(): string = "SHA-512"
    static fun blockSize(): i64 = 128
    static fun digestSize(): i64 = 64

    fun update(data: List<u8>) {
      if (self.result != null) panic("crypto.Sha512: update after finish")
      self.total = self.total + data.len()
      self.absorb(data)
    }

    fun finish(): Digest {
      val done = self.result
      if (done != null) return done
      self.absorb(padding(self.buffer.len(), 128, 16, self.total))
      val out: MutableList<u8> = []
      loop (w in self.state) {
        pushU64(out, w)
      }
      val d = Digest.of(out.toList())
      self.result = d
      d
    }
  }

  private fun absorb(data: List<u8>) {
    val n = data.len()
    var i = 0
    if (self.buffer.len() > 0) {
      loop (self.buffer.len() < 128 && i < n) {
        self.buffer.push(data.at(i))
        i = i + 1
      }
      if (self.buffer.len() < 128) return
      compress512(self.state, self.buffer.toList(), 0, self.scratch)
      self.buffer.clear()
    }
    loop (i + 128 <= n) {
      compress512(self.state, data, i, self.scratch)
      i = i + 128
    }
    loop (i < n) {
      self.buffer.push(data.at(i))
      i = i + 1
    }
  }
}

/// One 128-byte block into the eight-word state; `w` is the caller's
/// scratch space.
fun compress512(state: MutableList<u64>, block: List<u8>, at: i64, w: MutableList<u64>) {
  var i = 0
  loop (i < 16) {
    w.set(i, beU64(block, at + i * 8))
    i = i + 1
  }
  loop (i < 80) {
    val a = schedule512(w, i - 15)
    val b = schedule512(w, i - 2)
    val s0 = rotr64(a, 1) ^ rotr64(a, 8) ^ (a >> 7)
    val s1 = rotr64(b, 19) ^ rotr64(b, 61) ^ (b >> 6)
    w.set(i, schedule512(w, i - 16) +% s0 +% schedule512(w, i - 7) +% s1)
    i = i + 1
  }

  val [a0, b0, c0, d0, e0, f0, g0, h0] = state else panic("sha512: the state is 8 words")
  var a = a0
  var b = b0
  var c = c0
  var d = d0
  var e = e0
  var f = f0
  var g = g0
  var h = h0

  loop (r in 0..<80) {
    val sum1 = rotr64(e, 14) ^ rotr64(e, 18) ^ rotr64(e, 41)
    val choose = (e & f) ^ ((~e) & g)
    val t1 = h +% sum1 +% choose +% (K512.at(r) ?: panic("sha512: one constant per round")) +% schedule512(w, r)
    val sum0 = rotr64(a, 28) ^ rotr64(a, 34) ^ rotr64(a, 39)
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

  state.set(0, a0 +% a)
  state.set(1, b0 +% b)
  state.set(2, c0 +% c)
  state.set(3, d0 +% d)
  state.set(4, e0 +% e)
  state.set(5, f0 +% f)
  state.set(6, g0 +% g)
  state.set(7, h0 +% h)
}

/// The initial state of SHA-384: the fractional parts of the square roots
/// of the 9th to 16th primes.
val IV384: List<u64> = [
  0xcbbb9d5dc1059ed8, 0x629a292a367cd507, 0x9159015a3070dd17, 0x152fecd8f70e5939,
  0x67332667ffc00b31, 0x8eb44a8768581511, 0xdb0c2e0d64f98fa7, 0x47b5481dbefa4fa4,
]

/// SHA-384: SHA-512 with a different starting state, cut to 48 bytes. It
/// exists because truncating SHA-512 by itself would let a 384-bit digest
/// be extended into the 512-bit one; a different IV makes the two
/// unrelated. `crypto.sha384(data)` is the one-shot form.
public struct Sha384 {
  private state:      MutableList<u64>
  private buffer:     MutableList<u8>
  private scratch:    MutableList<u64>
  private var total:  i64 = 0
  private var result: Digest? = null

  implement Hasher {
    static fun start(): Sha384 = Sha384(
      state: IV384.toMutable(),
      buffer: [],
      scratch: MutableList<u64>.repeat(0, 80),
    )

    static fun algorithm(): string = "SHA-384"
    static fun blockSize(): i64 = 128
    static fun digestSize(): i64 = 48

    fun update(data: List<u8>) {
      if (self.result != null) panic("crypto.Sha384: update after finish")
      self.total = self.total + data.len()
      self.absorb(data)
    }

    fun finish(): Digest {
      val done = self.result
      if (done != null) return done
      self.absorb(padding(self.buffer.len(), 128, 16, self.total))
      val out: MutableList<u8> = []
      loop (w in self.state) {
        pushU64(out, w)
      }
      val d = Digest.of(out.take(48))
      self.result = d
      d
    }
  }

  private fun absorb(data: List<u8>) {
    val n = data.len()
    var i = 0
    if (self.buffer.len() > 0) {
      loop (self.buffer.len() < 128 && i < n) {
        self.buffer.push(data.at(i))
        i = i + 1
      }
      if (self.buffer.len() < 128) return
      compress512(self.state, self.buffer.toList(), 0, self.scratch)
      self.buffer.clear()
    }
    loop (i + 128 <= n) {
      compress512(self.state, data, i, self.scratch)
      i = i + 128
    }
    loop (i < n) {
      self.buffer.push(data.at(i))
      i = i + 1
    }
  }
}

/// Word `i` of the message schedule. The rounds read only 0..<80, and
/// `w` has 80 words, so this cannot fail.
fun schedule512(w: MutableList<u64>, i: i64): u64 = w.at(i) ?: panic("sha512: the rounds read the 80-word schedule inside 0..<80")
