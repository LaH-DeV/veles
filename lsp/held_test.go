package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const heldSrc = `use io

struct Server {
  name: string
  serving: Task<()>

  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
}

fun serve() {
  loop {
    await sleep(Duration.millis(5))
  }
}

fun serving(name: string): Server => Server(name, serving: async serve())

fun main() {
  with srv = serving("api")
  io.println(srv.name)
}
`

// D111: hovering a type that holds a task, or a function returning one,
// says to receive it with ` + "`with`" + `; hovering the ` + "`with`" + ` says the tasks are
// joined before the close.
func TestHeldInTheEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(heldSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": heldSrc}})
	doc := map[string]any{"uri": uri}
	hover := func(needle string, skip int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": at(t, heldSrc, needle, skip)})
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover("Server(name,", 0); !strings.Contains(h, "Holds a running task: receive it with `with`") {
		t.Errorf("hover on a task-holding type: %s", h)
	}
	if h := hover("serving(\"api\")", 0); !strings.Contains(h, "Holds a running task") {
		t.Errorf("hover on a function returning one: %s", h)
	}
	if h := hover("with srv", 0); !strings.Contains(h, "`srv` is closed after its tasks are cancelled and joined at the end of this block") {
		t.Errorf("hover on the with: %s", h)
	}
	res, _ := c.call("textDocument/inlayHint", map[string]any{"textDocument": doc,
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	var hints []inlayHint
	if err := json.Unmarshal(res, &hints); err != nil {
		t.Fatalf("inlayHint: %s", res)
	}
	if got, want := applyHints(heldSrc, hints), "}« stops tasks of, then closes srv»\n"; !strings.Contains(got, want) {
		t.Errorf("missing %q in\n%s", want, got)
	}
}
