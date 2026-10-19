package selfhost

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

// manifestCases are the manifests the manifest reader is compared on beyond
// the repository's own: every table, every key, and every way each refuses.
var manifestCases = map[string]string{
	"minimal":         "[package]\nname = \"app\"\n",
	"full-package":    "[package]\nname = \"app\"\nversion = \"1.2.3-rc.1\"\ndescription = \"d\"\nlicense = \"MIT\"\nveles = \"0.70\"\nregistry = \"acme/app\"\n",
	"veles-full":      "[package]\nname = \"app\"\nveles = \"0.70.1\"\n",
	"deps":            "[package]\nname = \"app\"\n[dependencies]\nmath = \"../math\"\nhttp = { registry = \"acme/http\", version = \"1.4.2\" }\npg = { git = \"https://x.org/pg\", version = \"2.1.0\" }\nfast = { git = \"https://x.org/f\", commit = \"3f2a9c1\" }\nlocal = { path = \"vendor/local\" }\n[dev-dependencies]\nclock = { registry = \"lah/clock\", version = \"0.3.0\" }\n",
	"workspace":       "[workspace]\nmembers = [\"app\", \"libs/math\", \"./tools/../cli\"]\n",
	"workspace-both":  "[package]\nname = \"root\"\n[workspace]\nmembers = [\"a\"]\n",
	"format":          "[package]\nname = \"a\"\n[format]\nindent = 4\nmax_blank_lines = 2\nimports = \"merged\"\ntrailing_commas = \"never\"\nline_endings = \"crlf\"\n",
	"format-tab":      "[package]\nname = \"a\"\n[format]\nindent = \"tab\"\nmax_blank_lines = \"3\"\n",
	"format-indent-s": "[package]\nname = \"a\"\n[format]\nindent = \"2\"\n",
	"lint":            "[package]\nname = \"a\"\n[lint]\nmax_width = 100\nimplicit_return = \"lambda\"\nlong_line = \"warn\"\nacme-lints.no_sleep = \"deny\"\n\"b.c\" = \"allow\"\n",
	"native":          "[package]\nname = \"a\"\n[native]\nlibs = [\"pq\", \"z\"]\nstatic-libs = [\"m\"]\nlib-paths = [\"vendor/lib\"]\npkg-config = [\"libpq\"]\n",
	"policy":          "[package]\nname = \"a\"\n[policy]\ndeny = [\"net\", \"unsafe\"]\nallow = { net = [\"acme/http\"], reviewed = [\"x/y\"] }\ntrust = { alice = \"ed25519:AAAA\" }\nrequire = { reviewed = 2 }\n",
	"registry":        "[package]\nname = \"a\"\n[registry]\nurl = \"https://r.example.com\"\nkey = \"ed25519:BBBB\"\n",
	"runtime":         "[package]\nname = \"a\"\n[runtime]\nthreads = 8\n",
	"runtime-auto":    "[package]\nname = \"a\"\n[runtime]\nthreads = \"auto\"\n",
	"empty-tables":    "[package]\nname = \"a\"\n[dependencies]\n[format]\n[lint]\n[native]\n[policy]\n[registry]\n[runtime]\n",

	"blank-str-prefix":           "[package]\nname = \"a\"\n[format]\nmax_blank_lines = \"  2 lines\"\n",
	"err-blank-overflow":         "[package]\nname = \"a\"\n[format]\nmax_blank_lines = \"99999999999999999999\"\n",
	"err-blank-underscore-first": "[package]\nname = \"a\"\n[format]\nmax_blank_lines = \"_1\"\n",
	"err-lint-typo-swap":         "[package]\nname = \"a\"\n[lint]\nlnog_line = \"warn\"\n",
	"err-lint-typo-case":         "[package]\nname = \"a\"\n[lint]\nLong_Line = \"warn\"\n",
	"err-lint-typo-prefix":       "[package]\nname = \"a\"\n[lint]\nimplicit_retur = \"warn\"\n",
	"err-lint-short":             "[package]\nname = \"a\"\n[lint]\nlo = \"warn\"\n",
	"blank-str-underscore":       "[package]\nname = \"a\"\n[format]\nmax_blank_lines = \"1_0\"\n",

	"err-toml":                 "[package\nname = \"a\"\n",
	"err-no-name":              "[package]\nversion = \"1.0.0\"\n",
	"err-empty":                "",
	"err-outside":              "name = \"a\"\n",
	"err-unknown-table":        "[pakage]\nname = \"a\"\n",
	"err-package-type":         "[package]\nname = 5\n",
	"err-package-key":          "[package]\nname = \"a\"\nauthor = \"x\"\n",
	"err-package-exports":      "[package]\nname = \"a\"\nexports = [\"x\"]\n",
	"err-version":              "[package]\nname = \"a\"\nversion = \"1.2\"\n",
	"err-version-zero":         "[package]\nname = \"a\"\nversion = \"01.2.3\"\n",
	"err-version-build":        "[package]\nname = \"a\"\nversion = \"1.2.3+b\"\n",
	"err-version-pre":          "[package]\nname = \"a\"\nversion = \"1.2.3-\"\n",
	"err-version-pre-empty-id": "[package]\nname = \"a\"\nversion = \"1.2.3-a..b\"\n",
	"err-version-pre-zero-id":  "[package]\nname = \"a\"\nversion = \"1.2.3-a.01\"\n",
	"version-pre-zero":         "[package]\nname = \"a\"\nversion = \"1.2.3-a.0\"\n",
	"err-version-big":          "[package]\nname = \"a\"\nversion = \"4294967296.0.0\"\n",
	"err-veles":                "[package]\nname = \"a\"\nveles = \"seventy\"\n",
	"err-registry-name":        "[package]\nname = \"a\"\nregistry = \"Acme/App\"\n",
	"err-dep-name":             "[package]\nname = \"a\"\n[dependencies]\n\"1x\" = \"../x\"\n",
	"err-dep-version-string":   "[package]\nname = \"a\"\n[dependencies]\nx = \"1.2.3\"\n",
	"err-dep-type":             "[package]\nname = \"a\"\n[dependencies]\nx = 3\n",
	"err-dep-key":              "[package]\nname = \"a\"\n[dependencies]\nx = { path = \"p\", branch = \"main\" }\n",
	"err-dep-key-type":         "[package]\nname = \"a\"\n[dependencies]\nx = { path = 4 }\n",
	"err-dep-two-sources":      "[package]\nname = \"a\"\n[dependencies]\nx = { path = \"p\", git = \"g\", version = \"1.0.0\" }\n",
	"err-dep-no-source":        "[package]\nname = \"a\"\n[dependencies]\nx = { version = \"1.0.0\" }\n",
	"err-dep-v":                "[package]\nname = \"a\"\n[dependencies]\nx = { registry = \"a/b\", version = \"v1.0.0\" }\n",
	"err-dep-bad-version":      "[package]\nname = \"a\"\n[dependencies]\nx = { registry = \"a/b\", version = \"1\" }\n",
	"err-dep-commit":           "[package]\nname = \"a\"\n[dependencies]\nx = { git = \"g\", commit = \"xyz\" }\n",
	"err-dep-path-version":     "[package]\nname = \"a\"\n[dependencies]\nx = { path = \"p\", version = \"1.0.0\" }\n",
	"err-dep-registry-name":    "[package]\nname = \"a\"\n[dependencies]\nx = { registry = \"ab\", version = \"1.0.0\" }\n",
	"err-dep-registry-commit":  "[package]\nname = \"a\"\n[dependencies]\nx = { registry = \"a/b\", commit = \"3f2a9c1\" }\n",
	"err-dep-registry-nover":   "[package]\nname = \"a\"\n[dependencies]\nx = { registry = \"a/b\" }\n",
	"err-dep-git-both":         "[package]\nname = \"a\"\n[dependencies]\nx = { git = \"g\", version = \"1.0.0\", commit = \"3f2a9c1\" }\n",
	"err-dep-git-neither":      "[package]\nname = \"a\"\n[dependencies]\nx = { git = \"g\" }\n",
	"err-ws-key":               "[workspace]\nmembers = [\"a\"]\nroot = \".\"\n",
	"err-ws-missing":           "[workspace]\n",
	"err-ws-empty":             "[workspace]\nmembers = []\n",
	"err-ws-type":              "[workspace]\nmembers = \"a\"\n",
	"err-ws-item":              "[workspace]\nmembers = [1]\n",
	"err-ws-dot":               "[workspace]\nmembers = [\".\"]\n",
	"err-ws-empty-member":      "[workspace]\nmembers = [\"\"]\n",
	"err-ws-abs":               "[workspace]\nmembers = [\"/x\"]\n",
	"err-ws-abs-backslash":     "[workspace]\nmembers = ['\\x']\n",
	"err-ws-abs-drive":         "[workspace]\nmembers = [\"C:/x\"]\n",
	"err-ws-up":                "[workspace]\nmembers = [\"../x\"]\n",
	"err-ws-up-deep":           "[workspace]\nmembers = [\"a/../../x\"]\n",
	"err-ws-up-backslash":      "[workspace]\nmembers = ['..\\x']\n",
	"err-ws-glob":              "[workspace]\nmembers = [\"libs/*\"]\n",
	"err-ws-twice":             "[workspace]\nmembers = [\"a\", \"./a\"]\n",
	"err-runtime-key":          "[package]\nname = \"a\"\n[runtime]\nworkers = 2\n",
	"err-runtime-zero":         "[package]\nname = \"a\"\n[runtime]\nthreads = 0\n",
	"err-runtime-big":          "[package]\nname = \"a\"\n[runtime]\nthreads = 257\n",
	"err-runtime-str":          "[package]\nname = \"a\"\n[runtime]\nthreads = \"many\"\n",
	"err-format-key":           "[package]\nname = \"a\"\n[format]\nwidth = 3\n",
	"err-format-indent-type":   "[package]\nname = \"a\"\n[format]\nindent = true\n",
	"err-format-indent-9":      "[package]\nname = \"a\"\n[format]\nindent = 9\n",
	"err-format-indent-0":      "[package]\nname = \"a\"\n[format]\nindent = 0\n",
	"err-format-indent-pad":    "[package]\nname = \"a\"\n[format]\nindent = \"04\"\n",
	"err-format-indent-word":   "[package]\nname = \"a\"\n[format]\nindent = \"wide\"\n",
	"err-format-blank-0":       "[package]\nname = \"a\"\n[format]\nmax_blank_lines = 0\n",
	"err-format-blank-str":     "[package]\nname = \"a\"\n[format]\nmax_blank_lines = \"many\"\n",
	"err-format-blank-bool":    "[package]\nname = \"a\"\n[format]\nmax_blank_lines = true\n",
	"err-format-imports":       "[package]\nname = \"a\"\n[format]\nimports = \"sorted\"\n",
	"err-format-commas":        "[package]\nname = \"a\"\n[format]\ntrailing_commas = \"always\"\n",
	"err-format-endings":       "[package]\nname = \"a\"\n[format]\nline_endings = \"cr\"\n",
	"err-lint-width":           "[package]\nname = \"a\"\n[lint]\nmax_width = 10\n",
	"err-lint-width-type":      "[package]\nname = \"a\"\n[lint]\nmax_width = \"wide\"\n",
	"err-lint-return":          "[package]\nname = \"a\"\n[lint]\nimplicit_return = \"always\"\n",
	"err-lint-return-type":     "[package]\nname = \"a\"\n[lint]\nimplicit_return = 3\n",
	"err-lint-unknown":         "[package]\nname = \"a\"\n[lint]\nzzzzzzzz = \"warn\"\n",
	"err-lint-typo":            "[package]\nname = \"a\"\n[lint]\nlong_lin = \"warn\"\n",
	"err-lint-level":           "[package]\nname = \"a\"\n[lint]\nlong_line = \"loud\"\n",
	"err-lint-level-type":      "[package]\nname = \"a\"\n[lint]\nlong_line = 1\n",
	"err-lint-lib-level":       "[package]\nname = \"a\"\n[lint]\nacme.rule = \"loud\"\n",
	"err-lint-lib-table":       "[package]\nname = \"a\"\n[lint.acme]\nrule = 5\n",
	"err-lint-lib-dot":         "[package]\nname = \"a\"\n[lint]\n\".rule\" = \"warn\"\n",
	"err-lint-lib-dot-end":     "[package]\nname = \"a\"\n[lint]\n\"acme.\" = \"warn\"\n",
	"err-native-type":          "[package]\nname = \"a\"\n[native]\nlibs = \"pq\"\n",
	"err-native-key":           "[package]\nname = \"a\"\n[native]\nframeworks = [\"x\"]\n",
	"err-native-key-list":      "[package]\nname = \"a\"\n[native]\nframeworks = 3\n",
	"err-native-item":          "[package]\nname = \"a\"\n[native]\nlibs = [1]\n",
	"err-native-option":        "[package]\nname = \"a\"\n[native]\npkg-config = [\"-lx\"]\n",
	"err-policy-key":           "[package]\nname = \"a\"\n[policy]\nlevel = 1\n",
	"err-policy-deny":          "[package]\nname = \"a\"\n[policy]\ndeny = [\"network\"]\n",
	"err-policy-deny-case":     "[package]\nname = \"a\"\n[policy]\ndeny = [\"Net\"]\n",
	"err-lint-level-case":      "[package]\nname = \"a\"\n[lint]\nlong_line = \"Deny\"\n",
	"err-policy-deny-type":     "[package]\nname = \"a\"\n[policy]\ndeny = \"net\"\n",
	"err-policy-allow-type":    "[package]\nname = \"a\"\n[policy]\nallow = [\"net\"]\n",
	"err-policy-allow-item":    "[package]\nname = \"a\"\n[policy]\nallow = { net = \"x\" }\n",
	"err-policy-allow-name":    "[package]\nname = \"a\"\n[policy]\nallow = { zebra = [\"x\"], apple = [\"y\"] }\n",
	"err-policy-trust-type":    "[package]\nname = \"a\"\n[policy]\ntrust = \"alice\"\n",
	"err-policy-trust-key":     "[package]\nname = \"a\"\n[policy]\ntrust = { alice = \"rsa:1\" }\n",
	"err-policy-require":       "[package]\nname = \"a\"\n[policy]\ntrust = { a = \"ed25519:1\" }\nrequire = { reviewed = 0 }\n",
	"err-policy-require-type":  "[package]\nname = \"a\"\n[policy]\nrequire = 1\n",
	"err-policy-no-trust":      "[package]\nname = \"a\"\n[policy]\nrequire = { reviewed = 1 }\n",
	"err-registry-type":        "[package]\nname = \"a\"\n[registry]\nurl = 1\n",
	"err-registry-url":         "[package]\nname = \"a\"\n[registry]\nurl = \"ftp://x\"\n",
	"err-registry-key":         "[package]\nname = \"a\"\n[registry]\nkey = \"rsa:x\"\n",
	"err-registry-key-name":    "[package]\nname = \"a\"\n[registry]\nmirror = \"x\"\n",
}

