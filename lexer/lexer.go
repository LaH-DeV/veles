package lexer

import "fmt"

type TokenKind int

const (
	EOF TokenKind = iota
	STRING
	INTEGER
	FLOAT
	INT_32
	INT_64
	FLOAT_32
	FLOAT_64
	IDENTIFIER

	OPEN_PAREN
	CLOSE_PAREN

	MODULE
	PARAM
	RESULT
	TYPE
	FUNC
	IMPORT
	EXPORT
)

var reserved_lu map[string]TokenKind = map[string]TokenKind{
	"module": MODULE,
	"param":  PARAM,
	"result": RESULT,
	"type":   TYPE,
	"func":   FUNC,
	"import": IMPORT,
	"export": EXPORT,
}

var reserved_types_lu map[string]TokenKind = map[string]TokenKind{
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
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

func newUniqueToken(kind TokenKind, value string) Token {
	return Token{
		kind, value,
	}
}
