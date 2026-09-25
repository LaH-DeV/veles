package driver

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

func TestReportICE(t *testing.T) {
	file := source.NewFile("app/main.vs", "fun main() {\n  boom()\n}\n")
	where := &source.Where{Phase: "generating code for", Name: "main", Span: source.Span{File: file, Start: 15, End: 19}}
	var out strings.Builder
	ReportICE(&out, "unsupported cast", where, false)
	want := "internal compiler error: unsupported cast\n" +
		"  while generating code for 'main' at app/main.vs:2:3\n" +
		"This is a bug in the Veles compiler, not in your program. Please report it\n" +
		"with the source it names.\n" +
		"Set VELES_DEBUG=1 to print the compiler's stack trace.\n"
	if out.String() != want {
		t.Errorf("got\n%s\nwant\n%s", out.String(), want)
	}

	out.Reset()
	ReportICE(&out, "x", nil, true)
	if !strings.Contains(out.String(), "goroutine") || strings.Contains(out.String(), "VELES_DEBUG") {
		t.Errorf("with the stack requested, the report should carry it instead of the hint:\n%s", out.String())
	}
}

func TestGuard(t *testing.T) {
	t.Setenv("VELES_DEBUG", "")
	if code := Guard(func() int { return 7 }); code != 7 {
		t.Errorf("a normal return passes through: got %d", code)
	}
	if code := Guard(func() int { panic("boom") }); code != ExitICE {
		t.Errorf("a panic is exit %d, got %d", ExitICE, code)
	}

	t.Setenv("VELES_DEBUG", "1")
	defer func() {
		if r := recover(); r != "boom" {
			t.Errorf("with VELES_DEBUG the panic continues, got %v", r)
		}
	}()
	Guard(func() int { panic("boom") })
	t.Error("unreachable: Guard should have re-panicked")
}
