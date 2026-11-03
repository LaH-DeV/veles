package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// D231: the share pass and codegen skip work where nothing else can hold a
// collection — a lent alias of a field (sema/lentalias.go), a kept
// parameter used once (ownedOnceParams), a local or a pointer made unique
// once where it is bound (OwnedCollections, UniquePointers). Each case below
// is one where the shortcut must not apply: the value must stay a copy of
// its own. Differential by construction: every line states what a copy
// gives.
func TestCopyShortcutsKeepValues(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `
use io

struct Bag {
  var items: List<i64> = []

  fun add(n: i64) {
    this.items.push(n)
  }

  // lent alias of a field: a later write to the field must not reach it
  fun direct(): string {
    val before = this.items
    this.items.push(9)
    "$before ${this.items}"
  }

  // the receiver handed to a method in between
  fun viaMethod(): string {
    val before = this.items
    this.add(8)
    "$before ${this.items}"
  }

  // another pointer of the receiver's type may be the receiver
  fun viaOther(other: *Bag): string {
    val before = this.items
    other.items.push(7)
    "$before ${this.items}"
  }
}

struct Holder {
  items: List<i64>
}

fun keep(xs: List<i64>): Holder => Holder(items: xs)

fun keepTwice(xs: List<i64>): (Holder, Holder) => (Holder(items: xs), Holder(items: xs))

fun keepAll(xs: List<i64>, n: i64): List<Holder> {
  val out: List<Holder> = []
  loop (_ in 0..<n) out.push(Holder(items: xs))
  out
}

fun copyOut(b: *Bag): List<i64> => b.items

fun make(): List<i64> => [1, 2]

fun main() {
  var bag = Bag(items: [1])
  io.println("direct: ${bag.direct()}")
  io.println("method: ${bag.viaMethod()}")
  val p = &bag
  io.println("other: ${bag.viaOther(p)}")

  // a local struct read through a lent alias, then changed
  val local = Bag(items: [5])
  val seen = local.items
  local.items.push(6)
  io.println("local: $seen ${local.items}")

  // a kept parameter: the caller's list stays its own
  val a: List<i64> = [1]
  val h = keep(a)
  a.push(2)
  io.println("keep: ${h.items} $a")
  var pair = keepTwice([3])
  pair.0.items.push(4)
  io.println("twice: ${pair.0.items} ${pair.1.items}")
  var all = keepAll([5], 2)
  all.at(0)?.items.push(6)
  io.println("all: ${all.map(x => x.items)}")

  // a pointer local whose list is copied while the pointer lives
  var box = Bag(items: [1])
  val q = &box.items
  val copied = box.items
  q.push(2)
  io.println("pointer copy: $copied ${box.items}")
  // a pointer local, and a call handed a pointer that reaches the list
  val r = &box.items
  val out = copyOut(&box)
  r.push(3)
  io.println("pointer call: $out ${box.items}")

  // an owned local from a call, copied before it changes
  val made = make()
  val kept = made
  made.push(3)
  io.println("owned: $kept $made")

  // a tuple's part taken, then the tuple read again: the part is a copy
  val both = ([1], 2)
  val first = both.0
  first.push(5)
  io.println("tuple again: $first ${both.0}")
  // a tuple destructured inside a loop from one made outside it
  val outer = ([7], 0)
  loop (round in 0..<2) {
    val (inner, _) = outer
    inner.push(round)
    io.println("tuple loop: $inner ${outer.0}")
  }
  // destructured from a call: the parts are moves, the values still right
  var (fresh, n) = keepTwice([8])
  fresh.items.push(9)
  io.println("tuple fresh: ${fresh.items} ${n.items}")
}
`
	want := "direct: [1] [1, 9]\n" +
		"method: [1, 9] [1, 9, 8]\n" +
		"other: [1, 9, 8] [1, 9, 8, 7]\n" +
		"local: [5] [5, 6]\n" +
		"keep: [1] [1, 2]\n" +
		"twice: [3, 4] [3]\n" +
		"all: [[5, 6], [5]]\n" +
		"pointer copy: [1] [1, 2]\n" +
		"pointer call: [1, 2] [1, 2, 3]\n" +
		"owned: [1, 2] [1, 2, 3]\n" +
		"tuple again: [1, 5] [1]\n" +
		"tuple loop: [7, 0] [7]\n" +
		"tuple loop: [7, 1] [7]\n" +
		"tuple fresh: [8, 9] [8]\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(strings.TrimPrefix(src, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "shortcuts.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}

