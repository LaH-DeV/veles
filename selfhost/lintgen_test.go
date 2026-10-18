package selfhost

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/format"
	"github.com/LaH-DeV/veles/source"
)

// std/lint is the API a lint rule is written against (D177): the Veles syntax
// tree, the file and span types it hangs on, and a Visitor with one callback
// per node. The tree is selfhost/ast written out again as public declarations,
// so a rule sees what the Veles front end builds, and this test regenerates
// the files from it: go test ./selfhost -run LintAPI -update. Without -update
// it fails when a generated file is stale.

var (
	lintTopDecl = regexp.MustCompile(`(?m)^(struct|sealed trait|enum|fun) `)
	lintMember  = regexp.MustCompile(`^  (var|val|static fun|static val|const fun|fun) `)
	lintStruct  = regexp.MustCompile(`(?m)^struct (\w+)`)
)

const lintHeader = "// Generated from %s by `go test ./selfhost -run LintAPI -update`; edit the source, not this file.\n\n"

func readLF(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// public makes every top-level declaration and every member of one visible
// outside the package: a rule lives in another package.
func lintPublic(src string) string {
	src = lintTopDecl.ReplaceAllString(src, "public ${1} ")
	var out []string
	inType := false
	for _, line := range strings.Split(src, "\n") {
		switch {
		case strings.HasPrefix(line, "public struct ") || strings.HasPrefix(line, "public enum ") || strings.HasPrefix(line, "public sealed trait "):
			inType = strings.HasSuffix(line, "{")
		case line == "}":
			inType = false
		case inType:
			if lintMember.MatchString(line) {
				line = "  public " + line[2:]
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func lintFiles(t *testing.T) map[string]string {
	ast := readLF(t, "ast/ast.vs")
	ast = strings.Replace(ast, "use lexer { Kind }\nuse source { File, Span }\n", "", 1)

	tok := readLF(t, "lexer/token.vs")
	kind := tok[strings.Index(tok, "enum Kind {"):]
	kind = kind[:strings.Index(kind, "\n}\n")+3]

	srcText := readLF(t, "source/source.vs")
	srcText = srcText[strings.Index(srcText, "/// One source file"):strings.Index(srcText, "enum Severity")]

	var nodes []string
	for _, m := range lintStruct.FindAllStringSubmatch(ast, -1) {
		nodes = append(nodes, m[1])
	}
	sort.Strings(nodes)
	var visitor strings.Builder
	visitor.WriteString(strings.Replace(lintHeader, "%s", "ast/ast.vs", 1))
	visitor.WriteString("/// What a rule looks at: one optional callback per kind of syntax node, called\n")
	visitor.WriteString("/// with the node and the context when the compiler enters it, parents before\n")
	visitor.WriteString("/// children. Set the callbacks the rule needs and leave the rest null.\n")
	visitor.WriteString("public struct Visitor {\n")
	for _, n := range nodes {
		visitor.WriteString("  public on" + n + ": (fun(" + n + ", Context))? = null\n")
	}
	visitor.WriteString("}\n")

	out := map[string]string{
		"ast.vs":     strings.Replace(lintHeader, "%s", "selfhost/ast/ast.vs", 1) + lintPublic(ast),
		"kind.vs":    strings.Replace(lintHeader, "%s", "selfhost/lexer/token.vs", 1) + lintPublic(kind),
		"source.vs":  strings.Replace(lintHeader, "%s", "selfhost/source/source.vs", 1) + lintPublic(srcText),
		"visitor.vs": visitor.String(),
	}
	// the files are kept as the formatter writes them
	for name, text := range out {
		formatted, diags := format.Source(source.NewFile("std/lint/"+name, text), format.Default)
		if diags.HasErrors() {
			t.Fatalf("%s does not parse:\n%s", name, diags.Render())
		}
		out[name] = formatted
	}
	return out
}

func TestLintAPI(t *testing.T) {
	for name, want := range lintFiles(t) {
		path := filepath.Join("..", "std", "lint", name)
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if got := readLF(t, path); got != want {
			t.Errorf("%s is stale: go test ./selfhost -run LintAPI -update", path)
		}
	}
}
