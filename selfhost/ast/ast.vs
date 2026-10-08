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
use lexer { Kind }, source { File, Span }

public struct Ident {
  public var name: string = ""
  public var pos:  Span = Span.none()
}

/// A parsed file: a module is every file in its directory.
public struct SourceFile {
  public var source: File
  public var decls:  List<Decl> = []
  /// The module documentation: a doc comment at the top, set off from the
  /// first declaration by a blank line.
  public var doc: string = ""
}

// ---------------------------------------------------------------------------
// types

public sealed trait Type

/// `i32`, `List<T>`, `Shape.Circle`, `io.Reader`.
public struct NamedType : Type {
  public var path: List<Ident> = []
  public var args: List<Type> = []
  public var pos:  Span = Span.none()
}

/// `T?`.
public struct NullableType : Type {
  public var elem: *Type
  public var pos:  Span = Span.none()
}

/// `*T`, or `*raw T` (unmanaged, D50).
public struct PointerType : Type {
  public var elem:  *Type
  public var isRaw: bool = false
  public var pos:   Span = Span.none()
}

/// `(A, B)`; `()` is the unit type.
public struct TupleType : Type {
  public var elems: List<Type> = []
  public var pos:   Span = Span.none()
}

/// `fun(A, B): R suspends throws E` (D40), `sendable fun(…)` (D35),
/// `extern fun(…)` (a C function pointer, D69).
public struct FunType : Type {
  public var params:   List<Type> = []
  public var ret:      (*Type)? = null  // null: unit
  public var effects:  Effects = Effects()
  public var sendable: bool = false
  public var c:        bool = false
  public var pos:      Span = Span.none()
}

/// A constant where a type argument goes: the `8` of `Array<u8, 8>` (D121).
public struct ConstType : Type {
  public var x:   *Expr
  public var pos: Span = Span.none()
}

/// `Self` in a trait or an implement.
public struct SelfType : Type {
  public var pos: Span = Span.none()
}

/// An associated type: `Self.Item`, `I.Item`.
public struct AssocType : Type {
  public var base: *Type
  public var name: Ident = Ident()
  public var pos:  Span = Span.none()
}

/// `A | B`: after `throws`, or as a type argument (D45).
public struct ErrorUnionType : Type {
  public var members: List<Type> = []
  public var pos:     Span = Span.none()
}

/// `suspends`, `throws`, `throws E`; `throws` with no type is inferred (D45).
public struct Effects {
  public var suspending:   bool = false
  public var suspendsSpan: Span = Span.none()
  public var throwing:     bool = false
  public var throwsSpan:   Span = Span.none()
  public var error:        (*Type)? = null
}

// ---------------------------------------------------------------------------
// declarations

public sealed trait Decl

/// `@name` or `@name(args)` on the line before a declaration.
public struct Attribute {
  public var name: Ident = Ident()
  public var args: List<Arg> = []
  public var pos:  Span = Span.none()
}

/// `use a, b.c as d` (M6), `public use` (D89).
public struct UseDecl : Decl {
  public var specs: List<UseSpec> = []
  public var pub:   bool = false
  public var pos:   Span = Span.none()
}

/// One import: `a.b`, `a.b as c`, `a.b { f, T as U }` (D85).
public struct UseSpec {
  public var path:  List<Ident> = []
  public var alias: Ident? = null
  public var names: List<UseName>? = null  // the braced names; null without braces
  public var pub:   bool = false
  public var pos:   Span = Span.none()
}

/// `f` or `f as g` in braces.
public struct UseName {
  public var name:  Ident = Ident()
  public var alias: Ident? = null
  public var pos:   Span = Span.none()
}

public struct TypeParam {
  public var name:    Ident = Ident()
  public var bounds:  List<Type> = []
  public var isConst: bool = false        // `<const N: i64>` (D121)
  public var at:      Span = Span.none()  // the word `const`
  public var of:      (*Type)? = null     // the constant's type
}

