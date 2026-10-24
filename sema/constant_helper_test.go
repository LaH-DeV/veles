package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// A constant — an annotation's condition — runs in the compiler, so it calls
// only a `const fun`. A test helper runs with the tests and cannot be one;
// the message says where it belongs instead of naming the trace call the
// checker wraps a helper in (it once said "'enter' is not a constant").
func TestConstantCallsTestHelper(t *testing.T) {
	dir := t.TempDir()
	src := "test fun twice(n: i64): i64 => n * 2\n\n@SkipIf(twice(2) == 4, \"a helper in the condition\")\ntest \"skipped by a helper\" {\n  expect(false)\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "a.test.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	CheckTests(pkg, diags, false)
	got := errorsOf(diags)
	want := "a constant cannot call 'twice': it is a test helper, which runs with the tests; move it to a module file as a 'const fun' (D113, D78)"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "'enter'") {
		t.Errorf("the trace wrapper leaked into the message")
	}
}
