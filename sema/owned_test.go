package sema

import (
	"sort"
	"strings"
	"testing"
)

// D231: OwnedCollections accepts a local list or map that is written and
// that nothing else holds once it is bound, and refuses one that is copied
// while alive, has its address taken (a prelude method's `this`), is
// captured, reassigned, sliced, snapshotted by a loop that changes it, lent
// to a call, or only read.
func TestOwnedCollections(t *testing.T) {
	prog := checkProgram(t, prelude+`
fun owned(n: i64): List<i64> {
  val built: List<i64> = []
  loop (i in 0..<n) built.push(i)
  val seen: Map<i64, bool> = [:]
  seen.set(1, true)
  io.println("${seen.len()}")
  built
}
fun refused(n: i64): i64 {
  val copied: List<i64> = []
  copied.push(1)
  val other = copied
  copied.push(2)
  val viaThis: List<i64> = []
  viaThis.push(n)
  val firstTwo = viaThis.take(2)
  val captured: List<i64> = []
  val add = (x: i64) => captured.push(x)
  add(1)
  var replaced: List<i64> = []
  replaced = [n]
  val sliced: List<i64> = []
  sliced.push(n)
  val view = sliced.slice(0, 1)
  val walked: List<i64> = []
  walked.push(n)
  loop (x in walked) {
    if (x < 3) walked.push(x + 10)
  }
  other.len() + firstTwo.len() + captured.len() + replaced.len() + view.len() + walked.len()
}
fun main() {
  io.println("${owned(3)} ${refused(1)}")
}`)
	got := map[string]string{}
	for _, fn := range prog.Funcs {
		if fn.Name != "v_main.owned" && fn.Name != "v_main.refused" {
			continue
		}
		vars, _ := OwnedCollections(fn)
		var names []string
		for v := range vars {
			names = append(names, v.Name)
		}
		sort.Strings(names)
		got[fn.Name] = strings.Join(names, " ")
	}
	if got["v_main.owned"] != "built seen" {
		t.Errorf("owned: got %q, want \"built seen\"", got["v_main.owned"])
	}
	// copied, viaThis, captured, replaced, sliced, walked are refused; the
	// copies made of them (other, firstTwo, view) are only read, and a local
	// only read is never made unique (that would copy a shared value)
	if got["v_main.refused"] != "" {
		t.Errorf("refused: got %q, want none", got["v_main.refused"])
	}
}
