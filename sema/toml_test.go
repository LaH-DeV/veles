package sema

import (
	"strings"
	"testing"
)

func TestTOMLValues(t *testing.T) {
	root, err := parseTOML(string(rune(0xFEFF)) + "# top\r\n[a]\r\ns = \"x\\ty\\u00e9\"  # c\r\nl = 'raw\\n'\r\nn = 1_000\r\nb = true\r\nlist = [\r\n  \"p\", # first\r\n  \"q\",\r\n]\r\nit = { k = \"v\", n = 2 }\r\n[a.sub]\r\nz = []\r\n")
	if err != nil {
		t.Fatal(err)
	}
	a := root.vals["a"].tab
	if got := a.vals["s"].str; got != "x\tyé" {
		t.Errorf("escapes: %q", got)
	}
	if got := a.vals["l"].str; got != `raw\n` {
		t.Errorf("literal string: %q", got)
	}
	if a.vals["n"].num != 1000 || !a.vals["b"].b {
		t.Errorf("number/bool: %+v %+v", a.vals["n"], a.vals["b"])
	}
	if l := a.vals["list"].list; len(l) != 2 || l[0].str != "p" || l[1].str != "q" {
		t.Errorf("list: %+v", l)
	}
	if it := a.vals["it"].tab; it.vals["k"].str != "v" || it.vals["n"].num != 2 {
		t.Errorf("inline table: %+v", it.vals)
	}
	if z := a.vals["sub"].tab.vals["z"]; z.kind != tomlList || len(z.list) != 0 {
		t.Errorf("nested table: %+v", z)
	}
}

func TestTOMLErrors(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"[a]\nk = 1\nk = 2\n", "line 3: key \"k\" is set twice"},
		{"[a]\n[a]\n", "line 2: table [a] is defined twice"},
		{"a = 1\n[a]\n", "is already a number, not a table"},
		{"[[a]]\n", "arrays of tables"},
		{"a.b = 1\n", "dotted keys are not supported"},
		{"k = \"open\n", "must close on its own line"},
		{"k = 1.5\n", "cannot read \"1.5\""},
		{"k = word\n", "strings are written in quotes"},
		{"k = [1,\n", "expected a value"},
		{"k = { a = 1,\n b = 2 }\n", "an inline table is one line"},
		{"k = 1 2\n", "unexpected text after the value"},
		{"k = \"a\\q\"\n", "unknown escape"},
	} {
		_, err := parseTOML(c.text)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.text, err, c.want)
		}
	}
}

func TestVersions(t *testing.T) {
	for _, c := range []struct{ in, str, major string }{
		{"1.4.2", "1.4.2", "1"},
		{"v2.0.0", "2.0.0", "2"},
		{"0.3.1", "0.3.1", "0.3"},
		{"0.0.2", "0.0.2", "0.0.2"},
		{"1.0.0-rc.1", "1.0.0-rc.1", "1"},
	} {
		v, err := ParseVersion(c.in)
		if err != nil || v.String() != c.str || v.MajorID() != c.major {
			t.Errorf("%s: %v %v %v", c.in, v, v.MajorID(), err)
		}
	}
	for _, bad := range []string{"1", "1.2", "1.2.3.4", "01.2.3", "1.2.x", "1.2.3+b", "1.2.3-", "1.2.3-a_b", ""} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
	// ordering: a pre-release sorts before its release, numbers before words
	order := []string{"0.9.0", "1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "2.0.0"}
	for i := range order {
		for j := range order {
			a, _ := ParseVersion(order[i])
			b, _ := ParseVersion(order[j])
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got := a.Compare(b); got != want {
				t.Errorf("%s vs %s: %d, want %d", order[i], order[j], got, want)
			}
		}
	}
}
