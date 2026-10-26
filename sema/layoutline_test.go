package sema

import (
	"testing"

	"github.com/LaH-DeV/veles/types"
)

// D215: hover on a struct shows its size, alignment and padding, and what
// `compact` would make it; a compact struct says so instead.
func TestStructLayoutLine(t *testing.T) {
	fields := func(ts ...types.Type) []*types.Field {
		var out []*types.Field
		for i, ft := range ts {
			out = append(out, &types.Field{Name: string(rune('a' + i)), Type: ft, Index: i})
		}
		return out
	}
	u8 := &types.Basic{Kind: types.U8, Name: "u8"}
	plain := &types.Struct{Name: "Sample", Fields: fields(u8, types.TI64, u8)}
	compact := &types.Struct{Name: "Sample", Compact: true, Fields: fields(u8, types.TI64, u8)}
	tight := &types.Struct{Name: "Tight", Fields: fields(types.TI64, u8, u8)}
	cases := []struct {
		st   *types.Struct
		want string
	}{
		{plain, "24 bytes, aligned 8, 14 of padding; 'compact' would make it 16"},
		{compact, "16 bytes, aligned 8, 6 of padding; fields placed by the compiler (compact)"},
		{tight, "16 bytes, aligned 8, 6 of padding"},
	}
	for _, c := range cases {
		if got := layoutLine(c.st); got != c.want {
			t.Errorf("%s: got %q, want %q", c.st.Name, got, c.want)
		}
	}
}
