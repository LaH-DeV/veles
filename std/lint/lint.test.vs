// Tests of std/lint's helpers: the fixes a rule builds and what reporting
// records. The compiler's side (running a rule on a file) is tested in the
// compiler, with examples/lintrules.

test fun context(text: string): Context => Context(file: File.of("a.vs", text), reports: [])

test "a report records its span, message and fix" {
  val ctx = context("let x = 1")
  val span = Span(file: ctx.file, start: 4, end: 5)
  ctx.report(span, "short name", fix: Fix.replace(span, "count"))
  ctx.report(span, "again")
  val reports = ctx.reports.toList()
  expect(reports.len() == 2)
  expect(reports.at(0)?.message == "short name")
  expect(reports.at(0)?.fix?.edits?.len() == 1)
  expect(reports.at(1)?.fix == null)
}

test "text reads what a span covers" {
  val ctx = context("let x = 1")
  expect(ctx.text(Span(file: ctx.file, start: 4, end: 5)) == "x")
  expect(ctx.text(Span(file: ctx.file, start: 40, end: 50)) == "")
}

test "fix helpers build their edits" {
  val file = File.of("a.vs", "abc")
  val span = Span(file, start: 1, end: 2)
  val replace = Fix.replace(span, "B")
  expect(replace.edits.at(0)?.text == "B")
  val remove = Fix.remove(span)
  expect(remove.edits.at(0)?.text == "")
  val insert = Fix.insertBefore(span, "-")
  val at = insert.edits.at(0)?.span
  expect(at?.start == 1 && at?.end == 1)
}
