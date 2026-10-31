# First-Party Templates: Decision Packet

Prepared 2026-10-29. This is a menu of decisions, not approval of the
recommendations and not documentation of implemented APIs. D227/D228 in
`veles-spec.md` remain the source of approved semantics.

The user's direction: exceptionally pleasant developer experience, with safety
as the default. Do not force a choice between strategies that could usefully
coexist as separate `template fun`s. Conversely, another name is not a reason
to duplicate engines or silently weaken a guarantee.

## Agreed Ground

- First-party scope: HTML, SQL, JSON and URL. Other domains are deferred.
  Logging message construction stays unchanged; localization is separate.
- Standalone `html` and `url` modules. No umbrella `template` or `web` module.
  SQL remains with `db`; JSON remains with `json`.
- D227 is implemented on macOS: contextual `template fun`, exactly
  `List<string>` followed by `List<V>`, no trailing parameters. Different tags
  can choose different value and result types.
- Invocation is tag-only: direct calls and function-value references are
  refused, including within the defining module. Runtime parsers remain
  ordinary functions. Standard-library processors are still trusted code;
  tag-only invocation is not a proof of their implementation's correctness.
- Expressions in holes follow ordinary evaluation, error and suspension rules.
  No new laziness or compile-time domain validator is implied by `template`.
- D228 chooses full contextual HTML, including JavaScript and CSS, and general
  absolute/relative URLs. These modules are not implemented yet.
- HTML template authors are trusted; runtime hole values are not. Constructing
  HTML from trusted literal structure is different from sanitizing untrusted HTML.

## Reading the Examples

All code fences use `text`: they are **proposed API sketches**, not programs
claimed to compile today. Names including `html.render`, `url.build`, `url.web`,
`json.document`, `html.data` and `WebUrl` are not currently available.

Current foundations are `db.sql`, `db.Sql`, `db.Param`, `db.ident`,
`codec.Value`, `json.toValue` and ordinary Veles template declarations. Current
`http.Response.html` accepts a string; accepting a typed `Html` is a proposal.

```text
use html
use url
use json
use db { sql }

val profile = try url.build"/people/${id}?tab=${tab}"
val page = try html.render"<a href=\"${profile}\">${name}</a>"
val payload = try json.document"""{"name": ${name}, "active": ${true}}"""
val query = sql"select id from users where name = ${name}"
```

The proposed `try` is conservative: runtime data or context validation can fail.
T3 considers static diagnostics without pretending runtime failures disappear.

## Decision Map

| ID  | Topic                 | Recommendation to Discuss                                       | Can Strategies Coexist?                            |
| --- | --------------------- | --------------------------------------------------------------- | -------------------------------------------------- |
| T1  | Names/imports         | Domain verbs; optional aliases                                  | Yes, without global registration                   |
| T2  | Results/conversions   | Domain types; explicit text extraction                          | Different guarantees should retain different types |
| T3  | Validation/effects    | Shared validators and an explicit static-validation hook        | Prepared plans and one-shot tags can share rules   |
| T4  | URL standards         | RFC core plus explicit browser strategy                         | Yes, with declared conversion boundaries           |
| T5  | URL holes             | Whole segments/query values; typed composition elsewhere        | Named policies, not arbitrary substitution         |
| T6  | URL value API         | Encoded/decoded distinction, resolution, query helpers, limits  | Parsing policy belongs in explicit entry points    |
| T7  | HTML text/attributes  | Contextual rendering and compatible fragments                   | Stricter variants may coexist                      |
| T8  | JS/CSS values         | Data insertion, typed CSS, explicit structured data             | Do not confuse data processors with code trust     |
| T9  | Composition/raw input | Typed joins/builders; no automatic raw promotion                | Explicit dangerous APIs only if approved           |
| T10 | JSON output           | Tree default; optional serialized-result variant                | Yes, one insertion contract with distinct results  |
| T11 | JSON holes/conversion | Complete values using existing encoding rules                   | More hole categories require their own contracts   |
| T12 | SQL evolution         | Preserve parameterization; decide placement checking separately | Dialects require matching query types/adapters     |
| T13 | HTTP integration      | Narrow boundary traits or distinct named functions              | No overloading, no broad Display conversion        |
| T14 | Options/reuse         | Ordinary functions and optional compiled plans                  | Multiple tags for coherent stable policies         |
| T15 | Delivery/quality      | Shared engines, established oracles, full agreed coverage       | Every strategy needs independent conformance       |

Recommendations below are not selections. Deferring a new API while retaining
current behavior remains an option; record deferred requirements explicitly.

## Worked Alternatives: One Application, Several Strategies

