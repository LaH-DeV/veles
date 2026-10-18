// Writing a lint rule (D177).
//
// A rule is an ordinary value in an ordinary package. The compiler runs it
// while it checks a package that turned the rule on in its manifest
// (`[lint] acme-lints.no_sleep = "warn"`), in its compile-time evaluator: a
// rule is a `const fun`, so it does no I/O and reads only the tree it is
// handed. A package lists its rules in a `public const fun rules()`:
//
//   use lint
//
//   const fun noSleep(): lint.Rule => lint.Rule(
//     name: "no_sleep",
//     doc: "a call to sleep in a test",
//     create: ctx => lint.Visitor(
//       onCallExpr: (call, ctx) => {
//         ...
//         ctx.report(call.pos, "do not sleep in a test", fix: lint.Fix.remove(call.pos))
//       },
//     ),
//   )
//
//   public const fun rules(): List<lint.Rule> => [noSleep()]
//
// `veles check --fix` applies the fixes a rule gives.

/// A change to the text of the file a rule is looking at.
public struct Edit {
  /// The bytes to replace; an empty span inserts.
  public span: Span
  public text: string
}

/// How to resolve a report: replace, insert or remove text.
public struct Fix {
  public title: string = ""
  public edits: List<Edit>

  /// Replace what `span` covers with `text`.
  public static const fun replace(span: Span, text: string): Fix =>
    Fix(edits: [Edit(span, text)])

  /// Delete what `span` covers.
  public static const fun remove(span: Span): Fix =>
    Fix(edits: [Edit(span, text: "")])

  /// Insert `text` where `span` starts.
  public static const fun insertBefore(span: Span, text: string): Fix =>
    Fix(edits: [Edit(span: Span(file: span.file, start: span.start, end: span.start), text)])
}

/// One finding: where, what to tell the author, and optionally how to fix it.
public struct Report {
  public span:    Span
  public message: string
  public fix:     Fix? = null
}

/// What a rule is given when it starts on a file, and reports through.
public struct Context {
  /// The file being looked at.
  public file:    File
  public reports: MutableList<Report>

  /// Tell the author about `span`. A `fix` makes `veles check --fix` able to
  /// resolve it.
  public const fun report(span: Span, message: string, fix: Fix? = null) {
    this.reports.push(Report(span, message, fix))
  }

  /// The source text `span` covers.
  public const fun text(span: Span): string =>
    this.file.content.substring(span.start, span.end) ?: ""
}

/// How loud a rule is when the package that uses it says nothing.
public enum Level {
  Allow
  Warn
  Deny
}

/// A lint: a name (the key in `[lint]`, after the package's), a line of
/// documentation for `veles lint --list`, the level it has by default, and
/// the function that makes its Visitor once per file.
public struct Rule {
  public name:   string
  public doc:    string = ""
  public level:  Level = Level.Warn
  public create: fun(Context): Visitor
}
