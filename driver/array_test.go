package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// D121: an `Array<T, N>` field of an extern struct is C's `T x[N]` — same
// size, same offsets, same stride in an array of the struct — and such a
// struct crosses by value as C passes it, whatever register class its
// elements put it in: integers, floats, a size past the registers.
func TestArraysAgreeWithC(t *testing.T) {
	clang, err := findClang()
	if err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	cSrc := `#include <stdint.h>
#include <stddef.h>
typedef struct { uint16_t family; uint16_t port; uint32_t addr; uint8_t zero[8]; } SockAddr;
typedef struct { int32_t v[3]; int8_t tag; } Mixed;
typedef struct { float x[4]; } Vec4;
typedef struct { int64_t v[10]; } Wide;
typedef struct { uint8_t rows[2][3]; } Grid;
typedef struct { int32_t v[40]; } Huge;

void c_sizes(int64_t *out) {
  out[0] = sizeof(SockAddr); out[1] = offsetof(SockAddr, zero);
  out[2] = sizeof(Mixed); out[3] = offsetof(Mixed, tag);
  out[4] = sizeof(Vec4); out[5] = sizeof(Wide); out[6] = sizeof(Grid);
}
void c_fill_sock(SockAddr *a, int32_t n) {
  for (int32_t i = 0; i < n; i++) {
    a[i].family = (uint16_t)(2 + i); a[i].port = (uint16_t)(80 + i); a[i].addr = (uint32_t)(1000 + i);
    for (int j = 0; j < 8; j++) a[i].zero[j] = (uint8_t)(i * 10 + j);
  }
}
int64_t c_sock_sum(SockAddr a) {
  int64_t s = a.family + a.port + (int64_t)a.addr;
  for (int j = 0; j < 8; j++) s += a.zero[j];
  return s;
}
int32_t c_mixed(Mixed m) { return m.v[0] + m.v[1] * 10 + m.v[2] * 100 + m.tag * 1000; }
Mixed c_make_mixed(int32_t b) { Mixed m = {{b, b + 1, b + 2}, (int8_t)(b + 3)}; return m; }
float c_vec4_sum(Vec4 v) { return v.x[0] + v.x[1] + v.x[2] + v.x[3]; }
Vec4 c_make_vec4(float x) { Vec4 v = {{x, x * 2, x * 3, x * 4}}; return v; }
int64_t c_wide_sum(Wide w) { int64_t s = 0; for (int i = 0; i < 10; i++) s += w.v[i]; return s; }
Wide c_make_wide(int64_t b) { Wide w; for (int i = 0; i < 10; i++) w.v[i] = b + i; return w; }
int64_t c_huge_sum(Huge h) { int64_t s = 0; for (int i = 0; i < 40; i++) s += (int64_t)h.v[i] * (i + 1); return s; }
Huge c_make_huge(int32_t b) { Huge h; for (int i = 0; i < 40; i++) h.v[i] = b + i; return h; }
int32_t c_grid_sum(Grid g) {
  int32_t s = 0;
  for (int i = 0; i < 2; i++) for (int j = 0; j < 3; j++) s += g.rows[i][j] * (i * 3 + j + 1);
  return s;
}
`
	vSrc := `use io

extern struct SockAddr {
  family: u16
  port: u16
  addr: u32
  zero: Array<u8, 8>
}

extern struct Mixed {
  v: Array<i32, 3>
  tag: i8
}

extern struct Vec4 {
  x: Array<f32, 4>
}

extern struct Wide {
  v: Array<i64, 10>
}

extern struct Grid {
  rows: Array<Array<u8, 3>, 2>
}

extern struct Huge {
  var v: Array<i32, 40>
}

extern "C" {
  fun c_sizes(out: *raw i64)
  fun c_fill_sock(a: *raw SockAddr, n: i32)
  fun c_sock_sum(a: SockAddr): i64
  fun c_mixed(m: Mixed): i32
  fun c_make_mixed(b: i32): Mixed
  fun c_vec4_sum(v: Vec4): f32
  fun c_make_vec4(x: f32): Vec4
  fun c_wide_sum(w: Wide): i64
  fun c_make_wide(b: i64): Wide
  fun c_grid_sum(g: Grid): i32
  fun c_huge_sum(h: Huge): i64
  fun c_make_huge(b: i32): Huge
}

fun main() {
  val sizes: MutableList<i64> = MutableList.repeat(0, 7)
  // SAFETY: c_sizes writes seven i64s
  sizes.withRaw(p => unsafe { c_sizes(p) })
  io.println("C ${sizes}")

  val socks: MutableList<SockAddr> = MutableList.make(2, i => SockAddr(family: 0, port: 0, addr: 0, zero: Array.make(0)))
  // SAFETY: two SockAddr of C's layout
  socks.withRaw(p => unsafe { c_fill_sock(p, 2) })
  val s = socks.at(1) ?: return
  io.println("sock ${s.family} ${s.port} ${s.addr} ${s.zero}")
  // SAFETY: by value, both sides agree on the class
  io.println("sum ${unsafe { c_sock_sum(s) }}")
  // SAFETY: element addresses of the lent storage
  val stride = socks.withRaw(p => unsafe { (p + 1).cast<*raw u8>() - p.cast<*raw u8>() })
  io.println("stride ${stride}")

  // SAFETY: integers by value, in and out
  val m = unsafe { c_make_mixed(5) }
  // SAFETY: as above
  val mixed = unsafe { c_mixed(m) }
  io.println("mixed ${m.v} ${m.tag} ${mixed}")
  val own = Mixed(v: [1, 2, 3], tag: -1)
  // SAFETY: as above
  io.println("own ${unsafe { c_mixed(own) }}")

  // SAFETY: floats by value, in and out
  val v = unsafe { c_make_vec4(1.5) }
  // SAFETY: as above
  val total = unsafe { c_vec4_sum(v) }
  io.println("vec ${v.x} ${total}")

  // SAFETY: past the registers: copied, or returned through memory
  val w = unsafe { c_make_wide(100) }
  // SAFETY: as above
  val wsum = unsafe { c_wide_sum(w) }
  io.println("wide ${w.v.at(9)} ${wsum}")

  val g = Grid(rows: [[1, 2, 3], [4, 5, 6]])
  // SAFETY: an array of arrays is C's two-dimensional array
  io.println("grid ${g.rows} ${unsafe { c_grid_sum(g) }}")

  // SAFETY: 160 bytes by value: in memory for both sides, the memory class
  val huge = unsafe { c_make_huge(100) }
  // SAFETY: as above
  val hsum = unsafe { c_huge_sum(huge) }
  var mine = huge
  mine.v.set(0, 1)
  // SAFETY: as above
  val msum = unsafe { c_huge_sum(mine) }
  io.println("huge ${huge.v.at(39)} ${hsum} ${msum}")
}
`
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("arrays.c", cSrc)
	write("main.vs", vSrc)
	write("veles.toml", "[package]\nname = \"arrays\"\n\n[native]\nlibs = [\"arrays.o\"]\n")
	cmd := exec.Command(clang, "-c", "-O1", "arrays.c", "-o", "arrays.o")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clang: %v\n%s", err, out)
	}
	want := "C [16, 8, 16, 12, 16, 80, 6]\n" +
		"sock 3 81 1001 [10, 11, 12, 13, 14, 15, 16, 17]\n" +
		"sum 1193\n" +
		"stride 16\n" +
		"mixed [5, 6, 7] 8 8765\n" +
		"own -679\n" +
		"vec [1.5, 3.0, 4.5, 6.0] 15.0\n" +
		"wide 109 1045\n" +
		"grid [[1, 2, 3], [4, 5, 6]] 91\n" +
		"huge 139 " + hugeSums() + "\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "arrays.exe")
		if code := Run(Options{Path: dir, Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v:\n got:\n%s\nwant:\n%s", release, got, want)
		}
	}
}

// hugeSums are the two sums the C side computes over Huge{v[i] = 100 + i}: all
// forty weighted by 1..40, and again with the first changed to 1.
func hugeSums() string {
	full, changed := 0, 0
	for i := 0; i < 40; i++ {
		full += (100 + i) * (i + 1)
		v := 100 + i
		if i == 0 {
			v = 1
		}
		changed += v * (i + 1)
	}
	return strconv.Itoa(full) + " " + strconv.Itoa(changed)
}
