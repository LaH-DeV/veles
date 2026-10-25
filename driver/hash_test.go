package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Hash flooding: map hashes are keyed by a secret drawn at start-up, so the
// hash of one string differs between two runs of one program, and integer
// keys that differ only above the index's bits no longer share a slot — under
// the unkeyed slot (hash & mask) 2^20 * i put all 200 000 keys on slot 0, and
// inserting them took 25 s. Constant maps, whose hashes the compiler can no
// longer know, are hashed before the first global initializer runs, which
// reads one here.
func TestHashFlooding(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

extern "C" {
  fun veles_hash_test_probes(n: i64, stride: i64): i64
  fun veles_hash_test_string(): i64
}

struct P {
  x: i64
  y: string
}

const NAMES: Map<string, i64> = ["one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10]
const PAIRS: Set<(i64, i64)> = [(1, 2), (3, 4)]
const NESTED: Map<string, List<string>> = ["a": ["x", "y"], "b": []]
const POINTS: Map<P, string> = [P(x: 1, y: "a"): "first", P(x: 2, y: "b"): "second"]

val fromConst = NAMES.get("nine") ?: 0

fun main() {
  // SAFETY: test hooks that only compute
  val key = unsafe { veles_hash_test_string() }
  io.println("key $key")
  loop (stride in [1, 1024, 1048576, 4294967296]) {
    // SAFETY: a test hook that only computes
    val run = unsafe { veles_hash_test_probes(100000, stride) }
    io.println("stride $stride short ${run < 200}")
  }
  io.println("${NAMES.get("ten")} ${NAMES.get("eleven")} $fromConst")
  io.println("${(1, 2) in PAIRS} ${(2, 1) in PAIRS} ${NESTED.get("a")}")
  io.println("${POINTS.get(P(x: 2, y: "b"))} ${POINTS.get(P(x: 2, y: "a"))}")
  var m: MutableMap<i64, i64> = [:]
  loop (i in 0..<200000) m.set(i * 1048576, i)
  var s: MutableSet<string> = []
  loop (i in 0..<5000) s.add("k$i")
  io.println("${m.len()} ${m.get(7 * 1048576)} ${s.len()} ${"k4999" in s} ${"k5000" in s}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "stride 1 short true\n" +
		"stride 1024 short true\n" +
		"stride 1048576 short true\n" +
		"stride 4294967296 short true\n" +
		"10 null 9\n" +
		"true false [x, y]\n" +
		"second null\n" +
		"200000 7 5000 true false\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "hash.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		var keys []int64
		for run := 0; run < 2; run++ {
			out, err := exec.Command(exe).Output()
			if err != nil {
				t.Fatalf("release=%v: %v\n%s", release, err, out)
			}
			got := strings.ReplaceAll(string(out), "\r\n", "\n")
			first, rest, _ := strings.Cut(got, "\n")
			k, err := strconv.ParseInt(strings.TrimPrefix(first, "key "), 10, 64)
			if err != nil {
				t.Fatalf("release=%v: first line %q", release, first)
			}
			keys = append(keys, k)
			if rest != want {
				t.Errorf("release=%v: stdout\n%s\nwant\n%s", release, rest, want)
			}
		}
		if keys[0] == keys[1] {
			t.Errorf("release=%v: the hash of one string was %d in two runs: the hash is not keyed per run", release, keys[0])
		}
	}
}
