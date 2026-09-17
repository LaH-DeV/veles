package ast

import (
	"fmt"
	"strings"
)

// Dump renders a file as an indented S-expression tree, used by
// `veles parse` and by the parser tests.
func Dump(f *File) string {
	p := &printer{}
	for _, d := range f.Decls {
		p.decl(d)
		p.nl()
	}
	return p.sb.String()
}

// DumpExpr renders a single expression; the parser tests compare against it.
func DumpExpr(e Expr) string {
	p := &printer{}
	p.expr(e)
	return p.sb.String()
}

func TypeString(t Type) string {
	p := &printer{}
	p.typ(t)
	return p.sb.String()
}

type printer struct {
	sb     strings.Builder
	indent int
}

func (p *printer) w(s string)                { p.sb.WriteString(s) }
func (p *printer) f(format string, a ...any) { fmt.Fprintf(&p.sb, format, a...) }
func (p *printer) nl()                       { p.sb.WriteString("\n" + strings.Repeat("  ", p.indent)) }
func (p *printer) open(name string)          { p.w("(" + name); p.indent++ }
func (p *printer) close()                    { p.indent--; p.w(")") }
func (p *printer) child(f func())            { p.nl(); f() }

func (p *printer) attrs(attrs []*Attribute) {
	for _, a := range attrs {
		p.f("@%s", a.Name.Name)
		if len(a.Args) > 0 {
			p.w("(")
			p.args(a.Args)
			p.w(")")
		}
		p.w(" ")
	}
}

func (p *printer) typeParams(tps []TypeParam) {
	if len(tps) == 0 {
		return
	}
	p.w("<")
	for i, tp := range tps {
		if i > 0 {
			p.w(", ")
		}
		p.w(tp.Name.Name)
		for j, b := range tp.Bounds {
			if j == 0 {
				p.w(": ")
			} else {
				p.w(" + ")
			}
			p.typ(b)
		}
	}
	p.w(">")
}

func (p *printer) effects(e Effects) {
	if e.Suspends {
		p.w(" suspends")
	}
	if e.Throws {
		p.w(" throws")
		if e.Error != nil {
			p.w(" ")
			p.typ(e.Error)
		}
	}
}

func (p *printer) decl(d Decl) {
	switch d := d.(type) {
	case *UseDecl:
		p.w("(use ")
		p.path(d.Path)
		if d.Alias != nil {
			p.w(" as " + d.Alias.Name)
		}
		if d.Items != nil {
			p.w(" {")
			for i, it := range d.Items {
				if i > 0 {
					p.w(", ")
				}
				p.w(it.Name.Name)
				if it.Alias != nil {
					p.w(" as " + it.Alias.Name)
				}
			}
			p.w("}")
		}
		p.w(")")
	case *FunDecl:
		p.fun(d)
	case *ErrorAliasDecl:
		p.attrs(d.Attrs)
		p.open("error ")
		if d.Pub {
			p.w("pub ")
		}
		p.w(d.Name.Name + " = ")
		p.typ(d.Members)
		p.close()
	case *StructDecl:
		p.attrs(d.Attrs)
		if d.Error {
			p.open("error ")
		} else {
			p.open("struct ")
		}
		if d.Pub {
			p.w("pub ")
		}
		if d.Extern {
			p.w("extern ")
		}
		p.w(d.Name.Name)
		p.typeParams(d.TypeParams)
		if d.Variant != nil {
			p.w(" : ")
			p.typ(d.Variant)
		}
		for _, fld := range d.Fields {
			p.child(func() {
				p.w("(field ")
				if fld.Pub {
					p.w("pub ")
				}
				p.w(fld.Name.Name + ": ")
				p.typ(fld.Type)
				if fld.Default != nil {
					p.w(" = ")
					p.expr(fld.Default)
				}
				p.w(")")
			})
		}
		for _, m := range d.Methods {
			p.child(func() { p.fun(m) })
		}
		p.close()
	case *TraitDecl:
		p.attrs(d.Attrs)
		p.open("trait ")
		if d.Pub {
			p.w("pub ")
		}
		if d.Sealed {
			p.w("sealed ")
		}
		p.w(d.Name.Name)
		p.typeParams(d.TypeParams)
		for i, s := range d.Supers {
			if i == 0 {
				p.w(" : ")
			} else {
				p.w(" + ")
			}
			p.typ(s)
		}
		for _, at := range d.AssocTypes {
			p.child(func() {
				p.w("(type " + at.Name.Name)
				for i, b := range at.Bounds {
					if i == 0 {
						p.w(": ")
					} else {
						p.w(" + ")
					}
					p.typ(b)
				}
				p.w(")")
			})
		}
		for _, m := range d.Methods {
			p.child(func() { p.fun(m) })
		}
		p.close()
	case *ImplDecl:
		p.attrs(d.Attrs)
		if d.Extend {
			p.open("extend")
			p.typeParams(d.TypeParams)
			p.w(" ")
			p.typ(d.Target)
		} else {
			p.open("impl")
			p.typeParams(d.TypeParams)
			p.w(" ")
			p.typ(d.Trait)
			p.w(" for ")
			p.typ(d.Target)
		}
		for _, at := range d.AssocTypes {
			p.child(func() {
				p.w("(type " + at.Name.Name + " = ")
				p.typ(at.Type)
				p.w(")")
			})
		}
		for _, m := range d.Methods {
			p.child(func() { p.fun(m) })
		}
		p.close()
	case *ValDecl:
		p.attrs(d.Attrs)
		p.w("(")
		if d.Pub {
			p.w("pub ")
		}
		p.w(d.Kind.String() + " " + d.Name.Name)
		if d.Type != nil {
			p.w(": ")
			p.typ(d.Type)
		}
		if d.Value != nil {
			p.w(" = ")
			p.expr(d.Value)
		}
		p.w(")")
	case *ExternBlock:
		p.open(fmt.Sprintf("extern %q", d.ABI))
		for _, fn := range d.Funs {
			p.child(func() { p.fun(fn) })
		}
		p.close()
	case *BadDecl:
		p.w("(bad-decl)")
	default:
		p.f("(?decl %T)", d)
	}
}

