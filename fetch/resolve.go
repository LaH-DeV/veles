package fetch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

func init() {
	sema.ResolveRemote = func(root string, man *sema.Manifest) (*sema.Remote, error) {
		res, err := Resolve(root, man)
		if err != nil {
			return nil, err
		}
		return res.Remote, nil
	}
}

// Result is a resolution: what the loader needs, a line per package
// selected for `veles fetch` to print, and the graph the other commands
// (`deps`, `vendor`, `remove`) read.
type Result struct {
	*sema.Remote
	Selected []string

	root     string
	rootName string
	selected map[string]*fetched // by identity
	r        *resolver
	sum      *Sum
	policy   sema.ManifestPolicy
}

// An edge of the dependency graph: a package's requirement, by the name the
// importer gives it. A remote one points at the version that was asked for
// (the selected one is looked up by identity); a path one at the directory.
type edge struct {
	importer string // sema.DirKey of the importing package's directory
	name     string
	node     *fetched
	pathDir  string
}

// ResolveOptions changes how a resolution gets its packages.
type ResolveOptions struct {
	// IgnoreVendor reads the module cache and the network even when the
	// project has a vendor directory (what `veles vendor` itself needs).
	IgnoreVendor bool
	// SkipPolicy does not refuse packages the [policy] denies (`veles audit`
	// reports them instead).
	SkipPolicy bool
	// Refresh asks the registry again about cached packages (yanks, tier).
	Refresh bool
}

type resolver struct {
	f        *Fetcher
	nodes    map[string]*fetched // every package version mentioned, by kind, source and ref
	visited  map[string]bool     // path-dependency packages already scanned
	edges    []edge
	warnings []string
}

// Resolve resolves the registry and git dependencies of the package at root
// (D138): it walks the graph, fetching each version a manifest mentions into
// the cache to read its own requirements, and selects for each (source, major)
// the highest version anyone asked for — minimal version selection, so the
// requirement lists alone determine the build and `veles.sum` makes it
// byte-for-byte reproducible. A graph with no remote dependency touches
// neither the network, the cache nor veles.sum.
func Resolve(root string, man *sema.Manifest) (*Result, error) {
	return ResolveWith(root, man, ResolveOptions{})
}

// ResolveWith is Resolve with options.
func ResolveWith(root string, man *sema.Manifest, opts ResolveOptions) (*Result, error) {
	res := &Result{Remote: &sema.Remote{Dirs: map[string]map[string]string{}}, root: root, rootName: man.Name}
	if !hasRemote(root, man, map[string]bool{}) {
		// nothing to fetch, but `veles deps` still draws the path dependencies
		r := &resolver{nodes: map[string]*fetched{}, visited: map[string]bool{sema.DirKey(root): true}}
		if err := r.scan(root, man, nil); err != nil {
			return nil, err
		}
		res.r, res.selected = r, map[string]*fetched{}
		return res, nil
	}
	sum, err := LoadSum(sumPathFor(root))
	if err != nil {
		return nil, err
	}
	f := NewFetcher(sum)
	reg := registryFor(root, man)
	if f.Proxy == "" {
		f.Proxy = reg.URL
	}
	f.RegistryKey = reg.Key
	f.Refresh = opts.Refresh
	if vendor := filepath.Join(filepath.Dir(sum.Path), "vendor"); !opts.IgnoreVendor {
		if info, err := os.Stat(vendor); err == nil && info.IsDir() {
			f.useVendor(vendor)
		}
	}
	r := &resolver{f: f, nodes: map[string]*fetched{}, visited: map[string]bool{sema.DirKey(root): true}}
	if err := r.scan(root, man, nil); err != nil {
		return nil, err
	}
	selected, err := r.selectVersions()
	if err != nil {
		return nil, err
	}
	res.r, res.selected, res.sum, res.policy = r, selected, sum, policyFor(root, man)
	for _, e := range r.edges {
		if e.node == nil {
			continue
		}
		sel := selected[identityOf(e.node)]
		if res.Dirs[e.importer] == nil {
			res.Dirs[e.importer] = map[string]string{}
		}
		res.Dirs[e.importer][e.name] = sel.dir
	}
	for _, n := range selected {
		res.Selected = append(res.Selected, n.pin.kind+" "+n.pin.source+" "+strings.TrimPrefix(n.sumKey(), n.pin.kind+" "+n.pin.source+" "))
	}
	sort.Strings(res.Selected)
	res.Warnings = append(append([]string{}, r.warnings...), f.Warnings...)
	if pol := policyFor(root, man); !opts.SkipPolicy && !pol.Empty() {
		report, err := res.Report()
		if err != nil {
			return nil, err
		}
		if vs := Violations(report, pol); len(vs) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "%s: [policy] refuses %d use(s):", filepath.Join(root, "veles.toml"), len(vs))
			for _, v := range vs {
				b.WriteString("\n  " + v.Describe())
			}
			return nil, fmt.Errorf("%s", b.String())
		}
	}
	if err := sum.Save(); err != nil {
		return nil, err
	}
	return res, nil
}

