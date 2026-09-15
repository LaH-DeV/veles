package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// Attributes (D51): only what the compiler implements. Unknown attributes
// are errors, since nothing could ever read them.

var knownAttrs = map[string]bool{"test": true, "deprecated": true, "inline": true, "noinline": true, "mustUse": true, "specialize": true}

// attrsOf validates a declaration's attributes and returns them by name.
func (c *Checker) attrsOf(attrs []*ast.Attribute, what string) map[string]*ast.Attribute {
	out := map[string]*ast.Attribute{}
	for _, a := range attrs {
		if !knownAttrs[a.Name.Name] {
			c.errorf(a.Pos, "unknown attribute '@%s'; user-defined attributes are deferred until there is a derivation story (D51)", a.Name.Name)
			continue
		}
		if _, dup := out[a.Name.Name]; dup {
			c.errorf(a.Pos, "duplicate attribute '@%s'", a.Name.Name)
		}
		out[a.Name.Name] = a
		switch a.Name.Name {
		case "deprecated":
			if len(a.Args) != 1 {
				c.errorf(a.Pos, "@deprecated takes one argument: the reason")
			} else if _, ok := a.Args[0].Value.(*ast.StringLit); !ok {
				c.errorf(a.Args[0].Value.Span(), "@deprecated reason must be a string literal")
			}
		case "test", "inline", "noinline", "mustUse", "specialize":
			if len(a.Args) != 0 {
				c.errorf(a.Pos, "@%s takes no arguments", a.Name.Name)
			}
			if what != "function" {
				c.errorf(a.Pos, "@%s applies to functions", a.Name.Name)
			}
		}
	}
	return out
}

// deprecationOf returns the reason text of a @deprecated attribute.
func deprecationOf(attrs map[string]*ast.Attribute) (string, bool) {
	a, ok := attrs["deprecated"]
	if !ok {
		return "", false
	}
	if len(a.Args) == 1 {
		if s, ok := a.Args[0].Value.(*ast.StringLit); ok && len(s.Parts) == 1 {
			return s.Parts[0].Text, true
		}
	}
	return "", true
}

// noteUse reports deprecated uses at call sites.
func (f *fnCtx) noteUse(t *FuncTemplate, span source.Span) {
	if t.Attrs == nil {
		return
	}
	if reason, ok := deprecationOf(t.Attrs); ok {
		if reason != "" {
			f.warnf(span, "'%s' is deprecated: %s", t.Name, reason)
		} else {
			f.warnf(span, "'%s' is deprecated", t.Name)
		}
	}
}
