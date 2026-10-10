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
  var sum: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []
    loop (i in from..<from + 1000) {
      tasks.push(async square(i))
    }
    loop (t in tasks) sum += await t
  }
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
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
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
  val w = Atomic(value: (0).wrapU16())
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
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
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
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
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

// Suspending calls run in their caller's task (concurrency review F3,
// 2026-10-09): a call that does not wait costs no task, and one that waits
// parks the task on its own frame and returns to its caller when it ends.
// What a call did when it was a task of its own must still hold: a failed
// child abandons a body three calls deep, closing innermost first (and the
// body's own resource last — it used to close first); a timeout cancels
// three calls deep; values, big values and errors come back across
// suspensions; 20 000-deep recursion suspends at the bottom; a panic two
// calls deep after a suspension unwinds both and is a value at gather.
func TestSuspendingCallsInOneTask(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

error Boom { n: i64 }

struct Res {
  name: string
  implement Closeable {
    fun close() { io.println("  closed ${this.name}") }
  }
}

struct Big {
  a: i64
  b: i64
  c: i64
  d: i64
  e: i64
  f: string
}

fun inner(ch: Channel<i64>): i64 {
  with r = Res(name: "inner")
  val v = await ch.recv() ?: return -1
  v
}

fun middle(ch: Channel<i64>): i64 {
  with r = Res(name: "middle")
  inner(ch) + 1
}

fun outer(ch: Channel<i64>): i64 {
  with r = Res(name: "outer")
  middle(ch) + 1
}

fun failSoon(): i64 throws Boom {
  await sleep(Duration.millis(20))
  throw Boom(n: 7)
}

fun abandoned(): i64 throws Boom {
  val ch = Channel<i64>()
  var got: i64 = 0
  scope {
    async failSoon()
    with r = Res(name: "body")
    got = outer(ch)
  }
  got
}

fun makeBig(n: i64): Big {
  await sleep(Duration.millis(1))
  Big(a: n, b: n + 1, c: n + 2, d: n + 3, e: n + 4, f: "big $n")
}

fun failLate(n: i64): i64 throws Boom {
  await sleep(Duration.millis(1))
  if (n > 2) throw Boom(n)
  n
}

fun relay(n: i64): i64 throws Boom => (try failLate(n)) * 10

fun down(n: i64): i64 {
  if (n == 0) {
    await sleep(Duration.millis(1))
    return 0
  }
  down(n - 1) + 1
}

fun panicky(ch: Channel<i64>): i64 {
  with r = Res(name: "panicky")
  val v = await ch.recv() ?: 0
  if (v == 42) panic("deep panic")
  v
}

fun relayPanic(ch: Channel<i64>): i64 {
  with r = Res(name: "relayPanic")
  panicky(ch)
}

fun feed(ch: Channel<i64>, v: i64) {
  await sleep(Duration.millis(5))
  ch.send(v)
}

fun main() {
  io.println("1:")
  io.println("  -> ${abandoned() catch (e) { e.n }}")
  io.println("2:")
  val ch = Channel<i64>()
  val r = withTimeout(Duration.millis(20), () => outer(ch))
  io.println("  -> timed out: ${r is Err}")
  io.println("3:")
  val ch2 = Channel<i64>(capacity: 1)
  ch2.send(40)
  io.println("  -> outer = ${outer(ch2)}")
  val b = makeBig(5)
  io.println("  -> big = ${b.a} ${b.e} ${b.f}")
  io.println("  -> relay(2) = ${relay(2) catch (e) { -e.n }}, relay(3) = ${relay(3) catch (e) { -e.n }}")
  io.println("  -> down(20000) = ${down(20000)}")
  io.println("4:")
  val ch3 = Channel<i64>()
  val g = gather {
    async relayPanic(ch3)
    async feed(ch3, 42)
  }
  val (p, _) = g
  io.println("  -> panicked: ${p is Err}")
}
`
	want := `1:
  closed inner
  closed middle
  closed outer
  closed body
  -> 7
2:
  closed inner
  closed middle
  closed outer
  -> timed out: true
3:
  closed inner
  closed middle
  closed outer
  -> outer = 42
  -> big = 5 9 big 5
  -> relay(2) = 20, relay(3) = -3
  -> down(20000) = 20000
4:
  closed panicky
  closed relayPanic
  -> panicked: true`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("calls%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		for _, threads := range []string{"1", "2", "8"} {
			run := exec.Command(exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=4096")
			out, err := run.CombinedOutput()
			got := strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n")
			if err != nil || got != want {
				t.Fatalf("release=%v threads %s: got\n%s\nwant\n%s\n(%v)", release, threads, got, want, err)
			}
		}
	}
}

// Atomics across threads (D144, concurrency review F5, 2026-10-09): a
// Treiber stack on an Atomic of a nullable pointer (compareExchange and
// compareAndSet with explicit orders), relaxed adds, fetchOr/fetchXor on
// bit flags, a release/acquire hand-off, and wrapping adds.
func TestAtomicOperations(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `// D144: atomics across threads — a lock-free stack on a nullable pointer,
// counters with relaxed adds, bit flags, and a release/acquire hand-off.
use io

struct Node {
  value: i64
  next:  (*Node)?
}

// a Treiber stack: push and pop are one compareExchange each, retried
struct Stack {
  head: Atomic<(*Node)?> = Atomic(value: null)

  fun push(value: i64) {
    var seen = this.head.load(order: MemoryOrder.Relaxed)
    loop {
      val node = &Node(value, next: seen)
      val found = this.head.compareExchange(seen, node, order: MemoryOrder.Release, failure: MemoryOrder.Relaxed)
      if (found == seen) return
      seen = found
    }
  }

  fun pop(): i64? {
    loop {
      val top = this.head.load(order: MemoryOrder.Acquire) ?: return null
      if (this.head.compareAndSet(top, top.next, order: MemoryOrder.Acquire)) return top.value
    }
  }
}

fun producer(s: Stack, from: i64, n: i64) {
  loop (i in from..<from + n) s.push(i)
}

fun consumer(s: Stack, sum: Atomic<i64>, count: Atomic<i64>, want: i64) {
  loop {
    if (count.load() >= want) return
    val v = s.pop()
    if (v != null) {
      val _ = sum.add(v, order: MemoryOrder.Relaxed)
      val _ = count.add(1)
    } else {
      yieldNow()
    }
  }
}

fun flagger(flags: Atomic<u64>, bit: u64, seen: Atomic<i64>) {
  val mask: u64 = 1 << bit
  val before = flags.fetchOr(mask)
  if (before & mask == 0) {
    val _ = seen.add(1)
  }
}

fun publish(data: Atomic<i64>, ready: Atomic<bool>) {
  data.store(42, order: MemoryOrder.Relaxed)
  ready.store(true, order: MemoryOrder.Release)
}

fun observe(data: Atomic<i64>, ready: Atomic<bool>): i64 {
  loop {
    if (ready.load(order: MemoryOrder.Acquire)) return data.load(order: MemoryOrder.Relaxed)
    yieldNow()
  }
}

fun main() {
  val s = Stack()
  val sum = Atomic(value: 0)
  val count = Atomic(value: 0)
  val tasks: i64 = 8
  val each: i64 = 20000
  scope {
    loop (k in 0..<tasks) {
      async producer(s, k * each, each)
      async consumer(s, sum, count, tasks * each)
    }
  }
  val n = tasks * each
  io.println("popped ${count.load()} of $n, sum right: ${sum.load() == n * (n - 1) / 2}, empty: ${s.pop() == null}")

  val flags = Atomic<u64>(value: 0)
  val seen = Atomic(value: 0)
  scope {
    loop (k in 0..<64) {
      val bit: u64 = k.wrapU64()
      async flagger(flags, bit, seen)
      async flagger(flags, bit, seen)
    }
  }
  io.println("each of 64 bits set once: ${seen.load() == 64}, all set: ${flags.load() == 0xFFFFFFFFFFFFFFFF}")
  io.println("xor back to zero: ${flags.fetchXor(0xFFFFFFFFFFFFFFFF) == 0xFFFFFFFFFFFFFFFF && flags.load() == 0}")

  val data = Atomic(value: 0)
  val ready = Atomic(value: false)
  var got: i64 = 0
  scope {
    val o = async observe(data, ready)
    async publish(data, ready)
    got = await o
  }
  io.println("release/acquire hand-off: $got")

  val wrap = Atomic<i32>(value: 2147483647)
  io.println("i32 add wraps: ${wrap.add(1)}, sub wraps back: ${wrap.sub(1)}")
}
`
	want := `popped 160000 of 160000, sum right: true, empty: true
each of 64 bits set once: true, all set: true
xor back to zero: true
release/acquire hand-off: 42
i32 add wraps: -2147483648, sub wraps back: 2147483647`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("atomics%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		for _, threads := range []string{"1", "2", "8"} {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			run := exec.CommandContext(ctx, exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=4096")
			out, err := run.CombinedOutput()
			cancel()
			got := strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n")
			if err != nil || got != want {
				t.Fatalf("release=%v threads %s: got\n%s\nwant\n%s\n(%v)", release, threads, got, want, err)
			}
		}
	}
}

// The synchronisation types (D146, concurrency review F6, 2026-10-09):
// RwLock readers and writers never see half a write, and taking one again
// in its own region panics; Event releases every waiter and stays set until
// reset, and its wait is a race arm and cancellable; Watch wakes a follower
// once per change it has not seen; Broadcast gives every subscriber every
// value and tells one that fell behind (Lagged); Lazy builds once and
// re-raises its init's panic in every get.
func TestSyncTypes(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `// D146: RwLock, Event, Lazy, Broadcast and Watch across threads.
use io
use os

struct Pair {
  var a: i64
  var b: i64 // always equal to a: a reader that saw them differ saw half a write
}

fun reader(p: RwLock<Pair>, n: i64, torn: Atomic<i64>) {
  loop (_ in 0..<n) {
    with c = p.read()
    if (c.a != c.b) torn.add(1)
  }
}

fun writer(p: RwLock<Pair>, n: i64) {
  loop (_ in 0..<n) {
    with w = p.write()
    w.a += 1
    w.b += 1
  }
}

fun waiter(e: Event, done: Atomic<i64>) {
  await e.wait()
  done.add(1)
}

fun follower(w: Watch<i64>, seen: Channel<i64>, step: Channel<bool>) {
  loop {
    await w.changed()
    val v = w.get()
    seen.send(v)
    step.send(true)
    if (v >= 3) return
  }
}

fun listener(sub: Subscription<string>, out: Channel<string>) {
  loop {
    val v = (try await sub.recv()) catch (e) { "lagged ${e.missed}" } ?: break
    out.send(v)
  }
  out.send("end")
}

val built = Atomic(value: 0)
val table = Lazy(init: () => {
  built.add(1)
  [1, 2, 3]
})

fun lazyReader(n: Atomic<i64>) {
  n.add(table.get().len())
}

val broken = Lazy<i64>(init: () => panic("no table today"))

fun breaks(): i64 => broken.get()

fun relock(how: string) {
  val l = RwLock(value: 1)
  when (how) {
    "read-read" => {
      with x = l.read()
      with y = l.read()
      io.println("${*x + *y}")
    }
    "write-read" => l.withWrite(p => {
      io.println("${l.get()} ${*p}")
    })
    else => {
      with x = l.read()
      l.set(2)
    }
  }
}

fun main() throws Lagged {
  if (os.args().len() > 0) {
    relock(os.args().at(0) ?: "")
    return
  }
  // RwLock: writers never interleave with readers or each other
  val p = RwLock(value: Pair(a: 0, b: 0))
  val torn = Atomic(value: 0)
  scope {
    loop (_ in 0..<6) async reader(p, 5000, torn)
    loop (_ in 0..<2) async writer(p, 5000)
  }
  io.println("rwlock: a ${p.get().a}, torn reads ${torn.load()}, withRead ${p.withRead(c => c.a + c.b)}")

  // Event
  val e = Event()
  val done = Atomic(value: 0)
  scope {
    loop (_ in 0..<5) async waiter(e, done)
    await sleep(Duration.millis(5))
    io.println("event: before set ${done.load()}, set ${e.isSet()}")
    e.set()
  }
  await e.wait()
  io.println("event: after set ${done.load()}, set ${e.isSet()}")
  e.reset()
  val r1 = race {
    e.wait() => "set"
    sleep(Duration.millis(10)) => "not set"
  }
  val r2 = withTimeout(Duration.millis(10), () => {
    await e.wait()
    1
  })
  io.println("event: after reset $r1, a wait cancelled by a timeout: ${r2 is Err}")

  // Watch
  val w = Watch(value: 0)
  val seen = Channel<i64>(capacity: 10)
  val step = Channel<bool>(capacity: 1)
  scope {
    async follower(w, seen, step)
    loop (i in 1..3) {
      w.set(i)
      val _ = await step.recv()
    }
  }
  seen.close()
  io.println("watch: ${seen.toList()}")
  val w2 = Watch(value: "a")
  w2.set("b")
  val r3 = race {
    w2.changed() => "changed to ${w2.get()}"
    sleep(Duration.millis(10)) => "no change"
  }
  val r4 = race {
    w2.changed() => "changed again"
    sleep(Duration.millis(10)) => "no change since the get"
  }
  io.println("watch: $r3; $r4")

  // Broadcast: each subscriber gets every value; one too far behind is told
  val news = Broadcast<string>(capacity: 2)
  val out1 = Channel<string>(capacity: 20)
  val out2 = Channel<string>(capacity: 20)
  with s1 = news.subscribe()
  news.send("a")
  news.send("b")
  with s2 = news.subscribe()
  news.send("c")
  news.send("d")
  news.close()
  scope {
    async listener(s1, out1)
    async listener(s2, out2)
  }
  out1.close()
  out2.close()
  io.println("broadcast: first ${out1.toList()}, second ${out2.toList()}")
  val n2 = Broadcast<i64>(capacity: 4)
  with s3 = n2.subscribe()
  val r5 = race {
    val v = try s3.recv() => "got $v"
    sleep(Duration.millis(10)) => "nothing yet"
  }
  n2.send(7)
  val r6 = race {
    val v = try s3.recv() => "got $v"
    sleep(Duration.millis(10)) => "nothing yet"
  }
  io.println("broadcast: $r5, then $r6")

  // Lazy: built once by whichever task asks first; a panic in init is every get's
  val n = Atomic(value: 0)
  scope {
    loop (_ in 0..<8) async lazyReader(n)
  }
  io.println("lazy: built ${built.load()} time, read ${n.load()}, ready ${table.isReady()}")
  loop (_ in 0..<2) {
    when (gather { async breaks() }) {
      is Ok(v) => io.println("lazy: $v")
      is Err(x) => io.println("lazy: panicked: ${x.message()}")
    }
  }
}
`
	want := `rwlock: a 10000, torn reads 0, withRead 20000
event: before set 0, set false
event: after set 5, set true
event: after reset not set, a wait cancelled by a timeout: true
watch: [1, 2, 3]
watch: changed to b; no change since the get
broadcast: first [lagged 2, c, d, end], second [c, d, end]
broadcast: nothing yet, then got 7
lazy: built 1 time, read 24, ready true
lazy: panicked: no table today
lazy: panicked: no table today`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("sync%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		for _, threads := range []string{"1", "2", "8"} {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			run := exec.CommandContext(ctx, exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=4096")
			out, err := run.CombinedOutput()
			cancel()
			got := strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n")
			if err != nil || got != want {
				t.Fatalf("release=%v threads %s: got\n%s\nwant\n%s\n(%v)", release, threads, got, want, err)
			}
		}
		for _, how := range []string{"read-read", "write-read", "read-write"} {
			out, err := exec.Command(exe, how).CombinedOutput()
			if err == nil || !strings.Contains(string(out), "panic: an RwLock was locked again while this task holds it") {
				t.Fatalf("release=%v %s: want the re-lock panic, got %q (%v)", release, how, out, err)
			}
		}
	}
}

// A close() that suspends (D147, concurrency review F7, 2026-10-10): a
// `with` on the type waits for it on every way out — the normal end, a
// panic (the close runs as a task of its own that takes the cleanups still
// to run, so they close innermost first even across two such closes), a
// cancellation while parked and one in a plain loop (D145) — and the close
// is shielded: a cancellation that arrives while it waits lets it finish.
// A generic `with` follows its type argument.
func TestSuspendingClose(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `// D147: a close() may suspend.
use io
use time

struct Conn {
  name: string
  implement Closeable {
    fun close() suspends {
      await sleep(Duration.millis(20))
      io.println("  closed ${this.name} (after a wait)")
    }
  }
}

struct File {
  name: string
  implement Closeable {
    fun close() {
      io.println("  closed ${this.name}")
    }
  }
}

fun normal(): i64 {
  with f = File(name: "outer file")
  with c = Conn(name: "conn")
  with g = File(name: "inner file")
  42
}

fun panics() {
  with f = File(name: "outer file")
  with c = Conn(name: "conn 1")
  with d = Conn(name: "conn 2")
  with g = File(name: "inner file")
  panic("boom")
}

fun parked() {
  with f = File(name: "outer file")
  with c = Conn(name: "conn")
  await sleep(Duration.seconds(10))
}

fun spin(n: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<n) x = (x * 31 + i) % 1000003
  x
}

