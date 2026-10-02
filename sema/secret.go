package sema

import (
	"path/filepath"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The prelude's Secret<T> (D112) holds text or bytes in storage the
// collector zeroes when it frees it. Three std-only builtins reach the
// runtime; bodies are checked per instantiation, so `T` is concrete here:
//
//	secretOf(v)          a wiped List<u8> copy of a string or List<u8>
//	secretWipe(bytes)    zero the storage now (close)
//	secretExpose<T>(bs)  a fresh, ordinary T copied out of it
var secretBuiltins = map[string]int{"secretOf": 1, "secretWipe": 1, "secretExpose": 1}

// isSecretStruct: the prelude's Secret, template or instance.
func isSecretStruct(t types.Type) bool {
	st, ok := t.(*types.Struct)
	if !ok {
		return false
	}
	st = templateOf(st)
	return st.Name == "Secret" && st.Module == "std.prelude"
}

// nonNull is t without its `?`.
func nonNull(t types.Type) types.Type {
	if n, ok := t.(*types.Nullable); ok {
		return n.Elem
	}
	return t
}

// secretHolds: the types a Secret may hold.
func secretHolds(t types.Type) bool {
	if types.Identical(t, types.TString) {
		return true
	}
	l, ok := t.(*types.List)
	return ok && !l.Mutable && types.Identical(l.Elem, types.TU8)
}

// checkSecretArg refuses Secret<T> for any T but text or bytes (D112),
// once per type: where it is first written. Instantiating the prelude's
// bodies for it says nothing more (secretCall stays quiet for it).
func (c *Checker) checkSecretArg(tmpl *types.Struct, args []types.Type, span source.Span) {
	if !isSecretStruct(tmpl) || len(args) != 1 || types.ContainsTypeParam(args[0]) || types.IsInvalid(args[0]) {
		return
	}
	if secretHolds(args[0]) || c.seen["secret:"+types.Key(args[0])] {
		return
	}
	if span.File == nil || strings.HasPrefix(filepath.ToSlash(span.File.Path), "std/") {
		return // a synthesized or prelude site: the user's own is reported
	}
	c.seen["secret:"+types.Key(args[0])] = true
	c.errorf(span, "a Secret holds text or bytes: 'Secret<string>' or 'Secret<List<u8>>', not 'Secret<%s>' (D112)", args[0])
}

func (f *fnCtx) secretCall(name string, typeArgs []types.Type, e *ast.CallExpr) Expr {
	if len(e.Args) != 1 {
		f.errorf(e.Pos, "%s takes 1 argument", name)
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	bytes := &types.List{Elem: types.TU8}
	switch name {
	case "secretOf":
		v := f.checkExpr(e.Args[0].Value, nil)
		if !secretHolds(v.Type()) {
			return bad() // Secret<T> for this T was refused where it was written
		}
		return &Builtin{exprBase{bytes}, "secret.of", []Expr{v}, e.Pos}
	case "secretWipe":
		v := f.checkExprTo(e.Args[0].Value, bytes)
		return &Builtin{exprBase{types.TUnit}, "secret.wipe", []Expr{v}, e.Pos}
	}
	if len(typeArgs) != 1 || !secretHolds(typeArgs[0]) {
		f.errorf(e.Pos, "secretExpose takes the type to copy out: secretExpose<string>(bytes) or secretExpose<List<u8>>(bytes)")
		f.checkArgsLoosely(e.Args)
		return bad()
	}
	v := f.checkExprTo(e.Args[0].Value, bytes)
	return &Builtin{exprBase{typeArgs[0]}, "secret.expose", []Expr{v}, e.Pos}
}
