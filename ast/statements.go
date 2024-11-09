package ast

import "github.com/LaH-DeV/veles/lexer"

type Program struct {
	Statements []Stmt

	Filetype lexer.Filetype
	Filename string
}

func (n Program) stmt() {}
func (n *Program) String() string {
	var str string
	for _, stmt := range n.Statements {
		str += stmt.String() + "\n"
	}
	return str
}

type ExpressionStmt struct {
	Expression Expr
}

func (n ExpressionStmt) stmt() {}
func (n ExpressionStmt) String() string {
	return n.Expression.String()
}

type FunctionDeclarationStmt struct {
	Exported   bool
	Identifier string
	Params     []FunctionParameter
	ReturnType string // TODO
}

func (n FunctionDeclarationStmt) stmt() {}
func (n FunctionDeclarationStmt) String() string {
	var str string
	if n.Exported {
		str += "pub "
	}
	str += "fn "
	if len(n.ReturnType) > 0 {
		str += n.ReturnType + " "
	}
	str += ":: " + n.Identifier + "("
	for i, param := range n.Params {
		if i > 0 {
			str += ", "
		}
		str += param.String()
	}
	str += ")"
	return str
}

type FunctionStmt struct {
	Exported   bool
	Identifier string
	Params     []FunctionParameter
	ReturnType string // TODO
	Body       []Stmt
}

func (n FunctionStmt) stmt() {}
func (n FunctionStmt) String() string {
	var str string
	if n.Exported {
		str += "pub "
	}
	str += "fn "
	if len(n.ReturnType) > 0 {
		str += n.ReturnType + " "
	}
	str += ":: " + n.Identifier
	if len(n.Params) > 0 {
		str += "("
		for i, param := range n.Params {
			if i > 0 {
				str += ", "
			}
			str += param.String()
		}
		str += ")"
	}
	str += " {\n"
	for _, stmt := range n.Body {
		str += "\t" + stmt.String() + "\n"
	}
	str += "}"
	return str
}

type VariableDeclarationStmt struct {
	Exported bool
	VarType  string
	VarName  string
	Value    Expr
}

func (n VariableDeclarationStmt) stmt() {}
func (n VariableDeclarationStmt) String() string {
	str := ""
	if n.Exported {
		str += "pub "
	}
	str += "let "
	if n.Value == nil {
		str += n.VarType + " " + n.VarName
	} else {
		str += n.VarType + " " + n.VarName + " = " + n.Value.String()
	}
	return str
}

type ReturnStmt struct {
	Value Expr
}

func (n *ReturnStmt) stmt() {}
func (n *ReturnStmt) String() string {
	if n.Value == nil {
		return "return"
	} else {
		return "return " + n.Value.String()
	}
}

type UseStmt struct {
	Module   string
	Segments []string
}

func (n *UseStmt) stmt() {}
func (n *UseStmt) String() string {
	if len(n.Segments) > 0 {
		var str string = "use " + n.Module
		for _, f := range n.Segments {
			str += "::" + f
		}
		return str
	}
	return "use " + n.Module
}

type ExternStmt struct {
	Statements []Stmt
}

func (n *ExternStmt) stmt() {}
func (n *ExternStmt) String() string {
	str := "extern {\n"
	for _, stmt := range n.Statements {
		str += "\t" + stmt.String() + "\n"
	}
	str += "}"
	return str
}
