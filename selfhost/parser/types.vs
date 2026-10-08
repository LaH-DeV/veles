// Types. A postfix `?` binds tighter than a
// prefix `*` (D5): `*T?` points to a nullable T, `(*T)?` is a nullable
// pointer.
use ast
use lexer { Kind }
use source { Span }

extend Parser {
  fun parseType(): ast.Type {
    val start = this.span()
    val base: ast.Type = when (this.cur().kind) {
      Kind.Star       => {
        this.next()
        val isRaw = this.accept(Kind.KwRaw)
        val elem = this.parseType()
        return ast.PointerType(elem: &elem, isRaw, pos: this.spanFrom(start))
      }
      Kind.LParen     => this.parseTupleType(start)
      Kind.KwFun      => {
        this.next()
        this.parseFunType(ast.FunType(), start)
      }
      Kind.KwExtern   => {
        // `extern fun(A): R`: a C function pointer (D69)
        this.next()
        if (!this.accept(Kind.KwFun)) {
          this.errorExpected("'fun' after 'extern' in a type (a C function pointer: 'extern fun(i32): i32')")
          return errorType(start)
        }
        this.parseFunType(ast.FunType(c: true), start)
      }
      Kind.Ident      => {
        if (this.atWord("sendable") && this.peek(1).kind == Kind.KwFun) {
          // `sendable fun(A): R` (D35): a word only here, like `extend`
          this.next()
          this.next()
          this.parseFunType(ast.FunType(sendable: true), start)
        } else {
          val path: MutableList<ast.Ident> = [this.ident()]
          loop (this.at(Kind.Dot) && this.peek(1).kind == Kind.Ident) {
            this.next()
            path.push(this.ident())
          }
          val args: List<ast.Type> = if (this.at(Kind.Lt)) this.parseTypeArgs() else []
          ast.NamedType(path: path.toList(), args, pos: this.spanFrom(start))
        }
      }
      Kind.KwSelfType => {
        this.next()
        // `Self.Item`: an associated type of the implementing type
        var t: ast.Type = ast.SelfType(pos: start)
        loop (this.at(Kind.Dot) && this.peek(1).kind == Kind.Ident) t = this.parseAssocType(t, start)
        t
      }
      else            => {
        this.errorExpected("a type")
        return errorType(start)
      }
    }
    this.typePostfix(base, start)
  }

  // `(A, B)`, `(A,)` and `()` are tuples; `(A)` is A
  fun parseTupleType(start: Span): ast.Type {
    this.next()  // (
    val elems: MutableList<ast.Type> = []
    var trailingComma = false
    loop (!this.atAny(Kind.RParen, Kind.EOF)) {
      elems.push(this.parseType())
      trailingComma = this.accept(Kind.Comma)
      if (!trailingComma) break
    }
    this.expect(Kind.RParen)
    val [only] = elems else return ast.TupleType(elems: elems.toList(), pos: this.spanFrom(start))
    if (trailingComma) ast.TupleType(elems: elems.toList(), pos: this.spanFrom(start)) else only
  }

  // `.Item` after base, the cursor on the dot
  fun parseAssocType(base: ast.Type, start: Span): ast.Type {
    this.next()  // .
    val name = this.ident()
    ast.AssocType(base: &base, name, pos: this.spanFrom(start))
  }

  // the `?`s after a type; in a type `T??` is two of them, not the Result operator
  fun typePostfix(base: ast.Type, start: Span): ast.Type {
    if (this.accept(Kind.Question)) return this.typePostfix(this.nullable(base, start), start)
    if (this.accept(Kind.Coalesce)) return this.typePostfix(this.nullable(this.nullable(base, start), start), start)
    base
  }

  fun nullable(elem: ast.Type, start: Span): ast.Type => ast.NullableType(elem: &elem, pos: this.spanFrom(start))

  // `<T, U>` where a type goes, the cursor on the `<`
  fun parseTypeArgs(): List<ast.Type> {
    val lt = this.next()
    val args: MutableList<ast.Type> = []
    if (this.atTypeClose()) this.errorAt(lt.span, "empty type argument list; drop the '<>'")
    loop (!this.atTypeClose()) {
      args.push(this.parseTypeArg())
      if (!this.accept(Kind.Comma)) break
    }
    this.expectTypeClose()
    args.toList()
  }

  // a type, or a constant (D121): a number or arithmetic on names
  fun parseTypeArg(): ast.Type {
    if (this.unionArgAhead()) return this.parseErrorType()
    if (!this.constArgAhead()) return this.parseType()
    val start = this.span()
    val saved = this.inTypeArg
    this.inTypeArg = true
    val x = this.parseBinary(BP_CMP)  // stops at '>' and ','
    this.inTypeArg = saved
    ast.ConstType(x: &x, pos: this.spanFrom(start))
  }

  // names joined by '|' to the end of the argument: an error union (D45)
  fun unionArgAhead(): bool {
    var i: i64 = 0
    var pipes: i64 = 0
    loop {
      if (this.peek(i).kind != Kind.Ident) return false
      i += 1
      loop (this.peek(i).kind == Kind.Dot && this.peek(i + 1).kind == Kind.Ident) i += 2
      when (this.peek(i).kind) {
        Kind.Pipe => {
          pipes += 1
          i += 1
        }
        Kind.Comma, Kind.Gt, Kind.Shr => return pipes > 0
        else => return false
      }
    }
  }

  // a constant argument: it begins with a number, or a name an arithmetic
  // operator follows
  fun constArgAhead(): bool => when (this.cur().kind) {
    Kind.Int    => true
    Kind.Minus  => this.peek(1).kind == Kind.Int   // a negative length: the checker says what is wrong
    Kind.LParen => this.peek(1).kind == Kind.Int  // `(1 << 4)`: no tuple type begins with a number
    Kind.Ident  => when (this.peek(1).kind) {
      Kind.Plus, Kind.Minus, Kind.Star, Kind.Slash, Kind.Percent, Kind.Shl, Kind.Amp, Kind.Pipe, Kind.Caret => true
      else => false
    }
    else        => false
  }

  // `>`, or the first half of a `>>` closing two lists (`List<List<i64>>`)
  fun atTypeClose(): bool => this.atAny(Kind.Gt, Kind.Shr, Kind.EOF)

  // a `>`; a `>>` is split, one `>` left for the enclosing list
  fun expectTypeClose() {
    if (this.at(Kind.Shr)) {
      val shr = this.cur()
      var first = shr
      first.kind = Kind.Gt
      first.text = ">"
      var second = first
      first.span = Span(file: shr.span.file, start: shr.span.start, end: shr.span.end - 1)
      second.span = Span(file: shr.span.file, start: shr.span.start + 1, end: shr.span.end)
      this.toks.set(this.pos, first)
      this.toks.insert(this.pos + 1, second)
    }
    this.expect(Kind.Gt)
  }

  // after `throws`: a type or a union `A | B`
  fun parseErrorType(): ast.Type {
    val start = this.span()
    val first = this.parseType()
    if (!this.at(Kind.Pipe)) return first
    val members: MutableList<ast.Type> = [first]
    loop (this.accept(Kind.Pipe)) members.push(this.parseType())
    ast.ErrorUnionType(members: members.toList(), pos: this.spanFrom(start))
  }

  // after `fun`: `(A, B): R suspends throws E`
  fun parseFunType(shape: ast.FunType, start: Span): ast.FunType {
    var ft = shape
    if (this.expect(Kind.LParen)) {
      val params: MutableList<ast.Type> = []
      loop (!this.atAny(Kind.RParen, Kind.EOF)) {
        params.push(this.parseType())
        if (!this.accept(Kind.Comma)) break
      }
      this.expect(Kind.RParen)
      ft.params = params.toList()
    }
    if (this.accept(Kind.Colon)) ft.ret = &this.parseType()
    ft.effects = this.parseEffects()
    ft.pos = this.spanFrom(start)
    ft
  }
}

// a type that did not parse
fun errorType(at: Span): ast.Type => ast.NamedType(path: [ast.Ident(name: "<error>", pos: at)], pos: at)
