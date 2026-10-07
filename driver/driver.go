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
	"time"

	"github.com/LaH-DeV/veles/codegen/llvm"
	"github.com/LaH-DeV/veles/docs"
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
	// Sanitize builds the C runtime under AddressSanitizer and
	// UndefinedBehaviorSanitizer and links their runtimes, so a memory error
	// in the runtime or in C code the program links (built with the same
	// flags) stops the program with a report.
	Sanitize    bool
	ProgramArgs []string
	// Fix applies the automatic corrections attached to warnings (check).
	Fix bool
	// Filter, in test mode, keeps the tests whose name contains it.
	Filter string
	// TestTimeout bounds each test (test mode); 0 means no bound.
	TestTimeout time.Duration
	// Jobs bounds how many tests run at once (D80): 0 is one per worker
	// thread, 1 runs them one at a time, in order.
	Jobs int
	// Timings prints the time each phase took, and the modules that cost
	// the most to parse and check, on standard error.
	Timings bool
	// ConstSteps is the step budget of one compile-time evaluation (D113,
	// `--const-steps`); 0 is the default, 10 million.
	ConstSteps int64

	// inWorkspace is set when a workspace run hands a member to Run: the member
	// is built as a package (not fanned out again), and a library member has
	// nothing to build rather than being an error.
	inWorkspace bool
}

// DefaultTestTimeout bounds each test when `--timeout` is not given: long
// enough for any honest test, short enough that a hung one ends a CI run.
const DefaultTestTimeout = 10 * time.Minute

// filterTests applies `--filter` to a test program. No match is an error:
// a mistyped filter that runs nothing must not pass.
func filterTests(prog *sema.Program, filter string) bool {
	if filter == "" {
		return true
	}
	var kept []*sema.Func
	for _, t := range prog.Tests {
		if strings.Contains(t.Display, filter) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		fmt.Fprintf(os.Stderr, "veles test: no test name contains %q (%d tests in the package)\n", filter, len(prog.Tests))
		return false
	}
	prog.TestsFiltered = len(prog.Tests) - len(kept)
	prog.Tests = kept
	return true
}

