package sema

// walkBlock visits every expression and statement in b, in evaluation
// order, calling visit on each (statements as `any`, expressions as Expr)
// before descending. Lambdas are separate functions (their bodies are not
// entered here: the Closure node is visited, its Fn is in Program.Funcs).
// A block's trailing value is announced by a blockValue first, for a
// visitor that follows where code is written (AllocSites).
func walkBlock(b *Block, visit func(node any)) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		walkStmt(s, visit)
	}
	if b.Value != nil {
		visit(blockValue{b})
		walkExpr(b.Value, visit)
	}
}

// blockValue: what follows is the value of this block, written at its
// ValuePos.
type blockValue struct{ block *Block }

func walkStmt(s Stmt, visit func(node any)) {
	if s == nil {
		return
	}
	visit(s)
	switch s := s.(type) {
	case *Block:
		walkBlock(s, visit)
	case *VarDecl:
		walkExpr(s.Init, visit)
	case *Assign:
		walkExpr(s.Target, visit)
		walkExpr(s.Value, visit)
	case *ExprStmt:
		walkExpr(s.X, visit)
	case *Return:
		walkExpr(s.Value, visit)
	case *Loop:
		walkExpr(s.Cond, visit)
		walkBlock(s.Body, visit)
		for _, p := range s.Post {
			walkStmt(p, visit)
		}
	case *With:
		walkExpr(s.Init, visit)
		walkExpr(s.Close, visit)
		walkBlock(s.Body, visit)
	case *ScopeBlock:
		walkExpr(s.On, visit)
		walkExpr(s.Workers, visit)
		if s.Held != nil {
			walkExpr(s.Held.Init, visit) // the value a `with` receives (D111)
		}
		walkBlock(s.Body, visit)
		if s.Held != nil {
			walkExpr(s.Held.Close, visit)
		}
	}
}

func walkExprs(es []Expr, visit func(node any)) {
	for _, e := range es {
		walkExpr(e, visit)
	}
}

func walkExpr(e Expr, visit func(node any)) {
	walker{visit}.expr(e)
}

// childVisitor is what eachChild hands an expression's children to.
type childVisitor interface {
	expr(Expr)
	block(*Block)
	stmt(Stmt)
}

// walker visits every node below an expression (walkExpr).
type walker struct{ visit func(node any) }

func (w walker) expr(e Expr) {
	if e == nil {
		return
	}
	w.visit(e)
	eachChild(e, w)
}

func (w walker) block(b *Block) { walkBlock(b, w.visit) }
func (w walker) stmt(s Stmt)    { walkStmt(s, w.visit) }

func eachExpr(es []Expr, v childVisitor) {
	for _, e := range es {
		v.expr(e)
	}
}

// eachChild hands e's direct children to v in evaluation order: its
// expressions, blocks and statements (a block or a statement whole, for v
// to enter). Lambdas' bodies are not among them (walkBlock).
func eachChild(e Expr, v childVisitor) {
	switch e := e.(type) {
	case *Call:
		eachExpr(e.Args, v)
	case *CallIndirect:
		v.expr(e.Fn)
		eachExpr(e.Args, v)
	case *CallVirtual:
		v.expr(e.Obj)
		eachExpr(e.Args, v)
	case *Builtin:
		eachExpr(e.Args, v)
	case *Binary:
		v.expr(e.L)
		v.expr(e.R)
	case *Unary:
		v.expr(e.X)
	case *Cast:
		v.expr(e.X)
	case *ToString:
		v.expr(e.X)
	case *StringConcat:
		eachExpr(e.Parts, v)
	case *FieldGet:
		v.expr(e.X)
	case *TupleGet:
		v.expr(e.X)
	case *StructLit:
		eachExpr(e.Fields, v)
	case *TupleLit:
		eachExpr(e.Elems, v)
	case *AddrOf:
		v.expr(e.X)
	case *Deref:
		v.expr(e.X)
	case *SomeWrap:
		v.expr(e.X)
	case *IsNull:
		v.expr(e.X)
	case *Unwrap:
		v.expr(e.X)
	case *MakeVariant:
		v.expr(e.Value)
	case *VariantTest:
		v.expr(e.X)
	case *VariantCast:
		v.expr(e.X)
	case *TypeTest:
		v.expr(e.X)
	case *Downcast:
		v.expr(e.X)
	case *TraitTest:
		v.expr(e.X)
	case *TraitCast:
		v.expr(e.X)
	case *If:
		v.expr(e.Cond)
		v.block(e.Then)
		v.block(e.Else)
	case *BlockExpr:
		v.block(e.Block)
	case *MakeResult:
		v.expr(e.Value)
	case *ResultIsErr:
		v.expr(e.X)
	case *ResultValue:
		v.expr(e.X)
	case *ErrorConvert:
		v.expr(e.X)
	case *UnionTest:
		v.expr(e.X)
	case *UnionCast:
		v.expr(e.X)
	case *ListLit:
		eachExpr(e.Elems, v)
	case *MapLit:
		for _, en := range e.Entries {
			v.expr(en[0])
			v.expr(en[1])
		}
	case *RangeLit:
		v.expr(e.Lo)
		v.expr(e.Hi)
	case *Elvis:
		v.expr(e.L)
		v.expr(e.R)
	case *Let:
		v.expr(e.Init)
		v.expr(e.Body)
	case *Match:
		v.expr(e.Init)
		for _, arm := range e.Arms {
			v.expr(arm.Test)
			for _, b := range arm.Binds {
				v.stmt(b)
			}
			v.expr(arm.Guard)
			v.block(arm.Body)
		}
	case *Try:
		v.expr(e.X)
	case *Throw:
		v.expr(e.Value)
	case *Box:
		v.expr(e.X)
	case *Share:
		v.expr(e.X)
	case *LazyRead:
		v.expr(e.Build)
	case *Launch:
		v.expr(e.Call)
	case *AwaitTask:
		v.expr(e.X)
	case *ScopeBlock:
		v.expr(e.On)
		v.expr(e.Workers)
		if e.Held != nil {
			v.expr(e.Held.Init) // the value a `with` receives (D111)
		}
		v.block(e.Body)
		if e.Held != nil {
			v.expr(e.Held.Close)
		}
	case *RaceFirst:
		v.expr(e.Tasks)
	case *Race:
		for _, arm := range e.Arms {
			v.expr(arm.Source)
			v.expr(arm.Value)
			v.block(arm.Body)
		}
	}
}
