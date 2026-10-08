// Prelude — what the test vocabulary (D78) is built on. Private: test code
// reaches it through the words the compiler knows, never by name.

/// Calls `body`. `expectPanics(body)` launches this in a task of its own
/// inside a `gather`, where a panic is a value (D52) instead of the end of
/// the test.
fun runTestBody<T>(body: sendable fun(): T): T => body()
