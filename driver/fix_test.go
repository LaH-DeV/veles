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
  fun show(): string => "a"
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
    fun show(): string => "a"
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
// forms need further passes because their edits overlap, and one run
// makes them all (it checks again while a pass still finds a fix).
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
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		data, _ := os.ReadFile(path)
		t.Fatalf("one check --fix should settle the nested forms too, exit %d:\n%s", code, data)
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
// interpolations included — and checks the result clean.
func TestCheckFixParseErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := "use io\n\nstruct C {\n  var n: i64 = 0\n  fun bump() {\n    self.n += 1\n  }\n  fun show(): string => \"${self.n} $self\"\n}\n\nfun main() {\n  val c = C()\n  c.bump()\n  io.println(c.show())\n}\n"
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		data, _ := os.ReadFile(path)
		t.Fatalf("check --fix should leave it clean, exit %d:\n%s", code, data)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "self") || strings.Count(string(data), "this") != 3 {
		t.Errorf("after --fix:\n%s", data)
	}
}

// A name the prelude writes for a module (D75) is qualified and its module
// imported by one `check --fix`, the `use` above the first declaration's
// doc comment, once however many names needed it.
func TestCheckFixQualifiesModuleNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := "/// A module doc.\n\n/// Counts things.\nfun count(v: Value, w: Value): i64 => 1\n\nfun main() {\n  val d = Depth(limit: 3)\n  io.println(\"${count(VNull(), VNull())} ${d.limit}\")\n}\n"
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code == 0 {
		// `io` is not a module name the fix may guess: the run ends with it
		// still unknown, and says so
		t.Fatalf("expected the unknown 'io' to remain")
	}
	data, _ := os.ReadFile(path)
	want := "/// A module doc.\n\nuse codec\nuse recursion\n\n/// Counts things.\nfun count(v: codec.Value, w: codec.Value): i64 => 1\n\nfun main() {\n  val d = recursion.Depth(limit: 3)\n  io.println(\"${count(codec.VNull(), codec.VNull())} ${d.limit}\")\n}\n"
	if string(data) != want {
		t.Errorf("after --fix:\n%s\n--- want ---\n%s", data, want)
	}
}

// A typo's nearest name is a guess: the editor offers it, but `check --fix`
// never rewrites a program to what it only probably meant. Nor does it
// throw away the binding the typo meant: `counter` is not "never used"
// (whose fix renames it to `_`) when `countr` was its use.
func TestCheckFixLeavesGuesses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := "use io\n\nfun main() {\n  val counter = [1, 2]\n  io.printn(\"${countr.size()}\")\n}\n"
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code == 0 {
		t.Fatalf("expected the misspelt names to remain errors")
	}
	if data, _ := os.ReadFile(path); string(data) != src {
		t.Errorf("--fix applied a guess:\n%s", data)
	}
}

// The mistakes a newcomer to Codable makes, each answered by a fix: a
// struct that is not Encodable (or Comparable) gets `implement X`, derived;
// `try` in front of `??` is dropped. One `check --fix` and it runs.
func TestCheckFixDerivesMissingImplements(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	src := "use io, json\n\nstruct Point {\n  x: i64\n  y: i64\n}\n\nstruct Tag { name: string }\n\nfun main() {\n  io.println(try json.encode(Point(x: 1, y: 2)) ?? \"\")\n  val tags = [Tag(name: \"b\"), Tag(name: \"a\")].sorted()\n  io.println(json.encode(tags) ?? \"\")\n}\n"
	os.WriteFile(path, []byte(src), 0o644)
	if code := Run(Options{Path: dir, Mode: "check", Fix: true}); code != 0 {
		data, _ := os.ReadFile(path)
		t.Fatalf("check --fix did not settle it, exit %d:\n%s", code, data)
	}
	data, _ := os.ReadFile(path)
	want := "use io\nuse json\n\nstruct Point {\n  x: i64\n  y: i64\n  implement Encodable\n}\n\nstruct Tag {\n  name: string\n  implement Comparable\n  implement Encodable\n}\n\nfun main() {\n  io.println(json.encode(Point(x: 1, y: 2)) ?? \"\")\n  val tags = [Tag(name: \"b\"), Tag(name: \"a\")].sorted()\n  io.println(json.encode(tags) ?? \"\")\n}\n"
	if string(data) != want {
		t.Errorf("after --fix:\n%s\n--- want ---\n%s", data, want)
	}
}
