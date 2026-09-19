package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

// Binding powers, weakest first. Each level binds tighter than the previous.
const (
	bpNone  = iota
	bpOr    // ||
	bpAnd   // &&
	bpEq    // == !=
	bpCmp   // < <= > >=
	bpNamed // is, !is
	bpElvis // ?:
	bpRange // .. ..<
	bpAdd   // + - +% -% | ^
	bpMul   // * / % *% & << >>
	bpCast  // as
	bpUnary // prefix
)

func infixBp(k lexer.TokenKind) int {
	switch k {
	case lexer.OrOr:
		return bpOr
	case lexer.AndAnd:
		return bpAnd
	case lexer.Eq, lexer.NotEq:
		return bpEq
	case lexer.Lt, lexer.LtEq, lexer.Gt, lexer.GtEq:
		return bpCmp
	case lexer.KwIs:
		return bpNamed
	case lexer.Elvis, lexer.OrFail:
		return bpElvis
	case lexer.Range, lexer.RangeLt:
		return bpRange
	case lexer.Plus, lexer.Minus, lexer.WrapPlus, lexer.WrapMinus, lexer.Pipe, lexer.Caret:
		return bpAdd // Go's grouping: | and ^ sit with + and -
	case lexer.Star, lexer.Slash, lexer.Percent, lexer.WrapStar, lexer.Amp, lexer.Shl, lexer.Shr:
		return bpMul // and &, <<, >> with *, /, %
	case lexer.KwAs:
		return bpCast
	}
	return bpNone
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseBinary(bpNone)
}

func (p *Parser) parseBinary(minBp int) ast.Expr {
	start := p.span()
	left := p.parseUnary()
	for {
		op := p.cur().Kind
		// `!is` is spelled as two tokens.
		notIs := op == lexer.Bang && p.peek(1).Kind == lexer.KwIs
		if notIs {
			op = lexer.KwIs
		}
		bp := infixBp(op)
		if bp == bpNone || bp <= minBp {
			return left
		}
		switch op {
		case lexer.KwIs:
			if notIs {
				p.next()
			}
			p.next()
			pat := p.parseTypePatternRest(p.prevSpan())
			left = &ast.IsExpr{X: left, Pat: pat, Not: notIs, Pos: p.spanFrom(start)}
		case lexer.KwAs:
			p.next()
			t := p.parseType()
			left = &ast.CastExpr{X: left, Type: t, Pos: p.spanFrom(start)}
		case lexer.Elvis:
			p.next()
			// right-associative: `a ?: b ?: c` is `a ?: (b ?: c)`
			right := p.parseBinary(bp - 1)
			left = &ast.ElvisExpr{L: left, R: right, Pos: p.spanFrom(start)}
		case lexer.OrFail:
			p.next()
			right := p.parseBinary(bp - 1)
			left = &ast.OrFailExpr{L: left, R: right, Pos: p.spanFrom(start)}
		case lexer.Range, lexer.RangeLt:
			p.next()
			right := p.parseBinary(bp)
			left = &ast.RangeExpr{Lo: left, Hi: right, Inclusive: op == lexer.Range, Pos: p.spanFrom(start)}
		default:
			p.next()
			right := p.parseBinary(bp)
			left = &ast.BinaryExpr{Op: op, L: left, R: right, Pos: p.spanFrom(start)}
		}
	}
}

