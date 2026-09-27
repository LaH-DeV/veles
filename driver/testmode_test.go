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
	return runTestFiles(t, map[string]string{"main.vs": src}, opts)
}

// runTestFiles is runTests over a module of several files.
func runTestFiles(t *testing.T, files map[string]string, opts Options) (string, int) {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
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

test "addition" {
  if (1 + 1 != 2) throw Mismatch(expected: 2, actual: 1 + 1)
}

test "failing" {
  throw Mismatch(expected: 1, actual: 2)
}

test "panicking" {
  val xs = [1, 2]
  if (xs.len() > 1) panic("boom")
}

test "endless" {
  var n = 0
  loop {
    n += 1
    if (n < 0) break
  }
}

test "after the spin" { }
`

// The test vocabulary (D78): a soft expect records and the test goes on,
// each failure with its location, the expression and both sides; require
// and fail end the test with their own line, not a panic report.
func TestTestVocabulary(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `error RangeError { value: i64 }

fun parsePort(text: string): i64 throws RangeError {
  val n = text.toInt() ?: 0
  if (n > 65535) throw RangeError(value: n)
  n
}

fun half(n: i64): i64? = if (n % 2 == 0) n / 2 else null

test "passes" {
  expect(parsePort("80") ?? 0 == 80)
  expect(require(half(8)) == 4)
  expectThrows<RangeError>(() => parsePort("70000"))
  expectPanics(() => { val _ = [1].at(5) ?: panic("out of range") })
}

test "keeps going" {
  val name = "veles"
  expect(name.len() == 6)
  expect(name == "Veles")
  expect(name.isEmpty())
}

test "stops" {
  val p = require(parsePort("99999"))
  expect(p == 0)
}

test "expectations about failing" {
  expectThrows(() => parsePort("1"))
  expectPanics(() => { val _ = [1].at(0) ?: panic("no") })
  fail("the end")
}
`
	out, code := runTests(t, src, Options{})
	want := "test passes ... ok\n" +
		"test keeps going ... FAILED\n" +
		"  main.vs:20:3: expect(name.len() == 6)\n      left:  5\n      right: 6\n" +
		"  main.vs:21:3: expect(name == \"Veles\")\n      left:  \"veles\"\n      right: \"Veles\"\n" +
		"  main.vs:22:3: expect(name.isEmpty())\n" +
		"test stops ... FAILED\n" +
		"  main.vs:26:11: require(parsePort(\"99999\"))\n      threw: RangeError(value: 99999)\n" +
		"test expectations about failing ... FAILED\n" +
		"  main.vs:31:3: expectThrows(() => parsePort(\"1\"))\n      nothing was thrown\n" +
		"  main.vs:32:3: expectPanics(() => { val _ = [1].at(0) ?: panic(\"no\") })\n      it returned without panicking\n" +
		"  main.vs:33:3: the end\n" +
		"\n1 passed, 3 failed: keeps going, stops, expectations about failing\n"
	if out != want || code != 1 {
		t.Fatalf("exit %d, output:\n%s\n--- want ---\n%s", code, out, want)
	}
}

// Suites (D78): a `suite "name" { }` block and a `*.test.vs` file group their
// tests under a heading, nested and indented; tests outside any suite come
// first; the summary and --filter use the qualified name.
func TestTestSuites(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	files := map[string]string{
		"main.vs": `fun parseInt(text: string): i64? = text.toInt()

suite "parser" {
  test fun expectParses(text: string, want: i64) {
    expect(parseInt(text) == want)
  }

  test "parses ints" {
    expectParses("42", 42)
  }

  suite "rejects" {
    test "letters" {
      expect(parseInt("x4") == null)
    }
    test "empty input" {
      expectParses("", 0)
    }
  }
}

test "adds" {
  expect(1 + 1 == 2)
}
`,
		"config.test.vs": `test "reads a config" {
  expect(1 == 1)
}

suite "errors" {
  test "empty input" {
    fail("todo")
  }
}
`,
	}
	out, code := runTestFiles(t, files, Options{})
	want := "test adds ... ok\n" +
		"config\n" +
		"  test reads a config ... ok\n" +
		"  errors\n" +
		"    test empty input ... FAILED\n" +
		"      config.test.vs:7:5: todo\n" +
		"parser\n" +
		"  test parses ints ... ok\n" +
		"  rejects\n" +
		"    test letters ... ok\n" +
		"    test empty input ... FAILED\n" +
		"      main.vs:5:5: expect(parseInt(text) == want)\n" +
		"          left:  null\n" +
		"          right: 0\n" +
		"          called from main.vs:17:7\n" +
		"\n4 passed, 2 failed: config / errors / empty input, parser / rejects / empty input\n"
	if out != want || code != 1 {
		t.Fatalf("exit %d, output:\n%s\n--- want ---\n%s", code, out, want)
	}
	out, _ = runTestFiles(t, files, Options{Filter: "parser / rejects"})
	if !strings.HasPrefix(out, "parser\n  rejects\n    test letters ... ok\n") || !strings.HasSuffix(out, "; 4 filtered out\n") {
		t.Errorf("filtered by suite:\n%s", out)
	}
}

func TestTestRunner(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}

	// the summary names every failure; the exit code is 1
	out, code := runTests(t, testModeSrc, Options{Filter: "in"})
	want := "test failing ... FAILED: Mismatch(expected: 1, actual: 2)\n" +
		"test panicking ... FAILED: panic: boom\n  at main.vs:13:21\n" +
		"test after the spin ... ok\n" +
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
	if strings.Contains(out, "after the spin") {
		t.Errorf("a test ran after the timeout:\n%s", out)
	}
	if time.Since(start) > time.Minute {
		t.Errorf("the timed-out run took %v", time.Since(start))
	}
}

// A test's own output (io.println inside it) is kept while it runs: a
// passing test's is dropped, a failing test's is shown under its failure,
// so the report is never interleaved with it (D78 known limit, fixed).
func TestTestOutputCaptured(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

test "quiet when it passes" {
  io.println("you do not see this")
  expect(1 + 1 == 2)
}

suite "noisy" {
  test "shows what it printed" {
    io.print("partial ")
    io.println("line")
    io.eprintln("to stderr")
    expect(1 == 2)
    io.print("no newline at the end")
  }

  test "panics" {
    io.println("before the panic")
    val _ = [1].at(3) ?: panic("gone")
  }
}
`
	out, code := runTests(t, src, Options{})
	want := "test quiet when it passes ... ok\n" +
		"noisy\n" +
		"  test shows what it printed ... FAILED\n" +
		"    main.vs:13:5: expect(1 == 2)\n        left:  1\n        right: 2\n" +
		"    output:\n" +
		"      partial line\n" +
		"      to stderr\n" +
		"      no newline at the end\n" +
		"  test panics ... FAILED: panic: gone\n" +
		"    at main.vs:19:26\n" +
		"    output:\n" +
		"      before the panic\n" +
		"\n1 passed, 2 failed: noisy / shows what it printed, noisy / panics\n"
	if out != want || code != 1 {
		t.Fatalf("exit %d, output:\n%s\n--- want ---\n%s", code, out, want)
	}
}

