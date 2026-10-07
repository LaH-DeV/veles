package driver

import (
	"os"
	"path/filepath"
	"runtime"
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

// TestFormatWritesAtomically: the rewrite keeps the file's permission bits
// and leaves no temporary file next to it.
func TestFormatWritesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	os.WriteFile(path, []byte("fun main() {\nx()\n}\n"), 0o600)
	if code := Format(FormatOptions{Paths: []string{path}}); code != 0 {
		t.Fatalf("format: exit %d", code)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("a temporary file was left behind: %v", entries)
	}
	if data, _ := os.ReadFile(path); string(data) != "fun main() {\n  x()\n}\n" {
		t.Errorf("not formatted: %q", data)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Errorf("mode changed to %v", info.Mode().Perm())
		}
	}
}

// TestFormatReportsWalkErrors: a tree that cannot be fully read fails the
// run, so `--check` never passes on files it did not see.
func TestFormatReportsWalkErrors(t *testing.T) {
	if code := Format(FormatOptions{Paths: []string{filepath.Join(t.TempDir(), "missing")}, Check: true}); code != 1 {
		t.Errorf("a missing path: exit %d, want 1", code)
	}
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a directory the test cannot read")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	os.Mkdir(locked, 0o755)
	os.WriteFile(filepath.Join(locked, "a.vs"), []byte("fun main() {\nx()\n}\n"), 0o644)
	os.Chmod(locked, 0)
	defer os.Chmod(locked, 0o755)
	if code := Format(FormatOptions{Paths: []string{dir}, Check: true}); code != 1 {
		t.Errorf("an unreadable directory: exit %d, want 1", code)
	}
}

// `veles fmt` leaves vendor/ alone: reformatting a vendored package would
// change the bytes veles.sum vouches for.
func TestFormatSkipsVendor(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "vendor", "git", "x"), 0o755)
	src := "fun main() {\nx()\n}\n"
	vendored := filepath.Join(dir, "vendor", "git", "x", "lib.vs")
	os.WriteFile(vendored, []byte(src), 0o644)
	os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644)
	if code := Format(FormatOptions{Paths: []string{dir}}); code != 0 {
		t.Fatalf("format: exit %d", code)
	}
	if data, _ := os.ReadFile(vendored); string(data) != src {
		t.Errorf("vendor/ was formatted: %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "main.vs")); string(data) == src {
		t.Error("main.vs was not formatted")
	}
}
