// Package ast defines the syntax tree for Veles source files.
//
// The tree covers the whole surface of the language specification, whether
// or not later phases implement a construct yet, so that the parser is the
// single place that knows the grammar.
package ast

import (
	"github.com/LaH-DeV/veles/lexer"
	"github.com/LaH-DeV/veles/source"
)

type Node interface {
	Span() source.Span
}

type Ident struct {
	Name string
	Pos  source.Span
}

func (i Ident) Span() source.Span { return i.Pos }
func (i Ident) String() string    { return i.Name }

// File is one parsed source file. A module is every File in a directory.
type File struct {
	Source *source.File
	Decls  []Decl
	// Doc is the file's module documentation: a doc comment at the very top,
	// set off from the first declaration by a blank line.
	Doc string
}

func (f *File) Span() source.Span {
	return source.Span{File: f.Source, Start: 0, End: len(f.Source.Content)}
}

// ---------------------------------------------------------------------------
// Types

type Type interface {
	Node
	typeNode()
}

// NamedType is a possibly qualified, possibly generic type name:
// `i32`, `List<T>`, `Shape.Circle`, `io.Reader`.
type NamedType struct {
	Path []Ident
	Args []Type
	Pos  source.Span
}

// NullableType is `T?`.
type NullableType struct {
	Elem Type
	Pos  source.Span
}

// PointerType is `*T` (GC-managed) or `*raw T` (unmanaged, D50).
type PointerType struct {
	Elem Type
	Raw  bool
	Pos  source.Span
}

// ResolvedType is a type the compiler already knows, written into a
// synthesized declaration (a derived impl, D58) that no source spells:
// the checker hands `T` back as it is. It never comes from the parser.
type ResolvedType struct {
	T   any // a types.Type; `any` keeps ast free of the types package
	Pos source.Span
}

// TupleType is `(A, B)`. The empty tuple `()` is the unit type.
type TupleType struct {
	Elems []Type
	Pos   source.Span
}

// FunType is `fun(A, B): R suspends throws E` (D40), or `sendable fun(...)`:
// a function value that may cross a task boundary (D35).
type FunType struct {
	Params   []Type
	Ret      Type // nil means unit
	Effects  Effects
	Sendable bool
	Pos      source.Span
}

// SelfType is the `Self` keyword in a trait or impl.
type SelfType struct {
	Pos source.Span
}

// AssocType is an associated-type projection: `Self::Item`, `I::Item`.
type AssocType struct {
	Base Type
	Name Ident
	Pos  source.Span
}

// ErrorUnionType is `A | B`; it may appear only after `throws` (D45).
type ErrorUnionType struct {
	Members []Type
	Pos     source.Span
}

func (t *NamedType) Span() source.Span      { return t.Pos }
func (t *NullableType) Span() source.Span   { return t.Pos }
func (t *PointerType) Span() source.Span    { return t.Pos }
func (t *TupleType) Span() source.Span      { return t.Pos }
func (t *FunType) Span() source.Span        { return t.Pos }
func (t *SelfType) Span() source.Span       { return t.Pos }
func (t *AssocType) Span() source.Span      { return t.Pos }
func (t *ErrorUnionType) Span() source.Span { return t.Pos }
func (t *ResolvedType) Span() source.Span   { return t.Pos }

func (*NamedType) typeNode()      {}
func (*NullableType) typeNode()   {}
func (*PointerType) typeNode()    {}
func (*TupleType) typeNode()      {}
func (*FunType) typeNode()        {}
func (*SelfType) typeNode()       {}
func (*AssocType) typeNode()      {}
func (*ErrorUnionType) typeNode() {}
func (*ResolvedType) typeNode()   {}

// Effects are the declared effects of a signature: `suspends`, `throws`,
// `throws E`. A `throws` with a nil Error type is inferred (D4/D45).
type Effects struct {
	Suspends     bool
	SuspendsSpan source.Span
	Throws       bool
	ThrowsSpan   source.Span
	Error        Type
}

// ---------------------------------------------------------------------------
// Declarations

type Decl interface {
	Node
	declNode()
}

// Attribute is `@name` or `@name(args)` on the line before a declaration.
type Attribute struct {
	Name Ident
	Args []Arg
	Pos  source.Span
}

