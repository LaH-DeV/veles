# Generators and lazy sequences — approaches, with examples (2026-10-20)

Asked by the user (decision 7 of `archive/veles-syntax-decisions.md`, I1): "investigate
problem further, give a lot of different examples from other languages and how
could we benefit from each viable approach". Research only; nothing is built or
decided. The same four small programs are written each way so the approaches
can be compared on the same job:

- **walk**: every node of a tree, depth first, lazily, with early exit;
- **tokens**: a lexer that hands tokens to a parser one at a time;
- **pages**: a paged HTTP API read as one sequence (it suspends and can fail);
- **zip**: two lazy sequences read side by side (the test of *external*
  iteration).

## 1. The problem, in Veles terms

What exists today:

- `Iterator` is a trait with `next(): T?`; `loop (x in c)` desugars through
  `Iterable` (D42); the std has lazy adapters (`map`, `filter`, `take`, …) on
  `Iterator`; `AsyncIterator` is the same with a suspending `next`.
- A lazy producer you write is a struct plus a hand-written `next()` — a state
  machine you maintain yourself. A recursive walk needs an explicit stack
  (`std/prelude/collections.vs`, the codec's `stack`).
- The compiler already compiles suspending functions into **heap-allocated
  state machines** (coroutine frames), and lambdas may suspend or throw (their
  effects are inferred). Loops are cancellation points (D145).
- `yield` is a parse error. Other lazy things are eager: `fs.walk` returns a
  `List`, `lines()` a `List<string>`.

What a solution has to answer in Veles:

1. **Laziness and early exit** — `take(10)` or `break` must stop the producer.
2. **Cleanup on early exit** — a `with` inside the producer must close when the
   consumer stops (D43/D100).
3. **Effects** — can the producer suspend (`suspends`) and throw (`throws`),
   with the error surfacing at the consuming `loop`?
4. **External iteration** — can two sequences be advanced alternately (`zip`,
   a merge)?
5. **Recursion** — can a walk call itself without cost per level per element?
6. **Cost** — allocation, indirect calls, compile-time work.
7. **Fit** — one new keyword or none; the effect system; the formatter, LSP
   hints and the self-hosted compiler.

## 2. The approaches

### A. Hand-written iterator state machines (Veles today; Java, Rust, Zig)

```veles
struct Walk : Iterator {                      // today
  type Item = *Node
  private stack: MutableList<*Node>
  fun next(): (*Node)? {
    val n = this.stack.pop() ?: return null
    loop (i in (0..<n.children.len()).reversed()) this.stack.push(n.children.at(i) ?: panic("in range"))
    n
  }
}
```

Zero language cost, fully external (zip works), any effect. The price is the
author's: every tree walk, every parser state, every paged API is rewritten by
hand as an explicit stack or a state enum. Java and Rust live here; Zig has no
other form.

### B. Generator functions with `yield` (Python, JavaScript, C#, Dart, C++23)

```python
def walk(n):                        # Python
    yield n
    for c in n.children:
        yield from walk(c)
```
```javascript
function* walk(n) { yield n; for (const c of n.children) yield* walk(c); }   // JS
```
```csharp
IEnumerable<Node> Walk(Node n) {                                               // C#
    yield return n;
    foreach (var c in n.Children) foreach (var x in Walk(c)) yield return x;   // no yield*
}
```
```cpp
std::generator<Node*> walk(Node* n) {                                          // C++23
    co_yield n;
    for (auto* c : n->children) co_yield std::ranges::elements_of(walk(c));
}
```
```dart
Iterable<Node> walk(Node n) sync* { yield n; for (var c in n.children) yield* walk(c); }   // Dart
```

*In Veles* (stackless, the frame is the iterator; pulled by `next()`):

```veles
fun walk(root: *Node): Iterator<*Node> {
  yield root
  loop (child in root.children) yield all walk(child)    // `yield all` = Python's `yield from`
}
loop (n in walk(tree).filter(n => n.isLeaf()).take(10)) …
```

Benefits: the natural way to write producers; **external iteration for free**
(zip works because `next()` resumes one frame); laziness; the reuse of the
coroutine machinery that exists. Costs: a keyword and an effect rule; the
frame allocation per generator (one heap object, like a suspended task);
`yield all` is needed to make recursion O(1) per element instead of O(depth).
Hardest part for Veles: **cleanup on early exit**. Python runs `finally`
blocks when a generator is closed or collected; JavaScript's `return()` runs
`finally`; C# disposes the enumerator. In Veles the `loop` would have to close
a generator it abandons (`break`, `take`, an error) — the `Closeable` protocol
for iterators, which the adapters (`take`, `filter`) must also forward.

### C. A builder function over a suspending lambda (Kotlin `sequence {}`, `flow {}`; Swift `AsyncStream`)

```kotlin
fun walk(n: Node): Sequence<Node> = sequence {          // Kotlin
    yield(n)
    for (c in n.children) yieldAll(walk(c))
}
```
```swift
let stream = AsyncStream<Int> { continuation in               // Swift
    for i in 1...3 { continuation.yield(i) }
    continuation.finish()
}
```

*In Veles*: `iterator { emit -> … emit(x) … }` where the body is a lambda that
may suspend; no keyword. Same capabilities as B (it is B with the coroutine
inside a lambda); the lambda's frame is the iterator. Benefits: no new
keyword, and the builder is ordinary library code once suspending lambdas can
be paused by the builder. Costs: `emit` is a function call, not a statement
(so `emit` inside nested lambdas is a trap the language must forbid), and
every producer is written inside a lambda body (one more level of nesting in
every generator).

### D. Internal iteration — the producer calls back (Go 1.23 `iter.Seq`, Ruby `each`, D `opApply`, Kotlin `forEach`)

```go
// Go 1.23: a sequence is a function that calls yield for each element
func Walk(n *Node) iter.Seq[*Node] {
    return func(yield func(*Node) bool) { walk(n, yield) }
}
func walk(n *Node, yield func(*Node) bool) bool {
    if !yield(n) { return false }                  // the consumer broke out
    for _, c := range n.Children { if !walk(c, yield) { return false } }
    return true
}
for n := range Walk(root) { if n.IsLeaf() { break } }     // the loop body becomes the callback
```
```ruby
def walk(n, &block)                                  # Ruby
  block.call(n)
  n.children.each { |c| walk(c, &block) }
end
walk(root) { |n| … }
```

*In Veles*: a producer is `fun walk(root: *Node, visit: fun(*Node): bool)` — a
function that calls `visit` for each element and stops when it answers
`false` — and `loop (n in walk(tree))` could rewrite the loop body into that
lambda exactly as Go does (`break` makes the lambda return `false`; `return` and
`continue` need the rewriting Go specified).

Benefits: **no coroutine, no frame, no allocation** — the loop body inlines into
the producer; recursion is the program's own recursion (O(1) per element);
**cleanup is ordinary** (`with` inside the producer closes when the callback
returns `false` and the producer unwinds); the producer may suspend and throw
in the lambda's inferred effects with no new rule; and no new keyword — the
only language change is the loop rewriting. Costs: **no external iteration**
(zip, merge, "the next token on demand" need to pull). Go added `iter.Pull`,
implemented with coroutines, for exactly that. A lexer handing tokens to a
parser is a pull relationship, so D alone does not serve it. D is also what Nim
calls *inline iterators* (the body is expanded into the iterator at compile
time) and what D (the language) calls `opApply` beside its pull ranges.

### E. Stackful coroutines and fibers (Lua, Ruby `Fiber`, Go goroutines, Java Loom)

```lua
local walk; walk = function(n)                          -- Lua
  coroutine.yield(n)
  for _, c in ipairs(n.children) do walk(c) end          -- yield from any depth
end
local gen = coroutine.wrap(function() walk(root) end)
for n in gen do … end
```

`yield` works from nested ordinary calls because each coroutine has its own
stack. Veles frames are stackless (heap state machines), so this needs a
different runtime; not on the table. Listed because it explains why Lua and
Java can write the walk with no `yield all`.

### F. A task and a channel (Go's older idiom, Veles `async` today)

```veles
scope {                                   // a sketch: the channel API is as in D146
  val rows = Channel<Row>(capacity: 8)
  async { loop (r in db.query(sql)) rows.send(r); rows.close() }
  loop (r in rows) handle(r)
}
```

Works now, and it is the right tool when producer and consumer should run
*concurrently*. As a generator it is heavy (a task and a channel per
sequence), pull becomes push-with-a-buffer, and early exit needs the
cancellation story of D156. Not a substitute; a neighbour.

### G. Lazy data (Haskell lists, Scala `LazyList`, Clojure `lazy-seq`, Elixir `Stream`, Java `Stream`)

```elixir
Stream.unfold(0, fn n -> {n, n + 1} end) |> Stream.take(5) |> Enum.to_list()   # Elixir
```

A lazily produced sequence as a value with a library of combinators
(`Stream.unfold`, `Stream.resource` with a cleanup function, `iterate`,
`generate`). *In Veles* this is the existing `Iterator` adapters plus
constructors: `Iterator.unfold(seed, step)`, `Iterator.iterate`,
`Iterator.generate`. Benefits: no language change; covers infinite and
stateful sequences that are one function of the previous state (fib, counters,
paging by cursor). Not general: a recursive walk is not a fold of one state
without a stack.

## 3. The four programs, side by side

| | A hand state | B `yield` | C builder | D callback / `loop` rewrite | G `unfold` |
|---|---|---|---|---|---|
| **walk** (recursive, early exit) | explicit stack, 15 lines | 4 lines, `yield all` | 5 lines | 4 lines, plain recursion | cannot |
| **tokens** (pull, parser reads one) | state enum, as today | natural | natural | cannot pull without a coroutine | needs one state value |
| **pages** (suspends, throws) | hand-written with a cursor | `async` generator: the suspending `next()` | same | callback may suspend; error at the `loop` | `unfold` with a suspending step |
| **zip** (two sequences) | works | works | works | needs `Pull` (a coroutine) | works |

Plus the criteria from §1:

| | A | B | C | D | G |
|---|---|---|---|---|---|
| lazy, early exit | yes | yes | yes | yes | yes |
| cleanup on early exit | author's job | must be designed (close protocol) | same as B | **free** (`with` in the producer) | the step function's |
| recursion O(1)/element | author's | with `yield all` | with `emitAll` | **free** | no |
| allocation | one object | one coroutine frame | one frame + lambda | **none** | one state object |
| new syntax | none | `yield`, `yield all` | none (a builder fn) | loop rewriting only | none |
| compiler work | none | coroutine transform + close protocol + effects | closures that pause | loop-body-to-lambda rewrite, `break`/`return` | none |
| fits the compiler port | no | yes (tree walks in `sema`) | yes | yes for walks, no for token streams | partly |

## 4. What each approach would buy in this repository

- **`fs.walk`** and directory listings: stream instead of building a `List`
  (B, C, D).
- **The lexer's token stream** (`selfhost/lexer`): pull (`B`, `C`, or the
  hand-written `A` it is now); the parser looks one token ahead, so it needs
  external iteration.
