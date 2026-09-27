package examples

import (
	"bytes"
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/internal/buildtest"
)

// -update rewrites the expected.txt of every example whose output differs.
var update = flag.Bool("update", false, "rewrite expected.txt with the actual output")

// TestExamples compiles every example directory with the veles binary and
// compares its standard output and exit code against expected.txt. An
// example with a commands.txt is a command-line tool: see runScript. A
// script (`name.vss`) is an example by itself when a `name.expected.txt` exists
// (create it empty, then -update); `name.stdin.txt`, when present, is its input.
func TestExamples(t *testing.T) {
	veles := buildtest.Compiler(t)
	type example struct {
		name     string // test name
		source   string // what veles build is pointed at: a directory or a script
		expected string // the expected.txt to compare against
	}
	var cases []example
	filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch {
		case d.Name() == "expected.txt":
			dir := filepath.Dir(path)
			cases = append(cases, example{name: dir, source: dir, expected: path})
		case strings.HasSuffix(d.Name(), ".vss"):
			stem := strings.TrimSuffix(path, ".vss")
			if _, err := os.Stat(stem + ".expected.txt"); err == nil {
				cases = append(cases, example{name: stem, source: path, expected: stem + ".expected.txt"})
			}
		}
		return nil
	})
	for _, ex := range cases {
		want, err := os.ReadFile(ex.expected)
		if err != nil {
			continue
		}
		t.Run(ex.name, func(t *testing.T) {
			exe := filepath.Join(t.TempDir(), filepath.Base(ex.name))
			if runtime.GOOS == "windows" {
				exe += ".exe"
			}
			build := exec.Command(veles, "build", ex.source, "-o", exe)
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("veles build failed: %v\n%s", err, out)
			}
			var got string
			if script, err := os.ReadFile(filepath.Join(ex.source, "commands.txt")); err == nil {
				got = runScript(t, exe, ex.source, string(script))
			} else {
				var stdin []byte
				if ex.source != ex.name {
					stdin, _ = os.ReadFile(ex.name + ".stdin.txt")
				}
				out, code := runOnce(t, exe, nil, nil, "", stdin)
				got = out + "exit=" + strconv.Itoa(code) + "\n"
			}
			if got != string(want) {
				if *update {
					if err := os.WriteFile(ex.expected, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
					t.Logf("updated %s", filepath.Base(ex.expected))
					return
				}
				t.Errorf("output mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
			}
		})
	}
}

// runTimeout bounds one run of an example. A hung example is killed and
// fails; without the bound it held the suite until `go test` gave up, and
// that left the process running (servers outlived the test for a day).
const runTimeout = 2 * time.Minute

// runOnce runs the program and returns its standard output and exit code.
func runOnce(t *testing.T, exe string, args, env []string, dir string, stdin []byte) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	run := exec.CommandContext(ctx, exe, args...)
	run.WaitDelay = 5 * time.Second
	run.Dir = dir
	if env != nil {
		run.Env = append(os.Environ(), env...)
	}
	if stdin != nil {
		run.Stdin = bytes.NewReader(stdin)
	}
	var stdout bytes.Buffer
	run.Stdout = &stdout
	err := run.Run()
	if ctx.Err() != nil {
		t.Fatalf("%s %s did not finish in %v; killed. Output so far:\n%s", filepath.Base(exe), strings.Join(args, " "), runTimeout, stdout.String())
	}
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("running: %v", err)
	}
	return stdout.String(), code
}

// runScript drives a command-line program: every non-blank line of
// commands.txt is one invocation, written like a shell command whose first
// word is the program (`NAME=value` words before it set the environment).
// All invocations share a fresh temporary working directory, seeded with a
// copy of the example's fixtures/ directory when it has one. The transcript
// echoes each command as `$ line`, then its output, then `exit=N`.
func runScript(t *testing.T, exe, example, script string) string {
	t.Helper()
	dir := t.TempDir()
	if fixtures := filepath.Join(example, "fixtures"); dirExists(fixtures) {
		if err := os.CopyFS(filepath.Join(dir, "fixtures"), os.DirFS(fixtures)); err != nil {
			t.Fatal(err)
		}
	}
	var out strings.Builder
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words := splitWords(line)
		var env []string
		for len(words) > 0 && strings.Contains(words[0], "=") {
			env = append(env, words[0])
			words = words[1:]
		}
		if len(words) == 0 {
			t.Fatalf("commands.txt: no program on line %q", line)
		}
		stdout, code := runOnce(t, exe, words[1:], env, dir, nil)
		out.WriteString("$ " + line + "\n" + stdout + "exit=" + strconv.Itoa(code) + "\n")
	}
	return out.String()
}

// splitWords splits a command line on spaces; double quotes group words.
func splitWords(line string) []string {
	var words []string
	var cur strings.Builder
	inWord, quoted := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted = !quoted
			inWord = true
		case r == ' ' && !quoted:
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
