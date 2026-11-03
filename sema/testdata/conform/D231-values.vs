// D231: lists, maps and sets are values. The removed spellings are errors
// with fixes; a change that would be lost on a copy is refused.
use io
use codec

const struct Fixed {
  items: List<i64>
}

struct Row {
  cells: List<i64>
}

fun spellings(): i64 {
  val a: MutableList<i64> = []                 // error: 'MutableList' was removed (D231): a 'List' is a value that can be changed in place
  val b = mut [1, 2]                           // error: 'mut' before a literal was removed (D231)
  val c = MutableSet<i64>()                    // error: 'MutableSet' was removed (D231): a 'Set' can be changed in place; write 'Set()'
  val d = b.toList()                           // error: 'toList()' on a List was removed (D231): 'b' is a List already
  val e = b.toMutable()                        // error: 'toMutable' was removed (D231): a copy of a collection can be changed in place, so 'b' is the copy
  val f = b.refAt(0)                           // error: 'refAt' was renamed 'at' (D231)
  a.len() + c.len() + d.len() + e.len() + (if (f == null) 0 else 1)
}

fun documents(v: codec.Value): bool {
  val second = v.at(1)                         // error: 'at' was renamed 'elementAt' (D231): a document's element read by index is a copy
  second == null
}

fun lost(rows: List<List<i64>>, rs: List<Row>) {
  (rows.get(0) ?: panic("one row")).push(1)    // error: 'push' changes a temporary copy of 'rows.get(0) ?: panic("one row")' that is then discarded
  rows.get(0)?.push(1)                         // error: 'push' changes a copy of the element 'rows.get(0)', so the change would be lost; reach the element itself with 'rows.at(0)'
  io.println("$rows $rs")
}

fun frozen() {
  var k = Fixed(items: [1])
  k.items.push(2)                              // error: would change a collection inside 'Fixed', a const struct, which is immutable all the way down (D212, D231)
  io.println("${k.items}")
}

fun main() {
  io.println("${spellings()}")
  lost([[1]], [Row(cells: [])])
  frozen()
}
