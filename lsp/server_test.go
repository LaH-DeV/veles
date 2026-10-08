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

fun add(a: i32, b: i32): i32 => a + b

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
	if !strings.Contains(string(res), `struct Point\n  val y: i32`) {
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
	src := "use io\n\nerror A { n: i64 }\nerror B\nerror Both = A | B\n\nfun f(): i64 throws Both => 1\n\nfun main() { io.println(\"${f()}\") }\n"
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
	src := "use io\n\n/// A parse failure.\nerror ParseError {\n  /// the offending text\n  text: string\n}\nerror RangeError { value: i64 }\nerror PortErrors = ParseError | RangeError\n\n/** Context around a cause. */\nerror ConfigError { key: string, cause: PortErrors }\n\n/// Parses a port.\nfun parsePort(text: string): i64 throws PortErrors => text.toInt() ?: throw ParseError(text: text)\n\nfun main() {\n  val c = ConfigError(key: \"k\", cause: RangeError(value: 1))\n  io.println(c.key)\n  val r = parsePort(\"80\")\n  if (r is Ok) io.println(\"$r\")\n}\n"
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
	if h := hover(18, 13); !strings.Contains(h, "val c: ConfigError") || !strings.Contains(h, `error ConfigError {\n  // ConfigError(key: string, cause: PortErrors)\n  val key: string\n  val cause: PortErrors\n  public fun message(): string\n  implement Error\n}`) {
		t.Errorf("hover on a value shows no shape: %s", h)
	}
	// the type name itself, with its doc comment
	if h := hover(17, 12); !strings.Contains(h, "Context around a cause.") || !strings.Contains(h, "error ConfigError {") {
		t.Errorf("hover on a type shows no doc/shape: %s", h)
	}
	// a field with a doc comment
	if h := hover(18, 16); !strings.Contains(h, `error ConfigError\n  val key: string`) {
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
	os.WriteFile(filepath.Join(root, "geometry", "lib.vs"), []byte("/// Points and shapes on a plane.\n\npublic fun twice(n: i64): i64 => n * 2\n"), 0o644)
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
	if h := hover(11, 30); !strings.Contains(h, `val cause: E  (smart cast from Both)`) {
		t.Errorf("narrowed field hover: %s", h)
	}
	// a type's hover opens with its shape, not the name twice
	if h := hover(3, 7); strings.Contains(h, `error E\n`+"```") || !strings.Contains(h, `error E {\n  // E(n: i64)\n  val n: i64`) {
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
	src := "use io\n\nfun main() {\n  val s = \" hi \"\n  io.println(s.trim())\n}\n\nstruct P { x: i64 }\n\nextend P {\n  /// Twice x.\n  fun double(): i64 => this.x * 2\n}\n"
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
	if !strings.Contains(string(res), `extend string\n  public const fun trim(): string`) || !strings.Contains(string(res), "without leading or trailing") {
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
	src := "extend string {\n  public fun shout(): string => this + \"!\"\n}\nfun wrong(): i64 => \"x\"\n"
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
	src := "trait Show {\n  fun show(): string\n}\n\nstruct P {\n  x: i64\n}\n\nimplement Show for P {\n  fun show(): string => \"p\"\n}\n\nfun main() { }\n"
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
		Title       string `json:"title"`
		Kind        string `json:"kind"`
		IsPreferred bool   `json:"isPreferred"`
		Edit        struct {
			Changes map[string][]lspTextEdit `json:"changes"`
		} `json:"edit"`
	}
	json.Unmarshal(res, &actions)
	if len(actions) != 1 || actions[0].Kind != "quickfix" || actions[0].Title != "Move into the body of 'P'" || !actions[0].IsPreferred {
		t.Fatalf("code actions: %s", res)
	}
	edits := actions[0].Edit.Changes[uri]
	if len(edits) != 2 {
		t.Fatalf("expected a deletion and an insertion in %s, got %s", uri, res)
	}
	// apply the edits (later offsets first) and check the result compiles
	// to the inline form
	text := src
	type off struct {
		start, end int
		text       string
	}
	var offs []off
	for _, e := range edits {
		offs = append(offs, off{positionToOffset(source.NewFile(path, src), e.Range.Start), positionToOffset(source.NewFile(path, src), e.Range.End), e.NewText})
	}
	sort.Slice(offs, func(i, j int) bool { return offs[i].start > offs[j].start })
	for _, o := range offs {
		text = text[:o.start] + o.text + text[o.end:]
	}
	want := "trait Show {\n  fun show(): string\n}\n\nstruct P {\n  x: i64\n\n  implement Show {\n    fun show(): string => \"p\"\n  }\n}\n\nfun main() { }\n"
	if text != want {
		t.Errorf("after the fix:\n%s\n--- want ---\n%s", text, want)
	}
}

// A typo's nearest name comes back as a quick fix too, but not a preferred
// one: an editor's "fix all" must not rewrite code to a guess.
func TestCodeActionTypoGuess(t *testing.T) {
	src := "use io\n\nfun main() {\n  val counter = 1\n  io.println(\"${countr}\")\n}\n"
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
			diags = nil
			for _, d := range env.Params.Diagnostics {
				if d.Severity == 1 {
					diags = append(diags, d) // the errors; 'counter' is never used is a warning
				}
			}
		}
	}
	if len(diags) != 1 || diags[0].Data == nil || !strings.Contains(diags[0].Message, "did you mean 'counter'?") {
		t.Fatalf("expected one error with a guess, got %+v", diags)
	}
	// the family, linked to its explanation (D79)
	if d := diags[0]; d.Code != "unknown-name" || d.CodeDescription == nil || !strings.HasSuffix(d.CodeDescription.Href, "errors.md#unknown-name") {
		t.Errorf("code %q, description %+v", d.Code, d.CodeDescription)
	}
	res, _ := c.call("textDocument/codeAction", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range":        diags[0].Range,
		"context":      map[string]any{"diagnostics": diags},
	})
	var actions []struct {
		Title       string `json:"title"`
		IsPreferred bool   `json:"isPreferred"`
		Edit        struct {
			Changes map[string][]lspTextEdit `json:"changes"`
		} `json:"edit"`
	}
	json.Unmarshal(res, &actions)
	if len(actions) != 1 || actions[0].Title != "Change to 'counter'" || actions[0].IsPreferred {
		t.Fatalf("code actions: %s", res)
	}
	if edits := actions[0].Edit.Changes[uri]; len(edits) != 1 || edits[0].NewText != "counter" {
		t.Errorf("edits: %s", res)
	}
}

