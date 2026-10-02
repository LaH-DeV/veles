package lsp

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Inlay hints show what the compiler inferred and the source leaves out:
// the type of a binding written without one (`val n = parse(s)` → `: i64`),
// a function's return type from its `= expr` body, `suspends` on a function
// that suspends without saying so (D2), and the error set of a bare `throws`
// (D45). A binding whose type is spelled on its right — `val p = Point(...)`
// — gets none: the hint would repeat the line.

const (
	hintType = 1
)

type inlayHint struct {
	Position     lspPosition `json:"position"`
	Label        string      `json:"label"`
	Kind         int         `json:"kind"`
	PaddingLeft  bool        `json:"paddingLeft,omitempty"`
	PaddingRight bool        `json:"paddingRight,omitempty"`
}

func (s *Server) inlayHints(params json.RawMessage) any {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range lspRange `json:"range"`
	}
	json.Unmarshal(params, &p)
	out := []inlayHint{}
	a, f := s.analysisFor(p.TextDocument.URI)
	if a == nil || a.index == nil {
		return out
	}
	from, to := positionToOffset(f, p.Range.Start), positionToOffset(f, p.Range.End)
	if to <= from {
		to = len(f.Content)
	}
	in := func(off int) bool { return off >= from && off <= to }
	// a binding in a generic body is checked once per instance: when the
	// instances disagree on its type, no single hint is true
	label := map[int]string{}
	var order []int
	for i := range a.index.Refs {
		r := &a.index.Refs[i]
		if r.Span.File != f || r.Span != r.Def || (r.Kind != "val" && r.Kind != "var") || r.Type == nil || r.Name == "_" || !in(r.Span.End) {
			continue
		}
		if types.IsInvalid(r.Type) || !bindingUntyped(f.Content, r.Span.End) || typeOnTheRight(f.Content, r.Span.End, r.Type) {
			continue
		}
		l, seen := label[r.Span.End]
		switch {
		case !seen:
			label[r.Span.End] = ": " + r.Type.String()
			order = append(order, r.Span.End)
		case l != ": "+r.Type.String():
			label[r.Span.End] = ""
		}
	}
	for _, end := range order {
		if l := label[end]; l != "" {
			out = append(out, inlayHint{Position: offsetToPosition(f, end), Label: l, Kind: hintType})
		}
	}
	// parsed again from the same File, so spans compare equal to the index's
	file := parser.ParseFile(f, &source.Diagnostics{})
	for _, fn := range funDecls(file) {
		inf, ok := a.index.Inferred[fn.Name.Pos]
		if !ok || !in(fn.Name.Pos.Start) {
			continue
		}
		out = append(out, effectHints(f, fn, inf)...)
	}
	for _, h := range closeHints(f, file, a.index.Held) {
		if in(positionToOffset(f, h.Position)) {
			out = append(out, h)
		}
	}
	return out
}

// bindingUntyped reports whether the name ending at end is written without
// a `: Type` after it.
func bindingUntyped(text string, end int) bool {
	rest := strings.TrimLeft(text[end:], " \t")
	return !strings.HasPrefix(rest, ":")
}

// typeOnTheRight reports whether the initializer after the name ending at
// end starts by naming t: a constructor or static call (`Point(`,
// `Point.origin()`, `Box<i64>(`), whose type the reader already sees.
func typeOnTheRight(text string, end int, t types.Type) bool {
	rest := strings.TrimLeft(text[end:], " \t")
	if !strings.HasPrefix(rest, "=") {
		return false
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	name := t.String()
	if i := strings.IndexByte(name, '<'); i >= 0 {
		name = name[:i]
	}
	if j := strings.LastIndexByte(name, '.'); j >= 0 {
		name = name[j+1:]
	}
	// the type as written, or qualified by its module (under any alias)
	cands := []string{rest}
	if k := strings.IndexFunc(rest, func(r rune) bool { return !isIdentRune(r) }); k > 0 && rest[k] == '.' {
		cands = append(cands, rest[k+1:])
	}
	for _, c := range cands {
		if len(c) > len(name) && strings.HasPrefix(c, name) && strings.IndexByte("(<.", c[len(name)]) >= 0 {
			return true
		}
	}
	return false
}

func isIdentRune(r rune) bool {
	return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r > 127
}

// funDecls lists every function and method declared in the file.
func funDecls(file *ast.File) []*ast.FunDecl {
	var out []*ast.FunDecl
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FunDecl:
			out = append(out, d)
		case *ast.StructDecl:
			out = append(out, d.Methods...)
		case *ast.ImplDecl:
			out = append(out, d.Methods...)
		case *ast.TraitDecl:
			out = append(out, d.Methods...)
		}
	}
	return out
}