// UseDecl is `use a, b.c as d`: one or more module imports in one
// statement (M6). Modules are the only thing imported; their members are
// always qualified (`geometry.Point`), and `as` renames the module.
type UseDecl struct {
	Specs []*UseSpec
	Pos   source.Span
}

// UseSpec is one import: `a.b` or `a.b as c`.
type UseSpec struct {
	Path  []Ident
	Alias *Ident
	Pos   source.Span
}

type TypeParam struct {
	Name   Ident
	Bounds []Type
}

type Param struct {
	Name     Ident
	Type     Type
	Default  Expr
	Variadic bool     // `name: T...` — the last parameter takes any number of arguments (a List<T> inside)
	Pattern  *Binding // lambda only: `((size, hash), files) => ...` destructures the argument (D37)
	Pos      source.Span
}

// FunDecl is a free function, an inherent method, a trait method
// (signature-only when Body and ExprBody are both nil), or an extern.
type FunDecl struct {
	Attrs    []*Attribute
	Doc      string // documentation comment, if any
	Pub      bool
	Internal bool // `internal`: the module level written out (M5); the default
	Private  bool // `private fun` — callable only inside the type's own declarations

	Static     bool // `static fun` — no receiver; called on the type (D23)
	Override   bool
	Unsafe     bool
	Extern     bool
	Name       Ident
	TypeParams []TypeParam
	Params     []Param
	Ret        Type // nil means unit
	Effects    Effects
	Body       *Block
	ExprBody   Expr // `fun f() = expr`
	Pos        source.Span
}

// Field is a struct field, optionally with a default.
type Field struct {
	Attrs     []*Attribute // `@key(...)`, `@skip`, `@required` (D58)
	Pub       bool
	Internal  bool // `internal`: the module level written out (M5); the default
	Private   bool // visible only inside the type's own declarations
	Var       bool // `var name: T` — assignable after construction (D22 v0.30)
	Val       bool // `val name: T` — the default, written out
	Protected bool // `protected var name: T` — assigned only by the type's own declarations (D22 v0.30)
	Doc       string
	Name      Ident
	Type      Type
	Default   Expr
	Pos       source.Span
}

// StructDecl is `struct Name<T> : SealedParent { fields; methods }`.
type StructDecl struct {
	Attrs      []*Attribute
	Doc        string
	Pub        bool
	Internal   bool      // `internal`: the module level written out (M5); the default
	Extern     bool      // `extern struct` — C layout
	Error      bool      // `error Name { }` — declared with an impl of Error (D4)
	ErrorImpl  *ImplDecl // the `impl Error for Name` an error declaration desugars to
	Name       Ident
	TypeParams []TypeParam
	Variant    Type // the sealed trait this struct is a variant of, or nil
	Fields     []*Field
	Methods    []*FunDecl
	Statics    []*ValDecl  // `static val name = expr`: constants in the type's namespace (`Type.name`)
	Impls      []*ImplDecl // `impl Trait { }` blocks written in the body (also in File.Decls)
	// Init is the `init { }` block: runs after every field is bound, may
	// assign the fields no caller supplied (D28 v0.30). Nil when absent.
	Init    *Block
	InitPos source.Span // the `init` keyword
	Pos     source.Span
}

type AssocTypeDecl struct {
	Name   Ident
	Bounds []Type
	Pos    source.Span
}

// TraitDecl is `trait Name { type Item; fun m(); }` or `sealed trait Name`.
type TraitDecl struct {
	Attrs      []*Attribute
	Doc        string
	Pub        bool
	Internal   bool // `internal`: the module level written out (M5); the default
	Sealed     bool
	Name       Ident
	TypeParams []TypeParam
	Supers     []Type
	AssocTypes []*AssocTypeDecl
	Methods    []*FunDecl
	Pos        source.Span
}

type AssocTypeBinding struct {
	Name Ident
	Type Type
}

// ImplDecl is `impl<T> Trait for Type { ... }`, or with Extend set
// `extend<T> Type { ... }`: inherent methods added to a type the package
// declares (D23; the built-in types belong to std). Trait is nil then.
type ImplDecl struct {
	Attrs      []*Attribute
	Extend     bool
	Inline     bool // written inside the target struct's body
	Braceless  bool // an empty impl written without `{ }` (D58): the body is derived
	Derived    bool // synthesized by the compiler (D58): a supertrait's, a variant's or an enum's impl; never from the parser
	TypeParams []TypeParam
	Trait      Type
	Target     Type
	AssocTypes []*AssocTypeBinding
	Methods    []*FunDecl
	Pos        source.Span
}

