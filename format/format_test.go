package format

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// checkRoundTrip formats src and verifies the three properties every
// formatter must have: the output parses, it means the same thing (an
// identical syntax tree), and formatting it again changes nothing.
func checkRoundTrip(t *testing.T, name, src string) string {
	t.Helper()
	orig := source.NewFile(name, src)
	diags := &source.Diagnostics{}
	before := parser.ParseFile(orig, diags)
	if diags.HasErrors() {
		t.Fatalf("%s does not parse: %s", name, diags.Render())
	}
	out, fdiags := Source(orig, Options{})
	if fdiags.HasErrors() {
		t.Fatalf("%s: formatter reported errors:\n%s", name, fdiags.Render())
	}
	diags = &source.Diagnostics{}
	after := parser.ParseFile(source.NewFile(name, out), diags)
	if diags.HasErrors() {
		t.Fatalf("%s: formatted output does not parse:\n%s\n--- output ---\n%s", name, diags.Render(), numbered(out))
	}
	normalizeUses(before)
	normalizeUses(after)
	if a, b := ast.Dump(before), ast.Dump(after); a != b {
		t.Fatalf("%s: formatting changed the syntax tree\n--- before ---\n%s\n--- after ---\n%s\n--- output ---\n%s", name, a, b, numbered(out))
	}
	again, _ := Source(source.NewFile(name, out), Options{})
	if again != out {
		t.Fatalf("%s: formatting is not idempotent\n--- first ---\n%s\n--- second ---\n%s", name, numbered(out), numbered(again))
	}
	return out
}

// normalizeUses merges every run of consecutive `use` declarations into one
// sorted declaration. The formatter reorders imports (useRun), which is the
// one change of syntax tree it is allowed: the meaning of a file does not
// depend on the order or grouping of its imports.
func normalizeUses(f *ast.File) {
	var out []ast.Decl
	for i := 0; i < len(f.Decls); i++ {
		u, ok := f.Decls[i].(*ast.UseDecl)
		if !ok {
			out = append(out, f.Decls[i])
			continue
		}
		merged := &ast.UseDecl{Specs: append([]*ast.UseSpec(nil), u.Specs...)}
		for i+1 < len(f.Decls) {
			next, ok := f.Decls[i+1].(*ast.UseDecl)
			if !ok {
				break
			}
			merged.Specs = append(merged.Specs, next.Specs...)
			i++
		}
		SortUseSpecs(merged.Specs)
		out = append(out, merged)
	}
	f.Decls = out
}

func numbered(s string) string {
	var sb strings.Builder
	for i, l := range strings.Split(s, "\n") {
		sb.WriteString(strings.Repeat(" ", 4-len(itoa(i+1))) + itoa(i+1) + "| " + l + "\n")
	}
	return sb.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// TestCorpus round-trips every Veles source in the repository and every
// code block in the tutorials.
func TestCorpus(t *testing.T) {
	var files []string
	for _, root := range []string{"../examples", "../std"} {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && (strings.HasSuffix(path, ".vs") || strings.HasSuffix(path, ".vss")) {
				files = append(files, path)
			}
			return nil
		})
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			out := checkRoundTrip(t, path, string(data))
			// the repository's own sources are kept formatted, so what
			// `veles fmt --check` would flag fails here instead of in review
			// (none of them sets a [format] style, so the defaults apply)
			if src := strings.ReplaceAll(string(data), "\r\n", "\n"); out != src {
				t.Errorf("%s is not formatted; run `veles fmt %s`", path, path)
			}
		})
	}
	fence := regexp.MustCompile("(?s)```veles\\n(.*?)```")
	filepath.WalkDir("../docs/documentation", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, _ := os.ReadFile(path)
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		for i, m := range fence.FindAllStringSubmatch(text, -1) {
			code := m[1]
			if strings.HasPrefix(strings.TrimSpace(code), "// fragment") {
				continue
			}
			t.Run(filepath.Base(path)+"/"+itoa(i+1), func(t *testing.T) {
				checkRoundTrip(t, path, code)
			})
		}
		return nil
	})
}

