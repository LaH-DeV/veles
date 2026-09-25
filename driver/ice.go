package driver

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/LaH-DeV/veles/source"
)

// ExitICE is the exit status of an internal compiler error: 1 is a program
// the compiler refused, 2 is a usage error.
const ExitICE = 3

// Guard runs a command and turns a panic inside it — a compiler bug, never
// the user's — into a short report on stderr and ExitICE, instead of a Go
// stack trace. With VELES_DEBUG set the report is followed by the trace and
// the panic continues, so the full picture is one variable away.
func Guard(run func() int) (code int) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		debugging := os.Getenv("VELES_DEBUG") != ""
		ReportICE(os.Stderr, r, source.CurrentWhere(), debugging)
		if debugging {
			panic(r)
		}
		code = ExitICE
	}()
	return run()
}

// ReportICE writes the internal-compiler-error report for a recovered panic.
func ReportICE(w io.Writer, r any, where *source.Where, withStack bool) {
	fmt.Fprintf(w, "internal compiler error: %v\n", r)
	if where != nil {
		name := where.Name
		if name == "" {
			name = "a function"
		} else {
			name = "'" + name + "'"
		}
		if where.Span.IsValid() {
			fmt.Fprintf(w, "  while %s %s at %s\n", where.Phase, name, where.Span)
		} else {
			fmt.Fprintf(w, "  while %s %s\n", where.Phase, name)
		}
	}
	fmt.Fprintln(w, "This is a bug in the Veles compiler, not in your program. Please report it")
	fmt.Fprintln(w, "with the source it names.")
	if withStack {
		w.Write(debug.Stack())
	} else {
		fmt.Fprintln(w, "Set VELES_DEBUG=1 to print the compiler's stack trace.")
	}
}