The following comparisons make the remaining choices visible at the call site.
Everything in this section is proposed, including the types in the signatures.

### A. General References and Browser URLs Together

```text
use url

val report = try url.build"../reports/${name}?download=${download}"
val publicReport = try url.web"https://example.com/reports/${name}?download=${download}"
```

Both can exist. The first is a general reference that can be resolved against
a base using RFC rules; the second chooses the browser URL algorithm. Neither
name alone grants safe-navigation or SSRF permission. If the second tag also
restricts schemes to HTTP(S), that extra restriction must be explicit in its
contract and result type, not attributed to WHATWG itself.

Illustrative declarations for signature review:

```text
// In url, not new global tags:
public template fun build(parts: List<string>, values: List<Component>): Url throws Invalid
public template fun web(parts: List<string>, values: List<Component>): WebUrl throws Invalid
```

Component needs implementations for supported scalar data, with explicit typed
wrappers/builders for broader structural insertions. It is not a license to
apply arbitrary Display conversion. A WebUrl may require a base for relative
input; because tags take exactly two parameters, base-dependent operations
belong in ordinary functions or a separately approved configured API. General
relative-reference support remains available through Url.

Compare three explicit policies rather than confuse them:

| Policy                 | Meaning                                | Example API Shape                               |
| ---------------------- | -------------------------------------- | ----------------------------------------------- |
| Reference syntax       | Parse/resolve a general URI reference  | url.build / url.parse                           |
| Browser interpretation | Use the browser URL algorithm          | url.web / url.parseWeb                          |
| Application permission | Permit navigation/fetch to this target | consumer validation or a narrower checked value |

**Recommendation:** allow the first two strategies to coexist, with checked
conversions and a separate consumer policy. One ordinary tag must not infer the
intended standard from its input. Decide implementation order separately from
whether both strategies belong in the eventual API.

### B. Contextual HTML and an Optional Stricter Renderer

```text
use html

val badge = try html.render"<strong>${name}</strong>"
val page = try html.render"<a href=\"${publicReport}\">${badge}</a>"
```

The default handles the full contexts selected in D228, with context-appropriate
encoders or explicit refusals for ambiguous positions. A stricter renderer
could reject script/style/event contexts for email or another restricted
audience, but must have a concrete documented policy and distinct useful name.
It is not a substitute for completing the default renderer.

```text
// Illustrative declaration in html:
public template fun render(parts: List<string>, values: List<Content>): Html throws RenderError
```

The ergonomic target is direct string/bool/number holes and Html composition,
with standard implementations of Content. Users should not implement a trait
just to put their name on a page. For custom structured values, compare:

```text
try html.render"""<script>const state = ${html.data(state)};</script>"""
// Alternative explicit-per-hole approach:
try html.render"""<script>const name = ${html.jsValue(name)};</script>"""
```

**Recommendation:** contextual defaults for ordinary scalar data; explicit
structured-data and typed CSS wrappers where intent otherwise becomes unclear.
Wrappers such as data/jsValue/css.px are proposed names. Decide whether they
encode eagerly and throw, or wrap a value for the renderer to encode and report
errors; examples must reflect that choice with normal try handling.

There should be no default tag that trusts all strings, and no universal Html
fragment that is also trusted as JavaScript, CSS, an attribute and a URL.

### C. JSON Tree and Serialized Output Together

```text
use json

val user = try json.document"""{"name": ${name}, "roles": ${roles}}"""
val envelope = try json.document"""{"user": ${user}, "source": "import"}"""
val serialized = try json.text"""{"user": ${user}, "source": "import"}"""
```

The first two produce structured values for composition. The last can target
serialization without requiring a tree. This is a legitimate pair of template
functions with the same insertion rules, not an either/or language decision.

```text
// Illustrative declarations in json:
public template fun document(parts: List<string>, values: List<Argument>): codec.Value throws TemplateError
public template fun text(parts: List<string>, values: List<Argument>): JsonText throws TemplateError
```

Argument must reuse existing Encodable semantics. Compare automatic standard
adapters for native scalar/list/map values against an explicit json.value(value)
adapter for custom Encodable types; neither option changes Veles's general
implicit-conversion rules. JsonText here is a proposed typed serialized result,
not an already implemented wrapper. Returning plain string is another choice;
its validity guarantee cannot survive arbitrary later concatenation.

**Recommendation:** tree-first ergonomics, a serialized variant if its API and
performance benefit justify it. Share parsing, placement and encoding rules;
do not give the text variant a less safe interpretation of holes.

### D. SQL Keeps Values and Fragments Separate

