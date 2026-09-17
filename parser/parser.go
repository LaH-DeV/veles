// Package parser builds an ast.File from tokens. It is a hand-written
// recursive-descent parser with Pratt-style expression parsing. Errors are
// recorded as diagnostics and the parser resynchronises at the next
// statement or declaration so that one mistake yields one message.
package parser

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

type Parser struct {
	file  *source.File
	toks  []lexer.Token
	pos   int
	diags *source.Diagnostics

	// noStructLit-style flag: while parsing an `if`/`when`/`loop` header we
	// are inside parentheses so this is not needed; kept for lambdas.
	lastErrPos int

	// errorFields is set while parsing an `error` body: a field may then
	// have a union type, the one place outside `throws` where a union
	// appears (a cause, D45).
	errorFields bool

	// leadDoc is a documentation comment seen on a declaration's attributes,
	// handed to the declaration that follows them.
	leadDoc string

	// noLambda is set while parsing the head of a `when` or `race` arm (a
	// guard, a condition, a source), where the `=>` that follows belongs to
	// the arm: `x > limit => ...` must not read `limit => ...` as a lambda.
	// Brackets reset it, so a lambda argument inside the head still works.
	noLambda int

	// parens records every parenthesised expression as the span of the
	// expression inside the parentheses; the tree itself has no node for
	// them, and the formatter keeps the ones the author wrote.
	parens map[[2]int]bool
}

// ParseFile parses one source file.
func ParseFile(file *source.File, diags *source.Diagnostics) *ast.File {
	toks, doc := lexer.TokenizeFile(file, diags)
	p := &Parser{file: file, toks: toks, diags: diags, lastErrPos: -1}
	f := p.parseFile()
	f.Doc = doc
	return f
}

// Layout is what the tree does not record but a formatter must keep: the
// comments, and which expressions the author parenthesised (as the spans
// of the expressions inside the parentheses).
type Layout struct {
	Comments []lexer.Comment
	Parens   map[[2]int]bool
}

// ParseFileLayout is ParseFile plus the file's Layout, for tools that
// reproduce the source (the formatter).
func ParseFileLayout(file *source.File, diags *source.Diagnostics) (*ast.File, *Layout) {
	toks, doc, comments := lexer.TokenizeAll(file, diags)
	p := &Parser{file: file, toks: toks, diags: diags, lastErrPos: -1, parens: map[[2]int]bool{}}
	f := p.parseFile()
	f.Doc = doc
	return f, &Layout{Comments: comments, Parens: p.parens}
}

// ParseExprString parses a standalone expression (used by tests).
func ParseExprString(src string, diags *source.Diagnostics) ast.Expr {
	file := source.NewFile("<expr>", src)
	toks := lexer.Tokenize(file, diags)
	p := &Parser{file: file, toks: toks, diags: diags, lastErrPos: -1}
	e := p.parseExpr()
	p.skipSemis()
	if !p.at(lexer.EOF) {
		p.errorExpected("end of expression")
	}
	return e
}

// ---------------------------------------------------------------------------
// token helpers

func (p *Parser) cur() lexer.Token { return p.toks[p.pos] }

func (p *Parser) peek(n int) lexer.Token {
	if p.pos+n < len(p.toks) {
		return p.toks[p.pos+n]
	}
	return p.toks[len(p.toks)-1]
}

func (p *Parser) at(kinds ...lexer.TokenKind) bool {
	k := p.cur().Kind
	for _, want := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

func (p *Parser) next() lexer.Token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *Parser) accept(kind lexer.TokenKind) bool {
	if p.at(kind) {
		p.next()
		return true
	}
	return false
}

func (p *Parser) span() source.Span { return p.cur().Span }

func (p *Parser) prevSpan() source.Span {
	if p.pos == 0 {
		return p.cur().Span
	}
	return p.toks[p.pos-1].Span
}

// spanFrom returns a span from start to the end of the previous token.
func (p *Parser) spanFrom(start source.Span) source.Span {
	return start.To(p.prevSpan())
}

