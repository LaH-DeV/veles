// Expressions: Pratt-style binary operators,
// prefix and postfix forms, primaries.
use ast
use lexer { Kind, Token, tokenizeRange }
use source { Diagnostics, Span }

// binding powers, weakest first
const BP_NONE: i64 = 0
const BP_OR: i64 = 1     // ||
const BP_AND: i64 = 2    // &&
const BP_EQ: i64 = 3     // == !=
const BP_CMP: i64 = 4    // < <= > >=
const BP_NAMED: i64 = 5  // is, !is, implements
const BP_ELVIS: i64 = 6  // ?: ?! ??
const BP_RANGE: i64 = 7  // .. ..<
const BP_ADD: i64 = 8    // + - +% -% | ^
const BP_MUL: i64 = 9    // * / % *% & << >>
const BP_CAST: i64 = 10  // as

fun infixBp(k: Kind): i64 => when (k) {
  Kind.OrOr => BP_OR
  Kind.AndAnd => BP_AND
  Kind.Eq, Kind.NotEq => BP_EQ
  Kind.Lt, Kind.LtEq, Kind.Gt, Kind.GtEq => BP_CMP
  Kind.KwIs => BP_NAMED
  Kind.Elvis, Kind.OrFail, Kind.Coalesce => BP_ELVIS
  Kind.Range, Kind.RangeLt => BP_RANGE
  Kind.Plus, Kind.Minus, Kind.WrapPlus, Kind.WrapMinus, Kind.Pipe, Kind.Caret => BP_ADD  // `|` and `^` with `+`
  Kind.Star, Kind.Slash, Kind.Percent, Kind.WrapStar, Kind.Amp, Kind.Shl, Kind.Shr => BP_MUL
  Kind.KwAs => BP_CAST
  else => BP_NONE
}

// a name, or `module.name`: what a template literal's tag may be
fun isTemplateTag(x: ast.Expr): bool => when (x) {
  is ast.NameExpr   => x.typeArgs.isEmpty()
  is ast.MemberExpr => !x.safe && x.typeArgs.isEmpty() && *x.x is ast.NameExpr
  else              => false
}

// the tag as written: `sql` or `db.sql`
fun templateTagText(x: ast.Expr): string => when (x) {
  is ast.NameExpr   => x.name
  is ast.MemberExpr => templateTagText(*x.x) + "." + x.name.name
  else              => "tag"
}

// a lambda parameter's name: the identifier, or `_`
fun paramName(t: Token): ast.Ident => ast.Ident(name: if (t.kind == Kind.Under) "_" else t.text, pos: t.span)

