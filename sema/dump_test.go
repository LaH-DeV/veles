package sema_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

var updateDump = flag.Bool("update-dump", false, "rewrite testdata/dump/*/main.*")

// dumpOf checks the package in dir and returns the dump of kind: one of the
// program dumps (after lowering) or of the phase dumps (between the passes).
func dumpOf(t *testing.T, dir, kind string, opts sema.DumpOptions) string {
	t.Helper()
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if sema.IsPhaseDump(kind) {
		d, err := sema.NewPhaseDump(&buf, kind, opts)
		if err != nil {
			t.Fatal(err)
		}
		if prog := sema.CheckWithDump(pkg, diags, false, false, d); diags.HasErrors() || prog == nil {
			t.Fatalf("%s does not check:\n%s", dir, diags.Render())
		}
		return buf.String()
	}
	prog := sema.Check(pkg, diags, false)
	if diags.HasErrors() || prog == nil {
		t.Fatalf("%s does not check:\n%s", dir, diags.Render())
	}
	if err := sema.DumpProgram(&buf, prog, kind, opts); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// allKinds is every dump there is: the program dumps (plan S10) and the phase
// dumps (S10b).
func allKinds() []string {
	return append(append([]string{}, sema.DumpKinds...), sema.PhaseKinds...)
}

// golden compares got with the file next to the fixture, or rewrites it.
func golden(t *testing.T, got, file, what string) {
	t.Helper()
	if *updateDump {
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v (run with -update-dump)", err)
	}
	// golden files are compared as written: LF, whatever the checkout did
	if w := strings.ReplaceAll(string(want), "\r\n", "\n"); got != w {
		t.Errorf("%s differs from %s (run with -update-dump to accept):\n%s", what, file, firstDifference(w, got))
	}
}

// TestDumps compares every dump of each fixture with the golden file next to
// it (`main.names`, `main.decls`, …), and with spans (`main.decls.spans`) for
// the dumps that carry them. They are what a checker written in Veles is gated
// against; `go test ./sema -run TestDumps -update-dump` rewrites them.
func TestDumps(t *testing.T) {
	dirs, _ := filepath.Glob(filepath.Join("testdata", "dump", "*"))
	if len(dirs) == 0 {
		t.Fatal("no dump fixtures")
	}
	for _, dir := range dirs {
		for _, kind := range allKinds() {
			t.Run(filepath.Base(dir)+"/"+kind, func(t *testing.T) {
				golden(t, dumpOf(t, dir, kind, sema.DumpOptions{}), filepath.Join(dir, "main."+kind), kind)
			})
			if kind == sema.DumpNames || kind == sema.DumpTypes || (kind == sema.DumpHIR && filepath.Base(dir) != "basics") {
				continue
			}
			t.Run(filepath.Base(dir)+"/"+kind+"+spans", func(t *testing.T) {
				golden(t, dumpOf(t, dir, kind, sema.DumpOptions{Spans: true}), filepath.Join(dir, "main."+kind+".spans"), kind+" with spans")
			})
		}
	}
}

// TestDumpsAreDeterministic checks a fixture twice in one process: nothing in
// a dump may depend on map order, pointer values or a variable counter that
// another check moved.
func TestDumpsAreDeterministic(t *testing.T) {
	for _, name := range []string{"effects", "imports"} {
		dir := filepath.Join("testdata", "dump", name)
		for _, kind := range allKinds() {
			a := dumpOf(t, dir, kind, sema.DumpOptions{Spans: true})
			b := dumpOf(t, dir, kind, sema.DumpOptions{Spans: true})
			if a != b {
				t.Errorf("%s dump of %s differs between two runs:\n%s", kind, name, firstDifference(a, b))
			}
		}
	}
}

// TestPhaseDumpsStopWhereTheyBelong: the scopes are written before any body is
// checked, so they are there even for a program that does not check.
func TestPhaseDumpsStopWhereTheyBelong(t *testing.T) {
	dir := t.TempDir()
	src := "use io\n\nfun helper(): i64 => \"not a number\"\n\nfun main() {\n  io.println(\"hi\")\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for kind, wantOutput := range map[string]bool{sema.DumpModules: true, sema.DumpScopes: true, sema.DumpDecls: false, sema.DumpEffects: false} {
		diags := &source.Diagnostics{}
		pkg, err := sema.LoadPackage(dir, diags)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		d, _ := sema.NewPhaseDump(&buf, kind, sema.DumpOptions{})
		sema.CheckWithDump(pkg, diags, false, false, d)
		if !diags.HasErrors() {
			t.Fatalf("the program must not check")
		}
		if (buf.Len() > 0) != wantOutput {
			t.Errorf("%s: wrote %d bytes for a program with an error in a body, want output %v", kind, buf.Len(), wantOutput)
		}
	}
}

// TestDumpStd: the standard library's symbols appear only when asked for.
func TestDumpStd(t *testing.T) {
	dir := filepath.Join("testdata", "dump", "basics")
	own := dumpOf(t, dir, sema.DumpNames, sema.DumpOptions{})
	all := dumpOf(t, dir, sema.DumpNames, sema.DumpOptions{Std: true})
	if strings.Contains(own, "v_std.") {
		t.Errorf("the package's own dump holds a std symbol:\n%s", own)
	}
	if !strings.Contains(all, "v_std.io.println") || len(all) <= len(own) {
		t.Errorf("the dump with std lacks std symbols")
	}
	for _, kind := range sema.PhaseKinds {
		own, all := dumpOf(t, dir, kind, sema.DumpOptions{}), dumpOf(t, dir, kind, sema.DumpOptions{Std: true})
		if strings.Contains(own, "std module") || strings.Contains(own, "module std/") || len(all) <= len(own) {
			t.Errorf("%s: std symbols must appear only with Std (own %d bytes, with std %d)", kind, len(own), len(all))
		}
	}
}

func firstDifference(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return "line " + itoa(i+1) + ":\n  want: " + w + "\n  got:  " + g
		}
	}
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
