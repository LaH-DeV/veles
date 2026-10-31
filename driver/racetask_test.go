package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D168/D169: a task the construct starts — a `race { }` contestant, one
// started in a loop or by `async { }`, an `async f()` arm of a `select` — is
// cancelled and joined when it loses, before the winner's value is given
// (its `with` cleanup has run by then); a task that fails first makes the
// race throw; a handle started before a select is not cancelled when it
// loses; `return` in an `async { }` gives its value. The loop at the end
// races many pairs at once, so the claims are exercised across threads. Run
// on 1 to 8 threads, three times each; nothing depends on timing except
// that a ten-second sleep loses.
func TestRaceTaskArms(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io { println }

error Down {
  host: string
}

struct Counted {
  cleaned: Atomic<i64>
  implement Closeable {
    fun close() { this.cleaned.add(1) }
  }
}

// a loser cancelled before it ever ran opened no 'with', so the counts
// compare the cleanups with the losers that started
val started = Atomic(value: 0)

fun slow(cleaned: Atomic<i64>): string {
  started.add(1)
  with c = Counted(cleaned)
  await sleep(Duration.seconds(10))
  "slow"
}

fun fast(): string {
  await sleep(Duration.millis(5))
  "fast"
}

fun failing(): string throws Down {
  await sleep(Duration.millis(5))
  throw Down(host: "eu")
}

fun finishes(done: Atomic<i64>): string {
  await sleep(Duration.millis(20))
  done.add(1)
  "finished"
}

fun pair(i: i64, cleaned: Atomic<i64>): string => if (i % 2 == 0) {
  race {
    async slow(cleaned)
    async fast()
  }
} else {
  race {
    async fast()
    async {
      await sleep(Duration.zero)
      slow(cleaned)
    }
  }
}

fun failsFirst(cleaned: Atomic<i64>): string throws Down => race {
  async failing()
  async slow(cleaned)
}

// a race over a list: the first of many, the rest cancelled
fun firstOf(n: i64, cleaned: Atomic<i64>): string => race {
  loop (i in 0..<n) {
    if (i == n - 1) async fast() else async slow(cleaned)
  }
}

fun main() throws Down {
  val cleaned = Atomic(value: 0)
  val won = select {
    async slow(cleaned)               => "slow"
    val v = async fast()              => "$v, loser cleaned ${cleaned.load()}"
  }
  println(won)

  val cleaned2 = Atomic(value: 0)
  val first = race {
    async slow(cleaned2)
    async fast()
  }
  println("$first, loser cleaned ${cleaned2.load()}")

  val cleaned3 = Atomic(value: 0)
  val failed = failsFirst(cleaned3) catch (e) { "failed at ${e.host}" }
  println("$failed, loser cleaned ${cleaned3.load()}")

  val timed = select {
    async slow(cleaned3)      => "slow"
    sleep(Duration.millis(1)) => "timer"
  }
  println("$timed, loser cleaned ${cleaned3.load()}")

  val done = Atomic(value: 0)
  scope {
    val t = async finishes(done)
    val r = select {
      val v = t                 => v
      sleep(Duration.millis(1)) => "timer"
    }
    println("handle arm: $r")
  }
  println("handle not cancelled ${done.load() == 1}")

  val cleaned4 = Atomic(value: 0)
  val before4 = started.load()
  println("${firstOf(8, cleaned4)} of 8, every started loser cleaned ${cleaned4.load() == started.load() - before4}")

  val sum = async {
    var total = 0
    loop (i in 1..100) total += i
    if (total < 0) return -1
    total
  }
  println("async block ${await sum}")

  val many = Atomic(value: 0)
  val before = started.load()
  val inputs: MutableList<i64> = []
  loop (i in 0..<200) inputs.push(i)
  val results = inputs.toList().mapConcurrent(i => pair(i, many), workers: 16)
  println("pairs ${results.filter(r => r == "fast").len()}, every started loser cleaned ${many.load() == started.load() - before}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "fast, loser cleaned 1\nfast, loser cleaned 1\nfailed at eu, loser cleaned 1\ntimer, loser cleaned 2\nhandle arm: timer\nhandle not cancelled true\nfast of 8, every started loser cleaned true\nasync block 5050\npairs 200, every started loser cleaned true\n"
	exe := filepath.Join(dir, "racetask.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, threads := range threadCounts("1", "2", "4", "8") {
		for run := 0; run < 3; run++ {
			cmd := exec.Command(exe)
			cmd.Env = append(os.Environ(), "VELES_THREADS="+threads)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("threads=%s run %d: %v\n%s", threads, run, err, out)
			}
			if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
				t.Fatalf("threads=%s run %d:\n got:\n%s\nwant:\n%s", threads, run, got, want)
			}
		}
	}
}