// clear() on a shared or lent collection gives its place the type's empty
// constant instead of copying the elements only to drop them (2026-11-02);
// one of its own is emptied in place. Every line is what value semantics
// give, whichever way the clear went.
func TestClearSharedTakesEmpty(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Bag {
  var items: List<i64> = []
}

fun wipe(p: *List<i64>) {
  p.clear()
}

fun main() {
  val a = [1, 2]
  val b = a
  a.clear()
  a.push(3)
  io.println("shared: $a $b")

  val c: List<i64> = []
  c.push(1)
  c.clear()
  c.push(2)
  io.println("own: $c")

  val m: Map<string, i64> = ["a": 1]
  val n = m
  m.clear()
  m.set("b", 2)
  io.println("map: $m $n")

  val s: Set<i64> = [1, 2]
  val kept = s
  s.clear()
  s.add(3)
  io.println("set: $s $kept")

  val d = [1]
  val e = d
  wipe(&d)
  d.push(5)
  io.println("pointer: $d $e")

  val bag = Bag(items: [4])
  val saved = bag.items
  bag.items.clear()
  bag.items.push(7)
  io.println("field: ${bag.items} $saved")

  val grid: List<List<i64>> = [[1, 2], [3]]
  val before = grid
  grid.at(0)?.clear()
  grid.at(0)?.push(8)
  io.println("element: $grid $before")

  val xs = [1, 2, 3]
  var walked = 0
  loop (x in xs) {
    walked += 1
    if (x == 1) xs.clear()
  }
  xs.push(9)
  io.println("loop: $walked $xs")

  val twice = [1]
  val other = twice
  twice.clear()
  twice.clear()
  io.println("twice: $twice ${twice.len()} $other")
}
`
	want := "shared: [3] [1, 2]\n" +
		"own: [2]\n" +
		"map: {b: 2} {a: 1}\n" +
		"set: {3} {1, 2}\n" +
		"pointer: [5] [1]\n" +
		"field: [7] [4]\n" +
		"element: [[8], [3]] [[1, 2], [3]]\n" +
		"loop: 3 [9]\n" +
		"twice: [] 0 [1]\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "clear.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}

// A value read through a pointer and handed to a parameter the callee only
// reads, a lent alias of a field of `this`, and a pointer made unique once:
// none may skip the copy when the callee can reach the same object some
// other way — a global, a struct or a closure it is handed, its receiver,
// another field (2026-11-03: each of these changed the callee's own
// parameter, or wrote into a list a callee had copied out). Each line
// prints a length before and after the callee's change, which value
// semantics keep equal.
func TestPointerReachedValuesStayValues(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Box {
  var items: List<i64>
}

// 1. the same box reached through a global
val reach = Mutex<(*Box)?>(value: null)

fun viaGlobal(xs: List<i64>): string {
  val before = xs.len()
  reach.withLock(r => { if (val b = *r) b.items.push(99) })
  "$before ${xs.len()}"
}

fun global(p: *Box): string => viaGlobal(p.items)

// 2. through a struct argument holding a pointer
struct Ctx {
  box: *Box
}

fun viaStruct(xs: List<i64>, c: Ctx): string {
  val before = xs.len()
  c.box.items.push(7)
  "$before ${xs.len()}"
}

fun structArg(p: *Box, c: Ctx): string => viaStruct(p.items, c)

// 3. through a function argument that captured the pointer
fun viaClosure(xs: List<i64>, change: fun()): string {
  val before = xs.len()
  change()
  "$before ${xs.len()}"
}

fun closureArg(p: *Box, change: fun()): string => viaClosure(p.items, change)

// 4. through the receiver of a method
struct Holder {
  box: *Box

  fun look(xs: List<i64>): string {
    val before = xs.len()
    box.items.push(5)
    "$before ${xs.len()}"
  }
}

fun receiver(p: *Box, h: Holder): string => h.look(p.items)

// 5. a field of the receiver's own struct, through a second pointer
struct Pair {
  var items: List<i64>
  var other: (*Pair)?

  fun lookOther(xs: List<i64>): string {
    val before = xs.len()
    if (val o = other) o.items.push(11)
    "$before ${xs.len()}"
  }
}

fun selfAlias(p: *Pair): string => p.lookOther(p.items)

struct Bag {
  var items: List<i64>

  // 6. a lent alias of this.items, then a call that changes the bag
  // through a global
  fun aliasThenCall(): string {
    val items = this.items
    val before = items.len()
    poke()
    "$before ${items.len()}"
  }
}

val reachBag = Mutex<(*Bag)?>(value: null)

fun poke() {
  reachBag.withLock(r => { if (val b = *r) b.items.push(99) })
}

// 7. a pointer local made unique once, then a call that copies the list
// out through a global, then a write through the pointer: the copy must
// not see the write
val saved = Mutex<List<i64>>(value: [])

fun keepCopy() {
  reachBag.withLock(r => {
    if (val b = *r) {
      val copy = b.items
      saved.withLock(s => { *s = copy })
    }
  })
}

fun uniqueThenCall(bag: *Bag): string {
  val items = &bag.items
  items.push(1)
  keepCopy()
  items.push(2)
  val kept = saved.withLock(s => *s)
  "$kept ${bag.items}"
}

fun main() {
  val p = &Box(items: [1, 2])
  reach.withLock(r => { *r = p })
  io.println("global: ${global(p)}")
  io.println("struct: ${structArg(p, Ctx(box: p))}")
  val q = &Box(items: [1, 2])
  io.println("closure: ${closureArg(q, () => q.items.push(3))}")
  io.println("receiver: ${receiver(p, Holder(box: p))}")
  val pair = &Pair(items: [1], other: null)
  pair.other = pair
  io.println("self: ${selfAlias(pair)}")
val bag = &Bag(items: [1, 2])
  reachBag.withLock(r => { *r = bag })
  io.println("alias: ${bag.aliasThenCall()}")
  io.println("unique: ${uniqueThenCall(bag)}")
}
`
	want := "global: 2 2\n" +
		"struct: 3 3\n" +
		"closure: 2 2\n" +
		"receiver: 4 4\n" +
		"self: 1 1\n" +
		"alias: 2 2\n" +
		"unique: [1, 2, 99, 1] [1, 2, 99, 1, 2]\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "reach.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}

