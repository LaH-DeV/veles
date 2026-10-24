// D165: the index-of family answers `i64?` — null when absent, never -1.
// The comparisons that tested the old -1 are errors with a fix; an index the
// family found is in range for the list it searched (D62).
use io

struct Shelf {
  fun indexOf(name: string): i64 => -1   // a method of one's own keeps its answer
}

fun lookups(xs: List<i64>, text: string, shelf: Shelf): i64 {
  if (xs.indexOf(3) >= 0) io.println("found") // error: 'indexOf' answers null when nothing is found, never -1 (D165): write 'xs.indexOf(3) != null', or 'contains' when the index is not needed
  if (text.indexOf("=") < 0) io.println("none") // error: 'indexOf' answers null when nothing is found, never -1 (D165): write 'text.indexOf("=") == null'
  if (text.lastIndexOf("/") == -1) io.println("flat") // error: 'lastIndexOf' answers null when nothing is found, never -1 (D165): write 'text.lastIndexOf("/") == null'
  if (xs.binarySearch(4) != -1) io.println("sorted hit") // error: 'binarySearch' answers null when nothing is found, never -1 (D165): write 'xs.binarySearch(4) != null'
  if (shelf.indexOf("a") >= 0) io.println("on the shelf")
  val at = xs.indexOfFirst(x => x > 2) ?: return 0
  val first: i64 = xs.at(at)               // in range: the search found it
  val loose = xs.indexOf(9) ?: 0
  val maybe: i64 = xs.at(loose) // error: type mismatch: expected 'i64', found 'i64?'
  first + maybe
}

fun main() {
  io.println("${lookups([1, 2, 3], "a=b", Shelf())}")
}
