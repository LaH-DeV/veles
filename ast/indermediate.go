package ast

type FunctionParameter struct {
	ParamName string
	ParamType string
}

func (n FunctionParameter) String() string {
	return n.ParamType + " " + n.ParamName
}
