// Lint rules (D177, std/lint): ordinary values in an ordinary package. A
// package that depends on this one turns them on in its manifest,
//
//   [lint]
//   acme_lints.no_sleep = "warn"
//
// and `veles check` runs them, in the compiler's evaluator, on its files.
use lint

/// `await sleep(d)`: the program waits for time instead of for the thing it
/// needs. The fix removes the statement.
const fun noSleep(): lint.Rule => lint.Rule(
  name: "no_sleep",
  doc: "a call to sleep",
  create: ctx => lint.Visitor(
    onAwaitExpr: (waited, c) => {
      val x = *waited.x
      when (x) {
        is lint.CallExpr => {
          val callee = *x.callee
          when (callee) {
            is lint.NameExpr => if (callee.name == "sleep") {
              c.report(waited.pos, "waiting for a fixed time hides what is being waited for", fix: lint.Fix.remove(waited.pos))
            }
            else             => { }
          }
        }
        else             => { }
      }
    },
  ),
)

/// A function whose name starts with an upper-case letter.
const fun lowerCaseFunctions(): lint.Rule => lint.Rule(
  name: "lower_case_functions",
  doc: "a function named with a capital",
  create: ctx => lint.Visitor(
    onFunDecl: (fn, c) => {
      val first = fn.name.name.slice(0, 1)
      if (first != first.toLower()) {
        c.report(fn.name.pos, "'${fn.name.name}' starts with a capital: functions are written in camelCase")
      }
    },
  ),
)

public const fun rules(): List<lint.Rule> => [noSleep(), lowerCaseFunctions()]
