// Generated from selfhost/ast/ast.vs by `go test ./selfhost -run LintAPI -update`; edit the source, not this file.

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

public sealed trait Type {
  /// `i32`, `List<T>`, `Shape.Circle`, `io.Reader`.
  struct NamedType {
    public var path: List<Ident> = []
    public var args: List<Type> = []
    public var pos:  Span = Span.none()
  }
  /// `T?`.
  struct NullableType {
    public var elem: *Type
    public var pos:  Span = Span.none()
  }
  /// `*T`, or `*raw T` (unmanaged, D50).
  struct PointerType {
    public var elem:  *Type
    public var isRaw: bool = false
    public var pos:   Span = Span.none()
  }
  /// `(A, B)`; `()` is the unit type. `(key: K, value: V)` labels every
  /// element (D166); `labels` is empty when none is labelled.
  struct TupleType {
    public var elems:  List<Type> = []
    public var labels: List<Ident> = []
    public var pos:    Span = Span.none()
  }
  /// `fun(A, B): R suspends throws E` (D40), `sendable fun(…)` (D35),
  /// `extern fun(…)` (a C function pointer, D69).
  struct FunType {
    public var params:   List<Type> = []
    public var ret:      (*Type)? = null  // null: unit
    public var effects:  Effects = Effects()
    public var sendable: bool = false
    public var c:        bool = false
    public var pos:      Span = Span.none()
  }
  /// A constant where a type argument goes: the `8` of `Array<u8, 8>` (D121).
  struct ConstType {
    public var x:   *Expr
    public var pos: Span = Span.none()
  }
  /// `This` in a trait or an implement.
  struct SelfType {
    public var pos: Span = Span.none()
  }
  /// An associated type: `This.Item`, `I.Item`.
  struct AssocType {
    public var base: *Type
    public var name: Ident = Ident()
    public var pos:  Span = Span.none()
  }
  /// `A | B`: after `throws`, or as a type argument (D45).
  struct ErrorUnionType {
    public var members: List<Type> = []
    public var pos:     Span = Span.none()
  }
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

public sealed trait Decl {
  /// `use a, b.c as d` (M6), `public use` (D89).
  struct UseDecl {
    public var specs: List<UseSpec> = []
    public var pub:   bool = false
    public var pos:   Span = Span.none()
  }
  struct FunDeclNode {
    public var decl: FunDecl
  }
  /// `struct Name<T> : SealedParent { fields; methods }`, an `extern struct`,
  /// an `extern union`, or an `error Name { }`.
  struct StructDecl {
    public var attrs:      List<Attribute> = []
    public var doc:        string = ""
    public var pub:        bool = false
    public var isInternal: bool = false
    public var isExtern:   bool = false
    public var union:      bool = false
    public var isError:    bool = false
    public var errorImpl:  (*ImplDecl)? = null
    public var annotation: AnnotationInfo? = null  // `annotation Name(params) on …` (D174): the parameters are `fields`
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
    // declared in the body of a sealed trait (D193), a variant of it; the
    // `: Trait` clause may have been supplied rather than written
    public var nested:         bool = false
    public var impliedVariant: bool = false
    // `indirect struct`: stored behind a pointer in its sealed value;
    // `inline struct`: kept in the value even in a tree (D198)
    public var indirect: bool = false
    public var isInline: bool = false
    public var pos:      Span = Span.none()
  }
  /// `trait Name { type Item; fun m() }`, or `sealed trait Name`.
  struct TraitDecl {
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
    // the structs declared in the body of a sealed trait (D193); each is also
    // in the file's declarations, as the separate form would be
    public var variants: List<StructDecl> = []
    public var indirect: bool = false  // `indirect sealed trait`: every variant behind a pointer (D198)
    public var pos:      Span = Span.none()
  }
  /// `implement<T> Trait for Type { }`, or `extend<T> Type { }` (D23).
  struct ImplDecl {
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
    public var withFun:    bool = false  // `extend T with fun f() …`, the one-function form (D173)
    public var pub:        bool = false  // `public extend`: the visibility of the methods that carry none
    public var isInternal: bool = false  // `internal extend`, likewise
    public var pos:        Span = Span.none()
  }
  /// A module-level `val`, `var` or `const`.
  struct ValDecl {
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
  struct ExternBlock {
    public var abi:  string = ""
    public var funs: List<FunDecl> = []
    public var pos:  Span = Span.none()
  }
  /// `executor name(args) { state and functions }` (D150).
  struct ExecutorDecl {
    public var doc:        string = ""
    public var pub:        bool = false
    public var isInternal: bool = false
    public var name:       Ident = Ident()
    public var args:       List<Arg> = []
    public var argsPos:    Span = Span.none()  // the parenthesised arguments; none without them
    public var state:      List<ValDecl> = []
    public var funs:       List<FunDecl> = []
    public var hasBody:    bool = false
    public var pos:        Span = Span.none()
  }
  /// `error Name = A | B | C` (D45).
  struct ErrorAliasDecl {
    public var attrs:      List<Attribute> = []
    public var doc:        string = ""
    public var pub:        bool = false
    public var isInternal: bool = false
    public var name:       Ident = Ident()
    public var members:    (*Type)? = null
    public var pos:        Span = Span.none()
  }
  /// `type Name<T> = Type` (D55).
  struct TypeAliasDecl {
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
  struct EnumDecl {
    public var attrs:      List<Attribute> = []
    public var doc:        string = ""
    public var pub:        bool = false
    public var isInternal: bool = false
    public var name:       Ident = Ident()
    public var base:       (*Type)? = null  // null: i64
    public var members:    List<EnumMember> = []
    public var pos:        Span = Span.none()
  }
  /// `test "name" { body }` (D78).
  struct TestDecl {
    public var doc:     string = ""
    public var attrs:   List<Attribute> = []  // `@Skip("why")` (D178)
    public var label:   string = ""           // `test listMethods "…"`: the optional identifier (D170)
    public var labelAt: Span = Span.none()
    public var name:    string = ""         // unquoted; empty when only a label is written
    public var at:      Span = Span.none()  // the sentence's string (the label's when there is none)
    public var body:    Block? = null
    public var pos:     Span = Span.none()
  }
  /// `suite "name" { … }` (D78).
  struct SuiteDecl {
    public var doc:     string = ""
    public var attrs:   List<Attribute> = []  // `@SkipIf(cond, "why")` (D178)
    public var label:   string = ""
    public var labelAt: Span = Span.none()
    public var name:    string = ""
    public var at:      Span = Span.none()
    public var decls:   List<Decl> = []
    public var pos:     Span = Span.none()
  }
  /// `skip(why)` or `skipIf(cond, why)` directly in a suite's body (D178): the
  /// call, evaluated once before the suite's first test.
  struct SkipDecl {
    public var call: *Expr
    public var pos:  Span = Span.none()
  }
  struct StaticAssertDecl {
    public var assert: StaticAssert
  }
  /// A declaration that did not parse.
  struct BadDecl {
    public var pos: Span = Span.none()
  }
}

/// `@name` or `@name(args)` on the line before a declaration.
public struct Attribute {
  public var qual: Ident = Ident()  // `@validate.Min(1)`: the module it is written through (D174); empty when bare
  public var name: Ident = Ident()
  public var args: List<Arg> = []
  public var pos:  Span = Span.none()
}

/// What only `annotation Name(params) on targets for Readers` has (D174); the
/// parameters are the struct's fields.
public struct AnnotationInfo {
  public var targets:    List<Ident> = []  // field, struct, fun, trait, enum, enumMember, variant
  public var readers:    List<Type> = []   // the traits whose reflection reads it
  public var onPos:      Span = Span.none()
  public var readersPos: Span = Span.none()  // the word `for`; none without a reader clause
  public var wrapper:    (*Expr)? = null     // `= retryCall`: the function that wraps what it stands on (D176)
  public var wrapperPos: Span = Span.none()  // the `=`
  public var keywordEnd: bool = false        // the last target is a keyword, after which no newline is inserted
}

/// One import: `a.b`, `a.b as c`, `a.b { f, T as U }` (D85).
public struct UseSpec {
  public var path:  List<Ident> = []
  public var alias: Ident? = null
  public var names: List<UseName>? = null  // the braced names; null without braces
  /// The `extend` clause (D185): the import brings extension functions of the
  /// module (D173). Without it, none; `extendNames` are the ones named in
  /// `extend { f, g }`, null for a bare `extend`: all of them.
  public var extend:      bool = false
  public var extendNames: List<UseName>? = null
  public var pub:         bool = false
  public var pos:         Span = Span.none()
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

public struct AssocTypeDecl {
  public var name:   Ident = Ident()
  public var bounds: List<Type> = []
  public var pos:    Span = Span.none()
}

public struct AssocTypeBinding {
  public var name: Ident = Ident()
  public var typ:  (*Type)? = null
}

public enum BindKind {
  Val
  Var
  Const
}

public fun bindWord(k: BindKind): string => when (k) {
  BindKind.Val   => "val"
  BindKind.Var   => "var"
  BindKind.Const => "const"
}

public struct EnumMember {
  public var attrs: List<Attribute> = []
  public var doc:   string = ""
  public var name:  Ident = Ident()
  public var value: (*Expr)? = null  // null: the previous value plus one
  public var pos:   Span = Span.none()
}

/// What a report calls a test or a suite: the label and the sentence
/// ("listMethods: incrementAt updates an element"), or whichever is written (D170).
public fun title(label: string, name: string): string => if (label.isEmpty()) name else if (name.isEmpty()) label else "$label: $name"

/// `static assert(cond, "why")` (D113), at module level or in a body.
public struct StaticAssert {
  public var cond:   *Expr
  public var reason: (*Expr)? = null  // null when left out (an error)
  public var pos:    Span = Span.none()
}

// ---------------------------------------------------------------------------
// statements

public sealed trait Stmt {
  struct BlockStmt {
    public var block: Block
  }
  struct ValStmt {
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
  struct ExprStmt {
    public var x: *Expr
  }
  /// `target = value`, or a compound `target += value`.
  struct AssignStmt {
    public var target: *Expr
    public var op:     Kind = Kind.Assign
    public var value:  *Expr
    public var pos:    Span = Span.none()
  }
  struct ReturnStmt {
    public var value: (*Expr)? = null
    public var pos:   Span = Span.none()
  }
  /// `throw e` (D4).
  struct ThrowStmt {
    public var value: (*Expr)? = null
    public var pos:   Span = Span.none()
  }
  struct BreakStmt {
    public var label: Ident? = null
    public var pos:   Span = Span.none()
  }
  struct ContinueStmt {
    public var label: Ident? = null
    public var pos:   Span = Span.none()
  }
  /// `loop { }`, `loop (cond) { }`, `loop (x in c) { }`.
  struct LoopStmt {
    public var isInline: bool = false  // `inline loop (f in T.fields)`: unrolled while compiling (D174)
    public var label:    Ident? = null
    public var cond:     (*Expr)? = null
    public var bound:    Binding? = null  // `loop (x in …)`
    public var iter:     (*Expr)? = null
    public var body:     Block = Block()
    public var pos:      Span = Span.none()
  }
  /// The statement form `with x = e` (D100); the rest of its block is its body.
  struct WithStmt {
    public var binding: WithBinding
    public var pos:     Span = Span.none()
  }
  /// `scope { }` (D34); `scope(on: e, workers: n) { }` (D143, D156).
  struct ScopeStmt {
    public var on:      (*Expr)? = null
    public var workers: (*Expr)? = null
    public var body:    Block = Block()
    public var pos:     Span = Span.none()
  }
  /// A local function.
  struct FunStmt {
    public var decl: *FunDecl
  }
  struct StaticAssertStmt {
    public var assert: StaticAssert
  }
  struct BadStmt {
    public var pos: Span = Span.none()
  }
}

public struct Block {
  public var stmts: List<Stmt> = []
  public var pos:   Span = Span.none()
}

/// The left side of `val`: a name or a tuple of bindings (D37).
public struct Binding {
  public var name:  Ident? = null       // a single name
  public var tuple: List<Binding> = []  // destructuring
  public var typ:   (*Type)? = null     // a single name's annotation
  public var ref:   bool = false        // `&x` in a loop head (D42)
  public var pos:   Span = Span.none()
}

/// `{ body }` or `{ e => body }`: a let-else's or a catch's failure branch.
public struct Handler {
  public var err:  Ident? = null
  public var body: Block = Block()
  public var pos:  Span = Span.none()
}

/// `name = e`, or `e` alone (D109): an empty name holds and closes the value
/// without naming it.
public struct WithBinding {
  public var name:  Ident = Ident()
  public var value: *Expr
}

// ---------------------------------------------------------------------------
// expressions

public sealed trait Expr {
  struct IntLit {
    public var text: string = ""
    public var pos:  Span = Span.none()
  }
  struct FloatLit {
    public var text: string = ""
    public var pos:  Span = Span.none()
  }
  /// A string, its interpolation split into parts.
  struct StringLit {
    public var parts: List<StringPart> = []
    public var pos:   Span = Span.none()
  }
  /// `tag"text ${x}"` (D129).
  struct TemplateExpr {
    public var tag: *Expr  // a NameExpr or a MemberExpr
    public var lit: StringLit
    public var pos: Span = Span.none()
  }
  struct CharLit {
    public var value: string = ""
    public var pos:   Span = Span.none()
  }
  /// `/pattern/flags` (D182): a regular expression.
  struct RegexLit {
    public var pattern: string = ""
    public var flags:   string = ""
    public var pos:     Span = Span.none()
  }
  struct BoolLit {
    public var value: bool = false
    public var pos:   Span = Span.none()
  }
  struct NullLit {
    public var pos: Span = Span.none()
  }
  /// `this`.
  struct ThisExpr {
    public var pos: Span = Span.none()
  }
  struct NameExpr {
    public var name:     string = ""
    public var typeArgs: List<Type> = []  // `Name<T>.f(…)`
    public var pos:      Span = Span.none()
  }
  /// `x.name` or `x?.name`.
  struct MemberExpr {
    public var x:        *Expr
    public var name:     Ident = Ident()
    public var safe:     bool = false
    public var typeArgs: List<Type> = []  // `mod.Name<T>.f(…)`
    public var grouped:  bool = false     // in parentheses: ends a `?.` chain (D70)
    public var pos:      Span = Span.none()
  }
  struct IndexExpr {
    public var x:     *Expr
    public var index: *Expr
    public var pos:   Span = Span.none()
  }
  /// `f(args)`, `Type<T>(args)`, `async f(args)`.
  struct CallExpr {
    public var callee:   *Expr
    public var typeArgs: List<Type> = []
    public var args:     List<Arg> = []
    public var isAsync:  bool = false
    public var grouped:  bool = false
    // written `async { body }` (D169): the callee is a lambda of no
    // parameters whose body is that block
    public var block: bool = false
    public var pos:   Span = Span.none()
  }
  struct UnaryExpr {
    public var op:  Kind = Kind.Minus  // Minus, Bang, Amp, Star, Tilde
    public var x:   *Expr
    public var pos: Span = Span.none()
  }
  struct BinaryExpr {
    public var op:    Kind = Kind.Plus
    public var left:  *Expr
    public var right: *Expr
    public var pos:   Span = Span.none()
  }
  /// `x ?! error`.
  struct OrFailExpr {
    public var left:  *Expr
    public var right: *Expr
    public var pos:   Span = Span.none()
  }
  /// `r ?? fallback`, or `r ?? { … }`.
  struct CoalesceExpr {
    public var left:    *Expr
    public var right:   (*Expr)? = null  // null when handler is set
    public var handler: Handler? = null
    public var pos:     Span = Span.none()
  }
  /// `do { body } catch (e) { handler }`, or `x catch (e) { handler }` (D98).
  struct CatchExpr {
    public var body:    Block? = null
    public var x:       (*Expr)? = null
    public var handler: Handler = Handler()
    public var pos:     Span = Span.none()
  }
  /// `x ?: default` (D30).
  struct ElvisExpr {
    public var left:  *Expr
    public var right: *Expr
    public var pos:   Span = Span.none()
  }
  /// `lo..hi` or `lo..<hi` (D29).
  struct RangeExpr {
    public var lo:        *Expr
    public var hi:        *Expr
    public var inclusive: bool = false
    public var pos:       Span = Span.none()
  }
  /// `x => e`, `(a, b) => e`, `(x: i32) => { … }` (D32).
  struct LambdaExpr {
    public var params: List<Param> = []
    public var ret:    (*Type)? = null
    public var body:   *Expr  // a BlockExpr for `{ … }`
    public var pos:    Span = Span.none()
  }
  /// `(a, b)`; `(key: k, value: v)` labels every element (D166), `labels`
  /// empty when none is labelled.
  struct TupleExpr {
    public var elems:  List<Expr> = []
    public var labels: List<Ident> = []
    public var pos:    Span = Span.none()
  }
  /// `[a, b]`, `mut [a, b]` (D25).
  struct ListLit {
    public var elems: List<Expr> = []
    public var isMut: bool = false
    public var pos:   Span = Span.none()
  }
  /// `["k": v]`, `[:]`, `mut [...]`.
  struct MapLit {
    public var entries: List<MapEntry> = []
    public var isMut:   bool = false
    public var pos:     Span = Span.none()
  }
  /// `if (cond) then else other`; a branch is always a block.
  struct IfExpr {
    public var cond:   *Expr
    public var then:   Block = Block()
    public var orElse: Block? = null  // or a block holding one IfExpr: `else if`
    public var pos:    Span = Span.none()
  }
  /// `val name = value` in an `if`'s condition (D95).
  struct LetCond {
    public var name:  Ident = Ident()
    public var value: *Expr
    public var pos:   Span = Span.none()
  }
  /// `when (subject) { arms }`, or `when { }` (D13).
  struct WhenExpr {
    public var subject: (*Expr)? = null
    public var bind:    Ident? = null  // `when (val r = subject)`
    public var arms:    List<WhenArm> = []
    public var pos:     Span = Span.none()
  }
  /// A block where an expression goes: lambda and arm bodies.
  struct BlockExpr {
    public var block: Block
  }
  struct TryExpr {
    public var x:   *Expr
    public var pos: Span = Span.none()
  }
  struct AwaitExpr {
    public var x:   *Expr
    public var pos: Span = Span.none()
  }
  /// `x is T`, `x !is T`.
  struct IsExpr {
    public var x:   *Expr
    public var pat: TypePat
    public var not: bool = false
    public var pos: Span = Span.none()
  }
  /// `x in c`, `x !in c` (D189).
  struct InExpr {
    public var x:   *Expr
    public var c:   *Expr
    public var not: bool = false
    public var pos: Span = Span.none()
  }
  /// `T implements Trait` (D117).
  struct ImplementsExpr {
    public var typ:       *Type
    public var traitType: *Type
    public var pos:       Span = Span.none()
  }
  /// `x as T`.
  struct CastExpr {
    public var x:   *Expr
    public var typ: *Type
    public var pos: Span = Span.none()
  }
  /// `gather { … }` (D36); `gather(on: e, workers: n) { … }` (D143, D156).
  struct GatherExpr {
    public var on:      (*Expr)? = null
    public var workers: (*Expr)? = null
    public var body:    Block = Block()
    public var pos:     Span = Span.none()
  }
  /// A `scope { … }` used as a value: its body's last expression (D156). In
  /// statement position a scope is a ScopeStmt.
  struct ScopeExpr {
    public var on:      (*Expr)? = null
    public var workers: (*Expr)? = null
    public var body:    Block = Block()
    public var pos:     Span = Span.none()
  }
  /// `race { body }` (D169): the tasks the body starts race. `arms` instead of
  /// `body` for the removed `race` with arms, now `select`.
  struct RaceExpr {
    public var body: Block? = null
    public var arms: List<RaceArm> = []
    public var pos:  Span = Span.none()
  }
  /// `select { val x = ch.recv() => body … }` (D169; the arms of D38/D108/D168).
  struct SelectExpr {
    public var arms: List<RaceArm> = []
    public var pos:  Span = Span.none()
  }
  struct UnsafeExpr {
    public var body: Block = Block()
    public var pos:  Span = Span.none()
  }
  /// `with (r = open()) { … }` (D43); a statement through ExprStmt.
  struct WithExpr {
    public var bindings: List<WithBinding> = []
    public var body:     Block = Block()
    public var pos:      Span = Span.none()
  }
  /// `return`, `break` or `continue` where an expression goes: `x ?: return`.
  struct ControlExpr {
    public var stmt: *Stmt
  }
  struct BadExpr {
    public var pos: Span = Span.none()
  }
}

/// `$x` and `${expr}` parts hold the expression; the others the text.
public struct StringPart {
  public var text:    string = ""
  public var expr:    (*Expr)? = null
  public var spec:    string = ""  // `${expr:spec}` (D171), as written
  public var specPos: Span = Span.none()
  public var hasSpec: bool = false
}

/// A call argument, perhaps named (D28).
public struct Arg {
  public var name:   Ident? = null
  public var value:  *Expr
  public var spread: bool = false  // `xs...`
}

public struct MapEntry {
  public var key:   *Expr
  public var value: *Expr
}

public struct WhenArm {
  public var patterns: List<Pattern> = []
  public var cond:     (*Expr)? = null  // the subjectless form
  public var guard:    (*Expr)? = null
  public var isElse:   bool = false
  public var body:     *Expr
  public var pos:      Span = Span.none()
}

public struct RaceArm {
  public var binding: Binding? = null  // `val x = …`
  public var source:  *Expr
  public var name:    Ident? = null    // `src as name =>` (D168, removed by D169)
  public var body:    (*Expr)? = null  // null for a bare arm (D168, removed by D169)
  public var pos:     Span = Span.none()
}

// ---------------------------------------------------------------------------
// patterns (D13)

public sealed trait Pattern {
  /// `_`.
  struct WildcardPat {
    public var pos: Span = Span.none()
  }
  /// A name the matched value is bound to.
  struct BindPat {
    public var name: Ident = Ident()
  }
  /// A constant: `1`, `"x"`, `null`, `true`.
  struct LiteralPat {
    public var value: *Expr
  }
  /// `in lo..hi`.
  struct RangePat {
    public var range: RangeExpr
    public var pos:   Span = Span.none()
  }
  /// `is T` or `is T(field, other: pat)` (D12).
  struct TypePat {
    public var typ:    *Type
    public var fields: List<FieldPat> = []
    public var hasArg: bool = false  // `is T()` rather than `is T`
    public var pos:    Span = Span.none()
  }
  /// `(a, b)`.
  struct TuplePat {
    public var elems: List<Pattern> = []
    public var pos:   Span = Span.none()
  }
  /// `[a, b]`, `[first, ..rest]` (D62).
  struct ListPat {
    public var elems: List<Pattern> = []
    public var pos:   Span = Span.none()
  }
  /// `..` or `..name` in a list pattern.
  struct RestPat {
    public var name: Ident? = null
    public var pos:  Span = Span.none()
  }
}

public struct FieldPat {
  public var name: Ident = Ident()
  public var pat:  (*Pattern)? = null  // null: bind under the field's name
}

// ---------------------------------------------------------------------------
// where a node is: one exhaustive `when` per family

public fun typeSpan(t: Type): Span => when (t) {
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

public fun declSpan(d: Decl): Span => when (d) {
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
  is SkipDecl         => d.pos
  is StaticAssertDecl => d.assert.pos
  is BadDecl          => d.pos
}

public fun stmtSpan(s: Stmt): Span => when (s) {
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

public fun exprSpan(e: Expr): Span => when (e) {
  is IntLit         => e.pos
  is FloatLit       => e.pos
  is StringLit      => e.pos
  is TemplateExpr   => e.pos
  is CharLit        => e.pos
  is RegexLit       => e.pos
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
  is InExpr         => e.pos
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

public fun patternSpan(p: Pattern): Span => when (p) {
  is WildcardPat => p.pos
  is BindPat     => p.name.pos
  is LiteralPat  => exprSpan(*p.value)
  is RangePat    => p.pos
  is TypePat     => p.pos
  is TuplePat    => p.pos
  is ListPat     => p.pos
  is RestPat     => p.pos
}
