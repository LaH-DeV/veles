package llvm

import (
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/types"
)

func cabiStruct(name string, fields ...types.Type) *types.Struct {
	s := &types.Struct{Name: name, Module: "t", Extern: true}
	for i, f := range fields {
		s.Fields = append(s.Fields, &types.Field{Name: string(rune('a' + i)), Type: f})
	}
	return s
}

// The C convention for a struct by value, on every target (plan A8). The
// expected signatures are what `clang --target=<triple> -O1 -S -emit-llvm`
// writes for the same C (`T f(T x)`, and the two register-exhaustion calls),
// with the struct's own type written T. The driver test runs the host's
// convention against real C; this one covers the targets the host is not.
func TestCStructClassification(t *testing.T) {
	i8, i16, i32, i64 := types.TU8, types.TI16, types.TI32, types.TI64
	f32, f64 := types.TF32, types.TF64
	s8 := cabiStruct("S8", i32, i32)
	shapes := map[string]*types.Struct{
		"S1":   cabiStruct("S1", i8),
		"S2":   cabiStruct("S2", i16),
		"S3":   cabiStruct("S3", i8, i8, i8),
		"S4":   cabiStruct("S4", i32),
		"S8":   s8,
		"S12":  cabiStruct("S12", i32, i32, i32),
		"S16":  cabiStruct("S16", i64, i64),
		"S24":  cabiStruct("S24", i8, i64, types.TU16),
		"F1":   cabiStruct("F1", f32),
		"F2":   cabiStruct("F2", f32, f32),
		"F3":   cabiStruct("F3", f32, f32, f32),
		"D1":   cabiStruct("D1", f64),
		"D2":   cabiStruct("D2", f64, f64),
		"D4":   cabiStruct("D4", f64, f64, f64, f64),
		"IF":   cabiStruct("IF", i32, f32),
		"DL":   cabiStruct("DL", f64, i64),
		"NEST": cabiStruct("NEST", s8, i32),
	}
	// "ret (param)" per shape for `T f(T x)`
	want := map[cABI]map[string]string{
		abiWin64: {
			"S1": "i8 (i8)", "S2": "i16 (i16)", "S3": "void (sret, ptr)", "S4": "i32 (i32)",
			"S8": "i64 (i64)", "S12": "void (sret, ptr)", "S16": "void (sret, ptr)", "S24": "void (sret, ptr)",
			"F1": "i32 (i32)", "F2": "i64 (i64)", "F3": "void (sret, ptr)", "D1": "i64 (i64)",
			"D2": "void (sret, ptr)", "D4": "void (sret, ptr)", "IF": "i64 (i64)", "DL": "void (sret, ptr)",
			"NEST": "void (sret, ptr)",
		},
		abiSysV: {
			"S1": "i8 (i8)", "S2": "i16 (i16)", "S3": "i24 (i24)", "S4": "i32 (i32)",
			"S8": "i64 (i64)", "S12": "{ i64, i32 } ({ i64, i32 })", "S16": "{ i64, i64 } ({ i64, i64 })",
			"S24": "void (sret, byval)", "F1": "float (float)", "F2": "<2 x float> (<2 x float>)",
			"F3": "{ <2 x float>, float } ({ <2 x float>, float })", "D1": "double (double)",
			"D2": "{ double, double } ({ double, double })", "D4": "void (sret, byval)", "IF": "i64 (i64)",
			"DL": "{ double, i64 } ({ double, i64 })", "NEST": "{ i64, i32 } ({ i64, i32 })",
		},
		abiAAPCS64: {
			"S1": "i8 (i64)", "S2": "i16 (i64)", "S3": "i24 (i64)", "S4": "i32 (i64)",
			"S8": "i64 (i64)", "S12": "[2 x i64] ([2 x i64])", "S16": "[2 x i64] ([2 x i64])",
			"S24": "void (sret, ptr)", "F1": "{ float } ([1 x float])", "F2": "{ float, float } ([2 x float])",
			"F3": "{ float, float, float } ([3 x float])", "D1": "{ double } ([1 x double])",
			"D2": "{ double, double } ([2 x double])", "D4": "{ double, double, double, double } ([4 x double])",
			"IF": "i64 (i64)", "DL": "[2 x i64] ([2 x i64])", "NEST": "[2 x i64] ([2 x i64])",
		},
	}
	defer func() { currentABI = targetABI }()
	for abi, cases := range want {
		currentABI = func() cABI { return abi }
		g := &gen{typeDecls: map[string]string{}}
		for name, w := range cases {
			s := shapes[name]
			if got := cabiRender(g, g.cSignature([]types.Type{s}, s), s); got != w {
				t.Errorf("abi %d, %s: got %q, want %q", abi, name, got, w)
			}
		}
		// earlier arguments use up the registers: SysV moves the struct to the
		// stack (clang's f_many and f_manyd); the others are unaffected
		s16, d2 := shapes["S16"], shapes["D2"]
		many := g.cSignature([]types.Type{i64, i64, i64, i64, i64, s16}, s16).params[5]
		manyd := g.cSignature([]types.Type{f64, f64, f64, f64, f64, f64, f64, d2}, d2).params[7]
		wantMany := map[cABI][2]cPass{abiWin64: {cIndirect, cIndirect}, abiSysV: {cByval, cByval}, abiAAPCS64: {cCoerce, cCoerce}}[abi]
		if many.pass != wantMany[0] || manyd.pass != wantMany[1] {
			t.Errorf("abi %d: after the registers are used, S16 passes as %d and D2 as %d, want %v", abi, many.pass, manyd.pass, wantMany)
		}
	}
}

func cabiRender(g *gen, sig cSig, s *types.Struct) string {
	llt := g.llType(s)
	ret := sig.retDecl(llt, "")
	var ps []string
	if sig.ret.pass == cIndirect {
		ps = append(ps, "sret")
	}
	for _, p := range sig.params {
		d := p.decl("")
		switch {
		case strings.Contains(d, "byval"):
			d = "byval"
		}
		ps = append(ps, d)
	}
	return ret + " (" + strings.Join(ps, ", ") + ")"
}
