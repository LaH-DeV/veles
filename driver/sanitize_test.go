package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// `--sanitize` links AddressSanitizer: a C call that writes past a block
// stops the program with a report, where a plain build carries on with
// memory corrupted. Skipped where clang has no sanitizer runtimes (MSYS2
// without compiler-rt).
func TestSanitizeCatchesOverflow(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

extern "C" {
  fun malloc(n: u64): *raw u8
  fun memset(p: *raw u8, c: i32, n: u64): *raw u8
  fun free(p: *raw u8)
}

fun main() {
  // SAFETY: deliberately wrong: clears one byte past a 4-byte block
  unsafe {
    val p = malloc(4)
    memset(p, 0, 5)
    free(p)
  }
  io.println("carried on")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "overflow.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: true}); code != 0 {
		t.Skip("no sanitizer runtimes for this clang")
	}
	out, err := exec.Command(exe).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "AddressSanitizer: heap-buffer-overflow") || strings.Contains(string(out), "carried on") {
		t.Fatalf("expected a heap-buffer-overflow report, got %v:\n%s", err, out)
	}
}
