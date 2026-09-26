package sema

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/types"
)

// Rendering derived code as Veles (D58).
//
// The formatter cannot do this: it is source-guided — it copies literals
// verbatim and reads the author's line breaks out of the original text —
// and a synthesized node has no text of its own (every one of them carries
// the span of the `implement` that asked for it). So this file prints the
// small, closed set of nodes `derive.go` builds, in the same style the
// formatter would produce: two-space indent, braced bodies always broken.
//
// A few forms have no spelling a programmer could write — a field's
// declared default, a resolved type in receiver position. They are printed
// the way the compiler names them, between angle brackets, rather than as
// something that looks like source and is not.

// DumpDerived renders one synthesized method as Veles source, for
// `veles explain --derive`, `VELES_DEBUG_DERIVE=1` and tests.
func DumpDerived(fd *ast.FunDecl) string {
	p := &dprinter{}
	p.fun(fd, 0)
	return strings.TrimRight(p.sb.String(), "\n")
}

// dumpDerivedImpl renders a whole implement — its header and the methods
// the compiler wrote for it; the ones the author wrote are left out.
func dumpDerivedImpl(impl *Impl) string {
	p := &dprinter{}
	p.line(0, implHeader(impl))
	for _, md := range impl.Decl.Methods {
		if !impl.Derived[md.Name.Name] {
			continue
		}
		p.sb.WriteString("\n")
		p.fun(md, 0)
	}
	return strings.TrimRight(p.sb.String(), "\n")
}

// implHeader is the declaration line of an implement as the compiler reads
// it, with the bounds inferred for a generic target spelled out — the
// empty `implement Codable` in `struct Page<T>` is `implement<T: Codable>
// Codable for Page<T>` (D58).
func implHeader(impl *Impl) string {
	head := "implement"
	if len(impl.TypeParams) > 0 {
		var ps []string
		for _, tp := range impl.TypeParams {
			s := tp.Name
			var bounds []string
			for _, b := range tp.Bounds {
				bounds = append(bounds, b.Name)
			}
			if len(bounds) > 0 {
				s += ": " + strings.Join(bounds, " + ")
			}
			ps = append(ps, s)
		}
		head += "<" + strings.Join(ps, ", ") + ">"
	}
	name := "?"
	if impl.Trait != nil {
		name = impl.Trait.Name
	}
	target := ""
	if impl.Target != nil {
		target = impl.Target.String()
	}
	return head + " " + name + " for " + target
}

type dprinter struct {
	sb strings.Builder
}

func (p *dprinter) line(ind int, s string) {
	p.sb.WriteString(strings.Repeat("  ", ind) + s + "\n")
}

func (p *dprinter) fun(fd *ast.FunDecl, ind int) {
	head := ""
	if fd.Static {
		head += "static "
	}
	head += "fun " + fd.Name.Name + "("
	var ps []string
	for _, prm := range fd.Params {
		s := prm.Name.Name
		if prm.Type != nil {
			s += ": " + dtype(prm.Type)
		}
		ps = append(ps, s)
	}
	head += strings.Join(ps, ", ") + ")"
	if fd.Ret != nil {
		head += ": " + dtype(fd.Ret)
	}
	if fd.Effects.Suspends {
		head += " suspends"
	}
	if fd.Effects.Throws {
		head += " throws"
		if fd.Effects.Error != nil {
			head += " " + dtype(fd.Effects.Error)
		}
	}
	if fd.Body == nil {
		p.line(ind, head)
		return
	}
	p.line(ind, head+" {")
	p.stmts(fd.Body.Stmts, ind+1)
	p.line(ind, "}")
}

func (p *dprinter) stmts(list []ast.Stmt, ind int) {
	for _, s := range list {
		p.stmt(s, ind)
	}
}

func (p *dprinter) stmt(s ast.Stmt, ind int) {
	switch s := s.(type) {
	case *ast.ExprStmt:
		p.exprLine(ind, "", s.X, "")
	case *ast.ValStmt:
		kw := "val "
		if s.Kind == ast.BindVar {
			kw = "var "
		}
		head := kw + bindingText(s.Binding)
		if s.Value == nil {
			p.line(ind, head)
			return
		}
		p.exprLine(ind, head+" = ", s.Value, "")
	case *ast.AssignStmt:
		p.exprLine(ind, dexpr(s.Target, ind)+" "+s.Op.String()+" ", s.Value, "")
	case *ast.ReturnStmt:
		if s.Value == nil {
			p.line(ind, "return")
			return
		}
		p.exprLine(ind, "return ", s.Value, "")
	case *ast.ThrowStmt:
		p.exprLine(ind, "throw ", s.Value, "")
	case *ast.BreakStmt:
		p.line(ind, "break"+labelText(s.Label))
	case *ast.ContinueStmt:
		p.line(ind, "continue"+labelText(s.Label))
	case *ast.LoopStmt:
		head := "loop"
		switch {
		case s.Var != nil:
			head += " (" + bindingText(*s.Var) + " in " + dexpr(s.Iter, ind) + ")"
		case s.Cond != nil:
			head += " (" + dexpr(s.Cond, ind) + ")"
		}
		p.line(ind, head+" {")
		p.stmts(s.Body.Stmts, ind+1)
		p.line(ind, "}")
	case *ast.Block:
		p.line(ind, "{")
		p.stmts(s.Stmts, ind+1)
		p.line(ind, "}")
	default:
		p.line(ind, fmt.Sprintf("<%T>", s))
	}
}

