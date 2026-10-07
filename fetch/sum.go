package fetch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// Sum is `veles.sum` (D132, D138): one line per package version in the
// build, committed with the source,
//
//	git https://github.com/veles-db/pg 2.1.0 h1:3b6f…
//	git https://example.com/fast commit:3f2a9c1d… h1:9a01…
//	registry acme/httputil 1.4.2 h1:77ce…
//
// A hash that differs from the line is a hard error: the tag was moved, the
// registry changed an archive, or the cache was edited.
type Sum struct {
	Path    string
	entries map[string]string // "kind source ref" -> hash
	dirty   bool
}

// sumPathFor is where a package's veles.sum lives: beside the manifest of the
// workspace that lists it as a member, else beside its own.
func sumPathFor(root string) string {
	for dir := filepath.Dir(root); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "veles.toml")); err == nil {
			if m, err := sema.ReadManifestIn(dir); err == nil && m != nil && m.Workspace != nil && isMember(dir, root, m.Workspace) {
				return filepath.Join(dir, "veles.sum")
			}
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}
	return filepath.Join(root, "veles.sum")
}

func isMember(wsDir, root string, w *sema.ManifestWorkspace) bool {
	rel, err := filepath.Rel(wsDir, root)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, m := range w.Members {
		if m == rel {
			return true
		}
	}
	return false
}

// LoadSum reads a sum file; a missing file is an empty one.
func LoadSum(path string) (*Sum, error) {
	s := &Sum{Path: path, entries: map[string]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	for n, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 4 || !strings.HasPrefix(f[3], "h1:") || (f[0] != "git" && f[0] != "registry") {
			return nil, fmt.Errorf("%s:%d: a veles.sum line is '<git|registry> <source> <version|commit:hash> h1:<hash>'", path, n+1)
		}
		key := strings.Join(f[:3], " ")
		if old, dup := s.entries[key]; dup && old != f[3] {
			return nil, fmt.Errorf("%s:%d: %s is listed twice with different hashes", path, n+1, key)
		}
		s.entries[key] = f[3]
	}
	return s, nil
}

// Has reports whether veles.sum already has a line for the package version
// (key is "kind source ref").
func (s *Sum) Has(key string) bool {
	_, ok := s.entries[key]
	return ok
}

// Check compares hash with the recorded one: an unknown key is added (the
// first fetch records it), a different hash is the error.
func (s *Sum) Check(key, hash string) error {
	if old, ok := s.entries[key]; ok {
		if old != hash {
			return fmt.Errorf("checksum mismatch for %s\n  %s records %s\n  the files hash to %s\n  The tag may have been moved, the archive changed, or the module cache edited. Nothing was built. If you trust the new contents, delete that line from veles.sum (and the version's directory in the module cache, if it is there)", key, s.Path, old, hash)
		}
		return nil
	}
	s.entries[key] = hash
	s.dirty = true
	return nil
}

// Save writes the file, sorted, when anything was added.
func (s *Sum) Save() error {
	if !s.dirty {
		return nil
	}
	keys := make([]string, 0, len(s.entries))
	for k := range s.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k + " " + s.entries[k] + "\n")
	}
	if err := os.WriteFile(s.Path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	s.dirty = false
	return nil
}
