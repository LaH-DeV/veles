// D121: an inline array is at most a gigabyte.
struct Table {
  cells: Array<i64, 300000000> // error: would be 2400000000 bytes; an inline array is at most
}

fun main() {
}
