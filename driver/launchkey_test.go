package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// `async { … }` (a suspending `fun(): T`) and `async job()` on a plain
// `fun(): T` value are started through different trampolines: their types'
// key left out suspension, so the two shared one and the second was called
// the first's way — "deadlock: every task is blocked", or a crash
// (2026-11-06, predating D231). Both orders, both profiles.
func TestAsyncBlockAndFunctionValueLaunch(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

fun first(): i64 suspends {
  with t = async {
    await sleep(Duration.millis(2))
    1
  }
  val one = await t
  val job = () => 2
  with u = async job()
  one + await u
}

fun second(): i64 suspends {
  val job = () => 3
  with u = async job()
  val three = await u
  with t = async { 4 }
  three + await t
}

fun main() suspends {
  io.println("${first()} ${second()}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "launch.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.TrimSpace(string(out)); got != "3 7" {
			t.Errorf("release=%v: got %q, want \"3 7\"", release, got)
		}
	}
}
