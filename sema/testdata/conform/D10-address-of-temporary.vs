// D10: the address of a temporary boxes the value, which is checked against
// the type the pointer is wanted to point to: a variant stays the variant, a
// literal takes the pointee's type, and a nullable pointer is wanted the
// same way. Without such a type, a construction is the sealed trait.
use io

sealed trait Node
struct Leaf : Node {
  value: i64
}
struct Pair : Node {
  left:  *Node
  right: *Node
}

struct Holder {
  leaf:      *Leaf
  var spare: (*Leaf)? = null
}

fun sum(n: Node): i64 = when (n) {
  is Leaf => n.value
  is Pair => sum(*n.left) + sum(*n.right)
}

fun main() {
  var h = Holder(leaf: &Leaf(value: 1))
  h.spare = &Leaf(value: 2)
  val small: *i32 = &5
  val tree = Pair(left: &Leaf(value: 3), right: &Pair(left: &Leaf(value: 4), right: &Leaf(value: 5)))
  val anyNode = &Leaf(value: 6) // a *Node
  val wrong: *Leaf = &Pair(left: anyNode, right: anyNode) // error: type mismatch: expected '*Leaf', found '*Node'
  io.println("${h.leaf.value} ${h.spare?.value ?: 0} ${*small} ${sum(tree)} ${sum(*anyNode)} $wrong")
}
