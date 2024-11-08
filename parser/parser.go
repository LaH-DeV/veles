package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

type parser struct {
	tokens []lexer.Token
	pos    int

	stmtLookup *map[lexer.TokenKind]stmtHandler
	nudLookup  *map[lexer.TokenKind]nudHandler
	ledLookup  *map[lexer.TokenKind]ledHandler
	bpLookup   *map[lexer.TokenKind]bindingPower

	filetype lexer.Filetype
}

func NewParser(filetype lexer.Filetype) *parser {
	switch filetype {
	case lexer.Vs:
		return vsParser()
	case lexer.Wat:
		return watParser()
	default:
		return nil
	}
}

func (p *parser) ParseFile(tokens []lexer.Token, filename string) *ast.Program {
	p.newState(tokens)

	body := make([]ast.Stmt, 0)

	for p.hasTokens() {
		p.skipNewlines()
		stmt := parse_stmt(p)
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	return &ast.Program{
		Statements: body,
		Filetype:   p.filetype,
		Filename:   filename,
	}
}

func vsParser() *parser {
	p := &parser{
		tokens:     []lexer.Token{},
		pos:        0,
		stmtLookup: &map[lexer.TokenKind]stmtHandler{},
		nudLookup:  &map[lexer.TokenKind]nudHandler{},
		ledLookup:  &map[lexer.TokenKind]ledHandler{},
		bpLookup:   &map[lexer.TokenKind]bindingPower{},
		filetype:   lexer.Vs,
	}

	p.led(lexer.PLUS, additive, parse_binary_expr)
	p.led(lexer.DASH, additive, parse_binary_expr)
	p.led(lexer.SLASH, multiplicative, parse_binary_expr)
	p.led(lexer.ASTERISK, multiplicative, parse_binary_expr)
	p.led(lexer.REMAINDER, multiplicative, parse_binary_expr)
	p.led(lexer.EXPONENTIATION, exponentiation, parse_binary_expr)

	p.nud(lexer.INTEGER, parse_primary_expr)
	p.nud(lexer.FLOAT, parse_primary_expr)
	p.nud(lexer.IDENTIFIER, parse_primary_expr)

	p.nud(lexer.OPEN_PAREN, parse_grouping_expr)
	return p
}

func watParser() *parser {
	p := &parser{
		tokens:     []lexer.Token{},
		pos:        0,
		stmtLookup: &map[lexer.TokenKind]stmtHandler{},
		nudLookup:  &map[lexer.TokenKind]nudHandler{},
		ledLookup:  &map[lexer.TokenKind]ledHandler{},
		bpLookup:   &map[lexer.TokenKind]bindingPower{},
		filetype:   lexer.Wat,
	}
	return p
}

func (p *parser) newState(tokens []lexer.Token) {
	p.tokens = tokens
	p.pos = 0
	// TODO: reset diagnostics
}
