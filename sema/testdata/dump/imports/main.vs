use io as out
use geo { Rect, area }
use geo.units

error TooBig {
  limit: i64
}

// no declared error set: it is inferred from the body (D45), which takes a
// second round when the body calls a function checked later
fun check(r: Rect): f64 throws {
  val a = area(r)
  if (a > 100.0) throw TooBig(limit: 100)
  a
}

fun main() throws {
  val r = Rect(w: units.cm(3.0), h: 4.0)
  out.println("${try check(r)}")
}
