// Prelude — what the test vocabulary (D78) is built on. Private: test code
// reaches it through the words the compiler knows, never by name.

/// Calls `body`. `expectPanics(body)` launches this in a task of its own
/// inside a `gather`, where a panic is a value (D52) instead of the end of
/// the test.
fun runTestBody<T>(body: sendable fun(): T): T => body()

/// Skips a test, or every test of a suite, without running it (D178). The
/// reason is shown in the report: `test x ... skipped: no ::1 here`.
///
/// ```veles
/// @Skip("waiting on the new parser")
/// test "parses dates" { ... }
/// ```
///
/// For a condition that is known when compiling, `@SkipIf`; for one that is
/// known only when the test runs, `skip(why)` and `skipIf(cond, why)` in the
/// test's body.
public annotation Skip(why: string) on test, suite

/// Skips a test, or every test of a suite, when the condition holds. The
/// condition is a constant (D113): a literal or a `val` of one works, an
/// environment variable does not (use `skipIf(...)` in the body); `build.target`
/// joins them with D160.
///
/// ```veles
/// @SkipIf(isCi, "needs a display")
/// test "ipv6 loopback" { ... }
/// ```
public annotation SkipIf(cond: bool, why: string) on test, suite
