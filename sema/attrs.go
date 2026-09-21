package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// Attributes (D51): only what the compiler implements. Unknown attributes
// are errors, since nothing could ever read them. The wire attributes —
// `@key`, `@skip`, `@required`, `@tag` — carry data that the derived
// `Codable` impls read (D58); they are still compiler-known.

var knownAttrs = map[string]bool{"test": true, "deprecated": true, "inline": true, "noinline": true, "mustUse": true, "specialize": true,
	"key": true, "skip": true, "required": true, "tag": true}

// attrsOf validates a declaration's attributes and returns them by name.
// `what` names the declaration kind: "function", "struct", "variant" (a
// struct of a sealed trait), "field", "member" (of an enum), "trait",
// "sealed", "enum", "value", "error", "type".
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
		case "key":
			c.checkKeyAttr(a, what)
		case "skip":
			if what != "field" {
				c.errorf(a.Pos, "@skip applies to a struct field: the field is left out of the wire format (D58)")
			}
			for _, arg := range a.Args {
				if arg.Name != nil || !isBareName(arg.Value) {
					c.errorf(arg.Value.Span(), "@skip takes format names, bare: @skip(json) skips the field for JSON only (D58)")
				}
			}
		case "required":
			if what != "field" {
				c.errorf(a.Pos, "@required applies to a struct field: the key must be present even though the field is nullable (D58)")
			}
			if len(a.Args) != 0 {
				c.errorf(a.Pos, "@required takes no arguments")
			}
		case "tag":
			if what != "sealed" {
				c.errorf(a.Pos, "@tag applies to a sealed trait: the key that names the variant on the wire (D58)")
			}
			c.checkTagAttr(a)
		}
	}
	return out
}

// checkKeyAttr validates `@key("name")` (every format) or
// `@key(json: "name", db: "other")` (by format) on a field, an enum member
// or a sealed variant.
func (c *Checker) checkKeyAttr(a *ast.Attribute, what string) {
	if what != "field" && what != "member" && what != "variant" {
		c.errorf(a.Pos, "@key applies to a struct field, an enum member or a sealed variant: its name on the wire (D58)")
		return
	}
	if len(a.Args) == 0 {
		c.errorf(a.Pos, "@key needs the name: @key(\"user_id\"), or by format @key(json: \"userId\", db: \"user_id\") (D58)")
		return
	}
	if what != "field" && (len(a.Args) != 1 || a.Args[0].Name != nil) {
		c.errorf(a.Pos, "@key on a %s takes one name for every format: @key(\"active\") (D58)", what)
		return
	}
	named := a.Args[0].Name != nil
	seen := map[string]bool{}
	for i, arg := range a.Args {
		if (arg.Name != nil) != named {
			c.errorf(arg.Value.Span(), "@key is either one name for every format or names by format, not both (D58)")
			return
		}
		if !named && i > 0 {
			c.errorf(arg.Value.Span(), "@key takes one name for every format; to differ by format write @key(json: \"a\", db: \"b\") (D58)")
			return
		}
		if named {
			if seen[arg.Name.Name] {
				c.errorf(arg.Name.Pos, "@key names the '%s' format twice", arg.Name.Name)
			}
			seen[arg.Name.Name] = true
		}
		if _, ok := arg.Value.(*ast.StringLit); !ok {
			c.errorf(arg.Value.Span(), "a @key name is a string literal")
		}
	}
}

// checkTagAttr validates `@tag("kind")` or `@tag("kind", content: "value")`.
func (c *Checker) checkTagAttr(a *ast.Attribute) {
	if len(a.Args) == 0 || a.Args[0].Name != nil {
		c.errorf(a.Pos, "@tag needs the key that names the variant: @tag(\"kind\"), or @tag(\"type\", content: \"value\") to put the fields under a key of their own (D58)")
		return
	}
	for i, arg := range a.Args {
		if i > 0 && (arg.Name == nil || arg.Name.Name != "content") {
			c.errorf(arg.Value.Span(), "@tag takes the key and optionally 'content:'")
			continue
		}
		if _, ok := arg.Value.(*ast.StringLit); !ok {
			c.errorf(arg.Value.Span(), "a @tag key is a string literal")
		}
	}
}

func isBareName(e ast.Expr) bool {
	n, ok := e.(*ast.NameExpr)
	return ok && len(n.TypeArgs) == 0
}

// stringArg returns the text of a plain string-literal argument.
func stringArg(arg ast.Arg) (string, bool) {
	if s, ok := arg.Value.(*ast.StringLit); ok && len(s.Parts) == 1 {
		return s.Parts[0].Text, true
	}
	return "", false
}

// wireKeys reads a `@key` attribute: the name for every format under "",
// and the per-format names by format. Nil when the attribute is absent.
func wireKeys(attrs []*ast.Attribute) map[string]string {
	for _, a := range attrs {
		if a.Name.Name != "key" {
			continue
		}
		out := map[string]string{}
		for _, arg := range a.Args {
			s, ok := stringArg(arg)
			if !ok {
				continue
			}
			if arg.Name == nil {
				out[""] = s
			} else {
				out[arg.Name.Name] = s
			}
		}
		return out
	}
	return nil
}

// skippedFormats reads a `@skip` attribute: nil when absent, an empty
// list for every format, or the formats named.
func skippedFormats(attrs []*ast.Attribute) (formats []string, present bool) {
	for _, a := range attrs {
		if a.Name.Name != "skip" {
			continue
		}
		for _, arg := range a.Args {
			if n, ok := arg.Value.(*ast.NameExpr); ok {
				formats = append(formats, n.Name)
			}
		}
		return formats, true
	}
	return nil, false
}

func hasAttr(attrs []*ast.Attribute, name string) bool {
	for _, a := range attrs {
		if a.Name.Name == name {
			return true
		}
	}
	return false
}

// tagOf reads a sealed trait's `@tag`: the discriminator key and the
// content key ("" for the internally tagged layout).
func tagOf(attrs []*ast.Attribute) (key, content string) {
	key = "type"
	for _, a := range attrs {
		if a.Name.Name != "tag" {
			continue
		}
		for _, arg := range a.Args {
			s, ok := stringArg(arg)
			if !ok {
				continue
			}
			if arg.Name == nil {
				key = s
			} else if arg.Name.Name == "content" {
				content = s
			}
		}
	}
	return key, content
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

// checkFieldAttrs validates a struct field's wire attributes (D58): a
// skipped field needs a default (the decoder could not construct the value
// otherwise), and `@required` only means something on a nullable field.
func (c *Checker) checkFieldAttrs(f *ast.Field) {
	attrs := c.attrsOf(f.Attrs, "field")
	if a, ok := attrs["skip"]; ok && f.Default == nil {
		c.errorf(a.Pos, "a @skip field needs a default: decoding leaves '%s' out, so it must have a value of its own (D58)", f.Name.Name)
	}
	if a, ok := attrs["required"]; ok {
		if _, nullable := f.Type.(*ast.NullableType); !nullable {
			c.errorf(a.Pos, "@required is for a nullable field whose key must still be present; '%s' is not nullable, so it is required already (D58)", f.Name.Name)
		}
	}
	if _, skipped := attrs["skip"]; skipped {
		if _, keyed := attrs["key"]; keyed {
			c.errorf(attrs["key"].Pos, "'%s' is skipped: its @key has no effect", f.Name.Name)
		}
	}
}
