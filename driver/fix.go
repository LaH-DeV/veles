package driver

import (
	"fmt"
	"os"
	"sort"

	"github.com/LaH-DeV/veles/format"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// ApplyFixes applies the automatic corrections attached to diagnostics
// (`veles check --fix`): every edit of every fix, latest offset first so
// earlier offsets stay valid, then the formatter over each changed file.
// It returns the number of fixes applied.
func ApplyFixes(diags *source.Diagnostics) (int, error) {
	type edit struct {
		source.TextEdit
		order int
	}
	byFile := map[*source.File][]edit{}
	n := 0
	order := 0
	// two fixes that make the same edit (both add 'use codec') make it once
	type key struct {
		file       *source.File
		start, end int
		text       string
	}
	seen := map[key]bool{}
	for _, d := range diags.Items {
		if d.Fix == nil {
			continue
		}
		counted := false
		for _, e := range d.Fix.Edits {
			if e.Span.File == nil || e.Span.File.Embedded {
				continue
			}
			if !counted {
				n++ // a fix with somewhere to write
				counted = true
			}
			k := key{e.Span.File, e.Span.Start, e.Span.End, e.NewText}
			if seen[k] {
				continue
			}
			seen[k] = true
			byFile[e.Span.File] = append(byFile[e.Span.File], edit{e, order})
			order++
		}
	}
	for f, edits := range byFile {
		// descending by offset; insertions at the same offset keep their
		// order by being applied last first
		sort.SliceStable(edits, func(i, j int) bool {
			if edits[i].Span.Start != edits[j].Span.Start {
				return edits[i].Span.Start > edits[j].Span.Start
			}
			return edits[i].order > edits[j].order
		})
		text := f.Content
		limit := len(text) // edits that overlap one already applied wait for the next run
		for _, e := range edits {
			if e.Span.Start < 0 || e.Span.End > len(f.Content) || e.Span.Start > e.Span.End {
				return n, fmt.Errorf("%s: fix edit out of range", f.Path)
			}
			if e.Span.End > limit {
				continue
			}
			text = text[:e.Span.Start] + e.NewText + text[e.Span.End:]
			limit = e.Span.Start
		}
		style, err := StyleFor(f.Path)
		if err != nil {
			return n, err
		}
		if formatted, fd := format.Source(source.NewFile(f.Path, text), style); !fd.HasErrors() {
			text = formatted
		}
		if err := os.WriteFile(f.Path, []byte(text), 0o644); err != nil {
			return n, err
		}
		fmt.Println("fixed", f.Path)
	}
	return n, nil
}

// fixUntilDone is `veles check --fix`: check, apply every fix, and again
// while a pass still finds one — a fix often uncovers the next (an error
// in a declaration stops the checker before the bodies, so what the
// bodies need is only seen once it is gone) — then report what is left.
// Bounded, so a fix that undoes another cannot loop forever.
func fixUntilDone(opts Options) int {
	total := 0
	for pass := 0; pass < 8; pass++ {
		diags := &source.Diagnostics{}
		pkg, err := sema.LoadPackage(opts.Path, diags)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		if !diags.HasErrors() {
			sema.Check(pkg, diags, opts.Release)
		}
		n, err := ApplyFixes(diags)
		if err != nil {
			fmt.Fprintln(os.Stderr, "veles:", err)
			return 1
		}
		if n == 0 {
			break
		}
		total += n
	}
	opts.Fix = false
	code := Run(opts)
	fmt.Fprintf(os.Stderr, "%d fix(es) applied\n", total)
	return code
}
