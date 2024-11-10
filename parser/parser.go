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
		stmt := parseStmt(p)
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
	p := baseParser()
	p.filetype = lexer.Vs

	p.led(lexer.ASSIGNMENT, assignment, parseAssignmentExpr)

	p.led(lexer.GREATER, relational, parseBinaryExpr)
	p.led(lexer.LESS, relational, parseBinaryExpr)
	p.led(lexer.GREATER_EQUAL, relational, parseBinaryExpr)
	p.led(lexer.LESS_EQUAL, relational, parseBinaryExpr)
	p.led(lexer.EQUAL, relational, parseBinaryExpr)
	p.led(lexer.NOT_EQUAL, relational, parseBinaryExpr)

	p.led(lexer.AND, logical, parseBinaryExpr)
	p.led(lexer.OR, logical, parseBinaryExpr)

	p.led(lexer.PLUS, additive, parseBinaryExpr)
	p.led(lexer.DASH, additive, parseBinaryExpr)
	p.led(lexer.SLASH, multiplicative, parseBinaryExpr)
	p.led(lexer.ASTERISK, multiplicative, parseBinaryExpr)
	p.led(lexer.REMAINDER, multiplicative, parseBinaryExpr)
	p.led(lexer.EXPONENTIATION, exponentiation, parseBinaryExpr)
	p.led(lexer.OPEN_PAREN, call, parseCallExpr)
	p.led(lexer.DOUBLE_COLON, member, parseMemberExpr)

	p.nud(lexer.FALSE, parsePrimaryExpr)
	p.nud(lexer.TRUE, parsePrimaryExpr)
	p.nud(lexer.INTEGER, parsePrimaryExpr)
	p.nud(lexer.FLOAT, parsePrimaryExpr)
	p.nud(lexer.IDENTIFIER, parsePrimaryExpr)

	p.nud(lexer.OPEN_PAREN, parseGroupingExpr)

	p.nud(lexer.DASH, parsePrefixExpr)
	p.nud(lexer.NOT, parsePrefixExpr)

	p.stmt(lexer.USE, parseUseStmt)
	p.stmt(lexer.RETURN, parseReturnStmt)
	p.stmt(lexer.LET, parseVariableDeclarationStmt)
	p.stmt(lexer.IF, parseIfStmt)
	p.stmt(lexer.FN, parseFunctionStmt)
	p.stmt(lexer.PUB, parsePublicStmt)
	p.stmt(lexer.EXTERN, parseExternStmt)

	return p
}

// TODO
func watParser() *parser {
	p := baseParser()
	p.filetype = lexer.Wat
	return p
}

func baseParser() *parser {
	p := &parser{
		tokens:     []lexer.Token{},
		pos:        0,
		stmtLookup: &map[lexer.TokenKind]stmtHandler{},
		nudLookup:  &map[lexer.TokenKind]nudHandler{},
		ledLookup:  &map[lexer.TokenKind]ledHandler{},
		bpLookup:   &map[lexer.TokenKind]bindingPower{},
		filetype:   lexer.Unrecognized,
	}
	return p
}

func (p *parser) newState(tokens []lexer.Token) {
	p.tokens = tokens
	p.pos = 0
	// TODO: reset diagnostics
}
