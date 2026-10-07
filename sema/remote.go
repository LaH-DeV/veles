package sema

// Remote is the outcome of resolving a program's registry and git
// dependencies (D138): which directory holds the version each package
// selected. The compiler core does no fetching; the `fetch` package installs
// ResolveRemote (the driver and the language server import it), so the
// loader reads dependencies from directories and nothing else.
type Remote struct {
	// Dirs is, for each package that has remote dependencies (keyed by
	// DirKey of its directory), the directory of the selected version of each
	// of them, by the package's local name for it.
	Dirs map[string]map[string]string
	// Warnings are reported once, before the program is checked.
	Warnings []string
}

// ResolveRemote resolves the dependency graph of the package at root, which
// has read manifest man. It is nil until the fetch package is linked in, and
// then every dependency that is not a path is refused.
var ResolveRemote func(root string, man *Manifest) (*Remote, error)

// DirKey is the key under which a package directory is known to the loader
// and to ResolveRemote: the resolved path, so a symlink or another spelling
// of it is the same package.
func DirKey(dir string) string { return identityOfDir(dir) }

// remoteDir returns the directory a remote dependency of the package was
// resolved to.
func (t *depTable) remoteDir(importer *Manifest, name string) (string, bool) {
	if t.remote == nil || importer == nil {
		return "", false
	}
	dir, ok := t.remote.Dirs[DirKey(importer.Dir)][name]
	return dir, ok
}