```text
use db { sql }

val filter = sql"name = ${name}"
val query = sql"select id from users where ${filter}"
```

This composes existing typed Sql fragments without interpolating parameter
text. New placement diagnostics and additional dialects are separate decisions,
not prerequisites for retaining this API. A future dialect-specific tag must
return the appropriate query type and use its matching adapter. It must not
silently reinterpret PostgreSQL Sql as another database's language.

### E. What Must Remain Consistent Across Variants

- Function names select coherent domain strategies, never secretly turn safety off.
- Source structure stays source structure; values do not become raw template input.
- Expressions evaluate once, in source order; no tag adds hidden I/O or lazy holes.
- Literal-only, empty and multiline templates follow the same declared contract.
- Error behavior, accepted values, resource limits and output types are documented
  per strategy. Do not remove try from an example merely to make it look shorter.
- A browser URL is not automatically permitted for every HTML or HTTP use.
- A JSON tree/text pair can share a contract; RFC/browser URL strategies cannot
  pretend to share all parsing or resolution rules.

## Fine-Grained Choices to Approve With T1-T15

These are not implementation details to silently choose later. They refine the
existing topics and should be answered as part of the same review, not as a
sequence of surprise questions during coding.

| Topic   | Concrete Choice                                | Suggested Direction and Reason                                                                                         |
| ------- | ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| T2/T6   | String serialization versus diagnostic display | Keep credentials out of default URL diagnostics; make full serialization explicit                                      |
| T3      | Runtime error shape                            | Module-specific kind/position/detail; useful stable cases without leaking unbounded input                              |
| T3      | Static hook contract                           | Validate trusted structure and hole positions without running expressions; share the runtime parser                    |
| T5      | Partial path/query holes                       | Whole segments/values first; explicit component builders for other positions                                           |
| T5      | Dot/empty/encoded segments                     | Reject unsafe navigation-changing dot values in web-safe construction; distinguish this from general reference parsing |
| T6      | Query replacement versus append                | Distinct replace/add operations; preserve repeats/order and absent versus empty values                                 |
| T6      | Unicode hosts                                  | Explicit IRI/IDNA/browser processing, not accidental percent-encoding of a hostname                                    |
| T6/T15  | Limits                                         | Bounded external parsing/output/depth with documented defaults and overrides; numeric defaults are still to select     |
| T7      | HTML URL schemes                               | Safe checked relative/HTTP(S) default; explicit policy for mailto/tel/data and protocol-relative targets               |
| T7      | Foreign/unquoted/ambiguous HTML contexts       | Implement and test or explicitly refuse; never generic-escape as fallback                                              |
| T8      | JS/CSS wrappers                                | Native data where unambiguous; explicit structured JS data and typed CSS values                                        |
| T8/T11  | Wrapper errors                                 | Choose eager conversion with try or deferred encoding within the throwing tag; do not hide a Result                    |
| T9      | Html fragment compatibility                    | Compose only in supported HTML-content contexts; explicit text extraction if markup is meant to appear as text         |
| T9/T11  | Nullables                                      | JSON null/SQL NULL; explicit HTML optional/empty policy; no accidental URL text null                                   |
| T9/T11  | Secret values                                  | Preserve existing SQL secret binding; reject accidental HTML/JSON disclosure unless explicitly exposed by the caller   |
| T10/T11 | Serialized JSON type                           | Compare JsonText with string; decide accepted HTTP boundary before adding a wrapper                                    |
| T11     | Duplicate keys and numeric fidelity            | Refuse duplicate literal keys/non-finite values; verify tree precision before promising lossless conversion            |
| T12     | SQL placement failures                         | Distinguish invalid hole placement from full grammar/schema validation; decide warning versus error explicitly         |
| T13     | Legacy HTTP text                               | Explicit compatibility policy or named migration path; never introduce overloads or a broad Display acceptance         |
| T14     | Runtime options and streaming                  | Ordinary APIs/typed plans; same safety contract and normal I/O/cancellation rules                                      |

The decision session should leave no uncertainty about what each safe default
guarantees and which explicit operation a caller uses for a more advanced case.
It need not approve every optional variant for the first release.

## T1. Names and Discoverability

**Problem:** `use url { url }` creates the unwanted same-name import situation.
Making users invent aliases before the simplest example works is poor ergonomics.

1. **Domain verbs, recommended:** `html.render`, `url.build`, `json.document`,
   existing `db.sql`. Module completion reveals the entry point and result type.
2. **Uniform `literal`:** `html.literal`, `url.literal`, `json.literal`.
   Predictable, but describes source spelling rather than the operation.
3. **Short imported tags:** optional aliases such as `markup`, `link` and
   `document`. Convenient for template-heavy files, not a required setup step.

