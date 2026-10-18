// The syntax tree the parser builds (veles-selfhost-frontend-plan.md, P2).
// Five families — types, declarations, statements, expressions, patterns —
// are five sealed traits, and a node holding a node of its own family holds
// it behind a pointer, so that every value has a size. A node that belongs
// to two families (`Block` is a statement and the body of everything;
// `StaticAssert` is a declaration and a statement) is a plain struct,
// wrapped by a variant in each family.
//
// Only what the parser makes is here; what the checker adds to a tree
// belongs to the checker. A field whose natural name is a keyword is
// spelled around it: `typ`, `isPrivate`, `isStatic`, `isAsync`, `orElse`…
use lexer { Kind }
use source { File, Span }

struct Ident {
  var name: string = ""
  var pos:  Span = Span.none()
}

/// A parsed file: a module is every file in its directory.
struct SourceFile {
  var source: File
  var decls:  List<Decl> = []
  /// The module documentation: a doc comment at the top, set off from the
  /// first declaration by a blank line.
  var doc: string = ""
}

// ---------------------------------------------------------------------------
// types

sealed trait Type

/// `i32`, `List<T>`, `Shape.Circle`, `io.Reader`.
struct NamedType : Type {
  var path: List<Ident> = []
  var args: List<Type> = []
  var pos:  Span = Span.none()
}

/// `T?`.
struct NullableType : Type {
  var elem: *Type
  var pos:  Span = Span.none()
}

/// `*T`, or `*raw T` (unmanaged, D50).
struct PointerType : Type {
  var elem:  *Type
  var isRaw: bool = false
  var pos:   Span = Span.none()
}

/// `(A, B)`; `()` is the unit type.
struct TupleType : Type {
  var elems: List<Type> = []
  var pos:   Span = Span.none()
}

/// `fun(A, B): R suspends throws E` (D40), `sendable fun(…)` (D35),
/// `extern fun(…)` (a C function pointer, D69).
struct FunType : Type {
  var params:   List<Type> = []
  var ret:      (*Type)? = null  // null: unit
  var effects:  Effects = Effects()
  var sendable: bool = false
  var c:        bool = false
  var pos:      Span = Span.none()
}

/// A constant where a type argument goes: the `8` of `Array<u8, 8>` (D121).
struct ConstType : Type {
  var x:   *Expr
  var pos: Span = Span.none()
}

/// `Self` in a trait or an implement.
struct SelfType : Type {
  var pos: Span = Span.none()
}

/// An associated type: `Self.Item`, `I.Item`.
struct AssocType : Type {
  var base: *Type
  var name: Ident = Ident()
  var pos:  Span = Span.none()
}

/// `A | B`: after `throws`, or as a type argument (D45).
struct ErrorUnionType : Type {
  var members: List<Type> = []
  var pos:     Span = Span.none()
}

/// `suspends`, `throws`, `throws E`; `throws` with no type is inferred (D45).
struct Effects {
  var suspending:   bool = false
  var suspendsSpan: Span = Span.none()
  var throwing:     bool = false
  var throwsSpan:   Span = Span.none()
  var error:        (*Type)? = null
}

// ---------------------------------------------------------------------------
// declarations

sealed trait Decl

/// `@name` or `@name(args)` on the line before a declaration.
struct Attribute {
  var name: Ident = Ident()
  var args: List<Arg> = []
  var pos:  Span = Span.none()
}

/// `use a, b.c as d` (M6), `public use` (D89).
struct UseDecl : Decl {
  var specs: List<UseSpec> = []
  var pub:   bool = false
  var pos:   Span = Span.none()
}

/// One import: `a.b`, `a.b as c`, `a.b { f, T as U }` (D85).
struct UseSpec {
  var path:  List<Ident> = []
  var alias: Ident? = null
  var names: List<UseName>? = null  // the braced names; null without braces
  var pub:   bool = false
  var pos:   Span = Span.none()
}

/// `f` or `f as g` in braces.
struct UseName {
  var name:  Ident = Ident()
  var alias: Ident? = null
  var pos:   Span = Span.none()
}

