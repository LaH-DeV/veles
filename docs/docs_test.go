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
	"runtime"
	"strings"
	"testing"
)

var fence = regexp.MustCompile("(?s)```veles\\n(.*?)```(?:\\s*\\n\\s*Output:\\s*\\n```text\\n(.*?)```)?")

func TestDocs(t *testing.T) {
	root, _ := filepath.Abs("..")
	veles := filepath.Join(root, "veles")
	if runtime.GOOS == "windows" {
		veles += ".exe"
	}
	if _, err := os.Stat(veles); err != nil {
		build := exec.Command("go", "build", "-o", veles, ".")
		build.Dir = root
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("building compiler: %v\n%s", err, out)
		}
	}
	var mds []string
	filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
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
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(code), 0o644); err != nil {
					t.Fatal(err)
				}
				hasMain := strings.Contains(code, "fun main(")
				mode := "check"
				if hasMain {
					mode = "run"
				}
				cmd := exec.Command(veles, mode, dir)
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
