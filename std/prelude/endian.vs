// Prelude — endian bytes (D130). A number to its bytes and back, and fixed
// width numbers read from and written to a List<u8> at an offset: what a
// binary format, a network header or a file with a layout is made of. "Be" is
// big endian (network order), "Le" little endian. isize and usize are 8 bytes:
// the targets are 64-bit.

// the low n bytes of v, most significant first
fun beBytesOf<const N: i64>(v: u64): Array<u8, N> {
  var out: Array<u8, N> = Array.make(0)
  var rest = v
  var i = N - 1
  loop (i >= 0) {
    out.set(i, (rest & 255).wrapU8())
    rest = rest >> 8
    i -= 1
  }
  out
}

// the low n bytes of v, least significant first
fun leBytesOf<const N: i64>(v: u64): Array<u8, N> {
  var out: Array<u8, N> = Array.make(0)
  var rest = v
  loop (i in 0..<N) {
    out.set(i, (rest & 255).wrapU8())
    rest = rest >> 8
  }
  out
}

fun beValueOf<const N: i64>(a: Array<u8, N>): u64 {
  var v: u64 = 0
  loop (b in a) {
    v = (v << 8) | b.toU64()
  }
  v
}

fun leValueOf<const N: i64>(a: Array<u8, N>): u64 {
  var v: u64 = 0
  var i = N - 1
  loop (i >= 0) {
    v = (v << 8) | (a.at(i) ?: 0).toU64()
    i -= 1
  }
  v
}

// n bytes at offset, big or little endian; null when they are not all there
fun readBytesAs(list: List<u8>, offset: i64, n: i64, big: bool): u64? {
  if (offset < 0 || n > list.len() || offset > list.len() - n) return null
  var v: u64 = 0
  loop (i in 0..<n) {
    val b = (list.at(if (big) offset + i else offset + n - 1 - i) ?: return null).toU64()
    v = (v << 8) | b
  }
  v
}

fun pushBytesOf(list: MutableList<u8>, v: u64, n: i64, big: bool) {
  loop (i in 0..<n) {
    val shift = (if (big) n - 1 - i else i) * 8
    list.push(((v >> shift.wrapU64()) & 255).wrapU8())
  }
}

extend i8 {
  /// The 1 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 1> = beBytesOf<1>(this.wrapU64())

  /// The 1 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 1> = leBytesOf<1>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 1>): i8 = beValueOf<1>(bytes).wrapI8()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 1>): i8 = leValueOf<1>(bytes).wrapI8()
}

extend u8 {
  /// The 1 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 1> = beBytesOf<1>(this.wrapU64())

  /// The 1 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 1> = leBytesOf<1>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 1>): u8 = beValueOf<1>(bytes).wrapU8()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 1>): u8 = leValueOf<1>(bytes).wrapU8()
}

extend i16 {
  /// The 2 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 2> = beBytesOf<2>(this.wrapU64())

  /// The 2 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 2> = leBytesOf<2>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 2>): i16 = beValueOf<2>(bytes).wrapI16()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 2>): i16 = leValueOf<2>(bytes).wrapI16()
}

extend u16 {
  /// The 2 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 2> = beBytesOf<2>(this.wrapU64())

  /// The 2 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 2> = leBytesOf<2>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 2>): u16 = beValueOf<2>(bytes).wrapU16()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 2>): u16 = leValueOf<2>(bytes).wrapU16()
}

extend i32 {
  /// The 4 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 4> = beBytesOf<4>(this.wrapU64())

  /// The 4 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 4> = leBytesOf<4>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 4>): i32 = beValueOf<4>(bytes).wrapI32()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 4>): i32 = leValueOf<4>(bytes).wrapI32()
}

extend u32 {
  /// The 4 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 4> = beBytesOf<4>(this.wrapU64())

  /// The 4 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 4> = leBytesOf<4>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 4>): u32 = beValueOf<4>(bytes).wrapU32()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 4>): u32 = leValueOf<4>(bytes).wrapU32()
}

extend i64 {
  /// The 8 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 8> = beBytesOf<8>(this.wrapU64())

  /// The 8 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 8> = leBytesOf<8>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 8>): i64 = beValueOf<8>(bytes).wrapI64()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 8>): i64 = leValueOf<8>(bytes).wrapI64()
}

