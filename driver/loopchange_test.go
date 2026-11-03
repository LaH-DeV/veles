package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D102, D231 addendum: a loop walks a list or a map as it was when the loop
// began. A change the body makes — here through a function handed the
// variable's address — lands in the variable, and the loop goes on over
// the original elements, in both profiles.
func TestLoopOverChangedCollection(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	for name, c := range map[string]struct{ src, want string }{
		"list": {`use io

fun grow(ys: *List<i64>) {
  ys.push(99)
}

fun main() {
  val xs: List<i64> = [1, 2, 3]
  var seen = 0
  loop (x in xs) {
    xs.set(0, x)
    if (x == 2) grow(&xs)
    seen += 1
  }
  io.println("$seen $xs")
}
`, "3 [3, 2, 3, 99]\n"},
		"map": {`use io

fun forget(m: *Map<string, i64>) {
  m.remove("b")
}

fun main() {
  val m: Map<string, i64> = ["a": 1, "b": 2, "c": 3]
  loop ((k, v) in m) {
    m.set(k, v + 1)
    if (k == "a") forget(&m)
  }
  io.println("$m")
}
`, "{a: 2, c: 4, b: 3}\n"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, release := range profiles() {
				exe := filepath.Join(dir, "loop.exe")
				if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
					t.Fatalf("build (release=%v) failed with exit %d", release, code)
				}
				out, err := exec.Command(exe).Output()
				if err != nil {
					t.Fatalf("release=%v: %v", release, err)
				}
				if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != c.want {
					t.Errorf("release=%v: got %q, want %q", release, got, c.want)
				}
			}
		})
	}
}

// D102 addendum: a loop over a list reached through a pointer walks it as
// it was too — the list is lent for the loop, so a change made meanwhile by
// a function that holds the pointer copies it — where it used to panic. A
// by-reference loop still walks the list itself and panics.
func TestLoopOverListReachedThroughAPointer(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Index {
  var tags: List<string> = []
}

fun addDerived(index: *Index, tag: string) {
  if (tag.startsWith("lang:")) index.tags.push("has-lang")
}

fun rewrite(index: *Index) {
  index.tags.set(0, "x")
}

fun note(index: *Index) {
  index.tags.push("seen")
}

fun main() {
  val index = Index(tags: ["lang:go", "draft"])
  val p = &index
  loop (t in p.tags) {
    addDerived(p, t)
  }
  var seen: List<string> = []
  loop (t in p.tags) {
    rewrite(p)
    seen.push(t)
  }
  io.println("${index.tags} $seen")
  loop (&t in p.tags) {
    *t = t.toUpper()
    note(p)
  }
  io.println("not reached")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "loop.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		var stderr strings.Builder
		cmd := exec.Command(exe)
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err == nil {
			t.Fatalf("release=%v: the by-reference loop should panic; it printed %q", release, out)
		}
		want := "[x, draft, has-lang] [lang:go, draft, has-lang]\n"
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got %q, want %q", release, got, want)
		}
		if !strings.Contains(stderr.String(), "'p.tags' changed while a loop walked it") {
			t.Errorf("release=%v: stderr %s", release, stderr.String())
		}
	}
}
