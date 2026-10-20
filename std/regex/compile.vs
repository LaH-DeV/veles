use utf8

// The tree to a flat program for the matching machine (vm.vs). A program is a
// list of instructions run by a Pike VM: every thread is a program counter and
// its capture slots, all threads advance over the text one code point at a
// time, so the cost is at most (program size × text length) however the
// pattern is written — there is no backtracking to blow up (D182).

const opChar: i64 = 0
const opSet: i64 = 1
const opAny: i64 = 2
const opSplit: i64 = 3
const opJmp: i64 = 4
const opSave: i64 = 5
const opAssert: i64 = 6
const opMatch: i64 = 7
/// Only in programs for the exhaustive engine (backreferences, lookaround):
const opBackref: i64 = 8
const opLook: i64 = 9
const opLookEnd: i64 = 10
const opAdvance: i64 = 11

/// The most instructions a pattern may compile to; `a{1000}{1000}` is refused
/// rather than built.
const maxProgram: i64 = 100000

/// One instruction. `x` and `y` mean what the op says: `opChar` the code
/// point, `opSet` the index of the set, `opSplit` the preferred and the other
/// target, `opJmp` the target, `opSave` the slot, `opAssert` the kind.
struct Inst {
  op: i64
  x:  i64
  y:  i64 = 0
}

struct Program {
  insts: List<Inst>
  sets:  List<List<i64>>
  /// Capturing groups, not counting the whole match.
  groups: i64
  error:  string
  /// `first[b]` is true when a match can begin with the byte `b`; only worth
  /// using when `skips` (the pattern cannot match the empty string and does not
  /// begin with `.`): a search with no thread alive jumps to the next such byte.
  first: List<bool>
  skips: bool
  /// Needs the exhaustive engine: it holds a backreference or lookaround.
  exact: bool
  /// Capture slots a thread of the exhaustive engine carries: two per group
  /// and one per loop that must see progress.
  slots: i64
  /// `headIndex[pc]` is where the backtracker marks its visited set for the
  /// instruction `pc`, numbering the heads 0, 1, 2… — the start and every target
  /// of a jump or split — and -1 for any other instruction: it has one way in,
  /// so it is reached twice only through a head that was. `headCount` is how
  /// many there are, which is what the visited set is sized by.
  headIndex: List<i64>
  headCount: i64
  /// Four numbers of 32 bits per set: which of the ASCII code points 0..127 are
  /// in it, so the common test is a shift and a mask.
  ascii: List<i64>
  /// `runs[pc]` is 1 where the split at `pc` starts a greedy loop over the one
  /// character instruction after it (`.*`, `[a-z]+`), 2 for a lazy one, else 0.
  runs: List<i64>
}

const fun build(parsed: Parsed): Program {
  val c = Builder(exact: parsed.exact, nextSlot: 2 * (parsed.groups + 1))
  c.emit(opSave, 0)
  c.node(parsed.root)
  c.emit(opSave, 1)
  c.emit(opMatch, 0)
  val insts = c.insts.toList()
  val sets = c.sets.toList()
  if (c.error != "") {
    // refused: nothing runs it, so the tables are not worth building
    return Program(insts, sets, groups: parsed.groups, error: c.error, first: [], skips: false, exact: parsed.exact, slots: c.nextSlot, headIndex: [], headCount: 0, ascii: [], runs: [])
  }
  val first = firstBytes(insts, sets)
  val heads = headIndexOf(insts)
  Program(insts, sets, groups: parsed.groups, error: c.error, first: first.0, skips: first.1, exact: parsed.exact, slots: c.nextSlot, headIndex: heads.0, headCount: heads.1, ascii: asciiMasks(sets), runs: runsOf(insts))
}

