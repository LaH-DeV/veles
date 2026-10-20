use utf8

// The matching machine: a Pike VM over a `Program`. All live threads move over
// the text together, one code point per step, and a thread that reaches an
// instruction another thread reached first is dropped — which is why the time
// is bounded by program size × text length. The order threads are kept in is
// the order the pattern prefers, so the first thread to reach `opMatch` is the
// leftmost-first match (Perl's, RE2's), and everything of lower priority is cut.
//
// No thread owns memory: a list of threads is two flat arrays, the program
// counters and, `width` numbers per thread, the capture positions. Following
// the empty moves of a thread works on one working copy of the captures that a
// `save` sets and the stack puts back (RE2's way), so a search allocates its
// arrays once, not a list per thread per character.

/// Threads alive at one position, in priority order.
struct Threads {
  pcs: MutableList<i64> = []
  /// `width` capture positions per thread, back to back; -1 is unset.
  slots: MutableList<i64> = []

  fun clear() {
    this.pcs.clear()
    this.slots.clear()
  }
}

/// A stack entry that is a request to explore a program counter has this in
/// `b`; any other value is a capture slot's old value to put back into slot `a`.
const explore: i64 = -2

/// The stacks and the working captures `addThread` uses; one per search.
struct Scratch {
  stackA: MutableList<i64> = []
  stackB: MutableList<i64> = []
  cur:    MutableList<i64>
}

/// Everything a search needs, allocated once: a machine runs any number of
/// searches (`findAll` runs one per match) without allocating again, because
/// the marks are compared with a stamp that only grows.
struct Machine {
  prog:      Program
  scratch:   Scratch
  marks:     MutableList<i64>
  var a:     Threads? = null
  var b:     Threads? = null
  var stamp: i64 = 0
  /// Run on threads even when the backtracker would do (tests compare the two).
  var threadsOnly: bool = false

  static fun of(prog: Program): Machine {
    val width = 2 * (prog.groups + 1)
    Machine(
      prog,
      scratch: Scratch(cur: MutableList<i64>.repeat(-1, width)),
      marks: [],
    )
  }

  /// Where a match begins at or after `from`: the capture slots (`start, end`
  /// per group, group 0 the whole match), or null. `anchored` allows a match
  /// only at `from`; `full` only one that ends at the end of the text.
  fun search(text: string, from: i64, anchored: bool, full: bool): List<i64>? {
    val prog = this.prog
    if (prog.exact) return exhaustive(prog, text, from, anchored, full)
    // a small enough job is done by the backtracker, which is several times
    // faster than the threads and just as bounded
    if (!this.threadsOnly && prog.headCount * (text.len() - from + 1) <= maxVisited) {
      return backtrack(prog, text, from, anchored, full, this.scratch)
    }
    // the threads' memory is made when they are first needed
    val marks = this.marks
    if (marks.len() == 0) {
      marks.addAll(MutableList<i64>.repeat(0, prog.insts.len()).toList())
    }
    val a = this.a ?: Threads()
    val b = this.b ?: Threads()
    this.a = a
    this.b = b
    // a search uses one stamp per character it steps over, plus two; the next
    // search starts past all of them
    val first = this.stamp + 1
    this.stamp += text.len() + 3
    return run(prog, this.scratch, marks, a, b, first, text, from, anchored, full)
  }
}

