package parser

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
)

// D166: tuple types and literals may label every element; `(x: 1)` is a
// labelled one-element tuple, not a parenthesised expression.
func TestLabelledTuples(t *testing.T) {
	cases := map[string]string{
		"val r: (quotient: i64, remainder: i64) = (quotient: 1, remainder: 2)": "(val r: (quotient: i64, remainder: i64) = (tuple quotient: 1 remainder: 2))",
		"val one: (only: i64) = (only: 1)":                                     "(val one: (only: i64) = (tuple only: 1))",
		"val plain: (i64, i64) = (1, 2)":                                       "(val plain: (i64, i64) = (tuple 1 2))",
	}
	for src, want := range cases {
		f, diags := parse(t, src+"\n")
		if diags.HasErrors() {
			t.Errorf("%q: %s", src, diags.Render())
			continue
		}
		if got := strings.TrimSpace(ast.Dump(f)); got != want {
			t.Errorf("%q:\n got %s\nwant %s", src, got, want)
		}
	}
}

func TestLabelledTupleErrors(t *testing.T) {
	cases := map[string]string{
		"val p = (x: 1, 2)\n":            "labels every element or none",
		"val p: (i64, y: i64) = p\n":     "labels every element or none",
		"val p = (x: 1, x: 2)\n":         "label 'x' is used twice",
		"fun f(): (a: i64, a: i64) => p": "label 'a' is used twice",
	}
	for src, want := range cases {
		_, diags := parse(t, src+"\n")
		if !strings.Contains(diags.Render(), want) {
			t.Errorf("%q: want an error containing %q, got:\n%s", src, want, diags.Render())
		}
	}
}

// `(…):` starts a lambda only when a return type and `=>` follow; a tuple
// key of a map literal is followed by its value.
func TestTupleKeyIsNotALambda(t *testing.T) {
	cases := map[string]string{
		`val m = [(1, 2): "a", (3, 4): "b"]`:          `(map [(tuple 1 2): (str "a")] [(tuple 3 4): (str "b")])`,
		`val m = [(1, 2): (3, 4)]`:                    "(map [(tuple 1 2): (tuple 3 4)])",
		`val m = [(1, 2): (y: i64): i64 => y]`:        "(map [(tuple 1 2): (lambda (y: i64): i64 y)])",
		`val f = (x: i64): i64 => x`:                  "(lambda",
		`val f = (x: i64): Map<string, i64> => [:]`:   "(lambda",
		`val f = (x: i64): fun(i64): i64 => (y => y)`: "(lambda",
		`val f = (x: i64): Array<u8, 4> => a`:         "(lambda",
	}
	for src, want := range cases {
		f, diags := parse(t, src+"\n")
		if diags.HasErrors() {
			t.Errorf("%q: %s", src, diags.Render())
			continue
		}
		if got := ast.Dump(f); !strings.Contains(got, want) {
			t.Errorf("%q:\n got %s\nwant it to contain %s", src, got, want)
		}
	}
}