func (p *Parser) errorf(sp source.Span, format string, args ...any) {
	// Suppress cascades: one error per token position.
	if sp.Start == p.lastErrPos {
		return
	}
	p.lastErrPos = sp.Start
	p.diags.Errorf(sp, format, args...)
}

func (p *Parser) errorExpected(what string) {
	p.errorf(p.span(), "expected %s, found %s", what, p.cur().Describe())
}

func (p *Parser) expect(kind lexer.TokenKind) (lexer.Token, bool) {
	if p.at(kind) {
		return p.next(), true
	}
	p.errorExpected(fmt.Sprintf("'%s'", kind))
	return p.cur(), false
}

func (p *Parser) expectIdent() (ast.Ident, bool) {
	if p.at(lexer.Ident) {
		t := p.next()
		return ast.Ident{Name: t.Text, Pos: t.Span}, true
	}
	p.errorExpected("identifier")
	return ast.Ident{Name: "_", Pos: p.span()}, false
}

func (p *Parser) skipSemis() {
	for p.at(lexer.Semi) {
		p.next()
	}
}

// expectTerminator consumes the end of a statement or member: a `;`, or
// nothing when the next token closes the enclosing block.
func (p *Parser) expectTerminator() {
	if p.at(lexer.Semi) {
		p.next()
		return
	}
	if p.at(lexer.RBrace, lexer.EOF) {
		return
	}
	if p.at(lexer.Arrow) {
		p.errorf(p.span(), "'->' is not an operator; use '=>' (D33)")
		p.syncStmt()
		return
	}
	p.errorf(p.span(), "expected end of statement, found %s", p.cur().Describe())
	p.syncStmt()
}

// syncStmt skips to the next statement boundary, honouring nesting.
func (p *Parser) syncStmt() {
	depth := 0
	for !p.at(lexer.EOF) {
		switch p.cur().Kind {
		case lexer.LBrace, lexer.LParen, lexer.LBracket:
			depth++
		case lexer.RBrace, lexer.RParen, lexer.RBracket:
			if depth == 0 {
				return
			}
			depth--
		case lexer.Semi:
			if depth == 0 {
				p.next()
				return
			}
		}
		p.next()
	}
}

// syncDecl skips to the next top-level declaration.
func (p *Parser) syncDecl() {
	depth := 0
	for !p.at(lexer.EOF) {
		k := p.cur().Kind
		if depth == 0 {
			switch k {
			case lexer.KwFun, lexer.KwStruct, lexer.KwTrait, lexer.KwImpl, lexer.KwSealed,
				lexer.KwPub, lexer.KwUse, lexer.KwExtern, lexer.KwVal, lexer.KwVar, lexer.KwConst, lexer.At:
				return
			}
			if p.atErrorDecl() || p.atExtendDecl() {
				return
			}
		}
		switch k {
		case lexer.LBrace:
			depth++
		case lexer.RBrace:
			if depth > 0 {
				depth--
			}
		}
		p.next()
	}
}

// ---------------------------------------------------------------------------
// file & declarations

func (p *Parser) parseFile() *ast.File {
	f := &ast.File{Source: p.file}
	for {
		p.skipSemis()
		if p.at(lexer.EOF) {
			break
		}
		start := p.pos
		d := p.parseDecl()
		if d != nil {
			f.Decls = append(f.Decls, d)
			if sd, ok := d.(*ast.StructDecl); ok {
				if sd.ErrorImpl != nil {
					f.Decls = append(f.Decls, sd.ErrorImpl) // `error` desugars to struct + impl
				}
				for _, impl := range sd.Impls {
					f.Decls = append(f.Decls, impl) // `impl Trait { }` in the body is an impl for the struct
				}
			}
		}
		if _, bad := d.(*ast.BadDecl); bad || p.pos == start {
			if p.pos == start {
				p.next()
			}
			p.syncDecl()
			continue
		}
		switch {
		case p.at(lexer.Semi):
			p.next()
		case p.at(lexer.EOF):
		case p.startsDecl():
			// Missing terminator (typically an unbalanced paren suppressed
			// newline insertion); report but keep the next declaration.
			p.errorf(p.span(), "expected newline before %s", p.cur().Describe())
		default:
			p.expectTerminator()
		}
	}
	return f
}

