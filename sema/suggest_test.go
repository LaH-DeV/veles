package sema

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

func TestDidYouMean(t *testing.T) {
	cases := []struct {
		name  string
		cands []string
		want  string
	}{
		{"countr", []string{"counter", "count", "main"}, "counter"},
		{"lenght", []string{"length", "len"}, "length"},
		{"pritnln", []string{"println", "print"}, "println"},       // a swap is one edit
		{"printn", []string{"print", "println"}, "println"},        // tie: the longer shared subsequence
		{"Println", []string{"println", "print"}, "println"},       // case alone
		{"toUppercase", []string{"toUpper", "toLower"}, "toUpper"}, // a long prefix
		{"size", []string{"len", "isEmpty"}, "len"},                // another language's name
		{"push_back", []string{"push", "pop"}, "push"},
		{"lenght", []string{"len", "isEmpty"}, "len"}, // a typo of another language's name
		{"size", []string{"isEmpty"}, ""},             // ... only when the type has it
		{"xs", []string{"ys"}, ""},                    // too short to guess
		{"banana", []string{"orange"}, ""},            // a different word
		{"lenght", []string{"$lenght", "__lenght"}, ""},
	}
	for _, c := range cases {
		if got := didYouMean(c.name, c.cands); got != c.want {
			t.Errorf("didYouMean(%q, %v) = %q, want %q", c.name, c.cands, got, c.want)
		}
	}
}

// Every "unknown" diagnostic says what was probably meant, and attaches it
// as a guessed fix (offered by the editor, never applied by `--fix`). A
// member written as the other kind (a field called, a method read) is told
// which it is instead.
func TestTypoSuggestions(t *testing.T) {
	cases := []struct{ src, msg, fix string }{
		{`fun main() { val counter = 1; io.println("${countr}") }`, "unknown name 'countr'; did you mean 'counter'?", "counter"},
		{`fun total(): i64 => 1
fun main() { io.println("${totl()}") }`, "unknown function 'totl'; did you mean 'total'?", "total"},
		{`fun main() { io.printn("x") }`, "module 'io' has no declaration 'printn'; did you mean 'io.println'?", "println"},
		{`fun main() { io.println("${[1].lenght()}") }`, "no method 'lenght' on type 'List<i64>'; did you mean 'len'?", "len"},
		{`fun main() { val xs: MutableList<i64> = []; xs.append(1) }`, "did you mean 'push'?", "push"},
		{`fun main() { io.println("x".toUpperCase()) }`, "did you mean 'toUpper'?", "toUpper"},
		{`fun main() { val m = [1: 2]; io.println("${m.has(1)}") }`, "did you mean 'containsKey'?", "containsKey"},
		{`struct P { x: i64
  fun length(): i64 => this.x }
fun main() { io.println("${P(x: 1).lenght()}") }`, "no method 'lenght' on type 'P'; did you mean 'length'?", "length"},
		{`struct P { total: i64 }
fun main() { io.println("${P(total: 1).totl}") }`, "has no field 'totl'; did you mean 'total'?", "total"},
		{`fun main() { io.println("${Duration.millis(1).toMilis()}") }`, "did you mean 'toMillis'?", "toMillis"},
		{`struct P { x: i64 }
fun main() { io.println("${P(x: 1).x()}") }`, "'x' is a field, not a method: drop the '()'", ""},
		{`fun main() { io.println("${[1].len}") }`, "'len' is a method: call it, 'len()'", ""},
	}
	for _, c := range cases {
		diags := checkSource(t, prelude+c.src)
		found := false
		for _, d := range diags.Items {
			if d.Severity != source.Error || !strings.Contains(d.Message, c.msg) {
				continue
			}
			found = true
			switch {
			case c.fix == "" && d.Fix != nil:
				t.Errorf("%q: expected no fix, got %+v", c.src, d.Fix)
			case c.fix != "" && (d.Fix == nil || !d.Fix.Guess || len(d.Fix.Edits) != 1 || d.Fix.Edits[0].NewText != c.fix):
				t.Errorf("%q: expected a guessed fix to %q, got %+v", c.src, c.fix, d.Fix)
			}
		}
		if !found {
			t.Errorf("%q: expected %q, got:\n%s", c.src, c.msg, diags.Render())
		}
	}
}

// A private member is not suggested from outside its type: taking the hint
// would only trade this error for "is private".
func TestTypoSuggestionsRespectPrivacy(t *testing.T) {
	diags := checkSource(t, prelude+`struct P {
  x: i64
  private fun secret(): i64 => this.x
}
fun main() { io.println("${P(x: 1).secrt()}") }`)
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "secrt") && strings.Contains(d.Message, "did you mean") {
			t.Errorf("suggested a private method: %s", d.Message)
		}
	}
}

// A call that failed checks its arguments only loosely; a lambda there has
// no expected type, and "cannot infer the type of lambda parameter" on top
// of the real error is noise.
func TestFailedCallLambdaQuiet(t *testing.T) {
	diags := checkSource(t, prelude+`fun main() { io.println("${[1].reduce(0, (a, b) => a + b)}") }`)
	for _, d := range diags.Items {
		if strings.Contains(d.Message, "cannot infer") {
			t.Errorf("cascading error: %s", d.Message)
		}
	}
	if !diags.HasErrors() {
		t.Errorf("expected the unknown method")
	}
}
