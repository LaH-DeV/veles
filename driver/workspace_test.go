package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workspaceTree writes a monorepo: `app` depends on `libs/mathlib` by path, and
// `tools` is a library with a test.
func workspaceTree(t *testing.T, rootManifest string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", rootManifest)
	write("libs/mathlib/veles.toml", "[package]\nname = \"mathlib\"\n")
	write("libs/mathlib/lib.vs", "public fun twice(n: i64): i64 => n * 2\n\ntest \"twice\" {\n  expect(twice(2) == 4)\n}\n")
	write("app/veles.toml", "[package]\nname = \"app\"\n[dependencies]\nmathlib = \"../libs/mathlib\"\n")
	write("app/main.vs", "use io\nuse mathlib\n\nfun main() {\n  io.println(\"${mathlib.twice(21)}\")\n}\n")
	write("tools/veles.toml", "[package]\nname = \"tools\"\n")
	write("tools/lib.vs", "public fun one(): i64 => 1\n\ntest \"one\" {\n  expect(one() == 1)\n}\n")
	return dir
}

// runCapturing runs the driver and returns its exit code and standard error.
func runCapturing(t *testing.T, opts Options) (int, string) {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "stderr.txt"))
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = f
	code := Run(opts)
	os.Stderr = saved
	f.Close()
	data, _ := os.ReadFile(f.Name())
	return code, string(data)
}

const wsManifest = "[workspace]\nmembers = [\"app\", \"libs/mathlib\", \"tools\"]\n"

// `veles check|test|build` at a workspace root runs every member.
func TestWorkspaceRunsEveryMember(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
	dir := workspaceTree(t, wsManifest)
	for _, mode := range []string{"check", "test"} {
		code, out := runCapturing(t, Options{Path: dir, Mode: mode})
		if code != 0 {
			t.Fatalf("%s: exit %d:\n%s", mode, code, out)
		}
		for _, want := range []string{"==> app (app)", "==> libs/mathlib (mathlib)", "==> tools (tools)", "3 workspace members ok"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q in:\n%s", mode, want, out)
			}
		}
	}
	// build: the program is built, the libraries have nothing to build
	code, out := runCapturing(t, Options{Path: dir, Mode: "build"})
	if code != 0 {
		t.Fatalf("build: exit %d:\n%s", code, out)
	}
	if strings.Count(out, "library: nothing to build") != 2 {
		t.Errorf("the two libraries should be skipped:\n%s", out)
	}
	exe := filepath.Join(dir, "bin", "app")
	if _, err := os.Stat(exe + ".exe"); err == nil {
		exe += ".exe"
	}
	if got := strings.TrimSpace(runExe(t, exe)); got != "42" {
		t.Errorf("app printed %q", got)
	}
}

// A broken member does not hide the others (app fails too: it uses the broken
// mathlib): all run, the exit is 1, and the
// summary names it.
func TestWorkspaceReportsEveryFailure(t *testing.T) {
	dir := workspaceTree(t, wsManifest)
	os.WriteFile(filepath.Join(dir, "tools", "lib.vs"), []byte("public fun one(): i64 => \"one\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "libs", "mathlib", "lib.vs"), []byte("public fun twice(n: i64): i64 => n * true\n"), 0o644)
	code, out := runCapturing(t, Options{Path: dir, Mode: "check"})
	if code != 1 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	for _, want := range []string{"==> app (app)", "==> tools (tools)", "3 of 3 workspace members failed: app, mathlib, tools"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// Inside a member the tools behave as they always did, and the workspace
// root's own refusals are plain.
func TestWorkspaceEdges(t *testing.T) {
	dir := workspaceTree(t, wsManifest)
	if code, out := runCapturing(t, Options{Path: filepath.Join(dir, "app"), Mode: "check"}); code != 0 || strings.Contains(out, "==>") {
		t.Errorf("a member alone: exit %d:\n%s", code, out)
	}
	if code, out := runCapturing(t, Options{Path: dir, Mode: "run"}); code != 2 || !strings.Contains(out, "veles run <member> (members: app, libs/mathlib, tools)") {
		t.Errorf("run at the root: exit %d:\n%s", code, out)
	}
	if code, out := runCapturing(t, Options{Path: dir, Mode: "build", Output: "x"}); code != 2 || !strings.Contains(out, "-o names one executable") {
		t.Errorf("-o at the root: exit %d:\n%s", code, out)
	}
	for _, c := range []struct{ manifest, want string }{
		{"[workspace]\nmembers = [\"app\", \"nope\"]\n", "workspace member 'nope'"},
		{"[workspace]\nmembers = [\"app\", \"app/../tools\", \"libs/mathlib\"]\n[package]\nname = \"app\"\n", "both named 'app'"},
	} {
		d := workspaceTree(t, c.manifest)
		if code, out := runCapturing(t, Options{Path: d, Mode: "check"}); code != 1 || !strings.Contains(out, c.want) {
			t.Errorf("%q: exit %d:\n%s", c.manifest, code, out)
		}
	}
	// a root that is also a package is a member too
	d := workspaceTree(t, "[package]\nname = \"root\"\n[workspace]\nmembers = [\"app\", \"libs/mathlib\"]\n")
	os.WriteFile(filepath.Join(d, "main.vs"), []byte("use io\n\nfun main() {\n  io.println(\"root\")\n}\n"), 0o644)
	if code, out := runCapturing(t, Options{Path: d, Mode: "check"}); code != 0 || !strings.Contains(out, "==> . (root)") || !strings.Contains(out, "3 workspace members ok") {
		t.Errorf("root package: exit %d:\n%s", code, out)
	}
}
