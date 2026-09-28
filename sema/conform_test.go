package sema

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// The conformance suite (plan A4). Each sema/testdata/conform/*.vs is a
// program named after the decision it exercises (`D05-nullable.vs`); a
// line that must draw a diagnostic says so at its end:
//
//	val n: i64 = "one"   // error: type mismatch
//	val unused = 1       // warning: is never used
//
// The text after `error:`/`warning:` must occur in a diagnostic of that
// severity reported on that line, and every diagnostic the file draws must
// be expected — a case says exactly what it provokes.
//
// Coverage: every diagnostic the checker can report is known by its format
// string, found by reading the checker's source. A full `go test ./sema`
// records which formats were reported (by any test, the suite included),
// and a format that nothing provoked must be listed in
// testdata/conform/uncovered.txt: a new diagnostic comes with a case, and
// one that gains a case leaves the list. `-conform-update` rewrites the list.

var conformUpdate = flag.Bool("conform-update", false, "rewrite testdata/conform/uncovered.txt from this run's coverage")

var reported = struct {
	sync.Mutex
	formats map[string]bool
}{formats: map[string]bool{}}

func TestMain(m *testing.M) {
	noteDiagnostic = func(format string) {
		reported.Lock()
		reported.formats[format] = true
		reported.Unlock()
	}
	code := m.Run()
	if code == 0 && wholeRun() {
		if err := checkCoverage(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
	}
	os.Exit(code)
}

// wholeRun: coverage means something only when every test ran.
func wholeRun() bool {
	for _, name := range []string{"test.run", "test.skip"} {
		if f := flag.Lookup(name); f != nil && f.Value.String() != "" {
			return false
		}
	}
	return !testing.Short()
}

var expectation = regexp.MustCompile(`//\s*(error|warning):\s*(.+?)\s*$`)

func TestConformance(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "conform", "*.vs"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no conformance cases: %v", err)
	}
	for _, path := range files {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".vs"), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			src := strings.ReplaceAll(string(data), "\r\n", "\n")
			type want struct {
				line     int
				severity source.Severity
				text     string
				matched  bool
			}
			var wants []*want
			for i, line := range strings.Split(src, "\n") {
				// `// error: a // warning: b`: one line may expect several
				for _, part := range strings.Split(line, "//")[1:] {
					if m := expectation.FindStringSubmatch("//" + part); m != nil {
						sev := source.Error
						if m[1] == "warning" {
							sev = source.Warning
						}
						wants = append(wants, &want{line: i + 1, severity: sev, text: m[2]})
					}
				}
			}
			diags := checkSource(t, src)
			for _, d := range diags.Items {
				if d.Severity != source.Error && d.Severity != source.Warning {
					continue
				}
				line, _ := d.Span.File.Position(d.Span.Start)
				expected := false
				for _, w := range wants {
					if w.line == line && w.severity == d.Severity && strings.Contains(d.Message, w.text) {
						w.matched, expected = true, true
					}
				}
				if !expected {
					t.Errorf("%s:%d: unexpected %s: %s", filepath.Base(path), line, d.Severity, d.Message)
				}
			}
			for _, w := range wants {
				if !w.matched {
					t.Errorf("%s:%d: expected a %s containing %q; got none on that line", filepath.Base(path), w.line, w.severity, w.text)
				}
			}
		})
	}
}

// checkerFormats reads the checker's source for the format of every
// diagnostic it can report: the literal format argument of each call to
// errorf, warnf, errorFix and warnFix. A format built at run time (a
// pass-through "%s") is not a diagnostic of its own and is skipped.
func checkerFormats() (map[string]string, error) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		return nil, err
	}
	formatArg := map[string]int{"errorf": 1, "warnf": 1, "errorFix": 2, "warnFix": 2}
	fset := token.NewFileSet()
	var files []*ast.File
	consts := map[string]*ast.BasicLit{} // `const name = "format"`, shared by several sites
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
		for _, decl := range file.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
				for _, spec := range gd.Specs {
					vs := spec.(*ast.ValueSpec)
					for i, name := range vs.Names {
						if i < len(vs.Values) {
							if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
								consts[name.Name] = lit
							}
						}
					}
				}
			}
		}
	}
	out := map[string]string{} // format -> first place it is reported from
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			i, ok := formatArg[sel.Sel.Name]
			if !ok || len(call.Args) <= i {
				return true
			}
			lit, ok := call.Args[i].(*ast.BasicLit)
			if id, isName := call.Args[i].(*ast.Ident); isName {
				lit, ok = consts[id.Name]
			}
			if !ok || lit.Kind != token.STRING {
				return true
			}
			format, err := strconv.Unquote(lit.Value)
			if err != nil || format == "%s" {
				return true
			}
			if _, seen := out[format]; !seen {
				out[format] = fset.Position(lit.Pos()).String()
			}
			return true
		})
	}
	return out, nil
}

const uncoveredPath = "testdata/conform/uncovered.txt"

// checkCoverage compares what this run reported with every format the
// checker has and with the list of those still without a case.
func checkCoverage() error {
	all, err := checkerFormats()
	if err != nil {
		return err
	}
	var uncovered []string
	for format := range all {
		if !reported.formats[format] {
			uncovered = append(uncovered, format)
		}
	}
	sort.Strings(uncovered)
	// listed format -> why it has no case ("" when nobody said), from
	// lines `"format"` or `"format"  # why`
	listed := map[string]string{}
	if data, err := os.ReadFile(uncoveredPath); err == nil {
		for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			quoted, err := strconv.QuotedPrefix(line)
			if err != nil {
				return fmt.Errorf("%s: %q does not start with a quoted format", uncoveredPath, line)
			}
			f, _ := strconv.Unquote(quoted)
			listed[f] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line[len(quoted):]), "#"))
		}
	}
	if *conformUpdate {
		var b strings.Builder
		b.WriteString("# Diagnostics no test provokes: one Go-quoted format per line, then\n")
		b.WriteString("# `# why` no program can reach it. A reachable one has a case in\n")
		b.WriteString("# sema/testdata/conform instead; a covered one must leave this list (`go test ./sema -conform-update`\n")
		b.WriteString("# rewrites it, keeping the reasons).\n")
		for _, f := range uncovered {
			b.WriteString(strconv.Quote(f))
			if why := listed[f]; why != "" {
				b.WriteString("  # " + why)
			}
			b.WriteString("\n")
		}
		return os.WriteFile(uncoveredPath, []byte(b.String()), 0o644)
	}
	var problems []string
	for _, f := range uncovered {
		why, ok := listed[f]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("  no test provokes %q (%s): add a case to sema/testdata/conform", f, all[f]))
		case why == "":
			// every diagnostic has a case or the reason it cannot have one
			// (2026-09-28: the list reached that state; it stays there)
			problems = append(problems, fmt.Sprintf("  %q (%s) is listed without a reason: add a case to sema/testdata/conform, or say after '#' why no program reaches it", f, all[f]))
		}
	}
	for f := range listed {
		if _, exists := all[f]; !exists {
			problems = append(problems, fmt.Sprintf("  %q is listed but no longer reported anywhere: remove it", f))
		} else if reported.formats[f] {
			problems = append(problems, fmt.Sprintf("  %q is covered now: remove it from the list", f))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("conformance coverage (%s; `go test ./sema -conform-update` rewrites it):\n%s", uncoveredPath, strings.Join(problems, "\n"))
}