fun computing(): i64 {
  with f = File(name: "outer file")
  with c = Conn(name: "conn")
  spin(200000000)
}

fun closeAll<T: Closeable>(x: T) {
  with y = x
  io.println("  holding")
}

fun main() {
  io.println("1. normal exit, innermost first:")
  io.println("  -> ${normal()}")

  io.println("2. a panic: the waiting closes still run, in order:")
  when (gather { async panics() }) {
    is Ok => io.println("  -> no panic?")
    is Err(p) => io.println("  -> panicked: ${p.message()}")
  }

  io.println("3. cancelled while parked: the close waits, shielded:")
  val r3 = withTimeout(Duration.millis(10), () => parked())
  io.println("  -> timed out ${r3 is Err}")

  io.println("4. cancelled in a plain loop (D145):")
  val r4 = withTimeout(Duration.millis(30), () => computing())
  io.println("  -> timed out ${r4 is Err}")

  io.println("5. generic: the instance follows the type:")
  closeAll(Conn(name: "generic conn"))
  closeAll(File(name: "generic file"))
  later()
}

fun quick(): i64 {
  with c = Conn(name: "conn closing when the timeout fires")
  1
}

fun later() {
  io.println("6. cancelled while the close waits: it finishes first:")
  val sw = time.Stopwatch.start()
  val r6 = withTimeout(Duration.millis(5), () => quick())
  io.println("  -> result ${r6 is Ok}, the close took its 20 ms: ${sw.elapsed().toMillis() >= 19}")
}
`
	want := `1. normal exit, innermost first:
  closed inner file
  closed conn (after a wait)
  closed outer file
  -> 42
