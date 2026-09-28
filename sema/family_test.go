package sema

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

var familiesDump = flag.Bool("families-dump", false, "print every diagnostic format under its family")

// Every message the front end can report belongs to a family (D79), so
// `see: veles explain <family>` and the editor's link never go missing: a
// new diagnostic either reads like its family's others or gets a family
// of its own, with a section in reference/errors.md.
func TestEveryDiagnosticHasAFamily(t *testing.T) {
	formats, err := frontEndFormats("*.go", "../parser/*.go", "../lexer/*.go")
	if err != nil {
		t.Fatal(err)
	}
	byFamily := map[string][]string{}
	for format, at := range formats {
		f := source.FamilyOf(sampleMessage(format))
		if f == "" {
			t.Errorf("%s: no family claims %q; extend a family's patterns in source/family.go or add a family (and its section in reference/errors.md)", at, format)
		}
		byFamily[f] = append(byFamily[f], format)
	}
	for _, name := range source.Families() {
		if len(byFamily[name]) == 0 {
			t.Errorf("family %q claims no diagnostic; remove it", name)
		}
	}
	if *familiesDump {
		for _, name := range source.Families() {
			sort.Strings(byFamily[name])
			t.Logf("== %s (%d)\n  %s", name, len(byFamily[name]), strings.Join(byFamily[name], "\n  "))
		}
	}
}

// frontEndFormats reads the format of every diagnostic in the matched Go
// files: the first string literal among the first three arguments of an
// errorf/warnf/errorFix/warnFix/Errorf/Warnf call (the lexer passes two
// positions before it). A format that is only "%s" passes a message built
// elsewhere and is left out.
func frontEndFormats(globs ...string) (map[string]string, error) {
	reporters := map[string]bool{"errorf": true, "warnf": true, "errorFix": true, "warnFix": true, "Errorf": true, "Warnf": true}
	fset := token.NewFileSet()
	out := map[string]string{}
	for _, glob := range globs {
		paths, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil, err
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !reporters[sel.Sel.Name] {
					return true
				}
				for i := 0; i < len(call.Args) && i < 3; i++ {
					lit, ok := call.Args[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					format, err := strconv.Unquote(lit.Value)
					if err == nil && format != "%s" {
						out[format] = fset.Position(lit.Pos()).String()
					}
					break
				}
				return true
			})
		}
	}
	return out, nil
}

var verb = regexp.MustCompile(`%[-+# 0]*\d*(\.\d+)?[a-zA-Z%]`)

// sampleMessage fills a format's verbs the way a real report would, so the
// families are tested against text a user sees: a name for each %s, a
// number for each %d.
func sampleMessage(format string) string {
	return verb.ReplaceAllStringFunc(format, func(v string) string {
		switch v[len(v)-1] {
		case '%':
			return "%"
		case 'd', 'X', 'x':
			return "3"
		case 'c':
			return "x"
		case 'q':
			return `"name"`
		}
		return "name"
	})
}
