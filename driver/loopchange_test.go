package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D102: a collection changed through a path the compiler cannot see — here
// a function handed the same list or map — makes the loop walking it panic
// when it next steps, at the loop's line, in both profiles. Replacing an
// element in place does not count.
func TestLoopOverChangedCollection(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	for name, c := range map[string]struct{ src, at string }{
		"list": {`use io

fun grow(ys: MutableList<i64>) {
  ys.push(99)
}

fun main() {
  val xs: MutableList<i64> = [1, 2, 3]
  loop (x in xs) {
    xs.set(0, x)
    if (x == 2) grow(xs)
  }
  io.println("not reached")
}
`, "main.vs:9:3"},
		"map": {`use io

fun forget(m: MutableMap<string, i64>) {
  m.remove("b")
}

fun main() {
  val m: MutableMap<string, i64> = ["a": 1, "b": 2, "c": 3]
  loop ((k, v) in m) {
    m.set(k, v + 1)
    if (k == "a") forget(m)
  }
  io.println("not reached")
}
`, "main.vs:9:3"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, release := range []bool{false, true} {
				exe := filepath.Join(dir, "loop.exe")
				if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
					t.Fatalf("build (release=%v) failed with exit %d", release, code)
				}
				var stderr strings.Builder
				cmd := exec.Command(exe)
				cmd.Stderr = &stderr
				out, err := cmd.Output()
				if err == nil {
					t.Fatalf("release=%v: the program should panic; it printed %q", release, out)
				}
				want := "panic: '" + map[string]string{"list": "xs", "map": "m"}[name] + "' changed while a loop walked it\n  at " + c.at
				if got := strings.ReplaceAll(stderr.String(), "\r\n", "\n"); !strings.HasPrefix(got, want) {
					t.Errorf("release=%v: stderr\n%s\nwant it to start with\n%s", release, got, want)
				}
			}
		})
	}
}
