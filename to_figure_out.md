1. What about `const struct` -> that could be a `data class` without methods and immutability enforced. It would be a nice way to define a struct that is meant to be immutable and only hold data.
   → Decided 2026-10-26 as D212 (methods allowed, deeply immutable); built 2026-10-27 (plan B30).
2. maybe we could have a `suspend loop` that would be just sugar for a loop with cede inside at the end?
```veles
loop {
  // do something
  cede
}

suspend loop {
  // do something
}
```
   → Decided 2026-10-26: not added; D145's automatic yield and `cede` (D197) stay, and plan F21 measures whether that is right.
3. Can we have an inlay hints to show the type of memory allocations? For example, when we create a new object or array, it would be nice to see the type of memory allocation (heap, stack, etc.) in the editor.
   → Decided 2026-10-26 as D213 (opt-in `heap` hints with hover, `veles explain --alloc`); built 2026-10-27 (plan B31).