package driver

import (
	"context"
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
fun readOutcome(c: net.Conn): string = when (val r = c.read()) {
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

fun outer(x: i64): i64 = step(x) + 1

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
		"  called from main.vs:13:26 in outer\n"
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
