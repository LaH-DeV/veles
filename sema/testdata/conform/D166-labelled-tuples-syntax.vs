// D166: a tuple labels every element or none, each label once.
fun mixed(): (x: i64, i64) => (1, 2) // error: a tuple labels every element or none: this element has no label (D166)

fun twice(): (i64, i64) => (x: 1, x: 2) // error: label 'x' is used twice in one tuple (D166)

fun main() {
  val p = (1, y: 2) // error: a tuple labels every element or none: this element has no label (D166)
}
