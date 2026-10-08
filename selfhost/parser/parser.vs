// The parser (P4, P5): a hand-written
// recursive descent with Pratt-style expressions; a mistake is a diagnostic,
// and the parser resynchronises at the next statement or declaration, so one
// mistake is one message. The harness compares every message with the
// bootstrap compiler's, word for word (gate G3).
//
// It reads the current language only: a spelling Veles has removed is a
// plain syntax error here (user, 2026-10-08).
//
// The other files of this module extend `Parser`: decl.vs, types.vs,
// stmt.vs, expr.vs and pattern.vs.
use ast, lexer { Kind, Token, kindText, tokenize }, source { Diagnostics, File, Span }

/// Parses one file; problems go into `diags`.
public fun parseFile(file: File, diags: *Diagnostics): ast.SourceFile {
  val scan = tokenize(file, diags)
  var parser = Parser(file, toks: scan.tokens.toMutable(), diags)
  var parsed = parser.parseFile()
  parsed.doc = scan.moduleDoc
  parsed
}

// where a function is declared, which decides what it may be
enum FunContext {
  Free
  Method
  Trait
  Extern
}

// what is written before a top-level declaration's keyword
struct DeclHead {
  attrs:      List<ast.Attribute>
  doc:        string
  pub:        bool
  isInternal: bool
  start:      Span

  // whether a visibility word was written, for a declaration that takes none
  fun marked(): bool = this.pub || this.isInternal

  // the word written, as a message quotes it
  fun visibility(): string = if (this.isInternal) "'internal'" else "'public'"
}

struct Parser {
  file:      File
  toks:      MutableList<Token>
  var diags: *Diagnostics
  var pos:   i64 = 0
  // one error per token position: a cascade is suppressed
  var lastErrPos: i64 = -1
  // the `...` that ended the parameter list just parsed (D123)
  var cVariadic: Span = Span.none()
  // in an `error` body a field may have a union type (a cause, D45)
  var errorFields: bool = false
  // in the operand of a `try`: a `catch` after it belongs to the whole
  // `try` (D98); any nested expression clears it
  var inTry: bool = false
  // in a constant type argument (`Array<u8, 4 * 16>`): `>>` closes two lists
  var inTypeArg: bool = false
  // a documentation comment seen on a declaration's attributes
  var leadDoc: string = ""
  // in the head of a `when` or `race` arm: the `=>` is the arm's, so
  // `x > limit => …` is not a lambda; brackets reset it
  var noLambda: i64 = 0

  // ---- tokens

  fun cur(): Token = this.toks.at(this.pos) ?: panic("parser: the cursor is always on a token")

  fun peek(n: i64): Token {
    val at = (this.pos + n).min(this.toks.len() - 1)
    this.toks.at(at) ?: panic("parser: a token list ends with EOF")
  }

  fun at(kind: Kind): bool = this.cur().kind == kind

  fun atAny(kinds: Kind...): bool = kinds.contains(this.cur().kind)

  // the word `text` at the cursor: a contextual keyword such as `extend`
  fun atWord(text: string): bool = this.at(Kind.Ident) && this.cur().text == text

  // the token at the cursor, which the cursor then leaves; EOF stays
  fun next(): Token {
    val t = this.cur()
    if (this.pos < this.toks.len() - 1) this.pos += 1
    t
  }

  // takes a token of this kind if it is there
  fun accept(kind: Kind): bool {
    if (!this.at(kind)) return false
    this.next()
    true
  }

  fun span(): Span = this.cur().span

  fun prevSpan(): Span = (this.toks.at(this.pos - 1) ?: this.cur()).span

  // from start to the end of the previous token
  fun spanFrom(start: Span): Span = start.to(this.prevSpan())

  fun errorAt(span: Span, message: string) {
    if (span.start == this.lastErrPos) return
    this.lastErrPos = span.start
    this.diags.errorAt(span, message)
  }

  fun errorExpected(what: string) {
    this.errorAt(this.span(), "expected $what, found ${this.cur().describe()}")
  }

  // takes a token of this kind, or reports that it is missing; whether it was there
  fun expect(kind: Kind): bool {
    if (this.accept(kind)) return true
    this.errorExpected("'${kindText(kind)}'")
    false
  }

  // the identifier at the cursor, taken; null, reported, when there is none
  fun expectIdent(): ast.Ident? {
    if (!this.at(Kind.Ident)) {
      this.errorExpected("identifier")
      return null
    }
    val t = this.next()
    ast.Ident(name: t.text, pos: t.span)
  }

