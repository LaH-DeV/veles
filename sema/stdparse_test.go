package sema

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// TestStdParsedOnce: loads checked at the same time each get their own copy
// of the embedded std (the checker rewrites the tree it is given, D158), the
// source file behind it is shared, and the checks agree.
func TestStdParsedOnce(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.vs"), []byte("use io\n\nfun main() {\n  val xs = [3, 1, 2].sorted()\n  io.println(\"${xs}\")\n}\n"), 0o644)
	const n = 8
	pkgs := make([]*Package, n)
	out := make([]string, n)
	var wg sync.WaitGroup
	for i := range pkgs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			diags := &source.Diagnostics{}
			pkg, err := LoadPackage(dir, diags)
			if err != nil {
				t.Error(err)
				return
			}
			Check(pkg, diags, false)
			pkgs[i], out[i] = pkg, diags.Render()
		}()
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	if out[0] != "" {
		t.Fatalf("diagnostics:\n%s", out[0])
	}
	a, b := pkgs[0].Modules["std/prelude"], pkgs[1].Modules["std/prelude"]
	if a == nil || b == nil || len(a.Files) == 0 || len(a.Files) != len(b.Files) {
		t.Fatal("the prelude was not loaded")
	}
	for i := range a.Files {
		if a.Files[i] == b.Files[i] {
			t.Fatalf("%s: two loads share one tree", a.Files[i].Source.Path)
		}
		if a.Files[i].Source != b.Files[i].Source {
			t.Fatalf("%s: the embedded source is read again", a.Files[i].Source.Path)
		}
	}
	for i := 1; i < n; i++ {
		if out[i] != out[0] {
			t.Fatalf("check %d disagrees:\n%s", i, out[i])
		}
	}
}
