package driver

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFormatUsesManifest: `[format]` in veles.toml sets the style, and
// --check reports files that would change without touching them.
func TestFormatUsesManifest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "veles.toml"), []byte("[package]\nname = \"app\"\n\n[format]\nindent = 4\n"), 0o644)
	path := filepath.Join(dir, "main.vs")
	src := "fun main() {\n  x()\n}\n"
	os.WriteFile(path, []byte(src), 0o644)
	if code := Format(FormatOptions{Paths: []string{dir}, Check: true}); code != 1 {
		t.Errorf("--check on an unformatted file: exit %d, want 1", code)
	}
	if data, _ := os.ReadFile(path); string(data) != src {
		t.Errorf("--check rewrote the file")
	}
	if code := Format(FormatOptions{Paths: []string{dir}}); code != 0 {
		t.Errorf("format: exit %d", code)
	}
	if data, _ := os.ReadFile(path); string(data) != "fun main() {\n    x()\n}\n" {
		t.Errorf("formatted with the manifest's indent: %q", data)
	}
	if code := Format(FormatOptions{Paths: []string{dir}, Check: true}); code != 0 {
		t.Errorf("--check after formatting: exit %d, want 0", code)
	}
}