type BindKind int

const (
	BindVal BindKind = iota
	BindVar
	BindConst
)

func (k BindKind) String() string {
	switch k {
	case BindVar:
		return "var"
	case BindConst:
		return "const"
	}
	return "val"
}

// ValDecl is a module-level `val`, `var` or `const`.
type ValDecl struct {
	Attrs    []*Attribute
	Doc      string
	Pub      bool
	Internal bool // `internal`: the module level written out (M5); the default
	Kind     BindKind
	Name     Ident
	Type     Type
	Value    Expr
	Pos      source.Span
}

// ExternBlock is `extern "C" { fun ...; }`.
type ExternBlock struct {
	ABI  string
	Funs []*FunDecl
	Pos  source.Span
}

// ErrorAliasDecl is `error Name = A | B | C`: a named error set (D45),
// transparent wherever a union may appear.
type ErrorAliasDecl struct {
	Attrs    []*Attribute
	Doc      string
	Pub      bool
	Internal bool // `internal`: the module level written out (M5); the default
	Name     Ident
	Members  Type // an ErrorUnionType, or a single type
	Pos      source.Span
}

// TypeAliasDecl is `type Name<T> = Type`: another name for a type (D55).
// It never makes a new type; the newtype is a one-field struct.
type TypeAliasDecl struct {
	Attrs      []*Attribute
	Doc        string
	Pub        bool
	Internal   bool // `internal`: the module level written out (M5); the default
	Name       Ident
	TypeParams []TypeParam
	Type       Type
	Pos        source.Span
}

// EnumDecl is `enum Name : Base { A = 1, B, C }`: a closed set of named
// values of one integer type (D57). Members without a value continue from
// the previous one; the first defaults to 0.
type EnumDecl struct {
	Attrs    []*Attribute
	Doc      string
	Pub      bool
	Internal bool // `internal`: the module level written out (M5); the default
	Name     Ident
	Base     Type // the underlying integer type, nil for i64
	Members  []*EnumMember
	Pos      source.Span
}

// EnumMember is one named value of an enum.
type EnumMember struct {
	Attrs []*Attribute // `@key("name")`: the member's name on the wire (D58)
	Doc   string
	Name  Ident
	Value Expr // nil: the previous value plus one
	Pos   source.Span
}

// BadDecl stands in for a declaration that failed to parse.
type BadDecl struct {
	Pos source.Span
}

func (d *UseDecl) Span() source.Span        { return d.Pos }
func (d *FunDecl) Span() source.Span        { return d.Pos }
func (d *StructDecl) Span() source.Span     { return d.Pos }
func (d *TraitDecl) Span() source.Span      { return d.Pos }
func (d *ImplDecl) Span() source.Span       { return d.Pos }
func (d *ValDecl) Span() source.Span        { return d.Pos }
func (d *ExternBlock) Span() source.Span    { return d.Pos }
func (d *BadDecl) Span() source.Span        { return d.Pos }
func (d *ErrorAliasDecl) Span() source.Span { return d.Pos }
func (d *TypeAliasDecl) Span() source.Span  { return d.Pos }
func (d *EnumDecl) Span() source.Span       { return d.Pos }

func (*ErrorAliasDecl) declNode() {}
func (*TypeAliasDecl) declNode()  {}
func (*EnumDecl) declNode()       {}
func (*UseDecl) declNode()        {}
func (*FunDecl) declNode()        {}
func (*StructDecl) declNode()     {}
func (*TraitDecl) declNode()      {}
func (*ImplDecl) declNode()       {}
func (*ValDecl) declNode()        {}
func (*ExternBlock) declNode()    {}
func (*BadDecl) declNode()        {}

// ---------------------------------------------------------------------------
// Statements

type Stmt interface {
	Node
	stmtNode()
}

type Block struct {
	Stmts []Stmt
	Pos   source.Span
}

// Binding is the left side of `val`: a name or a tuple of bindings (D37).
type Binding struct {
	Name  *Ident    // simple binding
	Tuple []Binding // destructuring
	Type  Type      // optional annotation (simple bindings only)
	Ref   bool      // `&x` in a loop head: bind the element in place (D42)
	Pos   source.Span
}

