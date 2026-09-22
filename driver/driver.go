// Package driver runs the compilation pipeline: load a package, parse every
// module, check, emit LLVM IR, and invoke clang (I1/I2).
package driver

import (
	"crypto/sha256"
	"encoding/hex"
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
	Mode              string // build | run | check | test
	Output            string
	EmitLLVM          bool
	KeepIntermediates bool
	Release           bool
	ProgramArgs       []string
	// Fix applies the automatic corrections attached to warnings (check).
	Fix bool
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
	// only build/run need a program; check accepts a library or a module
	pkg.NeedMain = opts.Mode == "build" || opts.Mode == "run"
	var prog *sema.Program
	if opts.Mode == "test" {
		prog = sema.CheckTests(pkg, diags, opts.Release)
	} else {
		prog = sema.Check(pkg, diags, opts.Release)
	}
	fmt.Fprint(os.Stderr, diags.Render())
	if opts.Mode == "check" && opts.Fix {
		// fixes hang off warnings (lints) and off errors for removed forms
		// with a mechanical replacement; a run with errors applies what it
		// can, and the next run reports what is left
		n, err := ApplyFixes(diags)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "%d fix(es) applied\n", n)
	}
	if diags.HasErrors() || prog == nil {
		return 1
	}
	if opts.Mode == "check" {
		return 0
	}

	ir := llvm.Generate(prog)

	name := opts.Output
	if name == "" {
		name = filepath.Base(strings.TrimSuffix(opts.Path, filepath.Ext(opts.Path)))
		if name == "." || name == "" {
			name = "main"
		}
	}
	if opts.EmitLLVM {
		llPath := strings.TrimSuffix(name, ".exe") + ".ll"
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
	} else {
		defer fmt.Fprintln(os.Stderr, "intermediates kept in", tmpDir)
	}
	// `run` and `test` executables are transient unless -o names them;
	// only `build` leaves one next to the caller.
	exe := name
	if opts.Mode != "build" && opts.Output == "" {
		exe = filepath.Join(tmpDir, name)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(exe, ".exe") {
		exe += ".exe"
	}
	llPath := filepath.Join(tmpDir, "program.ll")
	if err := os.WriteFile(llPath, []byte(ir), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	clang, err := findClang()
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	objs, err := runtimeObjects(clang, opts.Release, tmpDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	args := append([]string{"-o", exe, llPath}, objs...)
	args = append(args, "-Wno-override-module")
	args = append(args, codegenFlags(opts.Release)...)
	if runtime.GOOS != "windows" {
		args = append(args, "-lm") // tan, atan2, hypot: libm is separate outside the UCRT
	} else {
		args = append(args, "-lshell32") // CommandLineToArgvW: UTF-16 process arguments (veles_os.c)
		args = append(args, "-lws2_32")  // sockets (veles_net.c, WSAPoll in veles_task.c)
	}
	cmd := exec.Command(clang, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "veles: clang failed:", err)
		return 1
	}
	if opts.Mode == "build" {
		return 0
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

func codegenFlags(release bool) []string {
	if release {
		return []string{"-O2"}
	}
	return []string{"-O0", "-g"}
}

// runtimeSources is the C runtime every executable links, in link order.
var runtimeSources = []struct{ name, src string }{
	{"veles_rt", rt.Source},
	{"veles_gc", rt.GCSource},
	{"veles_task", rt.TaskSource},
	{"veles_os", rt.OSSource},
	{"veles_net", rt.NetSource},
}

// runtimeObjects returns object files for the C runtime. The runtime never
// changes between builds of the same compiler, so the objects are compiled
// once per (compiler, clang, flags) and kept in the user cache directory;
// without a usable cache they are compiled into tmpDir instead.
func runtimeObjects(clang string, release bool, tmpDir string) ([]string, error) {
	flags := codegenFlags(release)
	dir := runtimeCacheDir(clang, flags)
	if dir == "" {
		dir = tmpDir
	}
	var objs []string
	for _, s := range runtimeSources {
		obj := filepath.Join(dir, s.name+".o")
		objs = append(objs, obj)
		if dir != tmpDir {
			if _, err := os.Stat(obj); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		src := filepath.Join(tmpDir, s.name+".c")
		if err := os.WriteFile(src, []byte(s.src), 0o644); err != nil {
			return nil, err
		}
		// compile to a private name and rename so a concurrent build never
		// links a half-written object
		partial := filepath.Join(tmpDir, s.name+".o")
		args := append([]string{"-c", "-o", partial, src}, flags...)
		cmd := exec.Command(clang, args...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("compiling %s.c: %w", s.name, err)
		}
		if partial != obj {
			if err := os.Rename(partial, obj); err != nil {
				return nil, err
			}
		}
	}
	return objs, nil
}

// runtimeCacheDir names the cache directory for runtime objects built with
// this compiler's runtime sources by this clang with these flags, or ""
// when there is no user cache directory.
func runtimeCacheDir(clang string, flags []string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	h := sha256.New()
	fmt.Fprintln(h, clang, strings.Join(flags, " "), runtime.GOOS, runtime.GOARCH)
	if info, err := os.Stat(clang); err == nil {
		fmt.Fprintln(h, info.Size(), info.ModTime().UnixNano())
	}
	for _, s := range runtimeSources {
		fmt.Fprintln(h, s.name, len(s.src))
		h.Write([]byte(s.src))
	}
	return filepath.Join(base, "veles", "rt", hex.EncodeToString(h.Sum(nil))[:16])
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

// Explain answers a question about what the compiler made of a package,
// rather than compiling it. Today there is one question: `--derive`, which
// prints the implements D58 synthesized, as Veles source. TypeName, when
// set, limits the output to that type.
type ExplainOptions struct {
	Path     string
	Derive   bool
	TypeName string
}

func Explain(opts ExplainOptions) int {
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
	out := sema.ExplainDerived(pkg, diags, opts.TypeName)
	fmt.Fprint(os.Stderr, diags.Render())
	if len(out) == 0 {
		who := "this package"
		if opts.TypeName != "" {
			who = "'" + opts.TypeName + "'"
		}
		fmt.Fprintf(os.Stderr, "veles: nothing derived for %s\n", who)
		if diags.HasErrors() {
			return 1
		}
		return 0
	}
	fmt.Println(strings.Join(out, "\n\n"))
	if diags.HasErrors() {
		return 1
	}
	return 0
}