- **Tree walks in the checker** (`selfhost`, `sema` port): AST visiting,
  scope chains — recursive, early-exit, often effectful: **D is the best fit**,
  B next.
- **`db.query`, paged HTTP, `lines()` of a large file**: suspending sources
  (`AsyncIterator`): B (an async generator) or D (the callback suspends).
- **`regex` match iteration** and `String.split` laziness: B or D, small gain.

## 5. A way to decide

The approaches are not rivals; they answer different halves of the problem:

- **Push (D)** is cheapest for *traversals*: recursion, cleanup and effects come
  from the closures Veles already has. One language change (the loop rewrite),
  no frames.
- **Pull (B or C)** is needed for *streams read on demand* and for zip: it
  needs the coroutine transform and the close protocol.
- **`unfold`/`iterate` (G)** is a library gain whatever else is chosen.

A staged route: G now (library, no decision beyond names), D next (a loop
feature, no keyword), B/C only when a real program needs to pull a recursive
producer — and then with the close protocol designed once. The question to the
user is which of the two shapes matters more for the compiler port: traversals
(push) or token streams (pull). The lexer is already a hand-written pull
iterator and works; the traversals are where the code is longest.

## 6. Questions this leaves open

1. Is "no external iteration" acceptable for the first step (D), given that a
   hand-written `Iterator` remains for the cases that pull?
2. If B: `yield all` from the first version, or after? (C# has none and lives
   with the O(depth) cost.)
3. If D: how does `return` inside the loop body leave the *enclosing function*
   (Go's rewrite saves the exit kind in a variable and re-dispatches after the
   producer returns)? Veles already generates the same shape for `loop` bodies
   inside `with` blocks.
4. Does a generator frame count as `Sendable`, and may a task adopt one?
5. Names: `yield`/`yield all`, or `emit`; `Iterator.unfold`/`iterate`/`generate`.
