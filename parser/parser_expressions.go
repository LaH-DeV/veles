package parser

import (
	"fmt"
	"strconv"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
)

func parseBinaryExpr(p *parser, left ast.Expr, bp bindingPower) ast.Expr {
	operatorToken := p.advance()
	right := parseExpr(p, bp)

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

func parsePrimaryExpr(p *parser) ast.Expr {
	switch p.currentTokenKind() {
	case lexer.INTEGER:
		integer, _ := strconv.ParseInt(p.advance().Value, 0, 64)
		// TODO: Handle errors
		return ast.IntegerExpr{
			Value: integer,
		}
	case lexer.FLOAT:
		number, _ := strconv.ParseFloat(p.advance().Value, 64)
		// TODO: Handle errors
		return &ast.FloatExpr{Value: number}
	case lexer.IDENTIFIER:
		return ast.SymbolExpr{Value: p.advance().Value}
	default:
		panic(fmt.Sprintf("Veles :: Cannot create primary_expr from \"%s\"\n", lexer.TokenKindString(p.currentTokenKind())))
	}
}

func parseGroupingExpr(p *parser) ast.Expr {
	p.expect(lexer.OPEN_PAREN)
	expr := parseExpr(p, defaultBp)
	p.expect(lexer.CLOSE_PAREN)
	if expr == nil {
		return nil
	}
	return *expr
}

func parseExpr(p *parser, bp bindingPower) *ast.Expr {
	p.skipNewlines()

	nudHandler, exists := (*p.nudLookup)[p.currentTokenKind()]

	if !exists {
		// fmt.Printf("No nud handler found for token %s (%s)\n", lexer.TokenKindString(p.currentTokenKind()), p.currentToken().Value)
		// panic(fmt.Sprintf("NUD Handler expected for token %s\n", lexer.TokenKindString(p.currentTokenKind())))
		return nil
	}

	p.skipNewlines()
	expression := nudHandler(p)

	for p.lookupBp(p.currentTokenKind()) > bp {
		p.skipNewlines()

		ledHandler, exists := (*p.ledLookup)[p.currentTokenKind()]
		if !exists {
			// panic(fmt.Sprintf("LED Handler expected for token %s\n", lexer.TokenKindString(p.currentTokenKind())))
			//fmt.Printf("LED Handler expected for token %s\n", lexer.TokenKindString(p.currentTokenKind()))
			// return &expression
			return nil
		}

		expression = ledHandler(p, expression, p.lookupBp(p.currentTokenKind()))
	}

	return &expression
}

func parseCallExpr(p *parser, left ast.Expr, bp bindingPower) ast.Expr {
	p.advance()
	if left == nil {
		return nil // TODO
	}
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
		Callee:    left,
		Arguments: args,
	}
}

func parseMemberExpr(p *parser, left ast.Expr, bp bindingPower) ast.Expr {
	p.advance()                          // Skip the DOUBLE_COLON token
	member := p.expect(lexer.IDENTIFIER) // for now, we'll just assume that the member is an identifier
	return &ast.MemberExpr{
		Container: left,
		Member:    member.Value,
	}
}
