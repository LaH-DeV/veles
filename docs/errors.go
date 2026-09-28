// Package docs holds the documentation. The one part the compiler carries
// is reference/errors.md, so `veles explain <family>` answers offline and
// for the compiler in hand (D79); the rest is tested by docs_test.go.
package docs

import (
	_ "embed"
	"strings"
)

//go:embed documentation/reference/errors.md
var errorsDoc string

// Explain returns the section of reference/errors.md headed by a
// diagnostic family's name (`### private-to-module`), without the heading
// line, trimmed; false when there is none.
func Explain(family string) (string, bool) {
	text := strings.ReplaceAll(errorsDoc, "\r\n", "\n")
	head := "\n### " + family + "\n"
	i := strings.Index(text, head)
	if i < 0 {
		return "", false
	}
	body := text[i+len(head):]
	// the section runs to the next heading of its level or above
	for _, next := range []string{"\n### ", "\n## ", "\n# "} {
		if j := strings.Index(body, next); j >= 0 {
			body = body[:j]
		}
	}
	return strings.TrimSpace(body), true
}
