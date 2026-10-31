package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D196, end to end: the window of an immutable List shares the storage, the
// window of a MutableList is a copy, and a window keeps the buffer it looks
// into alive after the list it was cut from is gone. Run in a debug and a
// release build.
func TestListWindows(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

fun numbers(n: i64): List<i64> {
  val xs: MutableList<i64> = []
  loop (i in 0..<n) xs.push(i * 3)
  xs.freeze()
}

// a window out of a list that is gone once this returns
fun middle(): List<i64> {
  val all = numbers(100000)
  all.slice(40000, 40005)
}

fun churn(): i64 {
  var total: i64 = 0
  loop (round in 0..<200) {
    val junk: MutableList<i64> = []
    loop (i in 0..<5000) junk.push(i + round)
    total += junk.len()
  }
  total
}

// a List method a program writes may be called on a MutableList, whose own
// header it then sees as "this": its windows are copies
extend<T> List<T> {
  fun firstTwo(): List<T> => this.slice(0, 2)
}

fun main() {
  val all = numbers(20)
  val head = all.slice(0, 4)
  val tail = all.slice(16, 99)
  val inner = tail.slice(1, 3)
  io.println("$head $tail $inner")
  io.println("${all.slice(-5, 2)} ${all.slice(7, 3)} ${all.slice(5, 5).isEmpty()} ${all.take(3)} ${all.drop(17)}")

  // a mutable list's window is a copy: later changes do not show in it
  val m: MutableList<i64> = [1, 2, 3, 4]
  val copy = m.slice(0, 2)
  m.set(0, 99)
  m.push(5)
  io.println("$copy $m")

  // the same for the List methods, called on a mutable list
  val mm: MutableList<i64> = [1, 2, 3, 4, 5, 6]
  val t = mm.take(3)
  val d = mm.drop(4)
  val c = mm.chunked(2)
  val w = mm.windowed(5)
  val f2 = mm.firstTwo()
  mm.set(0, 99)
  mm.set(5, 77)
  io.println("$t $d $c $w $f2")

  // a mutable copy of a window is its own list
  val grown = head.toMutable()
  grown.push(1000)
  io.println("$head $grown")

  // the buffer outlives the list it came from
  val kept = middle()
  io.println("${churn()}")
  io.println("$kept ${kept.sorted().reversed()}")

  // windows of windows, and the adapters over them
  val chunks = all.chunked(6)
  val sums = chunks.map(c => c.fold(0, (a, b) => a + b))
  io.println("${chunks.len()} $sums ${all.windowed(18).len()} ${all.windowed(18).at(2)}")
}
`
	want := "[0, 3, 6, 9] [48, 51, 54, 57] [51, 54]\n" +
		"[0, 3] [] true [0, 3, 6] [51, 54, 57]\n" +
		"[1, 2] [99, 2, 3, 4, 5]\n" +
		"[1, 2, 3] [5, 6] [[1, 2], [3, 4], [5, 6]] [[1, 2, 3, 4, 5], [2, 3, 4, 5, 6]] [1, 2]\n" +
		"[0, 3, 6, 9] [0, 3, 6, 9, 1000]\n" +
		"1000000\n" +
		"[120000, 120003, 120006, 120009, 120012] [120012, 120009, 120006, 120003, 120000]\n" +
		"4 [45, 153, 261, 111] 3 [6, 9, 12, 15, 18, 21, 24, 27, 30, 33, 36, 39, 42, 45, 48, 51, 54, 57]\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "views.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: stdout\n%s\nwant\n%s", release, got, want)
		}
	}
}
