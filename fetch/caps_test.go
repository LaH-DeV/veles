package fetch

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/std"
)

func capsOf(t *testing.T, files map[string]string) Caps {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, files)
	var man *sema.Manifest
	if _, err := os.Stat(filepath.Join(dir, "veles.toml")); err == nil {
		m, err := sema.ReadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		man = m
	}
	c, err := PackageCaps(dir, man)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCapabilitiesComeFromSource(t *testing.T) {
	c := capsOf(t, map[string]string{
		"veles.toml": "[package]\nname = \"p\"\n[native]\nlibs = [\"z\"]\n",
		"lib.vs": `use io
use net
use log

extern "C" {
  fun getpid(): i32
}

public fun a(p: *raw u8): u8 => unsafe { p.read() }

public unsafe fun b(p: *raw u8): u8 => p.read()
`,
	})
	got := strings.Join(c.Names(), " ")
	if got != "unsafe extern native net" { // io and log are none of them
		t.Errorf("capabilities: %s", got)
	}
	if ev := c["net"]; len(ev) != 1 || ev[0] != "lib.vs:2 use net" {
		t.Errorf("net evidence: %v", ev)
	}
	if ev := c["unsafe"]; len(ev) != 2 || !strings.Contains(ev[0], "unsafe block") || !strings.Contains(ev[1], "unsafe fun b") {
		t.Errorf("unsafe evidence: %v", ev)
	}
	if ev := c["extern"]; len(ev) != 1 || ev[0] != "lib.vs:5 extern block" {
		t.Errorf("extern evidence: %v", ev)
	}
}

func TestCapabilitiesLeaveOutWhatIsNotBuilt(t *testing.T) {
	c := capsOf(t, map[string]string{
		"lib.vs":              "use io\npublic fun f() {}\n",
		"lib.test.vs":         "use net\nuse ffi\n",     // tests are not built into dependents
		"net/lib.vs":          "public fun mine() {}\n", // the package's own module called net
		"main.vs":             "use net\n",              // …so this is not std's net
		".hidden/lib.vs":      "use ffi\n",              // a hidden directory cannot be imported
		"sub/deeper/thing.vs": "use fs\n",               // a real module below the root
	})
	if got := strings.Join(c.Names(), " "); got != "fs" {
		t.Errorf("capabilities: %s", got)
	}
	if ev := c["fs"]; len(ev) != 1 || ev[0] != "sub/deeper/thing.vs:1 use fs" {
		t.Errorf("fs evidence: %v", ev)
	}
}

// Nowhere to hide: the loader reads any directory below the root as a module,
// whatever else is in it, so a capability there counts; and a directory that
// holds no sources is not a module, so it cannot shadow std's.
func TestCapabilitiesCannotBeHidden(t *testing.T) {
	c := capsOf(t, map[string]string{
		"lib.vs":              "use fs\n", // fs is std's: the fs/ directory below has no sources
		"fs/README.md":        "not a module\n",
		"other/veles.toml":    "[package]\nname = \"other\"\n", // another package's manifest does not make it unreachable
		"other/lib.vs":        "use ffi\n",
		"vendor/x/lib.vs":     "use net\n", // nor does the name vendor
		"node_modules/y/a.vs": "use os\n",
	})
	if got := strings.Join(c.Names(), " "); got != "net fs os ffi" { // the fixed order of sema.Capabilities
		t.Errorf("capabilities: %s", got)
	}
	for _, want := range []string{"net", "fs", "os", "ffi"} {
		if _, ok := c[want]; !ok {
			t.Errorf("%s was hidden", want)
		}
	}
}

// Only the modules whose purpose is a capability give it: http reads the
// clock and the environment inside, which is not the program's doing.
func TestCapabilitiesOfStdModules(t *testing.T) {
	for module, want := range map[string]string{
		"io": "", "json": "", "log": "", "http": "net", "tls": "net", "db": "net",
		"fs": "fs", "config": "fs os", "os": "os", "ffi": "ffi", "crypto": "", "time": "",
	} {
		var names []string
		for _, n := range sema.Capabilities {
			for _, c := range stdModuleCaps[module] {
				if c == n {
					names = append(names, n)
				}
			}
		}
		if got := strings.Join(names, " "); got != want {
			t.Errorf("use %s: %q, want %q", module, got, want)
		}
	}
	// every module the table names exists in std
	for module := range stdModuleCaps {
		if entries, err := fs.ReadDir(std.FS, module); err != nil || len(entries) == 0 {
			t.Errorf("stdModuleCaps names %q, which is not a std module", module)
		}
	}
}

func TestEvidenceIsCapped(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 9; i++ {
		sb.WriteString("public unsafe fun f" + string(rune('a'+i)) + "(): i64 = 1\n")
	}
	c := capsOf(t, map[string]string{"lib.vs": sb.String()})
	ev := c["unsafe"]
	if len(ev) != maxEvidence || ev[maxEvidence-1] != "…" {
		t.Errorf("evidence: %v", ev)
	}
}

func TestPolicyDenied(t *testing.T) {
	p := sema.ManifestPolicy{Deny: []string{"net", "unsafe"}, Allow: map[string][]string{"net": {"acme/httputil"}}}
	for _, c := range []struct {
		capability string
		names      []string
		want       bool
	}{
		{"net", []string{"acme/httputil", "httputil"}, false},
		{"net", []string{"other/thing", "thing"}, true},
		{"unsafe", []string{"acme/httputil"}, true}, // allowed for net only
		{"fs", []string{"other/thing"}, false},      // not denied
		{"net", []string{"https://x/y", "httputil"}, true},
	} {
		if got := p.Denied(c.capability, c.names...); got != c.want {
			t.Errorf("%s %v: %v", c.capability, c.names, got)
		}
	}
}