/// The bytes a match can begin with, and whether that is a useful filter.
/// Assertions are passed through, so the set may be larger than the truth —
/// never smaller — which is all skipping needs.
const fun firstBytes(insts: List<Inst>, sets: List<List<i64>>): (List<bool>, bool) {
  val first: MutableList<bool> = MutableList<bool>.repeat(false, 256)
  val seen: MutableList<bool> = MutableList<bool>.repeat(false, insts.len())
  val todo: MutableList<i64> = [0]
  var useful = true
  loop (todo.len() > 0) {
    val pc = todo.pop() ?: break
    if (seen.at(pc) ?: true) continue
    seen.set(pc, true)
    val inst = insts.at(pc) ?: break
    when (inst.op) {
      opJmp            => todo.push(inst.x)
      opSplit          => {
        todo.push(inst.x)
        todo.push(inst.y)
      }
      opSave, opAssert => todo.push(pc + 1)
      opChar           => {
        val bytes: MutableList<u8> = []
        utf8.encodeTo(bytes, inst.x)
        first.set((bytes.at(0) ?: 0).toI64(), true)
      }
      opSet            => {
        val ranges = sets.at(inst.x) ?: []
        var i = 0
        loop (i + 1 < ranges.len()) {
          val lo = ranges.at(i) ?: 0
          val hi = ranges.at(i + 1) ?: 0
          var b = lo
          loop (b <= hi.min(127)) {
            first.set(b, true)
            b += 1
          }
          if (hi >= 128) {
            var lead = 128
            loop (lead < 256) {
              first.set(lead, true)
              lead += 1
            }
          }
          i += 2
        }
      }
      else             => {
        // opAny or opMatch: a match may begin with anything, or be empty
        useful = false
      }
    }
  }
  (first.toList(), useful)
}

struct Builder {
  insts:     MutableList<Inst> = []
  sets:      MutableList<List<i64>> = []
  var error: string = ""
  /// The tree needs the exhaustive engine (see Program.exact).
  exact: bool
  /// The next free capture slot, past the groups', for loops that must see progress.
  var nextSlot: i64

  /// Appends an instruction and returns where it went.
  const fun emit(op: i64, x: i64, y: i64 = 0): i64 {
    if (this.insts.len() >= maxProgram) {
      if (this.error == "") this.error = "the pattern compiles to more than $maxProgram instructions"
      return -1
    }
    this.insts.push(Inst(op, x, y))
    this.insts.len() - 1
  }

  const fun patch(at: i64, x: i64, y: i64) {
    if (at < 0) return
    val old = this.insts.at(at) ?: panic("regex: patching an instruction that was emitted")
    this.insts.set(at, Inst(op: old.op, x, y))
  }

  const fun here(): i64 => this.insts.len()

  const fun set(ranges: List<i64>): i64 {
    this.sets.push(ranges)
    this.sets.len() - 1
  }

  const fun node(n: Node) {
    if (this.error != "") return
    when (n) {
      is Nothing => { }
      is Lit     => {
        val _ = this.emit(opChar, n.code)
      }
      is Dot     => {
        if (n.dotAll) {
          val _ = this.emit(opAny, 0)
        } else {
          val _ = this.emit(opSet, this.set(negate([10, 10])))
        }
      }
      is Ranges  => {
        if (n.ranges.len() == 2 && n.ranges.at(0) == n.ranges.at(1)) {
          val _ = this.emit(opChar, n.ranges.at(0) ?: 0)
        } else {
          val _ = this.emit(opSet, this.set(n.ranges))
        }
      }
      is Cat     => {
        loop (item in n.items) this.node(item)
      }
      is Choice  => {
        this.choice(n.items)
      }
      is Repeat  => {
        this.repeat(*n.body, n.min, n.max, n.lazy)
      }
      is Group   => {
        val _ = this.emit(opSave, 2 * n.index)
        this.node(*n.body)
        val _ = this.emit(opSave, 2 * n.index + 1)
      }
      is Anchor  => {
        val _ = this.emit(opAssert, n.kind)
      }
      is BackRef => {
        val _ = this.emit(opBackref, n.index, if (n.fold) 1 else 0)
      }
      is Look    => {
        // opLook kind, then the body, then opLookEnd; the look's y is where
        // the program goes on after it
        val kind = (if (n.behind) 2 else 0) + (if (n.negated) 1 else 0)
        val look = this.emit(opLook, kind)
        this.node(*n.body)
        val _ = this.emit(opLookEnd, 0)
        this.patch(look, kind, this.here())
      }
    }
  }

