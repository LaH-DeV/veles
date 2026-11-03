package sema

import (
	"sort"
	"strconv"
	"strings"

	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// Copy sites (D231 choice 3): a change to a list, map or set that may copy
// it, because the checker sees the value was shared since its last change —
// the editor's "copies squares (shared at line 12)" hint. A function is
// followed in the order it runs, after the share pass: a Share of a place
// (a variable, its fields, an element of a collection in it) marks it and
// everything inside it shared; a variable bound from a shared value is
// shared; a parameter the function changes is shared from its entry (the
// caller may have kept the value). A change through a shared place is a
// copy site, and the place is its own after it. `if`, `when` and `?:` join
// what their branches leave; a loop's body is followed twice, so a share
// late in the body reaches a change early in it; a path that returns,
// breaks or throws leaves nothing behind.
//
// The hint says "may": a share is a mark the runtime sets, and a value
// shared once is copied at its first change even when the other holder has
// died since.
//
// Two facts cross functions, settled in rounds until nothing new is
// learned: a field another function shared through `this` or a pointer
// (`fun items(): List<T> => this.items`) is shared on entry to every
// method and pointer parameter of that struct (WhyElsewhere), and a
// pointer parameter or receiver some call site handed a shared value is
// shared on entry (WhyCaller).

// CopySite is a change that may copy.
type CopySite struct {
	At       source.Span // the change
	Name     string      // what is copied: `squares`, `inv.items`, `an element of grid`
	SharedAt source.Span // the statement that shared it (the parameter, for WhyParam)
	Why      CopyWhy
	// Once: a loop repeats the change but not the share — only its first
	// round copies (D231 addendum 3)
	Once bool
	// Empty: a clear(), which gives the place an empty value of its own
	// instead of copying the elements it drops — nothing is allocated
	Empty bool
	// By: the function that shared it, for WhyElsewhere and WhyCaller
	By string
}

// CopyWhy is why a copy site's value is shared.
type CopyWhy int

const (
	WhyShared CopyWhy = iota // copied into another holder
	WhyLoop                  // a loop walks the value as it was (D231 addendum)
	WhyParam                 // a parameter: the caller may keep its value
	WhyElsewhere             // a field another function shared (By)
	WhyCaller                // a pointer parameter or receiver a caller (By) handed a shared value
)

// CopySites finds the copy sites of every function of prog, each once
// (generic instances share their sites), ordered by position.
func CopySites(prog *Program) []CopySite {
	var cross, next *copyCross
	var all [][]CopySite
	for round := 0; round < 8; round++ {
		next = &copyCross{fields: map[string]copyMark{}, callers: map[*Var]copyMark{}}
		all = all[:0]
		for _, fn := range prog.Funcs {
			if fn.Body == nil || fn.Extern {
				continue
			}
			cf := &copyFlow{fn: fn, alias: map[*Var]copyPlace{}, learn: next}
			st := newCopyState()
			for _, p := range fn.Params {
				if p != nil && !p.ReadOnly && !p.IsSelf && !p.Captured && holdsCollection(p.Type, map[types.Type]bool{}) {
					st.share(cf.key(copyPlace{root: p}), copyMark{at: p.Span, why: WhyParam})
				}
			}
			cf.st = st
			cross.seed(cf, fn)
			cf.block(fn.Body)
			all = append(all, cf.sites)
		}
		if cross.same(next) {
			break
		}
		cross = next
	}
	seen := map[source.Span]bool{}
	var out []CopySite
	for _, sites := range all {
		for _, s := range sites {
			if !seen[s.At] {
				seen[s.At] = true
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].At, out[j].At
		if a.File != b.File {
			return a.File.Path < b.File.Path
		}
		return a.Start < b.Start
	})
	return out
}

// copyCross is what one round of CopySites learns for the next: the struct
// fields shared through a pointer (by "Struct.field"), and the pointer
// parameters and receivers a call site handed a shared value.
type copyCross struct {
	fields  map[string]copyMark
	callers map[*Var]copyMark
}

// same: nothing new was learned (a nil c has learned nothing)
func (c *copyCross) same(o *copyCross) bool {
	if c == nil {
		return len(o.fields) == 0 && len(o.callers) == 0
	}
	if len(c.fields) != len(o.fields) || len(c.callers) != len(o.callers) {
		return false
	}
	for k := range o.fields {
		if _, ok := c.fields[k]; !ok {
			return false
		}
	}
	for v := range o.callers {
		if _, ok := c.callers[v]; !ok {
			return false
		}
	}
	return true
}

// seed marks what is shared on entry to fn from the last round's facts.
func (c *copyCross) seed(cf *copyFlow, fn *Func) {
	if c == nil {
		return
	}
	roots := append([]*Var{fn.Receiver}, fn.Params...)
	for _, v := range roots {
		if v == nil {
			continue
		}
		if m, ok := c.callers[v]; ok {
			cf.st.share(cf.key(copyPlace{root: v}), m)
		}
		if !v.IsSelf && !isPointer(v.Type) {
			continue // a value parameter's fields are its own copy's
		}
		st := structBehind(v.Type)
		if st == nil {
			continue
		}
		for i := range st.Fields {
			if m, ok := c.fields[st.String()+"."+st.Fields[i].Name]; ok {
				k := cf.key(copyPlace{root: v, path: []string{"." + st.Fields[i].Name}})
				if _, already := cf.st.sharedAt(k); !already {
					cf.st.share(k, m)
				}
			}
		}
	}
}

// funcLabel names fn for a hint: `Shelf.all` for a method.
func funcLabel(fn *Func) string {
	if fn.Receiver != nil {
		if st := structBehind(fn.Receiver.Type); st != nil && !strings.Contains(fn.Display, ".") {
			return st.Name + "." + fn.Display
		}
	}
	return fn.Display
}

// structBehind: the struct t is, or points to.
func structBehind(t types.Type) *types.Struct {
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
	}
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem
	}
	st, _ := t.(*types.Struct)
	return st
}

