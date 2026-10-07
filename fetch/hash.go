// Package fetch gets a program's registry and git dependencies (D138): it
// resolves the graph by minimal version selection per (source, major), keeps
// each version once in a module cache, and checks every one against
// `veles.sum`. The compiler core reads only directories; this package is the
// only code that touches the network or runs git.
package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// markerFile is written into a cached package after it is fetched: its
// hash, and for a commit pin the full commit. It is not part of the package
// and is left out of the hash.
const markerFile = ".veles-fetched"

// TreeHash is the `h1:` hash of a package's files: SHA-256 over each file's
// slash path and the SHA-256 of its exact bytes, sorted by path. Nothing but
// bytes goes in — no modes, no times, no line-ending conversion — so it is
// the same on every machine. `.git` and the marker are not part of the
// package; a symbolic link is refused, since a package must not point out
// of its own tree.
func TreeHash(dir string) (string, error) {
	type entry struct{ rel, path string }
	var files []entry
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == markerFile {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is a symbolic link or special file; a package holds plain files only", rel)
		}
		files = append(files, entry{rel, path})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	h := sha256.New()
	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(h, "%s\x00%x\n", f.rel, sum)
	}
	return "h1:" + hex.EncodeToString(h.Sum(nil)), nil
}

// removeAll deletes a directory tree, including the read-only files git
// leaves in `.git` (Windows refuses to remove those otherwise).
func removeAll(dir string) error {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			os.Chmod(path, 0o777)
		}
		return nil
	})
	return os.RemoveAll(dir)
}