2. a panic: the waiting closes still run, in order:
  closed inner file
  closed conn 2 (after a wait)
  closed conn 1 (after a wait)
  closed outer file
  -> panicked: boom
3. cancelled while parked: the close waits, shielded:
  closed conn (after a wait)
  closed outer file
  -> timed out true
4. cancelled in a plain loop (D145):
  closed conn (after a wait)
  closed outer file
  -> timed out true
5. generic: the instance follows the type:
  holding
  closed generic conn (after a wait)
  holding
  closed generic file
6. cancelled while the close waits: it finishes first:
  closed conn closing when the timeout fires (after a wait)
  -> result false, the close took its 20 ms: true`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("close%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		for _, threads := range []string{"1", "2", "8"} {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			run := exec.CommandContext(ctx, exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=4096")
			out, err := run.CombinedOutput()
			cancel()
			got := strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n")
			if err != nil || got != want {
				t.Fatalf("release=%v threads %s: got\n%s\nwant\n%s\n(%v)", release, threads, got, want, err)
			}
		}
	}
}

// Loops are cancellation points, and suspending loops yield (D145,
// concurrency review F4, 2026-10-09). A plain loop is cancelled at its
// back edge (withTimeout returned only when it finished before); a
// suspending loop gives the only thread up to a ticker; a lock region and
// a close() run to their end; checkCancelled unwinds; and a task unwound
// without its frame — by that cancellation, or by a panic — ends only
// once its scope's children have unwound (D3: the panic once left them
// running past it).
func TestLoopsAreCancellationPoints(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `// D145: loops are cancellation points; suspending loops yield.
use io
use os
use time