  // an identifier, or `_` in its place when it is missing (reported)
  fun ident(): ast.Ident = this.expectIdent() ?: ast.Ident(name: "_", pos: this.span())

  fun skipSemis() {
    loop (this.at(Kind.Semi)) this.next()
  }

  // the end of a statement or member: a `;`, or nothing before the `}` that
  // closes the block
  fun expectTerminator() {
    if (this.accept(Kind.Semi) || this.atAny(Kind.RBrace, Kind.EOF)) return
    if (this.at(Kind.Arrow)) {
      this.thinArrow(this.span(), "'->' is not an operator; use '=>' (D33)")
    } else {
      this.errorAt(this.span(), "expected end of statement, found ${this.cur().describe()}")
    }
    this.syncStmt()
  }

  // to the next statement boundary, over nesting
  fun syncStmt() {
    var depth: i64 = 0
    loop (!this.at(Kind.EOF)) {
      when (this.cur().kind) {
        Kind.LBrace, Kind.LParen, Kind.LBracket => depth += 1
        Kind.RBrace => {
          if (depth == 0) return  // closes the enclosing block: the caller's to take
          depth -= 1
        }
        Kind.RParen, Kind.RBracket => {
          if (depth == 0) {
            // a closer nobody will take: skipping it keeps the caller moving
            this.next()
            continue
          }
          depth -= 1
        }
        Kind.Semi => {
          if (depth == 0) {
            this.next()
            return
          }
        }
        else => { }
      }
      this.next()
    }
  }

  // to the next top-level declaration
  fun syncDecl() {
    var depth: i64 = 0
    loop (!this.at(Kind.EOF)) {
      if (depth == 0 && (this.startsDecl() || this.at(Kind.KwType))) return
      when (this.cur().kind) {
        Kind.LBrace => depth += 1
        Kind.RBrace => depth = (depth - 1).max(0)
        else        => { }
      }
      this.next()
    }
  }

  // ---- the file

  fun parseFile(): ast.SourceFile {
    val decls: MutableList<ast.Decl> = []
    loop {
      this.skipSemis()
      if (this.at(Kind.EOF)) break
      val start = this.pos
      val d = this.parseDecl()
      decls.push(d)
      if (d is ast.StructDecl) {
        if (val errorImpl = d.errorImpl) decls.push(*errorImpl)  // `error` is a struct and an implement
        loop (impl in d.impls) decls.push(impl)                  // an `implement Trait { }` in the body
      }
      if (d is ast.BadDecl || this.pos == start) {
        if (this.pos == start) this.next()
        this.syncDecl()
        continue
      }
      if (this.accept(Kind.Semi) || this.at(Kind.EOF)) continue
      if (this.startsDecl()) {
        // a missing terminator (an unbalanced paren kept the newline out);
        // reported, and the next declaration kept
        this.errorAt(this.span(), "expected newline before ${this.cur().describe()}")
      } else {
        this.expectTerminator()
      }
    }
    ast.SourceFile(source: this.file, decls: decls.toList())
  }

  fun startsDecl(): bool =
    this.atErrorDecl() || this.atExtendDecl() || this.atAny(
      Kind.KwFun, Kind.KwStruct, Kind.KwTrait,
      Kind.KwImplement, Kind.KwSealed, Kind.KwPublic, Kind.KwPrivate, Kind.KwInternal, Kind.KwUse, Kind.KwExtern,
      Kind.KwVal, Kind.KwVar, Kind.KwConst, Kind.At,
    )

  fun parseAttributes(): List<ast.Attribute> {
    val attrs: MutableList<ast.Attribute> = []
    if (this.at(Kind.At)) this.leadDoc = this.cur().doc
    loop (this.at(Kind.At)) {
      val start = this.span()
      this.next()
      var attr = ast.Attribute(name: this.ident())
      if (this.at(Kind.LParen)) attr.args = this.parseArgs()
      attr.pos = this.spanFrom(start)
      attrs.push(attr)
      this.skipSemis()
    }
    attrs.toList()
  }

  // the documentation comment above the declaration at the cursor: on its
  // attributes, or on its first token
  fun takeDoc(): string {
    val doc = this.leadDoc
    this.leadDoc = ""
    if (doc.isEmpty()) this.cur().doc else doc
  }

