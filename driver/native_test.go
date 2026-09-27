package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// The `[native]` table (D67) links C code the package's extern blocks call:
// an object file named directly under `libs`, and a static archive found
// by name through `lib-paths` under `static-libs`. The C side is compiled
// here with the same clang, so the test needs no system library.
func TestNativeLinking(t *testing.T) {
	clang, err := findClang()
	if err != nil {
		t.Skip("clang not available:", err)
	}
	ar := filepath.Join(filepath.Dir(clang), "llvm-ar")
	if _, err := exec.LookPath(ar); err != nil {
		t.Skip("llvm-ar not next to clang")
	}
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	write("c/twice.c", "#include <stdint.h>\nint64_t twice(int64_t x) { return 2 * x; }\n")
	write("c/plus.c", "#include <stdint.h>\nint64_t plus_one(int64_t x) { return x + 1; }\n")
	// C calling back into Veles by name: an `extern "C" fun` is a C symbol
	// the program never mentions itself (D69)
	write("c/back.c", "#include <stdint.h>\nint64_t tripled(int64_t);\nint64_t call_twice(int64_t x) { return tripled(tripled(x)); }\n")
	run(clang, "-c", "c/twice.c", "-o", "c/twice.o")
	run(clang, "-c", "c/plus.c", "-o", "c/plus.o")
	run(clang, "-c", "c/back.c", "-o", "c/back.o")
	run(ar, "rcs", "c/libplus.a", "c/plus.o")
	write("veles.toml", "[package]\nname = \"nat\"\n\n[native]\nlibs = [\"c/twice.o\", \"c/back.o\"]\nstatic-libs = [\"plus\"]\nlib-paths = [\"c\"]\n")
	write("main.vs", `use io

extern "C" {
  fun twice(x: i64): i64
  fun plus_one(x: i64): i64
  fun call_twice(x: i64): i64
}

extern "C" fun tripled(x: i64): i64 = x * 3

fun main() {
  io.println("${unsafe { twice(plus_one(20)) }} ${unsafe { call_twice(2) }}")
}
`)
	exe := filepath.Join(dir, "nat.exe")
	if code := Run(Options{Path: dir, Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	out, err := exec.Command(exe).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "42 18" {
		t.Fatalf("got %q, %v; want 42 18", out, err)
	}

	// a static library nobody has is an error that says where it looked
	write("veles.toml", "[package]\nname = \"nat\"\n\n[native]\nstatic-libs = [\"nosuchlib\"]\n")
	if _, err := nativeFlags(clang, mustManifests(t, dir)); err == nil || !strings.Contains(err.Error(), "no static archive for \"nosuchlib\"") {
		t.Errorf("missing archive: %v", err)
	}
}

// A foreign call runs in a safe region (D66): while one task sits in C
// for a second and a half, a task on another thread keeps allocating and
// the collections it triggers go ahead instead of waiting for the C call
// to return.
func TestForeignCallDoesNotStallCollection(t *testing.T) {
	exe := buildBlocking(t, `use io, time

extern "C" {
  fun block_ms(ms: i64)
}

fun blocker(): i64 {
  unsafe { block_ms(1500) }
  1
}

fun churn(): i64 {
  var total = 0
  loop (i in 0..<200000) {
    val xs = [i, i + 1, i + 2]
    total += xs.len()
  }
  total
}

fun main() {
  val sw = time.Stopwatch.start()
  scope {
    val _ = async blocker()
    val n = await async churn()
    io.println("churn $n, while C blocked: ${sw.elapsed() < Duration.millis(1000)}")
  }
}
`)
	run := exec.Command(exe)
	run.Env = append(os.Environ(), "VELES_THREADS=4", "VELES_GC_THRESHOLD=200000")
	out, err := run.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "churn 600000, while C blocked: true" {
		t.Fatalf("got %q, %v", out, err)
	}
}

// A blocking foreign call gives its run queue back (D66): with one thread,
// three tasks blocked in C at once still overlap, and a task computing and
// one sleeping on a timer run meanwhile — the monitor (asleep after the
// quiet opening sleep, woken by the first blocking call) hands the queue
// of the blocked thread to a spare, and the thread that blocked becomes a
// spare once its task parks.
func TestBlockingCallHandsOffItsThread(t *testing.T) {
	exe := buildBlocking(t, `use io, time

extern "C" {
  fun block_ms(ms: i64)
}

fun blocker(): i64 {
  unsafe { block_ms(600) }
  1
}

fun churn(): i64 {
  var total = 0
  loop (i in 0..<100000) {
    val xs = [i, i + 1, i + 2]
    total += xs.len()
  }
  total
}

fun ticker(): i64 {
  var ticks = 0
  loop (_ in 0..<5) {
    await sleep(Duration.millis(20))
    ticks += 1
  }
  ticks
}

fun main() {
  await sleep(Duration.millis(300))  // long enough for the monitor to go to sleep
  val sw = time.Stopwatch.start()
  scope {
    val blocked = [async blocker(), async blocker(), async blocker()]
    val n = await async churn()
    val ticks = await async ticker()
    val early = sw.elapsed() < Duration.millis(500)
    var done = 0
    loop (b in blocked) done += await b
    io.println("churn $n, ticks $ticks while C blocked: $early; $done calls overlapped: ${sw.elapsed() < Duration.millis(1500)}")
  }
}
`)
	for _, threads := range []string{"1", "2"} {
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=200000")
		out, err := run.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "churn 300000, ticks 5 while C blocked: true; 3 calls overlapped: true" {
			t.Fatalf("threads %s: got %q, %v", threads, out, err)
		}
	}
}

// buildBlocking builds a package whose main.vs may call block_ms(ms), a C
// function that sleeps its thread.
func buildBlocking(t *testing.T, main string) string {
	t.Helper()
	clang, err := findClang()
	if err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("c/block.c", `#include <stdint.h>
#ifdef _WIN32
#include <windows.h>
void block_ms(int64_t ms) { Sleep((DWORD)ms); }
#else
#include <unistd.h>
void block_ms(int64_t ms) { usleep((useconds_t)(ms * 1000)); }
#endif
`)
	cmd := exec.Command(clang, "-c", "c/block.c", "-o", "c/block.o")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clang: %v\n%s", err, out)
	}
	write("veles.toml", "[package]\nname = \"blk\"\n\n[native]\nlibs = [\"c/block.o\"]\n")
	write("main.vs", main)
	exe := filepath.Join(dir, "blk.exe")
	if code := Run(Options{Path: dir, Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	return exe
}

func mustManifests(t *testing.T, dir string) []*sema.Manifest {
	t.Helper()
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.NativeManifests()
}
