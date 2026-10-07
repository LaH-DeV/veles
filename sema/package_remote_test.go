package sema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

// A registry or git dependency is read and checked, and importing it says
// plainly that fetching is not built yet (D138, stage d) — not "unknown
// module", which would send the reader looking for a typo.
func TestRemoteDependencyIsNotFetchedYet(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("veles.toml", "[package]\nname = \"app\"\n[dependencies]\nhttputil = { registry = \"acme/httputil\", version = \"1.4.2\" }\n")
	write("main.vs", "use io\nuse httputil\nfun main() { io.println(\"x\") }\n")
	diags := &source.Diagnostics{}
	pkg, err := LoadPackage(root, diags)
	if err != nil {
		t.Fatal(err)
	}
	pkg.NeedMain = true
	if !diags.HasErrors() {
		Check(pkg, diags, false)
	}
	out := diags.Render()
	if !strings.Contains(out, "dependency 'httputil' comes from registry 'acme/httputil', which was not fetched") {
		t.Errorf("got: %s", out)
	}
}
