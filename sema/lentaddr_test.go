package sema

import "testing"

// A local whose address is only lent — read at once, or passed to a pointer
// parameter or a receiver the callee keeps no pointer to — stays in the
// frame; one whose pointer is kept, captured or bound to a name is a heap
// cell (D10, checklist D231 item 5).
func TestLentAddressStaysInFrame(t *testing.T) {
	prog := checkProgram(t, `use io

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
    var lent: List<i64> = []
    fill(&lent, 2)
    var counter = Counter(n: 0)
    bumpAll(&counter, 5)
    val read = (*&counter).n
    val empty: List<i64> = []
    var holder = Holder(keep: &empty)
    var kept: List<i64> = [1]
    keepIt(&holder, &kept)
    var captured: List<i64> = []
    fill(&captured, 3)
    val count = () => captured.len()
    var named: List<i64> = []
    val p = &named
    fill(p, 1)
    var viaValue: List<i64> = []
    val filler: fun(*List<i64>, i64) = fill
    filler(&viaValue, 2)
    var half = 0.5
    val bits = unsafe {
        *(&half).cast<*raw u64>()
    }
    io.println("${lent.len()} ${read} ${count()} ${named.len()} ${bits}")
}
`)
	want := map[string]bool{"lent": false, "counter": false, "holder": false, "half": false, "viaValue": false, "empty": true, "kept": true, "captured": true, "named": true}
	seen := map[string]bool{}
	for _, fn := range prog.Funcs {
		if fn.Display != "main" || fn.Body == nil {
			continue
		}
		walkBlock(fn.Body, func(n any) {
			d, ok := n.(*VarDecl)
			if !ok {
				return
			}
			heap, ok := want[d.Var.Name]
			if !ok {
				return
			}
			seen[d.Var.Name] = true
			if !d.Var.AddrTaken {
				t.Errorf("%s: not address-taken", d.Var.Name)
			}
			if d.Var.HeapCell() != heap {
				t.Errorf("%s: heap cell %v, want %v", d.Var.Name, d.Var.HeapCell(), heap)
			}
		})
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("%s: declaration not found", name)
		}
	}
}
