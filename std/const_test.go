package std

import (
	"flag"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/const-functions.txt")

var constFun = regexp.MustCompile(`\bconst fun\s`)

// TestConstFunctions lists every std function a constant may call (D113:
// declared `const fun`) and compares the list with
// testdata/const-functions.txt, so a mark that is dropped — the function then
// silently stops working in constants — or added shows up in review. After a
// deliberate change: go test ./std -run ConstFunctions -update.
func TestConstFunctions(t *testing.T) {
	var lines []string
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".vs") || strings.HasSuffix(path, ".test.vs") {
			return err
		}
		data, err := FS.ReadFile(path)
		if err != nil {
			return err
		}
		for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "//") || !constFun.MatchString(l) {
				continue
			}
			lines = append(lines, path+": "+signature(l))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"
	const golden = "testdata/const-functions.txt"
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (create it with -update)", err)
	}
	if w := strings.ReplaceAll(string(want), "\r\n", "\n"); w != got {
		t.Errorf("std's const functions changed; if that is meant, run: go test ./std -run ConstFunctions -update\n%s", lineDiff(w, got))
	}
}

// signature is a declaration line up to its body: the first ` = ` or ` {`
// outside parentheses and brackets (a default argument has one inside).
func signature(l string) string {
	depth := 0
	for i := 0; i < len(l); i++ {
		switch l[i] {
		case '(', '[', '<':
			depth++
		case ')', ']', '>':
			depth--
		case ' ':
			if depth == 0 && (strings.HasPrefix(l[i:], " = ") || strings.HasPrefix(l[i:], " {")) {
				return l[:i]
			}
		}
	}
	return strings.TrimSuffix(strings.TrimSpace(l), " =")
}

// lineDiff lists the lines only in want (-) and only in got (+).
func lineDiff(want, got string) string {
	in := func(s string) map[string]bool {
		m := map[string]bool{}
		for _, l := range strings.Split(s, "\n") {
			m[l] = true
		}
		return m
	}
	w, g := in(want), in(got)
	var out []string
	for l := range w {
		if !g[l] {
			out = append(out, "- "+l)
		}
	}
	for l := range g {
		if !w[l] {
			out = append(out, "+ "+l)
		}
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}
