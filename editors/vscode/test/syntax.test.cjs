const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { join } = require("node:path");
const test = require("node:test");

const grammar = JSON.parse(
  readFileSync(join(__dirname, "../syntaxes/veles.tmLanguage.json"), "utf8"),
);

test("template tags use the same function-call scope as ordinary calls", () => {
  const rule = grammar.repository.templateTag;
  assert.equal(rule.name, grammar.repository.calls.captures["1"].name);
  for (const [source, expected] of [
    ['html"<h1>"', "html"],
    ['val page = html"<h1>"', "html"],
    ['db.sql"select 1"', "sql"],
    ['html.render"<p>text</p>"', "render"],
    ['tag""', "tag"],
    ['html"""\n<h1>text</h1>\n"""', "html"],
  ]) {
    assert.equal(new RegExp(rule.match).exec(source)?.[0], expected, source);
  }
});

test("template tags require an identifier immediately adjacent to the quote", () => {
  const rule = grammar.repository.templateTag;
  for (const source of [
    'html "<h1>"',
    'html\n"<h1>"',
    '123html"text"',
    '"<h1>"',
  ]) {
    assert.equal(new RegExp(`^(?:${rule.match})`).test(source), false, source);
  }
  assert.equal(
    new RegExp(grammar.repository.calls.match).exec("html(parts, values)")?.[1],
    "html",
  );
});
