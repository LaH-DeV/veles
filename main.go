// Command veles is the bootstrap compiler for the Veles language.
//
//	veles build <file.vs | dir>   compile a package to a native executable
//	veles run   <file.vs | dir>   compile and run
//	veles check <file.vs | dir>   type-check only
//	veles parse <file.vs>         dump the syntax tree
//	veles tokens <file.vs>        dump the token stream
//	veles lsp                     language server over stdio
package main

import (
	"fmt"
	"os"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/driver"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/lsp"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: veles <build|run|test|check|parse|tokens> <path> [-o output] [--emit-llvm] [--keep] [--release] [-- args...] | veles lsp")
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 || (len(os.Args) < 3 && os.Args[1] != "lsp") {
		usage()
	}
	cmd := os.Args[1]
	path := ""
	if len(os.Args) > 2 {
		path = os.Args[2]
	}
	switch cmd {
	case "tokens":
		file := mustLoad(path)
		diags := &source.Diagnostics{}
		for _, t := range lexer.Tokenize(file, diags) {
			line, col := file.Position(t.Span.Start)
			fmt.Printf("%4d:%-3d %s\n", line, col, t)
		}
		fmt.Print(diags.Render())
		if diags.HasErrors() {
			os.Exit(1)
		}
	case "parse":
		file := mustLoad(path)
		diags := &source.Diagnostics{}
		f := parser.ParseFile(file, diags)
		fmt.Print(ast.Dump(f))
		fmt.Fprint(os.Stderr, diags.Render())
		if diags.HasErrors() {
			os.Exit(1)
		}
	case "lsp":
		if err := lsp.Serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "veles lsp:", err)
			os.Exit(1)
		}
		return
	case "build", "run", "check", "test":
		opts := driver.Options{Path: path, Mode: cmd}
		args := os.Args[3:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-o":
				if i+1 < len(args) {
					opts.Output = args[i+1]
					i++
				}
			case "--emit-llvm":
				opts.EmitLLVM = true
			case "--keep":
				opts.KeepIntermediates = true
			case "--release":
				opts.Release = true
			case "--":
				opts.ProgramArgs = args[i+1:]
				i = len(args)
			default:
				fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
				usage()
			}
		}
		os.Exit(driver.Run(opts))
	default:
		usage()
	}
}

func mustLoad(path string) *source.File {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		os.Exit(1)
	}
	return source.NewFile(path, string(data))
}
