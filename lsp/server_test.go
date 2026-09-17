package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/std"
)

const testSrc = `use io

struct Point { x: i32, y: i32 }

fun add(a: i32, b: i32): i32 = a + b

fun main() {
  val p = Point(x: 1, y: 2)
  val s = add(p.x, p.y)
  io.println("$s")
  val bad: string = s
  io.println("$bad")
}
`

// client drives a Server through an in-memory pipe.
type client struct {
	t      *testing.T
	w      io.Writer
	out    chan []byte
	nextID int
}

func newClient(t *testing.T) (*client, func()) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- Serve(inR, outW) }()
	c := &client{t: t, w: inW, out: make(chan []byte, 64)}
	go func() {
		br := newFramedReader(outR)
		for {
			msg, err := br()
			if err != nil {
				close(c.out)
				return
			}
			c.out <- msg
		}
	}()
	return c, func() {
		c.notify("exit", nil)
		inW.Close()
		<-done
	}
}

func newFramedReader(r io.Reader) func() ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	return func() ([]byte, error) {
		for {
			if i := bytes.Index(buf, []byte("\r\n\r\n")); i >= 0 {
				header := string(buf[:i])
				n := 0
				fmt.Sscanf(header, "Content-Length: %d", &n)
				if len(buf) >= i+4+n {
					msg := append([]byte(nil), buf[i+4:i+4+n]...)
					buf = buf[i+4+n:]
					return msg, nil
				}
			}
			k, err := r.Read(tmp)
			if k > 0 {
				buf = append(buf, tmp[:k]...)
			}
			if err != nil {
				return nil, err
			}
		}
	}
}

func (c *client) send(v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n%s", len(data), data)
}

func (c *client) notify(method string, params any) {
	c.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

// call sends a request and returns its result, collecting notifications
// that arrive first into c.notes.
func (c *client) call(method string, params any) (json.RawMessage, []json.RawMessage) {
	c.nextID++
	id := c.nextID
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	var notes []json.RawMessage
	for msg := range c.out {
		var env struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  *responseError  `json:"error"`
		}
		json.Unmarshal(msg, &env)
		if env.ID != nil && *env.ID == id {
			if env.Error != nil {
				c.t.Fatalf("%s: %s", method, env.Error.Message)
			}
			return env.Result, notes
		}
		notes = append(notes, msg)
	}
	c.t.Fatalf("server closed before answering %s", method)
	return nil, nil
}

func TestServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte("fun main() { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)

	c, stop := newClient(t)
	defer stop()

	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	// open with an unsaved buffer that differs from disk: the overlay wins
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": testSrc}})

	// hover on `add` in the call `add(p.x, p.y)` (line 8, col 10)
	res, notes := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 8, "character": 11}})
	if !strings.Contains(string(res), `fun add(a: i32, b: i32): i32`) {
		t.Errorf("hover on add: %s", res)
	}
	// the didOpen must have produced the type error on `val bad`
	var found bool
	for _, n := range notes {
		if strings.Contains(string(n), "publishDiagnostics") && strings.Contains(string(n), "type mismatch") {
			found = true
			var p struct {
				Params struct {
					Diagnostics []struct {
						Range lspRange `json:"range"`
					} `json:"diagnostics"`
				} `json:"params"`
			}
			json.Unmarshal(n, &p)
			if len(p.Params.Diagnostics) != 1 || p.Params.Diagnostics[0].Range.Start.Line != 10 {
				t.Errorf("diagnostic position: %s", n)
			}
		}
	}
	if !found {
		t.Errorf("no diagnostics published; notifications: %d", len(notes))
	}

	// hover on the field access `p.y` (line 8, char 21)
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 8, "character": 21}})
	if !strings.Contains(string(res), "Point.y: i32") {
		t.Errorf("hover on field: %s", res)
	}

	// definition of `Point` in the constructor call goes to the struct
	res, _ = c.call("textDocument/definition", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 7, "character": 11}})
	if !strings.Contains(string(res), `"line":2`) {
		t.Errorf("definition of Point: %s", res)
	}

	// local variable hover
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 9, "character": 16}})
	if !strings.Contains(string(res), "val s: i32") {
		t.Errorf("hover on local: %s", res)
	}

	// document symbols
	res, _ = c.call("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}})
	for _, want := range []string{`"name":"Point"`, `"name":"add"`, `"name":"main"`, `"name":"x"`} {
		if !strings.Contains(string(res), want) {
			t.Errorf("documentSymbol missing %s: %s", want, res)
		}
	}

	// completion after `p.` offers fields; plain completion offers keywords and decls
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 8, "character": 16}})
	if !strings.Contains(string(res), `"label":"x"`) || strings.Contains(string(res), `"label":"fun"`) {
		t.Errorf("member completion: %s", res)
	}
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 6, "character": 0}})
	if !strings.Contains(string(res), `"label":"fun"`) || !strings.Contains(string(res), `"label":"Point"`) {
		t.Errorf("top-level completion: %s", res)
	}

	// a hover with nothing under the cursor answers null, never an omitted result
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 1, "character": 0}})
	if string(res) != "null" {
		t.Errorf("empty hover result = %s", res)
	}

	// `io.` offers the io module's public functions only
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 9, "character": 5}})
	if !strings.Contains(string(res), `"label":"println"`) || strings.Contains(string(res), `"label":"fun"`) || strings.Contains(string(res), `"label":"Point"`) {
		t.Errorf("module completion: %s", res)
	}

	// mid-edit: the buffer does not parse, but `p.` still offers Point's fields
	broken := strings.Replace(testSrc, "  io.println(\"$s\")", "  io.println(\"$s\")\n  p.", 1)
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 3}, "contentChanges": []map[string]any{{"text": broken}}})
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 10, "character": 4}})
	if !strings.Contains(string(res), `"label":"y"`) || strings.Contains(string(res), `"label":"len"`) {
		t.Errorf("member completion while broken: %s", res)
	}
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 4}, "contentChanges": []map[string]any{{"text": testSrc}}})

	// fix the error via didChange: diagnostics clear
	fixed := strings.Replace(testSrc, "val bad: string = s\n  io.println(\"$bad\")", "val ok: i32 = s\n  io.println(\"$ok\")", 1)
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": fixed}}})
	_, notes = c.call("shutdown", nil)
	cleared := false
	for _, n := range notes {
		if strings.Contains(string(n), "publishDiagnostics") && strings.Contains(string(n), `"diagnostics":[]`) {
			cleared = true
		}
	}
	if !cleared {
		t.Errorf("diagnostics were not cleared after the fix")
	}
}

func TestPathToURI(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows drive letters")
	}
	got := pathToURI(`C:\Users\x\main.vs`)
	if got != "file:///c%3A/Users/x/main.vs" {
		t.Errorf("pathToURI = %s", got)
	}
	if p := uriToPath(got); p != `c:\Users\x\main.vs` {
		t.Errorf("uriToPath = %s", p)
	}
}

const throwsSrc = `use io

error ParseError { text: string }
error RangeError { value: i64 }

fun parsePort(text: string): i64 throws ParseError | RangeError {
  val n = text.toInt() ?: throw ParseError(text: text)
  if (n < 0 || n > 65535) throw RangeError(value: n)
  n
}

fun loadConfig(text: string): i64 throws {
  val port = try parsePort(text)
  port * 2
}

fun main() throws {
  val r = loadConfig("80")
  io.println("$r")
  if (r is Ok) io.println("$r")
}
`

// completionLabels returns the labels of a completion result.
func completionLabels(res json.RawMessage) []string {
	var p struct {
		Items []struct{ Label string } `json:"items"`
	}
	json.Unmarshal(res, &p)
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Label)
	}
	return out
}

func TestInferredThrowsAndReceiverCompletion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(throwsSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": throwsSrc}})

	// hover on an inferred-throws function renders the inferred union (D45)
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 17, "character": 12}})
	if !strings.Contains(string(res), "throws ParseError | RangeError") {
		t.Errorf("hover on inferred throws: %s", res)
	}

	// inside `if (r is Ok)` the hover shows the smart-cast type
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 19, "character": 28}})
	if !strings.Contains(string(res), "val r: i64") || !strings.Contains(string(res), "smart cast from Result") {
		t.Errorf("hover on smart-cast local: %s", res)
	}

	version := 1
	complete := func(src string, line, ch int) []string {
		version++
		c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": []map[string]any{{"text": src}}})
		res, _ := c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		return completionLabels(res)
	}
	withLine := func(extra string) string {
		return strings.Replace(throwsSrc, "  io.println(\"$r\")\n", "  io.println(\"$r\")\n"+extra, 1)
	}

	// `r.` on a Result: the type is known and has no members, so the list is
	// empty rather than every member name in the package
	if got := complete(withLine("  r.\n"), 19, 4); len(got) != 0 {
		t.Errorf("Result receiver offered %v", got)
	}
	// a binding introduced in the same (non-parsing) edit still resolves
	got := complete(withLine("  val pt = ParseError(text: \"a\")\n  pt.\n"), 20, 5)
	if len(got) != 2 || got[0] != "message" || got[1] != "text" {
		t.Errorf("new error binding mid-edit offered %v", got)
	}
}

