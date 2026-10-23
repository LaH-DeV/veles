package selfhost

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/format"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// Gate G2 compares the trees as `veles parse` prints them, and that text
// carries no spans. This gate compares every span instead: each side lists
// `Node.field start end` for every span-typed field of every node, and the
// sorted sets must be equal. The Go side walks the tree by reflection; the
// Veles side walks it with selfhost/ast/spans.vs, which this file generates
// from selfhost/ast/ast.vs (go test ./selfhost -run SpanWalker -update).

var (
	spanTrait   = regexp.MustCompile(`^sealed trait (\w+) \{`)
	spanStruct  = regexp.MustCompile(`^( *)struct (\w+) \{`)
	spanField   = regexp.MustCompile(`^ +var (\w+): +([^=/]+?) *(=.*)?(//.*)?$`)
	spanListOf  = regexp.MustCompile(`^List<(.+)>$`)
	spanPointer = regexp.MustCompile(`^\(\*(\w+)\)\?$`)
)

type spanNode struct {
	name   string
	fields [][2]string // name, type
}

// spanWalker generates spans.vs: one function per node struct and per sealed
// trait of the Veles syntax tree.
func spanWalker(t *testing.T) string {
	src := readLF(t, "ast/ast.vs")
	traits := map[string][]string{} // a sealed trait's variants
	var traitOrder []string
	var nodes []*spanNode
	var cur *spanNode
	trait := ""
	for _, line := range strings.Split(src, "\n") {
		if m := spanTrait.FindStringSubmatch(line); m != nil {
			trait = m[1]
			traitOrder = append(traitOrder, trait)
			continue
		}
		if line == "}" {
			trait, cur = "", nil
			continue
		}
		if m := spanStruct.FindStringSubmatch(line); m != nil {
			cur = &spanNode{name: m[2]}
			nodes = append(nodes, cur)
			if len(m[1]) == 2 && trait != "" {
				traits[trait] = append(traits[trait], m[2])
			}
			continue
		}
		if m := spanField.FindStringSubmatch(line); m != nil && cur != nil {
			cur.fields = append(cur.fields, [2]string{m[1], strings.TrimSpace(m[2])})
		}
	}
	known := map[string]bool{}
	for _, n := range nodes {
		known[n.name] = true
	}
	for _, tr := range traitOrder {
		known[tr] = true
	}

	var b strings.Builder
	b.WriteString("// Generated from ast/ast.vs by `go test ./selfhost -run SpanWalker -update`; edit the source, not this file.\n\n")
	b.WriteString("/// Every span of the tree, one line each: `Node.field start end` — what the\n")
	b.WriteString("/// span gate compares with the bootstrap parser's tree.\n")
	b.WriteString("fun spans(file: SourceFile): List<string> {\n  val out: MutableList<string> = []\n  spansOfSourceFile(file, out)\n  out.toList()\n}\n")
	for _, tr := range traitOrder {
		fmt.Fprintf(&b, "\nfun spansOf%s(node: %s, out: MutableList<string>) {\n  when (node) {\n", tr, tr)
		for _, v := range traits[tr] {
			fmt.Fprintf(&b, "    is %s => spansOf%s(node, out)\n", v, v)
		}
		b.WriteString("  }\n}\n")
	}
	for _, n := range nodes {
		var body strings.Builder
		depth := 0
		var emit func(expr, typ, indent string)
		emit = func(expr, typ, indent string) {
			depth++
			v := fmt.Sprintf("x%d", depth)
			switch {
			case typ == "Span":
				fmt.Fprintf(&body, "%sout.push(\"%s.%s ${%s.start} ${%s.end}\")\n", indent, n.name, strings.TrimPrefix(expr, "node."), expr, expr)
			case strings.HasSuffix(typ, "?") && spanPointer.MatchString(typ):
				inner := spanPointer.FindStringSubmatch(typ)[1]
				if known[inner] {
					fmt.Fprintf(&body, "%sif (val %s = %s) spansOf%s(*%s, out)\n", indent, v, expr, inner, v)
				}
			case strings.HasSuffix(typ, "?"):
				inner := strings.TrimSuffix(typ, "?")
				if needsWalk(inner, known) {
					fmt.Fprintf(&body, "%sif (val %s = %s) {\n", indent, v, expr)
					emit(v, inner, indent+"  ")
					fmt.Fprintf(&body, "%s}\n", indent)
				}
			case strings.HasPrefix(typ, "*"):
				if known[typ[1:]] {
					fmt.Fprintf(&body, "%sspansOf%s(*%s, out)\n", indent, typ[1:], expr)
				}
			case spanListOf.MatchString(typ):
				inner := spanListOf.FindStringSubmatch(typ)[1]
				if needsWalk(inner, known) {
					fmt.Fprintf(&body, "%sloop (%s in %s) {\n", indent, v, expr)
					emit(v, inner, indent+"  ")
					fmt.Fprintf(&body, "%s}\n", indent)
				}
			case known[typ]:
				fmt.Fprintf(&body, "%sspansOf%s(%s, out)\n", indent, typ, expr)
			}
		}
		for _, f := range n.fields {
			emit("node."+f[0], f[1], "  ")
		}
		param := "node"
		if body.Len() == 0 {
			param = "_"
		}
		fmt.Fprintf(&b, "\nfun spansOf%s(%s: %s, out: MutableList<string>) {\n%s}\n", n.name, param, n.name, body.String())
	}
	formatted, diags := format.Source(source.NewFile("selfhost/ast/spans.vs", b.String()), format.Default)
	if diags.HasErrors() {
		t.Fatalf("spans.vs does not parse:\n%s\n%s", diags.Render(), b.String())
	}
	return formatted
}