extend u64 {
  /// The 8 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 8> = beBytesOf<8>(this.wrapU64())

  /// The 8 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 8> = leBytesOf<8>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 8>): u64 = beValueOf<8>(bytes).wrapU64()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 8>): u64 = leValueOf<8>(bytes).wrapU64()
}

extend isize {
  /// The 8 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 8> = beBytesOf<8>(this.wrapU64())

  /// The 8 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 8> = leBytesOf<8>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 8>): isize = beValueOf<8>(bytes).wrapIsize()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 8>): isize = leValueOf<8>(bytes).wrapIsize()
}

extend usize {
  /// The 8 byte(s) of the number, most significant first (big endian, network order).
  public fun toBeBytes(): Array<u8, 8> = beBytesOf<8>(this.wrapU64())

  /// The 8 byte(s) of the number, least significant first (little endian).
  public fun toLeBytes(): Array<u8, 8> = leBytesOf<8>(this.wrapU64())

  /// The number whose big endian bytes these are.
  public static fun fromBeBytes(bytes: Array<u8, 8>): usize = beValueOf<8>(bytes).wrapUsize()

  /// The number whose little endian bytes these are.
  public static fun fromLeBytes(bytes: Array<u8, 8>): usize = leValueOf<8>(bytes).wrapUsize()
}

extend List<u8> {
  /// The i16 at `offset`, big endian; `null` when its 2 bytes are not all in the list.
  public fun readI16Be(offset: i64): i16? {
    val v = readBytesAs(this, offset, 2, true) ?: return null
    v.wrapI16()
  }

  /// The i16 at `offset`, little endian; `null` when its 2 bytes are not all in the list.
  public fun readI16Le(offset: i64): i16? {
    val v = readBytesAs(this, offset, 2, false) ?: return null
    v.wrapI16()
  }

  /// The u16 at `offset`, big endian; `null` when its 2 bytes are not all in the list.
  public fun readU16Be(offset: i64): u16? {
    val v = readBytesAs(this, offset, 2, true) ?: return null
    v.wrapU16()
  }

  /// The u16 at `offset`, little endian; `null` when its 2 bytes are not all in the list.
  public fun readU16Le(offset: i64): u16? {
    val v = readBytesAs(this, offset, 2, false) ?: return null
    v.wrapU16()
  }

  /// The i32 at `offset`, big endian; `null` when its 4 bytes are not all in the list.
  public fun readI32Be(offset: i64): i32? {
    val v = readBytesAs(this, offset, 4, true) ?: return null
    v.wrapI32()
  }

  /// The i32 at `offset`, little endian; `null` when its 4 bytes are not all in the list.
  public fun readI32Le(offset: i64): i32? {
    val v = readBytesAs(this, offset, 4, false) ?: return null
    v.wrapI32()
  }

  /// The u32 at `offset`, big endian; `null` when its 4 bytes are not all in the list.
  public fun readU32Be(offset: i64): u32? {
    val v = readBytesAs(this, offset, 4, true) ?: return null
    v.wrapU32()
  }

  /// The u32 at `offset`, little endian; `null` when its 4 bytes are not all in the list.
  public fun readU32Le(offset: i64): u32? {
    val v = readBytesAs(this, offset, 4, false) ?: return null
    v.wrapU32()
  }

  /// The i64 at `offset`, big endian; `null` when its 8 bytes are not all in the list.
  public fun readI64Be(offset: i64): i64? {
    val v = readBytesAs(this, offset, 8, true) ?: return null
    v.wrapI64()
  }

  /// The i64 at `offset`, little endian; `null` when its 8 bytes are not all in the list.
  public fun readI64Le(offset: i64): i64? {
    val v = readBytesAs(this, offset, 8, false) ?: return null
    v.wrapI64()
  }

  /// The u64 at `offset`, big endian; `null` when its 8 bytes are not all in the list.
  public fun readU64Be(offset: i64): u64? {
    val v = readBytesAs(this, offset, 8, true) ?: return null
    v.wrapU64()
  }

  /// The u64 at `offset`, little endian; `null` when its 8 bytes are not all in the list.
  public fun readU64Le(offset: i64): u64? {
    val v = readBytesAs(this, offset, 8, false) ?: return null
    v.wrapU64()
  }

  /// The isize at `offset`, big endian; `null` when its 8 bytes are not all in the list.
  public fun readIsizeBe(offset: i64): isize? {
    val v = readBytesAs(this, offset, 8, true) ?: return null
    v.wrapIsize()
  }

  /// The isize at `offset`, little endian; `null` when its 8 bytes are not all in the list.
  public fun readIsizeLe(offset: i64): isize? {
    val v = readBytesAs(this, offset, 8, false) ?: return null
    v.wrapIsize()
  }

  /// The usize at `offset`, big endian; `null` when its 8 bytes are not all in the list.
  public fun readUsizeBe(offset: i64): usize? {
    val v = readBytesAs(this, offset, 8, true) ?: return null
    v.wrapUsize()
  }

  /// The usize at `offset`, little endian; `null` when its 8 bytes are not all in the list.
  public fun readUsizeLe(offset: i64): usize? {
    val v = readBytesAs(this, offset, 8, false) ?: return null
    v.wrapUsize()
  }
}

