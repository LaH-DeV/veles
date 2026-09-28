// D12/D13/D37/D62: what a `when` arm may match, and what it binds.
use io

sealed trait Shape

struct Circle : Shape {
  r: i64
}

struct Dot : Shape { }

struct Box {
  n: i64
}

error Late { }
error Lost { }

fun combinedBinding(s: Shape): i64 = when (s) {
  is Circle(r), is Dot => r // error: a pattern that binds names cannot be combined with other patterns in one arm
}
fun redundantAlternative(n: i64): i64 = when (n) {
  1, _ => 1 // error: pattern matches everything; the other alternatives in this arm are redundant
}
fun armTypes(n: i64) {
  val _ = when (n) { // error: 'when' arms have incompatible types 'i64' and 'string'
    1 => 1
    else => "many"
  }
}
fun armWithoutValue(n: i64): i64 = when (n) {
  1 => 1
  else => io.println("none") // error: type mismatch: expected 'i64', found '()'
}
fun nullOnPlain(n: i64): i64 = when (n) {
  null => 0 // error: 'null' pattern on a non-nullable subject of type 'i64'
  else => n
}
fun constantOnStruct(b: Box): i64 = when (b) {
  1 => 1 // error: type mismatch
  else => 0
}
fun rangeOnText(s: string): i64 = when (s) {
  in 1..3 => 1 // error: range patterns need an integer subject, found 'string'
  else => 0
}
fun tupleOnNumber(n: i64): i64 = when (n) {
  (a, b) => a // error: tuple pattern on a non-tuple subject of type 'i64'
}
fun tupleArity(p: (i64, i64)): i64 = when (p) {
  (a, b, c) => a // error: tuple has 2 elements but the pattern has 3
}
fun noneWithField(x: i64?): i64 = when (x) {
  None(v) => 0 // error: 'None' has no fields
  else => 1
}
fun someWithTwo(x: i64?): i64 = when (x) {
  Some(a, b) => a // error: 'Some' has exactly one field
  else => 1
}
fun someOnPlain(n: i64): i64 = when (n) {
  Some(v) => v // error: 'Some' pattern on a non-nullable subject of type 'i64'
  else => 0
}
fun fails(n: i64): i64 throws Late | Lost = if (n > 0) throw Late() else throw Lost()
fun notInUnion(): i64 = when (val r = fails(1)) {
  is Ok(v) => v
  is Err(e) => when (e) {
    is Box => 1 // error: 'Box' is not a member of the error union 'Late | Lost'
    else => 0
  }
}
fun fieldsOfDot(s: Shape): i64 = when (s) {
  is Dot(x) => 1 // error: 'Dot' has no field 'x'
  is Circle => 0
}
fun unknownField(s: Shape): i64 = when (s) {
  is Circle(radius: r) => r // error: 'Circle' has no field 'radius'
  is Dot => 0
}
fun tooManyFields(s: Shape): i64 = when (s) {
  is Circle(5, 6) => 1 // error: 'Circle' has only 1 field(s)
  else => 0
}

fun main() {
  io.println("patterns")
}
