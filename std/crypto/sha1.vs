// SHA-1 (FIPS 180-4), for the protocols that still specify it. Collisions
// have been practical since 2017: never choose it for a signature, a
// content address or a password. `crypto.sha1Legacy` is the name on
// purpose.

val IV1: List<u32> = [0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0]

/// SHA-1, fed in pieces. See `crypto.sha1Legacy` for why the name says so:
/// this is here for the WebSocket handshake (RFC 6455) and for reading
/// existing Git or Subversion object names, nothing else.
public struct Sha1 {
  private state:      MutableList<u32>
  private buffer:     MutableList<u8>
  private scratch:    MutableList<u32>
  private var total:  i64 = 0
  private var result: Digest? = null

  implement Hasher {
    static fun start(): Sha1 = Sha1(
      state: IV1.toMutable(),
      buffer: [],
      scratch: MutableList<u32>.repeat(0, 80),
    )

    static fun algorithm(): string = "SHA-1"
    static fun blockSize(): i64 = 64
    static fun digestSize(): i64 = 20

    fun update(data: List<u8>) {
      if (self.result != null) panic("crypto.Sha1: update after finish")
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

  private fun absorb(data: List<u8>) {
    val n = data.len()
    var i = 0
    if (self.buffer.len() > 0) {
      loop (self.buffer.len() < 64 && i < n) {
        self.buffer.push(data.atOrPanic(i))
        i = i + 1
      }
      if (self.buffer.len() < 64) return
      compress1(self.state, self.buffer, 0, self.scratch)
      self.buffer.clear()
    }
    loop (i + 64 <= n) {
      compress1(self.state, data, i, self.scratch)
      i = i + 64
    }
    loop (i < n) {
      self.buffer.push(data.atOrPanic(i))
      i = i + 1
    }
  }
}

fun compress1(state: MutableList<u32>, block: List<u8>, at: i64, w: MutableList<u32>) {
  var i = 0
  loop (i < 16) {
    w.set(i, beU32(block, at + i * 4))
    i = i + 1
  }
  loop (i < 80) {
    val x = w.atOrPanic(i - 3) ^ w.atOrPanic(i - 8) ^ w.atOrPanic(i - 14) ^ w.atOrPanic(i - 16)
    w.set(i, rotl32(x, 1))
    i = i + 1
  }

  var a = state.atOrPanic(0)
  var b = state.atOrPanic(1)
  var c = state.atOrPanic(2)
  var d = state.atOrPanic(3)
  var e = state.atOrPanic(4)

  loop (r in 0..<80) {
    var f: u32 = 0
    var k: u32 = 0
    if (r < 20) {
      f = (b & c) | ((~b) & d)
      k = 0x5a827999
    } else if (r < 40) {
      f = b ^ c ^ d
      k = 0x6ed9eba1
    } else if (r < 60) {
      f = (b & c) | (b & d) | (c & d)
      k = 0x8f1bbcdc
    } else {
      f = b ^ c ^ d
      k = 0xca62c1d6
    }
    val t = rotl32(a, 5) +% f +% e +% k +% w.atOrPanic(r)
    e = d
    d = c
    c = rotl32(b, 30)
    b = a
    a = t
  }

  state.set(0, state.atOrPanic(0) +% a)
  state.set(1, state.atOrPanic(1) +% b)
  state.set(2, state.atOrPanic(2) +% c)
  state.set(3, state.atOrPanic(3) +% d)
  state.set(4, state.atOrPanic(4) +% e)
}
