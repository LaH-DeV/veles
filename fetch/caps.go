package fetch

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// Capabilities (D138) are computed from a package's source, not declared by
// its author, so a package cannot understate them: an `unsafe` block, an
// `extern` block, a [native] table, and the standard modules it imports —
// `net`, `fs`, `os`, `ffi` and the few modules whose purpose they are. The list is
// sema.Capabilities. This is what `veles add` and `veles audit` show and what
// [policy] in a manifest can refuse.

// Caps maps a capability to where the package shows it: a few lines of
// evidence, `path:line what`.
type Caps map[string][]string

const maxEvidence = 5

func (c Caps) add(capability, evidence string) {
	list := c[capability]
	switch {
	case len(list) < maxEvidence:
		c[capability] = append(list, evidence)
	case list[len(list)-1] != "…":
		list[len(list)-1] = "…"
	}
}

// Names are the capabilities in the fixed order of sema.Capabilities.
func (c Caps) Names() []string {
	var out []string
	for _, n := range sema.Capabilities {
		if _, ok := c[n]; ok {
			out = append(out, n)
		}
	}
	return out
}

func (c Caps) merge(o Caps) {
	for n, ev := range o {
		if _, ok := c[n]; !ok {
			c[n] = nil
		}
		for _, e := range ev {
			c.add(n, e)
		}
	}
}

// stdModuleCaps says which standard modules give a package a capability by
// being imported: the ones whose purpose it is. A module that merely uses
// them inside (`http` reads the clock and the environment) does not pass them
// on, or "uses os" would be true of every program with a socket.
var stdModuleCaps = map[string][]string{
	"net":    {"net"},
	"tls":    {"net"},
	"http":   {"net"},
	"db":     {"net"},
	"otel":   {"net"},
	"fs":     {"fs"},
	"config": {"fs", "os"},
	"os":     {"os"},
	"ffi":    {"ffi"},
}

// walkAST calls fn for every AST node below v (reflection: the tree has no
// visitor, and an audit is not a hot path).
func walkAST(v reflect.Value, fn func(node any)) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			walkAST(v.Elem(), fn)
		}
	case reflect.Pointer:
		if v.IsNil() || v.Elem().Kind() != reflect.Struct || v.Elem().Type().PkgPath() != reflect.TypeOf(ast.File{}).PkgPath() {
			return
		}
		fn(v.Interface())
		walkAST(v.Elem(), fn)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				walkAST(v.Field(i), fn)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkAST(v.Index(i), fn)
		}
	}
}

// hasVelesSources reports whether a directory holds .vs files directly.
func hasVelesSources(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".vs") {
			return true
		}
	}
	return false
}

// PackageCaps computes the capabilities of the package in dir from its own
// source (not its dependencies'). Tests and hidden directories are not
// part of what a dependent builds.
func PackageCaps(dir string, man *sema.Manifest) (Caps, error) {
	caps := Caps{}
	if man != nil && (len(man.Native.Libs)+len(man.Native.StaticLibs)+len(man.Native.LibPaths)+len(man.Native.PkgConfig)) > 0 {
		caps.add("native", "veles.toml [native]")
	}
	local := map[string]bool{} // names that are the package's own, not std
	if man != nil {
		for _, d := range man.Requires {
			local[d.Name] = true
		}
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			// a directory is a module of the package only if it holds sources: the
			// loader skips one that does not and finds std's module of that name, so
			// an empty `net/` must not hide `use net`
			if e.IsDir() && hasVelesSources(filepath.Join(dir, e.Name())) {
				local[e.Name()] = true
			}
		}
	}
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// A hidden directory cannot be imported; every other directory can, even
			// one named vendor or holding a veles.toml of its own (the loader reads a
			// directory below the root as a module whatever is in it), so a package
			// cannot hide a capability there.
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".vs") && !strings.HasSuffix(d.Name(), ".test.vs") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		file := source.NewFile(rel, string(data))
		f := parser.ParseFile(file, &source.Diagnostics{})
		at := func(span source.Span) string {
			line, _ := file.Position(span.Start)
			return fmt.Sprintf("%s:%d", rel, line)
		}
		for _, d := range f.Decls {
			u, ok := d.(*ast.UseDecl)
			if !ok {
				continue
			}
			for _, s := range u.Specs {
				if len(s.Path) == 0 || local[s.Path[0].Name] {
					continue
				}
				module := s.Path[0].Name
				for _, c := range stdModuleCaps[module] {
					caps.add(c, at(s.Pos)+" use "+module)
				}
			}
		}
		walkAST(reflect.ValueOf(f), func(node any) {
			switch n := node.(type) {
			case *ast.UnsafeExpr:
				caps.add("unsafe", at(n.Pos)+" unsafe block")
			case *ast.FunDecl:
				if n.Unsafe {
					caps.add("unsafe", at(n.Pos)+" unsafe fun "+n.Name.Name)
				}
				if n.ExportC {
					caps.add("extern", at(n.Pos)+` extern "C" fun `+n.Name.Name)
				}
			case *ast.ExternBlock:
				caps.add("extern", at(n.Pos)+" extern block")
			}
		})
	}
	return caps, nil
}
