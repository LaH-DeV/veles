package sema

import (
	"fmt"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Derivation (D58): an empty `impl Codable` (or `Encodable`, `Decodable`,
// `Comparable`) asks the compiler to write the body from the type's shape.
// The body is synthesized as ordinary syntax — the same nodes the parser
// makes, plus ResolvedType / TypeExpr / FieldDefaultExpr for what no
// source could spell — and checked like anything the author wrote, so the
// checker, the backend and the diagnostics see one kind of function.
//
// What is derived, and how:
//   - a struct encodes as an object of its non-skipped fields (a variant of
//     a sealed trait writes the discriminator first) and decodes by reading
//     keys until the object ends, recording a problem for what is missing;
//   - a sealed trait dispatches encoding to the variant and decodes by
//     buffering the object into a Value, reading the tag and decoding the
//     variant from the tree;
//   - an enum is its name (or its number when the format says so);
//   - Comparable orders lexicographically by field.
//
// A trait with supertraits (Codable) is derived by deriving the supers the
// type lacks (deriveSupers); a method the author wrote in its body goes to
// the super that declares it.

// derivable names the prelude traits the compiler can write, or "".
func (c *Checker) derivable(t *types.Trait) string {
	if t == nil || t.Module != "std.prelude" {
		return ""
	}
	switch t.Name {
	case "Encodable", "Decodable", "Comparable":
		return t.Name
	}
	return ""
}

// preludeType finds a prelude type by name (a trait, struct, enum).
func (c *Checker) preludeType(name string) types.Type {
	if sym := c.universe.LookupLocal(name); sym != nil && sym.Kind == SymType {
		if t := c.symType(sym); t != nil {
			return t
		}
		return sym.Type
	}
	return nil
}

// deriveMissing appends synthesized declarations for the methods of a
// derivable trait that the impl does not write. Silent when the trait is
// not derivable; reports why when the target cannot be derived.
func (c *Checker) deriveMissing(d *ast.ImplDecl, impl *Impl, trait *types.Trait) {
	kind := c.derivable(trait)
	if kind == "" {
		return
	}
	written := map[string]bool{}
	for _, md := range d.Methods {
		written[md.Name.Name] = true
	}
	for _, name := range trait.MethodList {
		if written[name] || c.traitDefault(trait, name) != nil {
			continue
		}
		fd, why := c.deriveMethod(d, impl, trait, kind, name)
		if fd == nil {
			if why != "" {
				c.errorf(d.Pos, "cannot derive '%s' for '%s': %s (D58)", trait.Name, impl.Target, why)
				c.deriveFailed[impl] = true
			}
			return
		}
		if c.debugDerive {
			fmt.Println(DumpDerived(fd))
		}
		d.Methods = append(d.Methods, fd)
		if impl.Derived == nil {
			impl.Derived = map[string]bool{}
		}
		impl.Derived[name] = true
	}
}

// deriveMethod synthesizes one trait method for the impl's target.
func (c *Checker) deriveMethod(d *ast.ImplDecl, impl *Impl, trait *types.Trait, kind, name string) (*ast.FunDecl, string) {
	b := &synth{sp: d.Pos}
	c.syntheticSpans[d.Pos] = true
	switch t := impl.Target.(type) {
	case *types.Struct:
		fields, why := c.derivedFields(t, kind)
		if why != "" {
			return nil, why
		}
		c.inferDeriveBounds(impl, trait, fields)
		switch kind {
		case "Encodable":
			return c.encodeStruct(b, t, fields), ""
		case "Decodable":
			return c.decodeStruct(b, t, fields), ""
		case "Comparable":
			return c.compareStruct(b, t, fields), ""
		}
	case *types.Sealed:
		if len(t.TypeParams) > 0 {
			return nil, "a generic sealed trait is not derived yet"
		}
		switch kind {
		case "Encodable":
			c.deriveVariants(d, impl, trait, t)
			return c.encodeSealed(b, t), ""
		case "Decodable":
			c.deriveVariants(d, impl, trait, t)
			return c.decodeSealed(b, t), ""
		case "Comparable":
			return nil, "a sealed trait has no field order to compare by; write compareTo"
		}
	case *types.Enum:
		switch kind {
		case "Encodable":
			return c.encodeEnum(b, t), ""
		case "Decodable":
			return c.decodeEnum(b, t), ""
		}
		return nil, "an enum compares by itself (D57)"
	}
	return nil, fmt.Sprintf("'%s' is not a struct, a sealed trait or an enum", impl.Target)
}

// ---------------------------------------------------------------------------
// fields

// derivedField is what the derive knows about one struct field.
type derivedField struct {
	index      int // into the struct's Fields
	name       string
	typ        types.Type
	keys       map[string]string // nil: the field's name, styled by the format
	skipAll    bool
	skipIn     []string // formats the field is left out of
	required   bool
	hasDefault bool
	init       bool // assigned by `init { }`: read, never decoded
	nullable   bool
}

// derivedFields reads the fields and their wire attributes; why is set
// when a field makes the derivation impossible.
func (c *Checker) derivedFields(st *types.Struct, kind string) ([]derivedField, string) {
	c.resolveStruct(templateOf(st))
	decl, _ := templateOf(st).Decl.(*ast.StructDecl)
	var out []derivedField
	for i, fld := range st.Fields {
		df := derivedField{index: i, name: fld.Name, typ: fld.Type, hasDefault: fld.HasDefault, init: fld.Init}
		if _, ok := fld.Type.(*types.Nullable); ok {
			df.nullable = true
		}
		if decl != nil && fld.Index < len(decl.Fields) {
			af := decl.Fields[fld.Index]
			df.keys = wireKeys(af.Attrs)
			if formats, present := skippedFormats(af.Attrs); present {
				if len(formats) == 0 {
					df.skipAll = true
				} else {
					df.skipIn = formats
				}
			}
			df.required = hasAttr(af.Attrs, "required")
		}
		if kind != "Comparable" && !df.skipAll {
			if why := c.notCodable(fld.Type); why != "" {
				return nil, fmt.Sprintf("field '%s' %s; mark it @skip (with a default) or write the impl", fld.Name, why)
			}
		}
		out = append(out, df)
	}
	if kind == "Decodable" {
		seen := map[string]int{}
		for _, df := range out {
			if df.skipAll || df.init {
				continue
			}
			k := df.name
			if df.keys != nil {
				if v, ok := df.keys[""]; ok {
					k = v
				}
			}
			if other, dup := seen[k]; dup {
				return nil, fmt.Sprintf("fields '%s' and '%s' share the key \"%s\"", out[other].name, df.name, k)
			}
			seen[k] = df.index
		}
	}
	return out, ""
}

// notCodable says why a field type can never be encoded or decoded — a
// function, a raw pointer, a handle with no wire form — or "".
func (c *Checker) notCodable(t types.Type) string {
	switch t := t.(type) {
	case *types.Func:
		return "is a function, which has no wire form"
	case *types.Pointer:
		if t.Raw {
			return "is a raw pointer"
		}
		return "is a pointer; wire values are copies"
	case *types.Channel:
		return "is a channel"
	case *types.Task:
		return "is a task"
	case *types.Nullable:
		return c.notCodable(t.Elem)
	case *types.Struct:
		switch t.Name {
		case "Mutex", "Atomic":
			if t.Module == "std.prelude" {
				return "is a " + t.Name + ", which is not a value"
			}
		}
	}
	return ""
}

// inferDeriveBounds gives an empty impl on a generic struct the bounds it
// needs: every type parameter a non-skipped field mentions must be the
// trait (D58: `impl Codable` in `Page<T>` reads as `impl<T: Codable>`).
func (c *Checker) inferDeriveBounds(impl *Impl, trait *types.Trait, fields []derivedField) {
	for _, tp := range impl.TypeParams {
		for _, df := range fields {
			if df.skipAll || !mentions(df.typ, tp) {
				continue
			}
			tp.Bounds = c.withSupers(tp.Bounds, trait)
			break
		}
	}
}

// ---------------------------------------------------------------------------
// the tag of a sealed family

// tagInfo is how a sealed trait's variants are told apart on the wire.
type tagInfo struct {
	key     string // the discriminator key
	content string // "" for internally tagged; the key the fields go under otherwise
}

func (c *Checker) tagOfSealed(s *types.Sealed) tagInfo {
	if d, ok := sealedTemplate(s).Decl.(*ast.TraitDecl); ok {
		k, content := tagOf(d.Attrs)
		return tagInfo{key: k, content: content}
	}
	return tagInfo{key: "type"}
}

// variantName is a variant's name on the wire: `@key("circle")` or its
// declared name.
func variantName(v *types.Struct) string {
	if d, ok := templateOf(v).Decl.(*ast.StructDecl); ok {
		if keys := wireKeys(d.Attrs); keys != nil {
			if n, ok := keys[""]; ok {
				return n
			}
		}
	}
	return v.Name
}

// ---------------------------------------------------------------------------
// a struct

// keyExpr is the key a field is written under, as an expression over the
// encoder or decoder `io`: the field's name styled by the format unless a
// `@key` says otherwise, by format when it does.
func (c *Checker) keyExpr(b *synth, df derivedField, io ast.Expr) ast.Expr {
	styled := b.call(b.name("styleKey"), b.str(df.name), b.mcall(io, "keys"))
	if df.keys == nil {
		return styled
	}
	var fallback ast.Expr = styled
	if k, ok := df.keys[""]; ok {
		fallback = b.str(k)
	}
	var arms []*ast.WhenArm
	for _, format := range sortedKeys(df.keys) {
		if format == "" {
			continue
		}
		arms = append(arms, &ast.WhenArm{Patterns: []ast.Pattern{&ast.LiteralPat{Value: b.str(format)}}, Body: b.str(df.keys[format]), Pos: b.sp})
	}
	if len(arms) == 0 {
		return fallback
	}
	arms = append(arms, &ast.WhenArm{Else: true, Body: fallback, Pos: b.sp})
	return &ast.WhenExpr{Subject: b.mcall(io, "format"), Arms: arms, Pos: b.sp}
}

// skippedHere is `io.format() == "a" || io.format() == "b"` for a field
// skipped in some formats.
func (c *Checker) skippedHere(b *synth, df derivedField, io ast.Expr) ast.Expr {
	var cond ast.Expr
	for _, format := range df.skipIn {
		test := b.bin(lexer.Eq, b.mcall(io, "format"), b.str(format))
		if cond == nil {
			cond = test
		} else {
			cond = b.bin(lexer.OrOr, cond, test)
		}
	}
	return cond
}

func (c *Checker) encodeStruct(b *synth, st *types.Struct, fields []derivedField) *ast.FunDecl {
	to := b.name("to")
	var body []ast.Stmt
	body = append(body, b.stmt(b.try(b.mcall(to, "beginObject"))))
	if st.Sealed != nil {
		tag := c.tagOfSealed(st.Sealed)
		if tag.content == "" {
			body = append(body,
				b.stmt(b.try(b.mcall(to, "key", b.str(tag.key)))),
				b.stmt(b.try(b.mcall(to, "writeString", b.str(variantName(st))))))
		}
	}
	for _, df := range fields {
		if df.skipAll {
			continue
		}
		write := []ast.Stmt{
			b.stmt(b.try(b.mcall(to, "key", c.keyExpr(b, df, to)))),
			b.stmt(b.try(b.mcall(b.member(b.self(), df.name), "encode", to))),
		}
		if len(df.skipIn) > 0 {
			body = append(body, b.ifStmt(b.not(c.skippedHere(b, df, to)), write, nil))
		} else {
			body = append(body, write...)
		}
	}
	body = append(body, b.stmt(b.try(b.mcall(to, "endObject"))))
	return b.fun("encode", false, []ast.Param{b.param("to", c.preludeType("Encoder"))}, nil, c.preludeType("EncodeError"), body)
}

func (c *Checker) decodeStruct(b *synth, st *types.Struct, fields []derivedField) *ast.FunDecl {
	from := b.name("from")
	var body []ast.Stmt
	// per decoded field: its key (computed once), a slot for the value, a
	// flag for whether the key was seen, and whether this format skips it
	type slot struct {
		df      derivedField
		key     string // local holding the key
		val     string // local holding the value
		present string // local: the key was seen
		skipped string // local: skipped in this format, or ""
	}
	var slots []slot
	for _, df := range fields {
		if df.skipAll || df.init {
			continue
		}
		s := slot{df: df, key: fmt.Sprintf("$k%d", df.index), val: fmt.Sprintf("$v%d", df.index)}
		if !df.nullable && !df.hasDefault || df.nullable && df.required {
			s.present = fmt.Sprintf("$p%d", df.index) // its absence is a problem
		}
		body = append(body, b.val(s.key, nil, c.keyExpr(b, df, from)))
		slotType := df.typ
		if !df.nullable {
			slotType = &types.Nullable{Elem: df.typ}
		}
		body = append(body, b.varStmt(s.val, slotType, b.null()))
		if s.present != "" {
			body = append(body, b.varStmt(s.present, nil, b.boolLit(false)))
		}
		if len(df.skipIn) > 0 {
			s.skipped = fmt.Sprintf("$s%d", df.index)
			body = append(body, b.val(s.skipped, nil, c.skippedHere(b, df, from)))
		}
		slots = append(slots, s)
	}
	// a nested value that could not be built has recorded its problems and
	// left the stream after itself: read on, fail at the end
	body = append(body, b.varStmt("$broken", nil, b.boolLit(false)))
	body = append(body, b.stmt(b.try(b.mcall(from, "beginObject"))))
	// loop { val k = try from.nextKey() ?: break; if (k == $k0) ... else try from.skip() }
	var read ast.Stmt = b.stmt(b.try(b.mcall(from, "skip")))
	if st.Sealed != nil {
		// a variant decoded on its own still meets its discriminator
		tag := c.tagOfSealed(st.Sealed)
		if tag.content == "" {
			read = b.ifStmt(b.bin(lexer.Eq, b.name("k"), b.str(tag.key)), []ast.Stmt{b.stmt(b.try(b.mcall(from, "skip")))}, []ast.Stmt{read})
		}
	}
	for i := len(slots) - 1; i >= 0; i-- {
		s := slots[i]
		cond := b.bin(lexer.Eq, b.name("k"), b.name(s.key))
		if s.skipped != "" {
			cond = b.bin(lexer.AndAnd, cond, b.not(b.name(s.skipped)))
		}
		decoded := b.call(b.member(b.typeExpr(s.df.typ), "decode"), from)
		var then []ast.Stmt
		if s.present != "" {
			then = append(then, b.assign(b.name(s.present), b.boolLit(true)))
		}
		then = append(then,
			b.stmt(&ast.WhenExpr{Subject: decoded, Arms: []*ast.WhenArm{
				{Patterns: []ast.Pattern{b.resultPat("Ok", "$ok")}, Body: b.blockExpr([]ast.Stmt{b.assign(b.name(s.val), b.name("$ok"))}), Pos: b.sp},
				{Patterns: []ast.Pattern{b.resultPat("Err", "")}, Body: b.blockExpr([]ast.Stmt{b.assign(b.name("$broken"), b.boolLit(true))}), Pos: b.sp},
			}, Pos: b.sp}),
		)
		read = b.ifStmt(cond, then, []ast.Stmt{read})
	}
	loopBody := []ast.Stmt{
		b.val("k", nil, &ast.ElvisExpr{L: b.try(b.mcall(from, "nextKey")), R: &ast.ControlExpr{Stmt: &ast.BreakStmt{Pos: b.sp}}, Pos: b.sp}),
		read,
	}
	body = append(body, &ast.LoopStmt{Body: b.block(loopBody), Pos: b.sp})
	body = append(body, b.stmt(b.try(b.mcall(from, "endObject"))))
	// what is missing: a required field whose key never came
	for _, s := range slots {
		if s.present == "" {
			continue
		}
		var absent ast.Expr = b.not(b.name(s.present))
		if s.skipped != "" {
			absent = b.bin(lexer.AndAnd, absent, b.not(b.name(s.skipped)))
		}
		body = append(body, b.ifStmt(absent, []ast.Stmt{
			b.stmt(b.mcall(from, "problemAt", b.call(b.name("childPath"), b.mcall(from, "path"), b.name(s.key)), b.str("missing"))),
			b.assign(b.name("$broken"), b.boolLit(true)),
		}, nil))
	}
	failure := b.callArgs(b.typeExpr(c.preludeType("DecodeError")), b.namedArg("problems", b.mcall(from, "problems")))
	body = append(body, b.ifStmt(b.name("$broken"), []ast.Stmt{&ast.ThrowStmt{Value: failure, Pos: b.sp}}, nil))
	// the value
	var args []ast.Arg
	for _, s := range slots {
		var value ast.Expr = b.name(s.val)
		switch {
		case s.df.nullable:
		case s.df.hasDefault:
			value = &ast.ElvisExpr{L: value, R: &ast.FieldDefaultExpr{Struct: st, Index: s.df.index, Pos: b.sp}, Pos: b.sp}
		default:
			value = &ast.ElvisExpr{L: value, R: b.throwExpr(failure), Pos: b.sp}
		}
		args = append(args, b.namedArg(s.df.name, value))
	}
	built := ast.Expr(&ast.CallExpr{Fun: b.typeExpr(st), Args: args, Pos: b.sp})
	if st.Sealed != nil {
		// constructing a variant makes a value of the family; `Self` is the
		// variant, which a narrowing arm gives back
		body = append(body, b.val("$x", nil, built))
		built = &ast.IfExpr{Cond: &ast.IsExpr{X: b.name("$x"), Pat: &ast.TypePat{Type: &ast.ResolvedType{T: st, Pos: b.sp}, Pos: b.sp}, Pos: b.sp},
			Then: b.block([]ast.Stmt{b.stmt(b.name("$x"))}),
			Else: b.block([]ast.Stmt{b.stmt(b.call(b.name("panic"), b.str("unreachable: a freshly built variant")))}), Pos: b.sp}
	}
	body = append(body, b.stmt(built))
	return b.fun("decode", true, []ast.Param{b.param("from", c.preludeType("Decoder"))}, st, c.preludeType("DecodeError"), body)
}

func (c *Checker) compareStruct(b *synth, st *types.Struct, fields []derivedField) *ast.FunDecl {
	ordering := c.preludeType("Ordering")
	equal := b.member(b.typeExpr(ordering), "Equal")
	var body []ast.Stmt
	for _, df := range fields {
		if df.skipAll {
			continue
		}
		local := fmt.Sprintf("$c%d", df.index)
		body = append(body,
			b.val(local, nil, b.mcall(b.member(b.self(), df.name), "compareTo", b.member(b.name("other"), df.name))),
			b.ifStmt(b.bin(lexer.NotEq, b.name(local), equal), []ast.Stmt{&ast.ReturnStmt{Value: b.name(local), Pos: b.sp}}, nil))
	}
	body = append(body, b.stmt(b.member(b.typeExpr(ordering), "Equal")))
	return b.fun("compareTo", false, []ast.Param{b.param("other", st)}, ordering, nil, body)
}

// ---------------------------------------------------------------------------
// a sealed trait

// deriveVariants makes sure every variant implements the trait, deriving
// an impl for each that lacks one; the sealed impl dispatches to them.
func (c *Checker) deriveVariants(d *ast.ImplDecl, impl *Impl, trait *types.Trait, s *types.Sealed) {
	c.resolveSealed(sealedTemplate(s))
	for _, v := range sealedTemplate(s).Variants {
		if c.findImplFor(v, trait) != nil {
			continue
		}
		nd := &ast.ImplDecl{Derived: true, Trait: &ast.ResolvedType{T: trait, Pos: d.Pos}, Target: &ast.ResolvedType{T: v, Pos: d.Pos}, Pos: d.Pos}
		c.declareImpl(impl.Module, impl.File, nd)
	}
}

func (c *Checker) encodeSealed(b *synth, s *types.Sealed) *ast.FunDecl {
	to := b.name("to")
	tag := c.tagOfSealed(s)
	var arms []*ast.WhenArm
	for _, v := range sealedTemplate(s).Variants {
		var arm ast.Expr
		if tag.content == "" {
			arm = b.try(b.mcall(b.self(), "encode", to))
		} else {
			arm = b.blockExpr([]ast.Stmt{
				b.stmt(b.try(b.mcall(to, "beginObject"))),
				b.stmt(b.try(b.mcall(to, "key", b.str(tag.key)))),
				b.stmt(b.try(b.mcall(to, "writeString", b.str(variantName(v))))),
				b.stmt(b.try(b.mcall(to, "key", b.str(tag.content)))),
				b.stmt(b.try(b.mcall(b.self(), "encode", to))),
				b.stmt(b.try(b.mcall(to, "endObject"))),
			})
		}
		arms = append(arms, &ast.WhenArm{Patterns: []ast.Pattern{&ast.TypePat{Type: &ast.ResolvedType{T: v, Pos: b.sp}, Pos: b.sp}}, Body: arm, Pos: b.sp})
	}
	body := []ast.Stmt{b.stmt(&ast.WhenExpr{Subject: b.self(), Arms: arms, Pos: b.sp})}
	return b.fun("encode", false, []ast.Param{b.param("to", c.preludeType("Encoder"))}, nil, c.preludeType("EncodeError"), body)
}

func (c *Checker) decodeSealed(b *synth, s *types.Sealed) *ast.FunDecl {
	from := b.name("from")
	tag := c.tagOfSealed(s)
	value := c.preludeType("Value")
	failure := func() ast.Stmt {
		return &ast.ThrowStmt{Value: b.callArgs(b.typeExpr(c.preludeType("DecodeError")), b.namedArg("problems", b.mcall(from, "problems"))), Pos: b.sp}
	}
	var body []ast.Stmt
	// val $v = try Value.decode(from)
	body = append(body, b.val("$v", nil, b.try(b.call(b.member(b.typeExpr(value), "decode"), from))))
	// val $tag = $v.get(KEY)?.asString()
	body = append(body, b.val("$tag", nil, b.mcallSafe(b.mcall(b.name("$v"), "get", b.str(tag.key)), "asString")))
	body = append(body, b.ifStmt(b.bin(lexer.Eq, b.name("$tag"), b.null()), []ast.Stmt{
		b.stmt(b.mcall(from, "problem", b.str(fmt.Sprintf("an object with a \"%s\" naming the variant of %s was expected", tag.key, s.Name)))),
		failure(),
	}, nil))
	// the tree the variant reads from
	var tree ast.Expr = b.name("$v")
	if tag.content != "" {
		tree = &ast.ElvisExpr{L: b.mcall(b.name("$v"), "get", b.str(tag.content)), R: b.call(b.typeExpr(c.preludeType("VNull"))), Pos: b.sp}
	}
	body = append(body, b.val("$sub", nil, b.call(b.member(b.typeExpr(c.preludeType("ValueDecoder")), "of"), tree,
		b.mcall(from, "format"), b.mcall(from, "enums"), b.mcall(from, "keys"))))
	var arms []*ast.WhenArm
	var names []string
	for _, v := range sealedTemplate(s).Variants {
		names = append(names, variantName(v))
		// bound to the sealed type first: the arm's value is the variant, the
		// when's the family
		arms = append(arms, &ast.WhenArm{Patterns: []ast.Pattern{&ast.LiteralPat{Value: b.str(variantName(v))}},
			Body: b.blockExpr([]ast.Stmt{
				b.val("$c", s, b.try(b.call(b.member(b.typeExpr(v), "decode"), b.name("$sub")))),
				b.stmt(b.name("$c")),
			}), Pos: b.sp})
	}
	arms = append(arms, &ast.WhenArm{Else: true, Body: b.blockExpr([]ast.Stmt{
		b.stmt(b.mcall(from, "problem", b.interp("unknown variant \"", b.name("$tag"), fmt.Sprintf("\" of %s (one of %s)", s.Name, joinQuoted(names))))),
		failure(),
	}), Pos: b.sp})
	body = append(body, b.val("$out", s, &ast.WhenExpr{Subject: b.name("$tag"), Arms: arms, Pos: b.sp}))
	// the variant's problems, re-rooted at this object's path
	body = append(body, &ast.LoopStmt{Var: &ast.Binding{Name: b.ident("$p"), Pos: b.sp}, Iter: b.mcall(b.name("$sub"), "problems"),
		Body: b.block([]ast.Stmt{b.stmt(b.mcall(from, "problemAt", b.call(b.name("joinPath"), b.mcall(from, "path"), b.member(b.name("$p"), "path")), b.member(b.name("$p"), "message")))}), Pos: b.sp})
	body = append(body, b.stmt(b.name("$out")))
	return b.fun("decode", true, []ast.Param{b.param("from", c.preludeType("Decoder"))}, s, c.preludeType("DecodeError"), body)
}

func joinQuoted(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += "\"" + n + "\""
	}
	return out
}

// ---------------------------------------------------------------------------
// an enum

// memberName is a member's name on the wire: `@key("active")` or as declared.
func memberName(e *types.Enum, m *types.EnumMember) string {
	if d, ok := e.Decl.(*ast.EnumDecl); ok {
		for _, md := range d.Members {
			if md.Name.Name == m.Name {
				if keys := wireKeys(md.Attrs); keys != nil {
					if n, ok := keys[""]; ok {
						return n
					}
				}
			}
		}
	}
	return m.Name
}

func (c *Checker) encodeEnum(b *synth, e *types.Enum) *ast.FunDecl {
	to := b.name("to")
	c.resolveEnum(e)
	var arms []*ast.WhenArm
	for _, m := range e.Members {
		arms = append(arms, &ast.WhenArm{Patterns: []ast.Pattern{&ast.LiteralPat{Value: b.member(b.typeExpr(e), m.Name)}}, Body: b.str(memberName(e, m)), Pos: b.sp})
	}
	byName := b.try(b.mcall(to, "writeString", &ast.WhenExpr{Subject: b.self(), Arms: arms, Pos: b.sp}))
	byNumber := b.try(b.mcall(to, "writeI64", &ast.CastExpr{X: b.member(b.self(), "value"), Type: &ast.ResolvedType{T: types.TI64, Pos: b.sp}, Pos: b.sp}))
	numeric := b.bin(lexer.Eq, b.mcall(to, "enums"), b.member(b.typeExpr(c.preludeType("EnumStyle")), "Number"))
	body := []ast.Stmt{b.ifStmt(numeric, []ast.Stmt{b.stmt(byNumber)}, []ast.Stmt{b.stmt(byName)})}
	return b.fun("encode", false, []ast.Param{b.param("to", c.preludeType("Encoder"))}, nil, c.preludeType("EncodeError"), body)
}

func (c *Checker) decodeEnum(b *synth, e *types.Enum) *ast.FunDecl {
	from := b.name("from")
	c.resolveEnum(e)
	first := b.member(b.typeExpr(e), e.Members[0].Name)
	var names []string
	var arms []*ast.WhenArm
	for _, m := range e.Members {
		names = append(names, memberName(e, m))
		arms = append(arms, &ast.WhenArm{Patterns: []ast.Pattern{&ast.LiteralPat{Value: b.str(memberName(e, m))}}, Body: b.member(b.typeExpr(e), m.Name), Pos: b.sp})
	}
	arms = append(arms, &ast.WhenArm{Else: true, Body: b.blockExpr([]ast.Stmt{
		b.stmt(b.mcall(from, "problem", b.interp("\"", b.name("$s"), fmt.Sprintf("\" is not a %s (one of %s)", e.Name, joinQuoted(names))))),
		b.stmt(first),
	}), Pos: b.sp})
	byName := []ast.Stmt{
		b.val("$s", nil, b.try(b.mcall(from, "readString"))),
		b.stmt(&ast.WhenExpr{Subject: b.name("$s"), Arms: arms, Pos: b.sp}),
	}
	byNumber := []ast.Stmt{
		b.val("$n", nil, b.try(b.mcall(from, "readI64"))),
		b.stmt(&ast.ElvisExpr{
			L: b.call(b.member(b.typeExpr(e), "fromValue"), &ast.CastExpr{X: b.name("$n"), Type: &ast.ResolvedType{T: e.Base, Pos: b.sp}, Pos: b.sp}),
			R: b.blockExpr([]ast.Stmt{
				b.stmt(b.mcall(from, "problem", b.interp("", b.name("$n"), fmt.Sprintf(" is not a %s", e.Name)))),
				b.stmt(first),
			}), Pos: b.sp}),
	}
	numeric := b.bin(lexer.Eq, b.mcall(from, "enums"), b.member(b.typeExpr(c.preludeType("EnumStyle")), "Number"))
	body := []ast.Stmt{b.stmt(&ast.IfExpr{Cond: numeric, Then: b.block(byNumber), Else: b.block(byName), Pos: b.sp})}
	return b.fun("decode", true, []ast.Param{b.param("from", c.preludeType("Decoder"))}, e, c.preludeType("DecodeError"), body)
}

// deriveEnumCodecs gives every enum its Encodable and Decodable impls
// (D58: an enum is Codable with nothing to write, like a primitive).
func (c *Checker) deriveEnumCodecs() {
	enc, _ := c.preludeType("Encodable").(*types.Trait)
	dec, _ := c.preludeType("Decodable").(*types.Trait)
	if enc == nil || dec == nil {
		return
	}
	for _, e := range c.enums {
		ctx := c.enumDecl[e]
		if ctx == nil || len(e.Members) == 0 {
			continue
		}
		d := ctx.decl.(*ast.EnumDecl)
		// a zero-width span at the name: diagnostics land on the enum, and
		// nothing the derived code records shadows the name's own hover
		at := source.Span{File: d.Name.Pos.File, Start: d.Name.Pos.Start, End: d.Name.Pos.Start}
		for _, trait := range []*types.Trait{enc, dec} {
			nd := &ast.ImplDecl{Derived: true, Trait: &ast.ResolvedType{T: trait, Pos: at}, Target: &ast.ResolvedType{T: e, Pos: at}, Pos: at}
			c.declareImpl(ctx.module, ctx.file, nd)
		}
	}
}

// ---------------------------------------------------------------------------
// supertraits

// superImplReq is an impl of a trait with supertraits, waiting for every
// impl in the program to be declared before the supers are derived.
type superImplReq struct {
	module  *Module
	file    *ast.File
	decl    *ast.ImplDecl
	impl    *Impl
	trait   *types.Trait
	methods map[*types.Trait][]*ast.FunDecl // written in this body, owned by a super
}

// routeSuperMethods moves the methods of an impl body that belong to a
// supertrait out of the body, to be declared with that super's impl.
func (c *Checker) routeSuperMethods(d *ast.ImplDecl, trait *types.Trait) map[*types.Trait][]*ast.FunDecl {
	if len(trait.Supers) == 0 {
		return nil
	}
	routed := map[*types.Trait][]*ast.FunDecl{}
	var own []*ast.FunDecl
	for _, md := range d.Methods {
		if _, mine := trait.Methods[md.Name.Name]; mine {
			own = append(own, md)
			continue
		}
		if s := superOwning(trait, md.Name.Name); s != nil {
			routed[s] = append(routed[s], md)
			continue
		}
		own = append(own, md) // reported as unknown by the caller
	}
	d.Methods = own
	return routed
}

// deriveSupers, once every impl is declared: each super the type lacks
// gets an impl — with the methods the author wrote for it, the rest
// derived — and methods written for a super the type already implements
// are an error.
func (c *Checker) deriveSupers(req superImplReq) {
	for _, s := range req.trait.Supers {
		written := req.methods[s]
		if existing := c.findImplFor(req.impl.Target, s); existing != nil {
			for _, md := range written {
				c.errorf(md.Name.Pos, "'%s' belongs to '%s', which '%s' already implements at %s; write it there", md.Name.Name, s.Name, req.impl.Target, existing.Decl.Pos)
			}
			continue
		}
		nd := &ast.ImplDecl{Derived: true, Inline: req.decl.Inline, TypeParams: req.decl.TypeParams,
			Trait: &ast.ResolvedType{T: s, Pos: req.decl.Pos}, Target: req.decl.Target, Methods: written, Pos: req.decl.Pos}
		c.declareImpl(req.module, req.file, nd)
	}
}

// ---------------------------------------------------------------------------
// building syntax

// synth builds syntax nodes that all carry one span: the impl's, so any
// diagnostic in derived code points at the line that asked for it.
type synth struct {
	sp source.Span
}

func (b *synth) ident(name string) *ast.Ident { return &ast.Ident{Name: name, Pos: b.sp} }
func (b *synth) name(name string) ast.Expr  { return &ast.NameExpr{Name: name, Pos: b.sp} }
func (b *synth) self() ast.Expr             { return &ast.SelfExpr{Pos: b.sp} }
func (b *synth) null() ast.Expr             { return &ast.NullLit{Pos: b.sp} }
func (b *synth) boolLit(v bool) ast.Expr    { return &ast.BoolLit{Value: v, Pos: b.sp} }
func (b *synth) str(s string) ast.Expr {
	return &ast.StringLit{Parts: []ast.StringPart{{Text: s}}, Pos: b.sp}
}

// interp is `"before${x}after"`.
func (b *synth) interp(before string, x ast.Expr, after string) ast.Expr {
	return &ast.StringLit{Parts: []ast.StringPart{{Text: before}, {Expr: x}, {Text: after}}, Pos: b.sp}
}
func (b *synth) typeExpr(t types.Type) ast.Expr {
	return &ast.TypeExpr{Type: &ast.ResolvedType{T: t, Pos: b.sp}, Pos: b.sp}
}
func (b *synth) member(x ast.Expr, name string) ast.Expr {
	return &ast.MemberExpr{X: x, Name: ast.Ident{Name: name, Pos: b.sp}, Pos: b.sp}
}
func (b *synth) call(fn ast.Expr, args ...ast.Expr) ast.Expr {
	var out []ast.Arg
	for _, a := range args {
		out = append(out, ast.Arg{Value: a})
	}
	return &ast.CallExpr{Fun: fn, Args: out, Pos: b.sp}
}
func (b *synth) callArgs(fn ast.Expr, args ...ast.Arg) ast.Expr {
	return &ast.CallExpr{Fun: fn, Args: args, Pos: b.sp}
}
func (b *synth) namedArg(name string, v ast.Expr) ast.Arg {
	return ast.Arg{Name: b.ident(name), Value: v}
}
func (b *synth) mcall(x ast.Expr, name string, args ...ast.Expr) ast.Expr {
	return b.call(b.member(x, name), args...)
}
func (b *synth) mcallSafe(x ast.Expr, name string, args ...ast.Expr) ast.Expr {
	m := &ast.MemberExpr{X: x, Name: ast.Ident{Name: name, Pos: b.sp}, Safe: true, Pos: b.sp}
	return b.call(m, args...)
}
func (b *synth) try(x ast.Expr) ast.Expr { return &ast.TryExpr{X: x, Pos: b.sp} }
func (b *synth) not(x ast.Expr) ast.Expr { return &ast.UnaryExpr{Op: lexer.Bang, X: x, Pos: b.sp} }
func (b *synth) bin(op lexer.TokenKind, l, r ast.Expr) ast.Expr {
	return &ast.BinaryExpr{Op: op, L: l, R: r, Pos: b.sp}
}
func (b *synth) throwExpr(v ast.Expr) ast.Expr {
	return &ast.ControlExpr{Stmt: &ast.ThrowStmt{Value: v, Pos: b.sp}}
}
func (b *synth) stmt(x ast.Expr) ast.Stmt { return &ast.ExprStmt{X: x} }
func (b *synth) block(stmts []ast.Stmt) *ast.Block {
	return &ast.Block{Stmts: stmts, Pos: b.sp}
}
func (b *synth) blockExpr(stmts []ast.Stmt) ast.Expr { return &ast.BlockExpr{Block: b.block(stmts)} }
func (b *synth) typ(t types.Type) ast.Type {
	if t == nil {
		return nil
	}
	return &ast.ResolvedType{T: t, Pos: b.sp}
}
func (b *synth) val(name string, t types.Type, v ast.Expr) ast.Stmt {
	return &ast.ValStmt{Kind: ast.BindVal, Binding: ast.Binding{Name: b.ident(name), Type: b.typ(t), Pos: b.sp}, Value: v, Pos: b.sp}
}
func (b *synth) varStmt(name string, t types.Type, v ast.Expr) ast.Stmt {
	return &ast.ValStmt{Kind: ast.BindVar, Binding: ast.Binding{Name: b.ident(name), Type: b.typ(t), Pos: b.sp}, Value: v, Pos: b.sp}
}
func (b *synth) assign(target, v ast.Expr) ast.Stmt {
	return &ast.AssignStmt{Target: target, Op: lexer.Assign, Value: v, Pos: b.sp}
}
func (b *synth) ifStmt(cond ast.Expr, then, els []ast.Stmt) ast.Stmt {
	x := &ast.IfExpr{Cond: cond, Then: b.block(then), Pos: b.sp}
	if els != nil {
		x.Else = b.block(els)
	}
	return &ast.ExprStmt{X: x}
}
func (b *synth) param(name string, t types.Type) ast.Param {
	return ast.Param{Name: ast.Ident{Name: name, Pos: b.sp}, Type: b.typ(t), Pos: b.sp}
}

// fun builds a method (or, static, a function on the type) with a block
// body; `throws` names the error type, or nil.
func (b *synth) fun(name string, static bool, params []ast.Param, ret types.Type, throws types.Type, body []ast.Stmt) *ast.FunDecl {
	fd := &ast.FunDecl{Static: static, Name: ast.Ident{Name: name, Pos: b.sp}, Params: params, Ret: b.typ(ret), Body: b.block(body), Pos: b.sp}
	if throws != nil {
		fd.Effects = ast.Effects{Throws: true, ThrowsSpan: b.sp, Error: b.typ(throws)}
	}
	return fd
}

// resultPat is `is Ok(name)` / `is Err` over a Result.
func (b *synth) resultPat(variant, bind string) ast.Pattern {
	p := &ast.TypePat{Type: &ast.NamedType{Path: []ast.Ident{{Name: variant, Pos: b.sp}}, Pos: b.sp}, Pos: b.sp}
	if bind != "" {
		p.HasArg = true
		p.Fields = []ast.FieldPat{{Name: ast.Ident{Name: bind, Pos: b.sp}}}
	}
	return p
}
