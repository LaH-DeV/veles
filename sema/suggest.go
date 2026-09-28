package sema

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Typo suggestions: "unknown name 'countr'; did you mean 'counter'?".
//
// A suggestion is a guess, so its fix is marked Guess: the editor offers it
// as a quick fix, `veles check --fix` never applies it.

// didYouMean returns the candidate a misspelt name most likely meant, or "".
//
// First, a name another language uses for the same thing (`size`, `push_back`,
// `toUpperCase`; foreignNames) when its Veles name is among the candidates.
// Then the nearest candidate by edits — an insertion, a deletion, a
// substitution or two neighbours swapped — ignoring case; a candidate
// differing only in case always wins. A candidate further than a third of
// the name's length is a different word, not a typo, unless one name is a
// long prefix of the other (`toUppercase` for `toUpper`). Names of one or
// two characters get no suggestion: every short name is near every other.
// Ties go to the candidate sharing the longest subsequence with the name
// (`printn` is `println`, not `print`), then to the alphabetically first,
// so the message is the same on every run.
func didYouMean(name string, cands []string) string {
	n := utf8.RuneCountInString(name)
	if n < 3 {
		return ""
	}
	lower := strings.ToLower(name)
	have := map[string]bool{}
	for _, c := range cands {
		have[c] = true
	}
	for _, alt := range foreignNames[strings.ReplaceAll(lower, "_", "")] {
		if have[alt] && alt != name {
			return alt
		}
	}
	limit := n / 3
	best, bestDist, bestCommon := "", limit+1, -1
	sorted := make([]string, 0, len(have))
	for c := range have {
		sorted = append(sorted, c)
	}
	sort.Strings(sorted)
	for _, c := range sorted {
		if c == name || c == "" || !isPlainName(c) {
			continue
		}
		lc := strings.ToLower(c)
		d := editDistance(lower, lc, bestDist+1)
		switch {
		case lc == lower:
			d = 0
		case d > limit && longPrefix(lower, lc):
			d = limit
		}
		if d > bestDist || d > limit {
			continue
		}
		common := commonSubsequence(lower, lc)
		if d < bestDist || common > bestCommon {
			best, bestDist, bestCommon = c, d, common
		}
	}
	if best == "" {
		// a typo of another language's name: `lenght` is `length`, so `len`
		foreign := make([]string, 0, len(foreignNames))
		for k := range foreignNames {
			foreign = append(foreign, k)
		}
		if k := didYouMeanPlain(strings.ReplaceAll(lower, "_", ""), foreign); k != "" {
			for _, alt := range foreignNames[k] {
				if have[alt] && alt != name {
					return alt
				}
			}
		}
	}
	return best
}

