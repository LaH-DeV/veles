package sema

import (
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
)

// Lint (notes #13): an `unsafe { }` block in a function that is not itself
// `unsafe fun` is a promise the compiler cannot check — the block is sound
// because of something the author knows. That something is written down in
// a `// SAFETY:` comment on the lines directly above the line the block
// starts on, or the function is declared `unsafe fun` and the obligation
// moves to its callers. A comment whose reason is empty still warns, so the
// stub the fix inserts cannot silence the lint on its own.
func (f *fnCtx) lintUnsafeBlock(e *ast.UnsafeExpr) {
	if f.unsafe > 0 || e.Pos.File == nil {
		return // inside `unsafe fun` or another unsafe block: already accounted for
	}
	file := e.Pos.File
	line, _ := file.Position(e.Pos.Start)
	reason, found := safetyComment(file, line)
	if !found && e.Body != nil {
		reason, found = safetyInside(file, e.Body.Pos.Start)
	}
	switch {
	case !found:
		f.warnFix(e.Pos, fixSafetyStub(e.Pos), "an 'unsafe' block needs a '// SAFETY:' comment on the line above saying why it is sound, or the function can be declared 'unsafe fun' so its callers take on the obligation")
	case reason == "":
		f.c.warnf(e.Pos, "the '// SAFETY:' comment above this 'unsafe' block gives no reason; say why the block is sound")
	}
}

// safetyComment looks for `SAFETY:` in the run of `//` comment lines that
// ends on the line above line (1-based). It reports the text after the
// marker, continued by any comment lines below it in the run.
func safetyComment(file *source.File, line int) (reason string, found bool) {
	first := line
	for first > 1 && isCommentLine(file.Line(first-1)) {
		first--
	}
	for n := first; n < line; n++ {
		text := commentText(file.Line(n))
		i := strings.Index(text, "SAFETY:")
		if i < 0 {
			continue
		}
		parts := []string{strings.TrimSpace(text[i+len("SAFETY:"):])}
		for m := n + 1; m < line; m++ {
			parts = append(parts, strings.TrimSpace(commentText(file.Line(m))))
		}
		return strings.TrimSpace(strings.Join(parts, " ")), true
	}
	return "", false
}

// safetyInside accepts the comment as the first thing in the block — after
// the `{` on its line, or on the lines right below it — the place for it
// when the block is a declaration's body (`fun pid(): i64 = unsafe {`),
// where the line above holds the declaration's doc comment.
func safetyInside(file *source.File, open int) (reason string, found bool) {
	line, _ := file.Position(open)
	rest := file.Line(line)
	if _, col := file.Position(open); col <= len(rest) {
		rest = rest[col:] // after the `{`
	}
	if i := strings.Index(rest, "//"); i >= 0 && strings.TrimSpace(rest[:i]) == "" {
		if j := strings.Index(rest, "SAFETY:"); j >= 0 {
			return strings.TrimSpace(rest[j+len("SAFETY:"):]), true
		}
	}
	last := line + 1
	for isCommentLine(file.Line(last)) {
		last++
	}
	return safetyComment(file, last)
}

func isCommentLine(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "//")
}

func commentText(s string) string {
	return strings.TrimLeft(strings.TrimSpace(s), "/")
}

// fixSafetyStub inserts `// SAFETY: ` above the line the block starts on,
// indented like it. The reason is left for the author: the lint keeps
// warning until it is written.
func fixSafetyStub(at source.Span) *source.Fix {
	src := at.File.Content
	ls := at.Start
	for ls > 0 && src[ls-1] != '\n' {
		ls--
	}
	indent := ls
	for indent < len(src) && (src[indent] == ' ' || src[indent] == '\t') {
		indent++
	}
	nl := "\n"
	if le := strings.IndexByte(src[at.Start:], '\n'); le > 0 && src[at.Start+le-1] == '\r' {
		nl = "\r\n"
	}
	stub := src[ls:indent] + "// SAFETY: " + nl
	return fixReplace("Add a '// SAFETY:' comment", source.Span{File: at.File, Start: ls, End: ls}, stub)
}
