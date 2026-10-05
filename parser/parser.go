// Package parser builds an ast.File from tokens. It is a hand-written
// recursive-descent parser with Pratt-style expression parsing. Errors are
// recorded as diagnostics and the parser resynchronises at the next
// statement or declaration so that one mistake yields one message.
package parser

import (
	"fmt"
	"strings"

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

	// cVariadic is the `...` that ended the parameter list just parsed (D123)
	cVariadic source.Span

	// errorFields is set while parsing an `error` body: a field may then
	// have a union type, the one place outside `throws` where a union
	// appears (a cause, D45).
	errorFields bool

	// inTry is set while parsing the operand of a `try`: a `catch` after it
	// belongs to the whole `try` expression (`try f().g() catch (e) { ... }`),
	// not to the last call in the chain (D98). Any nested expression — an
	// argument, a parenthesis, a block — clears it.
	inTry bool
	// inTypeArg is set while parsing a constant in a type-argument list
	// (`Array<u8, 4 * 16>`): there a `>>` closes two lists, it is not a shift
	inTypeArg bool

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
		p.thinArrow(p.span(), "'->' is not an operator; use '=>' (D33)")
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
		case lexer.RBrace:
			if depth == 0 {
				return // closes the enclosing block: the caller's to consume
			}
			depth--
		case lexer.RParen, lexer.RBracket:
			if depth == 0 {
				// an unbalanced closer nobody will consume: skipping it keeps
				// the caller's loop moving
				p.next()
				continue
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
				lexer.KwPub, lexer.KwPrivate, lexer.KwInternal, lexer.KwUse, lexer.KwExtern, lexer.KwVal, lexer.KwVar, lexer.KwConst, lexer.KwType, lexer.At:
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
		lexer.KwPub, lexer.KwPrivate, lexer.KwInternal, lexer.KwUse, lexer.KwExtern, lexer.KwVal, lexer.KwVar, lexer.KwConst, lexer.At)
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
	case *ast.TypeAliasDecl:
		d.Doc = doc
	case *ast.EnumDecl:
		d.Doc = doc
	case *ast.TestDecl:
		if d.Doc == "" {
			d.Doc = doc
		}
	case *ast.SuiteDecl:
		if d.Doc == "" {
			d.Doc = doc
		}
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
	// visibility (M5): `public` = the package, `internal` = the module,
	// which is also what nothing means; `private` belongs to members
	pub := p.accept(lexer.KwPub)
	internal := p.accept(lexer.KwInternal)
	if internal && pub {
		p.errorf(start, "'public' and 'internal' contradict each other; a declaration has one visibility (M5)")
	}
	if p.at(lexer.KwPrivate) {
		p.errorf(p.span(), "'private' belongs to a member of a struct; a top-level declaration is private to its module unless 'public' (M5)")
		p.next()
	}
	marked := pub || internal
	which := "'public'"
	if internal {
		which = "'internal'"
	}
	d := p.parseDeclKind(attrs, pub, marked, which, start)
	if internal {
		setInternal(d)
	}
	return d
}

// setInternal records an explicit `internal` on the declaration, so the
// formatter keeps the author's word (it changes nothing else).
func setInternal(d ast.Decl) {
	switch d := d.(type) {
	case *ast.FunDecl:
		d.Internal = true
	case *ast.StructDecl:
		d.Internal = true
	case *ast.TraitDecl:
		d.Internal = true
	case *ast.ValDecl:
		d.Internal = true
	case *ast.ErrorAliasDecl:
		d.Internal = true
	case *ast.TypeAliasDecl:
		d.Internal = true
	case *ast.EnumDecl:
		d.Internal = true
	}
}

