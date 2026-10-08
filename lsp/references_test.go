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

const refSrc = `use io

trait Shape {
  fun area(): i64
}

struct Sq {
  side: i64

  implement Shape {
    fun area(): i64 => this.side * this.side
  }
}

struct Point { x: i64, y: i64 }

fun measure<T: Shape>(s: T): i64 => s.area()

fun boxed(s: Shape): i64 => s.area()

fun scale(value: i64, factor: i64): i64 => value * factor

fun main() {
  val x = 3
  val p = Point(x, y: 4)
  val sq = Sq(side: 2)
  val a = sq.area() + measure(sq) + boxed(sq)
  val y = 10
  io.println("${scale(value: p.x, factor: a)} $x $y")
}
`

// at returns the LSP position of the nth (0-based) occurrence of needle
// in src, one character into it so the cursor sits on the name.
func at(t *testing.T, src, needle string, nth int) map[string]any {
	t.Helper()
	off := -1
	for i := 0; i <= nth; i++ {
		j := strings.Index(src[off+1:], needle)
		if j < 0 {
			t.Fatalf("%q occurrence %d not found", needle, nth)
		}
		off += 1 + j
	}
	p := offsetToPosition(source.NewFile("x", src), off+1)
	return map[string]any{"line": p.Line, "character": p.Character}
}

// try sends a request and returns its result or its error message.
func (c *client) try(method string, params any) (json.RawMessage, string) {
	c.nextID++
	id := c.nextID
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	for msg := range c.out {
		var env struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *responseError  `json:"error"`
		}
		json.Unmarshal(msg, &env)
		if env.ID != nil && *env.ID == id {
			if env.Error != nil {
				return nil, env.Error.Message
			}
			return env.Result, ""
		}
	}
	c.t.Fatalf("server closed before answering %s", method)
	return nil, ""
}

func openRefSrc(t *testing.T) (*client, func(), string, string) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(refSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": refSrc}})
	return c, stop, uri, path
}

func TestReferences(t *testing.T) {
	c, stop, uri, _ := openRefSrc(t)
	defer stop()
	doc := map[string]any{"uri": uri}
	count := func(res json.RawMessage) int {
		var locs []struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(res, &locs); err != nil {
			t.Fatalf("references: %s", res)
		}
		return len(locs)
	}

	// a field: the declaration, two `this.side`, the constructor label
	res, _ := c.call("textDocument/references", map[string]any{"textDocument": doc, "position": at(t, refSrc, "side", 0), "context": map[string]any{"includeDeclaration": true}})
	if n := count(res); n != 4 {
		t.Errorf("references to side: %d, want 4: %s", n, res)
	}
	res, _ = c.call("textDocument/references", map[string]any{"textDocument": doc, "position": at(t, refSrc, "side", 0), "context": map[string]any{"includeDeclaration": false}})
	if n := count(res); n != 3 {
		t.Errorf("references to side without the declaration: %d, want 3: %s", n, res)
	}

	// a trait method is one name with its implementations: the trait's
	// declaration, the implement's, the direct call, the bound call and
	// the call through a trait object
	for _, from := range []int{0, 1, 2, 3, 4} {
		res, _ = c.call("textDocument/references", map[string]any{"textDocument": doc, "position": at(t, refSrc, "area", from), "context": map[string]any{"includeDeclaration": true}})
		if n := count(res); n != 5 {
			t.Errorf("references to area from occurrence %d: %d, want 5: %s", from, n, res)
		}
	}

	// a parameter: its declaration, its use, and the named argument
	res, _ = c.call("textDocument/references", map[string]any{"textDocument": doc, "position": at(t, refSrc, "factor", 1), "context": map[string]any{"includeDeclaration": true}})
	if n := count(res); n != 3 {
		t.Errorf("references to factor: %d, want 3: %s", n, res)
	}

	// highlights in the file: the declaration is a write
	res, _ = c.call("textDocument/documentHighlight", map[string]any{"textDocument": doc, "position": at(t, refSrc, "sq", 1)})
	var hl []struct {
		Kind int `json:"kind"`
	}
	json.Unmarshal(res, &hl)
	if len(hl) != 4 || hl[0].Kind != 3 || hl[1].Kind != 2 {
		t.Errorf("highlights of sq: %s", res)
	}
}

// rename applies the server's rename and returns the new text, or the
// server's refusal.
func rename(t *testing.T, c *client, uri, name string, pos map[string]any) (string, string) {
	t.Helper()
	res, fail := c.try("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": pos, "newName": name})
	if fail != "" {
		return "", fail
	}
	var edit struct {
		Changes map[string][]lspTextEdit `json:"changes"`
	}
	if err := json.Unmarshal(res, &edit); err != nil {
		t.Fatalf("rename result: %s", res)
	}
	for u := range edit.Changes {
		if u != uri {
			t.Errorf("rename edited another file: %s", u)
		}
	}
	f := source.NewFile("x", refSrc)
	edits := edit.Changes[uri]
	sort.Slice(edits, func(i, j int) bool {
		return positionToOffset(f, edits[i].Range.Start) > positionToOffset(f, edits[j].Range.Start)
	})
	text := refSrc
	for _, e := range edits {
		text = text[:positionToOffset(f, e.Range.Start)] + e.NewText + text[positionToOffset(f, e.Range.End):]
	}
	return text, ""
}

