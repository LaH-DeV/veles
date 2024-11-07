package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

type parser struct {
	tokens []lexer.Token
	pos    int

	stmt_lookup *map[lexer.TokenKind]stmt_handler
	nud_lookup  *map[lexer.TokenKind]nud_handler
	led_lookup  *map[lexer.TokenKind]led_handler
	bp_lookup   *map[lexer.TokenKind]binding_power

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
		tokens:      []lexer.Token{},
		pos:         0,
		stmt_lookup: &map[lexer.TokenKind]stmt_handler{},
		nud_lookup:  &map[lexer.TokenKind]nud_handler{},
		led_lookup:  &map[lexer.TokenKind]led_handler{},
		bp_lookup:   &map[lexer.TokenKind]binding_power{},
		filetype:    lexer.Vs,
	}

	p.led(lexer.PLUS, additive, parse_binary_expr)
	p.led(lexer.DASH, additive, parse_binary_expr)
	p.led(lexer.SLASH, multiplicative, parse_binary_expr)
	p.led(lexer.ASTERISK, multiplicative, parse_binary_expr)
	p.led(lexer.REMAINDER, multiplicative, parse_binary_expr)

	p.nud(lexer.IDENTIFIER, parse_primary_expr)
	return p
}

func watParser() *parser {
	p := &parser{
		tokens:      []lexer.Token{},
		pos:         0,
		stmt_lookup: &map[lexer.TokenKind]stmt_handler{},
		nud_lookup:  &map[lexer.TokenKind]nud_handler{},
		led_lookup:  &map[lexer.TokenKind]led_handler{},
		bp_lookup:   &map[lexer.TokenKind]binding_power{},
		filetype:    lexer.Wat,
	}
	return p
}

func (p *parser) newState(tokens []lexer.Token) {
	p.tokens = tokens
	p.pos = 0
	// TODO: reset diagnostics
}
