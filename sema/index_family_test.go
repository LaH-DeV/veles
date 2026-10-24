package sema

import (
	"strings"
	"testing"
)

// applyOnlyFix applies the one fix of the diagnostic whose message contains
// want, and returns the fixed source.
func applyOnlyFix(t *testing.T, src, want string) string {
	t.Helper()
	for _, d := range checkSource(t, src).Items {
		if !strings.Contains(d.Message, want) {
			continue
		}
		if d.Fix == nil || len(d.Fix.Edits) != 1 {
			t.Fatalf("no single-edit fix on %q", d.Message)
		}
		e := d.Fix.Edits[0]
		return src[:e.Span.Start] + e.NewText + src[e.Span.End:]
	}
	t.Fatalf("no diagnostic containing %q", want)
	return ""
}

// D165: the fix for a comparison with the old -1 answer tests for null,
// and the fixed source compiles.
func TestIndexSentinelFix(t *testing.T) {
	for _, cond := range []string{`s.indexOf("=") >= 0`, `s.indexOf("=") < 0`, `s.lastIndexOf("=") == -1`, `[1, 2].indexOf(2) != -1`} {
		t.Run(cond, func(t *testing.T) {
			src := prelude + "fun main() {\n  val s = \"a=b\"\n  if (" + cond + ") io.println(\"x\")\n}\n"
			expectClean(t, applyOnlyFix(t, src, "never -1 (D165)"))
		})
	}
}

// D163: when a one-parameter type gains a second parameter, the positional
// call's fix names the first field, and the fixed source compiles.
func TestPositionalConstructorFixWhenTypeGrows(t *testing.T) {
	src := prelude + "struct Label {\n  text: string\n  bold: bool = false\n}\nfun main() { io.println(Label(\"hi\").text) }\n"
	fixed := applyOnlyFix(t, src, "construct 'Label' by field name")
	if !strings.Contains(fixed, `Label(text: "hi")`) {
		t.Fatalf("fix did not name the field:\n%s", fixed)
	}
	expectClean(t, fixed)
}
