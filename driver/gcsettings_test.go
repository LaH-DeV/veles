package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// `[runtime] gc.minHeap` and `gc.goal` in the program's veles.toml set when
// the collector runs; VELES_GC_MIN_HEAP and VELES_GC_GOAL override them
// (D200). Pinned by how many collections one program makes: it keeps ~3 MiB
// live and churns ~100 MiB of garbage.
func TestRuntimeGCManifest(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	const program = `use io

extern "C" {
  fun veles_gc_collections(): i64
}

fun main() {
  val keep: MutableList<string> = []
  loop (i in 0..<40000) keep.push("kept line number $i of the live set")
  var total = 0
  loop (i in 0..<1200000) {
    val garbage = "garbage $i, made long enough to need a heap object"
    total += garbage.len()
  }
  // SAFETY: a runtime hook that takes nothing
  val collections = unsafe { veles_gc_collections() }
  io.println("${keep.len()} ${total > 0} $collections")
}
`
	build := func(name, runtime string) string {
		dir := t.TempDir()
		files := map[string]string{
			"veles.toml": "[package]\nname = \"" + name + "\"\n" + runtime,
			"main.vs":    program,
		}
		for file, text := range files {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		exe := filepath.Join(dir, name+".exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
			t.Fatalf("%s: build failed with exit %d", name, code)
		}
		return exe
	}
	collections := func(exe string, env ...string) int {
		run := exec.Command(exe)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "VELES_GC_") {
				run.Env = append(run.Env, kv)
			}
		}
		run.Env = append(run.Env, env...)
		out, err := run.CombinedOutput()
		f := strings.Fields(string(out))
		if err != nil || len(f) != 3 || f[0] != "40000" || f[1] != "true" {
			t.Fatalf("%s %v: %v, output %q", filepath.Base(exe), env, err, out)
		}
		n, _ := strconv.Atoi(f[2])
		return n
	}

	plain := build("plain", "")
	big := build("big", "\n[runtime]\ngc.minHeap = \"1GiB\"\n")
	if n := collections(plain); n == 0 {
		t.Errorf("default 8 MiB: no collection while churning ~100 MiB")
	}
	if n := collections(big); n != 0 {
		t.Errorf("gc.minHeap = 1GiB: %d collections, want none", n)
	}
	if n := collections(big, "VELES_GC_MIN_HEAP=4MiB"); n == 0 {
		t.Errorf("VELES_GC_MIN_HEAP=4MiB over gc.minHeap = 1GiB: no collection")
	}

	// a goal of 1000% lets the heap grow to 11x the live data between
	// collections, 10% only to 1.1x: far fewer collections
	lazy := build("lazy", "\n[runtime]\ngc.minHeap = \"1MiB\"\ngc.goal = 1000\n")
	few, many := collections(lazy), collections(lazy, "VELES_GC_GOAL=10")
	t.Logf("collections: default %d, goal 1000 %d, goal 10 %d", collections(plain), few, many)
	if few*3 > many {
		t.Errorf("gc.goal = 1000: %d collections, VELES_GC_GOAL=10: %d; want the first well under the second", few, many)
	}
}