// didYouMeanPlain is the nearest of cands (all lowercase) within a third of
// name's length, without the foreign-name step.
func didYouMeanPlain(name string, cands []string) string {
	limit := utf8.RuneCountInString(name) / 3
	sort.Strings(cands)
	best, bestDist := "", limit+1
	for _, c := range cands {
		if d := editDistance(name, c, bestDist); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

// foreignNames maps what other languages call a member (lowercased, with
// `_` dropped) to the Veles names that may mean the same, most likely
// first. Only a name the type actually has is ever suggested.
var foreignNames = map[string][]string{
	"size":        {"len"},
	"length":      {"len"},
	"count":       {"len"},
	"put":         {"set"},
	"insert":      {"set", "add", "push"},
	"append":      {"push", "add"},
	"add":         {"push", "set"},
	"pushback":    {"push"},
	"get":         {"at"},
	"has":         {"contains", "containsKey"},
	"includes":    {"contains"},
	"containskey": {"contains"},
	"haskey":      {"containsKey"},
	"delete":      {"remove"},
	"erase":       {"remove"},
	"removeat":    {"remove"},
	"reduce":      {"fold"},
	"each":        {"forEach"},
	"sort":        {"sorted"},
	"reverse":     {"reversed"},
	"empty":       {"isEmpty"},
	"strip":       {"trim"},
	"trimleft":    {"trimStart"},
	"trimright":   {"trimEnd"},
	"lstrip":      {"trimStart"},
	"rstrip":      {"trimEnd"},
	"touppercase": {"toUpper"},
	"uppercase":   {"toUpper"},
	"upper":       {"toUpper"},
	"tolowercase": {"toLower"},
	"lowercase":   {"toLower"},
	"lower":       {"toLower"},
	"substr":      {"substring"},
	"parseint":    {"toInt"},
	"keyset":      {"keys"},
	"items":       {"entries"},
	"findindex":   {"indexOf"},
	"every":       {"all"},
	"some":        {"any"},
}

// longPrefix: one name starts the other, and the shorter is most of it.
func longPrefix(a, b string) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	return len(a) >= 4 && 2*len(a) >= len(b) && strings.HasPrefix(b, a)
}

// commonSubsequence is the length of the longest common subsequence.
func commonSubsequence(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			if ra[i-1] == rb[j-1] {
				cur[j] = prev[j-1] + 1
			} else {
				cur[j] = max(prev[j], cur[j-1])
			}
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

// isPlainName excludes the checker's own names (`$testFail`, `__x`), which
// a program never spells.
func isPlainName(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return (unicode.IsLetter(r) || r == '_') && !strings.HasPrefix(s, "__") && !strings.ContainsRune(s, '$')
}

// editDistance is the optimal-string-alignment distance between a and b
// (Levenshtein plus adjacent transpositions), or a value >= stop once it
// cannot come in under stop.
func editDistance(a, b string, stop int) int {
	ra, rb := []rune(a), []rune(b)
	if d := len(ra) - len(rb); d >= stop || -d >= stop {
		return stop
	}
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		rowMin := cur[0]
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			v := min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				v = min(v, prev2[j-2]+1)
			}
			cur[j] = v
			rowMin = min(rowMin, v)
		}
		if rowMin >= stop {
			return stop
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(rb)]
}

// typoFix is the quick fix that writes the suggestion in place of the
// misspelt name.
func typoFix(span source.Span, hit string) *source.Fix {
	if hit == "" || span.File == nil {
		return nil
	}
	fix := fixReplace("Change to '"+hit+"'", span, hit)
	fix.Guess = true
	return fix
}

// scopeNames lists every name visible from sc, innermost scope first.
func scopeNames(sc *Scope) []string {
	var out []string
	for ; sc != nil; sc = sc.parent {
		for name := range sc.symbols {
			out = append(out, name)
		}
	}
	return out
}

// moduleMembers lists the public declarations of an imported module.
func moduleMembers(m *Module) []string {
	if m == nil || m.Scope == nil {
		return nil
	}
	var out []string
	for name, sym := range m.Scope.symbols {
		if sym.Pub {
			out = append(out, name)
		}
	}
	return out
}

