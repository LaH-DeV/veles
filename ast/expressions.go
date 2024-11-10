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
	return n.Value
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

type AssignmentExpr struct {
	Assigne       Expr
	AssignedValue Expr
}

func (n AssignmentExpr) expr() {}
func (n AssignmentExpr) String() string {
	return n.Assigne.String() + " = " + n.Assigne.String()
}

type PrefixExpr struct {
	Operator lexer.Token
	Right    Expr
}

func (n PrefixExpr) expr() {}
func (n PrefixExpr) String() string {
	return n.Operator.Value + n.Right.String()
}

type CallExpr struct {
	Callee    Expr
	Arguments []Expr
}

func (n CallExpr) expr() {}
func (n CallExpr) String() string {
	var str string
	str += n.Callee.String() + "("
	for i, arg := range n.Arguments {
		if i > 0 {
			str += ", "
		}
		str += arg.String()
	}
	str += ")"
	return str
}

type MemberExpr struct {
	Container Expr
	Member    string
}

func (n MemberExpr) expr() {}
func (n MemberExpr) String() string {
	return fmt.Sprintf("%s::%s", n.Container.String(), n.Member)
}

type BooleanExpr struct {
	Value bool
}

func (n BooleanExpr) expr() {}
func (n BooleanExpr) String() string {
	return fmt.Sprintf("%v", n.Value)
}
