package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D100: a statement-form `with` closes on every way out of its block, in
// reverse order, and a with-task is cancelled at the end of its block (then
// joined), fails the block fast, and leaves nothing to stop once awaited.
// Run in both profiles: the lowering is shared, the cleanup paths are not.
func TestWithStatementExits(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io { println }

struct Res {
  name: string

  implement Closeable {
    fun close() { println("  close ${this.name}") }
  }
}

fun open(name: string): Res {
  println("  open $name")
  Res(name)
}

error Oops { }

fun fails(): i64 throws Oops => throw Oops()

fun end(): i64 {
  with a = open("a")
  with b = open("b")
  println("  body")
  a.name.len() + b.name.len()   // computed before the closes
}

fun early(): i64 {
  with a = open("a")
  if (a.name == "a") return 1
  with b = open("b")
  2
}

fun loops() {
  loop (i in 0..<4) {
    with l = open("l$i")
    if (i == 1) continue
    if (i == 2) break
    println("  iteration $i")
  }
}

fun failedTry(): i64 throws Oops {
  with a = open("a")
  val n = try fails()
  println("  not reached")
  n
}

fun thrown(): i64 throws Oops {
  with a = open("a")
  with b = open("b")
  throw Oops()
}

fun panics(): i64 {
  with a = open("a")
  await sleep(Duration.zero)
  [1].at(5) ?: panic("boom")
}

fun holds(name: string) {
  with r = open(name)
  await sleep(Duration.seconds(30))
  println("  never printed")
}

fun cancelled() {
  with outer = open("outer")
  with t = async holds("in-task")
  await sleep(Duration.millis(50))
  println("  block ends")
}

fun failsLater(): i64 throws Oops {
  await sleep(Duration.millis(20))
  throw Oops()
}

fun failFast(): i64 throws Oops {
  with a = open("a")
  with t = async failsLater()
  await sleep(Duration.seconds(30))
  println("  never printed")
  0
}

fun answer(): i64 {
  await sleep(Duration.millis(10))
  42
}

fun awaited(): i64 {
  with t = async answer()
  with r = open("r")
  await t
}

fun main() {
  println("end ${end()}")
  println("return ${early()}")
  println("loop")
  loops()
  println("failed try ${failedTry() is Err}")
  println("throw ${thrown() is Err}")
  println("panic ${gather { async panics() } is Err}")
  println("cancellation")
  cancelled()
  println("fail fast ${failFast() is Err}")
  println("awaited ${awaited()}")
}
`
	want := `  open a
  open b
  body
  close b
  close a
end 2
  open a
  close a
return 1
loop
  open l0
  iteration 0
  close l0
  open l1
  close l1
  open l2
  close l2
  open a
  close a
failed try true
  open a
  open b
  close b
  close a
throw true
  open a
  close a
panic true
cancellation
  open outer
  open in-task
  block ends
  close in-task
  close outer
  open a
  close a
fail fast true
  open r
  close r
awaited 42
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "with.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v:\n got:\n%s\nwant:\n%s", release, got, want)
		}
	}
}
