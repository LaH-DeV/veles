package parser

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

func parse_binary_expr(p *parser, left ast.Expr, bp binding_power) ast.Expr {
	operatorToken := p.advance()
	right := parse_expr(p, default_bp)

	if right == nil {
		//panic(fmt.Sprintf("Veles :: Cannot create binary_expr from \"%s\"\n", lexer.TokenKindString(p.currentTokenKind())))
		return nil
	}

	return ast.BinaryExpr{
		Left:     left,
		Operator: operatorToken,
		Right:    *right,
	}
}

func parse_primary_expr(p *parser) ast.Expr {
	switch p.currentTokenKind() {
	case lexer.IDENTIFIER:
		return ast.SymbolExpr{Value: p.advance().Value}
	default:
		panic(fmt.Sprintf("Veles :: Cannot create primary_expr from \"%s\"\n", lexer.TokenKindString(p.currentTokenKind())))
	}
}

func parse_expr(p *parser, bp binding_power) *ast.Expr {
	nud_fn, exists := (*p.nud_lookup)[p.currentTokenKind()]

	if !exists {
		// fmt.Printf("No nud handler found for token %s (%s)\n", lexer.TokenKindString(p.currentTokenKind()), p.currentToken().Value)
		// panic(fmt.Sprintf("NUD Handler expected for token %s\n", lexer.TokenKindString(p.currentTokenKind())))
		return nil
	}

	left := nud_fn(p)

	for (*p.bp_lookup)[p.currentTokenKind()] > bp {
		led_fn, exists := (*p.led_lookup)[p.currentTokenKind()]

		if !exists {
			// panic(fmt.Sprintf("LED Handler expected for token %s\n", lexer.TokenKindString(p.currentTokenKind())))
			//fmt.Printf("LED Handler expected for token %s\n", lexer.TokenKindString(p.currentTokenKind()))
			return &left
		}

		left = led_fn(p, left, bp)
	}

	return &left
}
