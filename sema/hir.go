package sema

import (
	"github.com/LaH-DeV/veles/ast"
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
	// MainReport renders an error escaping `main() throws` via Error.message()
	// (D4); nil when main does not throw.
	MainReport *Func
	Release    bool
	// Root is the package root: panic locations are printed relative to it (D64).
	Root string
	// PanicType is the prelude Panic struct (D52).
	PanicType types.Type
	// Tests are the tests (D78); in test mode the entry point runs them.
	Tests    []*Func
	TestMode bool
	// TestTimeoutMs bounds each test's run (0: unbounded) and TestsFiltered
	// counts the tests `veles test --filter` left out; the driver sets both.
	TestTimeoutMs int64
	TestsFiltered int
	// TestJobs is how many tests run at once (D80): 0 is one per worker
	// thread, 1 runs them one at a time.
	TestJobs int64
	// ResultType instantiates the prelude Result<T, E> for the backend.
	ResultType func(ok, err types.Type) types.Type
	// Custom maps types.Key of a struct or sealed type to the prelude-trait
	// methods that replace its structural equality, hash, ordering and text.
	Custom map[string]*CustomOps
}

// CustomOps are the instantiated impl methods of the prelude's Equatable,
// Hashable, Comparable and Display for one concrete type; a nil entry means
// the structural behaviour applies. Each takes the receiver by value.
type CustomOps struct {
	Equals   *Func // (self, other) bool
	Hash     *Func // (self) i64
	Compare  *Func // (self, other) i64
	ToString *Func // (self) string
}

// Func is a concrete (monomorphic) function.
type Func struct {
	Name    string // mangled, unique
	Display string // for diagnostics and panics
	// Suite is a test's path of suites (D78), for the runner's grouping.
	Suite    []string
	Sig      *types.Func
	Params   []*Var // excludes the receiver
	Receiver *Var   // `this` for methods, always a pointer to the receiver's place (D22 v0.30)
	Extern   bool   // C ABI, no body
	// Foreign: an extern outside std — code the runtime knows nothing
	// about. Codegen runs each call in a safe region (D66), so a C call
	// that blocks does not hold up a collection on other threads.
	Foreign bool
	// ExportC is the C symbol of an `extern "C" fun` (D69): codegen emits a
	// wrapper under this name that C calls, around the Veles body.
	ExportC string
	Inline  int // 1 @inline, -1 @noinline
	// Closure functions take an environment pointer first; CapVars are the
	// inner variables standing for captured outer ones (index = env slot).
	IsClosure bool
	CapVars   []*Var
	Body      *Block
	Span      source.Span

	// addrTaken variables are heap-allocated by codegen.
	Locals []*Var

	// checker-private state
	tmpl           *FuncTemplate
	subst          map[*types.TypeParam]types.Type
	inferredErrors []types.Type
	inferring      bool
	checked        bool
	suspends       bool // directly contains a suspension point
	raised         bool // an error reached recordError: something in the body can throw

	// Suspends is the inferred effect (D2): set by the suspension pass.
	Suspends bool
	// SelfEscapes: the method may keep a pointer to its receiver beyond
	// the call (a closure capturing `this`, `&this`, or a callee that does);
	// WritesSelf: it may assign the receiver's fields, directly or through a
	// callee. Both are set by the receiver pass (receivers.go, D22 v0.30).
	SelfEscapes bool
	WritesSelf  bool
	// FieldsUsed are the receiver's fields the method reads or writes,
	// by index, directly or through the methods it calls on self;
	// AllFields when it uses the receiver as a whole (copies it, hands out
	// a pointer to it, captures it). A method called on a struct under
	// construction (D28 derived defaults) must stay within the bound fields.
	FieldsUsed map[int]bool
	AllFields  bool
}