```text
try html.render"<p>${name}</p>"
try html.literal"<p>${name}</p>"
try markup"<p>${name}</p>"
```

JavaScript and Scala favor short interpolator names; Go favors explicit package
functions. Veles can support qualified discovery and optional aliases with its
existing lookup mechanism. Cost: public declarations, examples, completion/docs;
no new parser feature or self-hosting dependency.

## T2. Result Types and Conversions

**Problem:** flattening every result to string erases the property that made
the template useful. Creating a second JSON tree would duplicate `codec.Value`.

1. **Domain values, recommended:** private-content `Html`, structured `Url`,
   existing `Sql`, existing `codec.Value`. Explicit serialization or `toString`.
2. **Text everywhere:** easy integration, but downstream code cannot distinguish
   arbitrary input from processed values. Not recommended.
3. **Domain values with implicit string conversion:** less ceremony, but silent
   loss of type guarantees. Keep current types rather than add this by default.

```text
val target: url.Url = try url.build"/search?q=${query}"
val page: html.Html = try html.render"<a href=\"${target}\">Search</a>"
val output = page.toString() // explicit extraction; not a new trust proof
```

A general URL proves syntax, not safe navigation or safe fetching. An HTML
fragment is not a JavaScript/CSS fragment. Preserve `Sql.toString()` displaying
placeholders rather than values. Decide whether URL display redacts credentials
while an explicitly named serialization can include them.

Swift's typed interpolation and Python's structured templates preserve meaning
before rendering. Cost: public types/conversions and boundary APIs. Self-hosted
tools can retain typed fragments until final emission, without new coercion rules.

## T3. Static Validation and Error Behavior

**Problem:** a literal typo should be an editor diagnostic at the actual typo,
not an error discovered only on a deployed request. Runtime values can still
fail. Neither `template fun` nor `const fun` alone supplies a static validator
for a template containing runtime expressions.

1. **Library validators plus a narrow compiler hook, recommended target.** A
   validator checks literal structure and hole positions/categories without
   evaluating runtime expressions. Runtime processing shares those rules and
   handles data-dependent errors. The hook's spelling/result type needs a decision.
2. **Runtime-only first:** smaller compiler change, with typed errors and `try`.
   A viable explicit delivery choice, but inferior feedback for static mistakes.
3. **Compiler-known standard tags:** faster initial integration, but duplicated
   domain rules and special names complicate extension and self-hosting.
4. **Defer new validation machinery:** leave the domain work open rather than
   silently claiming runtime checking is compile-time safety.

```text
try json.document"""{"name": ${name},}"""
// Static diagnostic at the trailing comma, if static validation is approved.

try html.render"<a href=\"${input}\">Open</a>"
// A runtime scheme-policy failure still needs a typed error.
```

Keep `try` whenever runtime data can fail. Nonthrowing rendering requires a
proved total contract for accepted values, not just literal structure. Preserve
D157: no implicit Result/task formatting. Expressions evaluate once, in source
order; validation must not execute them early or duplicate side effects.

Diagnostics need domain, insertion context, source/hole position and a useful
fix. Avoid secrets and unbounded input in errors. Options for runtime errors are
a single module error with kind/position/detail or a documented union; choose
before exposing signatures. Invalid external input throws, not panics.

Go's HTML engine analyzes contexts; Rust checks format literals; Julia uses
macros. None requires Veles to approve general macros. Cost: sema/HIR, validator
contract, source mapping, LSP, selfhost parity and runtime/static differential tests.

## T4. URL Standards: Both Can Exist

**Problem:** RFC 3986 references and WHATWG browser URLs differ in acceptance
and normalization. A general RFC value can change interpretation in a browser.

1. **RFC core plus an explicit browser strategy, recommended target:**
   `url.build` for general references, proposed `url.web`/`url.parseWeb` for
   browser semantics. Separate result types or explicit strategy metadata.
2. **RFC core initially:** HTML accepts only a separately checked subset whose
   browser interpretation is safe. Explicitly defer a general WHATWG engine.
3. **Browser-default web tag plus separate RFC tag:** also coherent, provided
   general references promised by D228 remain supported rather than disappearing.

```text
try url.build"../reports/${name}?download=${flag}"
try url.web"https://example.com/reports/${name}?download=${flag}"
```

Use the same comparison inputs: backslashes, controls, percent escapes in hosts,
Unicode/IDNA, unusual IPv4 forms, IPv6/IPvFuture, empty authority and default
ports. Follow the selected standard; do not invent a hybrid normalization rule.

