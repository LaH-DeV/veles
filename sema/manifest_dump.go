package sema

import (
	"fmt"
	"strconv"
	"strings"
)

// LintsDump is the registry of lints as selfhost/linter/registry.vs prints it:
// name, default level and documentation, one per line.
func LintsDump() string {
	var sb strings.Builder
	for _, l := range Lints {
		fmt.Fprintf(&sb, "%s\t%s\t%s\n", l.Name, l.Default, l.Doc)
	}
	return sb.String()
}

// ManifestDump is the text the self-hosted manifest reader
// (selfhost/sema/manifest.vs) is compared against, plan R1: what parseManifest
// makes of a veles.toml, field by field, or the error. selfhost/sema/
// manifest_dump.vs prints the same.
func ManifestDump(path, text string) string {
	m, err := parseManifest(path, "", text)
	if err != nil {
		return "error " + err.Error() + "\n"
	}
	var sb strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&sb, format+"\n", a...) }
	w("name %s", strconv.Quote(m.Name))
	w("version %s", strconv.Quote(m.Version))
	w("description %s", strconv.Quote(m.Description))
	w("license %s", strconv.Quote(m.License))
	w("veles %s", strconv.Quote(m.Veles))
	w("registry-name %s", strconv.Quote(m.RegistryName))
	for _, k := range sortedKeys(m.Deps) {
		w("dep %s %s", strconv.Quote(k), strconv.Quote(m.Deps[k]))
	}
	for _, d := range m.Requires {
		w("require %s", dumpDependency(d))
	}
	for _, d := range m.DevRequires {
		w("dev-require %s", dumpDependency(d))
	}
	if m.Workspace == nil {
		w("workspace none")
	} else {
		w("workspace %s", quoteAll(m.Workspace.Members))
	}
	f := m.Format
	w("format indent=%s max-blank-lines=%d merge-imports=%t no-trailing-commas=%t line-endings=%s",
		strconv.Quote(f.Indent), f.MaxBlankLines, f.MergeImports, f.NoTrailingCommas, strconv.Quote(f.LineEndings))
	w("lint implicit-return=%d max-width=%d", int(m.Lint.ImplicitReturn), m.Lint.MaxWidth)
	for _, k := range sortedKeys(m.Lint.Levels) {
		w("lint-level %s %s", strconv.Quote(k), m.Lint.Levels[k])
	}
	for _, k := range sortedKeys(m.Lint.Library) {
		w("lint-library %s %s", strconv.Quote(k), m.Lint.Library[k])
	}
	n := m.Native
	w("native libs=%s static-libs=%s lib-paths=%s pkg-config=%s", quoteAll(n.Libs), quoteAll(n.StaticLibs), quoteAll(n.LibPaths), quoteAll(n.PkgConfig))
	w("policy deny=%s", quoteAll(m.Policy.Deny))
	for _, k := range sortedKeys(m.Policy.Allow) {
		w("policy-allow %s %s", strconv.Quote(k), quoteAll(m.Policy.Allow[k]))
	}
	for _, k := range sortedKeys(m.Policy.Trust) {
		w("policy-trust %s %s", strconv.Quote(k), strconv.Quote(m.Policy.Trust[k]))
	}
	for _, k := range sortedKeys(m.Policy.Require) {
		w("policy-require %s %d", strconv.Quote(k), m.Policy.Require[k])
	}
	w("registry url=%s key=%s", strconv.Quote(m.Registry.URL), strconv.Quote(m.Registry.Key))
	w("runtime threads=%d", m.Runtime.Threads)
	return sb.String()
}

func dumpDependency(d Dependency) string {
	return fmt.Sprintf("%s %s path=%s registry=%s git=%s version=%s commit=%s line=%d", strconv.Quote(d.Name), d.Kind(),
		strconv.Quote(d.Path), strconv.Quote(d.Registry), strconv.Quote(d.Git), strconv.Quote(d.Version), strconv.Quote(d.Commit), d.Line)
}

func quoteAll(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = strconv.Quote(s)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
