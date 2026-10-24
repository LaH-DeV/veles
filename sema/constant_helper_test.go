package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// A constant — an annotation's condition — runs in the compiler. A test
// helper the compiler can run may be called there (D208); one it cannot run
// says why, rather than naming the trace call the checker wraps a helper in
// (it once said "'enter' is not a constant").
func TestConstantCallsTestHelper(t *testing.T) {
	check := func(src string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "a.test.vs"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(dir, diags)
		if err != nil {
			t.Fatal(err)
		}
		CheckTests(pkg, diags, false)
		return errorsOf(diags)
	}
	if got := check("test fun twice(n: i64): i64 => n * 2\n\n@SkipIf(twice(2) == 4, \"why\")\ntest \"skipped\" {\n  expect(false)\n}\n"); got != "" {
		t.Errorf("a helper the compiler can run: got %q", got)
	}
	got := check("use io\n\ntest fun loud(): bool {\n  io.println(\"x\")\n  true\n}\n\n@SkipIf(loud(), \"why\")\ntest \"not skipped\" {\n  expect(true)\n}\n")
	want := "a constant cannot call 'loud': it is a test helper the compiler cannot run — it does I/O, starts a task, throws, or calls a function that is not a 'const fun' (D113, D208)"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "'enter'") {
		t.Errorf("the trace wrapper leaked into the message")
	}
}
