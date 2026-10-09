// `const fun` (D113): functions the compiler runs. A constant that calls one
// is computed while the program is compiled and laid out read-only in the
// binary, so nothing below is built when the program starts. The same
// functions are ordinary functions at run time.
use io

// the CRC-32 lookup table, from the polynomial
const fun crc32Table(): List<u32> {
  val table: MutableList<u32> = []
  loop (n in 0..<256) {
    var c = n.wrapU32()
    loop (_ in 0..<8) {
      c = if (c & 1 == 1) 0xEDB88320 ^ (c >> 1) else c >> 1
    }
    table.push(c)
  }
  table.toList()
}

const CRC: List<u32> = crc32Table()

fun crc32(text: string): u32 {
  var c: u32 = 0xFFFFFFFF
  loop (b in text.bytes()) {
    val index = ((c ^ b.toU32()) & 0xFF).toI64()
    c = (CRC.at(index) ?: 0) ^ (c >> 8)
  }
  c ^ 0xFFFFFFFF
}

// a lookup table from a list of names: the position is the value
const fun positions(names: List<string>): Map<string, i64> {
  val out: MutableMap<string, i64> = [:]
  loop ((i, name) in names.enumerate()) out.set(name, i)
  out.toMap()
}

const MONTHS: Map<string, i64> = positions(["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"])

// the primes below n, by the sieve of Eratosthenes
const fun primesBelow(n: i64): List<i64> {
  val composite = MutableList<bool>.repeat(false, n)
  val primes: MutableList<i64> = []
  loop (i in 2..<n) {
    if (composite.at(i) ?: true) continue
    primes.push(i)
    var j = i * i
    loop (j < n) {
      composite.set(j, true)
      j += i
    }
  }
  primes.toList()
}

const PRIMES: List<i64> = primesBelow(50)

static assert(PRIMES.len() == 15, "there are fifteen primes below fifty")

// text built from pieces, with the library's own string functions
const fun banner(title: string, width: i64): string {
  val rule = "=".repeat(width)
  val sb = StringBuilder()
  sb.appendLine(rule)
  sb.appendLine(title.toUpper().padStart((width + title.len()) / 2).padEnd(width))
  sb.append(rule)
  sb.toString()
}

const BANNER: string = banner("const fun", 24)

// a value type, built and changed with a method
struct Version {
  var major: i64
  var minor: i64

  const fun bumped(): Version {
    var next = this
    next.minor += 1
    next
  }

  const fun text(): string => "v${this.major}.${this.minor}"
}

const NEXT: Version = Version(major: 1, minor: 9).bumped().bumped()

fun main() {
  io.println(BANNER)
  io.println("crc32(\"veles\") = ${crc32("veles")}, crc32(\"\") = ${crc32("")}")
  io.println("table entry 255 = ${CRC.at(255)}")
  io.println("mar is month ${(MONTHS.get("mar") ?: -1) + 1} of ${MONTHS.len()}")
  io.println("primes below 50: ${PRIMES}")
  io.println("next version: ${NEXT.text()}")
}
