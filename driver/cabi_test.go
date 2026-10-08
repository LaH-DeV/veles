package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A struct shape for TestCStructsByValue: its fields in C and in Veles.
type cabiField struct {
	name, c, v string
	float      bool
	nested     *cabiShape // a field that is itself an extern struct
}

type cabiShape struct {
	name   string
	fields []cabiField
}

func cabiInt(name, c, v string) cabiField          { return cabiField{name: name, c: c, v: v} }
func cabiFloat(name, c, v string) cabiField        { return cabiField{name: name, c: c, v: v, float: true} }
func cabiNest(name string, s *cabiShape) cabiField { return cabiField{name: name, nested: s} }

// One shape per class each target's C convention distinguishes (plan A8):
// the integer sizes Windows passes in a register and the ones it does not,
// SysV's INTEGER and SSE eightbytes, mixed and partial ones and its memory
// class, AAPCS64's homogeneous float aggregates and its 16-byte limit, and
// a nested struct.
var cabiS8 = &cabiShape{"S8", []cabiField{cabiInt("a", "int32_t", "i32"), cabiInt("b", "int32_t", "i32")}}

var cabiShapes = []*cabiShape{
	{"S1", []cabiField{cabiInt("a", "uint8_t", "u8")}},
	{"S2", []cabiField{cabiInt("a", "int16_t", "i16")}},
	{"S3", []cabiField{cabiInt("a", "uint8_t", "u8"), cabiInt("b", "uint8_t", "u8"), cabiInt("c", "uint8_t", "u8")}},
	{"S4", []cabiField{cabiInt("a", "int32_t", "i32")}},
	cabiS8,
	{"S12", []cabiField{cabiInt("a", "int32_t", "i32"), cabiInt("b", "int32_t", "i32"), cabiInt("c", "int32_t", "i32")}},
	{"S16", []cabiField{cabiInt("a", "int64_t", "i64"), cabiInt("b", "int64_t", "i64")}},
	{"S24", []cabiField{cabiInt("a", "uint8_t", "u8"), cabiInt("b", "int64_t", "i64"), cabiInt("c", "uint16_t", "u16")}},
	{"F1", []cabiField{cabiFloat("a", "float", "f32")}},
	{"F2", []cabiField{cabiFloat("a", "float", "f32"), cabiFloat("b", "float", "f32")}},
	{"F3", []cabiField{cabiFloat("a", "float", "f32"), cabiFloat("b", "float", "f32"), cabiFloat("c", "float", "f32")}},
	{"D1", []cabiField{cabiFloat("a", "double", "f64")}},
	{"D2", []cabiField{cabiFloat("a", "double", "f64"), cabiFloat("b", "double", "f64")}},
	{"D4", []cabiField{cabiFloat("a", "double", "f64"), cabiFloat("b", "double", "f64"), cabiFloat("c", "double", "f64"), cabiFloat("d", "double", "f64")}},
	{"IF", []cabiField{cabiInt("a", "int32_t", "i32"), cabiFloat("b", "float", "f32")}},
	{"DL", []cabiField{cabiFloat("a", "double", "f64"), cabiInt("b", "int64_t", "i64")}},
	{"NEST", []cabiField{cabiNest("inner", cabiS8), cabiInt("z", "int32_t", "i32")}},
}

// leaves lists a shape's scalar fields as access paths.
func (s *cabiShape) leaves(prefix string) []cabiField {
	var out []cabiField
	for _, fl := range s.fields {
		if fl.nested != nil {
			out = append(out, fl.nested.leaves(prefix+fl.name+".")...)
			continue
		}
		fl.name = prefix + fl.name
		out = append(out, fl)
	}
	return out
}

