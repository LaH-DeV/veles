package main

import (
	"fmt"
	"log"
	"os"

	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/parser"
)

func main() {
	config, err := setup(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Veles :: Parsing file: \"%s\".\n", config.filepath)

	lex := lexer.NewLexer(config.filetype)

	if lex == nil {
		log.Fatal("Veles :: lexer error: could not create lexer.")
	}

	tokens := lex.Tokenize(config.source)

	fmt.Printf("Veles :: %d tokens found.\n", len(tokens))
	for _, token := range tokens {
		fmt.Println(lexer.TokenKindString(token.Kind))
	}

	par := parser.NewParser(config.filetype)

	if par == nil {
		log.Fatal("Veles :: parser error: could not create parser.")
	}

	ast := par.ParseFile(tokens, config.filepath)

	fmt.Printf("Veles :: %d statements found.\n\n", len(ast.Statements))
	for _, stmt := range ast.Statements {
		fmt.Println(stmt.String())
	}
}