public struct Param {
  public var name:         Ident = Ident()
  public var typ:          (*Type)? = null
  public var defaultValue: (*Expr)? = null
  public var variadic:     bool = false  // `name: T...`
  public var isLazy:       bool = false  // `lazy name: fun(): T` (D90)
  public var lazyPos:      Span = Span.none()
  public var pattern:      Binding? = null  // a lambda's destructured parameter (D37)
  public var pos:          Span = Span.none()
}

/// A function: free, a method, a trait's (no body), or an extern.
public struct FunDecl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var isPrivate:  bool = false
  public var isStatic:   bool = false
  public var isOverride: bool = false
  public var isUnsafe:   bool = false
  public var isConst:    bool = false  // `const fun` (D113)
  public var isExtern:   bool = false
  public var exportC:    bool = false  // `extern "C" fun name(…) { }` (D69)
  public var isTest:     bool = false  // `test fun` (D78)
  public var name:       Ident = Ident()
  public var typeParams: List<TypeParam> = []
  public var params:     List<Param> = []
  public var cVariadic:  bool = false  // `...` in an `extern "C"` block (D123)
  public var variadicAt: Span = Span.none()
  public var ret:        (*Type)? = null  // null: unit
  public var effects:    Effects = Effects()
  public var body:       Block? = null
  public var exprBody:   (*Expr)? = null  // `fun f() = expr`
  public var pos:        Span = Span.none()
}

public struct FunDeclNode : Decl {
  public var decl: FunDecl
}

/// A field, perhaps with a default.
public struct Field {
  public var attrs:        List<Attribute> = []
  public var pub:          bool = false
  public var isInternal:   bool = false
  public var isPrivate:    bool = false
  public var isVar:        bool = false
  public var isVal:        bool = false
  public var isProtected:  bool = false
  public var doc:          string = ""
  public var name:         Ident = Ident()
  public var typ:          (*Type)? = null
  public var defaultValue: (*Expr)? = null
  public var pos:          Span = Span.none()
}

/// `struct Name<T> : SealedParent { fields; methods }`, an `extern struct`,
/// an `extern union`, or an `error Name { }`.
public struct StructDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var isExtern:   bool = false
  public var union:      bool = false
  public var isError:    bool = false
  public var errorImpl:  (*ImplDecl)? = null
  public var name:       Ident = Ident()
  public var typeParams: List<TypeParam> = []
  public var variant:    (*Type)? = null
  public var fields:     List<Field> = []
  public var methods:    List<FunDecl> = []
  public var statics:    List<ValDecl> = []
  public var impls:      List<ImplDecl> = []
  public var init:       Block? = null
  public var initPos:    Span = Span.none()
  public var initParams: List<Param>? = null
  public var pos:        Span = Span.none()
}

public struct AssocTypeDecl {
  public var name:   Ident = Ident()
  public var bounds: List<Type> = []
  public var pos:    Span = Span.none()
}

/// `trait Name { type Item; fun m() }`, or `sealed trait Name`.
public struct TraitDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var isSealed:   bool = false
  public var name:       Ident = Ident()
  public var typeParams: List<TypeParam> = []
  public var supers:     List<Type> = []
  public var assocTypes: List<AssocTypeDecl> = []
  public var methods:    List<FunDecl> = []
  public var pos:        Span = Span.none()
}

public struct AssocTypeBinding {
  public var name: Ident = Ident()
  public var typ:  (*Type)? = null
}

/// `implement<T> Trait for Type { }`, or `extend<T> Type { }` (D23).
public struct ImplDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var extend:     bool = false
  public var inline:     bool = false  // written inside the struct's body
  public var braceless:  bool = false  // an empty implement with no `{ }` (D58)
  public var errorSugar: bool = false  // made by an `error Name` declaration
  public var typeParams: List<TypeParam> = []
  public var traitType:  (*Type)? = null  // null for `extend`
  public var target:     (*Type)? = null
  public var assocTypes: List<AssocTypeBinding> = []
  public var methods:    List<FunDecl> = []
  public var pos:        Span = Span.none()
}

public enum BindKind {
  Val
  Var
  Const
}

public fun bindWord(k: BindKind): string = when (k) {
  BindKind.Val   => "val"
  BindKind.Var   => "var"
  BindKind.Const => "const"
}

