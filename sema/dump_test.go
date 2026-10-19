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

var updateDump = flag.Bool("update-dump", false, "rewrite testdata/dump/*/main.{names,types,hir}")

func dumpOf(t *testing.T, dir, kind string, withStd bool) string {
	t.Helper()
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackage(dir, diags)
	if err != nil {
		t.Fatal(err)
	}
	prog := sema.Check(pkg, diags, false)
	if diags.HasErrors() || prog == nil {
		t.Fatalf("%s does not check:\n%s", dir, diags.Render())
	}
	var buf bytes.Buffer
	if err := sema.DumpProgram(&buf, prog, kind, withStd); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// TestDumps compares the three dumps (plan S10) of each fixture with the
// golden files next to it. They are what a checker written in Veles is gated
// against; `go test ./sema -run TestDumps -update-dump` rewrites them.
func TestDumps(t *testing.T) {
	dirs, _ := filepath.Glob(filepath.Join("testdata", "dump", "*"))
	if len(dirs) == 0 {
		t.Fatal("no dump fixtures")
	}
	for _, dir := range dirs {
		for _, kind := range sema.DumpKinds {
			t.Run(filepath.Base(dir)+"/"+kind, func(t *testing.T) {
				got := dumpOf(t, dir, kind, false)
				golden := filepath.Join(dir, "main."+kind)
				if *updateDump {
					if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("%v (run with -update-dump)", err)
				}
				// golden files are compared as written: LF, whatever the checkout did
				if w := strings.ReplaceAll(string(want), "\r\n", "\n"); got != w {
					t.Errorf("%s differs from %s (run with -update-dump to accept):\n%s", kind, golden, firstDifference(w, got))
				}
			})
		}
	}
}

// TestDumpsAreDeterministic checks a fixture twice in one process: nothing in
// a dump may depend on map order, pointer values or a variable counter that
// another check moved.
func TestDumpsAreDeterministic(t *testing.T) {
	dir := filepath.Join("testdata", "dump", "effects")
	for _, kind := range sema.DumpKinds {
		a := dumpOf(t, dir, kind, false)
		b := dumpOf(t, dir, kind, false)
		if a != b {
			t.Errorf("%s dump differs between two runs:\n%s", kind, firstDifference(a, b))
		}
	}
}

// TestDumpStd: the standard library's symbols appear only when asked for.
func TestDumpStd(t *testing.T) {
	dir := filepath.Join("testdata", "dump", "basics")
	own := dumpOf(t, dir, sema.DumpNames, false)
	all := dumpOf(t, dir, sema.DumpNames, true)
	if strings.Contains(own, "v_std.") {
		t.Errorf("the package's own dump holds a std symbol:\n%s", own)
	}
	if !strings.Contains(all, "v_std.io.println") || len(all) <= len(own) {
		t.Errorf("the dump with std lacks std symbols")
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