// D55: hover on a type alias shows its name, its definition as written
// and the full expansion; the binding's type prints the alias.
func TestTypeAliasHover(t *testing.T) {
	src := "use io\n\ntype Index = i64\n/// A group key.\ntype Key = (Index, u64)\n\nfun main() {\n  val k: Key = (1, 2)\n  io.println(\"${k.0}\")\n}\n"
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
	if h := hover(7, 10); !strings.Contains(h, "type Key = (Index, u64)  (= (i64, u64))") || !strings.Contains(h, "A group key.") {
		t.Errorf("hover on an alias use: %s", h)
	}
	if h := hover(7, 6); !strings.Contains(h, "val k: Key") {
		t.Errorf("hover on a binding typed by an alias: %s", h)
	}
}

// A struct's hover is its declaration with every implicit word spelled out
// — `val` for a bare field, `protected`, `static` — and the defaults. The
// unwritten visibility level has no word: it is the common case, and
// `internal` on every line only made the hover harder to read. A field's
// hover names the struct and shows the field's line.
func TestHoverSpellsOutModifiers(t *testing.T) {
	src := "use io\n\npublic struct Notes {\n  private var next: i64 = 1\n  items: bool = false\n  public protected var count: i64 = 0\n  static val empty = Notes()\n  public fun add(text: string) {\n    this.next += text.len()\n    this.count += 1\n  }\n  private fun bump() { }\n  public static fun of(n: i64): Notes => Notes(count: n)\n}\n\nfun main() {\n  val n = Notes()\n  n.add(\"x\")\n  io.println(\"${n.count} ${n.items}\")\n}\n"
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
	want := `public struct Notes {\n  // Notes(items: bool = ..., count: i64 = ...)\n  val items: bool = ...\n  public protected var count: i64 = ...\n  static val empty: Notes = ...\n  public fun add(text: string)\n  public static fun of(n: i64): Notes\n  // ... and 2 members not visible from here\n}`
	if h := hover(16, 11); !strings.Contains(h, want) {
		t.Errorf("struct hover: %s\nwant %s", h, want)
	}
	if h := hover(18, 19); !strings.Contains(h, `public struct Notes\n  public protected var count: i64 = ...`) {
		t.Errorf("field hover: %s", h)
	}
}

