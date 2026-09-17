package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

// parseType parses a type. Postfix `?` binds tighter than prefix `*` (D5),
// so `*T?` is a pointer to a nullable T and `(*T)?` a nullable pointer.
func (p *Parser) parseType() ast.Type {
	start := p.span()
	var t ast.Type
	switch p.cur().Kind {
	case lexer.Star:
		p.next()
		raw := p.accept(lexer.KwRaw)
		elem := p.parseType()
		return &ast.PointerType{Elem: elem, Raw: raw, Pos: p.spanFrom(start)}
	case lexer.LParen:
		p.next()
		var elems []ast.Type
		trailingComma := false
		for !p.at(lexer.RParen, lexer.EOF) {
			elems = append(elems, p.parseType())
			trailingComma = false
			if !p.accept(lexer.Comma) {
				break
			}
			trailingComma = true
		}
		p.expect(lexer.RParen)
		if len(elems) == 1 && !trailingComma {
			t = elems[0] // parenthesised type
		} else {
			t = &ast.TupleType{Elems: elems, Pos: p.spanFrom(start)}
		}
	case lexer.KwFun:
		p.next()
		ft := &ast.FunType{}
		if _, ok := p.expect(lexer.LParen); ok {
			for !p.at(lexer.RParen, lexer.EOF) {
				ft.Params = append(ft.Params, p.parseType())
				if !p.accept(lexer.Comma) {
					break
				}
			}
			p.expect(lexer.RParen)
		}
		if p.accept(lexer.Colon) {
			ft.Ret = p.parseType()
		}
		ft.Effects = p.parseEffects()
		ft.Pos = p.spanFrom(start)
		t = ft
	case lexer.KwSelfTy:
		p.next()
		t = &ast.SelfType{Pos: start}
	case lexer.Ident:
		nt := &ast.NamedType{}
		for {
			id, _ := p.expectIdent()
			nt.Path = append(nt.Path, id)
			if p.at(lexer.Dot) && p.peek(1).Kind == lexer.Ident {
				p.next()
				continue
			}
			break
		}
		if p.at(lexer.Lt) {
			nt.Args = p.parseTypeArgs()
		}
		nt.Pos = p.spanFrom(start)
		t = nt
	default:
		p.errorExpected("a type")
		return &ast.NamedType{Path: []ast.Ident{{Name: "<error>", Pos: start}}, Pos: start}
	}

	// postfix: `::Assoc` projections and `?`
	for {
		switch {
		case p.at(lexer.DblColon):
			p.next()
			name, _ := p.expectIdent()
			t = &ast.AssocType{Base: t, Name: name, Pos: p.spanFrom(start)}
		case p.at(lexer.Question):
			p.next()
			t = &ast.NullableType{Elem: t, Pos: p.spanFrom(start)}
		default:
			return t
		}
	}
}

// parseTypeArgs parses `<T, U>` in type context.
func (p *Parser) parseTypeArgs() []ast.Type {
	p.expect(lexer.Lt)
	var args []ast.Type
	for !p.at(lexer.Gt, lexer.EOF) {
		args = append(args, p.parseType())
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.Gt)
	return args
}

// parseErrorType parses the type after `throws`: a type or a union `A | B`.
func (p *Parser) parseErrorType() ast.Type {
	start := p.span()
	first := p.parseType()
	if !p.at(lexer.Pipe) {
		return first
	}
	u := &ast.ErrorUnionType{Members: []ast.Type{first}}
	for p.accept(lexer.Pipe) {
		u.Members = append(u.Members, p.parseType())
	}
	u.Pos = p.spanFrom(start)
	return u
}
