package driver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The socket reactor (runtime/c/veles_poll.c, plan E3) under load: 300
// connections at once, each 40 round trips through an echo server with a
// task per connection, while two more connections run full duplex — a
// writer task sends 2 MiB of lines as the reader takes the echo, so one
// socket waits to read and to write at the same time and is armed for both.
// Each total is known in advance; a lost wake hangs the run (and the
// timeout fails it), a wrong one changes a sum.
const reactorProgram = `use io, net

// echoes every line until the peer closes
fun serveConn(conn: net.Conn) {
  with c = conn
  loop {
    val line = when (c.readLine(max: 100000)) {
      is Ok(l)  => l ?: break
      is Err(_) => break
    }
    if (c.writeText(line + "\n") is Err) break
  }
}

fun acceptAll(l: net.Listener, n: i64) {
  scope {
    loop (_ in 0..<n) {
      val conn = when (l.accept()) {
        is Ok(c)  => c
        is Err(_) => break
      }
      async serveConn(conn)
    }
  }
}

// 40 round trips: the lengths of what came back
fun pingPong(port: i64, id: i64): i64 throws IoError | io.TooLong {
  with c = try net.connect("127.0.0.1", port)
  var sum: i64 = 0
  loop (k in 0..<40) {
    try c.writeText("client $id line $k\n")
    val back = try c.readLine(max: 1000) ?: return -1
    sum += back.len()
  }
  sum
}

fun writeLines(c: net.Conn, line: string, n: i64) throws IoError {
  loop (_ in 0..<n) try c.writeText(line)
}

// 2048 lines of 1 KiB written by one task while this one reads the echo
fun duplex(port: i64): i64 throws IoError | io.TooLong {
  with c = try net.connect("127.0.0.1", port)
  val line = "x".repeat(1023) + "\n"
  var got: i64 = 0
  scope {
    async writeLines(c, line, 2048)
    loop (_ in 0..<2048) {
      val back = try c.readLine(max: 2000) ?: break
      got += back.len() + 1
    }
  }
  got
}

fun main() throws IoError | io.TooLong {
  val l = try net.listen()
  val port = l.port()
  var pings: i64 = 0
  var streamed: i64 = 0
  scope {
    async acceptAll(l, 302)
    val clients: MutableList<Task<Result<i64, IoError | io.TooLong>>> = []
    loop (id in 0..<300) clients.push(async pingPong(port, id))
    val streams: MutableList<Task<Result<i64, IoError | io.TooLong>>> = []
    loop (_ in 0..<2) streams.push(async duplex(port))
    loop (t in clients) pings += try await t
    loop (t in streams) streamed += try await t
  }
  l.close()
  io.println("pings $pings streamed $streamed")
}
`

// what reactorProgram prints: "client <id> line <k>" for every id and k
func reactorWant() string {
	var pings int
	for id := 0; id < 300; id++ {
		for k := 0; k < 40; k++ {
			pings += len("client ") + len(itoa(id)) + len(" line ") + len(itoa(k))
		}
	}
	return "pings " + itoa(pings) + " streamed " + itoa(2*2048*1024) + "\n"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func runReactorProgram(t *testing.T, threads []string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(reactorProgram), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "reactor.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: true, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	want := reactorWant()
	for _, n := range threads {
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			run := exec.CommandContext(ctx, exe)
			run.Env = append(os.Environ(), "VELES_THREADS="+n)
			out, err := run.CombinedOutput()
			cancel()
			if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
				t.Fatalf("VELES_THREADS=%s, run %d: got %q, %v; want %q", n, i, got, err, want)
			}
		}
	}
}

func TestReactorManyConnections(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	runReactorProgram(t, []string{"1", "2", "8"})
}

// The portable poll() reactor, which macOS uses until kqueue (plan A7),
// built on Linux to test it: the same program, through the backend that
// interrupts a wait in progress whenever a descriptor is armed.
func TestReactorPollFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the poll() reactor is built on Linux to test it (Windows has only its own)")
	}
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	runtimeTestFlags = []string{"-DVELES_POLL_FALLBACK"}
	defer func() { runtimeTestFlags = nil }()
	runReactorProgram(t, []string{"1", "4"})
}

// The timer heap (veles_task.c): 3000 sleepers with scattered deadlines,
// every third cancelled while it waits — a removal from the middle of the
// heap — and the rest each woken no earlier than its deadline and not
// absurdly late. Cancelled ones never print; the totals say who woke.
func TestTimerHeapUnderThreads(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io, time

// sleeps ms; how late it woke, or -1 when it woke early
fun sleeper(ms: i64): i64 {
  val sw = time.Stopwatch.start()
  await sleep(Duration.millis(ms))
  val took = sw.elapsed().toMillis()
  if (took < ms) -1 else took - ms
}

fun main() {
  val kept: MutableList<Task<i64>> = []
  val doomed: MutableList<Task<i64>> = []
  var early: i64 = 0
  var late: i64 = 0
  var woke: i64 = 0
  scope {
    loop (i in 0..<3000) {
      val t = async sleeper((i * 7919) % 400 + 1)
      if (i % 3 == 0) doomed.push(t) else kept.push(t)
    }
    await sleep(Duration.millis(5))
    loop (t in doomed) t.cancel()
  }
  loop (t in kept) {
    val lateBy = await t
    woke += 1
    if (lateBy < 0) early += 1
    if (lateBy > 2000) late += 1
  }
  io.println("woke $woke early $early late $late")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "timers.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, n := range []string{"1", "8"} {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		run := exec.CommandContext(ctx, exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+n)
		out, err := run.CombinedOutput()
		cancel()
		want := "woke 2000 early 0 late 0\n"
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); err != nil || got != want {
			t.Fatalf("VELES_THREADS=%s: got %q, %v; want %q", n, got, err, want)
		}
	}
}
