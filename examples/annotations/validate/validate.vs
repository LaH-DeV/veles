// A validation library written with annotations (D174). An annotation is
// typed data on a declaration and does nothing by itself: `Validate` reads
// it while the program is compiled, so a type that says `implement Validate`
// gets its checks, and the table of its rules, with no code in the compiler.

/// The smallest value a number field may hold.
annotation Min(value: i64) on field for Validate

/// The most bytes a text field may hold.
annotation MaxLen(value: i64) on field for Validate

/// Leaves a field out of every check.
annotation Skip on field for Validate

struct Problem {
  val field:   string
  val message: string
}

trait Validate {
  /// Every rule the value breaks, one sentence each; empty when it passes.
  /// `inline loop` is unrolled once per field, so each copy reads its field
  /// with its own type: `<` on a number, `len()` on text.
  fun problems(): List<Problem> {
    val out: MutableList<Problem> = []
    inline loop (f in This.fields) {
      if (!f.has<Skip>()) {
        if (val min = f.attr<Min>()) {
          if (f.get(this) < min.value) out.push(Problem(field: f.name, message: "${f.name} must be at least ${min.value}"))
        }
        if (val most = f.attr<MaxLen>()) {
          if (f.get(this).len() > most.value) out.push(Problem(field: f.name, message: "${f.name} must be at most ${most.value} bytes"))
        }
      }
    }
    out
  }

  /// The rules of the type as text, built by the compiler from the
  /// annotations and the doc comments: nothing is computed at run time.
  static const fun rules(): List<string> {
    val out: MutableList<string> = []
    loop (f in This.fields) {
      if (f.has<Skip>()) continue
      var rule = f.name + ": " + f.typeName
      if (val min = f.attr<Min>()) rule += " (at least ${min.value})"
      if (val most = f.attr<MaxLen>()) rule += " (at most ${most.value} bytes)"
      if (!f.doc.isEmpty()) rule += " — " + f.doc
      out.push(rule)
    }
    out
  }
}
