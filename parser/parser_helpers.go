package parser

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

type stmt_handler func(p *parser) ast.Stmt
type nud_handler func(p *parser) ast.Expr
type led_handler func(p *parser, left ast.Expr, bp binding_power) ast.Expr
type binding_power int

const (
	default_bp binding_power = iota
	comma
	assignment
	logical
	relational
	additive
	multiplicative
	exponentiation
	unary
	call
	member
	primary
)

func (p *parser) hasTokens() bool {
	return p.pos < len(p.tokens) && p.currentTokenKind() != lexer.EOF
}

func (p *parser) currentToken() lexer.Token {
	return p.tokens[p.pos]
}

func (p *parser) advance() lexer.Token {
	tk := p.currentToken()
	p.pos++
	return tk
}

func (p *parser) currentTokenKind() lexer.TokenKind {
	return p.tokens[p.pos].Kind
}

func (p *parser) expectError(expectedKind lexer.TokenKind, err any) lexer.Token {
	kind := p.currentTokenKind()

	if kind != expectedKind {
		if err == nil {
			err = fmt.Sprintf("Veles :: parser: Expected \"%s\" but received \"%s\" instead\n", lexer.TokenKindString(expectedKind), lexer.TokenKindString(kind))
		}
		panic(err)
	}

	return p.advance()
}

func (p *parser) expect(expectedKind lexer.TokenKind) lexer.Token {
	return p.expectError(expectedKind, nil)
}

func (p *parser) expectOneOf(expectedKind ...lexer.TokenKind) lexer.Token {
	currentTokenKind := p.currentTokenKind()
	if !p.currentToken().IsOneOfMany(expectedKind...) {
		var expectedKindsString string
		for _, kind := range expectedKind {
			expectedKindsString += lexer.TokenKindString(kind) + ", "
		}
		panic(fmt.Sprintf("Expected one of: \"%s\" but recieved \"%s\" instead\n", expectedKindsString, lexer.TokenKindString(currentTokenKind)))
	}
	return p.advance()
}

func (p *parser) led(kind lexer.TokenKind, bp binding_power, led_fn led_handler) {
	(*p.bp_lookup)[kind] = bp
	(*p.led_lookup)[kind] = led_fn
}

func (p *parser) nud(kind lexer.TokenKind, nud_fn nud_handler) {
	(*p.bp_lookup)[kind] = primary
	(*p.nud_lookup)[kind] = nud_fn
}

func (p *parser) stmt(kind lexer.TokenKind, stmt_fn stmt_handler) {
	(*p.bp_lookup)[kind] = default_bp
	(*p.stmt_lookup)[kind] = stmt_fn
}
