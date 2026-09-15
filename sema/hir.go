package sema

import (
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
	"github.com/LaH-DeV/veles/types"
)

// The HIR is the checker's output: a fully typed and resolved tree in which
// every implicit operation of the surface language has been made explicit —
// `Some` wrapping, smart-cast narrowing, `try` propagation, pattern-match
// lowering, default arguments, generic instantiation. Code generation is a
// direct walk over it.

// Program is everything the code generator needs.
type Program struct {
	Funcs   []*Func   // every concrete function, including methods and instantiations
	Globals []*Global // module-level val/var/const in initialisation order
	Structs []*types.Struct
	Sealeds []*types.Sealed
	Main    *Func
	Release bool
	// ResultType instantiates the prelude Result<T, E> for the backend.
	ResultType func(ok, err types.Type) types.Type
}

// Func is a concrete (monomorphic) function.
type Func struct {
	Name     string // mangled, unique
	Display  string // for diagnostics and panics
	Sig      *types.Func
	Params   []*Var // excludes the receiver
	Receiver *Var   // `self` for methods
	Mut      bool   // `mut fun` — receiver passed by pointer
	Extern   bool   // C ABI, no body
	Body     *Block
	Span     source.Span

	// addrTaken variables are heap-allocated by codegen.
	Locals []*Var

	// checker-private state
	tmpl           *FuncTemplate
	subst          map[*types.TypeParam]types.Type
	inferredErrors []types.Type
	inferring      bool
	checked        bool
}

type Var struct {
	Name      string
	Type      types.Type
	Mutable   bool
	AddrTaken bool
	IsGlobal  bool
	Global    *Global
	ID        int
	Span      source.Span
	// narrowed holds the flow-sensitive type while checking; not used by codegen.
	narrowed types.Type
	captured bool
}

type Global struct {
	Name    string // mangled
	Display string
	Type    types.Type
	Mutable bool
	Init    Expr
	Span    source.Span
}

// ---------------------------------------------------------------------------
// Statements

type Stmt interface{ hirStmt() }

type Block struct {
	Stmts []Stmt
	// Value is the trailing expression when the block is used as a value.
	Value Expr
	Type  types.Type // type of Value, or unit
}

type VarDecl struct {
	Var  *Var
	Init Expr // may be nil for `var x: T`
}

type Assign struct {
	Target Expr // an lvalue expression
	Value  Expr
}

type ExprStmt struct {
	X Expr
}

type Return struct {
	Value Expr // nil for unit
}

type Loop struct {
	Label string
	Cond  Expr // nil: infinite
	Body  *Block
	// Post runs at the end of every iteration and on `continue`.
	Post []Stmt
	ID   int
}

type Break struct {
	Loop *Loop
}

type Continue struct {
	Loop *Loop
}

func (*Block) hirStmt()    {}
func (*VarDecl) hirStmt()  {}
func (*Assign) hirStmt()   {}
func (*ExprStmt) hirStmt() {}
func (*Return) hirStmt()   {}
func (*Loop) hirStmt()     {}
func (*Break) hirStmt()    {}
func (*Continue) hirStmt() {}

// ---------------------------------------------------------------------------
// Expressions

type Expr interface {
	Type() types.Type
}

type exprBase struct{ T types.Type }

func (e exprBase) Type() types.Type { return e.T }

type IntConst struct {
	exprBase
	Value uint64
	Neg   bool
}

type FloatConst struct {
	exprBase
	Value float64
}

type BoolConst struct {
	exprBase
	Value bool
}

type StringConst struct {
	exprBase
	Value string
}

type UnitConst struct{ exprBase }

// NullConst is the `None` of a nullable type.
type NullConst struct{ exprBase }

type VarRef struct {
	exprBase
	Var *Var
}

// FuncRef is a function used as a value (not yet supported by codegen).
type FuncRef struct {
	exprBase
	Fn *Func
}

// Call invokes a concrete function. For methods the receiver is Args[0],
// passed by pointer when the method is `mut`.
type Call struct {
	exprBase
	Fn   *Func
	Args []Expr
}

type BinOp int

const (
	OpAdd BinOp = iota
	OpSub
	OpMul
	OpDiv
	OpRem
	OpWrapAdd
	OpWrapSub
	OpWrapMul
	OpEq
	OpNe
	OpLt
	OpLe
	OpGt
	OpGe
	OpAnd // short-circuit
	OpOr  // short-circuit
)

func BinOpFromToken(k lexer.TokenKind) BinOp {
	switch k {
	case lexer.Plus, lexer.PlusEq:
		return OpAdd
	case lexer.Minus, lexer.MinusEq:
		return OpSub
	case lexer.Star, lexer.StarEq:
		return OpMul
	case lexer.Slash, lexer.SlashEq:
		return OpDiv
	case lexer.Percent, lexer.PercentEq:
		return OpRem
	case lexer.WrapPlus:
		return OpWrapAdd
	case lexer.WrapMinus:
		return OpWrapSub
	case lexer.WrapStar:
		return OpWrapMul
	case lexer.Eq:
		return OpEq
	case lexer.NotEq:
		return OpNe
	case lexer.Lt:
		return OpLt
	case lexer.LtEq:
		return OpLe
	case lexer.Gt:
		return OpGt
	case lexer.GtEq:
		return OpGe
	case lexer.AndAnd:
		return OpAnd
	case lexer.OrOr:
		return OpOr
	}
	return -1
}

