package ast

type Node interface {
	String() string
}

type Stmt interface {
	Node
	stmt()
}

type Expr interface {
	Node
	expr()
}

type Type interface {
	_type()
}

type FunctionParameter struct {
	ParamName string
	ParamType string
}

func (n FunctionParameter) String() string {
	return n.ParamType + " " + n.ParamName
}