func TestErrorSetHover(t *testing.T) {
	src := "use io\n\nerror A { n: i64 }\nerror B\nerror Both = A | B\n\nfun f(): i64 throws Both = 1\n\nfun main() { io.println(\"${f()}\") }\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	// hover on `Both` in the throws clause (line 6, col 20)
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 6, "character": 21}})
	if !strings.Contains(string(res), "error Both = A | B") {
		t.Errorf("hover on error set: %s", res)
	}
	// ...and unfolds it with a link to each member's declaration (A is on line 3)
	if !strings.Contains(string(res), "[`A`](file://") || !strings.Contains(string(res), "#L3,7)") {
		t.Errorf("hover on error set has no member links: %s", res)
	}
	// hover on the function keeps the set's name in the signature and unfolds it below
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 6, "character": 4}})
	if !strings.Contains(string(res), "fun f(): i64 throws Both") || !strings.Contains(string(res), "[`Both`](file://") {
		t.Errorf("hover on function with an error set: %s", res)
	}
	// go-to-definition on the set name lands on its declaration (line 4)
	res, _ = c.call("textDocument/definition", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 6, "character": 21}})
	if !strings.Contains(string(res), `"line":4`) {
		t.Errorf("definition of error set: %s", res)
	}
	res, _ = c.call("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}})
	if !strings.Contains(string(res), `"name":"Both"`) {
		t.Errorf("documentSymbol missing the error set: %s", res)
	}
}

func TestHoverDocsAndShapes(t *testing.T) {
	src := "use io\n\n/// A parse failure.\nerror ParseError {\n  /// the offending text\n  text: string\n}\nerror RangeError { value: i64 }\nerror PortErrors = ParseError | RangeError\n\n/** Context around a cause. */\nerror ConfigError { key: string, cause: PortErrors }\n\n/// Parses a port.\nfun parsePort(text: string): i64 throws PortErrors = text.toInt() ?: throw ParseError(text: text)\n\nfun main() {\n  val c = ConfigError(key: \"k\", cause: RangeError(value: 1))\n  io.println(c.key)\n  val r = parsePort(\"80\")\n  if (r is Ok) io.println(\"$r\")\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	hover := func(line, ch int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		return string(res)
	}
	// a value of a struct type: its declaration line, then what it has inside
	if h := hover(18, 13); !strings.Contains(h, "val c: ConfigError") || !strings.Contains(h, `error ConfigError {\n  key: string\n  cause: ParseError | RangeError\n  fun message(): string\n}`) {
		t.Errorf("hover on a value shows no shape: %s", h)
	}
	// the type name itself, with its doc comment
	if h := hover(17, 12); !strings.Contains(h, "Context around a cause.") || !strings.Contains(h, "error ConfigError {") {
		t.Errorf("hover on a type shows no doc/shape: %s", h)
	}
	// a field with a doc comment
	if h := hover(18, 16); !strings.Contains(h, "ConfigError.key: string") {
		t.Errorf("hover on a field: %s", h)
	}
	// a function: doc, the set's name in the signature, the set unfolded with member shapes
	if h := hover(19, 12); !strings.Contains(h, "Parses a port.") || !strings.Contains(h, "throws PortErrors") || !strings.Contains(h, "error ParseError { text: string }") {
		t.Errorf("hover on a function: %s", h)
	}
}

