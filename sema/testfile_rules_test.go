package sema

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/source"
)

// checkFiles checks a package made of the given files (name -> source).
func checkFiles(t *testing.T, files map[string]string) *source.Diagnostics {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	if !diags.HasErrors() {
		Check(pkg, diags, false)
	}
	return diags
}

func errorsOf(d *source.Diagnostics) string {
	var out []string
	for _, it := range d.Items {
		if it.Severity == source.Error {
			out = append(out, it.Message)
		}
	}
	return strings.Join(out, "\n")
}

// D78 (amended 2026-10-04): a test file holds tests and `test fun` helpers —
// a plain `fun` there is an error with a fix — and what it declares is
// invisible to the module's own files.
func TestTestFileFunctionsAreTestFuns(t *testing.T) {
	diags := checkFiles(t, map[string]string{
		"main.vs":      "fun main() { }\n",
		"main.test.vs": "fun helper(): i64 => 1\n\ntest fun fine(): i64 => 2\n\ntest \"uses both\" {\n  expect(helper() + fine() == 3)\n}\n",
	})
	got := errorsOf(diags)
	if !strings.Contains(got, "'helper' is a function in a *.test.vs file: write 'test fun helper'") {
		t.Fatalf("a plain fun in a test file was accepted:\n%s", got)
	}
	if strings.Contains(got, "'fine'") {
		t.Errorf("a 'test fun' was refused:\n%s", got)
	}
	var fix *source.Fix
	for _, it := range diags.Items {
		if strings.Contains(it.Message, "'helper'") {
			fix = it.Fix
		}
	}
	if fix == nil || len(fix.Edits) != 1 || fix.Edits[0].NewText != "test " {
		t.Errorf("no fix that writes 'test ' before fun: %+v", fix)
	}
}

func TestTestFileDeclarationsAreInvisibleToTheModule(t *testing.T) {
	const tests = "struct Fixture {\n  n: i64\n}\n\nval limit: i64 = 3\n\ntest \"a test sees them\" {\n  expect(Fixture(n: limit).n == 3)\n}\n"
	for name, main := range map[string]string{
		"'Fixture'": "fun make(): Fixture => Fixture(n: 1)\n\nfun main() { }\n",
		"'limit'":   "fun count(): i64 => limit\n\nfun main() { }\n",
	} {
		got := errorsOf(checkFiles(t, map[string]string{"main.vs": main, "main.test.vs": tests}))
		if !strings.Contains(got, name+" is declared in a *.test.vs file") {
			t.Errorf("%s is visible to the module's own file:\n%s", name, got)
		}
	}
}

func TestLibraryCheckSkipsForeignStdTests(t *testing.T) {
	for _, mode := range []string{"normal", "phase", "index"} {
		t.Run(mode, func(t *testing.T) {
			testLibraryCheckSkipsForeignStdTests(t, mode)
		})
	}
}

func testLibraryCheckSkipsForeignStdTests(t *testing.T, mode string) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{
		"lib.vs":      "public fun value(): i64 => 1\n",
		"lib.test.vs": "test \"own test is still checked\" { ownMissing() }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(dir, diags)
	if err != nil || diags.HasErrors() {
		t.Fatalf("load: %v\n%s", err, diags.Render())
	}
	pkg.NeedMain = false
	attached := false
	for _, module := range pkg.Modules {
		if module.Std && strings.HasSuffix(module.Path, "prelude") {
			file := source.NewFile("std/prelude/foreign.test.vs", "test \"foreign test must stay out\" { foreignMissing() }\n")
			module.Files = append(module.Files, parser.ParseFile(file, diags))
			attached = true
		}
	}
	if !attached {
		t.Fatal("prelude was not loaded")
	}
	switch mode {
	case "normal":
		Check(pkg, diags, false)
	case "phase":
		dump, err := NewPhaseDump(io.Discard, "effects", DumpOptions{})
		if err != nil {
			t.Fatal(err)
		}
		CheckWithDump(pkg, diags, false, false, dump)
	case "index":
		CheckIndex(pkg, diags)
	}
	got := errorsOf(diags)
	if strings.Contains(got, "foreignMissing") {
		t.Fatalf("library check included a foreign std test:\n%s", got)
	}
	if !strings.Contains(got, "ownMissing") {
		t.Fatalf("library check stopped checking its own test files:\n%s", got)
	}
}

func TestTestFileHelpersStayForTests(t *testing.T) {
	diags := checkFiles(t, map[string]string{
		"main.vs":      "fun main() { }\n",
		"main.test.vs": "test fun double(n: i64): i64 => n * 2\n\ntest \"doubles\" {\n  expect(double(2) == 4)\n}\n",
	})
	if got := errorsOf(diags); got != "" {
		t.Errorf("a test using its own helper failed:\n%s", got)
	}
}

// D129: a tag may be `module.name`, and only a public function of that module.
func TestTemplateTagQualifiedByModule(t *testing.T) {
	const lib = "public struct Query {\n  public text: string\n}\n\npublic template fun sql(parts: List<string>, values: List<i64>): Query => Query(text: parts.join(\"?\"))\n\ninternal template fun hidden(parts: List<string>, values: List<i64>): Query => Query(text: \"\")\n\npublic fun plain(parts: List<string>, values: List<i64>): Query => Query(text: \"\")\n"
	ok := checkFiles(t, map[string]string{
		"main.vs":   "use tags\n\nfun main() {\n  val q = tags.sql\"a ${1} b\"\n}\n",
		"tags/t.vs": lib,
	})
	if got := errorsOf(ok); got != "" {
		t.Errorf("a module-qualified template failed:\n%s", got)
	}
	for _, source := range []string{
		"use tags\nfun main() { tags.sql([\"runtime\", \"\"], [1]) }\n",
		"use tags\nfun main() { val alias = tags.sql }\n",
	} {
		diags := checkFiles(t, map[string]string{"main.vs": source, "tags/t.vs": lib})
		if got := errorsOf(diags); !strings.Contains(got, "tag-only") && !strings.Contains(got, "function value") {
			t.Fatalf("qualified template bypassed tag-only restriction: %s", got)
		}
	}
	private := checkFiles(t, map[string]string{
		"main.vs":   "use tags\n\nfun main() {\n  val q = tags.hidden\"x\"\n}\n",
		"tags/t.vs": lib,
	})
	if got := errorsOf(private); got == "" {
		t.Errorf("a private template function was usable from another module")
	}
	untagged := checkFiles(t, map[string]string{
		"main.vs":   "use tags\n\nfun main() {\n  val q = tags.plain\"x\"\n}\n",
		"tags/t.vs": lib,
	})
	if got := errorsOf(untagged); !strings.Contains(got, "'plain' is not a template") {
		t.Errorf("a plain function was taken as a template:\n%s", got)
	}
}