  fun parseDecl(): ast.Decl {
    val attrs = this.parseAttributes()
    val doc = this.takeDoc()
    val start = this.span()
    // visibility (M5): `public` is the package, `internal` the module — what
    // nothing means; `private` belongs to members
    val pub = this.accept(Kind.KwPublic)
    val isInternal = this.accept(Kind.KwInternal)
    if (isInternal && pub) this.errorAt(start, "'public' and 'internal' contradict each other; a declaration has one visibility (M5)")
    if (this.at(Kind.KwPrivate)) {
      this.errorAt(this.span(), "'private' belongs to a member of a struct; a top-level declaration is private to its module unless 'public' (M5)")
      this.next()
    }
    this.parseDeclKind(DeclHead(attrs, doc, pub, isInternal, start))
  }

  fun parseDeclKind(head: DeclHead): ast.Decl {
    when (this.cur().kind) {
      Kind.KwUse => {
        if (head.isInternal && !head.pub) this.errorAt(head.start, "'use' cannot be ${head.visibility()}")
        var d = this.parseUse()
        if (head.pub) {
          // a re-export (D89)
          d.pub = true
          d.pos = head.start.to(d.pos)
          d.specs = d.specs.map(s => {
            var spec = s
            spec.pub = true
            spec
          })
        }
        return d
      }
      Kind.KwFun, Kind.KwUnsafe, Kind.KwStatic => {
        if (!this.atStaticAssert()) return this.topLevelFun(head, this.parseFun(head.attrs, FunContext.Free))
        if (head.marked()) this.errorAt(head.start, "a 'static assert' cannot be ${head.visibility()}; it declares nothing")
        val [first, ..] = head.attrs else return ast.StaticAssertDecl(assert: this.parseStaticAssert())
        this.errorAt(first.pos, "a 'static assert' takes no attributes")
        return ast.StaticAssertDecl(assert: this.parseStaticAssert())
      }
      Kind.KwStruct => return this.parseStruct(head, false)
      Kind.Ident => {
        if (this.atErrorDecl()) return this.parseErrorDecl(head)
        val next = this.peek(1).kind
        if (this.atWord("suite") && next == Kind.String) {
          if (head.marked()) this.errorAt(head.start, "a suite cannot be ${head.visibility()}; it only groups tests")
          return this.parseSuite(head)
        }
        if (this.atWord("test") && next == Kind.String) {
          if (head.marked()) this.errorAt(head.start, "a test cannot be ${head.visibility()}; nothing calls it")
          return this.parseTest(head)
        }
        if (this.atWord("test") && next == Kind.KwFun) {
          this.next()  // test
          var fn = this.parseFun(head.attrs, FunContext.Free)
          fn.isTest = true
          return this.topLevelFun(head, fn)
        }
        if (this.atExtendDecl()) {
          if (head.marked()) this.errorAt(head.start, "'extend' cannot be ${head.visibility()}; mark the methods instead")
          return this.parseImpl(head.attrs, true)
        }
      }
      Kind.KwSealed, Kind.KwTrait => return this.parseTrait(head)
      Kind.KwImplement => {
        if (head.marked()) this.errorAt(head.start, "'implement' cannot be ${head.visibility()}; visibility follows the trait and type")
        return this.parseImpl(head.attrs, false)
      }
      Kind.KwConst => {
        // `const fun` (D113), or a constant
        if (this.peek(1).kind == Kind.KwFun) return this.topLevelFun(head, this.parseFun(head.attrs, FunContext.Free))
        return this.parseValDecl(head)
      }
      Kind.KwVal, Kind.KwVar => return this.parseValDecl(head)
      Kind.KwType => return this.parseTypeAlias(head)
      Kind.KwEnum => return this.parseEnum(head)
      Kind.KwExtern => {
        val next = this.peek(1)
        if (next.kind == Kind.KwStruct) {
          this.next()
          return this.parseStruct(head, true)
        }
        if (next.kind == Kind.Ident && next.text == "union" && this.peek(2).kind == Kind.Ident) {
          // `extern union U { … }` (D120): `union` is a word only here
          this.next()
          return this.externUnion(this.parseStruct(head, true))
        }
        if (next.kind == Kind.String && this.peek(2).kind == Kind.KwFun) return this.parseExportedFun(head)
        return this.parseExternBlock()
      }
      Kind.KwWith => {
        // read whole, so the error is one and the next declaration parses
        this.errorAt(this.span(), "'with' closes its resource when a block ends, and a module has no end; open it inside a function: 'with x = e' or 'with (x = e) { ... }' (D100)")
        this.parseStmt()
        return ast.BadDecl(pos: this.spanFrom(head.start))
      }
      else => { }
    }
    this.errorAt(this.span(), "expected a declaration, found ${this.cur().describe()}")
    ast.BadDecl(pos: this.span())
  }

