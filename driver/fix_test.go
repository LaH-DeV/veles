package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `veles check --fix` applies the quick fixes the lints attach: a
// redundant `mut`, an unused binding, a dead `else` on a sealed subject,
// and a top-level impl that belongs in the struct body.
func TestCheckFix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := `use io

sealed trait S
struct A : S { }
struct B : S { }

trait Show {
  fun show(): string
}

implement Show for A {
  fun show(): string = "a"
}

fun main() {
  var xs: MutableList<i64> = mut [1]
  var m: MutableMap<string, i64> = mut  [:]
  val unused = xs.len()
  val s: S = A()
  val n = when (s) {
    is A => 1
    is B => 2
    else => 3
  }
  io.println("$n ${m.len()}")
}
`
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		t.Fatalf("check --fix: exit %d", code)
	}
	want := `use io

sealed trait S
struct A : S {
  implement Show {
    fun show(): string = "a"
  }
}
struct B : S { }

trait Show {
  fun show(): string
}

fun main() {
  var xs: MutableList<i64> = [1]
  var m: MutableMap<string, i64> = [:]
  val _ = xs.len()
  val s: S = A()
  val n = when (s) {
    is A => 1
    is B => 2
  }
  io.println("$n ${m.len()}")
}
`
	if data, _ := os.ReadFile(path); string(data) != want {
		t.Errorf("after --fix:\n%s\n--- want ---\n%s", data, want)
	}
	// a second pass finds nothing left to fix
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		t.Errorf("second check --fix: exit %d", code)
	}
	if data, _ := os.ReadFile(path); string(data) != want {
		t.Errorf("second --fix changed the file")
	}
}

// The removed index forms (D25, v0.24) are errors with a fix, so
// `check --fix` migrates a file even though it does not compile; nested
// forms take further passes because their edits overlap.
func TestCheckFixIndexing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := `use io

fun main() {
  val xs = [1, 2, 3]
  var ys: MutableList<i64> = [0, 0]
  var m: MutableMap<string, i64> = [:]
  val grid = [[1, 2], [3, 4]]
  ys[0] = xs[1]
  ys[1] += 2
  m["k"] = grid[1][0]
  io.println("$ys ${m["k"] ?: 0}")
}
`
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code == 0 {
		t.Fatalf("first check --fix: expected errors to remain (nested forms), got exit 0")
	}
	passes := 1
	for ; passes < 4; passes++ {
		if Run(Options{Path: dir, Mode: "check", Fix: true}) == 0 {
			break
		}
	}
	if passes == 4 {
		t.Fatalf("check --fix did not converge in %d passes", passes)
	}
	want := `use io

fun main() {
  val xs = [1, 2, 3]
  var ys: MutableList<i64> = [0, 0]
  var m: MutableMap<string, i64> = [:]
  val grid = [[1, 2], [3, 4]]
  ys.set(0, xs.at(1) ?: panic("TODO: say why this cannot fail"))
  ys.set(1, (ys.at(1) ?: panic("TODO: say why this cannot fail")) + 2)
  m.set("k", (grid.at(1) ?: panic("TODO: say why this cannot fail")).at(0) ?: panic("TODO: say why this cannot fail"))
  io.println("$ys ${m.get("k") ?: 0}")
}
`
	if data, _ := os.ReadFile(path); string(data) != want {
		t.Errorf("after --fix:\n%s\n--- want ---\n%s", data, want)
	}
}

// A fix attached to a *parse* error is applied too: the old receiver
// spelling (D65) stops the load, and one `check --fix` migrates it —
// interpolations included — so the second run is clean.
func TestCheckFixParseErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := "use io\n\nstruct C {\n  var n: i64 = 0\n  fun bump() {\n    self.n += 1\n  }\n  fun show(): string = \"${self.n} $self\"\n}\n\nfun main() {\n  val c = C()\n  c.bump()\n  io.println(c.show())\n}\n"
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code == 0 {
		t.Fatalf("first run: the old spelling is an error, got exit 0")
	}
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		data, _ := os.ReadFile(path)
		t.Fatalf("second run should be clean, exit %d:\n%s", code, data)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "self") || strings.Count(string(data), "this") != 3 {
		t.Errorf("after --fix:\n%s", data)
	}
}