struct Res {
  name: string
  implement Closeable {
    fun close() { io.println("  closed ${this.name}") }
  }
}

// a long computation that never waits
fun spin(n: i64): i64 {
  var x: i64 = 1
  loop (i in 0..<n) x = (x * 31 + i) % 1000003
  x
}

fun spinWith(n: i64): i64 {
  with r = Res(name: "spinWith")
  spin(n)
}

// the same, as a suspending function (it could wait, and never does)
fun spinSuspending(n: i64): i64 {
  if (n < 0) await sleep(Duration.millis(1))
  var x: i64 = 1
  loop (i in 0..<n) x = (x * 31 + i) % 1000003
  x
}

fun ticker(sw: time.Stopwatch, first: Atomic<i64>) {
  await sleep(Duration.millis(50))
  val _ = first.swap(sw.elapsed().toMillis())
}

// a lock region and a close() are shielded
struct Slow {
  implement Closeable {
    fun close() {
      val n = spin(30000000)
      io.println("  slow close finished (${n != 12345678})")
    }
  }
}

fun holdLock(m: Mutex<i64>): i64 {
  m.withLock(p => {
    val _ = spin(30000000)
    *p = 1
    *p
  })
}

fun closeSlowly(): i64 {
  with s = Slow()
  spin(4000000000)
}

// a cancellation in plain code while a scope's child still unwinds
fun child(opened: Channel<bool>) {
  with r = Res(name: "child")
  opened.send(true)
  await sleep(Duration.seconds(10))
}

fun parentPlain(): i64 {
  val opened = Channel<bool>(capacity: 1)
  scope {
    async child(opened)
    val _ = await opened.recv()
    spin(4000000000)
  }
  0
}

fun parentPanics() {
  val opened = Channel<bool>(capacity: 1)
  scope {
    async child(opened)
    val _ = await opened.recv()
    panic("parent panics")
  }
}

fun checking(): i64 {
  with r = Res(name: "checking")
  var n: i64 = 0
  loop {
    n += 1
    if (n % 1000 == 0) checkCancelled()
    if (n < 0) break
  }
  n
}

error Boom { n: i64 }

fun failSoon() throws Boom {
  await sleep(Duration.millis(20))
  throw Boom(n: 7)
}

// a loop in a scope body notices a failed child at its back edge
fun bodyLoop(): i64 throws Boom {
  var x: i64 = 0
  scope {
    async failSoon()
    loop (i in 0..<4000000000) x = (x * 31 + i) % 1000003
  }
  x
}
fun fair() {
  io.println("2. a suspending loop gives its thread up:")
  val sw = time.Stopwatch.start()
  val first = Atomic(value: 0)
  scope {
    async ticker(sw, first)
    async spinSuspending(300000000)
  }
  io.println("  -> the ticker ran within 150 ms: ${first.load() < 150}")
}

