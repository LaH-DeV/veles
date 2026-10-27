package llvm_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/codegen/llvm"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// allocCall matches a call of a runtime entry point that allocates on the
// heap itself, and the line-table entry it carries.
var allocCall = regexp.MustCompile(`call [^@]*@veles_(gc_alloc|alloc|alloc_words|list_new|map_new|string_concat\w*|task_launch|task_spawn|task_new|chan_new|race_new)\(.*!dbg !(\d+)`)

// anyCall matches any call that carries a line: a site whose own allocation
// happens inside a runtime or standard-library function shows as one.
var anyCall = regexp.MustCompile(`call [^@]*@[\w.$]+\(.*!dbg !(\d+)`)

var (
	dbgRef     = regexp.MustCompile(`(?m)^  .*, !dbg !(\d+)$`)
	coldCall   = regexp.MustCompile(`call [^@]*@veles_\w*(panic|fail)\w*\(`)
	defineName = regexp.MustCompile(`^define [^@]*@([\w.$"]+)\(`)
	blockLabel = regexp.MustCompile(`^[\w.$-]+:`)
	diLocation = regexp.MustCompile(`^!(\d+) = !DILocation\(line: (\d+), column: \d+, scope: !(\d+)\)`)
	diProgram  = regexp.MustCompile(`^!(\d+) = distinct !DISubprogram\(.*file: !(\d+),`)
	diFile     = regexp.MustCompile(`^!(\d+) = !DIFile\(filename: "([^"]*)", directory: "([^"]*)"`)
)

// TestAllocSitesMatchTheIR holds D213's promise that a `heap` hint is
// shown where the generated code allocates and nowhere else. For every
// example and benchmark it builds the program with a line table, then
// checks both ways, by the lines of each site's statement (the table's
// granularity): every call of an allocating runtime entry point in the
// package's own code is on a hinted line, but one on the way to a panic or
// a test's failure; and every site is on a line with a call, but a mutable
// collection's (its storage comes with the first write), a module value's
// (start-up code has no line table) and one in code never emitted.
func TestAllocSitesMatchTheIR(t *testing.T) {
	var dirs []string
	for _, root := range []string{"../../examples", "../../bench"} {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && d.Name() == "main.vs" {
				dirs = append(dirs, filepath.Dir(path))
			}
			return nil
		})
	}
	if len(dirs) < 20 {
		t.Fatalf("found %d programs", len(dirs))
	}
	for _, dir := range dirs {
		t.Run(filepath.ToSlash(dir), func(t *testing.T) {
			t.Parallel()
			diags := &source.Diagnostics{}
			pkg, err := sema.LoadPackage(dir, diags)
			if err != nil {
				t.Skip(err)
			}
			prog := sema.Check(pkg, diags, false)
			if prog == nil || diags.HasErrors() {
				t.Skip("does not check on its own") // a package of a workspace
			}
			prog.DebugInfo = true
			ir := llvm.Generate(prog)
			lines, files, scopes := map[string]int{}, map[string]string{}, map[string]string{}
			locScope := map[string]string{}
			for _, l := range strings.Split(ir, "\n") {
				if m := diLocation.FindStringSubmatch(l); m != nil {
					n, _ := strconv.Atoi(m[2])
					lines[m[1]], locScope[m[1]] = n, m[3]
				} else if m := diProgram.FindStringSubmatch(l); m != nil {
					scopes[m[1]] = m[2]
				} else if m := diFile.FindStringSubmatch(l); m != nil && m[3] != "<veles std>" {
					files[m[1]] = filepath.Base(m[2]) // the package's own; std's are not checked
				}
			}
			where := func(id string) string { // "file:line", or "" for std
				file := files[scopes[locScope[id]]]
				if file == "" || lines[id] == 0 {
					return ""
				}
				return file + ":" + strconv.Itoa(lines[id])
			}
			allocating, calling, emitted := map[string]string{}, map[string]bool{}, map[string]bool{}
			// a line is emitted when an instruction carries it: a function the
			// generator began and then dropped (one only the compiler runs)
			// leaves line-table entries that nothing refers to
			for _, m := range dbgRef.FindAllStringSubmatch(ir, -1) {
				emitted[where(m[1])] = true
			}
			// block by block: one that ends in a panic or a test's failure
			// builds its message on the way out, which is not a site
			var fn string
			var block []string
			flush := func() {
				cold := false
				for _, l := range block {
					cold = cold || coldCall.MatchString(l)
				}
				for _, l := range block {
					if m := allocCall.FindStringSubmatch(l); m != nil && !cold {
						call := strings.TrimSpace(l)
						if i := strings.Index(call, ", !dbg"); i > 0 {
							call = call[:i]
						}
						allocating[where(m[2])] = fn + ": " + call
					}
					if m := anyCall.FindStringSubmatch(l); m != nil {
						calling[where(m[1])] = true
					}
				}
				block = block[:0]
			}
			for _, l := range strings.Split(ir, "\n") {
				switch {
				case strings.HasPrefix(l, "define "):
					flush()
					if m := defineName.FindStringSubmatch(l); m != nil {
						fn = m[1]
					}
				case l == "}" || blockLabel.MatchString(l):
					flush()
				}
				block = append(block, l)
			}
			flush()
			hinted := map[string]bool{}
			at := func(sp source.Span) string {
				if !sp.IsValid() {
					return ""
				}
				line, _ := sp.File.Position(sp.Start)
				return filepath.Base(sp.File.Path) + ":" + strconv.Itoa(line)
			}
			// the lines of a site's statement: within one written over several
			// lines (a `when`), the line table places its parts on any of them
			linesOf := func(s sema.AllocSite) []string {
				out := []string{at(s.Span)}
				if s.Stmt.IsValid() {
					first, _ := s.Stmt.File.Position(s.Stmt.Start)
					last, _ := s.Stmt.File.Position(s.Stmt.End)
					for l := first; l <= last; l++ {
						out = append(out, filepath.Base(s.Stmt.File.Path)+":"+strconv.Itoa(l))
					}
				}
				return out
			}
			for _, s := range sema.AllocSites(prog) {
				called, shown := false, false
				for _, l := range linesOf(s) {
					hinted[l] = true
					called = called || calling[l]
					shown = shown || emitted[l]
				}
				// a mutable collection's storage comes later; a module value's
				// initializer and code never emitted (a function only constants
				// call, one only the compiler runs) carry no line to check
				if !s.Later && !s.Global && shown && !called {
					t.Errorf("%s: a site with no call there: %s", at(s.Span), s.What)
				}
			}
			for loc, call := range allocating {
				if loc != "" && !hinted[loc] {
					t.Errorf("%s: an allocation with no site (%s)", loc, call)
				}
			}
		})
	}
}