type ValStmt struct {
	Kind    BindKind
	Binding Binding
	// Pattern is set instead of Binding for a let-else over a variant:
	// `val JObj(fields) = doc else { ... }`.
	Pattern Pattern
	Value   Expr
	// Else is the block of `val x = e else { ... }` (let-else): it runs when
	// e is null, an Err, or does not match Pattern, and must leave.
	Else *Handler
	Pos  source.Span
}

// Handler is `{ body }` or `{ e => body }`: the failure branch of a let-else
// or of `r ?? { e => ... }`, with a Result's error bound to Err.
type Handler struct {
	Err  *Ident
	Body *Block
	Pos  source.Span
}

type ExprStmt struct {
	X Expr
}

// AssignStmt is `target = value` or a compound `target += value`.
type AssignStmt struct {
	Target Expr
	Op     lexer.TokenKind // Assign, PlusEq, ...
	Value  Expr
	Pos    source.Span
}

type ReturnStmt struct {
	Value Expr
	Pos   source.Span
}

// ThrowStmt is `throw e`: fail the enclosing `throws` function with the
// error `e` (D4 sugar for `return Err(e)`).
type ThrowStmt struct {
	Value Expr
	Pos   source.Span
}

type BreakStmt struct {
	Label *Ident
	Pos   source.Span
}

type ContinueStmt struct {
	Label *Ident
	Pos   source.Span
}

// LoopStmt covers `loop { }`, `loop (cond) { }` and `loop (x in c) { }`.
type LoopStmt struct {
	Label *Ident
	Cond  Expr     // `loop (cond)`
	Var   *Binding // `loop (x in iter)`
	Iter  Expr
	Body  *Block
	Pos   source.Span
}

// WithStmt is `with (a = expr, b = expr) { }` (D43).
// WithExpr is `with (r = open()) { ... }` (D43): the bindings are closed
// on every way out of the body. It is an expression — its value is the
// body's — and appears as a statement through ExprStmt.
type WithExpr struct {
	Bindings []WithBinding
	Body     *Block
	Pos      source.Span
}

type WithBinding struct {
	Name  Ident
	Value Expr
}

// ScopeStmt is a structured-concurrency `scope { }` (D34).
type ScopeStmt struct {
	Body *Block
	Pos  source.Span
}

// FunStmt is a local function declaration.
type FunStmt struct {
	Fun *FunDecl
}

type BadStmt struct {
	Pos source.Span
}

func (s *Block) Span() source.Span        { return s.Pos }
func (s *ValStmt) Span() source.Span      { return s.Pos }
func (s *ExprStmt) Span() source.Span     { return s.X.Span() }
func (s *AssignStmt) Span() source.Span   { return s.Pos }
func (s *ReturnStmt) Span() source.Span   { return s.Pos }
func (s *ThrowStmt) Span() source.Span    { return s.Pos }
func (s *BreakStmt) Span() source.Span    { return s.Pos }
func (s *ContinueStmt) Span() source.Span { return s.Pos }
func (s *LoopStmt) Span() source.Span     { return s.Pos }

func (s *ScopeStmt) Span() source.Span { return s.Pos }
func (s *FunStmt) Span() source.Span   { return s.Fun.Pos }
func (s *BadStmt) Span() source.Span   { return s.Pos }

func (*Block) stmtNode()        {}
func (*ValStmt) stmtNode()      {}
func (*ExprStmt) stmtNode()     {}
func (*AssignStmt) stmtNode()   {}
func (*ReturnStmt) stmtNode()   {}
func (*ThrowStmt) stmtNode()    {}
func (*BreakStmt) stmtNode()    {}
func (*ContinueStmt) stmtNode() {}
func (*LoopStmt) stmtNode()     {}

func (*ScopeStmt) stmtNode() {}
func (*FunStmt) stmtNode()   {}
func (*BadStmt) stmtNode()   {}

// ---------------------------------------------------------------------------
// Expressions

type Expr interface {
	Node
	exprNode()
}

type IntLit struct {
	Text string
	Pos  source.Span
}

type FloatLit struct {
	Text string
	Pos  source.Span
}

// StringLit is a string with interpolation already split into parts.
type StringLit struct {
	Parts []StringPart
	Pos   source.Span
}

