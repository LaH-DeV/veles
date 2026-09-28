// Command veles is the bootstrap compiler for the Veles language.
//
//	veles build <file.vs | dir>   compile a package to a native executable
//	veles run   <file.vs | dir>   compile and run
//	veles test  <file.vs | dir>   run the tests, test "..." { } (--filter text, --timeout 10m, --jobs n)
//	veles check <file.vs | dir>   type-check only (--fix applies lint corrections)
//	veles parse <file.vs>         dump the syntax tree
//	veles tokens <file.vs>        dump the token stream
//	veles fmt   <paths...>        format source files in place (--check, --stdout)
//	veles explain <family>         what an error means and how to fix it (the "see:" line)
//	veles explain <file.vs | dir> --derive [Type]  print the implements the compiler wrote
//	veles new   <dir>             create a package that runs and tests (--template server: an HTTP service)
//	veles doc   [dir] [-o out]    the package's public API as Markdown
//	veles lsp                     language server over stdio
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/driver"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/lsp"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: veles <build|run|test|check|parse|tokens> <path> [-o output] [--emit-llvm] [--keep] [--release] [--sanitize] [--timings] [--fix] [--filter text] [--timeout 10m] [--jobs n] [-- args...] | veles explain <family> | veles explain <path> --derive [Type] | veles fmt <paths...> [--check] [--stdout] | veles new <dir> [--template app|server] | veles doc [dir] [-o out] | veles lsp")
	os.Exit(2)
}

func main() {
	os.Exit(driver.Guard(command))
}

// command runs one subcommand and returns its exit status. A panic in it is
// a compiler bug and is reported by driver.Guard.
func command() int {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	switch cmd {
	case "lsp", "build", "run", "check", "test", "doc":
		// the package in the current directory when no path is given
	default:
		if len(os.Args) < 3 {
			usage()
		}
	}
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
			// `veles explain private-to-module`: the name a "see:" line gives
			if opts.TypeName != "" {
				usage()
			}
			return driver.ExplainFamily(path)
		}
		return driver.Explain(opts)
	case "doc":
		path, out := "", ""
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			switch {
			case args[i] == "-o" && i+1 < len(args):
				out = args[i+1]
				i++
			case !strings.HasPrefix(args[i], "-") && path == "":
				path = args[i]
			default:
				fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
				usage()
			}
		}
		if path == "" {
			path = "."
		}
		return driver.Doc(path, out)
	case "new":
		// veles new <dir> [--template app|server]
		args, template := os.Args[2:], ""
		if len(args) == 3 && args[1] == "--template" {
			args, template = args[:1], args[2]
		}
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "usage: veles new <dir> [--template %s]\n", strings.Join(driver.Templates, "|"))
			return 2
		}
		return driver.New(args[0], template)
	case "lsp":
		if err := lsp.Serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "veles lsp:", err)
			return 1
		}
		return 0
	case "build", "run", "check", "test":
		// flags may come before or after the path; the one bare word is the
		// path, "." when there is none
		opts := driver.Options{Mode: cmd}
		if cmd == "test" {
			opts.TestTimeout = driver.DefaultTestTimeout
		}
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--filter", "--timeout", "--jobs":
				if cmd != "test" {
					fmt.Fprintf(os.Stderr, "%s is a flag of 'veles test'\n", args[i])
					usage()
				}
				if i+1 >= len(args) {
					fmt.Fprintf(os.Stderr, "%s needs a value\n", args[i])
					usage()
				}
				if args[i] == "--filter" {
					opts.Filter = args[i+1]
				} else if args[i] == "--jobs" {
					n, err := strconv.Atoi(args[i+1])
					if err != nil || n < 1 {
						fmt.Fprintf(os.Stderr, "--jobs takes how many tests may run at once, 1 or more, not %q\n", args[i+1])
						usage()
					}
					opts.Jobs = n
				} else {
					d, err := time.ParseDuration(args[i+1])
					if err != nil || d < 0 {
						fmt.Fprintf(os.Stderr, "--timeout takes a duration like 30s, 2m or 0 (no limit), not %q\n", args[i+1])
						usage()
					}
					opts.TestTimeout = d
				}
				i++
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
			case "--sanitize":
				opts.Sanitize = true
			case "--timings":
				opts.Timings = true
			case "--fix":
				opts.Fix = true
			case "--":
				opts.ProgramArgs = args[i+1:]
				i = len(args)
			default:
				if strings.HasPrefix(args[i], "-") || opts.Path != "" {
					fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
					usage()
				}
				opts.Path = args[i]
			}
		}
		if opts.Path == "" {
			opts.Path = "."
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
