package source

import (
	"strings"
	"testing"
)

// After the diagnostics, one "see:" line per family, in the order the
// families first came up (D79).
func TestRenderNamesEachFamilyOnce(t *testing.T) {
	f := NewFile("main.vs", "val a: u8 = 300\nval b: i8 = 200\nval c = cout\n")
	d := &Diagnostics{}
	d.Errorf(Span{File: f, Start: 12, End: 15}, "literal 300 does not fit in 'u8', which holds 0..255")
	d.Errorf(Span{File: f, Start: 28, End: 31}, "literal 200 does not fit in 'i8', which holds -128..127")
	d.Errorf(Span{File: f, Start: 40, End: 44}, "unknown name 'cout'")
	got := d.Render()
	want := "see: veles explain numbers\nsee: veles explain unknown-name\n"
	if !strings.HasSuffix(got, want) || strings.Count(got, "see: ") != 2 {
		t.Errorf("got:\n%s", got)
	}
}

func TestFamilyOfRealMessages(t *testing.T) {
	for msg, want := range map[string]string{
		"module 'io' has no declaration 'prinln'; did you mean 'io.println'?":                           "modules",
		"'helper' is private to module 'geo'; declare it 'public' there to use it from here (M5)":       "private-to-module",
		"'%' is not defined for floats; 'x.mod(y)' is the remainder, in 0.0..<|y|":                      "operators",
		"comparing a non-nullable 'i64' with null is always false; remove the test":                     "nullable",
		"this loop never repeats: every path through its body leaves it (break, return or throw); drop": "control-flow",
		"no sensible family would claim this":                                                           "",
	} {
		if got := FamilyOf(msg); got != want {
			t.Errorf("%q: family %q, want %q", msg, got, want)
		}
	}
}
