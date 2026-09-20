// Package lsp implements a Language Server Protocol server for Veles over
// stdio: diagnostics as you type, hover, go-to-definition, document symbols
// and completion. It reuses the compiler front end; unsaved buffers are
// passed to the loader as an overlay.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/driver"
	"github.com/LaH-DeV/veles/format"
	"github.com/LaH-DeV/veles/parser"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
)

// Serve runs the server until the client sends `exit` or the input closes.
func Serve(in io.Reader, out io.Writer) error {
	s := &Server{
		in:      bufio.NewReader(in),
		out:     out,
		docs:    map[string]*document{},
		overlay: map[string]string{},
	}
	return s.run()
}

type Server struct {
	in      *bufio.Reader
	out     io.Writer
	outMu   sync.Mutex
	docs    map[string]*document // by URI
	overlay map[string]string    // sema.OverlayKey(path) -> text

	// last analysis per package root, so hover/definition need no re-check
	analyses map[string]*analysis
	shutdown bool
	stdDir   string // materialised standard library, for go-to-definition into std
}

type document struct {
	uri  string
	path string
	text string
}

type analysis struct {
	pkg      *sema.Package
	index    *sema.Index // nil while the package does not parse
	lastGood *sema.Index // the most recent index that did; used by completion
	files    map[string]*source.File // OverlayKey -> parsed file
}

// ---------------------------------------------------------------------------
// JSON-RPC framing

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	// Result is always present on a success response — a `null` result is
	// a valid answer (no hover, no definition) and must not be omitted.
	Result any `json:"result"`
}

type errorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   *responseError  `json:"error"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

func (s *Server) run() error {
	s.analyses = map[string]*analysis{}
	for {
		msg, err := s.read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req request
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.Method == "exit" {
			return nil
		}
		s.handle(&req)
	}
}

func (s *Server) read() ([]byte, error) {
	length := -1
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			length, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(s.in, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *Server) send(v any) {
	data, _ := json.Marshal(v)
	s.outMu.Lock()
	defer s.outMu.Unlock()
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n", len(data))
	s.out.Write(data)
}

func (s *Server) reply(id json.RawMessage, result any) {
	if id == nil {
		return
	}
	s.send(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) replyError(id json.RawMessage, code int, msg string) {
	if id == nil {
		return
	}
	s.send(errorResponse{JSONRPC: "2.0", ID: id, Error: &responseError{code, msg}})
}

func (s *Server) notify(method string, params any) {
	s.send(notification{JSONRPC: "2.0", Method: method, Params: params})
}

// ---------------------------------------------------------------------------
// dispatch

func (s *Server) handle(req *request) {
	defer func() {
		if r := recover(); r != nil {
			s.replyError(req.ID, -32603, fmt.Sprintf("internal error: %v", r))
		}
	}()
	switch req.Method {
	case "initialize":
		s.materializeStd()
		s.reply(req.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync": map[string]any{
					"openClose": true,
					"change":    1, // full text
					"save":      map[string]any{"includeText": false},
				},
				"hoverProvider":          true,
				"definitionProvider":     true,
				"documentSymbolProvider": true,
				"documentFormattingProvider": true,
				"codeActionProvider":         map[string]any{"codeActionKinds": []string{"quickfix"}},
				"completionProvider":     map[string]any{"triggerCharacters": []string{"."}},
			},
			"serverInfo": map[string]any{"name": "veles-lsp", "version": "0.1"},
		})
	case "initialized":
	case "shutdown":
		s.shutdown = true
		s.reply(req.ID, nil)
	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		json.Unmarshal(req.Params, &p)
		s.open(p.TextDocument.URI, p.TextDocument.Text)
	case "textDocument/didChange":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		json.Unmarshal(req.Params, &p)
		if n := len(p.ContentChanges); n > 0 {
			s.open(p.TextDocument.URI, p.ContentChanges[n-1].Text)
		}
	case "textDocument/didSave":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		json.Unmarshal(req.Params, &p)
		if d := s.docs[p.TextDocument.URI]; d != nil {
			s.analyze(d)
		}
	case "textDocument/didClose":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		json.Unmarshal(req.Params, &p)
		if d := s.docs[p.TextDocument.URI]; d != nil {
			delete(s.overlay, sema.OverlayKey(d.path))
			delete(s.docs, p.TextDocument.URI)
		}
	case "textDocument/hover":
		s.reply(req.ID, s.hover(req.Params))
	case "textDocument/definition":
		s.reply(req.ID, s.definition(req.Params))
	case "textDocument/documentSymbol":
		s.reply(req.ID, s.documentSymbols(req.Params))
	case "textDocument/formatting":
		s.reply(req.ID, s.formatting(req.Params))
	case "textDocument/codeAction":
		s.reply(req.ID, s.codeAction(req.Params))
	case "textDocument/completion":
		s.reply(req.ID, s.completion(req.Params))
	default:
		if req.ID != nil {
			s.replyError(req.ID, -32601, "method not found: "+req.Method)
		}
	}
}

func (s *Server) open(uri, text string) {
	path := uriToPath(uri)
	d := &document{uri: uri, path: path, text: text}
	s.docs[uri] = d
	if s.inStdCache(path) {
		// a reference copy of the standard library: shown, not analysed
		s.publish(uri, []lspDiagnostic{})
		return
	}
	s.overlay[sema.OverlayKey(path)] = text
	s.analyze(d)
}

// ---------------------------------------------------------------------------
// analysis and diagnostics

func (s *Server) analyze(d *document) {
	diags := &source.Diagnostics{}
	pkg, err := sema.LoadPackageOverlay(d.path, diags, s.overlay)
	if err != nil {
		s.publish(d.uri, []lspDiagnostic{{
			Range:    lspRange{},
			Severity: 1,
			Source:   "veles",
			Message:  err.Error(),
		}})
		return
	}
	var index *sema.Index
	if !diags.HasErrors() {
		index = sema.CheckIndex(pkg, diags)
	}
	a := &analysis{pkg: pkg, index: index, lastGood: index, files: map[string]*source.File{}}
	if index == nil {
		if prev := s.analyses[pkg.Key()]; prev != nil {
			a.lastGood = prev.lastGood
		}
	}
	byFile := map[*source.File][]lspDiagnostic{}
	for _, m := range pkg.Modules {
		if m.Std && m != pkg.Given {
			continue // embedded: not a file the editor can show (the given module may be a std source tree being edited)
		}
		for _, f := range m.Files {
			a.files[sema.OverlayKey(f.Source.Path)] = f.Source
			byFile[f.Source] = nil
		}
	}
	for _, it := range diags.Items {
		f := it.Span.File
		if f == nil {
			f = a.files[sema.OverlayKey(d.path)]
			if f == nil {
				continue
			}
		}
		if _, tracked := byFile[f]; !tracked {
			continue
		}
		sev := 1
		if it.Severity == source.Warning {
			sev = 2
		} else if it.Severity == source.Note {
			sev = 3
		}
		byFile[f] = append(byFile[f], lspDiagnostic{
			Range:    spanToRange(it.Span),
			Severity: sev,
			Source:   "veles",
			Message:  it.Message,
			Data:     s.fixData(it.Fix),
		})
	}
	s.analyses[pkg.Key()] = a
	for f, items := range byFile {
		if items == nil {
			items = []lspDiagnostic{}
		}
		s.publish(s.uriFor(f.Path), items)
	}
}

func (s *Server) publish(uri string, items []lspDiagnostic) {
	s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": items})
}

// analysisFor finds the latest analysis covering a document.
func (s *Server) analysisFor(uri string) (*analysis, *source.File) {
	d := s.docs[uri]
	if d == nil {
		return nil, nil
	}
	key := sema.OverlayKey(d.path)
	for _, a := range s.analyses {
		if f, ok := a.files[key]; ok {
			return a, f
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// hover / definition

type positionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position lspPosition `json:"position"`
}

func (s *Server) refAt(params json.RawMessage) (*sema.Ref, *source.File) {
	var p positionParams
	json.Unmarshal(params, &p)
	a, f := s.analysisFor(p.TextDocument.URI)
	if a == nil || a.index == nil {
		return nil, nil
	}
	off := positionToOffset(f, p.Position)
	return a.index.RefAt(f, off), f
}

func (s *Server) hover(params json.RawMessage) any {
	ref, _ := s.refAt(params)
	if ref == nil {
		return nil
	}
	head, shape := ref.Detail, ref.Shape
	if shape != "" && (shape == head || strings.HasPrefix(shape, head+" {")) {
		// a type's shape opens with its own signature line: show it once
		head, shape = shape, ""
	}
	if ref.Where != "" {
		// a member: the declaration it belongs to, then the member itself
		head = ref.Where + "\n  " + head
	}
	value := "```veles\n" + head + "\n```"
	if ref.Doc != "" {
		value += "\n\n" + ref.Doc
	}
	if shape != "" {
		// a value of a struct type: what the type has inside
		value += "\n\n```veles\n" + shape + "\n```"
	}
	if u := ref.Unfold; u != nil {
		// the error set spelled out, each member a link to its declaration,
		// then each member's shape
		parts := make([]string, len(u.Members))
		for i, m := range u.Members {
			parts[i] = s.declLink(m.Name, m.Def)
		}
		value += "\n\n" + s.declLink(u.Name, u.Def) + " = " + strings.Join(parts, " | ")
		var shapes []string
		for _, m := range u.Members {
			if m.Shape != "" {
				shapes = append(shapes, m.Shape)
			}
		}
		if len(shapes) > 0 {
			value += "\n\n```veles\n" + strings.Join(shapes, "\n") + "\n```"
		}
	}
	return map[string]any{
		"contents": map[string]any{"kind": "markdown", "value": value},
		"range":    spanToRange(ref.Span),
	}
}

// declLink renders a name as a markdown link to its declaration
// (`file:///...#L<line>,<col>`, which the editor opens at that position),
// or as plain code when there is nowhere to go.
func (s *Server) declLink(name string, def source.Span) string {
	if !def.IsValid() || (strings.HasPrefix(def.File.Path, "std/") && s.stdDir == "") {
		return "`" + name + "`"
	}
	pos := offsetToPosition(def.File, def.Start)
	return fmt.Sprintf("[`%s`](%s#L%d,%d)", name, s.uriFor(def.File.Path), pos.Line+1, pos.Character+1)
}

