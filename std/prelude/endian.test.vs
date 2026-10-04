// Tests of the endian bytes (D130): a number to its bytes and back in both
// orders at every width, the readers' bounds, and the writers.

test "a number is its bytes, most significant first or last" {
  expect(0x0102.wrapU16().toBeBytes() == [1, 2])
  expect(0x0102.wrapU16().toLeBytes() == [2, 1])
  expect(0x01020304.wrapU32().toBeBytes() == [1, 2, 3, 4])
  expect(0x01020304.wrapU32().toLeBytes() == [4, 3, 2, 1])
  expect(0x0102030405060708.wrapU64().toBeBytes() == [1, 2, 3, 4, 5, 6, 7, 8])
  expect(0x0102030405060708.wrapU64().toLeBytes() == [8, 7, 6, 5, 4, 3, 2, 1])
  expect(200.wrapU8().toBeBytes() == [200])
  expect((-1).wrapI8().toLeBytes() == [255])
}

test "a negative number is its two's complement" {
  expect((-2).wrapI32().toBeBytes() == [255, 255, 255, 254])
  expect((-2).wrapI16().toLeBytes() == [254, 255])
  expect((-1).wrapI64().toBeBytes() == [255, 255, 255, 255, 255, 255, 255, 255])
  expect(i32.fromBeBytes([255, 255, 255, 254]) == -2)
  expect(i16.fromLeBytes([0, 128]) == -32768)
  expect(i64.fromBeBytes([128, 0, 0, 0, 0, 0, 0, 0]) == -9223372036854775807 - 1)
}

test "bytes become the number, at every width and in both orders" {
  expect(u16.fromBeBytes([1, 2]) == 258)
  expect(u16.fromLeBytes([1, 2]) == 513)
  expect(u32.fromBeBytes([0xde, 0xad, 0xbe, 0xef]) == 3735928559)
  expect(u64.fromLeBytes([1, 0, 0, 0, 0, 0, 0, 128]) == 9223372036854775809)
  expect(usize.fromBeBytes(65536.wrapUsize().toBeBytes()) == 65536)
  expect(isize.fromLeBytes((-7).wrapIsize().toLeBytes()) == -7)
}

test "to and from bytes agree for values at the edges" {
  loop (v in [0, 1, 255, 256, 65535, 65536, 2147483647, 4294967295, 4294967296]) {
    expect(u64.fromBeBytes(v.wrapU64().toBeBytes()) == v.wrapU64())
    expect(u64.fromLeBytes(v.wrapU64().toLeBytes()) == v.wrapU64())
    expect(u32.fromBeBytes(v.wrapU32().toBeBytes()) == v.wrapU32())
    expect(i32.fromLeBytes(v.wrapI32().toLeBytes()) == v.wrapI32())
  }
}

test "a list is read at an offset, null when the bytes are not all there" {
  val data: List<u8> = [1, 2, 3, 4, 5, 6, 7, 8]
  expect(data.readU16Be(0) == 258)
  expect(data.readU16Le(0) == 513)
  expect(data.readU32Be(2) == 50595078)
  expect(data.readU32Le(2) == 100992003)
  expect(data.readU64Be(0) == 72623859790382856)
  expect(data.readU16Be(6) == 1800)
  expect(data.readU16Be(7) == null)
  expect(data.readU32Be(5) == null)
  expect(data.readU64Le(1) == null)
  expect(data.readU16Be(-1) == null)
  expect(data.readU16Be(9223372036854775807) == null)
  val none: List<u8> = []
  expect(none.readU16Be(0) == null)
}

test "a signed read keeps the sign" {
  val data: List<u8> = [255, 254, 128, 0, 0, 0, 0, 0, 0, 0]
  expect(data.readI16Be(0) == -2)
  expect(data.readI16Le(0) == -257)
  expect(data.readI32Be(0) == -98304)
  expect(data.readI64Be(2) == -9223372036854775807 - 1)
}

test "the readers work on a list being built" {
  val out: MutableList<u8> = [0, 0, 0, 1]
  expect(out.readU32Be(0) == 1)
}

test "writers append the bytes in the order asked" {
  val out: MutableList<u8> = []
  out.pushU16Be(0x0102.wrapU16())
  out.pushU16Le(0x0102.wrapU16())
  out.pushU32Be(0xdeadbeef.wrapU32())
  out.pushI32Le((-2).wrapI32())
  out.pushU64Be(1.wrapU64())
  out.pushI16Be((-1).wrapI16())
  expect(out.toList() == [1, 2, 2, 1, 222, 173, 190, 239, 254, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 1, 255, 255])
  expect(out.readU32Be(4) == 3735928559)
  expect(out.readI32Le(8) == -2)
}

test "a header written and read back" {
  // a made-up record: magic, version, length, checksum
  val out: MutableList<u8> = []
  out.pushU32Be(0x56454c53.wrapU32())
  out.pushU16Le(3.wrapU16())
  out.pushU32Le(70000.wrapU32())
  out.pushI64Be((-5).wrapI64())
  val magic = out.readU32Be(0) ?: 0
  expect(magic == 0x56454c53.wrapU32())
  expect(out.slice(0, 4) == "VELS".bytes())
  expect(out.readU16Le(4) == 3)
  expect(out.readU32Le(6) == 70000)
  expect(out.readI64Be(10) == -5)
  expect(out.len() == 18)
}