Another `template fun` can select another strategy, but does not implement its
engine. Never parse under one standard and resolve under another implicitly.
Browser parsing alone does not impose an HTTP(S)-only or safe-navigation policy.
URL syntax validity, navigation permissions and SSRF protection are distinct.
DNS/private-address checks belong in the network client on every connection and
redirect, not a pure URL constructor.

Cost: potentially two engines, host processing, checked conversions and separate
oracle corpora. Go `net/url` is not a WHATWG oracle; use RFC vectors and browser
URL test data for their respective contracts. Self-hosted tools choose a declared
strategy without depending on network access.

## T5. URL Hole Placement

**Problem:** a hole represents data, not permission to supply another URL grammar.
Percent-encoding alone does not prove that downstream routing preserves intent.

1. **Whole segments/query values, recommended baseline:** literal scheme,
   authority and query names. Encode UTF-8 data with the appropriate component
   encoder; refuse unsupported placements with a builder alternative.
2. **More typed holes:** explicit host, port, fragment or multi-segment values.
   Useful expansion, each with a separate validation contract, not arbitrary Display.
3. **Substitute text then parse:** a value can change structure before parsing.
   Not recommended; ordinary parsers/builders remain the explicit low-level route.

```text
val account = "sales/europe"
val query = "tea & coffee"
try url.build"https://example.com/accounts/${account}?q=${query}"
// .../accounts/sales%2Feurope?q=tea%20%26%20coffee

try url.build"https://${host}/"         // baseline refusal: authority hole
try url.build"/prefix-${name}"          // baseline refusal: partial segment
base.segment(name).withQuery("q", query) // proposed ordinary builder route
```

Specify empty segments, dot segments, existing percent escapes, control/NUL
characters and encoded separators. Recommended web-safe segment construction
rejects `.`/`..` values: `%2E` is not a reliable defense against browser dot
normalization. A literal percent in data is encoded, not a pre-escaped bypass.
No repeated decoding. General parsing can retain dot segments; building an
untrusted segment has a stricter role. A URL is not a sandboxed filesystem path.

Named tags may select substantial policies, not every combination of options.
Cost: component-aware parsing, percent encoders, builder methods and hostile
normalization/round-trip tests. All strategies use existing tag lookup.

## T6. General URL Value API and Limits

Proposed public roles to settle together:

```text
url.parse(text, maxBytes: ...): Url throws url.Invalid
url.build"...": Url throws url.Invalid
base.resolve(reference): Url throws url.Invalid
base.segment(text): Url throws url.Invalid
base.withQuery(name, value): Url throws url.Invalid
target.toString(): string
```

1. **Coherent reference API, recommended:** parser, serialization, resolution,
   explicit component operations and query-field helpers.
2. **Parser/tag only initially:** smaller, but not completion of the already
   chosen resolution scope. Track missing operations explicitly.
3. **Text wrapper:** does not preserve the distinctions needed by composition.

Decide these semantics, not just method names:

- Preserve absent versus empty query/fragment: `/x`, `/x?`, `/x#` differ.
- Preserve encoded separators; distinguish raw/encoded from decoded accessors.
- Query fields are an optional layer over an opaque query. `+` is not universally
  a space. Preserve order and repeated keys; distinguish replace from append.
- Preserve reference representation by default; canonicalization is explicit.
  Equality must not claim universal server/browser equivalence.
- Separate userinfo and redact credentials in diagnostic display. Serialization
  versus redacted display needs unambiguous names and tests.
- URI syntax, Unicode IRI input and IDNA hostname conversion are different
  contracts. Choose strict ASCII URI parsing plus explicit conversion, or an
  explicitly named Unicode/browser API. Encoding data holes does not solve hosts.
- Bound external parsing: safe default with caller override (as D220 does) or a
  required limit. Reject over-limit input; never truncate silently.
- Resolution uses the same standard as parsing; RFC section 5.4 vectors are a gate.

The high-level tag should remain one expression; the low-level builder should
support runtime bases and dynamic names without hand-encoding. Pure URL APIs
must not suspend, perform DNS or depend on HTTP.

## T7. HTML Text and Attributes

**Problem:** element text, attribute text and URL values require different
treatment. Entity decoding may happen before a nested grammar interprets data.

1. **Contextual `html.render`, recommended:** parse literal structure and select
   context encoders. Html fragments splice only in compatible HTML-content contexts.
2. **Additional narrower tags:** a text-content-only renderer may be useful for
   a concrete audience, but cannot replace D228's full contextual renderer.
3. **Generic character escaping everywhere:** insufficient for the approved
   scope. A text-escape utility can exist without being called a template engine.

