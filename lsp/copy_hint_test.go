package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const copySrc = `use io

fun fill(into: *List<i64>, n: i64) {
  loop (i in 0..<n) into.push(i * i)
}

fun main() {
  val squares: List<i64> = []
  fill(&squares, 4)
  val before = squares
  squares.push(16)
  squares.push(25)
  loop (x in squares) {
    if (x < 3) squares.push(x + 100)
  }
  io.println("$before $squares")
  val held = squares
  squares.clear()
  io.println("$held $squares")
  val list2 = squares
  loop (x in squares) {
    list2.push(x)
  }
  io.println("$list2")
}
`

// D231 choice 3: a change that may copy a list shared since its last change
// gets a `copies` hint saying where it was shared, with why and what to do
// instead in its tooltip — on by default, off by the `copyHints` option or
// the `veles.inlayHints.copies` setting.
func TestCopyHints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(copySrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	hintsOf := func(c *client) []inlayHint {
		res, _ := c.call("textDocument/inlayHint", map[string]any{"textDocument": map[string]any{"uri": uri},
			"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
		var hints []inlayHint
		if err := json.Unmarshal(res, &hints); err != nil {
			t.Fatalf("inlayHint: %s", res)
		}
		var copies []inlayHint
		for _, h := range hints {
			if strings.HasPrefix(h.Label, "copies ") || strings.Contains(h.Label, " copied ") {
				copies = append(copies, h)
			}
		}
		return copies
	}
	open := func(c *client, options map[string]any) {
		c.call("initialize", map[string]any{"initializationOptions": options})
		c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": copySrc}})
	}

	c, stop := newClient(t)
	defer stop()
	open(c, map[string]any{})
	got := applyHints(copySrc, hintsOf(c))
	for _, want := range []string{
		// the full hint where the value was shared
		"val before = squares« squares copied at its first change, line 11»",
		"loop (x in squares) {« squares copied at its first change, line 14»",
		"val held = squares« squares copied as empty at line 18»",
		"val list2 = squares« list2 copied at its first change, line 22»",
		// a marker at a change two lines on or more, `once` in a loop
		"list2.push(x)« copies list2 once»",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	// no marker on the line after a share (its hint is in view); none after
	// the first change, which made squares its own again; none in fill,
	// whose push changes through a pointer
	for _, unwanted := range []string{"squares.push(16)«", "squares.push(25)«", "squares.push(x + 100)«", "squares.clear()«", "into.push(i * i)«"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected hint at %q in\n%s", unwanted, got)
		}
	}
	if h := hintsOf(c); len(h) == 0 || !strings.Contains(h[0].Tooltip.Value, "pass a pointer") {
		t.Errorf("tooltip: %+v", h)
	}
	c.notify("workspace/didChangeConfiguration", map[string]any{"settings": map[string]any{"veles": map[string]any{"inlayHints": map[string]any{"copies": false}}}})
	if h := hintsOf(c); len(h) != 0 {
		t.Errorf("hints after turning them off: %d", len(h))
	}
	c.notify("workspace/didChangeConfiguration", map[string]any{"settings": map[string]any{"veles": map[string]any{"inlayHints": map[string]any{"copies": true}}}})
	if h := hintsOf(c); len(h) != 5 {
		t.Errorf("hints after turning them on again: %d", len(h))
	}

	off, stopOff := newClient(t)
	defer stopOff()
	open(off, map[string]any{"copyHints": false})
	if h := hintsOf(off); len(h) != 0 {
		t.Errorf("hints with copyHints false: %+v", h)
	}
}

// A capture list's copy (D235) is copied on its first change in every
// call: the change says so in full, as a parameter's does; the original
// changed after the lambda was made points at the capture list.
func TestCaptureListCopyHints(t *testing.T) {
	src := "use io\n\nfun main() {\n  val names: List<string> = [\"x\"]\n  val grow = [names] (s: string) => {\n    names.push(s)\n    names.len()\n  }\n  io.println(\"${grow(\"a\")}\")\n  names.push(\"y\")\n  io.println(\"$names\")\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	res, _ := c.call("textDocument/inlayHint", map[string]any{"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	var hints []inlayHint
	if err := json.Unmarshal(res, &hints); err != nil {
		t.Fatalf("inlayHint: %s", res)
	}
	got := applyHints(src, hints)
	for _, want := range []string{
		"names.push(s)« copies names (each call)»",
		"[names] (s: string) => {« names copied at its first change, line 10»",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

const crossCopySrc = `use io

struct Shelf {
  var items: List<string>

  fun all(): List<string> => this.items

  fun add(s: string) {
    this.items.push(s)
  }
}

fun stock(into: *List<i64>) {
  into.push(1)
}

fun main() {
  val nums: List<i64> = [1]
  val other = nums
  stock(&nums)
  var shelf = Shelf(items: [])
  shelf.add("a")
  io.println("$other ${shelf.all().len()}")
}
`

// A copy the function cannot see from its own body: a field another method
// shared, a value a caller shared before handing over a pointer to it. The
// marker names the function that shared it; its tooltip says where.
func TestCrossFunctionCopyHints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(crossCopySrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": crossCopySrc}})
	res, _ := c.call("textDocument/inlayHint", map[string]any{"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	var hints []inlayHint
	if err := json.Unmarshal(res, &hints); err != nil {
		t.Fatalf("inlayHint: %s", res)
	}
	var copies []inlayHint
	for _, h := range hints {
		if strings.HasPrefix(h.Label, "copies ") {
			copies = append(copies, h)
		}
	}
	got := applyHints(crossCopySrc, copies)
	for _, want := range []string{
		"this.items.push(s)« copies this.items (shared in Shelf.all)»",
		"into.push(1)« copies into (shared by the caller, main)»",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	for _, h := range copies {
		if !strings.Contains(h.Tooltip.Value, "line ") {
			t.Errorf("tooltip without a place: %q", h.Tooltip.Value)
		}
	}
}
