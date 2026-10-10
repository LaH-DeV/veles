package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The `[native]` table (D67): four list keys, anything else refused with
// the keys that exist.
func TestManifestNative(t *testing.T) {
	read := func(text string) (*Manifest, error) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "veles.toml"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return readManifest(dir)
	}
	m, err := read("[package]\nname = \"p\"\n\n[native]\nlibs = [\"pq\", \"vendor/x.a\"]\nstatic-libs = [\"z\"]\nlib-paths = [\"vendor\"]\npkg-config = [\"libpq\"]\n")
	if err != nil {
		t.Fatal(err)
	}
	n := m.Native
	if strings.Join(n.Libs, ",") != "pq,vendor/x.a" || strings.Join(n.StaticLibs, ",") != "z" ||
		strings.Join(n.LibPaths, ",") != "vendor" || strings.Join(n.PkgConfig, ",") != "libpq" {
		t.Errorf("parsed %+v", n)
	}
	for _, c := range []struct{ text, want string }{
		{"[package]\nname = \"p\"\n[native]\nlink = [\"z\"]\n", "unknown [native] key \"link\" (libs, static-libs, lib-paths, pkg-config)"},
		{"[package]\nname = \"p\"\n[native]\nlibs = \"z\"\n", "[native] libs is a list"},
	} {
		if _, err := read(c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.text, err, c.want)
		}
	}
}

// The `[runtime]` table (D143): threads is "auto" or a count.
func TestManifestRuntime(t *testing.T) {
	read := func(text string) (*Manifest, error) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "veles.toml"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return readManifest(dir)
	}
	for text, want := range map[string]int64{
		"[package]\nname = \"p\"\n":                                0,
		"[package]\nname = \"p\"\n[runtime]\nthreads = \"auto\"\n": 0,
		"[package]\nname = \"p\"\n[runtime]\nthreads = 4\n":        4,
	} {
		m, err := read(text)
		if err != nil || m.Runtime.Threads != want {
			t.Errorf("%q: got %+v, %v; want %d", text, m, err, want)
		}
	}
	for _, c := range []struct{ text, want string }{
		{"[package]\nname = \"p\"\n[runtime]\nthreads = 0\n", "[runtime] threads is \"auto\" (one per core, the default) or a number from 1 to 256"},
		{"[package]\nname = \"p\"\n[runtime]\nthreads = \"many\"\n", "[runtime] threads is \"auto\""},
		{"[package]\nname = \"p\"\n[runtime]\nworkers = 2\n", "unknown [runtime] key \"workers\" (threads)"},
	} {
		if _, err := read(c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.text, err, c.want)
		}
	}
}
