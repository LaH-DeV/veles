package selfhost

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/sema"
)

// tomlCases are the texts the TOML reader is compared on beyond the repository's own
// manifests: what the manifest accepts, and every way the reader refuses.
var tomlCases = map[string]string{
	"empty":            "",
	"comments-only":    "# nothing\n   # here\n",
	"bom":              "\xEF\xBB\xBF[package]\nname = \"x\"\n",
	"crlf":             "[package]\r\nname = \"x\"\r\nversion = \"1.0.0\"\r\n",
	"strings":          "a = \"plain\"\nb = 'literal \\n stays'\nc = \"tab\\there \\\"quoted\\\" \\\\ back\"\nd = \"\\u00e9\\u4e2d\\u0041\"\n",
	"ints":             "a = 1\nb = -7\nc = +5\nd = 1_000_000\ne = 9223372036854775807\nf = -9223372036854775808\ng = 0\n",
	"bools":            "a = true\nb = false\n",
	"lists":            "a = []\nb = [1, 2, 3]\nc = [\n  \"x\",\n  \"y\",  # why\n]\nd = [[1], [2, 3]]\ne = [ { a = 1 }, { b = \"x\" } ]\n",
	"inline-tables":    "a = {}\nb = { x = 1, y = \"two\", z = [1, 2] }\nc = { n = { m = true } }\n",
	"tables":           "[a]\nx = 1\n[a.b]\ny = 2\n[c]\nz = 3\n",
	"parent-after":     "[a.b]\ny = 2\n[a]\nx = 1\n",
	"dotted-keys":      "a.b = 1\na.c = 2\n[t]\nx.y.z = \"deep\"\n",
	"dotted-then-head": "a.b = 1\n[a]\nc = 2\n",
	"quoted-keys":      "\"a b\" = 1\n'c.d' = 2\n[\"t u\"]\n\"k\" = 3\n",
	"spaces":           "[ a . b ]\n x  =  1 \t# trailing\n",
	"no-final-newline": "a = 1",
	"manifest": "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n\n[dependencies]\nmath = { path = \"../math\" }\nutil = \"1.2.3\"\n" +
		"web = { git = \"https://example.com/web\", rev = \"abc123\" }\n\n[format]\nindent = 4\nwidth = 100\n",

	"err-array-of-tables":  "[[a]]\nx = 1\n",
	"err-unclosed-header":  "[a\nx = 1\n",
	"err-bad-header-key":   "[a.]\n",
	"err-table-twice":      "[a]\nx = 1\n[a]\ny = 2\n",
	"err-table-twice-deep": "[a.b]\n[a.b]\n",
	"err-key-twice":        "a = 1\na = 2\n",
	"err-key-twice-dotted": "a.b = 1\na.b = 2\n",
	"err-inline-twice":     "a = { x = 1, x = 2 }\n",
	"err-not-a-table":      "a = 1\n[a]\n",
	"err-not-a-table-path": "a = 1\n[a.b]\n",
	"err-dotted-not-table": "a = 1\na.b = 2\n",
	"err-inline-as-table":  "a = { x = 1 }\n[a]\n",
	"err-no-equals":        "a 1\n",
	"err-no-key":           "= 1\n",
	"err-no-value":         "a =\n",
	"err-no-value-eof":     "a = ",
	"err-text-after":       "a = 1 2\n",
	"err-bare-word":        "a = hello\n",
	"err-float":            "a = 1.5\n",
	"err-date":             "a = 1979-05-27\n",
	"err-int-overflow":     "a = 9223372036854775808\n",
	"err-int-low-overflow": "a = -9223372036854775809\n",
	"err-sign-only":        "a = -\n",
	"err-underscores":      "a = _\n",
	"err-unterminated":     "a = \"open\n",
	"err-unterminated-eof": "a = \"open",
	"err-literal-eol":      "a = 'open\nb = 1\n",
	"err-escape":           "a = \"\\q\"\n",
	"err-escape-eof":       "a = \"\\",
	"err-escape-non-ascii": "a = \"\\é\"\n",
	"err-u-short":          "a = \"\\u12\"\n",
	"err-u-short-eof":      "a = \"\\u12",
	"err-u-bad":            "a = \"\\u12zz\"\n",
	"err-u-surrogate":      "a = \"\\ud800\"\n",
	"err-u-sign":           "a = \"\\u+123\"\n",
	"err-list-unclosed":    "a = [1, 2\nb = 1\n",
	"err-list-no-comma":    "a = [1 2]\n",
	"err-inline-newline":   "a = {\n x = 1 }\n",
	"err-inline-unclosed":  "a = { x = 1\n",
	"err-inline-no-equals": "a = { x }\n",
	"err-inline-no-comma":  "a = { x = 1 y = 2 }\n",
	"err-line-number":      "a = 1\n\n\n# c\nb = [\n  1,\n  oops,\n]\n",
}

// TestR1Toml is the first gate of phase R1 (veles-selfhost-sema-plan.md): the
// TOML reader written in Veles reads every manifest of the repository, and
// each case above, into the tree sema.TOMLDump prints — or refuses it with
// the same line and message.
func TestR1Toml(t *testing.T) {
	exe := buildSelfhost(t)
	type input struct{ name, path, text string }
	var inputs []input
	dir := t.TempDir()
	for name, text := range tomlCases {
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
			compare(t, exe, "--toml", in.path, sema.TOMLDump(in.text))
		})
	}
}