  // a function at module level, with what its head says
  fun topLevelFun(head: DeclHead, parsed: ast.FunDecl): ast.Decl {
    var fn = parsed
    fn.pub = head.pub
    if (head.isInternal) fn.isInternal = true
    if (!head.doc.isEmpty()) fn.doc = head.doc
    fn.pos = head.start.to(fn.pos)
    ast.FunDeclNode(decl: fn)
  }

  fun externUnion(parsed: ast.StructDecl): ast.Decl {
    var d = parsed
    d.union = true
    if (!d.typeParams.isEmpty() || d.variant != null || !d.methods.isEmpty() || d.init != null || !d.impls.isEmpty() || !d.statics.isEmpty()) {
      this.errorAt(d.name.pos, "an 'extern union' is a C layout: fields only — no type parameters, methods, 'init' or 'implement' (D120)")
    }
    d
  }

  // `use a, b.c as d`: modules, separated by commas; a comma at a line's end
  // goes on to the next line
  fun parseUse(): ast.UseDecl {
    val start = this.span()
    this.next()  // use
    val specs: MutableList<ast.UseSpec> = []
    loop {
      specs.push(this.parseUseSpec() ?: break)
      if (!this.accept(Kind.Comma)) break
      this.skipSemis()
    }
    if (specs.isEmpty()) this.errorAt(start, "'use' needs a module path")
    ast.UseDecl(specs: specs.toList(), pos: this.spanFrom(start))
  }

  fun parseUseSpec(): ast.UseSpec? {
    val start = this.span()
    val path: MutableList<ast.Ident> = [this.expectIdent() ?: return null]
    loop (this.accept(Kind.Dot)) path.push(this.expectIdent() ?: break)
    var s = ast.UseSpec(path: path.toList())
    if (this.accept(Kind.KwAs)) s.alias = this.ident()
    if (this.at(Kind.LBrace)) s.names = this.parseUseNames(s.path)
    s.pos = this.spanFrom(start)
    s
  }

  // the braces of `use m { f, T as U }` (D85)
  fun parseUseNames(path: List<ast.Ident>): List<ast.UseName> {
    val open = this.next()  // {
    this.skipSemis()
    val names: MutableList<ast.UseName> = []
    var starred = false
    loop (!this.atAny(Kind.RBrace, Kind.EOF)) {
      val start = this.span()
      if (this.at(Kind.Star)) {
        this.errorAt(start, "'use ${pathString(path)} { * }' does not exist: name what you use, so a reader sees where each name comes from (D85)")
        starred = true
        this.next()
      } else {
        var n = ast.UseName(name: this.expectIdent() ?: break)
        if (this.accept(Kind.KwAs)) n.alias = this.ident()
        n.pos = this.spanFrom(start)
        names.push(n)
      }
      this.skipSemis()
      if (!this.accept(Kind.Comma)) break
      this.skipSemis()
    }
    this.skipSemis()
    val closer = this.span()
    this.expect(Kind.RBrace)
    if (names.isEmpty() && !starred) {
      this.errorAt(open.span.to(closer), "'use ${pathString(path)} { }' names nothing: list what to import, or write 'use ${pathString(path)}'")
    }
    names.toList()
  }

  // ---- signatures

  fun parseTypeParams(): List<ast.TypeParam> {
    if (!this.at(Kind.Lt)) return []
    val lt = this.next()
    val tps: MutableList<ast.TypeParam> = []
    if (this.atTypeClose()) this.errorAt(lt.span, "empty type parameter list; drop the '<>'")
    loop (!this.atTypeClose()) {
      val isConst = this.at(Kind.KwConst)
      val constAt = if (isConst) this.next().span else Span.none()  // `<const N: i64>` (D121)
      val name = this.expectIdent() ?: break
      var tp = ast.TypeParam(name, isConst, at: constAt)
      if (isConst) {
        if (!this.expect(Kind.Colon)) break
        val of = this.parseType()
        tp.of = &of
        if (!isPlainName(of, "i64")) this.errorAt(ast.typeSpan(of), "a constant parameter is an 'i64': write 'const ${name.name}: i64' (D121)")
      } else if (this.accept(Kind.Colon)) {
        tp.bounds = this.parseBounds()
      }
      tps.push(tp)
      if (!this.accept(Kind.Comma)) break
    }
    this.expectTypeClose()
    tps.toList()
  }