fun run(prog: Program, scratch: Scratch, marks: MutableList<i64>, a: Threads, b: Threads, first: i64, text: string, from: i64, anchored: bool, full: bool): List<i64>? {
  val n = text.len()
  val width = 2 * (prog.groups + 1)
  var clist = a
  var nlist = b
  clist.clear()
  nlist.clear()
  var found: List<i64>? = null
  var stamp = first
  var pos = from
  loop {
    if (found == null && (!anchored || pos == from)) {
      // nothing is alive: skip to a byte a match could start with
      if (clist.pcs.len() == 0 && !anchored && prog.skips) {
        loop (pos < n && !(prog.first.at(text.byteAt(pos).toI64()) ?: true)) pos += 1
        if (pos >= n) break
      }
      loop (k in 0..<width) scratch.cur.set(k, -1)
      addThread(prog, text, clist, marks, stamp, 0, scratch, width, pos)
    }
    if (clist.pcs.len() == 0 && (found != null || anchored)) break
    var c = -1
    var size = 0
    if (pos < n) {
      val b = text.byteAt(pos)
      if (b < 128) {
        c = b.toI64()
        size = 1
      } else {
        val r = utf8.decode(text, pos) ?: panic("regex: text is valid UTF-8")
        c = r.code
        size = r.size
      }
    }
    val next = stamp + 1
    var i = 0
    loop (i < clist.pcs.len()) {
      val pc = clist.pcs.at(i) ?: panic("regex: thread in range")
      val inst = prog.insts.at(pc) ?: panic("regex: pc in range")
      val base = i * width
      i += 1
      var takes = false
      when (inst.op) {
        opChar => takes = c == inst.x
        opSet  => takes = c >= 0 && inSet(prog.sets.at(inst.x) ?: panic("regex: set in range"), c)
        opAny  => takes = c >= 0
        else   => {
          // opMatch: the highest-priority thread to get here wins, unless it
          // had to end at the end of the text and has not
          if (!full || pos == n) {
            val result: MutableList<i64> = []
            loop (k in 0..<width) result.push(clist.slots.at(base + k) ?: -1)
            found = result
            break
          }
        }
      }
      if (takes) {
        loop (k in 0..<width) scratch.cur.set(k, clist.slots.at(base + k) ?: -1)
        addThread(prog, text, nlist, marks, next, pc + 1, scratch, width, pos + size)
      }
    }
    val done = clist
    clist = nlist
    nlist = done
    nlist.clear()
    stamp = next
    if (c < 0) break
    pos += size
  }
  found
}

/// Follows jumps, splits, saves and assertions from `start` without moving,
/// and puts each instruction that needs a character (or ends the match) on
/// `list`, in priority order, with the captures as they stand on that path
/// (`scratch.cur` holds the ones the thread arrived with). A stack of its own,
/// not recursion: a pattern is untrusted input.
fun addThread(prog: Program, text: string, list: Threads, marks: MutableList<i64>, stamp: i64, start: i64, scratch: Scratch, width: i64, pos: i64) {
  val stackA = scratch.stackA
  val stackB = scratch.stackB
  val cur = scratch.cur
  stackA.push(start)
  stackB.push(explore)
  loop (stackA.len() > 0) {
    val a = stackA.pop() ?: break
    val b = stackB.pop() ?: break
    if (b != explore) {
      cur.set(a, b)
      continue
    }
    val pc = a
    if ((marks.at(pc) ?: 0) == stamp) continue
    marks.set(pc, stamp)
    val inst = prog.insts.at(pc) ?: panic("regex: pc in range")
    when (inst.op) {
      opJmp    => {
        stackA.push(inst.x)
        stackB.push(explore)
      }
      opSplit  => {
        // the second choice goes under the first, so the first is tried first
        stackA.push(inst.y)
        stackB.push(explore)
        stackA.push(inst.x)
        stackB.push(explore)
      }
      opSave   => {
        stackA.push(inst.x)
        stackB.push(cur.at(inst.x) ?: -1)
        cur.set(inst.x, pos)
        stackA.push(pc + 1)
        stackB.push(explore)
      }
      opAssert => {
        if (holds(inst.x, text, pos)) {
          stackA.push(pc + 1)
          stackB.push(explore)
        }
      }
      else     => {
        list.pcs.push(pc)
        loop (k in 0..<width) list.slots.push(cur.at(k) ?: -1)
      }
    }
  }
}

const fun isWordByte(b: u8): bool =>
  b.isAsciiAlphanumeric() || b == '_'

/// Does the zero-width test `kind` hold at byte offset `pos`? Word characters
/// are ASCII, so a byte before or after is enough to look at.
fun holds(kind: i64, text: string, pos: i64): bool {
  val n = text.len()
  when (kind) {
    assertBeginText => pos == 0
    assertEndText => pos == n
    assertBeginLine => pos == 0 || text.byteAt(pos - 1) == '\n'
    assertEndLine => pos == n || text.byteAt(pos) == '\n'
    assertWordBoundaryUnicode, assertNotWordBoundaryUnicode => {
      val before = wordBefore(text, pos)
      val after = wordAfter(text, pos)
      if (kind == assertWordBoundaryUnicode) before != after else before == after
    }
    else => {
      val before = pos > 0 && isWordByte(text.byteAt(pos - 1))
      val after = pos < n && isWordByte(text.byteAt(pos))
      if (kind == assertWordBoundary) before != after else before == after
    }
  }
}

