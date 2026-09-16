// Prelude — the observable form of a panic (D52): a task's panic surfaces
// at a `gather` boundary as this error; `scope` re-raises it instead.

pub error Panic { pub message: string }
