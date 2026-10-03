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
//	Array<T, N> [N x T]                                   (inline, D121)
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
	case *types.Array:
		return fmt.Sprintf("[%d x %s]", arrayLen(t), g.llType(t.Elem))
	case *types.Range:
		et := g.llType(t.Elem)
		return "{ " + et + ", " + et + ", i1 }"
	case *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return "ptr"
	case *types.Func:
		if t.C {
			return "ptr" // a C function pointer (D69)
		}
		return "{ ptr, ptr }" // code pointer + environment
	case *types.Trait:
		return "{ ptr, ptr }" // boxed data + vtable (D9)
	case *types.Struct:
		return g.structType(t)
	case *types.Sealed:
		return g.sealedType(t)
	case *types.ErrorUnion:
		return g.unionType(t)
	case *types.Enum:
		return g.llType(t.Base) // a value is its integer (D57)
	}
	panic(fmt.Sprintf("llType: unsupported type %s (%T)", t, t))
}

// isPtrLike reports whether a type is represented as a raw `ptr`, giving
// its nullable form a null niche.
func isPtrLike(t types.Type) bool {
	switch t := t.(type) {
	case *types.Pointer, *types.List, *types.Map, *types.Set, *types.Channel, *types.Task:
		return true
	case *types.Func:
		return t.C // a nullable C function pointer is NULL or the address, as in C
	}
	return false
}

func (g *gen) structType(s *types.Struct) string {
	name := "%S." + mangleType(s)
	if _, done := g.typeDecls[name]; done {
		return name
	}
	g.typeDecls[name] = "" // reserve (recursion through pointers is fine)
	g.typeDecls[name] = g.structBody(s)
	g.typeOrder = append(g.typeOrder, name)
	return name
}

// structBody writes a struct's LLVM type from its layout (D120). A packed
// struct is LLVM's packed struct; a union an integer as aligned as the
// union and the bytes after it, read and written through memory. Otherwise
// the fields in order, with `[n x i8]` before a field wherever its offset
// needs more than LLVM's own placement would give (an `@align(n)` field, a
// field of an over-aligned struct type) and after the last one when the
// size needs more. Such a struct records where each field went (fidx), and
// an over-aligned one its alignment, for allocas and globals (overAlign).
func (g *gen) structBody(s *types.Struct) string {
	if len(s.Fields) == 0 {
		return "{}"
	}
	p := g.layouter().StructPlan(s) // also makes fieldMap and overAlign
	if s.Union {
		a := min(p.Align, 16)
		body := fmt.Sprintf("{ i%d", a*8)
		if p.Size > a {
			body += fmt.Sprintf(", [%d x i8]", p.Size-a)
		}
		return body + " }"
	}
	var parts []string
	idx := make([]int, len(s.Fields))
	remapped := false
	at := 0
	for i, f := range s.Fields {
		natural := at
		if !s.Packed {
			natural = roundUpTo(at, g.llAlign(f.Type))
		}
		if p.Offsets[i] > natural {
			parts = append(parts, fmt.Sprintf("[%d x i8]", p.Offsets[i]-at))
			remapped = true
		}
		idx[i] = len(parts)
		parts = append(parts, g.llType(f.Type))
		fs, _ := g.layout(f.Type)
		at = p.Offsets[i] + fs
	}
	llSize := at
	if !s.Packed {
		llSize = roundUpTo(at, p.LLAlign)
	}
	if llSize != p.Size {
		parts = append(parts, fmt.Sprintf("[%d x i8]", p.Size-at))
	}
	if remapped {
		g.fieldMap[mangleType(s)] = idx
	}
	if p.Align > p.LLAlign {
		g.overAlign["%S."+mangleType(s)] = p.Align
	}
	body := "{ " + strings.Join(parts, ", ") + " }"
	if s.Packed {
		body = "<" + body + ">"
	}
	return body
}

// fidx is the LLVM index of field i of a value of type t: i itself, unless
// the struct's type carries padding before it (structBody).
func (g *gen) fidx(t types.Type, i int) int {
	if st, ok := t.(*types.Struct); ok {
		g.structType(st)
		if m := g.fieldMap[mangleType(st)]; m != nil {
			return m[i]
		}
	}
	return i
}

func roundUpTo(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) / a * a
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

// layouter is the layout state, made on first use.
func (g *gen) layouter() *types.Layout {
	if g.lay == nil {
		g.lay = &types.Layout{
			Sealed:     func(s *types.Sealed) int { return 8 + 8*g.payloadWords(s) },
			ErrorUnion: func(u *types.ErrorUnion) int { return 8 + 8*g.unionWords(u) },
		}
		g.fieldMap = map[string][]int{}
		g.overAlign = map[string]int{}
	}
	return g.lay
}

// layout is a value's size and guaranteed alignment (types.Layout: one
// computation for the checker and the code generator, D120 included).
func (g *gen) layout(t types.Type) (size, align int) {
	return g.layouter().Of(t)
}

// llAlign is the alignment LLVM gives t's representation: what places a
// value inside a tuple, a nullable or another unpadded aggregate.
func (g *gen) llAlign(t types.Type) int {
	return g.layouter().LLAlign(t)
}

// mangleType produces an identifier-safe name for a type.
// mangleType names a type in LLVM symbols. It starts from types.Key rather
// than the display string so that two structs with one name in different
// modules (a user's `error Timeout` next to the prelude's) get distinct
// LLVM types, descriptors, show/eq/hash helpers and vtables.
func mangleType(t types.Type) string {
	s := types.Key(t)
	r := strings.NewReplacer("<", "_", ">", "_", ", ", "_", ",", "_", "*", "P", "?", "N", "(", "T_", ")", "_", " ", "", "|", "_or_", "!", "never", "raw", "R", ":", "_", "&", "A", "#", "_")
	return r.Replace(s)
}