type StringPart struct {
	Text string
	Expr Expr // set for `$x` / `${expr}` parts
}

type CharLit struct {
	Value string
	Pos   source.Span
}

type BoolLit struct {
	Value bool
	Pos   source.Span
}

type NullLit struct {
	Pos source.Span
}

type SelfExpr struct {
	Pos source.Span
}

// FieldDefaultExpr is the default of field `Index` of struct `Struct` (a
// types.Struct), in a synthesized declaration (a derived decode, D58): the
// value a missing key falls back to, checked in the struct's own module
// exactly as a constructor call that omits the field would.
type FieldDefaultExpr struct {
	Struct any
	Index  int
	Pos    source.Span
}

// TypeExpr is a type in receiver position — `T.decode(from)` with `T`
// already resolved — in a synthesized declaration (a derived impl, D58).
// It never comes from the parser, which spells such a receiver as a name.
type TypeExpr struct {
	Type Type
	Pos  source.Span
}

type NameExpr struct {
	Name     string
	TypeArgs []Type // `Name<T>` before `.f(...)`: a generic type as a static call target
	Pos      source.Span
}

// MemberExpr is `x.name` or `x?.name`.
type MemberExpr struct {
	X    Expr
	Name Ident
	Safe bool
	// `mod.Name<T>` before `.f(...)`: a module-qualified generic type as a
	// static call target.
	TypeArgs []Type
	Pos      source.Span
}

type IndexExpr struct {
	X     Expr
	Index Expr
	Pos   source.Span
}

// Arg is a call argument, optionally named (D28).
type Arg struct {
	Name   *Ident
	Value  Expr
	Spread bool // `xs...`: the list is the whole variadic argument
}

// CallExpr is `f(args)`, `Type<T>(args)`, or `async f(args)`.
type CallExpr struct {
	Fun      Expr
	TypeArgs []Type
	Args     []Arg
	Async    bool
	Pos      source.Span
}

type UnaryExpr struct {
	Op  lexer.TokenKind // Minus, Bang, Amp (address-of), Star (deref)
	X   Expr
	Pos source.Span
}

type BinaryExpr struct {
	Op  lexer.TokenKind
	L   Expr
	R   Expr
	Pos source.Span
}

// OrFailExpr is `x ?! error`: a `T?` or a `Result<T, E1>` becomes a
// `Result<T, E2>` whose failure is the right operand (evaluated only then).
type OrFailExpr struct {
	L, R Expr
	Pos  source.Span
}

// CoalesceExpr is `r ?? fallback` or `r ?? { e => ... }`: a Result's value,
// or the fallback (a value, or a handler that yields one or leaves) when it
// is an Err. `??` is to a Result what `?:` is to a nullable.
type CoalesceExpr struct {
	L       Expr
	R       Expr     // nil when Handler is set
	Handler *Handler // `?? { e => ... }`
	Pos     source.Span
}

// ElvisExpr is `x ?: default` (D30).
type ElvisExpr struct {
	L   Expr
	R   Expr
	Pos source.Span
}

// RangeExpr is `lo..hi` (inclusive) or `lo..<hi` (exclusive) (D29).
type RangeExpr struct {
	Lo, Hi    Expr
	Inclusive bool
	Pos       source.Span
}

// LambdaExpr is `x => e`, `(a, b) => e`, `(x: i32) => { ... }` (D32).
type LambdaExpr struct {
	Params []Param
	Ret    Type
	Body   Expr // a BlockExpr for `{ ... }` bodies
	Pos    source.Span
}

type TupleExpr struct {
	Elems []Expr
	Pos   source.Span
}

// ListLit is `[a, b]`; `mut [a, b]` asks for a MutableList (D25 sugar).
type ListLit struct {
	Elems []Expr
	Mut   bool
	Pos   source.Span
}

type MapEntry struct {
	Key, Value Expr
}

// MapLit is `["k": v]` or `[:]`; `mut [...]` asks for a MutableMap.
type MapLit struct {
	Entries []MapEntry
	Mut     bool
	Pos     source.Span
}

// IfExpr is `if (cond) then else other`. Branch bodies are blocks; a single
// statement body is wrapped in a synthetic block by the parser.
type IfExpr struct {
	Cond Expr
	Then *Block
	Else *Block // nil, or a block holding a single IfExpr for `else if`
	Pos  source.Span
}