```text
val target = try url.build"/profiles/${id}"
val badge = try html.render"<strong>${name}</strong>"
try html.render"<a href=\"${target}\" title=\"${title}\">${badge}</a>"
try html.render"<div title=\"${badge}\">...</div>"
// Last line: refuse Html-as-attribute rather than treating it as trusted text.
```

Recommended default URL-attribute policy: checked relative references and HTTP(S)
navigation, with explicit policy/types for mailto, tel, data or other schemes.
Decide protocol-relative references too. A general Url is not a trust proof for
an HTML attribute. Controls, entities, browser reinterpretation, `srcset` and
CSS URLs need their own rules, not a string-prefix check.

Static scripts/styles are trusted author code. Dynamic tag/attribute names,
comments, doctypes, event-handler source and malformed boundaries can be refused
even with full contextual support. Foreign content, RCDATA/raw-text elements,
entities and unquoted attributes need support or explicit rejection, never an
accidental scanner fallback. Go html/template is a useful precedent, not a
promise of identical byte output. Cost: tokenizer/context engine, entity handling,
composition and hostile-input corpus; share validators with self-hosted tooling.

## T8. JavaScript and CSS Values

**Problem:** data, a quoted JS string, a JS expression and CSS syntax are not
interchangeable. Protect both the nested language and HTML raw-text termination.

1. **Data insertion plus explicit structured wrappers, recommended:** native
   strings/bools/numbers where unambiguous; `html.data<T: Encodable>(value)` in a
   complete JS-value position; typed colors, lengths, keywords and URLs for CSS.
2. **Explicit wrappers for every non-HTML hole:** more ceremony, but clear intent.
   These can be optional helpers alongside contextual defaults, not another tag.
3. **Named specialized processors:** viable for genuinely different policies;
   preserve result/context types rather than returning universal Html trust.

```text
try html.render"""
  <script>const state = ${html.data(state)};</script>
  <style>.panel { width: ${html.css.px(12)}; }</style>
  """
```

Those wrapper names are sketches. A JS-value hole gets data serialization
protecting the surrounding HTML context; a hole inside a JS string gets string
escaping, not an already quoted expression. `</script>`/`</style>` cannot be
reintroduced by data. Regex/template literals, comments and syntax-sensitive
JS positions require explicit handling or a useful refusal; do not guess the
division-versus-regex distinction from one preceding character.

CSS strings, functions, URLs, custom properties and declaration boundaries also
need a model. Non-finite numbers, invalid scalars and dangerous schemes fail
with typed errors. No string becomes executable JS or CSS by default. A future
trusted-code escape hatch needs separate approval and a distinct context type.
Swift demonstrates typed interpolation options; Go demonstrates nested grammar
encoding. Cost: JS/CSS analysis, encoders, adapters and adversarial oracle tests.
Another template name cannot remove that work.

## T9. Composition, Optional Values and Raw Input

1. **Typed fragments/joins, recommended:** complete HTML fragments compose in
   content positions; URL components, JSON values and SQL fragments retain their
   own rules. Provide collection composition instead of displaying a list.
2. **Automatic string promotion:** convenient but collapses the trust boundary.
3. **Explicit dangerous constructors:** if justified by migrations or generated
   trusted source, follow the explicitness of existing Sql.dangerouslyRaw. Do not
   add an HTML equivalent automatically or disguise it as a normal raw tag.

```text
val rows: MutableList<html.Html> = []
loop (user in users) rows.push(try html.render"<li>${user.name}</li>")
val list = try html.render"<ul>${html.join(rows)}</ul>"
```

If rendering throws, `users.map(user => try render(...))` follows D116 and
produces results to handle; it must not silently become a list of successful
fragments. Prefer a clear error-propagating loop or an approved collection helper.
Nullable policies are domain-specific: JSON null, SQL NULL, explicit empty/optional
HTML fragments, no accidental URL text `null`. Preserve D157 for Results/tasks.

Dynamic structures use normal builders; do not introduce a second template
language for loops/conditionals without a decision. A sanitizer for untrusted
HTML is a separate API with a policy, not the renderer under another name.
Cost: joins/builders, compatible-context types, error propagation and negative
tests. High-level callers should never need hand-written escaping.

## T10. JSON Result Strategies

**Problem:** JSON already has a tree and encoders. Do not serialize then parse
just to reconstruct values, or force everyone through a new duplicate tree.

1. **Tree default plus optional text variant, recommended target:**
   `json.document` returns codec.Value; a proposed `json.text` returns serialized
   JSON or a distinct serialized-value type. Same hole rules, different output.
