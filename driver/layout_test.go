package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A large argument is the caller's own value when nothing can change it
// during the call, and the callee copies it only when it writes it
// (veles-layout-spec.md §5.3). Value semantics hold either way: a callee's
// write, a later argument's write, a method on the receiver passed beside
// it, an address taken, a capture and an indirect call all see what they
// would with a copy per call.
func TestLargeArgumentsKeepValueSemantics(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io { println }

struct Page {
  var bytes: Array<i64, 16>
  var used: i64
  fun bump() {
    this.used += 1
  }
  fun total(): i64 => this.bytes.at(0) + this.used
  fun bumpGet(): i64 {
    this.used += 1
    this.used
  }
}

fun read(p: Page): i64 => p.bytes.at(0) * 100 + p.used

fun twice(a: Page, b: Page): i64 => read(a) + read(b)

fun writesField(p: Page): i64 {
  p.used = 99
  read(p)
}

fun writesArray(p: Page): i64 {
  p.bytes.set(0, 7)
  read(p)
}

fun callsMutating(p: Page): i64 {
  p.bump()
  read(p)
}

fun callsReading(p: Page): i64 => p.total()

fun captures(p: Page): fun(): i64 => () => read(p)

fun depth(p: Page, n: i64): i64 => if (n == 0) read(p) else depth(p, n - 1)

fun bumpAndGive(p: *Page): i64 {
  p.used += 1000
  0
}

fun page(first: i64, used: i64): Page {
  var bytes = Array<i64, 16>.make(0)
  bytes.set(0, first)
  Page(bytes, used)
}

fun main() {
  var x = page(1, 2)
  println("read ${read(x)}")
  println("twice ${twice(x, x)}")
  println("writesField ${writesField(x)} after ${read(x)}")
  println("writesArray ${writesArray(x)} after ${read(x)}")
  println("callsMutating ${callsMutating(x)} after ${read(x)}")
  println("callsReading ${callsReading(x)}")
  val f = captures(x)
  x.used = 50
  println("captured ${f()} now ${read(x)}")
  println("depth ${depth(x, 1000)}")
  var y = page(3, 4)
  println("later arg ${twice(y, page(y.bumpGet(), 0))} after ${read(y)}")
  println("later assign ${twice(y, if (y.used > 0) { y.used = 77; page(0, 0) } else page(0, 0))} after ${read(y)}")
  var z = page(5, 6)
  println("receiver arg ${z.total() + read(z)}")
  val fn = (p: Page) => read(p) + 1
  println("indirect ${fn(z)}")
  var w = page(8, 9)
  val pw = &w
  println("address taken ${twice(w, page(0, bumpAndGive(pw)))} after ${read(w)}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "read 102\ntwice 204\nwritesField 199 after 102\nwritesArray 702 after 102\n" +
		"callsMutating 103 after 102\ncallsReading 3\ncaptured 102 now 150\ndepth 150\n" +
		"later arg 804 after 305\nlater assign 305 after 377\nreceiver arg 517\nindirect 507\n" +
		"address taken 809 after 1809\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "args.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
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

// A nullable of a type holding a word that is never zero is that type, all
// zeros for null (veles-layout-spec.md §5.2): structs, closures, trait
// objects, tuples, arrays, big values in memory, nested nullables, and
// every way a nullable is made and read — narrowing, a write through it,
// `?.`, `?:`, `==`, hashing, printing, collections, channels' kin (pop),
// generics, sealed payloads, constants — give what the flag layout gave.
func TestNullablesLaidOutAsTheirValue(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io { println }

struct Ident {
  name: string
  var at: List<i64>
}

struct Big {
  a: i64
  b: i64
  c: i64
  d: i64
  e: i64
  f: i64
  g: i64
  h: i64
  var xs: List<i64>
}

trait Shape {
  fun area(): i64
}

struct Sq {
  side: i64
  implement Shape {
    fun area(): i64 => this.side * this.side
  }
}

sealed trait Tok {
  struct Word {
    id: Ident?
  }
  struct End { }
}

fun ident(name: string): Ident => Ident(name: name, at: [name.len()])

fun maybe(name: string): Ident? => if (name == "") null else ident(name)

fun big(k: i64): Big => Big(a: k, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, xs: [k, k])

fun maybeBig(k: i64): Big? => if (k < 0) null else big(k)

fun generic<T>(x: T?, d: T): T => x ?: d

fun describe(t: Tok): string {
  when (t) {
    is Tok.Word(id) => return "word ${id?.name ?: "-"}"
    is Tok.End => return "end"
  }
}

const NONE: Ident? = null

fun main() {
  val a = maybe("abc")
  val n = maybe("")
  println("a ${a?.name} ${a?.at} n ${n?.name} ${n == null} ${a == null}")
  println("show $a | $n")
  println("eq ${a == maybe("abc")} ${n == NONE} ${a == n}")
  var m = maybe("xy")
  if (m != null) {
    m.at = [7, 8, 9]
  }
  println("narrowed write ${m?.at}")
  m = null
  println("reset ${m == null}")
  val nested: Ident?? = Some(null)
  val nested2: Ident?? = null
  val nested3: Ident?? = Some(ident("q"))
  println("nested ${nested == null} ${nested2 == null} $nested3")
  when (nested) {
    null => println("outer null")
    Some(null) => println("inner null")
    Some(Some(x)) => println("value ${x.name}")
  }
  val f: (fun(): i64)? = () => 42
  val g: (fun(): i64)? = null
  println("fun ${if (f != null) f() else -1} ${g == null} ${f == null}")
  val s: Shape? = Sq(side: 3)
  val s2: Shape? = null
  println("trait ${s?.area()} ${s2 == null}")
  val tup: (i64, List<i64>)? = (1, [2])
  val tup0: (i64, List<i64>)? = null
  println("tuple $tup $tup0")
  val arr: Array<List<i64>, 2>? = [[1], [2]]
  val arr0: Array<List<i64>, 2>? = null
  println("array ${arr?.at(1)} ${arr0 == null}")
  val bm = maybeBig(5)
  val bn = maybeBig(-1)
  println("big ${bm?.xs} ${bn == null} ${(bm ?: big(0)).a}")
  val bigs: MutableList<Big?> = [big(1), null, big(3)]
  val popped = bigs.pop()
  println("big list ${bigs.len()} ${popped?.a} ${bigs.at(1)} ${bigs.at(0)?.a}")
  val ids: MutableList<Ident?> = [maybe("x"), null]
  println("pop ${ids.pop()} ${ids.pop()?.name} ${ids.pop()}")
  val byName: MutableMap<string, Ident?> = [:]
  byName.set("k", maybe("k"))
  byName.set("z", null)
  println("map ${byName.get("k")} ${byName.get("z")} ${byName.get("nope")}")
  val set: MutableSet<Ident?> = MutableSet<Ident?>()
  set.add(maybe("s"))
  set.add(null)
  set.add(maybe("s"))
  set.add(null)
  println("set ${set.len()} ${set.contains(null)} ${set.contains(maybe("s"))}")
  println("generic ${generic(maybe("g"), ident("d")).name} ${generic(maybe(""), ident("d")).name} ${generic<i64>(null, 4)}")
  println("sealed ${describe(Tok.Word(id: maybe("w")))} ${describe(Tok.Word(id: null))} ${describe(Tok.End())}")
  val dq = Deque<Ident>()
  dq.addLast(ident("one"))
  dq.addFirst(ident("zero"))
  println("deque ${dq.removeFirst()?.name} ${dq.removeLast()?.name} ${dq.removeFirst()}")
  val elvis = n?.at ?: [0]
  println("elvis $elvis ${n?.at?.len() ?: -1}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "a abc [3] n null true false\nshow Ident(name: abc, at: [3]) | null\neq true true false\n" +
		"narrowed write [7, 8, 9]\nreset true\nnested false true Ident(name: q, at: [1])\ninner null\n" +
		"fun 42 true false\ntrait 9 true\ntuple (1, [2]) null\narray [2] true\nbig [5, 5] true 5\n" +
		"big list 2 3 null 1\npop null x null\nmap Ident(name: k, at: [1]) null null\nset 2 true true\n" +
		"generic g d 4\nsealed word w word - end\ndeque zero one null\nelvis [0] -1\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "niche.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		run := exec.Command(exe)
		run.Env = append(os.Environ(), "VELES_GC_THRESHOLD=4096", "VELES_GC_POISON=1")
		out, err := run.CombinedOutput()
		if err != nil {
			t.Fatalf("release=%v: %v\n%s", release, err, out)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v:\n got:\n%s\nwant:\n%s", release, got, want)
		}
	}
}

// A `compact struct` places its fields by alignment (D215), and nothing but
// memory sees it: printing, field reads and writes, `==`, hashing, JSON, a
// constant table, a generic instance per argument order, sealed variants
// (boxed and inline), a nullable — all as with the declared order: the same
// program without the word prints the same.
func TestCompactStructsKeepDeclaredOrder(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io { println }
use json

compact struct Sample {
  flag: u8
  at: i64
  kind: u8
  var n: i32
  ok: bool
  name: string
  implement Codable
}

compact struct Pair<A, B> {
  small: A
  big: B
}

sealed trait Ev {
  compact struct Hit {
    tag: u8
    stamp: i64
    size: u16
  }
  indirect compact struct Big {
    a: u8
    b: i64
    c: u8
    d: List<i64>
  }
  struct Done { }
}

const SAMPLES = [Sample(flag: 1, at: 2, kind: 3, n: 4, ok: true, name: "c"), Sample(flag: 9, at: 8, kind: 7, n: 6, ok: false, name: "d")]

fun bump(s: Sample): Sample {
  var t = s
  t.n += 100
  t
}

fun describe(e: Ev): string {
  when (e) {
    is Ev.Hit(tag, stamp, size) => return "hit $tag $stamp $size"
    is Ev.Big(a, b, c, d) => return "big $a $b $c $d"
    is Ev.Done => return "done"
  }
}

fun main() {
  val s = Sample(flag: 1, at: 1000000000000, kind: 2, n: -5, ok: true, name: "x")
  println("$s")
  println("${s.flag} ${s.at} ${s.kind} ${s.n} ${s.ok} ${s.name}")
  val b = bump(s)
  println("bump ${b.n} same ${s == s} ${s == b} set ${setOf(s, b)}")
  val text = json.encode(s) ?? ""
  println(text)
  val back = try json.decode<Sample>(text) catch (_) { panic("decode") }
  println("back ${back == s}")
  loop (c in SAMPLES) println("const ${c.flag} ${c.n} ${c.name}")
  val p = Pair<u8, i64>(small: 7, big: 8)
  val q = Pair<i64, u8>(small: 9, big: 10)
  println("pairs ${p.small} ${p.big} ${q.small} ${q.big} $p")
  println("${describe(Ev.Hit(tag: 1, stamp: 2, size: 3))} ${describe(Ev.Big(a: 4, b: 5, c: 6, d: [7]))} ${describe(Ev.Done())}")
  val list: MutableList<Sample> = [s, b]
  val sorted = list.sortedBy((x) => -x.n.toI64())
  println("sorted ${sorted.at(0)?.n} ${sorted.at(1)?.n}")
  val maybe: Sample? = s
  println("nullable ${maybe?.name}")
}

fun setOf(a: Sample, b: Sample): i64 {
  val st = MutableSet<Sample>()
  st.add(a)
  st.add(b)
  st.add(a)
  st.len()
}
`
	want := "Sample(flag: 1, at: 1000000000000, kind: 2, n: -5, ok: true, name: x)\n1 1000000000000 2 -5 true x\n" +
		"bump 95 same true false set 2\n{\"flag\":1,\"at\":1000000000000,\"kind\":2,\"n\":-5,\"ok\":true,\"name\":\"x\"}\n" +
		"back true\nconst 1 4 c\nconst 9 6 d\npairs 7 8 9 10 Pair(small: 7, big: 8)\nhit 1 2 3 big 4 5 6 [7] done\n" +
		"sorted 95 -5\nnullable x\n"
	plain := strings.NewReplacer("compact struct", "struct").Replace(src)
	for i, text := range []string{src, plain} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, release := range []bool{false, true} {
			exe := filepath.Join(dir, "compact.exe")
			if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
				t.Fatalf("program %d: build (release=%v) failed with exit %d", i, release, code)
			}
			run := exec.Command(exe)
			run.Env = append(os.Environ(), "VELES_GC_THRESHOLD=4096", "VELES_GC_POISON=1")
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("program %d, release=%v: %v\n%s", i, release, err, out)
			}
			if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
				t.Errorf("program %d, release=%v:\n got:\n%s\nwant:\n%s", i, release, got, want)
			}
		}
	}
}

// A constant table of a struct whose LLVM type carries padding (an
// over-aligned struct, a field of one, an aligned extern field, a packed
// extern struct) is written with that padding, so clang accepts it.
func TestConstantTablesOfPaddedStructs(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io { println }

@align(64) struct Counter {
  hits: i64
}

struct Holder {
  tag: u8
  c:   Counter
}

@packed
extern struct Wire {
  kind: u8
  len:  u32
}

extern struct Spaced {
  a: u8
  @align(16)
  b: i32
}

const COUNTERS = [Counter(hits: 3), Counter(hits: 4)]
const HOLDERS = [Holder(tag: 1, c: Counter(hits: 5))]
const WIRES = [Wire(kind: 1, len: 7)]
const SPACED = [Spaced(a: 2, b: 9)]

fun main() {
  loop (c in COUNTERS) println("${c.hits}")
  loop (h in HOLDERS) println("${h.tag} ${h.c.hits}")
  loop (w in WIRES) println("${w.kind} ${w.len}")
  loop (s in SPACED) println("${s.a} ${s.b}")
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "3\n4\n1 5\n1 7\n2 9\n"
	for _, release := range []bool{false, true} {
		exe := filepath.Join(dir, "consts.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
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

// D120: Veles lays out a packed struct holding a union, a struct with an
// aligned field and a transparent handle exactly as C does — C fills and
// reads them through pointers, arrays of them have C's stride, and the
// handle crosses by value as the integer it wraps. An over-aligned Veles
// struct is aligned on the stack, in a list, and in a global.
func TestCLayoutAgreesWithC(t *testing.T) {
	t.Parallel()
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
