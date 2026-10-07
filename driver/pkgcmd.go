package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/fetch"
	"github.com/LaH-DeV/veles/sema"
)

// The commands that manage a package's dependencies (D132, D138): add,
// update, remove, deps and vendor. The first three edit veles.toml as text —
// one line, comments and order kept — then resolve the result; if that fails
// the original file is put back, so a bad version or an unreachable
// repository leaves nothing changed.

// PkgOptions are the flags of the dependency commands.
type PkgOptions struct {
	Dir  string   // the package, "." when empty
	Args []string // the specs or names
	As   string   // add: the local name
	Dev  bool     // add: into [dev-dependencies]
	All  bool     // update: every dependency
	Why  string   // deps: explain why a package is in the build
	// Detail, for audit, shows where each capability comes from.
	Detail bool
	// For the registry commands (D139).
	Reason string // yank: why
	Undo   bool   // yank: reverse it
	Claim  string // attest sign: the claim, "reviewed" by default
	Key    string // attest sign: the signing key's name
	Push   bool   // attest sign: also send it to the registry
}

var localNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// pkgRoot finds the package to change. A workspace root that is no package
// itself has no dependencies of its own to edit.
func pkgRoot(dir string, needPackage bool) (string, *sema.Manifest, error) {
	if dir == "" {
		dir = "."
	}
	root, _, err := sema.FindRoot(dir)
	if err != nil {
		return "", nil, err
	}
	if _, err := os.Stat(filepath.Join(root, "veles.toml")); err != nil {
		return "", nil, fmt.Errorf("%s has no veles.toml: run this in a package, or `veles new` makes one", root)
	}
	man, err := sema.ReadManifestIn(root)
	if err != nil {
		return "", nil, err
	}
	if man.Name == "" && needPackage {
		return "", nil, fmt.Errorf("%s is a workspace root with no [package] of its own; name a member with --dir, e.g. --dir %s", root, firstMember(man))
	}
	return root, man, nil
}

func firstMember(man *sema.Manifest) string {
	if man.Workspace != nil && len(man.Workspace.Members) > 0 {
		return man.Workspace.Members[0]
	}
	return "<member>"
}

// depValue is the TOML value of a dependency at a version.
func depValue(d sema.Dependency, version string) string {
	switch d.Kind() {
	case "path":
		return fmt.Sprintf("%q", filepath.ToSlash(d.Path))
	case "registry":
		return fmt.Sprintf("{ registry = %q, version = %q }", d.Registry, version)
	}
	if d.Commit != "" {
		return fmt.Sprintf("{ git = %q, commit = %q }", d.Git, d.Commit)
	}
	return fmt.Sprintf("{ git = %q, version = %q }", d.Git, version)
}

// change writes a new manifest text and resolves it; any failure restores
// the old file. The result is the new resolution.
func change(root string, newText func(old string) (string, error)) (*fetch.Result, error) {
	path := filepath.Join(root, "veles.toml")
	orig, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text, err := newText(string(orig))
	if err != nil {
		return nil, err
	}
	restore := func() { os.WriteFile(path, orig, 0o644) }
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return nil, err
	}
	man, err := sema.ReadManifestIn(root)
	if err != nil {
		restore()
		return nil, err
	}
	res, err := fetch.Resolve(root, man)
	if err != nil {
		restore()
		return nil, fmt.Errorf("%v\nveles.toml was not changed", err)
	}
	return res, nil
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "veles:", err)
	return 1
}

func existing(man *sema.Manifest, name string) (sema.Dependency, string, bool) {
	for _, d := range man.Requires {
		if d.Name == name {
			return d, "dependencies", true
		}
	}
	for _, d := range man.DevRequires {
		if d.Name == name {
			return d, "dev-dependencies", true
		}
	}
	return sema.Dependency{}, "", false
}

