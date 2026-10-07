package fetch

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/sema"
)

// PackageReport is one package of the build and what its source can do.
type PackageReport struct {
	Name    string // the package's own name
	Local   string // the root package's name for it; "" when only a dependency needs it
	Kind    string // "git", "registry" or "path"
	Source  string // the repository URL, owner/name, or the directory as written
	Version string // "" for a path or a commit pin
	Dir     string
	Own     Caps // what its own source does
	Total   Caps // its own and its dependencies'
	Tier    string
	Ref     string // the version or commit:<full> as veles.sum has it
	Hash    string // the h1 hash of its files
	// Reviews maps each claim to the trusted reviewers who attested it about
	// exactly these bytes; only filled when the policy requires reviews.
	Reviews map[string][]string
}

// Remote reports whether the package was fetched (a path dependency is the
// project's own code).
func (p PackageReport) Remote() bool { return p.Kind != "path" }

// Report analyses every package in the build: the selected registry and git
// packages, and the path dependencies. Sorted by kind then source.
func (res *Result) Report() ([]PackageReport, error) {
	if res.r == nil {
		return nil, nil
	}
	byDir := map[string]*PackageReport{}
	add := func(p PackageReport, man *sema.Manifest) error {
		key := sema.DirKey(p.Dir)
		if _, dup := byDir[key]; dup {
			return nil
		}
		own, err := PackageCaps(p.Dir, man)
		if err != nil {
			return fmt.Errorf("%s: %v", p.Source, err)
		}
		p.Own = own
		if man != nil {
			p.Name = man.Name
		}
		byDir[key] = &p
		return nil
	}
	for _, n := range res.selected {
		version := n.pin.version
		if n.pin.commit != "" {
			version = "commit " + short(n.commit)
		}
		ref := strings.TrimPrefix(n.sumKey(), n.pin.kind+" "+n.pin.source+" ")
		p := PackageReport{Kind: n.pin.kind, Source: n.pin.source, Version: version, Dir: n.dir, Tier: n.tier, Ref: ref, Hash: n.hash}
		if err := add(p, n.man); err != nil {
			return nil, err
		}
		if n.tier != "listed" {
			why := "a git repository is never in the listed tier"
			if n.pin.kind == "registry" {
				why = "the registry has not listed this version"
			}
			byDir[sema.DirKey(n.dir)].Own.add("unlisted", "tier: unreviewed ("+why+")")
		}
		if len(res.policy.Require) > 0 {
			rep := byDir[sema.DirKey(n.dir)]
			stmts, err := res.statements(n)
			if err != nil {
				return nil, err
			}
			rep.Reviews = TrustedReviewers(stmts, res.policy.Trust, n.pin.kind, n.pin.source, ref, n.hash)
		}
	}
	rootKey := sema.DirKey(res.root)
	for _, e := range res.r.edges {
		if e.node != nil {
			continue
		}
		rel, err := filepath.Rel(res.root, e.pathDir)
		if err != nil {
			rel = e.pathDir
		}
		var man *sema.Manifest
		if m, err := sema.ReadManifestIn(e.pathDir); err == nil {
			man = m
		}
		if err := add(PackageReport{Kind: "path", Source: filepath.ToSlash(rel), Dir: e.pathDir}, man); err != nil {
			return nil, err
		}
	}
	// local names, as the root writes them
	for _, e := range res.r.edges {
		if e.importer != rootKey {
			continue
		}
		dir := e.pathDir
		if e.node != nil {
			dir = res.selected[identityOf(e.node)].dir
		}
		if p := byDir[sema.DirKey(dir)]; p != nil && p.Local == "" {
			p.Local = e.name
		}
	}
	// totals: a package's own capabilities and those of everything below it
	children := map[string][]string{}
	for _, e := range res.r.edges {
		dir := e.pathDir
		if e.node != nil {
			dir = res.selected[identityOf(e.node)].dir
		}
		children[e.importer] = append(children[e.importer], sema.DirKey(dir))
	}
	var total func(key string, active map[string]bool) Caps
	done := map[string]Caps{}
	total = func(key string, active map[string]bool) Caps {
		if c, ok := done[key]; ok {
			return c
		}
		out := Caps{}
		if p := byDir[key]; p != nil {
			out.merge(p.Own)
		}
		if active[key] {
			return out
		}
		active[key] = true
		for _, child := range children[key] {
			out.merge(total(child, active))
		}
		delete(active, key)
		done[key] = out
		return out
	}
	var out []PackageReport
	for key, p := range byDir {
		p.Total = total(key, map[string]bool{})
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Source < out[j].Source
	})
	return out, nil
}

// Violation is a package that breaks the policy: it uses a capability the
// policy denies, or it lacks reviews the policy requires.
type Violation struct {
	Package    PackageReport
	Capability string // a capability, or the claim of a missing review
	Review     bool
	Have, Need int // for a missing review
}

