package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Collections landing while tasks are spawned on many threads (D66): a
// task created by `async` is held only in its owner's registers until it
// is queued, and a collection run by another thread then must find it in
// the owner's recorded registers. It once did not — the recording lost a
// register its own helper had reused — and the task was swept and reused
// while still in use. With VELES_GC_POISON a swept object is filled with
// 0xCD, so such a use crashes instead of passing unnoticed; a small
// threshold makes collections frequent.
func TestSpawnDuringCollections(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

fun square(i: i64): i64 {
  var acc: i64 = 0
  loop (k in 0..<50) acc += (i + k) % 7
  acc
}

fun batch(from: i64): i64 {
  val tasks: MutableList<Task<i64>> = []
  scope {
    loop (i in from..<from + 1000) {
      tasks.push(async square(i))
    }
  }
  var sum: i64 = 0
  loop (t in tasks) sum += await t
  sum
}

fun main() {
  var sum: i64 = 0
  loop (b in 0..<40) sum += batch(b * 1000)
  io.println("$sum")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "spawn.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for i := 0; i < 16; i++ {
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_THREADS=8", "VELES_GC_POISON=1", "VELES_GC_THRESHOLD=400000")
		out, err := run.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "5999995" {
			t.Fatalf("run %d: got %q, %v", i, out, err)
		}
	}
}

