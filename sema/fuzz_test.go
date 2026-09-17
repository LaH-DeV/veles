package sema

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/source"
)

// FuzzCheck: the checker must finish, without panicking, on any program the
// parser accepts (and on ones it does not: diagnostics, never a crash).
func FuzzCheck(f *testing.F) {
	for _, root := range []string{"../examples", "../std"} {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".vs") {
				if data, err := os.ReadFile(path); err == nil {
					f.Add(string(data))
				}
			}
			return nil
		})
	}
	fence := regexp.MustCompile("(?s)```veles\n(.*?)```")
	filepath.WalkDir("../docs/documentation", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, _ := os.ReadFile(path)
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		for _, m := range fence.FindAllStringSubmatch(text, -1) {
			f.Add(m[1])
		}
		return nil
	})
	dir := f.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte("fun main() { }\n"), 0o644); err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, src string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			diags := &source.Diagnostics{}
			pkg, err := LoadPackageOverlay(dir, diags, map[string]string{OverlayKey(path): src})
			if err != nil {
				return
			}
			if !diags.HasErrors() {
				Check(pkg, diags, false)
			}
		}()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Fatalf("checking did not finish within 20s:\n%s", src)
		}
	})
}