func (s *Server) definition(params json.RawMessage) any {
	ref, _ := s.refAt(params)
	if ref == nil || !ref.Def.IsValid() || (strings.HasPrefix(ref.Def.File.Path, "std/") && s.stdDir == "") {
		return nil
	}
	return map[string]any{"uri": s.uriFor(ref.Def.File.Path), "range": spanToRange(ref.Def)}
}

// ---------------------------------------------------------------------------
// document symbols (from the syntax tree alone, so they survive type errors)

const (
	symFunction = 12
	symField    = 8
	symStruct   = 23
	symIface    = 11
	symVariable = 13
	symConstant = 14
	symMethod   = 6
)

type docSymbol struct {
	Name           string      `json:"name"`
	Detail         string      `json:"detail,omitempty"`
	Kind           int         `json:"kind"`
	Range          lspRange    `json:"range"`
	SelectionRange lspRange    `json:"selectionRange"`
	Children       []docSymbol `json:"children,omitempty"`
}

func (s *Server) documentSymbols(params json.RawMessage) any {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	if d == nil {
		return []docSymbol{}
	}
	file := parser.ParseFile(source.NewFile(d.path, d.text), &source.Diagnostics{})
	out := []docSymbol{}
	for _, decl := range file.Decls {
		if sym, ok := declSymbol(decl); ok {
			out = append(out, sym)
		}
	}
	return out
}

func funSymbol(fn *ast.FunDecl, kind int) docSymbol {
	return docSymbol{Name: fn.Name.Name, Detail: funSignature(fn), Kind: kind,
		Range: spanToRange(fn.Pos), SelectionRange: spanToRange(fn.Name.Pos)}
}