/// Is `c` in the sorted, merged `lo, hi, …` ranges? Binary search.
const fun inSet(ranges: List<i64>, c: i64): bool {
  var lo = 0
  var hi = ranges.len() / 2 - 1
  loop (lo <= hi) {
    val mid = (lo + hi) / 2
    val from = ranges.at(2 * mid) ?: 0
    val to = ranges.at(2 * mid + 1) ?: 0
    if (c < from) {
      hi = mid - 1
    } else if (c > to) {
      lo = mid + 1
    } else {
      return true
    }
  }
  false
}

/// The most (instruction, position) pairs the backtracker will track; a bigger
/// job runs on threads, whose memory does not grow with the text.
const maxVisited: i64 = 262144

/// Stack entries of a fused loop (`Program.runs`): `a` is one of these minus the
/// instruction after the loop, and `b` packs positions in 20-bit fields. They sit
/// far below the entries that put a capture back (`-1 - slot`).
const runGreedy: i64 = -1099511627776
const runLazy: i64 = -2199023255552
const runField: i64 = 1048575

/// How many bytes the one-character instruction `inst` takes from `text` at
/// `pos`, or 0 when it does not match there (or the text has ended).
fun oneChar(inst: Inst, ascii: List<i64>, sets: List<List<i64>>, text: string, pos: i64): i64 {
  if (pos >= text.len()) return 0
  var c = text.byteAt(pos).toI64()
  var size = 1
  if (c >= 128) {
    val r = utf8.decode(text, pos) ?: panic("regex: text is valid UTF-8")
    c = r.code
    size = r.size
  }
  if (inst.op == opChar) return if (c == inst.x) size else 0
  if (inst.op == opSet) {
    val takes = if (c < 128) ((ascii.at(4 * inst.x + (c >> 5)) ?: 0) >> (c & 31) & 1) == 1 else inSet(sets.at(inst.x) ?: panic("regex: set in range"), c)
    return if (takes) size else 0
  }
  size
}

/// The start of the code point before byte offset `pos`.
fun previousStart(text: string, pos: i64): i64 {
  var q = pos - 1
  loop (q > 0 && utf8.isContinuation(text.byteAt(q))) q -= 1
  q
}

