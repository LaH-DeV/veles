package sema

import (
	"regexp"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Lint: a Closeable that is never closed (D115, a warning, family
// `resources`). A local bound with `val`/`var` to a Closeable the code
// just produced — a call or a constructor, not a field read or an alias —
// warns when it is never `close()`d and never handed off: returned,
// stored (a field, a collection, an outer variable), passed as an
// argument, captured by a lambda or bound to another name. The fix turns
// the `val` into `with`. Handing off is anything but calling a method on
// it (one that does not keep `this`) or reading its fields, so the
// analysis is local and silent on `serve(listener)`; it misses a callee
// that drops what it was given. A `with` binding is a `With` node, not a
// `VarDecl`, and `val _ = …` is the explicit discard, so neither is seen.
func (c *Checker) lintNeverClosed(prog *Program) {
	closeable := c.traitNamed("Closeable")
	if closeable == nil {
		return
	}
	isCloseable := func(t types.Type) bool {
		return t != nil && !types.IsInvalid(t) && c.findImplFor(t, closeable) != nil
	}
	for _, fn := range prog.Funcs {
		if fn.Body == nil {
			continue
		}
		var candidates []*Var
		walkBlock(fn.Body, func(n any) {
			if d, ok := n.(*VarDecl); ok && d.Var.checkUse && d.Var.Span.IsValid() && !c.syntheticSpans[d.Var.Span] &&
				producesValue(d.Init) && isCloseable(d.Var.Type) {
				candidates = append(candidates, d.Var)
			}
		})
		for _, v := range candidates {
			if !closedOrHandedOff(fn, v) {
				var fix *source.Fix
				if at, ok := valKeywordBefore(v.Span); ok {
					fix = fixReplace("Close it at the end of the block with 'with'", at, "with")
				}
				c.warnFix(v.Span, fix, "'%s' is never closed: bind it with 'with %s = …' so it is closed at the end of the block, close it, or hand it on (D115)", v.Name, v.Name)
			}
		}
	}
}

// producesValue: the initializer makes a new value — a call or a
// constructor, possibly under `try` — rather than reading one that
// something else already owns.
func producesValue(e Expr) bool {
	for {
		switch x := e.(type) {
		case *Try:
			e = x.X
		case *ResultValue:
			e = x.X
		case *BlockExpr:
			if x.Block == nil || x.Block.Value == nil {
				return false
			}
			e = x.Block.Value
		case *Call, *CallIndirect, *CallVirtual, *StructLit:
			return true
		default:
			return false
		}
	}
}

// closedOrHandedOff reports whether fn closes v or lets it go somewhere
// else. A use of v is harmless only as the receiver of a method that does
// not keep `this`, or as the root of a field read.
func closedOrHandedOff(fn *Func, v *Var) bool {
	done := false
	harmless := map[*VarRef]bool{}
	receiver := func(e Expr) *VarRef {
		if a, ok := e.(*AddrOf); ok {
			e = a.X
		}
		r, _ := e.(*VarRef)
		if r != nil && r.Var == v {
			return r
		}
		return nil
	}
	walkBlock(fn.Body, func(n any) {
		switch n := n.(type) {
		case *Call:
			if n.Fn.Receiver != nil && len(n.Args) > 0 {
				if r := receiver(n.Args[0]); r != nil {
					if isCloseMethod(n.Fn.Display) || n.Fn.SelfEscapes {
						done = true
					}
					harmless[r] = true
				}
			}
		case *CallVirtual:
			if r := receiver(n.Obj); r != nil {
				if slots, _ := objectSlots(n.Trait); n.Index < len(slots) && slots[n.Index].Name == "close" {
					done = true
				}
				harmless[r] = true
			}
		case *FieldGet:
			if r := receiver(n.X); r != nil {
				harmless[r] = true
			}
		case *Closure:
			for _, cv := range n.Captures {
				if cv == v {
					done = true
				}
			}
		}
	})
	if done {
		return true
	}
	walkBlock(fn.Body, func(n any) {
		if r, ok := n.(*VarRef); ok && r.Var == v && !harmless[r] {
			done = true
		}
	})
	return done
}

func isCloseMethod(display string) bool {
	return display == "close" || strings.HasSuffix(display, ".close")
}

var valKeyword = regexp.MustCompile(`(?:^|[ \t;{])(val|var)[ \t]+$`)

// valKeywordBefore finds the `val`/`var` keyword just before a binding's
// name on its line, for the fix that turns it into `with`.
func valKeywordBefore(name source.Span) (source.Span, bool) {
	if name.File == nil {
		return source.Span{}, false
	}
	src := name.File.Content
	lineStart := name.Start
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	m := valKeyword.FindStringSubmatchIndex(src[lineStart:name.Start])
	if m == nil {
		return source.Span{}, false
	}
	return source.Span{File: name.File, Start: lineStart + m[2], End: lineStart + m[3]}, true
}

// discardedCloseable warns on an expression statement whose value is a
// Closeable the call just produced: nothing can close it any more
// (D115). `val _ = f()` is the explicit discard.
func (f *fnCtx) discardedCloseable(s *ast.ExprStmt, x Expr) {
	call := s.X
	if t, ok := call.(*ast.TryExpr); ok {
		call = t.X
	}
	ce, ok := call.(*ast.CallExpr)
	if !ok || ce.Async {
		return
	}
	closeable := f.c.traitNamed("Closeable")
	t := x.Type()
	if closeable == nil || types.IsUnit(t) || types.IsNever(t) || types.IsInvalid(t) || f.findImpl(t, closeable) == nil {
		return
	}
	at := source.Span{File: s.X.Span().File, Start: s.X.Span().Start, End: s.X.Span().Start}
	fix := fixReplace("Close it at the end of the block with 'with'", at, "with _ = ")
	f.c.warnFix(s.X.Span(), fix, "the '%s' this returns is never closed: bind it with 'with', or discard it on purpose with 'val _ = …' (D115)", t)
}