// Methods and traits hover as their declarations, spelled out: the owner
// on its own line, the visibility written, `static`, `override`, a trait's
// associated types and which methods have default bodies.
func TestHoverMethodsAndTraits(t *testing.T) {
	src := "use io\n\n/// Something with an area.\npublic trait Shape {\n  type Unit\n  fun area(): f64\n  fun describe(): string => \"area ${this.area()}\"\n  static fun unit(): string\n}\n\nstruct Square {\n  side: f64\n  fun grow(by: f64): Square => Square(side: this.side + by)\n  private fun check() { }\n  public static fun of(side: f64): Square => Square(side)\n}\n\nimplement Shape for Square {\n  type Unit = string\n  fun area(): f64 => this.side * this.side\n  override fun describe(): string => \"square\"\n  static fun unit(): string => \"m\"\n}\n\nimplement Display for Square {\n  fun toString(): string => \"sq\"\n}\n\nextend Square {\n  public fun doubled(): Square => this.grow(this.side)\n}\n\nsealed trait Tree {\n  fun size(): i64\n}\nstruct Leaf : Tree { implement Tree { fun size(): i64 => 1 } }\nstruct Node : Tree {\n  kids: List<Tree>\n  implement Tree { fun size(): i64 => this.kids.len() }\n}\n\nfun helper(): i64 => 1\n\nfun main() {\n  val s = Square(side: 2.0)\n  val a = s.area()\n  val d = s.describe()\n  val g = s.grow(1.0).doubled()\n  val o = Square.of(1.0)\n  val h = helper()\n  io.println(\"$a $d $g $o $h ${Leaf().size()}\")\n}\n"
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
	// the hover's markdown, decoded (JSON escapes `<` and newlines)
	hover := func(line, ch int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	for _, tc := range []struct {
		line, ch int
		want     string
	}{
		// the trait at its declaration: head, associated type, default body marker, static
		{3, 14, "public trait Shape {\n  type Unit\n  fun area(): f64\n  fun describe(): string => ...\n  static fun unit(): string\n}"},
		{3, 14, "Something with an area."},
		// an inherent method at a use, under its struct
		{47, 13, "struct Square\n  fun grow(by: f64): Square"},
		// an impl method at a use: the impl as owner, `override` kept
		{46, 13, "implement Shape for Square\n  override fun describe(): string"},
		// an extend method at a use
		{47, 24, "extend Square\n  public fun doubled(): Square"},
		// a static at a use
		{48, 18, "struct Square\n  public static fun of(side: f64): Square"},
		// a free function at a use, and a private method at its declaration
		{49, 11, "fun helper(): i64"},
		{13, 15, "struct Square\n  private fun check()"},
		// a trait's default method at its declaration
		{6, 7, "public trait Shape\n  fun describe(): string => ..."},
		// a sealed trait at its declaration: methods, then variants
		{32, 14, "sealed trait Tree {\n  fun size(): i64\n  Leaf\n  Node { kids: List<Tree> }\n}"},
	} {
		if h := hover(tc.line, tc.ch); !strings.Contains(h, tc.want) {
			t.Errorf("hover at %d:%d: %q\nwant %q", tc.line, tc.ch, h, tc.want)
		}
	}
}

// Values and modules hover as declarations too: a global with its
// visibility, kind and initializer; a static under its struct; a module
// as `use` spells it, with its public surface.
func TestHoverValuesAndModules(t *testing.T) {
	src := "use io\n\n/// The upper bound.\npublic val limit: i64 = 10\nvar hits = 0\nconst name = \"v\"\n\nstruct Status {\n  code: i64\n  public static val ok = Status(code: 200)\n}\n\nfun main() {\n  hits += 1\n  io.println(\"$limit $hits $name ${Status.ok.code}\")\n}\n"
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
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	for _, tc := range []struct {
		line, ch int
		want     string
	}{
		// a public val at its declaration, with its doc
		{3, 12, "public val limit: i64 = 10"},
		{3, 12, "The upper bound."},
		// a var at a use; a const
		{13, 3, "var hits: i64 = 0"},
		{5, 7, "const name: string = \"v\""},
		// a static, under its struct, at its declaration and at a use
		{9, 21, "struct Status\n  public static val ok: Status = Status(code: 200)"},
		{14, 42, "struct Status\n  public static val ok: Status = Status(code: 200)"},
		// a std module: as `use` spells it, its origin, its public functions
		{0, 5, "module io {  // std\n  public error TooLong\n  public trait Stream : Closeable + Sendable\n  public fun println(s: string)"},
		{14, 3, "Console input and output."},
	} {
		if h := hover(tc.line, tc.ch); !strings.Contains(h, tc.want) {
			t.Errorf("hover at %d:%d: %q\nwant %q", tc.line, tc.ch, h, tc.want)
		}
	}
}

// Completion offers only what the cursor may name (M5): private members
// inside the type's own declarations, unmarked ones inside the module,
// public ones from anywhere.
func TestCompletionVisibility(t *testing.T) {
	// one dangling `.` per buffer: `//A` (inside next) or `//B` (in main)
	// is replaced with the receiver under test
	base := "use io\n\nstruct Parser {\n  private toks: List<string>\n  private var pos: i64 = 0\n  public val tag: string = \"p\"\n  static val zero = 0\n  public static fun of(n: i64): Parser => Parser(toks: [])\n  fun next(): string? {\n    val t = this.toks.at(this.pos)\n    //A\n    t\n  }\n  private fun bump() { this.pos += 1 }\n  public fun done(): bool => this.pos >= this.toks.len()\n}\n\nextend Parser {\n  fun rewind() { this.pos = 0 }\n}\n\nfun main() {\n  val p = Parser(toks: [\"a\"])\n  //B\n  io.println(\"${p.done()} ${p.tag}\")\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": base}})
	version := 1
	labels := func(text string, line, ch int) map[string]bool {
		version++
		c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": []map[string]any{{"text": text}}})
		res, _ := c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var r struct {
			Items []struct {
				Label string `json:"label"`
			} `json:"items"`
		}
		json.Unmarshal(res, &r)
		out := map[string]bool{}
		for _, it := range r.Items {
			out[it.Label] = true
		}
		return out
	}
	// `p.` in main: no private field, method or static; no statics at all
	got := labels(strings.Replace(base, "  //B\n", "  p.\n", 1), 23, 4)
	for _, want := range []string{"tag", "next", "done", "rewind"} {
		if !got[want] {
			t.Errorf("p. in main should offer %q: %v", want, got)
		}
	}
	for _, hidden := range []string{"toks", "pos", "bump", "zero", "of"} {
		if got[hidden] {
			t.Errorf("p. in main must not offer %q: %v", hidden, got)
		}
	}
	// `this.` inside the type: every instance member, still no statics
	got = labels(strings.Replace(base, "    //A\n", "    this.\n", 1), 10, 9)
	for _, want := range []string{"toks", "pos", "bump", "tag", "next", "done", "rewind"} {
		if !got[want] {
			t.Errorf("this. inside Parser should offer %q: %v", want, got)
		}
	}
	if got["zero"] || got["of"] {
		t.Errorf("this. must not offer statics: %v", got)
	}
	// `Parser.` in main: the type's namespace — the public static only
	got = labels(strings.Replace(base, "  //B\n", "  Parser.\n", 1), 23, 9)
	if !got["of"] || !got["zero"] || got["next"] || got["tag"] || got["toks"] {
		t.Errorf("Parser. in main should offer the statics only: %v", got)
	}
	// top-level: this module's names and the prelude, not another module's
	got = labels(base, 20, 0)
	if !got["Parser"] || !got["main"] || !got["MutableList"] || got["println"] {
		t.Errorf("top-level completion: %v", got)
	}
}