/// Depth-first search in the order the pattern prefers, which finds the same
/// leftmost-first match the threads do. It is not the exponential backtracking
/// of Perl: a visited set of (instruction, position) pairs means no state is
/// explored twice — a state that failed once fails again whatever the captures
/// — so the work is at most program size × text length, and it needs no
/// copies of the captures: one working array, put back as the search backs up.
///
/// A loop over one character (`.*`, `\w+`, `[A-Z]*?`) is not run instruction by
/// instruction: a greedy one takes the longest run at once and leaves one stack
/// entry that gives the characters back one at a time; a lazy one leaves an entry
/// that takes one more. Same order of attempts, none of the per-character stack
/// traffic.
fun backtrack(prog: Program, text: string, from: i64, anchored: bool, full: bool, scratch: Scratch): List<i64>? {
  val n = text.len()
  val width = 2 * (prog.groups + 1)
  val headCount = prog.headCount
  val visited: MutableList<i64> = MutableList<i64>.repeat(0, ((headCount * (n - from + 1)) >> 5) + 1)
  val insts = prog.insts
  val headIndex = prog.headIndex
  val runs = prog.runs
  val ascii = prog.ascii
  val sets = prog.sets
  val cur = scratch.cur
  val stackA = scratch.stackA
  val stackB = scratch.stackB
  stackA.clear()
  stackB.clear()
  var start = from
  loop (start <= n) {
    if (prog.skips && !anchored) {
      loop (start < n && !(prog.first.at(text.byteAt(start).toI64()) ?: true)) start += 1
      if (start >= n) return null
    }
    loop (k in 0..<width) cur.set(k, -1)
    stackA.push(0)
    stackB.push(start)
    loop (stackA.len() > 0) {
      val a0 = stackA.pop() ?: break
      val b0 = stackB.pop() ?: break
      var pc = a0
      var pos = b0
      if (a0 <= runLazy) {
        pc = runLazy - a0
        val body = insts.at(pc - 2) ?: panic("regex: pc in range")
        val taken = oneChar(body, ascii, sets, text, b0)
        if (taken == 0) continue
        pos = b0 + taken
        stackA.push(a0)
        stackB.push(pos)
      } else if (a0 <= runGreedy) {
        // give back one character of a greedy run, if any is left to give
        pc = runGreedy - a0
        pos = b0 & runField
        val low = b0 >> 20
        if (pos > low) {
          stackA.push(a0)
          stackB.push(previousStart(text, pos) | (low << 20))
        }
      } else if (a0 < 0) {
        // put back capture slot -1 - a0
        cur.set(-1 - a0, b0)
        continue
      }
      // follow one path as far as it goes without touching the stack: only a
      // split leaves an alternative behind
      loop {
        // only a way into the program that two paths can share needs the check
        val head = headIndex.at(pc) ?: 0
        if (head >= 0) {
          val bit = (pos - from) * headCount + head
          val word = bit >> 5
          val mask = 1 << (bit & 31)
          val seen = visited.at(word) ?: 0
          if ((seen & mask) != 0) break
          visited.set(word, seen | mask)
        }
        val inst = insts.at(pc) ?: panic("regex: pc in range")
        when (inst.op) {
          opJmp    => pc = inst.x
          opSplit  => {
            val kind = runs.at(pc) ?: 0
            if (kind == 1) {
              // greedy loop over the one-character instruction at pc + 1
              val body = insts.at(pc + 1) ?: panic("regex: pc in range")
              var end = pos
              loop {
                val taken = oneChar(body, ascii, sets, text, end)
                if (taken == 0) break
                end += taken
              }
              if (end > pos) {
                stackA.push(runGreedy - (pc + 3))
                stackB.push(previousStart(text, end) | (pos << 20))
              }
              pos = end
              pc += 3
            } else if (kind == 2) {
              // lazy loop: go on now, and leave an entry that takes one more
              stackA.push(runLazy - (pc + 3))
              stackB.push(pos)
              pc += 3
            } else {
              stackA.push(inst.y)
              stackB.push(pos)
              pc = inst.x
            }
          }
          opSave   => {
            stackA.push(-1 - inst.x)
            stackB.push(cur.at(inst.x) ?: -1)
            cur.set(inst.x, pos)
            pc += 1
          }
          opAssert => {
            if (!holds(inst.x, text, pos)) break
            pc += 1
          }
          opMatch  => {
            if (!full || pos == n) {
              val result: MutableList<i64> = []
              loop (k in 0..<width) result.push(cur.at(k) ?: -1)
              stackA.clear()
              stackB.clear()
              return result
            }
            break
          }
          else     => {
            val taken = oneChar(inst, ascii, sets, text, pos)
            if (taken == 0) break
            pc += 1
            pos += taken
          }
        }
      }
    }
    if (anchored) return null
    // the next start is the next code point
    start += if (start < n) (utf8.decode(text, start)?.size ?: 1) else 1
  }
  null
}

/// Is the code point just before byte offset `pos` a Unicode word character?
fun wordBefore(text: string, pos: i64): bool {
  val r = utf8.decodeLast(text, pos) ?: return false
  inSet(unicodeWord, r.code)
}

/// Is the code point at byte offset `pos` a Unicode word character?
fun wordAfter(text: string, pos: i64): bool {
  val r = utf8.decode(text, pos) ?: return false
  inSet(unicodeWord, r.code)
}

/// The exhaustive engine, for a pattern with a backreference or lookaround
/// (`Regex.compile(…, backtracking: true)`): depth-first in the order the
/// pattern prefers, with no memory of states already tried — what a
/// backreference needs, because the same instruction at the same position can
/// succeed with one capture and fail with another. That is also why it can take
/// exponential time on a pattern like `(a+)+\1` against a long run of `a`: the
/// risk the linear engines exist to remove, and the reason this one is asked
/// for by name.
fun exhaustive(prog: Program, text: string, from: i64, anchored: bool, full: bool): List<i64>? {
  val n = text.len()
  val cur: MutableList<i64> = MutableList<i64>.repeat(-1, prog.slots)
  var start = from
  loop (start <= n) {
    loop (k in 0..<prog.slots) cur.set(k, -1)
    if (runFrom(prog, text, 0, start, cur, if (full) n else -1)) {
      val width = 2 * (prog.groups + 1)
      val result: MutableList<i64> = []
      loop (k in 0..<width) result.push(cur.at(k) ?: -1)
      return result
    }
    if (anchored) return null
    start += if (start < n) (utf8.decode(text, start)?.size ?: 1) else 1
  }
  null
}

