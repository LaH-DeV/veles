// Prelude — the observable form of a panic (D52): a task's panic surfaces
// at a `gather` boundary as this error; `scope` re-raises it instead.

public error Panic {
  public message: string
  // where it panicked, `file:line:col` relative to the package root (D64);
  // empty for a panic raised inside the runtime
  public location: string = ""
}