// copyPlace is a variable and a path below it: field names and "[]" for any
// element of a collection.
type copyPlace struct {
	root *Var
	path []string
}

type copyMark struct {
	at    source.Span
	why   CopyWhy
	loops int    // the loops around the share
	by    string // the function that shared it, when it is another
}

// copyState: the shared places by key (a key covers every place below it)
// and the places made unique since they were shared; dead after a path that
// does not go on.
type copyState struct {
	shared map[string]copyMark
	unique map[string]bool
	dead   bool
}

func newCopyState() *copyState {
	return &copyState{shared: map[string]copyMark{}, unique: map[string]bool{}}
}

func (s *copyState) clone() *copyState {
	c := &copyState{shared: make(map[string]copyMark, len(s.shared)), unique: make(map[string]bool, len(s.unique)), dead: s.dead}
	for k, m := range s.shared {
		c.shared[k] = m
	}
	for k := range s.unique {
		c.unique[k] = true
	}
	return c
}

// join: shared where either is, unique where both are; a dead state adds
// nothing.
func (s *copyState) join(o *copyState) {
	switch {
	case o.dead:
		return
	case s.dead:
		*s = *o.clone()
		return
	}
	for k, m := range o.shared {
		if _, ok := s.shared[k]; !ok {
			s.shared[k] = m
		}
	}
	for k := range s.unique {
		if !o.unique[k] {
			delete(s.unique, k)
		}
	}
}

// placeUnder: key k is the place p or one below it
func placeUnder(k, p string) bool {
	return k == p || strings.HasPrefix(k, p+"/")
}

func (s *copyState) share(k string, m copyMark) {
	for u := range s.unique {
		if placeUnder(u, k) {
			delete(s.unique, u)
		}
	}
	for o := range s.shared {
		if placeUnder(o, k) {
			delete(s.shared, o)
		}
	}
	s.shared[k] = m
}