// An Atomic of a number or a bool takes no lock (D66): its operations are
// single instructions, and `update` retries a compare-and-swap. Eight
// threads race on each kind so a lost update shows up in the totals.
func TestAtomicWordsUnderThreads(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

fun bump(n: Atomic<i64>, f: Atomic<f64>, w: Atomic<u16>, flips: Atomic<bool>, s: Atomic<string>) {
  loop (_ in 0..<20000) {
    n.update(x => x + 1)
    f.update(x => x + 0.25)
    w.update(x => x.wrappingAdd(3))
    flips.update(x => !x)
    s.update(x => if (x.len() < 5) x + "a" else "")
  }
}

fun main() {
  val n = Atomic(value: 0)
  val f = Atomic(value: 0.0)
  val w = Atomic(value: 0 as u16)
  val flips = Atomic(value: false)
  val s = Atomic(value: "")
  scope {
    loop (_ in 0..<8) {
      async bump(n, f, w, flips, s)
    }
  }
  val old = n.swap(-1)
  n.store(n.load() - 1)
  io.println("${old} ${n.load()} ${f.load()} ${w.load()} ${flips.load()} ${s.load().len()}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "atomic.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	// 160000 updates each: 160000 × 3 mod 65536 = 21248; an even number of
	// flips; the string cycles through six states.
	want := "160000 -2 40000.0 21248 false 4"
	for i := 0; i < 8; i++ {
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_THREADS=8")
		out, err := run.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("run %d: got %q, want %q (%v)", i, out, want, err)
		}
	}
}

// Values cross a channel exactly once under threads (D16/D66): a blocked
// sender's value is taken straight from its slot — the whole exchange on a
// rendezvous channel, and the refill of a full buffer — and a blocked
// receiver's value is written straight into its slot. Producers and
// consumers race on both kinds; a value lost or delivered twice shows in
// the totals.
func TestChannelHandoffUnderThreads(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

fun produce(ch: Channel<i64>) {
  loop (i in 1..1000) ch.send(i)
}

fun consume(ch: Channel<i64>, sum: Atomic<i64>, count: Atomic<i64>) {
  loop {
    val v = await ch.recv() ?: break
    val _ = sum.update(s => s + v)
    val _ = count.update(c => c + 1)
  }
}

fun run(capacity: i64): string {
  val ch = if (capacity == 0) Channel<i64>() else Channel<i64>(capacity: capacity)
  ch.closeAfter(8000)
  val sum = Atomic(value: 0)
  val count = Atomic(value: 0)
  scope {
    loop (_ in 0..<8) {
      async produce(ch)
    }
    loop (_ in 0..<4) {
      async consume(ch, sum, count)
    }
  }
  "${count.load()} ${sum.load()}"
}

fun main() {
  io.println("${run(0)} ${run(1)} ${run(16)}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "chan.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	want := "8000 4004000 8000 4004000 8000 4004000"
	for _, threads := range []string{"1", "8", "8", "8"} {
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+threads)
		out, err := run.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("threads %s: got %q, want %q (%v)", threads, out, want, err)
		}
	}
}

// A race over two channels under threads (D38/D66): senders on both —
// one a rendezvous, one buffered — and a timer arm compete for the
// winner, and the scope's short-lived children keep waking the racing
// task for other reasons (it then registers again from nothing: it once
// re-added nodes that were already listed, cutting the list). Every value
// sent must be received exactly once.
func TestRaceOverChannelsUnderThreads(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

fun feed(ch: Channel<i64>, from: i64) {
  loop (i in from..<from + 1000) {
    ch.send(i)
    if (i % 3 == 0) await sleep(Duration.zero)   // slow enough that the race waits
  }
}

fun blip(n: i64): i64 {
  await sleep(Duration.zero)
  n
}

fun listen(ch: Channel<i64>): i64 = await ch.recv() ?: -1

fun nap(): i64 {
  await sleep(Duration.millis(20))
  0
}

// The racing task is woken while it waits — its scope's only child
// finishes — with another receiver listed behind its node on the same
// channel. Registering again must not cut that receiver off the list.
fun wokenMidRace(): i64 {
  val quiet = Channel<i64>(capacity: 1)
  var heard = 0
  scope {
    val l = async listen(quiet)
    scope {
      val _ = async nap()
      race {
        val v = quiet.recv() => heard = -100
        sleep(Duration.millis(100)) => {}
      }
    }
    quiet.send(7)
    heard += await l
  }
  heard
}

fun main() {
  val a = Channel<i64>()
  val b = Channel<i64>(capacity: 4)
  var sum = 0
  var got = 0
  var timeouts = 0
  scope {
    loop (k in 0..<4) {
      async feed(a, k * 1000)
      async feed(b, 4000 + k * 1000)
    }
    // the only child of this scope: finishing, it wakes the owner, which
    // is waiting in the race — a wake for another reason
    scope {
      loop (got < 8000) {
        val _ = async blip(got)
        race {
          val x = a.recv() => {
            sum += x ?: 0
            got += 1
          }
          val y = b.recv() => {
            sum += y ?: 0
            got += 1
          }
          sleep(Duration.millis(500)) => timeouts += 1
        }
      }
    }
  }
  io.println("$got $sum $timeouts ${wokenMidRace()}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "race.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	want := "8000 31996000 0 7" // 0 + 1 + … + 7999
	for _, threads := range []string{"1", "2", "8", "8", "8"} {
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+threads)
		out, err := run.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("threads %s: got %q, want %q (%v)", threads, out, want, err)
		}
	}
}

// Task-local values under threads (D72): 200 requests at once, each bound
// to its own id, each starting children that read it after suspending —
// on whichever thread resumes them. A child keeps the value bound when it
// started, and the binding ends with withValue also when the work throws.
func TestTaskLocalsUnderThreads(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

val requestId = TaskLocal(fallback: -1)

fun leaf(): i64 {
  await sleep(Duration.zero)
  requestId.get()
}

fun handle(id: i64): i64 = requestId.withValue(id, () => {
  var sum = 0
  scope {
    val a = async leaf()
    val b = async leaf()
    sum = await a + await b
  }
  sum + leaf()
})

error Nope { }

fun fails(): i64 throws Nope {
  await sleep(Duration.zero)
  throw Nope()
}

fun main() {
  val ids: MutableList<i64> = []
  loop (i in 0..<200) ids.push(i)
  val got = ids.toList().mapConcurrent(i => handle(i) - 3 * i, workers: 64)
  val wrong = got.filter(d => d != 0).len()
  val r = requestId.withValue(99, () => try fails())
  io.println("$wrong wrong, threw: ${r is Err}, outside: ${requestId.get()}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "tl.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	want := "0 wrong, threw: true, outside: -1"
	for _, threads := range []string{"1", "8", "8"} {
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=300000")
		out, err := run.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("threads %s: got %q, want %q (%v)", threads, out, want, err)
		}
	}
}

// IoError.kind is the same on every platform (D76): the runtime maps
// errno, and on Windows the Winsock codes, to one IoKind.
func TestIoErrorKinds(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use fs, io, net

fun kindOf<T>(r: Result<T, IoError>): IoKind = when (r) {
  is Ok(_) => IoKind.Other
  is Err(e) => e.kind
}

fun main() {
  io.println("${kindOf(fs.readFile("no/such/file.txt"))}")
  val l = net.listen() ?? { e => panic("listen: $e") }
  val port = l.port()
  io.println("${kindOf(net.listen(port: port))}")
  l.close()
  io.println("${kindOf(net.connect("127.0.0.1", port))}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "kinds.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	out, err := exec.Command(exe).CombinedOutput()
	got := strings.Fields(string(out))
	want := []string{"NotFound", "AddressInUse", "ConnectionRefused"}
	if err != nil || strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %q, %v; want %v", out, err, want)
	}
}