// TestStyle pins the canonical output for the constructs the corpus
// exercises loosely: spacing, parentheses, comments, blank lines.
func TestStyle(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"spacing and parens",
			"fun f(a:i64,b:i64):i64{val x=(a+b)*2\nreturn x}\n",
			"fun f(a: i64, b: i64): i64 {\n  val x = (a + b) * 2\n  return x\n}\n"},
		{"author parentheses kept, duplicates dropped",
			"fun f(a: i64): i64 => ((a)) + (a * 2) - (a + 1) * (-(a))\n",
			"fun f(a: i64): i64 => (a) + (a * 2) - (a + 1) * (-(a))\n"},
		{"required parentheses added",
			"fun f(a: i64): i64 => (a + 1) * 2 - (try g()).len()\nfun h(a: i64?): i64 => a ?: throw E()\n",
			"fun f(a: i64): i64 => (a + 1) * 2 - (try g()).len()\nfun h(a: i64?): i64 => a ?: throw E()\n"},
		{"or-fail binds under try",
			"fun f(): i64 throws E => try g() ?! E()\nfun h(): i64 throws E => (try g()) ?! E()\nfun k(): i64 throws E => try (g() ?! E())\nfun m(): i64 => with (r = open()) { r.n }\n",
			"fun f(): i64 throws E => try g() ?! E()\nfun h(): i64 throws E => (try g()) ?! E()\nfun k(): i64 throws E => try (g() ?! E())\nfun m(): i64 => with (r = open()) {\n  r.n\n}\n"},
		{"let-else and ??",
			`fun f() {
val a=g()??0
val b = g() catch (e) { e.n }
val c = g() else return
val d = g()
catch (_) { return }
val Some(x) = h() else continue
val t: i64?? = null
}
`,
			`fun f() {
  val a = g() ?? 0
  val b = g() catch (e) {
    e.n
  }
  val c = g() else return
  val d = g() catch (_) {
    return
  }
  val Some(x) = h() else continue
  val t: i64?? = null
}
`},
		{"do and catch (D98)",
			`fun f() {
val a = do { try g() } catch (e) { 0 }
val b = try g().h() catch (e) { -1 }
val c = g()
  catch { 0 }
}
`,
			`fun f() {
  val a = do {
    try g()
  } catch (e) {
    0
  }
  val b = try g().h() catch (e) {
    -1
  }
  val c = g() catch {
    0
  }
}
`},
		{"nested unary keeps parens",
			"fun f(a: i64): i64 => -(-a) + !(!b)\n",
			"fun f(a: i64): i64 => -(-a) + !(!b)\n"},
		{"postfix on operators",
			"fun f(x: f64): f64 => (-x).abs() + (x as f64).floor() + (try g()).len()\n",
			"fun f(x: f64): f64 => (-x).abs() + (x as f64).floor() + (try g()).len()\n"},
		{"comments keep their place",
			"// top\nuse io // trailing\n\n/* block */\nfun main() {\n  // inside\n  io.println(\"x\") // after\n  // before close\n}\n",
			"// top\nuse io  // trailing\n\n/* block */\nfun main() {\n  // inside\n  io.println(\"x\")  // after\n  // before close\n}\n"},
		{"blank lines capped at one",
			"use io\n\n\n\nfun main() {\n  val a = 1\n\n\n  val b = 2\n}\n",
			"use io\n\nfun main() {\n  val a = 1\n\n  val b = 2\n}\n"},
		{"blocks and bodies always break; empty ones stay",
			"struct P { x: i64, y: i64 }\nstruct E { }\nfun f(p: P) { loop (i in 0..<3) { g(i) } }\nfun g() { }\n",
			"struct P {\n  x: i64\n  y: i64\n}\nstruct E { }\nfun f(p: P) {\n  loop (i in 0..<3) {\n    g(i)\n  }\n}\nfun g() { }\n"},
		{"broken lists get trailing commas",
			"val xs = [\n  1,\n  2]\nfun f() => g(\n  a,\n  b: 2\n)\n",
			"val xs = [\n  1,\n  2,\n]\nfun f() => g(\n  a,\n  b: 2,\n)\n"},
		{"a broken list keeps the author's grouping: a grid stays a grid",
			"val k: List<u32> = [\n  1, 2, 3,\n  4, 5, 6]\nval one = [\n  1,\n  2]\n",
			"val k: List<u32> = [\n  1, 2, 3,\n  4, 5, 6,\n]\nval one = [\n  1,\n  2,\n]\n"},
		{"method chains keep their breaks",
			"fun f(xs: List<i64>): i64 => xs\n    .map(x => x + 1)\n  .len()\n",
			"fun f(xs: List<i64>): i64 => xs\n  .map(x => x + 1)\n  .len()\n"},
		{"elvis continuation",
			"fun f(a: i64?, b: i64?): i64 {\n  val z = a\n      ?: b\n      ?: 0\n  z\n}\n",
			"fun f(a: i64?, b: i64?): i64 {\n  val z = a\n    ?: b\n    ?: 0\n  z\n}\n"},
		{"empty impls drop their braces, field attributes stay",
			"struct U {\n  @key(\"user_id\") id: i64\n  @skip p: string = \"\"\n  implement Codable { }\n  implement Comparable\n  implement Display { fun toString(): string => \"u\" }\n}\nimplement Codable for geo.Point {\n}\nimplement Error for U { /* later */ }\n",
			"struct U {\n  @key(\"user_id\")\n  id: i64\n  @skip\n  p: string = \"\"\n  implement Codable\n  implement Comparable\n  implement Display {\n    fun toString(): string => \"u\"\n  }\n}\nimplement Codable for geo.Point\nimplement Error for U {\n  /* later */\n}\n"},
		{"enum members one per line",
			"public  enum Color:u8 { Red=1, Green\n  Blue  = 10 // ten\n  Pink }\nenum E { }\n",
			"public enum Color : u8 {\n  Red = 1\n  Green\n  Blue = 10  // ten\n  Pink\n}\nenum E { }\n"},
		{"if else if",
			"fun f(x: i64): string => if (x < 0) \"neg\" else if (x == 0) \"zero\" else \"pos\"\nfun g(x: i64) {\n  if (x > 0) {\n    a()\n  }\n  else {\n    b()\n  }\n}\n",
			"fun f(x: i64): string => if (x < 0) \"neg\" else if (x == 0) \"zero\" else \"pos\"\nfun g(x: i64) {\n  if (x > 0) {\n    a()\n  } else {\n    b()\n  }\n}\n"},
		{"when arms",
			"fun f(v: i64): string => when (v) {\n1 => \"one\"\n  2, 3   =>   \"few\"\n  in 4..9 => \"some\"\n  else => \"many\"\n}\n",
			"fun f(v: i64): string => when (v) {\n  1       => \"one\"\n  2, 3    => \"few\"\n  in 4..9 => \"some\"\n  else    => \"many\"\n}\n"},
		{"columns: comments and fields align, wide spreads do not",
			"struct Node {\n  value: T // v\n  left: *Tree<T> // l\n\n  right: *Tree<T>\n}\nval a = 1 // one\nval bbbb = 2 // two\nfun f(v: i64): string => when (v) {\n  1 => \"one\"\n  in 4..9 if v > 5 && v < 8 && v != 7 => \"some\"\n}\n",
			"struct Node {\n  value: T         // v\n  left:  *Tree<T>  // l\n\n  right: *Tree<T>\n}\nval a = 1     // one\nval bbbb = 2  // two\nfun f(v: i64): string => when (v) {\n  1 => \"one\"\n  in 4..9 if v > 5 && v < 8 && v != 7 => \"some\"\n}\n"},
		{"operator breaks stay on the author's side",
			"fun f(): i64 => a %\n  b + (c\n  ) - d\nfun g(): i64 => (a\n  % b)\n",
			"fun f(): i64 => a %\n  b + (c) - d\nfun g(): i64 => (a\n  % b)\n"},
		{"comments at chain breaks and in empty bodies",
			"fun f() {\n  x % [0]  // c1\n    .len()  // c2\n  when { // c3\n  }\n}\n",
			"fun f() {\n  x % [0]   // c1\n    .len()  // c2\n  when {\n    // c3\n  }\n}\n"},
		{"one-element tuple broken",
			"fun f() => (\n  0,\n)\n",
			"fun f() => (\n  0,\n)\n"},
		{"static, variadic, spread and when-val",
			"struct P { x: i64\n static fun of(xs: i64...): P => P(x: xs.len()) }\nfun f(s: string) => when (val r = P.of(1, [2]...)) { else => r }\n",
			"struct P {\n  x: i64\n  static fun of(xs: i64...): P => P(x: xs.len())\n}\nfun f(s: string) => when (val r = P.of(1, [2]...)) {\n  else => r\n}\n"},
		{"inline impls stay in the body",
			"struct P<T> { x: T\n  @inline implement Show { fun show(): string => \"p\" }\n  implement Iterator { type Item = T\n fun next(): T? => null } }\nimplement Other for P<i64> { fun o() { } }\n",
			"struct P<T> {\n  x: T\n  @inline\n  implement Show {\n    fun show(): string => \"p\"\n  }\n  implement Iterator {\n    type Item = T\n    fun next(): T? => null\n  }\n}\nimplement Other for P<i64> {\n  fun o() { }\n}\n"},
		{"patterns",
			"fun f(s: Shape): f64 => when (s) { is Circle(r) => r; is Rect(w, h: hh) if w > hh => w; Point => 0.0; null => 1.0 }\n",
			"fun f(s: Shape): f64 => when (s) {\n  is Circle(r)                => r\n  is Rect(w, h: hh) if w > hh => w\n  Point                       => 0.0\n  null                        => 1.0\n}\n"},
		{"an expression body: '=>', and a returned lambda parenthesized (D140)",
			"fun adder(n: i64): fun(i64): i64 => x => x + n\nfun kept(n: i64): fun(i64): i64 => (x => x * n)\n",
			"fun adder(n: i64): fun(i64): i64 => (x => x + n)\nfun kept(n: i64): fun(i64): i64 => (x => x * n)\n"},
		{"an arm's patterns broken after a comma stay broken",
			"fun f(k: Kind): bool => when (k) {\n  Kind.A, Kind.B,\n        Kind.C => true\n  Kind.D, Kind.E => false\n}\n",
			"fun f(k: Kind): bool => when (k) {\n  Kind.A, Kind.B,\n    Kind.C => true\n  Kind.D, Kind.E => false\n}\n"},
		{"lambdas",
			"val f = (a: i64, b: i64): i64 => a + b\nval g = x => x * 2\nval h = (x) => x\nfun k(xs: List<i64>) { xs.forEach(x => total += x); xs.map(x => { x }) }\n",
			"val f = (a: i64, b: i64): i64 => a + b\nval g = x => x * 2\nval h = (x) => x\nfun k(xs: List<i64>) {\n  xs.forEach(x => total += x)\n  xs.map(x => {\n    x\n  })\n}\n"},
		{"error declarations",
			"public error NotFound { key: string\n  fun message(): string => \"no $key\" }\nerror Failed { message: string }\nerror Set = NotFound | Failed\n",
			"public error NotFound {\n  key: string\n  fun message(): string => \"no $key\"\n}\nerror Failed {\n  message: string\n}\nerror Set = NotFound | Failed\n"},
		{"modifiers and generics keep author order",
			"public fun show<T: Show>(x: T): string => x.show()\nstruct S {\n  public fun bump() { }\n  override fun d(): string => \"\"\n}\n",
			"public fun show<T: Show>(x: T): string => x.show()\nstruct S {\n  public fun bump() { }\n  override fun d(): string => \"\"\n}\n"},
		{"var fields align with the bare ones",
			"struct C {\n  var n: i64 = 0\n  step: i64 = 1\n  private var hits:   i64 = 0\n  public var label: string = \"\"\n}\n",
			"struct C {\n  var n:            i64 = 0\n  step:             i64 = 1\n  private var hits: i64 = 0\n  public var label: string = \"\"\n}\n"},
		{"explicit val, internal and protected var are kept as written",
			"internal struct C {\n  val id: i64\n  internal protected var n: i64 = 0\n  public protected var m: i64 = 0\n  internal fun f() { }\n}\ninternal val k = 1\ninternal fun g() { }\n",
			"internal struct C {\n  val id:                   i64\n  internal protected var n: i64 = 0\n  public protected var m:   i64 = 0\n  internal fun f() { }\n}\ninternal val k = 1\ninternal fun g() { }\n"},
		{"init block keeps its place among the members",
			"struct P {\n  a: i64\n  b: string\n  init {\n    this.b = \"$a\"\n  }\n  fun f(): i64 => this.a\n}\n",
			"struct P {\n  a: i64\n  b: string\n  init {\n    this.b = \"$a\"\n  }\n\n  fun f(): i64 => this.a\n}\n"},
		{"init parameters print like a function's (D73)",
			"struct M<T> {\n  private cell: *T\n  init( value : T, n: i64 = 1 ) {\n    this.cell = &value\n  }\n}\n",
			"struct M<T> {\n  private cell: *T\n  init(value: T, n: i64 = 1) {\n    this.cell = &value\n  }\n}\n"},
		{"arm body broken after the arrow leaves no trailing space",
			"fun f(v: i64): string => when (v) {\n  1 =>  \n    \"one\"\n  else => \"more\"\n}\n",
			"fun f(v: i64): string => when (v) {\n  1 =>\n    \"one\"\n  else => \"more\"\n}\n"},
		{"arrows align across a block arm; a nested when is its own run",
			"fun f(c: u8): i64 => when (c) {\n  '(' => 1\n  ')' => {\n    when (c) {\n      1 => 10\n      222 => 20\n    }\n  }\n  // between arms\n  '+' => 3\n  else => 0\n}\n",
			"fun f(c: u8): i64 => when (c) {\n  '('  => 1\n  ')'  => {\n    when (c) {\n      1   => 10\n      222 => 20\n    }\n  }\n  // between arms\n  '+'  => 3\n  else => 0\n}\n"},
		{"types",
			"fun f(a: (*T)?, b: *T?, c: fun(i64, string): bool suspends throws E, d: (A, B), e: I.Item, g: *raw u8): Map<string, List<i64>> throws A | B { }\n",
			"fun f(a: (*T)?, b: *T?, c: fun(i64, string): bool suspends throws E, d: (A, B), e: I.Item, g: *raw u8): Map<string, List<i64>> throws A | B { }\n"},
		{"a nullable function keeps its parentheses (without them it returns a nullable)",
			"fun f(a: (fun(): i64)?, b: fun(): i64?, c: (sendable fun() suspends)?, d: (fun() throws E)?) { }\n",
			"fun f(a: (fun(): i64)?, b: fun(): i64?, c: (sendable fun() suspends)?, d: (fun() throws E)?) { }\n"},
		{"tests (D78)",
			"test   \"adds \\\"two\\\" numbers\"   {\n  expect(1 + 1 == 2) }\ntest   fun   helper(x: i64) { expect(x > 0) }\n",
			"test \"adds \\\"two\\\" numbers\" {\n  expect(1 + 1 == 2)\n}\ntest fun helper(x: i64) {\n  expect(x > 0)\n}\n"},
		{"suites (D78)",
			"suite   \"router\"   {\n  test fun h() { }\n\n  test \"routes\" { h() }\n  suite \"auth\" { test \"rejects\" { } }\n}\n",
			"suite \"router\" {\n  test fun h() { }\n\n  test \"routes\" {\n    h()\n  }\n  suite \"auth\" {\n    test \"rejects\" { }\n  }\n}\n"},
		{"attributes and extern",
			"@test\nfun t() { }\n@deprecated(\"use g\")\npublic fun f() { }\nextern \"C\" { fun strlen(s: *raw u8): i64\n  fun puts(s: string) }\n",
			"@test\nfun t() { }\n@deprecated(\"use g\")\npublic fun f() { }\nextern \"C\" {\n  fun strlen(s: *raw u8): i64\n  fun puts(s: string)\n}\n"},
		{"type aliases",
			"public type   Key=(i64,u64)\ntype StrMap<V>  =  Map<string,V>\n",
			"public type Key = (i64, u64)\ntype StrMap<V> = Map<string, V>\n"},
		{"use forms",
			"use io\nuse geometry   as   geo\nuse a.b as c\n",
			"use io\nuse a.b as c\nuse geometry as geo\n"},
		{"use with names: sorted, one line, trailing comma dropped",
			"use io{println as p,readLine,\n  eprintln,}, fs{ readFile }\n",
			"use fs { readFile }\nuse io { eprintln, println as p, readLine }\n"},
		{"use block: one per line, sorted, std first, blank lines dropped",
			"use time\nuse shapes\nuse os,\n  io\n\nuse fs\n\nfun main() { }\n",
			"use fs\nuse io\nuse os\nuse time\nuse shapes\n\nfun main() { }\n"},
		{"use block: a commented line stays as written, the rest is grouped",
			"use os  // needed\nuse io\nuse fs\n",
			"use os  // needed\nuse fs\nuse io\n"},
		{"use block: comments above and below are not inside it",
			"// header\nuse os\nuse io\n// about main\nfun main() { }\n",
			"// header\nuse io\nuse os\n// about main\nfun main() { }\n"},
		{"concurrency",
			"fun main() {\n  scope {\n    val t = async work(1)\n    val (a, b) = gather { async f(); async g() }\n    val w = race { val m = ch.recv() => m; sleep(100) => \"t\" }\n    with (f = open(\"a\")) { process(f) }\n  }\n}\n",
			"fun main() {\n  scope {\n    val t = async work(1)\n    val (a, b) = gather {\n      async f()\n      async g()\n    }\n    val w = race {\n      val m = ch.recv() => m\n      sleep(100)        => \"t\"\n    }\n    with (f = open(\"a\")) {\n      process(f)\n    }\n  }\n}\n"},
		// D106: a lambda that is an arm's value gets parentheses
		{"lambda arm",
			"fun pick(double: bool): fun(i64): i64 => when {\n  double => x => x * 2\n  else   => (x => x)\n}\n",
			"fun pick(double: bool): fun(i64): i64 => when {\n  double => (x => x * 2)\n  else   => (x => x)\n}\n"},
		// D100: printed as written — never converted to or from the block form
		{"with statement",
			"fun main() {\n  with  f = open(\"a\")\n  with t =   async work(1)\n  with (g = open(\"b\")) { take(g) }\n  take(f)\n}\n",
			"fun main() {\n  with f = open(\"a\")\n  with t = async work(1)\n  with (g = open(\"b\")) {\n    take(g)\n  }\n  take(f)\n}\n"},
		{"with items without a name (D109)",
			"fun main() {\n  with   sem.acquire()\n  with conn\n  with _ = open(\"c\")\n  with (f = open(\"a\"),   sem.acquire()) { take(f) }\n}\n",
			"fun main() {\n  with sem.acquire()\n  with conn\n  with _ = open(\"c\")\n  with (f = open(\"a\"), sem.acquire()) {\n    take(f)\n  }\n}\n"},
		{"trailing whitespace and CRLF",
			"use io\r\n\r\nfun main() {   \r\n  io.println(\"x\")  \r\n}\r\n",
			"use io\n\nfun main() {\n  io.println(\"x\")\n}\n"},
		// a broken parameter list used to flush every comment up to the end
		// of the *function*, which swept the body's comments into it
		{"a broken parameter list does not swallow the body's comments",
			"fun f(a: i64,\n      b: i64): i64 {\n  // about the first\n  val x = a + b\n  // about the second\n  x * 2\n}\n",
			"fun f(\n  a: i64,\n  b: i64,\n): i64 {\n  // about the first\n  val x = a + b\n  // about the second\n  x * 2\n}\n"},
		{"a comment inside a broken parameter list stays inside it",
			"fun f(a: i64,  // the first\n      // about the last\n      b: i64): i64 {\n  // about the body\n  a + b\n}\n",
			"fun f(\n  a: i64,  // the first\n  // about the last\n  b: i64,\n): i64 {\n  // about the body\n  a + b\n}\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := checkRoundTrip(t, c.name, c.in)
			if got != c.want {
				t.Errorf("\n--- got ---\n%s--- want ---\n%s", got, c.want)
			}
		})
	}
}

