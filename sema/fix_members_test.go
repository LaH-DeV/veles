package sema

import (
	"strings"
	"testing"
)

// An implement missing methods is offered one fix, on each of its errors,
// that adds them all as stubs; the result checks, and runs into the stub's
// panic only if the method is called. A guess: `check --fix` leaves it.
func TestAddMissingMethodsFix(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{
			"inline, one there already",
			"trait Shape {\n  fun area(): f64\n  fun name(): string\n  fun scale(by: f64): Shape\n}\nstruct Sq {\n  side: f64\n  implement Shape {\n    fun area(): f64 => this.side * this.side\n  }\n}\nfun main() { }\n",
			"trait Shape {\n  fun area(): f64\n  fun name(): string\n  fun scale(by: f64): Shape\n}\nstruct Sq {\n  side: f64\n  implement Shape {\n    fun area(): f64 => this.side * this.side\n    fun name(): string => panic(\"'name' is not written yet\")\n    fun scale(by: f64): Shape => panic(\"'scale' is not written yet\")\n  }\n}\nfun main() { }\n",
		},
		{
			"top level, empty on one line",
			"trait Named {\n  fun name(): string\n}\nstruct Empty {\n  n: i64\n}\nimplement Named for Empty { }\nfun main() { }\n",
			"trait Named {\n  fun name(): string\n}\nstruct Empty {\n  n: i64\n}\nimplement Named for Empty {\n  fun name(): string => panic(\"'name' is not written yet\")\n}\nfun main() { }\n",
		},
		{
			"a sealed variant with no implement",
			"sealed trait Shape {\n  fun area(): i64\n  fun name(): string\n}\nstruct Sq : Shape {\n  side: i64\n}\nfun main() { }\n",
			"sealed trait Shape {\n  fun area(): i64\n  fun name(): string\n}\nstruct Sq : Shape {\n  side: i64\n  implement Shape {\n    fun area(): i64 => panic(\"'area' is not written yet\")\n    fun name(): string => panic(\"'name' is not written yet\")\n  }\n}\nfun main() { }\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := checkSource(t, tc.src)
			var fixed string
			n := 0
			for _, d := range diags.Items {
				if !strings.Contains(d.Message, "is missing method") && !strings.Contains(d.Message, "does not implement") {
					continue
				}
				n++
				if d.Fix == nil || !d.Fix.Guess || len(d.Fix.Edits) != 1 {
					t.Fatalf("%s: fix %+v", d.Message, d.Fix)
				}
				e := d.Fix.Edits[0]
				fixed = tc.src[:e.Span.Start] + e.NewText + tc.src[e.Span.End:]
			}
			if n == 0 {
				t.Fatalf("no missing-method error:\n%s", diags.Render())
			}
			if fixed != tc.want {
				t.Fatalf("fixed:\n%s\n--- want ---\n%s", fixed, tc.want)
			}
			if after := checkSource(t, fixed); after.HasErrors() {
				t.Errorf("the fixed source does not check:\n%s", after.Render())
			}
		})
	}
}
