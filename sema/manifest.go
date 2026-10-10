package sema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Manifest is `veles.toml` (M1, D138): package identity lives here, never in
// source. The file is TOML (the subset toml.go reads):
//
//	[package]
//	name = "app"
//	version = "0.1.0"
//	veles = "0.70.0"                       # the oldest compiler that builds it
//
//	[dependencies]
//	mathlib = "../mathlib"                 # short for { path = "../mathlib" }
//	httputil = { registry = "acme/httputil", version = "1.4.2" }
//	pg = { git = "https://github.com/veles-db/pg", version = "2.1.0" }
//	fast = { git = "https://example.com/fast", commit = "3f2a9c1" }
//
//	[dev-dependencies]                     # only for `veles test` and *.test.vs
//	fakeclock = { registry = "lah/fakeclock", version = "0.3.0" }
//
//	[workspace]
//	members = ["app", "libs/mathlib"]      # a monorepo: build and test them all
//
// The key of a dependency is the name code writes in `use`. Only path
// dependencies load today; registry and git dependencies are read, checked
// and refused by the loader until fetching is built (plan E7 stage d).
type Manifest struct {
	Name        string
	Version     string
	Description string
	License     string
	// Veles is the minimum compiler version ("" when not stated).
	Veles string
	// Deps is the path dependencies, name -> path (relative to the manifest):
	// the part of Requires the loader resolves today.
	Deps map[string]string
	// Requires is every `[dependencies]` entry and DevRequires every
	// `[dev-dependencies]` entry, sorted by name.
	Requires    []Dependency
	DevRequires []Dependency
	// Workspace is the `[workspace]` table; nil in a package that is not a
	// monorepo root.
	Workspace *ManifestWorkspace
	Dir       string
	// Format holds the `[format]` table for `veles fmt`: Indent is "" for
	// the default, a run of spaces, or a tab; MaxBlankLines is 0 for the
	// default.
	Format ManifestFormat
	// Lint is the `[lint]` table: the opt-in lints of the package's code.
	Lint ManifestLint
	// Native is the `[native]` table (D67): the C libraries the package's
	// `extern` blocks need at link time.
	Native ManifestNative
	// Policy is the `[policy]` table (D138): which capabilities the packages
	// this one depends on may use.
	Policy ManifestPolicy
	// Registry is the `[registry]` table (D139): the registry's address and
	// the key its metadata is signed with.
	Registry ManifestRegistry
	// RegistryName is `[package] registry = "owner/name"`: where `veles
	// publish` publishes this package.
	RegistryName string
	// Runtime is the `[runtime]` table (D143): how the program runs. Only
	// the program's own manifest counts; a dependency's is not read.
	Runtime ManifestRuntime
}

// ManifestRuntime is the `[runtime]` table (D143).
type ManifestRuntime struct {
	// Threads is how many threads the default pool has: 0 for "auto" (one
	// per core, the default); VELES_THREADS overrides it.
	Threads int64
}

// Dependency is one entry of `[dependencies]`. Exactly one of Path,
// Registry and Git is set; a Git dependency has a Version or a Commit, a
// Registry one a Version.
type Dependency struct {
	Name     string // the local name: what `use` says
	Path     string // relative to the manifest
	Registry string // "owner/name"
	Git      string // a repository URL
	Version  string // an exact version, as written ("1.4.2")
	Commit   string // a commit pin, a hex prefix
	Line     int
}

// Kind is "path", "registry" or "git".
func (d Dependency) Kind() string {
	switch {
	case d.Path != "":
		return "path"
	case d.Registry != "":
		return "registry"
	}
	return "git"
}

// Source is the dependency's origin as the manifest spells it, for messages.
func (d Dependency) Source() string {
	switch d.Kind() {
	case "path":
		return d.Path
	case "registry":
		return d.Registry
	}
	return d.Git
}

// ManifestWorkspace is `[workspace]` (D138): the member directories of a
// monorepo, relative to the manifest that lists them.
type ManifestWorkspace struct {
	Members []string
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
	return parseManifest(path, dir, string(data))
}

// manifestSections are the tables a manifest has; each table checks its own
// keys, so a typo is an error rather than a setting that silently does
// nothing.
var manifestSections = []string{"package", "dependencies", "dev-dependencies", "workspace", "format", "lint", "native", "policy", "registry", "runtime"}