// TestR1Manifest is the second gate of phase R1: the manifest reader written
// in Veles makes of each manifest what parseManifest makes — every field, or
// the same error with the same file, line and message. The cases above and the
// repository's own manifests are the corpus.
func TestR1Manifest(t *testing.T) {
	exe := buildSelfhost(t)
	type input struct{ name, path, text string }
	var inputs []input
	dir := t.TempDir()
	for name, text := range manifestCases {
		path := filepath.Join(dir, name+".toml")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, input{"case/" + name, path, text})
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
			return fs.SkipDir
		}
		if d.IsDir() || d.Name() != "veles.toml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		inputs = append(inputs, input{"manifest/" + filepath.ToSlash(rel), path, string(data)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].name < inputs[j].name })
	for _, in := range inputs {
		in := in
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()
			compareManifest(t, exe, in.path, sema.ManifestDump(in.path, in.text))
		})
	}
}

// compareManifest runs `selfhost --manifest path` and compares what it prints.
func compareManifest(t *testing.T, exe, path, want string) {
	t.Helper()
	cmd := exec.Command(exe, "--manifest", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("--manifest: %v\n%s", err, stderr.String())
	}
	got := stdout.String()
	if got == want {
		return
	}
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			t.Fatalf("--manifest: line %d differs\nveles: %q\ngo:    %q", i+1, gl, wl)
		}
	}
}

// TestR1Lints: the registry of lints written in Veles (selfhost/linter) is the
// bootstrap compiler's, entry for entry: name, default level and documentation.
func TestR1Lints(t *testing.T) {
	exe := buildSelfhost(t)
	out, err := exec.Command(exe, "--lints").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), sema.LintsDump(); got != want {
		t.Errorf("the lint registry differs:\nveles:\n%s\ngo:\n%s", got, want)
	}
}
