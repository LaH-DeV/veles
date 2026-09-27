package llvm

import (
	"testing"

	"github.com/LaH-DeV/veles/types"
)

// A small integer crossing into C carries the extension clang gives it:
// a C function taking `bool` reads the whole byte, so an i1 passed without
// `zeroext` hands it whatever the register's upper bits held (seen as
// `random.boolean()` printing "true\0false\0in").
func TestCABIExtension(t *testing.T) {
	cases := []struct {
		t    types.Type
		want string
	}{
		{types.TBool, "zeroext "},
		{types.TU8, "zeroext "},
		{types.TU16, "zeroext "},
		{types.TI8, "signext "},
		{types.TI16, "signext "},
		{types.TI32, ""},
		{types.TU32, ""},
		{types.TI64, ""},
		{types.TF64, ""},
	}
	for _, c := range cases {
		if got := cExt(c.t); got != c.want {
			t.Errorf("cExt(%s) = %q, want %q", c.t, got, c.want)
		}
	}
	g := &gen{}
	if got := g.externParamTypes(types.TBool); len(got) != 1 || got[0] != "i1 zeroext" {
		t.Errorf("a bool parameter lowers to %q, want i1 zeroext", got)
	}
}