// sharedAt: the mark that makes place k shared, if any
func (s *copyState) sharedAt(k string) (copyMark, bool) {
	if s.unique[k] {
		return copyMark{}, false
	}
	for p := k; ; {
		if m, ok := s.shared[p]; ok {
			return m, true
		}
		i := strings.LastIndexByte(p, '/')
		if i < 0 {
			return copyMark{}, false
		}
		p = p[:i]
	}
}

// own: place k was copied by a change, or holds a fresh value
func (s *copyState) own(k string) {
	if _, ok := s.shared[k]; ok {
		delete(s.shared, k)
	} else {
		s.unique[k] = true
	}
}

// rebind: place k holds a new value, shared (m) or fresh (nil)
func (s *copyState) rebind(k string, m *copyMark) {
	for u := range s.unique {
		if placeUnder(u, k) {
			delete(s.unique, u)
		}
	}
	for o := range s.shared {
		if placeUnder(o, k) {
			delete(s.shared, o)
		}
	}
	if m != nil {
		s.shared[k] = *m
	} else if _, ok := s.sharedAt(k); ok {
		s.unique[k] = true
	}
}

type copyFlow struct {
	fn    *Func
	learn *copyCross // what this round tells the next
	st    *copyState
	alias map[*Var]copyPlace // lent temporaries and lent aliases: the place they read
	ids   map[*Var]int
	pos   source.Span // the statement running, where a share is written
	decl  *Var        // the variable the running VarDecl binds
	loops int         // the loops around the statement running
	sites []CopySite
}

func (f *copyFlow) key(p copyPlace) string {
	if f.ids == nil {
		f.ids = map[*Var]int{}
	}
	id, ok := f.ids[p.root]
	if !ok {
		id = len(f.ids)
		f.ids[p.root] = id
	}
	var b strings.Builder
	b.WriteString(strconv.Itoa(id))
	for _, s := range p.path {
		b.WriteByte('/')
		b.WriteString(s)
	}
	return b.String()
}

// place is e as a place, through fields, dereferences, unwraps and element
// accesses, and through the lent temporaries that stand for one.
func (f *copyFlow) place(e Expr) (copyPlace, bool) {
	switch x := e.(type) {
	case *VarRef:
		if p, ok := f.alias[x.Var]; ok {
			return p, true
		}
		return copyPlace{root: x.Var}, true
	case *FieldGet:
		p, ok := f.place(x.X)
		if !ok {
			return p, false
		}
		return copyPlace{p.root, append(append([]string(nil), p.path...), "."+x.Name)}, true
	case *TupleGet:
		p, ok := f.place(x.X)
		if !ok {
			return p, false
		}
		return copyPlace{p.root, append(append([]string(nil), p.path...), "#"+strconv.Itoa(x.Index))}, true
	case *Deref:
		return f.place(x.X)
	case *Unwrap:
		return f.place(x.X)
	case *VariantCast:
		return f.place(x.X)
	case *Builtin:
		switch x.Op {
		case "list.ref", "list.refUnchecked", "list.get", "list.getUnchecked", "map.ref", "map.refOrPanic", "map.get":
			p, ok := f.place(x.Args[0])
			if !ok {
				return p, false
			}
			return copyPlace{p.root, append(append([]string(nil), p.path...), "[]")}, true
		}
	}
	return copyPlace{}, false
}

// copyName is a place as the reader wrote it, or "" for a compiler temporary.
func copyName(p copyPlace) string {
	if p.root == nil || strings.HasPrefix(p.root.Name, "$") {
		return ""
	}
	s := p.root.Name
	for i, seg := range p.path {
		switch {
		case seg == "[]":
			if i == len(p.path)-1 {
				return "an element of " + s
			}
			s += "[…]"
		case strings.HasPrefix(seg, "#"):
			s += "." + seg[1:]
		default:
			s += seg
		}
	}
	return s
}

