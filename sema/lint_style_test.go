package sema

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// warnings returns the messages of the lint warnings a program draws.
func lintMessages(t *testing.T, src, lint string) []string {
	t.Helper()
	diags := checkSource(t, src)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors:\n%s", diags.Render())
	}
	var out []string
	for _, d := range diags.Items {
		if d.Severity == source.Warning && strings.Contains(d.Message, lint) {
			out = append(out, d.Message)
		}
	}
	return out
}

func TestRedundantDiscardLint(t *testing.T) {
	const marker = "discards a value"
	if got := lintMessages(t, prelude+`
fun main() {
  val seen: MutableSet<i64> = MutableSet<i64>()
  val _ = seen.add(1)
  io.println("${seen.len()}")
}`, marker); len(got) != 1 {
		t.Errorf("want one warning for a bool discarded, got %v", got)
	}
	if got := lintMessages(t, prelude+`
struct P { x: i64 }
fun make(): P => P(x: 1)
fun main() {
  val _ = make()
  io.println("done")
}`, marker); len(got) != 0 {
		t.Errorf("a struct value is not a plain discard, got %v", got)
	}
}

func TestElseIfLadderLint(t *testing.T) {
	const marker = "conditions in a row"
	ladder := func(body string) string {
		return prelude + "\nfun f(x: i64): string {\n" + body + "\n}\nfun main() { io.println(f(3)) }"
	}
	if got := lintMessages(t, ladder(`  if (x < 1) {
    "a"
  } else if (x < 2) {
    "b"
  } else if (x < 5) {
    "c"
  } else {
    "d"
  }`), marker); len(got) != 1 {
		t.Errorf("want one warning for a ladder of three conditions, got %v", got)
	}
	if got := lintMessages(t, ladder(`  if (x < 1) {
    "a"
  } else if (x < 2) {
    "b"
  } else {
    "d"
  }`), marker); len(got) != 0 {
		t.Errorf("two conditions are fine, got %v", got)
	}
	if got := lintMessages(t, ladder(`  if (x < 1) {
    "a"
  } else {
    if (x < 2) {
      "b"
    } else {
      if (x < 5) {
        "c"
      } else {
        "d"
      }
    }
  }`), marker); len(got) != 0 {
		t.Errorf("nested blocks are not a ladder, got %v", got)
	}
}

func TestCounterLoopLints(t *testing.T) {
	const asRange, skips = "write 'loop (", "this 'continue' skips"
	prog := func(body string) string {
		return prelude + "\nfun f(n: i64): i64 {\n  var total = 0\n" + body + "\n  total\n}\nfun main() { io.println(\"${f(3)}\") }"
	}
	if got := lintMessages(t, prog(`  var i = 0
  loop (i < n) {
    total += i
    i += 1
  }`), asRange); len(got) != 1 {
		t.Errorf("want the range-loop warning, got %v", got)
	}
	for name, body := range map[string]string{
		"counter used after": `  var i = 0
  loop (i < n) {
    total += i
    i += 1
  }
  total += i`,
		"stepping by two": `  var i = 0
  loop (i < n) {
    total += i
    i += 2
  }`,
		"bound changes": `  var i = 0
  var m = n
  loop (i < m) {
    total += i
    m -= 1
    i += 1
  }`,
		"counter changed elsewhere": `  var i = 0
  loop (i < n) {
    if (i == 1) i += 1
    total += i
    i += 1
  }`,
	} {
		if got := lintMessages(t, prog(body), asRange); len(got) != 0 {
			t.Errorf("%s: no warning expected, got %v", name, got)
		}
	}
	if got := lintMessages(t, prog(`  var i = 0
  loop (i < n) {
    if (i == 1) continue
    total += i
    i += 1
  }`), skips); len(got) != 1 {
		t.Errorf("want the continue warning, got %v", got)
	}
	if got := lintMessages(t, prog(`  var i = 0
  loop (i < n) {
    i += 1
    if (i == 1) continue
    total += i
  }`), skips); len(got) != 0 {
		t.Errorf("a continue after the step is fine, got %v", got)
	}
}
