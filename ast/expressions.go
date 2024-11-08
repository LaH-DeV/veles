package ast

import (
	"fmt"
	"strconv"

	"github.com/LaH-DeV/veles/lexer"
)

type BinaryExpr struct {
	Left     Expr
	Operator lexer.Token
	Right    Expr
}

func (n BinaryExpr) expr() {}
func (n BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Operator.Value, n.Right.String())
}

type SymbolExpr struct {
	Value string
}

func (n SymbolExpr) expr() {}
func (n SymbolExpr) String() string {
	return fmt.Sprintf("%s", n.Value)
}

type IntegerExpr struct {
	Value int64
}

func (n IntegerExpr) expr() {}
func (n IntegerExpr) String() string {
	return fmt.Sprintf("%d", n.Value)
}

type FloatExpr struct {
	Value float64
}

func (n FloatExpr) expr() {}
func (n FloatExpr) String() string {
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}
