package sema

import (
	"github.com/LaH-DeV/veles/ast"
)

// D89: `public use` re-exports. `public use geometry` puts the module
// `geometry` among the members of the module that writes it (`X.geometry`,
// and — when X is the package's root module — `use pkg.geometry` for another
// package); `public use shapes { area, Circle as Round }` puts those items
// there, flattened, so a facade can hide the layout behind it. Only the
// package's own modules can be re-exported.

// reexport adds what a `public use` names to the module scope of m.
func (c *Checker) reexport(m *Module, u *ast.UseSpec, dep *Module) {
	path := pathString(u.Path)
	if dep.Std || dep.Pkg != m.Pkg {
		c.errorf(u.Pos, "'public use %s' re-exports a module of another package; only this package's own modules can be re-exported (D89) — import it where it is used", path)
		return
	}
	if len(u.Names) == 0 {
		name := u.Path[len(u.Path)-1].Name
		if u.Alias != nil {
			name = u.Alias.Name
		}
		if old := m.Scope.Insert(&Symbol{Name: name, Kind: SymModule, Mod: dep, Pub: true, Module: m, Span: u.Pos}); old != nil {
			c.errorf(u.Pos, "'%s' is already declared in this module (at %s): re-export the module under another name, 'public use %s as …' (D89)", name, old.Span, path)
		}
		return
	}
	for _, n := range u.Names {
		member := dep.Scope.LookupLocal(n.Name.Name)
		if member == nil || !member.Pub {
			continue // declareUseNames has said so
		}
		bound := member
		name := n.Name.Name
		if n.Alias != nil {
			name = n.Alias.Name
			cp := *member
			cp.Name = name
			bound = &cp
		}
		if old := m.Scope.Insert(bound); old != nil {
			c.errorf(n.Pos, "'%s' is already declared in this module (at %s): re-export it under another name, 'public use %s { %s as … }' (D89)", name, old.Span, path, n.Name.Name)
		}
	}
}
