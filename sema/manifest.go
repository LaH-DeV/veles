package sema

import (
	"fmt"
	"os"
	"path/filepath"
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