func (p *printer) fun(d *FunDecl) {
	p.attrs(d.Attrs)
	p.open("fun ")
	if d.Pub {
		p.w("pub ")
	}
	if d.Static {
		p.w("static ")
	}
	if d.Override {
		p.w("override ")
	}
	if d.Unsafe {
		p.w("unsafe ")
	}
	if d.Mut {
		p.w("mut ")
	}
	p.w(d.Name.Name)
	p.typeParams(d.TypeParams)
	p.w("(")
	for i, prm := range d.Params {
		if i > 0 {
			p.w(", ")
		}
		p.w(prm.Name.Name)
		if prm.Type != nil {
			p.w(": ")
			p.typ(prm.Type)
		}
		if prm.Default != nil {
			p.w(" = ")
			p.expr(prm.Default)
		}
	}
	p.w(")")
	if d.Ret != nil {
		p.w(": ")
		p.typ(d.Ret)
	}
	p.effects(d.Effects)
	if d.ExprBody != nil {
		p.child(func() { p.w("= "); p.expr(d.ExprBody) })
	} else if d.Body != nil {
		p.child(func() { p.block(d.Body) })
	}
	p.close()
}

func (p *printer) path(path []Ident) {
	for i, id := range path {
		if i > 0 {
			p.w(".")
		}
		p.w(id.Name)
	}
}

func (p *printer) typ(t Type) {
	switch t := t.(type) {
	case nil:
		p.w("()")
	case *NamedType:
		p.path(t.Path)
		if len(t.Args) > 0 {
			p.w("<")
			for i, a := range t.Args {
				if i > 0 {
					p.w(", ")
				}
				p.typ(a)
			}
			p.w(">")
		}
	case *NullableType:
		if _, ptr := t.Elem.(*PointerType); ptr {
			p.w("(")
			p.typ(t.Elem)
			p.w(")")
		} else {
			p.typ(t.Elem)
		}
		p.w("?")
	case *PointerType:
		p.w("*")
		if t.Raw {
			p.w("raw ")
		}
		p.typ(t.Elem)
	case *TupleType:
		p.w("(")
		for i, e := range t.Elems {
			if i > 0 {
				p.w(", ")
			}
			p.typ(e)
		}
		p.w(")")
	case *FunType:
		p.w("fun(")
		for i, e := range t.Params {
			if i > 0 {
				p.w(", ")
			}
			p.typ(e)
		}
		p.w(")")
		if t.Ret != nil {
			p.w(": ")
			p.typ(t.Ret)
		}
		p.effects(t.Effects)
	case *SelfType:
		p.w("Self")
	case *AssocType:
		p.typ(t.Base)
		p.w("::" + t.Name.Name)
	case *ErrorUnionType:
		for i, m := range t.Members {
			if i > 0 {
				p.w(" | ")
			}
			p.typ(m)
		}
	default:
		p.f("?type(%T)", t)
	}
}

func (p *printer) block(b *Block) {
	p.open("block")
	for _, s := range b.Stmts {
		p.child(func() { p.stmt(s) })
	}
	p.close()
}

func (p *printer) binding(b Binding) {
	if b.Name != nil {
		p.w(b.Name.Name)
		if b.Type != nil {
			p.w(": ")
			p.typ(b.Type)
		}
		return
	}
	p.w("(")
	for i, e := range b.Tuple {
		if i > 0 {
			p.w(", ")
		}
		p.binding(e)
	}
	p.w(")")
}