func parseManifest(path, dir, text string) (*Manifest, error) {
	root, err := parseTOML(text)
	if err != nil {
		te := err.(*tomlError)
		return nil, fmt.Errorf("%s:%d: %s", path, te.line, te.msg)
	}
	r := &manifestReader{path: path}
	m := &Manifest{Deps: map[string]string{}, Dir: dir}
	for _, name := range root.keys() {
		v := root.vals[name]
		if v.kind != tomlTab {
			return nil, r.fail(v, "'%s' is outside any table: put it under [package] (the tables are %s)", name, strings.Join(manifestSections, ", "))
		}
		var err error
		switch name {
		case "package":
			err = r.pkg(m, v.tab)
		case "dependencies":
			m.Requires, err = r.deps(v.tab)
		case "dev-dependencies":
			m.DevRequires, err = r.deps(v.tab)
		case "workspace":
			m.Workspace, err = r.workspace(v.tab)
		case "format":
			err = r.format(m, v.tab)
		case "lint":
			err = r.lint(m, v.tab)
		case "native":
			err = r.native(m, v.tab)
		case "policy":
			err = r.policy(m, v.tab)
		case "registry":
			err = r.registry(m, v.tab)
		case "runtime":
			err = r.runtime(m, v.tab)
		default:
			err = r.fail(v, "unknown table [%s] (%s)", name, strings.Join(manifestSections, ", "))
		}
		if err != nil {
			return nil, err
		}
	}
	for _, d := range m.Requires {
		if d.Path != "" {
			m.Deps[d.Name] = d.Path
		}
	}
	if m.Name == "" && m.Workspace == nil {
		return nil, fmt.Errorf("%s: [package] name is required", path)
	}
	return m, nil
}

type manifestReader struct{ path string }

func (r *manifestReader) fail(v *tomlVal, format string, a ...any) error {
	return fmt.Errorf("%s:%d: %s", r.path, v.line, fmt.Sprintf(format, a...))
}

func (r *manifestReader) strList(v *tomlVal, what string, example string) ([]string, error) {
	if v.kind != tomlList {
		return nil, r.fail(v, "%s is a list, e.g. %s", what, example)
	}
	var out []string
	for _, item := range v.list {
		if item.kind != tomlStr {
			return nil, r.fail(item, "%s holds strings only", what)
		}
		out = append(out, item.str)
	}
	return out, nil
}

func (r *manifestReader) pkg(m *Manifest, t *tomlTable) error {
	for _, key := range t.keys() {
		v := t.vals[key]
		switch key {
		case "name", "version", "description", "license", "veles", "registry":
			if v.kind != tomlStr {
				return r.fail(v, "[package] %s is a string, e.g. %s = \"...\"", key, key)
			}
			s := v.str
			switch key {
			case "name":
				m.Name = s
			case "version":
				if _, err := ParseVersion(s); err != nil {
					return r.fail(v, "[package] version: %v", err)
				}
				m.Version = s
			case "description":
				m.Description = s
			case "license":
				m.License = s
			case "veles":
				if err := checkMinVersion(s); err != nil {
					return r.fail(v, "[package] veles: %v", err)
				}
				m.Veles = s
			case "registry":
				if !validRegistryName(s) {
					return r.fail(v, "[package] registry %q is owner/name, lowercase letters, digits, '-' and '_'", s)
				}
				m.RegistryName = s
			}
		case "exports":
			return r.fail(v, "'exports' was removed (D89): a package's surface is what its root module re-exports — write 'public use geometry' in lib.vs for each module listed here")
		default:
			return r.fail(v, "unknown [package] key %q (name, version, description, license, veles, registry)", key)
		}
	}
	return nil
}

// checkMinVersion accepts `0.70` and `0.70.0`: the oldest compiler.
func checkMinVersion(s string) error {
	if strings.Count(s, ".") == 1 {
		s += ".0"
	}
	_, err := ParseVersion(s)
	return err
}