func (p *Parser) parseDeclKind(attrs []*ast.Attribute, pub, marked bool, which string, start source.Span) ast.Decl {
	p.oldImplSpelling()
	switch p.cur().Kind {
	case lexer.KwUse:
		if marked && !pub {
			p.errorf(start, "'use' cannot be %s", which)
		}
		d := p.parseUse().(*ast.UseDecl)
		if pub { // a re-export (D89)
			d.Pub = true
			d.Pos = start.To(d.Pos)
			for _, s := range d.Specs {
				s.Pub = true
			}
		}
		return d
	case lexer.KwFun, lexer.KwUnsafe, lexer.KwStatic:
		if p.atStaticAssert() {
			if marked {
				p.errorf(start, "a 'static assert' cannot be %s; it declares nothing", which)
			}
			if len(attrs) > 0 {
				p.errorf(attrs[0].Pos, "a 'static assert' takes no attributes")
			}
			return p.parseStaticAssert()
		}
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
		if p.cur().Text == "suite" && p.peek(1).Kind == lexer.String {
			if marked {
				p.errorf(start, "a suite cannot be %s; it only groups tests", which)
			}
			return p.parseSuite(attrs, start)
		}
		if p.cur().Text == "test" && p.peek(1).Kind == lexer.String {
			if marked {
				p.errorf(start, "a test cannot be %s; nothing calls it", which)
			}
			return p.parseTest(attrs, start)
		}
		if p.cur().Text == "test" && p.peek(1).Kind == lexer.KwFun {
			p.next() // test
			fn := p.parseFun(attrs, funContextFree)
			fn.Test = true
			fn.Pub = pub
			fn.Pos = start.To(fn.Pos)
			return fn
		}
		if p.atExtendDecl() {
			if marked {
				p.errorf(start, "'extend' cannot be %s; mark the methods instead", which)
			}
			return p.parseImpl(attrs, true)
		}
	case lexer.KwSealed, lexer.KwTrait:
		return p.parseTrait(attrs, pub, start)
	case lexer.KwImpl:
		if marked {
			p.errorf(start, "'implement' cannot be %s; visibility follows the trait and type", which)
		}
		return p.parseImpl(attrs, false)
	case lexer.KwConst:
		if p.peek(1).Kind == lexer.KwFun {
			fn := p.parseFun(attrs, funContextFree) // `const fun` (D113)
			fn.Pub = pub
			fn.Pos = start.To(fn.Pos)
			return fn
		}
		return p.parseValDecl(attrs, pub, start)
	case lexer.KwVal, lexer.KwVar:
		return p.parseValDecl(attrs, pub, start)
	case lexer.KwType:
		return p.parseTypeAlias(attrs, pub, start)
	case lexer.KwEnum:
		return p.parseEnum(attrs, pub, start)
	case lexer.KwExtern:
		if p.peek(1).Kind == lexer.KwStruct {
			p.next()
			return p.parseStruct(attrs, pub, true, start)
		}
		if p.peek(1).Kind == lexer.Ident && p.peek(1).Text == "union" && p.peek(2).Kind == lexer.Ident {
			// `extern union U { … }` (D120): `union` is a word only here
			p.next()
			d := p.parseStruct(attrs, pub, true, start).(*ast.StructDecl)
			d.Union = true
			if d.TypeParams != nil || d.Variant != nil || len(d.Methods) > 0 || d.Init != nil || len(d.Impls) > 0 || len(d.Statics) > 0 {
				p.errorf(d.Name.Pos, "an 'extern union' is a C layout: fields only — no type parameters, methods, 'init' or 'implement' (D120)")
			}
			return d
		}
		if p.peek(1).Kind == lexer.String && p.peek(2).Kind == lexer.KwFun {
			return p.parseExportedFun(attrs, pub, start)
		}
		return p.parseExternBlock()
	case lexer.KwWith:
		// read whole, so the error is one and the next declaration parses
		p.errorf(p.span(), "'with' closes its resource when a block ends, and a module has no end; open it inside a function: 'with x = e' or 'with (x = e) { ... }' (D100)")
		p.parseStmt()
		return &ast.BadDecl{Pos: p.spanFrom(start)}
	}
	p.errorf(p.span(), "expected a declaration, found %s", p.cur().Describe())
	return &ast.BadDecl{Pos: p.span()}
}

