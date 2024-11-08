package parser

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

type stmtHandler func(p *parser) ast.Stmt
type nudHandler func(p *parser) ast.Expr
type ledHandler func(p *parser, left ast.Expr, bp bindingPower) ast.Expr
type bindingPower int

const (
	defaultBp bindingPower = iota
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

func (p *parser) skipNewlines() {
	for p.currentTokenKind() == lexer.NEWLINE {
		p.advance()
	}
}

func (p *parser) lookupBp(tokenKind lexer.TokenKind) bindingPower {
	bp, exists := (*p.bpLookup)[tokenKind]
	if !exists {
		return defaultBp
	}
	return bp
}

func (p *parser) currentTokenKind() lexer.TokenKind {
	return p.tokens[p.pos].Kind
}

func (p *parser) peek() lexer.Token {
	return p.tokens[p.pos+1]
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

func (p *parser) led(kind lexer.TokenKind, bp bindingPower, handler ledHandler) {
	(*p.bpLookup)[kind] = bp
	(*p.ledLookup)[kind] = handler
}

func (p *parser) nud(kind lexer.TokenKind, handler nudHandler) {
	(*p.bpLookup)[kind] = primary
	(*p.nudLookup)[kind] = handler
}

func (p *parser) stmt(kind lexer.TokenKind, handler stmtHandler) {
	(*p.stmtLookup)[kind] = handler
}
