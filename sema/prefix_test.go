package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/source"
)

// TestCheckEveryPrefix types each example in, line by line: every prefix of
// every program is loaded and checked, as the language server does while
// the user writes. None may panic or hang; diagnostics are the expected
// outcome for most of them.
func TestCheckEveryPrefix(t *testing.T) {
	var files []string
	filepath.WalkDir("../examples", func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, "main.vs") {
			files = append(files, path)
		}
		return nil
	})
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := strings.ReplaceAll(string(data), "\r\n", "\n")
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			main := filepath.Join(dir, "main.vs")
			os.WriteFile(main, []byte("fun main() { }\n"), 0o644)
			lines := strings.SplitAfter(src, "\n")
			for i := range lines {
				prefix := strings.Join(lines[:i+1], "")
				done := make(chan struct{})
				go func() {
					defer close(done)
					diags := &source.Diagnostics{}
					pkg, err := LoadPackageOverlay(dir, diags, map[string]string{OverlayKey(main): prefix})
					if err == nil && !diags.HasErrors() {
						Check(pkg, diags, false)
					}
				}()
				select {
				case <-done:
				case <-time.After(20 * time.Second):
					t.Fatalf("%s: checking the first %d lines did not finish", path, i+1)
				}
			}
		})
	}
}