/// A module-level `val`, `var` or `const`.
public struct ValDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var kind:       BindKind = BindKind.Val
  public var name:       Ident = Ident()
  public var typ:        (*Type)? = null
  public var value:      (*Expr)? = null
  public var pos:        Span = Span.none()
}

/// `extern "C" { fun …; }`.
public struct ExternBlock : Decl {
  public var abi:  string = ""
  public var funs: List<FunDecl> = []
  public var pos:  Span = Span.none()
}

/// `error Name = A | B | C` (D45).
public struct ErrorAliasDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var name:       Ident = Ident()
  public var members:    (*Type)? = null
  public var pos:        Span = Span.none()
}

/// `type Name<T> = Type` (D55).
public struct TypeAliasDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var name:       Ident = Ident()
  public var typeParams: List<TypeParam> = []
  public var typ:        (*Type)? = null
  public var pos:        Span = Span.none()
}

/// `enum Name : Base { A = 1, B, C }` (D57).
public struct EnumDecl : Decl {
  public var attrs:      List<Attribute> = []
  public var doc:        string = ""
  public var pub:        bool = false
  public var isInternal: bool = false
  public var name:       Ident = Ident()
  public var base:       (*Type)? = null  // null: i64
  public var members:    List<EnumMember> = []
  public var pos:        Span = Span.none()
}

public struct EnumMember {
  public var attrs: List<Attribute> = []
  public var doc:   string = ""
  public var name:  Ident = Ident()
  public var value: (*Expr)? = null  // null: the previous value plus one
  public var pos:   Span = Span.none()
}

/// `test "name" { body }` (D78).
public struct TestDecl : Decl {
  public var doc:  string = ""
  public var name: string = ""  // unquoted
  public var at:   Span = Span.none()
  public var body: Block? = null
  public var pos:  Span = Span.none()
}

/// `suite "name" { … }` (D78).
public struct SuiteDecl : Decl {
  public var doc:   string = ""
  public var name:  string = ""
  public var at:    Span = Span.none()
  public var decls: List<Decl> = []
  public var pos:   Span = Span.none()
}

/// `static assert(cond, "why")` (D113), at module level or in a body.
public struct StaticAssert {
  public var cond:   *Expr
  public var reason: (*Expr)? = null  // null when left out (an error)
  public var pos:    Span = Span.none()
}

public struct StaticAssertDecl : Decl {
  public var assert: StaticAssert
}

/// A declaration that did not parse.
public struct BadDecl : Decl {
  public var pos: Span = Span.none()
}

// ---------------------------------------------------------------------------
// statements

public sealed trait Stmt

public struct Block {
  public var stmts: List<Stmt> = []
  public var pos:   Span = Span.none()
}

public struct BlockStmt : Stmt {
  public var block: Block
}

/// The left side of `val`: a name or a tuple of bindings (D37).
public struct Binding {
  public var name:  Ident? = null       // a single name
  public var tuple: List<Binding> = []  // destructuring
  public var typ:   (*Type)? = null     // a single name's annotation
  public var ref:   bool = false        // `&x` in a loop head (D42)
  public var pos:   Span = Span.none()
}

public struct ValStmt : Stmt {
  public var kind:    BindKind = BindKind.Val
  public var binding: Binding = Binding()
  /// set instead of `binding` for a let-else over a variant:
  /// `val JObj(fields) = doc else { … }`
  public var pattern: (*Pattern)? = null
  public var value:   (*Expr)? = null
  /// `val x = e else { … }`: runs when e is null, an Err, or does not match
  public var orElse: Handler? = null
  public var pos:    Span = Span.none()
}

/// `{ body }` or `{ e => body }`: a let-else's or a catch's failure branch.
public struct Handler {
  public var err:  Ident? = null
  public var body: Block = Block()
  public var pos:  Span = Span.none()
}

public struct ExprStmt : Stmt {
  public var x: *Expr
}

/// `target = value`, or a compound `target += value`.
public struct AssignStmt : Stmt {
  public var target: *Expr
  public var op:     Kind = Kind.Assign
  public var value:  *Expr
  public var pos:    Span = Span.none()
}

public struct ReturnStmt : Stmt {
  public var value: (*Expr)? = null
  public var pos:   Span = Span.none()
}