func (r *manifestReader) deps(t *tomlTable) ([]Dependency, error) {
	var out []Dependency
	for _, name := range t.keys() {
		v := t.vals[name]
		if !validLocalName(name) {
			return nil, r.fail(v, "dependency name %q: the name code writes in `use`, so letters, digits and '_' and not starting with a digit", name)
		}
		d := Dependency{Name: name, Line: v.line}
		switch v.kind {
		case tomlStr:
			// `name = "../dir"` is the short form of `name = { path = "../dir" }`.
			if looksLikeVersion(v.str) {
				return nil, r.fail(v, "dependency '%s' = %q looks like a version, but a bare string is a path: write { registry = \"owner/name\", version = %q }", name, v.str, v.str)
			}
			d.Path = v.str
		case tomlTab:
			for _, key := range v.tab.keys() {
				item := v.tab.vals[key]
				s, ok := "", item.kind == tomlStr
				if ok {
					s = item.str
				}
				switch key {
				case "path", "registry", "git", "version", "commit":
					if !ok {
						return nil, r.fail(item, "dependency '%s': %s is a string", name, key)
					}
				default:
					return nil, r.fail(item, "dependency '%s': unknown key %q (path, registry, git, version, commit)", name, key)
				}
				switch key {
				case "path":
					d.Path = s
				case "registry":
					d.Registry = s
				case "git":
					d.Git = s
				case "version":
					d.Version = s
				case "commit":
					d.Commit = s
				}
			}
		default:
			return nil, r.fail(v, "dependency '%s' is a path string or a table such as { registry = \"owner/name\", version = \"1.0.0\" }", name)
		}
		if err := checkDependency(d); err != nil {
			return nil, r.fail(v, "dependency '%s': %v", name, err)
		}
		out = append(out, d)
	}
	return out, nil
}

func checkDependency(d Dependency) error {
	sources := 0
	for _, s := range []string{d.Path, d.Registry, d.Git} {
		if s != "" {
			sources++
		}
	}
	if sources != 1 {
		return detail("say where it comes from with exactly one of path, registry or git")
	}
	if d.Version != "" {
		if _, err := ParseVersion(d.Version); err != nil {
			return err
		}
		if strings.HasPrefix(d.Version, "v") {
			return detail("write the version without the 'v' (%q)", strings.TrimPrefix(d.Version, "v"))
		}
	}
	if d.Commit != "" && !validCommit(d.Commit) {
		return detail("commit %q is a hex prefix of 7 to 40 digits", d.Commit)
	}
	switch d.Kind() {
	case "path":
		if d.Version != "" || d.Commit != "" {
			return detail("a path dependency has no version or commit: it is the directory as it is")
		}
	case "registry":
		if !validRegistryName(d.Registry) {
			return detail("registry name %q is owner/name, lowercase letters, digits, '-' and '_'", d.Registry)
		}
		if d.Commit != "" {
			return detail("a registry dependency is pinned by version; a commit pin is for git")
		}
		if d.Version == "" {
			return detail("a registry dependency needs a version, e.g. version = \"1.0.0\"")
		}
	case "git":
		if (d.Version == "") == (d.Commit == "") {
			return detail("a git dependency is pinned by exactly one of version (a tag) or commit")
		}
	}
	return nil
}

func validLocalName(s string) bool {
	if s == "" || s[0] >= '0' && s[0] <= '9' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

func validRegistryName(s string) bool {
	owner, name, ok := strings.Cut(s, "/")
	if !ok {
		return false
	}
	for _, part := range []string{owner, name} {
		if part == "" {
			return false
		}
		for i := 0; i < len(part); i++ {
			c := part[i]
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
				return false
			}
		}
	}
	return true
}

