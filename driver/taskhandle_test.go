package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D141: in a fail-fast scope, awaiting a throwing child gives its value,
// and a child that fails is never awaited with one — the body is abandoned
// (an `is Err` arm could not run before either, and a `try` there was
// dead). The scope's failure also stops the suspending call the body was
// inside, so nothing it runs prints after the scope rethrew (D3; it ran on
// before). A race arm on a failed child never wins; a helper that awaits a
// failed handle is cancelled with the body; a child that returns a Result
// without `throws` hands its Err over as a value. Debug and release, on 1,
// 2, 4 and 8 threads.
func TestTaskHandles(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

error Boom { n: i64 }

fun work(n: i64, ms: i64): i64 throws Boom {
  await sleep(Duration.millis(ms))
  if (n < 0) throw Boom(n)
  n * 10
}

fun pool(): i64 throws Boom {
  var total: i64 = 0
  scope {
    val tasks: MutableList<Task<i64>> = []
    loop (i in 1..4) tasks.push(async work(i, 4 - i))
    loop (t in tasks) total += await t
  }
  total
}

fun failing(): i64 throws Boom {
  scope {
    val ok = async work(1, 30)
    val bad = async work(-2, 1)
    val a = await bad
    io.println("not reached: $a")
    return await ok
  }
}

fun helper(t: Task<i64>, log: Channel<string>): i64 {
  val v = await t
  log.send("helper got $v")
  v
}

fun ticking(log: Channel<string>): i64 {
  loop (_ in 0..<50) {
    await sleep(Duration.millis(2))
  }
  log.send("ticking finished")
  1
}

fun orphan(log: Channel<string>): i64 throws Boom {
  scope {
    async work(-3, 5)
    val n = ticking(log)
    io.println("not reached: $n")
  }
  0
}

fun viaHelper(log: Channel<string>): i64 throws Boom {
  scope {
    val bad = async work(-4, 1)
    return helper(bad, log)
  }
}

fun racing(): string throws Boom {
  scope {
    val bad = async work(-5, 1)
    race {
      val v = await bad              => return "won with $v"
      sleep(Duration.millis(200))   => return "timed out"
    }
  }
}

fun outcome(n: i64): Result<i64, Boom> {
  await sleep(Duration.millis(1))
  if (n < 0) return Err(Boom(n))
  Ok(n)
}

fun values(): string {
  scope {
    val t = async outcome(-6)
    when (val r = await t) {
      is Ok => return "ok $r"
      is Err => return "handled ${r.message()}"
    }
  }
}

fun main() {
  val log = Channel<string>(capacity: 16)
  io.println("pool ${pool() catch (e) { -1 }}")
  io.println("failing: ${failing() catch (e) { -e.n }}")
  io.println("orphan: ${orphan(log) catch (e) { -e.n }}")
  io.println("viaHelper: ${viaHelper(log) catch (e) { -e.n }}")
  io.println("racing: ${racing() catch (e) { "failed ${e.n}" }}")
  io.println(values())
  await sleep(Duration.millis(150))
  io.println("log: ${log.tryRecv() ?: "empty"}")
}
`
	want := "pool 100\n" +
		"failing: 2\n" +
		"orphan: 3\n" +
		"viaHelper: 4\n" +
		"racing: failed -5\n" +
		"handled Boom(n: -6)\n" +
		"log: empty\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, fmt.Sprintf("handles-%v.exe", release))
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
