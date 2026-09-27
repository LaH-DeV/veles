package driver

import (
	"os"
	"path/filepath"
	"testing"
)

// `veles new` makes a package that checks clean and whose test passes;
// it refuses a name other packages could not write in `use`, and never
// writes into a directory that has something in it.
func TestNew(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "hello")
	if code := New(dir); code != 0 {
		t.Fatalf("new: exit %d", code)
	}
	for _, f := range []string{"veles.toml", "main.vs", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if code := Run(Options{Path: dir, Mode: "check"}); code != 0 {
		t.Errorf("the new package does not check clean")
	}
	if code := New(dir); code == 0 {
		t.Errorf("new over a non-empty directory succeeded")
	}
	if code := New(filepath.Join(root, "my-app")); code != 2 {
		t.Errorf("'my-app' accepted as a package name")
	}
}
