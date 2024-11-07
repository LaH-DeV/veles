package main

import (
	"fmt"
	"os"
	"path"

	"github.com/LaH-DeV/veles/lexer"
)

type config struct {
	source   string
	filetype lexer.Filetype
	filepath string
}

func setup(args []string) (*config, error) {
	if len(args) < 2 || args[1] == "" {
		return nil, fmt.Errorf("Veles :: No files listed.")
	}

	filepath := args[1]
	filetype := getFiletype(path.Ext(filepath))

	if filetype == lexer.Unrecognized {
		return nil, fmt.Errorf("Veles :: Unrecognized file type for file: \"%s\".", filepath)
	}

	sourceBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("Veles :: %s.", err)
	}

	return &config{
		source:   string(sourceBytes),
		filetype: filetype,
		filepath: filepath,
	}, nil
}

func getFiletype(ext string) lexer.Filetype {
	switch ext {
	case ".vs":
		return lexer.Vs
	case ".wat":
		return lexer.Wat
	}
	return lexer.Unrecognized
}