type Var struct {
	Name      string
	Type      types.Type
	Mutable   bool
	AddrTaken bool
	Captured  bool // inside a closure: read through the environment
	IsSelf    bool // a method's receiver (or its stand-in inside a lambda): a pointer read as the value
	IsParam   bool // a function or lambda parameter

	InitText string // a local binding's initializer as written, for the hover
	ErrPoly  bool   // a parameter declared `fun(..) throws E` with E a type parameter of the enclosing function
	CapIndex int
	Outer    *Var // the enclosing function's variable this stands for
	IsGlobal bool
	Global   *Global
	ID       int
	Span     source.Span
	// used records a read of the variable; checkUse marks bindings the
	// checker reports when they are never read (see reportUnused).
	used     bool
	checkUse bool
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

	hasBreak    bool
	hasContinue bool
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

// Zero is the all-zero value of a type: the placeholder for the fields
// not yet bound in a partially built struct (D28 derived defaults). It is
// never read — the receiver pass proves a method called on the partial
// value touches only the fields already bound.
type Zero struct{ exprBase }

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
// passed by pointer (D22 v0.30); Recv records where that pointer points,
// for the receiver pass (receivers.go).
type Call struct {
	exprBase
	Fn   *Func
	Args []Expr
	// Span is where the call is written; zero for a call the compiler made
	// (a comparison, a derived body). A debug build records it for the
	// panic call chain (D81).
	Span source.Span

	Recv     RecvKind
	RecvRoot *Var        // RecvPlace: the local or global the place is rooted in, if any
	RecvSpan source.Span // the method name at the call
	RecvType types.Type  // the receiver's type
	RecvExpr ast.Expr    // the receiver as written, for fixes
	// InitMissing: a call on `this` inside an `init` block while these
	// owned fields (by index) are not yet assigned; the receiver pass
	// checks the method does not read them (D28).
	InitMissing []int
}

// RecvKind says what a method call's receiver pointer points at.
type RecvKind int

const (
	RecvNone   RecvKind = iota
	RecvPlace           // a variable, field or dereference: the method changes it in place
	RecvTemp            // a fresh copy of a temporary value
	RecvHandle          // a copy of a collection handle (D25): the same elements
)

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
	OpBitAnd
	OpBitOr
	OpBitXor
	OpShl
	OpShr
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
	case lexer.Amp:
		return OpBitAnd
	case lexer.Pipe:
		return OpBitOr
	case lexer.Caret:
		return OpBitXor
	case lexer.Shl:
		return OpShl
	case lexer.Shr:
		return OpShr
	}
	return -1
}

func (op BinOp) String() string {
	return [...]string{"+", "-", "*", "/", "%", "+%", "-%", "*%", "==", "!=", "<", "<=", ">", ">=", "&&", "||", "&", "|", "^", "<<", ">>"}[op]
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
	OpBitNot // ~x, integers
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

// Closure creates a function value from a lambda (D32). Captures are the
// enclosing function's variables shared by reference: they are heap cells
// (AddrTaken) whose addresses are stored in the environment.
type Closure struct {
	exprBase
	Fn       *Func
	Captures []*Var
}

// CallIndirect calls a function value ({ fn, env } pair).
type CallIndirect struct {
	exprBase
	Fn   Expr
	Args []Expr
}

// MapLit constructs a Map/MutableMap from key/value pairs.
type MapLit struct {
	exprBase
	Entries [][2]Expr
}

// With binds a Closeable resource for the block and closes it on every
// exit path — fallthrough, return, break and continue (D43).
type With struct {
	Var   *Var
	Init  Expr
	Close Expr // the close() call on Var
	Body  *Block
}

func (*With) hirStmt() {}

// Box coerces a value to a trait object (D9): the value is copied to the
// heap and paired with the vtable of its impl. Methods lists the concrete
// functions in the trait's method order.
type Box struct {
	exprBase
	X       Expr
	Trait   *types.Trait
	Methods []*Func
}

// CallVirtual invokes a trait method through a trait object's vtable.
// Index is the slot in the composed table (supertraits first, D58); Sig is
// the declaration in the trait that owns the method.
type CallVirtual struct {
	exprBase
	Obj   Expr
	Trait *types.Trait
	Index int
	Sig   *types.Func
	Args  []Expr
}

// ---------------------------------------------------------------------------
// concurrency (D2/D3/D16/D34/D36/D38)

// Launch starts Call as a child task of the lexically enclosing scope.
type Launch struct {
	exprBase
	Call  *Call
	Scope *ScopeBlock
	Index int
}

// AwaitTask waits for a task handle and yields its result.
type AwaitTask struct {
	exprBase
	X Expr
}

// ScopeBlock is `scope { }` (statement, fail-fast) or `gather { }`
// (expression yielding a tuple of Results). Elems are the per-launch
// Result types of a gather.
type ScopeBlock struct {
	exprBase
	Body     *Block
	Launches []*Launch
	Gather   bool
	Elems    []types.Type
	ErrTo    types.Type // enclosing function's error type (fail-fast rethrow)
	Span     source.Span
}

func (*ScopeBlock) hirStmt() {}

type RaceArmKind int

const (
	RaceRecv RaceArmKind = iota
	RaceSleep
	RaceTask
)

type RaceArm struct {
	Kind   RaceArmKind
	Source Expr // channel, milliseconds, or task
	Var    *Var // bound value for recv/task arms (nullable for recv)
	Body   *Block
}

type Race struct {
	exprBase
	Arms []*RaceArm
}