/// `throw e` (D4).
public struct ThrowStmt : Stmt {
  public var value: (*Expr)? = null
  public var pos:   Span = Span.none()
}

public struct BreakStmt : Stmt {
  public var label: Ident? = null
  public var pos:   Span = Span.none()
}

public struct ContinueStmt : Stmt {
  public var label: Ident? = null
  public var pos:   Span = Span.none()
}

/// `loop { }`, `loop (cond) { }`, `loop (x in c) { }`.
public struct LoopStmt : Stmt {
  public var label: Ident? = null
  public var cond:  (*Expr)? = null
  public var bound: Binding? = null  // `loop (x in …)`
  public var iter:  (*Expr)? = null
  public var body:  Block = Block()
  public var pos:   Span = Span.none()
}

/// `name = e`, or `e` alone (D109): an empty name holds and closes the value
/// without naming it.
public struct WithBinding {
  public var name:  Ident = Ident()
  public var value: *Expr
}

/// The statement form `with x = e` (D100); the rest of its block is its body.
public struct WithStmt : Stmt {
  public var binding: WithBinding
  public var pos:     Span = Span.none()
}

/// `scope { }` (D34).
public struct ScopeStmt : Stmt {
  public var body: Block = Block()
  public var pos:  Span = Span.none()
}

/// A local function.
public struct FunStmt : Stmt {
  public var decl: *FunDecl
}

public struct StaticAssertStmt : Stmt {
  public var assert: StaticAssert
}

public struct BadStmt : Stmt {
  public var pos: Span = Span.none()
}

// ---------------------------------------------------------------------------
// expressions

public sealed trait Expr

public struct IntLit : Expr {
  public var text: string = ""
  public var pos:  Span = Span.none()
}

public struct FloatLit : Expr {
  public var text: string = ""
  public var pos:  Span = Span.none()
}

/// A string, its interpolation split into parts.
public struct StringLit : Expr {
  public var parts: List<StringPart> = []
  public var pos:   Span = Span.none()
}

/// `$x` and `${expr}` parts hold the expression; the others the text.
public struct StringPart {
  public var text: string = ""
  public var expr: (*Expr)? = null
}

/// `tag"text ${x}"` (D129).
public struct TemplateExpr : Expr {
  public var tag: *Expr  // a NameExpr or a MemberExpr
  public var lit: StringLit
  public var pos: Span = Span.none()
}

public struct CharLit : Expr {
  public var value: string = ""
  public var pos:   Span = Span.none()
}

public struct BoolLit : Expr {
  public var value: bool = false
  public var pos:   Span = Span.none()
}

public struct NullLit : Expr {
  public var pos: Span = Span.none()
}

/// `this`.
public struct ThisExpr : Expr {
  public var pos: Span = Span.none()
}

public struct NameExpr : Expr {
  public var name:     string = ""
  public var typeArgs: List<Type> = []  // `Name<T>.f(…)`
  public var pos:      Span = Span.none()
}

/// `x.name` or `x?.name`.
public struct MemberExpr : Expr {
  public var x:        *Expr
  public var name:     Ident = Ident()
  public var safe:     bool = false
  public var typeArgs: List<Type> = []  // `mod.Name<T>.f(…)`
  public var grouped:  bool = false     // in parentheses: ends a `?.` chain (D70)
  public var pos:      Span = Span.none()
}

public struct IndexExpr : Expr {
  public var x:     *Expr
  public var index: *Expr
  public var pos:   Span = Span.none()
}

/// A call argument, perhaps named (D28).
public struct Arg {
  public var name:   Ident? = null
  public var value:  *Expr
  public var spread: bool = false  // `xs...`
}

/// `f(args)`, `Type<T>(args)`, `async f(args)`.
public struct CallExpr : Expr {
  public var callee:   *Expr
  public var typeArgs: List<Type> = []
  public var args:     List<Arg> = []
  public var isAsync:  bool = false
  public var grouped:  bool = false
  public var pos:      Span = Span.none()
}

public struct UnaryExpr : Expr {
  public var op:  Kind = Kind.Minus  // Minus, Bang, Amp, Star, Tilde
  public var x:   *Expr
  public var pos: Span = Span.none()
}

