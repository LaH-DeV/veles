package sema

import (
	"fmt"
	"strings"
	"testing"
)

// D231 choice 3: the changes that may copy a collection shared since its
// last change, for the editor's hint. A line ending `// copies <what>
// <why>` must be a copy site and every other line not; <why> is the line
// the value was shared at, `loop N` for a loop walking it, or `param`,
// after `as empty` for a clear() and `once` for a change a loop repeats
// but not the share.
func TestCopySites(t *testing.T) {
	src := prelude + `
struct Inventory {
  var items: List<string>
}

fun fill(into: *List<i64>, n: i64) {
  loop (i in 0..<n) into.push(i * i)
}

fun encode(bytes: List<u8>): List<u8> {
  bytes.push(0) // copies bytes param
  bytes.push(1)
  bytes
}

fun look(bytes: List<u8>): i64 => bytes.len()

fun branches(flag: bool): List<i64> {
  val xs: List<i64> = [1]
  if (flag) {
    val kept = xs
    io.println("${kept.len()}")
  } else {
    xs.push(3)
  }
  xs.push(2) // copies xs 22
  if (flag) {
    val early = xs
    return early
  }
  xs.push(4)
  xs
}

fun rounds(): List<i64> {
  val xs: List<i64> = []
  val log: List<List<i64>> = []
  loop (i in 0..<3) {
    xs.push(i) // copies xs 41
    log.push(xs)
  }
  xs
}

fun main() {
  val squares: List<i64> = []
  fill(&squares, 4)
  val before = squares
  squares.push(16) // copies squares 49
  squares.push(25)
  before.push(1) // copies before 49
  loop (x in squares) {
    if (x < 3) squares.push(x + 100) // copies squares once loop 53
  }
  val inv = Inventory(items: ["a"])
  val saved = inv.items
  inv.items.push("b") // copies inv.items 57
  inv.items.push("c")
  val grid: List<List<i64>> = [[1], [2]]
  val firstRow = grid.get(0) ?: return
  grid.at(0)?.push(9) // copies an element of grid 61
  val counts: Map<string, i64> = [:]
  val snapshot = counts
  counts.set("a", 1) // copies counts 64
  val copy = saved
  copy.push("x") // copies copy 66
  val logs = [1]
  val held = logs
  logs.clear() // copies logs as empty 69
  io.println("${held.len()} ${logs.len()}")
  val list2 = copy
  loop (s in copy) {
    list2.push(s) // copies list2 once 72
  }
  val each: List<string> = []
  loop (s in copy) {
    val fresh = each
    fresh.push(s) // copies fresh 78
  }
  io.println("${list2.len()} ${each.len()}")
  io.println("$before $squares ${saved.len()} ${encode([1]).len()} ${look([2])} ${firstRow.len()} ${branches(true).len()} ${rounds().len()} ${snapshot.len()} ${copy.len()}")
  handOver()
  var shelf = Shelf(items: [])
  shelf.add("a")
  io.println("${shelf.all().len()}")
}

struct Shelf {
  var items: List<string>

  fun all(): List<string> => this.items

  fun add(s: string) {
    this.items.push(s) // copies this.items elsewhere Shelf.all
    this.items.push(s)
  }
}

fun stock(into: *List<i64>) {
  into.push(1) // copies into caller handOver
}

fun handOver() {
  val nums: List<i64> = [1]
  val other = nums
  stock(&nums)
  io.println("$other")
}
`
	prog := checkProgram(t, src)
	lines := strings.Split(src, "\n")
	want := map[int]string{}
	for i, l := range lines {
		if _, c, ok := strings.Cut(l, "// copies "); ok {
			want[i+1] = c
		}
	}
	got := map[int]string{}
	for _, s := range CopySites(prog) {
		if !strings.HasSuffix(s.At.File.Path, "main.vs") {
			continue
		}
		line, _ := s.At.File.Position(s.At.Start)
		why := "param"
		switch s.Why {
		case WhyShared:
			l, _ := s.SharedAt.File.Position(s.SharedAt.Start)
			why = fmt.Sprint(l)
		case WhyLoop:
			l, _ := s.SharedAt.File.Position(s.SharedAt.Start)
			why = fmt.Sprintf("loop %d", l)
		case WhyElsewhere:
			why = "elsewhere " + s.By
		case WhyCaller:
			why = "caller " + s.By
		}
		if s.Once {
			why = "once " + why
		}
		if s.Empty {
			why = "as empty " + why
		}
		got[line] = s.Name + " " + why
	}
	for line, w := range want {
		if got[line] != w {
			t.Errorf("line %d (%s): got %q, want %q", line, strings.TrimSpace(lines[line-1]), got[line], w)
		}
	}
	for line, g := range got {
		if _, ok := want[line]; !ok {
			t.Errorf("line %d (%s): unexpected copy site %q", line, strings.TrimSpace(lines[line-1]), g)
		}
	}
}