fun main() {
  if (os.args().at(0) == "fair") {
    fair()
    return
  }
  val n: i64 = 4000000000

  io.println("1. a plain loop is cancelled at its back edge:")
  val sw = time.Stopwatch.start()
  val r1 = withTimeout(Duration.millis(100), () => spinWith(n))
  io.println("  -> timed out ${r1 is Err}, within 400 ms: ${sw.elapsed().toMillis() < 400}")

  fair()

  io.println("3. a lock region is shielded:")
  val m = Mutex(value: 0)
  val r3 = withTimeout(Duration.millis(5), () => holdLock(m))
  io.println("  -> timed out ${r3 is Err}, the region finished: ${m.get() > 0}")

  io.println("4. a close() is shielded:")
  val r4 = withTimeout(Duration.millis(100), () => closeSlowly())
  io.println("  -> timed out ${r4 is Err}")

  io.println("5. the end waits for a scope's children:")
  val r5 = withTimeout(Duration.millis(100), () => parentPlain())
  io.println("  -> timed out ${r5 is Err}")

  io.println("6. checkCancelled:")
  val r6 = withTimeout(Duration.millis(50), () => checking())
  io.println("  -> timed out ${r6 is Err}")

  io.println("7. a panic's end waits for a scope's children:")
  val g = gather { async parentPanics() }
  io.println("  -> panicked: ${g is Err}")

  io.println("8. a loop in a scope body is abandoned when a child fails:")
  val sw8 = time.Stopwatch.start()
  val e8 = bodyLoop() catch (e) { -e.n }
  io.println("  -> ${e8}, within 400 ms: ${sw8.elapsed().toMillis() < 400}")
}
`
	want := `1. a plain loop is cancelled at its back edge:
  closed spinWith
  -> timed out true, within 400 ms: true
2. a suspending loop gives its thread up:
  -> the ticker ran within 150 ms: true
3. a lock region is shielded:
  -> timed out true, the region finished: true
4. a close() is shielded:
  slow close finished (true)
  -> timed out true
5. the end waits for a scope's children:
  closed child
  -> timed out true
6. checkCancelled:
  closed checking
  -> timed out true
7. a panic's end waits for a scope's children:
  closed child
  -> panicked: true
