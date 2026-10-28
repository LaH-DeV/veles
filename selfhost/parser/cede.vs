// `cede` (D197) is a statement with nothing after it: no operand, no
// arguments, no value. The forms a reader brings from other languages —
// `cede()`, `await cede`, `val x = cede`, `fun f() => cede` — are each
// reported with what to write instead; where the statement is plainly what
// was meant (`=> cede` as a body) it is read as the block it should be, so
// nothing cascades. parser/cede.go says the same, word for word.
use ast
use lexer { Kind }
use source { Span }

extend Parser {
  // the statement at the cursor
  fun parseCede(): ast.Stmt {
    val t = this.next()
    this.cedeTrailer()
    ast.CedeStmt(pos: t.span)
  }

  // what follows `cede` on its line, reported and skipped: the statement
  // has ended. `else` may follow, in `if (c) cede else …`.
  fun cedeTrailer() {
    if (this.atAny(Kind.Semi, Kind.RBrace, Kind.EOF, Kind.KwElse)) return
    val t = this.cur()
    when {
      t.kind == Kind.LParen && this.peek(1).kind == Kind.RParen => {
        this.errorAt(t.span, "'cede' is a statement, not a function: write 'cede' without parentheses (D197)")
      }
      t.kind == Kind.LParen => {
        this.errorAt(t.span, "'cede' is a statement, not a function, and takes no arguments: write 'cede' alone (D197)")
      }
      usesAsValue(t.kind) => {
        this.errorAt(t.span, "'cede' is a statement and has no value: ${t.describe()} cannot follow it (D197)")
      }
      else => this.errorAt(t.span, "'cede' takes no operand, label, arguments or type arguments: write 'cede' alone (D197)")
    }
    this.skipToTerminator()
  }

  // the rest of a statement, up to the `;`, newline or `}` that ends it,
  // over nesting; the caller's loop takes the end
  fun skipToTerminator() {
    var depth: i64 = 0
    loop (!this.at(Kind.EOF)) {
      when (this.cur().kind) {
        Kind.LBrace, Kind.LParen, Kind.LBracket => depth += 1
        Kind.RBrace, Kind.RParen, Kind.RBracket => {
          if (depth == 0) return
          depth -= 1
        }
        Kind.Semi => {
          if (depth == 0) return
        }
        else => { }
      }
      this.next()
    }
  }

  // `cede` where a value goes — `val x = cede`, `f(cede)`, `return cede`
  fun cedeExpr(): ast.Expr {
    val t = this.next()
    this.errorAt(t.span, "'cede' is a statement and has no value: write it where a statement goes, on a line of its own (D197)")
    ast.BadExpr(pos: t.span)
  }

  // `await cede`, `try cede` and `async cede`: the prefix at start, `cede`
  // at the cursor
  fun cedePrefixed(start: Span, prefix: Kind): ast.Expr {
    val t = this.next()
    val message = when (prefix) {
      Kind.KwAwait => "'cede' is a statement that already marks where the task may wait: remove 'await' (D197)"
      Kind.KwTry   => "'cede' is a statement and cannot fail: remove 'try' (D197)"
      else         => "'async' starts a task from a call or a block, and 'cede' is neither: write 'async { cede }' (D197)"
    }
    this.errorAt(start, message)
    ast.BadExpr(pos: start.to(t.span))
  }

  // `fun name(…) => cede`: reported, and read as the block body `{ cede }`
  fun cedeFunBody(name: string): ast.Block {
    val t = this.next()
    this.errorAt(t.span, "'cede' is a statement and has no value, so it cannot be an expression body: write 'fun $name(…) { cede }' (D197)")
    ast.Block(stmts: [ast.CedeStmt(pos: t.span)], pos: t.span)
  }

  // `=> cede` after a lambda's parameters or a `when` or `race` arm:
  // reported, and read as the block `=> { cede }`
  fun cedeArmBody(): ast.Expr {
    val t = this.next()
    this.errorAt(t.span, "'cede' is a statement and has no value: write a block, '=> { cede }' (D197)")
    ast.BlockExpr(block: ast.Block(stmts: [ast.CedeStmt(pos: t.span)], pos: t.span))
  }
}

// a token that would use what is before it as a value: a member access,
// an operator, an assignment
fun usesAsValue(k: Kind): bool => when (k) {
  Kind.Dot, Kind.SafeDot, Kind.Elvis, Kind.OrFail, Kind.Coalesce, Kind.Question, Kind.Range, Kind.RangeLt,
    Kind.Plus, Kind.Minus, Kind.Star, Kind.Slash, Kind.Percent, Kind.WrapPlus, Kind.WrapMinus, Kind.WrapStar,
    Kind.Eq, Kind.NotEq, Kind.LtEq, Kind.Gt, Kind.GtEq, Kind.AndAnd, Kind.OrOr, Kind.Caret, Kind.Shl, Kind.Shr,
    Kind.Pipe, Kind.Amp, Kind.KwAs, Kind.KwIs, Kind.KwIn, Kind.KwCatch, Kind.Assign, Kind.PlusEq, Kind.MinusEq,
    Kind.StarEq, Kind.SlashEq, Kind.PercentEq, Kind.PipeEq, Kind.AmpEq, Kind.CaretEq, Kind.ShlEq, Kind.ShrEq,
    Kind.WrapPlusEq, Kind.WrapMinusEq, Kind.WrapStarEq => true
  else => false
}
