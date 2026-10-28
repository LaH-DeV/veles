package sema

import "github.com/LaH-DeV/veles/types"

// D218: a counted loop whose body calls nothing and does not suspend looks
// for attention (D66, D145) once per chunk of rounds, not on every round.
// The look is a volatile load, and on every round it keeps clang from
// unrolling, reassociating, vectorizing or folding a tight loop — 3 to 9
// times C's time, measured (plan F21). A chunk is at most 1024 rounds of a
// body that calls nothing, so what the look is for — a cancellation, a
// yield, a collection's stop — waits about a microsecond more at worst.
//
// Codegen runs the chunk as an inner loop whose one exit is the loop's own
// condition and `idx < limit` (LLVM folds the two into one bound it can
// count), and looks once after it. Every chunk runs its first round
// unconditionally — the loop's condition was just true — so a loop ends
// even where `limit` saturates at i64's top.

// LoopChunk is how a loop is cut into chunks; nil on a Loop that looks on
// every round.
type LoopChunk struct {
	Idx      *Var  // the induction variable: i64, stepped by one in Post only
	Limit    *Var  // where this chunk ends (exclusive), set by SetLimit
	SetLimit Stmt  // Limit = min(Idx + Rounds, i64 max), at each chunk's start
	InChunk  Expr  // Idx < Limit
	Rounds   int64 // the rounds of a chunk
}

const (
	maxChunkRounds = 1024
	minChunkRounds = 16
	// a chunk is about this many nodes of body: a larger body gets fewer
	// rounds, so the work between two looks stays about the same
	chunkWork = 16 * maxChunkRounds
)

// chunkLoop gives lp, a counted loop over idx, its chunks when its body,
// condition and step call nothing, suspend nowhere and hold no loop of
// their own; the declaration of the chunk's limit goes to pre.
func (f *fnCtx) chunkLoop(lp *Loop, idx *Var, pre *[]Stmt) {
	if !types.Identical(idx.Type, types.TI64) {
		return // i64 covers lists and the common ranges; a narrower range is rarely the hot loop
	}
	size, ok := chunkableSize(lp)
	if !ok {
		return
	}
	rounds := int64(chunkWork / max(size, 1))
	rounds = min(max(rounds, minChunkRounds), maxChunkRounds)
	limit := f.newTemp(types.TI64)
	*pre = append(*pre, &VarDecl{Var: limit, Init: &IntConst{exprBase{types.TI64}, 0, false}})
	ref := func(v *Var) Expr { return &VarRef{exprBase{v.Type}, v} }
	num := func(n int64) Expr { return &IntConst{exprBase{types.TI64}, uint64(n), false} } // n >= 0
	const top = int64(^uint64(0) >> 1)
	// idx > top - rounds ? top : idx + rounds — never overflowing
	near := &Binary{exprBase{types.TBool}, OpGt, ref(idx), num(top - rounds), lp.Pos}
	sum := &Binary{exprBase{types.TI64}, OpAdd, ref(idx), num(rounds), lp.Pos}
	end := &If{exprBase{types.TI64}, near, &Block{Value: num(top), Type: types.TI64}, &Block{Value: sum, Type: types.TI64}}
	lp.Chunk = &LoopChunk{
		Idx:      idx,
		Limit:    limit,
		SetLimit: &Assign{Target: ref(limit), Value: end},
		InChunk:  &Binary{exprBase{types.TBool}, OpLt, ref(idx), ref(limit), lp.Pos},
		Rounds:   rounds,
	}
}

// chunkableSize counts the nodes of a loop's condition, body and step, and
// says whether the loop may be cut into chunks: nothing in it calls a
// function (a call can run for any length of time between two looks),
// suspends, starts or waits for a task, opens a cleanup, or loops itself
// (an inner loop looks on its own rounds; cutting the outer one gains
// nothing).
func chunkableSize(lp *Loop) (int, bool) {
	size, ok := 0, true
	visit := func(n any) {
		size++
		switch n := n.(type) {
		case *Call, *CallIndirect, *CallVirtual, *Loop, *With, *ScopeBlock, *Launch, *AwaitTask, *Race:
			ok = false
		case *Builtin:
			if (susp{}).expr(n) {
				ok = false
			}
		}
	}
	walkExpr(lp.Cond, visit)
	walkBlock(lp.Body, visit)
	for _, s := range lp.Post {
		walkStmt(s, visit)
	}
	return size, ok
}