  // `A + B`: the bounds of a type parameter, the supertraits of a trait
  fun parseBounds(): List<ast.Type> {
    val bounds: MutableList<ast.Type> = [this.parseType()]
    loop (this.accept(Kind.Plus)) bounds.push(this.parseType())
    bounds.toList()
  }

  fun parseParams(): List<ast.Param> {
    if (!this.expect(Kind.LParen)) return []
    val params: MutableList<ast.Param> = []
    loop (!this.atAny(Kind.RParen, Kind.EOF)) {
      val start = this.span()
      if (this.at(Kind.Ellipsis)) {
        // C's variadic arguments (D123); the checker says where they are allowed
        this.cVariadic = this.next().span
        if (!this.at(Kind.RParen)) {
          this.errorAt(this.span(), "'...' ends the parameter list: C's variadic arguments come after every named one (D123)")
          this.syncParen()
        }
        break
      }
      // `lazy` is a modifier only in front of a parameter's name (D90)
      val isLazy = this.atWord("lazy") && this.peek(1).kind == Kind.Ident
      val lazyPos = if (isLazy) this.next().span else Span.none()
      val name = this.expectIdent() else {
        this.syncParen()
        break
      }
      var param = ast.Param(name, isLazy, lazyPos)
      if (this.expect(Kind.Colon)) {
        param.typ = &this.parseType()
        param.variadic = this.accept(Kind.Ellipsis)  // `parts: string...`
      }
      if (this.accept(Kind.Assign)) {
        val value = this.parseExpr()
        param.defaultValue = &value
        if (param.variadic) this.errorAt(ast.exprSpan(value), "a variadic parameter cannot have a default; it is empty when no argument is given")
      }
      param.pos = this.spanFrom(start)
      params.push(param)
      if (!this.accept(Kind.Comma)) break
    }
    this.expect(Kind.RParen)
    params.toList()
  }

  // to the `)` of the current list, or a `{` or a statement's end that shows
  // it was never closed
  fun syncParen() {
    var depth: i64 = 0
    loop (!this.at(Kind.EOF)) {
      when (this.cur().kind) {
        Kind.LParen            => depth += 1
        Kind.RParen            => {
          if (depth == 0) return
          depth -= 1
        }
        Kind.LBrace, Kind.Semi => {
          if (depth == 0) return
        }
        else                   => { }
      }
      this.next()
    }
  }

  fun parseEffects(): ast.Effects {
    var eff = ast.Effects()
    if (this.at(Kind.KwSuspends)) {
      eff.suspending = true
      eff.suspendsSpan = this.next().span
    }
    if (this.at(Kind.KwThrows)) {
      eff.throwing = true
      eff.throwsSpan = this.next().span
      if (this.atAny(Kind.Ident, Kind.KwSelfType)) eff.error = &this.parseErrorType()
    }
    if (this.at(Kind.KwSuspends)) {
      this.errorAt(this.span(), "'suspends' must come before 'throws'")
      this.next()
      eff.suspending = true
    }
    eff
  }

  // `throws A, B` in a declaration: error types are joined with '|' (D45);
  // the types after the comma join the union, so the rest parses as written
  fun commaInThrows(effects: ast.Effects): ast.Effects {
    var eff = effects
    loop (eff.error != null && this.at(Kind.Comma) && this.peek(1).kind == Kind.Ident) {
      val comma = this.next()
      this.errorAt(comma.span, "error types are joined with '|', not ','")
      val current = *eff.error
      val members = when (current) {
        is ast.ErrorUnionType => current.members
        else                  => [current]
      }
      val next = this.parseType()
      eff.error = &ast.ErrorUnionType(members: members.concat([next]), pos: ast.typeSpan(current).to(ast.typeSpan(next)))
    }
    eff
  }

