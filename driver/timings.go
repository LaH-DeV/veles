package driver

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/LaH-DeV/veles/sema"
)

// clock times the phases of one build for `--timings`; a nil clock does
// nothing, so the untimed path pays one nil check per phase.
type clock struct {
	last   time.Time
	start  time.Time
	phases []phase
	front  *sema.Timings
}

type phase struct {
	name string
	note string
	took time.Duration
}

func newClock(on bool) *clock {
	if !on {
		return nil
	}
	now := time.Now()
	return &clock{last: now, start: now, front: &sema.Timings{}}
}

// lap closes the phase that ran since the last lap.
func (c *clock) lap(name, note string) {
	if c == nil {
		return
	}
	now := time.Now()
	c.phases = append(c.phases, phase{name, note, now.Sub(c.last)})
	c.last = now
}

// frontEnd is the per-module record the loader and checker fill, or nil.
func (c *clock) frontEnd() *sema.Timings {
	if c == nil {
		return nil
	}
	return c.front
}

// report prints the phases, then the modules that cost the most — what to
// look at when a build is slow.
func (c *clock) report(w io.Writer) {
	if c == nil {
		return
	}
	fmt.Fprintf(w, "timings: %s\n", span(time.Since(c.start)))
	for _, p := range c.phases {
		line := fmt.Sprintf("  %-8s %8s", p.name, span(p.took))
		if p.note != "" {
			line += "  " + p.note
		}
		fmt.Fprintln(w, line)
	}
	type row struct {
		path string
		m    *sema.ModuleTiming
	}
	var rows []row
	for path, m := range c.front.Modules {
		if path == "" {
			path = "(root)"
		}
		rows = append(rows, row{path, m})
	}
	if len(rows) == 0 {
		return
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i].m.Parse+rows[i].m.Check, rows[j].m.Parse+rows[j].m.Check
		if a != b {
			return a > b
		}
		return rows[i].path < rows[j].path
	})
	shown := min(len(rows), 10)
	fmt.Fprintf(w, "modules, slowest first (%d of %d): parse + check\n", shown, len(rows))
	width := 0
	for _, r := range rows[:shown] {
		width = max(width, len(r.path))
	}
	for _, r := range rows[:shown] {
		fmt.Fprintf(w, "  %-*s %8s + %-8s %d %s, %s\n", width, r.path, span(r.m.Parse), span(r.m.Check),
			r.m.Files, plural(r.m.Files, "file"), size(r.m.Bytes))
	}
}

// frontEndNote summarises what was loaded, for the load phase's line.
func (c *clock) frontEndNote() string {
	if c == nil {
		return ""
	}
	files, bytes := 0, 0
	for _, m := range c.front.Modules {
		files += m.Files
		bytes += m.Bytes
	}
	return fmt.Sprintf("%d %s, %s, in %d %s", files, plural(files, "file"), size(bytes), len(c.front.Modules), plural(len(c.front.Modules), "module"))
}

// span prints a duration at the precision a person reads it at: 1.18s,
// 310ms, 4.2ms, 0.3ms.
func span(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= 10*time.Millisecond:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	}
}

func size(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
