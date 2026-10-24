package sema

import "testing"

// A generic function's type parameter is inferred from what a `try` around
// its call is expected to give, as it is without the `try`: a throwing
// function declares its value's type as its result.
func TestTryInfersFromExpectedType(t *testing.T) {
	expectClean(t, `use io

error Missing { name: string }

fun required<V: Parsable>(text: string): V throws Missing => V.parse(text) ?: throw Missing(name: text)

fun read(): i64 throws Missing {
  val n: i64 = try required("5")
  val s: string = try required("x")
  n + s.len() + try required("6")
}

fun main() {
  io.println("${read() catch (e) { -1 }}")
}`)
}