  fun parseFun(attrs: List<ast.Attribute>, ctx: FunContext): ast.FunDecl {
    val start = this.span()
    var fn = ast.FunDecl(attrs, doc: this.takeDoc())
    val topLevel = ctx == FunContext.Free || ctx == FunContext.Extern
    // modifiers
    loop {
      when (this.cur().kind) {
        Kind.KwPublic   => fn.pub = true
        Kind.KwOverride => fn.isOverride = true
        Kind.KwPrivate  => {
          fn.isPrivate = true
          if (topLevel) this.errorAt(this.span(), "'private' belongs to a member of a struct; a top-level declaration is private to its module unless 'public' (M5)")
        }
        Kind.KwInternal => fn.isInternal = true
        Kind.KwStatic   => {
          fn.isStatic = true
          if (topLevel) this.errorAt(this.span(), "'static' belongs to a function in a struct, trait, impl or extend body; a top-level function needs no marker")
        }
        Kind.KwUnsafe   => fn.isUnsafe = true
        Kind.KwConst    => fn.isConst = true
        else            => break
      }
      this.next()
    }
    if (!this.expect(Kind.KwFun)) {
      fn.name = ast.Ident(name: "_", pos: this.span())
      fn.pos = this.spanFrom(start)
      return fn
    }
    fn.name = this.ident()
    fn.typeParams = this.parseTypeParams()
    fn.params = this.parseParams()
    if (this.cVariadic.isValid()) {
      fn.cVariadic = true
      fn.variadicAt = this.cVariadic
      this.cVariadic = Span.none()
    }
    if (this.accept(Kind.Colon)) fn.ret = &this.parseType()
    fn.effects = this.commaInThrows(this.parseEffects())
    if (this.at(Kind.LBrace)) {
      fn.body = this.parseBlock()
    } else if (this.accept(Kind.Assign)) {
      fn.exprBody = &this.parseExpr()
    } else if (ctx == FunContext.Free || ctx == FunContext.Method) {
      this.errorAt(this.span(), "function '${fn.name.name}' needs a body: '{ ... }' or '= expr'")
    }
    if (ctx == FunContext.Extern && (fn.body != null || fn.exprBody != null)) {
      this.errorAt(fn.name.pos, "extern function '${fn.name.name}' cannot have a body")
    }
    fn.pos = this.spanFrom(start)
    fn
  }

  // between the members of a struct or a trait: a newline, `;` or `,`
  fun parseMemberSeparator() {
    if (this.atAny(Kind.Semi, Kind.Comma)) {
      this.next()
      this.skipSemis()
    } else if (!this.atAny(Kind.RBrace, Kind.EOF)) {
      this.errorAt(this.span(), "expected newline or ',' between members, found ${this.cur().describe()}")
      this.syncStmt()
    }
  }

  // `static assert(` starts a compile-time assertion (D113)
  fun atStaticAssert(): bool =
    this.at(Kind.KwStatic) && this.peek(1).kind == Kind.Ident && this.peek(1).text == "assert" && this.peek(2).kind == Kind.LParen

  fun parseStaticAssert(): ast.StaticAssert {
    val start = this.span()
    this.next()  // static
    this.next()  // assert
    this.next()  // (
    var d = ast.StaticAssert(cond: &this.parseExpr())
    if (this.accept(Kind.Comma) && !this.at(Kind.RParen)) {
      d.reason = &this.parseExpr()
      this.accept(Kind.Comma)
    } else {
      this.errorAt(this.span(), "a 'static assert' needs a reason, said when it fails: 'static assert(cond, \"why it must hold\")' (D113)")
    }
    this.expect(Kind.RParen)
    d.pos = this.spanFrom(start)
    d
  }

  // the end of a header that may stand without a body
  fun expectHeaderEnd(what: string) {
    if (!this.atAny(Kind.Semi, Kind.RBrace, Kind.EOF)) this.errorExpected(what)
  }

  // `extend Type {` or `extend<T> Type {`: `extend` is a word only there
  fun atExtendDecl(): bool {
    val next = this.peek(1).kind
    this.atWord("extend") && (next == Kind.Ident || next == Kind.Lt || next == Kind.Star || next == Kind.LParen)
  }

  // `error Name`: `error` is a word only at declaration position, with a name after it
  fun atErrorDecl(): bool = this.atWord("error") && this.peek(1).kind == Kind.Ident

  // `->` written for `=>`: reported even where another error stands
  fun thinArrow(span: Span, message: string) {
    this.diags.errorAt(span, message)
  }
}

// a dotted path, for messages
fun pathString(path: List<ast.Ident>): string = path.map(seg => seg.name).join(".")

// a one-segment name with no type arguments: `string`, `i64`
fun isPlainName(t: ast.Type, name: string): bool = when (t) {
  is ast.NamedType => t.args.isEmpty() && t.path.len() == 1 && t.path.all(seg => seg.name == name)
  else             => false
}