// A local's hover shows its initializer as written; a parameter says so.
func TestHoverLocals(t *testing.T) {
	src := "use io\n\nfun scale(factor: i64): i64 {\n  val base = factor * 10\n  var total = base + 1\n  total += factor\n  total\n}\n\nfun main() { io.println(\"${scale(2)}\") }\n"
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
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	for _, tc := range []struct {
		line, ch int
		want     string
	}{
		{3, 7, "val base: i64 = factor * 10"},
		{5, 3, "var total: i64 = base + 1"},
		{3, 14, "val factor: i64  (parameter)"},
	} {
		if h := hover(tc.line, tc.ch); !strings.Contains(h, tc.want) {
			t.Errorf("hover at %d:%d: %q\nwant %q", tc.line, tc.ch, h, tc.want)
		}
	}
}

// `init` is shown in the struct's hover and never offered by completion.
func TestInitBlockInTooling(t *testing.T) {
	src := "use io\n\nstruct P {\n  a: i64\n  b: string\n  init {\n    this.b = \"$a\"\n  }\n  fun f(): i64 => this.a\n}\n\nfun main() {\n  val p = P(a: 1)\n  //X\n  io.println(\"${p.b} ${p.f()}\")\n}\n"
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
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 12, "character": 11}})
	if !strings.Contains(string(res), `val b: string  // assigned by init\n  fun f(): i64\n}`) {
		t.Errorf("struct hover: %s", res)
	}
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 5, "character": 3}})
	if !strings.Contains(string(res), "runs after every construction; not callable") {
		t.Errorf("init hover: %s", res)
	}
	for _, variant := range []string{"  p.\n", "  P.\n"} {
		text := strings.Replace(src, "  //X\n", variant, 1)
		c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": text}}})
		res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 13, "character": len(strings.TrimSpace(variant)) + 2}})
		if strings.Contains(string(res), `"label":"init"`) || strings.Contains(string(res), `$init`) {
			t.Errorf("completion after %q offers init: %s", strings.TrimSpace(variant), res)
		}
	}
}

// The struct hover lists what the reader could name from where they are:
// everything inside the type, no private members from the rest of the
// module, public members only from another module.
func TestHoverViewpoint(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "veles.toml"), []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "store"), 0o755)
	store := "public struct Notes {\n  private var next: i64 = 1\n  items: List<string> = []\n  public val tag: string = \"n\"\n  fun size(): i64 => this.items.len()\n  public fun add(text: string): i64 {\n    this.next += 1\n    this.next\n  }\n  private fun bump() { }\n}\n\nfun inModule(): i64 {\n  val n = Notes()\n  n.size()\n}\n"
	os.WriteFile(filepath.Join(root, "store", "store.vs"), []byte(store), 0o644)
	main := "use io, store\n\nfun main() {\n  val n = store.Notes()\n  io.println(\"${n.add(\"x\")} ${n.tag}\")\n}\n"
	os.WriteFile(filepath.Join(root, "main.vs"), []byte(main), 0o644)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{"rootUri": pathToURI(root)})
	c.notify("initialized", map[string]any{})
	storeURI := pathToURI(filepath.Join(root, "store", "store.vs"))
	mainURI := pathToURI(filepath.Join(root, "main.vs"))
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": storeURI, "languageId": "veles", "version": 1, "text": store}})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "veles", "version": 1, "text": main}})
	hover := func(uri string, line, ch int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	// inside a method of Notes: everything
	h := hover(storeURI, 0, 15)
	if !strings.Contains(h, "private var next") || !strings.Contains(h, "private fun bump()") || strings.Contains(h, "not visible") {
		t.Errorf("inside the type: %q", h)
	}
	// elsewhere in the module: no private members, unmarked ones yes
	h = hover(storeURI, 13, 11)
	if strings.Contains(h, "private") || !strings.Contains(h, "val items") || !strings.Contains(h, "fun size()") || !strings.Contains(h, "// ... and 2 members not visible from here") {
		t.Errorf("in the module: %q", h)
	}
	// from another module: public only, and the constructor as seen from there
	h = hover(mainURI, 3, 17)
	if strings.Contains(h, "items") || strings.Contains(h, "size()") || !strings.Contains(h, "public val tag") || !strings.Contains(h, "public fun add") || !strings.Contains(h, "// Notes(tag: string = ...)") || !strings.Contains(h, "// ... and 4 members not visible from here") {
		t.Errorf("from another module: %q", h)
	}
}