extend Parser {
  fun parseExpr(): ast.Expr {
    val savedTry = this.inTry
    val savedArg = this.inTypeArg
    this.inTry = false
    this.inTypeArg = false
    val x = this.parseBinary(BP_NONE)
    this.inTry = savedTry
    this.inTypeArg = savedArg
    x
  }

  // the operators that bind tighter than minBp, and their operands
  fun parseBinary(minBp: i64): ast.Expr {
    val start = this.span()
    var left = this.parseUnary()
    loop {
      // `!is` is two tokens
      val notIs = this.at(Kind.Bang) && this.peek(1).kind == Kind.KwIs
      val op = if (notIs) Kind.KwIs else this.cur().kind
      if (left is ast.NameExpr && this.atWord("implements")) {
        if (BP_NAMED <= minBp) return left
        left = this.parseImplements(left, start)
        continue
      }
      val bp = if (op == Kind.Shr && this.inTypeArg) BP_NONE else infixBp(op)
      if (bp == BP_NONE || bp <= minBp) return left
      left = this.parseInfix(left, op, bp, notIs, start)
    }
  }

  // one infix operator at the cursor, with left before it
  fun parseInfix(left: ast.Expr, op: Kind, bp: i64, notIs: bool, start: Span): ast.Expr {
    if (notIs) this.next()  // !
    this.next()
    when (op) {
      Kind.KwIs                => {
        val pat = this.parseTypePatternRest(this.prevSpan())
        return ast.IsExpr(x: &left, pat, not: notIs, pos: this.spanFrom(start))
      }
      Kind.KwAs                => {
        val typ = this.parseType()
        return ast.CastExpr(x: &left, typ: &typ, pos: this.spanFrom(start))
      }
      Kind.Elvis               => {
        // right-associative: `a ?: b ?: c` is `a ?: (b ?: c)`; so are `?!` and `??`
        val right = this.parseBinary(bp - 1)
        return ast.ElvisExpr(left: &left, right: &right, pos: this.spanFrom(start))
      }
      Kind.OrFail              => {
        val right = this.parseBinary(bp - 1)
        return ast.OrFailExpr(left: &left, right: &right, pos: this.spanFrom(start))
      }
      Kind.Coalesce            => {
        if (this.at(Kind.LBrace)) return ast.CoalesceExpr(left: &left, handler: this.parseHandler(), pos: this.spanFrom(start))
        val right = this.parseBinary(bp - 1)
        return ast.CoalesceExpr(left: &left, right: &right, pos: this.spanFrom(start))
      }
      Kind.Range, Kind.RangeLt => {
        val hi = this.parseBinary(bp)
        return ast.RangeExpr(lo: &left, hi: &hi, inclusive: op == Kind.Range, pos: this.spanFrom(start))
      }
      else                     => {
        val right = this.parseBinary(bp)
        return ast.BinaryExpr(op, left: &left, right: &right, pos: this.spanFrom(start))
      }
    }
  }

  // `T implements Trait` (D117): a word only here, after a name
  fun parseImplements(name: ast.NameExpr, start: Span): ast.Expr {
    this.next()  // implements
    val traitType = this.parseType()
    val typ: ast.Type = ast.NamedType(path: [ast.Ident(name: name.name, pos: name.pos)], args: name.typeArgs, pos: name.pos)
    ast.ImplementsExpr(typ: &typ, traitType: &traitType, pos: this.spanFrom(start))
  }

  fun parseUnary(): ast.Expr {
    val start = this.span()
    when (this.cur().kind) {
      Kind.Minus, Kind.Bang, Kind.Amp, Kind.Star, Kind.Tilde => {
        val op = this.next().kind
        val x = this.parseUnary()
        return ast.UnaryExpr(op, x: &x, pos: this.spanFrom(start))
      }
      Kind.KwVal => {
        // `val name = value` in an `if` condition (D95); the value binds
        // tighter than `&&`, so the chain goes on
        this.next()
        val name = this.ident()
        this.expect(Kind.Assign)
        val value = this.parseBinary(BP_AND)
        return ast.LetCond(name, value: &value, pos: this.spanFrom(start))
      }
      Kind.KwTry => {
        this.next()
        val savedTry = this.inTry
        this.inTry = true
        var operand = this.parseUnary()
        this.inTry = savedTry
        // `try x ?! e` is `try (x ?! e)`
        loop (this.at(Kind.OrFail)) operand = this.parseInfix(operand, Kind.OrFail, BP_ELVIS, false, start)
        val tried = ast.TryExpr(x: &operand, pos: this.spanFrom(start))
        // `try f().g() catch (e) { … }`: the pair covers the whole chain (D98)
        return if (this.atCatch()) this.parseCatchPostfix(tried, start) else tried
      }
      Kind.KwAwait => {
        this.next()
        val x = this.parseUnary()
        return ast.AwaitExpr(x: &x, pos: this.spanFrom(start))
      }
      Kind.KwAsync => {
        this.next()
        val x = this.parseUnary()
        when (x) {
          is ast.CallExpr => {
            var call = x
            call.isAsync = true
            call.pos = this.spanFrom(start)
            return call
          }
          else            => {
            this.errorAt(this.spanFrom(start), "'async' must prefix a call expression (D2)")
            return x
          }
        }
      }
      else => return this.parsePostfix()
    }
  }

  fun parsePostfix(): ast.Expr {
    val start = this.span()
    var x = this.parsePrimary()
    loop {
      x = this.parsePostfixOp(x, start) ?: return x
    }
  }

  // one postfix form after x — a member, a call, an index, a template, a
  // `catch`, type arguments — or null when none follows
  fun parsePostfixOp(x: ast.Expr, start: Span): ast.Expr? {
    when (this.cur().kind) {
      Kind.Dot, Kind.SafeDot => {
        val safe = this.next().kind == Kind.SafeDot
        val name = if (this.at(Kind.Int)) this.tupleIndex(x) else this.ident()
        return ast.MemberExpr(x: &x, name, safe, pos: this.spanFrom(start))
      }
      Kind.LParen            => {
        val args = this.parseArgs()
        return ast.CallExpr(callee: &x, args, pos: this.spanFrom(start))
      }
      Kind.String            => {
        // `tag"text ${x}"` (D129): a string right after a name, no space
        if (!isTemplateTag(x)) return null
        val t = this.next()
        if (ast.exprSpan(x).end != t.span.start) {
          this.errorAt(t.span, "a template literal has nothing between the name and the string: write '${templateTagText(x)}\"…\"' (D129); a string after a name is otherwise an error")
        }
        return ast.TemplateExpr(tag: &x, lit: this.stringLit(t), pos: this.spanFrom(start))
      }
      Kind.KwCatch           => return if (this.inTry) null else this.parseCatchPostfix(x, start)
      Kind.Semi              => {
        // a `catch` may start the next line, as an `else` may
        if (this.inTry || !this.atCatch()) return null
        this.next()
        return x
      }
      Kind.LBracket          => {
        this.next()
        val index = this.parseExpr()
        this.expect(Kind.RBracket)
        return ast.IndexExpr(x: &x, index: &index, pos: this.spanFrom(start))
      }
      Kind.Lt                => {
        // perhaps `Name<T>(…)`: a generic call, tried and undone if not
        if (!(x is ast.NameExpr || x is ast.MemberExpr)) return null
        val typeArgs = this.tryTypeArgsBeforeCall() ?: return null
        if (this.at(Kind.LParen)) {
          val args = this.parseArgs()
          return ast.CallExpr(callee: &x, typeArgs, args, pos: this.spanFrom(start))
        }
        // `Name<T>.f(…)`, `mod.Name<T>.f(…)`: a static function of a generic type (D23)
        return when (x) {
          is ast.MemberExpr => ast.MemberExpr(x: x.x, name: x.name, safe: x.safe, typeArgs, pos: this.spanFrom(start))
          is ast.NameExpr   => ast.NameExpr(name: x.name, typeArgs, pos: this.spanFrom(start))
          else              => x
        }
      }
      else                   => return null
    }
  }

  // `pair.0`: a tuple element's number, after the dot
  fun tupleIndex(x: ast.Expr): ast.Ident {
    val t = this.next()
    // `0 .0` would print back as `0.0`
    if (x is ast.IntLit || x is ast.FloatLit) this.errorAt(t.span, "a number has no tuple elements")
    ast.Ident(name: t.text, pos: t.span)
  }

  // `<T, U>` then `(`, or then `.name(`; on failure nothing is taken and
  // nothing reported
  fun tryTypeArgsBeforeCall(): List<ast.Type>? {
    val savedPos = this.pos
    val savedDiags = this.diags
    val savedErr = this.lastErrPos
    var scratch = Diagnostics()
    this.diags = &scratch
    val typeArgs = this.parseTypeArgs()
    val staticCall = this.at(Kind.Dot) && this.peek(1).kind == Kind.Ident && this.peek(2).kind == Kind.LParen
    this.diags = savedDiags
    this.lastErrPos = savedErr
    if (scratch.hasErrors() || !(this.at(Kind.LParen) || staticCall)) {
      this.pos = savedPos
      return null
    }
    typeArgs
  }

  fun parseArgs(): List<ast.Arg> {
    val saved = this.noLambda
    this.noLambda = 0
    this.expect(Kind.LParen)
    val args: MutableList<ast.Arg> = []
    loop (!this.atAny(Kind.RParen, Kind.EOF)) {
      var name: ast.Ident? = null
      if (this.at(Kind.Ident) && this.peek(1).kind == Kind.Colon) {
        val t = this.next()
        this.next()  // :
        name = ast.Ident(name: t.text, pos: t.span)
      }
      val value = this.parseExpr()
      val spread = this.accept(Kind.Ellipsis)  // `f(xs...)`
      args.push(ast.Arg(name, value: &value, spread))
      if (value is ast.BadExpr) {
        this.syncParen()
        break
      }
      if (!this.accept(Kind.Comma)) break
    }
    this.expect(Kind.RParen)
    this.noLambda = saved
    args.toList()
  }

  fun parsePrimary(): ast.Expr {
    val start = this.span()
    val t = this.cur()
    when (t.kind) {
      Kind.Int => {
        this.next()
        return ast.IntLit(text: t.text, pos: t.span)
      }
      Kind.Float => {
        this.next()
        return ast.FloatLit(text: t.text, pos: t.span)
      }
      Kind.String => {
        this.next()
        return this.stringLit(t)
      }
      Kind.Char => {
        this.next()
        return ast.CharLit(value: (t.parts.at(0) ?: lexer.StringPart()).text, pos: t.span)
      }
      Kind.KwTrue, Kind.KwFalse => {
        this.next()
        return ast.BoolLit(value: t.kind == Kind.KwTrue, pos: t.span)
      }
      Kind.KwNull => {
        this.next()
        return ast.NullLit(pos: t.span)
      }
      Kind.KwThis => {
        this.next()
        return ast.ThisExpr(pos: t.span)
      }
      Kind.Ident => {
        if (this.peek(1).kind == Kind.FatArrow && this.noLambda == 0) return this.parseLambda()
        this.next()
        return ast.NameExpr(name: t.text, pos: t.span)
      }
      Kind.Under => {
        // `_ => …`: a lambda that ignores its argument
        if (this.peek(1).kind == Kind.FatArrow && this.noLambda == 0) return this.parseLambda()
      }
      Kind.LParen => {
        if (this.noLambda == 0 && this.looksLikeLambda()) return this.parseLambda()
        return this.parseParenOrTuple()
      }
      Kind.LBracket => return this.parseCollectionLit()
      Kind.KwMut => {
        // `mut [1, 2]`, `mut ["k": v]`, `mut [:]`
        this.next()
        if (!this.at(Kind.LBracket)) {
          this.errorAt(this.span(), "'mut' in an expression must be followed by a collection literal, e.g. 'mut [1, 2]' or 'mut [:]'")
          return ast.BadExpr(pos: this.spanFrom(start))
        }
        val lit = this.parseCollectionLit()
        when (lit) {
          is ast.ListLit => {
            var list = lit
            list.isMut = true
            list.pos = this.spanFrom(start)
            return list
          }
          is ast.MapLit  => {
            var map = lit
            map.isMut = true
            map.pos = this.spanFrom(start)
            return map
          }
          else           => return lit
        }
      }
      Kind.KwIf => return this.parseIf()
      Kind.KwWhen => return this.parseWhen()
      Kind.KwWith => return this.parseWith()
      Kind.KwGather => {
        this.next()
        val on = this.parseOn("gather")
        val body = this.parseBlock()
        return ast.GatherExpr(on, body, pos: this.spanFrom(start))
      }
      Kind.KwUnsafe => {
        this.next()
        val body = this.parseBlock()
        return ast.UnsafeExpr(body, pos: this.spanFrom(start))
      }
      Kind.KwRace => return this.parseRace()
      Kind.KwDo => return this.parseDoCatch()
      Kind.KwCatch => {
        this.errorAt(t.span, "'catch' follows an expression that can fail or a 'do' block: 'f() catch (e) { ... }', 'do { ... } catch (e) { ... }' (D98)")
        this.next()
        val handler = this.parseCatchHandler()
        return ast.CatchExpr(body: ast.Block(pos: t.span), handler, pos: this.spanFrom(start))
      }
      Kind.KwScope => {
        this.errorAt(t.span, "'scope' is a statement, not an expression; use 'gather' to collect results (D36)")
        this.next()
        this.parseBlock()
        return ast.BadExpr(pos: this.spanFrom(start))
      }
      Kind.KwFun => {
        this.errorAt(t.span, "anonymous functions are written as lambdas: '(x) => ...' (D32)")
        return ast.BadExpr(pos: t.span)
      }
      Kind.KwReturn, Kind.KwThrow, Kind.KwBreak, Kind.KwContinue => return ast.ControlExpr(stmt: &this.parseStmt())
      Kind.Arrow => {
        this.thinArrow(t.span, "'->' is not an operator; use '=>' (D33)")
        this.next()
        return ast.BadExpr(pos: t.span)
      }
      else => { }
    }
    this.errorAt(t.span, "expected an expression, found ${t.describe()}")
    ast.BadExpr(pos: t.span)
  }

  // a string literal; each interpolated expression is scanned and parsed on
  // its own, with the file's offsets
  fun stringLit(t: Token): ast.StringLit {
    val parts: MutableList<ast.StringPart> = []
    loop (part in t.parts) {
      if (!part.isExpr) {
        parts.push(ast.StringPart(text: part.text))
        continue
      }
      val toks = tokenizeRange(this.file, part.span.start, part.span.end, this.diags)
      var sub = Parser(file: this.file, toks: toks.toMutable(), diags: this.diags)
      val e = sub.parseExpr()
      sub.skipSemis()
      if (!sub.at(Kind.EOF)) sub.errorAt(sub.span(), "unexpected ${sub.cur().describe()} in string interpolation")
      parts.push(ast.StringPart(expr: &e))
    }
    ast.StringLit(parts: parts.toList(), pos: t.span)
  }

  // from `(` to its `)`: whether `=>` (or `: Type =>`) follows (D37)
  fun looksLikeLambda(): bool {
    var depth: i64 = 0
    var i: i64 = 0
    loop {
      when (this.peek(i).kind) {
        Kind.LParen, Kind.LBracket, Kind.LBrace => depth += 1
        Kind.RParen, Kind.RBracket, Kind.RBrace => {
          depth -= 1
          if (depth == 0) {
            val after = this.peek(i + 1).kind
            return after == Kind.FatArrow || after == Kind.Colon
          }
        }
        Kind.EOF => return false
        else => { }
      }
      i += 1
    }
  }

  fun parseLambda(): ast.Expr {
    val start = this.span()
    val params: MutableList<ast.Param> = []
    var ret: (*ast.Type)? = null
    if (this.atAny(Kind.Ident, Kind.Under)) {
      val t = this.next()
      params.push(ast.Param(name: paramName(t), pos: t.span))
    } else {
      this.expect(Kind.LParen)
      loop (!this.atAny(Kind.RParen, Kind.EOF)) {
        val paramStart = this.span()
        if (this.at(Kind.LParen)) {
          // a tuple pattern destructures the argument
          val b = this.parseBinding()
          params.push(ast.Param(name: ast.Ident(name: "\$tuple", pos: b.pos), pattern: b, pos: this.spanFrom(paramStart)))
          if (!this.accept(Kind.Comma)) break
          continue
        }
        val name = if (this.at(Kind.Under)) paramName(this.next()) else this.expectIdent()
        if (name == null) {
          this.syncParen()
          break
        }
        var param = ast.Param(name)
        if (this.accept(Kind.Colon)) param.typ = &this.parseType()
        param.pos = this.spanFrom(paramStart)
        params.push(param)
        if (!this.accept(Kind.Comma)) break
      }
      this.expect(Kind.RParen)
      if (this.accept(Kind.Colon)) ret = &this.parseType()
    }
    this.expect(Kind.FatArrow)
    val body = this.parseArmBody()
    ast.LambdaExpr(params: params.toList(), ret, body: &body, pos: this.spanFrom(start))
  }

  // the right side of `=>`: a block, an expression, or an assignment —
  // `x => total += x` — which becomes a one-statement block
  fun parseArmBody(): ast.Expr {
    if (this.at(Kind.LBrace)) return ast.BlockExpr(block: this.parseBlock())
    val start = this.span()
    val body = this.parseExpr()
    if (!this.atAssignOp()) return body
    val op = this.next().kind
    val value = this.parseExpr()
    val pos = this.spanFrom(start)
    ast.BlockExpr(block: ast.Block(stmts: [ast.AssignStmt(target: &body, op, value: &value, pos)], pos))
  }

  fun parseParenOrTuple(): ast.Expr {
    val saved = this.noLambda
    this.noLambda = 0
    val x = this.parenOrTuple()
    this.noLambda = saved
    x
  }

  fun parenOrTuple(): ast.Expr {
    val start = this.span()
    this.next()  // (
    if (this.accept(Kind.RParen)) return ast.TupleExpr(pos: this.spanFrom(start))  // unit
    val elems: MutableList<ast.Expr> = []
    var trailingComma = false
    loop (!this.atAny(Kind.RParen, Kind.EOF)) {
      elems.push(this.parseExpr())
      trailingComma = this.accept(Kind.Comma)
      if (!trailingComma) break
    }
    this.expect(Kind.RParen)
    val [only] = elems else return ast.TupleExpr(elems: elems.toList(), pos: this.spanFrom(start))
    if (trailingComma) return ast.TupleExpr(elems: elems.toList(), pos: this.spanFrom(start))
    // parentheses end a `?.` chain (D70)
    when (only) {
      is ast.MemberExpr => {
        var member = only
        member.grouped = true
        return member
      }
      is ast.CallExpr   => {
        var call = only
        call.grouped = true
        return call
      }
      else              => return only
    }
  }

  fun parseCollectionLit(): ast.Expr {
    val saved = this.noLambda
    this.noLambda = 0
    val x = this.collectionLit()
    this.noLambda = saved
    x
  }

  fun collectionLit(): ast.Expr {
    val start = this.span()
    this.next()  // [
    if (this.at(Kind.Colon) && this.peek(1).kind == Kind.RBracket) {
      this.next()
      this.next()
      return ast.MapLit(pos: this.spanFrom(start))
    }
    if (this.accept(Kind.RBracket)) return ast.ListLit(pos: this.spanFrom(start))
    val first = this.parseExpr()
    if (this.accept(Kind.Colon)) {
      val value = this.parseExpr()
      val entries: MutableList<ast.MapEntry> = [ast.MapEntry(key: &first, value: &value)]
      loop (this.accept(Kind.Comma) && !this.at(Kind.RBracket)) {
        val key = this.parseExpr()
        this.expect(Kind.Colon)
        val entryValue = this.parseExpr()
        entries.push(ast.MapEntry(key: &key, value: &entryValue))
      }
      this.expect(Kind.RBracket)
      return ast.MapLit(entries: entries.toList(), pos: this.spanFrom(start))
    }
    val elems: MutableList<ast.Expr> = [first]
    loop (this.accept(Kind.Comma) && !this.at(Kind.RBracket)) elems.push(this.parseExpr())
    this.expect(Kind.RBracket)
    ast.ListLit(elems: elems.toList(), pos: this.spanFrom(start))
  }

  fun parseIf(): ast.Expr {
    val start = this.span()
    this.next()  // if
    val cond = this.parseCondition()
    var e: ast.IfExpr = ast.IfExpr(cond: &cond, then: this.parseBodyOrStmt())
    // `else` may follow on the next line after a `}`, unless it is the
    // `else =>` arm of an enclosing `when`
    if (this.at(Kind.Semi) && this.cur().autoSemi && this.peek(1).kind == Kind.KwElse && this.peek(2).kind != Kind.FatArrow) this.next()
    if (this.accept(Kind.KwElse)) {
      if (this.at(Kind.KwIf)) {
        val elseStart = this.span()
        val inner = this.parseIf()
        e.orElse = ast.Block(stmts: [ast.ExprStmt(x: &inner)], pos: this.spanFrom(elseStart))
      } else {
        e.orElse = this.parseBodyOrStmt()
      }
    }
    e.pos = this.spanFrom(start)
    e
  }

  // the `(cond)` of an `if`; a BadExpr when the `(` is missing
  fun parseCondition(): ast.Expr {
    val missing: ast.Expr = ast.BadExpr(pos: this.span())
    if (!this.expect(Kind.LParen)) return missing
    // `name = value` is read as the binding it was meant to be, so nothing cascades
    val cond = if (this.missingVal()) this.parseUnmarkedBinding() else this.parseExpr()
    this.closeCondition()
    cond
  }

  // `name = value && …` at the cursor, which missingVal reported: read as
  // if `val` were written
  fun parseUnmarkedBinding(): ast.Expr {
    val name = this.next()
    this.next()  // =
    val value = this.parseBinary(BP_AND)
    var cond: ast.Expr = ast.LetCond(name: ast.Ident(name: name.text, pos: name.span), value: &value, pos: this.spanFrom(name.span))
    loop (this.at(Kind.AndAnd)) cond = this.parseInfix(cond, Kind.AndAnd, BP_AND, false, name.span)
    cond
  }

  fun parseWhen(): ast.Expr {
    val start = this.span()
    this.next()  // when
    var w: ast.WhenExpr = ast.WhenExpr()
    if (this.accept(Kind.LParen)) {
      if (this.at(Kind.KwVal) && this.peek(1).kind == Kind.Ident && this.peek(2).kind == Kind.Assign) {
        // `when (val r = expr)`: a name for the subject the arms can use
        this.next()
        w.bind = this.ident()
        this.next()  // =
      } else if (this.missingVal()) {
        val name = this.next()
        w.bind = ast.Ident(name: name.text, pos: name.span)
        this.next()  // =
      }
      w.subject = &this.parseExpr()
      this.expect(Kind.RParen)
    }
    if (!this.expect(Kind.LBrace)) {
      w.pos = this.spanFrom(start)
      return w
    }
    val arms: MutableList<ast.WhenArm> = []
    loop {
      this.skipSemis()
      if (this.atAny(Kind.RBrace, Kind.EOF)) break
      val before = this.pos
      arms.push(this.parseWhenArm(w.subject != null))
      if (this.pos == before) this.next()
      if (!this.atAny(Kind.Semi, Kind.RBrace, Kind.EOF)) {
        this.errorAt(this.span(), "expected newline between 'when' arms, found ${this.cur().describe()}")
        this.syncStmt()
      }
    }
    this.expect(Kind.RBrace)
    w.arms = arms.toList()
    w.pos = this.spanFrom(start)
    w
  }

  fun parseWhenArm(hasSubject: bool): ast.WhenArm {
    val start = this.span()
    var arm = ast.WhenArm(body: &ast.BadExpr(pos: start))
    if (this.accept(Kind.KwElse)) {
      arm.isElse = true
    } else if (hasSubject) {
      val patterns: MutableList<ast.Pattern> = [this.parsePattern(false)]
      loop (this.accept(Kind.Comma)) patterns.push(this.parsePattern(false))
      arm.patterns = patterns.toList()
      if (this.accept(Kind.KwIf)) arm.guard = &this.parseArmHead()
    } else {
      arm.cond = &this.parseArmHead()
    }
    if (this.expectArmArrow()) {
      arm.body = &this.parseArmBody()
    } else {
      this.syncStmt()
      arm.body = &ast.BadExpr(pos: this.span())
    }
    arm.pos = this.spanFrom(start)
    arm
  }

  fun parseRace(): ast.Expr {
    val start = this.span()
    this.next()  // race
    if (!this.expect(Kind.LBrace)) return ast.RaceExpr(pos: this.spanFrom(start))
    val arms: MutableList<ast.RaceArm> = []
    loop {
      this.skipSemis()
      if (this.atAny(Kind.RBrace, Kind.EOF)) break
      val armStart = this.span()
      var binding: ast.Binding? = null
      if (this.accept(Kind.KwVal)) {
        binding = this.parseBinding()
        this.expect(Kind.Assign)
      }
      val source = this.parseArmHead()
      if (!this.expectArmArrow()) {
        this.syncStmt()
        continue
      }
      val body = this.parseArmBody()
      arms.push(ast.RaceArm(binding, source: &source, body: &body, pos: this.spanFrom(armStart)))
      if (!this.atAny(Kind.Semi, Kind.RBrace, Kind.EOF)) {
        this.errorAt(this.span(), "expected newline between 'race' arms, found ${this.cur().describe()}")
        this.syncStmt()
      }
    }
    this.expect(Kind.RBrace)
    ast.RaceExpr(arms: arms.toList(), pos: this.spanFrom(start))
  }

  // the `=>` between an arm's head and its body; Kotlin's `->` is reported
  // and read as `=>`, so the arms after it check normally
  fun expectArmArrow(): bool {
    if (!this.at(Kind.Arrow)) return this.expect(Kind.FatArrow)
    this.thinArrow(this.span(), "'->' is not an operator; an arm is 'pattern => value' (D33)")
    this.next()
    true
  }

  // the `)` after a condition, with a hint for `=` written for `==`
  fun closeCondition() {
    if (this.at(Kind.Assign)) {
      this.errorAt(this.span(), "'=' assigns; use '==' to compare")
      this.next()
      this.parseExpr()
    }
    this.expect(Kind.RParen)
  }

  // the expression before an arm's `=>`, where `name => …` is the arm's arrow
  fun parseArmHead(): ast.Expr {
    this.noLambda += 1
    val e = this.parseExpr()
    this.noLambda -= 1
    e
  }

  // an `if`/`when` head starting `name =`: a binding written without its
  // `val` (D101); one error, and the caller reads it as the binding
  fun missingVal(): bool {
    if (!this.at(Kind.Ident) || this.peek(1).kind != Kind.Assign) return false
    val name = this.cur()
    this.errorAt(name.span, "a binding in a condition is written 'val ${name.text} = …' (D101)")
    true
  }
}
