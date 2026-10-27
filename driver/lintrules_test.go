package driver

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyDir(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A lint rule written in Veles (D177, examples/features/lintrules): `veles check` runs
// the rules the manifest names, and `--fix` applies the fix a rule reports.
func TestLibraryLintFix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	copyDir(t, "../examples/features/lintrules", root)
	app := filepath.Join(root, "app")
	if code := Run(Options{Path: app, Mode: "check"}); code == 0 {
		t.Fatal("check passed although lower_case_functions is \"deny\"")
	}
	if code := Run(Options{Path: app, Mode: "check", Fix: true}); code == 0 {
		t.Fatal("check --fix passed although a denied report has no fix")
	}
	data, err := os.ReadFile(filepath.Join(app, "main.vs"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sleep") {
		t.Errorf("the no_sleep fix was not applied:\n%s", data)
	}
	if !strings.Contains(string(data), "fun Shout") {
		t.Errorf("--fix changed more than the rule's edit:\n%s", data)
	}
}