// parseUse parses `use a, b.c as d`: a comma-separated list of module
// imports; a comma at the end of a line continues the list.
func (p *Parser) parseUse() ast.Decl {
	start := p.span()
	p.next() // use
	d := &ast.UseDecl{}
	for {
		s := p.parseUseSpec()
		if s == nil {
			break
		}
		d.Specs = append(d.Specs, s)
		if !p.accept(lexer.Comma) {
			break
		}
		p.skipSemis()
	}
	if len(d.Specs) == 0 {
		p.errorf(start, "'use' needs a module path")
	}
	d.Pos = p.spanFrom(start)
	return d
}

func (p *Parser) parseUseSpec() *ast.UseSpec {
	start := p.span()
	s := &ast.UseSpec{}
	var dot source.Span // the `.` before a brace: the removed spelling `use m.{ a }`
	for {
		if p.at(lexer.LBrace) && len(s.Path) > 0 {
			p.parseUseNames(s)
			if dot.IsValid() {
				p.errorf(dot, "'use %s.{ … }' is spelled 'use %s { … }' (D85)", pathString(s.Path), pathString(s.Path))
				p.diags.Items[len(p.diags.Items)-1].Fix = &source.Fix{Title: "Remove the '.'", Edits: []source.TextEdit{{Span: dot}}}
			}
			s.Pos = p.spanFrom(start)
			return s
		}
		seg, ok := p.expectIdent()
		if !ok {
			break
		}
		s.Path = append(s.Path, seg)
		if !p.at(lexer.Dot) {
			break
		}
		dot = p.next().Span
	}
	if len(s.Path) == 0 {
		return nil
	}
	if p.accept(lexer.KwAs) {
		alias, _ := p.expectIdent()
		s.Alias = &alias
	}
	if p.at(lexer.LBrace) {
		p.parseUseNames(s)
	}
	s.Pos = p.spanFrom(start)
	return s
}

