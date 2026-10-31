package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// D218: a counted loop whose body calls nothing looks for attention once
// per chunk of up to 1024 rounds (codegen chunkedLoop). The rounds are the
// same whatever the chunks — ranges of every size around a chunk's edge,
// empty, negative, inclusive up to i64's top, lists both kinds, `break`,
// `continue`, `return` and labels across chunk edges — and what the looks
// are for still arrives inside one: a deadline stops a chunked loop that
// would run for minutes, and a collection another task needs goes on while
// a chunked loop runs (and, in a suspending function, the loop yields).
func TestChunkedLoops(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `
// D218: a tight counted loop looks for attention once per chunk.
use io
use time

// --- the rounds, whatever the chunks
fun count(lo: i64, hi: i64): i64 {
  var n: i64 = 0
  loop (_ in lo..<hi) n += 1
  n
}

fun countInclusive(lo: i64, hi: i64): i64 {
  var n: i64 = 0
  var last: i64 = 0
  loop (i in lo..hi) {
    n += 1
    last = i
  }
  n * 1000 + last % 1000
}

fun sumSquares(n: i64): i64 {
  var total: i64 = 0
  loop (i in 0..<n) total +%= i *% i
  total
}

fun firstOver(xs: List<i64>, limit: i64): i64 {
  var found: i64 = -1
  loop (i in 0..<xs.len()) {
    if (xs.at(i) > limit) {
      found = i
      break
    }
  }
  found
}

fun evensBelow(n: i64): i64 {
  var sum: i64 = 0
  loop (i in 0..<n) {
    if (i % 2 == 1) continue
    sum += i
  }
  sum
}

fun indexOf(xs: List<i64>, v: i64): i64 {
  loop (i in 0..<xs.len()) {
    if (xs.at(i) == v) return i
  }
  -1
}

fun listSum(xs: List<i64>): i64 {
  var s: i64 = 0
  loop (x in xs) s += x
  s
}

fun mutableSum(xs: MutableList<i64>): i64 {
  var s: i64 = 0
  loop (x in xs) s += x
  s
}

fun triples(n: i64): List<i64> {
  val out: MutableList<i64> = []
  loop (i in 0..<n) out.push(i * 3)
  out
}

fun pairs(n: i64): i64 {
  var hits: i64 = 0
  loop :rows (r in 0..<n) {
    loop (c in 0..<n) {
      if (c > r) continue rows
      if (r * c > 1000000) break rows
      hits += 1
    }
  }
  hits
}

// --- what the looks are for, inside a chunked loop
fun spin(rounds: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<rounds) x = x *% 6364136223846793005 +% i
  x
}

// the same, in a function that may suspend: its loop also yields
fun spinSuspending(rounds: i64): i64 {
  if (rounds < 0) await sleep(Duration.millis(1))
  var x: i64 = 1
  loop (i in 0..<rounds) x = x *% 6364136223846793005 +% i
  x
}

fun outcome<T>(r: Result<T, Timeout>): string => when (r) {
  is Ok  => "finished"
  is Err => "timed out"
}

fun allocator(stop: Atomic<bool>, made: Atomic<i64>) {
  loop (!stop.load()) {
    val xs: MutableList<string> = []
    loop (k in 0..<100) xs.push("item $k")
    made.add(xs.len())
    cede
  }
}

fun main() {
  io.println("1. rounds:")
  val sizes = [0, 1, 15, 16, 17, 1023, 1024, 1025, 2048, 100000]
  io.println("  ${sizes.map(n => count(0, n))}")
  io.println("  ${count(-5000, -10)} ${count(10, 3)}")
  val top = 9223372036854775807
  io.println("  to the top: ${countInclusive(top - 2500, top)} ${countInclusive(top, top)}")
  io.println("  ${sumSquares(3000)} ${sumSquares(1)}")
  val xs = triples(5000)
  val ms = MutableList<i64>.make(3000, i => i)
  io.println("  break: ${firstOver(xs, 3000)} ${firstOver(xs, 1000000)}, return: ${indexOf(xs, 4500)} ${indexOf(xs, 7)}")
  io.println("  continue: ${evensBelow(4097)}, a list: ${listSum(xs)} ${mutableSum(ms)}, labels: ${pairs(3000)}")

  io.println("2. a timeout stops a chunked loop:")
  val sw = time.Stopwatch.start()
  val r = withTimeout(Duration.millis(20), () => spin(100000000000))
  io.println("  ${outcome(r)} within a second: ${sw.elapsed().toMillis() < 1000}")

  io.println("3. collections go on beside one:")
  val stop = Atomic(value: false)
  val made = Atomic<i64>(value: 0)
  var x: i64 = 0
  scope {
    async allocator(stop, made)
    x = spinSuspending(300000000)
    stop.store(true)
  }
  io.println("  ${x != 0} ${made.load() > 0}")
}
`
	want := `1. rounds:
  [0, 1, 15, 16, 17, 1023, 1024, 1025, 2048, 100000]
  4990 0
  to the top: 2501807 1807
  8995500500 0
  break: 1001 -1, return: 1500 -1
  continue: 4196352, a list: 37492500 4498500, labels: 502501
2. a timeout stops a chunked loop:
  timed out within a second: true
3. collections go on beside one:
  true true`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, fmt.Sprintf("chunks%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		for _, threads := range threadCounts("1", "2", "4", "8") {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