  /// `a|b|c`: each in order, the first that leads to a match wins.
  const fun choice(items: List<Node>) {
    val jumps: MutableList<i64> = []
    loop ((i, item) in items.enumerate()) {
      if (i < items.len() - 1) {
        val split = this.emit(opSplit, 0)
        this.node(item)
        jumps.push(this.emit(opJmp, 0))
        this.patch(split, split + 1, this.here())
      } else {
        this.node(item)
      }
    }
    val end = this.here()
    loop (j in jumps) this.patch(j, end, 0)
  }

  const fun repeat(body: Node, min: i64, max: i64, lazy: bool) {
    var copies = 0
    loop (copies < min && this.error == "") {
      this.node(body)
      copies += 1
    }
    if (max == -1) {
      // body*: split to the body or past the loop
      val split = this.emit(opSplit, 0)
      if (this.exact) {
        // the exhaustive engine does not drop a thread that repeats a state,
        // so a body that can match nothing would loop forever: note where an
        // iteration started and refuse one that moved nowhere
        val slot = this.nextSlot
        this.nextSlot += 1
        val _ = this.emit(opSave, slot)
        this.node(body)
        val _ = this.emit(opAdvance, slot)
      } else {
        this.node(body)
      }
      val _ = this.emit(opJmp, split)
      val end = this.here()
      if (lazy) this.patch(split, end, split + 1) else this.patch(split, split + 1, end)
      return
    }
    // up to max - min more copies, each one optional and only after the last
    val splits: MutableList<i64> = []
    loop (copies < max && this.error == "") {
      splits.push(this.emit(opSplit, 0))
      this.node(body)
      copies += 1
    }
    val end = this.here()
    loop (split in splits) {
      if (lazy) this.patch(split, end, split + 1) else this.patch(split, split + 1, end)
    }
  }
}

/// Where a path through the program can arrive from more than one place.
const fun headIndexOf(insts: List<Inst>): (List<i64>, i64) {
  val heads: MutableList<bool> = MutableList<bool>.repeat(false, insts.len())
  if (insts.len() > 0) heads.set(0, true)
  loop (inst in insts) {
    if (inst.op == opJmp || inst.op == opSplit) {
      if (inst.x >= 0 && inst.x < insts.len()) heads.set(inst.x, true)
    }
    if (inst.op == opSplit) {
      if (inst.y >= 0 && inst.y < insts.len()) heads.set(inst.y, true)
    }
  }
  // number the heads in program order
  val index: MutableList<i64> = MutableList<i64>.repeat(-1, insts.len())
  var count = 0
  loop (pc in 0..<insts.len()) {
    if (heads.at(pc) ?: false) {
      index.set(pc, count)
      count += 1
    }
  }
  (index.toList(), count)
}

/// For each set, four 32-bit words of the ASCII code points it holds.
const fun asciiMasks(sets: List<List<i64>>): List<i64> {
  val out: MutableList<i64> = []
  loop (ranges in sets) {
    val words: MutableList<i64> = MutableList<i64>.repeat(0, 4)
    var i = 0
    loop (i + 1 < ranges.len()) {
      val to = (ranges.at(i + 1) ?: 0).min(127)
      var c = (ranges.at(i) ?: 0).max(0)
      loop (c <= to) {
        val w = c >> 5
        words.set(w, (words.at(w) ?: 0) | (1 << (c & 31)))
        c += 1
      }
      i += 2
    }
    loop (w in words) out.push(w)
  }
  out
}

/// Finds the loops over one character: `split, char, jmp back` as the compiler
/// writes `x*` (and `x+`, after its first copy) for a single-character `x`.
const fun runsOf(insts: List<Inst>): List<i64> {
  val runs: MutableList<i64> = MutableList<i64>.repeat(0, insts.len())
  var pc = 0
  loop (pc + 2 < insts.len()) {
    val split = insts.at(pc) ?: break
    val body = insts.at(pc + 1) ?: break
    val back = insts.at(pc + 2) ?: break
    val single = body.op == opChar || body.op == opSet || body.op == opAny
    if (split.op == opSplit && single && back.op == opJmp && back.x == pc) {
      if (split.x == pc + 1 && split.y == pc + 3) {
        runs.set(pc, 1)
      } else if (split.x == pc + 3 && split.y == pc + 1) {
        runs.set(pc, 2)
      }
    }
    pc += 1
  }
  runs.toList()
}