struct TypeParam {
  var name:    Ident = Ident()
  var bounds:  List<Type> = []
  var isConst: bool = false        // `<const N: i64>` (D121)
  var at:      Span = Span.none()  // the word `const`
  var of:      (*Type)? = null     // the constant's type
}

struct Param {
  var name:         Ident = Ident()
  var typ:          (*Type)? = null
  var defaultValue: (*Expr)? = null
  var variadic:     bool = false  // `name: T...`
  var isLazy:       bool = false  // `lazy name: fun(): T` (D90)
  var lazyPos:      Span = Span.none()
  var pattern:      Binding? = null  // a lambda's destructured parameter (D37)
  var pos:          Span = Span.none()
}

/// A function: free, a method, a trait's (no body), or an extern.
struct FunDecl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var isPrivate:  bool = false
  var isStatic:   bool = false
  var isOverride: bool = false
  var isUnsafe:   bool = false
  var isConst:    bool = false  // `const fun` (D113)
  var isExtern:   bool = false
  var exportC:    bool = false  // `extern "C" fun name(…) { }` (D69)
  var isTest:     bool = false  // `test fun` (D78)
  var name:       Ident = Ident()
  var typeParams: List<TypeParam> = []
  var params:     List<Param> = []
  var cVariadic:  bool = false  // `...` in an `extern "C"` block (D123)
  var variadicAt: Span = Span.none()
  var ret:        (*Type)? = null  // null: unit
  var effects:    Effects = Effects()
  var body:       Block? = null
  var exprBody:   (*Expr)? = null  // `fun f() = expr`
  var pos:        Span = Span.none()
}

struct FunDeclNode : Decl {
  var decl: FunDecl
}

/// A field, perhaps with a default.
struct Field {
  var attrs:        List<Attribute> = []
  var pub:          bool = false
  var isInternal:   bool = false
  var isPrivate:    bool = false
  var isVar:        bool = false
  var isVal:        bool = false
  var isProtected:  bool = false
  var doc:          string = ""
  var name:         Ident = Ident()
  var typ:          (*Type)? = null
  var defaultValue: (*Expr)? = null
  var pos:          Span = Span.none()
}

/// `struct Name<T> : SealedParent { fields; methods }`, an `extern struct`,
/// an `extern union`, or an `error Name { }`.
struct StructDecl : Decl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var isExtern:   bool = false
  var union:      bool = false
  var isError:    bool = false
  var errorImpl:  (*ImplDecl)? = null
  var name:       Ident = Ident()
  var typeParams: List<TypeParam> = []
  var variant:    (*Type)? = null
  var fields:     List<Field> = []
  var methods:    List<FunDecl> = []
  var statics:    List<ValDecl> = []
  var impls:      List<ImplDecl> = []
  var init:       Block? = null
  var initPos:    Span = Span.none()
  var initParams: List<Param>? = null
  var pos:        Span = Span.none()
}

struct AssocTypeDecl {
  var name:   Ident = Ident()
  var bounds: List<Type> = []
  var pos:    Span = Span.none()
}

/// `trait Name { type Item; fun m() }`, or `sealed trait Name`.
struct TraitDecl : Decl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var isSealed:   bool = false
  var name:       Ident = Ident()
  var typeParams: List<TypeParam> = []
  var supers:     List<Type> = []
  var assocTypes: List<AssocTypeDecl> = []
  var methods:    List<FunDecl> = []
  var pos:        Span = Span.none()
}

struct AssocTypeBinding {
  var name: Ident = Ident()
  var typ:  (*Type)? = null
}

/// `implement<T> Trait for Type { }`, or `extend<T> Type { }` (D23).
struct ImplDecl : Decl {
  var attrs:      List<Attribute> = []
  var extend:     bool = false
  var inline:     bool = false  // written inside the struct's body
  var braceless:  bool = false  // an empty implement with no `{ }` (D58)
  var errorSugar: bool = false  // made by an `error Name` declaration
  var typeParams: List<TypeParam> = []
  var traitType:  (*Type)? = null  // null for `extend`
  var target:     (*Type)? = null
  var assocTypes: List<AssocTypeBinding> = []
  var methods:    List<FunDecl> = []
  var withFun:    bool = false  // `extend T with fun f() …`, the one-function form (D173)
  var pub:        bool = false  // `public extend`: the visibility of the methods that carry none
  var isInternal: bool = false  // `internal extend`, likewise
  var pos:        Span = Span.none()
}

