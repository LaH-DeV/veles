// A global without a declared type whose initializer reads it back has no
// type to start from either: the cycle is found while checking.
val first = second // error: initialization cycle: 'first' is read while its own initializer is computed
val second = first

fun main() {
}
