package format

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// FuzzFormat: on any input the formatter finishes without panicking; when
// the input parses, the output parses to the same tree and formatting it
// again changes nothing.
func FuzzFormat(f *testing.F) {
	for _, root := range []string{"../examples", "../std"} {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".vs") {
				if data, err := os.ReadFile(path); err == nil {
					f.Add(string(data))
				}
			}
			return nil
		})
	}
	fence := regexp.MustCompile("(?s)```veles\n(.*?)```")
	filepath.WalkDir("../docs/documentation", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, _ := os.ReadFile(path)
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		for _, m := range fence.FindAllStringSubmatch(text, -1) {
			f.Add(m[1])
		}
		return nil
	})
	f.Fuzz(func(t *testing.T, src string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			fuzzOne(t, src)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("formatting did not finish within 10s")
		}
	})
}

func fuzzOne(t *testing.T, src string) {
	orig := source.NewFile("fuzz.vs", src)
	diags := &source.Diagnostics{}
	before := parser.ParseFile(orig, diags)
	out, _ := Source(orig, Options{})
	if diags.HasErrors() {
		return // unparsable input is returned unchanged; nothing more to check
	}
	diags = &source.Diagnostics{}
	after := parser.ParseFile(source.NewFile("fuzz.vs", out), diags)
	if diags.HasErrors() {
		t.Fatalf("formatted output does not parse:\n%s\n--- input ---\n%q\n--- output ---\n%s", diags.Render(), src, out)
	}
	if a, b := ast.Dump(before), ast.Dump(after); a != b {
		t.Fatalf("formatting changed the syntax tree\n--- input ---\n%q\n--- output ---\n%s\n--- before ---\n%s\n--- after ---\n%s", src, out, a, b)
	}
	if again, _ := Source(source.NewFile("fuzz.vs", out), Options{}); again != out {
		t.Fatalf("formatting is not idempotent\n--- input ---\n%q\n--- first ---\n%s\n--- second ---\n%s", src, out, again)
	}
}
