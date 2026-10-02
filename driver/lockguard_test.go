package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D107: `with n = m.lock()` holds the Mutex to the end of the block and
// unlocks it on every way out — the end, a return, a thrown error, a panic
// — so the next lock succeeds; 8 tasks on up to 8 threads lose no
// increment; and locking the same Mutex again inside the region panics.
func TestLockGuard(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io { println }

error Oops { }

fun bump(counter: Mutex<i64>, times: i64) {
  loop (_ in 0..<times) {
    with n = counter.lock()
    *n += 1
  }
}

fun early(m: Mutex<i64>): i64 {
  with n = m.lock()
  if (*n >= 0) return *n
  -1
}

fun thrown(m: Mutex<i64>): i64 throws Oops {
  with (n = m.lock()) {
    *n += 1
    throw Oops()
  }
}

fun panics(m: Mutex<i64>) {
  with m.lock()
  [1].at(5) ?: panic("boom")
}

fun relock(m: Mutex<i64>) {
  with n = m.lock()
  println("  inner ${m.get()}")
}

fun main() {
  val counter = Mutex(value: 0)
  scope {
    loop (_ in 0..<8) {
      async bump(counter, 10000)
    }
  }
  println("total ${counter.get()}")
  println("early ${early(counter)}")
  println("thrown ${thrown(counter) is Err} ${counter.get()}")
  println("panic ${gather { async panics(counter) } is Err} ${counter.get()}")
  relock(counter)
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "total 80000\nearly 80000\nthrown true 80001\npanic true 80001\n"
	exe := filepath.Join(dir, "lock.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, threads := range []string{"1", "2", "4", "8"} {
		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(), "VELES_THREADS="+threads)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err == nil {
			t.Fatalf("threads=%s: the second lock did not panic\n%s", threads, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("threads=%s:\n got:\n%s\nwant:\n%s", threads, got, want)
		}
		if report := stderr.String(); strings.Contains(report, "Sanitizer") || strings.Contains(report, "runtime error:") {
			t.Fatalf("threads=%s: a sanitizer report:\n%s", threads, report)
		}
		if !strings.Contains(stderr.String(), "a Mutex was locked again while this task holds it") {
			t.Errorf("threads=%s: the panic does not say what happened:\n%s", threads, stderr.String())
		}
	}
}
