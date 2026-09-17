package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LaH-DeV/veles/source"
)

// corpusFiles returns every .vs file of the repository plus the code
// blocks of the documentation, as fuzzing seeds.
func corpusFiles(t testing.TB) []string {
	t.Helper()
	var out []string
	for _, root := range []string{"../examples", "../std"} {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".vs") {
				data, err := os.ReadFile(path)
				if err == nil {
					out = append(out, string(data))
				}
			}
			return nil
		})
	}
	filepath.WalkDir("../docs/documentation", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(data)
		for {
			i := strings.Index(text, "```veles\n")
			if i < 0 {
				i = strings.Index(text, "```veles\r\n")
				if i < 0 {
					break
				}
			}
			text = text[i+len("```veles"):]
			j := strings.Index(text, "```")
			if j < 0 {
				break
			}
			out = append(out, strings.TrimLeft(text[:j], "\r\n"))
			text = text[j+3:]
		}
		return nil
	})
	return out
}

// withDeadline runs fn and fails the test if it does not return in time:
// a parser that loops forever on some input is a bug like any other.
func withDeadline(t testing.TB, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not finish within %v", what, d)
	}
}

// FuzzParse: the parser must finish, without panicking, on any input.
func FuzzParse(f *testing.F) {
	for _, src := range corpusFiles(f) {
		f.Add(src)
	}
	f.Add("fun f() { (b & 1)")
	f.Add("struct R {\n  fun g(): bool = (b & 1)\n")
	f.Fuzz(func(t *testing.T, src string) {
		withDeadline(t, 5*time.Second, "parse", func() {
			diags := &source.Diagnostics{}
			ParseFileLayout(source.NewFile("fuzz.vs", src), diags)
		})
	})
}
