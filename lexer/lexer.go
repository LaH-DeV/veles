package lexer

import "fmt"

type TokenKind int

const (
	EOF TokenKind = iota
	STRING
	INTEGER
	FLOAT
	IDENTIFIER

	INT_32
	INT_64
	FLOAT_32
	FLOAT_64
	BOOL

	OPEN_PAREN
	CLOSE_PAREN
	OPEN_CURLY
	CLOSE_CURLY
	DOUBLE_COLON
	COLON
	PLUS
	DASH
	SLASH
	ASTERISK
	REMAINDER
	EXPONENTIATION

	EQUALS
	ASSIGNMENT
	COMMA
	SEMICOLON

	MODULE
	PARAM
	RESULT
	TYPE
	FUNC
	IMPORT
	EXPORT
	RETURN
	FN
	IF
	PUB
	USE
	DROP
	LET
	EXTERN
	AS

	TRUE
	FALSE

	NEWLINE
)

var reserved_lu_wat map[string]TokenKind = map[string]TokenKind{
	"module": MODULE,
	"param":  PARAM,
	"result": RESULT,
	"type":   TYPE,
	"func":   FUNC,
	"import": IMPORT,
	"export": EXPORT,
	"return": RETURN,
	"drop":   DROP,
}

var reserved_lu_vs map[string]TokenKind = map[string]TokenKind{
	"module": MODULE,
	"fn":     FN,
	"pub":    PUB,
	"use":    USE,
	"return": RETURN,
	"let":    LET,
	"extern": EXTERN,
	"as":     AS,
	"if":     IF,

	"false": FALSE,
	"true":  TRUE,
}

var reserved_types_vs map[string]TokenKind = map[string]TokenKind{
	"i32": INT_32,
	"i64": INT_64,
	"f32": FLOAT_32,
	"f64": FLOAT_64,

	"bool": BOOL,
}

var reserved_types_wat map[string]TokenKind = map[string]TokenKind{
	"i32": INT_32,
	"i64": INT_64,
	"f32": FLOAT_32,
	"f64": FLOAT_64,
}

type Token struct {
	Kind  TokenKind
	Value string
}

func TokenKindString(kind TokenKind) string {
	switch kind {
	case EOF:
		return "eof"
	case IF:
		return "if"
	case DROP:
		return "drop"
	case BOOL:
		return "bool"
	case FALSE:
		return "false"
	case TRUE:
		return "true"
	case NEWLINE:
		return "newline"
	case STRING:
		return "string"
	case INT_32:
		return "i32"
	case INT_64:
		return "i64"
	case FLOAT_32:
		return "f32"
	case FLOAT_64:
		return "f64"
	case INTEGER:
		return "integer"
	case FLOAT:
		return "float"
	case IDENTIFIER:
		return "identifier"
	case OPEN_PAREN:
		return "open_paren"
	case CLOSE_PAREN:
		return "close_paren"
	case OPEN_CURLY:
		return "open_curly"
	case CLOSE_CURLY:
		return "close_curly"
	case DOUBLE_COLON:
		return "double_colon"
	case COLON:
		return "colon"
	case PLUS:
		return "plus"
	case DASH:
		return "dash"
	case SLASH:
		return "slash"
	case ASTERISK:
		return "asterisk"
	case REMAINDER:
		return "remainder"
	case EXPONENTIATION:
		return "exponentiation"
	case EQUALS:
		return "equals"
	case ASSIGNMENT:
		return "assignment"
	case COMMA:
		return "comma"
	case SEMICOLON:
		return "semicolon"
	case MODULE:
		return "module"
	case PARAM:
		return "param"
	case RESULT:
		return "result"
	case TYPE:
		return "type"
	case FUNC:
		return "func"
	case IMPORT:
		return "import"
	case EXPORT:
		return "export"
	case RETURN:
		return "return"
	case FN:
		return "fn"
	case PUB:
		return "pub"
	case USE:
		return "use"
	case LET:
		return "let"
	case EXTERN:
		return "extern"
	case AS:
		return "as"
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

func newUniqueToken(kind TokenKind, value string) Token {
	return Token{
		kind, value,
	}
}

func (tk Token) IsOneOfMany(expectedTokens ...TokenKind) bool {
	for _, expected := range expectedTokens {
		if expected == tk.Kind {
			return true
		}
	}

	return false
}
