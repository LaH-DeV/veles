package driver

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

// D208: a test the compiler can run runs there too, and the report says so;
// a test it cannot run (I/O) runs in the program only; a test helper the
// compiler can run may decide `@SkipIf`; and when the compiler's answer
// differs from the program's, the test fails naming the side.
func TestTestsRunInTheCompiler(t *testing.T) {
	files := map[string]string{
		"text.vs": "const fun slug(s: string): string => s.trim().toLower().replace(\" \", \"-\")\n",
		"text.test.vs": `use io

test fun twice(n: i64): i64 => n * 2

test "slug joins the words" {
  expect(slug("  Hello World ") == "hello-world")
}

test "printing runs in the program only" {
  io.println("from the program")
  expect(twice(2) == 4)
}

@SkipIf(twice(2) == 4, "a helper decided it")
test "skipped by a helper" {
  expect(false)
}
`,
	}
	out, code := runTestFiles(t, files, Options{})
	for _, want := range []string{
		"test slug joins the words ... ok (also in the compiler)",
		"test printing runs in the program only ... ok\n",
		"test skipped by a helper ... skipped: a helper decided it",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in:\n%s", want, out)
		}
	}
	if code != 0 {
		t.Errorf("exit %d, want 0:\n%s", code, out)
	}

	// the compiler's `replace` made to answer wrongly: the program passes,
	// the compiler does not — a disagreement, reported as a failure
	sema.SkewReplaceForTest = true
	defer func() { sema.SkewReplaceForTest = false }()
	out, code = runTestFiles(t, files, Options{})
	for _, want := range []string{
		"test slug joins the words ... FAILED in the compiler only: the program passed, so the compiler and the program disagree",
		"text.test.vs:6:3: expect(slug(\"  Hello World \") == \"hello-world\")",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("skewed: want %q in:\n%s", want, out)
		}
	}
	if code == 0 {
		t.Errorf("skewed: exit 0, want a failure:\n%s", out)
	}
}
