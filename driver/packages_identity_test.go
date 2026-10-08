package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D138 end to end: a diamond (`a` and `b` both depend on `util`, `app` names
// it a third way) links one `util`, so its module-level state is shared; and
// two packages with one name and one function name link side by side.
func TestPackageIdentityRuns(t *testing.T) {
	if _, err := findClang(); err != nil {
		t.Skip("clang not available:", err)
	}
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
	write("util/veles.toml", "[package]\nname = \"util\"\n")
	write("util/lib.vs", "val counter = Atomic(value: 0)\n\npublic fun bump(): i64 => counter.update((n) => n + 1)\n")
	write("a/veles.toml", "[package]\nname = \"a\"\n[dependencies]\nutil = \"../util\"\n")
	write("a/lib.vs", "use util\n\npublic fun fromA(): i64 => util.bump()\n")
	write("b/veles.toml", "[package]\nname = \"b\"\n[dependencies]\nhelper = \"../util\"\n")
	write("b/lib.vs", "use helper\n\npublic fun fromB(): i64 => helper.bump()\n")
	write("pg1/veles.toml", "[package]\nname = \"pg\"\nversion = \"1.4.0\"\n")
	write("pg1/lib.vs", "public fun version(): i64 => 1\n")
	write("pg2/veles.toml", "[package]\nname = \"pg\"\nversion = \"2.0.0\"\n")
	write("pg2/lib.vs", "public fun version(): i64 => 2\n")
	write("app/veles.toml", "[package]\nname = \"app\"\n[dependencies]\na = \"../a\"\nb = \"../b\"\nutil3 = \"../util\"\npg1 = \"../pg1\"\npg2 = \"../pg2\"\n")
	write("app/main.vs", "use io\nuse a\nuse b\nuse util3\nuse pg1\nuse pg2\n\nfun main() {\n  io.println(\"${a.fromA()} ${b.fromB()} ${util3.bump()} ${pg1.version()} ${pg2.version()}\")\n}\n")

	exe := filepath.Join(dir, "app.exe")
	if code := Run(Options{Path: filepath.Join(dir, "app"), Mode: "build", Output: exe}); code != 0 {
		t.Fatalf("build failed with exit %d", code)
	}
	if got := strings.TrimSpace(runExe(t, exe)); got != "1 2 3 1 2" {
		t.Errorf("got %q, want %q (one shared counter, two versions of pg)", got, "1 2 3 1 2")
	}
}
