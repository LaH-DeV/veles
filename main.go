package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/LaH-DeV/veles/lexer"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		log.Fatalf("Veles :: No files listed.")
	}

	filepath := os.Args[1]
	filetype := path.Ext(filepath)

	if filetype != ".wat" && filetype != ".vs" {
		log.Fatalf("Veles :: Unrecognized file type: \"%s\".", filetype)
	} else if filetype[0] == '.' {
		filetype = filetype[1:]
	}

	fmt.Printf("Veles :: Parsing file: \"%s\".\n", filepath)

	sourceBytes, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Veles :: %s.", err)
	}

	tokens := lexer.Tokenize(string(sourceBytes), lexer.Filetype(filetype))

	// time to parse the tokens
	fmt.Printf("Veles :: %d tokens found.\n", len(tokens))
	//for _, token := range tokens {
	//fmt.Println(lexer.TokenKindString(token.Kind))
	//}
}
