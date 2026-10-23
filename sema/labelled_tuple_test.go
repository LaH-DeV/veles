package sema

import (
	"sort"
	"strings"
	"testing"
)

// D166: the fix for a tuple whose labels differ from the expected ones drops
// the value's labels — from a literal by deleting them, from a name by
// rebuilding it — and the fixed source compiles with the positions kept.
func TestLabelledTupleFix(t *testing.T) {
	cases := []struct{ value, fixed string }{
		{"(y: 2, x: 1)", "(2, 1)"},
		{"p", "(p.0, p.1)"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			src := prelude + "fun main() {\n  val p = (y: 2, x: 1)\n  val q: (x: i64, y: i64) = " + tc.value + "\n  io.println(\"${q.x} ${p.y}\")\n}\n"
			fixed := applyFixEdits(t, src, "the labels differ")
			if !strings.Contains(fixed, "val q: (x: i64, y: i64) = "+tc.fixed+"\n") {
				t.Fatalf("fix did not drop the labels:\n%s", fixed)
			}
			expectClean(t, fixed)
		})
	}
}

// applyFixEdits applies every edit of the fix on the diagnostic whose
// message contains want.
func applyFixEdits(t *testing.T, src, want string) string {
	t.Helper()
	for _, d := range checkSource(t, src).Items {
		if !strings.Contains(d.Message, want) {
			continue
		}
		if d.Fix == nil {
			t.Fatalf("no fix on %q", d.Message)
		}
		edits := d.Fix.Edits
		sort.Slice(edits, func(i, j int) bool { return edits[i].Span.Start > edits[j].Span.Start })
		for _, e := range edits {
			src = src[:e.Span.Start] + e.NewText + src[e.Span.End:]
		}
		return src
	}
	t.Fatalf("no diagnostic containing %q", want)
	return ""
}

// D166: one generic instance serves every labelling of its type arguments,
// and each caller sees its own labels in the result.
func TestLabelledTupleSharesInstance(t *testing.T) {
	src := prelude + "fun pick<T>(x: T): T => x\nfun main() {\n  val a = pick((lo: 1, hi: 2))\n  val b = pick((1, 2))\n  val c = pick((x: 1, y: 2))\n  io.println(\"${a.lo} ${b.0} ${c.y}\")\n}\n"
	expectClean(t, src)
}
