package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// D197: `cede` offers the executor a turn and goes on at the next
// statement. Nothing here assumes which task runs next: a peer is seen to
// run by a flag only it sets, waited for by recursion so that no loop's
// back edge (D145) can give the thread up in cede's place; affinity by the
// OS thread's id; cancellation by deadlines that pass in a lock region,
// where nothing is interrupted, so the cede after it is the first point
// that may see them. Run twice in a row, case 4 also pins that a yield
// abandoned by a cancellation is forgotten (a stale TB_YIELDED made the
// next cede return at once, unchecked).
func TestCede(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	tid := "fun GetCurrentThreadId(): u32\n}\n\n// SAFETY: asks the OS which thread runs\nfun tid(): i64 => unsafe { GetCurrentThreadId().toI64() }"
	if runtime.GOOS != "windows" {
		tid = "fun pthread_self(): u64\n}\n\n// SAFETY: asks the OS which thread runs\nfun tid(): i64 => unsafe { pthread_self().wrapI64() }"
	}
	src := `// D197: cede offers the executor a turn.
use io
use time

extern "C" {
  ` + tid + `

struct Res {
  closes: Atomic<i64>
  implement Closeable {
    fun close() {
      this.closes.add(1)
    }
  }
}

// --- a lone task: it goes on, once, at the next statement
fun level(n: i64, trail: MutableList<i64>): i64 {
  val mine = n * 10
  cede
  if (n == 0) return mine
  val below = level(n - 1, trail)
  cede
  trail.push(n)
  mine + below
}

fun churn(rounds: i64): string {
  val keep: MutableList<string> = []
  loop (i in 0..<rounds) {
    keep.push("item $i")
    val junk = MutableList<i64>.make(64, j => j * i)
    if (junk.len() != 64) panic("churn: a list of 64 has ${junk.len()}")
    cede
  }
  "${keep.len()} ${keep.at(rounds - 1) ?: "none"}"
}

// --- a peer runs: no loop here, so only cede can give the thread up
fun until(flag: Atomic<bool>, spins: i64): i64 {
  if (flag.load()) return spins
  cede
  until(flag, spins + 1)
}

fun setter(flag: Atomic<bool>) {
  flag.store(true)
}

fun peerRuns(): bool {
  val flag = Atomic(value: false)
  var spins: i64 = -1
  scope {
    async setter(flag)
    spins = until(flag, 0)
  }
  spins >= 0
}

// --- the executor and its thread are kept
fun stays(rounds: i64): bool {
  val first = tid()
  var same = true
  loop (_ in 0..<rounds) {
    cede
    if (tid() != first) same = false
  }
  same
}

fun bothStay(): bool {
  var a = false
  var b = false
  scope {
    val x = async stays(500)
    val y = async stays(500)
    a = await x
    b = await y
  }
  a && b
}

// --- cancellation
fun spinFor(ms: i64) {
  val sw = time.Stopwatch.start()
  loop (sw.elapsed().toMillis() < ms) {}
}

// the deadline passes in the lock region, where nothing is interrupted:
// the cede after it is the first point that may see it
fun pending(m: Mutex<i64>, closes: Atomic<i64>, after: Atomic<bool>) {
  with r = Res(closes)
  m.withLock(p => spinFor(30))
  cede
  after.store(true)
}

fun cedeForever(closes: Atomic<i64>) {
  with r = Res(closes)
  loop { cede }
}

error Broke { }

fun breaks() throws Broke {
  cede
  throw Broke()
}

fun failFast(closes: Atomic<i64>) throws Broke {
  with r = Res(closes)
  scope {
    async breaks()
    loop { cede }
  }
}

fun abandon(closes: Atomic<i64>): i64 {
  scope {
    async cedeForever(closes)
    cede
    return 7
  }
}

// a close() that cedes is cleanup: the deadline does not cut it short
fun helperCedes(steps: Atomic<i64>) {
  cede
  steps.add(1)
}

struct Conn {
  steps: Atomic<i64>
  implement Closeable {
    fun close() suspends {
      loop (_ in 0..<3) {
        cede
        helperCedes(this.steps)
      }
    }
  }
}

fun closing(m: Mutex<i64>, steps: Atomic<i64>, after: Atomic<bool>) {
  with c = Conn(steps)
  m.withLock(p => spinFor(30))
  cede
  after.store(true)
}

fun outcome<T>(r: Result<T, Timeout>): string => when (r) {
  is Ok  => "finished"
  is Err => "timed out"
}

fun main() {
  io.println("1. a lone task:")
  val trail: MutableList<i64> = []
  io.println("  ${level(5, trail)} $trail")
  io.println("  ${churn(2000)}")
  cede
  cede;
  io.println("  resumed")

  io.println("2. a peer runs while this task cedes: ${peerRuns()}")

  io.println("3. the executor's thread is kept:")
  with th = try Executor.thread(name: "fixed") catch (e) { panic("no thread: ${e.message()}") }
  io.println("  one task: ${th.run(() => stays(1000))}")
  io.println("  two tasks: ${th.run(() => bothStay())}")

  io.println("4. a cancellation pending before cede:")
  val m = Mutex<i64>(value: 0)
  loop (round in 1..2) {
    val closes = Atomic<i64>(value: 0)
    val after = Atomic(value: false)
    val r = withTimeout(Duration.millis(5), () => pending(m, closes, after))
    io.println("  round $round: ${outcome(r)}, went on past cede: ${after.load()}, closed ${closes.load()}")
  }

  io.println("5. a cancellation while queued:")
  val closes = Atomic<i64>(value: 0)
  io.println("  ${outcome(withTimeout(Duration.millis(5), () => cedeForever(closes)))}, closed ${closes.load()}")

  io.println("6. a failed sibling is seen:")
  val bodyCloses = Atomic<i64>(value: 0)
  failFast(bodyCloses) catch (e) { io.println("  the scope threw ${e.message()}") }
  io.println("  closed ${bodyCloses.load()}")

  io.println("7. a child abandoned by its scope is joined:")
  val childCloses = Atomic<i64>(value: 0)
  io.println("  ${abandon(childCloses)}, closed ${childCloses.load()}")

  io.println("8. a close() that cedes runs to its end:")
  val steps = Atomic<i64>(value: 0)
  val after = Atomic(value: false)
  val r = withTimeout(Duration.millis(5), () => closing(m, steps, after))
  io.println("  ${outcome(r)}, went on past cede: ${after.load()}, close steps ${steps.load()}")

  io.println("9. nested limits:")
  val innerCloses = Atomic<i64>(value: 0)
  val inner = withTimeout(Duration.seconds(10), () => outcome(withTimeout(Duration.millis(5), () => cedeForever(innerCloses))))
  io.println("  inner: ${inner ?? "the outer limit fired"}, closed ${innerCloses.load()}")
  var innerCaught = false
  val outer = withTimeout(Duration.millis(5), () => {
    val got = withTimeout(Duration.seconds(10), () => cedeForever(innerCloses))
    innerCaught = true
    got
  })
  io.println("  outer: ${outcome(outer)}, the inner limit saw it: $innerCaught, closed ${innerCloses.load()}")

  io.println("10. sleeps after cedes keep their contracts:")
  val sw = time.Stopwatch.start()
  await sleep(Duration.millis(20))
  io.println("  slept 20 ms: ${sw.elapsed().toMillis() >= 20}")
  await sleep(Duration.zero)
  await sleep(Duration.millis(-5))
  cede
  io.println("  done")
}
`
	want := `1. a lone task:
  150 [1, 2, 3, 4, 5]
  2000 item 1999
  resumed
2. a peer runs while this task cedes: true
3. the executor's thread is kept:
  one task: true
  two tasks: true
4. a cancellation pending before cede:
  round 1: timed out, went on past cede: false, closed 1
  round 2: timed out, went on past cede: false, closed 1
5. a cancellation while queued:
  timed out, closed 1
6. a failed sibling is seen:
  the scope threw Broke
  closed 1
7. a child abandoned by its scope is joined:
  7, closed 1
8. a close() that cedes runs to its end:
  timed out, went on past cede: false, close steps 3
9. nested limits:
  inner: timed out, closed 1
  outer: timed out, the inner limit saw it: false, closed 2
10. sleeps after cedes keep their contracts:
  slept 20 ms: true
  done`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("cede%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		for _, threads := range []string{"1", "2", "4", "8"} {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			run := exec.CommandContext(ctx, exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=4096", "VELES_GC_POISON=1")
			out, err := run.CombinedOutput()
			cancel()
			got := strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n")
			if err != nil || got != want {
				t.Fatalf("release=%v threads %s: got\n%s\nwant\n%s\n(%v)", release, threads, got, want, err)
			}
		}
	}
}
