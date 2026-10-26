package types

import "sort"

// Layout is how a value lies in memory: C's rules — each field at the next
// offset its alignment allows, the size a multiple of the alignment — with
// D120's attributes on structs. The checker reads it for `@align`'s
// checks, the code generator for LLVM types, offsets and descriptors, so
// the two cannot disagree.
//
// Two alignments are kept apart. Align is what the program guarantees:
// `@align(64)` raises it, `@packed` lowers it to 1. LLAlign is what LLVM
// would give the type's representation on its own; it is what places a
// value inside a tuple, a nullable or a range, whose LLVM types are written
// without explicit padding, while a struct's LLVM type carries padding
// wherever its fields' offsets need more than LLVM would do (StructPlan).
type Layout struct {
	// Sealed and ErrorUnion size the tagged types, which the code
	// generator lays out in words; nil means 8 bytes of tag and nothing else.
	Sealed     func(*Sealed) int
	ErrorUnion func(*ErrorUnion) int

	plans map[*Struct]StructPlan
}

// Of is a value's size and guaranteed alignment.
func (l *Layout) Of(t Type) (size, align int) {
	switch t := t.(type) {
	case *Basic:
		switch t.Kind {
		case Unit, Never, Invalid:
			return 0, 1
		case Bool, I8, U8:
			return 1, 1
		case I16, U16:
			return 2, 2
		case I32, U32, F32:
			return 4, 4
		case String:
			return 16, 8
		}
		return 8, 8
	case *Pointer, *List, *Map, *Set, *Channel, *Task:
		return 8, 8
	case *Func:
		if t.C {
			return 8, 8
		}
		return 16, 8
	case *Trait:
		return 16, 8
	case *Nullable:
		if PtrLike(t.Elem) {
			return 8, 8
		}
		if _, ok := l.Niche(t.Elem); ok {
			return l.Of(t.Elem)
		}
		return l.natural([]Type{TBool, t.Elem})
	case *Tuple:
		return l.natural(t.Elems)
	case *Array:
		size, align := l.Of(t.Elem)
		n, _ := t.N()
		return size * int(n), align
	case *Range:
		return l.natural([]Type{t.Elem, t.Elem, TBool})
	case *Struct:
		p := l.StructPlan(t)
		return p.Size, p.Align
	case *Sealed:
		if l.Sealed != nil {
			return l.Sealed(t), 8
		}
		return 8, 8
	case *ErrorUnion:
		if l.ErrorUnion != nil {
			return l.ErrorUnion(t), 8
		}
		return 8, 8
	case *Enum:
		return l.Of(t.Base)
	}
	return 8, 8
}

// PayloadOffset is where a sealed value's or an error union's payload
// starts: after the tag, at its words' alignment. The runtime's fail-fast
// `await` checks it (veles_task_set_unwrap).
const PayloadOffset = 8

// NichePath leads to a word that is never zero in a valid value (D214):
// Path indexes struct fields, tuple elements, element 0 of an array and
// the part of a closure (its code) or trait object (its method table);
// Offset is that word's byte offset.
type NichePath struct {
	Offset int
	Path   []int
}

// Niche finds the first never-zero word of t, so that `t?` can be t
// itself with all-zero bits for null (veles-layout-spec.md §5.2): a
// non-raw pointer, a collection, channel or task handle, a closure's code,
// a trait object's table, at a fixed offset through struct fields, tuple
// elements and the first element of an array. C's types (extern structs
// and unions, raw and C function pointers) and anything whose zero is a
// value (numbers, strings, nullables, sealed values) have none.
func (l *Layout) Niche(t Type) (NichePath, bool) {
	switch t := t.(type) {
	case *Pointer:
		return NichePath{}, !t.Raw
	case *List, *Map, *Set, *Channel, *Task:
		return NichePath{}, true
	case *Func:
		return NichePath{Path: []int{0}}, !t.C
	case *Trait:
		return NichePath{Offset: 8, Path: []int{1}}, true
	case *Struct:
		if t.Extern || t.Union || t.Packed {
			return NichePath{}, false
		}
		p := l.StructPlan(t)
		for i, f := range t.Fields {
			if n, ok := l.Niche(f.Type); ok {
				return NichePath{Offset: p.Offsets[i] + n.Offset, Path: append([]int{i}, n.Path...)}, true
			}
		}
	case *Tuple:
		offs := l.Offsets(t.Elems)
		for i, e := range t.Elems {
			if n, ok := l.Niche(e); ok {
				return NichePath{Offset: offs[i] + n.Offset, Path: append([]int{i}, n.Path...)}, true
			}
		}
	case *Array:
		if n, _ := t.N(); n >= 1 {
			if in, ok := l.Niche(t.Elem); ok {
				return NichePath{Offset: in.Offset, Path: append([]int{0}, in.Path...)}, true
			}
		}
	}
	return NichePath{}, false
}

// VariantBoxed reports whether a variant is stored behind a pointer in its
// sealed value (D198): `indirect`, or larger than a word in a recursive
// sealed type unless `inline`.
func (l *Layout) VariantBoxed(v *Struct) bool {
	tmpl := v
	if v.Template != nil {
		tmpl = v.Template
	}
	if v.Sealed == nil || tmpl.Inline {
		return false
	}
	if tmpl.Indirect {
		return true
	}
	if !v.Sealed.IsRecursive() {
		return false
	}
	size, _ := l.Of(v)
	return size > 8
}