// WhenExpr is `when (subject) { arms }` or the subjectless `when { }` (D13).
type WhenExpr struct {
	Subject Expr
	Bind    *Ident // `when (val r = subject)`: a name for the subject, scoped to the arms
	Arms    []*WhenArm
	Pos     source.Span
}

type WhenArm struct {
	Patterns []Pattern // one or more patterns separated by commas
	Cond     Expr      // subjectless form: `cond => ...`
	Guard    Expr      // `if guard` after the pattern
	Else     bool      // `else => ...`
	Body     Expr
	Pos      source.Span
}

// BlockExpr is a block in expression position (lambda / arm bodies).
type BlockExpr struct {
	Block *Block
}

type TryExpr struct {
	X   Expr
	Pos source.Span
}

type AwaitExpr struct {
	X   Expr
	Pos source.Span
}

// IsExpr is the boolean type test `x is T`.
type IsExpr struct {
	X   Expr
	Pat *TypePat
	Not bool
	Pos source.Span
}

// CastExpr is `x as T`.
type CastExpr struct {
	X    Expr
	Type Type
	Pos  source.Span
}

// GatherExpr is `gather { async a(); async b() }` (D36).
type GatherExpr struct {
	Body *Block
	Pos  source.Span
}

// RaceExpr is `race { val x = ch.recv() => body ... }` (D38).
type RaceExpr struct {
	Arms []*RaceArm
	Pos  source.Span
}

type RaceArm struct {
	Binding *Binding // `val job = ...` or nil
	Source  Expr
	Body    Expr
	Pos     source.Span
}

type UnsafeExpr struct {
	Body *Block
	Pos  source.Span
}

type BadExpr struct {
	Pos source.Span
}

func (e *IntLit) Span() source.Span           { return e.Pos }
func (e *FloatLit) Span() source.Span         { return e.Pos }
func (e *StringLit) Span() source.Span        { return e.Pos }
func (e *CharLit) Span() source.Span          { return e.Pos }
func (e *BoolLit) Span() source.Span          { return e.Pos }
func (e *NullLit) Span() source.Span          { return e.Pos }
func (e *SelfExpr) Span() source.Span         { return e.Pos }
func (e *TypeExpr) Span() source.Span         { return e.Pos }
func (e *FieldDefaultExpr) Span() source.Span { return e.Pos }
func (e *NameExpr) Span() source.Span         { return e.Pos }
func (e *MemberExpr) Span() source.Span       { return e.Pos }
func (e *IndexExpr) Span() source.Span        { return e.Pos }
func (e *CallExpr) Span() source.Span         { return e.Pos }
func (e *UnaryExpr) Span() source.Span        { return e.Pos }
func (e *BinaryExpr) Span() source.Span       { return e.Pos }
func (e *ElvisExpr) Span() source.Span        { return e.Pos }
func (e *CoalesceExpr) Span() source.Span     { return e.Pos }
func (e *OrFailExpr) Span() source.Span       { return e.Pos }
func (e *WithExpr) Span() source.Span         { return e.Pos }
func (e *RangeExpr) Span() source.Span        { return e.Pos }
func (e *LambdaExpr) Span() source.Span       { return e.Pos }
func (e *TupleExpr) Span() source.Span        { return e.Pos }
func (e *ListLit) Span() source.Span          { return e.Pos }
func (e *MapLit) Span() source.Span           { return e.Pos }
func (e *IfExpr) Span() source.Span           { return e.Pos }
func (e *WhenExpr) Span() source.Span         { return e.Pos }
func (e *BlockExpr) Span() source.Span        { return e.Block.Pos }
func (e *TryExpr) Span() source.Span          { return e.Pos }
func (e *AwaitExpr) Span() source.Span        { return e.Pos }
func (e *IsExpr) Span() source.Span           { return e.Pos }
func (e *CastExpr) Span() source.Span         { return e.Pos }
func (e *GatherExpr) Span() source.Span       { return e.Pos }
func (e *RaceExpr) Span() source.Span         { return e.Pos }
func (e *UnsafeExpr) Span() source.Span       { return e.Pos }
func (e *BadExpr) Span() source.Span          { return e.Pos }