func (p *printer) stmt(s Stmt) {
	switch s := s.(type) {
	case *Block:
		p.block(s)
	case *ValStmt:
		p.w("(" + s.Kind.String() + " ")
		p.binding(s.Binding)
		if s.Value != nil {
			p.w(" = ")
			p.expr(s.Value)
		}
		p.w(")")
	case *ExprStmt:
		p.expr(s.X)
	case *AssignStmt:
		p.w("(" + s.Op.String() + " ")
		p.expr(s.Target)
		p.w(" ")
		p.expr(s.Value)
		p.w(")")
	case *ReturnStmt:
		p.w("(return")
		if s.Value != nil {
			p.w(" ")
			p.expr(s.Value)
		}
		p.w(")")
	case *ThrowStmt:
		p.w("(throw ")
		p.expr(s.Value)
		p.w(")")
	case *BreakStmt:
		p.w("(break")
		if s.Label != nil {
			p.w(" " + s.Label.Name)
		}
		p.w(")")
	case *ContinueStmt:
		p.w("(continue")
		if s.Label != nil {
			p.w(" " + s.Label.Name)
		}
		p.w(")")
	case *LoopStmt:
		p.open("loop")
		if s.Label != nil {
			p.w(" @" + s.Label.Name)
		}
		if s.Var != nil {
			p.w(" (")
			p.binding(*s.Var)
			p.w(" in ")
			p.expr(s.Iter)
			p.w(")")
		} else if s.Cond != nil {
			p.w(" (")
			p.expr(s.Cond)
			p.w(")")
		}
		p.child(func() { p.block(s.Body) })
		p.close()
	case *WithStmt:
		p.open("with")
		for _, b := range s.Bindings {
			p.w(" (" + b.Name.Name + " = ")
			p.expr(b.Value)
			p.w(")")
		}
		p.child(func() { p.block(s.Body) })
		p.close()
	case *ScopeStmt:
		p.open("scope")
		p.child(func() { p.block(s.Body) })
		p.close()
	case *FunStmt:
		p.fun(s.Fun)
	case *BadStmt:
		p.w("(bad-stmt)")
	default:
		p.f("(?stmt %T)", s)
	}
}

func (p *printer) args(args []Arg) {
	for i, a := range args {
		if i > 0 {
			p.w(", ")
		}
		if a.Name != nil {
			p.w(a.Name.Name + ": ")
		}
		p.expr(a.Value)
	}
}