extend MutableList<u8> {
  /// Appends the 2 bytes of `x`, big endian.
  public fun pushI16Be(x: i16) {
    pushBytesOf(this, x.wrapU64(), 2, true)
  }

  /// Appends the 2 bytes of `x`, little endian.
  public fun pushI16Le(x: i16) {
    pushBytesOf(this, x.wrapU64(), 2, false)
  }

  /// Appends the 2 bytes of `x`, big endian.
  public fun pushU16Be(x: u16) {
    pushBytesOf(this, x.wrapU64(), 2, true)
  }

  /// Appends the 2 bytes of `x`, little endian.
  public fun pushU16Le(x: u16) {
    pushBytesOf(this, x.wrapU64(), 2, false)
  }

  /// Appends the 4 bytes of `x`, big endian.
  public fun pushI32Be(x: i32) {
    pushBytesOf(this, x.wrapU64(), 4, true)
  }

  /// Appends the 4 bytes of `x`, little endian.
  public fun pushI32Le(x: i32) {
    pushBytesOf(this, x.wrapU64(), 4, false)
  }

  /// Appends the 4 bytes of `x`, big endian.
  public fun pushU32Be(x: u32) {
    pushBytesOf(this, x.wrapU64(), 4, true)
  }

  /// Appends the 4 bytes of `x`, little endian.
  public fun pushU32Le(x: u32) {
    pushBytesOf(this, x.wrapU64(), 4, false)
  }

  /// Appends the 8 bytes of `x`, big endian.
  public fun pushI64Be(x: i64) {
    pushBytesOf(this, x.wrapU64(), 8, true)
  }

  /// Appends the 8 bytes of `x`, little endian.
  public fun pushI64Le(x: i64) {
    pushBytesOf(this, x.wrapU64(), 8, false)
  }

  /// Appends the 8 bytes of `x`, big endian.
  public fun pushU64Be(x: u64) {
    pushBytesOf(this, x.wrapU64(), 8, true)
  }

  /// Appends the 8 bytes of `x`, little endian.
  public fun pushU64Le(x: u64) {
    pushBytesOf(this, x.wrapU64(), 8, false)
  }

  /// Appends the 8 bytes of `x`, big endian.
  public fun pushIsizeBe(x: isize) {
    pushBytesOf(this, x.wrapU64(), 8, true)
  }

  /// Appends the 8 bytes of `x`, little endian.
  public fun pushIsizeLe(x: isize) {
    pushBytesOf(this, x.wrapU64(), 8, false)
  }

  /// Appends the 8 bytes of `x`, big endian.
  public fun pushUsizeBe(x: usize) {
    pushBytesOf(this, x.wrapU64(), 8, true)
  }

  /// Appends the 8 bytes of `x`, little endian.
  public fun pushUsizeLe(x: usize) {
    pushBytesOf(this, x.wrapU64(), 8, false)
  }
}