2. **Tree only initially:** explicit existing json.encode for output. Smaller API;
   add a direct-output variant after measurement demonstrates the need.
3. **Text only:** easy sinks, poor composition; existing encoders may already
   serve that need better than another template API.

```text
val payload = try json.document"""{"name": ${name}, "age": ${36}}"""
val envelope = try json.document"""{"user": ${payload}}"""
val wire = try json.text"""{"user": ${payload}}""" // optional separate strategy
```

This is a good case for multiple template functions, not a forced either/or.
Avoid claiming the text version is faster until tested, or that a returned
string preserves its JSON validity when later concatenated with arbitrary text.
Cost: tree/streaming processing paths and parity/benchmark tests.

## T11. JSON Holes, Encoders and Policies

1. **Complete JSON values, recommended:** holes at value positions; strings are
   quoted, numbers stay numbers, null stays null, collections/Encodable structs
   become data. Literal keys initially; dynamic objects use builder APIs.
2. **String-fragment/key holes too:** possible with explicit context handling,
   but additional key-collision/duplicate policies and user surprises.
3. **Text substitution then parse:** parsing does not restore lost data boundaries;
   not a safe default merely because the final document is valid JSON.

```text
try json.document"""{"name": ${name}, "roles": ${roles}}"""
try json.document"""{"name": "${name}"}"""
// Proposed diagnostic for second line: remove quotes around the value hole.
```

Reuse Encodable/json.toValue rather than Display. Decide exact accepted types and
the domain conversion trait/wrapper needed by the homogeneous values list; do
not invent implicit conversions or assume trait object-safety without checking.

Specify duplicate keys, numeric precision, non-finite floats, Unicode scalar
handling, depth/output limits and secrets. Recommended: reject non-finite values
and duplicate literal keys. If codec.Value cannot preserve a number, report it
or approve a representation change before promising lossless conversion. Do not
change existing encoder policies accidentally. Pretty printing, casing and enum
options use normal APIs or configured plans, not extra template parameters.
Cost: parser hole support, encoder integration, limits and round-trip vectors.

## T12. SQL: Preserve Strengths, Specify New Guarantees

1. **Preserve parameterization and strengthen placement feedback, recommended:**
   plain data is bound; db.ident handles identifier syntax; Sql fragments compose;
   secrets/values are not logged. Declare the scope of any new lexical validator.
2. **Full parser/schema checking:** valuable future checked-query strategy, with
   dialect/version costs and no silent build-time database dependency.
3. **Keep existing tag behavior:** valid if documented precisely; parameterization
   alone must not be advertised as a complete SQL parser or schema validator.

```text
sql"select * from users where name = ${name}"
sql"select * from users where name = '${name}'" // placement warning/error if approved
val order = db.ident(column) ?: return
sql"select * from users order by ${order}"
```

Identifier syntax validation is not authorization: applications still allowlist
permitted columns. A parameter is not an identifier. Lexical checks must handle
strings, escaped identifiers, comments and dollar-quoting for the chosen dialect.
Existing db is PostgreSQL-specific. Another tag for another dialect requires an
appropriate query type, placeholder strategy and driver; not one universal Sql
whose meaning changes at execution. Cost: SQL library/tests and optional shared
static validator; deterministic offline self-hosting, no network in compilation.

## T13. HTTP Integration Without Overloading

**Problem:** current Response.html/Response.json accept text and client methods
accept URL text. Veles has no overloading (D179); adding signatures side by side
is not an option. Broad Display arguments would erase type guarantees.

1. **Narrow boundary traits, recommended where supported:** Html implements an
   HTML-body contract; URL types implement a checked request-target contract.
   Explicitly decide whether string remains supported for compatibility.
2. **Distinct named typed/text functions:** migrate the safe API to typed input;
   legacy text gets an explicit spelling such as htmlText, with diagnostics/fixes.
3. **Existing text boundary plus explicit conversion:** smallest transition,
   but safety types are discarded at the boundary. Do not call it the final
   safe-default integration without a decision.

```text
val page = try html.render"<p>${name}</p>"
http.Response.html(page) // proposed typed boundary, not a second overload
```

Preserve ordinary error handling and adapter behavior. HTTP URL adaptation still
checks supported scheme, authority, userinfo, fragment and network policy.
Dependency direction is http -> html/url, never the reverse. Cost: public
signatures, whole-tree migration if required, client/server and handler tests.

## T14. Runtime Options and Reusable Plans

1. **Coherent named tags plus normal APIs, recommended:** url.build/url.web
   select stable semantics. Ordinary functions/builders handle bases, policies,
   sizes, serialization options and output writers.
