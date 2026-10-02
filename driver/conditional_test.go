package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D116: a function that suspends only through a `suspends` parameter is
// called plain when nothing it is given suspends — under `withLock`, in a
// lock region, in `init` — and as a coroutine when something does, with
// cancellation reaching inside the lambda. A parameter passed on, and one
// captured by a lambda passed on, keep the call plain. Also: a call through
// a suspending function value evaluates its arguments once (found
// 2026-10-02: they were evaluated twice). Debug and release, 1, 2, 4 and 8
// threads.
func TestConditionalSuspension(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

fun mapS<T, U, E>(xs: List<T>, f: fun(T): U suspends throws E): List<U> throws E {
  val out = MutableList<U>()
  loop (x in xs) {
    out.push(try f(x))
  }
  out.toList()
}

// passes its parameter on, and captures it in a lambda passed on
fun twice<T, E>(xs: List<T>, f: fun(T): T suspends throws E): List<T> throws E {
  val once = try mapS(xs, f)
  try mapS(once, x => try f(x))
}

fun slow(n: i64): i64 {
  await sleep(Duration.millis(1))
  n * 10
}

struct Totals {
  sum: i64
  init(xs: List<i64>) {
    this.sum = mapS(xs, x => x + 1).sum()
  }
}

fun next(counter: MutableList<i64>): i64 {
  counter.push(1)
  counter.len()
}

fun main() {
  val m = Mutex<i64>(value: 5)
  val added = m.withLock(p => mapS([1, 2, 3], x => x + *p))
  io.println("under withLock: $added")
  with (n = m.lock()) {
    io.println("in a lock region: ${twice([1, 2], x => x * *n)}")
  }
  io.println("suspending: ${mapS([1, 2, 3], x => slow(x))}")
  io.println("twice suspending: ${twice([1, 2], x => slow(x))}")
  io.println("init: ${Totals(xs: [1, 2, 3]).sum}")
  val waited = withTimeout(Duration.millis(20), () => mapS([1, 2, 3], x => {
    await sleep(Duration.seconds(10))
    x
  })) catch (e) { [] }
  io.println("cancelled inside the lambda: ${waited.isEmpty()}")
  val counter = MutableList<i64>()
  val f: fun(i64): i64 suspends = n => slow(n)
  io.println("through a value: ${f(next(counter))}, argument evaluated ${counter.len()} time(s)")
}
`
	want := "under withLock: [6, 7, 8]\n" +
		"in a lock region: [25, 50]\n" +
		"suspending: [10, 20, 30]\n" +
		"twice suspending: [100, 200]\n" +
		"init: 9\n" +
		"cancelled inside the lambda: true\n" +
		"through a value: 10, argument evaluated 1 time(s)\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("cond-%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		for _, threads := range []string{"1", "2", "4", "8"} {
			cmd := exec.Command(exe)
			cmd.Env = append(os.Environ(), "VELES_THREADS="+threads)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("release=%v threads=%s: %v\n%s", release, threads, err, out)
			}
			if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
				t.Errorf("release=%v threads=%s: output\n%s\nwant\n%s", release, threads, got, want)
			}
		}
	}
}
