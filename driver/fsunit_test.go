package driver

import (
	"path/filepath"
	"testing"
)

// std/fs's own tests (open files, stat) run as part of the suite.
func TestStdFsUnitTests(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "fs"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/fs: exit %d", code)
	}
}
