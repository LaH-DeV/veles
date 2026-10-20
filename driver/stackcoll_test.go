package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D188: the header of a mutable collection that never leaves its function
// lives in the function's frame. The same program is built with and without
// that (VELES_NO_STACK_COLLECTIONS) and run under a tiny collection threshold,
// so a header the collector cannot see, a header reused across loop
// iterations, or one that escaped after all shows up as a difference. The
// escaping shapes (returned, passed, stored, captured, pushed into another
// collection) must still be heap objects: they outlive the call that made
// them, and the program reads them afterwards.
func TestStackCollections(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Acc {
  items: MutableList<i64> = []
  names: MutableList<string> = []
  index: MutableMap<string, i64> = [:]
  seen: MutableSet<i64> = MutableSet<i64>()
}

struct Holder {
  items: MutableList<i64>
}

fun churn(n: i64): i64 {
  var total = 0
  loop (i in 0..<n) {
    val s = "garbage-$i-" + "x".repeat(40)
    total += s.len()
  }
  total
}

fun local(n: i64): i64 {
  val xs: MutableList<i64> = []
  loop (i in 0..<n) xs.push(i * 2)
  var s = 0
  loop (x in xs) s += x
  s + xs.len()
}

fun inLoop(): i64 {
  var s = 0
  loop (i in 0..<5) {
    val xs: MutableList<i64> = []
    s += xs.len()
    xs.push(i)
    xs.push(i + 1)
    s += xs.len() * 10
  }
  s
}

fun literal(): i64 {
  val xs: MutableList<i64> = [5, 6, 7]
  xs.push(8)
  (xs.at(0) ?: 0) + (xs.at(3) ?: 0) + xs.len()
}

fun nested(): i64 {
  val outer: MutableList<MutableList<i64>> = []
  loop (i in 0..<4) {
    val inner: MutableList<i64> = []
    inner.push(i)
    inner.push(i * i)
    outer.push(inner)
  }
  var s = 0
  loop (inner in outer) {
    loop (x in inner) s += x
  }
  s
}

fun maps(): i64 {
  val m: MutableMap<string, i64> = [:]
  val seen: MutableSet<i64> = MutableSet<i64>()
  loop (i in 0..<30) {
    m.set("k${i % 12}", i)
    val _ = seen.add(i % 7)
  }
  var s = 0
  loop ((k, v) in m) s += v + k.len()
  s + seen.len() * 1000 + (m.get("k3") ?: 0)
}

fun fields(): i64 {
  val acc = Acc()
  loop (i in 0..<20) {
    acc.items.push(i)
    acc.names.push("n$i")
    acc.index.set("n$i", i)
    val _ = acc.seen.add(i % 5)
  }
  acc.items.len() + acc.names.len() * 10 + acc.index.len() * 100 + acc.seen.len() * 1000 + (acc.index.get("n7") ?: 0)
}

fun underPressure(): i64 {
  val xs: MutableList<string> = []
  val m: MutableMap<i64, string> = [:]
  loop (i in 0..<200) {
    xs.push("item-$i")
    m.set(i, "value-$i")
    val _ = churn(50)
  }
  var s = 0
  loop (x in xs) s += x.len()
  loop ((k, v) in m) s += v.len() + k
  s
}

fun returned(): List<i64> {
  val out: MutableList<i64> = []
  loop (i in 0..<6) out.push(i)
  out
}

fun fill(xs: MutableList<i64>) {
  xs.push(100)
  xs.push(200)
}

fun filled(): List<i64> {
  val xs: MutableList<i64> = []
  fill(xs)
  xs
}

fun stored(): Holder {
  val xs: MutableList<i64> = [1, 2, 3]
  Holder(items: xs)
}

fun pushedInto(sink: MutableList<MutableList<i64>>) {
  val xs: MutableList<i64> = [9, 8]
  sink.push(xs)
}

fun captured(): i64 {
  val xs: MutableList<i64> = []
  val add = (x: i64) => xs.push(x)
  add(4)
  add(5)
  xs.len() * 10 + (xs.at(1) ?: 0)
}

fun copies(): i64 {
  val xs: MutableList<i64> = [3, 1, 2]
  val sorted = xs.sorted()
  val part = xs.slice(1, 3)
  val dup = xs.toList()
  xs.addAll(dup)
  (sorted.at(0) ?: 0) + part.len() * 10 + xs.len() * 100
}

fun main() {
  io.println("local ${local(10)} ${local(0)}")
  io.println("loop ${inLoop()}")
  io.println("literal ${literal()}")
  io.println("nested ${nested()}")
  io.println("maps ${maps()}")
  io.println("fields ${fields()}")
  io.println("pressure ${underPressure()}")
  val r = returned()
  val f = filled()
  val h = stored()
  val kept: MutableList<MutableList<i64>> = []
  pushedInto(kept)
  val c = churn(2000)
  io.println("escaped ${r} ${f} ${h.items} ${kept} $c")
  io.println("captured ${captured()}")
  io.println("copies ${copies()}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var outputs []string
	for _, variant := range []struct {
		name string
		off  bool
	}{{"stack", false}, {"heap", true}} {
		for _, release := range []bool{false, true} {
			if variant.off {
				t.Setenv("VELES_NO_STACK_COLLECTIONS", "1")
			} else {
				t.Setenv("VELES_NO_STACK_COLLECTIONS", "")
			}
			exe := filepath.Join(dir, variant.name+".exe")
			if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
				t.Fatalf("%s release=%v: build failed with exit %d", variant.name, release, code)
			}
			cmd := exec.Command(exe)
			var env []string
			for _, kv := range os.Environ() {
				if !strings.HasPrefix(kv, "VELES_GC_") {
					env = append(env, kv)
				}
			}
			cmd.Env = append(env, "VELES_GC_THRESHOLD=4096", "VELES_GC_POISON=1")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s release=%v: %v\n%s", variant.name, release, err, out)
			}
			outputs = append(outputs, strings.ReplaceAll(string(out), "\r\n", "\n"))
		}
	}
	want := `local 100 0
loop 100
literal 17
nested 20
maps 7335
fields 7227
pressure 23080
escaped [0, 1, 2, 3, 4, 5] [100, 200] [1, 2, 3] [[9, 8]] 104890
captured 25
copies 621
`
	for i, got := range outputs {
		if got != outputs[0] {
			t.Errorf("run %d differs from run 0:\n%s\nvs\n%s", i, got, outputs[0])
		}
	}
	if outputs[0] != want {
		t.Errorf("output\n%s\nwant\n%s", outputs[0], want)
	}
}
