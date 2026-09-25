package lexer

import (
	"fmt"

	"github.com/LaH-DeV/veles/source"
)

type TokenKind int

const (
	EOF TokenKind = iota
	Illegal

	// Literals
	Ident
	Int
	Float
	String // may carry interpolation parts
	Char

	// Punctuation
	LParen   // (
	RParen   // )
	LBrace   // {
	RBrace   // }
	LBracket // [
	RBracket // ]
	Comma    // ,
	Semi     // ; (explicit or auto-inserted)
	Colon    // :
	DblColon // ::
	Dot      // .
	SafeDot  // ?.
	Elvis    // ?:
	OrFail   // ?!
	Coalesce // ??
	Question // ?
	FatArrow // =>
	Arrow    // ->
	At       // @
	Amp      // &
	Pipe     // |
	Range    // ..
	RangeLt  // ..<
	Ellipsis // ... (variadic parameter, spread argument)
	Under    // _

	// Operators
	Assign    // =
	PlusEq    // +=
	MinusEq   // -=
	StarEq    // *=
	SlashEq   // /=
	PercentEq // %=
	Plus      // +
	Minus     // -
	Star      // *
	Slash     // /
	Percent   // %
	WrapPlus  // +%
	WrapMinus // -%
	WrapStar  // *%
	Eq        // ==
	NotEq     // !=
	Lt        // <
	LtEq      // <=
	Gt        // >
	GtEq      // >=
	AndAnd    // &&
	OrOr      // ||
	Bang      // !
	Caret     // ^ (bitwise xor)
	Tilde     // ~ (bitwise not)
	Shl       // <<
	Shr       // >>

	// Keywords
	keywordStart
	KwFun
	KwVal
	KwVar
	KwConst
	KwIf
	KwElse
	KwLoop
	KwBreak
	KwContinue
	KwReturn
	KwStruct
	KwTrait
	KwImpl
	KwSealed
	KwPub
	KwPrivate
	KwInternal  // the unwritten module level, spelled out (M5)
	KwProtected // `protected var`: a field assigned only by its type (D22)
	KwUse
	KwWhen
	KwIs
	KwAs
	KwIn
	KwThrows
	KwThrow
	KwSuspends
	KwTry
	KwAsync
	KwAwait
	KwScope
	KwGather
	KwRace
	KwWith
	KwUnsafe
	KwExtern
	KwMut
	KwOverride
	KwStatic
	KwTrue
	KwFalse
	KwNull
	KwSelf   // self
	KwSelfTy // Self
	KwRaw
	KwType
	KwEnum
	KwFor
	// Reserved for future use; lexed as keywords so they cannot be identifiers.
	KwReserved
	keywordEnd
)

var keywords = map[string]TokenKind{
	"fun":       KwFun,
	"val":       KwVal,
	"var":       KwVar,
	"const":     KwConst,
	"if":        KwIf,
	"else":      KwElse,
	"loop":      KwLoop,
	"break":     KwBreak,
	"continue":  KwContinue,
	"return":    KwReturn,
	"struct":    KwStruct,
	"trait":     KwTrait,
	"implement": KwImpl,
	"sealed":    KwSealed,
	"public":    KwPub,
	"private":   KwPrivate,
	"internal":  KwInternal,
	"protected": KwProtected,
	"use":       KwUse,
	"when":      KwWhen,
	"is":        KwIs,
	"as":        KwAs,
	"in":        KwIn,
	"throws":    KwThrows,
	"throw":     KwThrow,
	"suspends":  KwSuspends,
	"try":       KwTry,
	"async":     KwAsync,
	"await":     KwAwait,
	"scope":     KwScope,
	"gather":    KwGather,
	"race":      KwRace,
	"with":      KwWith,
	"unsafe":    KwUnsafe,
	"extern":    KwExtern,
	"mut":       KwMut,
	"override":  KwOverride,
	"static":    KwStatic,
	"true":      KwTrue,
	"false":     KwFalse,
	"null":      KwNull,
	"self":      KwSelf,
	"Self":      KwSelfTy,
	"raw":       KwRaw,
	"type":      KwType,
	"enum":      KwEnum,
	"for":       KwFor,

	// reserved
	"match": KwReserved, "defer": KwReserved, "go": KwReserved,
	"yield": KwReserved, "where": KwReserved, "super": KwReserved, "this": KwReserved,
	"catch": KwReserved, "interface": KwReserved, "class": KwReserved,
	"while": KwReserved, "do": KwReserved, "finally": KwReserved,
	"import": KwReserved, "package": KwReserved, "module": KwReserved, "let": KwReserved,
}

