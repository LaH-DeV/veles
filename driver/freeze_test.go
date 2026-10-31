package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D188: freeze() turns a mutable collection immutable in place. The result is
// the same object as a List / Map / Set; every change through an old name
// panics; a pointer made before the freeze never reaches the frozen contents,
// because the storage is copied when one was made.
func TestFreeze(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	run := func(t *testing.T, body string, release bool) (string, string, error) {
		t.Helper()
		src := "use io\n\n" + body
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		exe := filepath.Join(dir, "freeze.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build failed with exit %d\n%s", code, src)
		}
		var out, errb strings.Builder
		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(), "VELES_GC_THRESHOLD=4096")
		cmd.Stdout, cmd.Stderr = &out, &errb
		err := cmd.Run()
		return strings.ReplaceAll(out.String(), "\r\n", "\n"), strings.ReplaceAll(errb.String(), "\r\n", "\n"), err
	}

	t.Run("works", func(t *testing.T) {
		t.Parallel()
		const body = `
struct Report {
  protected var lines: MutableList<string> = []

  fun add(line: string): bool {
    if (this.lines.isFrozen()) return false
    this.lines.push(line)
    true
  }

  fun finish(): List<string> => this.lines.freeze()
}

struct Acc {
  items: MutableList<i64> = []
}

fun fill(xs: MutableList<i64>) {
  xs.push(100)
  xs.push(200)
}

// not provable by the move rule: the list is also kept by the Acc
fun built(): List<i64> {
  val acc = Acc()
  val out: MutableList<i64> = []
  fill(out)
  acc.items.push(1)
  out.freeze()
}

fun grown(): List<i64> {
  val xs: MutableList<i64> = []
  loop (i in 0..<1000) xs.push(i)
  xs.freeze()
}

fun churn(n: i64): i64 {
  var s = 0
  loop (i in 0..<n) {
    val t = "garbage-$i-" + "y".repeat(30)
    s += t.len()
  }
  s
}

fun main() {
  val a = built()
  val b = grown()
  val c = churn(3000)
  var sum = 0
  loop (x in b) sum += x
  io.println("built ${a} grown ${b.len()} ${sum} ${churn(1) > 0 && c > 0}")

  // a name that still sees it as mutable can read it
  val xs: MutableList<i64> = [1, 2, 3]
  val frozen = xs.freeze()
  io.println("reads ${xs.len()} ${xs.at(1)} ${frozen} ${frozen == [1, 2, 3]}")

  // a pointer made before the freeze does not reach the frozen contents
  val ys: MutableList<i64> = [10, 20, 30]
  val p = ys.ref(0) ?: return
  val fy = ys.freeze()
  *p = 99
  io.println("pointer ${fy} ${*p}")

  // maps and sets
  val m: MutableMap<string, i64> = ["a": 1, "b": 2]
  val q = m.ref("a") ?: return
  val fm = m.freeze()
  *q = 50
  io.println("map ${fm.get("a")} ${fm.len()} ${m.get("b")} ${fm.containsKey("b")}")
  val s: MutableSet<i64> = MutableSet<i64>()
  val _ = s.add(4)
  val fs = s.freeze()
  io.println("set ${fs.contains(4)} ${fs.len()}")

  // asking: before, after, through another name, and on a protected field
  val z: MutableList<i64> = [1]
  val alias = z
  val before = z.isFrozen()
  val fz = z.freeze()
  io.println("asked ${before} ${z.isFrozen()} ${alias.isFrozen()} ${fz.len()}")
  val zm: MutableMap<string, i64> = [:]
  val zs: MutableSet<i64> = MutableSet<i64>()
  val askedBefore = "${zm.isFrozen()} ${zs.isFrozen()}"
  val _ = zm.freeze()
  val _ = zs.freeze()
  io.println("asked ${askedBefore} ${zm.isFrozen()} ${zs.isFrozen()}")
  val r = Report()
  val first = r.add("a")
  val lines = r.finish()
  io.println("report ${first} ${r.add("b")} ${lines}")
}
`
		want := "built [100, 200] grown 1000 499500 true\n" +
			"reads 3 2 [1, 2, 3] true\n" +
			"pointer [10, 20, 30] 99\n" +
			"map 1 2 2 true\n" +
			"set true 1\n" +
			"asked false true true 1\n" +
			"asked false false true true\n" +
			"report true false [a]\n"
		for _, release := range profiles() {
			out, errs, err := run(t, body, release)
			if err != nil {
				t.Fatalf("release=%v: %v\n%s\n%s", release, err, out, errs)
			}
			if out != want {
				t.Errorf("release=%v: stdout\n%s\nwant\n%s", release, out, want)
			}
		}
	})

	// every change through an old name is a panic that says why
	for _, c := range []struct{ name, setup, use string }{
		{"push", `val xs: MutableList<i64> = [1, 2]`, `xs.push(3)`},
		{"push into a full list", `val xs: MutableList<i64> = [1, 2]`, `loop (i in 0..<10) xs.push(i)`},
		{"pop", `val xs: MutableList<i64> = [1, 2]`, `val _ = xs.pop()`},
		{"clear", `val xs: MutableList<i64> = [1, 2]`, `xs.clear()`},
		{"reserve", `val xs: MutableList<i64> = [1, 2]`, `xs.reserve(50)`},
		{"addAll", `val xs: MutableList<i64> = [1, 2]`, `xs.addAll([7, 8])`},
		{"set", `val xs: MutableList<i64> = [1, 2]`, `xs.set(0, 9)`},
		{"element assignment", `val xs: MutableList<i64> = [1, 2]`, `*(xs.ref(1) ?: return) += 5`},
		{"a pointer", `val xs: MutableList<i64> = [1, 2]`, `val _ = xs.ref(0)`},
		{"a loop with pointers", `val xs: MutableList<i64> = [1, 2]`, `loop (&x in xs) *x += 1`},
		{"map set", `val xs: MutableMap<string, i64> = ["a": 1]`, `xs.set("b", 2)`},
		{"map remove", `val xs: MutableMap<string, i64> = ["a": 1]`, `val _ = xs.remove("a")`},
		{"map clear", `val xs: MutableMap<string, i64> = ["a": 1]`, `xs.clear()`},
		{"map pointer", `val xs: MutableMap<string, i64> = ["a": 1]`, `val _ = xs.ref("a")`},
		{"set add", `val xs: MutableSet<i64> = [1]`, `val _ = xs.add(2)`},
		{"set remove", `val xs: MutableSet<i64> = [1]`, `val _ = xs.remove(1)`},
		{"freezing twice", `val xs: MutableList<i64> = [1, 2]`, `val _ = xs.freeze()`},
	} {
		t.Run("panics: "+c.name, func(t *testing.T) {
			t.Parallel()
			body := "fun main() {\n  " + c.setup + "\n  val frozen = xs.freeze()\n  io.println(\"frozen ${frozen}\")\n  " + c.use + "\n  io.println(\"not reached\")\n}\n"
			for _, release := range profiles() {
				out, errs, err := run(t, body, release)
				if err == nil {
					t.Fatalf("release=%v: changing a frozen collection should panic\n%s", release, out)
				}
				if !strings.HasPrefix(out, "frozen ") || strings.Contains(out, "not reached") {
					t.Errorf("release=%v: stdout %q", release, out)
				}
				if !strings.Contains(errs, "was frozen by freeze()") {
					t.Errorf("release=%v: stderr\n%s\nwant the frozen panic", release, errs)
				}
			}
		})
	}

	t.Run("immutable receivers are refused", func(t *testing.T) {
		t.Parallel()
		for _, src := range []string{
			"fun main() {\n  val xs: List<i64> = [1]\n  val _ = xs.freeze()\n}\n",
			"fun main() {\n  val m: Map<string, i64> = [\"a\": 1]\n  val _ = m.freeze()\n}\n",
		} {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte("use io\n\n"+src), 0o644); err != nil {
				t.Fatal(err)
			}
			if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "check"}); code == 0 {
				t.Errorf("freeze on an immutable collection should be an error:\n%s", src)
			}
		}
	})
}