func (p *Parser) startsDecl() bool {
	return p.atErrorDecl() || p.atExtendDecl() || p.at(lexer.KwFun, lexer.KwStruct, lexer.KwTrait, lexer.KwImpl, lexer.KwSealed,
		lexer.KwPub, lexer.KwUse, lexer.KwExtern, lexer.KwVal, lexer.KwVar, lexer.KwConst, lexer.At)
}

func (p *Parser) parseAttributes() []*ast.Attribute {
	var attrs []*ast.Attribute
	if p.at(lexer.At) {
		p.leadDoc = p.cur().Doc
	}
	for p.at(lexer.At) {
		start := p.span()
		p.next()
		name, _ := p.expectIdent()
		attr := &ast.Attribute{Name: name}
		if p.at(lexer.LParen) {
			attr.Args = p.parseArgs()
		}
		attr.Pos = p.spanFrom(start)
		attrs = append(attrs, attr)
		p.skipSemis()
	}
	return attrs
}

func (p *Parser) parseDecl() ast.Decl {
	attrs := p.parseAttributes()
	doc := p.takeDoc()
	return withDoc(p.parseDeclBody(attrs), doc)
}

// withDoc attaches a documentation comment to the declaration it precedes.
func withDoc(d ast.Decl, doc string) ast.Decl {
	if doc == "" {
		return d
	}
	switch d := d.(type) {
	case *ast.FunDecl:
		d.Doc = doc
	case *ast.StructDecl:
		d.Doc = doc
	case *ast.TraitDecl:
		d.Doc = doc
	case *ast.ValDecl:
		d.Doc = doc
	case *ast.ErrorAliasDecl:
		d.Doc = doc
	}
	return d
}

// takeDoc returns the documentation comment written above the declaration
// starting at the cursor: on its attributes, or on its first token.
func (p *Parser) takeDoc() string {
	doc := p.leadDoc
	p.leadDoc = ""
	if doc == "" {
		doc = p.cur().Doc
	}
	return doc
}

func (p *Parser) parseDeclBody(attrs []*ast.Attribute) ast.Decl {
	start := p.span()
	pub := p.accept(lexer.KwPub)

	switch p.cur().Kind {
	case lexer.KwUse:
		if pub {
			p.errorf(start, "'use' cannot be 'pub'")
		}
		return p.parseUse()
	case lexer.KwFun, lexer.KwUnsafe, lexer.KwStatic:
		fn := p.parseFun(attrs, funContextFree)
		fn.Pub = pub
		fn.Pos = start.To(fn.Pos)
		return fn
	case lexer.KwStruct:
		return p.parseStruct(attrs, pub, false, start)
	case lexer.Ident:
		if p.atErrorDecl() {
			return p.parseErrorDecl(attrs, pub, start)
		}
		if p.atExtendDecl() {
			if pub {
				p.errorf(start, "'extend' cannot be 'pub'; mark the methods instead")
			}
			return p.parseImpl(attrs, true)
		}
	case lexer.KwSealed, lexer.KwTrait:
		return p.parseTrait(attrs, pub, start)
	case lexer.KwImpl:
		if pub {
			p.errorf(start, "'impl' cannot be 'pub'; visibility follows the trait and type")
		}
		return p.parseImpl(attrs, false)
	case lexer.KwVal, lexer.KwVar, lexer.KwConst:
		return p.parseValDecl(attrs, pub, start)
	case lexer.KwExtern:
		if p.peek(1).Kind == lexer.KwStruct {
			p.next()
			return p.parseStruct(attrs, pub, true, start)
		}
		return p.parseExternBlock()
	}
	p.errorf(p.span(), "expected a declaration, found %s", p.cur().Describe())
	return &ast.BadDecl{Pos: p.span()}
}