enum BindKind {
  Val
  Var
  Const
}

fun bindWord(k: BindKind): string => when (k) {
  BindKind.Val   => "val"
  BindKind.Var   => "var"
  BindKind.Const => "const"
}

/// A module-level `val`, `var` or `const`.
struct ValDecl : Decl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var kind:       BindKind = BindKind.Val
  var name:       Ident = Ident()
  var typ:        (*Type)? = null
  var value:      (*Expr)? = null
  var pos:        Span = Span.none()
}

/// `extern "C" { fun …; }`.
struct ExternBlock : Decl {
  var abi:  string = ""
  var funs: List<FunDecl> = []
  var pos:  Span = Span.none()
}

/// `executor name(args) { state and functions }` (D150).
struct ExecutorDecl : Decl {
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var name:       Ident = Ident()
  var args:       List<Arg> = []
  var state:      List<ValDecl> = []
  var funs:       List<FunDecl> = []
  var hasBody:    bool = false
  var pos:        Span = Span.none()
}

/// `error Name = A | B | C` (D45).
struct ErrorAliasDecl : Decl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var name:       Ident = Ident()
  var members:    (*Type)? = null
  var pos:        Span = Span.none()
}

/// `type Name<T> = Type` (D55).
struct TypeAliasDecl : Decl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var name:       Ident = Ident()
  var typeParams: List<TypeParam> = []
  var typ:        (*Type)? = null
  var pos:        Span = Span.none()
}

/// `enum Name : Base { A = 1, B, C }` (D57).
struct EnumDecl : Decl {
  var attrs:      List<Attribute> = []
  var doc:        string = ""
  var pub:        bool = false
  var isInternal: bool = false
  var name:       Ident = Ident()
  var base:       (*Type)? = null  // null: i64
  var members:    List<EnumMember> = []
  var pos:        Span = Span.none()
}

struct EnumMember {
  var attrs: List<Attribute> = []
  var doc:   string = ""
  var name:  Ident = Ident()
  var value: (*Expr)? = null  // null: the previous value plus one
  var pos:   Span = Span.none()
}

/// `test "name" { body }` (D78).
struct TestDecl : Decl {
  var doc:     string = ""
  var label:   string = ""  // `test listMethods "…"`: the optional identifier (D170)
  var labelAt: Span = Span.none()
  var name:    string = ""  // unquoted; empty when only a label is written
  var at:      Span = Span.none()  // the sentence's string (the label's when there is none)
  var body: Block? = null
  var pos:  Span = Span.none()
}

/// What a report calls a test or a suite: the label and the sentence
/// ("listMethods: incrementAt updates an element"), or whichever is written (D170).
fun title(label: string, name: string): string => if (label.isEmpty()) name else if (name.isEmpty()) label else "$label: $name"

/// `suite "name" { … }` (D78).
struct SuiteDecl : Decl {
  var doc:     string = ""
  var label:   string = ""
  var labelAt: Span = Span.none()
  var name:    string = ""
  var at:      Span = Span.none()
  var decls:   List<Decl> = []
  var pos:   Span = Span.none()
}

/// `static assert(cond, "why")` (D113), at module level or in a body.
struct StaticAssert {
  var cond:   *Expr
  var reason: (*Expr)? = null  // null when left out (an error)
  var pos:    Span = Span.none()
}

struct StaticAssertDecl : Decl {
  var assert: StaticAssert
}

/// A declaration that did not parse.
struct BadDecl : Decl {
  var pos: Span = Span.none()
}

// ---------------------------------------------------------------------------
// statements

sealed trait Stmt

struct Block {
  var stmts: List<Stmt> = []
  var pos:   Span = Span.none()
}

struct BlockStmt : Stmt {
  var block: Block
}

