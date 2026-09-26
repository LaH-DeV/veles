package sema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Manifest is `veles.toml` (M1): package identity lives here, never in
// source. The bootstrap reads a small TOML subset:
//
//	[package]
//	name = "app"
//	version = "0.1.0"
//	exports = ["geometry"]          # modules other packages may import
//
//	[dependencies]
//	mathlib = "../mathlib"                 # short for { path = "../mathlib" }
//
// Dependencies are path-based; the registry and minimal version selection
// (M7) are not part of the bootstrap.
type Manifest struct {
	Name    string
	Version string
	Exports []string
	Deps    map[string]string // name -> path (relative to the manifest)
	Dir     string
	// Format holds the `[format]` table for `veles fmt`: Indent is "" for
	// the default, a run of spaces, or a tab; MaxBlankLines is 0 for the
	// default.
	Format ManifestFormat
	// Native is the `[native]` table (D67): the C libraries the package's
	// `extern` blocks need at link time.
	Native ManifestNative
}

// ManifestNative is what a package links besides Veles code (D67):
//
//	[native]
//	libs = ["pq"]                 # -lpq: the shared library (or its import library)
//	static-libs = ["z"]           # linked into the binary: no DLL/.so at run time
//	lib-paths = ["vendor/lib"]    # searched first; relative to veles.toml
//	pkg-config = ["libpq"]        # flags from `pkg-config --libs`
//
// A `libs`/`static-libs` entry that names a file (it has a directory or an
// archive extension) is linked as that file, relative to veles.toml.
type ManifestNative struct {
	Libs       []string
	StaticLibs []string
	LibPaths   []string
	PkgConfig  []string
}

func (n ManifestNative) empty() bool {
	return len(n.Libs)+len(n.StaticLibs)+len(n.LibPaths)+len(n.PkgConfig) == 0
}

func readManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "veles.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	m := &Manifest{Deps: map[string]string{}, Dir: dir}
	section := ""
	for n, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			return nil, fmt.Errorf("%s:%d: expected 'key = value'", path, n+1)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		switch section {
		case "package":
			switch key {
			case "name":
				m.Name = tomlString(val)
			case "version":
				m.Version = tomlString(val)
			case "exports":
				m.Exports = tomlStringList(val)
			}
		case "dependencies":
			// `name = "../dir"` is the short form of `name = { path = "../dir" }`.
			p := tomlInlinePath(val)
			if p == "" && strings.HasPrefix(val, "\"") {
				p = tomlString(val)
			}
			if p == "" {
				return nil, fmt.Errorf("%s:%d: dependency '%s' must be a path in the bootstrap, e.g. '%s = \"../%s\"' (M7's registry is not implemented)", path, n+1, key, key, key)
			}
			m.Deps[key] = p
		case "format":
			switch key {
			case "indent":
				switch v := tomlString(val); v {
				case "tab":
					m.Format.Indent = "\t"
				default:
					width, err := strconv.Atoi(v)
					if err != nil || width < 1 || width > 8 {
						return nil, fmt.Errorf("%s:%d: [format] indent must be a number of spaces (1-8) or \"tab\"", path, n+1)
					}
					m.Format.Indent = strings.Repeat(" ", width)
				}
			case "max_blank_lines":
				limit, err := strconv.Atoi(tomlString(val))
				if err != nil || limit < 1 {
					return nil, fmt.Errorf("%s:%d: [format] max_blank_lines must be a positive number", path, n+1)
				}
				m.Format.MaxBlankLines = limit
			default:
				return nil, fmt.Errorf("%s:%d: unknown [format] key %q (indent, max_blank_lines)", path, n+1, key)
			}
		case "native":
			if !strings.HasPrefix(val, "[") {
				return nil, fmt.Errorf("%s:%d: [native] %s is a list, e.g. %s = [\"z\"]", path, n+1, key, key)
			}
			list := tomlStringList(val)
			switch key {
			case "libs":
				m.Native.Libs = list
			case "static-libs":
				m.Native.StaticLibs = list
			case "lib-paths":
				m.Native.LibPaths = list
			case "pkg-config":
				m.Native.PkgConfig = list
			default:
				return nil, fmt.Errorf("%s:%d: unknown [native] key %q (libs, static-libs, lib-paths, pkg-config)", path, n+1, key)
			}
		}
	}
	if m.Name == "" {
		return nil, fmt.Errorf("%s: [package] name is required", path)
	}
	return m, nil
}

func tomlString(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}

func tomlStringList(v string) []string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[")
	v = strings.TrimSuffix(v, "]")
	var out []string
	for _, part := range strings.Split(v, ",") {
		if s := tomlString(strings.TrimSpace(part)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func tomlInlinePath(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "{") {
		return ""
	}
	v = strings.TrimSuffix(strings.TrimPrefix(v, "{"), "}")
	for _, part := range strings.Split(v, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "path" {
			return tomlString(strings.TrimSpace(kv[1]))
		}
	}
	return ""
}

// exported reports whether a module path of this package may be imported
// from another package: listed in exports, or the root module.
func (m *Manifest) exported(modPath string) bool {
	if modPath == "" {
		return true
	}
	for _, e := range m.Exports {
		if e == modPath {
			return true
		}
	}
	return false
}

// stripComment removes a `#` comment that is not inside a string.
func stripComment(line string) string {
	inString := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inString = !inString
		case '#':
			if !inString {
				return line[:i]
			}
		}
	}
	return line
}

// ManifestFormat is the `[format]` table.
type ManifestFormat struct {
	Indent        string
	MaxBlankLines int
}

// ReadManifest reads the `veles.toml` of the package containing path, or
// returns nil when there is none.
func ReadManifest(path string) (*Manifest, error) {
	root, _, err := FindRoot(path)
	if err != nil {
		return nil, err
	}
	return readManifest(root)
}

// NativeManifests is every manifest in the program with a `[native]`
// table: this package's and its dependencies', each once. The driver turns
// them into linker flags; paths in them are relative to Manifest.Dir.
func (p *Package) NativeManifests() []*Manifest {
	var out []*Manifest
	seen := map[string]bool{}
	var visit func(q *Package)
	visit = func(q *Package) {
		if q == nil {
			return
		}
		if m := q.Manifest; m != nil && !seen[m.Dir] {
			seen[m.Dir] = true
			if !m.Native.empty() {
				out = append(out, m)
			}
		}
		names := make([]string, 0, len(q.Deps))
		for name := range q.Deps {
			names = append(names, name)
		}
		sort.Strings(names) // link order must not depend on map order
		for _, name := range names {
			visit(q.Deps[name])
		}
	}
	visit(p)
	return out
}