2. **Compiled plans as complementary APIs:** prepare trusted structure once and
   render typed data many times. Decide input provenance and hole binding before
   exposing html.compile; do not smuggle in an unrestricted template loader.
3. **New tag-factory syntax now:** `html(mode: Strict)"..."` would change D227's
   name-based tag grammar and evaluation rules. Defer unless real usage needs it.

Never call a weaker-safety variant only fast or simple. If a stricter renderer
rejects more contexts, name/document the policy. Streaming output must propagate
I/O errors and cancellation normally; distinguish output size, input size and
nesting limits. Prepared plans do not imply a builder protocol or lazy holes.
Cost: extra public types only where justified and measured, no new syntax merely
to offer another template function.

## T15. Implementation and Quality

1. **Shared Veles library engines plus established oracles, recommended:** runtime
   and static validation share parsing rules. Go contextual HTML is one reference,
   RFC vectors test RFC resolution, browser/web-platform tests cover WHATWG.
2. **Proven native engine:** separate dependency/FFI/platform decision. Linking an
   HTML parser does not supply JS/CSS context analysis or output encoders.
3. **Minimal optimistic scanners:** not a substitute for the full contextual
   engine already chosen. Reduced scope needs explicit approval.

Stages: settle this packet; URL primitives/resolution; full HTML context engine;
JSON template/SQL refinements; static hook/prepared plans; HTTP integration,
docs, measurements and platform gates. Incremental implementation is acceptable;
silently narrowing the final scope is not. Keep unfinished requirements tracked.

Required gates:

- Positive/negative placement/type tests for every supported context; empty,
  adjacent and multiline holes; raw/cooked escaping and Unicode.
- Evaluation once in source order; generics/nullables/errors/suspension/D157;
  tag-only and qualified-call restrictions; formatter/hover/fix parity.
- HTML entities, foreign/RCDATA/raw-text contexts, nested grammars, malicious
  terminators, URL schemes, JS strings/comments/regex/templates and CSS URLs.
- RFC normal/abnormal resolution, empty query/fragment, percent escapes, dot
  segments, authorities/IPv6/IPvFuture and declared Unicode/browser policies.
- JSON precision/non-finite/duplicate/depth cases and tree/text encoder parity;
  SQL fragment/value separation and dialect-specific placement coverage.
- Typed errors and actionable source spans; bounded parsing/output/depth;
  cancellation and streaming where applicable; no secrets in diagnostics.
- Benchmarks against normal builders and relevant Go references. Independent
  conformance for each strategy; no unmeasured zero-allocation promises.
- Full build/vet/tests and Windows/Linux/macOS verification, plus fuzz/sanitizer
  work scaled to actual native/runtime/parser changes.

## Decision Session

This is preparation only. Do not treat recommendations as approvals or ask one
new question on each coding iteration. Review together, then batch:

1. T1/T2/T3/T14: public shape, types, effects/static validation and options.
2. T4/T5/T6: URL standards, component rules, representation and bounds.
3. T7/T8/T9: HTML contexts, JS/CSS data, composition and escape hatches.
4. T10/T11/T12/T13/T15: JSON/SQL, integration and implementation strategy.

Simple safe examples must stay short. Completion should expose accepted values;
errors should name the context and suggest the next valid expression. A lower-
level route can remain explicit without pushing escaping work onto every caller.
Unresolved choices stay open; approved D227/D228 must not be silently rewritten.

## References

- Veles: `veles-spec.md` D129/D157/D179/D227/D228; `std/db/sql.vs`,
  `std/json/json.vs`, `std/http/client.vs`, `sema/template.go`.
- [JavaScript tags](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals)
- [Scala interpolators](https://docs.scala-lang.org/scala3/book/string-interpolation.html)
- [Swift interpolation](https://github.com/swiftlang/swift-evolution/blob/main/proposals/0228-fix-expressiblebystringinterpolation.md)
- [C# handlers](https://learn.microsoft.com/en-us/dotnet/csharp/advanced-topics/performance/interpolated-string-handler)
- [Python structured templates](https://docs.python.org/3/library/string.templatelib.html)
- [Go contextual HTML](https://pkg.go.dev/html/template)
- [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986)
- [WHATWG URL](https://url.spec.whatwg.org/)
- [Browser URL conformance data](https://github.com/web-platform-tests/wpt/tree/master/url)
- [Rust formatting arguments](https://doc.rust-lang.org/std/macro.format_args.html)
- [Julia prefixed literals](https://docs.julialang.org/en/v1/manual/metaprogramming/#Non-Standard-String-Literals)
- [Java's withdrawn template proposal](https://openjdk.org/jeps/465) (research precedent, not a current guarantee)