// Violations lists, for the registry and git packages of the build, each
// capability of its own source that the policy denies and does not allow, and
// each required review a package lacks.
func Violations(report []PackageReport, policy sema.ManifestPolicy) []Violation {
	var out []Violation
	for _, p := range report {
		if !p.Remote() {
			continue
		}
		for _, c := range p.Own.Names() {
			if policy.Denied(c, p.Source, p.Name) {
				out = append(out, Violation{Package: p, Capability: c})
			}
		}
		claims := make([]string, 0, len(policy.Require))
		for c := range policy.Require {
			claims = append(claims, c)
		}
		sort.Strings(claims)
		for _, c := range claims {
			if have := len(p.Reviews[c]); policy.ReviewMissing(c, have, p.Source, p.Name) {
				out = append(out, Violation{Package: p, Capability: c, Review: true, Have: have, Need: policy.Require[c]})
			}
		}
	}
	return out
}

// Describe is the violation as an error message: what, where, and how to
// allow it.
func (v Violation) Describe() string {
	p := v.Package
	if v.Review {
		return fmt.Sprintf("%s %s has %d of the %d trusted %q review(s) [policy] requires; have someone you trust read it and run `veles attest sign`, or exempt it with `allow = { %s = [%q] }`", p.Source, p.Version, v.Have, v.Need, v.Capability, v.Capability, p.Source)
	}
	where := ""
	if ev := p.Own[v.Capability]; len(ev) > 0 {
		where = " (" + ev[0] + ")"
	}
	verb := "uses"
	if v.Capability == "unlisted" {
		verb = "is"
	}
	return fmt.Sprintf("%s %s %s %s%s, which [policy] denies; allow it with `allow = { %s = [%q] }` or drop %s from `deny`", p.Source, p.Version, verb, v.Capability, where, v.Capability, p.Source, v.Capability)
}

// workspaceRoot is the manifest of the workspace a package is a member of,
// or nil.
func workspaceRoot(root string) *sema.Manifest {
	if dir := filepath.Dir(sumPathFor(root)); dir != root {
		if m, err := sema.ReadManifestIn(dir); err == nil {
			return m
		}
	}
	return nil
}

// policyFor is the policy that governs a package: its own [policy], else the
// one in the manifest of the workspace it belongs to.
func policyFor(root string, man *sema.Manifest) sema.ManifestPolicy {
	if !man.Policy.Empty() {
		return man.Policy
	}
	if ws := workspaceRoot(root); ws != nil {
		return ws.Policy
	}
	return man.Policy
}

// registryFor is the registry settings of a package, or of its workspace.
func registryFor(root string, man *sema.Manifest) sema.ManifestRegistry {
	if man.Registry != (sema.ManifestRegistry{}) {
		return man.Registry
	}
	if ws := workspaceRoot(root); ws != nil {
		return ws.Registry
	}
	return man.Registry
}

// statements gathers the signed reviews of a package version: those the
// project holds, and for a registry package those the registry serves. The
// registry being unreachable is not an error — the local ones still count.
func (res *Result) statements(n *fetched) ([]Statement, error) {
	local, err := LoadAttestations(res.root)
	if err != nil {
		return nil, err
	}
	if n.pin.kind == "registry" && res.r != nil && res.r.f != nil {
		if remote, err := res.r.f.registryAttestations(n.pin.source, n.pin.version); err == nil {
			local = append(local, remote...)
		}
	}
	return local, nil
}

// CapNames are a package's capabilities without the tier, which has a column
// of its own.
func CapNames(c Caps) []string {
	var out []string
	for _, n := range c.Names() {
		if n != "unlisted" {
			out = append(out, n)
		}
	}
	return out
}

// FormatReport is the table `veles audit` prints.
func FormatReport(report []PackageReport, detail bool) string {
	if len(report) == 0 {
		return "no dependencies\n"
	}
	var b strings.Builder
	width := 0
	label := func(p PackageReport) string {
		s := p.Source
		if p.Version != "" {
			s += " " + p.Version
		}
		return s
	}
	for _, p := range report {
		if n := len(label(p)); n > width {
			width = n
		}
	}
	for _, p := range report {
		caps := strings.Join(CapNames(p.Own), " ")
		if caps == "" {
			caps = "-"
		}
		via := ""
		if extra := missing(CapNames(p.Total), CapNames(p.Own)); len(extra) > 0 {
			via = "   (dependencies add: " + strings.Join(extra, " ") + ")"
		}
		tier := p.Tier
		if p.Kind == "path" {
			tier = "own"
		}
		fmt.Fprintf(&b, "%-*s  %-8s %-10s %s%s\n", width, label(p), p.Kind, tier, caps, via)
		if detail {
			for _, c := range CapNames(p.Own) {
				for _, ev := range p.Own[c] {
					fmt.Fprintf(&b, "    %-6s %s\n", c, ev)
				}
			}
			claims := make([]string, 0, len(p.Reviews))
			for c := range p.Reviews {
				claims = append(claims, c)
			}
			sort.Strings(claims)
			for _, c := range claims {
				fmt.Fprintf(&b, "    %-6s by %s\n", c, strings.Join(p.Reviews[c], ", "))
			}
		}
	}
	return b.String()
}

func missing(all, have []string) []string {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	var out []string
	for _, a := range all {
		if !set[a] {
			out = append(out, a)
		}
	}
	return out
}

// Policy is the policy that governs the package at root.
func Policy(root string, man *sema.Manifest) sema.ManifestPolicy { return policyFor(root, man) }
