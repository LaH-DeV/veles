// Tokens. The order of `Kind`'s members is fixed: the harness compares a
// kind's `value` with the bootstrap compiler's number for it, so a new kind
// goes where it goes there.
use source { Span }

public enum Kind {
  EOF
  Illegal
  // literals
  Ident
  Int
  Float
  String
  Char
  // punctuation
  LParen
  RParen
  LBrace
  RBrace
  LBracket
  RBracket
  Comma
  Semi
  Colon
  DblColon
  Dot
  SafeDot
  Elvis
  OrFail
  Coalesce
  Question
  FatArrow
  Arrow
  At
  Amp
  Pipe
  Range
  RangeLt
  Ellipsis
  Under
  // operators
  Assign
  PlusEq
  MinusEq
  StarEq
  SlashEq
  PercentEq
  Plus
  Minus
  Star
  Slash
  Percent
  WrapPlus
  WrapMinus
  WrapStar
  Eq
  NotEq
  Lt
  LtEq
  Gt
  GtEq
  AndAnd
  OrOr
  Bang
  Caret
  Tilde
  Shl
  Shr
  // keywords, between the two markers
  KeywordStart
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
  KwImplement
  KwSealed
  KwPublic
  KwPrivate
  KwInternal
  KwProtected
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
  KwThis
  KwSelfType
  KwRaw
  KwType
  KwEnum
  KwFor
  KwDo
  KwCatch
  // reserved for future use: lexed as keywords so they cannot be names
  KwReserved
  KeywordEnd
}

/// The keywords and the reserved words.
public const KEYWORDS: Map<string, Kind> = [
  "fun": Kind.KwFun, "val": Kind.KwVal, "var": Kind.KwVar, "const": Kind.KwConst, "if": Kind.KwIf,
  "else": Kind.KwElse, "loop": Kind.KwLoop, "break": Kind.KwBreak, "continue": Kind.KwContinue,
  "return": Kind.KwReturn, "struct": Kind.KwStruct, "trait": Kind.KwTrait, "implement": Kind.KwImplement,
  "sealed": Kind.KwSealed, "public": Kind.KwPublic, "private": Kind.KwPrivate, "internal": Kind.KwInternal,
  "protected": Kind.KwProtected, "use": Kind.KwUse, "when": Kind.KwWhen, "is": Kind.KwIs, "as": Kind.KwAs,
  "in": Kind.KwIn, "throws": Kind.KwThrows, "throw": Kind.KwThrow, "suspends": Kind.KwSuspends,
  "try": Kind.KwTry, "async": Kind.KwAsync, "await": Kind.KwAwait, "scope": Kind.KwScope,
  "gather": Kind.KwGather, "race": Kind.KwRace, "with": Kind.KwWith, "unsafe": Kind.KwUnsafe,
  "extern": Kind.KwExtern, "mut": Kind.KwMut, "override": Kind.KwOverride, "static": Kind.KwStatic,
  "true": Kind.KwTrue, "false": Kind.KwFalse, "null": Kind.KwNull, "this": Kind.KwThis,
  "Self": Kind.KwSelfType, "raw": Kind.KwRaw, "type": Kind.KwType, "enum": Kind.KwEnum, "for": Kind.KwFor,
  "do": Kind.KwDo, "catch": Kind.KwCatch,
  "match": Kind.KwReserved, "defer": Kind.KwReserved, "go": Kind.KwReserved, "yield": Kind.KwReserved,
  "where": Kind.KwReserved, "super": Kind.KwReserved, "interface": Kind.KwReserved,
  "class": Kind.KwReserved, "while": Kind.KwReserved, "finally": Kind.KwReserved,
  "import": Kind.KwReserved, "package": Kind.KwReserved, "module": Kind.KwReserved, "let": Kind.KwReserved,
]

const fun kindNames(): Map<Kind, string> {
  val names: MutableMap<Kind, string> = [
    Kind.EOF: "end of file", Kind.Illegal: "illegal token", Kind.Ident: "identifier",
    Kind.Int: "integer literal", Kind.Float: "float literal", Kind.String: "string literal",
    Kind.Char: "character literal", Kind.Semi: "newline", Kind.KwReserved: "reserved keyword",
  ]
  val symbols = [
    "(", ")", "{", "}", "[", "]", ",", ";", ":", "::", ".", "?.", "?:", "?!", "??", "?", "=>", "->", "@",
    "&", "|", "..", "..<", "...", "_", "=", "+=", "-=", "*=", "/=", "%=", "+", "-", "*", "/", "%", "+%",
    "-%", "*%", "==", "!=", "<", "<=", ">", ">=", "&&", "||", "!", "^", "~", "<<", ">>",
  ]
  // the symbols are in the order of their kinds, from `(` on; Semi has a word
  var kind = Kind.LParen.value
  loop (text in symbols) {
    val k = Kind.fromValue(kind) ?: panic("kindNames: a symbol past the last operator")
    if (!names.containsKey(k)) names.set(k, text)
    kind += 1
  }
  loop ((word, k) in KEYWORDS) {
    if (k != Kind.KwReserved && !names.containsKey(k)) names.set(k, "'$word'")
  }
  names.toMap()
}

const NAMES: Map<Kind, string> = kindNames()

/// How a kind is spelled in a message: the symbol (`(`, `=>`), the quoted
/// keyword (`'fun'`), or a word for a class of tokens (`identifier`).
public fun kindText(k: Kind): string = NAMES.get(k) ?: "token(${k.value})"

public fun isKeyword(k: Kind): bool = k > Kind.KeywordStart && k < Kind.KeywordEnd

/// One piece of a string literal: a run of text with its escapes decoded,
/// or the source of an interpolated expression and where it is.
public struct StringPart {
  public text:   string = ""
  public isExpr: bool = false
  public expr:   string = ""
  public span:   Span = Span(file: null, start: 0, end: 0)
}

public struct Token {
  public var kind:     Kind
  public var text:     string  // the source text
  public var span:     Span
  public var parts:    List<StringPart> = []  // a String's or a Char's
  public var autoSemi: bool = false           // a Semi the newline rule inserted
  public var doc:      string = ""            // the documentation comment written directly above

  /// The token as a diagnostic names it: `identifier 'x'`, `keyword 'fun'`,
  /// `end of line`, `')'`.
  public fun describe(): string {
    when (this.kind) {
      Kind.Ident => return "identifier '${this.text}'"
      Kind.Int, Kind.Float, Kind.String, Kind.Char => return "${kindText(this.kind)} '${this.text}'"
      Kind.Semi => return if (this.autoSemi) "end of line" else "';'"
      Kind.EOF => return "end of file"
      else => { }
    }
    if (isKeyword(this.kind)) return "keyword '${this.text}'"
    "'${kindText(this.kind)}'"
  }
}

/// A `// …` line or a `/* … */` block, as written.
public struct Comment {
  public span: Span
  public text: string
}
