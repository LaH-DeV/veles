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
  val n = atomic(0)
  val f = atomic(0.0)
  val w = atomic(0 as u16)
  val flips = atomic(false)
  val s = atomic("")
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