// memberNames lists what `x.name` may name on a value of type rt from
// here: its fields, and the methods callable on it — the built-ins, the
// struct's own, those of extend blocks and of the traits it implements.
// Members the reader could not use (private ones, statics) are left out:
// suggesting one would only trade this error for the next.
func (f *fnCtx) memberNames(rt types.Type) (fields, methods []string) {
	if p, ok := rt.(*types.Pointer); ok && !p.Raw {
		rt = p.Elem
	}
	for _, d := range BuiltinMethods(rt) {
		if !d.Static() {
			methods = append(methods, d.Name)
		}
	}
	callable := func(t *FuncTemplate) bool {
		if t.Decl == nil || t.Decl.Static {
			return false
		}
		inherent := t.Owner != nil || (t.Impl != nil && t.Impl.Trait == nil)
		if !inherent {
			return true
		}
		if t.Decl.Private {
			owner := t.Owner
			if owner == nil {
				owner, _ = t.Impl.Target.(*types.Struct)
			}
			return owner == nil || f.insideType(owner)
		}
		return t.Pub || t.Module == f.module
	}
	switch ct := rt.(type) {
	case *types.Struct:
		for _, fld := range ct.Fields {
			if (!fld.Private || f.insideType(ct)) && (fld.Pub || fld.Private || ct.Module == f.module.prefix()) {
				fields = append(fields, fld.Name)
			}
		}
		for name, t := range f.c.methods[templateOf(ct)] {
			if callable(t) {
				methods = append(methods, name)
			}
		}
	case *types.Sealed:
		if tr := sealedTemplate(ct).Trait; tr != nil {
			for name := range tr.Methods {
				methods = append(methods, name)
			}
		}
	case *types.Trait:
		for name := range ct.Methods {
			methods = append(methods, name)
		}
	}
	for _, view := range receiverViews(rt) {
		for _, ext := range f.c.extends {
			if !unify(ext.Target, view, map[*types.TypeParam]types.Type{}) {
				continue
			}
			for name, t := range ext.Methods {
				if callable(t) {
					methods = append(methods, name)
				}
			}
		}
	}
	for trait, impls := range f.c.impls {
		for _, impl := range impls {
			if unify(impl.Target, rt, map[*types.TypeParam]types.Type{}) {
				for name := range trait.Methods {
					methods = append(methods, name)
				}
				break
			}
		}
	}
	return fields, methods
}

// unknownNameHint completes "unknown name/function 'name'": a declaration
// of that name in a module to import (with its certain fix), else the
// closest name in scope (with a guessed one).
func (f *fnCtx) unknownNameHint(span source.Span, name string, bare bool) (string, *source.Fix) {
	var hint string
	if bare {
		hint = f.c.suggestUnknownName(f.module, name)
	} else {
		hint = f.c.suggestUnknown(f.module, name)
	}
	if hint != "" {
		return hint, f.c.unknownFix(f.file, f.module, span, name)
	}
	// inside a method, a member of the receiver named bare: certain, not a
	// guess — the receiver is never implicit (D65)
	if self := f.selfRef(); self != nil {
		rt := self.Type
		if p, ok := rt.(*types.Pointer); ok {
			rt = p.Elem
		}
		fields, methods := f.memberNames(rt)
		for _, m := range append(fields, methods...) {
			if m == name {
				return "; a member is reached through the receiver: 'this." + name + "'", fixReplace("Write 'this."+name+"'", span, "this."+name)
			}
		}
	}
	if hit := didYouMean(name, scopeNames(f.scope)); hit != "" {
		if sym := f.lookup(hit); sym != nil && sym.Kind == SymLocal {
			// the misspelling was its use: without this, "'counter' is
			// never used" came with a fix renaming it to `_`, and
			// `check --fix` threw the binding away over a typo
			markUsed(sym.Var)
		}
		return "; did you mean '" + hit + "'?", typoFix(span, hit)
	}
	return "", nil
}

// noMethodHint completes "no method 'name' on ...": the field of that name
// when there is one (`p.x()` where `x` is a field), else the closest method.
func (f *fnCtx) noMethodHint(rt types.Type, name string) (string, string) {
	fields, methods := f.memberNames(rt)
	for _, fld := range fields {
		if fld == name {
			return "; '" + name + "' is a field, not a method: drop the '()'", ""
		}
	}
	if hit := didYouMean(name, methods); hit != "" {
		return "; did you mean '" + hit + "'?", hit
	}
	return "", ""
}

// noFieldHint completes "has no field 'name'": the method of that name
// when there is one (`xs.len` for `xs.len()`), else the closest field.
func (f *fnCtx) noFieldHint(rt types.Type, name string) (string, string) {
	fields, methods := f.memberNames(rt)
	for _, m := range methods {
		if m == name {
			return "; '" + name + "' is a method: call it, '" + name + "()'", ""
		}
	}
	if hit := didYouMean(name, fields); hit != "" {
		return "; did you mean '" + hit + "'?", hit
	}
	return "", ""
}