/// The left side of `val`: a name or a tuple of bindings (D37).
struct Binding {
  var name:  Ident? = null       // a single name
  var tuple: List<Binding> = []  // destructuring
  var typ:   (*Type)? = null     // a single name's annotation
  var ref:   bool = false        // `&x` in a loop head (D42)
  var pos:   Span = Span.none()
}

struct ValStmt : Stmt {
  var kind:    BindKind = BindKind.Val
  var binding: Binding = Binding()
  /// set instead of `binding` for a let-else over a variant:
  /// `val JObj(fields) = doc else { … }`
  var pattern: (*Pattern)? = null
  var value:   (*Expr)? = null
  /// `val x = e else { … }`: runs when e is null, an Err, or does not match
  var orElse: Handler? = null
  var pos:    Span = Span.none()
}

/// `{ body }` or `{ e => body }`: a let-else's or a catch's failure branch.
struct Handler {
  var err:  Ident? = null
  var body: Block = Block()
  var pos:  Span = Span.none()
}

struct ExprStmt : Stmt {
  var x: *Expr
}

/// `target = value`, or a compound `target += value`.
struct AssignStmt : Stmt {
  var target: *Expr
  var op:     Kind = Kind.Assign
  var value:  *Expr
  var pos:    Span = Span.none()
}

struct ReturnStmt : Stmt {
  var value: (*Expr)? = null
  var pos:   Span = Span.none()
}

/// `throw e` (D4).
struct ThrowStmt : Stmt {
  var value: (*Expr)? = null
  var pos:   Span = Span.none()
}

struct BreakStmt : Stmt {
  var label: Ident? = null
  var pos:   Span = Span.none()
}

struct ContinueStmt : Stmt {
  var label: Ident? = null
  var pos:   Span = Span.none()
}

/// `loop { }`, `loop (cond) { }`, `loop (x in c) { }`.
struct LoopStmt : Stmt {
  var label: Ident? = null
  var cond:  (*Expr)? = null
  var bound: Binding? = null  // `loop (x in …)`
  var iter:  (*Expr)? = null
  var body:  Block = Block()
  var pos:   Span = Span.none()
}

/// `name = e`, or `e` alone (D109): an empty name holds and closes the value
/// without naming it.
struct WithBinding {
  var name:  Ident = Ident()
  var value: *Expr
}

/// The statement form `with x = e` (D100); the rest of its block is its body.
struct WithStmt : Stmt {
  var binding: WithBinding
  var pos:     Span = Span.none()
}

/// `scope { }` (D34); `scope(on: e, workers: n) { }` (D143, D156).
struct ScopeStmt : Stmt {
  var on:      (*Expr)? = null
  var workers: (*Expr)? = null
  var body:    Block = Block()
  var pos:     Span = Span.none()
}

/// A local function.
struct FunStmt : Stmt {
  var decl: *FunDecl
}

struct StaticAssertStmt : Stmt {
  var assert: StaticAssert
}

struct BadStmt : Stmt {
  var pos: Span = Span.none()
}

// ---------------------------------------------------------------------------
// expressions

sealed trait Expr

struct IntLit : Expr {
  var text: string = ""
  var pos:  Span = Span.none()
}

struct FloatLit : Expr {
  var text: string = ""
  var pos:  Span = Span.none()
}

/// A string, its interpolation split into parts.
struct StringLit : Expr {
  var parts: List<StringPart> = []
  var pos:   Span = Span.none()
}

/// `$x` and `${expr}` parts hold the expression; the others the text.
struct StringPart {
  var text: string = ""
  var expr: (*Expr)? = null
  var spec: string = ""  // `${expr:spec}` (D171), as written
  var specPos: Span = Span.none()
  var hasSpec: bool = false
}

/// `tag"text ${x}"` (D129).
struct TemplateExpr : Expr {
  var tag: *Expr  // a NameExpr or a MemberExpr
  var lit: StringLit
  var pos: Span = Span.none()
}

struct CharLit : Expr {
  var value: string = ""
  var pos:   Span = Span.none()
}

struct BoolLit : Expr {
  var value: bool = false
  var pos:   Span = Span.none()
}

struct NullLit : Expr {
  var pos: Span = Span.none()
}

/// `this`.
struct ThisExpr : Expr {
  var pos: Span = Span.none()
}

