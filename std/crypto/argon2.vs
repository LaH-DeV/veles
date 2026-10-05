// Argon2 (RFC 9106): the memory-hard function behind `hashPassword`. A password
// is run through a matrix of `memoryKiB` blocks of 1 KiB, `passes` times, in
// a pattern that makes every shortcut cost memory — so a stolen database of
// hashes is expensive to attack on a GPU or an ASIC, where plain SHA-256 is
// not. Argon2id, the variant for passwords, takes its first half-pass from
// addresses that do not depend on the password (no timing leak) and the rest
// from the data (the memory-hardness).
//
// Written to the RFC, one lane after the other, so the output is the one a
// parallel implementation computes; checked against the RFC's test vectors.

/// The variants: `d` is data-dependent addressing, `i` independent, `id` the mix.
enum Argon2Type {
  D
  I
  Id
}

fun argon2TypeCode(t: Argon2Type): i64 = when (t) {
  Argon2Type.D  => 0
  Argon2Type.I  => 1
  Argon2Type.Id => 2
}

const argonVersion: i64 = 19  // 0x13

// the word indexes the two halves of the block permutation touch: eight groups of
// sixteen words, first the rows of the 8x8 matrix of 16-byte registers, then its columns

fun buildRows(): List<i64> {
  val out: MutableList<i64> = []
  loop (r in 0..<8) loop (k in 0..<16) {
    out.push(r * 16 + k)
  }
  out.toList()
}

fun buildCols(): List<i64> {
  val out: MutableList<i64> = []
  loop (c in 0..<8) loop (r in 0..<8) {
    out.push(r * 16 + c * 2)
    out.push(r * 16 + c * 2 + 1)
  }
  out.toList()
}

// BlaMka: a + b + 2 * lo(a) * lo(b), where lo is the low 32 bits
fun blamka(a: u64, b: u64): u64 = a +% b +% (((a & 0xffffffff) * (b & 0xffffffff)) << 1)

// the quarter round on four words of z, named by their indexes
fun argonGB(z: MutableList<u64>, ia: i64, ib: i64, ic: i64, id: i64) {
  var a = z.wordAt(ia)
  var b = z.wordAt(ib)
  var c = z.wordAt(ic)
  var d = z.wordAt(id)
  a = blamka(a, b)
  d = rotr64(d ^ a, 32)
  c = blamka(c, d)
  b = rotr64(b ^ c, 24)
  a = blamka(a, b)
  d = rotr64(d ^ a, 16)
  c = blamka(c, d)
  b = rotr64(b ^ c, 63)
  z.set(ia, a)
  z.set(ib, b)
  z.set(ic, c)
  z.set(id, d)
}

// the permutation on the sixteen words that `tbl` names from `base`
fun argonPerm(z: MutableList<u64>, tbl: List<i64>, base: i64) {
  val i0 = tbl.at(base) ?: 0
  val i1 = tbl.at(base + 1) ?: 0
  val i2 = tbl.at(base + 2) ?: 0
  val i3 = tbl.at(base + 3) ?: 0
  val i4 = tbl.at(base + 4) ?: 0
  val i5 = tbl.at(base + 5) ?: 0
  val i6 = tbl.at(base + 6) ?: 0
  val i7 = tbl.at(base + 7) ?: 0
  val i8 = tbl.at(base + 8) ?: 0
  val i9 = tbl.at(base + 9) ?: 0
  val i10 = tbl.at(base + 10) ?: 0
  val i11 = tbl.at(base + 11) ?: 0
  val i12 = tbl.at(base + 12) ?: 0
  val i13 = tbl.at(base + 13) ?: 0
  val i14 = tbl.at(base + 14) ?: 0
  val i15 = tbl.at(base + 15) ?: 0
  argonGB(z, i0, i4, i8, i12)
  argonGB(z, i1, i5, i9, i13)
  argonGB(z, i2, i6, i10, i14)
  argonGB(z, i3, i7, i11, i15)
  argonGB(z, i0, i5, i10, i15)
  argonGB(z, i1, i6, i11, i12)
  argonGB(z, i2, i7, i8, i13)
  argonGB(z, i3, i4, i9, i14)
}

