package driver

import (
	"strings"
	"testing"
)

// A failed `expect` shows what its condition was made of (D78, chapter 14):
// a call's receiver and arguments, the operand of `&&` that was false and
// both of `||`, `!`'s operand taken apart, both sides of `in`; a call on a
// side of a comparison adds its arguments. Each part is evaluated once, in
// order; the right operand of `&&` sees what the left one proved; long
// texts point at the difference; a message leads the report.
func TestExpectShowsParts(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `fun twice(n: i64): i64 => n * 2

fun within(base: string, file: string): bool => file.startsWith(base) && !file.contains("..")

fun parsePort(text: string): i64 => text.toInt() ?: 0

fun bump(counter: MutableList<i64>): i64 {
  counter.push(1)
  counter.len()
}

test "calls" {
  val base = "/srv"
  val file = "/srv/../etc"
  expect(within(base, file))
  val xs = [1, 2, 3]
  expect(xs.contains(4))
}

test "and" {
  val xs = [1, 2, 3]
  val n = 5
  expect(xs.len() == 3 && twice(n) == 11)
}

test "or" {
  val xs = [1, 2, 3]
  val n = 5
  expect(n > 10 || xs.isEmpty())
}

test "not and in" {
  val xs = [1, 2, 3]
  expect(!(xs.len() == 3))
  val x = 4
  expect(x in xs)
}

test "comparison" {
  val input = "x"
  expect(parsePort(input) > 0)
}

test "once" {
  val counter = MutableList<i64>()
  expect(bump(counter) == 2)
  expect(counter.len() == 1)
}

test "facts" {
  val s: string? = "abcd"
  expect(s != null && s.len() == 3)
}

test "difference" {
  val sentence = "the quick brown fox jumps over the lazy dog"
  expect(sentence == "the quick brown fox jumps over the lazy cat")
}

test "message" {
  val input = "x"
  expect(parsePort(input) > 0, "case ${input}")
}
`
	want := []string{
		"test calls ... FAILED",
		"  main.vs:15:3: expect(within(base, file))",
		`      base: "/srv"`,
		`      file: "/srv/../etc"`,
		"  main.vs:17:3: expect(xs.contains(4))",
		"      xs: [1, 2, 3]",
		"test and ... FAILED",
		"  main.vs:23:3: expect(xs.len() == 3 && twice(n) == 11)",
		"      twice(n): 10",
		"      n:        5",
		"test or ... FAILED",
		"  main.vs:29:3: expect(n > 10 || xs.isEmpty())",
		"      n:  5",
		"      xs: [1, 2, 3]",
		"test not and in ... FAILED",
		"  main.vs:34:3: expect(!(xs.len() == 3))",
		"      left:  3",
		"      right: 3",
		"      xs:    [1, 2, 3]",
		"  main.vs:36:3: expect(x in xs)",
		"      x:  4",
		"      xs: [1, 2, 3]",
		"test comparison ... FAILED",
		"  main.vs:41:3: expect(parsePort(input) > 0)",
		"      left:  0",
		"      right: 0",
		`      input: "x"`,
		// bump ran once: counter holds one element, and the second
		// expectation, which reads it, passes
		"test once ... FAILED",
		"  main.vs:46:3: expect(bump(counter) == 2)",
		"      left:    1",
		"      right:   2",
		"      counter: [1]",
		"test facts ... FAILED",
		"  main.vs:52:3: expect(s != null && s.len() == 3)",
		"      s.len(): 4",
		`      s:       "abcd"`,
		"test difference ... FAILED",
		`  main.vs:57:3: expect(sentence == "the quick brown fox jumps over the lazy cat")`,
		`      left:  "the quick brown fox jumps over the lazy dog"`,
		`      right: "the quick brown fox jumps over the lazy cat"`,
		"                                                      ^ differs from here",
		"test message ... FAILED",
		"  main.vs:62:3: case x",
		"      expect(parsePort(input) > 0)",
		"      left:  0",
		"      right: 0",
		`      input: "x"`,
		"",
		"0 passed, 9 failed: calls, and, or, not and in, comparison, once, facts, difference, message",
		"",
	}
	out, code := runTests(t, src, Options{})
	if out != strings.Join(want, "\n") || code != 1 {
		t.Fatalf("exit %d, output:\n%s\n--- want ---\n%s", code, out, strings.Join(want, "\n"))
	}
}
