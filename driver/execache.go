package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// The program cache: `veles run` and `veles test` keep the programs they
// link, so running again — with another `--filter`, or after an edit that
// did not change the program — skips clang. An entry is keyed by all the
// link reads: the IR, clang, its arguments (the runtime objects are named
// by their own cache key) and the program's name, which the program sees
// as its own. The entries used last are kept, exeCacheSize of them.
//
// Not served: `veles build` and `-o` (the caller wants the file itself), and
// a package with native libraries, whose files are not in the key.

const exeCacheSize = 32

// exeCacheRoot is the cache's directory, or "" when there is no user cache
// directory.
func exeCacheRoot() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "veles", "exe")
}

func exeCacheKey(ir, clang string, args []string, name string) string {
	h := sha256.New()
	fmt.Fprintln(h, clang, runtime.GOOS, runtime.GOARCH, name)
	if info, err := os.Stat(clang); err == nil {
		fmt.Fprintln(h, info.Size(), info.ModTime().UnixNano())
	}
	for _, a := range args {
		fmt.Fprintln(h, len(a), a)
	}
	fmt.Fprintln(h, len(ir))
	h.Write([]byte(ir))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// exeCacheLookup returns the cached program for key, marked as used, or ""
// when there is none.
func exeCacheLookup(root, key, name string) string {
	entry := filepath.Join(root, key)
	p := filepath.Join(entry, name)
	if info, err := os.Stat(p); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	now := time.Now()
	_ = os.Chtimes(entry, now, now) // the entry's time orders the eviction
	return p
}

// exeCacheStore installs the program just linked at exe under key and drops
// the entries used longest ago. The entry is put together in a directory
// of its own and renamed into place, so no run ever starts a half-copied
// program. Failing to cache is not an error: the build has its program.
func exeCacheStore(root, key, name, exe string) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return
	}
	tmp, err := os.MkdirTemp(root, "tmp-")
	if err != nil {
		return
	}
	if err := linkOrCopy(exe, filepath.Join(tmp, name)); err != nil {
		os.RemoveAll(tmp)
		return
	}
	if err := os.Rename(tmp, filepath.Join(root, key)); err != nil {
		os.RemoveAll(tmp) // a concurrent run installed the same program first
	}
	exeCachePrune(root, exeCacheSize)
}

// exeCachePrune keeps the keep entries used last. An entry another process
// is running cannot be removed on Windows; it goes on a later prune. A
// directory left by a store that died half-way is removed after an hour.
func exeCachePrune(root string, keep int) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return
	}
	type entry struct {
		path string
		used time.Time
	}
	var live []entry
	for _, e := range ents {
		info, err := e.Info()
		if err != nil || !e.IsDir() {
			continue
		}
		p := filepath.Join(root, e.Name())
		if strings.HasPrefix(e.Name(), "tmp-") {
			if time.Since(info.ModTime()) > time.Hour {
				os.RemoveAll(p)
			}
			continue
		}
		live = append(live, entry{p, info.ModTime()})
	}
	if len(live) <= keep {
		return
	}
	sort.Slice(live, func(i, j int) bool { return live[i].used.After(live[j].used) })
	for _, e := range live[keep:] {
		os.RemoveAll(e.path)
	}
}

// linkOrCopy puts src's contents at dst: a hard link where the two are on
// one volume, a copy otherwise.
func linkOrCopy(src, dst string) error {
	if os.Link(src, dst) == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