// Add is `veles add <spec>[@version] [--as name] [--dev] [--dir d]`.
func Add(o PkgOptions) int {
	if len(o.Args) != 1 {
		return fail(fmt.Errorf("usage: veles add <path | git URL | owner/name>[@version] [--as name] [--dev] [--dir package]"))
	}
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	spec, err := fetch.ParseSpec(o.Args[0])
	if err != nil {
		return fail(err)
	}
	name := o.As
	if name == "" {
		name = spec.Name
		if spec.Kind == "path" {
			target := spec.Source
			if !filepath.IsAbs(target) {
				target = filepath.Join(root, target)
			}
			if pm, err := sema.ReadManifestIn(target); err == nil && pm != nil && pm.Name != "" {
				name = pm.Name
			}
		}
	}
	if !localNameRE.MatchString(name) {
		return fail(fmt.Errorf("%q cannot be used as a dependency name; choose one with --as", name))
	}
	if d, _, ok := existing(man, name); ok {
		return fail(fmt.Errorf("this package already depends on '%s' (%s); `veles update %s` changes its version, or add this one under another name with --as", name, d.Source(), name))
	}
	table := "dependencies"
	if o.Dev {
		table = "dev-dependencies"
	}
	dep := sema.Dependency{Name: name}
	version := spec.Version
	switch spec.Kind {
	case "path":
		dep.Path = spec.Source
	case "git", "registry":
		if spec.Kind == "git" {
			dep.Git = spec.Source
		} else {
			dep.Registry = spec.Source
		}
		if version == "" {
			f := fetch.NewClient(root, man)
			vs, err := f.Versions(spec.Kind, spec.Source)
			if err != nil {
				return fail(err)
			}
			v, ok := f.LatestUsable(spec.Kind, spec.Source, vs, "")
			if !ok {
				return fail(fmt.Errorf("%s has no tagged release (a version like v1.2.0); name one with %s@<version>, or pin an untagged commit by editing veles.toml", spec.Source, o.Args[0]))
			}
			version = v.String()
		}
		dep.Version = version
	}
	res, err := change(root, func(old string) (string, error) {
		text, _, err := fetch.SetDependency(old, table, name, depValue(dep, version))
		return text, err
	})
	if err != nil {
		return fail(err)
	}
	what := dep.Path
	if dep.Kind() != "path" {
		what = dep.Source() + " " + version
	}
	fmt.Printf("added %s = %s to [%s]\n", name, what, table)
	if line := capabilityLine(res, name); line != "" {
		fmt.Println(line)
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	return 0
}

// Update is `veles update <name>[@version]... | --all`: each named dependency
// goes to the newest release of its major — or to the version given — and
// the newest of a later major is mentioned, never taken.
func Update(o PkgOptions) int {
	if (len(o.Args) == 0) == !o.All {
		return fail(fmt.Errorf("usage: veles update <name>[@version]... | --all [--dir package]"))
	}
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	type target struct {
		dep     sema.Dependency
		table   string
		version string // "" for the newest of the major
	}
	var targets []target
	if o.All {
		for _, d := range man.Requires {
			targets = append(targets, target{d, "dependencies", ""})
		}
		for _, d := range man.DevRequires {
			targets = append(targets, target{d, "dev-dependencies", ""})
		}
	}
	for _, arg := range o.Args {
		name, version := arg, ""
		if i := strings.LastIndex(arg, "@"); i > 0 {
			name, version = arg[:i], arg[i+1:]
			v, err := sema.ParseVersion(version)
			if err != nil {
				return fail(err)
			}
			version = v.String()
		}
		d, table, ok := existing(man, name)
		if !ok {
			return fail(fmt.Errorf("this package has no dependency '%s' (it has: %s)", name, strings.Join(dependencyNames(man), ", ")))
		}
		targets = append(targets, target{d, table, version})
	}
	f := fetch.NewClient(root, man)
	var notes, lines []string
	type edit struct{ table, name, value string }
	var edits []edit
	for _, t := range targets {
		d := t.dep
		if d.Kind() == "path" || d.Commit != "" {
			if !o.All {
				return fail(fmt.Errorf("'%s' is %s, which has no version to update", d.Name, map[bool]string{true: "a path", false: "a commit pin"}[d.Kind() == "path"]))
			}
			continue
		}
		cur, _ := sema.ParseVersion(d.Version)
		next := t.version
		vs, err := f.Versions(d.Kind(), d.Source())
		if err != nil {
			return fail(err)
		}
		if next == "" {
			latest, ok := f.LatestUsable(d.Kind(), d.Source(), vs, cur.MajorID())
			if !ok || latest.Compare(cur) <= 0 {
				next = d.Version
			} else {
				next = latest.String()
			}
			if newest, ok := f.LatestUsable(d.Kind(), d.Source(), vs, ""); ok && newest.MajorID() != cur.MajorID() && newest.Compare(cur) > 0 {
				notes = append(notes, fmt.Sprintf("%s: a newer major, %s, is available; `veles update %s@%s` takes it (read its changes first)", d.Name, newest, d.Name, newest))
			}
		}
		if next == d.Version {
			lines = append(lines, fmt.Sprintf("%s %s is up to date", d.Name, d.Version))
			continue
		}
		edits = append(edits, edit{t.table, d.Name, depValue(d, next)})
		lines = append(lines, fmt.Sprintf("%s %s -> %s", d.Name, d.Version, next))
	}
	if len(edits) > 0 {
		res, err := change(root, func(old string) (string, error) {
			text := old
			for _, e := range edits {
				var err error
				if text, _, err = fetch.SetDependency(text, e.table, e.name, e.value); err != nil {
					return "", err
				}
			}
			return text, nil
		})
		if err != nil {
			return fail(err)
		}
		for _, w := range res.Warnings {
			fmt.Fprintln(os.Stderr, "warning:", w)
		}
		printLines(lines)
		lines = nil
		if n, err := res.PruneSum(); err != nil {
			return fail(err)
		} else if n > 0 {
			fmt.Printf("pruned %d unused lines from veles.sum\n", n)
		}
	}
	printLines(lines)
	printLines(notes)
	return 0
}

func printLines(lines []string) {
	for _, l := range lines {
		fmt.Println(l)
	}
}

func dependencyNames(man *sema.Manifest) []string {
	var names []string
	for _, d := range man.Requires {
		names = append(names, d.Name)
	}
	for _, d := range man.DevRequires {
		names = append(names, d.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []string{"none"}
	}
	return names
}

// Remove is `veles remove <name>...`.
func Remove(o PkgOptions) int {
	if len(o.Args) == 0 {
		return fail(fmt.Errorf("usage: veles remove <name>... [--dir package]"))
	}
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	for _, name := range o.Args {
		if _, _, ok := existing(man, name); !ok {
			return fail(fmt.Errorf("this package has no dependency '%s' (it has: %s)", name, strings.Join(dependencyNames(man), ", ")))
		}
	}
	res, err := change(root, func(old string) (string, error) {
		text := old
		for _, name := range o.Args {
			_, table, _ := existing(man, name)
			var err error
			if text, _, err = fetch.RemoveDependency(text, table, name); err != nil {
				return "", err
			}
		}
		return text, nil
	})
	if err != nil {
		return fail(err)
	}
	fmt.Printf("removed %s\n", strings.Join(o.Args, ", "))
	pruned, err := res.PruneSum()
	if err == nil && res.Selected == nil {
		pruned, err = fetch.PruneAllSum(root)
	}
	if err != nil {
		return fail(err)
	}
	if pruned > 0 {
		fmt.Printf("pruned %d unused lines from veles.sum\n", pruned)
	}
	return 0
}

// Deps is `veles deps [dir] [--why name]`: the resolved build list as a tree.
func Deps(o PkgOptions) int {
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	res, err := fetch.Resolve(root, man)
	if err != nil {
		return fail(err)
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if o.Why != "" {
		chains, ok := res.Why(o.Why)
		if !ok {
			return fail(fmt.Errorf("nothing called '%s' is in the build of %s (try `veles deps`)", o.Why, man.Name))
		}
		fmt.Println(chains)
		return 0
	}
	fmt.Print(res.Tree())
	return 0
}

// Vendor is `veles vendor [dir]`.
func Vendor(o PkgOptions) int {
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	n, err := fetch.Vendor(root, man)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("vendored %d packages into vendor/; builds read them from there now (delete vendor/ to use the module cache again)\n", n)
	return 0
}

// Audit is `veles audit [dir] [--detail]`: what each package in the build can
// do (D138), computed from its source, and whether the project's [policy]
// allows it. The exit status is 1 when it does not.
func Audit(o PkgOptions) int {
	root, man, err := pkgRoot(o.Dir, true)
	if err != nil {
		return fail(err)
	}
	res, err := fetch.ResolveWith(root, man, fetch.ResolveOptions{SkipPolicy: true, Refresh: true})
	if err != nil {
		return fail(err)
	}
	report, err := res.Report()
	if err != nil {
		return fail(err)
	}
	fmt.Print(fetch.FormatReport(report, o.Detail))
	policy := fetch.Policy(root, man)
	if policy.Empty() {
		fmt.Println("\nno [policy] in veles.toml: nothing is denied (deny = [\"net\", \"unsafe\", …] makes a build refuse a dependency that uses them)")
		return 0
	}
	vs := fetch.Violations(report, policy)
	if len(vs) == 0 {
		fmt.Printf("\n[policy] denies %s: no dependency breaks it\n", strings.Join(policy.Deny, ", "))
		return 0
	}
	fmt.Printf("\n[policy] denies %s, and %d use(s) break it:\n", strings.Join(policy.Deny, ", "), len(vs))
	for _, v := range vs {
		fmt.Println("  " + v.Describe())
	}
	return 1
}

// capabilityLine is what `veles add` says about the dependency it added.
func capabilityLine(res *fetch.Result, name string) string {
	report, err := res.Report()
	if err != nil {
		return ""
	}
	for _, p := range report {
		if p.Local == name {
			caps := fetch.CapNames(p.Total)
			tier := "; tier: " + p.Tier
			if len(caps) == 0 {
				return fmt.Sprintf("%s can use no unsafe code, C, network, files or processes (with its dependencies)%s", name, tier)
			}
			return fmt.Sprintf("%s can use: %s (with its dependencies)%s; `veles audit --detail` shows where", name, strings.Join(caps, ", "), tier)
		}
	}
	return ""
}
