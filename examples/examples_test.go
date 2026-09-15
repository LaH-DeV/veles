package examples

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// TestExamples compiles every example directory with the veles binary and
// compares its standard output and exit code against expected.txt.
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
			run := exec.Command(exe)
			var stdout bytes.Buffer
			run.Stdout = &stdout
			err := run.Run()
			code := 0
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else if err != nil {
				t.Fatalf("running: %v", err)
			}
			got := stdout.String() + "exit=" + strconv.Itoa(code) + "\n"
			if got != string(want) {
				t.Errorf("output mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
			}
		})
	}
}
