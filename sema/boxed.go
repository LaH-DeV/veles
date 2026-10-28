package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// D198: a sealed type that contains itself — a tree — stores its variants
// behind a pointer, so a value is its tag and one word and every node is
// allocated at its own size; a sealed type that is not a tree stays inline
// (D12) unless a variant is marked `indirect`. markRecursive finds the
// trees; lintLargeVariants points at a non-tree type sized for one variant
// much larger than the rest.

// markRecursive sets Recursive on every sealed type one of whose variants
// reaches the type again: through a pointer, a collection, a nullable, a
// tuple or array, another struct's fields or another sealed type's variants.
func (c *Checker) markRecursive() {
	for _, s := range c.sealeds {
		seen := map[any]bool{}
		for _, v := range s.Variants {
			for _, f := range v.Fields {
				if reachesSealed(f.Type, s, seen) {
					s.Recursive = true
				}
			}
		}
	}
}

func reachesSealed(t types.Type, target *types.Sealed, seen map[any]bool) bool {
	switch t := t.(type) {
	case *types.Pointer:
		return reachesSealed(t.Elem, target, seen)
	case *types.Nullable:
		return reachesSealed(t.Elem, target, seen)
	case *types.List:
		return reachesSealed(t.Elem, target, seen)
	case *types.Set:
		return reachesSealed(t.Elem, target, seen)
	case *types.Array:
		return reachesSealed(t.Elem, target, seen)
	case *types.Channel:
		return reachesSealed(t.Elem, target, seen)
	case *types.Task:
		return reachesSealed(t.Result, target, seen)
	case *types.Map:
		return reachesSealed(t.Key, target, seen) || reachesSealed(t.Value, target, seen)
	case *types.Tuple:
		for _, e := range t.Elems {
			if reachesSealed(e, target, seen) {
				return true
			}
		}
	case *types.Sealed:
		tmpl := sealedTemplate(t)
		if tmpl == target {
			return true
		}
		if seen[tmpl] {
			return false
		}
		seen[tmpl] = true
		for _, v := range tmpl.Variants {
			if reachesSealed(v, target, seen) {
				return true
			}
		}
	case *types.Struct:
		tmpl := templateOf(t)
		if seen[tmpl] {
			return false
		}
		seen[tmpl] = true
		for _, f := range tmpl.Fields {
			if reachesSealed(f.Type, target, seen) {
				return true
			}
		}
	}
	return false
}

// lintLargeVariants warns of a sealed type that is not a tree whose largest
// variant is more than four words and more than twice the next: every
// value is sized for it, so most values carry room they never use.
func (c *Checker) lintLargeVariants() {
	lay := &types.Layout{}
	for _, s := range c.sealeds {
		if s.Recursive || len(s.TypeParams) > 0 || len(s.Variants) < 2 {
			continue // a tree is boxed already; a generic one's sizes wait for its arguments
		}
		var big *types.Struct
		bigWords, next := 0, 0
		for _, v := range s.Variants {
			if v.Indirect || v.Inline {
				continue // the author chose
			}
			size, _ := lay.Of(v)
			w := (size + 7) / 8
			switch {
			case w > bigWords:
				next, bigWords, big = bigWords, w, v
			case w > next:
				next = w
			}
		}
		if big == nil || bigWords <= 4 || bigWords <= 2*next {
			continue
		}
		d, _ := big.Decl.(*ast.StructDecl)
		if d == nil {
			continue
		}
		var fix *source.Fix
		if d.Nested {
			at := source.Span{File: d.Pos.File, Start: d.Pos.Start, End: d.Pos.Start}
			fix = fixReplace("Mark '"+big.Name+"' indirect", at, "indirect ")
		}
		where := "mark it 'indirect'"
		if !d.Nested {
			where = "declare it in the body of '" + s.Name + "' as 'indirect struct'"
		}
		c.warnLint("large_variant", d.Name.Pos, fix, "variant '%s' is %s, the others at most %s: %s so values of '%s' are not sized for it (D198)",
			big.Name, wordCount(bigWords), wordCount(next), where, s.Name)
	}
}

func wordCount(n int) string {
	if n == 1 {
		return "1 word"
	}
	return fmt.Sprintf("%d words", n)
}
