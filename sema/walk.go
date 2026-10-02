package sema

// walkBlock visits every expression and statement in b, in evaluation
// order, calling visit on each (statements as `any`, expressions as Expr)
// before descending. Lambdas are separate functions (their bodies are not
// entered here: the Closure node is visited, its Fn is in Program.Funcs).
func walkBlock(b *Block, visit func(node any)) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		walkStmt(s, visit)
	}
	if b.Value != nil {
		walkExpr(b.Value, visit)
	}
}

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
		walkBlock(s.Body, visit)
	}
}

func walkExprs(es []Expr, visit func(node any)) {
	for _, e := range es {
		walkExpr(e, visit)
	}
}

func walkExpr(e Expr, visit func(node any)) {
	if e == nil {
		return
	}
	visit(e)
	switch e := e.(type) {
	case *Call:
		walkExprs(e.Args, visit)
	case *CallIndirect:
		walkExpr(e.Fn, visit)
		walkExprs(e.Args, visit)
	case *CallVirtual:
		walkExpr(e.Obj, visit)
		walkExprs(e.Args, visit)
	case *Builtin:
		walkExprs(e.Args, visit)
	case *Binary:
		walkExpr(e.L, visit)
		walkExpr(e.R, visit)
	case *Unary:
		walkExpr(e.X, visit)
	case *Cast:
		walkExpr(e.X, visit)
	case *ToString:
		walkExpr(e.X, visit)
	case *StringConcat:
		walkExprs(e.Parts, visit)
	case *FieldGet:
		walkExpr(e.X, visit)
	case *TupleGet:
		walkExpr(e.X, visit)
	case *StructLit:
		walkExprs(e.Fields, visit)
	case *TupleLit:
		walkExprs(e.Elems, visit)
	case *AddrOf:
		walkExpr(e.X, visit)
	case *Deref:
		walkExpr(e.X, visit)
	case *SomeWrap:
		walkExpr(e.X, visit)
	case *IsNull:
		walkExpr(e.X, visit)
	case *Unwrap:
		walkExpr(e.X, visit)
	case *MakeVariant:
		walkExpr(e.Value, visit)
	case *VariantTest:
		walkExpr(e.X, visit)
	case *VariantCast:
		walkExpr(e.X, visit)
	case *TypeTest:
		walkExpr(e.X, visit)
	case *Downcast:
		walkExpr(e.X, visit)
	case *TraitTest:
		walkExpr(e.X, visit)
	case *TraitCast:
		walkExpr(e.X, visit)
	case *If:
		walkExpr(e.Cond, visit)
		walkBlock(e.Then, visit)
		walkBlock(e.Else, visit)
	case *BlockExpr:
		walkBlock(e.Block, visit)
	case *MakeResult:
		walkExpr(e.Value, visit)
	case *ResultIsErr:
		walkExpr(e.X, visit)
	case *ResultValue:
		walkExpr(e.X, visit)
	case *ErrorConvert:
		walkExpr(e.X, visit)
	case *UnionTest:
		walkExpr(e.X, visit)
	case *UnionCast:
		walkExpr(e.X, visit)
	case *ListLit:
		walkExprs(e.Elems, visit)
	case *MapLit:
		for _, en := range e.Entries {
			walkExpr(en[0], visit)
			walkExpr(en[1], visit)
		}
	case *RangeLit:
		walkExpr(e.Lo, visit)
		walkExpr(e.Hi, visit)
	case *Elvis:
		walkExpr(e.L, visit)
		walkExpr(e.R, visit)
	case *Let:
		walkExpr(e.Init, visit)
		walkExpr(e.Body, visit)
	case *Match:
		walkExpr(e.Init, visit)
		for _, arm := range e.Arms {
			walkExpr(arm.Test, visit)
			for _, b := range arm.Binds {
				walkStmt(b, visit)
			}
			walkExpr(arm.Guard, visit)
			walkBlock(arm.Body, visit)
		}
	case *Try:
		walkExpr(e.X, visit)
	case *Throw:
		walkExpr(e.Value, visit)
	case *Box:
		walkExpr(e.X, visit)
	case *Launch:
		walkExpr(e.Call, visit)
	case *AwaitTask:
		walkExpr(e.X, visit)
	case *ScopeBlock:
		walkBlock(e.Body, visit)
	case *Race:
		for _, arm := range e.Arms {
			walkExpr(arm.Source, visit)
			walkExpr(arm.Value, visit)
			walkBlock(arm.Body, visit)
		}
	}
}