// pointsAt is the place a pointer expression points at: `&place`, through
// the wraps, blocks and branches `xs.at(i)` lowers to.
func (f *copyFlow) pointsAt(e Expr) (copyPlace, bool) {
	switch x := e.(type) {
	case *AddrOf:
		return f.place(x.X)
	case *SomeWrap:
		return f.pointsAt(x.X)
	case *Unwrap:
		return f.pointsAt(x.X)
	case *BlockExpr:
		if x.Block != nil && x.Block.Value != nil {
			return f.pointsAt(x.Block.Value)
		}
	case *If:
		if x.Then != nil && x.Then.Value != nil {
			if p, ok := f.pointsAt(x.Then.Value); ok {
				return p, true
			}
		}
		if x.Else != nil && x.Else.Value != nil {
			return f.pointsAt(x.Else.Value)
		}
	case *Let:
		return f.pointsAt(x.Body)
	case *VarRef:
		if p, ok := f.alias[x.Var]; ok {
			return p, true
		}
	}
	return copyPlace{}, false
}

// isPointer: t is a pointer, or a nullable one
func isPointer(t types.Type) bool {
	if n, ok := t.(*types.Nullable); ok {
		t = n.Elem
	}
	_, ok := t.(*types.Pointer)
	return ok
}

// sharedValue reports whether e's value is a second holder of something
// that stays alive: a Share wherever the value comes from.
func sharedValue(e Expr) bool {
	switch x := e.(type) {
	case *Share:
		return true
	case *SomeWrap:
		return sharedValue(x.X)
	case *Unwrap:
		return sharedValue(x.X)
	case *Elvis:
		return sharedValue(x.L) || sharedValue(x.R)
	case *BlockExpr:
		return x.Block != nil && x.Block.Value != nil && sharedValue(x.Block.Value)
	case *If:
		return x.Then != nil && x.Then.Value != nil && sharedValue(x.Then.Value) ||
			x.Else != nil && x.Else.Value != nil && sharedValue(x.Else.Value)
	case *Let:
		return sharedValue(x.Body)
	}
	return false
}

// bind: place p receives init's value
func (f *copyFlow) bind(p copyPlace, init Expr) {
	if !holdsCollection(p.root.Type, map[types.Type]bool{}) {
		return
	}
	k := f.key(p)
	if init != nil && sharedValue(init) {
		f.st.rebind(k, &copyMark{at: f.pos, why: WhyShared, loops: f.loops})
		return
	}
	f.st.rebind(k, nil)
}

// change: the collection at e is changed at span at — through any element
// accesses on the way, each a change of the collection holding it
func (f *copyFlow) change(e Expr, at source.Span, empty bool) {
	for x := e; x != nil; {
		switch y := x.(type) {
		case *Builtin:
			if y.Op == "list.ref" || y.Op == "list.refUnchecked" || y.Op == "map.ref" || y.Op == "map.refOrPanic" {
				f.change(y.Args[0], at, false)
			}
			x = nil
		case *FieldGet:
			x = y.X
		case *TupleGet:
			x = y.X
		case *Deref:
			x = y.X
		case *Unwrap:
			x = y.X
		case *VariantCast:
			x = y.X
		default:
			x = nil
		}
	}
	if _, ok := e.Type().(*types.Array); ok {
		return // an array is held inline: changed in place, never copied
	}
	p, ok := f.place(e)
	if !ok || f.st.dead {
		return
	}
	k := f.key(p)
	if m, shared := f.st.sharedAt(k); shared {
		// a change the compiler wrote (no span) copies as well, unseen
		if n := copyName(p); n != "" && at.IsValid() {
			f.sites = append(f.sites, CopySite{At: at, Name: n, SharedAt: m.at, Why: m.why, Empty: empty, Once: f.loops > m.loops && m.why != WhyElsewhere && m.why != WhyCaller, By: m.by})
		}
		f.st.own(k)
	}
}

