package parser

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

func parseStmt(p *parser) ast.Stmt {
	if p.currentTokenKind() == lexer.EOF || p.currentTokenKind() == lexer.CLOSE_CURLY {
		return nil
	}

	stmt_fn, exists := (*p.stmtLookup)[p.currentTokenKind()]

	if exists {
		return stmt_fn(p)
	}

	return parseExpressionStmt(p)
}

func parseExpressionStmt(p *parser) *ast.ExpressionStmt {
	expression := parseExpr(p, default_bp)

	p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE, lexer.EOF)

	if expression == nil {
		return nil
	}

	return &ast.ExpressionStmt{
		Expression: *expression,
	}
}

func parseFunctionParameters(p *parser) []ast.FunctionParameter {
	p.expect(lexer.OPEN_PAREN)

	var parameters []ast.FunctionParameter = make([]ast.FunctionParameter, 0)

	for p.currentTokenKind() != lexer.CLOSE_PAREN {
		paramType := parseType(p).Value
		paramName := p.expect(lexer.IDENTIFIER).Value
		parameters = append(parameters, ast.FunctionParameter{
			ParamName: paramName,
			ParamType: paramType,
		})
		if p.currentTokenKind() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_PAREN)

	return parameters
}

func parseBlockStmt(p *parser) []ast.Stmt {
	p.expect(lexer.OPEN_CURLY)

	var statements []ast.Stmt = make([]ast.Stmt, 0)

	for p.currentTokenKind() != lexer.CLOSE_CURLY {
		p.skipNewlines()
		stmt := parseStmt(p)
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return statements
}

func parseType(p *parser) lexer.Token {
	return p.expectOneOf(lexer.INT_32, lexer.INT_64, lexer.FLOAT_32, lexer.FLOAT_64, lexer.IDENTIFIER, lexer.VOID)
}

func parseFunctionDeclarationStmt(p *parser, pub bool) ast.Stmt {
	p.advance()

	returnType := parseType(p).Value

	p.expect(lexer.DOUBLE_COLON)

	functionName := p.expectError(lexer.IDENTIFIER, fmt.Sprintf("Expected an identifier for function declaration")).Value
	parameters := parseFunctionParameters(p)

	body := parseBlockStmt(p)

	return &ast.FunctionDeclarationStmt{
		Exported:   pub,
		Identifier: functionName,
		Params:     parameters,
		ReturnType: returnType,
		Body:       body,
	}
}

func parseReturnStmt(p *parser) ast.Stmt {
	p.advance()

	var expr ast.Expr = nil
	if p.currentTokenKind() != lexer.SEMICOLON && p.currentTokenKind() != lexer.NEWLINE {
		res := parseExpr(p, default_bp)
		p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE)
		if res != nil {
			expr = *res
		}
	}
	return &ast.ReturnStmt{
		Value: expr,
	}
}