public struct BinaryExpr : Expr {
  public var op:    Kind = Kind.Plus
  public var left:  *Expr
  public var right: *Expr
  public var pos:   Span = Span.none()
}

/// `x ?! error`.
public struct OrFailExpr : Expr {
  public var left:  *Expr
  public var right: *Expr
  public var pos:   Span = Span.none()
}

/// `r ?? fallback`, or `r ?? { … }`.
public struct CoalesceExpr : Expr {
  public var left:    *Expr
  public var right:   (*Expr)? = null  // null when handler is set
  public var handler: Handler? = null
  public var pos:     Span = Span.none()
}

/// `do { body } catch (e) { handler }`, or `x catch (e) { handler }` (D98).
public struct CatchExpr : Expr {
  public var body:    Block? = null
  public var x:       (*Expr)? = null
  public var handler: Handler = Handler()
  public var pos:     Span = Span.none()
}

/// `x ?: default` (D30).
public struct ElvisExpr : Expr {
  public var left:  *Expr
  public var right: *Expr
  public var pos:   Span = Span.none()
}

/// `lo..hi` or `lo..<hi` (D29).
public struct RangeExpr : Expr {
  public var lo:        *Expr
  public var hi:        *Expr
  public var inclusive: bool = false
  public var pos:       Span = Span.none()
}

/// `x => e`, `(a, b) => e`, `(x: i32) => { … }` (D32).
public struct LambdaExpr : Expr {
  public var params: List<Param> = []
  public var ret:    (*Type)? = null
  public var body:   *Expr  // a BlockExpr for `{ … }`
  public var pos:    Span = Span.none()
}

public struct TupleExpr : Expr {
  public var elems: List<Expr> = []
  public var pos:   Span = Span.none()
}

/// `[a, b]`, `mut [a, b]` (D25).
public struct ListLit : Expr {
  public var elems: List<Expr> = []
  public var isMut: bool = false
  public var pos:   Span = Span.none()
}

public struct MapEntry {
  public var key:   *Expr
  public var value: *Expr
}

/// `["k": v]`, `[:]`, `mut [...]`.
public struct MapLit : Expr {
  public var entries: List<MapEntry> = []
  public var isMut:   bool = false
  public var pos:     Span = Span.none()
}

/// `if (cond) then else other`; a branch is always a block.
public struct IfExpr : Expr {
  public var cond:   *Expr
  public var then:   Block = Block()
  public var orElse: Block? = null  // or a block holding one IfExpr: `else if`
  public var pos:    Span = Span.none()
}

/// `val name = value` in an `if`'s condition (D95).
public struct LetCond : Expr {
  public var name:  Ident = Ident()
  public var value: *Expr
  public var pos:   Span = Span.none()
}

/// `when (subject) { arms }`, or `when { }` (D13).
public struct WhenExpr : Expr {
  public var subject: (*Expr)? = null
  public var bind:    Ident? = null  // `when (val r = subject)`
  public var arms:    List<WhenArm> = []
  public var pos:     Span = Span.none()
}

public struct WhenArm {
  public var patterns: List<Pattern> = []
  public var cond:     (*Expr)? = null  // the subjectless form
  public var guard:    (*Expr)? = null
  public var isElse:   bool = false
  public var body:     *Expr
  public var pos:      Span = Span.none()
}

/// A block where an expression goes: lambda and arm bodies.
public struct BlockExpr : Expr {
  public var block: Block
}

public struct TryExpr : Expr {
  public var x:   *Expr
  public var pos: Span = Span.none()
}

public struct AwaitExpr : Expr {
  public var x:   *Expr
  public var pos: Span = Span.none()
}

/// `x is T`, `x !is T`.
public struct IsExpr : Expr {
  public var x:   *Expr
  public var pat: TypePat
  public var not: bool = false
  public var pos: Span = Span.none()
}

/// `T implements Trait` (D117).
public struct ImplementsExpr : Expr {
  public var typ:       *Type
  public var traitType: *Type
  public var pos:       Span = Span.none()
}