struct NameExpr : Expr {
  var name:     string = ""
  var typeArgs: List<Type> = []  // `Name<T>.f(…)`
  var pos:      Span = Span.none()
}

/// `x.name` or `x?.name`.
struct MemberExpr : Expr {
  var x:        *Expr
  var name:     Ident = Ident()
  var safe:     bool = false
  var typeArgs: List<Type> = []  // `mod.Name<T>.f(…)`
  var grouped:  bool = false     // in parentheses: ends a `?.` chain (D70)
  var pos:      Span = Span.none()
}

struct IndexExpr : Expr {
  var x:     *Expr
  var index: *Expr
  var pos:   Span = Span.none()
}

/// A call argument, perhaps named (D28).
struct Arg {
  var name:   Ident? = null
  var value:  *Expr
  var spread: bool = false  // `xs...`
}

/// `f(args)`, `Type<T>(args)`, `async f(args)`.
struct CallExpr : Expr {
  var callee:   *Expr
  var typeArgs: List<Type> = []
  var args:     List<Arg> = []
  var isAsync:  bool = false
  var grouped:  bool = false
  // written `async { body }` (D169): the callee is a lambda of no
  // parameters whose body is that block
  var block: bool = false
  var pos:   Span = Span.none()
}

struct UnaryExpr : Expr {
  var op:  Kind = Kind.Minus  // Minus, Bang, Amp, Star, Tilde
  var x:   *Expr
  var pos: Span = Span.none()
}

struct BinaryExpr : Expr {
  var op:    Kind = Kind.Plus
  var left:  *Expr
  var right: *Expr
  var pos:   Span = Span.none()
}

/// `x ?! error`.
struct OrFailExpr : Expr {
  var left:  *Expr
  var right: *Expr
  var pos:   Span = Span.none()
}

/// `r ?? fallback`, or `r ?? { … }`.
struct CoalesceExpr : Expr {
  var left:    *Expr
  var right:   (*Expr)? = null  // null when handler is set
  var handler: Handler? = null
  var pos:     Span = Span.none()
}

/// `do { body } catch (e) { handler }`, or `x catch (e) { handler }` (D98).
struct CatchExpr : Expr {
  var body:    Block? = null
  var x:       (*Expr)? = null
  var handler: Handler = Handler()
  var pos:     Span = Span.none()
}

/// `x ?: default` (D30).
struct ElvisExpr : Expr {
  var left:  *Expr
  var right: *Expr
  var pos:   Span = Span.none()
}

/// `lo..hi` or `lo..<hi` (D29).
struct RangeExpr : Expr {
  var lo:        *Expr
  var hi:        *Expr
  var inclusive: bool = false
  var pos:       Span = Span.none()
}

/// `x => e`, `(a, b) => e`, `(x: i32) => { … }` (D32).
struct LambdaExpr : Expr {
  var params: List<Param> = []
  var ret:    (*Type)? = null
  var body:   *Expr  // a BlockExpr for `{ … }`
  var pos:    Span = Span.none()
}

struct TupleExpr : Expr {
  var elems: List<Expr> = []
  var pos:   Span = Span.none()
}

/// `[a, b]`, `mut [a, b]` (D25).
struct ListLit : Expr {
  var elems: List<Expr> = []
  var isMut: bool = false
  var pos:   Span = Span.none()
}

struct MapEntry {
  var key:   *Expr
  var value: *Expr
}

/// `["k": v]`, `[:]`, `mut [...]`.
struct MapLit : Expr {
  var entries: List<MapEntry> = []
  var isMut:   bool = false
  var pos:     Span = Span.none()
}

/// `if (cond) then else other`; a branch is always a block.
struct IfExpr : Expr {
  var cond:   *Expr
  var then:   Block = Block()
  var orElse: Block? = null  // or a block holding one IfExpr: `else if`
  var pos:    Span = Span.none()
}

/// `val name = value` in an `if`'s condition (D95).
struct LetCond : Expr {
  var name:  Ident = Ident()
  var value: *Expr
  var pos:   Span = Span.none()
}

