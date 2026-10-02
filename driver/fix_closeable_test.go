package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D115: the fixes for a Closeable never closed — `val` becomes `with`, and a
// discarded result is bound by `with _ =` — leave a program that checks
// without the warning.
func TestCheckFixNeverClosed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := `use io

struct Res {
  n: i64
  implement Closeable {
    fun close() {
      io.println("closed")
    }
  }
}

fun make(n: i64): Res = Res(n)

fun main() {
  val r = make(1)
  make(2)
  io.println("${r.n}")
}
`
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		t.Fatalf("check --fix: exit %d", code)
	}
	got, _ := os.ReadFile(path)
	want := strings.Replace(strings.Replace(src, "  val r = make(1)", "  with r = make(1)", 1), "  make(2)", "  with _ = make(2)", 1)
	if string(got) != want {
		t.Fatalf("after --fix:\n%s\nwant:\n%s", got, want)
	}
	if code := Run(Options{Path: dir, Mode: "check"}); code != 0 {
		t.Fatalf("check after the fix: exit %d", code)
	}
}
