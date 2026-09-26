package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// nativeFlags turns the `[native]` tables of a program's manifests (D67)
// into clang link arguments: library paths first, then the libraries in
// manifest order, a package before its dependencies.
func nativeFlags(clang string, manifests []*sema.Manifest) ([]string, error) {
	var paths, libs []string
	for _, m := range manifests {
		where := filepath.Join(m.Dir, "veles.toml")
		var dirs []string
		for _, p := range m.Native.LibPaths {
			dir := resolve(m.Dir, p)
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				return nil, fmt.Errorf("%s: [native] lib-paths: %q is not a directory", where, p)
			}
			dirs = append(dirs, dir)
			paths = append(paths, "-L"+dir)
		}
		for _, lib := range m.Native.Libs {
			if isLibFile(lib) {
				libs = append(libs, resolve(m.Dir, lib))
			} else {
				libs = append(libs, "-l"+lib)
			}
		}
		for _, lib := range m.Native.StaticLibs {
			if isLibFile(lib) {
				libs = append(libs, resolve(m.Dir, lib))
				continue
			}
			archive, err := findArchive(clang, lib, dirs)
			if err != nil {
				return nil, fmt.Errorf("%s: [native] static-libs: %v", where, err)
			}
			libs = append(libs, archive)
		}
		if len(m.Native.PkgConfig) > 0 {
			flags, err := pkgConfigLibs(m.Native.PkgConfig)
			if err != nil {
				return nil, fmt.Errorf("%s: [native] pkg-config: %v", where, err)
			}
			libs = append(libs, flags...)
		}
	}
	return append(paths, libs...), nil
}

// isLibFile: an entry that names a file rather than a library — it has a
// directory part or a library/object extension.
func isLibFile(s string) bool {
	if strings.ContainsAny(s, `/\`) {
		return true
	}
	for _, ext := range []string{".a", ".lib", ".o", ".obj", ".so", ".dylib", ".dll"} {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

func resolve(dir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(dir, p)
}

// findArchive locates the static archive for `name`: the manifest's own
// lib-paths first, then wherever clang would look. Naming the archive
// rather than passing -lname is what makes the link static on every
// platform — the linker cannot prefer a shared library it is not shown.
func findArchive(clang, name string, dirs []string) (string, error) {
	candidates := []string{"lib" + name + ".a"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, name+".lib")
	}
	for _, dir := range dirs {
		for _, c := range candidates {
			if p := filepath.Join(dir, c); fileExists(p) {
				return p, nil
			}
		}
	}
	for _, c := range candidates {
		out, err := exec.Command(clang, "-print-file-name="+c).Output()
		if err != nil {
			continue
		}
		// clang echoes the bare name back when it finds nothing
		if p := strings.TrimSpace(string(out)); p != c && fileExists(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("no static archive for %q (looked for %s in lib-paths and clang's library directories)", name, strings.Join(candidates, " or "))
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func pkgConfigLibs(names []string) ([]string, error) {
	tool, err := exec.LookPath("pkg-config")
	if err != nil {
		return nil, fmt.Errorf("pkg-config is not installed; name the libraries under 'libs' (and 'lib-paths') instead")
	}
	out, err := exec.Command(tool, append([]string{"--libs"}, names...)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return strings.Fields(string(out)), nil
}