/// `when (subject) { arms }`, or `when { }` (D13).
struct WhenExpr : Expr {
  var subject: (*Expr)? = null
  var bind:    Ident? = null  // `when (val r = subject)`
  var arms:    List<WhenArm> = []
  var pos:     Span = Span.none()
}

struct WhenArm {
  var patterns: List<Pattern> = []
  var cond:     (*Expr)? = null  // the subjectless form
  var guard:    (*Expr)? = null
  var isElse:   bool = false
  var body:     *Expr
  var pos:      Span = Span.none()
}

/// A block where an expression goes: lambda and arm bodies.
struct BlockExpr : Expr {
  var block: Block
}

struct TryExpr : Expr {
  var x:   *Expr
  var pos: Span = Span.none()
}

struct AwaitExpr : Expr {
  var x:   *Expr
  var pos: Span = Span.none()
}

/// `x is T`, `x !is T`.
struct IsExpr : Expr {
  var x:   *Expr
  var pat: TypePat
  var not: bool = false
  var pos: Span = Span.none()
}

/// `T implements Trait` (D117).
struct ImplementsExpr : Expr {
  var typ:       *Type
  var traitType: *Type
  var pos:       Span = Span.none()
}

/// `x as T`.
struct CastExpr : Expr {
  var x:   *Expr
  var typ: *Type
  var pos: Span = Span.none()
}

/// `gather { … }` (D36); `gather(on: e, workers: n) { … }` (D143, D156).
struct GatherExpr : Expr {
  var on:      (*Expr)? = null
  var workers: (*Expr)? = null
  var body:    Block = Block()
  var pos:     Span = Span.none()
}

/// A `scope { … }` used as a value: its body's last expression (D156). In
/// statement position a scope is a ScopeStmt.
struct ScopeExpr : Expr {
  var on:      (*Expr)? = null
  var workers: (*Expr)? = null
  var body:    Block = Block()
  var pos:     Span = Span.none()
}

/// `race { body }` (D169): the tasks the body starts race. `arms` instead of
/// `body` for the removed `race` with arms, now `select`.
struct RaceExpr : Expr {
  var body: Block? = null
  var arms: List<RaceArm> = []
  var pos:  Span = Span.none()
}

/// `select { val x = ch.recv() => body … }` (D169; the arms of D38/D108/D168).
struct SelectExpr : Expr {
  var arms: List<RaceArm> = []
  var pos:  Span = Span.none()
}

struct RaceArm {
  var binding: Binding? = null  // `val x = …`
  var source:  *Expr
  var name:    Ident? = null    // `src as name =>` (D168, removed by D169)
  var body:    (*Expr)? = null  // null for a bare arm (D168, removed by D169)
  var pos:     Span = Span.none()
}

struct UnsafeExpr : Expr {
  var body: Block = Block()
  var pos:  Span = Span.none()
}

/// `with (r = open()) { … }` (D43); a statement through ExprStmt.
struct WithExpr : Expr {
  var bindings: List<WithBinding> = []
  var body:     Block = Block()
  var pos:      Span = Span.none()
}

/// `return`, `break` or `continue` where an expression goes: `x ?: return`.
struct ControlExpr : Expr {
  var stmt: *Stmt
}

struct BadExpr : Expr {
  var pos: Span = Span.none()
}

// ---------------------------------------------------------------------------
// patterns (D13)

sealed trait Pattern

/// `_`.
struct WildcardPat : Pattern {
  var pos: Span = Span.none()
}

/// A name the matched value is bound to.
struct BindPat : Pattern {
  var name: Ident = Ident()
}

/// A constant: `1`, `"x"`, `null`, `true`.
struct LiteralPat : Pattern {
  var value: *Expr
}

/// `in lo..hi`.
struct RangePat : Pattern {
  var range: RangeExpr
  var pos:   Span = Span.none()
}

/// `is T` or `is T(field, other: pat)` (D12).
struct TypePat : Pattern {
  var typ:    *Type
  var fields: List<FieldPat> = []
  var hasArg: bool = false  // `is T()` rather than `is T`
  var pos:    Span = Span.none()
}

struct FieldPat {
  var name: Ident = Ident()
  var pat:  (*Pattern)? = null  // null: bind under the field's name
}

