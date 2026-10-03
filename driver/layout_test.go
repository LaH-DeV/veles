package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D120: Veles lays out a packed struct holding a union, a struct with an
// aligned field and a transparent handle exactly as C does — C fills and
// reads them through pointers, arrays of them have C's stride, and the
// handle crosses by value as the integer it wraps. An over-aligned Veles
// struct is aligned on the stack, in a list, and in a global.
func TestCLayoutAgreesWithC(t *testing.T) {
	clang, err := findClang()
	if err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	cSrc := `#include <stdint.h>
#include <stddef.h>
#pragma pack(push, 1)
typedef union { void *ptr; int32_t fd; uint64_t u64; } EpollData;
typedef struct { uint32_t events; EpollData data; } EpollEvent;
#pragma pack(pop)
typedef struct { uint8_t a; _Alignas(16) int32_t b; uint8_t c; } Aligned;

void c_sizes(int64_t *out) {
  out[0] = sizeof(EpollEvent); out[1] = offsetof(EpollEvent, data); out[2] = sizeof(EpollData);
  out[3] = sizeof(Aligned); out[4] = offsetof(Aligned, b); out[5] = offsetof(Aligned, c); out[6] = _Alignof(Aligned);
}
void c_fill_events(EpollEvent *e, int32_t n) {
  for (int32_t i = 0; i < n; i++) { e[i].events = (uint32_t)i + 1; e[i].data.u64 = 100 + (uint64_t)i; }
}
int64_t c_sum_events(const EpollEvent *e, int32_t n) {
  int64_t s = 0;
  for (int32_t i = 0; i < n; i++) s += (int64_t)e[i].events * 1000 + e[i].data.fd;
  return s;
}
void c_fill_aligned(Aligned *a, int32_t n) {
  for (int32_t i = 0; i < n; i++) { a[i].a = (uint8_t)i; a[i].b = 10 * i; a[i].c = (uint8_t)(20 + i); }
}
int32_t c_aligned_by_value(Aligned a) { return a.a + a.b + a.c; }
int32_t c_twice(int32_t fd) { return fd * 2; }
int64_t c_address_mod(const void *p, int64_t m) { return (int64_t)((uintptr_t)p % (uintptr_t)m); }
`
	vSrc := `use io

extern union EpollData {
  ptr: *raw ()
  var fd: i32
  var u64: u64
}

@packed
extern struct EpollEvent {
  var events: u32
  var data: EpollData
}

extern struct Aligned {
  a: u8
  @align(16)
  b: i32
  c: u8
}

@transparent struct Fd { handle: i32 }

@align(64) struct Counter { var hits: i64 }

extern "C" {
  fun c_sizes(out: *raw i64)
  fun c_fill_events(e: *raw EpollEvent, n: i32)
  fun c_sum_events(e: *raw EpollEvent, n: i32): i64
  fun c_fill_aligned(a: *raw Aligned, n: i32)
  fun c_aligned_by_value(a: Aligned): i32
  fun c_twice(fd: Fd): Fd
  fun c_address_mod(p: *raw u8, m: i64): i64
}

val shared = Counter(hits: 1)

fun main() {
  val sizes: MutableList<i64> = MutableList.repeat(0, 7)
  // SAFETY: c_sizes writes seven i64s
  sizes.withRaw(p => unsafe { c_sizes(p) })
  io.println("C ${sizes}")

  val events: MutableList<EpollEvent> = MutableList.make(3, i => EpollEvent(events: 0, data: EpollData(fd: 0)))
  // SAFETY: three events of C's layout
  events.withRaw(p => unsafe { c_fill_events(p, 3) })
  // SAFETY: the u64 field is what C wrote
  val second = unsafe { events.at(1)?.data.u64 ?: 0 }
  io.println("events ${events.at(0)?.events} ${events.at(2)?.events} ${second}")
  var mine = events.at(2) ?: EpollEvent(events: 0, data: EpollData(fd: 0))
  mine.events = 9
  // SAFETY: writing the field C reads
  unsafe { mine.data.fd = 5 }
  events.set(2, mine)
  // SAFETY: as above
  val sum = events.withRaw(p => unsafe { c_sum_events(p, 3) })
  io.println("sum ${sum}")
  // SAFETY: element addresses of the lent storage
  val stride = events.withRaw(p => unsafe { (p + 1).cast<*raw u8>() - p.cast<*raw u8>() })
  io.println("event stride ${stride}")

  val aligned: MutableList<Aligned> = MutableList.repeat(Aligned(a: 0, b: 0, c: 0), 2)
  // SAFETY: two Aligned of C's layout
  aligned.withRaw(p => unsafe { c_fill_aligned(p, 2) })
  val one = aligned.at(1) ?: Aligned(a: 0, b: 0, c: 0)
  // SAFETY: by value, both sides agree on the class
  io.println("aligned ${one.a} ${one.b} ${one.c} ${unsafe { c_aligned_by_value(one) }}")
  // SAFETY: as for events
  val astride = aligned.withRaw(p => unsafe { (p + 1).cast<*raw u8>() - p.cast<*raw u8>() })
  io.println("aligned stride ${astride}")

  // SAFETY: an integer in, an integer out
  io.println("fd ${unsafe { c_twice(Fd(handle: 21)) }.handle}")

  var local = Counter(hits: 2)
  local.hits = local.hits + shared.hits
  val counters: MutableList<Counter> = [Counter(hits: 1), local]
  val first = counters.ref(0) ?: return
  val last = counters.ref(1) ?: return
  // SAFETY: only the addresses are read
  val listMod = unsafe { c_address_mod(first.cast<*raw u8>(), 64) + c_address_mod(last.cast<*raw u8>(), 64) }
  // SAFETY: as above
  val localMod = unsafe { c_address_mod((&local).cast<*raw u8>(), 64) }
  io.println("counters ${counters.at(1)?.hits} ${listMod} ${localMod}")
}
`
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("layout.c", cSrc)
	write("main.vs", vSrc)
	write("veles.toml", "[package]\nname = \"layout\"\n\n[native]\nlibs = [\"layout.o\"]\n")
	cmd := exec.Command(clang, "-c", "-O1", "layout.c", "-o", "layout.o")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clang: %v\n%s", err, out)
	}
	want := "C [12, 4, 8, 32, 16, 20, 16]\n" +
		"events 1 3 101\n" +
		"sum 12206\n" +
		"event stride 12\n" +
		"aligned 1 10 21 32\n" +
		"aligned stride 32\n" +
		"fd 42\n" +
		"counters 3 0 0\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "layout.exe")
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
