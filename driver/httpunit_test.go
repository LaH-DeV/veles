package driver

import (
	"path/filepath"
	"testing"
)

// std/http's own tests (cookies, forms) run as part of the suite.
func TestStdHttpUnitTests(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "http"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/http: exit %d", code)
	}
}
