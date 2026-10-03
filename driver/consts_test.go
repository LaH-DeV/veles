package driver

import (
	"strings"
	"testing"
)

// D113: constants are computed by the compiler, so what they compute must
// be what the program would compute — the same text for a float, the same
// hash for a key (a constant Map/Set is laid out with the compiler's hashes
// and searched with the runtime's) — in a debug and a release build. And
// module-level values are computed in the order they read each other.
func TestConstantsMatchRunTime(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `enum Kind: u8 { Fun = 1, Val = 2, When = 3 }

struct Key { name: string, n: i64 }

const SHORT: f64 = 0.1 + 0.2
const BIG = 1e20
const EDGE = 123456789012345.0
const HUGE = 1e15
const TINY = 0.00015
const TINIER = 1.5e-7
const NEG_ZERO = -0.0
const THIRD: f32 = 1.0 / 3.0
const ODD: f32 = 16777217.0
const MAXED = 1.7976931348623157e308 * 10.0
const TEXTS: List<string> = ["${SHORT}", "${BIG}", "${EDGE}", "${HUGE}", "${TINY}", "${TINIER}", "${NEG_ZERO}", "${MAXED}", "${THIRD}", "${ODD}"]

const BY_NAME: Map<string, Kind> = ["fun": Kind.Fun, "val": Kind.Val, "when": Kind.When, "": Kind.Fun]
const BY_INT: Map<i64, string> = [0: "zero", -1: "minus one", 9223372036854775807: "max", -9223372036854775808: "min"]
const BY_SMALL: Map<i8, bool> = [-128: true, -1: false, 127: true]
const BY_WORD: Map<u64, i64> = [18446744073709551615: 1, 0: 2]
const BY_KIND: Map<Kind, i64> = [Kind.Fun: 1, Kind.When: 3]
const BY_PAIR: Set<(i64, string)> = [(1, "a"), (2, "b")]
const BY_KEY: Set<Key> = [Key(name: "a", n: 1), Key(name: "b", n: -2)]
const BY_MAYBE: Set<i64?> = [null, 0, 5]
const BY_FLOAT: Set<f64> = [0.5, -2.25, 1e300]
const BY_F32: Set<f32> = [0.5, 3.0]
const BY_BOOL: Map<bool, string> = [true: "yes", false: "no"]
const BY_LIST: Set<List<i64>> = [[], [1, 2], [3]]
const BIG_SET: Set<i64> = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33]

val late: i64 = early() + 1
fun early(): i64 = base * 2
val base: i64 = "four".len()

test "a constant's text is the run time's" {
  val f = [SHORT, BIG, EDGE, HUGE, TINY, TINIER, NEG_ZERO, MAXED]
  loop (i in 0..<f.len()) {
    expect(TEXTS.at(i) == "${f.at(i)}")
  }
  expect(TEXTS.at(8) == "${THIRD}")
  expect(TEXTS.at(9) == "${ODD}")
}

test "every key of a constant table is found" {
  loop ((k, _) in BY_NAME.toMutable()) { expect(BY_NAME.containsKey(k)) }
  loop ((k, _) in BY_INT.toMutable()) { expect(BY_INT.containsKey(k)) }
  loop ((k, _) in BY_SMALL.toMutable()) { expect(BY_SMALL.containsKey(k)) }
  loop ((k, _) in BY_WORD.toMutable()) { expect(BY_WORD.containsKey(k)) }
  loop ((k, _) in BY_KIND.toMutable()) { expect(BY_KIND.containsKey(k)) }
  loop ((k, _) in BY_BOOL.toMutable()) { expect(BY_BOOL.containsKey(k)) }
  loop (k in BY_PAIR.toMutable()) { expect(BY_PAIR.contains(k)) }
  loop (k in BY_KEY.toMutable()) { expect(BY_KEY.contains(k)) }
  loop (k in BY_MAYBE.toMutable()) { expect(BY_MAYBE.contains(k)) }
  loop (k in BY_FLOAT.toMutable()) { expect(BY_FLOAT.contains(k)) }
  loop (k in BY_F32.toMutable()) { expect(BY_F32.contains(k)) }
  loop (k in BY_LIST.toMutable()) { expect(BY_LIST.contains(k)) }
  loop (k in 0..<34) { expect(BIG_SET.contains(k)) }
  expect(!BIG_SET.contains(34))
  expect(BY_NAME.get("val") == Kind.Val && BY_NAME.get("loop") == null)
  expect(BY_INT.get(-9223372036854775808) == "min")
  expect(BY_KEY.contains(Key(name: "b", n: -2)) && !BY_KEY.contains(Key(name: "b", n: 2)))
}

test "a constant table is copied, never changed" {
  val copy = BY_INT.toMutable()
  copy.set(7, "seven")
  expect(copy.len() == 5 && BY_INT.len() == 4)
  expect(BY_INT == BY_INT && BY_LIST.len() == 3)
}

test "globals are computed in the order they read each other" {
  expect(late == 9)
}
`
	for _, release := range []bool{false, true} {
		out, code := runTests(t, src, Options{Release: release})
		if code != 0 || !strings.Contains(out, "4 passed") {
			t.Fatalf("release=%v: exit %d, output:\n%s", release, code, out)
		}
	}
}