// exprLine writes one statement whose text is prefix + the expression +
// suffix. A block-shaped expression (if/when/block) breaks over lines; any
// other is one line.
func (p *dprinter) exprLine(ind int, prefix string, e ast.Expr, suffix string) {
	switch x := e.(type) {
	case *ast.IfExpr:
		p.line(ind, prefix+"if ("+dexpr(x.Cond, ind)+") {")
		p.stmts(x.Then.Stmts, ind+1)
		if x.Else == nil {
			p.line(ind, "}"+suffix)
			return
		}
		// `else if` is an else block holding a single if
		if len(x.Else.Stmts) == 1 {
			if es, ok := x.Else.Stmts[0].(*ast.ExprStmt); ok {
				if _, isIf := es.X.(*ast.IfExpr); isIf {
					p.exprLine(ind, "} else ", es.X, suffix)
					return
				}
			}
		}
		p.line(ind, "} else {")
		p.stmts(x.Else.Stmts, ind+1)
		p.line(ind, "}"+suffix)
	case *ast.WhenExpr:
		head := prefix + "when"
		if x.Subject != nil {
			subj := dexpr(x.Subject, ind)
			if x.Bind != nil {
				subj = "val " + x.Bind.Name + " = " + subj
			}
			head += " (" + subj + ")"
		}
		p.line(ind, head+" {")
		for _, arm := range x.Arms {
			p.arm(arm, ind+1)
		}
		p.line(ind, "}"+suffix)
	case *ast.BlockExpr:
		p.line(ind, prefix+"{")
		p.stmts(x.Block.Stmts, ind+1)
		p.line(ind, "}"+suffix)
	case *ast.ControlExpr:
		p.stmt(x.Stmt, ind)
	default:
		p.line(ind, prefix+dexpr(e, ind)+suffix)
	}
}

func (p *dprinter) arm(a *ast.WhenArm, ind int) {
	head := ""
	switch {
	case a.Else:
		head = "else"
	case a.Cond != nil:
		head = dexpr(a.Cond, ind)
	default:
		var pats []string
		for _, pat := range a.Patterns {
			pats = append(pats, dpattern(pat, ind))
		}
		head = strings.Join(pats, ", ")
	}
	if a.Guard != nil {
		head += " if " + dexpr(a.Guard, ind)
	}
	p.exprLine(ind, head+" => ", a.Body, "")
}

// ---------------------------------------------------------------------------
// expressions

// dexpr renders an expression on one line. ind is the indent of the line it
// starts on, so a nested block-shaped expression can indent its own lines.
func dexpr(e ast.Expr, ind int) string {
	switch e := e.(type) {
	case nil:
		return ""
	case *ast.NameExpr:
		s := e.Name
		if len(e.TypeArgs) > 0 {
			var as []string
			for _, a := range e.TypeArgs {
				as = append(as, dtype(a))
			}
			s += "<" + strings.Join(as, ", ") + ">"
		}
		return s
	case *ast.SelfExpr:
		return "this"
	case *ast.NullLit:
		return "null"
	case *ast.BoolLit:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.IntLit:
		return e.Text
	case *ast.FloatLit:
		return e.Text
	case *ast.CharLit:
		return "'" + e.Value + "'"
	case *ast.StringLit:
		var sb strings.Builder
		sb.WriteString("\"")
		for _, part := range e.Parts {
			if part.Expr != nil {
				sb.WriteString("${" + dexpr(part.Expr, ind) + "}")
				continue
			}
			sb.WriteString(escapeVeles(part.Text))
		}
		sb.WriteString("\"")
		return sb.String()
	case *ast.MemberExpr:
		dot := "."
		if e.Safe {
			dot = "?."
		}
		return dexpr(e.X, ind) + dot + e.Name.Name
	case *ast.CallExpr:
		var as []string
		for _, a := range e.Args {
			s := dexpr(a.Value, ind)
			if a.Name != nil {
				s = a.Name.Name + ": " + s
			}
			if a.Spread {
				s += "..."
			}
			as = append(as, s)
		}
		head := dexpr(e.Fun, ind)
		if len(e.TypeArgs) > 0 {
			var ts []string
			for _, t := range e.TypeArgs {
				ts = append(ts, dtype(t))
			}
			head += "<" + strings.Join(ts, ", ") + ">"
		}
		return head + "(" + strings.Join(as, ", ") + ")"
	case *ast.TryExpr:
		return "try " + dexpr(e.X, ind)
	case *ast.UnaryExpr:
		return e.Op.String() + dexpr(e.X, ind)
	case *ast.BinaryExpr:
		return dopnd(e.L, ind) + " " + e.Op.String() + " " + dopnd(e.R, ind)
	case *ast.ElvisExpr:
		return dopnd(e.L, ind) + " ?: " + dopnd(e.R, ind)
	case *ast.CastExpr:
		return dopnd(e.X, ind) + " as " + dtype(e.Type)
	case *ast.IsExpr:
		op := " is "
		if e.Not {
			op = " !is "
		}
		return dopnd(e.X, ind) + op + dpattern(e.Pat, ind)[len("is "):]
	case *ast.TypeExpr:
		return dtype(e.Type)
	case *ast.PreludeName:
		return e.Name
	case *ast.FieldDefaultExpr:
		return "<default of " + fieldDefaultName(e) + ">"
	case *ast.ControlExpr:
		switch s := e.Stmt.(type) {
		case *ast.ThrowStmt:
			return "throw " + dexpr(s.Value, ind)
		case *ast.BreakStmt:
			return "break" + labelText(s.Label)
		case *ast.ContinueStmt:
			return "continue" + labelText(s.Label)
		case *ast.ReturnStmt:
			if s.Value == nil {
				return "return"
			}
			return "return " + dexpr(s.Value, ind)
		}
		return "<control>"
	case *ast.IfExpr, *ast.WhenExpr, *ast.BlockExpr:
		// a block-shaped expression nested inside a one-line one: print it
		// over lines and splice it in at this indent
		p := &dprinter{}
		p.exprLine(ind, "", e, "")
		lines := strings.Split(strings.TrimRight(p.sb.String(), "\n"), "\n")
		for i := range lines {
			lines[i] = strings.TrimPrefix(lines[i], strings.Repeat("  ", ind))
		}
		return strings.Join(lines, "\n"+strings.Repeat("  ", ind))
	}
	return fmt.Sprintf("<%T>", e)
}

