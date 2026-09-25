package llvm_test

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/codegen/llvm"
	"github.com/LaH-DeV/veles/driver"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// -update rewrites the golden files (main.ll, output.txt) that differ.
var update = flag.Bool("update", false, "rewrite golden IR and output files")

// TestGolden lowers each golden/<name>/main.vs and compares the IR of the
// fixture's own module — its types, globals and functions, not the
// prelude's — with golden/<name>/main.ll. The golden file is a diff
// generator: a codegen change shows its effect as readable text, and an
// accidental one fails loudly. Correctness is checked separately: when clang
// is available each fixture is also built and run, and its output compared
// with golden/<name>/output.txt, so the IR is known to be valid LLVM that
// computes the right thing, not merely unchanged.
func TestGolden(t *testing.T) {
	mains, err := filepath.Glob(filepath.Join("golden", "*", "main.vs"))
	if err != nil || len(mains) == 0 {
		t.Fatalf("no golden fixtures: %v", err)
	}
	_, clangErr := driver.ClangPath()
	for _, main := range mains {
		dir := filepath.Dir(main)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			t.Parallel()
			diags := &source.Diagnostics{}
			pkg, err := sema.LoadPackage(dir, diags)
			if err != nil {
				t.Fatal(err)
			}
			pkg.NeedMain = true
			prog := sema.Check(pkg, diags, false)
			if len(diags.Items) > 0 || prog == nil {
				t.Fatalf("a fixture must check without diagnostics:\n%s", diags.Render())
			}
			compare(t, filepath.Join(dir, "main.ll"), ModuleIR(llvm.Generate(prog), "main"))

			if clangErr != nil {
				t.Skipf("not running: %v", clangErr)
			}
			exe := filepath.Join(t.TempDir(), "fixture")
			if runtime.GOOS == "windows" {
				exe += ".exe"
			}
			if code := driver.Run(driver.Options{Path: dir, Mode: "build", Output: exe}); code != 0 {
				t.Fatalf("veles build: exit %d", code)
			}
			out, err := exec.Command(exe).Output()
			if err != nil {
				t.Fatalf("running the fixture: %v", err)
			}
			compare(t, filepath.Join(dir, "output.txt"), strings.ReplaceAll(string(out), "\r\n", "\n"))
		})
	}
}

func compare(t *testing.T, path, got string) {
	t.Helper()
	want, _ := os.ReadFile(path)
	if got == strings.ReplaceAll(string(want), "\r\n", "\n") {
		return
	}
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", path)
		return
	}
	t.Errorf("%s differs (go test ./codegen/llvm -update to accept):\n%s", path, lineDiff(string(want), got))
}

var (
	// a top-level entity: a type (%S.main.X = type), a global (@x = ...),
	// or a function (define ... @x(...) {)
	typeLine   = regexp.MustCompile(`^%([^ ]+) = type `)
	globalLine = regexp.MustCompile(`^@([^ ]+) = `)
	defineLine = regexp.MustCompile(`^define [^@]*@([^(]+)\(`)
	strRef     = regexp.MustCompile(`@\.str\.[0-9]+\b`)
)

// ModuleIR keeps the top-level entities whose name mentions `module.` —
// the module's functions (`@v_main.f`), its types (`%S.main.T`), their
// descriptors, vtables and printing helpers — and drops everything the
// prelude and std contribute. String constants are numbered across the whole
// program, so they are renumbered in order of first use and their
// definitions appended: a prelude change cannot shift them.
func ModuleIR(ir, module string) string {
	mark := module + "."
	lines := strings.Split(ir, "\n")
	strDefs := map[string]string{}
	for _, l := range lines {
		if m := globalLine.FindStringSubmatch(l); m != nil && strings.HasPrefix(m[1], ".str.") {
			strDefs["@"+m[1]] = l[len(m[0]):]
		}
	}
	var kept []string
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		name := ""
		if m := typeLine.FindStringSubmatch(l); m != nil {
			name = m[1]
		} else if m := globalLine.FindStringSubmatch(l); m != nil {
			name = m[1]
		} else if m := defineLine.FindStringSubmatch(l); m != nil {
			name = m[1]
		}
		if name == "" || !strings.Contains(name, mark) {
			continue
		}
		kept = append(kept, l)
		if strings.HasPrefix(l, "define ") {
			for i+1 < len(lines) && lines[i] != "}" {
				i++
				kept = append(kept, lines[i])
			}
			kept = append(kept, "")
		}
	}
	renamed := map[string]string{}
	var order []string
	out := strRef.ReplaceAllStringFunc(strings.Join(kept, "\n"), func(ref string) string {
		if r, ok := renamed[ref]; ok {
			return r
		}
		r := "@.str." + strconv.Itoa(len(order)+1)
		renamed[ref] = r
		order = append(order, ref)
		return r
	})
	var b strings.Builder
	b.WriteString(strings.TrimRight(out, "\n"))
	b.WriteString("\n")
	if len(order) > 0 {
		b.WriteString("\n")
		for _, ref := range order {
			b.WriteString(renamed[ref] + " = " + strDefs[ref] + "\n")
		}
	}
	return b.String()
}

// lineDiff shows the first differing lines, enough to see what moved.
func lineDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	shown := 0
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, c string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			c = g[i]
		}
		if a == c {
			continue
		}
		b.WriteString("line " + strconv.Itoa(i+1) + ":\n  - " + a + "\n  + " + c + "\n")
		if shown++; shown == 12 {
			b.WriteString("  ...\n")
			break
		}
	}
	return b.String()
}
