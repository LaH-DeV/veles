// Everything the lint system can be configured with (D177), declared in one
// place: the lints, and the options that are not a level. A lint has a name, a
// default level and a line of documentation, and a package sets its level in
// `[lint]` of its manifest:
//
//     [lint]
//     unused_import = "deny"      # allow | warn | deny
//     long_line = "warn"
//     max_width = 120
//
// "allow" drops the diagnostic, "warn" reports it as a warning, "deny" as an
// error. The level applies to the package's own code; a dependency follows its
// own manifest and the standard library the defaults.
//
// An option is a setting of the lint system that is not a level: `max_width` is
// the column the `long_line` lint allows; `implicit_return` is itself a lint,
// but what it sets is where a body's last expression may be its value, not a
// level.
//
// To add a lint: one `LintInfo` in `lints()`, in name order, and the check that
// reports it by that name. To add an option: a field in `LintSettings` and one
// `Setting` in `lintOptions()`. Either way its key exists in `[lint]` from then
// on, is documented from the line here, and is suggested when mistyped.
use settings { Setting }

/// What a lint does when it finds something.
enum LintLevel {
  /// Silent.
  Allow
  Warn
  /// An error.
  Deny
}

/// One lint of the registry.
struct LintInfo {
  name:    string
  doc:     string
  default: LintLevel
}

/// Where a body's last expression may be its value without `return`, from the
/// most permissive to the strictest.
enum ImplicitReturn {
  /// Anywhere: a function's, a method's or a lambda's `{ }` body ends in its value.
  Full
  /// In what is written with an arrow — every lambda, and `fun f() => expr`.
  Lambda
  /// Only in a body without braces.
  Expr
}

/// What a package set in `[lint]`.
struct LintSettings {
  var implicitReturn: ImplicitReturn = ImplicitReturn.Full
  /// The column `long_line` allows; null for the default.
  var maxWidth: i64? = null
  /// The level set for each lint of the registry; a lint missing has its default.
  levels: MutableMap<string, LintLevel> = [:]
  /// The level set for each rule of a dependency, keyed `package.rule`
  /// (rules written in Veles, std/lint). A rule runs only when named here.
  library: MutableMap<string, LintLevel> = [:]
}

/// The column `long_line` allows when a package does not set `max_width`.
const DEFAULT_MAX_WIDTH: i64 = 120

/// The options of the lint system.
fun lintOptions(): List<Setting<LintSettings>> => [
  Setting(key: "max_width", doc: "the column long_line allows, 20 to 1000 (default 120)", read: (section, value, settings) => {
    val columns = value.asInt()
    if (columns == null || columns < 20 || columns > 1000) {
      throw section.failAt(value, "[lint] max_width is a number of columns from 20 to 1000, e.g. max_width = 120 (the default)")
    }
    settings.maxWidth = columns
  }),
  Setting.choice(
    "implicit_return", "where a body's last expression may be its value: full, lambda or expr", ["full", "lambda", "expr"],
    "[lint] implicit_return is \"full\" (a body's last expression is its value anywhere, the default), \"lambda\" (in a lambda and an '=> expr' body) or \"expr\" (only in a body without braces)",
    (word, settings) => {
      settings.implicitReturn = when (word) {
        "full"   => ImplicitReturn.Full
        "lambda" => ImplicitReturn.Lambda
        else     => ImplicitReturn.Expr
      }
    },
  ),
]

/// Every lint whose level a package may set, sorted by name. `implicit_return`
/// is a lint too, but its setting picks where a body may end in its value
/// rather than a level, so it is not listed.
fun lints(): List<LintInfo> => [
  warn("continue_skips_step", "a `continue` that jumps over the step of a hand-written counting loop"),
  warn("count_loop", "a loop that only counts is written `iter.len()`"),
  warn("else_if_ladder", "three or more `else if` branches in a row are a `when { }`"),
  warn("inline_implement", "an `implement` beside a type can be written inside its body"),
  warn("large_variant", "a variant much larger than the others of a sealed type that is not a tree, which every value is sized for: mark it `indirect`"),
  silent("long_line", "a line longer than `max_width` columns (default 120); the formatter never reflows"),
  warn("loop_is_range", "a counting loop that is `loop (i in a..<b)` written by hand"),
  warn("loop_true", "`loop (true)` is written `loop { }`"),
  warn("needless_throws", "a `throws` clause on a function that cannot throw"),
  warn("negated_is", "`!(x is T)` reads better as `x !is T`"),
  warn("nested_try", "a `try` inside a `try`"),
  warn("never_closed", "a `Closeable` that is never closed or handed on"),
  warn("param_copy", "a parameter changed in a function whose caller never sees the change"),
  warn("pattern_form", "a pattern written in a longer form than needed"),
  warn("redundant_copy", "a `.toList()` of a collection that is not used afterwards: without it the collection is moved immutable"),
  warn("redundant_discard", "`val _ = f()` for a value a statement may simply ignore"),
  warn("redundant_implement", "an `implement Trait { }` around the methods of a variant of that sealed trait"),
  warn("redundant_label", "an argument label that repeats the variable passed (`f(x: x)`)"),
  warn("redundant_mut", "`mut` on a literal whose expected type is already mutable"),
  warn("redundant_public", "`public` where the declaration is already public"),
  warn("self_address", "`&x` stored into `x` itself, so the value would contain itself"),
  warn("stale_reference", "a reference into a collection used after a call that may move its elements"),
  warn("unreachable_else", "an `else` arm after every variant or member has an arm"),
  warn("unreachable_fallback", "a `?:` fallback on a value known to be in range"),
  warn("unsafe_without_reason", "an `unsafe` block without a `// SAFETY:` comment"),
  warn("unused_import", "a name imported with `use m { name }` but never used"),
  warn("unused_variable", "a variable that is never used"),
]

/// A lint that is a warning unless a package says otherwise.
fun warn(name: string, doc: string): LintInfo => LintInfo(name, doc, default: LintLevel.Warn)

/// A lint that is off unless a package asks for it.
fun silent(name: string, doc: string): LintInfo => LintInfo(name, doc, default: LintLevel.Allow)

/// The names of the lints.
fun lintNames(): List<string> => lints().map(lint => lint.name)

fun lintNamed(name: string): LintInfo? {
  loop (lint in lints()) {
    if (lint.name == name) return lint
  }
  null
}

/// The level spelled `allow`, `warn` or `deny`: the member of that name.
fun lintLevelNamed(text: string): LintLevel? => if (text == text.toLower()) LintLevel.parse(text.capitalize()) else null

/// How a level is spelled: its member's name, lowercase.
fun lintLevelName(level: LintLevel): string => level.toString().toLower()
