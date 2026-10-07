package sema

import (
	"strings"
	"testing"
)

func parseOK(t *testing.T, text string) *Manifest {
	t.Helper()
	m, err := parseManifest("veles.toml", ".", text)
	if err != nil {
		t.Fatalf("%v\n%s", err, text)
	}
	return m
}

// D138: dependencies come from a path, the registry or git; the short
// string form stays a path.
func TestManifestDependencies(t *testing.T) {
	m := parseOK(t, `
[package]
name = "app"
version = "0.1.0"
description = "d"
license = "MIT"
veles = "0.70"

[dependencies]
mathlib = "../mathlib"
httputil = { registry = "acme/httputil", version = "1.4.2" }
pg = { git = "https://github.com/veles-db/pg", version = "2.1.0" }
fast = { git = "https://example.com/fast", commit = "3f2a9c1" }
local = { path = "libs/local" }

[dev-dependencies]
fakeclock = { registry = "lah/fakeclock", version = "0.3.0" }
`)
	if m.Name != "app" || m.Version != "0.1.0" || m.License != "MIT" || m.Description != "d" || m.Veles != "0.70" {
		t.Errorf("package: %+v", m)
	}
	var got []string
	for _, d := range m.Requires {
		got = append(got, d.Name+":"+d.Kind()+":"+d.Source()+":"+d.Version+d.Commit)
	}
	want := "fast:git:https://example.com/fast:3f2a9c1 httputil:registry:acme/httputil:1.4.2 local:path:libs/local: mathlib:path:../mathlib: pg:git:https://github.com/veles-db/pg:2.1.0"
	if strings.Join(got, " ") != want {
		t.Errorf("requires:\n got %s\nwant %s", strings.Join(got, " "), want)
	}
	if len(m.Deps) != 2 || m.Deps["mathlib"] != "../mathlib" || m.Deps["local"] != "libs/local" {
		t.Errorf("path deps for the loader: %v", m.Deps)
	}
	if len(m.DevRequires) != 1 || m.DevRequires[0].Registry != "lah/fakeclock" {
		t.Errorf("dev: %+v", m.DevRequires)
	}
}

func TestManifestWorkspace(t *testing.T) {
	m := parseOK(t, "[workspace]\nmembers = [\"app\", \"libs/mathlib\", \"./tools\"]\n")
	if m.Workspace == nil || strings.Join(m.Workspace.Members, ",") != "app,libs/mathlib,tools" {
		t.Errorf("members: %+v", m.Workspace)
	}
	if m.Name != "" {
		t.Errorf("a virtual root has no name: %q", m.Name)
	}
	// a root that is also a package
	m = parseOK(t, "[package]\nname = \"root\"\n[workspace]\nmembers = [\"a\"]\n")
	if m.Name != "root" || len(m.Workspace.Members) != 1 {
		t.Errorf("root package: %+v", m)
	}
}

