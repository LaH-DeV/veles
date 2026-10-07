package fetch

import (
	"strings"
	"testing"
)

func TestSetDependencyKeepsTheRestOfTheFile(t *testing.T) {
	text := "# my app\n[package]\nname = \"app\"\n\n[dependencies]\n# the math\nmathlib = \"../mathlib\"   # local\n\n[format]\nindent = 4\n"
	got, replaced, err := SetDependency(text, "dependencies", "pg", `{ git = "https://x/pg", version = "2.1.0" }`)
	if err != nil || replaced {
		t.Fatal(replaced, err)
	}
	want := "# my app\n[package]\nname = \"app\"\n\n[dependencies]\n# the math\nmathlib = \"../mathlib\"   # local\npg = { git = \"https://x/pg\", version = \"2.1.0\" }\n\n[format]\nindent = 4\n"
	if got != want {
		t.Errorf("add:\n%s\nwant:\n%s", got, want)
	}
	got, replaced, err = SetDependency(got, "dependencies", "pg", `{ git = "https://x/pg", version = "2.2.0" }`)
	if err != nil || !replaced || !strings.Contains(got, `version = "2.2.0" }`) || strings.Contains(got, "2.1.0") || strings.Count(got, "pg =") != 1 {
		t.Errorf("replace (%v, %v):\n%s", replaced, err, got)
	}
}

func TestSetDependencyCreatesTheTable(t *testing.T) {
	got, _, err := SetDependency("[package]\nname = \"app\"\n", "dependencies", "m", `"../m"`)
	if err != nil || got != "[package]\nname = \"app\"\n\n[dependencies]\nm = \"../m\"\n" {
		t.Errorf("%v\n%s", err, got)
	}
	got, _, _ = SetDependency("[package]\nname = \"app\"\n[dependencies]\n", "dependencies", "m", `"../m"`)
	if got != "[package]\nname = \"app\"\n[dependencies]\nm = \"../m\"\n" {
		t.Errorf("empty table:\n%s", got)
	}
	got, _, _ = SetDependency("", "dev-dependencies", "m", `"../m"`)
	if got != "[dev-dependencies]\nm = \"../m\"\n" {
		t.Errorf("empty file:\n%s", got)
	}
	// [dependencies] must not match [dev-dependencies]
	got, _, _ = SetDependency("[dev-dependencies]\nt = \"../t\"\n", "dependencies", "m", `"../m"`)
	if got != "[dev-dependencies]\nt = \"../t\"\n\n[dependencies]\nm = \"../m\"\n" {
		t.Errorf("sibling table:\n%s", got)
	}
}

func TestEditsKeepCRLF(t *testing.T) {
	text := "[package]\r\nname = \"app\"\r\n[dependencies]\r\na = \"../a\"\r\n"
	got, _, _ := SetDependency(text, "dependencies", "b", `"../b"`)
	if got != "[package]\r\nname = \"app\"\r\n[dependencies]\r\na = \"../a\"\r\nb = \"../b\"\r\n" {
		t.Errorf("add: %q", got)
	}
	got, ok, _ := RemoveDependency(got, "dependencies", "a")
	if !ok || got != "[package]\r\nname = \"app\"\r\n[dependencies]\r\nb = \"../b\"\r\n" {
		t.Errorf("remove: %q", got)
	}
}

func TestRemoveDependency(t *testing.T) {
	text := "[dependencies]\n\"a\" = \"../a\"\nb = { path = \"../b\" }\n[dev-dependencies]\nb = \"../b\"\n"
	got, ok, err := RemoveDependency(text, "dependencies", "b")
	if err != nil || !ok || got != "[dependencies]\n\"a\" = \"../a\"\n[dev-dependencies]\nb = \"../b\"\n" {
		t.Errorf("%v %v\n%s", ok, err, got)
	}
	got, ok, _ = RemoveDependency(text, "dependencies", "a")
	if !ok || strings.Contains(got, `"a"`) {
		t.Errorf("quoted key:\n%s", got)
	}
	if _, ok, _ := RemoveDependency(text, "dependencies", "zzz"); ok {
		t.Error("removed something that was not there")
	}
}

func TestEditRefusesSubTables(t *testing.T) {
	text := "[dependencies.pg]\ngit = \"u\"\nversion = \"1.0.0\"\n"
	if _, _, err := SetDependency(text, "dependencies", "pg", `"x"`); err == nil || !strings.Contains(err.Error(), "[dependencies.pg] table; edit it by hand") {
		t.Errorf("set: %v", err)
	}
	if _, _, err := RemoveDependency(text, "dependencies", "pg"); err == nil || !strings.Contains(err.Error(), "remove it by hand") {
		t.Errorf("remove: %v", err)
	}
}

func TestParseSpec(t *testing.T) {
	for _, c := range []struct {
		in                  string
		kind, source, v, nm string
	}{
		{"../mathlib", "path", "../mathlib", "", "mathlib"},
		{"./libs/my-lib", "path", "./libs/my-lib", "", "my_lib"},
		{"https://github.com/veles-db/pg", "git", "https://github.com/veles-db/pg", "", "pg"},
		{"https://github.com/veles-db/pg.git@2.1.0", "git", "https://github.com/veles-db/pg", "2.1.0", "pg"},
		{"github.com/acme/http-util@v1.4.2", "git", "https://github.com/acme/http-util", "1.4.2", "http_util"},
		{"git@github.com:acme/x", "git", "git@github.com:acme/x", "", "x"},
		{"git@github.com:acme/x@1.0.0", "git", "git@github.com:acme/x", "1.0.0", "x"},
		{"acme/httputil", "registry", "acme/httputil", "", "httputil"},
		{"acme/httputil@1.4.2", "registry", "acme/httputil", "1.4.2", "httputil"},
	} {
		s, err := ParseSpec(c.in)
		if err != nil || s.Kind != c.kind || s.Source != c.source || s.Version != c.v || s.Name != c.nm {
			t.Errorf("%s: %+v, %v", c.in, s, err)
		}
	}
	for in, want := range map[string]string{
		"":              "say what to add",
		"Acme/Thing":    "is not a path, a git URL or a registry name",
		"justaname":     "is not a path",
		"../x@1.0.0":    "a path has no version",
		"acme/x@banana": "is not a version",
		"acme/x@1.2":    "is not a version",
	} {
		if _, err := ParseSpec(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want %q", in, err, want)
		}
	}
}