func TestOptions(t *testing.T) {
	src := "fun main() {\n  if (true) {\n    x()\n  }\n\n\n\n  y()\n}\n"
	out, _ := Source(source.NewFile("t.vs", src), Options{Indent: "\t", MaxBlankLines: 2})
	want := "fun main() {\n\tif (true) {\n\t\tx()\n\t}\n\n\n\ty()\n}\n"
	if out != want {
		t.Errorf("tab indent, two blank lines:\n--- got ---\n%s--- want ---\n%s", out, want)
	}
	out, _ = Source(source.NewFile("t.vs", src), Options{Indent: "    "})
	want = "fun main() {\n    if (true) {\n        x()\n    }\n\n    y()\n}\n"
	if out != want {
		t.Errorf("four spaces:\n--- got ---\n%s--- want ---\n%s", out, want)
	}
}

// A `public use` (D89) is a declaration of its own: the formatter neither
// merges it into the run of plain imports around it nor sorts its specs, and
// keeps its `public`.
func TestPublicUseIsNotMerged(t *testing.T) {
	src := "use os,io\npublic use geometry\nuse time\npublic   use shapes { area,   Circle as Round }, util as tools\nuse io\n"
	want := "use io\nuse os\npublic use geometry\nuse time\npublic use shapes { area, Circle as Round }, util as tools\nuse io\n"
	if got := checkRoundTrip(t, "public_use.vs", src); got != want {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, want)
	}
}