var kindNames = map[TokenKind]string{
	EOF: "end of file", Illegal: "illegal token", Ident: "identifier", Int: "integer literal",
	Float: "float literal", String: "string literal", Char: "character literal",
	LParen: "(", RParen: ")", LBrace: "{", RBrace: "}", LBracket: "[", RBracket: "]",
	Comma: ",", Semi: "newline", Colon: ":", DblColon: "::", Dot: ".", SafeDot: "?.",
	Elvis: "?:", OrFail: "?!", Coalesce: "??", Question: "?", FatArrow: "=>", Arrow: "->", At: "@", Amp: "&", Pipe: "|",
	Range: "..", RangeLt: "..<", Ellipsis: "...", Under: "_",
	Assign: "=", PlusEq: "+=", MinusEq: "-=", StarEq: "*=", SlashEq: "/=", PercentEq: "%=",
	Plus: "+", Minus: "-", Star: "*", Slash: "/", Percent: "%", WrapPlus: "+%",
	WrapMinus: "-%", WrapStar: "*%", Eq: "==", NotEq: "!=", Lt: "<", LtEq: "<=", Gt: ">",
	GtEq: ">=", AndAnd: "&&", OrOr: "||", Bang: "!", Caret: "^", Tilde: "~", Shl: "<<", Shr: ">>",
}

func (k TokenKind) String() string {
	if s, ok := kindNames[k]; ok {
		return s
	}
	for name, kind := range keywords {
		if kind == k && k != KwReserved {
			return "'" + name + "'"
		}
	}
	if k == KwReserved {
		return "reserved keyword"
	}
	return fmt.Sprintf("token(%d)", int(k))
}

func (k TokenKind) IsKeyword() bool { return k > keywordStart && k < keywordEnd }

// StringPart is one piece of a (possibly interpolated) string literal.
// Either Text is set (a literal run, escapes already processed) or Expr is
// set (raw source of an interpolated expression) along with its span.
type StringPart struct {
	Text   string
	IsExpr bool
	Expr   string
	Span   source.Span
}

type Token struct {
	Kind  TokenKind
	Text  string // raw source text
	Span  source.Span
	Parts []StringPart // only for String
	// AutoSemi marks a Semi that was inserted by the newline rule.
	AutoSemi bool
	// Doc is the documentation comment (`/// ...` lines or `/** ... */`)
	// written directly above this token, markers stripped; the parser
	// attaches it to the declaration the token starts.
	Doc string
}

func (t Token) String() string {
	switch t.Kind {
	case Ident, Int, Float, String, Char:
		return fmt.Sprintf("%s %q", t.Kind, t.Text)
	case Semi:
		if t.AutoSemi {
			return "newline"
		}
		return "';'"
	default:
		return t.Kind.String()
	}
}

// Describe renders a token for diagnostics.
func (t Token) Describe() string {
	switch t.Kind {
	case Ident:
		return fmt.Sprintf("identifier '%s'", t.Text)
	case Int, Float, String, Char:
		return fmt.Sprintf("%s '%s'", t.Kind, t.Text)
	case Semi:
		if t.AutoSemi {
			return "end of line"
		}
		return "';'"
	case EOF:
		return "end of file"
	}
	if t.Kind.IsKeyword() {
		return fmt.Sprintf("keyword '%s'", t.Text)
	}
	return fmt.Sprintf("'%s'", t.Kind.String())
}
