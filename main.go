package main

import (
	"fmt"
	"log"
	"os"

	"github.com/LaH-DeV/veles/lexer"
)

func main() {
	config, err := setup(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Veles :: Parsing file: \"%s\".\n", config.filepath)

	lex := lexer.NewLexer(config.filetype)

	if lex == nil {
		log.Fatal("Veles :: lexer error: unrecognized filetype.")
	}

	tokens := lex.Tokenize(config.source)

	fmt.Printf("Veles :: %d tokens found.\n", len(tokens))
	// for _, token := range tokens {
	// 	fmt.Println(lexer.TokenKindString(token.Kind))
	// }

	// time to parse the tokens
}
