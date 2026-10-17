package driver

import (
	"path/filepath"
	"testing"
)

// std/fmt's own tests (the format helpers) run as part of the suite.
func TestStdFmtUnitTests(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "fmt"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/fmt: exit %d", code)
	}
}
