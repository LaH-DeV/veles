package lexer

import (
	"fmt"
	"regexp"
)

func TokenizeWat(source string) []Token {
	patterns := []regexPattern{
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
	lex := createLexer(source, patterns, &reserved_lu_wat, &reserved_types_lu)
	return lex.tokenize()
}

func TokenizeVs(source string) []Token {
	patterns := []regexPattern{
		{regexp.MustCompile(`\s+`), skipHandler},
		{regexp.MustCompile(`\/\/.*`), commentHandler},
		// {regexp.MustCompile(`"[^"]*"`), stringHandler},
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
		{regexp.MustCompile(`\-`), defaultHandler(MINUS, "-")},
		{regexp.MustCompile(`\==`), defaultHandler(EQUALS, "==")},
		{regexp.MustCompile(`\=`), defaultHandler(ASSIGNMENT, "=")},
		{regexp.MustCompile(`\,`), defaultHandler(COMMA, ",")},
		{regexp.MustCompile(`\;`), defaultHandler(SEMICOLON, ";")},
	}
	lex := createLexer(source, patterns, &reserved_lu_vs, &reserved_types_lu)
	return lex.tokenize()
}

type regexPattern struct {
	regex   *regexp.Regexp
	handler regexHandler
}

type lexer struct {
	source   string
	pos      int
	line     int
	Tokens   []Token
	patterns []regexPattern
	keywords *map[string]TokenKind
	types    *map[string]TokenKind
}

func (lex *lexer) tokenize() []Token {
	for !lex.at_eof() {
		matched := false

		for _, pattern := range lex.patterns {
			loc := pattern.regex.FindStringIndex(lex.remainder())
			if loc != nil && loc[0] == 0 {
				pattern.handler(lex, pattern.regex)
				matched = true
				break // Exit the loop after the first match
			}
		}

		if !matched {
			panic(fmt.Sprintf("lexer error: unrecognized token near '%v'", lex.remainder()))
		}
	}
	lex.push(newUniqueToken(EOF, "EOF"))
	return lex.Tokens
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

func createLexer(source string, patterns []regexPattern, keywords *map[string]TokenKind, types *map[string]TokenKind) *lexer {
	return &lexer{
		pos:      0,
		line:     1,
		source:   source,
		Tokens:   make([]Token, 0),
		patterns: patterns,
		keywords: keywords,
		types:    types,
	}
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
	} else if kind, found := reserved_types_lu[match]; found {
		lex.push(newUniqueToken(kind, match))
	} else {
		lex.push(newUniqueToken(IDENTIFIER, match))
	}

	lex.advanceN(len(match))
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
