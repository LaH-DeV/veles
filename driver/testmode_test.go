package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runTests runs `veles test` on src and returns the program's output and
// the exit code; stdout is redirected to a file for the duration.
func runTests(t *testing.T, src string, opts Options) (string, int) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "out.txt")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = out
	opts.Path, opts.Mode = dir, "test"
	code := Run(opts)
	os.Stdout = saved
	out.Close()
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n"), code
}

const testModeSrc = `error Mismatch { expected: i64, actual: i64 }

@test
fun addition() throws Mismatch {
  if (1 + 1 != 2) throw Mismatch(expected: 2, actual: 1 + 1)
}

@test
fun failing() throws Mismatch {
  throw Mismatch(expected: 1, actual: 2)
}

@test
fun panicking() {
  val xs = [1, 2]
  if (xs.len() > 1) panic("boom")
}

@test
fun endless() {
  var n = 0
  loop {
    n += 1
    if (n < 0) break
  }
}

@test
fun afterTheSpin() { }
`

func TestTestRunner(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}

	// the summary names every failure; the exit code is 1
	out, code := runTests(t, testModeSrc, Options{Filter: "in"})
	want := "test failing ... FAILED: Mismatch(expected: 1, actual: 2)\n" +
		"test panicking ... FAILED: panic: boom\n  at main.vs:16:21\n" +
		"test afterTheSpin ... ok\n" +
		"\n1 passed, 2 failed: failing, panicking; 2 filtered out\n"
	if out != want || code != 1 {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}

	// --filter keeps matching names and counts the rest
	out, code = runTests(t, testModeSrc, Options{Filter: "addition"})
	if out != "test addition ... ok\n\n1 passed, 0 failed; 4 filtered out\n" || code != 0 {
		t.Errorf("filtered run: exit %d\n%s", code, out)
	}
	out, code = runTests(t, testModeSrc, Options{Filter: "fail"})
	if !strings.HasSuffix(out, "\n0 passed, 1 failed: failing; 4 filtered out\n") || code != 1 {
		t.Errorf("filtered failing run: exit %d\n%s", code, out)
	}

	// a filter that matches nothing is an error, not a green run
	if _, code = runTests(t, testModeSrc, Options{Filter: "nothing"}); code != 1 {
		t.Errorf("an empty filter match exited %d", code)
	}

	// a test that never ends is reported and ends the run
	start := time.Now()
	out, code = runTests(t, testModeSrc, Options{TestTimeout: 300 * time.Millisecond})
	if !strings.Contains(out, "test endless ... FAILED: timed out after 300ms\n\ntimed out: endless; 1 test after it did not run\n") || code != 1 {
		t.Errorf("timeout: exit %d\n%s", code, out)
	}
	if strings.Contains(out, "afterTheSpin") {
		t.Errorf("a test ran after the timeout:\n%s", out)
	}
	if time.Since(start) > time.Minute {
		t.Errorf("the timed-out run took %v", time.Since(start))
	}
}
