// Package selfhost holds the Veles rewrite of the front end
// (veles-selfhost-frontend-plan.md) and this harness, which is its only
// golden file: the Veles program, built with the Go compiler, is run on every
// file of the corpus, and what it prints must equal, byte for byte, what the
// Go front end computes for the same file in-process. Nothing here is
// updated by hand; when the two disagree, one of them is wrong.
package selfhost

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/internal/buildtest"
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// TestP0Source is gate P0: positions, lines and rendered diagnostics agree
// on every corpus file.
func TestP0Source(t *testing.T) {
	exe := buildSelfhost(t)
	for _, file := range corpus(t) {
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			f := source.NewFile(file.path, file.content)
			compare(t, exe, "--positions", file.path, goPositions(f))
			compare(t, exe, "--lines", file.path, goLines(f))
			compare(t, exe, "--render", file.path, goRender(f))
		})
	}
}

// buildSelfhost builds selfhost/ with a freshly built compiler.
func buildSelfhost(t *testing.T) string {
	t.Helper()
	veles := buildtest.Compiler(t)
	exe := filepath.Join(t.TempDir(), "selfhost")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	build := exec.Command(veles, "build", ".", "-o", exe, "--release")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building selfhost: %v\n%s", err, out)
	}
	return exe
}

type corpusFile struct {
	name, path, content string
	// removed: the file uses a spelling Veles has removed (removedSpellings)
	removed bool
}

// removedSpellings are in the Go parser's reports of the spellings Veles has
// removed: `use m.{ }`, `mut fun`, `fun <T> f`, `impl`, the receiver `self`,
// `T::Item` and the `{ e => }` handlers. The Go parser reads them so that
// `veles check --fix` migrates an old program; the self-hosted parser reads
// the current language only (user, 2026-10-08). A file that uses one is
// outside gates G1–G3, and TestRemovedSpellings checks that the Veles side
// rejects it instead.
var removedSpellings = []string{
	".{ … }' is spelled 'use ",
	"'mut fun' no longer exists",
	"type parameters follow the function name",
	"'impl' is spelled 'implement'",
	"the receiver is spelled 'this'",
	"'::' is not Veles",
	"the error is named in a head",
	"a handler that sees the error is",
}

// usesRemovedSpelling reports whether the Go parser finds a removed spelling
// in the file.
func usesRemovedSpelling(path, content string) bool {
	var diags source.Diagnostics
	parser.ParseFile(source.NewFile(path, content), &diags)
	for _, d := range diags.Items {
		if removedSpellingIn(d.Message) != "" {
			return true
		}
	}
	return false
}

// removedSpellingIn is the entry of removedSpellings in the message, or "".
func removedSpellingIn(message string) string {
	for _, spelling := range removedSpellings {
		if strings.Contains(message, spelling) {
			return spelling
		}
	}
	return ""
}