8. a loop in a scope body is abandoned when a child fails:
  -> -7, within 400 ms: true`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("loops%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		// a plain loop needs a second thread for the timeout to run on (D145)
		for _, threads := range []string{"2", "4", "8"} {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			run := exec.CommandContext(ctx, exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=4096")
			out, err := run.CombinedOutput()
			cancel()
			got := strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n")
			if err != nil || got != want {
				t.Fatalf("release=%v threads %s: got\n%s\nwant\n%s\n(%v)", release, threads, got, want, err)
			}
		}
		// one thread: only a suspending loop can give it up
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		run := exec.CommandContext(ctx, exe, "fair")
		run.Env = append(os.Environ(), "VELES_THREADS=1")
		out, err := run.CombinedOutput()
		cancel()
		got := strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n"))
		if err != nil || !strings.HasSuffix(got, "the ticker ran within 150 ms: true") {
			t.Fatalf("release=%v, one thread: got %q (%v)", release, got, err)
		}
	}
}

// Many tasks parked on one channel (concurrency review B1, 2026-10-09):
// 50k plain receivers and 50k racing ones that time out, so nodes leave
// from the middle of the list, then 50k sends. Appending once walked the
// whole list and unlinking scanned it, under a lock waiters spun on
// without yielding: parking 100k receivers kept every core busy for
// seconds. Now both are O(1), and the run is bounded in time.
func TestManyWaitersOnOneChannel(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

fun waiter(ch: Channel<i64>, sum: Atomic<i64>) {
  val v = await ch.recv() ?: return
  val _ = sum.update(s => s + v)
}

fun impatient(ch: Channel<i64>, k: i64, missed: Atomic<i64>) {
  race {
    val v = ch.recv() => panic("an impatient receiver got $v")
    sleep(Duration.millis(1 + k % 50)) => { val _ = missed.update(m => m + 1) }
  }
}

fun main() {
  val n: i64 = 50000
  val ch = Channel<i64>()
  val sum = Atomic(value: 0)
  val missed = Atomic(value: 0)
  scope {
    loop (i in 0..<n) {
      async waiter(ch, sum)
      async impatient(ch, i, missed)
    }
    loop {
      if (missed.load() == n) break
      await sleep(Duration.millis(1))
    }
    loop (i in 1..n) ch.send(i)
  }
  io.println("${sum.load()} ${missed.load()}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "waiters.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	want := "1250025000 50000"
	for _, threads := range []string{"1", "8"} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		run := exec.CommandContext(ctx, exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+threads)
		start := time.Now()
		out, err := run.CombinedOutput()
		cancel()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("threads %s: got %q, want %q (%v, after %v)", threads, out, want, err, time.Since(start))
		}
	}
}

// A race with more arms than the runtime once held (concurrency review,
// 2026-10-09): a race kept a fixed array of 16 arms and the compiler set
// no limit, so a 17th arm wrote past it — a crash. A race is now sized to
// its arms.
func TestRaceWithManyArms(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("use io\n\nfun main() {\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "  val c%d = Channel<i64>(capacity: 1)\n", i)
	}
	b.WriteString("  c19.send(19)\n  race {\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "    val v%d = c%d.recv() => io.println(\"arm %d got ${v%d}\")\n", i, i, i, i)
	}
	b.WriteString("    sleep(Duration.seconds(5)) => io.println(\"timed out\")\n  }\n}\n")
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("race%v.exe", release))
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release, Sanitize: *sanitize}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "arm 19 got 19" {
			t.Fatalf("release=%v: got %q (%v), want %q", release, out, err, "arm 19 got 19")
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

fun listen(ch: Channel<i64>): i64 => await ch.recv() ?: -1

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
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
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

fun handle(id: i64): i64 => requestId.withValue(id, () => {
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
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
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

fun kindOf<T>(r: Result<T, IoError>): IoKind => when (r) {
  is Ok(_) => IoKind.Other
  is Err(e) => e.kind
}

fun main() {
  io.println("${kindOf(fs.readFile("no/such/file.txt"))}")
  val l = net.listen() catch (e) { panic("listen: $e") }
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
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	out, err := exec.Command(exe).CombinedOutput()
	got := strings.Fields(string(out))
	want := []string{"NotFound", "AddressInUse", "ConnectionRefused"}
	if err != nil || strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %q, %v; want %v", out, err, want)
	}
}

// A socket closed twice, or used after its close, must not reach the
// socket the system has since given the same number: systems reuse a
// freed number at once. Before the number was taken out on close, the
// second `a.close()` here closed `b` (reproduced on Windows), and a read
// on `a` would have read `b`'s peer.
func TestSocketClosedTwice(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io, net

fun main() throws {
  val l = try net.listen()
  val a = try net.connect("127.0.0.1", l.port())
  val copy = a
  val sa = try l.accept()
  a.close()
  val b = try net.connect("127.0.0.1", l.port())
  val sb = try l.accept()
  a.close()
  copy.close()
  io.println(if (a.read().err) "read after close: error" else "read after close: data")
  try b.writeText("hello")
  io.println("b: ${try sb.read().len()} bytes")
  sa.close()
  sb.close()
  b.close()
  l.close()
  l.close()
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "sockets.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	out, err := exec.Command(exe).CombinedOutput()
	want := "read after close: error\nb: 5 bytes\n"
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", out, err, want)
	}
}

// A connection closed by one task while another is inside a read on it
// (plan A5): the read fails promptly instead of waiting for data that can
// no longer come, the descriptor really closes once the read lets go (the
// peer sees the end of the stream), and while connections come and go on
// eight threads a read never returns bytes sent to a newer connection that
// was given the same descriptor number. The socket keeps a count of the
// operations holding it and closes only when the last one lets go (Go's
// fdMutex); the close wakes the parked operations.
func TestSocketCloseDuringRead(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io, net

// what a read on c ended with
fun readOutcome(c: net.Conn): string => when (val r = c.read()) {
  is Ok(bytes) => if (bytes.isEmpty()) "end" else "data ${bytes.decodeUtf8() ?: "?"}"
  is Err       => "error"
}

fun round(l: net.Listener, i: i64): string throws IoError {
  val client = try net.connect("127.0.0.1", l.port())
  val server = try l.accept()
  var outcome = ""
  scope {
    val reader = async readOutcome(server)
    await sleep(Duration.millis(2))
    server.close()
    // a newer connection, likely on the number just freed, with data waiting
    val other = try net.connect("127.0.0.1", l.port())
    val otherServer = try l.accept()
    try other.writeText("stray $i")
    outcome = await reader
    other.close()
    otherServer.close()
  }
  val peer = readOutcome(client)
  client.close()
  if (outcome != "error" || peer != "end") return "round $i: reader $outcome, peer $peer"
  ""
}

fun main() throws IoError {
  val l = try net.listen()
  val sw = time.Stopwatch.start()
  var bad = 0
  loop (i in 1..200) {
    val problem = try round(l, i)
    if (!problem.isEmpty()) {
      bad += 1
      if (bad <= 3) io.println(problem)
    }
  }
  l.close()
  io.println("bad rounds: $bad, prompt: ${sw.elapsed() < Duration.seconds(20)}")
}
`
	src = strings.Replace(src, "use io, net", "use io, net, time", 1)
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "closeread.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, threads := range []string{"1", "8"} {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		run := exec.CommandContext(ctx, exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+threads)
		out, err := run.CombinedOutput()
		cancel()
		want := "bad rounds: 0, prompt: true\n"
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
			t.Fatalf("VELES_THREADS=%s: got %q, %v; want %q", threads, got, err, want)
		}
	}
}

