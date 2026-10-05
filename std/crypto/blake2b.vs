// BLAKE2b (RFC 7693), one-shot, for what Argon2 needs: any output length from
// 1 to 64 bytes, optionally keyed. It is not exposed as a hash of its own;
// `sha256` and `sha512` are the digests std offers.

const B2IV: List<u64> = [
  0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
  0x510e527fade682d1, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
]

// the message word order of each of the ten distinct rounds (the 11th and 12th repeat the 1st and 2nd)
const B2SIGMA: List<i64> = [
  0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
  14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3,
  11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4,
  7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8,
  9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13,
  2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9,
  12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11,
  13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10,
  6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5,
  10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0,
]

// the mixing function: four words of the working vector and two message words
fun b2g(v: MutableList<u64>, a: i64, b: i64, c: i64, d: i64, x: u64, y: u64) {
  var va = v.wordAt(a)
  var vb = v.wordAt(b)
  var vc = v.wordAt(c)
  var vd = v.wordAt(d)
  va = va +% vb +% x
  vd = rotr64(vd ^ va, 32)
  vc = vc +% vd
  vb = rotr64(vb ^ vc, 24)
  va = va +% vb +% y
  vd = rotr64(vd ^ va, 16)
  vc = vc +% vd
  vb = rotr64(vb ^ vc, 63)
  v.set(a, va)
  v.set(b, vb)
  v.set(c, vc)
  v.set(d, vd)
}

// one 128-byte block into the state; `counter` is the bytes hashed so far including this block
fun b2compress(h: MutableList<u64>, block: List<u8>, counter: u64, last: bool, v: MutableList<u64>, m: MutableList<u64>) {
  loop (i in 0..<16) {
    var w: u64 = 0
    loop (j in 0..<8) {
      w = w | ((block.at(i * 8 + j) ?: 0).toU64() << (8 * j))
    }
    m.set(i, w)
  }
  loop (i in 0..<8) {
    v.set(i, h.wordAt(i))
    v.set(i + 8, B2IV.at(i) ?: 0)
  }
  v.set(12, v.wordAt(12) ^ counter)
  if (last) v.set(14, ~v.wordAt(14))
  loop (r in 0..<12) {
    val s = (r % 10) * 16
    loop (col in 0..<4) {
      // columns, then diagonals
      b2g(v, col, col + 4, col + 8, col + 12, m.wordAt(B2SIGMA.at(s + 2 * col) ?: 0), m.wordAt(B2SIGMA.at(s + 2 * col + 1) ?: 0))
    }
    b2g(v, 0, 5, 10, 15, m.wordAt(B2SIGMA.at(s + 8) ?: 0), m.wordAt(B2SIGMA.at(s + 9) ?: 0))
    b2g(v, 1, 6, 11, 12, m.wordAt(B2SIGMA.at(s + 10) ?: 0), m.wordAt(B2SIGMA.at(s + 11) ?: 0))
    b2g(v, 2, 7, 8, 13, m.wordAt(B2SIGMA.at(s + 12) ?: 0), m.wordAt(B2SIGMA.at(s + 13) ?: 0))
    b2g(v, 3, 4, 9, 14, m.wordAt(B2SIGMA.at(s + 14) ?: 0), m.wordAt(B2SIGMA.at(s + 15) ?: 0))
  }
  loop (i in 0..<8) {
    h.set(i, h.wordAt(i) ^ v.wordAt(i) ^ v.wordAt(i + 8))
  }
}

/// BLAKE2b of `data`, `outLen` bytes (1 to 64), under `key` (empty for none, at most 64 bytes).
/// Panics for a length or key outside those.
fun blake2b(data: List<u8>, outLen: i64, key: List<u8> = []): List<u8> {
  if (outLen < 1 || outLen > 64) panic("blake2b: the output is 1 to 64 bytes, got $outLen")
  if (key.len() > 64) panic("blake2b: the key is at most 64 bytes, got ${key.len()}")
  val h = B2IV.toMutable()
  h.set(0, h.wordAt(0) ^ 0x01010000 ^ (key.len().wrapU64() << 8) ^ outLen.wrapU64())
  val v: MutableList<u64> = MutableList<u64>.repeat(0, 16)
  val m: MutableList<u64> = MutableList<u64>.repeat(0, 16)
  // a key is a first block of its own, padded with zeros
  val input: MutableList<u8> = []
  if (!key.isEmpty()) {
    input.addAll(key)
    loop (input.len() < 128) {
      input.push(0)
    }
  }
  input.addAll(data)
  val total = input.len()
  var at = 0
  // every block but the last (the last is the final block even when it is full, and even when empty)
  loop (total - at > 128) {
    b2compress(h, input.slice(at, at + 128), (at + 128).wrapU64(), false, v, m)
    at += 128
  }
  val tail: MutableList<u8> = input.slice(at, total).toMutable()
  loop (tail.len() < 128) {
    tail.push(0)
  }
  b2compress(h, tail.toList(), total.wrapU64(), true, v, m)
  val out: MutableList<u8> = []
  loop (w in h) loop (j in 0..<8) {
    out.push((w >> (8 * j)).wrapU8())
  }
  out.slice(0, outLen)
}

extend MutableList<u64> {
  // the word at `i`, for code that indexes inside what it allocated
  fun wordAt(i: i64): u64 = this.at(i) ?: panic("crypto: a word index outside its block")
}
