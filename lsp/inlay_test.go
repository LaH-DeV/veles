package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
)

const inlaySrc = `use io

error Bad { text: string }

struct Point { x: i64, y: i64 }

fun parse(text: string): i64 throws {
  val n = text.toInt() ?: throw Bad(text)
  n
}

fun double(n: i64) => n * 2

fun wait() {
  await sleep(Duration.zero)
}

fun main() throws {
  val count = try parse("21")
  val typed: i64 = 1
  val p = Point(x: 1, y: 2)
  val words = ["a", "b"].map(w => w.len())
  wait()
  io.println("${double(count)} $typed ${p.x} $words")
}
`

// applyHints writes each hint's label into the source at its position,
// so a test reads the result as the programmer would see it.
func applyHints(src string, hints []inlayHint) string {
	f := source.NewFile("x", src)
	sort.SliceStable(hints, func(i, j int) bool {
		return positionToOffset(f, hints[i].Position) > positionToOffset(f, hints[j].Position)
	})
	for _, h := range hints {
		at := positionToOffset(f, h.Position)
		label := h.Label
		if h.PaddingLeft {
			label = " " + label
		}
		if h.PaddingRight {
			label += " "
		}
		src = src[:at] + "«" + label + "»" + src[at:]
	}
	return src
}

func TestInlayHints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(inlaySrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": inlaySrc}})
	res, _ := c.call("textDocument/inlayHint", map[string]any{"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 100, "character": 0}}})
	var hints []inlayHint
	if err := json.Unmarshal(res, &hints); err != nil {
		t.Fatalf("inlayHint: %s", res)
	}
	got := applyHints(inlaySrc, hints)
	for _, want := range []string{
		"fun parse(text: string): i64 throws« Bad» {", // the error set of a bare throws
		"val n« : i64» = text.toInt()",                // an untyped binding
		"fun double(n: i64)«: i64» => n * 2",           // an expression body's return type
		"fun wait() «suspends »{",                     // inferred suspension
		"fun main() «suspends »throws« Bad» {",        // both, in the order they are written
		"val count« : i64» = try parse",
		"val typed: i64 = 1",          // written: no hint
		"val p = Point(x: 1, y: 2)",   // the type is on the right: no hint
		"val words« : List<i64>» = [", // a call's result
		"map(w«: string» => w.len())", // a lambda parameter
	} {
		want = strings.ReplaceAll(want, "« : ", "«: ")
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

// A binding in a generic body has a type per instance: no hint claims one.
func TestInlayHintsGeneric(t *testing.T) {
	src := "use io\n\nfun twice<T>(x: T): List<T> {\n  val pair = [x, x]\n  pair\n}\n\nfun main() {\n  io.println(\"${twice(1)} ${twice(\"a\")}\")\n}\n"
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
	if strings.Contains(string(res), "pair") || strings.Contains(string(res), "List<i64>") {
		t.Errorf("a generic body got an instance's type: %s", res)
	}
}

const allocSrc = `use io

fun summary(names: List<string>, limit: i64): string {
  val pick = (n: string) => n.len() > limit
  val kept = names.filter(n => pick(n))
  "${kept.len()} of ${names.len()}"
}

fun main() {
  io.println(summary(["a", "bb"], 1))
}
`

// D213: a `heap` hint after each allocating expression, with what it
// allocates in its tooltip, only when the client asks for them; turned off
// again by a configuration change.
func TestAllocationHints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(allocSrc), 0o644); err != nil {
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
		var heap []inlayHint
		for _, h := range hints {
			if h.Label == "heap" {
				heap = append(heap, h)
			}
		}
		return heap
	}
	open := func(c *client, options map[string]any) {
		c.call("initialize", map[string]any{"initializationOptions": options})
		c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": allocSrc}})
	}

	off, stop := newClient(t)
	open(off, map[string]any{})
	if h := hintsOf(off); len(h) != 0 {
		t.Errorf("hints without asking: %+v", h)
	}
	stop()

	c, stop := newClient(t)
	defer stop()
	open(c, map[string]any{"allocationHints": true})
	got := applyHints(allocSrc, hintsOf(c))
	for _, want := range []string{
		"(n: string) => n.len() > limit« heap»", // a closure capturing limit
		"pick(n)« heap»)« heap»",               // a closure capturing pick, in a new list
		"of ${names.len()}\"« heap»",            // string building
		"summary([\"a\", \"bb\"]« heap»",        // a list literal
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	var tips []string
	for _, h := range hintsOf(c) {
		tips = append(tips, h.Tooltip.Value)
	}
	if all := strings.Join(tips, "\n"); !strings.Contains(all, "a closure capturing 'limit'") || !strings.Contains(all, "pass 'limit' as a parameter") {
		t.Errorf("tooltips: %s", all)
	}
	c.notify("workspace/didChangeConfiguration", map[string]any{"settings": map[string]any{"veles": map[string]any{"inlayHints": map[string]any{"allocations": false}}}})
	if h := hintsOf(c); len(h) != 0 {
		t.Errorf("hints after turning them off: %d", len(h))
	}
}