// corpus is every Veles source in the repository — std, examples, bench,
// the docs' programs' directories, the conformance cases (wrong on purpose),
// this front end itself — and files written for the edges: empty, CRLF,
// lone `\r`, no final newline, multibyte characters at line ends, ten
// thousand lines.
func corpus(t *testing.T) []corpusFile {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var files []corpusFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".vs") && !strings.HasSuffix(name, ".vss") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(data) {
			return nil // a Veles string is UTF-8; such a file comes with the lexer (P1)
		}
		rel, _ := filepath.Rel(root, path)
		files = append(files, corpusFile{name: filepath.ToSlash(rel), path: path, content: string(data), removed: usesRemovedSpelling(path, string(data))})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 100 {
		t.Fatalf("the corpus has %d files; is the working directory selfhost/?", len(files))
	}
	var long strings.Builder
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(&long, "val x%d = %d\n", i, i)
	}
	edges := map[string]string{
		"empty":         "",
		"newline":       "\n",
		"one-byte":      "a",
		"crlf":          "fun main() {\r\n  io.println(\"hi\")\r\n}\r\n",
		"crlf-no-final": "a\r\nb",
		"lone-cr":       "a\rb\r\r\n\r",
		"blank-lines":   "\n\n\n",
		"multibyte":     "é\n€€\n𝄞\r\n// ünïcödé ✓",
		"tabs":          "\tfun\t main() {\n\t\t}\n",
		"long":          long.String(),
		// what the lexer must get wrong the same way
		"unterminated-string":   "val s = \"abc\nval t = 1\n",
		"unterminated-block":    "val a = 1 /* not closed\n/* nested /* */\n",
		"unterminated-interp":   "val s = \"a ${b + \"c\" \n",
		"interp-nested":         "val s = \"${ m.get(\"{\") ?: \"}\" } and $name and $ alone and ${x}${y}\"\n",
		"escapes":               "val s = \"\\q \\u{zz} \\u{} \\u{110000} \\u{D800} \\u{FFFFFFFFF} \\u{1F600} \\u 9 \\0\\$\\'\"\n",
		"escape-multibyte":      "val s = \"\\é\"\nval c = '\\é'\n",
		"chars":                 "val a = 'ab'\nval b = '\nval c = 'é'\nval d = '\\n'\nval e = '",
		"numbers":               "val a = 12abc\nval b = 0x\nval c = 1e\nval d = 1.e5\nval e = pair.0.1\nval f = 0b102\nval g = 1_000.5e-3\nval h = 0o78\nval i = 1..5\nval j = 1...2\n",
		"strange-characters":    "val a = 1 # 2\nval b\u00a0= 3\nval c = 4\u200b\nval d = \u202e5\n// comment \u202e here\nval e = \"\u2066x\"\nval f = 1\ufeff\nval g = 1\x01 + 2\x7f\n",
		"bom":                   "\ufeff/// the module\n\nfun main() {}\n",
		"docs": "/// first\n/// second\n\n/// detached\n\n/** block\n * doc\n */\nfun a() {}\n////not doc\n/**/\n/***/\nfun b() {}\n/// next to\nval x = 1\n/** one line */ val y = 2\n",
		"chains":                "val x = a\n  .b()\n  // a comment\n  ?.c\n  ?: d\n  ?! e\n  ?? f\n  /* block */ .g\nval r = 1\n..2\nval t: List<List<i64>>\nval u = m?\n",
		"nesting":               "f(a,\n  b)\nxs[1,\n2]\n{\n  x\n}\nfoo(]\n)\n}\n",
		// what the parser must recover from the same way: one per path
		"p-decls": "use\nuse a { }\nuse a { * }\nuse a.b as\npublic internal fun f() {}\nprivate val x = 1\n" +
			"extend X for Y {}\nerror E = A | B\nenum E<T> { A, B = 2, 3 }\n" +
			"test fun helper() = 1\nsuite \"\" { val x = 1 }\ntest \"${x}\" { }\ntest \"t\" ()\nstatic assert(x)\n" +
			"extern \"Go\" { fun f() }\nextern union U<T> { a: i32 }\nwith x = f()\n@attr static assert(true, \"y\")\n" +
			"public suite \"s\" { }\nstruct S { x: , init(a) { } public init { } init { } static var y = 1 private static val z = 2 }\n" +
			"trait T { val x: i64 }\nimplement T for { }\nextern \"C\" fun g() \nfun h(...) {}\nfun k(a: i64... = []) {}\n",
		"p-signatures": "fun f(a: i64, lazy b: fun(): i64, c: *raw u8, d: (*i64)?, e: extern fun(i32): i32, f: sendable fun() suspends throws E, g: Self.Item, i: List<List<i64>>, j: Array<u8, 4 * 16>, k: Result<i64, A | b.B>, l: (), m: (i64,), n: T??) {}\n" +
			"fun g(): i64 throws A, B, C {}\nfun h() throws suspends {}\nfun i<const N: i64, const M: u8, T: A + B>() {}\noverride static fun k() {}\n" +
			"fun l(x: i64 = 1, ...) {}\nfun m() = \nfun n()\nfun o() { } fun p() { }\n",
		"p-removed": "use a.{ b }\nimpl Foo for Bar {}\nfun <T> f<U>() {}\nstruct S { mut fun j() {} }\nfun g(h: T::Item) {}\n" +
			"fun k() {\n  self.x\n}\nfun m() {\n  x ?? { e => 1 }\n}\nfun n() {\n  x catch { e => 1 }\n}\n",
		"p-statements": "fun f() {\n  val = 3\n  var x: i64\n  val y\n  val [a, b] = xs\n  val (a, [b]) = p else { return }\n  val Some(v) = o else return\n" +
			"  val (q, r) = t\n  val (s) = u\n  loop :outer (x in y) { break outer; continue outer }\n  for (x in y) { }\n  loop (x = 1) { }\n" +
			"  with a = f(), b = g()\n  with (a = f(), _ = g(), h()) { }\n  if (x) with y = z\n  scope { }\n  struct S {}\n  error E {}\n  extend X {}\n" +
			"  return )\n  throw\n  x += 1; x -= 2; x *= 3; x /= 4; x %= 5\n  ) stray\n  a -> b\n  fun inner() = 1\n}\n",
		"p-expressions": "fun f() {\n  x.0.1\n  0.0 .0\n  f<T>(x) < y\n  a >> b < c\n  a < b > c\n  x ?? { 1 }\n  this.x\n" +
			"  async x\n  async f()\n  try a.b() catch (e) { }\n  try x ?! y\n  do { } while\n  do { }\n  catch { }\n  x catch { 1 }\n" +
			"  f(a: 1, b...)\n  f(,)\n  (a, b) => a\n  (a: i64, (b, c)): i64 => { a }\n  _ => 1\n  x => y += 1\n  mut [1]\n  mut x\n  [:]\n  [1: 2, 3: 4,]\n  [1, 2,]\n" +
			"  if (x = 1) a else b\n  if (y = f() && z) { }\n  when (val r = x) { is Ok(v) => v\n is Err => 0 }\n  when (r = x) { else => 1 }\n" +
			"  when { a > b => 1 a => 2 }\n  when (x) { 1, 2 -> 3\n in 1..5 => 4\n -1 => 5\n Name => 6\n a.B(x, y: z) => 7\n is T() => 8\n [a, ..rest] => 9\n [.., _] => 10\n (a, b) if a > b => 11 }\n" +
			"  race { val v = ch.recv() => v\n sleep(d) -> 0\n x }\n  gather { }\n  unsafe { }\n  scope\n  fun\n  x is T(a) && y !is U\n  T implements Display\n  x as T\n" +
			"  sql\"a ${b}\"\n  sql \"a\"\n  db.sql\"x\"\n  \"${a b}\"\n  \"${}\"\n  1..<2\n  a ?: return\n  !(return)\n  f(x ?: return, y)\n  ~a\n  *p\n  &x\n  -1\n}\n",
	}
	// the fuzzers' findings that are text (a Veles string is always UTF-8)
	found, _ := filepath.Glob(filepath.Join(root, "*", "testdata", "fuzz", "*", "*"))
	for _, path := range found {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "string(") || !strings.HasSuffix(line, ")") {
				continue
			}
			text, err := strconv.Unquote(line[len("string(") : len(line)-1])
			if err == nil && utf8.ValidString(text) {
				edges["fuzz-"+filepath.Base(filepath.Dir(path))+"-"+filepath.Base(path)+"-"+strconv.Itoa(i)] = text
			}
		}
	}
	dir := t.TempDir()
	for name, content := range edges {
		path := filepath.Join(dir, name+".vs")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, corpusFile{name: "edge/" + name, path: path, content: content, removed: usesRemovedSpelling(path, content)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files
}

// compare runs the Veles side and reports the first line that differs.
func compare(t *testing.T, exe, mode, path, want string) {
	t.Helper()
	cmd := exec.Command(exe, mode, path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s: %v\n%s", mode, err, stderr.String())
	}
	got := stdout.String()
	if got == want {
		return
	}
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			t.Fatalf("%s: line %d differs\nveles: %q\ngo:    %q", mode, i+1, gl, wl)
		}
	}
}

