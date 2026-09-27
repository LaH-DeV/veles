package sema

import (
	"sort"
	"strings"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// ExplainDerived renders every implement the compiler synthesized for the
// package (D58) as Veles source: the header, then the methods it wrote.
// `only`, when non-empty, keeps just the implements whose target is that
// type. The package is type-checked first, since nothing is derived until
// the impls it needs exist; errors go to diags and what was derived before
// them is still returned.
func ExplainDerived(pkg *Package, diags *source.Diagnostics, only string) []string {
	_, c := checkCollect(pkg, diags, false, false, nil)
	if c == nil {
		return nil
	}
	var out []string
	seen := map[*Impl]bool{}
	for _, list := range c.impls {
		for _, impl := range list {
			if seen[impl] || impl.Decl == nil || len(impl.Derived) == 0 {
				continue
			}
			seen[impl] = true
			// the standard library derives for its own enums too; this
			// command answers for the code in front of the user, unless
			// they name a std type outright
			if only == "" && impl.Module != nil && strings.HasPrefix(impl.Module.Path, "std/") {
				continue
			}
			if only != "" && targetName(impl.Target) != only {
				continue
			}
			out = append(out, dumpDerivedImpl(impl))
		}
	}
	sort.Strings(out)
	return out
}

// targetName is the bare name of an implement's target, as a user would
// type it on the command line: `Page` for `Page<T>`, `Point` for `geo.Point`.
func targetName(t types.Type) string {
	switch t := t.(type) {
	case *types.Struct:
		return templateOf(t).Name
	case *types.Sealed:
		return sealedTemplate(t).Name
	case *types.Enum:
		return t.Name
	}
	if t == nil {
		return ""
	}
	s := t.String()
	if i := strings.IndexAny(s, "<"); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	return s
}

// derivedSummary is the hover text for an implement whose body the compiler
// wrote (D58): the header with any inferred bounds, and the signatures it
// synthesized. The wire shape belongs to the target, not to one implement,
// so indexDerived adds it once below them all.
func (c *Checker) derivedSummary(impl *Impl) string {
	if len(impl.Derived) == 0 || impl.Decl == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(implHeader(impl) + "   // derived (D58)\n")
	for _, md := range impl.Decl.Methods {
		if !impl.Derived[md.Name.Name] {
			continue
		}
		p := &dprinter{}
		sig := *md
		sig.Body = nil
		p.fun(&sig, 1)
		sb.WriteString(strings.TrimRight(p.sb.String(), "\n") + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// wireShape lists, for a struct whose Codable implement was derived, the
// key every field takes, which keys may be missing, and the fields that
// never reach the wire. A `@key` with per-format names is shown as the
// mapping it is: the encoder decides at run time which one applies.
func (c *Checker) wireShape(impl *Impl) string {
	kind := c.derivable(impl.Trait)
	if kind != "Encodable" && kind != "Decodable" {
		return ""
	}
	st, ok := impl.Target.(*types.Struct)
	if !ok {
		return ""
	}
	fields, why := c.derivedFields(st, kind)
	if why != "" {
		return ""
	}
	var keys, optional, skipped []string
	for _, df := range fields {
		if df.skipAll {
			skipped = append(skipped, df.name+" (@skip)")
			continue
		}
		if len(df.skipIn) > 0 {
			skipped = append(skipped, df.name+" (@skip "+strings.Join(df.skipIn, ", ")+")")
		}
		keys = append(keys, df.name+keyText(df))
		switch {
		case df.init:
			optional = append(optional, df.name+" (assigned by init, never read)")
		case df.hasDefault:
			optional = append(optional, df.name+" (field default)")
		case df.nullable && !df.required:
			optional = append(optional, df.name+" (null when missing)")
		}
	}
	var sb strings.Builder
	if len(keys) > 0 {
		sb.WriteString("keys      " + strings.Join(keys, " · ") + "\n")
	}
	if len(optional) > 0 {
		sb.WriteString("optional  " + strings.Join(optional, " · ") + "\n")
	}
	if len(skipped) > 0 {
		sb.WriteString("skipped   " + strings.Join(skipped, " · ") + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// keyText renders the wire name a field takes: nothing when it is the
// field's own name styled by the format, `→ "x"` for one name everywhere,
// or the per-format mapping.
func keyText(df derivedField) string {
	if len(df.keys) == 0 {
		return ""
	}
	if all, ok := df.keys[""]; ok {
		return " → \"" + all + "\""
	}
	var parts []string
	for _, f := range sortedKeys(df.keys) {
		parts = append(parts, "\""+df.keys[f]+"\" ("+f+")")
	}
	return " → " + strings.Join(parts, ", ")
}

// indexDerived records, on the trait name of every implement the author
// wrote whose body the compiler filled in, a reference carrying the
// summary of what was written (D58). It runs after the derivation queue
// has drained, so an empty `implement Codable` can report the Encodable
// and Decodable implements that were made from it — they carry the same
// span as the line that asked for them.
func (c *Checker) indexDerived(mods []*Module) {
	byPos := map[source.Span][]*Impl{}
	for _, list := range c.impls {
		for _, impl := range list {
			if impl.Decl != nil && len(impl.Derived) > 0 {
				byPos[impl.Decl.Pos] = append(byPos[impl.Decl.Pos], impl)
				c.indexWire(impl)
			}
		}
	}
	if len(byPos) == 0 {
		return
	}
	for _, m := range mods {
		for _, f := range m.Files {
			for _, decl := range f.Decls {
				d, ok := decl.(*ast.ImplDecl)
				if !ok || d.Derived || d.Extend || d.Trait == nil {
					continue
				}
				made := byPos[d.Pos]
				if len(made) == 0 {
					continue
				}
				sort.Slice(made, func(i, j int) bool { return made[i].Trait.Name < made[j].Trait.Name })
				var parts []string
				wire := ""
				for _, impl := range made {
					if s := c.derivedSummary(impl); s != "" {
						parts = append(parts, s)
					}
					if wire == "" {
						// the same for every implement made from this line:
						// it is the target's shape, not one trait's
						wire = c.wireShape(impl)
					}
				}
				if len(parts) == 0 {
					continue
				}
				text := strings.Join(parts, "\n\n")
				if wire != "" {
					text += "\n\n" + wire
				}
				c.refDerivedImpl(d, text)
			}
		}
	}
}

// indexWire records in Index.Wire the declarations whose names a derived
// Encodable/Decodable/Codable implement puts on the wire: the fields not
// skipped and the variants, each unless a `@key` names it for every
// format. Enum members are left out: every enum is Codable without asking,
// so marking them would refuse every member rename, encoded or not.
func (c *Checker) indexWire(impl *Impl) {
	switch impl.Trait.Name {
	case "Encodable", "Decodable", "Codable":
	default:
		return
	}
	if c.index.Wire == nil {
		c.index.Wire = map[source.Span]string{}
	}
	named := func(attrs []*ast.Attribute) bool {
		_, ok := wireKeys(attrs)[""]
		return ok
	}
	put := func(span source.Span, target string) {
		if _, seen := c.index.Wire[span]; !seen && span.IsValid() {
			c.index.Wire[span] = impl.Trait.Name + " for " + target
		}
	}
	fields := func(st *types.Struct) {
		d, ok := templateOf(st).Decl.(*ast.StructDecl)
		if !ok {
			return
		}
		for _, f := range d.Fields {
			if !named(f.Attrs) && !hasAttr(f.Attrs, "skip") {
				put(f.Name.Pos, d.Name.Name)
			}
		}
	}
	switch t := impl.Target.(type) {
	case *types.Struct:
		fields(t)
	case *types.Sealed:
		for _, v := range sealedTemplate(t).Variants {
			if d, ok := templateOf(v).Decl.(*ast.StructDecl); ok && !named(d.Attrs) {
				put(d.Name.Pos, t.Name)
			}
			fields(v)
		}
	}
}

// refDerivedImpl records the reference on the implement's trait name. The
// trait's own methods are left out of Shape: the summary already lists the
// signatures, and a hover has room for one of the two.
func (c *Checker) refDerivedImpl(d *ast.ImplDecl, text string) {
	span := d.Trait.Span()
	if !span.IsValid() {
		return
	}
	ref := Ref{Span: span, Kind: "trait", Detail: "implement", Derived: text}
	for _, list := range c.impls {
		for _, impl := range list {
			if impl.Decl == d && impl.Trait != nil {
				ref.Name = impl.Trait.Name
				ref.Type = impl.Trait
				ref.Detail = traitHead(impl.Trait)
				ref.Doc = docOfType(impl.Trait)
				if td, ok := impl.Trait.Decl.(*ast.TraitDecl); ok {
					ref.Def = td.Name.Pos
				}
			}
		}
	}
	c.index.Refs = append(c.index.Refs, ref)
}