/// Runs the program from instruction `startPc` at byte offset `startPos`;
/// true when it reaches `opMatch` (or `opLookEnd`, for a lookaround's body)
/// at `mustEnd`, or anywhere when `mustEnd` is -1. On success `cur` holds the
/// captures of the path that got there; on failure it is as it was.
fun runFrom(prog: Program, text: string, startPc: i64, startPos: i64, cur: MutableList<i64>, mustEnd: i64): bool {
  val n = text.len()
  val stackA: MutableList<i64> = [startPc]
  val stackB: MutableList<i64> = [startPos]
  loop (stackA.len() > 0) {
    val a = stackA.pop() ?: break
    val b = stackB.pop() ?: break
    if (a < 0) {
      // put back capture slot -1 - a
      cur.set(-1 - a, b)
      continue
    }
    val pos = b
    val inst = prog.insts.at(a) ?: panic("regex: pc in range")
    when (inst.op) {
      opJmp              => {
        stackA.push(inst.x)
        stackB.push(pos)
      }
      opSplit            => {
        stackA.push(inst.y)
        stackB.push(pos)
        stackA.push(inst.x)
        stackB.push(pos)
      }
      opSave             => {
        stackA.push(-1 - inst.x)
        stackB.push(cur.at(inst.x) ?: -1)
        cur.set(inst.x, pos)
        stackA.push(a + 1)
        stackB.push(pos)
      }
      opAdvance          => {
        // a loop iteration that matched nothing goes no further
        if ((cur.at(inst.x) ?: -1) != pos) {
          stackA.push(a + 1)
          stackB.push(pos)
        }
      }
      opAssert           => {
        if (holds(inst.x, text, pos)) {
          stackA.push(a + 1)
          stackB.push(pos)
        }
      }
      opBackref          => {
        val from = cur.at(2 * inst.x) ?: -1
        val to = cur.at(2 * inst.x + 1) ?: -1
        if (from >= 0 && to >= from && pos + (to - from) <= n && sameText(text, from, pos, to - from, inst.y == 1)) {
          stackA.push(a + 1)
          stackB.push(pos + (to - from))
        }
      }
      opLook             => {
        val saved = cur.toList()
        val behind = inst.x >= 2
        val negated = inst.x % 2 == 1
        var held = false
        if (behind) {
          // the body ends here, and starts at some earlier character
          var s = pos
          loop (s >= 0 && !held) {
            if (s == n || !utf8.isContinuation(text.byteAt(s))) {
              held = runFrom(prog, text, a + 1, s, cur, pos)
            }
            s -= 1
          }
        } else {
          held = runFrom(prog, text, a + 1, pos, cur, -1)
        }
        if (negated || !held) {
          // a look that must not match, or did not, leaves no captures behind
          loop (k in 0..<saved.len()) cur.set(k, saved.at(k))
        }
        if (held != negated) {
          if (!negated) {
            // captures the body set are undone if the search backs up past here
            loop (k in 0..<saved.len()) {
              if ((cur.at(k) ?: -1) != saved.at(k)) {
                stackA.push(-1 - k)
                stackB.push(saved.at(k))
              }
            }
          }
          stackA.push(inst.y)
          stackB.push(pos)
        }
      }
      opMatch, opLookEnd => {
        if (mustEnd < 0 || pos == mustEnd) return true
      }
      else               => {
        if (pos < n) {
          var c = text.byteAt(pos).toI64()
          var size = 1
          if (c >= 128) {
            val r = utf8.decode(text, pos) ?: panic("regex: text is valid UTF-8")
            c = r.code
            size = r.size
          }
          val takes = if (inst.op == opChar) c == inst.x else if (inst.op == opSet) inSet(prog.sets.at(inst.x) ?: panic("regex: set in range"), c) else true
          if (takes) {
            stackA.push(a + 1)
            stackB.push(pos + size)
          }
        }
      }
    }
  }
  false
}

/// Do `length` bytes at `a` and at `b` of `text` read the same? Ignoring ASCII
/// case when `fold`.
fun sameText(text: string, a: i64, b: i64, length: i64, fold: bool): bool {
  loop (k in 0..<length) {
    var x = text.byteAt(a + k)
    var y = text.byteAt(b + k)
    if (fold) {
      x = x.toAsciiLower()
      y = y.toAsciiLower()
    }
    if (x != y) return false
  }
  true
}
