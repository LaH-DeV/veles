package sema

import (
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Lint: a function written `throws` whose body cannot raise anything. The
// clause is not free: every caller has to `try` the call or handle a
// Result that can never be an error. The fix removes the clause; the
// callers' `try` then becomes the "needs a Result" error, which carries its
// own fix, so `veles check --fix` settles the whole program in two passes.
//
// Kept silent where the clause is part of a contract rather than a claim
// about this body: a `public` function (a library may reserve `throws` so
// that adding a failure later is not a breaking change), trait methods and
// their implementations (the signature is the trait's), externs, a
// declared error that mentions a type parameter (whether it throws depends
// on the instance), and a function used as a value (a function value does
// not take on `throws`, so it may be written that way to fit a throwing
// function type). A clause nobody wrote — a test's (D78) — is not linted.
func (c *Checker) lintNeedlessThrows() {
	if c.roundDiags.HasErrors() {
		// an error in a body may be the very expression that would have
		// thrown; "nothing here can throw" would then be wrong advice
		return
	}
	for _, t := range c.templates {
		d := t.Decl
		if d == nil || !d.Effects.Throws || !d.Effects.ThrowsSpan.IsValid() || (d.Body == nil && d.ExprBody == nil) {
			continue
		}
		if t.Pub || t.Extern || t.Impl != nil || t.Trait != nil || t.ValueUsed || len(t.Instances) == 0 {
			continue
		}
		if t.Sig == nil || (t.Sig.Effects.Error != nil && types.ContainsTypeParam(t.Sig.Effects.Error)) {
			continue
		}
		raised := false
		for _, inst := range t.Instances {
			if inst.raised {
				raised = true
				break
			}
		}
		if raised {
			continue
		}
		clause := d.Effects.ThrowsSpan
		if d.Effects.Error != nil {
			clause = clause.To(d.Effects.Error.Span())
		}
		c.warnFix(clause, fixDropClause("Remove 'throws'", clause),
			"'%s' is declared 'throws' but nothing in its body can throw, so every caller pays for a 'try' it does not need; remove the clause (D45)", t.Name)
	}
}

// fixDropClause deletes span together with the spaces before it, so
// `fun f(): i64 throws E {` becomes `fun f(): i64 {`.
func fixDropClause(title string, span source.Span) *source.Fix {
	if span.File == nil {
		return nil
	}
	src := span.File.Content
	start := span.Start
	for start > 0 && (src[start-1] == ' ' || src[start-1] == '\t') {
		start--
	}
	return fixReplace(title, source.Span{File: span.File, Start: start, End: span.End}, "")
}

// fixDropKeyword deletes the keyword kw that starts span (which may cover
// the whole expression) and the spaces after it, so `try f()` becomes `f()`.
func fixDropKeyword(title, kw string, span source.Span) *source.Fix {
	if span.File == nil {
		return nil
	}
	src := span.File.Content
	if span.Start+len(kw) > len(src) || src[span.Start:span.Start+len(kw)] != kw {
		return nil
	}
	end := span.Start + len(kw)
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return fixReplace(title, source.Span{File: span.File, Start: span.Start, End: end}, "")
}
