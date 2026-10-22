package ast_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
)

// sources parses every embedded std file and every Veles source in the
// repository, broken ones included.
func sources(t testing.TB) []*ast.File {
	var out []*ast.File
	add := func(path string, data []byte) {
		out = append(out, parser.ParseFile(source.NewFile(path, string(data)), &source.Diagnostics{}))
	}
	fs.WalkDir(std.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".vs") {
			data, _ := fs.ReadFile(std.FS, path)
			add(path, data)
		}
		return nil
	})
	filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != ".." {
			return filepath.SkipDir
		}
		if strings.HasSuffix(path, ".vs") || strings.HasSuffix(path, ".vss") {
			if data, err := os.ReadFile(path); err == nil {
				add(path, data)
			}
		}
		return nil
	})
	if len(out) < 100 {
		t.Fatalf("found only %d sources", len(out))
	}
	return out
}

// TestClone: a copy prints as the original; it shares no node or slice
// with it (a checker rewriting one can never be seen through the other);
// and it shares within itself exactly what the original does — a node the
// original reaches twice is one node of the copy, and two nodes stay two.
func TestClone(t *testing.T) {
	for _, f := range sources(t) {
		c := ast.Clone(f)
		if ast.Dump(c) != ast.Dump(f) {
			t.Fatalf("%s: the copy prints differently", f.Source.Path)
		}
		if c.Source != f.Source {
			t.Fatalf("%s: the source file is copied, not shared", f.Source.Path)
		}
		w := lockstep{t: t, path: f.Source.Path, toCopy: map[unsafe.Pointer]unsafe.Pointer{}, fromCopy: map[unsafe.Pointer]unsafe.Pointer{}}
		w.walk(reflect.ValueOf(f), reflect.ValueOf(c))
	}
}

var astPkg = reflect.TypeOf(ast.File{}).PkgPath()

// lockstep walks an original and its copy together.
type lockstep struct {
	t                *testing.T
	path             string
	toCopy, fromCopy map[unsafe.Pointer]unsafe.Pointer
}

func (w *lockstep) walk(a, b reflect.Value) {
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() || a.Type().Elem().PkgPath() != astPkg {
			return
		}
		pa, pb := a.UnsafePointer(), b.UnsafePointer()
		if pa == pb {
			w.t.Fatalf("%s: the copy shares a %s with the original", w.path, a.Type())
		}
		if seen, ok := w.toCopy[pa]; ok {
			if seen != pb {
				w.t.Fatalf("%s: a %s the original reaches twice is two nodes of the copy", w.path, a.Type())
			}
			return
		}
		if _, ok := w.fromCopy[pb]; ok {
			w.t.Fatalf("%s: two %s nodes of the original are one node of the copy", w.path, a.Type())
		}
		w.toCopy[pa], w.fromCopy[pb] = pb, pa
		w.walk(a.Elem(), b.Elem())
	case reflect.Interface:
		if !a.IsNil() {
			w.walk(a.Elem(), b.Elem())
		}
	case reflect.Slice:
		if a.Cap() > 0 && a.UnsafePointer() == b.UnsafePointer() {
			w.t.Fatalf("%s: the copy shares a %s with the original", w.path, a.Type())
		}
		for i := 0; i < a.Len(); i++ {
			w.walk(a.Index(i), b.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			w.walk(a.Field(i), b.Field(i))
		}
	}
}

func preludeSources() (paths, texts []string) {
	fs.WalkDir(std.FS, "prelude", func(path string, d fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".vs") {
			data, _ := fs.ReadFile(std.FS, path)
			paths, texts = append(paths, path), append(texts, string(data))
		}
		return nil
	})
	return
}

// The two benchmarks compare what a load of the prelude costs parsed and
// copied from the parsed cache (sema/stdparse.go).
func BenchmarkParsePrelude(b *testing.B) {
	paths, texts := preludeSources()
	for i := 0; i < b.N; i++ {
		for j := range paths {
			parser.ParseFile(source.NewFile(paths[j], texts[j]), &source.Diagnostics{})
		}
	}
}

func BenchmarkClonePrelude(b *testing.B) {
	paths, texts := preludeSources()
	var files []*ast.File
	for j := range paths {
		files = append(files, parser.ParseFile(source.NewFile(paths[j], texts[j]), &source.Diagnostics{}))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, f := range files {
			ast.Clone(f)
		}
	}
}