/// `(a, b)`.
struct TuplePat : Pattern {
  var elems: List<Pattern> = []
  var pos:   Span = Span.none()
}

/// `[a, b]`, `[first, ..rest]` (D62).
struct ListPat : Pattern {
  var elems: List<Pattern> = []
  var pos:   Span = Span.none()
}

/// `..` or `..name` in a list pattern.
struct RestPat : Pattern {
  var name: Ident? = null
  var pos:  Span = Span.none()
}

// ---------------------------------------------------------------------------
// where a node is: one exhaustive `when` per family

fun typeSpan(t: Type): Span => when (t) {
  is NamedType      => t.pos
  is NullableType   => t.pos
  is PointerType    => t.pos
  is TupleType      => t.pos
  is FunType        => t.pos
  is ConstType      => t.pos
  is SelfType       => t.pos
  is AssocType      => t.pos
  is ErrorUnionType => t.pos
}

fun declSpan(d: Decl): Span => when (d) {
  is UseDecl          => d.pos
  is FunDeclNode      => d.decl.pos
  is StructDecl       => d.pos
  is TraitDecl        => d.pos
  is ImplDecl         => d.pos
  is ValDecl          => d.pos
  is ExternBlock      => d.pos
  is ExecutorDecl     => d.pos
  is ErrorAliasDecl   => d.pos
  is TypeAliasDecl    => d.pos
  is EnumDecl         => d.pos
  is TestDecl         => d.pos
  is SuiteDecl        => d.pos
  is StaticAssertDecl => d.assert.pos
  is BadDecl          => d.pos
}

fun stmtSpan(s: Stmt): Span => when (s) {
  is BlockStmt        => s.block.pos
  is ValStmt          => s.pos
  is ExprStmt         => exprSpan(*s.x)
  is AssignStmt       => s.pos
  is ReturnStmt       => s.pos
  is ThrowStmt        => s.pos
  is BreakStmt        => s.pos
  is ContinueStmt     => s.pos
  is LoopStmt         => s.pos
  is WithStmt         => s.pos
  is ScopeStmt        => s.pos
  is FunStmt          => s.decl.pos
  is StaticAssertStmt => s.assert.pos
  is BadStmt          => s.pos
}

fun exprSpan(e: Expr): Span => when (e) {
  is IntLit         => e.pos
  is FloatLit       => e.pos
  is StringLit      => e.pos
  is TemplateExpr   => e.pos
  is CharLit        => e.pos
  is BoolLit        => e.pos
  is NullLit        => e.pos
  is ThisExpr       => e.pos
  is NameExpr       => e.pos
  is MemberExpr     => e.pos
  is IndexExpr      => e.pos
  is CallExpr       => e.pos
  is UnaryExpr      => e.pos
  is BinaryExpr     => e.pos
  is OrFailExpr     => e.pos
  is CoalesceExpr   => e.pos
  is CatchExpr      => e.pos
  is ElvisExpr      => e.pos
  is RangeExpr      => e.pos
  is LambdaExpr     => e.pos
  is TupleExpr      => e.pos
  is ListLit        => e.pos
  is MapLit         => e.pos
  is IfExpr         => e.pos
  is LetCond        => e.pos
  is WhenExpr       => e.pos
  is BlockExpr      => e.block.pos
  is TryExpr        => e.pos
  is AwaitExpr      => e.pos
  is IsExpr         => e.pos
  is ImplementsExpr => e.pos
  is CastExpr       => e.pos
  is GatherExpr     => e.pos
  is ScopeExpr      => e.pos
  is RaceExpr       => e.pos
  is SelectExpr     => e.pos
  is UnsafeExpr     => e.pos
  is WithExpr       => e.pos
  is ControlExpr    => stmtSpan(*e.stmt)
  is BadExpr        => e.pos
}

fun patternSpan(p: Pattern): Span => when (p) {
  is WildcardPat => p.pos
  is BindPat     => p.name.pos
  is LiteralPat  => exprSpan(*p.value)
  is RangePat    => p.pos
  is TypePat     => p.pos
  is TuplePat    => p.pos
  is ListPat     => p.pos
  is RestPat     => p.pos
}