func (p *Parser) parseUse() ast.Decl {
	start := p.span()
	p.next() // use
	d := &ast.UseDecl{}
	for {
		if p.at(lexer.LBrace) {
			p.next()
			d.Items = []ast.UseItem{}
			for !p.at(lexer.RBrace, lexer.EOF) {
				p.skipSemis()
				name, ok := p.expectIdent()
				if !ok {
					p.syncStmt()
					break
				}
				it := ast.UseItem{Name: name}
				if p.accept(lexer.KwAs) {
					alias, _ := p.expectIdent()
					it.Alias = &alias
				}
				d.Items = append(d.Items, it)
				p.skipSemis()
				if !p.accept(lexer.Comma) {
					break
				}
				p.skipSemis()
			}
			p.expect(lexer.RBrace)
			break
		}
		seg, ok := p.expectIdent()
		if !ok {
			break
		}
		d.Path = append(d.Path, seg)
		if !p.accept(lexer.Dot) {
			break
		}
	}
	if len(d.Path) == 0 {
		p.errorf(start, "'use' needs a module path")
	}
	if p.accept(lexer.KwAs) {
		alias, _ := p.expectIdent()
		d.Alias = &alias
	}
	d.Pos = p.spanFrom(start)
	return d
}

type funContext int

const (
	funContextFree   funContext = iota // module level
	funContextMethod                   // inside struct / impl
	funContextTrait                    // inside trait (body optional)
	funContextExtern                   // inside extern block (no body)
)

func (p *Parser) parseTypeParams() []ast.TypeParam {
	if !p.at(lexer.Lt) {
		return nil
	}
	p.next()
	var tps []ast.TypeParam
	for !p.at(lexer.Gt, lexer.EOF) {
		name, ok := p.expectIdent()
		if !ok {
			break
		}
		tp := ast.TypeParam{Name: name}
		if p.accept(lexer.Colon) {
			tp.Bounds = append(tp.Bounds, p.parseType())
			for p.accept(lexer.Plus) {
				tp.Bounds = append(tp.Bounds, p.parseType())
			}
		}
		tps = append(tps, tp)
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.Gt)
	return tps
}

func (p *Parser) parseParams() []ast.Param {
	if _, ok := p.expect(lexer.LParen); !ok {
		return nil
	}
	var params []ast.Param
	for !p.at(lexer.RParen, lexer.EOF) {
		start := p.span()
		name, ok := p.expectIdent()
		if !ok {
			p.syncParen()
			break
		}
		prm := ast.Param{Name: name}
		if _, ok := p.expect(lexer.Colon); ok {
			prm.Type = p.parseType()
		}
		if p.accept(lexer.Assign) {
			prm.Default = p.parseExpr()
		}
		prm.Pos = p.spanFrom(start)
		params = append(params, prm)
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.RParen)
	return params
}

// syncParen skips to the closing paren of the current list, or to a `{` or
// statement end that shows the list was never closed.
func (p *Parser) syncParen() {
	depth := 0
	for !p.at(lexer.EOF) {
		switch p.cur().Kind {
		case lexer.LParen:
			depth++
		case lexer.RParen:
			if depth == 0 {
				return
			}
			depth--
		case lexer.LBrace, lexer.Semi:
			if depth == 0 {
				return
			}
		}
		p.next()
	}
}

func (p *Parser) parseEffects() ast.Effects {
	var eff ast.Effects
	if p.at(lexer.KwSuspends) {
		eff.Suspends = true
		eff.SuspendsSpan = p.next().Span
	}
	if p.at(lexer.KwThrows) {
		eff.Throws = true
		eff.ThrowsSpan = p.next().Span
		if p.at(lexer.Ident, lexer.KwSelfTy) {
			eff.Error = p.parseErrorType()
		}
	}
	if p.at(lexer.KwSuspends) {
		p.errorf(p.span(), "'suspends' must come before 'throws'")
		p.next()
		eff.Suspends = true
	}
	return eff
}

