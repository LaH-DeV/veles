// Command veles is the bootstrap compiler for the Veles language.
//
//	veles build <file.vs | dir>   compile a package to a native executable
//	veles run   <file.vs | dir>   compile and run
//	veles check <file.vs | dir>   type-check only (--fix applies lint corrections)
//	veles parse <file.vs>         dump the syntax tree
//	veles tokens <file.vs>        dump the token stream
//	veles fmt   <paths...>        format source files in place (--check, --stdout)
//	veles explain <file.vs | dir> --derive [Type]  print the implements the compiler wrote
//	veles lsp                     language server over stdio
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/driver"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/lsp"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: veles <build|run|test|check|parse|tokens> <path> [-o output] [--emit-llvm] [--keep] [--release] [--fix] [-- args...] | veles explain <path> --derive [Type] | veles fmt <paths...> [--check] [--stdout] | veles lsp")
	os.Exit(2)
}

func main() {
	os.Exit(driver.Guard(command))
}

// command runs one subcommand and returns its exit status. A panic in it is
// a compiler bug and is reported by driver.Guard.
func command() int {
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
			return 1
		}
	case "parse":
		file := mustLoad(path)
		diags := &source.Diagnostics{}
		f := parser.ParseFile(file, diags)
		fmt.Print(ast.Dump(f))
		fmt.Fprint(os.Stderr, diags.Render())
		if diags.HasErrors() {
			return 1
		}
	case "fmt":
		opts := driver.FormatOptions{}
		for _, a := range os.Args[2:] {
			switch a {
			case "--check":
				opts.Check = true
			case "--stdout":
				opts.Stdout = true
			default:
				if strings.HasPrefix(a, "-") {
					fmt.Fprintf(os.Stderr, "unknown flag %q\n", a)
					usage()
				}
				opts.Paths = append(opts.Paths, a)
			}
		}
		if len(opts.Paths) == 0 {
			usage()
		}
		return driver.Format(opts)
	case "explain":
		opts := driver.ExplainOptions{Path: path}
		for _, a := range os.Args[3:] {
			switch {
			case a == "--derive":
				opts.Derive = true
			case strings.HasPrefix(a, "-"):
				fmt.Fprintf(os.Stderr, "unknown flag %q\n", a)
				usage()
			default:
				opts.TypeName = a
			}
		}
		if !opts.Derive {
			fmt.Fprintln(os.Stderr, "veles explain: say what to explain (today: --derive)")
			usage()
		}
		return driver.Explain(opts)
	case "lsp":
		if err := lsp.Serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "veles lsp:", err)
			return 1
		}
		return 0
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
			case "--fix":
				opts.Fix = true
			case "--":
				opts.ProgramArgs = args[i+1:]
				i = len(args)
			default:
				fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
				usage()
			}
		}
		return driver.Run(opts)
	default:
		usage()
	}
	return 0
}

func mustLoad(path string) *source.File {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "veles:", err)
		os.Exit(1)
	}
	return source.NewFile(path, string(data))
}
