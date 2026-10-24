package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D156 at run time: a `val` handle outside a scope runs alongside its block
// and, when the block fails before awaiting it, is cancelled and joined
// before the error leaves (its `with`s closed, its work not done); a child
// that fails while the block waits elsewhere fails the block. `awaitAll`
// gives the values, and on the first failure cancels and joins its other
// inline launches. `workers:` never lets more children run than it says,
// with `on:` and without, in `gather` too. A lazy `mapConcurrent` keeps the
// source's order, has `workers` calls under way before the first result is
// used, and a `break` out of its loop cancels and joins the calls in
// flight. A `scope` gives its body's value. Debug and release, on 1, 2, 4
// and 8 threads.
func TestLessCeremony(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

error Boom { n: i64 }

struct Guard {
  count: Atomic<i64>
  implement Closeable {
    fun close() {
      val _ = this.count.add(1)
    }
  }
}

fun work(n: i64, ms: i64): i64 throws Boom {
  await sleep(Duration.millis(ms))
  if (n < 0) throw Boom(n)
  n * 10
}

fun fail(n: i64, ms: i64): i64 throws Boom {
  await sleep(Duration.millis(ms))
  throw Boom(n)
}

fun name(n: i64): string {
  await sleep(Duration.millis(2))
  "n$n"
}

fun guarded(n: i64, ms: i64, closed: Atomic<i64>, finished: Atomic<i64>): i64 {
  with g = Guard(count: closed)
  if (ms > 0) await sleep(Duration.millis(ms))
  val _ = finished.add(1)
  n
}

fun overlapped(): i64 throws Boom {
  val a = async work(2, 20)
  val b = async work(3, 10)
  await a + await b
}

fun abandoned(closed: Atomic<i64>, finished: Atomic<i64>): i64 throws Boom {
  val t = async guarded(1, 300, closed, finished)
  val x = try fail(-1, 5)
  x + await t
}

fun childFails(closed: Atomic<i64>): i64 throws Boom {
  val t = async fail(-2, 5)
  with g = Guard(count: closed)
  await sleep(Duration.millis(500))
  io.println("not reached")
  await t
}

fun allValues(): string throws Boom {
  val a = async work(1, 10)
  val (x, s) = awaitAll(a, async name(2))
  "$x $s"
}

fun allFail(closed: Atomic<i64>, finished: Atomic<i64>): i64 throws Boom {
  val (a, b) = awaitAll(async guarded(1, 300, closed, finished), async fail(-3, 5))
  a + b
}

fun allListed(): i64 throws Boom {
  scope {
    val tasks: MutableList<Task<i64>> = []
    loop (i in 1..4) tasks.push(async work(i, 5 - i))
    return awaitAll(tasks).sum()
  }
}

fun job(running: Atomic<i64>, seen: Channel<i64>) {
  seen.send(running.add(1))
  await sleep(Duration.millis(3))
  val _ = running.sub(1)
}

fun peakOf(seen: Channel<i64>): i64 {
  seen.close()
  var peak: i64 = 0
  loop {
    val n = await seen.recv() ?: break
    peak = peak.max(n)
  }
  peak
}

fun limitedPeak(): i64 {
  val running: Atomic<i64> = Atomic(value: 0)
  val seen = Channel<i64>(capacity: 64)
  scope(workers: 3) {
    loop (_ in 0..<12) async job(running, seen)
  }
  peakOf(seen)
}

fun placedPeak(cpu: Executor): i64 {
  val running: Atomic<i64> = Atomic(value: 0)
  val seen = Channel<i64>(capacity: 64)
  scope(on: cpu, workers: 2) {
    loop (_ in 0..<12) async job(running, seen)
  }
  peakOf(seen)
}

fun gatherPeak(): i64 {
  val running: Atomic<i64> = Atomic(value: 0)
  val seen = Channel<i64>(capacity: 64)
  val _ = gather(workers: 1) {
    async job(running, seen)
    async job(running, seen)
    async job(running, seen)
  }
  peakOf(seen)
}

fun delayed(x: i64): i64 {
  await sleep(Duration.millis(x * 3))
  x
}

fun lazyOrder(): string {
  var out = ""
  loop (s in [5, 1, 4, 2].iter().mapConcurrent(x => delayed(x), workers: 3)) {
    out = out + "$s "
  }
  out
}

fun counted(started: Atomic<i64>, x: i64): i64 {
  val _ = started.add(1)
  await sleep(Duration.millis(5))
  x
}

fun runAhead(): i64 {
  val started: Atomic<i64> = Atomic(value: 0)
  var whenFirst: i64 = 0
  loop (x in [1, 2, 3, 4, 5].iter().mapConcurrent(x => counted(started, x), workers: 3)) {
    if (x == 1) whenFirst = started.load()
  }
  whenFirst
}

fun tracked(n: i64, started: Atomic<i64>, closed: Atomic<i64>, finished: Atomic<i64>): i64 {
  val _ = started.add(1)
  guarded(n, if (n == 1) 0 else 300, closed, finished)
}

// how many calls had started when the loop broke depends on scheduling;
// that every one of them was closed when it ended does not. The first call
// ends at once, so the loop waits until a second one has started (the
// calls run ahead, so one will): the break then always cancels a call
// inside its 'with'. Without the wait a loaded machine could break before
// any other call ran, and a call cancelled before it starts never runs.
fun stopped(): string {
  val started: Atomic<i64> = Atomic(value: 0)
  val closed: Atomic<i64> = Atomic(value: 0)
  val finished: Atomic<i64> = Atomic(value: 0)
  var got: i64 = 0
  loop (x in [1, 2, 3, 4, 5].iter().mapConcurrent(x => tracked(x, started, closed, finished), workers: 3)) {
    got = x
    loop (started.load() < 2) await sleep(Duration.millis(1))
    break
  }
  val all = started.load() == closed.load() && started.load() >= 2
  "stopped at $got, every started call closed: $all, finished ${finished.load()}"
}

fun valued(): i64 throws Boom {
  val v = scope {
    val a = async work(4, 5)
    val b = async work(5, 1)
    await a + await b
  }
  v + 1
}

fun main() throws ThreadError {
  with cpu = try Executor.pool(threads: 2, name: "cpu")
  io.println("overlapped ${overlapped() catch (e) { e.n }}")
  val closed: Atomic<i64> = Atomic(value: 0)
  val finished: Atomic<i64> = Atomic(value: 0)
  val r = abandoned(closed, finished) catch (e) { e.n }
  io.println("abandoned $r, closed ${closed.load()} finished ${finished.load()}")
  val closed2: Atomic<i64> = Atomic(value: 0)
  io.println("childFails ${childFails(closed2) catch (e) { e.n }}, closed ${closed2.load()}")
  io.println("allValues ${allValues() catch (e) { "failed" }}")
  val closed3: Atomic<i64> = Atomic(value: 0)
  val finished3: Atomic<i64> = Atomic(value: 0)
  val r3 = allFail(closed3, finished3) catch (e) { e.n }
  io.println("allFail $r3, closed ${closed3.load()} finished ${finished3.load()}")
  io.println("allListed ${allListed() catch (e) { e.n }}")
  io.println("peaks ${limitedPeak()} ${placedPeak(cpu)} ${gatherPeak()}")
  io.println("lazy ${lazyOrder()}")
  io.println("runAhead ${runAhead()}")
  io.println(stopped())
  io.println("valued ${valued() catch (e) { e.n }}")
}
`
	want := "overlapped 50\n" +
		"abandoned -1, closed 1 finished 0\n" +
		"childFails -2, closed 1\n" +
		"allValues 10 n2\n" +
		"allFail -3, closed 1 finished 0\n" +
		"allListed 100\n" +
		"peaks 3 2 1\n" +
		"lazy 5 1 4 2 \n" +
		"runAhead 3\n" +
		"stopped at 1, every started call closed: true, finished 1\n" +
		"valued 91\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("ceremony-%v.exe", release))
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