func (p *Parser) parseFun(attrs []*ast.Attribute, ctx funContext) *ast.FunDecl {
	start := p.span()
	fn := &ast.FunDecl{Attrs: attrs, Doc: p.takeDoc()}
	// modifiers
	for {
		switch p.cur().Kind {
		case lexer.KwPub:
			fn.Pub = true
		case lexer.KwMut:
			fn.Mut = true
		case lexer.KwOverride:
			fn.Override = true
		case lexer.KwStatic:
			fn.Static = true
			if ctx == funContextFree || ctx == funContextExtern {
				p.errorf(p.span(), "'static' belongs to a function in a struct, trait, impl or extend body; a top-level function needs no marker")
			}
		case lexer.KwUnsafe:
			fn.Unsafe = true
		default:
			goto done
		}
		p.next()
	}
done:
	if _, ok := p.expect(lexer.KwFun); !ok {
		fn.Name = ast.Ident{Name: "_", Pos: p.span()}
		fn.Pos = p.spanFrom(start)
		return fn
	}
	fn.TypeParams = p.parseTypeParams()
	fn.Name, _ = p.expectIdent()
	if p.at(lexer.Lt) {
		if fn.TypeParams != nil {
			p.errorf(p.span(), "type parameters were already given before the function name")
		}
		fn.TypeParams = p.parseTypeParams()
	}
	fn.Params = p.parseParams()
	if p.accept(lexer.Colon) {
		fn.Ret = p.parseType()
	}
	fn.Effects = p.parseEffects()

	switch {
	case p.at(lexer.LBrace):
		fn.Body = p.parseBlock()
	case p.at(lexer.Assign):
		p.next()
		fn.ExprBody = p.parseExpr()
	default:
		if ctx == funContextFree || ctx == funContextMethod {
			p.errorf(p.span(), "function '%s' needs a body: '{ ... }' or '= expr'", fn.Name.Name)
		}
	}
	if ctx == funContextExtern && (fn.Body != nil || fn.ExprBody != nil) {
		p.errorf(fn.Name.Pos, "extern function '%s' cannot have a body", fn.Name.Name)
	}
	fn.Pos = p.spanFrom(start)
	return fn
}

// parseMemberSeparator handles the separators between struct/trait members:
// newlines, `;`, or `,` (so `{ w: f64, h: f64 }` works on one line).
func (p *Parser) parseMemberSeparator() bool {
	if p.at(lexer.Semi, lexer.Comma) {
		p.next()
		p.skipSemis()
		return true
	}
	if p.at(lexer.RBrace, lexer.EOF) {
		return true
	}
	p.errorf(p.span(), "expected newline or ',' between members, found %s", p.cur().Describe())
	p.syncStmt()
	return false
}

func (p *Parser) parseStruct(attrs []*ast.Attribute, pub, extern bool, start source.Span) ast.Decl {
	p.next() // struct
	d := &ast.StructDecl{Attrs: attrs, Pub: pub, Extern: extern}
	d.Name, _ = p.expectIdent()
	d.TypeParams = p.parseTypeParams()
	if p.accept(lexer.Colon) {
		d.Variant = p.parseType()
	}
	if p.at(lexer.LBrace) {
		p.next()
		p.skipSemis()
		for !p.at(lexer.RBrace, lexer.EOF) {
			mattrs := p.parseAttributes()
			switch p.cur().Kind {
			case lexer.KwFun, lexer.KwMut, lexer.KwOverride, lexer.KwUnsafe, lexer.KwStatic:
				d.Methods = append(d.Methods, p.parseFun(mattrs, funContextMethod))
			case lexer.KwImpl:
				d.Impls = append(d.Impls, p.parseInlineImpl(mattrs, d))
			case lexer.KwPub:
				if p.peek(1).Kind == lexer.Ident {
					d.Fields = append(d.Fields, p.parseField(true))
				} else {
					d.Methods = append(d.Methods, p.parseFun(mattrs, funContextMethod))
				}
			case lexer.Ident:
				d.Fields = append(d.Fields, p.parseField(false))
			default:
				p.errorf(p.span(), "expected a field or method, found %s", p.cur().Describe())
				p.syncStmt()
				continue
			}
			p.parseMemberSeparator()
		}
		p.expect(lexer.RBrace)
	} else {
		p.expectTerminatorPeek("'{' after struct name")
	}
	d.Pos = p.spanFrom(start)
	return d
}

func (p *Parser) expectTerminatorPeek(what string) {
	if !p.at(lexer.Semi, lexer.RBrace, lexer.EOF) {
		p.errorExpected(what)
	}
}

