package sema

import (
	"strings"
	"testing"
)

const copyManifest = "[package]\nname = \"app\"\nversion = \"0.1.0\"\n"

// D164 visibility: a field the call site cannot see cannot be named in a
// `copy`, and keeps its value; D163 counts only the parameters the call
// site may supply, so a private field with a default does not stop a
// one-parameter construction from outside the type — but does inside it.
func TestCopyAndPositionalFollowVisibility(t *testing.T) {
	lib := `struct Wrap {
  value: i64
  private hits: i64 = 0
  fun again(): Wrap => Wrap(this.value)
  fun counted(): Wrap => this.copy(hits: this.hits + 1)
}
`
	clean := checkTree(t, map[string]string{
		"veles.toml": copyManifest,
		"geo/lib.vs": strings.Replace(lib, "  fun again(): Wrap => Wrap(this.value)\n", "", 1),
		"main.vs":    "use io { println }\nuse geo\nfun main() {\n  val w = geo.Wrap(5)\n  val v = w.copy(value: 6)\n  println(\"${v.value}\")\n}\n",
	})
	if strings.Contains(clean, "error") {
		t.Errorf("positional construction and copy from outside should be clean:\n%s", clean)
	}
	inside := checkTree(t, map[string]string{
		"veles.toml": copyManifest,
		"geo/lib.vs": lib,
		"main.vs":    "use geo\nfun main() {\n  val _ = geo.Wrap(5)\n}\n",
	})
	if !strings.Contains(inside, "construct 'Wrap' by field name: 'value: this.value'") {
		t.Errorf("inside the type a private field with a default is a second parameter:\n%s", inside)
	}
	hidden := checkTree(t, map[string]string{
		"veles.toml": copyManifest,
		"geo/lib.vs": strings.Replace(lib, "  fun again(): Wrap => Wrap(this.value)\n", "", 1),
		"main.vs":    "use geo\nfun main() {\n  val _ = geo.Wrap(5).copy(hits: 9)\n}\n",
	})
	if !strings.Contains(hidden, "field 'hits' of 'Wrap' is not visible here, so 'copy' cannot change it") {
		t.Errorf("a private field cannot be named in copy from outside:\n%s", hidden)
	}
}