// hasRemote reports whether any package reachable by path from root has a
// registry or git dependency, without fetching anything.
func hasRemote(dir string, man *sema.Manifest, seen map[string]bool) bool {
	for _, d := range man.Requires {
		if d.Kind() != "path" {
			return true
		}
		child := filepath.Join(dir, d.Path)
		if abs, err := filepath.Abs(child); err == nil {
			child = abs
		}
		if seen[sema.DirKey(child)] {
			continue
		}
		seen[sema.DirKey(child)] = true
		if _, err := os.Stat(filepath.Join(child, "veles.toml")); err != nil {
			continue
		}
		cm, err := sema.ReadManifestIn(child)
		if err != nil || cm == nil {
			continue
		}
		if hasRemote(child, cm, seen) {
			return true
		}
	}
	return false
}

func at(dir string, d sema.Dependency) string {
	return fmt.Sprintf("%s:%d: dependency '%s'", filepath.Join(dir, "veles.toml"), d.Line, d.Name)
}

// scan reads the requirements of one package: a path dependency is scanned
// in place, a remote one is fetched and, the first time it is seen, scanned
// too. owner is the fetched package being scanned, nil for the program's own
// packages.
func (r *resolver) scan(dir string, man *sema.Manifest, owner *fetched) error {
	for _, d := range man.Requires {
		if d.Kind() == "path" {
			if owner != nil {
				return fmt.Errorf("%s: a package fetched from %s %s cannot depend on a directory; publish that dependency and require it by version", at(dir, d), owner.pin.kind, owner.pin.source)
			}
			child, err := filepath.Abs(filepath.Join(dir, d.Path))
			if err != nil {
				return err
			}
			r.edges = append(r.edges, edge{importer: sema.DirKey(dir), name: d.Name, pathDir: child})
			if r.visited[sema.DirKey(child)] {
				continue
			}
			r.visited[sema.DirKey(child)] = true
			if _, err := os.Stat(filepath.Join(child, "veles.toml")); err != nil {
				continue // the loader reports a dependency without a manifest where it is imported
			}
			cm, err := sema.ReadManifestIn(child)
			if err != nil {
				return fmt.Errorf("%s: %v", at(dir, d), err)
			}
			if err := r.scan(child, cm, nil); err != nil {
				return err
			}
			continue
		}
		p, err := pinOf(d, owner != nil)
		if err != nil {
			return fmt.Errorf("%s: %v", at(dir, d), err)
		}
		if owner != nil && p.commit != "" {
			r.warnings = append(r.warnings, fmt.Sprintf("package '%s' (%s %s) depends on %s at commit %s: a package others build on should be pinned by version, since version selection cannot run over a commit (D132)", owner.man.Name, owner.pin.kind, owner.pin.source, p.source, p.commit))
		}
		key := p.kind + " " + p.source + " " + p.ref()
		node, seen := r.nodes[key]
		if !seen {
			if node, err = r.f.get(p); err != nil {
				return fmt.Errorf("%s: %v", at(dir, d), err)
			}
			r.nodes[key] = node
		}
		r.edges = append(r.edges, edge{importer: sema.DirKey(dir), name: d.Name, node: node})
		if !seen {
			if err := r.scan(node.dir, node.man, node); err != nil {
				return err
			}
		}
	}
	return nil
}

// identityOf is what makes two versions one package (D132, D138): the
// source and the major. A commit pin has no version of its own, so it is its
// own kind of identity.
func identityOf(n *fetched) string {
	major := "commit"
	if n.pin.commit == "" {
		v, _ := sema.ParseVersion(n.pin.version)
		major = v.MajorID()
	}
	return n.pin.kind + " " + n.pin.source + " @" + major
}

// selectVersions picks, for each identity, the highest version required; two
// different commits of one repository, or a tag and a commit that are one
// (repository, major), are errors that name both.
func (r *resolver) selectVersions() (map[string]*fetched, error) {
	keys := make([]string, 0, len(r.nodes))
	for k := range r.nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys) // errors name the same pair every time
	sel := map[string]*fetched{}
	for _, k := range keys {
		n := r.nodes[k]
		id := identityOf(n)
		cur, have := sel[id]
		switch {
		case !have:
			sel[id] = n
		case n.pin.commit != "":
			if cur.commit != n.commit {
				return nil, fmt.Errorf("%s %s is pinned to two commits, %s and %s; a repository is pinned to one", n.pin.kind, n.pin.source, short(cur.commit), short(n.commit))
			}
		default:
			a, _ := sema.ParseVersion(cur.pin.version)
			b, _ := sema.ParseVersion(n.pin.version)
			if b.Compare(a) > 0 {
				sel[id] = n
			}
		}
	}
	// a commit pin is the same package as a tag when its manifest says it is
	// the same major
	for _, c := range sel {
		if c.pin.commit == "" || c.man == nil || c.man.Version == "" {
			continue
		}
		cv, err := sema.ParseVersion(c.man.Version)
		if err != nil {
			continue
		}
		id := c.pin.kind + " " + c.pin.source + " @" + cv.MajorID()
		if t, clash := sel[id]; clash && t.pin.commit == "" {
			return nil, fmt.Errorf("%s %s is pinned by tag %s and by commit %s, which its manifest calls version %s: they are one package (major %s), so say one of them", c.pin.kind, c.pin.source, t.pin.version, short(c.commit), c.man.Version, cv.MajorID())
		}
	}
	return sel, nil
}

func short(commit string) string {
	if len(commit) > 10 {
		return commit[:10]
	}
	return commit
}