// out[ooff..] = G(x[xoff..], y[yoff..]) (xor the old content of out when `keep`): the compression
// function, with `rr` and `z` as 128-word scratch
fun argonG(out: MutableList<u64>, ooff: i64, x: MutableList<u64>, xoff: i64, y: MutableList<u64>, yoff: i64, keep: bool, rr: MutableList<u64>, z: MutableList<u64>, rows: List<i64>, cols: List<i64>) {
  loop (k in 0..<128) {
    val r = x.wordAt(xoff + k) ^ y.wordAt(yoff + k)
    rr.set(k, r)
    z.set(k, r)
  }
  loop (r in 0..<8) {
    argonPerm(z, rows, r * 16)
  }
  loop (c in 0..<8) {
    argonPerm(z, cols, c * 16)
  }
  if (keep) {
    loop (k in 0..<128) {
      out.set(ooff + k, out.wordAt(ooff + k) ^ z.wordAt(k) ^ rr.wordAt(k))
    }
  } else {
    loop (k in 0..<128) {
      out.set(ooff + k, z.wordAt(k) ^ rr.wordAt(k))
    }
  }
}

// the next block of 128 addresses: the counter in word 6 goes up and G is applied twice to the input
fun argonNext(input: MutableList<u64>, tmp: MutableList<u64>, addr: MutableList<u64>, zero: MutableList<u64>, rr: MutableList<u64>, z: MutableList<u64>, rows: List<i64>, cols: List<i64>) {
  input.set(6, input.wordAt(6) +% 1)
  argonG(tmp, 0, zero, 0, input, 0, false, rr, z, rows, cols)
  argonG(addr, 0, zero, 0, tmp, 0, false, rr, z, rows, cols)
}

fun le32(out: MutableList<u8>, n: i64) {
  loop (j in 0..<4) {
    out.push((n >> (8 * j)).wrapU8())
  }
}

// H': BLAKE2b stretched to any length (RFC 9106 §3.3)
fun argonHash(input: List<u8>, outLen: i64): List<u8> {
  val head: MutableList<u8> = []
  le32(head, outLen)
  head.addAll(input)
  if (outLen <= 64) return blake2b(head.toList(), outLen)
  val out: MutableList<u8> = []
  var v = blake2b(head.toList(), 64)
  out.addAll(v.slice(0, 32))
  var left = outLen - 32
  loop (left > 64) {
    v = blake2b(v, 64)
    out.addAll(v.slice(0, 32))
    left -= 32
  }
  out.addAll(blake2b(v, left))
  out.toList()
}

// a block as 128 words, from 1024 bytes
fun wordsOf(bytes: List<u8>, into: MutableList<u64>, at: i64) {
  loop (i in 0..<128) {
    var w: u64 = 0
    loop (j in 0..<8) {
      w = w | ((bytes.at(i * 8 + j) ?: 0).toU64() << (8 * j))
    }
    into.set(at + i, w)
  }
}

