// Package buildtest builds the veles compiler for the end-to-end suites
// (examples/, docs/). It exists for two reasons, both about a green run
// having to mean a working compiler:
//
//   - The binary is built fresh into the test's temporary directory, so a
//     stale veles.exe lying in the tree is never what gets tested.
//   - `go test` caches a result keyed on the files a test opens itself; the
//     `go build` child process is invisible to it. So before building,
//     every compiler source is opened here, and editing any of them
//     invalidates the cached result of both suites.
package buildtest

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Compiler builds the compiler and returns the path of the binary.
func Compiler(t testing.TB) string {
	t.Helper()
	root := moduleRoot(t)
	touchSources(t, root)
	exe := filepath.Join(t.TempDir(), "veles")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	build := exec.Command("go", "build", "-o", exe, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building compiler: %v\n%s", err, out)
	}
	return exe
}

// moduleRoot walks up from the test's working directory to go.mod.
func moduleRoot(t testing.TB) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("buildtest: no go.mod above the working directory")
		}
		dir = parent
	}
}

// skipDirs are not part of the compiler binary: the suites themselves,
// the editor integrations, and anything hidden.
var skipDirs = map[string]bool{"examples": true, "docs": true, "editors": true, "node_modules": true, "testdata": true}

// touchSources reads every file the compiler binary is made from: Go
// sources (tests excluded) and the std/ and runtime/ trees it embeds.
func touchSources(t testing.TB, root string) {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (skipDirs[name] || strings.HasPrefix(name, ".")) {
				return fs.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		embedded := strings.HasPrefix(rel, "std"+string(filepath.Separator)) || strings.HasPrefix(rel, "runtime"+string(filepath.Separator))
		goSource := strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
		if !goSource && !(embedded && (strings.HasSuffix(name, ".vs") || strings.HasSuffix(name, ".c") || strings.HasSuffix(name, ".h"))) && name != "go.mod" && name != "go.sum" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(io.Discard, f)
		f.Close()
		return err
	})
	if err != nil {
		t.Fatalf("buildtest: reading compiler sources: %v", err)
	}
}
