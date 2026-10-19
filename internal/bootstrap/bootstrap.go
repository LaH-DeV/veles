// Package bootstrap is the harness of the self-hosting fixpoint (plan S14).
//
// Three compilers take part. Stage 0 is the bootstrap compiler written in Go.
// Stage 1 is the Veles compiler's own source, built by stage 0. Stage 2 is
// the same source built by stage 1. The compiler is self-hosted when
//
//   - stage 1 and stage 2 emit identical LLVM IR for the compiler's source
//     (the fixpoint: building changed nothing), and
//   - stage 0 and stage 1 emit identical IR for it (the Veles compiler agrees
//     with the Go one, byte for byte, as the self-hosted parser does).
//
// A "compiler" here is anything that answers the `veles build <dir> -o out
// [--release] [--emit-llvm]` command line. Nothing in this package knows what
// the compiler source is: the harness runs now, against a stand-in, and
// gates the real port when `selfhost/compiler` exists.
package bootstrap

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Compiler is an executable that speaks the `veles` command line.
type Compiler struct {
	// Name labels it in reports: "stage0", "stage1", "stage2".
	Name string
	Exe  string
}

func exeName(base string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(base, ".exe") {
		return base + ".exe"
	}
	return base
}

// Build compiles the package in srcDir to an executable under work and
// returns it as a Compiler named name. The build is `--release`: the
// profile whose output is compared must be the profile that ships.
func (c Compiler) Build(srcDir, work, name string) (Compiler, error) {
	exe := exeName(filepath.Join(work, name))
	cmd := exec.Command(c.Exe, "build", ".", "-o", exe, "--release")
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return Compiler{}, fmt.Errorf("%s: building %s: %v\n%s", c.Name, srcDir, err, out)
	}
	return Compiler{Name: name, Exe: exe}, nil
}

// EmitIR writes the LLVM IR the compiler produces for the package in srcDir
// and returns its text, line endings normalised.
func (c Compiler) EmitIR(srcDir, work, name string) (string, error) {
	out := filepath.Join(work, name)
	cmd := exec.Command(c.Exe, "build", ".", "-o", out, "--release", "--emit-llvm")
	cmd.Dir = srcDir
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s: emitting IR of %s: %v\n%s", c.Name, srcDir, err, b)
	}
	data, err := os.ReadFile(strings.TrimSuffix(out, ".exe") + ".ll")
	if err != nil {
		return "", fmt.Errorf("%s: %v", c.Name, err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n"), nil
}

// Result is what one run of the fixpoint found.
type Result struct {
	// IR0, IR1, IR2 are the IR each stage emitted for the compiler's source.
	IR0, IR1, IR2 string
	// Fixpoint: stage 1 and stage 2 agree. AgreesWithStage0: stage 0 and 1 do.
	Fixpoint         bool
	AgreesWithStage0 bool
	// Diff1 and Diff2 describe the first difference of (IR1, IR2) and of
	// (IR0, IR1); empty when there is none.
	Diff1, Diff2 string
}

// Rebuild is how a stage produces the next compiler from the source. The
// default is Compiler.Build; a test substitutes one to exercise the checks.
type Rebuild func(from Compiler, srcDir, work, name string) (Compiler, error)

// Fixpoint runs the three stages over the compiler source in srcDir, using
// work for the executables and IR. A nil rebuild is Compiler.Build.
func Fixpoint(stage0 Compiler, srcDir, work string, rebuild Rebuild) (*Result, error) {
	if rebuild == nil {
		rebuild = func(from Compiler, srcDir, work, name string) (Compiler, error) {
			return from.Build(srcDir, work, name)
		}
	}
	stage0.Name = "stage0"
	stage1, err := rebuild(stage0, srcDir, work, "stage1")
	if err != nil {
		return nil, err
	}
	stage2, err := rebuild(stage1, srcDir, work, "stage2")
	if err != nil {
		return nil, err
	}
	r := &Result{}
	if r.IR0, err = stage0.EmitIR(srcDir, work, "ir0"); err != nil {
		return nil, err
	}
	if r.IR1, err = stage1.EmitIR(srcDir, work, "ir1"); err != nil {
		return nil, err
	}
	if r.IR2, err = stage2.EmitIR(srcDir, work, "ir2"); err != nil {
		return nil, err
	}
	r.Diff1 = FirstDifference("stage1", r.IR1, "stage2", r.IR2)
	r.Diff2 = FirstDifference("stage0", r.IR0, "stage1", r.IR1)
	r.Fixpoint = r.Diff1 == ""
	r.AgreesWithStage0 = r.Diff2 == ""
	return r, nil
}

// FirstDifference describes where two texts first differ, "" when they do
// not: the line number and both lines, with a line of context before.
func FirstDifference(aName, a, bName, b string) string {
	if a == b {
		return ""
	}
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x == y {
			continue
		}
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "first difference at line %d (%d and %d lines in all)\n", i+1, len(al), len(bl))
		if i > 0 {
			fmt.Fprintf(&buf, "   %s\n", al[i-1])
		}
		fmt.Fprintf(&buf, "  %s: %s\n  %s: %s", aName, x, bName, y)
		return buf.String()
	}
	return ""
}
