// Patterns (D13).
use ast, lexer { Kind }, source { Span }

extend Parser {
  // one `when` pattern. With binding false a bare name is a value to compare
  // with (Kotlin's rule for top-level arms); with binding true, inside a
  // destructuring, it binds a new name
  fun parsePattern(binding: bool): ast.Pattern {
    val start = this.span()
    when (this.cur().kind) {
      Kind.Under => {
        this.next()
        return ast.WildcardPat(pos: start)
      }
      Kind.KwIs => {
        this.next()
        return this.parseTypePatternRest(start)
      }
      Kind.KwIn => {
        this.next()
        val range = this.parseBinary(BP_ELVIS)
        when (range) {
          is ast.RangeExpr => return ast.RangePat(range, pos: this.spanFrom(start))
          else             => {
            this.errorAt(ast.exprSpan(range), "expected a range after 'in'")
            return ast.LiteralPat(value: &range)
          }
        }
      }
      Kind.LParen => {
        this.next()
        val elems: MutableList<ast.Pattern> = []
        loop (!this.atAny(Kind.RParen, Kind.EOF)) {
          elems.push(this.parsePattern(true))
          if (!this.accept(Kind.Comma)) break
        }
        this.expect(Kind.RParen)
        return ast.TuplePat(elems: elems.toList(), pos: this.spanFrom(start))
      }
      Kind.LBracket => {
        // `[a, b]`, `[first, ..rest]`, `[.., last]` (D62)
        this.next()
        val elems: MutableList<ast.Pattern> = []
        loop (!this.atAny(Kind.RBracket, Kind.EOF)) {
          elems.push(if (this.at(Kind.Range)) this.parseRestPattern() else this.parsePattern(true))
          if (!this.accept(Kind.Comma)) break
        }
        this.expect(Kind.RBracket)
        return ast.ListPat(elems: elems.toList(), pos: this.spanFrom(start))
      }
      Kind.KwNull, Kind.KwTrue, Kind.KwFalse, Kind.Int, Kind.Float, Kind.String, Kind.Char => return ast.LiteralPat(value: &this.parsePrimary())
      Kind.Minus => {
        this.next()
        val x = this.parsePrimary()
        return ast.LiteralPat(value: &ast.UnaryExpr(op: Kind.Minus, x: &x, pos: this.spanFrom(start)))
      }
      Kind.Ident => {
        if (binding) {
          // `Some(x)` inside a destructuring is a nested variant; a plain name binds
          if (this.peek(1).kind == Kind.LParen || (this.peek(1).kind == Kind.Dot && this.peek(2).kind == Kind.Ident)) return this.parseTypePatternRest(start)
          val t = this.next()
          return ast.BindPat(name: ast.Ident(name: t.text, pos: t.span))
        }
        // `Variant(…)` or `Sealed.Variant(…)` destructures; a bare name is a
        // constant (or a variant without fields, the checker decides)
        var i: i64 = 0
        loop (this.peek(i).kind == Kind.Ident && this.peek(i + 1).kind == Kind.Dot) i += 2
        if (this.peek(i).kind == Kind.Ident && this.peek(i + 1).kind == Kind.LParen) return this.parseTypePatternRest(start)
        if (this.peek(1).kind == Kind.FatArrow) {
          // `Name => …`: the name is the pattern, not a lambda
          val t = this.next()
          return ast.LiteralPat(value: &ast.NameExpr(name: t.text, pos: t.span))
        }
        return ast.LiteralPat(value: &this.parsePostfix())
      }
      else => { }
    }
    this.errorAt(this.span(), "expected a pattern, found ${this.cur().describe()}")
    ast.LiteralPat(value: &ast.BadExpr(pos: this.span()))
  }

  // `..`, `..name` or `.._` in a list pattern, the cursor on the `..`
  fun parseRestPattern(): ast.Pattern {
    val start = this.next().span
    var rest: ast.RestPat = ast.RestPat()
    if (this.at(Kind.Ident)) {
      val t = this.next()
      rest.name = ast.Ident(name: t.text, pos: t.span)
    } else {
      this.accept(Kind.Under)  // `.._` is `..`
    }
    rest.pos = this.spanFrom(start)
    rest
  }

  // after `is`: a type and an optional destructuring list
  fun parseTypePatternRest(start: Span): ast.TypePat = this.parseTypePatternFields(this.parseType(), start)

  fun parseTypePatternFields(t: ast.Type, start: Span): ast.TypePat {
    var tp: ast.TypePat = ast.TypePat(typ: &t)
    if (this.accept(Kind.LParen)) {
      tp.hasArg = true
      val fields: MutableList<ast.FieldPat> = []
      loop (!this.atAny(Kind.RParen, Kind.EOF)) {
        fields.push(this.parseFieldPattern())
        if (!this.accept(Kind.Comma)) break
      }
      this.expect(Kind.RParen)
      tp.fields = fields.toList()
    }
    tp.pos = this.spanFrom(start)
    tp
  }

  // `name: pattern`, `name` (bound under the field's name), or a positional
  // sub-pattern, matched to the variant's one field
  fun parseFieldPattern(): ast.FieldPat {
    var fp = ast.FieldPat()
    val after = this.peek(1).kind
    if (this.at(Kind.Ident) && (after == Kind.Colon || after == Kind.Comma || after == Kind.RParen)) {
      val id = this.next()
      fp.name = ast.Ident(name: id.text, pos: id.span)
      if (after != Kind.Colon) return fp
      this.next()  // :
    }
    fp.pat = &this.parsePattern(true)
    fp
  }
}