// TestP1Tokens is gate G1: the self-hosted lexer's tokens, comments, module
// documentation and diagnostics agree with lexer.TokenizeAll's.
func TestP1Tokens(t *testing.T) {
	exe := buildSelfhost(t)
	for _, file := range corpus(t) {
		if file.removed {
			continue // `self` is a keyword only in Go's lexer: TestRemovedSpellings
		}
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			compare(t, exe, "--tokens", file.path, goTokens(source.NewFile(file.path, file.content)))
		})
	}
}

func goTokens(f *source.File) string {
	var diags source.Diagnostics
	toks, doc, comments := lexer.TokenizeAll(f, &diags)
	var out strings.Builder
	for _, t := range toks {
		auto := 0
		if t.AutoSemi {
			auto = 1
		}
		fmt.Fprintf(&out, "tok %d %s %d %d %d %s\n", int(t.Kind), t.Kind, t.Span.Start, t.Span.End, auto, escaped(t.Text))
		if t.Doc != "" {
			fmt.Fprintf(&out, "  doc %s\n", escaped(t.Doc))
		}
		for _, p := range t.Parts {
			if p.IsExpr {
				fmt.Fprintf(&out, "  expr %d %d %s\n", p.Span.Start, p.Span.End, escaped(p.Expr))
			} else {
				fmt.Fprintf(&out, "  text %s\n", escaped(p.Text))
			}
		}
	}
	for _, c := range comments {
		fmt.Fprintf(&out, "comment %d %d %s\n", c.Span.Start, c.Span.End, escaped(c.Text))
	}
	fmt.Fprintf(&out, "moduledoc %s\n", escaped(doc))
	for _, d := range diags.Items {
		fmt.Fprintf(&out, "diag %s\n", escaped(d.String()))
	}
	return out.String()
}

