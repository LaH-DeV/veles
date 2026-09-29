// UUIDs (RFC 9562). Two versions are worth generating: v4, sixteen random
// bytes, and v7, a millisecond timestamp followed by random bits — the
// same uniqueness, but sorted by creation time, which is what a database
// index wants.
use time

/// A 128-bit identifier. `toString()` is the canonical lower-case form
/// `0190d3e1-7c00-7000-8000-9a5b1c2d3e4f`; `parse` reads that, or the same
/// digits without the dashes.
///
/// ```veles
/// val id = crypto.uuidV7()
/// io.println("$id")                            // sorted by creation time
/// val back = crypto.Uuid.parse(text) ?: return
/// ```
///
/// UUIDs compare and sort by their bytes, so a list of v7 ids sorts into
/// creation order.
public struct Uuid {
  private data: List<u8>

  /// A random UUID (version 4): 122 random bits. The version and variant
  /// take the other six, so two v4 ids collide about as often as two
  /// 122-bit random numbers do — never, in practice.
  public static fun v4(): Uuid {
    val b = randomBytes(16).toMutable()
    if (b.len() != 16) panic("Uuid.v4: randomBytes(16) returned ${b.len()} bytes")
    b.set(6, (b.at(6) & 0x0f) | 0x40)  // version 4
    b.set(8, (b.at(8) & 0x3f) | 0x80)  // variant 10
    Uuid(data: b.toList())
  }

  /// A time-ordered UUID (version 7): 48 bits of Unix milliseconds, a
  /// 12-bit counter, then 62 random bits.
  ///
  /// Successive calls are **strictly increasing**, which is the point of
  /// v7 — as a primary key it appends to the index instead of scattering
  /// writes across it. Milliseconds alone would not be enough (a server
  /// makes many ids per millisecond), so the twelve bits the RFC calls
  /// `rand_a` hold a counter instead (RFC 9562 §6.2, the "fixed-length
  /// dedicated counter" method): it starts at a random point in the lower
  /// half of its range each new millisecond, so the ids are not
  /// guessable, and 2048 of them fit before it has to borrow a
  /// millisecond from the future. A clock that jumps backwards is
  /// ignored rather than obeyed.
  public static fun v7(): Uuid {
    val (ms, counter) = nextTick()
    val r = randomBytes(8)
    if (r.len() != 8) panic("Uuid.v7: randomBytes(8) returned ${r.len()} bytes")
    val b: MutableList<u8> = []
    var s = 40
    loop (s >= 0) {
      b.push(((ms >> s) & 255).wrapU8())
      s = s - 8
    }
    b.push(0x70 | ((counter >> 8) & 0x0f).wrapU8())  // version 7 + counter high
    b.push((counter & 255).wrapU8())                 // counter low
    b.push((r.at(0) & 0x3f) | 0x80)                  // variant 10
    loop (i in 1..<r.len()) b.push(r.at(i))
    Uuid(data: b.toList())
  }

  /// The all-zero UUID, `00000000-0000-0000-0000-000000000000`: the RFC's
  /// "nil" value, for a column that must hold a UUID and means "none".
  public static fun zero(): Uuid = Uuid(data: MutableList<u8>.repeat(0, 16).toList())

  /// Sixteen bytes as a UUID, whatever they say about their version — for
  /// reading an id out of a binary column. Panics unless there are exactly
  /// sixteen.
  @caller_location
  public static fun of(bytes: List<u8>): Uuid {
    if (bytes.len() != 16) panic("crypto.Uuid.of: a UUID is 16 bytes, got ${bytes.len()}")
    Uuid(data: bytes)
  }

  /// The UUID the text spells, or `null` when it does not. Accepts the
  /// canonical `8-4-4-4-12` form and the same 32 digits with no dashes, in
  /// either case; rejects everything else, braces and `urn:uuid:`
  /// prefixes included.
  public static fun parse(text: string): Uuid? {
    val digits = if (text.len() == 36) {
      if (text.byteAt(8) != 45 || text.byteAt(13) != 45 || text.byteAt(18) != 45 || text.byteAt(23) != 45) return null
      text.replace("-", "")
    } else if (text.len() == 32) {
      text
    } else {
      return null
    }
    if (digits.len() != 32) return null
    val out: MutableList<u8> = []
    var i = 0
    loop (i < 32) {
      val hi = nibble(digits.byteAt(i)) ?: return null
      val lo = nibble(digits.byteAt(i + 1)) ?: return null
      out.push((hi << 4) | lo)
      i = i + 2
    }
    Uuid(data: out.toList())
  }