func (p *Parser) parseField(pub bool) *ast.Field {
	start := p.span()
	doc := p.takeDoc()
	if pub {
		p.next()
	}
	f := &ast.Field{Pub: pub, Doc: doc}
	f.Name, _ = p.expectIdent()
	if _, ok := p.expect(lexer.Colon); ok {
		if p.errorFields {
			f.Type = p.parseErrorType()
		} else {
			f.Type = p.parseType()
		}
	}
	if p.accept(lexer.Assign) {
		f.Default = p.parseExpr()
	}
	f.Pos = p.spanFrom(start)
	return f
}

func (p *Parser) parseTrait(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	d := &ast.TraitDecl{Attrs: attrs, Pub: pub}
	if p.accept(lexer.KwSealed) {
		d.Sealed = true
	}
	if _, ok := p.expect(lexer.KwTrait); !ok {
		return &ast.BadDecl{Pos: p.span()}
	}
	d.Name, _ = p.expectIdent()
	d.TypeParams = p.parseTypeParams()
	if p.accept(lexer.Colon) {
		d.Supers = append(d.Supers, p.parseType())
		for p.accept(lexer.Plus) {
			d.Supers = append(d.Supers, p.parseType())
		}
	}
	if p.at(lexer.LBrace) {
		p.next()
		p.skipSemis()
		for !p.at(lexer.RBrace, lexer.EOF) {
			mattrs := p.parseAttributes()
			switch p.cur().Kind {
			case lexer.KwType:
				ts := p.span()
				p.next()
				at := &ast.AssocTypeDecl{}
				at.Name, _ = p.expectIdent()
				if p.accept(lexer.Colon) {
					at.Bounds = append(at.Bounds, p.parseType())
					for p.accept(lexer.Plus) {
						at.Bounds = append(at.Bounds, p.parseType())
					}
				}
				at.Pos = p.spanFrom(ts)
				d.AssocTypes = append(d.AssocTypes, at)
			case lexer.KwFun, lexer.KwMut, lexer.KwUnsafe, lexer.KwPub, lexer.KwStatic:
				d.Methods = append(d.Methods, p.parseFun(mattrs, funContextTrait))
			default:
				p.errorf(p.span(), "expected 'type' or 'fun' in trait body, found %s", p.cur().Describe())
				p.syncStmt()
				continue
			}
			p.parseMemberSeparator()
		}
		p.expect(lexer.RBrace)
	}
	d.Pos = p.spanFrom(start)
	return d
}

// parseImpl parses `impl<T> Trait for Type { ... }` or, with extend set,
// `extend<T> Type { ... }`.
func (p *Parser) parseImpl(attrs []*ast.Attribute, extend bool) ast.Decl {
	start := p.span()
	p.next() // impl / extend
	d := &ast.ImplDecl{Attrs: attrs, Extend: extend}
	d.TypeParams = p.parseTypeParams()
	if extend {
		d.Target = p.parseType()
		if p.at(lexer.KwFor) {
			p.errorf(p.span(), "'extend' names the type being extended, not a trait; use 'impl Trait for Type' to implement a trait")
			p.next()
			d.Target = p.parseType()
		}
	} else {
		d.Trait = p.parseType()
		if p.at(lexer.KwFor) {
			p.next()
			d.Target = p.parseType()
		} else {
			p.errorf(p.span(), "expected 'for' after trait name in impl; inherent methods go inside the struct body (D23)")
			d.Target = d.Trait
		}
	}
	p.parseImplBody(d, extend)
	d.Pos = p.spanFrom(start)
	return d
}