// Two scripts in one directory are two programs: neither sees the other's
// `main`, and each keeps its own analysis (hover works in both).
func TestScriptsShareADirectory(t *testing.T) {
	dir := t.TempDir()
	srcA := "use io\n\nfun main() {\n  io.println(\"a\")\n}\n"
	srcB := "use io\n\nfun main() {\n  val n = 1\n  io.println(\"$n\")\n}\n"
	pathA := filepath.Join(dir, "a.vss")
	pathB := filepath.Join(dir, "b.vss")
	for p, s := range map[string]string{pathA: srcA, pathB: srcB} {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	for p, s := range map[string]string{pathA: srcA, pathB: srcB} {
		c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": pathToURI(p), "languageId": "veles", "version": 1, "text": s}})
	}
	// hover on `n` in b.vss
	res, notes := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": pathToURI(pathB)}, "position": map[string]any{"line": 3, "character": 6}})
	for _, n := range notes {
		if strings.Contains(string(n), "already declared") {
			t.Errorf("scripts see each other: %s", n)
		}
	}
	if !strings.Contains(string(res), "i64") {
		t.Errorf("hover in a script: %s", res)
	}
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": pathToURI(pathA)}, "position": map[string]any{"line": 2, "character": 5}})
	if !strings.Contains(string(res), "main") {
		t.Errorf("hover in the other script: %s", res)
	}
}

// Enums in the editor (D57): hover on the type spells the members out, hover
// on a member shows its value, `E.` completes the members and the statics,
// `e.` completes `value` and the catalogued methods.
func TestEnumTooling(t *testing.T) {
	base := "use io\n\n/// Traffic light phases.\nenum Phase : u8 {\n  Red = 1\n  /// Caution.\n  Amber\n  Green = 10\n}\n\nfun main() {\n  val p = Phase.Amber\n  io.println(\"$p ${p.value} ${Phase.Red < p}\")\n  //A\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "main.vs")
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": base}})
	hover := func(line, ch int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		}
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	for _, tc := range []struct {
		line, ch int
		want     string
	}{
		{3, 6, "enum Phase : u8 {\n  Red = 1\n  Amber = 2\n  Green = 10\n}"},
		{3, 6, "Traffic light phases."},
		{11, 17, "Phase.Amber = 2"},
		{11, 17, "Caution."},
		{6, 3, "Phase.Amber = 2"},
		{12, 22, "Phase.value: u8"},
	} {
		if h := hover(tc.line, tc.ch); !strings.Contains(h, tc.want) {
			t.Errorf("hover at %d:%d: %q\nwant %q", tc.line, tc.ch, h, tc.want)
		}
	}
	version := 1
	labels := func(text string, line, ch int) map[string]bool {
		version++
		c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": []map[string]any{{"text": text}}})
		res, _ := c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var r struct {
			Items []struct {
				Label string `json:"label"`
			} `json:"items"`
		}
		json.Unmarshal(res, &r)
		out := map[string]bool{}
		for _, it := range r.Items {
			out[it.Label] = true
		}
		return out
	}
	got := labels(strings.Replace(base, "  //A\n", "  Phase.\n", 1), 13, 8)
	for _, want := range []string{"Red", "Amber", "Green", "values", "fromValue", "parse"} {
		if !got[want] {
			t.Errorf("Phase. should offer %q: %v", want, got)
		}
	}
	if got["toString"] || got["value"] {
		t.Errorf("Phase. offers instance members: %v", got)
	}
	got = labels(strings.Replace(base, "  //A\n", "  p.\n", 1), 13, 4)
	for _, want := range []string{"value", "toString", "compareTo"} {
		if !got[want] {
			t.Errorf("p. should offer %q: %v", want, got)
		}
	}
	if got["Red"] || got["values"] {
		t.Errorf("p. offers the type's namespace: %v", got)
	}
}

// D58: a struct's shape lists its impls, derived ones marked; a trait that
// another listed impl requires (Codable's Encodable and Decodable) is not
// listed again.
func TestHoverShapeFoldsDerivedImpls(t *testing.T) {
	src := "use io\n\nstruct Note {\n  id: i64\n  implement Codable\n}\nsealed trait S\nstruct A : S { x: i64 }\nimplement Codable for S\n\nfun main() {\n  val n = Note(id: 1)\n  val a: S = A(x: 1)\n  io.println(\"$n $a\")\n}\n"
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
	res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 11, "character": 6}})
	h := string(res)
	if !strings.Contains(h, `implement Codable\n}`) || strings.Contains(h, "implement Encodable") || strings.Contains(h, "implement Decodable") {
		t.Errorf("Codable should stand for the impls it derived: %s", h)
	}
	// a variant whose impls came from the family's line: derived, and marked
	res, _ = c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 12, "character": 13}})
	h = string(res)
	if !strings.Contains(h, `implement Decodable  // derived`) || !strings.Contains(h, `implement Encodable  // derived`) {
		t.Errorf("derived impls should be listed and marked: %s", h)
	}
}

