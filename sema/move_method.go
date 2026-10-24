package sema

import (
	"reflect"

	"github.com/LaH-DeV/veles/ast"
	"github.com/LaH-DeV/veles/types"
)

// D201: D188's move of `acc.items` out of a struct local, widened to a local
// whose methods were called. A method is summarised from its syntax tree
// (bare member names read as `this.name`, D158), so the answer does not
// depend on the order bodies are checked in:
//
//   - keeps: `this` may outlive the call — returned, stored, passed on,
//     captured by a lambda, handed to a task, its method taken as a value;
//   - used: the fields it mentions;
//   - aliased: the fields whose storage it may hand out or replace — any
//     mention but a receiver of the collection's own methods, a loop that
//     binds no pointer, or an argument to a function that does not keep it.
//
// A method's summary is the union over every method it reaches through
// `this` (a cycle included); anything not resolved to a method with a body of
// its own — a trait's default, an extern, a decorated one — keeps.

type methodFacts struct {
	keeps   bool
	used    map[string]bool
	aliased map[string]bool
	calls   []string // methods of the same type called on `this`
}

// receiverSummary is what calling method name on a st does with the
// receiver, the methods it reaches included; nil when it cannot be told.
func (c *Checker) receiverSummary(st *types.Struct, name string) *methodFacts {
	tmpl := templateOf(st)
	out := &methodFacts{used: map[string]bool{}, aliased: map[string]bool{}}
	seen := map[string]bool{}
	work := []string{name}
	for len(work) > 0 {
		m := work[len(work)-1]
		work = work[:len(work)-1]
		if seen[m] {
			continue
		}
		seen[m] = true
		facts := c.methodFactsOf(tmpl, m)
		if facts.keeps {
			return nil
		}
		for f := range facts.used {
			out.used[f] = true
		}
		for f := range facts.aliased {
			out.aliased[f] = true
		}
		work = append(work, facts.calls...)
	}
	return out
}

// methodFactsOf is one method's own facts, memoized.
func (c *Checker) methodFactsOf(tmpl *types.Struct, name string) *methodFacts {
	t := c.methods[tmpl][name]
	keeps := &methodFacts{keeps: true}
	if t == nil || t.Decl == nil {
		return keeps
	}
	d := t.Decl
	if c.receiverFacts == nil {
		c.receiverFacts = map[*ast.FunDecl]*methodFacts{}
	}
	if facts := c.receiverFacts[d]; facts != nil {
		return facts
	}
	facts := c.computeMethodFacts(t, tmpl)
	c.receiverFacts[d] = facts
	return facts
}

func (c *Checker) computeMethodFacts(t *FuncTemplate, st *types.Struct) *methodFacts {
	d := t.Decl
	keeps := &methodFacts{keeps: true}
	if d.Extern || d.ExportC || d.Static || t.Trait != nil || t.Exec != nil || len(c.annotations[d]) > 0 {
		return keeps
	}
	var body any
	switch {
	case d.Body != nil:
		body = d.Body
	case d.ExprBody != nil:
		body = d.ExprBody
	default:
		return keeps
	}
	// a member written bare is `this.name` unless the body binds the name
	members := map[string]bool{}
	for _, fl := range st.Fields {
		members[fl.Name] = true
	}
	for m := range c.methods[st] {
		members[m] = true
	}
	bw := &bareWalker{bound: map[string]bool{}, seen: map[uintptr]bool{}}
	for _, p := range d.Params {
		bw.bound[p.Name.Name] = true
	}
	bw.walk(reflect.ValueOf(body), true)
	isField := func(name string) bool { return structHasField(st, name) }
	// thisMember is the member `e` names on the receiver: `this.name` or a
	// bare `name`
	thisMember := func(e ast.Expr) (string, bool) {
		switch e := e.(type) {
		case *ast.MemberExpr:
			if _, ok := e.X.(*ast.SelfExpr); ok && !e.Safe {
				return e.Name.Name, true
			}
		case *ast.NameExpr:
			if members[e.Name] && !bw.bound[e.Name] && len(e.TypeArgs) == 0 {
				return e.Name, true
			}
		}
		return "", false
	}

	facts := &methodFacts{used: map[string]bool{}, aliased: map[string]bool{}}
	allowed := map[ast.Expr]bool{} // field mentions that keep nothing, and method calls
	scope := c.fileScope(t.Module, t.File)
	declared := identNames(d)
	walkAST(body, func(n any) bool {
		if facts.keeps {
			return false
		}
		switch n := n.(type) {
		case *ast.LambdaExpr:
			walkAST(n.Body, func(m any) bool {
				switch m := m.(type) {
				case *ast.SelfExpr:
					facts.keeps = true
				case ast.Expr:
					if _, ok := thisMember(m); ok {
						facts.keeps = true
					}
				}
				return !facts.keeps
			})
			return false
		case *ast.CallExpr:
			if n.Async || n.Block {
				facts.keeps = true // a task holds what it is handed
				return false
			}
			if name, ok := thisMember(n.Fun); ok && !isField(name) {
				facts.calls = append(facts.calls, name)
				allowed[n.Fun] = true
			}
			if m, ok := n.Fun.(*ast.MemberExpr); ok && !m.Safe && moveMethods[m.Name.Name] {
				if name, ok := thisMember(m.X); ok && isField(name) {
					allowed[m.X] = true
				}
			}
			for i, a := range n.Args {
				if name, ok := thisMember(a.Value); ok && isField(name) && c.argStays(n, i, scope, declared) {
					allowed[a.Value] = true
				}
			}
		case *ast.LoopStmt:
			if name, ok := thisMember(n.Iter); ok && isField(name) && (n.Var == nil || !hasRefBinding(n.Var)) {
				allowed[n.Iter] = true
			}
		case *ast.ValStmt:
			if name, ok := thisMember(n.Value); ok && isField(name) && copiesElements(n.Pattern) {
				allowed[n.Value] = true
			}
		case *ast.AssignStmt:
			if name, ok := thisMember(n.Target); ok && isField(name) {
				facts.used[name] = true
				facts.aliased[name] = true // replaced: which collection is it now?
				allowed[n.Target] = true
			}
		case *ast.MemberExpr:
			if name, ok := thisMember(n); ok {
				facts.member(n, name, isField(name), allowed)
				return false // its `this` is accounted for
			}
		case *ast.NameExpr:
			if name, ok := thisMember(n); ok {
				facts.member(n, name, isField(name), allowed)
			}
		case *ast.SelfExpr:
			facts.keeps = true // `this` itself: passed, returned, stored, `&this`
		}
		return true
	})
	if facts.keeps {
		return keeps
	}
	return facts
}

// copiesElements reports whether p is a list pattern that only copies
// elements out — `val [.., last] = xs else …` — and binds no rest, which
// could share the storage.
func copiesElements(p ast.Pattern) bool {
	lp, ok := p.(*ast.ListPat)
	if !ok {
		return false
	}
	for _, e := range lp.Elems {
		if r, ok := e.(*ast.RestPat); ok && r.Name != nil {
			return false
		}
	}
	return true
}

// member records one mention of the receiver's member.
func (facts *methodFacts) member(e ast.Expr, name string, field bool, allowed map[ast.Expr]bool) {
	switch {
	case field:
		facts.used[name] = true
		if !allowed[e] {
			facts.aliased[name] = true
		}
	case !allowed[e]:
		facts.keeps = true // a method taken as a value holds its receiver
	}
}
