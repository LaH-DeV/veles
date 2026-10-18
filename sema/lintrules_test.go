package sema

import (
	"github.com/LaH-DeV/veles/source"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ruleAllowedMissing are the fields of std/lint's tree that the Go syntax tree
// has no counterpart for: they are left at their zero value.
var ruleAllowedMissing = map[string]bool{
	"File.lineStarts": true, // computed from the text
}

// TestRuleASTFields checks that every field of every node std/lint declares
// is found in the Go syntax tree the compiler builds it from, so a field added
// on one side and not the other fails here rather than reaching a rule as a
// zero value.
func TestRuleASTFields(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "../ast/ast.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	goFields := map[string][]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			for _, f := range st.Fields.List {
				for _, name := range f.Names {
					goFields[ts.Name.Name] = append(goFields[ts.Name.Name], name.Name)
				}
			}
		}
		return true
	})
	data, err := os.ReadFile("../std/lint/ast.vs")
	if err != nil {
		t.Fatal(err)
	}
	structRe := regexp.MustCompile(`^public struct (\w+)`)
	fieldRe := regexp.MustCompile(`^  public var (\w+):`)
	var cur string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if m := structRe.FindStringSubmatch(line); m != nil {
			cur = m[1]
			continue
		}
		m := fieldRe.FindStringSubmatch(line)
		if m == nil || cur == "" {
			continue
		}
		goName := cur
		if n, ok := ruleGoName[cur]; ok {
			_ = n
		}
		for g, v := range ruleGoName {
			if v == cur {
				goName = g
			}
		}
		fields, known := goFields[goName]
		if !known {
			continue // a wrapper variant (FunDeclNode, BlockStmt…): built from the node it wraps
		}
		found := false
		for _, gf := range fields {
			for _, want := range ruleFieldNames(cur, m[1]) {
				if strings.EqualFold(gf, want) {
					found = true
				}
			}
		}
		if !found && !ruleAllowedMissing[cur+"."+m[1]] {
			t.Errorf("std/lint %s.%s has no field in the Go syntax tree %s (%v): name it in ruleFieldAlias or the Go field the same", cur, m[1], goName, fields)
		}
	}
}

// libraryLintRun checks a copy of examples/lintrules with the app's manifest
// and main.vs replaced, and returns the rendered diagnostics.
func libraryLintRun(t *testing.T, manifest, main string) (*source.Diagnostics, string) {
	t.Helper()
	root := t.TempDir()
	copyTree(t, "../examples/lintrules/acme-lints", filepath.Join(root, "acme-lints"))
	app := filepath.Join(root, "app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"veles.toml": manifest, "main.vs": main} {
		if err := os.WriteFile(filepath.Join(app, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(app, diags)
	if err != nil {
		t.Fatal(err)
	}
	Check(pkg, diags, false)
	return diags, diags.Render()
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

const libraryLintManifest = "[package]\nname = \"app\"\n\n[dependencies]\nacme_lints = \"../acme-lints\"\n\n[lint]\n"

const libraryLintMain = "use io { println }\n\nfun Shout(text: string): string => text.toUpper()\n\nfun main() {\n  println(Shout(\"hi\"))\n  await sleep(Duration.millis(5))\n}\n"

// TestLibraryLints: a rule written in Veles runs on the package that names it,
// at the level the manifest gave, and gives its fix to the diagnostic.
func TestLibraryLints(t *testing.T) {
	d, text := libraryLintRun(t, libraryLintManifest+"acme_lints.no_sleep = \"warn\"\nacme_lints.lower_case_functions = \"deny\"\n", libraryLintMain)
	var sleep, lower *source.Diagnostic
	for i, it := range d.Items {
		switch {
		case strings.Contains(it.Message, "[acme_lints.no_sleep]"):
			sleep = &d.Items[i]
		case strings.Contains(it.Message, "[acme_lints.lower_case_functions]"):
			lower = &d.Items[i]
		}
	}
	if sleep == nil || lower == nil || len(d.Items) != 2 {
		t.Fatalf("diagnostics:\n%s", text)
	}
	if sleep.Severity != source.Warning || lower.Severity != source.Error {
		t.Errorf("levels: no_sleep %v, lower_case_functions %v", sleep.Severity, lower.Severity)
	}
	if got := sleep.Span.File.Content[sleep.Span.Start:sleep.Span.End]; got != "await sleep(Duration.millis(5))" {
		t.Errorf("no_sleep reported %q", got)
	}
	if sleep.Fix == nil || len(sleep.Fix.Edits) != 1 || sleep.Fix.Edits[0].NewText != "" || sleep.Fix.Edits[0].Span != sleep.Span {
		t.Errorf("no_sleep fix: %+v", sleep.Fix)
	}
	if lower.Fix != nil {
		t.Errorf("lower_case_functions gave a fix: %+v", lower.Fix)
	}

	// a rule the package does not name does not run; "allow" is off
	d, text = libraryLintRun(t, libraryLintManifest+"acme_lints.no_sleep = \"allow\"\n", libraryLintMain)
	if len(d.Items) != 0 {
		t.Errorf("rules ran without being asked:\n%s", text)
	}
}

func TestLibraryLintErrors(t *testing.T) {
	for _, c := range []struct{ lint, want string }{
		{"acme_lints.no_sleeps = \"warn\"\n", "package 'acme_lints' has no rule 'no_sleeps'"},
		{"acme_lints.no_sleeps = \"warn\"\n", "did you mean 'no_sleep'?"},
		{"other.rule = \"warn\"\n", "'other' is not a dependency of this package"},
	} {
		d, text := libraryLintRun(t, libraryLintManifest+c.lint, libraryLintMain)
		if !d.HasErrors() || !strings.Contains(text, c.want) {
			t.Errorf("%q: want %q in:\n%s", c.lint, c.want, text)
		}
	}
}
