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

	p.led(lexer.PLUS, additive, parseBinaryExpr)
	p.led(lexer.DASH, additive, parseBinaryExpr)
	p.led(lexer.SLASH, multiplicative, parseBinaryExpr)
	p.led(lexer.ASTERISK, multiplicative, parseBinaryExpr)
	p.led(lexer.REMAINDER, multiplicative, parseBinaryExpr)
	p.led(lexer.EXPONENTIATION, exponentiation, parseBinaryExpr)

	p.led(lexer.OPEN_PAREN, call, func(p *parser, left ast.Expr, bp bindingPower) ast.Expr {
		p.advance()
		symbExpr, ok := left.(ast.SymbolExpr)
		if !ok {
			return nil // TODO
		}
		fName := symbExpr.Value
		args := make([]ast.Expr, 0)
		if p.currentTokenKind() != lexer.CLOSE_PAREN {
			for {
				expr := *parseExpr(p, defaultBp)
				if expr == nil {
					break // TODO
				}
				args = append(args, expr)
				if p.currentTokenKind() != lexer.COMMA {
					break
				}
				p.advance()
			}
		}
		p.expect(lexer.CLOSE_PAREN)
		return &ast.CallExpr{
			Function:  fName,
			Arguments: args,
		}
	})

	p.nud(lexer.INTEGER, parsePrimaryExpr)
	p.nud(lexer.FLOAT, parsePrimaryExpr)
	p.nud(lexer.IDENTIFIER, parsePrimaryExpr)

	p.nud(lexer.OPEN_PAREN, parseGroupingExpr)

	p.stmt(lexer.USE, parseUseStmt)
	p.stmt(lexer.RETURN, parseReturnStmt)
	p.stmt(lexer.DROP, parseDropStmt)
	p.stmt(lexer.LET, parseVariableDeclarationStmt)
	p.stmt(lexer.FN, func(p *parser) ast.Stmt {
		return parseFunctionDeclarationStmt(p, false)
	})
	p.stmt(lexer.PUB, func(p *parser) ast.Stmt {
		p.advance()
		switch p.currentTokenKind() {
		case lexer.FN:
			return parseFunctionDeclarationStmt(p, true)
		default:
			// TODO: report error
			return nil
		}
	})

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