func TestRename(t *testing.T) {
	c, stop, uri, _ := openRefSrc(t)
	defer stop()

	cases := []struct {
		name     string
		needle   string
		nth      int
		newName  string
		want     []string // substrings of the result
		wantFail string   // substring of the refusal
	}{
		{name: "field through a pun", needle: "x:", nth: 0, newName: "col",
			want: []string{"struct Point { col: i64, y: i64 }", "Point(col: x, y: 4)", "value: p.col,", "$x $y"}},
		{name: "variable through a pun", needle: "x = 3", nth: 0, newName: "row",
			want: []string{"val row = 3", "Point(x: row, y: 4)", "$row $y"}},
		{name: "parameter and its label", needle: "factor", nth: 0, newName: "k",
			want: []string{"fun scale(value: i64, k: i64): i64 => value * k", "k: a)"}},
		{name: "trait method and implementations", needle: "area", nth: 2, newName: "size",
			want: []string{"  fun size(): i64\n", "fun size(): i64 => this.side", "=> s.size()\n\nfun boxed", "boxed(s: Shape): i64 => s.size()", "sq.size()"}},
		{name: "a name another use would resolve to", needle: "y = 10", nth: 0, newName: "x", wantFail: "would"},
		{name: "a local that captures a call", needle: "sq =", nth: 0, newName: "measure", wantFail: "would"},
		{name: "a keyword", needle: "sq =", nth: 0, newName: "fun", wantFail: "not an identifier"},
		{name: "a module", needle: "io.println", nth: 0, newName: "out", wantFail: "module"},
		{name: "the standard library", needle: "println", nth: 0, newName: "say", wantFail: "standard library"},
		{name: "a builtin type", needle: "i64", nth: 0, newName: "int", wantFail: "built into"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, fail := rename(t, c, uri, tc.newName, at(t, refSrc, tc.needle, tc.nth))
			if tc.wantFail != "" {
				if !strings.Contains(fail, tc.wantFail) {
					t.Errorf("want a refusal containing %q, got %q\n%s", tc.wantFail, fail, text)
				}
				return
			}
			if fail != "" {
				t.Fatalf("refused: %s", fail)
			}
			for _, w := range tc.want {
				if !strings.Contains(text, w) {
					t.Errorf("missing %q in\n%s", w, text)
				}
			}
		})
	}

	// prepareRename names what is renamed; the receiver is not a name
	res, fail := c.try("textDocument/prepareRename", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": at(t, refSrc, "measure", 0)})
	if fail != "" || !strings.Contains(string(res), `"placeholder":"measure"`) {
		t.Errorf("prepareRename: %s %s", res, fail)
	}
	if _, fail = c.try("textDocument/prepareRename", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": at(t, refSrc, "this", 0)}); fail == "" {
		t.Errorf("prepareRename on this: %q", fail)
	}
}

// A derived Codable writes field, member and variant names on the wire:
// renaming one would still compile and silently change the encoded form,
// so the rename is refused unless a @key already fixes the name.
func TestRenameKeepsWireNames(t *testing.T) {
	src := `use io

struct User {
  name: string
  @key("user_id") id: i64
  @skip cache: string = ""
  implement Codable
}

enum Role {
  admin
  guest
}

fun main() {
  val u = User(name: "ann", id: 1)
  io.println("${u.name} ${u.id} ${u.cache} ${Role.admin}")
}
`
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
	for _, tc := range []struct {
		needle string
		nth    int
		wire   bool
	}{
		{"name", 0, true},   // on the wire as "name"
		{"admin", 1, false}, // an enum is encoded by name only where it is encoded: not refused
		{"id", 1, false},    // @key("user_id") holds the name
		{"cache", 0, false}, // @skip: never written
		{"u =", 0, false},   // a local
	} {
		_, fail := c.try("textDocument/prepareRename", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": at(t, src, tc.needle, tc.nth)})
		if got := strings.Contains(fail, "on the wire"); got != tc.wire {
			t.Errorf("prepareRename on %q: refusal %q, want wire refusal %v", tc.needle, fail, tc.wire)
		}
	}
}

// A rename reaches every module of the package, open in the editor or not.
func TestRenameAcrossModules(t *testing.T) {
	dir := t.TempDir()
	main := "use io\nuse util\n\nfun main() {\n  io.println(\"${util.twice(2)}\")\n}\n"
	util := "public fun twice(n: i64): i64 => n * 2\n"
	if err := os.WriteFile(filepath.Join(dir, "main.vs"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "util"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "util", "util.vs"), []byte(util), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(filepath.Join(dir, "main.vs"))
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": main}})
	res, fail := c.try("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": at(t, main, "twice", 0), "newName": "double"})
	if fail != "" {
		t.Fatalf("refused: %s", fail)
	}
	var edit struct {
		Changes map[string][]lspTextEdit `json:"changes"`
	}
	json.Unmarshal(res, &edit)
	utilURI := pathToURI(filepath.Join(dir, "util", "util.vs"))
	if len(edit.Changes[uri]) != 1 || len(edit.Changes[utilURI]) != 1 || edit.Changes[utilURI][0].NewText != "double" {
		t.Errorf("rename across modules: %s", res)
	}
}

func TestWorkspaceSymbols(t *testing.T) {
	c, stop, _, _ := openRefSrc(t)
	defer stop()
	res, _ := c.call("workspace/symbol", map[string]any{"query": "scl"})
	if !strings.Contains(string(res), `"name":"scale"`) || strings.Contains(string(res), `"name":"measure"`) {
		t.Errorf("query scl: %s", res)
	}
	res, _ = c.call("workspace/symbol", map[string]any{"query": "SIDE"})
	if !strings.Contains(string(res), `"containerName":"Sq"`) {
		t.Errorf("query SIDE: %s", res)
	}
	if strings.Contains(string(res), "std/") || strings.Contains(string(res), "println") {
		t.Errorf("the standard library leaked into workspace symbols: %s", res)
	}
}
