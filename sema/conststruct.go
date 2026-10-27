package sema

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D212: `const struct` — a type immutable all the way down. No field is
// `var`, and every field's type is deeply immutable (immutableProblem), so
// a value can be shared between tasks, be a map key or a module `val`, and
// nothing reached through it changes. A generic one is checked again for
// each set of type arguments, where they are written.

// checkConstStructs checks every `const struct` declaration: a type
// parameter passes here and is checked per instance (checkConstArgs).
func (c *Checker) checkConstStructs() {
	for _, s := range c.structs {
		if !s.Const {
			continue
		}
		d, _ := s.Decl.(*ast.StructDecl)
		if d == nil {
			continue
		}
		for i, f := range s.Fields {
			at := d.Name.Pos
			if i < len(d.Fields) && d.Fields[i].Name.Name == f.Name {
				at = d.Fields[i].Name.Pos
			}
			if f.Var {
				c.errorf(at, "'%s' is a 'var' field; a const struct holds only immutable data: drop 'var', and build a changed copy with 'copy' (D212)", f.Name)
				continue
			}
			if culprit, advice := immutableProblem(f.Type); culprit != nil {
				c.errorf(at, "%s; a const struct holds only immutable data: %s (D212)", isA(f.Name, f.Type, culprit), advice)
			}
		}
	}
}

// checkConstArgs checks an instance of a generic `const struct` where its
// type arguments are written: `Pair<i64, MutableList<i64>>` is refused
// there, naming the field and the argument.
func (c *Checker) checkConstArgs(tmpl *types.Struct, args []types.Type, span source.Span) {
	if !tmpl.Const || len(args) != len(tmpl.TypeParams) {
		return
	}
	if span.File == nil {
		span = c.typeSite // made by a generic call's inference
	}
	m := map[*types.TypeParam]types.Type{}
	for i, tp := range tmpl.TypeParams {
		if types.ContainsTypeParam(args[i]) || types.IsInvalid(args[i]) {
			return // inside generic code: checked where that code is instantiated
		}
		m[tp] = args[i]
	}
	for _, f := range tmpl.Fields {
		if !types.ContainsTypeParam(f.Type) {
			continue // checked at the declaration
		}
		t := c.hooks.Subst(f.Type, m)
		culprit, advice := immutableProblem(t)
		if culprit == nil {
			continue
		}
		key := "const:" + tmpl.Name + ":" + types.Key(t) + ":" + f.Name
		if c.seen[key] {
			continue
		}
		c.seen[key] = true
		c.errorf(span, "'%s' is a const struct, so its field %s; a const struct holds only immutable data: %s (D212)",
			tmpl.Name, isA(f.Name, t, culprit), advice)
	}
}

// isA says what a field holds: "'items' is a 'MutableList<i64>'", or for a
// problem further in "'rows' is a 'List<MutableList<i64>>', which holds a
// 'MutableList<i64>'".
func isA(name string, t, culprit types.Type) string {
	if types.Identical(t, culprit) {
		return fmt.Sprintf("'%s' is a '%s'", name, t)
	}
	return fmt.Sprintf("'%s' is a '%s', which holds a '%s'", name, t, culprit)
}

// immutableProblem finds the part of t that can change, and says what to
// use instead; nil when t is deeply immutable: numbers, bool, char, enums,
// string, immutable List/Map/Set of such, ranges, arrays, tuples,
// nullables, const structs, error types of such, and sealed types whose
// variants are all const structs. A type parameter passes (its instances
// are checked).
func immutableProblem(t types.Type) (types.Type, string) {
	return immutableIn(t, map[types.Type]bool{})
}

func immutableIn(t types.Type, seen map[types.Type]bool) (types.Type, string) {
	switch tt := t.(type) {
	case *types.Basic, *types.Enum, *types.Range, *types.TypeParam, *types.Assoc:
		return nil, ""
	case *types.Nullable:
		return immutableIn(tt.Elem, seen)
	case *types.Array:
		return immutableIn(tt.Elem, seen)
	case *types.Tuple:
		for _, e := range tt.Elems {
			if culprit, advice := immutableIn(e, seen); culprit != nil {
				return culprit, advice
			}
		}
		return nil, ""
	case *types.List:
		if tt.Mutable {
			return t, fmt.Sprintf("use '%s'", &types.List{Elem: tt.Elem})
		}
		return immutableIn(tt.Elem, seen)
	case *types.Set:
		if tt.Mutable {
			return t, fmt.Sprintf("use '%s'", &types.Set{Elem: tt.Elem})
		}
		return immutableIn(tt.Elem, seen)
	case *types.Map:
		if tt.Mutable {
			return t, fmt.Sprintf("use '%s'", &types.Map{Key: tt.Key, Value: tt.Value})
		}
		if culprit, advice := immutableIn(tt.Key, seen); culprit != nil {
			return culprit, advice
		}
		return immutableIn(tt.Value, seen)
	case *types.Struct:
		if tt.Const || seen[tt] {
			return nil, ""
		}
		seen[tt] = true
		if ok, _ := syncWrapper(tt); ok {
			return t, "a synchronized type is shared state, not data; keep it outside, beside the value"
		}
		if d, _ := templateOf(tt).Decl.(*ast.StructDecl); d != nil && d.Error {
			// an error cannot be declared const: it counts when its fields do
			for _, f := range tt.Fields {
				if f.Var {
					return t, fmt.Sprintf("'%s' has a 'var' field '%s'", tt, f.Name)
				}
				if culprit, advice := immutableIn(f.Type, seen); culprit != nil {
					return culprit, advice
				}
			}
			return nil, ""
		}
		if tt.Module == "std.prelude" && tt.Name == "StringBuilder" {
			return t, "use 'string'"
		}
		if tt.Module == "std" || strings.HasPrefix(tt.Module, "std.") {
			return t, fmt.Sprintf("'%s' can change in place", tt)
		}
		return t, fmt.Sprintf("declare '%s' as 'const struct', so the promise is written where it is kept", templateOf(tt).Name)
	case *types.Sealed:
		if seen[tt] {
			return nil, ""
		}
		seen[tt] = true
		for _, v := range tt.Variants {
			if !v.Const {
				return v, fmt.Sprintf("a sealed type counts when every variant does: declare '%s' as 'const struct'", templateOf(v).Name)
			}
		}
		return nil, ""
	case *types.ErrorUnion:
		for _, m := range tt.Members {
			if culprit, advice := immutableIn(m, seen); culprit != nil {
				return culprit, advice
			}
		}
		return nil, ""
	case *types.Pointer:
		return t, "a pointer reaches a value that can change; hold the value itself"
	case *types.Func:
		return t, "a function value can hold state it changes; hold data, or pass the function where it is used"
	case *types.Trait:
		return t, "an object of a trait can hold state; hold a sealed type of const structs, or a type parameter"
	case *types.Channel:
		return t, "a channel is shared state, not data; keep it outside, beside the value"
	case *types.Task:
		return t, "a task handle is shared state, not data; keep it outside, beside the value"
	}
	return t, fmt.Sprintf("'%s' is not immutable data", t)
}
