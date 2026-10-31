// Package docs holds the tutorials. This test keeps them honest: every
// ```veles block in every Markdown file under docs/ is type-checked with the
// bootstrap compiler, blocks with a `main` are also run, and when a block is
// followed by an "Output:" ```text block the program's output must match it.
//
// A block whose first line is `// fragment` is a snippet that is not a
// complete module and is only shown, not compiled.
package docs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/internal/buildtest"
)

// fence is a Veles block — on its own, or the `[Veles]` tab of a
// `::: code-group` that shows the same program in other languages — and the
// Output block after it, or after the group's closing `:::`. The other tabs
// are not compiled; each must carry a `[Label]` and its lines cannot start a
// fence, so a match never runs past its own group.
var fence = regexp.MustCompile("(?s)```veles(?: \\[[^\\]\\n]*\\])?\\n(.*?)```" +
	"(?:(?:\\s*\\n" + otherTab + ")*\\s*\\n:::)?" +
	"(?:\\s*\\n\\s*Output:\\s*\\n```text\\n(.*?)```)?")

// otherTab is a code-group tab in another language: "```go [Go]", lines
// that do not open or close a fence, "```".
const otherTab = "```[^\\s`]+ \\[[^\\]\\n]+\\]\\n(?:(?:[^`\\n]|`[^`\\n]|``[^`\\n])[^\\n]*\\n|\\n)*```"

func TestMain(m *testing.M) { buildtest.Main(m) }

// TestDocs runs the blocks in parallel, each in a directory of its own that
// is also its working directory and its os.tempDir(), so two blocks never
// see each other's files.
func TestDocs(t *testing.T) {
	veles := buildtest.Compiler(t)
	var mds []string
	filepath.WalkDir("./documentation", func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
			mds = append(mds, path)
		}
		return nil
	})
	if len(mds) == 0 {
		t.Fatal("no markdown files found")
	}
	for _, md := range mds {
		data, err := os.ReadFile(md)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		for i, m := range fence.FindAllStringSubmatch(text, -1) {
			code, want := m[1], m[2]
			if strings.HasPrefix(strings.TrimSpace(code), "// fragment") {
				continue
			}
			name := strings.TrimSuffix(filepath.Base(md), ".md") + "/" + itoa(i+1)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				src := filepath.Join(dir, "src")
				tmp := filepath.Join(dir, "tmp")
				for _, d := range []string{src, tmp} {
					if err := os.Mkdir(d, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(src, "main.vs"), []byte(code), 0o644); err != nil {
					t.Fatal(err)
				}
				hasMain := strings.Contains(code, "fun main(")
				mode := "check"
				if hasMain {
					mode = "run"
				}
				cmd := exec.Command(veles, mode, src)
				cmd.Dir = tmp
				cmd.Env = append(os.Environ(), "TMPDIR="+tmp, "TMP="+tmp, "TEMP="+tmp)
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				if err != nil && !(hasMain && want != "" && (strings.Contains(stderr.String(), "panic") || strings.Contains(stderr.String(), "main failed"))) {
					t.Fatalf("veles %s failed: %v\n%s\n--- code ---\n%s", mode, err, stderr.String(), code)
				}
				if want != "" {
					got := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
					if strings.TrimRight(got, "\n") != strings.TrimRight(want, "\n") {
						t.Errorf("output mismatch\n--- got ---\n%s--- want ---\n%s--- code ---\n%s", got, want, code)
					}
				}
			})
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// A code group's Veles tab is the block that runs and its Output follows the
// group; a plain block followed by an unlabelled one never reaches a later
// group's `:::`.
func TestFenceReadsCodeGroups(t *testing.T) {
	group := "::: code-group\n\n```veles [Veles]\nfun main() { }\n```\n\n```go [Go]\nfunc main() {\n\t`raw`\n}\n```\n\n:::\n\nOutput:\n\n```text\nhi\n```\n"
	m := fence.FindAllStringSubmatch(group, -1)
	if len(m) != 1 || m[0][1] != "fun main() { }\n" || m[0][2] != "hi\n" {
		t.Fatalf("code group read as %q", m)
	}
	plain := "```veles\nfun a() { }\n```\n```text\nerror: …\n```\n\nprose\n\n" + group
	m = fence.FindAllStringSubmatch(plain, -1)
	if len(m) != 2 || m[0][1] != "fun a() { }\n" || m[0][2] != "" || m[1][2] != "hi\n" {
		t.Fatalf("a plain block ran into the group: %q", m)
	}
}