func declSymbol(decl ast.Decl) (docSymbol, bool) {
	switch d := decl.(type) {
	case *ast.FunDecl:
		return funSymbol(d, symFunction), true
	case *ast.ErrorAliasDecl:
		return docSymbol{Name: d.Name.Name, Detail: "error = " + ast.TypeString(d.Members), Kind: symStruct,
			Range: spanToRange(d.Pos), SelectionRange: spanToRange(d.Name.Pos)}, true
	case *ast.TypeAliasDecl:
		return docSymbol{Name: d.Name.Name, Detail: "type = " + ast.TypeString(d.Type), Kind: symStruct,
			Range: spanToRange(d.Pos), SelectionRange: spanToRange(d.Name.Pos)}, true
	case *ast.StructDecl:
		sym := docSymbol{Name: d.Name.Name, Kind: symStruct, Range: spanToRange(d.Pos), SelectionRange: spanToRange(d.Name.Pos)}
		if d.Variant != nil {
			sym.Detail = ": " + ast.TypeString(d.Variant)
		} else if d.Error {
			sym.Detail = "error"
		}
		for _, f := range d.Fields {
			sym.Children = append(sym.Children, docSymbol{Name: f.Name.Name, Detail: ast.TypeString(f.Type), Kind: symField,
				Range: spanToRange(f.Pos), SelectionRange: spanToRange(f.Name.Pos)})
		}
		for _, m := range d.Methods {
			sym.Children = append(sym.Children, funSymbol(m, symMethod))
		}
		return sym, true
	case *ast.TraitDecl:
		sym := docSymbol{Name: d.Name.Name, Kind: symIface, Range: spanToRange(d.Pos), SelectionRange: spanToRange(d.Name.Pos)}
		if d.Sealed {
			sym.Detail = "sealed"
		}
		for _, m := range d.Methods {
			sym.Children = append(sym.Children, funSymbol(m, symMethod))
		}
		return sym, true
	case *ast.ImplDecl:
		name := "extend " + ast.TypeString(d.Target)
		sel := d.Target.Span()
		if !d.Extend {
			name = "impl " + ast.TypeString(d.Trait) + " for " + ast.TypeString(d.Target)
			sel = d.Trait.Span()
		}
		sym := docSymbol{Name: name, Kind: symIface, Range: spanToRange(d.Pos), SelectionRange: spanToRange(sel)}
		for _, m := range d.Methods {
			sym.Children = append(sym.Children, funSymbol(m, symMethod))
		}
		return sym, true
	case *ast.ValDecl:
		kind := symVariable
		if d.Kind == ast.BindConst {
			kind = symConstant
		}
		sym := docSymbol{Name: d.Name.Name, Kind: kind, Range: spanToRange(d.Pos), SelectionRange: spanToRange(d.Name.Pos)}
		if d.Type != nil {
			sym.Detail = ast.TypeString(d.Type)
		}
		return sym, true
	case *ast.ExternBlock:
		sym := docSymbol{Name: "extern " + strconv.Quote(d.ABI), Kind: symIface, Range: spanToRange(d.Pos), SelectionRange: spanToRange(d.Pos)}
		for _, m := range d.Funs {
			sym.Children = append(sym.Children, funSymbol(m, symFunction))
		}
		return sym, true
	}
	return docSymbol{}, false
}