func TestManifestErrors(t *testing.T) {
	pkg := "[package]\nname = \"p\"\n"
	for _, c := range []struct{ text, want string }{
		{"", "[package] name is required"},
		{"name = \"p\"\n", "'name' is outside any table"},
		{pkg + "[extras]\n", "unknown table [extras]"},
		{pkg + "[package]\n", "table [package] is defined twice"},
		{"[package]\nname = \"p\"\nauthor = \"x\"\n", "unknown [package] key \"author\""},
		{"[package]\nname = 5\n", "[package] name is a string"},
		{"[package]\nname = \"p\"\nversion = \"1.2\"\n", "[package] version: \"1.2\" is not a version"},
		{"[package]\nname = \"p\"\nveles = \"x\"\n", "[package] veles:"},
		{"[package]\nname = \"p\"\nexports = [\"a\"]\n", "'exports' was removed (D89)"},
		{pkg + "[dependencies]\nm = \"1.4.2\"\n", "looks like a version, but a bare string is a path"},
		{pkg + "[dependencies]\nm = 3\n", "is a path string or a table"},
		{pkg + "[dependencies]\n\"my-lib\" = \"../x\"\n", "dependency name \"my-lib\""},
		{pkg + "[dependencies]\nm = {}\n", "exactly one of path, registry or git"},
		{pkg + "[dependencies]\nm = { path = \"../m\", git = \"u\", version = \"1.0.0\" }\n", "exactly one of path, registry or git"},
		{pkg + "[dependencies]\nm = { path = \"../m\", version = \"1.0.0\" }\n", "a path dependency has no version or commit"},
		{pkg + "[dependencies]\nm = { registry = \"acme/m\" }\n", "needs a version"},
		{pkg + "[dependencies]\nm = { registry = \"m\", version = \"1.0.0\" }\n", "is owner/name"},
		{pkg + "[dependencies]\nm = { registry = \"Acme/m\", version = \"1.0.0\" }\n", "is owner/name"},
		{pkg + "[dependencies]\nm = { registry = \"a/m\", version = \"v1.0.0\" }\n", "without the 'v'"},
		{pkg + "[dependencies]\nm = { registry = \"a/m\", version = \"1.0.0\", commit = \"3f2a9c1\" }\n", "pinned by version"},
		{pkg + "[dependencies]\nm = { git = \"u\" }\n", "exactly one of version (a tag) or commit"},
		{pkg + "[dependencies]\nm = { git = \"u\", version = \"1.0.0\", commit = \"3f2a9c1\" }\n", "exactly one of version (a tag) or commit"},
		{pkg + "[dependencies]\nm = { git = \"u\", commit = \"zz\" }\n", "hex prefix of 7 to 40"},
		{pkg + "[dependencies]\nm = { git = \"u\", tag = \"1\" }\n", "unknown key \"tag\""},
		{pkg + "[workspace]\n", "needs members"},
		{pkg + "[workspace]\nmembers = [\"../x\"]\n", "leaves the workspace"},
		{pkg + "[workspace]\nmembers = [\"libs/*\"]\n", "patterns such as \"libs/*\" are not supported yet"},
		{pkg + "[workspace]\nmembers = [\"a\", \"./a\"]\n", "listed twice"},
		{pkg + "[workspace]\nmembers = [\".\"]\n", "a directory below the root"},
		{pkg + "[workspace]\nmembers = \"a\"\n", "members is a list"},
		{pkg + "[workspace]\nroot = \"a\"\n", "unknown [workspace] key \"root\""},
		{pkg + "[format]\nindent = 9\n", "[format] indent must be"},
		{pkg + "[format]\nwidth = 2\n", "unknown [format] key \"width\""},
		{pkg + "[format]\nmax_blank_lines = 0\n", "max_blank_lines must be a positive number"},
		{pkg + "[native]\nlibs = [1]\n", "holds strings only"},
	} {
		_, err := parseManifest("veles.toml", ".", c.text)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q:\n got %v\nwant %q", c.text, err, c.want)
		} else if !strings.HasPrefix(err.Error(), "veles.toml:") {
			t.Errorf("%q: no location: %v", c.text, err)
		}
	}
}

// the format table keeps its old spellings: a number or "tab", as a number or a string
func TestManifestFormatForms(t *testing.T) {
	m := parseOK(t, "[package]\nname = \"p\"\n[format]\nindent = 4\nmax_blank_lines = 2\n")
	if m.Format.Indent != "    " || m.Format.MaxBlankLines != 2 {
		t.Errorf("%+v", m.Format)
	}
	m = parseOK(t, "[package]\nname = \"p\"\n[format]\nindent = \"tab\"\n")
	if m.Format.Indent != "\t" {
		t.Errorf("%+v", m.Format)
	}
	m = parseOK(t, "[package]\nname = \"p\"\n[format]\nindent = \"2\"\n")
	if m.Format.Indent != "  " {
		t.Errorf("%+v", m.Format)
	}
}