// A failure inside a `test fun` helper names the line of the test that
// called it — through nested helpers, innermost first, and into a task the
// helper started — and a call that returned leaves nothing behind.
func TestTestHelperCallSites(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `test fun expectEven(n: i64) {
  expect(n % 2 == 0)
}

test fun expectAllEven(xs: List<i64>) {
  loop (x in xs) expectEven(x)
}

test fun doubled(n: i64): i64 {
  expect(n > 0)
  n * 2
}

test "helpers" {
  expectEven(4)
  expectAllEven([2, 3])
  expect(doubled(-1) == -2)
  expect(1 == 2)
}

test "in a task" {
  scope {
    val t = async doubled(0)
    val _ = await t
  }
}
`
	out, code := runTests(t, src, Options{})
	want := "test helpers ... FAILED\n" +
		"  main.vs:2:3: expect(n % 2 == 0)\n" +
		"      left:  1\n      right: 0\n" +
		"      called from main.vs:6:18\n" +
		"      called from main.vs:16:3\n" +
		"  main.vs:10:3: expect(n > 0)\n" +
		"      left:  -1\n      right: 0\n" +
		"      called from main.vs:17:10\n" +
		"  main.vs:18:3: expect(1 == 2)\n" +
		"      left:  1\n      right: 2\n" +
		"test in a task ... FAILED\n" +
		"  main.vs:10:3: expect(n > 0)\n" +
		"      left:  0\n      right: 0\n" +
		"\n0 passed, 2 failed: helpers, in a task\n"
	if out != want || code != 1 {
		t.Fatalf("exit %d, output:\n%s\n--- want ---\n%s", code, out, want)
	}
}