// The prelude and the compiler's own names hover like declared ones: a
// primitive type shows what it has, a variant constructor what it builds,
// `panic` its contract, and a built-in method its signature for this
// receiver (`List<i64>.sorted(): List<i64>`, not `List<T>`).
func TestBuiltinNamesHover(t *testing.T) {
	src := "use io\n\nfun main() {\n  val xs = [3, 1, 2].sorted()\n  val o: i64? = null\n  val r: Result<i64, IoError> = Ok(1)\n  if (xs.len() > 5) panic(\"no\")\n  io.println(\"$o $r\")\n}\n"
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
		var h struct{ Contents struct{ Value string } }
		json.Unmarshal(res, &h)
		return h.Contents.Value
	}
	if h := hover(3, 23); !strings.Contains(h, "fun List\u003ci64\u003e.sorted(): List\u003ci64\u003e") {
		t.Errorf("built-in method for its receiver: %s", h)
	}
	if h := hover(4, 11); !strings.Contains(h, "builtin type i64 {") || !strings.Contains(h, "fun saturatingAdd(y: i64): i64") || !strings.Contains(h, "implement Comparable") || !strings.Contains(h, "signed 64-bit integer") {
		t.Errorf("primitive type: %s", h)
	}
	if h := hover(5, 33); !strings.Contains(h, "Ok(value: T)") || !strings.Contains(h, "sealed trait Result") {
		t.Errorf("variant constructor: %s", h)
	}
	if h := hover(6, 21); !strings.Contains(h, "fun panic(message: string): Never") {
		t.Errorf("panic: %s", h)
	}
}

// D75: names the prelude writes for a module are completed after that
// module's name, and not among the global names.
func TestCompletionOfPreludeHomes(t *testing.T) {
	src := "use codec, io\n\nfun main() {\n  val v = codec.VNull()\n  io.println(\"$v\")\n}\n"
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
	has := func(labels []string, want string) bool {
		for _, l := range labels {
			if l == want {
				return true
			}
		}
		return false
	}
	// `codec.V|` on line 3
	res, _ := c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 3, "character": 16}})
	if got := completionLabels(res); !has(got, "Value") || !has(got, "ValueEncoder") || !has(got, "KeyStyle") {
		t.Errorf("codec. offered %v", got)
	}
	// a bare name at the start of line 4: global names, not the machinery
	res, _ = c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 4, "character": 2}})
	if got := completionLabels(res); has(got, "ValueEncoder") || has(got, "Depth") || !has(got, "StringBuilder") {
		t.Errorf("global completion offered %v", got)
	}
}

// Named imports (D85): inside `use m { }` the module's members are offered,
// and what a braced import brought in is a bare name in scope — under its
// alias when it has one.
func TestCompletionOfNamedImports(t *testing.T) {
	src := "use io { println, eprintln as warn }, os\n\nfun main() {\n  println(\"x\")\n  \n}\n"
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
	has := func(labels []string, want string) bool {
		for _, l := range labels {
			if l == want {
				return true
			}
		}
		return false
	}
	at := func(line, ch int) []string {
		res, _ := c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		return completionLabels(res)
	}
	// on the empty line in main: the imported names, the alias, not the original
	if got := at(4, 2); !has(got, "println") || !has(got, "warn") || has(got, "eprintln") {
		t.Errorf("bare completion offered %v", got)
	}
	// inside the braces, after `println, `: the members of io
	if got := at(0, 15); !has(got, "readLine") || !has(got, "eprintln") {
		t.Errorf("completion inside use io { } offered %v", got)
	}
}

// Auto-import (D85): completing `readL` offers readLine from io with the
// import as an additional edit — into the braces of an existing `use io {
// … }`, after a bare `use io`, or as a new line.
func TestAutoImportOnCompletion(t *testing.T) {
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
	version := 0
	edits := func(src string, line, ch int) (string, bool) {
		version++
		method := "textDocument/didChange"
		if version == 1 {
			method = "textDocument/didOpen"
			c.notify(method, map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": version, "text": src}})
		} else {
			c.notify(method, map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": []map[string]any{{"text": src}}})
		}
		res, _ := c.call("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": ch}})
		var list struct {
			Items []struct {
				Label string `json:"label"`
				Edits []struct {
					Range   lspRange `json:"range"`
					NewText string   `json:"newText"`
				} `json:"additionalTextEdits"`
			} `json:"items"`
		}
		json.Unmarshal(res, &list)
		for _, it := range list.Items {
			if it.Label == "readLine" && len(it.Edits) == 1 {
				e := it.Edits[0]
				return fmt.Sprintf("%d:%d %q", e.Range.Start.Line, e.Range.Start.Character, e.NewText), true
			}
		}
		return "", false
	}
	if got, ok := edits("fun main() {\n  readL\n}\n", 1, 7); !ok || got != `0:0 "use io { readLine }\n\n"` {
		t.Errorf("no import yet: %q %v", got, ok)
	}
	if got, ok := edits("use os\n\nfun main() {\n  readL\n}\n", 3, 7); !ok || got != `0:6 "\nuse io { readLine }"` {
		t.Errorf("after the last use: %q %v", got, ok)
	}
	if got, ok := edits("use io\n\nfun main() {\n  readL\n}\n", 3, 7); !ok || got != `0:6 " { readLine }"` {
		t.Errorf("bare use io: %q %v", got, ok)
	}
	if got, ok := edits("use io { println }\n\nfun main() {\n  readL\n}\n", 3, 7); !ok || got != `0:16 ", readLine"` {
		t.Errorf("braces: %q %v", got, ok)
	}
	// no prefix, no flood; an imported name is not offered twice
	if _, ok := edits("use io { readLine }\n\nfun main() {\n  \n}\n", 3, 2); ok {
		t.Error("an imported name should not be offered for import again")
	}
}

