package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// An empty list owns no element storage until the first push: `[]` costs the
// header and nothing else. Pinned by live bytes after a collection (a header
// is under 80 bytes; a header plus the old four-element buffer was over 100),
// and by growing such lists from nothing, through reserve, append and copy.
func TestEmptyListNoStorage(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

extern "C" {
  fun veles_gc_collect()
  fun veles_gc_live_bytes(): i64
}

fun live(): i64 {
  // SAFETY: a runtime hook that takes nothing
  unsafe { veles_gc_collect() }
  // SAFETY: a runtime hook that takes nothing
  unsafe { veles_gc_live_bytes() }
}

struct Libs {
  libs:  List<string> = []
  paths: List<string> = []
  names: Set<string> = []
  index: Map<string, i64> = [:]
}

fun main() {
  val n = 20000
  val s0 = live()
  val libs = MutableList<Libs>.make(n, _ => Libs())
  val perLibs = (live() - s0) / n
  val l0 = libs.at(0)?.libs ?: panic("none")
  var inside = 0
  loop (x in l0) inside += x.len() + 1
  io.println("shared empties: ${perLibs < 80} ${l0.len()} ${inside} ${libs.at(7)?.names?.contains("x")} ${libs.at(9)?.index?.get("x")} ${l0 == (libs.at(5)?.paths ?: l0)}")
  val before = live()
  val lists = MutableList<MutableList<i64>>.make(n, _ => [])
  val per = (live() - before) / n
  io.println("small: ${per < 80}")
  loop (i in 0..<n) {
    if (i % 3 == 0) {
      lists.at(i)?.push(i)
    }
  }
  var total = 0
  loop (l in lists) {
    total += l.len()
  }
  io.println("pushed: $total")
  val a: MutableList<i64> = []
  a.reserve(10)
  a.push(7)
  val b: MutableList<i64> = []
  val none: List<i64> = []
  b.addAll(a.toList())
  b.addAll(none)
  val c = b.toList()
  val e: MutableList<i64> = []
  io.println("grown: ${a.len()} ${b.len()} ${c.len()} ${e.toList().len()} ${e.pop()}")
  val s: MutableList<u8> = []
  io.println("bytes: ${s.toList().len()} ${"".bytes().len()}")

  val m0 = live()
  val maps = MutableList<MutableMap<string, i64>>.make(n, _ => [:])
  val sets = MutableList<MutableSet<i64>>.make(n, _ => MutableSet<i64>())
  val perMap = (live() - m0) / n
  io.println("small maps and sets: ${perMap < 300}")
  val mm: MutableMap<string, i64> = [:]
  io.println("empty map: ${mm.len()} ${mm.get("a")} ${mm.containsKey("a")} ${mm.remove("a")} ${mm.toMap().len()}")
  mm.clear()
  mm.set("a", 1)
  mm.set("b", 2)
  mm.remove("a")
  loop (i in 0..<100) mm.set("k$i", i)
  val cp = mm.toMap()
  mm.clear()
  mm.set("z", 26)
  io.println("map grown: ${cp.len()} ${cp.get("k99")} ${mm.len()} ${mm.get("z")} ${mm.get("b")}")
  val rm: MutableMap<i64, i64> = [:]
  rm.reserve(100)
  loop (i in 0..<100) rm.set(i, i * i)
  io.println("reserved: ${rm.len()} ${rm.get(99)}")
  val ss: MutableSet<i64> = MutableSet<i64>()
  io.println("empty set: ${ss.len()} ${ss.contains(1)} ${ss.toList().len()}")
  loop (i in 0..<50) ss.add(i % 10)
  io.println("set grown: ${ss.len()} ${ss.contains(9)} ${ss.contains(10)}")
  // maps and sets of every size around the point where the hash index starts
  var bad = 0
  loop (size in 0..<40) {
    val q: MutableMap<i64, i64> = [:]
    val qs: MutableSet<i64> = MutableSet<i64>()
    loop (k in 0..<size) {
      q.set(k * 7, k)
      qs.add(k * 7)
    }
    loop (k in 0..<size) {
      if (k % 2 == 1) {
        q.remove(k * 7)
        qs.remove(k * 7)
      }
    }
    loop (k in 0..<size) {
      if (q.containsKey(k * 7) == (k % 2 == 1)) bad += 1
      if (qs.contains(k * 7) == (k % 2 == 1)) bad += 1
    }
    loop (k in 0..<size) {
      if (k % 2 == 1) {
        q.set(k * 7 + 1, k)
        qs.add(k * 7 + 1)
      }
    }
    if (q.len() != size || qs.len() != size) bad += 1
    var total = 0
    loop (k in 0..<size) {
      total += (q.get(k * 7) ?: q.get(k * 7 + 1) ?: -1000000) - k
    }
    if (total != 0) bad += 1
    val copy = q.toMap()
    q.clear()
    q.set(1, 1)
    if (copy.len() != size || q.len() != 1 || q.get(1) != 1) bad += 1
  }
  io.println("map sizes: $bad bad")
  val sb = StringBuilder()
  io.println("builder: ${sb.isEmpty()} '${sb.toString()}'")
  sb.append("ab")
  sb.append("")
  sb.appendLine("cd")
  io.println("builder text: ${sb.len()} ${sb.toString().replace("\n", "|")}")
  val e1 = ""
  val w = "word"
  val z: MutableList<i64> = []
  val frozen = z.toList()
  z.push(1)
  val frozen2 = z.toList()
  z.clear()
  io.println("toList: ${frozen.len()} ${frozen2.len()} ${z.len()} ${frozen == frozen2}")
  var joined = 0
  loop (i in 0..<1000) {
    val t = "${i}" + e1
    val u = e1 + "${i}"
    val v = "${i}${e1}"
    joined += t.len() + u.len() + v.len() + (w + e1).len()
  }
  io.println("joined: $joined ${w + e1 + e1} ${e1 + e1 + w}")
  io.println("concat: ${(e1 + w).len()} ${(w + e1).len()} ${(e1 + e1).len()} ${w + e1 + w} ${maps.len()} ${sets.len()}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "shared empties: true 0 0 false null true\n" +
		"small: true\npushed: 6667\ngrown: 1 1 1 0 null\nbytes: 0 0\n" +
		"small maps and sets: true\n" +
		"empty map: 0 null false false 0\n" +
		"map grown: 101 99 1 26 null\n" +
		"reserved: 100 9801\n" +
		"empty set: 0 false 0\n" +
		"set grown: 10 true false\n" +
		"map sizes: 0 bad\n" +
		"builder: true ''\n" +
		"builder text: 5 abcd|\n" +
		"toList: 0 1 0 false\n" +
		"joined: 12670 word word\n" +
		"concat: 4 4 0 wordword 20000 20000\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "emptylist.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: stdout\n%s\nwant\n%s", release, got, want)
		}
	}
}
