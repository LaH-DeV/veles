package sema

import "testing"

// A cycle of constants with declared types is reported whichever of them
// the checker reaches first (it used to be dropped silently when the first
// one declared was checked before the others had been looked at).
func TestConstCycleIsAlwaysReported(t *testing.T) {
	for name, src := range map[string]string{
		"two":          "const A: i64 = B + 1\nconst B: i64 = A\nfun main() {\n}\n",
		"two, flipped": "const B: i64 = A\nconst A: i64 = B + 1\nfun main() {\n}\n",
		"three":        "const A: i64 = B\nconst B: i64 = C\nconst C: i64 = A + 1\nfun main() {\n}\n",
		"itself":       "const A: i64 = A + 1\nfun main() {\n}\n",
		"with io":      "use io\nconst A: i64 = B + 1\nconst B: i64 = A\nfun main() {\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			expectError(t, src, "is defined in terms of itself")
		})
	}
}
