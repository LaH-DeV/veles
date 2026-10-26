1. What about `const struct` -> that could be a `data class` without methods and immutability enforced. It would be a nice way to define a struct that is meant to be immutable and only hold data.
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
3. Can we have an inlay hints to show the type of memory allocations? For example, when we create a new object or array, it would be nice to see the type of memory allocation (heap, stack, etc.) in the editor.