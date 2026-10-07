package fetch

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// Unpack unzips an archive into a fresh directory below tmpRoot with the
// refusals the client applies to every archive (unsafe paths, links, size).
func Unpack(tmpRoot, zipPath string) (string, error) { return unzip(tmpRoot, zipPath) }

// Token is the credential for registry writes: $VELES_TOKEN, else the
// contents of ~/.veles/token.
func Token() string {
	if t := strings.TrimSpace(os.Getenv("VELES_TOKEN")); t != "" {
		return t
	}
	data, err := os.ReadFile(filepath.Join(Home(), "token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// RegistryURL is the registry a package talks to: $VELES_PROXY, else
// `[registry] url` of its manifest or workspace.
func RegistryURL(root string, man *sema.Manifest) string {
	if p := os.Getenv("VELES_PROXY"); p != "" {
		return p
	}
	return registryFor(root, man).URL
}

// ZipPackage archives the package in dir as `veles publish` uploads it: its
// files at the root of the archive, without `.git`, `vendor/`, `bin/`,
// `veles.sum` and hidden files and directories.
func ZipPackage(dir string) ([]byte, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || (d.IsDir() && (name == "vendor" || name == "bin" || name == "node_modules")) || (!d.IsDir() && name == "veles.sum") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is a symbolic link or special file; a package holds plain files only", filepath.ToSlash(rel))
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	// In a git repository the archive holds what git would: tracked files and
	// untracked ones that are not ignored. A compiled program, an object file or
	// a scratch file the repository's .gitignore keeps out is not published.
	if visible, ok := gitVisible(dir); ok {
		kept := files[:0]
		for _, f := range files {
			if visible[f] {
				kept = append(kept, f)
			}
		}
		files = kept
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		f, err := w.Create(rel)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(data); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// call makes one write request and returns the status and body.
func (f *Fetcher) call(method, url, token string, body []byte) (int, string, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	text, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, strings.TrimSpace(string(text)), nil
}

func registryCall(proxy, method, name, version, ext, token string, body []byte) (int, string, error) {
	if proxy == "" {
		return 0, "", fmt.Errorf("no registry: set [registry] url in veles.toml or VELES_PROXY")
	}
	if err := checkTokenTransport(proxy, token); err != nil {
		return 0, "", err
	}
	f := &Fetcher{}
	return f.call(method, strings.TrimRight(proxy, "/")+"/"+name+"/@v/"+version+"."+ext, token, body)
}

// Publish uploads the package in root to the registry (D139). The manifest
// must name its version and where it publishes. It returns the name and
// version published.
func Publish(root string, man *sema.Manifest, proxy, token string) (string, string, error) {
	if man.Name == "" || man.Version == "" || man.RegistryName == "" {
		return "", "", fmt.Errorf("to publish, veles.toml needs [package] name, version and registry = \"owner/name\"")
	}
	if token == "" {
		return "", "", fmt.Errorf("publishing needs a token: set VELES_TOKEN or write it to %s", filepath.Join(Home(), "token"))
	}
	data, err := ZipPackage(root)
	if err != nil {
		return "", "", err
	}
	code, text, err := registryCall(proxy, http.MethodPut, man.RegistryName, man.Version, "zip", token, data)
	if err != nil {
		return "", "", err
	}
	switch code {
	case http.StatusCreated:
		return man.RegistryName, man.Version, nil
	default:
		return "", "", fmt.Errorf("the registry refused %s %s (%d): %s", man.RegistryName, man.Version, code, text)
	}
}

// Yank marks a published version yanked, or undoes that.
func Yank(proxy, name, version, reason, token string, undo bool) error {
	if token == "" {
		return fmt.Errorf("this needs a token: set VELES_TOKEN or write it to %s", filepath.Join(Home(), "token"))
	}
	method := http.MethodPut
	if undo {
		method = http.MethodDelete
	}
	code, text, err := registryCall(proxy, method, name, version, "yank", token, []byte(reason))
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("the registry refused (%d): %s", code, text)
	}
	return nil
}

// PushStatement sends a signed review to the registry that serves the
// package version it is about.
func PushStatement(proxy string, st Statement, token string) error {
	if st.Kind != "registry" {
		return fmt.Errorf("only a review of a registry package can be pushed to a registry; commit it under attestations/ instead")
	}
	if token == "" {
		return fmt.Errorf("this needs a token: set VELES_TOKEN or write it to %s", filepath.Join(Home(), "token"))
	}
	code, text, err := registryCall(proxy, http.MethodPost, st.Source, st.Ref, "attest", token, []byte(st.String()))
	if err != nil {
		return err
	}
	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("the registry refused the review (%d): %s", code, text)
	}
	return nil
}

// NewClient is a Fetcher for talking to the registry of the package at root
// (its address and pinned key), for listing versions and the like. It has no
// veles.sum: it does not fetch packages.
func NewClient(root string, man *sema.Manifest) *Fetcher {
	f := NewFetcher(nil)
	reg := registryFor(root, man)
	if f.Proxy == "" {
		f.Proxy = reg.URL
	}
	f.RegistryKey = reg.Key
	return f
}

// checkTokenTransport refuses to send a token over plain http to a machine
// that is not this one: the token would cross the network readable by anyone
// on the path.
func checkTokenTransport(base, token string) error {
	if token == "" {
		return nil
	}
	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("the registry address %q is not a URL: %v", base, err)
	}
	if u.Scheme == "https" {
		return nil
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
		return nil
	}
	return fmt.Errorf("refusing to send a token to %s over %s: use an https:// registry (plain http is allowed only for localhost)", host, u.Scheme)
}

// gitVisible lists, as slash paths relative to dir, the files git considers
// part of the working tree below dir (tracked, or untracked and not ignored).
// ok is false when dir is not in a git repository or git is not installed.
func gitVisible(dir string) (map[string]bool, bool) {
	if inside, err := git(dir, "rev-parse", "--is-inside-work-tree"); err != nil || inside != "true" {
		return nil, false
	}
	out, err := git(dir, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, false
	}
	visible := map[string]bool{}
	for _, f := range strings.Split(out, "\x00") {
		if f = strings.TrimSpace(f); f != "" {
			visible[filepath.ToSlash(f)] = true
		}
	}
	return visible, true
}
