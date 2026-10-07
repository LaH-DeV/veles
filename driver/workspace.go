package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// A workspace (D138) is a monorepo root: a directory whose veles.toml has
// `[workspace] members = [...]`. `veles build`, `test`, `check` (and `check
// --fix`) pointed at that directory run over every member in turn, each as the
// package it is; pointed at anything inside a member they behave as always.
// A failing member does not stop the others — a CI run should show every
// broken member at once — and the exit status is 1 if any failed.

// workspaceAt returns the manifest when path is exactly the root directory of
// a workspace.
func workspaceAt(path string) (string, *sema.Manifest, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, nil
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return "", nil, nil
	}
	if _, err := os.Stat(filepath.Join(abs, "veles.toml")); err != nil {
		return "", nil, nil
	}
	man, err := sema.ReadManifestIn(abs)
	if err != nil {
		return "", nil, err
	}
	if man == nil || man.Workspace == nil {
		return "", nil, nil
	}
	return abs, man, nil
}

type workspaceMember struct {
	dir  string // absolute
	rel  string // as listed
	name string
}

// members resolves and checks the member list: each a directory with a
// manifest of its own that names a package, no two with one name, none a
// workspace itself.
func workspaceMembers(root string, man *sema.Manifest) ([]workspaceMember, error) {
	var out []workspaceMember
	byName := map[string]string{}
	if man.Name != "" {
		out = append(out, workspaceMember{dir: root, rel: ".", name: man.Name})
		byName[man.Name] = "."
	}
	for _, rel := range man.Workspace.Members {
		dir := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(filepath.Join(dir, "veles.toml")); err != nil {
			return nil, fmt.Errorf("workspace member '%s' (%s) has no veles.toml: every member is a package with a manifest of its own", rel, filepath.Join(root, "veles.toml"))
		}
		mm, err := sema.ReadManifestIn(dir)
		if err != nil {
			return nil, err
		}
		switch {
		case mm.Workspace != nil:
			return nil, fmt.Errorf("workspace member '%s' is itself a workspace; list its members in the root manifest instead", rel)
		case mm.Name == "":
			return nil, fmt.Errorf("workspace member '%s' has no [package] name", rel)
		}
		if other, dup := byName[mm.Name]; dup {
			return nil, fmt.Errorf("workspace members '%s' and '%s' are both named '%s'; a package name is unique in a workspace", other, rel, mm.Name)
		}
		byName[mm.Name] = rel
		out = append(out, workspaceMember{dir: dir, rel: rel, name: mm.Name})
	}
	return out, nil
}

// runWorkspace runs opts over every member of the workspace at root.
func runWorkspace(opts Options, root string, man *sema.Manifest) int {
	members, err := workspaceMembers(root, man)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	}
	if opts.Mode == "run" {
		var names []string
		for _, m := range members {
			names = append(names, m.rel)
		}
		fmt.Fprintf(os.Stderr, "veles: a workspace holds several packages, so there is nothing to run at its root: veles run <member> (members: %s)\n", strings.Join(names, ", "))
		return 2
	}
	if opts.Output != "" {
		fmt.Fprintln(os.Stderr, "veles: -o names one executable, and a workspace builds one per member; build a member to choose its name")
		return 2
	}
	var failed []string
	for _, m := range members {
		fmt.Fprintf(os.Stderr, "==> %s (%s)\n", m.rel, m.name)
		member := opts
		member.Path = m.dir
		member.inWorkspace = true
		if opts.Mode == "build" {
			// next to the members, not in the caller's directory, where an
			// executable named like its member would collide with the member's own
			// directory
			member.Output = filepath.Join(root, "bin", m.name)
			os.MkdirAll(filepath.Join(root, "bin"), 0o755)
		}
		if code := Run(member); code != 0 {
			failed = append(failed, m.name)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(os.Stderr, "veles: %d of %d workspace members failed: %s\n", len(failed), len(members), strings.Join(failed, ", "))
		return 1
	}
	fmt.Fprintf(os.Stderr, "veles: %d workspace members ok\n", len(members))
	return 0
}
