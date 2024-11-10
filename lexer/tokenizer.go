package lexer

import (
	"fmt"
	"regexp"
)

type Filetype int

const (
	Unrecognized Filetype = iota
	Vs
	Wat
)

func NewLexer(filetype Filetype) *lexer {
	switch filetype {
	case Vs:
		return vsLexer()
	case Wat:
		return watLexer()
	default:
		return nil
	}
}

type regexPattern struct {
	regex   *regexp.Regexp
	handler regexHandler
}

type lexer struct {
	source string
	pos    int
	line   int
	Tokens []Token

	patterns *[]regexPattern
	keywords *map[string]TokenKind
	types    *map[string]TokenKind

	filetype Filetype
}

func (lex *lexer) Tokenize(source string) []Token {
	lex.newState(source)
	for !lex.at_eof() {
		matched := false
		for _, pattern := range *lex.patterns {
			loc := pattern.regex.FindStringIndex(lex.remainder())
			if loc != nil && loc[0] == 0 {
				pattern.handler(lex, pattern.regex)
				matched = true
				break // Exit the loop after the first match
			}
		}
		if !matched {
			// TODO: add diagnostics and error handling
			// we shouldn't panic here, but instead add a diagnostic and continue
			// the continuation will require a strategy
			panic(fmt.Sprintf("Veles :: lexer error: unrecognized token near '%v'", lex.remainder()))
		}
	}
	lex.push(newUniqueToken(EOF, "EOF"))
	return lex.Tokens
}

func (lex *lexer) newState(source string) {
	lex.source = source
	lex.pos = 0
	lex.line = 1
	lex.Tokens = make([]Token, 0)
	// TODO: reset diagnostics
}

func (lex *lexer) advanceN(n int) {
	lex.pos += n
}

func (lex *lexer) at() byte {
	return lex.source[lex.pos]
}

func (lex *lexer) advance() {
	lex.pos += 1
}

func (lex *lexer) remainder() string {
	return lex.source[lex.pos:]
}

func (lex *lexer) push(token Token) {
	lex.Tokens = append(lex.Tokens, token)
}

func (lex *lexer) at_eof() bool {
	return lex.pos >= len(lex.source)
}

type regexHandler func(lex *lexer, regex *regexp.Regexp)

// Created a default handler which will simply create a token with the matched contents. This handler is used with most simple tokens.
func defaultHandler(kind TokenKind, value string) regexHandler {
	return func(lex *lexer, _ *regexp.Regexp) {
		lex.advanceN(len(value))
		lex.push(newUniqueToken(kind, value))
	}
}

func stringHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	stringLiteral := lex.remainder()[match[0]:match[1]]

	lex.push(newUniqueToken(STRING, stringLiteral))
	lex.advanceN(len(stringLiteral))
}

func integerHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	lex.push(newUniqueToken(INTEGER, match))
	lex.advanceN(len(match))
}

func floatHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	lex.push(newUniqueToken(FLOAT, match))
	lex.advanceN(len(match))
}

func symbolHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())

	if kind, found := (*lex.keywords)[match]; found {
		lex.push(newUniqueToken(kind, match))
	} else if kind, found := (*lex.types)[match]; found {
		lex.push(newUniqueToken(kind, match))
	} else {
		lex.push(newUniqueToken(IDENTIFIER, match))
	}

	lex.advanceN(len(match))
}

func newlineHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	lex.advanceN(match[1])
	lex.push(newUniqueToken(NEWLINE, "\n"))
}

func skipHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	lex.advanceN(match[1])
}

func commentHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	if match != nil {
		// Advance past the entire comment.
		lex.advanceN(match[1])
		lex.line++
	}
}

func baseLexer() *lexer {
	return &lexer{
		pos:      0,
		line:     1,
		Tokens:   make([]Token, 0),
		patterns: nil,
		keywords: nil,
		types:    nil,
		filetype: Unrecognized,
	}
}