// SealedPayloadWords is the room after a sealed value's tag: its largest
// inline variant, a word for a boxed one.
func (l *Layout) SealedPayloadWords(s *Sealed) int {
	words := 0
	for _, v := range s.Variants {
		w := 1
		if !l.VariantBoxed(v) {
			size, _ := l.Of(v)
			w = (size + 7) / 8
		}
		words = max(words, w)
	}
	return words
}

// LLAlign is the alignment LLVM gives the type's representation.
func (l *Layout) LLAlign(t Type) int {
	switch t := t.(type) {
	case *Struct:
		return l.StructPlan(t).LLAlign
	case *Nullable:
		if PtrLike(t.Elem) {
			return 8
		}
		return max(1, l.LLAlign(t.Elem))
	case *Tuple:
		a := 1
		for _, e := range t.Elems {
			a = max(a, l.LLAlign(e))
		}
		return a
	case *Range:
		return max(1, l.LLAlign(t.Elem))
	case *Array:
		return l.LLAlign(t.Elem)
	case *Enum:
		return l.LLAlign(t.Base)
	}
	_, a := l.Of(t)
	return a
}

// Offsets places elems as LLVM places the members of an unpadded struct.
func (l *Layout) Offsets(elems []Type) []int {
	offs := make([]int, len(elems))
	at := 0
	for i, e := range elems {
		s, _ := l.Of(e)
		a := l.LLAlign(e)
		at = roundUp(at, a)
		offs[i] = at
		at += s
	}
	return offs
}

// natural lays elems out as LLVM would (a tuple, a nullable, a range).
func (l *Layout) natural(elems []Type) (size, align int) {
	align = 1
	end := 0
	offs := l.Offsets(elems)
	for i, e := range elems {
		s, _ := l.Of(e)
		align = max(align, l.LLAlign(e))
		end = offs[i] + s
	}
	return roundUp(end, align), align
}

// StructPlan is a struct's layout: each field's offset and alignment, the
// size, the alignment guaranteed, and the alignment LLVM gives its type.
type StructPlan struct {
	Offsets []int
	Size    int
	Align   int
	LLAlign int
}

// StructPlan lays out a struct: in declaration order, each field at the
// next multiple of its alignment (its type's, or its `@align(n)`), every
// field at 0 in a union; `@packed` drops the padding; `@align(n)` raises
// the struct's alignment; the size rounds up to the alignment.
func (l *Layout) StructPlan(s *Struct) StructPlan {
	if p, ok := l.plans[s]; ok {
		return p
	}
	p := l.plan(s)
	if l.plans == nil {
		l.plans = map[*Struct]StructPlan{}
	}
	l.plans[s] = p
	return p
}

func (l *Layout) plan(s *Struct) StructPlan {
	p := StructPlan{Offsets: make([]int, len(s.Fields)), Align: 1, LLAlign: 1}
	end := 0
	for _, i := range l.placementOrder(s) {
		f := s.Fields[i]
		fs, fa := l.Of(f.Type)
		lla := l.LLAlign(f.Type)
		if f.Align > fa {
			fa = f.Align
		}
		if s.Packed {
			fa, lla = 1, 1
			if f.Align > 0 {
				fa = f.Align
			}
		}
		p.Align = max(p.Align, fa)
		p.LLAlign = max(p.LLAlign, lla)
		if s.Union {
			p.Offsets[i] = 0
			end = max(end, fs)
			continue
		}
		at := roundUp(end, fa)
		p.Offsets[i] = at
		end = at + fs
	}
	if s.Align > p.Align {
		p.Align = s.Align
	}
	if s.Union {
		// a union's LLVM type is an integer of its alignment and bytes after
		p.LLAlign = min(p.Align, 16)
	}
	if s.Packed {
		p.LLAlign = 1
	}
	p.Size = roundUp(end, p.Align)
	return p
}

// placementOrder is the order a struct's fields are placed in: as declared,
// or for a `compact struct` by alignment, largest first, as declared among
// equals — which leaves no padding between fields whose sizes are
// multiples of their alignments, as every Veles type's is (D215).
func (l *Layout) placementOrder(s *Struct) []int {
	order := make([]int, len(s.Fields))
	for i := range order {
		order[i] = i
	}
	if !s.Compact || s.Packed || s.Union {
		return order
	}
	align := make([]int, len(s.Fields))
	for i, f := range s.Fields {
		_, align[i] = l.Of(f.Type)
	}
	sort.SliceStable(order, func(a, b int) bool { return align[order[a]] > align[order[b]] })
	return order
}

// PtrLike reports whether a value of t is one machine pointer, so `t?` is
// that pointer or null.
func PtrLike(t Type) bool {
	switch t := t.(type) {
	case *Pointer, *List, *Map, *Set, *Channel, *Task:
		return true
	case *Func:
		return t.C // a nullable C function pointer is NULL or the address, as in C
	}
	return false
}

func roundUp(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) / a * a
}
