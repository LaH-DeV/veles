package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D111: a value holding tasks, received with `with`. Its tasks are
// cancelled and joined at the end of the block, before its close(); a task
// that ends with an Err fails the block and the error comes out of it; a
// panic in one continues in the block; a value built by a suspending
// helper hands its tasks over the same way; a field argument that fails
// after an `async` leaves no task started. Debug and release, on 1, 2, 4
// and 8 threads.
func TestHeldTasks(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

error Broken { }

struct Server {
  name: string
  serving: Task<()>
  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
}

fun serve(name: string, log: Channel<string>) {
  announce(name, log)
  loop {
    await sleep(Duration.millis(1))
  }
}

fun announce(name: string, log: Channel<string>) {
  log.send("started $name")
}

fun serving(name: string, log: Channel<string>): Server {
  await sleep(Duration.millis(1))   // a suspending helper: a task of its own underneath
  Server(name, serving: async serve(name, log))
}

fun breaks(): string throws Broken {
  await sleep(Duration.millis(5))
  throw Broken()
}

struct Fragile {
  job: Task<Result<string, Broken>>
}

sealed trait Slot
struct Busy : Slot {
  job: Task<Result<string, Broken>>
}
struct Free : Slot { }

struct Maybe {
  job: Task<Result<string, Broken>>?
}

fun fails(): i64 throws Broken {
  throw Broken()
}

struct Tracked {
  job: Task<()>
  n: i64
}

fun noisy() {
  io.println("noisy started")
}

fun built(): Tracked throws Broken => Tracked(job: async noisy(), n: try fails())

fun building(): string throws Broken {
  with t = try built()
  "n=${t.n}"
}

fun failingBlock(): string throws Broken {
  with f = Fragile(job: async breaks())
  await sleep(Duration.millis(500))
  "not reached"
}

fun failingVariant(): string throws Broken {
  val slot: Slot = Free()
  with s = busy()
  await sleep(Duration.millis(500))
  "not reached ${slot is Free}"
}

fun busy(): Slot => Busy(job: async breaks())

fun failingNullable(): string throws Broken {
  with m = Maybe(job: async breaks())
  await sleep(Duration.millis(500))
  "not reached"
}

fun panics() {
  await sleep(Duration.millis(2))
  panic("held task panicked")
}

struct Doomed {
  job: Task<()>
}

fun main() {
  val log = Channel<string>(capacity: 16)
  with (srv = serving("api", log)) {
    io.println("${await log.recv() ?: "-"} in ${srv.name}")
  }
  io.println("after the block")
  io.println("block: ${failingBlock() catch (e) { "failed: $e" }}")
  io.println("variant: ${failingVariant() catch (e) { "failed: $e" }}")
  io.println("nullable: ${failingNullable() catch (e) { "failed: $e" }}")
  io.println("built: ${building() catch (e) { "no task started: $e" }}")
  when (val r = gather { async panicking() }) {
    is Ok  => io.println("no panic")
    is Err => io.println("panic: ${r.message()}")
  }
}

fun panicking() {
  with d = Doomed(job: async panics())
  await sleep(Duration.millis(500))
  io.println("not reached")
}
`
	want := "started api in api\nclose api\nafter the block\n" +
		"block: failed: Broken\n" +
		"variant: failed: Broken\n" +
		"nullable: failed: Broken\n" +
		"built: no task started: Broken\n" +
		"panic: held task panicked\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("held-%v.exe", release))
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
