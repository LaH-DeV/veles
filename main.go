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

	// if there is a second argument, it is the type of file (wat or vs)
	filetype, err := getFileType(os.Args)

	if err != nil {
		log.Fatal(err)
	}

	sourceBytes, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}

	tokens := []lexer.Token{}
	switch filetype {
	case "wat":
		tokens = lexer.TokenizeWat(string(sourceBytes))
	case "vs":
		tokens = lexer.TokenizeVs(string(sourceBytes))
	default:
		log.Fatal("Unrecognized file type")
	}

	for _, token := range tokens {
		fmt.Println(lexer.TokenKindString(token.Kind))
	}
}

func getFileType(args []string) (string, error) {
	filetype := "vs"
	if len(args[1:]) > 1 {
		if args[2] == "wat" || args[2] == "vs" {
			filetype = args[2]
		} else {
			return "", fmt.Errorf("Invalid file type")
		}
	}
	return filetype, nil
}
