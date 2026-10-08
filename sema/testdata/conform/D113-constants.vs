// D113 parts 1 and 2: a constant is evaluated when the program is compiled.
// What would fail at run time fails the build, in every profile; what the
// compiler cannot evaluate is refused, naming it.
use io
enum Level: u8 { Low = 1, High = 2 }

struct Point { x: i64 }

struct Label {
  text: string
  implement Display { fun toString(): string => "<" + this.text + ">" }
}

struct Money {
  cents: i64
  implement Equatable { fun equals(other: Money): bool => this.cents == other.cents }
  implement Hashable { fun hash(): i64 => this.cents }
}

fun three(): i64 => 3
val runtime: i64 = 3

const KB: i64 = 1024
const MB = KB * 1024
const MIN8: i8 = -128
const ORIGIN = Point(x: 1)
const LABEL = Label(text: "a")
const PRICE = Money(cents: 100)
const PRIMES: List<i64> = [2, 3, 5]
const LEVELS: Map<string, Level> = ["low": Level.Low, "high": Level.High]
const NAME = "veles-${MB}"
const THIRD: i64 = PRIMES.at(2)
const LAST: i64 = PRIMES.at(-1)

const OVER: i64 = 9223372036854775807 + 1 // error: constant overflow: 9223372036854775807 + 1 is 9223372036854775808, outside 'i64'
const WRAPS: i64 = 9223372036854775807 +% 1
const NEG: i8 = -MIN8 // error: constant overflow: -(-128) does not fit 'i8'
const DIV = KB / (KB - 1024) // error: division by zero in a constant: 1024 / 0
const SHIFT: u8 = 1 << 8 // error: a shift by 8 in a constant is outside 0..7, the width of 'u8'
const CALL = three() // error: a constant cannot call 'three'
const BYTE = NAME.toF64() // error: a constant cannot call 'toF64': it is not a 'const fun'
const SHOWN = "${ORIGIN} ${Level.High} ${PRIMES}"
const SHOWN_OWN = "${LABEL}" // error: interpolating a 'Label' runs its own 'toString', which is not a 'const fun'
const SAME = PRICE == PRICE // error: '==' here is not a constant expression: 'Money' hashes or compares with its own 'hash'/'equals'
const BAD_KEYS: Set<Money> = [PRICE] // error: a constant cannot have type 'Set<Money>': 'Money' hashes or compares with its own
const GROWS: MutableList<i64> = [1] // error: a constant cannot have type 'MutableList<i64>': a 'MutableList' can change and a constant cannot; use 'List'
const FN: fun(): i64 = three // error: a constant cannot have type 'fun(): i64': a function value is not a constant
const READ = runtime + 1 // error: 'runtime' is a 'val', computed at run time, not a constant
const OUT = PRIMES.at(3) // error: index 3 is out of range for a constant list of length 3, so 'at' is always null (D113)
const NULL_NEVER: u8? = KB.toU8() // error: 'KB' is the constant 1024, which does not fit 'u8', so this is always null

const LOOP_A: i64 = LOOP_B + 1
const LOOP_B: i64 = LOOP_A // error: the constant 'LOOP_A' is defined in terms of itself

fun main() {
  val near = PRIMES.at(1) + LEVELS.len() + THIRD + LAST
  val far = PRIMES.at(-4) // error: index -4 is out of range for a constant list of length 3, so 'at' is always null (D113)
  val both = near + WRAPS
  val pick = if (NAME.len() > 3) both else 0
  io.println("${pick} ${far}")
}