func validCommit(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// looksLikeVersion: a bare "1.4.2" in [dependencies] is a mistake worth a
// pointed message, since a path never looks like that.
func looksLikeVersion(s string) bool {
	_, err := ParseVersion(s)
	return err == nil
}

func (r *manifestReader) workspace(t *tomlTable) (*ManifestWorkspace, error) {
	w := &ManifestWorkspace{}
	for _, key := range t.keys() {
		v := t.vals[key]
		if key != "members" {
			return nil, r.fail(v, "unknown [workspace] key %q (members)", key)
		}
		list, err := r.strList(v, "[workspace] members", `members = ["app", "libs/mathlib"]`)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, member := range list {
			clean := filepath.ToSlash(filepath.Clean(member))
			switch {
			case member == "" || clean == ".":
				return nil, r.fail(v, "[workspace] members: a member is a directory below the root, not %q", member)
			case filepath.IsAbs(member) || clean == ".." || strings.HasPrefix(clean, "../"):
				return nil, r.fail(v, "[workspace] members: %q leaves the workspace; a member is a directory below the root manifest", member)
			case strings.ContainsAny(member, "*?["):
				return nil, r.fail(v, "[workspace] members: patterns such as %q are not supported yet; list each member directory", member)
			case seen[clean]:
				return nil, r.fail(v, "[workspace] members: %q is listed twice", member)
			}
			seen[clean] = true
			w.Members = append(w.Members, clean)
		}
	}
	if w.Members == nil {
		return nil, r.fail(&tomlVal{line: t.line}, "[workspace] needs members = [\"dir\", ...]")
	}
	return w, nil
}

// runtime reads `[runtime]` (D143): threads = "auto" or a count.
func (r *manifestReader) runtime(m *Manifest, t *tomlTable) error {
	for _, key := range t.keys() {
		v := t.vals[key]
		switch key {
		case "threads":
			switch {
			case v.kind == tomlStr && v.str == "auto":
				m.Runtime.Threads = 0
			case v.kind == tomlInt && v.num >= 1 && v.num <= 256:
				m.Runtime.Threads = v.num
			default:
				return r.fail(v, "[runtime] threads is \"auto\" (one per core, the default) or a number from 1 to 256, e.g. threads = 4")
			}
		default:
			return r.fail(v, "unknown [runtime] key %q (threads)", key)
		}
	}
	return nil
}

func (r *manifestReader) format(m *Manifest, t *tomlTable) error {
	for _, key := range t.keys() {
		v := t.vals[key]
		switch key {
		case "indent":
			text := v.str
			if v.kind == tomlInt {
				text = fmt.Sprint(v.num)
			} else if v.kind != tomlStr {
				return r.fail(v, "[format] indent must be a number of spaces (1-8) or \"tab\"")
			}
			if text == "tab" {
				m.Format.Indent = "\t"
				continue
			}
			width := 0
			if _, err := fmt.Sscanf(text, "%d", &width); err != nil || width < 1 || width > 8 || fmt.Sprint(width) != text {
				return r.fail(v, "[format] indent must be a number of spaces (1-8) or \"tab\"")
			}
			m.Format.Indent = strings.Repeat(" ", width)
		case "max_blank_lines":
			limit := int64(0)
			switch v.kind {
			case tomlInt:
				limit = v.num
			case tomlStr:
				fmt.Sscanf(v.str, "%d", &limit)
			}
			if limit < 1 {
				return r.fail(v, "[format] max_blank_lines must be a positive number")
			}
			m.Format.MaxBlankLines = int(limit)
		case "imports":
			if v.kind != tomlStr || (v.str != "lines" && v.str != "merged") {
				return r.fail(v, "[format] imports is \"lines\" (one 'use' per module, the default) or \"merged\" (one 'use a, b' per origin)")
			}
			m.Format.MergeImports = v.str == "merged"
		default:
			return r.fail(v, "unknown [format] key %q (indent, max_blank_lines, imports)", key)
		}
	}
	return nil
}

// lintKeys are the settings of a `[lint]` table.
var lintKeys = []string{"implicit_return"}

func (r *manifestReader) lint(m *Manifest, t *tomlTable) error {
	for _, key := range t.keys() {
		v := t.vals[key]
		switch key {
		case "implicit_return":
			level, ok := implicitReturnLevels[v.str]
			if v.kind != tomlStr || !ok {
				return r.fail(v, "[lint] implicit_return is \"full\" (a body's last expression is its value anywhere, the default), \"lambda\" (in a lambda and an '=> expr' body) or \"expr\" (only in a body without braces)")
			}
			m.Lint.ImplicitReturn = level
		default:
			return r.fail(v, "unknown [lint] key %q (%s)", key, strings.Join(lintKeys, ", "))
		}
	}
	return nil
}

func (r *manifestReader) native(m *Manifest, t *tomlTable) error {
	for _, key := range t.keys() {
		v := t.vals[key]
		if v.kind != tomlList {
			if key == "libs" || key == "static-libs" || key == "lib-paths" || key == "pkg-config" {
				return r.fail(v, "[native] %s is a list, e.g. %s = [\"z\"]", key, key)
			}
		}
		var dst *[]string
		switch key {
		case "libs":
			dst = &m.Native.Libs
		case "static-libs":
			dst = &m.Native.StaticLibs
		case "lib-paths":
			dst = &m.Native.LibPaths
		case "pkg-config":
			dst = &m.Native.PkgConfig
		default:
			return r.fail(v, "unknown [native] key %q (libs, static-libs, lib-paths, pkg-config)", key)
		}
		list, err := r.strList(v, "[native] "+key, key+` = ["z"]`)
		if err != nil {
			return err
		}
		if key == "pkg-config" {
			// a name is passed to pkg-config as an argument: one that starts with '-' would be an option
			for _, name := range list {
				if strings.HasPrefix(name, "-") {
					return r.fail(v, "[native] pkg-config: %q is a package name, not an option", name)
				}
			}
		}
		*dst = list
	}
	return nil
}

// ManifestFormat is the `[format]` table.
type ManifestFormat struct {
	Indent        string
	MaxBlankLines int
	// MergeImports is `imports = "merged"`: `use a, b` rather than a line each.
	MergeImports bool
}

// ManifestLint is the `[lint]` table: the lints a package opts into. Each is
// an error in that package's own code, with a fix; dependencies and the
// standard library are checked by their own manifests.
type ManifestLint struct {
	// ImplicitReturn is where a body's last expression may be its value
	// without `return`.
	ImplicitReturn ImplicitReturn
}

// ImplicitReturn is the `[lint] implicit_return` setting, from the most
// permissive to the strictest.
type ImplicitReturn int

const (
	// ImplicitFull, "full" (the default): anywhere — a function's, a
	// method's or a lambda's `{ }` body ends in its value.
	ImplicitFull ImplicitReturn = iota
	// ImplicitLambda, "lambda": in what is written with an arrow — every
	// lambda, braced or not, and `fun f() => expr` — but a function's or
	// method's `{ }` body says `return`.
	ImplicitLambda
	// ImplicitExpr, "expr": only in a body without braces (`fun f() =
	// expr`, `x => expr`); every `{ }` body says `return`.
	ImplicitExpr
)

var implicitReturnLevels = map[string]ImplicitReturn{"full": ImplicitFull, "lambda": ImplicitLambda, "expr": ImplicitExpr}

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

// Capabilities are what a package's source can do that a build should know
// about (D138). They are computed from the source, never declared:
//
//	unsafe   an `unsafe` block or function
//	extern   an `extern` block, or a function C can call
//	native   a [native] table: it links C libraries
//	net      it imports net, tls, http, db or otel
//	fs       it imports fs or config
//	os       it imports os (processes, environment) or config
//	ffi      it imports ffi
//	unlisted it is not in the registry's listed tier (D139): a git package,
//	         a commit pin, or a registry version the registry has not listed
//
// The last is not something the source does; it is here so that one policy
// vocabulary says "deny unlisted" the way it says "deny net".
var Capabilities = []string{"unsafe", "extern", "native", "net", "fs", "os", "ffi", "unlisted"}

func validCapability(s string) bool {
	for _, c := range Capabilities {
		if c == s {
			return true
		}
	}
	return false
}

// ManifestPolicy is the `[policy]` table:
//
//	[policy]
//	deny = ["net", "unsafe", "unlisted"]           # no dependency may use these…
//	allow = { net = ["acme/httputil"] }            # …except the packages named here
//	trust = { alice = "ed25519:…" }                # whose reviews count (D139)
//	require = { reviewed = 1 }                     # how many trusted reviews a package needs
//
// A package is named by its source — `owner/name` or the git URL as written
// in the dependency — or by its own package name. The policy applies to
// registry and git packages; a path dependency is the project's own code.
// `allow` takes capabilities and the claims named in `require`.
type ManifestPolicy struct {
	Deny    []string
	Allow   map[string][]string
	Trust   map[string]string // reviewer name -> "ed25519:<base64 public key>"
	Require map[string]int    // claim -> distinct trusted keys needed
}

func (p ManifestPolicy) Empty() bool { return len(p.Deny) == 0 && len(p.Require) == 0 }

func (p ManifestPolicy) allowed(key string, names []string) bool {
	for _, allowed := range p.Allow[key] {
		for _, n := range names {
			if n == allowed {
				return true
			}
		}
	}
	return false
}

// Denied reports whether a package named by any of names may not use the
// capability.
func (p ManifestPolicy) Denied(capability string, names ...string) bool {
	denied := false
	for _, d := range p.Deny {
		if d == capability {
			denied = true
		}
	}
	return denied && !p.allowed(capability, names)
}

// ReviewMissing reports whether a package named by any of names lacks the
// reviews the policy requires for a claim, given how many distinct trusted
// reviewers made it.
func (p ManifestPolicy) ReviewMissing(claim string, have int, names ...string) bool {
	need := p.Require[claim]
	return need > have && !p.allowed(claim, names)
}

func (r *manifestReader) policy(m *Manifest, t *tomlTable) error {
	m.Policy.Allow = map[string][]string{}
	var allowAt *tomlVal
	for _, key := range t.keys() {
		v := t.vals[key]
		switch key {
		case "deny":
			list, err := r.strList(v, "[policy] deny", `deny = ["net"]`)
			if err != nil {
				return err
			}
			for _, c := range list {
				if !validCapability(c) {
					return r.fail(v, "[policy] deny: unknown capability %q (%s)", c, strings.Join(Capabilities, ", "))
				}
			}
			m.Policy.Deny = list
		case "allow":
			if v.kind != tomlTab {
				return r.fail(v, "[policy] allow is a table of capability = [packages], e.g. allow = { net = [\"acme/httputil\"] }")
			}
			allowAt = v
			for _, c := range v.tab.keys() {
				list, err := r.strList(v.tab.vals[c], "[policy] allow."+c, c+` = ["owner/name"]`)
				if err != nil {
					return err
				}
				m.Policy.Allow[c] = list
			}
		case "trust":
			if v.kind != tomlTab {
				return r.fail(v, "[policy] trust is a table of reviewer = key, e.g. trust = { alice = \"ed25519:…\" }")
			}
			m.Policy.Trust = map[string]string{}
			for _, who := range v.tab.keys() {
				item := v.tab.vals[who]
				if item.kind != tomlStr || !strings.HasPrefix(item.str, "ed25519:") {
					return r.fail(item, "[policy] trust.%s is a public key written ed25519:<base64>", who)
				}
				m.Policy.Trust[who] = item.str
			}
		case "require":
			if v.kind != tomlTab {
				return r.fail(v, "[policy] require is a table of claim = count, e.g. require = { reviewed = 1 }")
			}
			m.Policy.Require = map[string]int{}
			for _, claim := range v.tab.keys() {
				item := v.tab.vals[claim]
				if item.kind != tomlInt || item.num < 1 {
					return r.fail(item, "[policy] require.%s is how many trusted reviewers must attest it, 1 or more", claim)
				}
				m.Policy.Require[claim] = int(item.num)
			}
		default:
			return r.fail(v, "unknown [policy] key %q (deny, allow, trust, require)", key)
		}
	}
	if len(m.Policy.Require) > 0 && len(m.Policy.Trust) == 0 {
		return r.fail(&tomlVal{line: t.line}, "[policy] require needs trust = { name = \"ed25519:…\" }: whose reviews count?")
	}
	if allowAt != nil {
		for c := range m.Policy.Allow {
			if _, claim := m.Policy.Require[c]; !validCapability(c) && !claim {
				return r.fail(allowAt, "[policy] allow: %q is neither a capability (%s) nor a claim in require", c, strings.Join(Capabilities, ", "))
			}
		}
	}
	return nil
}

// ManifestRegistry is the `[registry]` table (D139): where registry packages
// come from, and the key their metadata is signed with.
//
//	[registry]
//	url = "https://registry.example.com"
//	key = "ed25519:…"
//
// VELES_PROXY overrides url. Without a key the metadata is still checked
// against the archive and veles.sum, but the registry itself is trusted on
// first use.
type ManifestRegistry struct {
	URL string
	Key string
}

func (r *manifestReader) registry(m *Manifest, t *tomlTable) error {
	for _, key := range t.keys() {
		v := t.vals[key]
		if v.kind != tomlStr {
			return r.fail(v, "[registry] %s is a string", key)
		}
		switch key {
		case "url":
			if !strings.HasPrefix(v.str, "http://") && !strings.HasPrefix(v.str, "https://") {
				return r.fail(v, "[registry] url is an http:// or https:// address")
			}
			m.Registry.URL = v.str
		case "key":
			if !strings.HasPrefix(v.str, "ed25519:") {
				return r.fail(v, "[registry] key is a public key written ed25519:<base64>")
			}
			m.Registry.Key = v.str
		default:
			return r.fail(v, "unknown [registry] key %q (url, key)", key)
		}
	}
	return nil
}

// ReadManifestIn reads the `veles.toml` in exactly dir, or returns nil when
// there is none. Unlike ReadManifest it never looks in a parent directory,
// which is the right question for a directory already known to be a package
// (a dependency, a workspace member): a missing manifest there must not be
// answered with whatever package happens to enclose it.
func ReadManifestIn(dir string) (*Manifest, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return readManifest(abs)
}