func (*IntLit) exprNode()           {}
func (*FloatLit) exprNode()         {}
func (*StringLit) exprNode()        {}
func (*CharLit) exprNode()          {}
func (*BoolLit) exprNode()          {}
func (*NullLit) exprNode()          {}
func (*SelfExpr) exprNode()         {}
func (*TypeExpr) exprNode()         {}
func (*FieldDefaultExpr) exprNode() {}
func (*NameExpr) exprNode()         {}
func (*MemberExpr) exprNode()       {}
func (*IndexExpr) exprNode()        {}
func (*CallExpr) exprNode()         {}
func (*UnaryExpr) exprNode()        {}
func (*BinaryExpr) exprNode()       {}
func (*ElvisExpr) exprNode()        {}
func (*CoalesceExpr) exprNode()     {}
func (*OrFailExpr) exprNode()       {}
func (*WithExpr) exprNode()         {}
func (*RangeExpr) exprNode()        {}
func (*LambdaExpr) exprNode()       {}
func (*TupleExpr) exprNode()        {}
func (*ListLit) exprNode()          {}
func (*MapLit) exprNode()           {}
func (*IfExpr) exprNode()           {}
func (*WhenExpr) exprNode()         {}
func (*BlockExpr) exprNode()        {}
func (*TryExpr) exprNode()          {}
func (*AwaitExpr) exprNode()        {}
func (*IsExpr) exprNode()           {}
func (*CastExpr) exprNode()         {}
func (*GatherExpr) exprNode()       {}
func (*RaceExpr) exprNode()         {}
func (*UnsafeExpr) exprNode()       {}
func (*BadExpr) exprNode()          {}

// ---------------------------------------------------------------------------
// Patterns (D13)

type Pattern interface {
	Node
	patternNode()
}

// WildcardPat is `_`.
type WildcardPat struct {
	Pos source.Span
}

// BindPat binds the matched value to a name.
type BindPat struct {
	Name Ident
}

// LiteralPat matches a constant expression: `1`, `"x"`, `null`, `true`.
type LiteralPat struct {
	Value Expr
}

// RangePat is `in lo..hi`.
type RangePat struct {
	Range *RangeExpr
	Pos   source.Span
}

// TypePat is `is T` or `is T(field, other: pat)`. Fields bind by name (D12);
// a bare name is shorthand for `name: name`.
type TypePat struct {
	Type   Type
	Fields []FieldPat
	HasArg bool // distinguishes `is T` from `is T()`
	Pos    source.Span
}

type FieldPat struct {
	Name Ident
	Pat  Pattern // nil means bind under the field name
}

// TuplePat destructures positionally: `(a, b)`.
type TuplePat struct {
	Elems []Pattern
	Pos   source.Span
}

// ListPat destructures a list by position (D62): `[a, b]` matches exactly
// two elements; one RestPat among Elems (`..` or `..rest`) matches any
// number of them.
type ListPat struct {
	Elems []Pattern
	Pos   source.Span
}

// RestPat is `..` or `..name` inside a ListPat; Name is nil for `..`.
type RestPat struct {
	Name *Ident
	Pos  source.Span
}

func (p *WildcardPat) Span() source.Span { return p.Pos }
func (p *BindPat) Span() source.Span     { return p.Name.Pos }
func (p *LiteralPat) Span() source.Span  { return p.Value.Span() }
func (p *RangePat) Span() source.Span    { return p.Pos }
func (p *TypePat) Span() source.Span     { return p.Pos }
func (p *TuplePat) Span() source.Span    { return p.Pos }
func (p *ListPat) Span() source.Span     { return p.Pos }
func (p *RestPat) Span() source.Span     { return p.Pos }

func (*WildcardPat) patternNode() {}
func (*BindPat) patternNode()     {}
func (*LiteralPat) patternNode()  {}
func (*RangePat) patternNode()    {}
func (*TypePat) patternNode()     {}
func (*TuplePat) patternNode()    {}
func (*ListPat) patternNode()     {}
func (*RestPat) patternNode()     {}

// ControlExpr is `return`, `break` or `continue` used in expression
// position (e.g. `x ?: return null`); its type is Never.
type ControlExpr struct {
	Stmt Stmt
}

func (e *ControlExpr) Span() source.Span { return e.Stmt.Span() }
func (*ControlExpr) exprNode()           {}
