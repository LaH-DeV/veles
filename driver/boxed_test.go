package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// D198: a tree's variants live behind a hidden pointer, a `indirect`
// variant too, an `inline` one in the value. Nothing a program can see
// changes: a write through a narrowed copy leaves the original alone,
// equality, hashing and printing read the variant, values cross into tasks,
// and the collector keeps every box a value reaches — checked with a
// collection every few kilobytes, in debug and release, on one and four
// threads.
func TestBoxedVariantsKeepValueSemantics(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	src := `use io

sealed trait Expr {
  struct Num {
    var value: i64
  }
  struct Add {
    left:  *Expr
    right: *Expr
    var label: string = ""
  }
  struct Neg {
    inner: *Expr
  }
  inline struct Hole { }
}

fun eval(e: Expr): i64 => when (e) {
  is Num  => e.value
  is Add  => eval(*e.left) + eval(*e.right)
  is Neg  => -eval(*e.inner)
  is Hole => 0
}

// a deep tree, built bottom up
fun chain(n: i64): Expr {
  var e: Expr = Num(value: 1)
  loop (i in 0..<n) {
    val prev = e
    e = if (i % 2 == 0) Add(left: &prev, right: &Num(value: i)) else Neg(inner: &prev)
  }
  e
}

sealed trait Message {
  struct Ping {
    at: i64
  }
  indirect struct Upload {
    name: string
    var size: i64
    tags: List<string>
  }
}

fun treeValue(n: i64): i64 => eval(chain(n))

fun size(m: Message): i64 => when (m) {
  is Ping   => 0
  is Upload => m.size
}

fun main() {
  val two: Expr = Num(value: 2)
  val sum: Expr = Add(left: &Num(value: 3), right: &two, label: "a")
  var copy = sum
  if (copy is Add) copy.label = "changed"
  var n = two
  if (n is Num) n.value += 40
  io.println("$sum | $copy | $two $n | ${sum == copy} ${two == Num(value: 2)}")

  val seen: Set<Expr> = [two, Num(value: 2), Hole(), n]
  io.println("set ${seen.len()}")

  var up: Message = Upload(name: "a", size: 10, tags: ["x"])
  val kept = up
  if (up is Upload) up.size = 99
  io.println("upload ${size(kept)} ${size(up)}")

  val total = scope {
    val tasks: MutableList<Task<i64>> = []
    loop (k in 0..<8) tasks.push(async treeValue(2000 + k))
    awaitAll(tasks).fold(0, (acc, v) => acc + v)
  }
  io.println("trees $total ${eval(chain(3001))}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "Add(left: &Num(value: 3), right: &Num(value: 2), label: a) | Add(left: &Num(value: 3), right: &Num(value: 2), label: changed) | Num(value: 2) Num(value: 42) | false true\n" +
		"set 3\n" +
		"upload 10 99\n"
	for _, release := range profiles() {
		exe := filepath.Join(dir, "prog")
		if release {
			exe += "-release"
		}
		if runtime.GOOS == "windows" {
			exe += ".exe"
		}
		if code := Run(Options{Path: dir, Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		var trees string
		for _, threads := range threadCounts("1", "4") {
			cmd := exec.Command(exe)
			cmd.Env = append(os.Environ(), "VELES_THREADS="+threads, "VELES_GC_THRESHOLD=65536")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("release=%v threads=%s: %v\n%s", release, threads, err, out)
			}
			text := strings.ReplaceAll(string(out), "\r\n", "\n")
			if !strings.HasPrefix(text, want) {
				t.Errorf("release=%v threads=%s:\n%s\nwant it to start with:\n%s", release, threads, text, want)
			}
			last := text[len(want):]
			if trees == "" {
				trees = last
			} else if last != trees {
				t.Errorf("release=%v threads=%s: %q, but %q on the first run", release, threads, last, trees)
			}
		}
	}
}