// privateHint ends a "private to module" error with what can be done about
// it: declare the member public — when the module is the user's to change.
// `mod` is a module path ("net/http"), or a mangle prefix ("net.http").
func privateHint(mod string) string {
	switch {
	case mod == "std" || strings.HasPrefix(mod, "std/") || strings.HasPrefix(mod, "std."):
		return "; it is not part of the standard library's API"
	case strings.HasPrefix(mod, "dep/") || strings.HasPrefix(mod, "dep."):
		return "; it is not part of that package's API"
	}
	return "; declare it 'public' there to use it from here"
}

// arityError reports a built-in method called with the wrong number of
// arguments, and shows the call as the catalogue writes it for this
// receiver: "'at' takes 1 argument: at(i: i64): string?".
func (f *fnCtx) arityError(span source.Span, recv types.Type, name string, n int) {
	shape := ""
	if d := lookupBuiltinDoc(builtinFamily(recv), name); d != nil {
		shape = ": " + name + receiverSig(d.Sig, recv)
	}
	f.errorf(span, "'%s' takes %d %s%s", name, n, plural(n, "argument"), shape)
}

// plural is `word` for one and `word`+"s" otherwise.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// Nearest is the "did you mean" among candidates for a name outside a
// program — a command's argument — or "" when none is close.
func Nearest(name string, cands []string) string { return didYouMean(name, cands) }

// staticBuilders are what other languages call the function that builds a
// collection of n elements; Veles has `MutableList<T>.make(n, init)`.
var staticBuilders = map[string]bool{"generate": true, "filled": true, "fill": true, "repeat": true, "of": true, "init": true, "new": true, "create": true, "withSize": true, "tabulate": true}

// staticHint completes "no static function 'name' on 'T'": the nearest
// static of T, or of its Mutable counterpart for a List, Map or Set (whose
// builders live on the mutable type), written out in full.
func (f *fnCtx) staticHint(t types.Type, name string) string {
	head := typeHead(t)
	if i := strings.IndexByte(head, '<'); i >= 0 {
		head = head[:i]
	}
	return f.staticHintHead(head, name)
}

// staticHintHead is staticHint by the type's bare name (`List`).
func (f *fnCtx) staticHintHead(head, name string) string {
	owner := map[string]string{}
	var cands []string
	for _, h := range []string{head, "Mutable" + head} {
		for _, ext := range f.c.extends {
			eh := typeHead(ext.Target)
			if i := strings.IndexByte(eh, '<'); i >= 0 {
				eh = eh[:i]
			}
			if eh != h {
				continue
			}
			for mname, mt := range ext.Methods {
				if mt.Decl.Static {
					if _, seen := owner[mname]; !seen {
						owner[mname] = h
						cands = append(cands, mname)
					}
				}
			}
		}
	}
	hit := ""
	if owner[name] != "" {
		hit = name // there, on this type or its mutable counterpart
	} else {
		hit = didYouMean(name, cands)
	}
	if hit == "" && staticBuilders[name] && owner["make"] != "" {
		hit = "make"
	}
	if hit == "" {
		return ""
	}
	return fmt.Sprintf("; did you mean '%s<T>.%s(...)'?", owner[hit], hit)
}

// staticNamed reports whether a generic type has a static function of that
// name at all — in its body or an extend block — so a missing type
// argument is not blamed for a name that does not exist.
func (f *fnCtx) staticNamed(st *types.Struct, name string) bool {
	if t, ok := f.c.methods[templateOf(st)][name]; ok && t.Decl.Static {
		return true
	}
	for _, ext := range f.c.extends {
		if t, ok := ext.Methods[name]; ok && t.Decl.Static {
			if es, isStruct := ext.Target.(*types.Struct); isStruct && templateOf(es) == templateOf(st) {
				return true
			}
		}
	}
	return false
}
