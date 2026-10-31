# Coming from another language

One page per language you may already know: what carries over to Veles
unchanged, what is spelled differently, and the handful of rules that will
surprise you because your language does the opposite. Each page is short
and links to the chapter that explains a thing in full; read it once
before the chapters, or keep it open beside them.

- [From Go](go.md) — errors are thrown and checked, not returned; nullable
  types instead of zero values and `nil`; tasks that cannot leak.
- [From TypeScript](typescript.md) — sound types, one `null`, integers that
  are integers, and `async` without promises in every signature.
- [From Swift](swift.md) — `?:` where you write `??`, `else return` where
  you write `guard`, and collections shared by reference.
- [From Kotlin](kotlin.md) — the closest syntax: what is the same, and what
  Veles took out (`!!`, inheritance, unchecked exceptions, `suspend`).
- [From Rust](rust.md) — `Result` and traits without lifetimes or the
  borrow checker; a collector, and a different line against data races.

The chapters on values, functions, collections, structs, nullable types,
errors, traits, sealed types, closures and concurrency (2 to 10 and 12)
each show one program with tabs in these five languages, so you can read
a Veles program beside the one you would have written.
