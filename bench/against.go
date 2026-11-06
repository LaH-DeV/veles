package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// compareAgainst is `go run ./bench -against <rev>`: this tree's compiler
// against the one at a git revision, each building its own tree's version
// of every benchmark both trees have (a language change rewrites the
// programs: D231's did most of them), run `rounds` times; equal checksums
// say the two versions still do the same work. The two sides take
// turns going first, round by round: two runs back to back are not alike —
// on Windows the second of a pair ran some rows up to 40% slower whichever
// tree it was (2026-11-06) — so a fixed order would charge that to one side.
// Both builds must print the same checksum. No Go reference runs: the
// question is the change, not the ratio.
func compareAgainst(root, tmp, veles, rev, filter string, rounds int) {
	wt := filepath.Join(tmp, "against")
	run(root, "git", "worktree", "add", "--detach", wt, rev)
	// removed on every way out: os.Exit skips deferred calls
	cleanup := func() { exec.Command("git", "-C", root, "worktree", "remove", "--force", wt).Run() }
	defer cleanup()
	old := filepath.Join(tmp, "veles-against"+exe())
	run(wt, "go", "build", "-o", old, ".")
	compilers := [2]string{old, veles}

	dirs, err := filepath.Glob(filepath.Join(root, "bench", "*", "main.vs"))
	must(err)
	for name := range programs {
		dirs = append(dirs, filepath.Join(root, "bench", name, "main.vs"))
	}
	sort.Strings(dirs)
	fmt.Printf("| benchmark | %s | this tree | change |\n|---|---:|---:|---:|\n", rev)
	failed := false
	for _, main := range dirs {
		dir := filepath.Dir(main)
		name := filepath.Base(dir)
		if !strings.Contains(name, filter) {
			continue
		}
		var args []string
		dirs := [2]string{filepath.Join(wt, "bench", name), dir}
		if p, ok := programs[name]; ok {
			dirs, args = [2]string{filepath.Join(wt, p.dir), filepath.Join(root, p.dir)}, p.args
		}
		if _, err := os.Stat(dirs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "  %-10s not at %s: skipped\n", name, rev)
			continue
		}
		var bins [2]string
		ok := true
		for side, c := range compilers {
			bins[side] = filepath.Join(tmp, fmt.Sprintf("%s-%d%s", name, side, exe()))
			cmd := exec.Command(c, "build", dirs[side], "-o", bins[side], "--release")
			cmd.Dir = [2]string{wt, root}[side]
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "bench: %s does not build with the %s compiler: %v\n%s", name, [2]string{rev, "this tree's"}[side], err, out)
				ok = false
			}
		}
		if !ok {
			failed = true
			continue
		}
		var times [2][]time.Duration
		var sums [2]string
		for r := 0; r < rounds; r++ {
			order := [2]int{0, 1}
			if r%2 == 1 {
				order = [2]int{1, 0}
			}
			for _, side := range order {
				cmd := exec.Command(bins[side], args...)
				cmd.Dir = [2]string{wt, root}[side]
				out, err := cmd.Output()
				if err != nil {
					cleanup()
					fmt.Fprintf(os.Stderr, "bench: %s (%s side): %v\n", name, [2]string{rev, "this tree"}[side], err)
					os.Exit(1)
				}
				res := parse(name, out)
				times[side] = append(times[side], res.veles)
				sums[side] = res.checksum
			}
		}
		// a benchmark whose input is the tree itself (`parse` reads std) does
		// different work when the trees differ: flagged, not failed
		note := ""
		if sums[0] != sums[1] {
			note = " (checksums differ: not the same work)"
			fmt.Fprintf(os.Stderr, "bench: %s: %s computed %s, this tree %s\n", name, rev, sums[0], sums[1])
		}
		a, b := spread(times[0]), spread(times[1])
		change := (float64(b.median)/float64(a.median) - 1) * 100
		fmt.Printf("| %s | %s (%s–%s) | %s (%s–%s) | %+.0f%%%s |\n", name, round(a.median), round(a.min), round(a.max), round(b.median), round(b.min), round(b.max), change, note)
		fmt.Fprintf(os.Stderr, "  %-10s done\n", name)
	}
	fmt.Printf("\n%d rounds each, alternating which goes first; times are the Veles program's own measure.\n", rounds)
	if failed {
		cleanup()
		os.Exit(1)
	}
}

type durations struct{ min, median, max time.Duration }

func spread(ts []time.Duration) durations {
	s := append([]time.Duration{}, ts...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return durations{s[0], s[len(s)/2], s[len(s)-1]}
}