func TestModuleDocHover(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "geometry"), 0o755)
	os.WriteFile(filepath.Join(root, "veles.toml"), []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n"), 0o644)
	os.WriteFile(filepath.Join(root, "geometry", "lib.vs"), []byte("/// Points and shapes on a plane.\n\npub fun twice(n: i64): i64 = n * 2\n"), 0o644)
	src := "use io\nuse geometry\n\nerror E { n: i64 }\nerror F { m: i64 }\nerror Both = E | F\nerror Wrap { cause: Both }\n\nfun main() {\n  val w = Wrap(cause: E(n: 1))\n  when (w.cause) {\n    is E => io.println(\"${w.cause.n} ${geometry.twice(2)}\")\n    is F => io.println(\"f\")\n  }\n}\n"
	path := filepath.Join(root, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	hover := func(line, ch int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		return string(res)
	}
	// the module name in `use geometry` and in `geometry.twice(...)` both show the module doc
	for _, p := range [][2]int{{1, 5}, {11, 41}} {
		if h := hover(p[0], p[1]); !strings.Contains(h, "module geometry") || !strings.Contains(h, "Points and shapes on a plane.") {
			t.Errorf("module hover at %v: %s", p, h)
		}
	}
	// a smart-cast field shows the narrowed type once, with its origin
	if h := hover(11, 30); !strings.Contains(h, `Wrap.cause: E  (smart cast from E | F)`) {
		t.Errorf("narrowed field hover: %s", h)
	}
	// a type's hover opens with its shape, not the name twice
	if h := hover(3, 7); strings.Contains(h, `error E\n`+"```") || !strings.Contains(h, `error E {\n  n: i64`) {
		t.Errorf("type hover doubles the name: %s", h)
	}
}

func TestBuiltinHoverAndDefinition(t *testing.T) {
	src := "use io\n\nfun main() {\n  val xs = [1, 2, 3]\n  io.println(\"${xs.len()} ${2.0.sqrt()}\")\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	// a built-in method hovers with its catalogue signature and description
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 4, "character": 20}})
	if !strings.Contains(string(res), ".len(): i64  (built in)") || !strings.Contains(string(res), "Number of elements.") {
		t.Errorf("hover on a built-in: %s", res)
	}
	// ...and its definition is the stub file materialised in the cache
	res, _ = c.call("textDocument/definition", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 4, "character": 32}})
	if !strings.Contains(string(res), "builtins.vs") {
		t.Errorf("definition of a built-in: %s", res)
	}
	// a standard-library function's definition is its (cached) source
	res, _ = c.call("textDocument/definition", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 4, "character": 6}})
	if !strings.Contains(string(res), "std/io/io.vs") && !strings.Contains(string(res), "std%2Fio%2Fio.vs") && !strings.Contains(string(res), "veles/std/io/io.vs") {
		t.Errorf("definition of io.println: %s", res)
	}
}

func TestExtendMethods(t *testing.T) {
	// line 3: val s = " hi "; 4: io.println(s.trim())
	src := "use io\n\nfun main() {\n  val s = \" hi \"\n  io.println(s.trim())\n}\n\nstruct P { x: i64 }\n\nextend P {\n  /// Twice x.\n  fun double(): i64 = self.x * 2\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	// a prelude extend method hovers like any function, with its doc and
	// the receiver type as owner
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 4, "character": 16}})
	if !strings.Contains(string(res), "fun string.trim(): string") || !strings.Contains(string(res), "without leading or trailing") {
		t.Errorf("hover on an extend method: %s", res)
	}
	// its definition is the prelude source
	res, _ = c.call("textDocument/definition", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 4, "character": 16}})
	if !strings.Contains(string(res), "string.vs") {
		t.Errorf("definition of an extend method: %s", res)
	}
	// completion on a string offers the built-ins and the prelude's extends
	// (the dangling `.` does not parse; the receiver resolves through the
	// last good analysis)
	broken := strings.Replace(src, "  io.println(s.trim())\n", "  io.println(s.trim())\n  s.\n", 1)
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": broken}}})
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 5, "character": 4}})
	labels := completionLabels(res)
	for _, want := range []string{"len", "trim", "split", "padStart"} {
		found := false
		for _, l := range labels {
			found = found || l == want
		}
		if !found {
			t.Errorf("completion on a string lacks %q: %v", want, labels)
		}
	}
	// document symbols list the extend block under its target
	res, _ = c.call("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}})
	if !strings.Contains(string(res), "extend P") {
		t.Errorf("document symbols: %s", res)
	}
}

