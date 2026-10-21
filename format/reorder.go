package format

import (
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
)

// A sealed trait's body reads shape first: the associated types and the
// methods, then the variants declared inside it (D193). The printer keeps the
// source order of members, since a comment belongs to the position it stands
// at, so the order is put right in the text before the file is printed: each
// member moves with the comment lines directly above it and the comment that
// ends its last line.

// member is the source range of one member of a trait body, from the start of
// the line its leading comments begin on to the end of the line it ends on.
type member struct {
	lo, hi  int
	variant bool
}

// reorderVariants returns the source with the variants of every sealed trait
// moved after its other members, or "" when nothing moves (or a trait is laid
// out in a way this does not understand: then it is left as written).
func reorderVariants(f *ast.File) string {
	src := f.Source.Content
	type edit struct {
		from, to int
		text     string
	}
	var edits []edit
	for _, d := range f.Decls {
		td, ok := d.(*ast.TraitDecl)
		if !ok || len(td.Variants) == 0 {
			continue
		}
		from, to, text, ok := reorderTrait(src, td)
		if ok {
			edits = append(edits, edit{from, to, text})
		}
	}
	if len(edits) == 0 {
		return ""
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].from > edits[j].from })
	for _, e := range edits {
		src = src[:e.from] + e.text + src[e.to:]
	}
	return src
}

func reorderTrait(src string, td *ast.TraitDecl) (from, to int, text string, ok bool) {
	end := td.Pos.End
	if end < 1 || end > len(src) || src[end-1] != '}' {
		return 0, 0, "", false
	}
	open := strings.Index(src[td.Name.Pos.End:end], "{")
	if open < 0 {
		return 0, 0, "", false
	}
	open += td.Name.Pos.End
	if strings.Contains(src[open:end], "/*") {
		return 0, 0, "", false
	}
	var ms []member
	add := func(start, stop int, variant bool) bool {
		lo := lineStart(src, start)
		if strings.TrimSpace(src[lo:start]) != "" {
			return false
		}
		hi := lineEnd(src, stop)
		if rest := strings.TrimSpace(src[stop:hi]); rest != "" && !strings.HasPrefix(rest, "//") {
			return false
		}
		ms = append(ms, member{lo: lo, hi: hi, variant: variant})
		return true
	}
	for _, at := range td.AssocTypes {
		if !add(at.Pos.Start, at.Pos.End, false) {
			return 0, 0, "", false
		}
	}
	for _, m := range td.Methods {
		if !add(declStart(m), m.Pos.End, false) {
			return 0, 0, "", false
		}
	}
	for _, v := range td.Variants {
		if !add(declStart(v), v.Pos.End, true) {
			return 0, 0, "", false
		}
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].lo < ms[j].lo })
	// comment lines directly above a member go with it
	floor := lineEnd(src, open) // the line of the brace is not a member's
	for i := range ms {
		limit := floor
		if i > 0 {
			limit = ms[i-1].hi
		}
		for ms[i].lo > limit+1 {
			prev := lineStart(src, ms[i].lo-1)
			line := strings.TrimSpace(src[prev:ms[i].lo])
			if prev <= limit || !strings.HasPrefix(line, "//") {
				break
			}
			ms[i].lo = prev
		}
		if i > 0 && ms[i].lo < ms[i-1].hi {
			return 0, 0, "", false
		}
	}
	moves := false
	seenVariant := false
	for _, m := range ms {
		if m.variant {
			seenVariant = true
		} else if seenVariant {
			moves = true
		}
	}
	if !moves {
		return 0, 0, "", false
	}
	// the gap before a member (blank lines) travels with it
	gaps := make([]string, len(ms))
	prevEnd := open + 1
	for i, m := range ms {
		gaps[i] = src[prevEnd:m.lo]
		prevEnd = m.hi
	}
	tail := src[prevEnd : end-1]
	var b strings.Builder
	first, wroteVariant := true, false
	for _, wantVariant := range []bool{false, true} {
		for i, m := range ms {
			if m.variant != wantVariant {
				continue
			}
			g := gaps[i]
			if first {
				g = gaps[0]
				first = false
			} else if wantVariant && !wroteVariant && strings.Count(g, "\n") < 2 {
				g = "\n" + g // a blank line between the shape and the variants
			}
			if wantVariant {
				wroteVariant = true
			}
			b.WriteString(g)
			b.WriteString(src[m.lo:m.hi])
		}
	}
	b.WriteString(tail)
	return open + 1, end - 1, b.String(), true
}

func lineStart(s string, pos int) int {
	for pos > 0 && s[pos-1] != '\n' {
		pos--
	}
	return pos
}

func lineEnd(s string, pos int) int {
	for pos < len(s) && s[pos] != '\n' {
		pos++
	}
	return pos
}
