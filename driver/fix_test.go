package driver

import (
	"os"
	"path/filepath"
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

impl Show for A {
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
  impl Show {
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
