// D113 part 3: `static assert(cond, "why")` at module level or in a body,
// checked at compile time; a failure quotes the reason and the values of
// the constants the condition names. In a generic body it is checked per
// instance, so it can assert what a type argument implements (D117).
use io

enum Wire: u8 { V1 = 1, V2 = 2 }

const HEADER_SIZE: i64 = 16
const FIELDS: List<string> = ["id", "len"]
const VERSION = Wire.V2

static assert(HEADER_SIZE == 16, "the wire header is 16 bytes")
static assert(HEADER_SIZE == 8 * 3, "the wire header is ${HEADER_SIZE * 2} bytes") // error: static assert failed: the wire header is 32 bytes (HEADER_SIZE = 16)
static assert(FIELDS.len() == 3 && VERSION == Wire.V1, "three fields, version 1") // error: static assert failed: three fields, version 1 (FIELDS = ["id", "len"], VERSION = Wire.V2)

fun sorts<T>(xs: List<T>): bool {
  static assert(T implements Comparable, "sorts needs ordered elements") // error: static assert failed: sorts needs ordered elements
  return xs.len() > 1
}

struct Unordered { n: i64 }

fun main() {
  static assert(FIELDS.at(0) == "id", "the id comes first")
  val n = 3
  static assert(n == 3, "a local is not a constant") // error: 'n' is not a constant (D113)
  io.println("${sorts([1, 2])} ${sorts([Unordered(n: 1)])}")
}
