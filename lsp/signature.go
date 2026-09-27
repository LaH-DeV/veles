package lsp

import (
	"encoding/json"
	"strings"

	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/sema"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Signature help: inside the parentheses of a call, the callee's parameter
// list with the one being written highlighted — by position, or by name
// once the argument starts `name:`. The call is found from the tokens
// before the cursor, so it works while the buffer does not parse (the
// usual state in the middle of typing a call); the callee is resolved in
// the current index when there is one, else in the last good one.

type signatureInfo struct {
	Label         string          `json:"label"`
	Documentation *markup         `json:"documentation,omitempty"`
	Parameters    []parameterInfo `json:"parameters"`
}

type parameterInfo struct {
	Label string `json:"label"`
}

type markup struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// openCall describes the call the cursor is inside.
type openCall struct {
	callee source.Span // the name before `(`
	arg    int         // the argument being written, from 0
	named  string      // its name, when written `name: ...`
}

// callAt finds the innermost unclosed `(` of a call before off.
func callAt(f *source.File, off int) (openCall, bool) {
	toks := callTokens(f, lexer.Tokenize(f, &source.Diagnostics{}), off)
	type frame struct {
		kind  lexer.TokenKind
		index int // the token index of the bracket
		arg   int
		start int // token index where the current argument starts
	}
	var stack []frame
	for i, t := range toks {
		if t.Kind == lexer.EOF || t.Span.Start >= off {
			break
		}
		switch t.Kind {
		case lexer.LParen, lexer.LBracket, lexer.LBrace:
			stack = append(stack, frame{kind: t.Kind, index: i, start: i + 1})
		case lexer.RParen, lexer.RBracket, lexer.RBrace:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case lexer.Comma:
			if n := len(stack); n > 0 {
				stack[n-1].arg++
				stack[n-1].start = i + 1
			}
		}
	}
	if len(stack) == 0 || stack[len(stack)-1].kind != lexer.LParen {
		return openCall{}, false
	}
	top := stack[len(stack)-1]
	// the callee: the name before `(`, past any `<type arguments>`
	j := top.index - 1
	if j >= 0 && toks[j].Kind == lexer.Gt {
		depth := 0
		for ; j >= 0; j-- {
			if toks[j].Kind == lexer.Gt {
				depth++
			} else if toks[j].Kind == lexer.Lt {
				depth--
				if depth == 0 {
					j--
					break
				}
			}
		}
	}
	if j < 0 || toks[j].Kind != lexer.Ident {
		return openCall{}, false // `(a + b)`, `if (`, a lambda call through an expression
	}
	call := openCall{callee: toks[j].Span, arg: top.arg}
	if k := top.start; k+1 < len(toks) && toks[k].Kind == lexer.Ident && toks[k+1].Kind == lexer.Colon && toks[k+1].Span.Start < off {
		call.named = toks[k].Text
	}
	return call, true
}

// callTokens opens up the string the cursor is in: a call inside an
// interpolation (`"${scale(x, |)}"`) is made of the tokens of that `${}`,
// not of the one string token around it.
func callTokens(f *source.File, toks []lexer.Token, off int) []lexer.Token {
	for i, t := range toks {
		if t.Kind != lexer.String || off <= t.Span.Start || off > t.Span.End {
			continue
		}
		for _, part := range t.Parts {
			if !part.IsExpr || off < part.Span.Start || off > part.Span.End {
				continue
			}
			inner := lexer.TokenizeRange(f, part.Span.Start, part.Span.End, &source.Diagnostics{})
			out := append([]lexer.Token{}, toks[:i]...)
			out = append(out, callTokens(f, inner, off)...)
			return out
		}
	}
	return toks
}

func (s *Server) signatureHelp(params json.RawMessage) any {
	var p positionParams
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	a, af := s.analysisFor(p.TextDocument.URI)
	if d == nil || a == nil {
		return nil
	}
	f := source.NewFile(d.path, d.text)
	call, ok := callAt(f, positionToOffset(f, p.Position))
	if !ok {
		return nil
	}
	name := d.text[call.callee.Start:call.callee.End]
	var ref *sema.Ref
	if a.index != nil && af != nil && af.Content == d.text {
		ref = a.index.RefAt(af, call.callee.Start)
	}
	if ref == nil || ref.Name != name {
		ref = refNamedIn(a.lastGood, sema.OverlayKey(d.path), name, call.callee.Start)
	}
	if ref == nil {
		return nil
	}
	sig, ok := signatureOf(ref)
	if !ok {
		return nil
	}
	active := call.arg
	if call.named != "" {
		for i, prm := range sig.Parameters {
			if strings.HasPrefix(prm.Label, call.named+":") {
				active = i
			}
		}
	} else if n := len(sig.Parameters); n > 0 && active >= n && strings.HasSuffix(sig.Parameters[n-1].Label, "...") {
		active = n - 1 // the variadic parameter takes the rest
	}
	return map[string]any{"signatures": []signatureInfo{sig}, "activeSignature": 0, "activeParameter": active}
}

// signatureOf spells a callable reference as `name(p: T, ...): R`: a
// function, a method, or a struct's implicit constructor.
func signatureOf(ref *sema.Ref) (signatureInfo, bool) {
	var prms []parameterInfo
	ret := ""
	switch t := ref.Type.(type) {
	case *types.Func:
		for _, prm := range t.Params {
			label := prm.Name + ": " + prm.Type.String()
			if prm.Variadic {
				if l, ok := prm.Type.(*types.List); ok {
					label = prm.Name + ": " + l.Elem.String() + "..."
				}
			} else if prm.HasDefault {
				label += " = …"
			}
			prms = append(prms, parameterInfo{label})
		}
		if t.Ret != nil && !types.IsUnit(t.Ret) {
			ret = ": " + t.Ret.String()
		}
		if t.Effects.Suspends {
			ret += " suspends"
		}
		if t.Effects.Throws {
			ret += " throws"
			if t.Effects.Error != nil && !types.IsNever(t.Effects.Error) {
				ret += " " + t.Effects.Error.String()
			}
		}
	case *types.Struct:
		if ref.Kind != "struct" {
			return signatureInfo{}, false
		}
		for _, fld := range t.Fields {
			if fld.Init {
				continue // assigned by `init`, not given by the caller
			}
			label := fld.Name + ": " + fld.Type.String()
			if fld.HasDefault {
				label += " = …"
			}
			prms = append(prms, parameterInfo{label})
		}
	default:
		return signatureInfo{}, false
	}
	labels := make([]string, len(prms))
	for i, prm := range prms {
		labels[i] = prm.Label
	}
	sig := signatureInfo{Label: ref.Name + "(" + strings.Join(labels, ", ") + ")" + ret, Parameters: prms}
	if sig.Parameters == nil {
		sig.Parameters = []parameterInfo{}
	}
	if ref.Doc != "" {
		sig.Documentation = &markup{Kind: "markdown", Value: ref.Doc}
	}
	return sig, true
}
