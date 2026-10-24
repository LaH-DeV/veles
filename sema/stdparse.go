package sema

import (
	"io/fs"
	"sync"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
)

// The embedded std is parsed once per process, not once per package load:
// the language server loads the whole prelude on every edit, and a test that
// checks many programs loads it for each. A file of the embedded std is the
// same bytes for the life of the process, so its path is its key; a std
// module read from disk (an edited `<repo>/std/<name>`, loadStdFromDisk) and
// every dependency go through loadLocal and are parsed afresh.
//
// The checker rewrites the tree it is given — D158 turns a bare member name
// into `this.name`, D176 wraps a decorated body — and what it writes depends
// on the whole program, so no two checks may share one: each load gets its
// own deep copy (ast.Clone) of the parsed file, about half the time and a
// third of the garbage of parsing it again. The cached original is never
// handed out.
var stdParsed sync.Map // "<module>/<file>.vs" -> *parsedStd

type parsedStd struct {
	once  sync.Once
	src   *source.File
	file  *ast.File
	diags []source.Diagnostic
}

// parseStd is a private copy of one embedded std file, parsed for m;
// whatever its parse reported is reported to this package again.
func (p *Package) parseStd(m *Module, path string) *ast.File {
	v, _ := stdParsed.LoadOrStore(path, &parsedStd{})
	e := v.(*parsedStd)
	e.once.Do(func() {
		data, _ := fs.ReadFile(std.FS, path)
		e.src = source.NewFile("std/"+path, string(data))
		e.src.Embedded = true
		diags := &source.Diagnostics{}
		e.file = parser.ParseFile(e.src, diags)
		e.diags = diags.Items
	})
	return p.timeParse(m, e.src, func() *ast.File {
		p.diags.Items = append(p.diags.Items, e.diags...)
		return ast.Clone(e.file)
	})
}