func funSignature(fn *ast.FunDecl) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i, p := range fn.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name.Name)
		if p.Type != nil {
			sb.WriteString(": " + ast.TypeString(p.Type))
		}
	}
	sb.WriteString(")")
	if fn.Ret != nil {
		sb.WriteString(": " + ast.TypeString(fn.Ret))
	}
	if fn.Effects.Suspends {
		sb.WriteString(" suspends")
	}
	if fn.Effects.Throws {
		sb.WriteString(" throws")
		if fn.Effects.Error != nil {
			sb.WriteString(" " + ast.TypeString(fn.Effects.Error))
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// positions and URIs

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
	// Data carries a lint's autofix as a ready workspace edit; the client
	// hands it back with a codeAction request (LSP round-trips `data`).
	Data *lspFix `json:"data,omitempty"`
}

type lspFix struct {
	Title   string                  `json:"title"`
	Changes map[string][]lspTextEdit `json:"changes"`
}

type lspTextEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

// fixData converts a diagnostic's fix to LSP edits keyed by document URI.
func (s *Server) fixData(fix *source.Fix) *lspFix {
	if fix == nil {
		return nil
	}
	out := &lspFix{Title: fix.Title, Changes: map[string][]lspTextEdit{}}
	for _, e := range fix.Edits {
		if e.Span.File == nil {
			return nil
		}
		uri := s.uriFor(e.Span.File.Path)
		out.Changes[uri] = append(out.Changes[uri], lspTextEdit{Range: spanToRange(e.Span), NewText: e.NewText})
	}
	return out
}

// codeAction offers the autofix of every diagnostic in the request's
// context that carries one, as a quick fix.
func (s *Server) codeAction(params json.RawMessage) any {
	var p struct {
		Context struct {
			Diagnostics []lspDiagnostic `json:"diagnostics"`
		} `json:"context"`
	}
	json.Unmarshal(params, &p)
	actions := []any{}
	for _, d := range p.Context.Diagnostics {
		if d.Data == nil {
			continue
		}
		actions = append(actions, map[string]any{
			"title":       d.Data.Title,
			"kind":        "quickfix",
			"diagnostics": []lspDiagnostic{d},
			"edit":        map[string]any{"changes": d.Data.Changes},
		})
	}
	return actions
}

// offsetToPosition converts a byte offset to a 0-based line and a UTF-16
// column, as LSP requires.
func offsetToPosition(f *source.File, off int) lspPosition {
	if off < 0 {
		off = 0
	}
	if off > len(f.Content) {
		off = len(f.Content)
	}
	line, _ := f.Position(off)
	lineStart := off
	for lineStart > 0 && f.Content[lineStart-1] != '\n' {
		lineStart--
	}
	col := 0
	for i := lineStart; i < off; {
		r, size := utf8.DecodeRuneInString(f.Content[i:])
		col += len(utf16.Encode([]rune{r}))
		i += size
	}
	return lspPosition{Line: line - 1, Character: col}
}

func positionToOffset(f *source.File, p lspPosition) int {
	off := 0
	for line := 0; line < p.Line && off < len(f.Content); {
		if f.Content[off] == '\n' {
			line++
		}
		off++
	}
	units := 0
	for off < len(f.Content) && units < p.Character && f.Content[off] != '\n' {
		r, size := utf8.DecodeRuneInString(f.Content[off:])
		units += len(utf16.Encode([]rune{r}))
		off += size
	}
	return off
}

func spanToRange(sp source.Span) lspRange {
	if !sp.IsValid() {
		return lspRange{}
	}
	return lspRange{Start: offsetToPosition(sp.File, sp.Start), End: offsetToPosition(sp.File, sp.End)}
}

func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	p := u.Path
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, "/")
		p = filepath.FromSlash(p)
	}
	return p
}

// uriFor returns the URI to use for a file: the one the client opened it
// under when it is an open document (clients compare URIs textually), else
// a VS Code-style file URI (lower-case drive letter, encoded colon).
func (s *Server) uriFor(path string) string {
	if cached := s.stdPath(path); cached != "" {
		return pathToURI(cached)
	}
	key := sema.OverlayKey(path)
	for _, d := range s.docs {
		if sema.OverlayKey(d.path) == key {
			return d.uri
		}
	}
	return pathToURI(path)
}

func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		if len(abs) >= 2 && abs[1] == ':' {
			abs = strings.ToLower(abs[:1]) + abs[1:]
		}
		abs = "/" + abs
	}
	u := (&url.URL{Scheme: "file", Path: abs}).String()
	// VS Code spells the drive colon as %3A
	if runtime.GOOS == "windows" {
		u = "file:" + strings.Replace(strings.TrimPrefix(u, "file:"), ":/", "%3A/", 1)
	}
	return u
}

// formatting answers textDocument/formatting with one edit replacing the
// whole document, in the style of the file's package (`[format]` in
// veles.toml). A file that does not parse is left alone: null, no edits.
func (s *Server) formatting(params json.RawMessage) any {
	var p positionParams
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	if d == nil {
		return nil
	}
	style, err := driver.StyleFor(d.path)
	if err != nil {
		style = format.Default
	}
	file := source.NewFile(d.path, d.text)
	out, diags := format.Source(file, style)
	if diags.HasErrors() || out == d.text {
		return []any{}
	}
	return []map[string]any{{
		"range":   lspRange{Start: lspPosition{0, 0}, End: offsetToPosition(file, len(d.text))},
		"newText": out,
	}}
}
