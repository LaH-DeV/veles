package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D234–D236 at run time: a Deque and a PriorityQueue are values (a copy
// keeps what it held when the original changes, a pointer shares, a loop
// walks the deque as it was); copy() on every collection is a copy that
// stays apart; a capture list copies when the lambda is made while a plain
// capture shares, each call starts from the snapshot (a change lasts for
// the call), and a copied pointer still writes through.
func TestValueQueuesCopiesAndCaptureLists(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Job {
  var n: i64
}

struct Total {
  var sum: i64
  fun add(x: i64) { this.sum += x }
}

fun drain(q: *Deque<i64>): i64 {
  var s = 0
  loop { val x = q.removeFirst() ?: break; s += x }
  s
}

fun count(c: Job): i64 => c.n

struct Ledger {
  entries: List<i64>

  // D237: a block reading this reads a copy of the receiver
  fun countLater(): i64 suspends {
    with t = async {
      await sleep(Duration.millis(3))
      this.entries.len()
    }
    this.entries.push(9)
    await t
  }
}

fun main() suspends {
  // D234: deques
  val d = Deque<i64>()
  d.addLast(1)
  d.addLast(2)
  val d2 = d
  d.removeFirst()
  d2.addFirst(0)
  io.println("deque: $d $d2")
  io.println("drain: ${drain(&d2)} $d2 $d")
  loop (x in d) d.addLast(x * 10)
  io.println("loop: $d")
  val jobs = Deque<Job>()
  jobs.addLast(Job(n: 1))
  val jobs2 = jobs
  jobs.at(0)?.n += 5
  io.println("at: ${jobs.get(0)?.n} ${jobs2.get(0)?.n}")

  // D234: priority queues, Sendable when the elements are
  val pq = Heap<i64>()
  pq.push(3)
  pq.push(1)
  val pq2 = pq
  pq.pop()
  pq2.push(0)
  with t = async pq2.len()
  io.println("pq: $pq $pq2 ${pq2.pop()} ${pq.pop()} ${await t}")

  // D236: copy()
  val xs: List<i64> = [1]
  val m: Map<string, i64> = ["a": 1]
  val s: Set<i64> = [1]
  val a: Array<i64, 2> = [1, 2]
  val xs2 = xs.copy()
  val m2 = m.copy()
  val s2 = s.copy()
  var a2 = a.copy()
  val d3 = d.copy()
  val pq3 = pq2.copy()
  xs.push(2)
  m.set("b", 2)
  s.add(2)
  a2.set(0, 9)
  d.clear()
  pq2.push(-1)
  io.println("copy: $xs $xs2 $m $m2 ${s.len()} ${s2.len()} $a $a2 $d $d3 ${pq2.len()} ${pq3.len()}")

  // D235: capture lists
  val routes: List<string> = ["/a"]
  val shared = () => "$routes"
  val boot = [routes] () => "$routes"
  routes.push("/b")
  io.println("capture: ${shared()} ${boot()}")
  var counter = Job(n: 3)
  val seenAt = [seen = counter.n] () => seen
  counter.n = 10
  val names: List<string> = ["x"]
  val joined = [names] (sfx: string) => names.map(n => n + sfx).join(",")
  names.push("y")
  var total = Total(sum: 0)
  val p = &total
  val add = [p] (x: i64) => p.add(x)
  add(4)
  add(5)
  val job = Job(n: 7)
  val run = [job] () => count(job)
  with u = async run()
  counter.n = 0
  io.println("list: ${seenAt()} ${joined("!")} ${total.sum} ${await u}")

  // each call starts from the snapshot: a change lasts for the call
  val bumps = [counter] () => {
    counter.n += 1
    counter.n
  }
  val grow = [names] (x: string) => {
    names.push(x)
    "$names"
  }
  io.println("fresh: ${bumps()} ${bumps()} ${counter.n} ${grow("p")} ${grow("q")} $names")

  // an iterator holds the list as it was when it was made (2026-11-06:
  // xs.iter() lent its receiver, and saw a later push)
  val src: List<i64> = [1, 2, 3]
  val it = src.iter().map(x => x * 10)
  val plain = src.iter()
  src.push(4)
  io.println("iter: ${it.toList()} ${plain.toList()} $src")

  // D237: an async block reads the collections it captured as they were
  // when its task started, whatever the maker does meanwhile
  val shared: List<i64> = [1, 2, 3]
  with sum = async {
    await sleep(Duration.millis(5))
    var s = 0
    loop (x in shared) s += x
    s
  }
  shared.push(100)
  shared.clear()
  val rounds: List<i64> = []
  val seen: List<i64> = []
  val lock = Mutex(value: seen)
  scope {
    loop (i in 0..<3) {
      rounds.push(i)
      async {
        await sleep(Duration.millis(2))
        val n = rounds.len()
        lock.withLock(s => s.push(n))
      }
    }
    rounds.clear()
  }
  val got = lock.withLock(s => s.sorted())
  var ledger = Ledger(entries: [1, 2])
  val counted = ledger.countLater()
  io.println("tasks: ${await sum} $got $shared $rounds $counted ${ledger.entries}")
}
`
	want := "deque: [2] [0, 1, 2]\n" +
		"drain: 3 [] [2]\n" +
		"loop: [2, 20]\n" +
		"at: 6 1\n" +
		"pq: [3] [0, 3, 1] 0 3 3\n" +
		"copy: [1, 2] [1] {a: 1, b: 2} {a: 1} 2 1 [1, 2] [9, 2] [] [2, 20] 3 2\n" +
		"capture: [/a, /b] [/a]\n" +
		"list: 3 x! 9 7\n" +
		"fresh: 1 1 0 [x, y, p] [x, y, q] [x, y]\n" +
		"iter: [10, 20, 30] [1, 2, 3] [1, 2, 3, 4]\n" +
		"tasks: 6 [1, 2, 3] [] [] 2 [1, 2, 9]\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "values.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}