// parseUseNames parses the braces of `use m { f, T as U }` (D85): names
// separated by commas, a line break or a trailing comma allowed.
func (p *Parser) parseUseNames(s *ast.UseSpec) {
	open := p.next() // {
	p.skipSemis()
	starred := false
	for !p.at(lexer.RBrace) && !p.at(lexer.EOF) {
		start := p.span()
		if p.at(lexer.Star) {
			p.errorf(start, "'use %s { * }' does not exist: name what you use, so a reader sees where each name comes from (D85)", pathString(s.Path))
			starred = true
			p.next()
		} else {
			name, ok := p.expectIdent()
			if !ok {
				break
			}
			n := &ast.UseName{Name: name}
			if p.accept(lexer.KwAs) {
				alias, _ := p.expectIdent()
				n.Alias = &alias
			}
			n.Pos = p.spanFrom(start)
			s.Names = append(s.Names, n)
		}
		p.skipSemis()
		if !p.accept(lexer.Comma) {
			break
		}
		p.skipSemis()
	}
	p.skipSemis()
	closer := p.span()
	p.expect(lexer.RBrace)
	if len(s.Names) == 0 && !starred {
		p.errorf(source.Span{File: open.Span.File, Start: open.Span.Start, End: closer.End}, "'use %s { }' names nothing: list what to import, or write 'use %s'", pathString(s.Path), pathString(s.Path))
	}
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
	lt := p.next()
	var tps []ast.TypeParam
	if p.atTypeClose() {
		p.errorf(lt.Span, "empty type parameter list; drop the '<>'")
	}
	for !p.atTypeClose() {
		var constAt source.Span
		isConst := p.at(lexer.KwConst)
		if isConst {
			constAt = p.next().Span // `<const N: i64>` (D121)
		}
		name, ok := p.expectIdent()
		if !ok {
			break
		}
		tp := ast.TypeParam{Name: name, Const: isConst, At: constAt}
		if isConst {
			if _, ok := p.expect(lexer.Colon); !ok {
				break
			}
			tp.Of = p.parseType()
			if nt, ok := tp.Of.(*ast.NamedType); !ok || len(nt.Path) != 1 || nt.Path[0].Name != "i64" || len(nt.Args) != 0 {
				p.errorf(tp.Of.Span(), "a constant parameter is an 'i64': write 'const %s: i64' (D121)", name.Name)
			}
			tps = append(tps, tp)
			if !p.accept(lexer.Comma) {
				break
			}
			continue
		}
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
	p.expectTypeClose()
	return tps
}

func (p *Parser) parseParams() []ast.Param {
	if _, ok := p.expect(lexer.LParen); !ok {
		return nil
	}
	var params []ast.Param
	for !p.at(lexer.RParen, lexer.EOF) {
		start := p.span()
		if p.at(lexer.Ellipsis) {
			// C's variadic arguments (D123); the checker says where they are allowed
			p.cVariadic = p.next().Span
			if !p.at(lexer.RParen) {
				p.errorf(p.span(), "'...' ends the parameter list: C's variadic arguments come after every named one (D123)")
				p.syncParen()
			}
			break
		}
		// `lazy` is a modifier only in front of a parameter's name (D90)
		lazy := p.at(lexer.Ident) && p.cur().Text == "lazy" && p.peek(1).Kind == lexer.Ident
		var lazyPos source.Span
		if lazy {
			lazyPos = p.span()
			p.next()
		}
		name, ok := p.expectIdent()
		if !ok {
			p.syncParen()
			break
		}
		prm := ast.Param{Name: name, Lazy: lazy, LazyPos: lazyPos}
		if _, ok := p.expect(lexer.Colon); ok {
			prm.Type = p.parseType()
			if p.accept(lexer.Ellipsis) {
				prm.Variadic = true // `parts: string...`: any number of arguments, a List inside
			}
		}
		if p.accept(lexer.Assign) {
			prm.Default = p.parseExpr()
			if prm.Variadic {
				p.errorf(prm.Default.Span(), "a variadic parameter cannot have a default; it is empty when no argument is given")
			}
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
			// removed in v0.30 (D22): every method may assign the receiver's
			// `var` fields; the marker no longer exists
			sp := p.span()
			end := sp.End
			if src := sp.File; src != nil {
				for end < len(src.Content) && (src.Content[end] == ' ' || src.Content[end] == '\t') {
					end++
				}
			}
			p.errorf(sp, "'mut fun' no longer exists: a method may assign the fields declared 'var' (D22); remove 'mut' and mark the fields it changes 'var'")
			if n := len(p.diags.Items); n > 0 && p.diags.Items[n-1].Span.Start == sp.Start {
				p.diags.Items[n-1].Fix = &source.Fix{Title: "Remove 'mut'", Edits: []source.TextEdit{{Span: source.Span{File: sp.File, Start: sp.Start, End: end}}}}
			}
		case lexer.KwOverride:
			fn.Override = true
		case lexer.KwPrivate:
			fn.Private = true
			if ctx == funContextFree || ctx == funContextExtern {
				p.errorf(p.span(), "'private' belongs to a member of a struct; a top-level declaration is private to its module unless 'public' (M5)")
			}
		case lexer.KwInternal:
			fn.Internal = true
		case lexer.KwStatic:
			fn.Static = true
			if ctx == funContextFree || ctx == funContextExtern {
				p.errorf(p.span(), "'static' belongs to a function in a struct, trait, impl or extend body; a top-level function needs no marker")
			}
		case lexer.KwUnsafe:
			fn.Unsafe = true
		case lexer.KwConst:
			fn.Const = true
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
	if p.at(lexer.Lt) {
		// `fun <T> name(...)`: the pre-v0.33 spelling; the type parameters
		// follow the name, as on a struct (`fun name<T>(...)`)
		p.errorf(p.span(), "type parameters follow the function name: write 'fun %s<...>(...)' (v0.33)", p.peekIdentAfterTypeParams())
		fn.TypeParams = p.parseTypeParams()
	}
	fn.Name, _ = p.expectIdent()
	if p.at(lexer.Lt) {
		if fn.TypeParams != nil {
			p.errorf(p.span(), "type parameters were already given before the function name")
		}
		fn.TypeParams = p.parseTypeParams()
	}
	fn.Params = p.parseParams()
	if p.cVariadic.File != nil {
		fn.CVariadic, fn.VariadicAt, p.cVariadic = true, p.cVariadic, source.Span{}
	}
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

// atStaticAssert: `static assert(` starts a compile-time assertion (D113).
func (p *Parser) atStaticAssert() bool {
	return p.at(lexer.KwStatic) && p.peek(1).Kind == lexer.Ident && p.peek(1).Text == "assert" && p.peek(2).Kind == lexer.LParen
}

// parseStaticAssert parses `static assert(cond, "why")`; the reason is
// required, as for a test's `assert` (D78).
func (p *Parser) parseStaticAssert() *ast.StaticAssert {
	start := p.span()
	p.next() // static
	p.next() // assert
	p.next() // (
	d := &ast.StaticAssert{Cond: p.parseExpr()}
	if p.accept(lexer.Comma) && !p.at(lexer.RParen) {
		d.Reason = p.parseExpr()
		p.accept(lexer.Comma)
	} else {
		p.errorf(p.span(), "a 'static assert' needs a reason, said when it fails: 'static assert(cond, \"why it must hold\")' (D113)")
	}
	p.expect(lexer.RParen)
	d.Pos = p.spanFrom(start)
	return d
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
			// a member may start with a visibility word (M5: public /
			// internal / private); what follows says whether it is a field,
			// a static or a method
			at := 0
			vis := p.cur().Kind
			hasVis := vis == lexer.KwPub || vis == lexer.KwPrivate || vis == lexer.KwInternal
			if hasVis {
				at = 1
			}
			if !hasVis {
				p.oldImplSpelling()
			}
			switch p.peek(at).Kind {
			case lexer.Ident, lexer.KwVar, lexer.KwVal, lexer.KwProtected:
				if p.peek(at).Kind == lexer.Ident && p.peek(at).Text == "init" && (p.peek(at+1).Kind == lexer.LBrace || p.peek(at+1).Kind == lexer.LParen) {
					// `init { }` / `init(value: T) { }`: contextual — a field
					// named init is `init: T`
					if hasVis {
						p.errorf(p.span(), "'init' has no visibility: it is not callable, it runs at every construction (D28)")
						p.next()
					}
					if d.Init != nil {
						p.errorf(p.span(), "a struct has one 'init' block")
					}
					d.InitPos = p.span()
					p.next() // init
					if p.at(lexer.LParen) {
						d.InitParams = p.parseParams()
						if p.cVariadic.File != nil {
							p.errorf(p.cVariadic, "only a function in an 'extern \"C\"' block takes C's variadic arguments; an 'init' takes 'name: T...' (D123)")
							p.cVariadic = source.Span{}
						}
						for _, prm := range d.InitParams {
							if prm.Type == nil {
								p.errorf(prm.Name.Pos, "an 'init' parameter needs a type: '%s: T'", prm.Name.Name)
							}
						}
					}
					d.Init = p.parseBlock()
					break
				}
				d.Fields = append(d.Fields, p.parseField(mattrs, vis == lexer.KwPub, vis == lexer.KwPrivate, vis == lexer.KwInternal, hasVis))
			case lexer.KwStatic:
				if p.peek(at+1).Kind == lexer.KwVal || p.peek(at+1).Kind == lexer.KwVar {
					if hasVis {
						p.next()
					}
					if vis == lexer.KwPrivate {
						p.errorf(p.prevSpan(), "a static value cannot be 'private'; it is the module's unless 'public' (D23)")
					}
					d.Statics = append(d.Statics, p.parseStaticVal(mattrs, vis == lexer.KwPub, vis == lexer.KwInternal))
				} else {
					d.Methods = append(d.Methods, p.parseFun(mattrs, funContextMethod))
				}
			case lexer.KwFun, lexer.KwMut, lexer.KwOverride, lexer.KwUnsafe, lexer.KwConst:
				d.Methods = append(d.Methods, p.parseFun(mattrs, funContextMethod))
			case lexer.KwImpl:
				if hasVis {
					p.errorf(p.span(), "an inline 'implement' has no visibility of its own; it follows the trait and type")
					p.next()
				}
				d.Impls = append(d.Impls, p.parseInlineImpl(mattrs, d))
			default:
				p.errorf(p.peek(at).Span, "expected a field or method, found %s", p.peek(at).Describe())
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

// parseStaticVal parses `static val name[: T] = expr` inside a struct body:
// a constant in the type's namespace, read as `Type.name`.
func (p *Parser) parseStaticVal(attrs []*ast.Attribute, pub, internal bool) *ast.ValDecl {
	start := p.span()
	doc := p.takeDoc()
	p.next() // static
	if p.at(lexer.KwVar) {
		p.errorf(p.span(), "a static member is a 'val': a type's constants do not change (a mutable global belongs at module level)")
	}
	p.next() // val / var
	d := &ast.ValDecl{Attrs: attrs, Doc: doc, Pub: pub, Internal: internal, Kind: ast.BindVal}
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

func (p *Parser) expectTerminatorPeek(what string) {
	if !p.at(lexer.Semi, lexer.RBrace, lexer.EOF) {
		p.errorExpected(what)
	}
}

// parseField parses `[public|internal|private] [protected] [val|var] name: T [= default]`
// with the visibility word (if hasVis) still at the cursor.
func (p *Parser) parseField(attrs []*ast.Attribute, pub, private, internal, hasVis bool) *ast.Field {
	start := p.span()
	doc := p.takeDoc()
	if hasVis {
		p.next()
	}
	f := &ast.Field{Attrs: attrs, Pub: pub, Private: private, Internal: internal, Doc: doc}
	// mutability (D22): bare or `val` = set once by the constructor; `var`
	// = assignable by anyone who sees the field; `protected var` =
	// assignable only by the type's own declarations
	if p.at(lexer.KwProtected) {
		sp := p.span()
		p.next()
		f.Protected = true
		switch {
		case private:
			p.errorf(sp, "'private protected' is redundant: a private field is already assignable only by the type; write 'private var' (D22)")
		case !p.at(lexer.KwVar):
			p.errorf(sp, "'protected' qualifies 'var': a bare field is never assigned, so there is nothing to protect; write 'protected var %s' (D22)", p.cur().Text)
		}
	}
	switch p.cur().Kind {
	case lexer.KwVar:
		f.Var = true
		p.next()
	case lexer.KwVal:
		f.Val = true
		p.next()
	}
	if p.at(lexer.KwVar, lexer.KwVal, lexer.KwProtected) {
		p.errorf(p.span(), "a field is 'val' (the default), 'var' or 'protected var' (D22)")
		p.next()
	}
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
			case lexer.KwFun, lexer.KwMut, lexer.KwUnsafe, lexer.KwPub, lexer.KwInternal, lexer.KwStatic:
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
			p.errorf(p.span(), "'extend' names the type being extended, not a trait; use 'implement Trait for Type'")
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
		p.errorf(p.span(), "an impl inside a struct body uses the struct's type parameters; for other bounds write 'implement<T: ...> Trait for %s<T>' at top level", sd.Name.Name)
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
// or extend block. A trait impl may have no body at all — `impl Codable`
// in a struct, `impl Codable for geo.Point` at top level — which asks the
// compiler to derive it (D58); an `extend` always has braces.
func (p *Parser) parseImplBody(d *ast.ImplDecl, extend bool) {
	if !extend && !p.at(lexer.LBrace) {
		d.Braceless = true
		p.expectTerminatorPeek("'{' or the end of the line after the impl header")
		return
	}
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
			case lexer.KwFun, lexer.KwMut, lexer.KwOverride, lexer.KwUnsafe, lexer.KwConst, lexer.KwPub, lexer.KwPrivate, lexer.KwInternal, lexer.KwStatic:
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

// parseExportedFun parses `extern "C" fun name(...): R { body }`: a Veles
// function with the C calling convention, for C to call back (D69).
func (p *Parser) parseExportedFun(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	doc := p.takeDoc() // a doc comment sits on `extern`, the first token
	p.next()           // extern
	t := p.next()
	if len(t.Parts) != 1 || t.Parts[0].IsExpr || t.Parts[0].Text != "C" {
		p.errorf(t.Span, "unsupported ABI; only \"C\" is defined")
	}
	fn := p.parseFun(attrs, funContextFree)
	fn.ExportC = true
	fn.Pub = fn.Pub || pub
	fn.Pos = start.To(fn.Pos)
	if fn.Doc == "" {
		fn.Doc = doc
	}
	if fn.Body == nil && fn.ExprBody == nil {
		p.errorf(fn.Name.Pos, "an 'extern \"C\" fun' is a Veles function C calls, so it needs a body; a C function Veles calls is declared in an 'extern \"C\" { }' block")
	}
	return fn
}

// parseTest parses `test "name" { body }` (D78). `test` is contextual: only
// `test` followed by a string starts one, so a function or value named
// `test` stays legal.
func (p *Parser) parseTest(attrs []*ast.Attribute, start source.Span) ast.Decl {
	doc := p.takeDoc()
	p.next() // test
	t := p.next()
	d := &ast.TestDecl{Doc: doc, At: t.Span, Name: p.fixedName(t, "test", "test \"parses an empty list\" { ... }")}
	if len(attrs) > 0 {
		p.errorf(attrs[0].Pos, "a test takes no attributes")
	}
	if !p.at(lexer.LBrace) {
		p.errorf(p.span(), "expected '{' to open the test's body, found %s", p.cur().Describe())
		d.Pos = p.spanFrom(start)
		return d
	}
	d.Body = p.parseBlock()
	d.Pos = p.spanFrom(start)
	return d
}

// fixedName is the text of a test's or a suite's name: a plain string,
// neither interpolated nor blank.
func (p *Parser) fixedName(t lexer.Token, what, example string) string {
	name := ""
	for _, part := range t.Parts {
		if part.IsExpr {
			p.errorf(t.Span, "a %s's name is fixed text; '${...}' has nothing to interpolate here", what)
			return name
		}
		name += part.Text
	}
	if strings.TrimSpace(name) == "" {
		p.errorf(t.Span, "a %s needs a name that says what it checks: %s", what, example)
	}
	return name
}

// parseSuite parses `suite "name" { ... }` (D78): tests, suites and `test fun`
// helpers. `suite` is contextual, like `test`.
func (p *Parser) parseSuite(attrs []*ast.Attribute, start source.Span) ast.Decl {
	doc := p.takeDoc()
	p.next() // suite
	t := p.next()
	d := &ast.SuiteDecl{Doc: doc, At: t.Span, Name: p.fixedName(t, "suite", "suite \"the router\" { ... }")}
	if len(attrs) > 0 {
		p.errorf(attrs[0].Pos, "a suite takes no attributes")
	}
	if _, ok := p.expect(lexer.LBrace); !ok {
		d.Pos = p.spanFrom(start)
		return d
	}
	for {
		p.skipSemis()
		if p.at(lexer.RBrace, lexer.EOF) {
			break
		}
		before := p.pos
		inner := p.parseDecl()
		switch x := inner.(type) {
		case *ast.TestDecl, *ast.SuiteDecl:
			d.Decls = append(d.Decls, inner)
		case *ast.FunDecl:
			if !x.Test {
				p.errorf(x.Name.Pos, "a suite holds tests, suites and helpers; a helper is 'test fun %s', and a function the program uses belongs outside the suite", x.Name.Name)
			}
			d.Decls = append(d.Decls, inner)
		case *ast.BadDecl:
		default:
			p.errorf(inner.Span(), "a suite holds tests ('test \"...\" { }'), 'test fun' helpers and other suites")
		}
		if p.pos == before {
			p.next()
		}
	}
	p.expect(lexer.RBrace)
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
	d.ErrorImpl = &ast.ImplDecl{TypeParams: d.TypeParams, Trait: errorTrait, Target: target, Methods: impl, ErrorSugar: true, Pos: d.Pos}
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

// parseEnum parses `enum Name [: Base] { A = 1, B, C }` (D57): members are
// separated like struct fields, each an identifier with an optional
// constant value.
func (p *Parser) parseEnum(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	p.next() // enum
	d := &ast.EnumDecl{Attrs: attrs, Pub: pub}
	d.Name, _ = p.expectIdent()
	if p.at(lexer.Lt) {
		p.errorf(p.span(), "an enum is not generic: it is a closed set of values of one integer type (D57)")
		p.parseTypeParams()
	}
	if p.accept(lexer.Colon) {
		d.Base = p.parseType()
	}
	if !p.at(lexer.LBrace) {
		p.expectTerminatorPeek("'{' after enum name")
		d.Pos = p.spanFrom(start)
		return d
	}
	p.next()
	p.skipSemis()
	for !p.at(lexer.RBrace, lexer.EOF) {
		mstart := p.span()
		mattrs := p.parseAttributes()
		doc := p.takeDoc()
		if !p.at(lexer.Ident) {
			p.errorf(p.span(), "expected an enum member name, found %s", p.cur().Describe())
			p.syncStmt()
			continue
		}
		m := &ast.EnumMember{Attrs: mattrs, Doc: doc}
		m.Name, _ = p.expectIdent()
		if p.accept(lexer.Assign) {
			m.Value = p.parseExpr()
		}
		m.Pos = p.spanFrom(mstart)
		d.Members = append(d.Members, m)
		p.parseMemberSeparator()
	}
	p.expect(lexer.RBrace)
	d.Pos = p.spanFrom(start)
	return d
}

// parseTypeAlias parses `type Name<T> = Type` at module level (D55).
func (p *Parser) parseTypeAlias(attrs []*ast.Attribute, pub bool, start source.Span) ast.Decl {
	p.next() // type
	d := &ast.TypeAliasDecl{Attrs: attrs, Pub: pub}
	d.Name, _ = p.expectIdent()
	if p.at(lexer.Lt) {
		d.TypeParams = p.parseTypeParams()
	}
	if _, ok := p.expect(lexer.Assign); !ok {
		p.syncStmt()
		d.Pos = p.spanFrom(start)
		return d
	}
	d.Type = p.parseType()
	d.Pos = p.spanFrom(start)
	return d
}

// pathString joins a dotted path for diagnostics.
func pathString(path []ast.Ident) string {
	parts := make([]string, len(path))
	for i, seg := range path {
		parts[i] = seg.Name
	}
	return strings.Join(parts, ".")
}

// oldImplSpelling turns the pre-v0.33 keyword `impl` at declaration
// position into `implement`, with an error that carries the fix: the
// block still parses, so one rename yields one message.
func (p *Parser) oldImplSpelling() {
	if !p.at(lexer.Ident) || p.cur().Text != "impl" {
		return
	}
	switch p.peek(1).Kind {
	case lexer.Ident, lexer.Lt:
	default:
		return
	}
	p.errorf(p.span(), "'impl' is spelled 'implement' (v0.33: `implement Display for Point { }`, `implement Codable`)")
	p.toks[p.pos].Kind = lexer.KwImpl
}

// oldSelfSpelling reports the pre-v0.40 receiver `self` with a fix that
// writes `this` (D65). The expression still parses as the receiver, so a
// file with the old spelling checks normally apart from these errors and
// `veles check --fix` migrates it in one pass.
func (p *Parser) oldSelfSpelling(t lexer.Token) {
	p.diags.Items = append(p.diags.Items, source.Diagnostic{
		Severity: source.Error,
		Span:     t.Span,
		Message:  "the receiver is spelled 'this' (v0.40, D65)",
		Fix:      &source.Fix{Title: "Replace 'self' with 'this'", Edits: []source.TextEdit{{Span: t.Span, NewText: "this"}}},
	})
}

// peekIdentAfterTypeParams is the identifier that follows a `<...>` at the
// cursor, for a message; "name" when there is none.
func (p *Parser) peekIdentAfterTypeParams() string {
	depth := 0
	for i := 0; p.pos+i < len(p.toks); i++ {
		switch p.peek(i).Kind {
		case lexer.Lt:
			depth++
		case lexer.Gt, lexer.Shr:
			depth--
			if p.peek(i).Kind == lexer.Shr {
				depth--
			}
			if depth <= 0 {
				if p.peek(i+1).Kind == lexer.Ident {
					return p.peek(i + 1).Text
				}
				return "name"
			}
		case lexer.LParen, lexer.LBrace, lexer.EOF:
			return "name"
		}
	}
	return "name"
}
