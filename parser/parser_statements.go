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
	expression := parseExpr(p, defaultBp)

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

func parseVariableDeclarationStmt(p *parser) ast.Stmt {
	p.advance() // Skip the LET token for now

	varType := parseType(p).Value
	varName := p.expect(lexer.IDENTIFIER).Value

	var expr ast.Expr = nil
	if p.currentTokenKind() == lexer.ASSIGNMENT {
		p.advance()
		expr = *parseExpr(p, defaultBp)
	}

	// TODO.
	if expr == nil {
		expr = &ast.IntegerExpr{
			Value: 0,
		}
	}

	p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE, lexer.EOF)

	return &ast.VariableDeclarationStmt{
		VarType: varType,
		VarName: varName,
		Value:   expr,
	}
}

func parseDropStmt(p *parser) ast.Stmt {
	p.advance()

	var expr ast.Expr = nil
	if p.currentTokenKind() != lexer.SEMICOLON && p.currentTokenKind() != lexer.NEWLINE {
		res := parseExpr(p, defaultBp)
		p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE)
		if res != nil {
			expr = *res
		}
	}
	return &ast.DropStmt{
		Value: expr,
	}
}

func parseReturnStmt(p *parser) ast.Stmt {
	p.advance()

	var expr ast.Expr = nil
	if p.currentTokenKind() != lexer.SEMICOLON && p.currentTokenKind() != lexer.NEWLINE {
		res := parseExpr(p, defaultBp)
		p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE)
		if res != nil {
			expr = *res
		}
	}
	return &ast.ReturnStmt{
		Value: expr,
	}
}

// TODO - it will be enhanced in the future
func parseUseStmt(p *parser) ast.Stmt {
	p.advance() // Skip the USE token

	moduleName := p.expect(lexer.IDENTIFIER).Value

	var functions []string = make([]string, 0)

	if p.currentTokenKind() == lexer.DOUBLE_COLON {
		p.advance()
		var parens bool = false
		if p.currentTokenKind() == lexer.OPEN_PAREN {
			parens = true
			p.advance()
		}
		for p.currentTokenKind() != lexer.SEMICOLON && p.currentTokenKind() != lexer.NEWLINE && p.currentTokenKind() != lexer.CLOSE_PAREN {
			ident := p.expect(lexer.IDENTIFIER).Value
			functions = append(functions, ident)
			if p.currentTokenKind() != lexer.COMMA {
				break
			}
			p.advance()
		}
		if parens {
			p.expect(lexer.CLOSE_PAREN)
		}
	}

	p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE, lexer.EOF)

	return &ast.UseStmt{
		Module:    moduleName,
		Functions: functions,
	}
}
