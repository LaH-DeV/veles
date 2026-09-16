package lsp

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/std"
)

// The standard library is embedded in the compiler, so `std/io/io.vs` is not
// a file the editor can open. For go-to-definition the server writes the
// embedded sources — plus the built-ins stub (sema.BuiltinStub) — into a
// cache directory once per start, and maps `std/...` paths there. Those
// files are reference copies: the server does not analyse them.

// stdCacheDir is where the standard library sources are materialised, or
// "" when no cache directory is available.
func stdCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "veles", "std")
}

// materializeStd writes the embedded standard library and the built-ins
// stub into the cache directory, replacing files whose content changed.
func (s *Server) materializeStd() {
	dir := stdCacheDir()
	if dir == "" {
		return
	}
	write := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if old, err := os.ReadFile(p); err == nil && string(old) == content {
			return
		}
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	fs.WalkDir(std.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".vs") {
			return nil
		}
		data, err := fs.ReadFile(std.FS, path)
		if err == nil {
			write(path, string(data))
		}
		return nil
	})
	stub := sema.BuiltinStub()
	write(strings.TrimPrefix(sema.BuiltinStubPath, "std/"), stub.Content)
	s.stdDir = dir
}

// stdPath maps a `std/...` source path to its cached copy, or "" when the
// cache is unavailable.
func (s *Server) stdPath(path string) string {
	if s.stdDir == "" || !strings.HasPrefix(path, "std/") {
		return ""
	}
	return filepath.Join(s.stdDir, filepath.FromSlash(strings.TrimPrefix(path, "std/")))
}

// inStdCache reports whether a file lives in the materialised standard
// library, which the server shows but does not analyse.
func (s *Server) inStdCache(path string) bool {
	if s.stdDir == "" {
		return false
	}
	rel, err := filepath.Rel(s.stdDir, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}
