package sema

import "testing"

// D144: an order written as a MemoryOrder case is checked against the
// operation; the integer operations exist only on integer Atomics.
func TestAtomicMemoryOrders(t *testing.T) {
	expectClean(t, prelude+`enum Phase { Idle, Running }
fun main() {
  val n = Atomic(value: 0)
  val _ = n.load(order: MemoryOrder.Acquire) + n.add(1, order: MemoryOrder.Relaxed) + n.sub(1)
  n.store(1, order: MemoryOrder.Release)
  val _ = n.fetchAnd(1) + n.fetchOr(2, order: MemoryOrder.AcqRel) + n.fetchXor(3)
  val _ = n.compareAndSet(1, 2, order: MemoryOrder.AcqRel, failure: MemoryOrder.Acquire)
  val _ = n.compareExchange(1, 2, order: MemoryOrder.Release, failure: MemoryOrder.Relaxed)
  val _ = n.swap(3, order: MemoryOrder.AcqRel) + n.update(x => x + 1, order: MemoryOrder.Relaxed)
  val o = if (n.load() > 0) MemoryOrder.Release else MemoryOrder.SeqCst
  val _ = n.load(order: o) // known when it runs: not checked
  val small = Atomic<u8>(value: 1)
  val _ = small.add(255)
  val p = Atomic(value: Phase.Idle)
  val _ = p.compareAndSet(Phase.Idle, Phase.Running)
  val q = Atomic<(*i64)?>(value: null)
  val _ = q.compareAndSet(null, &5)
}`)
	expectError(t, prelude+`fun main() { val _ = Atomic(value: 0).load(order: MemoryOrder.Release) }`,
		"MemoryOrder.Release is not an order for a load; use Relaxed, Acquire or SeqCst (D144)")
	expectError(t, prelude+`fun main() { val _ = Atomic(value: 0).load(order: MemoryOrder.AcqRel) }`,
		"MemoryOrder.AcqRel is not an order for a load")
	expectError(t, prelude+`fun main() { Atomic(value: 0).store(1, order: MemoryOrder.Acquire) }`,
		"MemoryOrder.Acquire is not an order for a store; use Relaxed, Release or SeqCst (D144)")
	expectError(t, prelude+`fun main() { val _ = Atomic(value: 0).compareAndSet(0, 1, failure: MemoryOrder.Release) }`,
		"MemoryOrder.Release is not an order for the load of a failed compareAndSet; use Relaxed, Acquire or SeqCst (D144)")
	expectError(t, prelude+`fun main() { val _ = Atomic(value: 0).compareExchange(0, 1, order: MemoryOrder.Release, failure: MemoryOrder.Acquire) }`,
		"the failure order MemoryOrder.Acquire is stronger than the order MemoryOrder.Release")
	expectError(t, prelude+`fun main() { val _ = Atomic(value: 0).compareAndSet(0, 1, order: MemoryOrder.Relaxed, failure: MemoryOrder.SeqCst) }`,
		"the failure order MemoryOrder.SeqCst is stronger than the order MemoryOrder.Relaxed")
	expectError(t, prelude+`fun main() { val _ = Atomic(value: "x").add(1) }`,
		"no method 'add' on type 'Atomic<string>'; 'add' is on an Atomic of an integer")
	expectError(t, prelude+`fun main() { val _ = Atomic(value: 1.5).fetchOr(1) }`,
		"'fetchOr' is on an Atomic of an integer")
}
