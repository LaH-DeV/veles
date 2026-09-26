use io

fun main() {
  val xs = [1, 2, 3]
  var i = 0
  loop (i < 10) {
    io.println("${xs.at(i) ?: panic("index $i is past the end of the list")}")
    i += 1
  }
}