func (op BinOp) String() string {
	return [...]string{"+", "-", "*", "/", "%", "+%", "-%", "*%", "==", "!=", "<", "<=", ">", ">=", "&&", "||"}[op]
}

// Binary is an arithmetic, comparison or logical operation on operands of
// the same primitive type (strings included for == != < etc. and +).
type Binary struct {
	exprBase
	Op   BinOp
	L, R Expr
	Span source.Span
}

type UnOp int

const (
	OpNeg UnOp = iota
	OpNot
)

type Unary struct {
	exprBase
	Op   UnOp
	X    Expr
	Span source.Span
}

// Cast converts between numeric types.
type Cast struct {
	exprBase
	X Expr
}

// ToString converts a primitive or composite value to its string form for
// interpolation.
type ToString struct {
	exprBase
	X Expr
}

// StringConcat joins parts (all strings).
type StringConcat struct {
	exprBase
	Parts []Expr
}

// FieldGet reads a struct field. X may be an lvalue or a value.
type FieldGet struct {
	exprBase
	X     Expr
	Index int
	Name  string
}

// TupleGet reads a tuple element.
type TupleGet struct {
	exprBase
	X     Expr
	Index int
}

// StructLit builds a struct value from all fields in declaration order.
type StructLit struct {
	exprBase
	Struct *types.Struct
	Fields []Expr
}

type TupleLit struct {
	exprBase
	Elems []Expr
}

// AddrOf takes the address of an lvalue (`&x`); the variable is heap
// allocated (D10 escape analysis is approximated by "address ever taken").
type AddrOf struct {
	exprBase
	X Expr
}

// Deref reads through a pointer.
type Deref struct {
	exprBase
	X Expr
}

// SomeWrap lifts T to T? (implicit promotion, D5).
type SomeWrap struct {
	exprBase
	X Expr
}

// IsNull tests a nullable for None.
type IsNull struct {
	exprBase
	X Expr
}

// Unwrap extracts T from a T? already known to be Some (after a smart cast
// or pattern test).
type Unwrap struct {
	exprBase
	X Expr
}

// MakeVariant wraps a variant struct value into its sealed type.
type MakeVariant struct {
	exprBase
	Sealed  *types.Sealed
	Variant *types.Struct
	Value   Expr
}

// VariantTest checks the tag of a sealed value.
type VariantTest struct {
	exprBase
	X       Expr
	Variant *types.Struct
}

// VariantCast extracts the variant payload from a sealed value known to
// hold that variant.
type VariantCast struct {
	exprBase
	X       Expr
	Variant *types.Struct
}

// If is the conditional expression; when used as a statement Type is unit.
type If struct {
	exprBase
	Cond Expr
	Then *Block
	Else *Block // may be nil
}

// BlockExpr evaluates a block for its value.
type BlockExpr struct {
	exprBase
	Block *Block
}

// MakeResult builds a Result value: Ok(value) or Err(error).
type MakeResult struct {
	exprBase
	IsErr bool
	Value Expr
}

// ResultIsErr tests a Result.
type ResultIsErr struct {
	exprBase
	X Expr
}

// ResultValue extracts the Ok payload or the Err payload.
type ResultValue struct {
	exprBase
	X   Expr
	Err bool
}

// ErrorConvert re-tags an error value into a wider union (D45 width
// subtyping). From is the source error type, the target is the expr type.
type ErrorConvert struct {
	exprBase
	X    Expr
	From types.Type
}

// UnionTest checks whether an error union holds member Member.
type UnionTest struct {
	exprBase
	X      Expr
	Member types.Type
}

// UnionCast extracts a known member from an error union.
type UnionCast struct {
	exprBase
	X      Expr
	Member types.Type
}

// ListLit constructs a List/MutableList from elements.
type ListLit struct {
	exprBase
	Elems []Expr
}

// Builtin is a call into a runtime-provided operation on builtin types.
type Builtin struct {
	exprBase
	Op   string // e.g. "list.len", "list.get", "list.push", "string.len"
	Args []Expr
	Span source.Span
}

// RangeLit constructs a Range value.
type RangeLit struct {
	exprBase
	Lo, Hi    Expr
	Inclusive bool
}

// Panic aborts with a message; typed Never.
type Panic struct {
	exprBase
	Message string
}

// Elvis: L ?: R where L is nullable.
type Elvis struct {
	exprBase
	L, R Expr
}

// Let evaluates Init into a temporary and then Body, which may refer to it.
// Used for lowering `?.`, `?:` and `when`.
type Let struct {
	exprBase
	Var  *Var
	Init Expr
	Body Expr
}

// Match is the lowered form of `when` (D13). Arms are tried in order: if
// Test holds (nil means always), Binds run, then Guard is checked (nil
// means none); the first arm whose guard holds supplies the value. When no
// arm matches and the match is not Exhaustive, the result is unit (statement
// form) — exhaustive matches never fall through.
type Match struct {
	exprBase
	Subject    *Var
	Init       Expr
	Arms       []*MatchArm
	Exhaustive bool
	Span       source.Span
}

type MatchArm struct {
	Test  Expr
	Binds []Stmt
	Guard Expr
	Body  *Block
}

// Try propagates an error: X is a Result value; on Err the enclosing
// function returns Err(convert(error)), on Ok the payload is the value.
type Try struct {
	exprBase
	X    Expr
	From types.Type // callee error type
	To   types.Type // enclosing function's error type
}

// Throw returns Err(Value) from the enclosing throwing function; typed Never.
type Throw struct {
	exprBase
	Value Expr
	From  types.Type
	To    types.Type
}
