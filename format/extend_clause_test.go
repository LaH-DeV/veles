package format

import "testing"

// TestUseExtendClause: the `extend` clause of a use (D185) is kept, its names
// sorted like the braced ones, and each form is stable.
func TestUseExtendClause(t *testing.T) {
	for src, want := range map[string]string{
		"use strx extend\n":                                   "use strx extend\n",
		"use   strx   extend   { shout,slug }\n":              "use strx extend { shout, slug }\n",
		"use strx extend { slug, shout }\n":                   "use strx extend { shout, slug }\n",
		"use regex { Regex,Match } extend { findAll,find }\n": "use regex { Match, Regex } extend { find, findAll }\n",
		"use a.b as c extend { f }\n":                         "use a.b as c extend { f }\n",
		"public use regex { Regex } extend { find }\n":        "public use regex { Regex } extend { find }\n",
	} {
		got := checkRoundTrip(t, "extend_clause.vs", src)
		if got != want {
			t.Errorf("%q formatted:\n%s\nwant:\n%s", src, got, want)
		}
	}
}