// dopnd is dexpr with parentheses when the operand is itself an operator
// expression, so precedence never has to be guessed.
func dopnd(e ast.Expr, ind int) string {
	switch e.(type) {
	case *ast.BinaryExpr, *ast.ElvisExpr, *ast.CastExpr, *ast.IsExpr:
		return "(" + dexpr(e, ind) + ")"
	}
	return dexpr(e, ind)
}

func dpattern(pat ast.Pattern, ind int) string {
	switch pat := pat.(type) {
	case *ast.WildcardPat:
		return "_"
	case *ast.BindPat:
		return pat.Name.Name
	case *ast.LiteralPat:
		return dexpr(pat.Value, ind)
	case *ast.TypePat:
		s := "is " + dtype(pat.Type)
		if !pat.HasArg {
			return s
		}
		var fs []string
		for _, f := range pat.Fields {
			if f.Pat == nil {
				fs = append(fs, f.Name.Name)
				continue
			}
			fs = append(fs, f.Name.Name+": "+dpattern(f.Pat, ind))
		}
		return s + "(" + strings.Join(fs, ", ") + ")"
	case *ast.TuplePat:
		var es []string
		for _, el := range pat.Elems {
			es = append(es, dpattern(el, ind))
		}
		return "(" + strings.Join(es, ", ") + ")"
	case *ast.ListPat:
		var es []string
		for _, el := range pat.Elems {
			es = append(es, dpattern(el, ind))
		}
		return "[" + strings.Join(es, ", ") + "]"
	case *ast.RestPat:
		if pat.Name != nil {
			return ".." + pat.Name.Name
		}
		return ".."
	}
	return fmt.Sprintf("<%T>", pat)
}

// ---------------------------------------------------------------------------
// leaves

func dtype(t ast.Type) string {
	if t == nil {
		return ""
	}
	if rt, ok := t.(*ast.ResolvedType); ok {
		if tt, ok := rt.T.(types.Type); ok {
			return tt.String()
		}
		return "<type>"
	}
	return ast.TypeString(t)
}

func bindingText(b ast.Binding) string {
	var s string
	switch {
	case b.Name != nil:
		s = b.Name.Name
	case len(b.Tuple) > 0:
		var es []string
		for _, el := range b.Tuple {
			es = append(es, bindingText(el))
		}
		s = "(" + strings.Join(es, ", ") + ")"
	}
	if b.Ref {
		s = "&" + s
	}
	if b.Type != nil {
		s += ": " + dtype(b.Type)
	}
	return s
}

func labelText(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return " :" + id.Name
}

// fieldDefaultName names the field a FieldDefaultExpr stands for.
func fieldDefaultName(e *ast.FieldDefaultExpr) string {
	st, ok := e.Struct.(*types.Struct)
	if !ok || e.Index < 0 || e.Index >= len(st.Fields) {
		return fmt.Sprintf("field %d", e.Index)
	}
	return st.Name + "." + st.Fields[e.Index].Name
}

func escapeVeles(s string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\t", "\\t",
		"\r", "\\r",
		"$", "\\$",
	)
	return r.Replace(s)
}

// unused guards against the lexer import being dropped by a future edit:
// operator spellings come from lexer.TokenKind.String().
var _ = lexer.Assign
