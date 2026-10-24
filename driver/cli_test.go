package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// std/cli in a program of its own package (D167, D203, D204): the struct's
// fields are the program's, unmarked, so `T.build` in `cli.Args` sets them
// because the type opted in; `--help` and `--version` print and exit 0, a
// usage problem prints every problem to stderr and exits 2, and a flag
// falls back on its environment variable.
func TestCliProgram(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	const program = `use io
use cli { Flag, Positional }

/// Find duplicate files under a directory.
struct Options {
  /// The directory to scan.
  @Positional
  dir: string
  /// How many files to hash at once.
  @Flag(short: "w", env: "VELES_CLI_WORKERS")
  workers: i64 = 4
  /// Say what is compared.
  @Flag(short: "v")
  verbose: bool = false
  implement cli.Args
}

fun main() {
  val o = cli.args<Options>()
  io.println("dir=${o.dir} workers=${o.workers} verbose=${o.verbose}")
}
`
	dir := t.TempDir()
	files := map[string]string{
		"veles.toml": "[package]\nname = \"dedup\"\nversion = \"1.2.0\"\n",
		"main.vs":    program,
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	exe := filepath.Join(dir, "dedup.exe")
	if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	for _, c := range []struct {
		args   []string
		env    string
		code   int
		stdout string // contained in stdout
		stderr string // contained in stderr
	}{
		{[]string{"src", "-vw8"}, "", 0, "dir=src workers=8 verbose=true", ""},
		{[]string{"src"}, "VELES_CLI_WORKERS=9", 0, "dir=src workers=9 verbose=false", ""},
		{[]string{"--help"}, "", 0, "-w, --workers <WORKERS>  How many files to hash at once. (default: 4) [env: VELES_CLI_WORKERS]", ""},
		{[]string{"--version"}, "", 0, "dedup 1.2.0", ""},
		{[]string{"--wrkers", "-q"}, "", 2, "", "error: unknown flag '--wrkers'; did you mean '--workers'?\nerror: unknown flag '-q'\nerror: the argument <dir> is missing\n\nUsage: dedup [options] <dir>\nRun 'dedup --help' for more."},
	} {
		run := exec.Command(exe, c.args...)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "VELES_CLI_") {
				run.Env = append(run.Env, kv)
			}
		}
		if c.env != "" {
			run.Env = append(run.Env, c.env)
		}
		var stdout, stderr strings.Builder
		run.Stdout, run.Stderr = &stdout, &stderr
		err := run.Run()
		code := 0
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		out := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
		errText := strings.ReplaceAll(stderr.String(), "\r\n", "\n")
		if code != c.code || !strings.Contains(out, c.stdout) || !strings.Contains(errText, c.stderr) {
			t.Errorf("%v %s: exit %d, stdout %q, stderr %q; want exit %d, stdout with %q, stderr with %q", c.args, c.env, code, out, errText, c.code, c.stdout, c.stderr)
		}
	}
}
