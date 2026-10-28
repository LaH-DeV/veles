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
   → Decided 2026-10-26: not added; D145's automatic yield and `cede` (D197) stay, and plan F21 measures whether that is right. `cede` built 2026-10-27 (plan F20); measured the same day (F21): the automatic yield wakes a waiting task within ~16 ms (Go: 20–28 ms), so it stays; a `cede` at every round costs ~110 ns each on one thread, every 4096 rounds costs nothing measurable (`bench/results.md`). Q65 (how often a back edge polls) is open.
3. Can we have an inlay hints to show the type of memory allocations? For example, when we create a new object or array, it would be nice to see the type of memory allocation (heap, stack, etc.) in the editor.
   → Decided 2026-10-26 as D213 (opt-in `heap` hints with hover, `veles explain --alloc`); built 2026-10-27 (plan B31).