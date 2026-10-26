package types

import "testing"

// veles-layout-spec.md §5.2: a `T?` is T itself when T holds a word that is
// never zero, found at a fixed offset; everything C writes, and every type
// whose zero is a value, keeps the flag.
func TestNullableNiche(t *testing.T) {
	list := &List{Elem: TI64}
	str := func(name string, fields ...Type) *Struct {
		s := &Struct{Name: name}
		for i, f := range fields {
			s.Fields = append(s.Fields, &Field{Name: string(rune('a' + i)), Type: f, Index: i})
		}
		return s
	}
	ident := str("Ident", TString, list)
	ext := str("Ext", list)
	ext.Extern = true
	packed := str("Packed", TBool, list)
	packed.Packed = true
	arr := func(n int64, elem Type) *Array { return &Array{Elem: elem, Len: &Const{V: n}} }
	cases := []struct {
		name   string
		t      Type
		offset int
		ok     bool
		size   int // of t?
	}{
		{"string field then a list", ident, 16, true, 24},
		{"a pointer to a value", &Pointer{Elem: TI64}, 0, true, 8},
		{"a raw pointer may be null", &Pointer{Elem: TI64, Raw: true}, 0, false, 8},
		{"a closure's code", &Func{Ret: TI64}, 0, true, 16},
		{"a C function pointer may be null", &Func{Ret: TI64, C: true}, 0, false, 8},
		{"a trait object's method table", &Trait{Name: "Shape"}, 8, true, 16},
		{"a tuple's second element", &Tuple{Elems: []Type{TI64, list}}, 8, true, 16},
		{"an array's first element", arr(2, list), 0, true, 16},
		{"an empty array has none", arr(0, list), 0, false, 8},
		{"C lays out an extern struct", ext, 0, false, 16},
		{"a packed struct", packed, 0, false, 10},
		{"a string's zero is a value", TString, 0, false, 24},
		{"a number's zero is a value", TI64, 0, false, 16},
		{"a nullable's zero is null", &Nullable{Elem: ident}, 0, false, 32},
	}
	l := &Layout{}
	for _, c := range cases {
		n, ok := l.Niche(c.t)
		if ok != c.ok || ok && n.Offset != c.offset {
			t.Errorf("%s: Niche = %+v, %v; want offset %d, %v", c.name, n, ok, c.offset, c.ok)
		}
		if size, _ := l.Of(&Nullable{Elem: c.t}); size != c.size {
			t.Errorf("%s: size of the nullable = %d, want %d", c.name, size, c.size)
		}
	}
}
