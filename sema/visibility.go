package sema

import "strings"

// D142: four levels. `private` belongs to a type's own declarations (M5),
// `internal` to the module, nothing to the package, `public` to other
// packages. The standard library is one package, whatever its modules.

// pkgOfPath is the package a module belongs to, from its module path
// ("dep/acme/shapes" or "std/io" or "geo") or its dotted mangle prefix
// ("dep.acme.shapes"): a dependency's own, "std", or "" for the entry's.
func pkgOfPath(path string) string {
	for _, sep := range []string{"/", "."} {
		if rest, ok := strings.CutPrefix(path, "dep"+sep); ok {
			name, _, _ := strings.Cut(rest, sep)
			return "dep" + sep + name
		}
		if path == "std" || strings.HasPrefix(path, "std"+sep) {
			return "std"
		}
	}
	return ""
}

// samePackage reports whether two module paths (or prefixes) are modules
// of one package.
func samePackage(a, b string) bool {
	return strings.ReplaceAll(pkgOfPath(a), ".", "/") == strings.ReplaceAll(pkgOfPath(b), ".", "/")
}

// declVisible is the one visibility rule for a top-level declaration, a
// field, a method or a static member, seen from module `from`: the owner's
// own module always; `internal` stops there; the same package sees the rest;
// another package sees `public` (or what a facade exported).
func declVisible(pub, internal, exported bool, ownerPath, fromPath string) bool {
	switch {
	case ownerPath == fromPath:
		return true
	case internal:
		return false
	case samePackage(ownerPath, fromPath):
		return true
	}
	return pub || exported
}

// symVisible reports whether a module member can be named from module from.
func symVisible(sym *Symbol, from *Module) bool {
	if sym.Module == nil || from == nil {
		return sym.Pub || sym.Module == nil
	}
	return declVisible(sym.Pub, sym.Internal, sym.Exported, sym.Module.Path, from.Path)
}

// fieldVisible is symVisible for a field of a type declared in the module
// whose mangle prefix is owner (`private` is settled by the caller).
func fieldVisible(pub, internal bool, ownerPrefix string, from *Module) bool {
	return declVisible(pub, internal, false, ownerPrefix, from.prefix())
}

// memberVisible is symVisible for a member looked up through the module
// symbol `via`: a module a facade exported whole (`public use m`) also shows
// its unmarked declarations to other packages.
func memberVisible(via, member *Symbol, from *Module) bool {
	if via != nil && via.Facade && !member.Internal {
		return true
	}
	return symVisible(member, from)
}