  /// Byte `i` of the id. Every constructor makes sixteen bytes, and the
  /// readers ask for `i` in `0..<16`.
  fun byte(i: i64): u8 = this.data.at(i) ?: panic("Uuid: the data is always sixteen bytes")

  /// The sixteen bytes, big-endian as the RFC lays them out.
  public fun bytes(): List<u8> = this.data

  /// The version digit: 4 for `v4()`, 7 for `v7()`, 0 for `zero()`.
  public fun version(): i64 = ((this.byte(6) >> 4) & 0x0f).toI64()

  /// The milliseconds a version 7 id was made at, or `null` for any other
  /// version.
  public fun timestamp(): i64? {
    if (this.version() != 7) return null
    var v = 0
    var i = 0
    loop (i < 6) {
      v = (v << 8) | (this.byte(i).toI64())
      i = i + 1
    }
    v
  }

  /// True for the all-zero UUID.
  public fun isZero(): bool = this.data.all(b => b == 0)

  implement Display {
    fun toString(): string {
      val out = StringBuilder()
      var i = 0
      loop (i < 16) {
        if (i == 4 || i == 6 || i == 8 || i == 10) out.appendByte(45)
        val b = this.byte(i)
        out.appendByte(hexDigit(b >> 4))
        out.appendByte(hexDigit(b & 15))
        i = i + 1
      }
      out.toString()
    }
  }

  implement Comparable {
    /// By the bytes, so version 7 ids sort into creation order.
    fun compareTo(other: Uuid): Ordering {
      var i = 0
      loop (i < 16) {
        val a = this.byte(i)
        val b = other.byte(i)
        if (a < b) return Ordering.Less
        if (a > b) return Ordering.Greater
        i = i + 1
      }
      Ordering.Equal
    }
  }
}

/// A random UUID (version 4). `crypto.Uuid.v4()` said shorter.
public fun uuidV4(): Uuid = Uuid.v4()

/// A time-ordered UUID (version 7) — the one to reach for when the id ends
/// up in a database. `crypto.Uuid.v7()` said shorter.
public fun uuidV7(): Uuid = Uuid.v7()

// The v7 clock. Two ids from the same millisecond are told apart by the
// counter, and a millisecond that runs out of counter borrows the next
// one — so `v7()` never returns the same value twice and never goes
// backwards, whatever the system clock does.
// It is one clock for every task, whichever thread runs it (D66), so it
// sits behind a lock.
struct V7Clock {
  var millis:  i64
  var counter: i64

  fun tick(): (i64, i64) {
    val now = time.now().toMillis()
    if (now > this.millis) {
      this.millis = now
      // a random start in the lower half leaves 2048 increments and keeps
      // the counter from being a visible sequence
      this.counter = (randomU64() % 2048).wrapI64()
      return (this.millis, this.counter)
    }
    // the same millisecond, or a clock that went backwards: keep the
    // timestamp we already published and count
    this.counter = this.counter + 1
    if (this.counter > 4095) {
      this.millis = this.millis + 1
      this.counter = (randomU64() % 2048).wrapI64()
    }
    (this.millis, this.counter)
  }
}

val v7Clock = Mutex(value: V7Clock(millis: -1, counter: 0))

// the timestamp and counter of the next v7 id, taken together
fun nextTick(): (i64, i64) = v7Clock.withLock(c => c.tick())

fun hexDigit(nibble: u8): u8 = if (nibble < 10) 48 +% nibble else 87 +% nibble

fun nibble(b: u8): u8? {
  if (b >= 48 && b <= 57) return b -% 48
  if (b >= 97 && b <= 102) return b -% 87
  if (b >= 65 && b <= 70) return b -% 55
  null
}
