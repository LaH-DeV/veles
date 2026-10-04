package sema

import (
	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// deriveSchema writes `static fun schema(format, keys): Schema` (D125) next
// to a derived `decode`: the fields with the keys `decode` would match for
// that format and key style, which are required, which can be null, and
// each default as text; an enum's member names; a sealed family as a type no
// flat source can give. It follows `decode`: when the author wrote `decode`
// by hand the fields say nothing about what it reads, so the trait's default
// (one opaque value) stays.
func (c *Checker) deriveSchema(b *synth, impl *Impl) (*ast.FunDecl, string) {
	if !impl.Derived["decode"] {
		return nil, ""
	}
	schemaT := c.preludeType("Schema")
	params := []ast.Param{b.param("format", types.TString), b.param("keys", c.preludeType("KeyStyle"))}
	format := func() ast.Expr { return b.name("format") }
	keys := func() ast.Expr { return b.name("keys") }
	build := func(name string, args ...ast.Expr) ast.Expr {
		return b.call(b.member(b.typeExpr(schemaT), name), args...)
	}
	only := func(e ast.Expr) *ast.FunDecl {
		return b.fun("schema", true, params, schemaT, nil, []ast.Stmt{b.stmt(e)})
	}
	switch t := impl.Target.(type) {
	case *types.Enum:
		c.resolveEnum(t)
		var names []string
		for _, m := range t.Members {
			names = append(names, memberName(t, m))
		}
		text := b.member(b.typeExpr(c.preludeType("SchemaKind")), "Text")
		return only(build("leaf", text, b.str("one of "+joinQuoted(names)))), ""
	case *types.Sealed:
		return only(build("unsupported", b.str("a family of types ("+t.Name+")"))), ""
	case *types.Struct:
		fields, why := c.derivedFields(t, "Decodable")
		if why != "" {
			return nil, ""
		}
		field := c.preludeType("SchemaField")
		encodable, _ := c.preludeType("Encodable").(*types.Trait)
		f := &fnCtx{c: c}
		body := []ast.Stmt{b.val("$fields", &types.List{Elem: field, Mutable: true}, &ast.ListLit{Pos: b.sp})}
		for _, df := range fields {
			if df.skipAll || df.init {
				continue
			}
			// the default as text, for what has a text form (a Secret has none)
			var fallback ast.Expr = b.null()
			if df.hasDefault && encodable != nil && !types.ContainsTypeParam(df.typ) && f.implements(df.typ, encodable) {
				fallback = b.call(b.prelude("defaultText"), &ast.FieldDefaultExpr{Struct: t, Index: df.index, Pos: b.sp}, format(), keys())
			}
			required := !df.nullable && !df.hasDefault || df.nullable && df.required
			add := b.stmt(b.mcall(b.name("$fields"), "push", b.callArgs(b.typeExpr(field),
				b.namedArg("name", b.str(df.name)),
				b.namedArg("key", c.keyExprIn(b, df, format, keys)),
				b.namedArg("schema", b.call(b.member(b.typeExpr(df.typ), "schema"), format(), keys())),
				b.namedArg("required", b.boolLit(required)),
				b.namedArg("nullable", b.boolLit(df.nullable)),
				b.namedArg("fallback", fallback))))
			if len(df.skipIn) > 0 {
				body = append(body, b.ifStmt(b.not(c.skippedIn(b, df, format)), []ast.Stmt{add}, nil))
			} else {
				body = append(body, add)
			}
		}
		body = append(body, b.stmt(build("object", b.str(t.Name), b.mcall(b.name("$fields"), "toList"))))
		return b.fun("schema", true, params, schemaT, nil, body), ""
	}
	return nil, ""
}
