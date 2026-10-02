package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D114: an unchecked access still checks in a debug build, panicking
// "unchecked index … out of bounds" at the caller; a release build does
// not check. Release is run only with an index that stays inside the
// allocation (one past the end of a list with spare capacity, the byte
// after a string's last), so the test reads no foreign memory.
func TestUncheckedAccess(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	for name, c := range map[string]struct{ src, panic string }{
		"atUnchecked": {`use io

fun main() {
  val xs: MutableList<i64> = [1, 2, 3]
  xs.reserve(8)
  val n = xs.len()
  // SAFETY: deliberately one past the end, inside the reserved capacity
  val x = unsafe { xs.atUnchecked(n) }
  io.println("read ${x == x}")
}
`, "panic: unchecked index 3 out of bounds for list of length 3\n  at main.vs:8:20"},
		"setUnchecked": {`use io

fun main() {
  val xs: MutableList<i64> = [1, 2, 3]
  xs.reserve(8)
  val n = xs.len()
  // SAFETY: deliberately one past the end, inside the reserved capacity
  unsafe { xs.setUnchecked(n, 4) }
  io.println("wrote ${xs.len()}")
}
`, "panic: unchecked index 3 out of bounds for list of length 3\n  at main.vs:8:12"},
		"byteAtUnchecked": {`use io

fun main() {
  val s = "abc"
  // SAFETY: deliberately negative
  val b = unsafe { s.byteAtUnchecked(-1 + s.len() - 3) }
  io.println("read ${b == b}")
}
`, "panic: unchecked index -1 out of bounds for string of length 3\n  at main.vs:6:20"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "unchecked.exe")
			if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
				t.Fatalf("build failed with exit %d", code)
			}
			var stderr strings.Builder
			cmd := exec.Command(exe)
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err == nil {
				t.Fatalf("debug: the program should panic; it printed %q", out)
			}
			if got := strings.ReplaceAll(stderr.String(), "\r\n", "\n"); !strings.HasPrefix(got, c.panic) {
				t.Errorf("debug: stderr\n%s\nwant it to start with\n%s", got, c.panic)
			}
			if name == "byteAtUnchecked" {
				return // a negative offset is outside the string: not run in release
			}
			if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true}); code != 0 {
				t.Fatalf("release build failed with exit %d", code)
			}
			out, err = exec.Command(exe).Output()
			if err != nil {
				t.Fatalf("release: the access is not checked, so the program should end normally: %v", err)
			}
			if len(out) == 0 {
				t.Errorf("release: printed nothing")
			}
		})
	}
}
