# Guard binding — design note (for decision I1) — DECIDED 2026-09-25: let-else + `??` on Result (spec D61)

Asked 2026-09-25: *"We can investigate this with more examples... I think
(a) is not bad, but maybe `??` would be a cleaner syntax for (b) but not
only for result/nullable? (what we need more?)"*

This note answers that: what is already covered, what is not, what `??`
would and would not buy, and what "not only Result/nullable" would take.
Nothing is built until you choose.

---

## 1. What already works — the `T?` case

Kotlin's rule is already Veles's: `?:` takes a right side of type `Never`,
so the nullable case needs no new syntax.

```veles
// fragment
val user = users.get(id) ?: return http.notFound()
val first = xs.first() ?: break
val port = os.env("PORT")?.toInt() ?: 8080
```

So the question is only about the three things `?:` cannot do.

## 2. The three gaps

**Gap A — a `Result`, when the error matters.** Today:

```veles
// fragment
val doc = when (val r = json.parse(text)) {
  is Ok  => r
  is Err => {
    log.warn("bad config: ${r.message()}")
    return defaults
  }
}
```

Five lines whose only job is to bail out; the happy path — `doc` — is
buried in the middle. `try` does not help: this function does not throw,
it falls back.

**Gap B — a `Result`, when a default is enough.** Today
`parse(s).getOrDefault(0)` works (prelude method), which is fine but reads
differently from the `T?` case (`s.toInt() ?: 0`).

**Gap C — a refutable pattern.** Today:

```veles
// fragment
val fields = when (doc) {
  is JObj(fields) => fields
  else => return null
}
```

There is no way to say "this must be a `JObj`, else leave" in one line.

## 3. The options

### (a) `val <pattern> = <expr> else { <block> }` — let-else

Rust's `let ... else`, Swift's `guard let ... else`. The block **must
diverge** (`return`, `break`, `continue`, `throw`, `panic`); the checker
enforces it, with a message that says so.

```veles
// fragment
val doc = json.parse(text) else { e =>          // Gap A: e is the error
  log.warn("bad config: ${e.message()}")
  return defaults
}
val JObj(fields) = doc else { return null }       // Gap C: a pattern
val Some(user) = users.get(id) else { return http.notFound() }   // T?, same rule
val (host, port) = splitHost(addr) else { throw Invalid(addr) }  // Result of a tuple
```

- One rule for `T?`, `Result` and every refutable pattern.
- The `e =>` binding is optional: `else { return defaults }` ignores it.
- Edge: the bindings are in scope *after* the statement, not inside the
  `else` (they do not exist there) — the same as Rust and Swift.
- Edge: `val x = f() else { ... }` where `f()` is plain `i64` (nothing can
  fail) is an error: "nothing to match; drop the 'else'".
- Edge: a `var` form (`var x = ... else`) works the same.
- Cost: parser (`else` after a `val` initializer; today a newline ends the
  statement, so `else` on the next line needs the same continuation rule
  as `if`/`else`), checker (divergence, pattern binding reuse from `when`),
  formatter. A day or two.

### (b) `?:` extended to `Result` — the value form

```veles
// fragment
val n = parse(s) ?: 0                             // Gap B: the error is dropped
val doc = json.parse(text) ?: { e =>              // Gap A, with the error
  log.warn("bad config: ${e.message()}")
  return defaults
}
```

- The left side may be `T?` or `Result<T, E>`; the right side is a `T`, a
  `Never`, or a block `{ e => ... }` that yields a `T` or diverges.
- Unlike (a), the block may *produce a value* (a fallback computed from
  the error), not only leave.
- Does not cover Gap C (patterns).
- Edge: `{ e => ... }` looks like a block containing a `when` arm; it needs
  its own parse rule after `?:`.

### (b′) the same thing spelled `??`

Your question. What `??` means elsewhere matters, because it is what a
reader coming from those languages will assume:

| language | `??` means |
|---|---|
| Swift, C#, JavaScript/TypeScript, Dart, PHP | *null-coalescing*: `a ?? b` is `a` unless it is null, then `b` |
| Kotlin, Groovy | (none — `?:` is the same operator) |
| Veles today | (none — `?:` is the same operator, D30) |

So `??` in every language that has it is exactly Veles's `?:`. Adding it
for Results would give Veles two operators for "the value, or else this",
split by the type on the left — the thing D30 and the R16 decision on `?!`
avoided (`??` was already rejected once as the spelling of `?!`, for
this reason). A reader who knows Swift would read `parse(s) ?? 0` correctly,
but would then also expect `users.get(id) ?? guest` to work and would find
that it is spelled `?:` there.

If `??` is attractive because `?:` reads badly on a `Result`, the better
move is the reverse: **extend `?:` to `Result`** (option b) so there is one
operator for both, and it is the one Kotlin readers already know.

### (c) "Not only Result/nullable" — a trait

What your question points at: user types that have a success side and a
failure side (`Validation<T>`, an HTTP `Response` that is 2xx or not, a
`Parsed<T>` with warnings) taking part in `?:`, `?!`, `try` and let-else.
That is Rust's `Try` trait, and it would look like:

```veles
// fragment
trait Fallible {
  type Value
  type Failure
  /// the value, or the failure — how `?:`, `try` and let-else see this type
  fun split(): Result<Self.Value, Self.Failure>
}

implement Fallible for Validation<T> { ... }

val clean = validate(form) else { f => return badRequest(f.problems) }
val total = try compute(order)          // if compute returns a Fallible
```

- Pro: one mechanism for every "maybe" type, including library ones.
- Con: it makes `try`, `?:`, `?!` and let-else depend on a trait lookup
  rather than on two built-in types; error messages get more abstract
  ("'X' is not Fallible"); `T?` and `Result` would implement it in the
  prelude. Rust took years to settle `Try` and it is still unstable.
- Needs: associated types (exist), a prelude trait, and every one of those
  four forms to go through it. A week, and a real design of its own.
- It composes with (a) and (b): they can be built for `T?`/`Result` first
  and generalised later without a second spelling.

## 4. Side by side on the three gaps

| | Gap A (Result, use error) | Gap B (Result, default) | Gap C (pattern) | new operators |
|---|---|---|---|---|
| (a) let-else | ✓ `else { e => ...; return }` | ✗ (use `getOrDefault`) | ✓ | none (a form of `val`) |
| (b) `?:` on Result | ✓ `?: { e => ... }` | ✓ `?: 0` | ✗ | none (widens `?:`) |
| (b′) `??` | ✓ | ✓ | ✗ | one, meaning `?:` elsewhere |
| (a)+(b) | ✓ | ✓ | ✓ | none |
| (c) trait | extends whichever of the above you pick to user types | | | |

## 5. Recommendation

**(a) + (b), both, and (c) later if a library type asks for it.**

- (a) is the only option that covers patterns, and it is the one you
  already found "not bad".
- (b) makes the Result case read exactly like the nullable one
  (`s.toInt() ?: 0`, `parse(s) ?: 0`), with no new operator.
- Not `??`: every language that has it means Veles's `?:`, so it would be
  a second spelling of an existing operator.
- (c) is a real feature but a separate decision; (a) and (b) do not
  foreclose it.