// parseInlineImpl parses `impl Trait { ... }` inside the body of struct
// sd (D23): an impl of the trait for the struct itself, with the struct's
// type parameters. It joins the file's declarations like a top-level impl;
// Inline marks it for the formatter.
func (p *Parser) parseInlineImpl(attrs []*ast.Attribute, sd *ast.StructDecl) *ast.ImplDecl {
	start := p.span()
	p.next() // impl
	d := &ast.ImplDecl{Attrs: attrs, Inline: true, TypeParams: sd.TypeParams}
	if p.at(lexer.Lt) {
		p.errorf(p.span(), "an impl inside a struct body uses the struct's type parameters; for other bounds write 'impl<T: ...> Trait for %s<T>' at top level", sd.Name.Name)
		p.parseTypeParams()
	}
	d.Trait = p.parseType()
	if p.at(lexer.KwFor) {
		p.errorf(p.span(), "an impl inside a struct body is for that struct; drop 'for'")
		p.next()
		p.parseType()
	}
	target := &ast.NamedType{Path: []ast.Ident{sd.Name}, Pos: sd.Name.Pos}
	for _, tp := range sd.TypeParams {
		target.Args = append(target.Args, &ast.NamedType{Path: []ast.Ident{tp.Name}, Pos: tp.Name.Pos})
	}
	d.Target = target
	p.parseImplBody(d, false)
	d.Pos = p.spanFrom(start)
	return d
}

// parseImplBody parses the `{ type ... = ...; fun ... }` body of an impl
// or extend block.
func (p *Parser) parseImplBody(d *ast.ImplDecl, extend bool) {
	if _, ok := p.expect(lexer.LBrace); ok {
		p.skipSemis()
		for !p.at(lexer.RBrace, lexer.EOF) {
			mattrs := p.parseAttributes()
			switch p.cur().Kind {
			case lexer.KwType:
				if extend {
					p.errorf(p.span(), "'extend' blocks add methods only; associated types belong to trait impls")
				}
				p.next()
				b := &ast.AssocTypeBinding{}
				b.Name, _ = p.expectIdent()
				if _, ok := p.expect(lexer.Assign); ok {
					b.Type = p.parseType()
				}
				d.AssocTypes = append(d.AssocTypes, b)
			case lexer.KwFun, lexer.KwMut, lexer.KwOverride, lexer.KwUnsafe, lexer.KwPub, lexer.KwStatic:
				d.Methods = append(d.Methods, p.parseFun(mattrs, funContextMethod))
			default:
				if extend {
					p.errorf(p.span(), "expected 'fun' in extend body, found %s", p.cur().Describe())
				} else {
					p.errorf(p.span(), "expected 'type' or 'fun' in impl body, found %s", p.cur().Describe())
				}
				p.syncStmt()
				continue
			}
			p.parseMemberSeparator()
		}
		p.expect(lexer.RBrace)
	}
}

// atExtendDecl recognises the contextual keyword `extend` at declaration
// position: `extend Type {` or `extend<T> Type {`.
func (p *Parser) atExtendDecl() bool {
	next := p.peek(1).Kind
	return p.at(lexer.Ident) && p.cur().Text == "extend" && (next == lexer.Ident || next == lexer.Lt || next == lexer.Star || next == lexer.LParen)
}

func (p *Parser) parseValDecl(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	d := &ast.ValDecl{Attrs: attrs, Pub: pub}
	switch p.next().Kind {
	case lexer.KwVar:
		d.Kind = ast.BindVar
	case lexer.KwConst:
		d.Kind = ast.BindConst
	default:
		d.Kind = ast.BindVal
	}
	d.Name, _ = p.expectIdent()
	if p.accept(lexer.Colon) {
		d.Type = p.parseType()
	}
	if _, ok := p.expect(lexer.Assign); ok {
		d.Value = p.parseExpr()
	}
	d.Pos = p.spanFrom(start)
	return d
}

func (p *Parser) parseExternBlock() ast.Decl {
	start := p.span()
	p.next() // extern
	d := &ast.ExternBlock{ABI: "C"}
	if p.at(lexer.String) {
		t := p.next()
		if len(t.Parts) == 1 && !t.Parts[0].IsExpr {
			d.ABI = t.Parts[0].Text
		}
		if d.ABI != "C" {
			p.errorf(t.Span, "unsupported ABI %q; only \"C\" is defined", d.ABI)
		}
	} else {
		p.errorExpected("ABI string (\"C\") after 'extern'")
	}
	if _, ok := p.expect(lexer.LBrace); ok {
		p.skipSemis()
		for !p.at(lexer.RBrace, lexer.EOF) {
			mattrs := p.parseAttributes()
			if !p.at(lexer.KwFun, lexer.KwPub, lexer.KwUnsafe) {
				p.errorf(p.span(), "expected 'fun' in extern block, found %s", p.cur().Describe())
				p.syncStmt()
				continue
			}
			fn := p.parseFun(mattrs, funContextExtern)
			fn.Extern = true
			d.Funs = append(d.Funs, fn)
			p.parseMemberSeparator()
		}
		p.expect(lexer.RBrace)
	}
	d.Pos = p.spanFrom(start)
	return d
}

