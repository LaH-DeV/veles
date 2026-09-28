package sema

import "github.com/LaH-DeV/veles/source"

// D75: the prelude holds what almost every program uses. The machinery
// behind `implement Codable`, the recursion guard and the FFI's layout
// marker are written in the prelude — the prelude's own impls and the
// derived code need them there, one compilation unit — but a program
// reaches them through a module: `use codec` then `codec.Value`. The
// module's directory holds its documentation; its names are these.
var preludeHomes = map[string]string{
	// the codec machinery
	"Problem": "codec", "Problems": "codec", "EnumStyle": "codec", "DurationStyle": "codec",
	"KeyStyle": "codec", "styleKey": "codec", "Kind": "codec", "Encoder": "codec",
	"Decoder": "codec", "childPath": "codec", "indexPath": "codec", "joinPath": "codec",
	"finish": "codec", "Value": "codec", "VNull": "codec", "VBool": "codec", "VInt": "codec",
	"VFloat": "codec", "VString": "codec", "VList": "codec", "VObject": "codec",
	"ValueDecoder": "codec", "ValueEncoder": "codec",
	// the recursion guard
	"Depth": "recursion", "maxRecursionDepth": "recursion", "tooDeepMessage": "recursion",
	// memory C can read as it is
	"CLayout": "ffi",
}

// PreludeHome is the module a prelude name is reached through, or "" when
// it is global.
func PreludeHome(name string) string { return preludeHomes[name] }

// preludeSym finds a prelude declaration by name, global or not: derived
// code and the checker's own lowering reach the machinery wherever a
// program would have to import it.
func (c *Checker) preludeSym(name string) *Symbol {
	if prelude, ok := c.pkg.Modules["std/prelude"]; ok && prelude.Scope != nil {
		if sym := prelude.Scope.LookupLocal(name); sym != nil {
			return sym
		}
	}
	return c.universe.LookupLocal(name)
}

// noMember reports `mod.name` where the module has no such declaration.
// Two mistakes get a fix: a global name written with a module
// (`time.Duration` — the prelude's, D75), and a name the prelude writes for
// another module (`json.Value` for `codec.Value`). Anything else gets the
// module's closest public name as a guess (`io.printn`).
func (c *Checker) noMember(modSpan, nameSpan source.Span, mod, name string, m *Module) {
	whole := source.Span{File: modSpan.File, Start: modSpan.Start, End: nameSpan.End}
	if sym := c.universe.LookupLocal(name); sym != nil && sym.Pub {
		c.errorFix(nameSpan, fixReplace("Write '"+name+"'", whole, name),
			"module '%s' has no declaration '%s'; '%s' is global (the prelude): write '%s'", mod, name, name, name)
		return
	}
	if home := PreludeHome(name); home != "" && home != mod {
		c.errorf(nameSpan, "module '%s' has no declaration '%s'; it is '%s.%s' (add 'use %s')", mod, name, home, name, home)
		return
	}
	if hit := didYouMean(name, moduleMembers(m)); hit != "" {
		c.errorFix(nameSpan, typoFix(nameSpan, hit), "module '%s' has no declaration '%s'; did you mean '%s.%s'?", mod, name, mod, hit)
		return
	}
	c.errorf(nameSpan, "module '%s' has no declaration '%s'", mod, name)
}
