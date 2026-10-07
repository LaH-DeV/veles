package fetch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

// projectWith is project with extra manifest text (a [policy] table).
func projectWith(t *testing.T, requires, extra string) (string, *sema.Manifest) {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"veles.toml": "[package]\nname = \"app\"\n[dependencies]\n" + requires + "\n" + extra,
		"main.vs":    "fun main() {}\n",
	})
	man, err := sema.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, man
}

func netLib(t *testing.T) *repo {
	t.Helper()
	lib := newRepo(t)
	lib.release("1.0.0", pkgFiles("netlib", "1.0.0", "", "use net\npublic fun f() {}\n"))
	return lib
}

func TestPolicyRefusesADeniedCapability(t *testing.T) {
	home(t)
	lib := netLib(t)
	dir, man := projectWith(t, "netlib = "+gitDep(lib, "1.0.0"), "[policy]\ndeny = [\"net\", \"unsafe\"]\n")
	_, err := Resolve(dir, man)
	if err == nil {
		t.Fatal("a dependency that uses net was accepted")
	}
	for _, want := range []string{
		"veles.toml: [policy] refuses 1 use(s):",
		lib.url() + " 1.0.0 uses net (lib.vs:1 use net), which [policy] denies",
		"allow = { net = [\"" + lib.url() + "\"] }",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "veles.sum")); err == nil {
		t.Error("a refused package was recorded in veles.sum")
	}
}

func TestPolicyAllowByName(t *testing.T) {
	home(t)
	lib := netLib(t)
	for _, allow := range []string{lib.url(), "netlib"} { // by source, or by the package's own name
		dir, man := projectWith(t, "x = "+gitDep(lib, "1.0.0"), "[policy]\ndeny = [\"net\"]\nallow = { net = [\""+allow+"\"] }\n")
		if _, err := Resolve(dir, man); err != nil {
			t.Errorf("allowed by %s: %v", allow, err)
		}
	}
	// allowing net does not allow unsafe
	dir, man := projectWith(t, "x = "+gitDep(lib, "1.0.0"), "[policy]\ndeny = [\"net\"]\nallow = { fs = [\"netlib\"] }\n")
	if _, err := Resolve(dir, man); err == nil {
		t.Error("an allow for fs let net through")
	}
}

// The package that uses the capability is named, however deep it is; a path
// dependency is the project's own code and is not policed.
func TestPolicyNamesTheOffenderAndSparesPathDeps(t *testing.T) {
	home(t)
	lib := netLib(t)
	mid := newRepo(t)
	mid.release("1.0.0", pkgFiles("mid", "1.0.0", "netlib = "+gitDep(lib, "1.0.0"), "use netlib\npublic fun m() {}\n"))
	dir, man := projectWith(t, "mid = "+gitDep(mid, "1.0.0"), "[policy]\ndeny = [\"net\"]\n")
	_, err := Resolve(dir, man)
	if err == nil || !strings.Contains(err.Error(), lib.url()+" 1.0.0 uses net") || strings.Contains(err.Error(), mid.url()+" 1.0.0 uses") {
		t.Errorf("got %v", err)
	}
	// the same capability in a path dependency is fine
	own := t.TempDir()
	writeFiles(t, own, map[string]string{"veles.toml": "[package]\nname = \"own\"\n", "lib.vs": "use net\npublic fun f() {}\n"})
	dir2, man2 := projectWith(t, "own = \""+filepath.ToSlash(own)+"\"", "[policy]\ndeny = [\"net\"]\n")
	if _, err := Resolve(dir2, man2); err != nil {
		t.Errorf("path dependency: %v", err)
	}
}

func TestReportTotalsIncludeDependencies(t *testing.T) {
	home(t)
	lib := netLib(t)
	mid := newRepo(t)
	mid.release("1.0.0", pkgFiles("mid", "1.0.0", "netlib = "+gitDep(lib, "1.0.0"), "use netlib\npublic fun m() {}\n"))
	dir, man := projectWith(t, "mid = "+gitDep(mid, "1.0.0"), "")
	res := resolveOK(t, dir, man)
	report, err := res.Report()
	if err != nil || len(report) != 2 {
		t.Fatalf("%v %+v", err, report)
	}
	byName := map[string]PackageReport{}
	for _, p := range report {
		byName[p.Name] = p
	}
	m, n := byName["mid"], byName["netlib"]
	if m.Local != "mid" || n.Local != "" {
		t.Errorf("local names: mid %q, netlib %q (netlib is not the project's own dependency)", m.Local, n.Local)
	}
	if len(CapNames(m.Own)) != 0 || strings.Join(CapNames(m.Total), " ") != "net" || strings.Join(CapNames(n.Own), " ") != "net" {
		t.Errorf("mid own %v total %v, netlib own %v", m.Own.Names(), m.Total.Names(), n.Own.Names())
	}
	if m.Tier != "unreviewed" || m.Own["unlisted"] == nil {
		t.Errorf("a git package is never listed: tier %q, caps %v", m.Tier, m.Own.Names())
	}
	table := FormatReport(report, true)
	for _, want := range []string{mid.url() + " 1.0.0", "(dependencies add: net)", "    net    lib.vs:1 use net"} {
		if !strings.Contains(table, want) {
			t.Errorf("missing %q in:\n%s", want, table)
		}
	}
	// SkipPolicy reports what Resolve would refuse
	dir2, man2 := projectWith(t, "mid = "+gitDep(mid, "1.0.0"), "[policy]\ndeny = [\"net\"]\n")
	res2, err := ResolveWith(dir2, man2, ResolveOptions{SkipPolicy: true})
	if err != nil {
		t.Fatal(err)
	}
	rep2, _ := res2.Report()
	if vs := Violations(rep2, man2.Policy); len(vs) != 1 || vs[0].Capability != "net" || vs[0].Package.Name != "netlib" {
		t.Errorf("violations: %+v", vs)
	}
}

// A workspace's members answer to the policy of its root unless they have their own.
func TestWorkspacePolicyIsInherited(t *testing.T) {
	home(t)
	lib := netLib(t)
	ws := t.TempDir()
	writeFiles(t, ws, map[string]string{
		"veles.toml":   "[workspace]\nmembers = [\"a\", \"b\"]\n[policy]\ndeny = [\"net\"]\n",
		"a/veles.toml": "[package]\nname = \"a\"\n[dependencies]\nnetlib = " + gitDep(lib, "1.0.0") + "\n",
		"a/main.vs":    "fun main() {}\n",
		"b/veles.toml": "[package]\nname = \"b\"\n[dependencies]\nnetlib = " + gitDep(lib, "1.0.0") + "\n[policy]\ndeny = [\"fs\"]\n",
		"b/main.vs":    "fun main() {}\n",
	})
	a, _ := sema.ReadManifest(filepath.Join(ws, "a"))
	if _, err := Resolve(filepath.Join(ws, "a"), a); err == nil || !strings.Contains(err.Error(), "uses net") {
		t.Errorf("member a inherits the root's policy: %v", err)
	}
	b, _ := sema.ReadManifest(filepath.Join(ws, "b"))
	if _, err := Resolve(filepath.Join(ws, "b"), b); err != nil {
		t.Errorf("member b has its own policy, which allows net: %v", err)
	}
}
