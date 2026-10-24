package driver

import (
	"os"
	"path/filepath"
	"testing"
)

// ownStdTests are the std modules whose tests a driver test of their own
// runs (with a server, a database or a fixture it sets up).
var ownStdTests = map[string]bool{"db": true, "fmt": true, "fs": true, "http": true, "log": true, "regex": true, "time": true}

// Every other std module with `*.test.vs` files has them run here, so a new
// module's tests are part of the suite without a test of their own (until
// 2026-10-24 ten modules' tests — prelude, crypto, toml and config among
// them — ran only by hand).
func TestStdModuleTests(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dirs, err := os.ReadDir(filepath.Join("..", "std"))
	if err != nil {
		t.Fatal(err)
	}
	ran := 0
	for _, d := range dirs {
		name := d.Name()
		if !d.IsDir() || ownStdTests[name] {
			continue
		}
		dir := filepath.Join("..", "std", name)
		if tests, _ := filepath.Glob(filepath.Join(dir, "*.test.vs")); len(tests) == 0 {
			continue
		}
		ran++
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if code := Run(Options{Path: dir, Mode: "test"}); code != 0 {
				t.Fatalf("veles test std/%s: exit %d", name, code)
			}
		})
	}
	if ran < 10 {
		t.Fatalf("found the tests of %d std modules; is the working directory driver/?", ran)
	}
}