// A local whose address is only lent stays in the frame instead of a heap
// cell (lentAddrPass, checklist D231 item 5): each round's fresh list, a
// struct changed through a lent pointer, and next to them a pointer kept
// in a struct and a capture, which stay heap cells and see later writes.
func TestLentAddressKeepsValues(t *testing.T) {
	src := `use io

struct Holder {
    var keep: *List<i64>
}

struct Counter {
    var n: i64

    fun bump() {
        n += 1
    }
}

fun fill(xs: *List<i64>, n: i64) {
    loop (i in 0..<n) {
        xs.push(i)
    }
}

fun keepIt(h: *Holder, xs: *List<i64>) {
    h.keep = xs
}

fun bumpAll(c: *Counter, times: i64) {
    loop (_ in 0..<times) {
        c.bump()
    }
}

fun main() {
    var total = 0
    loop (round in 0..<3) {
        var xs: List<i64> = []
        fill(&xs, round + 2)
        total += xs.len()
        xs.push(99)
    }
    io.println("lent: ${total}")
    val empty: List<i64> = []
    var h = Holder(keep: &empty)
    var ys: List<i64> = [1]
    keepIt(&h, &ys)
    ys.push(2)
    io.println("kept: ${h.keep.len()}")
    var c = Counter(n: 0)
    bumpAll(&c, 5)
    val before = c
    bumpAll(&c, 1)
    io.println("counter: ${before.n} ${c.n}")
    var zs: List<i64> = []
    fill(&zs, 3)
    val count = () => zs.len()
    fill(&zs, 2)
    io.println("captured: ${count()}")
}
`
	want := "lent: 9\nkept: 2\ncounter: 5 6\ncaptured: 5\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "lent.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}