func (p *printer) expr(e Expr) {
	switch e := e.(type) {
	case nil:
		p.w("<nil>")
	case *IntLit:
		p.w(e.Text)
	case *FloatLit:
		p.w(e.Text)
	case *StringLit:
		p.w("(str")
		for _, part := range e.Parts {
			if part.Expr != nil {
				p.w(" ${")
				p.expr(part.Expr)
				p.w("}")
			} else {
				p.f(" %q", part.Text)
			}
		}
		p.w(")")
	case *CharLit:
		p.f("'%s'", e.Value)
	case *BoolLit:
		p.f("%v", e.Value)
	case *NullLit:
		p.w("null")
	case *SelfExpr:
		p.w("self")
	case *NameExpr:
		p.w(e.Name)
		if len(e.TypeArgs) > 0 {
			p.w("<")
			for i, t := range e.TypeArgs {
				if i > 0 {
					p.w(", ")
				}
				p.typ(t)
			}
			p.w(">")
		}
	case *MemberExpr:
		p.w("(")
		if e.Safe {
			p.w("?. ")
		} else {
			p.w(". ")
		}
		p.expr(e.X)
		p.w(" " + e.Name.Name + ")")
	case *IndexExpr:
		p.w("(index ")
		p.expr(e.X)
		p.w(" ")
		p.expr(e.Index)
		p.w(")")
	case *CallExpr:
		if e.Async {
			p.w("(async-call ")
		} else {
			p.w("(call ")
		}
		p.expr(e.Fun)
		if len(e.TypeArgs) > 0 {
			p.w("<")
			for i, t := range e.TypeArgs {
				if i > 0 {
					p.w(", ")
				}
				p.typ(t)
			}
			p.w(">")
		}
		if len(e.Args) > 0 {
			p.w(" ")
			p.args(e.Args)
		}
		p.w(")")
	case *UnaryExpr:
		p.w("(" + e.Op.String() + " ")
		p.expr(e.X)
		p.w(")")
	case *BinaryExpr:
		p.w("(" + e.Op.String() + " ")
		p.expr(e.L)
		p.w(" ")
		p.expr(e.R)
		p.w(")")
	case *ElvisExpr:
		p.w("(?: ")
		p.expr(e.L)
		p.w(" ")
		p.expr(e.R)
		p.w(")")
	case *RangeExpr:
		if e.Inclusive {
			p.w("(.. ")
		} else {
			p.w("(..< ")
		}
		p.expr(e.Lo)
		p.w(" ")
		p.expr(e.Hi)
		p.w(")")
	case *LambdaExpr:
		p.w("(lambda (")
		for i, prm := range e.Params {
			if i > 0 {
				p.w(", ")
			}
			p.w(prm.Name.Name)
			if prm.Type != nil {
				p.w(": ")
				p.typ(prm.Type)
			}
		}
		p.w(")")
		if e.Ret != nil {
			p.w(": ")
			p.typ(e.Ret)
		}
		p.w(" ")
		p.expr(e.Body)
		p.w(")")
	case *TupleExpr:
		p.w("(tuple")
		for _, el := range e.Elems {
			p.w(" ")
			p.expr(el)
		}
		p.w(")")
	case *ListLit:
		if e.Mut {
			p.w("(mut-list")
		} else {
			p.w("(list")
		}
		for _, el := range e.Elems {
			p.w(" ")
			p.expr(el)
		}
		p.w(")")
	case *MapLit:
		if e.Mut {
			p.w("(mut-map")
		} else {
			p.w("(map")
		}
		for _, en := range e.Entries {
			p.w(" [")
			p.expr(en.Key)
			p.w(": ")
			p.expr(en.Value)
			p.w("]")
		}
		p.w(")")
	case *IfExpr:
		p.open("if ")
		p.expr(e.Cond)
		p.child(func() { p.block(e.Then) })
		if e.Else != nil {
			p.child(func() { p.w("else "); p.block(e.Else) })
		}
		p.close()
	case *WhenExpr:
		p.open("when")
		if e.Subject != nil {
			p.w(" ")
			p.expr(e.Subject)
		}
		for _, arm := range e.Arms {
			p.child(func() {
				p.w("(")
				switch {
				case arm.Else:
					p.w("else")
				case arm.Cond != nil:
					p.expr(arm.Cond)
				default:
					for i, pat := range arm.Patterns {
						if i > 0 {
							p.w(", ")
						}
						p.pattern(pat)
					}
				}
				if arm.Guard != nil {
					p.w(" if ")
					p.expr(arm.Guard)
				}
				p.w(" => ")
				p.expr(arm.Body)
				p.w(")")
			})
		}
		p.close()
	case *BlockExpr:
		p.block(e.Block)
	case *TryExpr:
		p.w("(try ")
		p.expr(e.X)
		p.w(")")
	case *AwaitExpr:
		p.w("(await ")
		p.expr(e.X)
		p.w(")")
	case *IsExpr:
		if e.Not {
			p.w("(!is ")
		} else {
			p.w("(is ")
		}
		p.expr(e.X)
		p.w(" ")
		p.typ(e.Pat.Type)
		if e.Pat.HasArg {
			p.w("(...)")
		}
		p.w(")")
	case *CastExpr:
		p.w("(as ")
		p.expr(e.X)
		p.w(" ")
		p.typ(e.Type)
		p.w(")")
	case *GatherExpr:
		p.open("gather")
		p.child(func() { p.block(e.Body) })
		p.close()
	case *RaceExpr:
		p.open("race")
		for _, arm := range e.Arms {
			p.child(func() {
				p.w("(")
				if arm.Binding != nil {
					p.w("val ")
					p.binding(*arm.Binding)
					p.w(" = ")
				}
				p.expr(arm.Source)
				p.w(" => ")
				p.expr(arm.Body)
				p.w(")")
			})
		}
		p.close()
	case *UnsafeExpr:
		p.open("unsafe")
		p.child(func() { p.block(e.Body) })
		p.close()
	case *ControlExpr:
		p.stmt(e.Stmt)
	case *BadExpr:
		p.w("(bad-expr)")
	default:
		p.f("(?expr %T)", e)
	}
}

func (p *printer) pattern(pat Pattern) {
	switch pat := pat.(type) {
	case *WildcardPat:
		p.w("_")
	case *BindPat:
		p.w(pat.Name.Name)
	case *LiteralPat:
		p.expr(pat.Value)
	case *RangePat:
		p.w("in ")
		p.expr(pat.Range)
	case *TypePat:
		p.w("is ")
		p.typ(pat.Type)
		if pat.HasArg {
			p.w("(")
			for i, fp := range pat.Fields {
				if i > 0 {
					p.w(", ")
				}
				p.w(fp.Name.Name)
				if fp.Pat != nil {
					p.w(": ")
					p.pattern(fp.Pat)
				}
			}
			p.w(")")
		}
	case *TuplePat:
		p.w("(")
		for i, el := range pat.Elems {
			if i > 0 {
				p.w(", ")
			}
			p.pattern(el)
		}
		p.w(")")
	default:
		p.f("?pat(%T)", pat)
	}
}
