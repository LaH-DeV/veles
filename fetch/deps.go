package fetch

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// a line of the dependency tree
type treeNode struct {
	label    string
	dir      string
	children []*treeNode
	repeat   bool // already shown above: not expanded again
	match    func(name string) bool
}

// graph builds the tree below one package directory.
func (res *Result) graph(dir string, onPath, shown map[string]bool) []*treeNode {
	key := sema.DirKey(dir)
	var out []*treeNode
	for _, e := range res.r.edges {
		if e.importer != key {
			continue
		}
		var n *treeNode
		if e.node == nil {
			rel, err := filepath.Rel(res.root, e.pathDir)
			if err != nil {
				rel = e.pathDir
			}
			n = &treeNode{dir: e.pathDir, label: fmt.Sprintf("%s  path %s", e.name, filepath.ToSlash(rel))}
			name := e.name
			n.match = func(q string) bool { return q == name || q == filepath.Base(e.pathDir) }
		} else {
			sel := res.selected[identityOf(e.node)]
			version := sel.pin.version
			if sel.pin.commit != "" {
				version = "commit " + short(sel.commit)
			}
			label := e.name + " " + version
			if e.node != sel && e.node.pin.version != sel.pin.version {
				label = fmt.Sprintf("%s %s -> %s", e.name, e.node.pin.version, version)
			}
			n = &treeNode{dir: sel.dir, label: label + "  (" + sel.pin.kind + " " + sel.pin.source + ")"}
			name := e.name
			n.match = func(q string) bool { return q == name || q == sel.pin.source || (sel.man != nil && q == sel.man.Name) }
		}
		d := sema.DirKey(n.dir)
		switch {
		case onPath[d]:
			n.label += "  (cycle)"
		case shown[d]:
			n.repeat = true
			n.label += "  (*)"
		default:
			shown[d] = true
			onPath[d] = true
			if _, err := os.Stat(filepath.Join(n.dir, "veles.toml")); err == nil {
				n.children = res.graph(n.dir, onPath, shown)
			}
			delete(onPath, d)
		}
		out = append(out, n)
	}
	return out
}

// Tree is the dependency tree as `veles deps` prints it: each package once
// (a second sight is marked `(*)`), with `a -> b` where version selection
// raised what a manifest asked for.
func (res *Result) Tree() string {
	var b strings.Builder
	b.WriteString(res.rootName + "\n")
	root := res.graph(res.root, map[string]bool{sema.DirKey(res.root): true}, map[string]bool{sema.DirKey(res.root): true})
	var write func(nodes []*treeNode, prefix string)
	write = func(nodes []*treeNode, prefix string) {
		for i, n := range nodes {
			last := i == len(nodes)-1
			branch, next := "+- ", "|  "
			if last {
				branch, next = "`- ", "   "
			}
			b.WriteString(prefix + branch + n.label + "\n")
			write(n.children, prefix+next)
		}
	}
	write(root, "")
	return b.String()
}

// Why answers `veles deps --why <name>`: every chain from the program to a
// package called name (a local name, a manifest name or a source), one per
// line. The second result is false when nothing matches.
func (res *Result) Why(name string) (string, bool) {
	var lines []string
	// expand repeated subtrees too: a package may be reached several ways
	var expand func(dir string, chain []string, onPath map[string]bool)
	expand = func(dir string, chain []string, onPath map[string]bool) {
		for _, n := range res.graph1(dir) {
			d := sema.DirKey(n.dir)
			if onPath[d] {
				continue
			}
			here := append(append([]string{}, chain...), n.label)
			if n.match(name) {
				lines = append(lines, strings.Join(here, "\n  -> "))
			}
			onPath[d] = true
			expand(n.dir, here, onPath)
			delete(onPath, d)
		}
	}
	expand(res.root, []string{res.rootName}, map[string]bool{sema.DirKey(res.root): true})
	sort.Strings(lines)
	return strings.Join(lines, "\n"), len(lines) > 0
}

// graph1 is the direct children of one package directory.
func (res *Result) graph1(dir string) []*treeNode {
	nodes := res.graph(dir, map[string]bool{}, map[string]bool{})
	for _, n := range nodes {
		n.children = nil
	}
	return nodes
}

// PruneSum drops the lines of veles.sum that nothing in this resolution uses
// any more (after `veles remove` or `update`) and returns how many. In a
// workspace the file is shared by members whose graphs this resolution does
// not see, so it is left alone there.
func (res *Result) PruneSum() (int, error) {
	if res.sum == nil || res.sum.Path != filepath.Join(res.root, "veles.sum") {
		return 0, nil
	}
	keep := map[string]bool{}
	for _, n := range res.r.nodes {
		keep[n.sumKey()] = true
	}
	n := res.sum.Prune(keep)
	return n, res.sum.Save()
}

// PruneAllSum removes veles.sum beside root when no registry or git
// dependency is left (the case Resolve returns early for).
func PruneAllSum(root string) (int, error) {
	if sumPathFor(root) != filepath.Join(root, "veles.sum") {
		return 0, nil
	}
	s, err := LoadSum(filepath.Join(root, "veles.sum"))
	if err != nil || len(s.entries) == 0 {
		return 0, err
	}
	n := len(s.entries)
	return n, os.Remove(s.Path)
}

// Prune deletes every entry not in keep and returns how many went.
func (s *Sum) Prune(keep map[string]bool) int {
	n := 0
	for k := range s.entries {
		if !keep[k] {
			delete(s.entries, k)
			n++
		}
	}
	if n > 0 {
		s.dirty = true
	}
	return n
}

// Vendor copies every package version the resolution mentions into
// `vendor/` beside veles.sum, so the project builds without the network or
// the module cache (D132). When `vendor/` exists, builds read packages from
// it and from nowhere else, still checking every one against veles.sum. It
// returns the number of packages copied.
func Vendor(root string, man *sema.Manifest) (int, error) {
	res, err := ResolveWith(root, man, ResolveOptions{IgnoreVendor: true})
	if err != nil {
		return 0, err
	}
	if res.r == nil || len(res.r.nodes) == 0 {
		return 0, fmt.Errorf("there are no registry or git dependencies to vendor")
	}
	dir := filepath.Join(filepath.Dir(res.sum.Path), "vendor")
	if err := removeAll(dir); err != nil {
		return 0, err
	}
	out := &Fetcher{vendor: dir}
	n := 0
	for _, node := range res.r.nodes {
		if err := copyTree(node.dir, out.cacheDir(node.pin)); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}