// D232: `&x` boxes as a trait object over x itself — through a local, a
// field reached through `this`, an object kept in a list and read back by
// a downcast — and a copy made on purpose stays apart.
func TestBoxedPointerChangesThePlace(t *testing.T) {
	src := `use io

trait Sink {
  fun put(b: u8)
  fun size(): i64
}

struct Collect {
  var bytes: List<u8> = []

  implement Sink {
    fun put(b: u8) {
      this.bytes.push(b)
    }

    fun size(): i64 => this.bytes.len()
  }
}

struct Writer {
  var out: Collect = Collect()

  fun flush(data: List<u8>) {
    writeAll(&this.out, data)
  }
}

fun writeAll(to: Sink, data: List<u8>) {
  loop (b in data) to.put(b)
}

fun main() {
  var out = Collect()
  writeAll(&out, [1, 2, 3])
  io.println("local: ${out.bytes.len()}")
  var w = Writer()
  w.flush([4, 5])
  io.println("field: ${w.out.bytes.len()}")
  var a = Collect()
  var b = Collect()
  val sinks: List<Sink> = [&a, &b, a.copy()]
  loop (s in sinks) s.put(7)
  io.println("kept: ${a.bytes.len()} ${b.bytes.len()} ${sinks.get(2)?.size()}")
  val first = sinks.get(0) ?: panic("three sinks")
  if (first is Collect) {
    io.println("downcast: ${first.bytes.len()}")
  }
  a.put(8)
  io.println("after: ${first.size()}")
}
`
	want := "local: 3\nfield: 2\nkept: 1 1 1\ndowncast: 1\nafter: 2\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "box.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}

// A call on the way out of a function does not undo a pointer made unique
// once (sema/uniqueptr.go exitCalls) — but a call followed by `continue`,
// or by more statements that write, does: the copy a callee took must not
// see the later writes.
func TestExitCallsKeepCopiesApart(t *testing.T) {
	t.Parallel()
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	src := `use io

struct Log {
  var items: List<i64>

  fun fill(n: i64): i64 {
    val items = &this.items
    loop (i in 0..<n) {
      items.push(i)
      if (i == 1) {
        snapshot(&this)
        if (n > 0) continue
        return -1
      }
      if (i == 3) {
        snapshot(&this)
        return i
      }
    }
    n
  }

  fun grow(stop: bool): i64 throws Stop {
    val items = &this.items
    items.push(1)
    if (stop) throw tooMany(&this)
    items.push(2)
    items.len()
  }
}

error Stop { }

val saved = Mutex<List<List<i64>>>(value: [])

fun snapshot(log: *Log) {
  val copy = log.items
  saved.withLock(s => s.push(copy))
}

fun tooMany(log: *Log): Stop {
  snapshot(log)
  Stop()
}

fun main() {
  var log = Log(items: [])
  val stopped = log.fill(10)
  io.println("fill: ${stopped} ${log.items} ${saved.get()}")
  var other = Log(items: [7])
  val r = other.grow(true) catch (e) { -1 }
  other.items.push(8)
  io.println("grow: ${r} ${other.items} ${saved.get()}")
}
`
	want := "fill: 3 [0, 1, 2, 3] [[0, 1], [0, 1, 2, 3]]\n" +
		"grow: -1 [7, 1, 8] [[0, 1], [0, 1, 2, 3], [7, 1]]\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, release := range profiles() {
		exe := filepath.Join(dir, "exit.exe")
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "build", Output: exe, Release: release}); code != 0 {
			t.Fatalf("build (release=%v) failed with exit %d", release, code)
		}
		out, err := exec.Command(exe).Output()
		if err != nil {
			t.Fatalf("release=%v: %v", release, err)
		}
		if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
			t.Errorf("release=%v: got\n%s\nwant\n%s", release, got, want)
		}
	}
}
