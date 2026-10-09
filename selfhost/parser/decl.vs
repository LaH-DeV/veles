// Declarations.
use ast
use lexer { Kind, Token }
use source { Span }

extend Parser {
  fun parseStruct(head: DeclHead, isExtern: bool): ast.StructDecl {
    this.next()  // struct
    var declaration: ast.StructDecl = ast.StructDecl(attrs: head.attrs, doc: head.doc, pub: head.pub, isInternal: head.isInternal, isExtern)
    declaration.name = this.ident()
    declaration.typeParams = this.parseTypeParams()
    if (this.accept(Kind.Colon)) declaration.variant = &this.parseType()
    if (!this.accept(Kind.LBrace)) {
      this.expectHeaderEnd("'{' after struct name")
      declaration.pos = this.spanFrom(head.start)
      return declaration
    }
    this.skipSemis()
    val fields: MutableList<ast.Field> = []
    val methods: MutableList<ast.FunDecl> = []
    val statics: MutableList<ast.ValDecl> = []
    val impls: MutableList<ast.ImplDecl> = []
    loop (!this.atAny(Kind.RBrace, Kind.EOF)) {
      val memberAttrs = this.parseAttributes()
      // a member may start with a visibility word (M5); what follows says
      // whether it is a field, a static or a method
      val vis = this.cur().kind
      val hasVis = vis == Kind.KwPublic || vis == Kind.KwPrivate || vis == Kind.KwInternal
      val at: i64 = if (hasVis) 1 else 0
      val member = this.peek(at)
      val afterMember = this.peek(at + 1).kind
      when (member.kind) {
        Kind.Ident, Kind.KwVar, Kind.KwVal, Kind.KwProtected => {
          // `init { }`, `init(value: T) { }`; a field named init is `init: T`
          if (member.text == "init" && (afterMember == Kind.LBrace || afterMember == Kind.LParen)) {
            declaration = this.parseInit(declaration, hasVis)
          } else {
            fields.push(this.parseField(memberAttrs, vis))
          }
        }
        Kind.KwStatic => {
          if (afterMember == Kind.KwVal || afterMember == Kind.KwVar) {
            if (hasVis) this.next()
            if (vis == Kind.KwPrivate) this.errorAt(this.prevSpan(), "a static value cannot be 'private'; it is the module's unless 'public' (D23)")
            statics.push(this.parseStaticVal(memberAttrs, vis == Kind.KwPublic, vis == Kind.KwInternal))
          } else {
            methods.push(this.parseFun(memberAttrs, FunContext.Method))
          }
        }
        Kind.KwFun, Kind.KwOverride, Kind.KwUnsafe, Kind.KwConst => methods.push(this.parseFun(memberAttrs, FunContext.Method))
        Kind.KwImplement => {
          if (hasVis) {
            this.errorAt(this.span(), "an inline 'implement' has no visibility of its own; it follows the trait and type")
            this.next()
          }
          impls.push(this.parseInlineImpl(memberAttrs, declaration.name, declaration.typeParams))
        }
        else => {
          this.errorAt(member.span, "expected a field or method, found ${member.describe()}")
          this.syncStmt()
          continue
        }
      }
      this.parseMemberSeparator()
    }
    this.expect(Kind.RBrace)
    declaration.fields = fields.toList()
    declaration.methods = methods.toList()
    declaration.statics = statics.toList()
    declaration.impls = impls.toList()
    declaration.pos = this.spanFrom(head.start)
    declaration
  }

  // the `init` block of a struct, the cursor on its visibility word (an
  // error) or on `init`
  fun parseInit(parsed: ast.StructDecl, hasVis: bool): ast.StructDecl {
    var declaration = parsed
    if (hasVis) {
      this.errorAt(this.span(), "'init' has no visibility: it is not callable, it runs at every construction (D28)")
      this.next()
    }
    if (declaration.init != null) this.errorAt(this.span(), "a struct has one 'init' block")
    declaration.initPos = this.span()
    this.next()  // init
    if (this.at(Kind.LParen)) {
      val params = this.parseParams()
      declaration.initParams = if (params.isEmpty()) null else params  // `init()` takes none
      if (this.cVariadic.isValid()) {
        this.errorAt(this.cVariadic, "only a function in an 'extern \"C\"' block takes C's variadic arguments; an 'init' takes 'name: T...' (D123)")
        this.cVariadic = Span.none()
      }
      loop (param in params) {
        if (param.typ == null) this.errorAt(param.name.pos, "an 'init' parameter needs a type: '${param.name.name}: T'")
      }
    }
    declaration.init = this.parseBlock()
    declaration
  }

  // `static val name[: T] = expr` in a struct body: read as `Type.name`
  fun parseStaticVal(attrs: List<ast.Attribute>, pub: bool, isInternal: bool): ast.ValDecl {
    val start = this.span()
    val doc = this.takeDoc()
    this.next()  // static
    if (this.at(Kind.KwVar)) this.errorAt(this.span(), "a static member is a 'val': a type's constants do not change (a mutable global belongs at module level)")
    this.next()  // val / var
    var d: ast.ValDecl = ast.ValDecl(attrs, doc, pub, isInternal, kind: ast.BindKind.Val)
    d.name = this.ident()
    if (this.accept(Kind.Colon)) d.typ = &this.parseType()
    if (this.expect(Kind.Assign)) d.value = &this.parseExpr()
    d.pos = this.spanFrom(start)
    d
  }

  // `[public|internal|private] [protected] [val|var] name: T [= default]`,
  // the visibility word `vis` (if any) still at the cursor
  fun parseField(attrs: List<ast.Attribute>, vis: Kind): ast.Field {
    val start = this.span()
    val doc = this.takeDoc()
    val pub = vis == Kind.KwPublic
    val isPrivate = vis == Kind.KwPrivate
    val isInternal = vis == Kind.KwInternal
    if (pub || isPrivate || isInternal) this.next()
    var f = ast.Field(attrs, pub, isInternal, isPrivate, doc)
    // mutability (D22)
    if (this.at(Kind.KwProtected)) {
      val protectedAt = this.next().span
      f.isProtected = true
      if (isPrivate) {
        this.errorAt(protectedAt, "'private protected' is redundant: a private field is already assignable only by the type; write 'private var' (D22)")
      } else if (!this.at(Kind.KwVar)) {
        this.errorAt(protectedAt, "'protected' qualifies 'var': a bare field is never assigned, so there is nothing to protect; write 'protected var ${this.cur().text}' (D22)")
      }
    }
    f.isVar = this.accept(Kind.KwVar)
    if (!f.isVar) f.isVal = this.accept(Kind.KwVal)
    if (this.atAny(Kind.KwVar, Kind.KwVal, Kind.KwProtected)) {
      this.errorAt(this.span(), "a field is 'val' (the default), 'var' or 'protected var' (D22)")
      this.next()
    }
    f.name = this.ident()
    if (this.expect(Kind.Colon)) f.typ = &(if (this.errorFields) this.parseErrorType() else this.parseType())
    if (this.accept(Kind.Assign)) f.defaultValue = &this.parseExpr()
    f.pos = this.spanFrom(start)
    f
  }

  fun parseTrait(head: DeclHead): ast.Decl {
    var d: ast.TraitDecl = ast.TraitDecl(attrs: head.attrs, doc: head.doc, pub: head.pub, isInternal: head.isInternal)
    d.isSealed = this.accept(Kind.KwSealed)
    if (!this.expect(Kind.KwTrait)) return ast.BadDecl(pos: this.span())
    d.name = this.ident()
    d.typeParams = this.parseTypeParams()
    if (this.accept(Kind.Colon)) d.supers = this.parseBounds()
    if (this.accept(Kind.LBrace)) {
      this.skipSemis()
      val assocTypes: MutableList<ast.AssocTypeDecl> = []
      val methods: MutableList<ast.FunDecl> = []
      loop (!this.atAny(Kind.RBrace, Kind.EOF)) {
        val memberAttrs = this.parseAttributes()
        if (this.at(Kind.KwType)) {
          val typeStart = this.next().span
          var assoc = ast.AssocTypeDecl(name: this.ident())
          if (this.accept(Kind.Colon)) assoc.bounds = this.parseBounds()
          assoc.pos = this.spanFrom(typeStart)
          assocTypes.push(assoc)
        } else if (this.atAny(Kind.KwFun, Kind.KwUnsafe, Kind.KwPublic, Kind.KwInternal, Kind.KwStatic)) {
          methods.push(this.parseFun(memberAttrs, FunContext.Trait))
        } else {
          this.errorAt(this.span(), "expected 'type' or 'fun' in trait body, found ${this.cur().describe()}")
          this.syncStmt()
          continue
        }
        this.parseMemberSeparator()
      }
      this.expect(Kind.RBrace)
      d.assocTypes = assocTypes.toList()
      d.methods = methods.toList()
    }
    d.pos = this.spanFrom(head.start)
    d
  }

  // `implement<T> Trait for Type { … }`, or `extend<T> Type { … }`
  fun parseImpl(attrs: List<ast.Attribute>, extend: bool): ast.ImplDecl {
    val start = this.span()
    this.next()  // implement / extend
    var d: ast.ImplDecl = ast.ImplDecl(attrs, extend)
    d.typeParams = this.parseTypeParams()
    if (extend) {
      d.target = &this.parseType()
      if (this.at(Kind.KwFor)) {
        this.errorAt(this.span(), "'extend' names the type being extended, not a trait; use 'implement Trait for Type'")
        this.next()
        d.target = &this.parseType()
      }
    } else {
      val traitType = this.parseType()
      d.traitType = &traitType
      if (this.accept(Kind.KwFor)) {
        d.target = &this.parseType()
      } else {
        this.errorAt(this.span(), "expected 'for' after trait name in impl; inherent methods go inside the struct body (D23)")
        d.target = &traitType
      }
    }
    d = this.parseImplBody(d)
    d.pos = this.spanFrom(start)
    d
  }

  // `implement Trait { … }` in a struct's body (D23): for the struct, with
  // its type parameters
  fun parseInlineImpl(attrs: List<ast.Attribute>, structName: ast.Ident, typeParams: List<ast.TypeParam>): ast.ImplDecl {
    val start = this.span()
    this.next()  // implement
    var d: ast.ImplDecl = ast.ImplDecl(attrs, inline: true, typeParams)
    if (this.at(Kind.Lt)) {
      this.errorAt(this.span(), "an impl inside a struct body uses the struct's type parameters; for other bounds write 'implement<T: ...> Trait for ${structName.name}<T>' at top level")
      this.parseTypeParams()
    }
    d.traitType = &this.parseType()
    if (this.at(Kind.KwFor)) {
      this.errorAt(this.span(), "an impl inside a struct body is for that struct; drop 'for'")
      this.next()
      this.parseType()
    }
    d.target = &selfType(structName, typeParams)
    d = this.parseImplBody(d)
    d.pos = this.spanFrom(start)
    d
  }

  // the `{ type … = …; fun … }` of an implement or an extend; a trait's
  // implement may have none, which asks for a derived one (D58)
  fun parseImplBody(header: ast.ImplDecl): ast.ImplDecl {
    var d = header
    if (!d.extend && !this.at(Kind.LBrace)) {
      d.braceless = true
      this.expectHeaderEnd("'{' or the end of the line after the impl header")
      return d
    }
    if (!this.expect(Kind.LBrace)) return d
    this.skipSemis()
    val assocTypes: MutableList<ast.AssocTypeBinding> = []
    val methods: MutableList<ast.FunDecl> = []
    loop (!this.atAny(Kind.RBrace, Kind.EOF)) {
      val memberAttrs = this.parseAttributes()
      if (this.at(Kind.KwType)) {
        if (d.extend) this.errorAt(this.span(), "'extend' blocks add methods only; associated types belong to trait impls")
        this.next()
        var binding = ast.AssocTypeBinding(name: this.ident())
        if (this.expect(Kind.Assign)) binding.typ = &this.parseType()
        assocTypes.push(binding)
      } else if (this.atAny(Kind.KwFun, Kind.KwOverride, Kind.KwUnsafe, Kind.KwConst, Kind.KwPublic, Kind.KwPrivate, Kind.KwInternal, Kind.KwStatic)) {
        methods.push(this.parseFun(memberAttrs, FunContext.Method))
      } else {
        val expected = if (d.extend) "'fun' in extend body" else "'type' or 'fun' in impl body"
        this.errorAt(this.span(), "expected $expected, found ${this.cur().describe()}")
        this.syncStmt()
        continue
      }
      this.parseMemberSeparator()
    }
    this.expect(Kind.RBrace)
    d.assocTypes = assocTypes.toList()
    d.methods = methods.toList()
    d
  }

  fun parseValDecl(head: DeclHead): ast.ValDecl {
    var d: ast.ValDecl = ast.ValDecl(attrs: head.attrs, doc: head.doc, pub: head.pub, isInternal: head.isInternal)
    d.kind = bindKind(this.next().kind)
    d.name = this.ident()
    if (this.accept(Kind.Colon)) d.typ = &this.parseType()
    if (this.expect(Kind.Assign)) d.value = &this.parseExpr()
    d.pos = this.spanFrom(head.start)
    d
  }

  fun parseExternBlock(): ast.ExternBlock {
    val start = this.span()
    this.next()  // extern
    var d: ast.ExternBlock = ast.ExternBlock(abi: "C")
    if (this.at(Kind.String)) {
      val t = this.next()
      when (t.parts) {
        [part] => if (!part.isExpr) d.abi = part.text
        else   => { }
      }
      if (t.parts.len() != 1 || d.abi != "C") this.errorAt(t.span, "unsupported ABI ${ast.quote(d.abi)}; only \"C\" is defined")
    } else {
      this.errorExpected("ABI string (\"C\") after 'extern'")
    }
    if (this.expect(Kind.LBrace)) {
      this.skipSemis()
      val funs: MutableList<ast.FunDecl> = []
      loop (!this.atAny(Kind.RBrace, Kind.EOF)) {
        val memberAttrs = this.parseAttributes()
        if (!this.atAny(Kind.KwFun, Kind.KwPublic, Kind.KwUnsafe)) {
          this.errorAt(this.span(), "expected 'fun' in extern block, found ${this.cur().describe()}")
          this.syncStmt()
          continue
        }
        var fn = this.parseFun(memberAttrs, FunContext.Extern)
        fn.isExtern = true
        funs.push(fn)
        this.parseMemberSeparator()
      }
      this.expect(Kind.RBrace)
      d.funs = funs.toList()
    }
    d.pos = this.spanFrom(start)
    d
  }

  // `extern "C" fun name(…): R { body }`: a Veles function C calls (D69)
  fun parseExportedFun(head: DeclHead): ast.Decl {
    val externDoc = this.takeDoc()  // a doc comment sits on `extern`, the first token
    this.next()                     // extern
    val abi = this.next()
    val isC = when (abi.parts) {
      [part] => !part.isExpr && part.text == "C"
      else   => false
    }
    if (!isC) this.errorAt(abi.span, "unsupported ABI; only \"C\" is defined")
    var fn = this.parseFun(head.attrs, FunContext.Free)
    fn.exportC = true
    fn.pub = fn.pub || head.pub
    if (head.isInternal) fn.isInternal = true
    fn.pos = head.start.to(fn.pos)
    if (!head.doc.isEmpty()) {
      fn.doc = head.doc
    } else if (fn.doc.isEmpty()) {
      fn.doc = externDoc
    }
    if (fn.body == null && fn.exprBody == null) {
      this.errorAt(fn.name.pos, "an 'extern \"C\" fun' is a Veles function C calls, so it needs a body; a C function Veles calls is declared in an 'extern \"C\" { }' block")
    }
    ast.FunDeclNode(decl: fn)
  }

  // `test "name" { body }` (D78): `test` with a string after it
  fun parseTest(head: DeclHead): ast.Decl {
    val doc = this.takeDoc()
    this.next()  // test
    val t = this.next()
    var d: ast.TestDecl = ast.TestDecl(doc: if (doc.isEmpty()) head.doc else doc, name: this.fixedName(t, "test", "test \"parses an empty list\" { ... }"), at: t.span)
    if (val first = head.attrs.at(0)) this.errorAt(first.pos, "a test takes no attributes")
    if (this.at(Kind.LBrace)) {
      d.body = this.parseBlock()
    } else {
      this.errorAt(this.span(), "expected '{' to open the test's body, found ${this.cur().describe()}")
    }
    d.pos = this.spanFrom(head.start)
    d
  }

  // a test's or a suite's name: plain text, neither interpolated nor blank
  fun fixedName(t: Token, what: string, example: string): string {
    var name = ""
    loop (part in t.parts) {
      if (part.isExpr) {
        this.errorAt(t.span, "a $what's name is fixed text; '\${...}' has nothing to interpolate here")
        return name
      }
      name += part.text
    }
    if (name.trim().isEmpty()) this.errorAt(t.span, "a $what needs a name that says what it checks: $example")
    name
  }

  // `suite "name" { … }` (D78): tests, suites and `test fun` helpers
  fun parseSuite(head: DeclHead): ast.Decl {
    val doc = this.takeDoc()
    this.next()  // suite
    val t = this.next()
    var d: ast.SuiteDecl = ast.SuiteDecl(doc: if (doc.isEmpty()) head.doc else doc, name: this.fixedName(t, "suite", "suite \"the router\" { ... }"), at: t.span)
    if (val first = head.attrs.at(0)) this.errorAt(first.pos, "a suite takes no attributes")
    if (this.expect(Kind.LBrace)) d.decls = this.parseSuiteBody()
    d.pos = this.spanFrom(head.start)
    d
  }

  // a suite's declarations, through its `}`
  fun parseSuiteBody(): List<ast.Decl> {
    val decls: MutableList<ast.Decl> = []
    loop {
      this.skipSemis()
      if (this.atAny(Kind.RBrace, Kind.EOF)) break
      val before = this.pos
      val inner = this.parseDecl()
      when (inner) {
        is ast.TestDecl, is ast.SuiteDecl => decls.push(inner)
        is ast.FunDeclNode => {
          if (!inner.decl.isTest) {
            this.errorAt(inner.decl.name.pos, "a suite holds tests, suites and helpers; a helper is 'test fun ${inner.decl.name.name}', and a function the program uses belongs outside the suite")
          }
          decls.push(inner)
        }
        is ast.BadDecl => { }
        else => this.errorAt(ast.declSpan(inner), "a suite holds tests ('test \"...\" { }'), 'test fun' helpers and other suites")
      }
      if (this.pos == before) this.next()
    }
    this.expect(Kind.RBrace)
    decls.toList()
  }

  // `error Name<T> { fields; methods }` (D4): a struct plus `implement Error
  // for Name`; `message` goes to the implement, the rest stay the struct's
  fun parseErrorDecl(head: DeclHead): ast.Decl {
    val kw = this.span()
    if (this.peek(2).kind == Kind.Assign) return this.parseErrorAlias(head)
    this.errorFields = true
    var d = this.parseStruct(head, false)
    this.errorFields = false
    d.isError = true
    if (d.variant != null) this.errorAt(kw, "an 'error' cannot be a variant of a sealed trait; declare it with 'struct'")
    val inherent: MutableList<ast.FunDecl> = []
    val implemented: MutableList<ast.FunDecl> = []
    loop (m in d.methods) {
      if (m.name.name == "message") {
        var message = m
        message.isOverride = true
        implemented.push(message)
      } else {
        if (m.isOverride) this.errorAt(m.name.pos, "only 'message' can be overridden in an error; '${m.name.name}' is an ordinary method")
        inherent.push(m)
      }
    }
    // a `message: string` field is the message: the method reads it
    if (implemented.isEmpty()) {
      loop (f in d.fields) {
        val typ = f.typ ?: continue
        if (f.name.name == "message" && isPlainName(*typ, "string")) implemented.push(messageFromField(f.name.pos))
      }
    }
    d.methods = inherent.toList()
    d.errorImpl = &ast.ImplDecl(
      errorSugar: true,
      typeParams: d.typeParams,
      traitType: &ast.NamedType(path: [ast.Ident(name: "Error", pos: kw)], pos: kw),
      target: &selfType(d.name, d.typeParams),
      methods: implemented.toList(),
      pos: d.pos,
    )
    d
  }

  // `error Name = A | B | C` (D45)
  fun parseErrorAlias(head: DeclHead): ast.Decl {
    this.next()  // error
    var d: ast.ErrorAliasDecl = ast.ErrorAliasDecl(attrs: head.attrs, doc: head.doc, pub: head.pub, isInternal: head.isInternal)
    d.name = this.ident()
    this.expect(Kind.Assign)
    d.members = &this.parseErrorType()
    d.pos = this.spanFrom(head.start)
    d
  }

  // `enum Name [: Base] { A = 1, B, C }` (D57)
  fun parseEnum(head: DeclHead): ast.Decl {
    this.next()  // enum
    var d: ast.EnumDecl = ast.EnumDecl(attrs: head.attrs, doc: head.doc, pub: head.pub, isInternal: head.isInternal)
    d.name = this.ident()
    if (this.at(Kind.Lt)) {
      this.errorAt(this.span(), "an enum is not generic: it is a closed set of values of one integer type (D57)")
      this.parseTypeParams()
    }
    if (this.accept(Kind.Colon)) d.base = &this.parseType()
    if (!this.accept(Kind.LBrace)) {
      this.expectHeaderEnd("'{' after enum name")
      d.pos = this.spanFrom(head.start)
      return d
    }
    this.skipSemis()
    val members: MutableList<ast.EnumMember> = []
    loop (!this.atAny(Kind.RBrace, Kind.EOF)) {
      val memberStart = this.span()
      val memberAttrs = this.parseAttributes()
      val doc = this.takeDoc()
      if (!this.at(Kind.Ident)) {
        this.errorAt(this.span(), "expected an enum member name, found ${this.cur().describe()}")
        this.syncStmt()
        continue
      }
      var m = ast.EnumMember(attrs: memberAttrs, doc, name: this.ident())
      if (this.accept(Kind.Assign)) m.value = &this.parseExpr()
      m.pos = this.spanFrom(memberStart)
      members.push(m)
      this.parseMemberSeparator()
    }
    this.expect(Kind.RBrace)
    d.members = members.toList()
    d.pos = this.spanFrom(head.start)
    d
  }

  // `type Name<T> = Type` (D55)
  fun parseTypeAlias(head: DeclHead): ast.Decl {
    this.next()  // type
    var d: ast.TypeAliasDecl = ast.TypeAliasDecl(attrs: head.attrs, doc: head.doc, pub: head.pub, isInternal: head.isInternal)
    d.name = this.ident()
    d.typeParams = this.parseTypeParams()
    if (this.expect(Kind.Assign)) {
      d.typ = &this.parseType()
    } else {
      this.syncStmt()
    }
    d.pos = this.spanFrom(head.start)
    d
  }
}

// `Name<T, U>` for a struct and its type parameters, at the name
fun selfType(name: ast.Ident, typeParams: List<ast.TypeParam>): ast.Type {
  val args: List<ast.Type> = typeParams.map(tp => ast.NamedType(path: [tp.name], pos: tp.name.pos))
  ast.NamedType(path: [name], args, pos: name.pos)
}

// `override fun message(): string = this.message`, for an error whose
// `message: string` field is at pos
fun messageFromField(pos: Span): ast.FunDecl => ast.FunDecl(
  isOverride: true,
  name: ast.Ident(name: "message", pos),
  ret: &ast.NamedType(path: [ast.Ident(name: "string", pos)], pos),
  exprBody: &ast.MemberExpr(x: &ast.ThisExpr(pos), name: ast.Ident(name: "message", pos), pos),
  pos,
)

// what a `val`, `var` or `const` keyword declares
fun bindKind(keyword: Kind): ast.BindKind => when (keyword) {
  Kind.KwVar   => ast.BindKind.Var
  Kind.KwConst => ast.BindKind.Const
  else         => ast.BindKind.Val
}
