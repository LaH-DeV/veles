package driver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Plan S8: a debug build carries DWARF, so gdb stops at a Veles line and at
// a function by its name, steps line by line — onto a block's last
// expression too — and prints arguments and locals: numbers, a string's
// text, a struct's fields, and those of a suspending function after it
// resumed. VELES_GDB names a gdb that is not on PATH.
func TestDebuggerReadsVelesSource(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	gdb := os.Getenv("VELES_GDB")
	if gdb == "" {
		var err error
		if gdb, err = exec.LookPath("gdb"); err != nil {
			t.Skip("gdb not available:", err)
		}
	}
	dir := t.TempDir()
	src := `use io

struct Point {
  x: i64
  y: i64
}

fun distance(a: Point, b: Point): i64 {
  val dx = a.x - b.x
  val dy = a.y - b.y
  dx * dx + dy * dy
}

fun fetch(id: i64): i64 {
  await sleep(Duration.millis(id))
  val doubled = id * 2
  doubled + 1
}

fun report(): string {
  val far = Point(x: 3, y: 4)
  val name = "far point"
  var total: i64 = 0
  loop (i in 0..<2) {
    total += distance(Point(x: 0, y: 0), far) + i
  }
  "$name $total"
}

fun main() {
  val line = report()
  val task = async fetch(3)  // after report: its breakpoint is set by then
  io.println("$line ${await task}")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "prog")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if code := Run(Options{Path: dir, Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	script := `set pagination off
set confirm off
break main.vs:25
run
info locals
delete
break distance
continue
info args
next
next
info locals
delete
break main.vs:17
continue
info args
info locals
kill
`
	cmds := filepath.Join(dir, "cmds.gdb")
	if err := os.WriteFile(cmds, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, _ := exec.CommandContext(ctx, gdb, "-q", "-batch", "-x", cmds, exe).CombinedOutput()
	text := string(out)
	for _, want := range []string{
		"main.vs:25",              // a breakpoint by file and line
		"far = {x = 3, y = 4}",    // a struct's fields
		`"far point"`,             // a string's text
		"total = 0",               // a number
		"distance (a=",            // a breakpoint by function name, its arguments
		"b = {x = 3, y = 4}",      // info args
		"11\t  dx * dx + dy * dy", // stepped onto the block's last expression
		"dx = -3",                 // locals after stepping
		"dy = -4",
		"fetch (id=3)", // a suspending function, after its sleep
		"doubled = 6",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("gdb's output lacks %q:\n%s", want, text)
		}
	}
}