// Editing the compiler's own prelude (`<repo>/std/prelude/*.vs`): the
// server checks it as the std module, so its extend blocks on built-in
// types are legal and real errors in the edited file are reported.
func TestStdSourceTreeDiagnostics(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "std", "prelude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(std.FS, "prelude")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, _ := fs.ReadFile(std.FS, "prelude/"+e.Name())
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src := "extend string {\n  pub fun shout(): string = self + \"!\"\n}\nfun wrong(): i64 = \"x\"\n"
	path := filepath.Join(dir, "zz_extra.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	_, notes := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 0}})
	var published string
	for _, n := range notes {
		if strings.Contains(string(n), "publishDiagnostics") {
			published += string(n)
		}
	}
	if strings.Contains(published, "outside the standard library") {
		t.Errorf("prelude source treated as a user package: %s", published)
	}
	if !strings.Contains(published, "type mismatch") {
		t.Errorf("no diagnostics for the edited prelude file: %s", published)
	}
}

func TestFormatting(t *testing.T) {
	src := "use io\nfun main(){io.println( \"x\" )}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	res, _ := c.call("textDocument/formatting", map[string]any{"textDocument": map[string]any{"uri": uri}, "options": map[string]any{"tabSize": 2}})
	var edits []struct {
		Range   lspRange `json:"range"`
		NewText string   `json:"newText"`
	}
	json.Unmarshal(res, &edits)
	if len(edits) != 1 || edits[0].NewText != "use io\nfun main() {\n  io.println(\"x\")\n}\n" || edits[0].Range.End.Line != 2 {
		t.Errorf("formatting edits: %s", res)
	}
	// an already formatted document gets no edits
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": edits[0].NewText}}})
	res, _ = c.call("textDocument/formatting", map[string]any{"textDocument": map[string]any{"uri": uri}})
	if string(res) != "[]" {
		t.Errorf("formatting a formatted document: %s", res)
	}
	// a document that does not parse is not touched
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 3}, "contentChanges": []map[string]any{{"text": "fun main( {"}}})
	res, _ = c.call("textDocument/formatting", map[string]any{"textDocument": map[string]any{"uri": uri}})
	if string(res) != "[]" {
		t.Errorf("formatting a broken document: %s", res)
	}
}

// A lint's autofix travels as the diagnostic's `data` and comes back as a
// quick fix from textDocument/codeAction: here, moving a top-level impl
// into the struct's body (D23).
func TestCodeActionInlineImpl(t *testing.T) {
	src := "trait Show {\n  fun show(): string\n}\n\nstruct P {\n  x: i64\n}\n\nimpl Show for P {\n  fun show(): string = \"p\"\n}\n\nfun main() { }\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	// the diagnostics arrive as a notification before the next answer
	_, notes := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 0}})
	var diags []lspDiagnostic
	for _, n := range notes {
		var env struct {
			Method string `json:"method"`
			Params struct {
				Diagnostics []lspDiagnostic `json:"diagnostics"`
			} `json:"params"`
		}
		json.Unmarshal(n, &env)
		if env.Method == "textDocument/publishDiagnostics" {
			diags = env.Params.Diagnostics
		}
	}
	if len(diags) != 1 || diags[0].Data == nil || !strings.Contains(diags[0].Message, "inside the body of 'P'") {
		t.Fatalf("expected one warning with a fix, got %+v", diags)
	}
	res, _ := c.call("textDocument/codeAction", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range":        diags[0].Range,
		"context":      map[string]any{"diagnostics": diags},
	})
	var actions []struct {
		Title string `json:"title"`
		Kind  string `json:"kind"`
		Edit  struct {
			Changes map[string][]lspTextEdit `json:"changes"`
		} `json:"edit"`
	}
	json.Unmarshal(res, &actions)
	if len(actions) != 1 || actions[0].Kind != "quickfix" || actions[0].Title != "Move into the body of 'P'" {
		t.Fatalf("code actions: %s", res)
	}
	edits := actions[0].Edit.Changes[uri]
	if len(edits) != 2 {
		t.Fatalf("expected a deletion and an insertion in %s, got %s", uri, res)
	}
	// apply the edits (later offsets first) and check the result compiles
	// to the inline form
	text := src
	type off struct{ start, end int; text string }
	var offs []off
	for _, e := range edits {
		offs = append(offs, off{positionToOffset(source.NewFile(path, src), e.Range.Start), positionToOffset(source.NewFile(path, src), e.Range.End), e.NewText})
	}
	sort.Slice(offs, func(i, j int) bool { return offs[i].start > offs[j].start })
	for _, o := range offs {
		text = text[:o.start] + o.text + text[o.end:]
	}
	want := "trait Show {\n  fun show(): string\n}\n\nstruct P {\n  x: i64\n\n  impl Show {\n    fun show(): string = \"p\"\n  }\n}\n\nfun main() { }\n"
	if text != want {
		t.Errorf("after the fix:\n%s\n--- want ---\n%s", text, want)
	}
}
