// Extension functions (D173): `extend T with fun …` or `extend T { }` on a
// type this package does not own. A file sees them by importing the module.

/// "Hello World" becomes "hello-world".
public extend string with fun slug(): string => this.trim().toLower().replace(" ", "-")

public extend string {
  fun shout(): string => this.toUpper() + "!"
  fun isBlank(): bool => this.trim().isEmpty()
}

public extend<T: Comparable> List<T> with fun isSorted(): bool => this.sorted() == this

public extend i64 with fun clamped(lo: i64, hi: i64): i64 => if (this < lo) lo else if (this > hi) hi else this
