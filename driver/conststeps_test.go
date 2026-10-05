package driver

import (
	"os"
	"path/filepath"
	"testing"
)

// D113: a constant's evaluation has a step budget, 10 million by default and
// `--const-steps n` otherwise: a runaway `const fun` ends the build with an
// error naming the constant, and a bigger budget lets an honest long one run.
func TestConstStepsBudget(t *testing.T) {
	dir := t.TempDir()
	src := `const fun total(n: i64): i64 {
  var i = 0
  var sum = 0
  loop (i < n) {
    sum += i
    i += 1
  }
  sum
}

const TOTAL: i64 = total(100000)

fun main() {
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Run(Options{Path: dir, Mode: "check"}); code != 0 {
		t.Fatalf("the default budget should be enough for 100000 iterations: exit %d", code)
	}
	if code := Run(Options{Path: dir, Mode: "check", ConstSteps: 1000}); code == 0 {
		t.Fatal("a budget of 1000 steps should stop a loop of 100000 iterations")
	}
	if code := Run(Options{Path: dir, Mode: "check", ConstSteps: 100_000_000}); code != 0 {
		t.Fatalf("a bigger budget must not change the result: exit %d", code)
	}
}