// TestCStructsByValue passes every shape across the C boundary by value in
// each direction — Veles calling C, C calling Veles by name, and Veles
// calling through an `extern fun` pointer — plus calls whose earlier
// arguments use up the argument registers, so SysV must move the struct to
// the stack. C adds 1 to every field and Veles adds 10, so a field that
// lands in the wrong register or half shows in the output.
func TestCStructsByValue(t *testing.T) {
	clang, err := findClang()
	if err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	var c, v, want strings.Builder
	c.WriteString("#include <stdint.h>\n")
	v.WriteString("use io\n\n")
	for _, s := range cabiShapes {
		// shapes are declared in table order, so S8 precedes NEST
		c.WriteString("typedef struct {")
		for _, fl := range s.fields {
			if fl.nested != nil {
				fmt.Fprintf(&c, " %s %s;", fl.nested.name, fl.name)
			} else {
				fmt.Fprintf(&c, " %s %s;", fl.c, fl.name)
			}
		}
		fmt.Fprintf(&c, " } %s;\n", s.name)
		n := s.name
		bump := func(x string) string {
			var b strings.Builder
			for _, l := range s.leaves("") {
				fmt.Fprintf(&b, " %s.%s += 1;", x, l.name)
			}
			return b.String()
		}
		fmt.Fprintf(&c, "%s c_%s(%s x) {%s return x; }\n", n, n, n, bump("x"))
		fmt.Fprintf(&c, "%s v_%s(%s);\n", n, n, n)
		fmt.Fprintf(&c, "%s back_%s(%s x) { return v_%s(x); }\n", n, n, n, n)
		fmt.Fprintf(&c, "%s via_%s(%s (*f)(%s), %s x) { return f(x); }\n", n, n, n, n, n)

		fmt.Fprintf(&v, "extern struct %s {\n", n)
		for _, fl := range s.fields {
			if fl.nested != nil {
				fmt.Fprintf(&v, "  %s: %s\n", fl.name, fl.nested.name)
			} else {
				fmt.Fprintf(&v, "  %s: %s\n", fl.name, fl.v)
			}
		}
		v.WriteString("}\n\n")
		fmt.Fprintf(&v, "extern \"C\" fun v_%s(x: %s): %s => %s\n\n", n, n, n, cabiBuild(s, "x.", " + 10"))
	}
	// the registers used up before the struct (SysV: five of six integer,
	// seven of eight vector registers left one short of a two-register struct)
	c.WriteString("S16 many(int64_t a, int64_t b, int64_t c, int64_t d, int64_t e, S16 x) { x.a += a + b + c + d + e; return x; }\n")
	c.WriteString("D2 manyd(double a, double b, double c, double d, double e, double f, double g, D2 x) { x.a += a + b + c + d + e + f + g; return x; }\n")

	v.WriteString("extern \"C\" {\n")
	for _, s := range cabiShapes {
		n := s.name
		fmt.Fprintf(&v, "  fun c_%s(x: %s): %s\n  fun back_%s(x: %s): %s\n  fun via_%s(f: extern fun(%s): %s, x: %s): %s\n", n, n, n, n, n, n, n, n, n, n, n)
	}
	v.WriteString("  fun many(a: i64, b: i64, c: i64, d: i64, e: i64, x: S16): S16\n")
	v.WriteString("  fun manyd(a: f64, b: f64, c: f64, d: f64, e: f64, f: f64, g: f64, x: D2): D2\n}\n\n")

	v.WriteString("fun main() {\n")
	for i, s := range cabiShapes {
		n := s.name
		vals := cabiValues(s, i)
		fmt.Fprintf(&v, "  val a%s = %s\n", n, cabiBuildValues(s, vals))
		for _, call := range []string{"c_" + n + "(a" + n + ")", "back_" + n + "(a" + n + ")", "via_" + n + "(&v_" + n + ", a" + n + ")"} {
			fmt.Fprintf(&v, "  // SAFETY: the test's own C functions, pure\n  val r%d%s = unsafe { %s }\n", len(call), "_"+strings.Split(call, "(")[0], call)
			fmt.Fprintf(&v, "  io.println(\"%s %s\")\n", strings.Split(call, "(")[0], cabiShow(s, "r"+fmt.Sprint(len(call))+"_"+strings.Split(call, "(")[0]))
		}
		var c1, c10 []string
		for j, l := range s.leaves("") {
			c1 = append(c1, cabiFmt(vals[j]+1, l.float))
			c10 = append(c10, cabiFmt(vals[j]+10, l.float))
		}
		fmt.Fprintf(&want, "c_%s %s\nback_%s %s\nvia_%s %s\n", n, strings.Join(c1, " "), n, strings.Join(c10, " "), n, strings.Join(c10, " "))
	}
	v.WriteString("  // SAFETY: the test's own C functions, pure\n")
	v.WriteString("  val m = unsafe { many(1, 2, 3, 4, 5, S16(a: 100, b: 200)) }\n")
	v.WriteString("  // SAFETY: as above\n")
	v.WriteString("  val md = unsafe { manyd(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, D2(a: 0.5, b: 0.25)) }\n")
	v.WriteString("  io.println(\"many ${m.a} ${m.b} ${md.a} ${md.b}\")\n}\n")
	want.WriteString("many 115 200 28.5 0.25\n")

	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("shapes.c", c.String())
	write("main.vs", v.String())
	write("veles.toml", "[package]\nname = \"cabi\"\n\n[native]\nlibs = [\"shapes.o\"]\n")
	cmd := exec.Command(clang, "-c", "-O1", "shapes.c", "-o", "shapes.o")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clang: %v\n%s", err, out)
	}
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "cabi.exe")
		if code := Run(Options{Path: dir, Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d; program:\n%s", release, code, v.String())
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want.String() {
			t.Errorf("release=%v:\n got:\n%s\nwant:\n%s", release, got, want.String())
		}
	}
}

// cabiValues gives shape i's fields distinct starting values: integers, or
// binary fractions for floats so every sum prints exactly.
func cabiValues(s *cabiShape, i int) []float64 {
	var out []float64
	for j, l := range s.leaves("") {
		x := float64(i*3 + j + 1)
		if l.float {
			x += 0.5
		}
		out = append(out, x)
	}
	return out
}

func cabiFmt(x float64, float bool) string {
	if float {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", x), "0"), ".")
	}
	return fmt.Sprint(int64(x))
}

// cabiBuild writes a constructor of s whose leaves are `from+path+suffix`.
func cabiBuild(s *cabiShape, from, suffix string) string {
	return cabiCtor(s, "", func(path string, l cabiField) string { return from + path + suffix })
}

func cabiBuildValues(s *cabiShape, vals []float64) string {
	k := 0
	return cabiCtor(s, "", func(path string, l cabiField) string {
		x := cabiFmt(vals[k], l.float)
		if l.float && !strings.Contains(x, ".") {
			x += ".0"
		}
		k++
		return x
	})
}

func cabiCtor(s *cabiShape, prefix string, leaf func(string, cabiField) string) string {
	var parts []string
	for _, fl := range s.fields {
		if fl.nested != nil {
			parts = append(parts, fl.name+": "+cabiCtor(fl.nested, prefix+fl.name+".", leaf))
			continue
		}
		parts = append(parts, fl.name+": "+leaf(prefix+fl.name, fl))
	}
	return s.name + "(" + strings.Join(parts, ", ") + ")"
}

func cabiShow(s *cabiShape, x string) string {
	var parts []string
	for _, l := range s.leaves("") {
		parts = append(parts, "${"+x+"."+l.name+"}")
	}
	return strings.Join(parts, " ")
}
