// Package driver runs the compilation pipeline: load a package, parse every
// module, check, emit LLVM IR, and invoke clang (I1/I2).
package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LaH-DeV/veles/codegen/llvm"
	rt "github.com/LaH-DeV/veles/runtime"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

type Options struct {
	Path              string
	Mode              string // build | run | check
	Output            string
	EmitLLVM          bool
	KeepIntermediates bool
	Release           bool
	ProgramArgs       []string
}

// Run executes the pipeline and returns a process exit code.
func Run(opts Options) int {
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackage(opts.Path, diags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	if diags.HasErrors() {
		fmt.Fprint(os.Stderr, diags.Render())
		return 1
	}
	var prog *sema.Program
	if opts.Mode == "test" {
		prog = sema.CheckTests(pkg, diags, opts.Release)
	} else {
		prog = sema.Check(pkg, diags, opts.Release)
	}
	fmt.Fprint(os.Stderr, diags.Render())
	if diags.HasErrors() || prog == nil {
		return 1
	}
	if opts.Mode == "check" {
		return 0
	}

	ir := llvm.Generate(prog)

	base := opts.Output
	if base == "" {
		name := filepath.Base(strings.TrimSuffix(opts.Path, filepath.Ext(opts.Path)))
		if name == "." || name == "" {
			name = "main"
		}
		base = name
	}
	exe := base
	if runtime.GOOS == "windows" && !strings.HasSuffix(exe, ".exe") {
		exe += ".exe"
	}
	llPath := strings.TrimSuffix(exe, ".exe") + ".ll"
	if opts.EmitLLVM {
		if err := os.WriteFile(llPath, []byte(ir), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		fmt.Println("wrote", llPath)
		return 0
	}

	tmpDir, err := os.MkdirTemp("", "veles-build-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	if !opts.KeepIntermediates {
		defer os.RemoveAll(tmpDir)
	}
	llPath = filepath.Join(tmpDir, "program.ll")
	rtPath := filepath.Join(tmpDir, "veles_rt.c")
	gcPath := filepath.Join(tmpDir, "veles_gc.c")
	if err := os.WriteFile(llPath, []byte(ir), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	if err := os.WriteFile(rtPath, []byte(rt.Source), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	if err := os.WriteFile(gcPath, []byte(rt.GCSource), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	taskPath := filepath.Join(tmpDir, "veles_task.c")
	if err := os.WriteFile(taskPath, []byte(rt.TaskSource), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	clang, err := findClang()
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	args := []string{"-o", exe, llPath, rtPath, gcPath, taskPath, "-Wno-override-module"}
	if opts.Release {
		args = append(args, "-O2")
	} else {
		args = append(args, "-O0", "-g")
	}
	cmd := exec.Command(clang, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "veles: clang failed:", err)
		if opts.KeepIntermediates {
			fmt.Fprintln(os.Stderr, "intermediates kept in", tmpDir)
		}
		return 1
	}
	if opts.KeepIntermediates {
		fmt.Fprintln(os.Stderr, "intermediates kept in", tmpDir)
	}
	if opts.Mode == "build" {
		return 0
	}
	if opts.Mode == "test" {
		defer os.Remove(exe)
	}
	abs, _ := filepath.Abs(exe)
	run := exec.Command(abs, opts.ProgramArgs...)
	run.Stdin, run.Stdout, run.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	return 0
}

// findClang locates clang on PATH or in the usual MSYS2/LLVM locations.
func findClang() (string, error) {
	if p := os.Getenv("VELES_CLANG"); p != "" {
		return p, nil
	}
	if p, err := exec.LookPath("clang"); err == nil {
		return p, nil
	}
	candidates := []string{
		`C:\msys64\ucrt64\bin\clang.exe`,
		`C:\msys64\clang64\bin\clang.exe`,
		`C:\msys64\mingw64\bin\clang.exe`,
		`C:\Program Files\LLVM\bin\clang.exe`,
		"/usr/local/opt/llvm/bin/clang",
		"/opt/homebrew/opt/llvm/bin/clang",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("clang not found; install LLVM/clang or set VELES_CLANG to its path (spec I1/I2: Veles emits LLVM IR)")
}
