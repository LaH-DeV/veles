package driver

import (
	"path/filepath"
	"testing"
)

// std/regex's own tests (the engine, both matchers and the API) run as part of
// the suite.
func TestStdRegexUnitTests(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	if code := Run(Options{Path: filepath.Join("..", "std", "regex"), Mode: "test"}); code != 0 {
		t.Fatalf("veles test std/regex: exit %d", code)
	}
}
