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