// needsWalk: a value of this type holds a span somewhere.
func needsWalk(typ string, known map[string]bool) bool {
	if typ == "Span" || known[typ] || known[strings.TrimPrefix(typ, "*")] {
		return true
	}
	if m := spanPointer.FindStringSubmatch(typ); m != nil {
		return known[m[1]]
	}
	if m := spanListOf.FindStringSubmatch(typ); m != nil {
		return needsWalk(m[1], known)
	}
	return strings.HasSuffix(typ, "?") && needsWalk(strings.TrimSuffix(typ, "?"), known)
}

func TestSpanWalker(t *testing.T) {
	want := spanWalker(t)
	path := filepath.Join("ast", "spans.vs")
	if *update {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got := readLF(t, path); got != want {
		t.Errorf("%s is stale: go test ./selfhost -run SpanWalker -update", path)
	}
}

// TestP6Spans is the span gate: every span the Veles parser records is the
// one the Go parser records, node for node.
func TestP6Spans(t *testing.T) {
	exe := buildSelfhost(t)
	for _, file := range corpus(t) {
		if file.removed {
			continue
		}
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			compare(t, exe, "--spans", file.path, goSpans(source.NewFile(file.path, file.content)))
		})
	}
}

// TestP6Layout is gate P6: the formatter's input besides the tree — the
// comments and the spans of parenthesised expressions — agrees with
// parser.ParseFileLayout's.
func TestP6Layout(t *testing.T) {
	exe := buildSelfhost(t)
	for _, file := range corpus(t) {
		if file.removed {
			continue
		}
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			compare(t, exe, "--layout", file.path, goLayout(source.NewFile(file.path, file.content)))
		})
	}
}

func goLayout(f *source.File) string {
	var diags source.Diagnostics
	_, layout := parser.ParseFileLayout(f, &diags)
	var out strings.Builder
	for _, c := range layout.Comments {
		fmt.Fprintf(&out, "comment %d %d %s\n", c.Span.Start, c.Span.End, escaped(c.Text))
	}
	var parens []string
	for sp := range layout.Parens {
		parens = append(parens, fmt.Sprintf("paren %d %d", sp[0], sp[1]))
	}
	sort.Strings(parens)
	for _, p := range parens {
		out.WriteString(p + "\n")
	}
	return out.String()
}

// the Go nodes and span fields whose Veles names differ
var (
	spanNodeNames  = map[string]string{"File": "SourceFile", "SelfExpr": "ThisExpr"}
	spanFieldNames = map[string]string{"AnnotationInfo.ForPos": "readersPos"}
)

var (
	spanType = reflect.TypeOf(source.Span{})
	fileType = reflect.TypeOf((*source.File)(nil))
)

// goSpans lists every span of the Go parser's tree, sorted, each once (the
// parser shares some nodes between two places of the tree).
func goSpans(f *source.File) string {
	var diags source.Diagnostics
	tree := parser.ParseFile(f, &diags)
	seen := map[string]bool{}
	visited := map[uintptr]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() || visited[v.Pointer()] {
				return
			}
			visited[v.Pointer()] = true
			walk(v.Elem())
		case reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem())
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		case reflect.Struct:
			t := v.Type()
			name := t.Name()
			if n, ok := spanNodeNames[name]; ok {
				name = n
			}
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				if !field.IsExported() || field.Type == fileType {
					continue
				}
				if field.Type.Kind() == reflect.Interface && field.Type.NumMethod() == 0 {
					continue // a types.Type the checker fills in
				}
				if field.Type == spanType {
					sp := v.Field(i).Interface().(source.Span)
					r := []rune(field.Name)
					r[0] = unicode.ToLower(r[0])
					fname := string(r)
					if n, ok := spanFieldNames[t.Name()+"."+field.Name]; ok {
						fname = n
					}
					seen[fmt.Sprintf("%s.%s %d %d", name, fname, sp.Start, sp.End)] = true
					continue
				}
				walk(v.Field(i))
			}
		}
	}
	walk(reflect.ValueOf(tree))
	lines := make([]string, 0, len(seen))
	for l := range seen {
		lines = append(lines, l)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

var _ = ast.Dump // the tree the walk reads is ast's