// Run executes the pipeline and returns a process exit code.
func Run(opts Options) int {
	if !opts.inWorkspace {
		root, man, err := workspaceAt(opts.Path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		if man != nil {
			return runWorkspace(opts, root, man)
		}
	}
	if opts.Mode == "check" && opts.Fix {
		return fixUntilDone(opts)
	}
	diags := &source.Diagnostics{}
	clk := newClock(opts.Timings)
	pkg, err := sema.LoadPackageTimed(opts.Path, diags, clk.frontEnd())
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	if diags.HasErrors() {
		fmt.Fprint(os.Stderr, diags.Render())
		return 1
	}
	clk.lap("load", clk.frontEndNote())
	// only build/run need a program; check accepts a library or a module
	pkg.NeedMain = (opts.Mode == "build" || opts.Mode == "run") && !opts.inWorkspace
	pkg.ConstSteps = opts.ConstSteps
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
	clk.lap("check", "types, effects, lowering")
	if opts.inWorkspace && opts.Mode == "build" && prog.Main == nil {
		fmt.Fprintln(os.Stderr, "veles: library: nothing to build (it has no main)")
		return 0
	}
	if opts.Mode == "check" {
		clk.report(os.Stderr)
		return 0
	}
	if opts.Mode == "test" {
		if !filterTests(prog, opts.Filter) {
			return 1
		}
		prog.TestTimeoutMs = opts.TestTimeout.Milliseconds()
		prog.TestJobs = int64(opts.Jobs)
	}

	ir := llvm.Generate(prog)
	clk.lap("codegen", size(len(ir))+" of LLVM IR")

	name := opts.Output
	if name == "" {
		name = filepath.Base(strings.TrimSuffix(opts.Path, filepath.Ext(opts.Path)))
		if pkg.Manifest != nil && pkg.Manifest.Name != "" && !strings.HasSuffix(opts.Path, ".vs") && !strings.HasSuffix(opts.Path, ".vss") {
			name = pkg.Manifest.Name // a package builds to its own name, wherever it is run from
		}
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
	objs, err := runtimeObjects(clang, opts.Release, opts.Sanitize, tmpDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	clk.lap("runtime", "the C runtime's objects (built once, then cached)")
	args := append([]string{"-o", exe, llPath}, objs...)
	args = append(args, "-Wno-override-module")
	args = append(args, codegenFlags(opts.Release)...)
	if opts.Sanitize {
		args = append(args, sanitizeFlags...)
	}
	if runtime.GOOS != "windows" {
		args = append(args, "-lm") // tan, atan2, hypot: libm is separate outside the UCRT
		args = append(args, "-ldl") // OpenSSL is loaded on first use (veles_tlsio.c)
	} else {
		args = append(args, "-lshell32") // CommandLineToArgvW: UTF-16 process arguments (veles_os.c)
		args = append(args, "-lws2_32")  // sockets (veles_net.c, the reactor in veles_poll.c)
		args = append(args, "-lbcrypt")  // BCryptGenRandom: the system CSPRNG (veles_os.c)
		args = append(args, "-lsecur32", "-lcrypt32", "-lncrypt") // SChannel and the system trust store (veles_tlsio.c)
	}
	native, err := nativeFlags(clang, pkg.NativeManifests())
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	args = append(args, native...)
	cmd := exec.Command(clang, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "veles: clang failed:", err)
		if opts.Sanitize {
			fmt.Fprintln(os.Stderr, "veles: --sanitize links clang's sanitizer runtimes (compiler-rt): with MSYS2, `pacman -S mingw-w64-ucrt-x86_64-compiler-rt`; on Linux they come with clang")
		}
		return 1
	}
	clk.lap("clang", strings.Join(codegenFlags(opts.Release), " ")+": compile the IR and link")
	clk.report(os.Stderr)
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

// runtimeFlags compiles the C runtime. It is optimised in every build: a
// debug build is for stepping through the program, not the executor, and
// an unoptimised runtime makes every channel operation and allocation
// several times slower.
func runtimeFlags(release, sanitize bool) []string {
	if sanitize {
		// -O1 keeps the reports' stacks readable at a bearable speed
		return append([]string{"-O1", "-g"}, sanitizeFlags...)
	}
	if release {
		return []string{"-O2"}
	}
	return []string{"-O2", "-g"}
}

// sanitizeFlags instrument the C runtime and link the sanitizer runtimes
// (`--sanitize`). Undefined behaviour stops the program like a memory
// error does, rather than printing and carrying on. The Veles code itself
// is not instrumented: it is bounds-checked already, and its heap is the
// collector's, which AddressSanitizer does not see into.
var sanitizeFlags = []string{"-fsanitize=address,undefined", "-fno-sanitize-recover=undefined", "-fno-omit-frame-pointer"}

// runtimeSources is the C runtime every executable links, in link order.
var runtimeSources = []struct{ name, src string }{
	{"veles_sync", rt.SyncSource},
	{"veles_stack", rt.StackSource},
	{"veles_rt", rt.Source},
	{"veles_gc", rt.GCSource},
	{"veles_task", rt.TaskSource},
	{"veles_os", rt.OSSource},
	{"veles_net", rt.NetSource},
	{"veles_poll", rt.PollSource},
	{"veles_tlsio", rt.TLSIOSource},
	{"veles_ffi", rt.FFISource},
}

// runtimeObjects returns object files for the C runtime. The runtime never
// changes between builds of the same compiler, so the objects are compiled
// once per (compiler, clang, flags) and kept in the user cache directory;
// without a usable cache they are compiled into tmpDir instead.
func runtimeObjects(clang string, release, sanitize bool, tmpDir string) ([]string, error) {
	flags := runtimeFlags(release, sanitize)
	dir := runtimeCacheDir(clang, flags)
	if dir == "" {
		dir = tmpDir
	}
	var objs []string
	// the header the sources include, next to them
	if err := os.WriteFile(filepath.Join(tmpDir, "veles_tls.h"), []byte(rt.TLSHeader), 0o644); err != nil {
		return nil, err
	}
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
				// A concurrent build got there first (Windows refuses to
				// rename over a file another process has open): the object
				// it installed is the same — same sources, clang and flags.
				// If none is there after all, link this build's own copy.
				if _, serr := os.Stat(obj); serr != nil {
					objs[len(objs)-1] = partial
				}
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
	fmt.Fprintln(h, "veles_tls.h", len(rt.TLSHeader))
	h.Write([]byte(rt.TLSHeader))
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

// ClangPath reports the clang a build would use, for tests that need to
// know whether native code can be produced on this machine.
func ClangPath() (string, error) { return findClang() }

// ExplainFamily prints the explanation of a diagnostic family — the name
// after "see: veles explain" — from the copy of reference/errors.md the
// compiler carries (D79).
func ExplainFamily(name string) int {
	if text, ok := docs.Explain(name); ok {
		fmt.Println(text)
		return 0
	}
	msg := fmt.Sprintf("veles explain: there is no family %q", name)
	if hit := sema.Nearest(name, source.Families()); hit != "" {
		msg += "; did you mean '" + hit + "'?"
	}
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprintln(os.Stderr, "families: "+strings.Join(source.Families(), ", "))
	return 1
}
