package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/sema"
)

// `--timings` on check: the phases that ran, and each module's parse and
// check time, the root's included.
func TestTimingsReportsPhasesAndModules(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.vs"), []byte("use io\n\nfun main() {\n  io.println(\"hi\")\n}\n"), 0o644)
	out := filepath.Join(t.TempDir(), "stderr.txt")
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = f
	code := Run(Options{Path: dir, Mode: "check", Timings: true})
	os.Stderr = saved
	f.Close()
	data, _ := os.ReadFile(out)
	got := string(data)
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, got)
	}
	for _, want := range []string{"timings: ", "\n  load ", "1 file", "\n  check ", "modules, slowest first", "(root)", "std/prelude", "std/io"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "codegen") {
		t.Errorf("check ran no codegen, yet reports it:\n%s", got)
	}
}

func TestSpanReadsLikeAPerson(t *testing.T) {
	for d, want := range map[time.Duration]string{
		1180 * time.Millisecond: "1.18s",
		310 * time.Millisecond:  "310ms",
		4200 * time.Microsecond: "4.2ms",
		4 * time.Millisecond:    "4.0ms",
		300 * time.Microsecond:  "0.3ms",
	} {
		if got := span(d); got != want {
			t.Errorf("span(%v) = %q, want %q", d, got, want)
		}
	}
}

// A nil clock — no `--timings` — does nothing and reports nothing.
func TestNoClockNoTimings(t *testing.T) {
	var c *clock
	c.lap("load", "")
	if c.frontEnd() != (*sema.Timings)(nil) || c.frontEndNote() != "" {
		t.Error("a nil clock recorded something")
	}
	c.report(nil) // must not touch the writer
}