// effectHints places the inferred parts of fn's signature: the return type
// after the parameter list, `suspends` before the body, and the error set
// after a bare `throws`.
func effectHints(f *source.File, fn *ast.FunDecl, inf sema.Inferred) []inlayHint {
	var out []inlayHint
	text := f.Content
	// the body's start: `{` or the `=` before an expression body
	bodyAt := -1
	switch {
	case fn.Body != nil:
		bodyAt = fn.Body.Pos.Start
	case fn.ExprBody != nil:
		i := fn.ExprBody.Span().Start - 1
		for i >= 0 && text[i] != '=' {
			i--
		}
		bodyAt = i
	}
	if bodyAt < 0 {
		return nil
	}
	if inf.Ret != "" {
		// right after `)`: before any effect written, else before the body
		limit := bodyAt
		for _, sp := range []source.Span{fn.Effects.SuspendsSpan, fn.Effects.ThrowsSpan} {
			if sp.IsValid() && sp.Start < limit {
				limit = sp.Start
			}
		}
		at := limit
		for at > 0 && (text[at-1] == ' ' || text[at-1] == '\t') {
			at--
		}
		if at > 0 && text[at-1] == ')' {
			out = append(out, inlayHint{Position: offsetToPosition(f, at), Label: ": " + inf.Ret, Kind: hintType})
		}
	}
	if inf.Suspends {
		at := bodyAt
		if fn.Effects.ThrowsSpan.IsValid() && fn.Effects.ThrowsSpan.Start < at {
			at = fn.Effects.ThrowsSpan.Start // `suspends` is written before `throws`
		}
		out = append(out, inlayHint{Position: offsetToPosition(f, at), Label: "suspends", Kind: hintType, PaddingRight: true})
	}
	if inf.Throws != "" && fn.Effects.ThrowsSpan.IsValid() {
		out = append(out, inlayHint{Position: offsetToPosition(f, fn.Effects.ThrowsSpan.End), Label: inf.Throws, Kind: hintType, PaddingLeft: true})
	}
	return out
}

// closeHints places, after the `}` of every block that holds statement-form
// `with`s (D100), what that brace closes and in which order — the last
// opened first: `closes second, first; cancels server; closes listener`.
func closeHints(f *source.File, file *ast.File, held map[source.Span]bool) []inlayHint {
	var out []inlayHint
	walkNodes(reflect.ValueOf(file), func(n any) {
		b, ok := n.(*ast.Block)
		if !ok || b.Pos.End <= 0 || b.Pos.End > len(f.Content) || f.Content[b.Pos.End-1] != '}' {
			return
		}
		var parts []string
		verb := ""
		for i := len(b.Stmts) - 1; i >= 0; i-- {
			w, ok := b.Stmts[i].(*ast.WithStmt)
			if !ok {
				continue
			}
			v, name := "closes", w.Binding.Name.Name
			if name == "" { // `with sem.acquire()` (D109): the expression
				name = strings.TrimSpace(f.Content[w.Binding.Value.Span().Start:w.Binding.Value.Span().End])
			}
			if c, isCall := w.Binding.Value.(*ast.CallExpr); isCall && c.Async {
				if w.Binding.Name.Name != "" && awaitedAfter(b.Stmts[i+1:], w.Binding.Name.Name) {
					continue // finished by then: the end has nothing to stop
				}
				v = "cancels"
			} else if held[w.Binding.Value.Span()] {
				// D111: the value's tasks are stopped, then it is closed
				v = "stops tasks of, then closes"
			} else if m, isLock := lockCallee(w.Binding.Value); isLock {
				// `with n = notes.lock()` (D107): the close is the unlock
				v, name = "unlocks", strings.TrimSpace(f.Content[m.X.Span().Start:m.X.Span().End])
			}
			if v != verb {
				parts = append(parts, v+" "+name)
				verb = v
			} else {
				parts[len(parts)-1] += ", " + name
			}
		}
		if len(parts) > 0 {
			out = append(out, inlayHint{Position: offsetToPosition(f, b.Pos.End), Label: strings.Join(parts, "; "), Kind: hintType, PaddingLeft: true})
		}
	})
	return out
}

// lockCallee is the `x.lock` of a call `x.lock()`.
func lockCallee(e ast.Expr) (*ast.MemberExpr, bool) {
	c, ok := e.(*ast.CallExpr)
	if !ok || len(c.Args) != 0 {
		return nil, false
	}
	m, ok := c.Fun.(*ast.MemberExpr)
	return m, ok && m.Name.Name == "lock"
}

// walkNodes calls visit on every AST node reachable from v.
func walkNodes(v reflect.Value, visit func(any)) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			walkNodes(v.Elem(), visit)
		}
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.Type().Elem().Kind() == reflect.Struct {
			visit(v.Interface())
		}
		walkNodes(v.Elem(), visit)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(source.Span{}) {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				walkNodes(v.Field(i), visit)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkNodes(v.Index(i), visit)
		}
	}
}

// awaitedAfter reports whether one of stmts certainly awaits the task
// named name: `await name` outside any branch, loop, lambda or nested
// block, where it might not run.
func awaitedAfter(stmts []ast.Stmt, name string) bool {
	found := false
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if found {
			return
		}
		switch v.Kind() {
		case reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem())
			}
		case reflect.Ptr:
			if v.IsNil() {
				return
			}
			switch n := v.Interface().(type) {
			case *ast.LambdaExpr, *ast.IfExpr, *ast.WhenExpr, *ast.LoopStmt, *ast.Block, *ast.BlockExpr, *ast.RaceExpr, *ast.CoalesceExpr:
				return
			case *ast.AwaitExpr:
				if ne, ok := n.X.(*ast.NameExpr); ok && ne.Name == name {
					found = true
					return
				}
			}
			walk(v.Elem())
		case reflect.Struct:
			if v.Type() == reflect.TypeOf(source.Span{}) {
				return
			}
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	for _, s := range stmts {
		walk(reflect.ValueOf(s))
	}
	return found
}
