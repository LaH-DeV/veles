// Statements and blocks.
use ast
use lexer { Kind }
use source { Span }

extend Parser {
  // `{ statements }`
  fun parseBlock(): ast.Block {
    val start = this.span()
    if (!this.expect(Kind.LBrace)) return ast.Block(pos: start)
    val stmts: MutableList<ast.Stmt> = []
    loop {
      this.skipSemis()
      if (this.atAny(Kind.RBrace, Kind.EOF)) break
      val before = this.pos
      stmts.push(this.parseStmt())
      if (this.pos == before) {
        this.errorAt(this.span(), "unexpected ${this.cur().describe()}")
        this.next()
        this.syncStmt()
        continue
      }
      this.expectTerminator()
    }
    this.expect(Kind.RBrace)
    ast.Block(stmts: stmts.toList(), pos: this.spanFrom(start))
  }

  // a `{ block }`, or one statement wrapped in a block
  fun parseBodyOrStmt(): ast.Block {
    if (this.at(Kind.LBrace)) return this.parseBlock()
    // the body may start on the next line (`if (c)` newline `a`), as in Kotlin
    if (this.at(Kind.Semi) && this.cur().autoSemi && this.peek(1).kind != Kind.RBrace && this.peek(1).kind != Kind.EOF) {
      this.next()
      if (this.at(Kind.LBrace)) return this.parseBlock()
    }
    val start = this.span()
    if (this.atWithStmt()) {
      this.errorAt(this.span(), "'with x = e' closes x when its block ends, and a body without braces has nothing after it; write braces around the body, or the block form 'with (x = e) { ... }' (D100)")
    }
    val s = this.parseStmt()
    ast.Block(stmts: [s], pos: this.spanFrom(start))
  }

  fun parseBinding(): ast.Binding {
    val start = this.span()
    if (this.accept(Kind.LParen)) {
      val tuple: MutableList<ast.Binding> = []
      loop (!this.atAny(Kind.RParen, Kind.EOF)) {
        tuple.push(this.parseBinding())
        if (!this.accept(Kind.Comma)) break
      }
      this.expect(Kind.RParen)
      if (tuple.len() < 2) this.errorAt(this.spanFrom(start), "tuple destructuring needs at least two names")
      return ast.Binding(tuple: tuple.toList(), pos: this.spanFrom(start))
    }
    if (this.at(Kind.Under)) {
      val t = this.next()
      return ast.Binding(name: ast.Ident(name: "_", pos: t.span), pos: t.span)
    }
    // `&x`: the loop variable is the element in place (D42)
    val isRef = this.accept(Kind.Amp)
    var b = ast.Binding(name: this.ident(), ref: isRef)
    if (this.accept(Kind.Colon)) b.typ = &this.parseType()
    b.pos = this.spanFrom(start)
    b
  }

  fun parseStmt(): ast.Stmt {
    val start = this.span()
    if (this.atStaticAssert()) return ast.StaticAssertStmt(assert: this.parseStaticAssert())
    when (this.cur().kind) {
      Kind.KwVal, Kind.KwVar, Kind.KwConst => return this.parseValStmt(start)
      Kind.KwReturn => {
        this.next()
        var s: ast.ReturnStmt = ast.ReturnStmt()
        // a bare `return` also ends at a closing bracket or a comma, where it
        // stands as an expression: `!(return)`, `f(x ?: return)`
        if (!this.atAny(Kind.Semi, Kind.RBrace, Kind.EOF, Kind.RParen, Kind.RBracket, Kind.Comma)) s.value = &this.parseExpr()
        s.pos = this.spanFrom(start)
        return s
      }
      Kind.KwThrow => {
        this.next()
        var s: ast.ThrowStmt = ast.ThrowStmt()
        if (this.atAny(Kind.Semi, Kind.RBrace, Kind.EOF)) {
          this.errorAt(this.span(), "'throw' needs an error value, e.g. 'throw ParseError(text: s)'")
        } else {
          s.value = &this.parseExpr()
        }
        s.pos = this.spanFrom(start)
        return s
      }
      Kind.KwBreak, Kind.KwContinue => {
        val isBreak = this.next().kind == Kind.KwBreak
        var label: ast.Ident? = null
        if (this.at(Kind.Ident)) {
          val t = this.next()
          label = ast.Ident(name: t.text, pos: t.span)
        }
        if (isBreak) return ast.BreakStmt(label, pos: this.spanFrom(start))
        return ast.ContinueStmt(label, pos: this.spanFrom(start))
      }
      Kind.KwLoop => return this.parseLoop(start)
      Kind.KwFor => {
        this.errorAt(this.span(), "there is no 'for'; every loop is spelled 'loop': 'loop (x in xs)', 'loop (cond)' or 'loop { }'")
        return this.parseLoop(start)
      }
      Kind.KwWith => {
        if (this.atWithStmt()) return this.parseWithStmt()
        return ast.ExprStmt(x: &this.parseWith())
      }
      Kind.KwScope => {
        this.next()
        val on = this.parseOn("scope")
        val body = this.parseBlock()
        return ast.ScopeStmt(on, body, pos: this.spanFrom(start))
      }
      Kind.KwFun => return ast.FunStmt(decl: &this.parseFun([], FunContext.Free))
      Kind.KwPublic, Kind.KwStruct, Kind.KwTrait, Kind.KwImplement, Kind.KwUse, Kind.KwSealed => {
        this.errorAt(this.span(), "${this.cur().describe()} is only allowed at module level")
        this.syncStmt()
        return ast.BadStmt(pos: start)
      }
      else => { }
    }
    if (this.atErrorDecl() || this.atExtendDecl()) {
      this.errorAt(this.span(), "'${this.cur().text}' declarations are only allowed at module level")
      this.syncStmt()
      return ast.BadStmt(pos: start)
    }
    // an expression or an assignment
    val x = this.parseExpr()
    if (x is ast.BadExpr) {
      this.syncStmt()
      return ast.BadStmt(pos: start)
    }
    if (this.atAssignOp()) {
      val op = this.next().kind
      return ast.AssignStmt(target: &x, op, value: &this.parseExpr(), pos: this.spanFrom(start))
    }
    ast.ExprStmt(x: &x)
  }

  // `=` or a compound assignment such as `+=`
  fun atAssignOp(): bool => this.atAny(Kind.Assign, Kind.PlusEq, Kind.MinusEq, Kind.StarEq, Kind.SlashEq, Kind.PercentEq)

  fun parseValStmt(start: Span): ast.Stmt {
    var s: ast.ValStmt = ast.ValStmt(kind: bindKind(this.next().kind))
    // `val JObj(fields) = doc else { … }`: a name and `(` is a variant
    // pattern, which only a let-else binds; so is a list pattern (D62)
    val isPattern = this.at(Kind.LBracket) || this.at(Kind.LParen) && this.tupleHoldsPattern() ||
      this.at(Kind.Ident) && (this.peek(1).kind == Kind.LParen || (this.peek(1).kind == Kind.Dot && this.peek(2).kind == Kind.Ident && this.peek(3).kind == Kind.LParen))
    if (isPattern) {
      s.pattern = &this.parsePattern(true)
    } else {
      s.binding = this.parseBinding()
    }
    if (this.accept(Kind.Assign)) {
      s.value = &this.parseExpr()
    } else if (s.kind != ast.BindKind.Var || s.binding.typ == null || s.pattern != null) {
      this.errorAt(this.span(), "'${ast.bindWord(s.kind)}' binding needs an initializer")
    }
    // let-else: the `else` may start the next line
    if (this.at(Kind.Semi) && this.cur().autoSemi && this.peek(1).kind == Kind.KwElse) this.next()
    if (this.accept(Kind.KwElse)) {
      // `else { … }`, or `else return`: one statement, like the body of `if (c) stmt`
      val handlerStart = this.span()
      val body = this.parseBodyOrStmt()
      s.orElse = ast.Handler(body, pos: this.spanFrom(handlerStart))
    } else if (val pattern = s.pattern) {
      val example = if (*pattern is ast.ListPat) "[a, b]" else "Variant(field)"
      this.errorAt(this.span(), "a pattern in a 'val' can fail to match, so it needs 'else { ... }' to say what happens then: 'val $example = x else { return }'")
    }
    s.pos = this.spanFrom(start)
    s
  }

  // `loop (…) body`, or the `for` the caller reported
  fun parseLoop(start: Span): ast.Stmt {
    if (!this.accept(Kind.KwFor)) this.expect(Kind.KwLoop)
    var s: ast.LoopStmt = ast.LoopStmt()
    if (this.accept(Kind.Colon)) s.label = this.ident()  // `loop :outer (x in xs) { … break outer }`
    if (this.at(Kind.LParen)) {
      // `loop (x in c)` or `loop (cond)`: a binding before `in` is iteration
      val iterates = this.looksLikeForIn()
      this.next()
      if (iterates) {
        s.bound = this.parseBinding()
        this.expect(Kind.KwIn)
        s.iter = &this.parseExpr()
        this.expect(Kind.RParen)
      } else {
        s.cond = &this.parseExpr()
        this.closeCondition()
      }
    }
    s.body = this.parseBodyOrStmt()
    s.pos = this.spanFrom(start)
    s
  }

  // `( binding in`, not consumed
  fun looksLikeForIn(): bool {
    var depth: i64 = 0
    loop (i in 1..64) {
      when (this.peek(i).kind) {
        Kind.LParen => depth += 1
        Kind.RParen => {
          if (depth == 0) return false
          depth -= 1
        }
        Kind.KwIn => return depth == 0
        Kind.Ident, Kind.Comma, Kind.Under, Kind.Colon, Kind.Amp => { }  // part of a binding
        Kind.EOF => return false
        else => {
          if (depth == 0) return false
        }
      }
    }
    false
  }

  // `with name = e` or `with e` (D100, D109), as against `with (`
  fun atWithStmt(): bool => this.at(Kind.KwWith) && this.peek(1).kind != Kind.LParen

  // the `(on: executor)` of `scope(on: e)` and `gather(on: e)` (D143); null
  // when the block follows directly
  fun parseOn(what: string): (*ast.Expr)? {
    if (!this.at(Kind.LParen)) return null
    this.next()
    if (!this.at(Kind.Ident) || this.cur().text != "on" || this.peek(1).kind != Kind.Colon) {
      this.errorAt(this.span(), "'$what(…)' takes one argument, the executor its tasks run on: '$what(on: pool) { … }' (D143)")
    } else {
      this.next()
      this.next()
    }
    val on = &this.parseExpr()
    this.expect(Kind.RParen)
    on
  }

  // `name = e`, or `e` held without a name (D109)
  fun parseWithItem(): ast.WithBinding {
    if (!this.atAny(Kind.Ident, Kind.Under) || this.peek(1).kind != Kind.Assign) return ast.WithBinding(value: &this.parseExpr())
    val t = this.next()
    this.next()  // =
    ast.WithBinding(name: ast.Ident(name: t.text, pos: t.span), value: &this.parseExpr())
  }

  // `with name = expr` or `with expr`: one item
  fun parseWithStmt(): ast.WithStmt {
    val start = this.span()
    this.next()  // with
    val binding = this.parseWithItem()
    if (this.at(Kind.Comma)) {
      this.errorAt(this.span(), "'with x = e' binds one resource; write each on its own line ('with a = …' then 'with b = …'), or use the block form 'with (a = …, b = …) { ... }' (D100)")
      loop (this.accept(Kind.Comma)) this.parseWithItem()  // read the rest as written, so nothing cascades
    }
    ast.WithStmt(binding, pos: this.spanFrom(start))
  }

  // `with (a = e, b = f) { body }`
  fun parseWith(): ast.Expr {
    val start = this.span()
    if (this.atWithStmt()) {
      // an operand, an expression body or an arm: no rest of a block to live through
      this.errorAt(this.span(), "'with x = e' is a statement: it closes x when its block ends, so it cannot stand inside an expression; write the block form 'with (x = e) { ... }', or put it on its own line in a braced block (D100)")
      val w = this.parseWithStmt()
      return ast.WithExpr(bindings: [w.binding], body: ast.Block(pos: w.pos), pos: this.spanFrom(start))
    }
    this.next()  // with
    val bindings: MutableList<ast.WithBinding> = []
    if (this.expect(Kind.LParen)) {
      loop (!this.atAny(Kind.RParen, Kind.EOF)) {
        bindings.push(this.parseWithItem())
        if (!this.accept(Kind.Comma)) break
      }
      this.expect(Kind.RParen)
    }
    val body = this.parseBlock()
    ast.WithExpr(bindings: bindings.toList(), body, pos: this.spanFrom(start))
  }

  // `do { body } catch (e) { handler }` (D98), the cursor on `do`; the
  // `catch` may start the next line
  fun parseDoCatch(): ast.Expr {
    val start = this.span()
    this.next()  // do
    val body = this.parseBlock()
    if (this.at(Kind.Semi) && this.cur().autoSemi && this.peek(1).kind == Kind.KwCatch) this.next()
    if (!this.accept(Kind.KwCatch)) {
      if (this.at(Kind.KwReserved) && this.cur().text == "while") {
        this.errorAt(this.span(), "there is no do-while: write 'loop { ...; if (!cond) break }', or 'loop (cond) { ... }' to test first; 'do' opens 'do { ... } catch (e) { ... }' (D98)")
      } else {
        this.errorAt(this.span(), "a 'do' block needs a 'catch (e) { ... }' after it: what to do when a 'try' or 'throw' inside fails (D98)")
      }
      // a well-formed placeholder, not a BadExpr, which would swallow the terminator
      val at = this.spanFrom(start)
      return ast.CatchExpr(body, handler: ast.Handler(body: ast.Block(pos: at), pos: at), pos: at)
    }
    val handler = this.parseCatchHandler()
    ast.CatchExpr(body, handler, pos: this.spanFrom(start))
  }

  // the `{ … }` after `??`: the block that stands in for an `Err`
  fun parseHandler(): ast.Handler {
    val start = this.span()
    val body = this.parseBlock()
    ast.Handler(body, pos: this.spanFrom(start))
  }

  // a `catch` here or at the start of the next line
  fun atCatch(): bool => this.at(Kind.KwCatch) || this.at(Kind.Semi) && this.cur().autoSemi && this.peek(1).kind == Kind.KwCatch

  // `catch (e) { handler }` after an expression (D98)
  fun parseCatchPostfix(x: ast.Expr, start: Span): ast.Expr {
    this.accept(Kind.Semi)
    this.next()  // catch
    val handler = this.parseCatchHandler()
    ast.CatchExpr(x: &x, handler, pos: this.spanFrom(start))
  }

  // after the word `catch`: `(e) { … }`, which binds the error, or `{ … }`
  fun parseCatchHandler(): ast.Handler {
    val start = this.span()
    var h = ast.Handler()
    if (this.accept(Kind.LParen)) {
      if (this.atAny(Kind.Ident, Kind.Under)) {
        val t = this.next()
        h.err = ast.Ident(name: t.text, pos: t.span)
      } else {
        this.errorExpected("the name for the error")
      }
      this.expect(Kind.RParen)
    }
    h.body = this.parseBlock()
    h.pos = this.spanFrom(start)
    h
  }

  // whether the parenthesised binding at the cursor nests a pattern that can
  // fail — a list pattern or a variant: `val (label, [a, b]) = pair else …`
  fun tupleHoldsPattern(): bool {
    var depth: i64 = 0
    var i: i64 = 0
    loop {
      when (this.peek(i).kind) {
        Kind.LParen => depth += 1
        Kind.RParen => {
          depth -= 1
          if (depth == 0) return false
        }
        Kind.LBracket => return true
        Kind.Ident => {
          if (this.peek(i + 1).kind == Kind.LParen && depth > 0) return true
        }
        Kind.EOF, Kind.Assign, Kind.LBrace, Kind.Semi => return false
        else => { }
      }
      i += 1
    }
  }
}
