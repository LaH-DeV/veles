package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const withSrc = `use io

struct Res {
  name: string

  implement Closeable {
    fun close() { io.println("close ${this.name}") }
  }
}

fun serve() {
  loop {
    await sleep(Duration.millis(5))
  }
}

fun main() {
  with listener = Res(name: "listener")
  with server = async serve()
  with first = Res(name: "first")
  with second = Res(name: "second")
  io.println("${first.name} ${second.name} ${listener.name}")
}

fun awaited(flag: bool) {
  with r = Res(name: "r")
  with done = async serve()
  with maybe = async serve()
  if (flag) await maybe
  await done
}
`

// D100: hovering `with` says where the resource closes, and an inlay hint
// after the block's closing brace lists what closes there, in order.
func TestWithInTheEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(withSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": withSrc}})
	doc := map[string]any{"uri": uri}
	hover := func(needle string) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": at(t, withSrc, needle, 0)})
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover("with first"); !strings.Contains(h, "with first: Res") || !strings.Contains(h, "`first` is closed at the end of this block, line 23") {
		t.Errorf("hover on with: %s", h)
	}
	if h := hover("with server"); !strings.Contains(h, "`server` is cancelled, then joined") {
		t.Errorf("hover on a with-task: %s", h)
	}
	res, _ := c.call("textDocument/inlayHint", map[string]any{"textDocument": doc,
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	var hints []inlayHint
	if err := json.Unmarshal(res, &hints); err != nil {
		t.Fatalf("inlayHint: %s", res)
	}
	got := applyHints(withSrc, hints)
	if want := "}« closes second, first; cancels server; closes listener»\n"; !strings.Contains(got, want) {
		t.Errorf("missing %q in\n%s", want, got)
	}
	// an awaited task has finished by the end of its block; one awaited
	// only on some path may still be running
	if want := "  await done\n}« cancels maybe; closes r»\n"; !strings.Contains(got, want) {
		t.Errorf("missing %q in\n%s", want, got)
	}
}

const lockSrc = `use io

struct Notes {
  var count: i64 = 0
}

fun main() {
  val notes = Mutex(value: Notes())
  val sem = Res2()
  with n = notes.lock()
  with sem
  n.count += 1
  io.println("${n.count}")
}

struct Res2 {
  implement Closeable {
    fun close() { }
  }
}
`

// D107/D109: hovering the `with` of a lock shows the pointer it binds and
// that the region may not suspend; the inlay hint says the brace unlocks
// it, and names an item without a name by its expression.
func TestLockInTheEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(lockSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": lockSrc}})
	doc := map[string]any{"uri": uri}
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": at(t, lockSrc, "with n", 0)})
	var h struct{ Contents struct{ Value string } }
	json.Unmarshal(res, &h)
	if !strings.Contains(h.Contents.Value, "with n: *Notes") || !strings.Contains(h.Contents.Value, "nothing in between may suspend") {
		t.Errorf("hover on the with of a lock: %s", h.Contents.Value)
	}
	res, _ = c.call("textDocument/inlayHint", map[string]any{"textDocument": doc,
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	var hints []inlayHint
	if err := json.Unmarshal(res, &hints); err != nil {
		t.Fatalf("inlayHint: %s", res)
	}
	got := applyHints(lockSrc, hints)
	if want := "}« closes sem; unlocks notes»\n"; !strings.Contains(got, want) {
		t.Errorf("missing %q in\n%s", want, got)
	}
}
