# Views over text and bytes — what Veles has, what other languages do (2026-10-20)

Asked by the user while settling the string `slice` (decision 4 of
`veles-syntax-decisions.md`): "add recommended slice but we should also take a
look at a view/Span type". Research only; nothing here is built.

## 1. What Veles already has (checked in `runtime/c/veles_rt.c`)

- **A `string` is a view.** It is `{ data, len }`. `substring` returns
  `out->data = s + lo` — no copy, no allocation — and so do `split` and `lines`
  ("strings that share s's bytes, D10: the collector follows an interior
  pointer"). So text already has the thing a `Span`/`string_view` is.
- **A `List<T>` is not.** `List.slice(from, to)` copies the range in one bulk
  `memcpy` into a new buffer (`veles_list_slice`). The same for `windowed`,
  `chunked`, `take`, `drop` in the prelude, which are built on `slice`.
- **An immutable `List` can never change** (D35, D63): no `push`, no `set`, and
  a `MutableList` becomes one only by a move or by `freeze()` (D188), which
  closes every other name. That is the property views need and most languages
  have to enforce with lifetimes or borrow rules.
- **The collector follows interior pointers** (D10), so a header whose `data`
  points into the middle of another list's buffer keeps that buffer alive.

How much is `slice` used? 21 calls in the hand-written code: `fuzz` (HTTP
parsing, 4), the prelude (4), the database wire protocol (3), `blake2b` (3),
`argon2` (2), `dotenv` (2), `binfile` (2), `inflate` (1). Several are followed
by `decodeUtf8()` (a copy into text anyway). So the measured gain today is
small; the case for a view is binary parsers and network buffers that will be
written, and `windowed(n)`, which copies n elements per window.

## 2. What other languages do

| Language | The view | Can it dangle or alias? | Notes |
|---|---|---|---|
| Go | slice `[]T`: pointer, length, capacity | aliases the array; `append` may or may not write through; a small slice keeps the whole array alive | the model people compare with; `s[a:b]` is O(1) |
| Rust | `&[T]`, `&str`, `&mut [T]` | no: the borrow checker proves it | needs lifetimes in types that store one |
| Zig | `[]T`, `[]const u8` | yes (no checker) | same shape as Go without capacity |
| C# | `Span<T>`, `ReadOnlySpan<char>` (a `ref struct`: stack only), `Memory<T>` (heap-storable) | no for `Span` (it cannot escape), yes for `Memory` | two types because of stack-only safety; cannot be a field or cross an `await` |
| Swift | `ArraySlice<T>`, `Substring` | keeps the base alive; **indices keep the parent's numbering** | a classic surprise: `slice[0]` may be out of range |
| Java | `String.substring` (copies since JDK 7u6), `ByteBuffer.slice()` | views for buffers; strings copy | strings used to share and leak memory |
| JavaScript | `typed.subarray()` is a view, `.slice()` copies; V8 "sliced strings" share | shares the buffer | the two verbs name the two behaviours |
| Python | `memoryview(bytes)`; `bytes[a:b]` copies | memoryview pins the buffer | a separate type for the zero-copy case |
| C++ | `std::string_view`, `std::span<T>` | dangles when the owner dies | the cautionary tale |
| D | slices (pointer + length) | GC-owned, aliasing | like Go |

Two lessons that apply here. *Substrings that share memory retain the whole
original* — Java reversed its design over it, Go documents it; the cure is an
explicit copy (`strings.Clone`, `toList()`). *A view type separate from the
owner is only worth it where the owner is mutable or manually managed*; where
the owner is immutable and garbage-collected (Veles strings, and an immutable
`List`), the view and the owner can be the same type.

## 3. Options for Veles

**A. Nothing more.** Strings are views; list slices copy. Fine while list
slices are rare.

**B. Make `List.slice` share when the receiver is an immutable `List`.** No new
type and no new syntax. A `List` header already has `data`, `len`, `cap`; a
slice of an immutable list is a new header whose `data` is `base + from * elem`
and `len = to - from`. Nothing can write through either header, so sharing is
invisible. A `MutableList` receiver still copies (a later `push` or `set` on
the original would show through). `freeze()` makes a builder's result sliceable
for free. `windowed(n)` and `chunked(n)` become O(1) per window. Cost: the
runtime function, the check of the receiver's static mutability in codegen, and
tests; about an hour. Hazard: the retention problem above (a 4-byte slice of a
1 GiB buffer keeps the buffer); the documented cure is `toList()`, which copies
and detaches.

```veles
val data = file.readBytes()              // List<u8>, immutable
val header = data.slice(0, 16)           // O(1) today a 16-byte copy; with B: a header into data
val body = data.slice(16, data.len())    // O(1) with B; today a copy of everything
```

**C. An explicit read-only window type** in the library:
`struct View<T> { base: List<T>; from: i64; len: i64 }` with `at`, `len`,
`slice` (O(1), clamped), `iter`, `toList`, `==`, `startsWith`, `indexOf`,
`decodeUtf8`. Makes the sharing visible in the type and could also cover
`Array`. Costs a second vocabulary of methods (everything `List` has, or a
trait both implement) for no gain over B while lists are immutable.

**D. A mutable window** (Rust `&mut [T]`, Go `xs[a:b]` written through): write
into a range of a list without a copy, for in-place algorithms
(`sort(xs.window(a, b))`, partitioning, a block cipher over a buffer). Needs
bounds that survive growth (the window must refuse when the list grows or
moves), so it is a runtime-checked handle, or a borrow rule the language does
not have. A real design on its own; no evidence in the code yet.

## 4. What I would do

B now (it is a pure optimization with one documented hazard), C and D only when
a program needs them. For text nothing is missing: `slice` (decision 4) is the
clamped, never-null spelling of what `substring` already does without copying.

Open for the user: whether the retention hazard of B is acceptable, and whether
`List.slice` of a `MutableList` should keep copying (it must) or whether an
explicit `xs.toList().slice(…)` spelling is enough to opt into sharing there.
