package sema

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

func TestLintRegistryIsSorted(t *testing.T) {
	names := make([]string, len(Lints))
	for i, l := range Lints {
		names[i] = l.Name
		if l.Doc == "" {
			t.Errorf("lint %s has no documentation", l.Name)
		}
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("Lints is not sorted by name: %v", names)
	}
}

func TestManifestLintLevels(t *testing.T) {
	pkg := "[package]\nname = \"p\"\n"
	m := parseOK(t, pkg+"[lint]\nunused_import = \"deny\"\nlong_line = \"warn\"\nmax_width = 100\nredundant_mut = \"allow\"\n")
	if m.lintLevel("unused_import") != LintDeny || m.lintLevel("long_line") != LintWarn || m.lintLevel("redundant_mut") != LintAllow {
		t.Errorf("levels %+v", m.Lint.Levels)
	}
	if m.Lint.MaxWidth != 100 {
		t.Errorf("max_width %d", m.Lint.MaxWidth)
	}
	// defaults: warnings stay warnings, long_line is off
	m = parseOK(t, pkg)
	if m.lintLevel("unused_variable") != LintWarn || m.lintLevel("long_line") != LintAllow {
		t.Errorf("defaults: unused_variable %v, long_line %v", m.lintLevel("unused_variable"), m.lintLevel("long_line"))
	}
	for _, c := range []struct{ text, want string }{
		{pkg + "[lint]\nunused_import = \"error\"\n", "[lint] unused_import is \"allow\""},
		{pkg + "[lint]\nunused_import = true\n", "[lint] unused_import is \"allow\""},
		{pkg + "[lint]\nunused_imports = \"deny\"\n", "did you mean \"unused_import\"?"},
		{pkg + "[lint]\nacme.no_sleep = \"loud\"\n", "[lint] acme.no_sleep is \"allow\""},
		{pkg + "[lint]\nmax_width = 5\n", "[lint] max_width is a number of columns"},
		{pkg + "[format]\ntrailing_commas = \"always\"\n", "[format] trailing_commas is \"multiline\""},
		{pkg + "[format]\nline_endings = \"cr\"\n", "[format] line_endings is \"lf\""},
	} {
		if _, err := parseManifest("veles.toml", ".", c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q:\n got %v\nwant %q", c.text, err, c.want)
		}
	}
	// a rule of a dependency (std/lint): quoted, bare dotted and table keys
	lib := parseOK(t, pkg+"[lint]\n\"acme.no_sleep\" = \"warn\"\nacme_lints.lower = \"deny\"\n\n[lint.other]\nrule = \"allow\"\n")
	if lib.Lint.Library["acme.no_sleep"] != LintWarn || lib.Lint.Library["acme_lints.lower"] != LintDeny || lib.Lint.Library["other.rule"] != LintAllow {
		t.Errorf("library lints %+v", lib.Lint.Library)
	}
	f := parseOK(t, pkg+"[format]\ntrailing_commas = \"never\"\nline_endings = \"crlf\"\n")
	if !f.Format.NoTrailingCommas || f.Format.LineEndings != "crlf" {
		t.Errorf("format %+v", f.Format)
	}
}

// The `[lint]` levels act on the diagnostics of the package's own code:
// "allow" drops one, "deny" makes it an error; `long_line` is off by default.
func TestLintLevelsApply(t *testing.T) {
	long := "  io.println(\"" + strings.Repeat("word ", 30) + "\")"
	src := "use io\nuse fs { readFile }\n\nfun main() {\n  val unused = 1\n" + long + "\n}\n"
	run := func(lint string) *source.Diagnostics {
		root := t.TempDir()
		for name, text := range map[string]string{"veles.toml": "[package]\nname = \"app\"\n" + lint, "main.vs": src} {
			if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		diags := &source.Diagnostics{}
		pkg, err := LoadPackage(root, diags)
		if err != nil {
			t.Fatal(err)
		}
		Check(pkg, diags, false)
		return diags
	}
	count := func(d *source.Diagnostics, lint string, sev source.Severity) int {
		n := 0
		for _, it := range d.Items {
			if it.Lint == lint && it.Severity == sev {
				n++
			}
		}
		return n
	}

	d := run("")
	if count(d, "unused_import", source.Warning) != 1 || count(d, "unused_variable", source.Warning) != 1 || count(d, "long_line", source.Warning) != 0 || d.HasErrors() {
		t.Errorf("defaults:\n%s", d.Render())
	}
	d = run("[lint]\nunused_import = \"allow\"\nunused_variable = \"deny\"\nlong_line = \"warn\"\n")
	if count(d, "unused_import", source.Warning) != 0 || count(d, "unused_variable", source.Error) != 1 || count(d, "long_line", source.Warning) != 1 {
		t.Errorf("configured:\n%s", d.Render())
	}
	d = run("[lint]\nlong_line = \"deny\"\nmax_width = 400\n")
	if count(d, "long_line", source.Error) != 0 {
		t.Errorf("a line under max_width is reported:\n%s", d.Render())
	}
}
