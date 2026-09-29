package sema

import "testing"

const rawPtrPrelude = prelude + `extern "C" {
  fun malloc(n: u64): *raw u8
  fun calloc(n: u64, size: u64): *raw ()
}
`

// D50: `p + n`, `p - n`, `p += n`, `p - q` and the orderings on a raw
// pointer, inside `unsafe`; the result types are the pointer, i64 and bool.
func TestRawPointerArithmetic(t *testing.T) {
	expectClean(t, rawPtrPrelude+`fun main() {
  // SAFETY: only addresses are computed and compared; nothing is read
  unsafe {
    val p = malloc(16).cast<*raw i64>()
    var q: *raw i64 = p + 2
    q -= 1
    q += (1).wrapU8()
    val n: i64 = q - p
    val before: bool = p < q && p <= q && q > p && q >= p
    io.println("$n $before")
  }
}`)
	cases := []struct{ name, body, want string }{
		{"outside unsafe", `val p = unsafe { malloc(8) }
  val q = p + 1`, "pointer arithmetic requires an 'unsafe' block (D50)"},
		{"integer first", `unsafe { val p = malloc(8); val q = 1 + p }`, "write the pointer first: 'p + 1'"},
		{"no multiplication", `unsafe { val p = malloc(8); val q = p * 2 }`, "operator '*' is not defined for '*raw u8'"},
		{"a float step", `unsafe { val p = malloc(8); val q = p + 1.5 }`, "moves by a whole number of elements, not by a 'f64'"},
		{"void pointer", `unsafe { val v = calloc(1, 8); val q = v + 1 }`, "cast it to '*raw u8' to move by bytes"},
		{"two pointer types", `unsafe { val p = malloc(8); val n = p - p.cast<*raw i64>() }`, "'p - q' counts elements between two pointers of one type"},
		{"order with a number", `unsafe { val p = malloc(8); val b = p < 3 }`, "both sides must be pointers of one type"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, rawPtrPrelude+"fun main() {\n  // SAFETY: probing a diagnostic\n  "+c.body+"\n}", c.want)
		})
	}
}
