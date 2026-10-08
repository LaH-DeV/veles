package sema

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// tree writes files (slash paths relative to root) and returns a loader.
type tree struct {
	t    *testing.T
	root string
}

func newTree(t *testing.T) *tree { return &tree{t: t, root: t.TempDir()} }

func (tr *tree) write(rel, src string) {
	p := filepath.Join(tr.root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		tr.t.Fatal(err)
	}
}

// pkg writes a package: its manifest and a root module.
func (tr *tree) pkg(dir, manifest, lib string) {
	tr.write(dir+"/veles.toml", manifest)
	tr.write(dir+"/lib.vs", lib)
}

func (tr *tree) load(dir string) (*Package, string) {
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(filepath.Join(tr.root, dir), diags)
	if err != nil {
		tr.t.Fatal(err)
	}
	if !diags.HasErrors() {
		Check(pkg, diags, false)
	}
	return pkg, diags.Render()
}

func depPrefixes(p *Package) []string {
	var out []string
	for k := range p.Modules {
		if strings.HasPrefix(k, "dep/") {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// D138: a package is one instance however it is reached. `a` and `b` both
// use `util`, and `app` reaches `util` under a second name: three routes, one
// set of modules and symbols.
func TestDiamondIsOneInstance(t *testing.T) {
	tr := newTree(t)
	tr.pkg("util", "[package]\nname = \"util\"\n", "public fun one(): i64 => 1\n")
	tr.pkg("a", "[package]\nname = \"a\"\n[dependencies]\nutil = \"../util\"\n", "use util\npublic fun fromA(): i64 => util.one() + 1\n")
	tr.pkg("b", "[package]\nname = \"b\"\n[dependencies]\nhelper = \"../util\"\n", "use helper\npublic fun fromB(): i64 => helper.one() + 2\n")
	tr.pkg("app", "[package]\nname = \"app\"\n[dependencies]\na = \"../a\"\nb = \"../b\"\nutil2 = \"../util\"\n",
		"use io\nuse a\nuse b\nuse util2\nfun main() { io.println(\"${a.fromA() + b.fromB() + util2.one()}\") }\n")
	pkg, out := tr.load("app")
	if out != "" {
		t.Fatalf("diagnostics: %s", out)
	}
	if got := strings.Join(depPrefixes(pkg), " "); got != "dep/a dep/b dep/util" {
		t.Errorf("modules: %s (util must be loaded once, whatever it is called)", got)
	}
}

// Two different packages with one name (two versions of a library, kept as
// two checkouts) are two instances with distinct module prefixes, so their
// symbols cannot collide.
func TestSamePackageNameTwice(t *testing.T) {
	tr := newTree(t)
	tr.pkg("pg1", "[package]\nname = \"pg\"\nversion = \"1.4.0\"\n", "public fun version(): i64 => 1\n")
	tr.pkg("pg2", "[package]\nname = \"pg\"\nversion = \"2.0.0\"\n", "public fun version(): i64 => 2\n")
	tr.pkg("app", "[package]\nname = \"app\"\n[dependencies]\npg1 = \"../pg1\"\npg2 = \"../pg2\"\n",
		"use io\nuse pg1\nuse pg2\nfun main() { io.println(\"${pg1.version()}${pg2.version()}\") }\n")
	pkg, out := tr.load("app")
	if out != "" {
		t.Fatalf("diagnostics: %s", out)
	}
	got := depPrefixes(pkg)
	if len(got) != 2 || got[0] == got[1] || !strings.HasPrefix(got[0], "dep/pg") || !strings.HasPrefix(got[1], "dep/pg") {
		t.Errorf("two instances expected: %v", got)
	}
}

// Only a package's own requires are nameable: `app` does not see what `a`
// depends on.
func TestOnlyDirectRequiresAreNameable(t *testing.T) {
	tr := newTree(t)
	tr.pkg("util", "[package]\nname = \"util\"\n", "public fun one(): i64 => 1\n")
	tr.pkg("a", "[package]\nname = \"a\"\n[dependencies]\nutil = \"../util\"\n", "use util\npublic use util\npublic fun x(): i64 => util.one()\n")
	tr.pkg("app", "[package]\nname = \"app\"\n[dependencies]\na = \"../a\"\n", "use io\nuse util\nfun main() { io.println(\"${util.one()}\") }\n")
	_, out := tr.load("app")
	if !strings.Contains(out, "unknown module 'util'") {
		t.Errorf("got: %s", out)
	}
}

func TestDependencyNameClashes(t *testing.T) {
	tr := newTree(t)
	tr.pkg("lib", "[package]\nname = \"lib\"\n", "public fun f(): i64 => 1\n")
	tr.pkg("app", "[package]\nname = \"app\"\n[dependencies]\nio = \"../lib\"\ngeometry = { path = \"../lib\" }\n",
		"use io\nfun main() { io.println(\"x\") }\n")
	tr.write("app/geometry/lib.vs", "public fun g(): i64 => 1\n")
	_, out := tr.load("app")
	for _, want := range []string{
		"dependency 'io' (veles.toml:4) has the name of a standard module",
		"dependency 'geometry' (veles.toml:5) has the name of a module of this package",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in: %s", want, out)
		}
	}
}

func TestPackageCannotDependOnItself(t *testing.T) {
	tr := newTree(t)
	tr.pkg("a", "[package]\nname = \"a\"\n[dependencies]\nb = \"../b\"\n", "use b\npublic fun x(): i64 => b.y()\n")
	tr.pkg("b", "[package]\nname = \"b\"\n[dependencies]\na = \"../a\"\n", "use a\npublic fun y(): i64 => 1\n")
	tr.write("a/main.vs", "use io\nfun main() { io.println(\"${x()}\") }\n")
	_, out := tr.load("a")
	if !strings.Contains(out, "is this package itself") {
		t.Errorf("got: %s", out)
	}
}