func (p *Parser) parseUnary() ast.Expr {
	start := p.span()
	switch p.cur().Kind {
	case lexer.Minus, lexer.Bang, lexer.Amp, lexer.Star, lexer.Tilde:
		op := p.next().Kind
		x := p.parseUnary()
		return &ast.UnaryExpr{Op: op, X: x, Pos: p.spanFrom(start)}
	case lexer.KwTry:
		p.next()
		x := p.parseUnary()
		// `try x ?! e` is `try (x ?! e)`: the operator rewrites the failure,
		// `try` propagates it — `(try x) ?! e` would need x to be nullable
		// and is written with the parentheses
		for p.at(lexer.OrFail) {
			p.next()
			right := p.parseBinary(bpElvis - 1)
			x = &ast.OrFailExpr{L: x, R: right, Pos: p.spanFrom(start)}
		}
		return &ast.TryExpr{X: x, Pos: p.spanFrom(start)}
	case lexer.KwAwait:
		p.next()
		x := p.parseUnary()
		return &ast.AwaitExpr{X: x, Pos: p.spanFrom(start)}
	case lexer.KwAsync:
		p.next()
		x := p.parseUnary()
		call, ok := x.(*ast.CallExpr)
		if !ok {
			p.errorf(p.spanFrom(start), "'async' must prefix a call expression (D2)")
			return x
		}
		call.Async = true
		call.Pos = p.spanFrom(start)
		return call
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() ast.Expr {
	start := p.span()
	x := p.parsePrimary()
	for {
		switch p.cur().Kind {
		case lexer.Dot, lexer.SafeDot:
			safe := p.next().Kind == lexer.SafeDot
			var name ast.Ident
			if p.at(lexer.Int) {
				// tuple index: `pair.0`
				t := p.next()
				switch x.(type) {
				case *ast.IntLit, *ast.FloatLit:
					// `0 .0` would print back as the float `0.0`
					p.errorf(t.Span, "a number has no tuple elements")
				}
				name = ast.Ident{Name: t.Text, Pos: t.Span}
			} else {
				name, _ = p.expectIdent()
			}
			x = &ast.MemberExpr{X: x, Name: name, Safe: safe, Pos: p.spanFrom(start)}
		case lexer.LParen:
			args := p.parseArgs()
			x = &ast.CallExpr{Fun: x, Args: args, Pos: p.spanFrom(start)}
		case lexer.LBracket:
			p.next()
			idx := p.parseExpr()
			p.expect(lexer.RBracket)
			x = &ast.IndexExpr{X: x, Index: idx, Pos: p.spanFrom(start)}
		case lexer.Lt:
			// Possibly `Name<T>(...)`: a generic call. Try it speculatively.
			if !isGenericCallee(x) {
				return x
			}
			_, isName := x.(*ast.NameExpr)
			targs, ok, dot := p.tryTypeArgsBeforeCall(isName)
			if !ok {
				return x
			}
			if dot {
				// `Name<T>.f(...)`: a static function of a generic type (D23)
				n := x.(*ast.NameExpr)
				x = &ast.NameExpr{Name: n.Name, TypeArgs: targs, Pos: p.spanFrom(start)}
				continue
			}
			args := p.parseArgs()
			x = &ast.CallExpr{Fun: x, TypeArgs: targs, Args: args, Pos: p.spanFrom(start)}
		default:
			return x
		}
	}
}

func isGenericCallee(x ast.Expr) bool {
	switch x.(type) {
	case *ast.NameExpr, *ast.MemberExpr:
		return true
	}
	return false
}

// tryTypeArgsBeforeCall attempts to parse `<T, U>` followed by `(`, or —
// after a bare name — by `.name(` (dot is then set). On failure nothing is
// consumed and no diagnostics are emitted.
func (p *Parser) tryTypeArgsBeforeCall(allowDot bool) (targs []ast.Type, ok, dot bool) {
	savedPos, savedDiags, savedErr := p.pos, p.diags, p.lastErrPos
	scratch := &source.Diagnostics{}
	p.diags = scratch
	targs = p.parseTypeArgs()
	dot = allowDot && p.at(lexer.Dot) && p.peek(1).Kind == lexer.Ident && p.peek(2).Kind == lexer.LParen
	ok = !scratch.HasErrors() && (p.at(lexer.LParen) || dot)
	p.diags, p.lastErrPos = savedDiags, savedErr
	if !ok {
		p.pos = savedPos
		return nil, false, false
	}
	return targs, true, dot
}

func (p *Parser) parseArgs() []ast.Arg {
	saved := p.noLambda
	p.noLambda = 0
	defer func() { p.noLambda = saved }()
	p.expect(lexer.LParen)
	var args []ast.Arg
	for !p.at(lexer.RParen, lexer.EOF) {
		var arg ast.Arg
		if p.at(lexer.Ident) && p.peek(1).Kind == lexer.Colon {
			t := p.next()
			p.next()
			arg.Name = &ast.Ident{Name: t.Text, Pos: t.Span}
		}
		arg.Value = p.parseExpr()
		if p.accept(lexer.Ellipsis) {
			arg.Spread = true // `f(xs...)`: a list passed as the variadic parameter
		}
		args = append(args, arg)
		if _, bad := arg.Value.(*ast.BadExpr); bad {
			p.syncParen()
			break
		}
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.RParen)
	return args
}

func (p *Parser) parsePrimary() ast.Expr {
	start := p.span()
	t := p.cur()
	switch t.Kind {
	case lexer.Int:
		p.next()
		return &ast.IntLit{Text: t.Text, Pos: t.Span}
	case lexer.Float:
		p.next()
		return &ast.FloatLit{Text: t.Text, Pos: t.Span}
	case lexer.String:
		p.next()
		return p.stringLit(t)
	case lexer.Char:
		p.next()
		return &ast.CharLit{Value: t.Parts[0].Text, Pos: t.Span}
	case lexer.KwTrue, lexer.KwFalse:
		p.next()
		return &ast.BoolLit{Value: t.Kind == lexer.KwTrue, Pos: t.Span}
	case lexer.KwNull:
		p.next()
		return &ast.NullLit{Pos: t.Span}
	case lexer.KwSelf:
		p.next()
		return &ast.SelfExpr{Pos: t.Span}
	case lexer.Ident:
		if p.peek(1).Kind == lexer.FatArrow && p.noLambda == 0 {
			return p.parseLambda()
		}
		p.next()
		return &ast.NameExpr{Name: t.Text, Pos: t.Span}
	case lexer.Under:
		// `_ => ...`: a lambda that ignores its argument
		if p.peek(1).Kind == lexer.FatArrow && p.noLambda == 0 {
			return p.parseLambda()
		}
	case lexer.LParen:
		if p.noLambda == 0 && p.looksLikeLambda() {
			return p.parseLambda()
		}
		return p.parseParenOrTuple()
	case lexer.LBracket:
		return p.parseCollectionLit()
	case lexer.KwMut:
		// `mut [1, 2]` / `mut ["k": v]` / `mut [:]`: a mutable collection
		// literal without spelling the MutableList/MutableMap annotation.
		start := p.span()
		p.next()
		if !p.at(lexer.LBracket) {
			p.errorf(p.span(), "'mut' in an expression must be followed by a collection literal, e.g. 'mut [1, 2]' or 'mut [:]'")
			return &ast.BadExpr{Pos: p.spanFrom(start)}
		}
		lit := p.parseCollectionLit()
		switch l := lit.(type) {
		case *ast.ListLit:
			l.Mut = true
			l.Pos = p.spanFrom(start)
		case *ast.MapLit:
			l.Mut = true
			l.Pos = p.spanFrom(start)
		}
		return lit
	case lexer.KwIf:
		return p.parseIf()
	case lexer.KwWhen:
		return p.parseWhen()
	case lexer.KwWith:
		return p.parseWith()
	case lexer.KwGather:
		p.next()
		body := p.parseBlock()
		return &ast.GatherExpr{Body: body, Pos: p.spanFrom(start)}
	case lexer.KwUnsafe:
		p.next()
		body := p.parseBlock()
		return &ast.UnsafeExpr{Body: body, Pos: p.spanFrom(start)}
	case lexer.KwRace:
		return p.parseRace()
	case lexer.KwScope:
		p.errorf(t.Span, "'scope' is a statement, not an expression; use 'gather' to collect results (D36)")
		p.next()
		p.parseBlock()
		return &ast.BadExpr{Pos: p.spanFrom(start)}
	case lexer.KwFun:
		p.errorf(t.Span, "anonymous functions are written as lambdas: '(x) => ...' (D32)")
		return &ast.BadExpr{Pos: t.Span}
	case lexer.KwReturn, lexer.KwThrow, lexer.KwBreak, lexer.KwContinue:
		return &ast.ControlExpr{Stmt: p.parseStmt()}
	case lexer.Arrow:
		p.errorf(t.Span, "'->' is not an operator; use '=>' (D33)")
		p.next()
		return &ast.BadExpr{Pos: t.Span}
	}
	p.errorf(t.Span, "expected an expression, found %s", t.Describe())
	return &ast.BadExpr{Pos: t.Span}
}

func (p *Parser) stringLit(t lexer.Token) ast.Expr {
	lit := &ast.StringLit{Pos: t.Span}
	for _, part := range t.Parts {
		if !part.IsExpr {
			lit.Parts = append(lit.Parts, ast.StringPart{Text: part.Text})
			continue
		}
		toks := lexer.TokenizeRange(p.file, part.Span.Start, part.Span.End, p.diags)
		sub := &Parser{file: p.file, toks: toks, diags: p.diags, lastErrPos: -1}
		e := sub.parseExpr()
		sub.skipSemis()
		if !sub.at(lexer.EOF) {
			sub.errorf(sub.span(), "unexpected %s in string interpolation", sub.cur().Describe())
		}
		lit.Parts = append(lit.Parts, ast.StringPart{Expr: e})
	}
	return lit
}

// looksLikeLambda scans from `(` to its matching `)` and reports whether a
// `=>` (or `: Type =>`) follows — the TypeScript disambiguation (D37).
func (p *Parser) looksLikeLambda() bool {
	depth := 0
	i := 0
	for {
		t := p.peek(i)
		switch t.Kind {
		case lexer.LParen, lexer.LBracket, lexer.LBrace:
			depth++
		case lexer.RParen, lexer.RBracket, lexer.RBrace:
			depth--
			if depth == 0 {
				after := p.peek(i + 1).Kind
				return after == lexer.FatArrow || after == lexer.Colon
			}
		case lexer.EOF:
			return false
		}
		i++
	}
}

func (p *Parser) parseLambda() ast.Expr {
	start := p.span()
	l := &ast.LambdaExpr{}
	if p.at(lexer.Ident, lexer.Under) {
		t := p.next()
		l.Params = []ast.Param{{Name: p.paramName(t), Pos: t.Span}}
	} else {
		p.expect(lexer.LParen)
		for !p.at(lexer.RParen, lexer.EOF) {
			ps := p.span()
			if p.at(lexer.LParen) {
				// a tuple pattern in parameter position destructures the argument
				b := p.parseBinding()
				l.Params = append(l.Params, ast.Param{Name: ast.Ident{Name: "$tuple", Pos: b.Pos}, Pattern: &b, Pos: p.spanFrom(ps)})
				if !p.accept(lexer.Comma) {
					break
				}
				continue
			}
			var name ast.Ident
			if p.at(lexer.Under) {
				name = p.paramName(p.next())
			} else {
				var ok bool
				if name, ok = p.expectIdent(); !ok {
					p.syncParen()
					break
				}
			}
			prm := ast.Param{Name: name}
			if p.accept(lexer.Colon) {
				prm.Type = p.parseType()
			}
			prm.Pos = p.spanFrom(ps)
			l.Params = append(l.Params, prm)
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RParen)
		if p.accept(lexer.Colon) {
			l.Ret = p.parseType()
		}
	}
	p.expect(lexer.FatArrow)
	l.Body = p.parseArmBody()
	l.Pos = p.spanFrom(start)
	return l
}

// parseArmBody parses the right side of `=>` (a lambda, a `when` or `race`
// arm): a block, an expression, or an assignment — `x => total += x`,
// `cond => self.pos += 1` — which becomes a one-statement block.
func (p *Parser) parseArmBody() ast.Expr {
	if p.at(lexer.LBrace) {
		return &ast.BlockExpr{Block: p.parseBlock()}
	}
	start := p.span()
	body := p.parseExpr()
	switch p.cur().Kind {
	case lexer.Assign, lexer.PlusEq, lexer.MinusEq, lexer.StarEq, lexer.SlashEq, lexer.PercentEq:
		op := p.next().Kind
		val := p.parseExpr()
		as := &ast.AssignStmt{Target: body, Op: op, Value: val, Pos: p.spanFrom(start)}
		return &ast.BlockExpr{Block: &ast.Block{Stmts: []ast.Stmt{as}, Pos: as.Pos}}
	}
	return body
}

func (p *Parser) parseParenOrTuple() ast.Expr {
	saved := p.noLambda
	p.noLambda = 0
	defer func() { p.noLambda = saved }()
	start := p.span()
	p.next() // (
	if p.at(lexer.RParen) {
		p.next()
		return &ast.TupleExpr{Pos: p.spanFrom(start)} // unit
	}
	var elems []ast.Expr
	trailing := false
	for !p.at(lexer.RParen, lexer.EOF) {
		elems = append(elems, p.parseExpr())
		trailing = false
		if !p.accept(lexer.Comma) {
			break
		}
		trailing = true
	}
	p.expect(lexer.RParen)
	if len(elems) == 1 && !trailing {
		if p.parens != nil {
			sp := elems[0].Span()
			p.parens[[2]int{sp.Start, sp.End}] = true
		}
		return elems[0]
	}
	return &ast.TupleExpr{Elems: elems, Pos: p.spanFrom(start)}
}

func (p *Parser) parseCollectionLit() ast.Expr {
	saved := p.noLambda
	p.noLambda = 0
	defer func() { p.noLambda = saved }()
	start := p.span()
	p.next() // [
	if p.at(lexer.Colon) && p.peek(1).Kind == lexer.RBracket {
		p.next()
		p.next()
		return &ast.MapLit{Pos: p.spanFrom(start)}
	}
	if p.at(lexer.RBracket) {
		p.next()
		return &ast.ListLit{Pos: p.spanFrom(start)}
	}
	first := p.parseExpr()
	if p.at(lexer.Colon) {
		m := &ast.MapLit{}
		p.next()
		m.Entries = append(m.Entries, ast.MapEntry{Key: first, Value: p.parseExpr()})
		for p.accept(lexer.Comma) {
			if p.at(lexer.RBracket) {
				break
			}
			k := p.parseExpr()
			p.expect(lexer.Colon)
			v := p.parseExpr()
			m.Entries = append(m.Entries, ast.MapEntry{Key: k, Value: v})
		}
		p.expect(lexer.RBracket)
		m.Pos = p.spanFrom(start)
		return m
	}
	l := &ast.ListLit{Elems: []ast.Expr{first}}
	for p.accept(lexer.Comma) {
		if p.at(lexer.RBracket) {
			break
		}
		l.Elems = append(l.Elems, p.parseExpr())
	}
	p.expect(lexer.RBracket)
	l.Pos = p.spanFrom(start)
	return l
}

func (p *Parser) parseIf() ast.Expr {
	start := p.span()
	p.next() // if
	e := &ast.IfExpr{}
	if _, ok := p.expect(lexer.LParen); ok {
		e.Cond = p.parseExpr()
		p.closeCondition()
	} else {
		e.Cond = &ast.BadExpr{Pos: p.span()}
	}
	e.Then = p.parseBodyOrStmt()
	// `else` may follow on the next line after a `}` — unless it is the
	// `else =>` arm of an enclosing `when`, when this `if` is an arm body.
	if p.at(lexer.Semi) && p.cur().AutoSemi && p.peek(1).Kind == lexer.KwElse && p.peek(2).Kind != lexer.FatArrow {
		p.next()
	}
	if p.accept(lexer.KwElse) {
		if p.at(lexer.KwIf) {
			es := p.span()
			inner := p.parseIf()
			e.Else = &ast.Block{Stmts: []ast.Stmt{&ast.ExprStmt{X: inner}}, Pos: p.spanFrom(es)}
		} else {
			e.Else = p.parseBodyOrStmt()
		}
	}
	e.Pos = p.spanFrom(start)
	return e
}

func (p *Parser) parseWhen() ast.Expr {
	start := p.span()
	p.next() // when
	w := &ast.WhenExpr{}
	if p.accept(lexer.LParen) {
		if p.at(lexer.KwVal) && p.peek(1).Kind == lexer.Ident && p.peek(2).Kind == lexer.Assign {
			// `when (val r = expr)`: the subject gets a name the arms can use
			p.next()
			name, _ := p.expectIdent()
			w.Bind = &name
			p.next() // =
		}
		w.Subject = p.parseExpr()
		p.expect(lexer.RParen)
	}
	if _, ok := p.expect(lexer.LBrace); !ok {
		w.Pos = p.spanFrom(start)
		return w
	}
	for {
		p.skipSemis()
		if p.at(lexer.RBrace, lexer.EOF) {
			break
		}
		before := p.pos
		arm := p.parseWhenArm(w.Subject != nil)
		w.Arms = append(w.Arms, arm)
		if p.pos == before {
			p.next()
		}
		if !p.at(lexer.Semi, lexer.RBrace, lexer.EOF) {
			p.errorf(p.span(), "expected newline between 'when' arms, found %s", p.cur().Describe())
			p.syncStmt()
		}
	}
	p.expect(lexer.RBrace)
	w.Pos = p.spanFrom(start)
	return w
}

func (p *Parser) parseWhenArm(hasSubject bool) *ast.WhenArm {
	start := p.span()
	arm := &ast.WhenArm{}
	switch {
	case p.at(lexer.KwElse):
		p.next()
		arm.Else = true
	case hasSubject:
		for {
			arm.Patterns = append(arm.Patterns, p.parsePattern(false))
			if !p.accept(lexer.Comma) {
				break
			}
		}
		if p.accept(lexer.KwIf) {
			arm.Guard = p.parseArmHead()
		}
	default:
		arm.Cond = p.parseArmHead()
	}
	if _, ok := p.expect(lexer.FatArrow); !ok {
		p.syncStmt()
		arm.Body = &ast.BadExpr{Pos: p.span()}
		arm.Pos = p.spanFrom(start)
		return arm
	}
	arm.Body = p.parseArmBody()
	arm.Pos = p.spanFrom(start)
	return arm
}

func (p *Parser) parseRace() ast.Expr {
	start := p.span()
	p.next() // race
	r := &ast.RaceExpr{}
	if _, ok := p.expect(lexer.LBrace); !ok {
		r.Pos = p.spanFrom(start)
		return r
	}
	for {
		p.skipSemis()
		if p.at(lexer.RBrace, lexer.EOF) {
			break
		}
		as := p.span()
		arm := &ast.RaceArm{}
		if p.accept(lexer.KwVal) {
			b := p.parseBinding()
			arm.Binding = &b
			p.expect(lexer.Assign)
		}
		arm.Source = p.parseArmHead()
		if _, ok := p.expect(lexer.FatArrow); !ok {
			p.syncStmt()
			continue
		}
		arm.Body = p.parseArmBody()
		arm.Pos = p.spanFrom(as)
		r.Arms = append(r.Arms, arm)
		if !p.at(lexer.Semi, lexer.RBrace, lexer.EOF) {
			p.errorf(p.span(), "expected newline between 'race' arms, found %s", p.cur().Describe())
			p.syncStmt()
		}
	}
	p.expect(lexer.RBrace)
	r.Pos = p.spanFrom(start)
	return r
}

// closeCondition expects the `)` after a condition, with a hint for the
// most common slip: `=` where `==` was meant.
func (p *Parser) closeCondition() {
	if p.at(lexer.Assign) {
		p.errorf(p.span(), "'=' assigns; use '==' to compare")
		p.next()
		p.parseExpr()
	}
	p.expect(lexer.RParen)
}

// parseArmHead parses the expression before an arm's `=>` (a guard, a
// subjectless condition, a race source), where `name => ...` is the arm's
// arrow, not a lambda.
func (p *Parser) parseArmHead() ast.Expr {
	p.noLambda++
	e := p.parseExpr()
	p.noLambda--
	return e
}

// paramName is the identifier for a lambda parameter token: an identifier's
// text, or `_` for the discard.
func (p *Parser) paramName(t lexer.Token) ast.Ident {
	if t.Kind == lexer.Under {
		return ast.Ident{Name: "_", Pos: t.Span}
	}
	return ast.Ident{Name: t.Text, Pos: t.Span}
}
