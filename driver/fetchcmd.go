package driver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/sema"
)

// Fetch is `veles fetch [dir]` (D138): resolve the registry and git
// dependencies of the package at path — or of every member of a workspace —
// download what the module cache lacks, and record or verify each in
// veles.sum. A build does the same on its own; this is for CI (fetch once,
// then build offline) and for the language server, which never downloads.
func Fetch(path string) int {
	roots := []string{}
	if root, man, err := workspaceAt(path); err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		return 1
	} else if man != nil {
		members, err := workspaceMembers(root, man)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		for _, m := range members {
			roots = append(roots, m.dir)
		}
	} else {
		root, _, err := sema.FindRoot(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		roots = append(roots, root)
	}
	total := 0
	for _, root := range roots {
		if _, err := os.Stat(filepath.Join(root, "veles.toml")); err != nil {
			fmt.Fprintf(os.Stderr, "veles: %s has no veles.toml: there is nothing to fetch\n", root)
			return 1
		}
		man, err := sema.ReadManifestIn(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		res, err := fetch.ResolveWith(root, man, fetch.ResolveOptions{Refresh: true})
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		for _, w := range res.Warnings {
			fmt.Fprintln(os.Stderr, "warning:", w)
		}
		for _, line := range res.Selected {
			fmt.Println(line)
		}
		total += len(res.Selected)
	}
	if total == 0 {
		fmt.Println("no registry or git dependencies")
	}
	return 0
}