// escaped is selfhost/main.vs's `escaped`: the backslash, line breaks, tabs
// and other control bytes escaped. A Veles string is always UTF-8, so a
// byte that is not (Go's lexer can leave one in a token after `"\é"`) is
// printed as U+FFFD, as the Veles lexer stores it.
func escaped(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			out.WriteString("�")
		case r == '\\':
			out.WriteString(`\\`)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == '\t':
			out.WriteString(`\t`)
		case r < 0x20 || r == 0x7F:
			fmt.Fprintf(&out, `\x%02x`, r)
		default:
			out.WriteString(text[i : i+size])
		}
		i += size
	}
	return out.String()
}

// TestP3Quote checks the tree printer's `%q`: every line of every corpus
// file quoted as Go's strconv.Quote does.
func TestP3Quote(t *testing.T) {
	exe := buildSelfhost(t)
	for _, file := range corpus(t) {
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			f := source.NewFile(file.path, file.content)
			var want strings.Builder
			for n := 1; n <= strings.Count(file.content, "\n")+1; n++ {
				want.WriteString(strconv.Quote(f.Line(n)) + "\n")
			}
			compare(t, exe, "--quote", file.path, want.String())
		})
	}
}

// TestP5Parse is gates G2 and G3: the self-hosted parser's tree, as
// `veles parse` prints it, and every diagnostic of the parse, agree with the
// Go parser's on every corpus file in the current language.
func TestP5Parse(t *testing.T) {
	exe := buildSelfhost(t)
	for _, file := range corpus(t) {
		if file.removed {
			continue // outside the gates: TestRemovedSpellings
		}
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			var diags source.Diagnostics
			tree := parser.ParseFile(source.NewFile(file.path, file.content), &diags)
			var want strings.Builder
			want.WriteString(ast.Dump(tree))
			for _, d := range diags.Items {
				want.WriteString("diag " + escaped(d.String()) + "\n")
			}
			compare(t, exe, "--parse", file.path, want.String())
		})
	}
}

// TestRemovedSpellings covers the files the gates leave out: the
// self-hosted parser reports an error in each, where the Go parser reports
// the removed spelling, and edge/p-removed holds every spelling of
// removedSpellings, so the list cannot fall behind the Go parser's messages.
func TestRemovedSpellings(t *testing.T) {
	exe := buildSelfhost(t)
	seen := map[string]bool{}
	for _, file := range corpus(t) {
		if !file.removed {
			continue
		}
		if file.name == "edge/p-removed" {
			var diags source.Diagnostics
			parser.ParseFile(source.NewFile(file.path, file.content), &diags)
			for _, d := range diags.Items {
				seen[removedSpellingIn(d.Message)] = true
			}
		}
		file := file
		t.Run(file.name, func(t *testing.T) {
			t.Parallel()
			out, err := exec.Command(exe, "--parse", file.path).Output()
			if err != nil {
				t.Fatalf("--parse: %v", err)
			}
			if !strings.Contains(string(out), ": error: ") {
				t.Errorf("the self-hosted parser accepts a removed spelling:\n%s", out)
			}
		})
	}
	for _, spelling := range removedSpellings {
		if !seen[spelling] {
			t.Errorf("edge/p-removed does not make the Go parser report %q", spelling)
		}
	}
}

func goPositions(f *source.File) string {
	var out strings.Builder
	for offset := 0; offset <= len(f.Content); offset++ {
		line, col := f.Position(offset)
		out.WriteString(strconv.Itoa(line) + ":" + strconv.Itoa(col) + "\n")
	}
	return out.String()
}

func goLines(f *source.File) string {
	count := strings.Count(f.Content, "\n") + 1
	var out strings.Builder
	fmt.Fprintf(&out, "%d\n", count)
	for n := 0; n <= count+1; n++ {
		fmt.Fprintf(&out, "%d|%s\n", n, f.Line(n))
	}
	return out.String()
}

func goRender(f *source.File) string {
	var diags source.Diagnostics
	for offset, k := 0, 0; offset <= len(f.Content); offset, k = offset+37, k+1 {
		span := source.Span{File: f, Start: offset, End: offset + k%5}
		if k%2 == 0 {
			diags.Errorf(span, "probe %d", k)
		} else {
			diags.Warnf(span, "probe %d", k)
		}
	}
	return diags.Render()
}