// os.run starts the program with no shell: every argument arrives as one
// argument, byte for byte. Before, the arguments were joined into a shell
// command line and quoted only when they held a space or a quote, so
// `x;echo pwned` ran a second command. The program runs itself as the child
// and prints what it received.
func TestRunPassesArgumentsVerbatim(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io, os

fun main() throws {
  val args = os.args()
  if (args.at(0) == "child") {
    loop (a in args.drop(1)) io.println("[$a]")
    return
  }
  val tricky = ["a b", "x;echo pwned", "$(echo hi)", "q\"uote", "\\\"", "back\\slash\\", "", "%PATH%", "* ?", "tab\there"]
  val r = try os.run(os.program(), ["child"].concat(tricky))
  io.print(r.stdout.replace("\r\n", "\n"))
  io.println("code ${r.code}")
  when (os.run(os.program(), ["child", "nul\u{0}byte"])) {
    is Ok     => io.println("a NUL was passed")
    is Err(e) => io.println("${e.kind}")
  }
  when (os.run("no-such-program-anywhere")) {
    is Ok     => io.println("ran")
    is Err(e) => io.println("${e.kind}")
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "argv.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	out, err := exec.Command(exe).CombinedOutput()
	want := "[a b]\n[x;echo pwned]\n[$(echo hi)]\n[q\"uote]\n[\\\"]\n[back\\slash\\]\n[]\n[%PATH%]\n[* ?]\n[tab\there]\n" +
		"code 0\nInvalidInput\nNotFound\n"
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

// os.run (D82): input goes in and the pipe closes, standard error is kept
// apart by default, merged or let through on request, no input is an empty
// one; and a child that writes a megabyte of errors before it reads a
// megabyte of input, then writes a megabyte of output, finishes — the
// three pipes are served at once, or it and its parent would wait for
// each other for ever.
func TestRunInputAndStderr(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io, os

fun child(mode: string) {
  when (mode) {
    "echo"    => {
      io.eprintln("to stderr")
      io.print(io.readAll())
    }
    "noinput" => io.println("${io.readAll().len()} bytes in")
    "flood"   => {
      io.eprintln("e".repeat(1000000))
      val input = io.readAll()
      io.println("${input.len()} " + "o".repeat(1000000))
    }
    else      => { }
  }
}

fun main() throws {
  val args = os.args()
  if (args.at(0) == "child") return child(args.at(1) ?: "")
  val me = os.program()
  val a = try os.run(me, ["child", "echo"], input: "hello\nworld\n")
  io.println("echo: stdout ${a.stdout.len()} ${a.stdout.startsWith("hello")} stderr '${a.stderr.trim()}'")
  val b = try os.run(me, ["child", "echo"], input: "abc", stderr: os.Stderr.Merge)
  io.println("merge: ${b.stdout.contains("to stderr")} ${b.stdout.contains("abc")} stderr '${b.stderr}'")
  val c = try os.run(me, ["child", "noinput"])
  io.println("no input: ${c.stdout.trim()}")
  val d = try os.run(me, ["child", "flood"], input: "i".repeat(1000000))
  io.println("flood: stdout ${d.stdout.len()} stderr ${d.stderr.len()} ${d.stdout.startsWith("1000000 ")}")
  val e = try os.run(me, ["child", "echo"], input: "x", stderr: os.Stderr.Inherit)
  io.println("inherit: stderr '${e.stderr}'")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "runio.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatal("os.run did not finish: the pipes deadlocked")
	}
	nl := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
	want := "echo: stdout 12 true stderr 'to stderr'\n" +
		"merge: true true stderr ''\n" +
		"no input: 0 bytes in\n" +
		"flood: stdout 1000009 stderr 1000001 true\n" +
		"inherit: stderr ''\n"
	if got := nl(stdout.String()); err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
	// Inherit let the child's line through to this process's error stream
	if !strings.Contains(stderr.String(), "to stderr") {
		t.Errorf("inherited standard error: %q", stderr.String())
	}
}

// A path holding a NUL byte is refused, never cut short at the NUL on its
// way to the system: `dir/..\0/x` would pass a `..` check as one odd
// segment and then open `dir/..`.
func TestPathsWithNulRefused(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use fs, io, net, os

fun report(r: Result<string, IoError>) {
  when (r) {
    is Ok     => io.println("read")
    is Err(e) => io.println("${e.kind}: ${e.message()}")
  }
}

fun main() throws {
  try fs.writeFile("real.txt", "x")
  report(fs.readFile("real.txt\u{0}.png"))
  report(fs.readFile("real.txt"))
  io.println("${fs.exists("real.txt\u{0}")} ${fs.isFile("real.txt\u{0}")} ${fs.exists("real.txt")}")
  when (fs.rename("real.txt", "..\u{0}/x")) {
    is Ok     => io.println("renamed")
    is Err(e) => io.println("${e.kind}")
  }
  when (net.connect("localhost\u{0}.example", 80)) {
    is Ok     => io.println("connected")
    is Err(e) => io.println("${e.kind}")
  }
  io.println("${os.env("PATH\u{0}x")}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "nul.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	run := exec.Command(exe)
	run.Dir = dir
	out, err := run.CombinedOutput()
	want := "InvalidInput: a path cannot hold a NUL byte: real.txt\\0.png\nread\nfalse false true\nInvalidInput\nInvalidInput\nnull\n"
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

// The standard streams carry bytes unchanged on every platform. On Windows
// they were C text streams: each "\n" printed came out as "\r\n", and stdin
// ended at the first 0x1A byte. readLine still drops a trailing '\r', so a
// CRLF input reads the same everywhere; readAll keeps the bytes as sent.
func TestStandardStreamsAreBytes(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

fun main() {
  io.print("[${io.readLine() ?: "-"}]")
  io.print("[${io.readLine() ?: "-"}]")
  io.print("[${io.readAll()}]")
  io.println("end")
  io.eprintln("err")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "streams.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	run := exec.Command(exe)
	run.Stdin = strings.NewReader("one\r\ntwo\x1athree\nrest\r\nlast")
	var stdout, stderr strings.Builder
	run.Stdout, run.Stderr = &stdout, &stderr
	if err := run.Run(); err != nil {
		t.Fatal(err)
	}
	if want := "[one][two\x1athree][rest\r\nlast]end\n"; stdout.String() != want {
		t.Errorf("stdout %q, want %q", stdout.String(), want)
	}
	if stderr.String() != "err\n" {
		t.Errorf("stderr %q, want %q", stderr.String(), "err\n")
	}
}

// A panic in a debug build prints the calls that led to it (D81) — through
// nested calls, through a function that suspends, and with a note in a
// release build, which keeps no chain.
func TestPanicPrintsCallChain(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io, time

fun check(x: i64): i64 {
  if (x > 2) panic("too big: $x")
  return x
}

fun step(x: i64): i64 {
  await sleep(Duration.millis(1))
  return check(x)
}

fun outer(x: i64): i64 => step(x) + 1

fun main() {
  io.println("${outer(1)}")
  scope {
    val a = async outer(5)
    io.println("${await a}")
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	panicOf := func(release bool) string {
		exe := filepath.Join(dir, "chain.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		var stderr strings.Builder
		cmd := exec.Command(exe)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			t.Fatal("the program should panic")
		}
		return strings.ReplaceAll(stderr.String(), "\r\n", "\n")
	}
	want := "panic: too big: 5\n" +
		"  at main.vs:4:14 in check\n" +
		"  called from main.vs:10:10 in step\n" +
		"  called from main.vs:13:27 in outer\n"
	if got := panicOf(false); got != want {
		t.Errorf("debug build:\n%s\nwant:\n%s", got, want)
	}
	want = "panic: too big: 5\n" +
		"  at main.vs:4:14\n" +
		"  (a debug build shows the call chain)\n"
	if got := panicOf(true); got != want {
		t.Errorf("release build:\n%s\nwant:\n%s", got, want)
	}
}

// D88: a misuse panic in a `@caller_location` std function reports the
// caller's line in both profiles — through a marked function that calls
// another marked one (`toString(radix:)` and its `checkRadix`) too — and
// the debug chain says it once, not again as a "called from".
func TestCallerLocationPanics(t *testing.T) {
	dir := t.TempDir()
	src := `use io

fun bad(xs: MutableList<i64>) {
  xs.swap(0, 7)
}

fun badRadix(n: i64) {
  io.println(n.toString(radix: 1))
}

fun main() {
  val xs: MutableList<i64> = [1, 2, 3]
  io.println(255.toString(radix: 16))
  if (xs.len() == 3) badRadix(5)
  bad(xs)
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	panicOf := func(release bool) string {
		exe := filepath.Join(dir, "caller.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		var stderr strings.Builder
		cmd := exec.Command(exe)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			t.Fatal("the program should panic")
		}
		return strings.ReplaceAll(stderr.String(), "\r\n", "\n")
	}
	want := "panic: toString: radix must be between 2 and 36, got 1\n" +
		"  at main.vs:8:14 in badRadix\n" +
		"  called from main.vs:14:22 in main\n"
	if got := panicOf(false); got != want {
		t.Errorf("debug build:\n%s\nwant:\n%s", got, want)
	}
	want = "panic: toString: radix must be between 2 and 36, got 1\n" +
		"  at main.vs:8:14\n" +
		"  (a debug build shows the call chain)\n"
	if got := panicOf(true); got != want {
		t.Errorf("release build:\n%s\nwant:\n%s", got, want)
	}
}

// Stack overflow is a panic, and the stack is big (D92): a recursion of a
// million frames runs, and one that never ends prints `panic: stack
// overflow` with the size of the stack and — in a debug build — the calls
// it was in, then exits with the panic exit code 101, in main and inside a
// task alike. VELES_STACK sets the size in megabytes.
func TestStackOverflowIsReported(t *testing.T) {
	dir := t.TempDir()
	src := `use io { println }, os

fun depth(n: i64): i64 {
  if (n == 0) return 1
  val r = depth(n - 1)
  (r ^ (r << 1)) + n % 7
}

fun main() {
  val n = (os.args().at(0) ?: "10").toInt() ?: 10
  if (os.args().at(1) == "task") {
    scope {
      val t = async depth(n)
      println("depth $n: ${await t}")
    }
  } else {
    println("depth $n: ${depth(n)}")
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	build := func(release bool) string {
		exe := filepath.Join(dir, "stack.exe")
		if release {
			exe = filepath.Join(dir, "stack-release.exe")
		}
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build failed with exit %d", code)
		}
		return exe
	}
	run := func(exe string, env []string, args ...string) (string, string, int) {
		var stdout, stderr strings.Builder
		cmd := exec.Command(exe, args...)
		cmd.Env = append(os.Environ(), env...)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		code := 0
		if err := cmd.Run(); err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			code = ee.ExitCode()
		}
		return strings.ReplaceAll(stdout.String(), "\r\n", "\n"), strings.ReplaceAll(stderr.String(), "\r\n", "\n"), code
	}
	for _, release := range []bool{false, true} {
		exe := build(release)
		profile := map[bool]string{false: "debug", true: "release"}[release]
		for _, mode := range []string{"main", "task"} {
			// a million frames fit
			out, errText, code := run(exe, nil, "1000000", mode)
			if code != 0 || !strings.HasPrefix(out, "depth 1000000: ") {
				t.Errorf("%s/%s: a million frames: exit %d, stdout %q, stderr %q", profile, mode, code, out, errText)
			}
			// one that never ends is a panic naming the stack
			out, errText, code = run(exe, nil, "1000000000", mode)
			if code != 101 || out != "" {
				t.Errorf("%s/%s: overflow: exit %d, stdout %q (want 101 and nothing)", profile, mode, code, out)
			}
			if !strings.HasPrefix(errText, "panic: stack overflow\n  the stack is 256 MB; VELES_STACK=<megabytes> sets it\n") {
				t.Errorf("%s/%s: overflow report:\n%s", profile, mode, errText)
			}
			if release {
				if !strings.Contains(errText, "(a debug build shows the call chain)") {
					t.Errorf("%s/%s: the release report should say a debug build has the chain:\n%s", profile, mode, errText)
				}
			} else if !strings.Contains(errText, "  in depth\n  called from main.vs:5:11 in depth\n") ||
				!strings.Contains(errText, " calls deep; the chain keeps the first 4096)") {
				t.Errorf("%s/%s: the debug report should name the recursion:\n%s", profile, mode, errText)
			}
		}
		// VELES_STACK sizes it: 1 MB does not hold a million frames and says so
		_, errText, code := run(exe, []string{"VELES_STACK=1"}, "1000000", "main")
		if code != 101 || !strings.Contains(errText, "the stack is 1 MB;") {
			t.Errorf("%s: VELES_STACK=1: exit %d, stderr %q", profile, code, errText)
		}
		// and a value that is not a size is reported and ignored
		out, errText, code := run(exe, []string{"VELES_STACK=lots"}, "10", "main")
		if code != 0 || !strings.HasPrefix(out, "depth 10: ") || !strings.Contains(errText, "VELES_STACK=lots is not a number of megabytes from 1 to 4096; using 256") {
			t.Errorf("%s: VELES_STACK=lots: exit %d, stdout %q, stderr %q", profile, code, out, errText)
		}
	}
}
