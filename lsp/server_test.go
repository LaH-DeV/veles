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
	if !strings.Contains(string(res), `internal struct Point\n  internal val y: i32`) {
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
	if h := hover(18, 13); !strings.Contains(h, "val c: ConfigError") || !strings.Contains(h, `internal error ConfigError {\n  internal val key: string\n  internal val cause: PortErrors\n  public fun message(): string\n  impl Error\n}`) {
		t.Errorf("hover on a value shows no shape: %s", h)
	}
	// the type name itself, with its doc comment
	if h := hover(17, 12); !strings.Contains(h, "Context around a cause.") || !strings.Contains(h, "error ConfigError {") {
		t.Errorf("hover on a type shows no doc/shape: %s", h)
	}
	// a field with a doc comment
	if h := hover(18, 16); !strings.Contains(h, `internal error ConfigError\n  internal val key: string`) {
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
	os.WriteFile(filepath.Join(root, "geometry", "lib.vs"), []byte("/// Points and shapes on a plane.\n\npublic fun twice(n: i64): i64 = n * 2\n"), 0o644)
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
	if h := hover(11, 30); !strings.Contains(h, `internal val cause: E  (smart cast from Both)`) {
		t.Errorf("narrowed field hover: %s", h)
	}
	// a type's hover opens with its shape, not the name twice
	if h := hover(3, 7); strings.Contains(h, `error E\n`+"```") || !strings.Contains(h, `internal error E {\n  internal val n: i64`) {
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
	if !strings.Contains(string(res), `extend string\n  public fun trim(): string`) || !strings.Contains(string(res), "without leading or trailing") {
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
	src := "extend string {\n  public fun shout(): string = self + \"!\"\n}\nfun wrong(): i64 = \"x\"\n"
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
// — `internal` for the unwritten level, `val` for a bare field — and the
// defaults; a field's hover names the struct and shows the field's line.
func TestHoverSpellsOutModifiers(t *testing.T) {
	src := "use io\n\npublic struct Notes {\n  private var next: i64 = 1\n  items: bool = false\n  public protected var count: i64 = 0\n  static val empty = Notes()\n  public fun add(text: string) {\n    self.next += text.len()\n    self.count += 1\n  }\n  private fun bump() { }\n  public static fun of(n: i64): Notes = Notes(count: n)\n}\n\nfun main() {\n  val n = Notes()\n  n.add(\"x\")\n  io.println(\"${n.count} ${n.items}\")\n}\n"
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
	want := `public struct Notes {\n  private var next: i64 = 1\n  internal val items: bool = false\n  public protected var count: i64 = 0\n  internal static val empty: Notes = Notes()\n  public fun add(text: string)\n  private fun bump()\n  public static fun of(n: i64): Notes\n}`
	if h := hover(16, 11); !strings.Contains(h, want) {
		t.Errorf("struct hover: %s\nwant %s", h, want)
	}
	if h := hover(18, 19); !strings.Contains(h, `public struct Notes\n  public protected var count: i64 = 0`) {
		t.Errorf("field hover: %s", h)
	}
}

// Methods and traits hover as their declarations, spelled out: the owner
// on its own line, the visibility written, `static`, `override`, a trait's
// associated types and which methods have default bodies.
func TestHoverMethodsAndTraits(t *testing.T) {
	src := "use io\n\n/// Something with an area.\npublic trait Shape {\n  type Unit\n  fun area(): f64\n  fun describe(): string = \"area ${self.area()}\"\n  static fun unit(): string\n}\n\nstruct Square {\n  side: f64\n  fun grow(by: f64): Square = Square(side: self.side + by)\n  private fun check() { }\n  public static fun of(side: f64): Square = Square(side)\n}\n\nimpl Shape for Square {\n  type Unit = string\n  fun area(): f64 = self.side * self.side\n  override fun describe(): string = \"square\"\n  static fun unit(): string = \"m\"\n}\n\nimpl Display for Square {\n  fun toString(): string = \"sq\"\n}\n\nextend Square {\n  public fun doubled(): Square = self.grow(self.side)\n}\n\nsealed trait Tree {\n  fun size(): i64\n}\nstruct Leaf : Tree { impl Tree { fun size(): i64 = 1 } }\nstruct Node : Tree {\n  kids: List<Tree>\n  impl Tree { fun size(): i64 = self.kids.len() }\n}\n\nfun helper(): i64 = 1\n\nfun main() {\n  val s = Square(side: 2.0)\n  val a = s.area()\n  val d = s.describe()\n  val g = s.grow(1.0).doubled()\n  val o = Square.of(1.0)\n  val h = helper()\n  io.println(\"$a $d $g $o $h ${Leaf().size()}\")\n}\n"
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
		{3, 14, "public trait Shape {\n  type Unit\n  fun area(): f64\n  fun describe(): string = ...\n  static fun unit(): string\n}"},
		{3, 14, "Something with an area."},
		// an inherent method at a use, under its struct
		{47, 13, "internal struct Square\n  internal fun grow(by: f64): Square"},
		// an impl method at a use: the impl as owner, `override` kept
		{46, 13, "impl Shape for Square\n  override fun describe(): string"},
		// an extend method at a use
		{47, 24, "extend Square\n  public fun doubled(): Square"},
		// a static at a use
		{48, 18, "internal struct Square\n  public static fun of(side: f64): Square"},
		// a free function at a use, and a private method at its declaration
		{49, 11, "internal fun helper(): i64"},
		{13, 15, "internal struct Square\n  private fun check()"},
		// a trait's default method at its declaration
		{6, 7, "public trait Shape\n  fun describe(): string = ..."},
		// a sealed trait at its declaration: methods, then variants
		{32, 14, "internal sealed trait Tree {\n  fun size(): i64\n  Leaf\n  Node { kids: List<Tree> }\n}"},
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
		{13, 3, "internal var hits: i64 = 0"},
		{5, 7, "internal const name: string = \"v\""},
		// a static, under its struct, at its declaration and at a use
		{9, 21, "internal struct Status\n  public static val ok: Status = Status(code: 200)"},
		{14, 42, "internal struct Status\n  public static val ok: Status = Status(code: 200)"},
		// a std module: as `use` spells it, its origin, its public functions
		{0, 5, "module io {  // std\n  public fun println(s: string)"},
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
	src := "use io\n\nstruct Parser {\n  private toks: List<string>\n  private var pos: i64 = 0\n  public val tag: string = \"p\"\n  private static val zero = 0\n  fun next(): string? {\n    val t = self.toks.at(self.pos)\n    self.\n    t\n  }\n  private fun bump() { self.pos += 1 }\n  public fun done(): bool = self.pos >= self.toks.len()\n}\n\nextend Parser {\n  fun rewind() { self.pos = 0 }\n}\n\nfun main() {\n  val p = Parser(toks: [\"a\"])\n  p.\n  io.println(\"${p.done()} ${p.tag}\")\n}\n"
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
	labels := func(line, ch int) map[string]bool {
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
	// `p.` in main: no private field, method or static; module-level and public ones yes
	got := labels(22, 4)
	for _, want := range []string{"tag", "next", "done", "rewind"} {
		if !got[want] {
			t.Errorf("p. in main should offer %q: %v", want, got)
		}
	}
	for _, hidden := range []string{"toks", "pos", "bump", "zero"} {
		if got[hidden] {
			t.Errorf("p. in main must not offer private %q: %v", hidden, got)
		}
	}
	// `self.` inside the type: everything
	got = labels(9, 9)
	for _, want := range []string{"toks", "pos", "bump", "tag", "next", "done", "rewind"} {
		if !got[want] {
			t.Errorf("self. inside Parser should offer %q: %v", want, got)
		}
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