// A renamed import (D85) is a declaration of its own at the alias in the
// `use` line: hover says what it stands for, and renaming it changes the
// alias and its uses and nothing of the module's; a bare unrenamed name
// hovers with the module it came from.
func TestRenamedImportHoverAndRename(t *testing.T) {
	src := "use io { println, eprintln as warn }\n\nfun main() {\n  println(\"a\")\n  warn(\"b\")\n  warn(\"c\")\n}\n"
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
	doc := map[string]any{"uri": uri}

	hover := func(pos map[string]any) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": doc, "position": pos})
		return string(res)
	}
	if got := hover(at(t, src, "warn", 1)); !strings.Contains(got, "alias of io.eprintln") || !strings.Contains(got, "eprintln") {
		t.Errorf("hover on an alias use: %s", got)
	}
	if got := hover(at(t, src, "warn", 0)); !strings.Contains(got, "alias of io.eprintln") {
		t.Errorf("hover on the alias in the use line: %s", got)
	}
	if got := hover(at(t, src, "println(\"a\")", 0)); !strings.Contains(got, "module io") {
		t.Errorf("hover on a bare imported name: %s", got)
	}

	// rename from a use: the alias and both uses, and no edit at `eprintln`
	res, fail := c.try("textDocument/rename", map[string]any{"textDocument": doc, "position": at(t, src, "warn", 2), "newName": "alert"})
	if fail != "" {
		t.Fatalf("rename refused: %s", fail)
	}
	var edit struct {
		Changes map[string][]lspTextEdit `json:"changes"`
	}
	json.Unmarshal(res, &edit)
	edits := edit.Changes[uri]
	if len(edits) != 3 {
		t.Fatalf("rename of an alias made %d edits, want 3: %s", len(edits), res)
	}
	f := source.NewFile("x", src)
	for _, e := range edits {
		off := positionToOffset(f, e.Range.Start)
		if src[off:off+4] != "warn" || e.NewText != "alert" {
			t.Errorf("edit %+v at %q", e, src[off:off+4])
		}
	}
}

// A function that suspends without saying so (D2) shows it in its hover, at
// its declaration and at every call — the effect is part of what a caller
// must know, and the inlay hint alone is easy to miss.
func TestHoverShowsInferredSuspends(t *testing.T) {
	src := "struct Pacer {\n  gap: Duration\n\n  fun pause() {\n    await sleep(this.gap)\n  }\n}\n\nfun wait() {\n  await sleep(Duration.millis(1))\n}\n\nfun outer() { wait() }\n\nfun pure(): i64 => 1\n\nfun main() {\n  outer()\n  Pacer(gap: Duration.millis(1)).pause()\n  val _ = pure()\n}\n"
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
	hover := func(line, col int) string {
		res, _ := c.call("textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": col}})
		return string(res)
	}
	for _, tc := range []struct {
		what      string
		line, col int
		want      string
		suspends  bool
	}{
		{"declaration", 8, 5, "fun wait() suspends", true},
		{"transitive declaration", 12, 5, "fun outer() suspends", true},
		{"call site", 12, 15, "fun wait() suspends", true},
		{"call in main", 17, 3, "fun outer() suspends", true},
		{"method declaration", 3, 7, "fun pause() suspends", true},
		{"method call", 18, 35, "fun pause() suspends", true},
		{"pure function", 14, 5, "fun pure(): i64", false},
	} {
		got := hover(tc.line, tc.col)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: hover has no %q: %s", tc.what, tc.want, got)
		}
		if !tc.suspends && strings.Contains(got, "suspends") {
			t.Errorf("%s: hover says suspends: %s", tc.what, got)
		}
	}
}

// Go-to-definition and hover follow a re-export (D89) to the declaration it
// stands for: a flattened, renamed item and a re-exported module.
func TestDefinitionFollowsReexport(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) string {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("lib/veles.toml", "[package]\nname = \"mathlib\"\nversion = \"0.1.0\"\n")
	write("lib/lib.vs", "public use shapes { area as surface }\npublic use geometry\n")
	write("lib/shapes/lib.vs", "/// The area of a unit.\npublic fun area(n: i64): i64 => n * n\n")
	write("lib/geometry/lib.vs", "public fun twice(n: i64): i64 => n * 2\n")
	write("app/veles.toml", "[package]\nname = \"app\"\nversion = \"0.1.0\"\n\n[dependencies]\nmathlib = \"../lib\"\n")
	src := "use io { println }\nuse mathlib, mathlib.geometry\n\nfun main() {\n  println(\"${mathlib.surface(3)} ${geometry.twice(2)}\")\n}\n"
	path := write("app/main.vs", src)
	uri := pathToURI(path)
	c, stop := newClient(t)
	defer stop()
	c.call("initialize", map[string]any{})
	c.notify("initialized", map[string]any{})
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "veles", "version": 1, "text": src}})
	at := func(method string, line, col int) string {
		res, _ := c.call(method, map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": col}})
		return string(res)
	}
	// `surface` in `mathlib.surface(3)` (line 4): the declaration is `area` in shapes
	def := at("textDocument/definition", 4, 24)
	if !strings.Contains(def, "shapes/lib.vs") {
		t.Errorf("definition of a re-exported item: %s", def)
	}
	if hov := at("textDocument/hover", 4, 24); !strings.Contains(hov, "fun area(n: i64): i64") || !strings.Contains(hov, "The area of a unit.") {
		t.Errorf("hover of a re-exported item: %s", hov)
	}
	// `twice` through the re-exported module `geometry` (line 4)
	if def := at("textDocument/definition", 4, 46); !strings.Contains(def, "geometry/lib.vs") {
		t.Errorf("definition through a re-exported module: %s", def)
	}
}

