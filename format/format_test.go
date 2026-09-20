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
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".vs") {
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
			checkRoundTrip(t, path, string(data))
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
			"fun f(a: i64): i64 = ((a)) + (a * 2) - (a + 1) * (-(a))\n",
			"fun f(a: i64): i64 = (a) + (a * 2) - (a + 1) * (-(a))\n"},
		{"required parentheses added",
			"fun f(a: i64): i64 = (a + 1) * 2 - (try g()).len()\nfun h(a: i64?): i64 = a ?: throw E()\n",
			"fun f(a: i64): i64 = (a + 1) * 2 - (try g()).len()\nfun h(a: i64?): i64 = a ?: throw E()\n"},
		{"or-fail binds under try",
			"fun f(): i64 throws E = try g() ?! E()\nfun h(): i64 throws E = (try g()) ?! E()\nfun k(): i64 throws E = try (g() ?! E())\nfun m(): i64 = with (r = open()) { r.n }\n",
			"fun f(): i64 throws E = try g() ?! E()\nfun h(): i64 throws E = (try g()) ?! E()\nfun k(): i64 throws E = try (g() ?! E())\nfun m(): i64 = with (r = open()) {\n  r.n\n}\n"},
		{"nested unary keeps parens",
			"fun f(a: i64): i64 = -(-a) + !(!b)\n",
			"fun f(a: i64): i64 = -(-a) + !(!b)\n"},
		{"postfix on operators",
			"fun f(x: f64): f64 = (-x).abs() + (x as f64).floor() + (try g()).len()\n",
			"fun f(x: f64): f64 = (-x).abs() + (x as f64).floor() + (try g()).len()\n"},
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
			"val xs = [\n  1,\n  2]\nfun f() = g(\n  a,\n  b: 2\n)\n",
			"val xs = [\n  1,\n  2,\n]\nfun f() = g(\n  a,\n  b: 2,\n)\n"},
		{"method chains keep their breaks",
			"fun f(xs: List<i64>): i64 = xs\n    .map(x => x + 1)\n  .len()\n",
			"fun f(xs: List<i64>): i64 = xs\n  .map(x => x + 1)\n  .len()\n"},
		{"elvis continuation",
			"fun f(a: i64?, b: i64?): i64 {\n  val z = a\n      ?: b\n      ?: 0\n  z\n}\n",
			"fun f(a: i64?, b: i64?): i64 {\n  val z = a\n    ?: b\n    ?: 0\n  z\n}\n"},
		{"if else if",
			"fun f(x: i64): string = if (x < 0) \"neg\" else if (x == 0) \"zero\" else \"pos\"\nfun g(x: i64) {\n  if (x > 0) {\n    a()\n  }\n  else {\n    b()\n  }\n}\n",
			"fun f(x: i64): string = if (x < 0) \"neg\" else if (x == 0) \"zero\" else \"pos\"\nfun g(x: i64) {\n  if (x > 0) {\n    a()\n  } else {\n    b()\n  }\n}\n"},
		{"when arms",
			"fun f(v: i64): string = when (v) {\n1 => \"one\"\n  2, 3   =>   \"few\"\n  in 4..9 => \"some\"\n  else => \"many\"\n}\n",
			"fun f(v: i64): string = when (v) {\n  1       => \"one\"\n  2, 3    => \"few\"\n  in 4..9 => \"some\"\n  else    => \"many\"\n}\n"},
		{"columns: comments and fields align, wide spreads do not",
			"struct Node {\n  value: T // v\n  left: *Tree<T> // l\n\n  right: *Tree<T>\n}\nval a = 1 // one\nval bbbb = 2 // two\nfun f(v: i64): string = when (v) {\n  1 => \"one\"\n  in 4..9 if v > 5 && v < 8 && v != 7 => \"some\"\n}\n",
			"struct Node {\n  value: T         // v\n  left:  *Tree<T>  // l\n\n  right: *Tree<T>\n}\nval a = 1     // one\nval bbbb = 2  // two\nfun f(v: i64): string = when (v) {\n  1 => \"one\"\n  in 4..9 if v > 5 && v < 8 && v != 7 => \"some\"\n}\n"},
		{"operator breaks stay on the author's side",
			"fun f(): i64 = a %\n  b + (c\n  ) - d\nfun g(): i64 = (a\n  % b)\n",
			"fun f(): i64 = a %\n  b + (c) - d\nfun g(): i64 = (a\n  % b)\n"},
		{"comments at chain breaks and in empty bodies",
			"fun f() {\n  x % [0]  // c1\n    .len()  // c2\n  when { // c3\n  }\n}\n",
			"fun f() {\n  x % [0]   // c1\n    .len()  // c2\n  when {\n    // c3\n  }\n}\n"},
		{"one-element tuple broken",
			"fun f() = (\n  0,\n)\n",
			"fun f() = (\n  0,\n)\n"},
		{"static, variadic, spread and when-val",
			"struct P { x: i64\n static fun of(xs: i64...): P = P(x: xs.len()) }\nfun f(s: string) = when (val r = P.of(1, [2]...)) { else => r }\n",
			"struct P {\n  x: i64\n  static fun of(xs: i64...): P = P(x: xs.len())\n}\nfun f(s: string) = when (val r = P.of(1, [2]...)) {\n  else => r\n}\n"},
		{"inline impls stay in the body",
			"struct P<T> { x: T\n  @inline impl Show { fun show(): string = \"p\" }\n  impl Iterator { type Item = T\n fun next(): T? = null } }\nimpl Other for P<i64> { fun o() { } }\n",
			"struct P<T> {\n  x: T\n  @inline\n  impl Show {\n    fun show(): string = \"p\"\n  }\n  impl Iterator {\n    type Item = T\n    fun next(): T? = null\n  }\n}\nimpl Other for P<i64> {\n  fun o() { }\n}\n"},
		{"patterns",
			"fun f(s: Shape): f64 = when (s) { is Circle(r) => r; is Rect(w, h: hh) if w > hh => w; Point => 0.0; null => 1.0 }\n",
			"fun f(s: Shape): f64 = when (s) {\n  is Circle(r)                => r\n  is Rect(w, h: hh) if w > hh => w\n  Point                       => 0.0\n  null                        => 1.0\n}\n"},
		{"lambdas",
			"val f = (a: i64, b: i64): i64 => a + b\nval g = x => x * 2\nval h = (x) => x\nfun k(xs: List<i64>) { xs.forEach(x => total += x); xs.map(x => { x }) }\n",
			"val f = (a: i64, b: i64): i64 => a + b\nval g = x => x * 2\nval h = (x) => x\nfun k(xs: List<i64>) {\n  xs.forEach(x => total += x)\n  xs.map(x => {\n    x\n  })\n}\n"},
		{"error declarations",
			"public error NotFound { key: string\n  fun message(): string = \"no $key\" }\nerror Failed { message: string }\nerror Set = NotFound | Failed\n",
			"public error NotFound {\n  key: string\n  fun message(): string = \"no $key\"\n}\nerror Failed {\n  message: string\n}\nerror Set = NotFound | Failed\n"},
		{"modifiers and generics keep author order",
			"public fun <T: Show> show(x: T): string = x.show()\nstruct S {\n  public fun bump() { }\n  override fun d(): string = \"\"\n}\n",
			"public fun <T: Show> show(x: T): string = x.show()\nstruct S {\n  public fun bump() { }\n  override fun d(): string = \"\"\n}\n"},
		{"var fields align with the bare ones",
			"struct C {\n  var n: i64 = 0\n  step: i64 = 1\n  private var hits:   i64 = 0\n  public var label: string = \"\"\n}\n",
			"struct C {\n  var n:            i64 = 0\n  step:             i64 = 1\n  private var hits: i64 = 0\n  public var label: string = \"\"\n}\n"},
		{"explicit val, internal and protected var are kept as written",
			"internal struct C {\n  val id: i64\n  internal protected var n: i64 = 0\n  public protected var m: i64 = 0\n  internal fun f() { }\n}\ninternal val k = 1\ninternal fun g() { }\n",
			"internal struct C {\n  val id:                   i64\n  internal protected var n: i64 = 0\n  public protected var m:   i64 = 0\n  internal fun f() { }\n}\ninternal val k = 1\ninternal fun g() { }\n"},
		{"arm body broken after the arrow leaves no trailing space",
			"fun f(v: i64): string = when (v) {\n  1 =>  \n    \"one\"\n  else => \"more\"\n}\n",
			"fun f(v: i64): string = when (v) {\n  1 =>\n    \"one\"\n  else => \"more\"\n}\n"},
		{"arrows align across a block arm; a nested when is its own run",
			"fun f(c: u8): i64 = when (c) {\n  '(' => 1\n  ')' => {\n    when (c) {\n      1 => 10\n      222 => 20\n    }\n  }\n  // between arms\n  '+' => 3\n  else => 0\n}\n",
			"fun f(c: u8): i64 = when (c) {\n  '('  => 1\n  ')'  => {\n    when (c) {\n      1   => 10\n      222 => 20\n    }\n  }\n  // between arms\n  '+'  => 3\n  else => 0\n}\n"},
		{"types",
			"fun f(a: (*T)?, b: *T?, c: fun(i64, string): bool suspends throws E, d: (A, B), e: I.Item, g: *raw u8): Map<string, List<i64>> throws A | B { }\n",
			"fun f(a: (*T)?, b: *T?, c: fun(i64, string): bool suspends throws E, d: (A, B), e: I.Item, g: *raw u8): Map<string, List<i64>> throws A | B { }\n"},
		{"attributes and extern",
			"@test\nfun t() { }\n@deprecated(\"use g\")\npublic fun f() { }\nextern \"C\" { fun strlen(s: *raw u8): i64\n  fun puts(s: string) }\n",
			"@test\nfun t() { }\n@deprecated(\"use g\")\npublic fun f() { }\nextern \"C\" {\n  fun strlen(s: *raw u8): i64\n  fun puts(s: string)\n}\n"},
		{"type aliases",
			"public type   Key=(i64,u64)\ntype StrMap<V>  =  Map<string,V>\n",
			"public type Key = (i64, u64)\ntype StrMap<V> = Map<string, V>\n"},
		{"use forms",
			"use io\nuse geometry   as   geo\nuse a.b as c\n",
			"use io\nuse a.b as c, geometry as geo\n"},
		{"use block: merged, sorted, std first, blank lines dropped",
			"use time\nuse shapes\nuse os,\n  io\n\nuse fs\n\nfun main() { }\n",
			"use fs, io, os, time\nuse shapes\n\nfun main() { }\n"},
		{"use block: a commented line stays as written, the rest is grouped",
			"use os  // needed\nuse io\nuse fs\n",
			"use os  // needed\nuse fs, io\n"},
		{"use block: comments above and below are not inside it",
			"// header\nuse os\nuse io\n// about main\nfun main() { }\n",
			"// header\nuse io, os\n// about main\nfun main() { }\n"},
		{"concurrency",
			"fun main() {\n  scope {\n    val t = async work(1)\n    val (a, b) = gather { async f(); async g() }\n    val w = race { val m = ch.recv() => m; sleep(100) => \"t\" }\n    with (f = open(\"a\")) { process(f) }\n  }\n}\n",
			"fun main() {\n  scope {\n    val t = async work(1)\n    val (a, b) = gather {\n      async f()\n      async g()\n    }\n    val w = race {\n      val m = ch.recv() => m\n      sleep(100)        => \"t\"\n    }\n    with (f = open(\"a\")) {\n      process(f)\n    }\n  }\n}\n"},
		{"trailing whitespace and CRLF",
			"use io\r\n\r\nfun main() {   \r\n  io.println(\"x\")  \r\n}\r\n",
			"use io\n\nfun main() {\n  io.println(\"x\")\n}\n"},
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
