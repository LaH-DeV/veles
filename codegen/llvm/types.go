package llvm

import (
	"fmt"
	"strings"

	"github.com/LaH-DeV/veles/types"
)

// Type lowering (spec §5 / D5 / D12):
//
//	unit        {}
//	bool        i1
//	iN, uN      iN            (isize/usize are i64)
//	f32, f64    float, double
//	string      %str = { ptr, i64 }
//	*T, *raw T  ptr
//	T?          ptr when T is a pointer (niche), else { i1, T }
//	(A, B)      { A, B }
//	Range<T>    { T, T, i1 }
//	List<T>     ptr (runtime object)
//	struct      %S.name = { fields }
//	sealed      %V.name = { i32 tag, [N x i64] payload }  (inline tagged union)
//	A | B       { i32 tag, [N x i64] }                    (error union)

const strType = "%str"

// llType returns the LLVM type for a Veles type, declaring named types on
// first use.
func (g *gen) llType(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		switch t.Kind {
		case types.Unit, types.Never, types.Invalid:
			return "{}"
		case types.Bool:
			return "i1"
		case types.I8, types.U8:
			return "i8"
		case types.I16, types.U16:
			return "i16"
		case types.I32, types.U32:
			return "i32"
		case types.I64, types.U64, types.ISize, types.USize:
			return "i64"
		case types.F32:
			return "float"
		case types.F64:
			return "double"
		case types.String:
			return strType
		}
	case *types.Pointer:
		return "ptr"
	case *types.Nullable:
		if isPtrLike(t.Elem) {
			return "ptr"
		}
		return "{ i1, " + g.llType(t.Elem) + " }"
	case *types.Tuple:
		parts := make([]string, len(t.Elems))
		for i, e := range t.Elems {
			parts[i] = g.llType(e)
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	case *types.Range:
		et := g.llType(t.Elem)
		return "{ " + et + ", " + et + ", i1 }"
	case *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return "ptr"
	case *types.Func:
		return "{ ptr, ptr }" // code pointer + environment
	case *types.Trait:
		return "{ ptr, ptr }" // boxed data + vtable (D9)
	case *types.Struct:
		return g.structType(t)
	case *types.Sealed:
		return g.sealedType(t)
	case *types.ErrorUnion:
		return g.unionType(t)
	}
	panic(fmt.Sprintf("llType: unsupported type %s (%T)", t, t))
}

// isPtrLike reports whether a type is represented as a raw `ptr`, giving
// its nullable form a null niche.
func isPtrLike(t types.Type) bool {
	switch t.(type) {
	case *types.Pointer, *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return true
	}
	return false
}

func (g *gen) structType(s *types.Struct) string {
	name := "%S." + mangleType(s)
	if _, done := g.typeDecls[name]; done {
		return name
	}
	g.typeDecls[name] = "" // reserve (recursion through pointers is fine)
	parts := make([]string, len(s.Fields))
	for i, f := range s.Fields {
		parts[i] = g.llType(f.Type)
	}
	body := "{}"
	if len(parts) > 0 {
		body = "{ " + strings.Join(parts, ", ") + " }"
	}
	g.typeDecls[name] = body
	g.typeOrder = append(g.typeOrder, name)
	return name
}

func (g *gen) sealedType(s *types.Sealed) string {
	name := "%V." + mangleType(s)
	if _, done := g.typeDecls[name]; done {
		return name
	}
	g.typeDecls[name] = ""
	words := 0
	for _, v := range s.Variants {
		sz, _ := g.layout(v)
		if w := (sz + 7) / 8; w > words {
			words = w
		}
	}
	g.typeDecls[name] = fmt.Sprintf("{ i32, [%d x i64] }", words)
	g.typeOrder = append(g.typeOrder, name)
	return name
}

func (g *gen) unionType(u *types.ErrorUnion) string {
	name := "%U." + mangleType(u)
	if _, done := g.typeDecls[name]; done {
		return name
	}
	g.typeDecls[name] = ""
	g.typeDecls[name] = fmt.Sprintf("{ i32, [%d x i64] }", g.unionWords(u))
	g.typeOrder = append(g.typeOrder, name)
	return name
}

func (g *gen) unionWords(t types.Type) int {
	words := 0
	for _, m := range types.UnionMembers(t) {
		sz, _ := g.layout(m)
		if w := (sz + 7) / 8; w > words {
			words = w
		}
	}
	return words
}

// payloadWords returns the number of i64 words in a sealed/union payload.
func (g *gen) payloadWords(t types.Type) int {
	switch t := t.(type) {
	case *types.Sealed:
		words := 0
		for _, v := range t.Variants {
			sz, _ := g.layout(v)
			if w := (sz + 7) / 8; w > words {
				words = w
			}
		}
		return words
	case *types.ErrorUnion:
		return g.unionWords(t)
	}
	return 1
}

// taggedType returns the LLVM type of a sealed or union value, or of a
// single-member error type which is represented as itself.
func (g *gen) errType(t types.Type) string {
	return g.llType(t)
}

// layout computes size and alignment following the x86-64 (and AArch64)
// natural layout rules, which is what LLVM uses for these types.
func (g *gen) layout(t types.Type) (size, align int) {
	switch t := t.(type) {
	case *types.Basic:
		switch t.Kind {
		case types.Unit, types.Never, types.Invalid:
			return 0, 1
		case types.Bool, types.I8, types.U8:
			return 1, 1
		case types.I16, types.U16:
			return 2, 2
		case types.I32, types.U32, types.F32:
			return 4, 4
		case types.String:
			return 16, 8
		default:
			return 8, 8
		}
	case *types.Pointer, *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return 8, 8
	case *types.Func, *types.Trait:
		return 16, 8
	case *types.Nullable:
		if isPtrLike(t.Elem) {
			return 8, 8
		}
		return structLayout([]types.Type{types.TBool, t.Elem}, g)
	case *types.Tuple:
		return structLayout(t.Elems, g)
	case *types.Range:
		return structLayout([]types.Type{t.Elem, t.Elem, types.TBool}, g)
	case *types.Struct:
		var fs []types.Type
		for _, f := range t.Fields {
			fs = append(fs, f.Type)
		}
		return structLayout(fs, g)
	case *types.Sealed:
		return 8 + 8*g.payloadWords(t), 8
	case *types.ErrorUnion:
		return 8 + 8*g.unionWords(t), 8
	}
	return 8, 8
}

func structLayout(fields []types.Type, g *gen) (size, align int) {
	align = 1
	for _, f := range fields {
		fs, fa := g.layout(f)
		if fa > align {
			align = fa
		}
		size = (size + fa - 1) / fa * fa
		size += fs
	}
	size = (size + align - 1) / align * align
	return size, align
}

// mangleType produces an identifier-safe name for a type.
func mangleType(t types.Type) string {
	s := t.String()
	r := strings.NewReplacer("<", "_", ">", "_", ", ", "_", ",", "_", "*", "P", "?", "N", "(", "T_", ")", "_", " ", "", "|", "_or_", "!", "never", "raw", "R", ":", "_", "&", "A")
	return r.Replace(s)
}
