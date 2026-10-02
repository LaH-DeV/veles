package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A program built with --sanitize runs to its end on Linux, threads and
// all. Each runtime thread gives its signal handler an alternate stack of
// its own; AddressSanitizer unmaps the one it installed when a thread ends,
// so the runtime must put that one back first — when it did not, every
// sanitized program aborted as its first thread ended ("failed to
// deallocate", 2026-10-02), and -sanitize runs reported nothing else.
// Always on where the sanitizer runtimes come with clang.
func TestSanitizedProgramEndsCleanly(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the sanitizer runtimes come with clang on Linux only")
	}
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io { println }

fun square(n: i64): i64 {
  await sleep(Duration.millis(1))
  n * n
}

fun main() {
  val (a, b) = gather {
    async square(3)
    async square(4)
  }
  println("${a.getOrDefault(0) + b.getOrDefault(0)}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "san")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: true}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, threads := range []string{"1", "8"} {
		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(), "VELES_THREADS="+threads)
		out, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "25" {
			t.Fatalf("threads=%s: %v\n%s", threads, err, out)
		}
	}
}