// atErrorDecl reports whether the cursor is at an `error Name` declaration.
// `error` is a contextual keyword: it is an ordinary identifier everywhere
// else (`is Err(error) => ...`), and a declaration only at declaration
// position when a name follows.
func (p *Parser) atErrorDecl() bool {
	return p.at(lexer.Ident) && p.cur().Text == "error" && p.peek(1).Kind == lexer.Ident
}

// parseErrorDecl parses `error Name<T> { fields; methods }` (D4). It is
// sugar for a struct plus `impl Error for Name`: methods marked `override`
// go to the impl (they replace Error's defaults), the rest stay inherent.
func (p *Parser) parseErrorDecl(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	kw := p.span()
	if p.peek(2).Kind == lexer.Assign {
		return p.parseErrorAlias(attrs, pub, start)
	}
	p.errorFields = true
	d := p.parseStruct(attrs, pub, false, start).(*ast.StructDecl)
	p.errorFields = false
	d.Error = true
	if d.Variant != nil {
		p.errorf(kw, "an 'error' cannot be a variant of a sealed trait; declare it with 'struct'")
	}
	// `message` is the override of Error's default; `override` is implied
	// here (there is exactly one trait in play), the other methods are
	// inherent.
	var inherent, impl []*ast.FunDecl
	for _, m := range d.Methods {
		if m.Name.Name == "message" {
			m.Override = true
			impl = append(impl, m)
		} else {
			if m.Override {
				p.errorf(m.Name.Pos, "only 'message' can be overridden in an error; '%s' is an ordinary method", m.Name.Name)
			}
			inherent = append(inherent, m)
		}
	}
	// a `message: string` field is the message: forward the method to it
	if len(impl) == 0 {
		for _, f := range d.Fields {
			if nt, ok := f.Type.(*ast.NamedType); ok && f.Name.Name == "message" && len(nt.Path) == 1 && nt.Path[0].Name == "string" && len(nt.Args) == 0 {
				pos := f.Name.Pos
				impl = append(impl, &ast.FunDecl{
					Override: true,
					Name:     ast.Ident{Name: "message", Pos: pos},
					Ret:      &ast.NamedType{Path: []ast.Ident{{Name: "string", Pos: pos}}, Pos: pos},
					ExprBody: &ast.MemberExpr{X: &ast.SelfExpr{Pos: pos}, Name: ast.Ident{Name: "message", Pos: pos}, Pos: pos},
					Pos:      pos,
				})
			}
		}
	}
	d.Methods = inherent
	target := &ast.NamedType{Path: []ast.Ident{d.Name}, Pos: d.Name.Pos}
	for _, tp := range d.TypeParams {
		target.Args = append(target.Args, &ast.NamedType{Path: []ast.Ident{tp.Name}, Pos: tp.Name.Pos})
	}
	errorTrait := &ast.NamedType{Path: []ast.Ident{{Name: "Error", Pos: kw}}, Pos: kw}
	d.ErrorImpl = &ast.ImplDecl{TypeParams: d.TypeParams, Trait: errorTrait, Target: target, Methods: impl, Pos: d.Pos}
	return d
}

// parseErrorAlias parses `error Name = A | B | C`: a named error set
// (D45), transparent wherever a union may appear.
func (p *Parser) parseErrorAlias(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	p.next() // error
	d := &ast.ErrorAliasDecl{Attrs: attrs, Pub: pub}
	d.Name, _ = p.expectIdent()
	p.expect(lexer.Assign)
	d.Members = p.parseErrorType()
	d.Pos = p.spanFrom(start)
	return d
}
