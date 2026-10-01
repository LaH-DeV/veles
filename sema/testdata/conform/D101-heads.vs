// D101: a head that holds an expression writes `val`; without it, one error
// with the fix, and the head is read as the binding (no error follows it).
fun missingVal(s: string): i64 {
  if (n = s.toInt() && n > 0) { // error: a binding in a condition is written 'val n = …'
    return n
  }
  when (m = s.toInt()) { // error: a binding in a condition is written 'val m = …'
    null => return 0
    else => return 1
  }
}
