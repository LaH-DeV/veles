// Package buildtest builds the veles compiler for the end-to-end suites
// (examples/, docs/, selfhost/, internal/bootstrap/). It exists for two
// reasons, both about a green run having to mean a working compiler:
//
//   - The binary is built fresh into a temporary directory, so a stale
//     veles.exe lying in the tree is never what gets tested.
//   - `go test` caches a result keyed on the files a test opens itself; the
//     `go build` child process is invisible to it. So before building,
//     every compiler source is opened here, and editing any of them
//     invalidates the cached result of every suite.
//
// It is built once per test binary, not once per test: a package that uses
// it runs its tests through Main, which removes what was built at the end.
package buildtest

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	mainRunning bool
	mu          sync.Mutex
	sharedDir   string
	built       = map[string]*product{}
)

// product is one thing Built made: its path, or why it could not be made.
type product struct {
	once sync.Once
	path string
	err  error
}

// Main runs a package's tests and then removes everything Built made. A
// package that calls Compiler or Built has `func TestMain(m *testing.M) {
// buildtest.Main(m) }`.
func Main(m *testing.M) {
	mainRunning = true
	code := m.Run()
	if sharedDir != "" {
		os.RemoveAll(sharedDir)
	}
	os.Exit(code)
}

// Built returns the path build made, running build only the first time a
// test of this binary asks for name; later callers wait for it and share
// the result, a failure included. build writes into dir, which lives until
// Main removes it.
func Built(t testing.TB, name string, build func(dir string) (string, error)) string {
	t.Helper()
	if !mainRunning {
		t.Fatal("buildtest: the package's TestMain must call buildtest.Main(m)")
	}
	mu.Lock()
	if sharedDir == "" {
		dir, err := os.MkdirTemp("", "veles-buildtest-")
		if err != nil {
			mu.Unlock()
			t.Fatal(err)
		}
		sharedDir = dir
	}
	p := built[name]
	if p == nil {
		p = &product{}
		built[name] = p
	}
	dir := sharedDir
	mu.Unlock()
	p.once.Do(func() {
		sub := filepath.Join(dir, name)
		if p.err = os.MkdirAll(sub, 0o755); p.err == nil {
			p.path, p.err = build(sub)
		}
	})
	if p.err != nil {
		t.Fatalf("building %s: %v", name, p.err)
	}
	return p.path
}

// Exe names an executable called name in dir, with .exe on Windows.
func Exe(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

// Compiler builds the compiler (once per test binary) and returns the path
// of the binary.
func Compiler(t testing.TB) string {
	t.Helper()
	return Built(t, "veles", func(dir string) (string, error) {
		root := moduleRoot(t)
		touchSources(t, root)
		exe := Exe(dir, "veles")
		build := exec.Command("go", "build", "-o", exe, ".")
		build.Dir = root
		if out, err := build.CombinedOutput(); err != nil {
			return "", fmt.Errorf("%v\n%s", err, out)
		}
		return exe, nil
	})
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
