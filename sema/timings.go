package sema

import (
	"time"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// Timings is where a timed build (`veles build --timings`) records the
// front end's time, per module. Nil when nobody asked: every hook below is
// one nil check on the untimed path.
type Timings struct {
	Modules map[string]*ModuleTiming // by module path ("" is the root)
}

// ModuleTiming is one module's share: its source, the time to parse it,
// and the time spent checking the bodies declared in it — every round of
// the effect fixpoint and every instance of a generic included, since that
// is where the time went.
type ModuleTiming struct {
	Files int
	Bytes int
	Parse time.Duration
	Check time.Duration
}

func (t *Timings) module(path string) *ModuleTiming {
	if t.Modules == nil {
		t.Modules = map[string]*ModuleTiming{}
	}
	mt := t.Modules[path]
	if mt == nil {
		mt = &ModuleTiming{}
		t.Modules[path] = mt
	}
	return mt
}

// LoadPackageTimed is LoadPackage recording each module's parse time in t;
// hand the package to Check and the bodies' check time lands there too.
func LoadPackageTimed(entry string, diags *source.Diagnostics, t *Timings) (*Package, error) {
	return loadPackage(entry, diags, nil, t)
}

// parse parses one source file of m, timed when the build asked.
func (p *Package) parse(m *Module, f *source.File) *ast.File {
	if p.timings == nil {
		return parser.ParseFile(f, p.diags)
	}
	start := time.Now()
	out := parser.ParseFile(f, p.diags)
	mt := p.timings.module(m.Path)
	mt.Parse += time.Since(start)
	mt.Files++
	mt.Bytes += len(f.Content)
	return out
}

// timeBody runs a body check, adding its time to the module it is declared
// in when the build asked.
func (c *Checker) timeBody(fn *Func, check func()) {
	if c.pkg == nil || c.pkg.timings == nil || fn.tmpl == nil || fn.tmpl.Module == nil {
		check()
		return
	}
	start := time.Now()
	check()
	c.pkg.timings.module(fn.tmpl.Module.Path).Check += time.Since(start)
}
