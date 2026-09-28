package docs

import (
	"regexp"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// Every diagnostic family has its section in reference/errors.md, which
// `veles explain` prints and an editor links to (D79), and every section
// under "At compile time" is a family — a renamed family cannot leave its
// old section behind.
func TestEveryFamilyIsExplained(t *testing.T) {
	families := map[string]bool{}
	for _, f := range source.Families() {
		families[f] = true
		if text, ok := Explain(f); !ok || text == "" {
			t.Errorf("family %q has no section '### %s' in reference/errors.md", f, f)
		}
	}
	doc := strings.ReplaceAll(errorsDoc, "\r\n", "\n")
	start := strings.Index(doc, "\n## At compile time\n")
	end := strings.Index(doc, "\n## At run time\n")
	if start < 0 || end < start {
		t.Fatal("reference/errors.md lost its 'At compile time' / 'At run time' parts")
	}
	for _, m := range regexp.MustCompile(`(?m)^### (.+)$`).FindAllStringSubmatch(doc[start:end], -1) {
		if !families[m[1]] {
			t.Errorf("section '### %s' is not a diagnostic family; a family's section is headed by its name (source/family.go)", m[1])
		}
	}
}

func TestExplainTakesOneSection(t *testing.T) {
	text, ok := Explain("val-else")
	if !ok || !strings.Contains(text, "#### `the 'else' of a 'val ... else' must leave`") || strings.Contains(text, "### numbers") {
		t.Errorf("the val-else section:\n%s", text)
	}
	if _, ok := Explain("no-such-family"); ok {
		t.Error("an unknown family was explained")
	}
}
