use io { println }

fun main() {
  val xs = [1, 2, 3]
  loop (i in 0..<10) {
    println("${xs.at(i) ?: panic("index $i is past the end of the list")}")
  }
}