func (f *copyFlow) block(b *Block) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		f.stmt1(s)
	}
	if b.Value != nil {
		f.expr(b.Value)
	}
}

func (f *copyFlow) stmt1(s Stmt) {
	switch s := s.(type) {
	case *Block:
		f.block(s)
	case *VarDecl:
		f.at(s.Pos)
		v := s.Var
		if v.Lent && v.LentFor == nil && s.Init != nil {
			if p, ok := f.place(s.Init); ok {
				f.alias[v] = p // reads the place it stands for
				f.expr(s.Init)
				return
			}
		}
		outer := f.decl
		f.decl = v
		f.expr(s.Init)
		f.decl = outer
		if isPointer(v.Type) {
			if p, ok := f.pointsAt(s.Init); ok {
				f.alias[v] = p // a change through it is a change of the place
			}
			return
		}
		f.at(s.Pos)
		f.bind(copyPlace{root: v}, s.Init)
	case *Assign:
		f.at(s.Pos)
		f.expr(s.Value)
		f.target(s.Target)
		if p, ok := f.place(s.Target); ok && holdsCollection(s.Target.Type(), map[types.Type]bool{}) {
			k := f.key(p)
			if sharedValue(s.Value) {
				f.st.rebind(k, &copyMark{at: s.Pos, why: WhyShared, loops: f.loops})
			} else {
				f.st.rebind(k, nil)
			}
		}
	case *ExprStmt:
		f.at(s.Pos)
		f.expr(s.X)
		switch s.X.(type) {
		case *Throw, *Panic:
			f.st.dead = true
		}
	case *Return:
		f.at(s.Pos)
		f.expr(s.Value)
		f.st.dead = true
	case *Break, *Continue:
		f.st.dead = true
	case *Loop:
		f.loop(s)
	case *With:
		f.at(s.Pos)
		f.expr(s.Init)
		f.block(s.Body)
		f.expr(s.Close)
	case *ScopeBlock:
		f.expr(s)
	}
}

// target: an assignment through an element access changes the collection
// holding the element (`grid.at(0)?.x = 1`); the hint goes on the access.
func (f *copyFlow) target(e Expr) {
	for x := e; x != nil; {
		switch y := x.(type) {
		case *Builtin:
			if y.Op == "list.ref" || y.Op == "list.refUnchecked" || y.Op == "map.ref" || y.Op == "map.refOrPanic" {
				f.change(y.Args[0], y.Span, false)
			}
			return
		case *FieldGet:
			x = y.X
		case *TupleGet:
			x = y.X
		case *Deref:
			x = y.X
		case *Unwrap:
			x = y.X
		case *VariantCast:
			x = y.X
		default:
			return
		}
	}
}

// loop: the body twice — the second time with what an iteration leaves
// joined in — and the state after it joins the loop not running at all
func (f *copyFlow) loop(l *Loop) {
	if f.st.dead {
		return
	}
	before := f.st.clone()
	f.loops++
	defer func() { f.loops-- }()
	for pass := 0; pass < 2; pass++ {
		f.st.dead = false
		f.at(l.Pos)
		f.expr(l.Cond)
		f.block(l.Body)
		f.st.dead = false
		for _, p := range l.Post {
			f.stmt1(p)
		}
		if pass == 0 {
			f.st.join(before)
		}
	}
	f.st.dead = false
	f.st.join(before)
}

