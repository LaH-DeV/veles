package parser

import (
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

	p.skipNewlines()

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

func parseFunctionDeclaration(p *parser) ast.Stmt {
	initialToken := p.advance()
	var pub bool = false
	var extern bool = false
	if initialToken.Kind == lexer.EXTERN {
		extern = true
		p.advance() // Skip the FN token
	} else if initialToken.Kind == lexer.PUB {
		pub = true
		p.advance() // Skip the FN token
	}

	var returnType string
	if p.currentTokenKind() != lexer.DOUBLE_COLON {
		returnType = parseType(p).Value
	}
	p.expect(lexer.DOUBLE_COLON)

	functionName := p.expectError(lexer.IDENTIFIER, "Expected an identifier for function declaration").Value

	var parameters []ast.FunctionParameter
	if p.currentTokenKind() == lexer.OPEN_PAREN {
		parameters = parseFunctionParameters(p)
	} else {
		parameters = make([]ast.FunctionParameter, 0)
	}

	return &ast.FunctionDeclaration{
		Extern:     extern,
		Exported:   pub,
		Identifier: functionName,
		Params:     parameters,
		ReturnType: returnType,
	}
}

func parseFunctionStmt(p *parser) ast.Stmt {
	stmt := parseFunctionDeclaration(p)

	fn, ok := stmt.(*ast.FunctionDeclaration)
	if !ok {
		return nil
	}

	body := parseBlockStmt(p)
	return &ast.FunctionStmt{
		Exported:   fn.Exported,
		Identifier: fn.Identifier,
		Params:     fn.Params,
		ReturnType: fn.ReturnType,
		Body:       body,
	}
}

func parseVariableDeclarationStmt(p *parser) ast.Stmt {
	var pub bool = false

	if p.currentTokenKind() == lexer.PUB {
		pub = true
		p.advance() // PUB token
		p.advance() // LET token
	} else {
		p.advance() // LET token
	}

	varType := parseType(p).Value
	varName := p.expect(lexer.IDENTIFIER).Value

	var expr ast.Expr = nil
	if p.currentTokenKind() == lexer.ASSIGNMENT {
		p.advance()
		res := parseExpr(p, defaultBp)
		if res != nil {
			expr = *res
		}
	}

	// TODO.
	if expr == nil {
		expr = &ast.IntegerExpr{
			Value: 0,
		}
	}
	p.skipNewlines()

	return &ast.VariableDeclarationStmt{
		Exported: pub, // should be allowed only for unscoped variables
		VarType:  varType,
		VarName:  varName,
		Value:    expr,
	}
}

func parseReturnStmt(p *parser) ast.Stmt {
	p.advance()

	var expr ast.Expr = nil
	if p.currentTokenKind() != lexer.NEWLINE {
		res := parseExpr(p, defaultBp)
		if res == nil {
			p.skipNewlines()
		} else {
			expr = *res
		}
	}
	return &ast.ReturnStmt{
		Value: expr,
	}
}

func parseUseStmt(p *parser) ast.Stmt {
	p.advance() // Skip the USE token

	moduleName := p.expect(lexer.IDENTIFIER).Value
	var alias string

	var segments []string = make([]string, 0)

	if p.currentTokenKind() == lexer.DOUBLE_COLON {
		p.advance()
		for p.currentTokenKind() != lexer.NEWLINE && p.currentTokenKind() != lexer.CLOSE_PAREN {
			ident := p.expect(lexer.IDENTIFIER).Value
			segments = append(segments, ident)
			if p.currentTokenKind() != lexer.DOUBLE_COLON {
				break
			}
			p.advance() // Skip the DOUBLE_COLON token
		}
	}

	if p.currentTokenKind() == lexer.AS {
		p.advance()
		alias = p.expect(lexer.IDENTIFIER).Value
	}

	p.skipNewlines()

	return &ast.UseStmt{
		Module:   moduleName,
		Alias:    alias,
		Segments: segments,
	}
}

func parsePublicStmt(p *parser) ast.Stmt {
	switch p.peek().Kind {
	case lexer.FN:
		return parseFunctionStmt(p)
	case lexer.LET:
		return parseVariableDeclarationStmt(p)
	default:
		p.advance()
		// TODO: report error
		return nil
	}
}

func parseExternStmt(p *parser) ast.Stmt {
	switch p.peek().Kind {
	case lexer.FN:
		fn := parseFunctionDeclaration(p)
		return &ast.ExternStmt{
			Statement: fn,
		}
	default:
		p.advance()
		// TODO: report error
		return nil
	}
}