// `[format] imports`: one `use` per import by default, sorted with the
// standard library first; "merged" writes one comma-separated `use` per
// origin. Either form is read the same, and each style is stable.
func TestImportStyles(t *testing.T) {
	src := "use source { Span, File }, lexer\nuse utf8\n\nuse io, mylib.text as text\n\nfun main() { }\n"
	for _, c := range []struct {
		style ImportStyle
		want  string
	}{
		{ImportsPerLine, "use io\nuse utf8\nuse lexer\nuse mylib.text as text\nuse source { File, Span }\n\nfun main() { }\n"},
		{ImportsMerged, "use io, utf8\nuse lexer, mylib.text as text, source { File, Span }\n\nfun main() { }\n"},
	} {
		got, diags := Source(source.NewFile("t.vs", src), Options{Imports: c.style})
		if diags.HasErrors() || got != c.want {
			t.Errorf("style %d:\n--- got ---\n%s--- want ---\n%s%s", c.style, got, c.want, diags.Render())
			continue
		}
		if again, _ := Source(source.NewFile("t.vs", got), Options{Imports: c.style}); again != got {
			t.Errorf("style %d is not stable:\n%s", c.style, again)
		}
	}
}

// A `lazy` parameter (D90) keeps its modifier through the formatter.
func TestLazyParameterRoundTrips(t *testing.T) {
	src := "fun debug(lazy msg: fun(): string, fields: Field...) {\n  emit(msg, fields)\n}\n"
	if got := checkRoundTrip(t, "lazy.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}

// `if (val x = e && ...)` (D95) prints as written: the value binds tighter
// than `&&`, so a value with `||` keeps its parentheses, and chains, else
// branches and else-if survive.
func TestIfValRoundTrips(t *testing.T) {
	src := "fun f(c: Cookie, xs: List<string>) {\n" +
		"  if (val age = c.maxAge) out.append(age)\n" +
		"  if (val d = c.domain && d.len() > 1) out.append(d) else out.append(\"-\")\n" +
		"  if (ready && val a = first(xs) && val b = a.toInt() && b > 1) {\n    show(a, b)\n  }\n" +
		"  if (val v = (x || y)) show(v) else if (val w = z ?: q) show(w)\n" +
		"}\n"
	if got := checkRoundTrip(t, "ifval.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}

// `static assert(cond, "why")` (D113) prints as written, at module level
// and in a body.
func TestStaticAssertRoundTrips(t *testing.T) {
	src := "const KB: i64 = 1024\n\nstatic assert(KB * 4 == 4096, \"a page is ${KB * 4} bytes\")\n\n" +
		"fun f<T>(x: T): T {\n  static assert(T implements Comparable, \"T sorts\")\n  return x\n}\n"
	if got := checkRoundTrip(t, "assert.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}

// C's `...` (D123) ends an extern parameter list, on one line or broken.
func TestCVariadicRoundTrips(t *testing.T) {
	src := "extern \"C\" {\n  fun printf(format: *raw u8, ...): i32\n  fun anything(...)\n  fun open(\n    path: *raw u8,\n    flags: i32,\n    ...\n  ): i32\n}\n"
	if got := checkRoundTrip(t, "variadic.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}

// C layout (D120): `extern union`, @packed, @align on a struct and on a
// field, @transparent print as written.
func TestCLayoutRoundTrips(t *testing.T) {
	src := "extern union Data {\n  ptr: *raw ()\n  fd:  i32\n}\n\n@packed\nextern struct Event {\n  events: u32\n  data:   Data\n}\n\n" +
		"extern struct Spaced {\n  a: u8\n  @align(16)\n  b: i32\n}\n\n@transparent\nstruct Fd {\n  handle: i32\n}\n\n@align(64)\nstruct Counter {\n  var hits: i64\n}\n"
	if got := checkRoundTrip(t, "layout.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}

// D121: constant parameters and constant type arguments print as written.
func TestConstGenericsRoundTrip(t *testing.T) {
	src := "struct Buf<const N: i64> {\n  data: Array<u8, N>\n}\n\nfun zeros<T, const N: i64>(): Array<T, N> => Array.make(0)\n\n" +
		"struct S {\n  a: Array<u8, 4 * 16>\n  b: MutableMap<string, Array<u8, 200>>\n  c: Array<Array<i64, 3>, 2>\n  d: Buf<8>\n}\n"
	if got := checkRoundTrip(t, "constgen.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}

// D129: a template literal is printed as written, and formatting it again
// changes nothing.
func TestTemplateLiteralRoundTrips(t *testing.T) {
	src := "use db\n\nfun find(name: string, age: i64) {\n  val q = db.sql\"select * from users where name = ${name} and age > ${age}\"\n  val plain = sql\"select 1\"\n}\n"
	out := checkRoundTrip(t, "template.vs", src)
	if !strings.Contains(out, "db.sql\"select * from users where name = ${name} and age > ${age}\"") {
		t.Errorf("the literal was changed:\n%s", out)
	}
}

// D113: `const fun` prints as written, beside the other modifiers.
func TestConstFunRoundTrips(t *testing.T) {
	src := "const KB: i64 = 1024\n\nconst fun square(n: i64): i64 => n * n\n\npublic const fun cube(n: i64): i64 => n * n * n\n\n" +
		"struct Counter {\n  var n: i64\n\n  const fun bump(by: i64) {\n    this.n += by\n  }\n\n  public static const fun zero(): Counter => Counter(n: 0)\n}\n"
	if got := checkRoundTrip(t, "constfun.vs", src); got != src {
		t.Errorf("formatted:\n%s\nwant:\n%s", got, src)
	}
}
