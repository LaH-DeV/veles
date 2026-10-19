package bootstrap_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/internal/bootstrap"
	"github.com/LaH-DeV/veles/internal/buildtest"
)

func TestFirstDifference(t *testing.T) {
	if d := bootstrap.FirstDifference("a", "x\ny\n", "b", "x\ny\n"); d != "" {
		t.Errorf("equal texts differ: %s", d)
	}
	d := bootstrap.FirstDifference("stage1", "one\ntwo\nthree", "stage2", "one\ntwo\nTHREE")
	for _, want := range []string{"line 3", "stage1: three", "stage2: THREE", "   two"} {
		if !strings.Contains(d, want) {
			t.Errorf("difference lacks %q:\n%s", want, d)
		}
	}
	if d := bootstrap.FirstDifference("a", "x", "b", "x\ny"); !strings.Contains(d, "line 2") {
		t.Errorf("a longer text is a difference at its extra line: %q", d)
	}
}

// TestFixpointMechanics runs the three stages over a small program with a
// compiler that rebuilds itself as itself: every stage is the Go compiler, so
// the fixpoint and the agreement with stage 0 hold, and the harness's
// plumbing (build, emit IR, compare) is proved before a Veles compiler
// exists to put through it.
func TestFixpointMechanics(t *testing.T) {
	stage0 := bootstrap.Compiler{Name: "stage0", Exe: buildtest.Compiler(t)}
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "veles.toml"), []byte("[package]\nname = \"probe\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prog := "use io\n\nfun fib(n: i64): i64 => if (n < 2) n else fib(n - 1) + fib(n - 2)\n\nfun main() {\n  io.println(\"${fib(10)}\")\n}\n"
	if err := os.WriteFile(filepath.Join(src, "main.vs"), []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	same := func(from bootstrap.Compiler, srcDir, work, name string) (bootstrap.Compiler, error) {
		return bootstrap.Compiler{Name: name, Exe: from.Exe}, nil
	}
	r, err := bootstrap.Fixpoint(stage0, src, t.TempDir(), same)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Fixpoint || !r.AgreesWithStage0 {
		t.Fatalf("a compiler that is its own successor must reach the fixpoint:\n%s\n%s", r.Diff1, r.Diff2)
	}
	if !strings.Contains(r.IR0, "define") {
		t.Errorf("stage 0 emitted no IR:\n%.200s", r.IR0)
	}
}

// TestSelfHostedCompiler is the S14 gate: the Veles compiler's source
// (selfhost/compiler, built by S11's port) goes through the three stages.
// Until that package exists the test skips; it must not be deleted then.
func TestSelfHostedCompiler(t *testing.T) {
	src, _ := filepath.Abs(filepath.Join("..", "..", "selfhost", "compiler"))
	if _, err := os.Stat(filepath.Join(src, "veles.toml")); err != nil {
		t.Skip("selfhost/compiler does not exist yet (plan S11)")
	}
	stage0 := bootstrap.Compiler{Name: "stage0", Exe: buildtest.Compiler(t)}
	r, err := bootstrap.Fixpoint(stage0, src, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Fixpoint {
		t.Errorf("stage 1 and stage 2 emit different IR:\n%s", r.Diff1)
	}
	if !r.AgreesWithStage0 {
		t.Errorf("the Veles compiler and the Go compiler emit different IR:\n%s", r.Diff2)
	}
}

// TestBuildProducesARunnableStage: Compiler.Build, the step that turns a
// stage's source into the next compiler, yields a working executable.
func TestBuildProducesARunnableStage(t *testing.T) {
	stage0 := bootstrap.Compiler{Name: "stage0", Exe: buildtest.Compiler(t)}
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "veles.toml"), []byte("[package]\nname = \"probe\"\nversion = \"0.1.0\"\n"), 0o644)
	os.WriteFile(filepath.Join(src, "main.vs"), []byte("use io\n\nfun main() {\n  io.println(\"stage1 says hi\")\n}\n"), 0o644)
	next, err := stage0.Build(src, t.TempDir(), "stage1")
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(next.Exe).Output()
	if err != nil || !strings.Contains(string(out), "stage1 says hi") {
		t.Fatalf("stage 1 did not run: %v %q", err, out)
	}
}