func (f *copyFlow) expr(e Expr) {
	if e == nil || f.st.dead {
		return
	}
	switch x := e.(type) {
	case *If:
		f.expr(x.Cond)
		after := f.st
		f.st = after.clone()
		f.block(x.Then)
		then := f.st
		f.st = after.clone()
		f.block(x.Else)
		f.st.join(then)
		return
	case *Elvis:
		f.expr(x.L)
		after := f.st
		f.st = after.clone()
		f.expr(x.R)
		f.st.join(after)
		return
	case *Match:
		f.expr(x.Init)
		after := f.st
		var out *copyState
		for _, arm := range x.Arms {
			f.st = after.clone()
			f.expr(arm.Test)
			for _, b := range arm.Binds {
				f.stmt1(b)
			}
			f.expr(arm.Guard)
			f.block(arm.Body)
			if out == nil {
				out = f.st
			} else {
				out.join(f.st)
			}
		}
		if out == nil {
			out = after
		} else if !x.Exhaustive {
			out.join(after)
		}
		f.st = out
		return
	case *Let:
		f.expr(x.Init)
		if p, ok := f.place(x.Init); ok && x.Var.Lent {
			f.alias[x.Var] = p
		} else if isPointer(x.Var.Type) {
			if p, ok := f.pointsAt(x.Init); ok {
				f.alias[x.Var] = p
			}
		} else {
			f.bind(copyPlace{root: x.Var}, x.Init)
		}
		f.expr(x.Body)
		return
	case *Closure:
		return // a separate function; what it captures was shared where captured
	case *Share:
		f.expr(x.X)
		if p, ok := f.place(x.X); ok {
			why := WhyShared
			if f.decl != nil && f.decl.LentFor != nil {
				why = WhyLoop
			}
			f.st.share(f.key(p), copyMark{at: f.pos, why: why, loops: f.loops})
			// a field of a struct reached through a pointer: every function
			// with that struct behind a pointer may see it shared
			if len(p.path) > 0 && strings.HasPrefix(p.path[0], ".") && (p.root.IsSelf || isPointer(p.root.Type)) {
				if st := structBehind(p.root.Type); st != nil {
					k := st.String() + p.path[0]
					if _, ok := f.learn.fields[k]; !ok {
						f.learn.fields[k] = copyMark{at: f.pos, why: WhyElsewhere, by: funcLabel(f.fn)}
					}
				}
			}
		}
		return
	case *Call:
		eachChild(e, f)
		// a pointer handed to the callee whose value is shared here: the
		// callee's first change through it copies
		params := x.Fn.Params
		args := x.Args
		if x.Fn.Receiver != nil && len(args) > 0 {
			f.callerShare(x.Fn.Receiver, args[0])
			args = args[1:]
		}
		for i, a := range args {
			if i < len(params) && params[i] != nil && isPointer(params[i].Type) {
				f.callerShare(params[i], a)
			}
		}
		return
	case *Builtin:
		eachChild(e, f)
		if collectionWrites[x.Op] && len(x.Args) > 0 {
			f.change(x.Args[0], x.Span, x.Op == "list.clear" || x.Op == "map.clear")
		}
		return
	case *Throw, *Panic:
		eachChild(e, f)
		f.st.dead = true
		return
	case *AddrOf:
		// a pointer to an element (`xs.at(i)`) makes its collection the
		// place's own first
		eachChild(e, f)
		if b, ok := x.X.(*Builtin); ok && (b.Op == "list.ref" || b.Op == "list.refUnchecked") {
			f.change(b.Args[0], b.Span, false)
		}
		return
	}
	eachChild(e, f)
}

// callerShare records that param receives a pointer to a shared value.
func (f *copyFlow) callerShare(param *Var, arg Expr) {
	if f.st.dead || f.learn == nil {
		return
	}
	p, ok := f.pointsAt(arg)
	if !ok {
		return
	}
	if _, shared := f.st.sharedAt(f.key(p)); !shared {
		return
	}
	if _, ok := f.learn.callers[param]; !ok {
		f.learn.callers[param] = copyMark{at: f.pos, why: WhyCaller, by: funcLabel(f.fn)}
	}
}

// at: the statement at sp runs; one the compiler made (no span) runs as
// part of the one written around it
func (f *copyFlow) at(sp source.Span) {
	if sp.IsValid() {
		f.pos = sp
	}
}

// block and stmt make copyFlow a childVisitor for eachChild
func (f *copyFlow) stmt(s Stmt) { f.stmt1(s) }
