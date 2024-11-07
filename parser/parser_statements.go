package parser

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

func parse_stmt(p *parser) ast.Stmt {
	for p.currentTokenKind() == lexer.NEWLINE {
		p.advance()
	}

	stmt_fn, exists := (*p.stmt_lookup)[p.currentTokenKind()]

	if exists {
		return stmt_fn(p)
	}

	return parse_expression_stmt(p)
}

func parse_expression_stmt(p *parser) *ast.ExpressionStmt {
	expression := parse_expr(p, default_bp)

	p.expectOneOf(lexer.SEMICOLON, lexer.NEWLINE, lexer.EOF)

	if expression == nil {
		return nil
	}

	return &ast.ExpressionStmt{
		Expression: *expression,
	}
}
