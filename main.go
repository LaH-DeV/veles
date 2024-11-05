package main

import (
	"fmt"
	"log"
	"os"

	"github.com/LaH-DeV/veles/lexer"
)

func main() {
	if len(os.Args[1:]) <= 0 {
		log.Fatal("No arguments provided")
	}
	filepath := os.Args[1]

	sourceBytes, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}

	tokens := lexer.Tokenize(string(sourceBytes))

	for _, token := range tokens {
		fmt.Println(lexer.TokenKindString(token.Kind))
	}
}
