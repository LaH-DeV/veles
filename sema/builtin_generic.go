package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// builtinGeneric resolves a built-in generic named in a type position —
// `List<T>`, `Array<T, N>`, `Map<K, V>`, … — or is nil when name is not one.
func (c *Checker) builtinGeneric(env *typeEnv, t *ast.NamedType, name string) types.Type {
	switch name {
	case "List", "MutableList":
		if len(t.Args) != 1 {
			c.errorf(t.Pos, "%s takes exactly one type argument", name)
			return types.TInvalid
		}
		return &types.List{Elem: c.resolveType(env, t.Args[0]), Mutable: name == "MutableList"}
	case "Array":
		if len(t.Args) != 2 {
			c.errorf(t.Pos, "Array takes an element type and a length: 'Array<u8, 16>' (D121)")
			return types.TInvalid
		}
		return c.arrayType(c.resolveType(env, t.Args[0]), c.resolveTypeArg(env, t.Args[1], true), t.Pos)
	case "Range":
		if len(t.Args) != 1 {
			c.errorf(t.Pos, "Range takes exactly one type argument")
			return types.TInvalid
		}
		return &types.Range{Elem: c.resolveType(env, t.Args[0])}
	case "Option":
		if len(t.Args) != 1 {
			c.errorf(t.Pos, "Option takes exactly one type argument")
			return types.TInvalid
		}
		return &types.Nullable{Elem: c.resolveType(env, t.Args[0])}
	case "Map", "MutableMap":
		if len(t.Args) != 2 {
			c.errorf(t.Pos, "%s takes two type arguments", name)
			return types.TInvalid
		}
		k := c.resolveType(env, t.Args[0])
		c.checkHashable(k, t.Args[0].Span())
		return &types.Map{Key: k, Value: c.resolveType(env, t.Args[1]), Mutable: name == "MutableMap"}
	case "Channel":
		if len(t.Args) != 1 {
			c.errorf(t.Pos, "Channel takes one type argument")
			return types.TInvalid
		}
		return &types.Channel{Elem: c.resolveType(env, t.Args[0])}
	case "Task":
		if len(t.Args) != 1 {
			c.errorf(t.Pos, "Task takes one type argument")
			return types.TInvalid
		}
		return &types.Task{Result: c.resolveType(env, t.Args[0])}
	case "Set", "MutableSet":
		if len(t.Args) != 1 {
			c.errorf(t.Pos, "%s takes one type argument", name)
			return types.TInvalid
		}
		k := c.resolveType(env, t.Args[0])
		c.checkHashable(k, t.Args[0].Span())
		return &types.Set{Elem: k, Mutable: name == "MutableSet"}
	}
	return nil
}