// Hovering the word `lazy` of a parameter explains it (D90), and hovering
// the parameter still shows it in the signature. The word is not a name: a
// rename or find-references on it changes nothing.
func TestHoverOnLazyWord(t *testing.T) {
	src := "fun show(lazy msg: fun(): string, times: i64 = 1) {\n  println(msg())\n}\n\nfun main() {\n  show(\"hi\")\n}\n"
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
	at := func(method string, line, col int, extra map[string]any) string {
		p := map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": col}}
		for k, v := range extra {
			p[k] = v
		}
		res, _ := c.call(method, p)
		return string(res)
	}
	got := at("textDocument/hover", 0, 10, nil) // inside the word `lazy`
	for _, want := range []string{"lazy", "parameter modifier", "wraps the argument in a lambda", "only if and when it calls the parameter"} {
		if !strings.Contains(got, want) {
			t.Errorf("hover on `lazy` has no %q: %s", want, got)
		}
	}
	if got := at("textDocument/hover", 0, 5, nil); !strings.Contains(got, "lazy msg: fun(): string") { // the function name
		t.Errorf("hover on the function should keep `lazy` in its signature: %s", got)
	}
	if got := at("textDocument/hover", 0, 34, nil); strings.Contains(got, "parameter modifier") {
		t.Errorf("hover on an ordinary parameter must not describe `lazy`: %s", got)
	}
	// the word is not a name: a rename is refused, not applied
	c.nextID++
	id := c.nextID
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": "textDocument/rename", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 10}, "newName": "eager"}})
	for msg := range c.out {
		var env struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *responseError  `json:"error"`
		}
		json.Unmarshal(msg, &env)
		if env.ID != nil && *env.ID == id {
			if env.Error == nil {
				t.Errorf("rename on the word `lazy` must be refused, got %s", env.Result)
			}
			break
		}
	}
}

// The name a `catch (e)` binds (D98) is an ordinary local: hover shows the
// error union where it is used and definition goes to the binding.
func TestHoverOnCatchBinding(t *testing.T) {
	src := "use io { println }\n\nerror Bad { message: string }\n\nfun parse(s: string): i64 throws Bad => try s.toInt() ?! Bad(message: \"no\")\n\nfun main() {\n  do {\n    println(\"${try parse(\"1\")}\")\n  } catch (problem) {\n    println(problem.message())\n  }\n}\n"
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
	at := func(method string, line, col int) string {
		res, _ := c.call(method, map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": col}})
		return string(res)
	}
	// the bound error and its use in the handler: named, and typed as the error
	if got := at("textDocument/hover", 9, 14); !strings.Contains(got, "problem") || !strings.Contains(got, "Bad") {
		t.Errorf("hover on the bound error: %s", got)
	}
	if got := at("textDocument/hover", 10, 14); !strings.Contains(got, "problem") || !strings.Contains(got, "Bad") {
		t.Errorf("hover on its use: %s", got)
	}
	if got := at("textDocument/definition", 10, 14); !strings.Contains(got, `"line":9`) {
		t.Errorf("definition of the use: %s", got)
	}
}

// The name of `if (val x = e)` (D95) is an ordinary local: hover shows its
// non-null value's type where it is used, definition goes to the binding,
// and the name is gone in the else branch.
func TestHoverOnIfValBinding(t *testing.T) {
	src := "use io { println }\n\nfun find(name: string): string? => null\n\nfun main() {\n  if (val found = find(\"a\") && found.len() > 1) println(found)\n  else println(\"none\")\n}\n"
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
	at := func(method string, line, col int) string {
		res, _ := c.call(method, map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": col}})
		return string(res)
	}
	// the binding itself, and its use in the rest of the chain
	if got := at("textDocument/hover", 5, 10); !strings.Contains(got, "found") {
		t.Errorf("hover on the binding: %s", got)
	}
	if got := at("textDocument/hover", 5, 32); !strings.Contains(got, "found") {
		t.Errorf("hover on its use in the condition: %s", got)
	}
	if got := at("textDocument/hover", 5, 58); !strings.Contains(got, "found") {
		t.Errorf("hover on its use in the branch: %s", got)
	}
	// go-to-definition from the use lands on the binding's line
	if got := at("textDocument/definition", 5, 58); !strings.Contains(got, `"line":5`) {
		t.Errorf("definition of the use: %s", got)
	}
}