/// The Argon2 tag of `password` under `salt`: `memoryKiB` KiB, `passes`
/// passes, `lanes` lanes, `tagLen` bytes out; `secret` and `data` are
/// the optional key and associated data of the RFC. Panics for parameters the
/// RFC forbids (memory under 8 KiB per lane, no passes, a salt under 8 bytes,
/// a tag under 4 bytes).
fun argon2(kind: Argon2Type, password: List<u8>, salt: List<u8>, memoryKiB: i64, passes: i64, lanes: i64, tagLen: i64, secret: List<u8> = [], data: List<u8> = []): List<u8> {
  if (lanes < 1 || lanes > 16777215) panic("argon2: lanes must be 1 to 16777215, got $lanes")
  if (passes < 1) panic("argon2: passes must be at least 1, got $passes")
  if (tagLen < 4) panic("argon2: the tag must be at least 4 bytes, got $tagLen")
  if (salt.len() < 8) panic("argon2: the salt must be at least 8 bytes, got ${salt.len()}")
  if (memoryKiB < 8 * lanes) panic("argon2: memory must be at least 8 KiB per lane, got $memoryKiB KiB for $lanes lanes")
  val typeCode = argon2TypeCode(kind)

  // H0: everything the parameters and inputs are, hashed once
  val seed: MutableList<u8> = []
  le32(seed, lanes)
  le32(seed, tagLen)
  le32(seed, memoryKiB)
  le32(seed, passes)
  le32(seed, argonVersion)
  le32(seed, typeCode)
  le32(seed, password.len())
  seed.addAll(password)
  le32(seed, salt.len())
  seed.addAll(salt)
  le32(seed, secret.len())
  seed.addAll(secret)
  le32(seed, data.len())
  seed.addAll(data)
  val h0 = blake2b(seed.toList(), 64)

  // the matrix: `lanes` rows of `columns` blocks, a block being 128 words
  val columns = (memoryKiB / (4 * lanes)) * 4
  val segment = columns / 4
  val mem: MutableList<u64> = MutableList<u64>.repeat(0, lanes * columns * 128)
  loop (l in 0..<lanes) loop (c in 0..<2) {
    val start: MutableList<u8> = []
    start.addAll(h0)
    le32(start, c)
    le32(start, l)
    wordsOf(argonHash(start.toList(), 1024), mem, (l * columns + c) * 128)
  }

  val rows = buildRows()
  val cols = buildCols()
  val rr: MutableList<u64> = MutableList<u64>.repeat(0, 128)
  val z: MutableList<u64> = MutableList<u64>.repeat(0, 128)
  val zero: MutableList<u64> = MutableList<u64>.repeat(0, 128)
  val input: MutableList<u64> = MutableList<u64>.repeat(0, 128)
  val addr: MutableList<u64> = MutableList<u64>.repeat(0, 128)
  val tmp: MutableList<u64> = MutableList<u64>.repeat(0, 128)

  loop (pass in 0..<passes) loop (slice in 0..<4) loop (lane in 0..<lanes) {
    // addresses that do not depend on the data: Argon2i always, Argon2id in the first half of the first pass
    val independent = typeCode == 1 || (typeCode == 2 && pass == 0 && slice < 2)
    if (independent) {
      loop (k in 0..<128) {
        input.set(k, 0)
      }
      input.set(0, pass.wrapU64())
      input.set(1, lane.wrapU64())
      input.set(2, slice.wrapU64())
      input.set(3, (lanes * columns).wrapU64())
      input.set(4, passes.wrapU64())
      input.set(5, typeCode.wrapU64())
    }
    var first = 0
    if (pass == 0 && slice == 0) {
      // the first two blocks of the lane are made; the address stream starts at its first block
      first = 2
      if (independent) argonNext(input, tmp, addr, zero, rr, z, rows, cols)
    }
    loop (idx in first..<segment) {
      val col = slice * segment + idx
      val cur = (lane * columns + col) * 128
      val prevCol = if (col == 0) columns - 1 else col - 1
      val prev = (lane * columns + prevCol) * 128
      var pseudo: u64 = 0
      if (independent) {
        // a fresh block of 128 addresses every 128 references
        if (idx % 128 == 0) argonNext(input, tmp, addr, zero, rr, z, rows, cols)
        pseudo = addr.wordAt(idx % 128)
      } else {
        pseudo = mem.wordAt(prev)
      }
      val j1 = pseudo & 0xffffffff
      val j2 = pseudo >> 32
      var refLane = lane
      if (!(pass == 0 && slice == 0)) refLane = (j2 % lanes.wrapU64()).wrapI64()
      // how many blocks of the reference lane may be referenced (RFC 9106 §3.4.1.2)
      var area: i64 = 0
      if (pass == 0) {
        if (slice == 0) {
          area = idx - 1
        } else if (refLane == lane) {
          area = slice * segment + idx - 1
        } else {
          area = slice * segment - (if (idx == 0) 1 else 0)
        }
      } else if (refLane == lane) {
        area = columns - segment + idx - 1
      } else {
        area = columns - segment - (if (idx == 0) 1 else 0)
      }
      val x = (j1 * j1) >> 32
      val y = (area.wrapU64() * x) >> 32
      val zz = area.wrapU64() - 1 - y
      var startPos: i64 = 0
      if (pass != 0) startPos = if (slice == 3) 0 else (slice + 1) * segment
      val refCol = (startPos + zz.wrapI64()) % columns
      val ref = (refLane * columns + refCol) * 128
      // from the second pass on, the new block is mixed with the one it replaces
      argonG(mem, cur, mem, prev, mem, ref, pass > 0, rr, z, rows, cols)
    }
  }

  // the tag: the last block of every lane xored together, stretched to the length wanted
  val last: MutableList<u64> = MutableList<u64>.repeat(0, 128)
  loop (l in 0..<lanes) {
    val at = (l * columns + columns - 1) * 128
    loop (k in 0..<128) {
      last.set(k, last.wordAt(k) ^ mem.wordAt(at + k))
    }
  }
  val bytes: MutableList<u8> = []
  loop (w in last) loop (j in 0..<8) {
    bytes.push((w >> (8 * j)).wrapU8())
  }
  argonHash(bytes.toList(), tagLen)
}
