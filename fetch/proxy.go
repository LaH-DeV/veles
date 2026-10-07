package fetch

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// The proxy protocol (D138), the first part of the registry's: a base URL
// (VELES_PROXY) and, for a package `owner/name` at a version,
//
//	GET <base>/<owner>/<name>/@v/<version>.zip   the package's files at the zip's root
//
// The archive is immutable once published. Its contents are hashed like a
// git checkout, so a registry and a repository serving the same files agree.
const (
	maxArchive      = 256 << 20 // bytes downloaded
	maxUnpacked     = 512 << 20 // bytes after unzipping: a zip bomb stops here
	maxArchiveFiles = 20000
)

func (f *Fetcher) client() *http.Client {
	if f.HTTP != nil {
		return f.HTTP
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

// proxyCheckout downloads and unpacks one version into a fresh directory.
func (f *Fetcher) proxyCheckout(tmpRoot, name, version string) (string, error) {
	if f.Proxy == "" {
		return "", fmt.Errorf("registry package '%s' needs a proxy: the registry service is not live yet, so set VELES_PROXY to a proxy that serves it, or depend on its git repository instead", name)
	}
	url := strings.TrimRight(f.Proxy, "/") + "/" + name + "/@v/" + version + ".zip"
	resp, err := f.client().Get(url)
	if err != nil {
		return "", fmt.Errorf("cannot fetch %s %s: %v", name, version, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cannot fetch %s %s: %s answered %s", name, version, url, resp.Status)
	}
	zf, err := os.CreateTemp(tmpRoot, "zip-")
	if err != nil {
		return "", err
	}
	defer os.Remove(zf.Name())
	n, err := io.Copy(zf, io.LimitReader(resp.Body, maxArchive+1))
	zf.Close()
	if err != nil {
		return "", fmt.Errorf("cannot fetch %s %s: %v", name, version, err)
	}
	if n > maxArchive {
		return "", fmt.Errorf("%s %s: the archive is larger than %d MB", name, version, maxArchive>>20)
	}
	return unzip(tmpRoot, zf.Name())
}

// unzip unpacks into a fresh directory, refusing every way an archive can
// write outside it or pretend to be something else: absolute and `..`
// paths, links, duplicate names, and sizes past a limit.
func unzip(tmpRoot, zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("the archive is not a zip: %v", err)
	}
	defer r.Close()
	if len(r.File) > maxArchiveFiles {
		return "", fmt.Errorf("the archive holds more than %d files", maxArchiveFiles)
	}
	dir, err := os.MkdirTemp(tmpRoot, "zip-")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(dir)
		}
	}()
	seen := map[string]bool{}
	var total uint64
	for _, zf := range r.File {
		name := zf.Name
		clean := path.Clean(name)
		if unsafeArchivePath(name, clean) {
			return "", fmt.Errorf("the archive has an unsafe path %q", name)
		}
		if zf.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("the archive has a symbolic link %q; a package holds plain files only", name)
		}
		if zf.FileInfo().IsDir() {
			continue
		}
		// compared case-insensitively: on a case-insensitive disk A.vs and a.vs are one
		// file, and the hash would depend on which was written last
		fold := strings.ToLower(clean)
		if seen[fold] {
			return "", fmt.Errorf("the archive lists %q twice (names that differ only in case count as the same)", name)
		}
		seen[fold] = true
		total += zf.UncompressedSize64
		if total > maxUnpacked {
			return "", fmt.Errorf("the archive unpacks to more than %d MB", maxUnpacked>>20)
		}
		target := filepath.Join(dir, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		src, err := zf.Open()
		if err != nil {
			return "", err
		}
		out, err := os.Create(target)
		if err != nil {
			src.Close()
			return "", err
		}
		n, err := io.Copy(out, io.LimitReader(src, int64(zf.UncompressedSize64)+1))
		src.Close()
		out.Close()
		if err != nil {
			return "", err
		}
		if uint64(n) > zf.UncompressedSize64 {
			return "", fmt.Errorf("the archive entry %q is larger than it says", name)
		}
	}
	ok = true
	return dir, nil
}

// unsafeArchivePath: a name that could write outside the unpack directory or
// name something other than a plain file of the package — absolute and
// `..` paths, backslashes and colons (drive letters and NTFS alternate
// data streams), empty and `.` parts, and any `.git` directory, which the
// hash would skip.
func unsafeArchivePath(name, clean string) bool {
	if strings.HasPrefix(name, "/") || strings.ContainsAny(name, `\:`) || clean == ".." || strings.HasPrefix(clean, "../") {
		return true
	}
	for _, part := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return true
		}
	}
	return false
}
