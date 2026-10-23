package main

// bench/parse (plan S7): the self-hosted front end — selfhost/'s lexer and
// parser, written in Veles — against the Go front end it was ported from,
// over every `.vs` file of std. The Veles side is `selfhost --bench-parse
// std` rather than a bench/parse/main.vs, because the parser lives in the
// selfhost package; the checksum is the declarations found.

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// programs are benchmarks whose Veles side is a program elsewhere in the
// repository, run with arguments from the repository's root.
var programs = map[string]struct {
	dir  string
	args []string
}{
	"parse": {dir: "selfhost", args: []string{"--bench-parse", "std"}},
}

func init() { references["parse"] = refParse }

func refParse() string {
	start := time.Now()
	var files []*source.File
	must(filepath.WalkDir(filepath.Join(moduleRoot(), "std"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".vs") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, source.NewFile(path, string(data)))
		return nil
	}))
	setup(start)
	decls := 0
	for _, f := range files {
		var diags source.Diagnostics
		decls += len(parser.ParseFile(f, &diags).Decls)
	}
	return strconv.Itoa(decls)
}
