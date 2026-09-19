package examples

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// -update rewrites the expected.txt of every example whose output differs.
var update = flag.Bool("update", false, "rewrite expected.txt with the actual output")

// TestExamples compiles every example directory with the veles binary and
// compares its standard output and exit code against expected.txt. An
// example with a commands.txt is a command-line tool: see runScript.
func TestExamples(t *testing.T) {
	veles := filepath.Join("..", "veles.exe")
	if runtime.GOOS != "windows" {
		veles = filepath.Join("..", "veles")
	}
	if _, err := os.Stat(veles); err != nil {
		build := exec.Command("go", "build", "-o", veles, "..")
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("building compiler: %v\n%s", err, out)
		}
	}
	var dirs []string
	filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "expected.txt" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	for _, dir := range dirs {
		want, err := os.ReadFile(filepath.Join(dir, "expected.txt"))
		if err != nil {
			continue
		}
		t.Run(dir, func(t *testing.T) {
			exe := filepath.Join(t.TempDir(), filepath.Base(dir))
			if runtime.GOOS == "windows" {
				exe += ".exe"
			}
			build := exec.Command(veles, "build", dir, "-o", exe)
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("veles build failed: %v\n%s", err, out)
			}
			var got string
			if script, err := os.ReadFile(filepath.Join(dir, "commands.txt")); err == nil {
				got = runScript(t, exe, dir, string(script))
			} else {
				out, code := runOnce(t, exe, nil, nil, "")
				got = out + "exit=" + strconv.Itoa(code) + "\n"
			}
			if got != string(want) {
				if *update {
					if err := os.WriteFile(filepath.Join(dir, "expected.txt"), []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
					t.Logf("updated expected.txt")
					return
				}
				t.Errorf("output mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
			}
		})
	}
}

// runOnce runs the program and returns its standard output and exit code.
func runOnce(t *testing.T, exe string, args, env []string, dir string) (string, int) {
	t.Helper()
	run := exec.Command(exe, args...)
	run.Dir = dir
	if env != nil {
		run.Env = append(os.Environ(), env...)
	}
	var stdout bytes.Buffer
	run.Stdout = &stdout
	err := run.Run()
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
		stdout, code := runOnce(t, exe, words[1:], env, dir)
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