/// `x as T`.
public struct CastExpr : Expr {
  public var x:   *Expr
  public var typ: *Type
  public var pos: Span = Span.none()
}

/// `gather { … }` (D36).
public struct GatherExpr : Expr {
  public var body: Block = Block()
  public var pos:  Span = Span.none()
}

/// `race { val x = ch.recv() => body … }` (D38).
public struct RaceExpr : Expr {
  public var arms: List<RaceArm> = []
  public var pos:  Span = Span.none()
}

public struct RaceArm {
  public var binding: Binding? = null
  public var source:  *Expr
  public var body:    *Expr
  public var pos:     Span = Span.none()
}

public struct UnsafeExpr : Expr {
  public var body: Block = Block()
  public var pos:  Span = Span.none()
}

/// `with (r = open()) { … }` (D43); a statement through ExprStmt.
public struct WithExpr : Expr {
  public var bindings: List<WithBinding> = []
  public var body:     Block = Block()
  public var pos:      Span = Span.none()
}

/// `return`, `break` or `continue` where an expression goes: `x ?: return`.
public struct ControlExpr : Expr {
  public var stmt: *Stmt
}

public struct BadExpr : Expr {
  public var pos: Span = Span.none()
}

// ---------------------------------------------------------------------------
// patterns (D13)

public sealed trait Pattern

/// `_`.
public struct WildcardPat : Pattern {
  public var pos: Span = Span.none()
}

/// A name the matched value is bound to.
public struct BindPat : Pattern {
  public var name: Ident = Ident()
}

/// A constant: `1`, `"x"`, `null`, `true`.
public struct LiteralPat : Pattern {
  public var value: *Expr
}

/// `in lo..hi`.
public struct RangePat : Pattern {
  public var range: RangeExpr
  public var pos:   Span = Span.none()
}

/// `is T` or `is T(field, other: pat)` (D12).
public struct TypePat : Pattern {
  public var typ:    *Type
  public var fields: List<FieldPat> = []
  public var hasArg: bool = false  // `is T()` rather than `is T`
  public var pos:    Span = Span.none()
}

public struct FieldPat {
  public var name: Ident = Ident()
  public var pat:  (*Pattern)? = null  // null: bind under the field's name
}

/// `(a, b)`.
public struct TuplePat : Pattern {
  public var elems: List<Pattern> = []
  public var pos:   Span = Span.none()
}

/// `[a, b]`, `[first, ..rest]` (D62).
public struct ListPat : Pattern {
  public var elems: List<Pattern> = []
  public var pos:   Span = Span.none()
}

/// `..` or `..name` in a list pattern.
public struct RestPat : Pattern {
  public var name: Ident? = null
  public var pos:  Span = Span.none()
}

// ---------------------------------------------------------------------------
// where a node is: one exhaustive `when` per family

public fun typeSpan(t: Type): Span = when (t) {
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

public fun declSpan(d: Decl): Span = when (d) {
  is UseDecl          => d.pos
  is FunDeclNode      => d.decl.pos
  is StructDecl       => d.pos
  is TraitDecl        => d.pos
  is ImplDecl         => d.pos
  is ValDecl          => d.pos
  is ExternBlock      => d.pos
  is ErrorAliasDecl   => d.pos
  is TypeAliasDecl    => d.pos
  is EnumDecl         => d.pos
  is TestDecl         => d.pos
  is SuiteDecl        => d.pos
  is StaticAssertDecl => d.assert.pos
  is BadDecl          => d.pos
}

public fun stmtSpan(s: Stmt): Span = when (s) {
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

public fun exprSpan(e: Expr): Span = when (e) {
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
  is RaceExpr       => e.pos
  is UnsafeExpr     => e.pos
  is WithExpr       => e.pos
  is ControlExpr    => stmtSpan(*e.stmt)
  is BadExpr        => e.pos
}

public fun patternSpan(p: Pattern): Span = when (p) {
  is WildcardPat => p.pos
  is BindPat     => p.name.pos
  is LiteralPat  => exprSpan(*p.value)
  is RangePat    => p.pos
  is TypePat     => p.pos
  is TuplePat    => p.pos
  is ListPat     => p.pos
  is RestPat     => p.pos
}
