package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D108: a `race` send arm puts its value in the channel exactly when it
// wins. Producers race a send against an immediate timer, consumers
// receive plainly and through a race of their own (a send arm meeting a
// receive arm of another race on a rendezvous channel), and every value
// a producer counted as sent arrives once, none it counted as dropped
// arrives at all. A send arm on a closed channel panics. Run on 1 to 8
// threads, five times each.
func TestRaceSendArms(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io { println }

// every other offer waits a millisecond for a consumer; the rest give up
// at once unless one is already waiting
fun offer(c: Channel<i64>, v: i64, patience: Duration): bool {
  race {
    c.send(v)       => true
    sleep(patience) => false
  }
}

fun produce(id: i64, c: Channel<i64>, sent: Atomic<i64>, dropped: Atomic<i64>) {
  loop (i in 0..<2000) {
    val patience = if (i % 2 == 0) Duration.zero else Duration.millis(1)
    if (offer(c, id * 100000 + i, patience)) {
      sent.update(n => n + 1)
    } else {
      dropped.update(n => n + 1)
    }
  }
}

fun producers(c: Channel<i64>, sent: Atomic<i64>, dropped: Atomic<i64>) {
  scope {
    loop (id in 0..<4) {
      async produce(id, c, sent, dropped)
    }
  }
  c.close()
}

fun plainConsumer(c: Channel<i64>, out: Channel<i64>) {
  loop {
    val v = await c.recv() ?: break
    out.send(v)
  }
}

fun racingConsumer(c: Channel<i64>, out: Channel<i64>) {
  loop {
    val v = race {
      val x = c.recv()          => x ?: -1
      sleep(Duration.millis(1)) => -2
    }
    if (v == -1) break
    if (v >= 0) out.send(v)
  }
}

fun main() {
  val c = Channel<i64>()
  val out = Channel<i64>(capacity: 8000)
  val sent = Atomic(value: 0)
  val dropped = Atomic(value: 0)
  scope {
    async producers(c, sent, dropped)
    async plainConsumer(c, out)
    async racingConsumer(c, out)
  }
  out.close()
  val seen: MutableSet<i64> = MutableSet<i64>()
  var dup = 0
  loop {
    val v = out.tryRecv() ?: break
    if (seen.contains(v)) dup += 1
    seen.add(v)
  }
  println("attempts ${sent.load() + dropped.load()}")
  println("received what was sent ${seen.len() == sent.load()} duplicates ${dup}")
  val full = Channel<i64>(capacity: 1)
  full.send(1)
  val lost = race {
    full.send(2)         => "sent"
    sleep(Duration.zero) => "not sent"
  }
  println("full ${lost} ${await full.recv()} ${full.tryRecv()}")
  val closed = Channel<i64>(capacity: 1)
  closed.close()
  val g = gather {
    async sendClosed(closed)
  }
  println("closed panics ${g is Err}")
}

fun sendClosed(c: Channel<i64>) {
  race {
    c.send(1)                    => println("not reached")
    sleep(Duration.seconds(5))   => println("not reached")
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "attempts 8000\nreceived what was sent true duplicates 0\nfull not sent 1 null\nclosed panics true\n"
	exe := filepath.Join(dir, "race.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Sanitize: *sanitize}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, threads := range []string{"1", "2", "4", "8"} {
		for run := 0; run < 5; run++ {
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
