package driver

import (
	"os"
	"path/filepath"
	"testing"
)

// D188, superseded by D231: a collection is a value, so freeze() is gone
// and its call is an error (with a fix that drops it). The run-time
// behaviour it had — panics on a change through an old name — no longer
// exists: a change through another name changes that name's copy.
func TestFreezeRemoved(t *testing.T) {
	t.Parallel()
	for _, src := range []string{
		"fun main() {\n  val xs: List<i64> = [1]\n  io.println(\"${xs.freeze()}\")\n}\n",
		"fun main() {\n  val m: Map<string, i64> = [\"a\": 1]\n  io.println(\"${m.freeze()}\")\n}\n",
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte("use io\n\n"+src), 0o644); err != nil {
			t.Fatal(err)
		}
		if code := Run(Options{Path: filepath.Join(dir, "main.vs"), Mode: "check"}); code == 0 {
			t.Errorf("freeze() should be an error:\n%s", src)
		}
	}
}
