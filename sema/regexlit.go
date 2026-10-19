package sema

import (
	"sort"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// A regular expression literal `/pattern/flags` (D182) is the call
// `regex.literal("pattern", "flags")`, whose result is a `regex.Regex`. Before
// it is lowered, the pattern is run through `regex.validate` — the same parser
// and compiler `Regex.compile` uses — by the compile-time evaluator, so a bad
// pattern is an error here with the parser's own message instead of a panic
// when the program runs.

// regexModuleName is the name this file gives the standard `regex` module
// (`use regex`, or `use regex as re`), "" when it does not import it.
func (f *fnCtx) regexModuleName() string {
	var names []string
	for s := f.scope; s != nil; s = s.parent {
		for name, sym := range s.symbols {
			if sym.Kind == SymModule && sym.Mod != nil && sym.Mod.Path == "std/regex" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[0]
}

func (f *fnCtx) regexLit(e *ast.RegexLit, want types.Type) Expr {
	alias := f.regexModuleName()
	if alias == "" {
		f.c.errorFix(e.Pos, f.useRegexFix(e.Pos), "a regular expression literal needs the 'regex' module: add 'use regex' at the top of the file (D182)")
		return bad()
	}
	call := func(name string) ast.Expr {
		str := func(text string) ast.Arg {
			return ast.Arg{Value: &ast.StringLit{Parts: []ast.StringPart{{Text: text}}, Pos: e.Pos}}
		}
		return &ast.CallExpr{
			Fun:  &ast.MemberExpr{X: &ast.NameExpr{Name: alias, Pos: e.Pos}, Name: ast.Ident{Name: name, Pos: e.Pos}, Pos: e.Pos},
			Args: []ast.Arg{str(e.Pattern), str(e.Flags)},
			Pos:  e.Pos,
		}
	}
	check := f.checkExprTo(call("validate"), types.TString)
	if types.IsInvalid(check.Type()) {
		return bad()
	}
	if v, ok := f.c.evalConst(check, e.Pos).(*CString); ok && v.V != "" {
		f.errorf(e.Pos, "this regular expression cannot be compiled: %s (D182)", v.V)
		return bad()
	}
	return f.checkExpr(call("literal"), want)
}

// useRegexFix adds `use regex` to the file.
func (f *fnCtx) useRegexFix(at source.Span) *source.Fix {
	if f.file == nil || at.File == nil {
		return nil
	}
	pos, beforeUse := useInsertAt(f.file, at.File.Content)
	line := "use regex\n"
	if !beforeUse {
		line += "\n"
	}
	return &source.Fix{Title: "Add 'use regex'", Edits: []source.TextEdit{{Span: source.Span{File: at.File, Start: pos, End: pos}, NewText: line}}}
}
