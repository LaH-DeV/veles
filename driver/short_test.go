package driver

import (
	"strconv"
	"testing"
)

// `go test -short ./driver` is the quick check while working; the full run
// is the gate (CLAUDE.md, "Done means"). Under -short a test builds its
// program in one profile, runs it with one thread count, and the stress and
// soak tests do not run.

// profiles is the build profiles a test covers, as `release` flags: debug
// and release, or debug alone under -short.
func profiles() []bool {
	if testing.Short() {
		return []bool{false}
	}
	return []bool{false, true}
}

// threadCounts is the thread counts a test runs its program with: all of
// them, repeats included (a repeat gives a rare interleaving another
// chance), or under -short the largest once — the most threads is where
// the runtime's races are.
func threadCounts(counts ...string) []string {
	if !testing.Short() || len(counts) == 0 {
		return counts
	}
	most, mostN := counts[0], -1
	for _, c := range counts {
		if n, err := strconv.Atoi(c); err == nil && n > mostN {
			most, mostN = c, n
		}
	}
	return []string{most}
}

// skipInShort skips a stress or soak test under -short: what it looks for
// takes many rounds to show, and the full run keeps it.
func skipInShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("a stress test: skipped under -short")
	}
}

func TestThreadCountsUnderShort(t *testing.T) {
	got := threadCounts("1", "8", "2", "8")
	if testing.Short() {
		if len(got) != 1 || got[0] != "8" {
			t.Errorf("-short: %v, want [8]", got)
		}
		if p := profiles(); len(p) != 1 || p[0] {
			t.Errorf("-short: profiles %v, want [false]", p)
		}
		return
	}
	if len(got) != 4 {
		t.Errorf("full: %v, want every count", got)
	}
	if p := profiles(); len(p) != 2 {
		t.Errorf("full: profiles %v, want both", p)
	}
}