func vsLexer() *lexer {
	lex := baseLexer()
	lex.keywords = &reserved_lu_vs
	lex.types = &reserved_types_vs
	lex.filetype = Vs
	lex.patterns = &[]regexPattern{
		// {regexp.MustCompile(`\n`), defaultHandler(NEWLINE, "\n")},
		// get the newline character but ignore multiple newlines
		// + is a greedy operator, so it will match as many newlines as possible
		// ? is a non-greedy operator, so it will match as few newlines as possible
		// 'at least one newline' is the same as 'one or more newlines' which is the same as '+'
		{regexp.MustCompile(`\n+`), newlineHandler},
		{regexp.MustCompile(`\s+`), skipHandler},
		{regexp.MustCompile(`\/\/.*`), commentHandler},
		// {regexp.MustCompile(`"[^"]*"`), stringHandler},
		// TODO float and integer :: need to change '_' to be optional and only allowed between digits (one, not multiple)
		{regexp.MustCompile(`[0-9_]+(\.[0-9_]+)`), floatHandler},
		{regexp.MustCompile(`[0-9_]+`), integerHandler},
		{regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`), symbolHandler},
		{regexp.MustCompile(`\(`), defaultHandler(OPEN_PAREN, "(")},
		{regexp.MustCompile(`\)`), defaultHandler(CLOSE_PAREN, ")")},
		{regexp.MustCompile(`\{`), defaultHandler(OPEN_CURLY, "{")},
		{regexp.MustCompile(`\}`), defaultHandler(CLOSE_CURLY, "}")},
		{regexp.MustCompile(`\::`), defaultHandler(DOUBLE_COLON, "::")},
		{regexp.MustCompile(`\:`), defaultHandler(COLON, ":")},
		{regexp.MustCompile(`\+`), defaultHandler(PLUS, "+")},
		{regexp.MustCompile(`\-`), defaultHandler(DASH, "-")},
		{regexp.MustCompile(`\/`), defaultHandler(SLASH, "/")},
		{regexp.MustCompile(`\%`), defaultHandler(REMAINDER, "%")},
		{regexp.MustCompile(`\*\*`), defaultHandler(EXPONENTIATION, "**")},
		{regexp.MustCompile(`\*`), defaultHandler(ASTERISK, "*")},
		{regexp.MustCompile(`\==`), defaultHandler(EQUALS, "==")},
		{regexp.MustCompile(`\=`), defaultHandler(ASSIGNMENT, "=")},
		{regexp.MustCompile(`\,`), defaultHandler(COMMA, ",")},
		{regexp.MustCompile(`\;`), defaultHandler(SEMICOLON, ";")},
	}
	return lex
}

func watLexer() *lexer {
	lex := baseLexer()
	lex.keywords = &reserved_lu_wat
	lex.types = &reserved_types_wat
	lex.filetype = Wat
	lex.patterns = &[]regexPattern{
		{regexp.MustCompile(`\s+`), skipHandler},
		{regexp.MustCompile(`\;;.*`), commentHandler},
		{regexp.MustCompile(`"[^"]*"`), stringHandler},
		{regexp.MustCompile(`[0-9_]+(\.[0-9_]+)`), floatHandler},
		{regexp.MustCompile(`[0-9_]+`), integerHandler},
		// rewrite the next regex (identifiers starting with $) but keywords not
		{regexp.MustCompile(`\$[a-zA-Z_][a-zA-Z0-9_]*`), symbolHandler},                       // identifiers starting with $
		{regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*`), symbolHandler}, // identifiers starting with a letter but with '.' in the middle like i32.const
		{regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`), symbolHandler},                         // identifiers starting with a letter
		{regexp.MustCompile(`\(`), defaultHandler(OPEN_PAREN, "(")},
		{regexp.MustCompile(`\)`), defaultHandler(CLOSE_PAREN, ")")},
	}
	return lex
}
