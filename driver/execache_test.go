package driver

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// `veles test` links a program once: running it again, with another
// `--filter` too, runs the program the cache kept. `veles build` always
// links: the caller wants the file.
func TestProgramCacheSkipsTheLink(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := t.TempDir()
	// a name no earlier run used, so the first run links
	fresh := strconv.FormatInt(time.Now().UnixNano(), 36)
	src := "test \"one " + fresh + "\" {\n  expect(1 + 1 == 2)\n}\n\ntest \"two\" {\n  expect(\"a\".len() == 1)\n}\n\nfun main() { }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	const linked, cached = "compile the IR and link", "skipped: this program was linked before (cached)"
	runs := []struct {
		filter, timing, summary string
	}{
		{"", linked, "2 passed, 0 failed\n"},
		{"", cached, "2 passed, 0 failed\n"},
		{"two", cached, "1 passed, 0 failed; 1 filtered out\n"},
	}
	for i, r := range runs {
		code, out, errs := capture(t, func() int {
			return Run(Options{Path: dir, Mode: "test", Timings: true, Filter: r.filter, Sanitize: *sanitize})
		})
		if code != 0 || !strings.HasSuffix(out, r.summary) {
			t.Fatalf("run %d: exit %d, output:\n%s", i+1, code, out)
		}
		if !strings.Contains(errs, r.timing) {
			t.Errorf("run %d: timings without %q:\n%s", i+1, r.timing, errs)
		}
	}
	exe := filepath.Join(t.TempDir(), "built")
	for i := 0; i < 2; i++ {
		code, _, errs := capture(t, func() int {
			return Run(Options{Path: dir, Mode: "build", Output: exe, Timings: true})
		})
		if code != 0 || !strings.Contains(errs, linked) {
			t.Fatalf("build %d: exit %d, timings:\n%s", i+1, code, errs)
		}
	}
}

func TestExeCacheKeyCoversWhatTheLinkReads(t *testing.T) {
	base := exeCacheKey("ir", "clang", []string{"a.o", "-O0"}, "main")
	for name, k := range map[string]string{
		"ir":                  exeCacheKey("ir2", "clang", []string{"a.o", "-O0"}, "main"),
		"clang":               exeCacheKey("ir", "clang2", []string{"a.o", "-O0"}, "main"),
		"arguments":           exeCacheKey("ir", "clang", []string{"a.o", "-O2"}, "main"),
		"argument boundaries": exeCacheKey("ir", "clang", []string{"a.o-", "O0"}, "main"),
		"name":                exeCacheKey("ir", "clang", []string{"a.o", "-O0"}, "app"),
	} {
		if k == base {
			t.Errorf("another %s gives the same key", name)
		}
	}
	if exeCacheKey("ir", "clang", []string{"a.o", "-O0"}, "main") != base {
		t.Error("the same link gives another key")
	}
}

func TestExeCacheStoreAndLookup(t *testing.T) {
	root := t.TempDir()
	if p := exeCacheLookup(root, "k", "prog"); p != "" {
		t.Fatalf("an empty cache found %s", p)
	}
	exe := filepath.Join(t.TempDir(), "prog")
	os.WriteFile(exe, []byte("program"), 0o755)
	exeCacheStore(root, "k", "prog", exe)
	exeCacheStore(root, "k", "prog", exe) // a second store of one key keeps the first
	p := exeCacheLookup(root, "k", "prog")
	if data, err := os.ReadFile(p); err != nil || string(data) != "program" {
		t.Fatalf("lookup gave %q: %q, %v", p, data, err)
	}
	if p := exeCacheLookup(root, "k", "other"); p != "" {
		t.Errorf("another name found %s", p)
	}
	ents, _ := os.ReadDir(root)
	if len(ents) != 1 {
		t.Errorf("the cache holds %d entries, want 1 (no store left a temporary)", len(ents))
	}
}

// Pruning keeps the entries used last, whatever order they were made in,
// and clears the temporaries of stores that died, once they are old.
func TestExeCachePruneKeepsTheLastUsed(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	at := func(name string, ago time.Duration) {
		p := filepath.Join(root, name)
		os.MkdirAll(p, 0o755)
		os.Chtimes(p, now.Add(-ago), now.Add(-ago))
	}
	at("a", 5*time.Minute)
	at("b", 1*time.Minute)
	at("c", 4*time.Minute)
	at("d", 2*time.Minute)
	at("tmp-old", 2*time.Hour)
	at("tmp-new", time.Minute)
	// "c" was made long ago, then used: lookup marks it
	os.WriteFile(filepath.Join(root, "c", "prog"), []byte("x"), 0o755)
	os.Chtimes(filepath.Join(root, "c"), now.Add(-4*time.Minute), now.Add(-4*time.Minute))
	if exeCacheLookup(root, "c", "prog") == "" {
		t.Fatal("lookup missed c")
	}
	exeCachePrune(root, 2)
	var left []string
	ents, _ := os.ReadDir(root)
	for _, e := range ents {
		left = append(left, e.Name())
	}
	if got := strings.Join(left, " "); got != "b c tmp-new" {
		t.Errorf("left %q, want \"b c tmp-new\"", got)
	}
}
