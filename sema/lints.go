package sema

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/source"
)

// The lint registry (D177). A lint is a check a team may want louder, quieter
// or off: it has a name, a default level and a line of documentation, and a
// package sets its level in `[lint]` of its manifest:
//
//	[lint]
//	unused_import = "deny"      # allow | warn | deny
//	long_line = "warn"
//	max_width = 120
//
// "allow" drops the diagnostic, "warn" reports it as a warning, "deny" as an
// error. The level applies to the package's own code; a dependency follows its
// own manifest and the standard library the defaults.

// LintLevel is what a lint does when it finds something.
type LintLevel int

const (
	LintAllow LintLevel = iota // silent
	LintWarn                   // a warning
	LintDeny                   // an error
)

func (l LintLevel) String() string {
	switch l {
	case LintAllow:
		return "allow"
	case LintDeny:
		return "deny"
	}
	return "warn"
}

func parseLintLevel(s string) (LintLevel, bool) {
	switch s {
	case "allow":
		return LintAllow, true
	case "warn":
		return LintWarn, true
	case "deny":
		return LintDeny, true
	}
	return 0, false
}

// LintInfo describes one lint of the registry.
type LintInfo struct {
	Name    string
	Doc     string
	Default LintLevel
}

// Lints is every lint whose level a package may set, sorted by name.
// `implicit_return` is a lint too, but its setting picks where a body may end
// in its value rather than a level (lint_return.go), so it is not listed.
var Lints = []LintInfo{
	{"continue_skips_step", "a `continue` that jumps over the step of a hand-written counting loop", LintWarn},
	{"count_loop", "a loop that only counts is written `iter.len()`", LintWarn},
	{"else_if_ladder", "three or more `else if` branches in a row are a `when { }`", LintWarn},
	{"inline_implement", "an `implement` beside a type can be written inside its body", LintWarn},
	{"large_variant", "a variant much larger than the others of a sealed type that is not a tree, which every value is sized for: mark it `indirect`", LintWarn},
	{"long_line", "a line longer than `max_width` columns (default 120); the formatter never reflows", LintAllow},
	{"loop_is_range", "a counting loop that is `loop (i in a..<b)` written by hand", LintWarn},
	{"loop_true", "`loop (true)` is written `loop { }`", LintWarn},
	{"needless_throws", "a `throws` clause on a function that cannot throw", LintWarn},
	{"negated_is", "`!(x is T)` reads better as `x !is T`", LintWarn},
	{"nested_try", "a `try` inside a `try`", LintWarn},
	{"never_closed", "a `Closeable` that is never closed or handed on", LintWarn},
	{"param_copy", "a parameter changed in a function whose caller never sees the change", LintWarn},
	{"pattern_form", "a pattern written in a longer form than needed", LintWarn},
	{"redundant_copy", "a `.toList()` of a collection that is not used afterwards: without it the collection is moved immutable", LintWarn},
	{"redundant_discard", "`val _ = f()` for a value a statement may simply ignore", LintWarn},
	{"redundant_implement", "an `implement Trait { }` around the methods of a variant of that sealed trait", LintWarn},
	{"redundant_label", "an argument label that repeats the variable passed (`f(x: x)`)", LintWarn},
	{"redundant_mut", "`mut` on a literal whose expected type is already mutable", LintWarn},
	{"redundant_public", "`public` where the declaration is already public", LintWarn},
	{"self_address", "`&x` stored into `x` itself, so the value would contain itself", LintWarn},
	{"stale_reference", "a reference into a collection used after a call that may move its elements", LintWarn},
	{"unreachable_else", "an `else` arm after every variant or member has an arm", LintWarn},
	{"unreachable_fallback", "a `?:` fallback on a value known to be in range", LintWarn},
	{"unsafe_without_reason", "an `unsafe` block without a `// SAFETY:` comment", LintWarn},
	{"unused_import", "a name imported with `use m { name }` but never used", LintWarn},
	{"unused_variable", "a variable that is never used", LintWarn},
}

// LintNamed returns the registry entry of a lint.
func LintNamed(name string) (LintInfo, bool) {
	for _, l := range Lints {
		if l.Name == name {
			return l, true
		}
	}
	return LintInfo{}, false
}

// DefaultMaxWidth is `[lint] max_width` when a package does not set it.
const DefaultMaxWidth = 120

// lintLevel is the level of a lint for a package: its manifest's setting, or
// the lint's default.
func (m *Manifest) lintLevel(name string) LintLevel {
	if m != nil {
		if l, ok := m.Lint.Levels[name]; ok {
			return l
		}
	}
	info, _ := LintNamed(name)
	return info.Default
}

// lintManifests maps the directory of every non-std module to its package's
// manifest, so a diagnostic can find the settings of the package it is in.
func (c *Checker) lintManifests() map[string]*Manifest {
	out := map[string]*Manifest{}
	if c.pkg == nil {
		return out
	}
	for _, m := range c.pkg.Modules {
		if !m.Std && m.Pkg != nil {
			out[filepath.Clean(m.Dir)] = m.Pkg.Manifest
		}
	}
	return out
}

// applyLintLevels runs the `[lint]` settings over a round's diagnostics and
// adds the lints that read source text rather than the checked tree.
func (c *Checker) applyLintLevels(items []source.Diagnostic) []source.Diagnostic {
	mans := c.lintManifests()
	items = append(items, c.longLines(mans)...)
	items = append(items, c.libraryLints()...)
	out := make([]source.Diagnostic, 0, len(items))
	for _, d := range items {
		if d.Lint == "" {
			out = append(out, d)
			continue
		}
		var man *Manifest
		if d.Span.File != nil && !d.Span.File.Embedded {
			man = mans[filepath.Clean(filepath.Dir(d.Span.File.Path))]
		}
		switch man.lintLevel(d.Lint) {
		case LintAllow:
			continue
		case LintDeny:
			d.Severity = source.Error
		}
		out = append(out, d)
	}
	return out
}

// longLines is the `long_line` lint: a line wider than the package's
// `max_width`, counted in characters with a tab as four. A line with nothing
// to break at (a URL, one long string) is left alone: no layout would shorten
// it. The lint is off unless a manifest turns it on, so a package that does
// not ask pays for no scan.
func (c *Checker) longLines(mans map[string]*Manifest) []source.Diagnostic {
	if c.pkg == nil {
		return nil
	}
	keys := make([]string, 0, len(c.pkg.Modules))
	for k := range c.pkg.Modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []source.Diagnostic
	for _, k := range keys {
		m := c.pkg.Modules[k]
		if m.Std || m.Pkg == nil || m.Pkg.Manifest.lintLevel("long_line") == LintAllow {
			continue
		}
		width := m.Pkg.Manifest.Lint.MaxWidth
		if width == 0 {
			width = DefaultMaxWidth
		}
		for _, f := range m.Files {
			if f.Source == nil {
				continue
			}
			offset := 0
			for _, line := range strings.SplitAfter(f.Source.Content, "\n") {
				start := offset
				offset += len(line)
				text := strings.TrimRight(line, "\r\n")
				cols := utf8.RuneCountInString(text) + 3*strings.Count(text, "\t")
				if cols <= width || !strings.ContainsAny(strings.TrimSpace(text), " \t") {
					continue
				}
				out = append(out, source.Diagnostic{
					Severity: source.Warning,
					Span:     source.Span{File: f.Source, Start: start, End: start + len(text)},
					Message:  fmt.Sprintf("this line is %d columns wide, over [lint] max_width = %d: break it at an operator or after a comma", cols, width),
					Lint:     "long_line",
				})
			}
		}
	}
	return out
}
