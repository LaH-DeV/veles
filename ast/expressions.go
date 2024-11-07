package ast

import (
	"fmt"

	"github.com/LaH-DeV/veles/lexer"
)

type BinaryExpr struct {
	Left     Expr
	Operator lexer.Token
	Right    Expr
}

func (n BinaryExpr) expr() {}
func (n BinaryExpr) String() string {
	return fmt.Sprintf("Binary: %s \"%s\" (operator) %s", n.Left.String(), n.Operator.Value, n.Right.String())
}

type SymbolExpr struct {
	Value string
}

func (n SymbolExpr) expr() {}
func (n SymbolExpr) String() string {
	return fmt.Sprintf("\"%s\" (symbol)", n.Value)
}
