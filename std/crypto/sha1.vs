// SHA-1 (FIPS 180-4), for the protocols that still specify it. Collisions
// have been practical since 2017: never choose it for a signature, a
// content address or a password. `crypto.sha1Legacy` is the name on
// purpose.

const IV1: List<u32> = [0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0]

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
    static fun start(): Sha1 => Sha1(
      state: IV1.toMutable(),
      buffer: [],
      scratch: MutableList<u32>.repeat(0, 80),
    )

    static fun algorithm(): string => "SHA-1"
    static fun blockSize(): i64 => 64
    static fun digestSize(): i64 => 20

    fun update(data: List<u8>) {
      if (this.result != null) panic("crypto.Sha1: update after finish")
      this.total = this.total + data.len()
      this.absorb(data)
    }

    fun finish(): Digest {
      val done = this.result
      if (done != null) return done
      this.absorb(padding(this.buffer.len(), 64, 8, this.total))
      val out: MutableList<u8> = []
      loop (w in this.state) {
        pushU32(out, w)
      }
      val d = Digest.of(out.toList())
      this.result = d
      d
    }
  }

  private fun absorb(data: List<u8>) {
    val n = data.len()
    var i = 0
    if (this.buffer.len() > 0) {
      loop (this.buffer.len() < 64 && i < n) {
        this.buffer.push(data.at(i))
        i = i + 1
      }
      if (this.buffer.len() < 64) return
      compress1(this.state, this.buffer.toList(), 0, this.scratch)
      this.buffer.clear()
    }
    loop (i + 64 <= n) {
      compress1(this.state, data, i, this.scratch)
      i = i + 64
    }
    loop (i < n) {
      this.buffer.push(data.at(i))
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
    val x = schedule1(w, i - 3) ^ schedule1(w, i - 8) ^ schedule1(w, i - 14) ^ schedule1(w, i - 16)
    w.set(i, rotl32(x, 1))
    i = i + 1
  }

  val [a0, b0, c0, d0, e0] = state else panic("sha1: the state is 5 words")
  var a = a0
  var b = b0
  var c = c0
  var d = d0
  var e = e0

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
    val t = rotl32(a, 5) +% f +% e +% k +% schedule1(w, r)
    e = d
    d = c
    c = rotl32(b, 30)
    b = a
    a = t
  }

  state.set(0, a0 +% a)
  state.set(1, b0 +% b)
  state.set(2, c0 +% c)
  state.set(3, d0 +% d)
  state.set(4, e0 +% e)
}

/// Word `i` of the message schedule. The rounds read only 0..<80, and
/// `w` has 80 words, so this cannot fail.
fun schedule1(w: MutableList<u32>, i: i64): u32 => w.at(i) ?: panic("sha1: the rounds read the 80-word schedule inside 0..<80")
