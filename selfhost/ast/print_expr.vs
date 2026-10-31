use lexer { kindText }
use text { quote }

extend Printer {
  fun args(args: List<Arg>) {
    loop ((i, a) in args.enumerate()) {
      if (i > 0) this.write(", ")
      if (val name = a.name) this.write(name.name + ": ")
      this.expr(*a.value)
      if (a.spread) this.write("...")
    }
  }

  // a missing expression prints as `<nil>`
  fun optExpr(e: (*Expr)?) {
    val x = e ?: return this.write("<nil>")
    this.expr(*x)
  }

  fun typeArgs(types: List<Type>) {
    if (types.isEmpty()) return
    this.write("<")
    this.typeList(types)
    this.write(">")
  }

  fun expr(e: Expr) {
    when (e) {
      is IntLit         => this.write(e.text)
      is FloatLit       => this.write(e.text)
      is StringLit      => this.stringLit(e)
      is TemplateExpr   => {
        this.write("(template ")
        this.expr(*e.tag)
        this.write(" ")
        this.stringLit(e.lit)
        this.write(")")
      }
      is CharLit        => this.write("'${e.value}'")
      is RegexLit       => this.write("(regex ${quote(e.pattern)} ${quote(e.flags)})")
      is BoolLit        => this.write(if (e.value) "true" else "false")
      is NullLit        => this.write("null")
      is ThisExpr       => this.write("this")
      is NameExpr       => {
        this.write(e.name)
        this.typeArgs(e.typeArgs)
      }
      is MemberExpr     => {
        this.write(if (e.safe) "(?. " else "(. ")
        this.expr(*e.x)
        this.write(" ${e.name.name})")
      }
      is IndexExpr      => {
        this.write("(index ")
        this.expr(*e.x)
        this.write(" ")
        this.expr(*e.index)
        this.write(")")
      }
      is CallExpr       => {
        if (e.block) {
          // `async { … }` (D169): the callee is a lambda whose body is the block
          val lambda = *e.callee
          if (lambda is LambdaExpr) {
            val body = *lambda.body
            if (body is BlockExpr) {
              this.open("async-block")
              this.nl()
              this.block(body.block)
              this.close()
              return
            }
          }
        }
        this.write(if (e.isAsync) "(async-call " else "(call ")
        this.expr(*e.callee)
        this.typeArgs(e.typeArgs)
        if (!e.args.isEmpty()) {
          this.write(" ")
          this.args(e.args)
        }
        this.write(")")
      }
      is UnaryExpr      => {
        this.write("(${kindText(e.op)} ")
        this.expr(*e.x)
        this.write(")")
      }
      is BinaryExpr     => {
        this.write("(${kindText(e.op)} ")
        this.expr(*e.left)
        this.write(" ")
        this.expr(*e.right)
        this.write(")")
      }
      is ElvisExpr      => this.pair("?:", *e.left, *e.right)
      is OrFailExpr     => this.pair("?!", *e.left, *e.right)
      is CatchExpr      => {
        this.open("catch")
        val x = e.x
        val body = e.body
        this.nl()
        if (x != null) {
          this.expr(*x)
        } else if (body != null) {
          this.block(body)
        }
        this.write(" ")
        this.handler(e.handler)
        this.close()
      }
      is CoalesceExpr   => {
        this.write("(?? ")
        this.expr(*e.left)
        this.write(" ")
        val handler = e.handler
        if (handler != null) this.handler(handler) else this.optExpr(e.right)
        this.write(")")
      }
      is WithExpr       => {
        this.open("with")
        loop (b in e.bindings) {
          this.write(if (b.name.name.isEmpty()) " (" else " (${b.name.name} = ")
          this.expr(*b.value)
          this.write(")")
        }
        this.nl()
        this.block(e.body)
        this.close()
      }
      is RangeExpr      => this.rangeExpr(e)
      is LambdaExpr     => {
        if (e.bare) {
          this.write("(block-lambda ")
          this.expr(*e.body)
          this.write(")")
          return
        }
        this.write("(lambda (")
        loop ((i, param) in e.params.enumerate()) {
          if (i > 0) this.write(", ")
          if (val pattern = param.pattern) {
            this.binding(pattern)
            continue
          }
          this.write(param.name.name)
          if (val typ = param.typ) {
            this.write(": ")
            this.typ(*typ)
          }
        }
        this.write(")")
        if (val ret = e.ret) {
          this.write(": ")
          this.typ(*ret)
        }
        this.write(" ")
        this.expr(*e.body)
        this.write(")")
      }
      is TupleExpr      => {
        this.write("(tuple")
        loop ((i, elem) in e.elems.enumerate()) {
          this.write(" ")
          if (val label = e.labels.at(i)) this.write("${label.name}: ")
          this.expr(elem)
        }
        this.write(")")
      }
      is ListLit        => this.list(if (e.isMut) "(mut-list" else "(list", e.elems)
      is MapLit         => {
        this.write(if (e.isMut) "(mut-map" else "(map")
        loop (entry in e.entries) {
          this.write(" [")
          this.expr(*entry.key)
          this.write(": ")
          this.expr(*entry.value)
          this.write("]")
        }
        this.write(")")
      }
      is IfExpr         => {
        this.open("if ")
        this.expr(*e.cond)
        this.nl()
        this.block(e.then)
        if (val orElse = e.orElse) {
          this.nl()
          this.write("else ")
          this.block(orElse)
        }
        this.close()
      }
      is WhenExpr       => this.whenExpr(e)
      is BlockExpr      => this.block(e.block)
      is TryExpr        => this.single("try", *e.x)
      is AwaitExpr      => this.single("await", *e.x)
      is LetCond        => {
        this.write("(val ${e.name.name} ")
        this.expr(*e.value)
        this.write(")")
      }
      is IsExpr         => {
        this.write(if (e.not) "(!is " else "(is ")
        this.expr(*e.x)
        this.write(" ")
        this.typ(*e.pat.typ)
        if (e.pat.hasArg) this.write("(...)")
        this.write(")")
      }
      is InExpr         => {
        this.write(if (e.not) "(!in " else "(in ")
        this.expr(*e.x)
        this.write(" ")
        this.expr(*e.c)
        this.write(")")
      }
      is ImplementsExpr => {
        this.write("(implements ")
        this.typ(*e.typ)
        this.write(" ")
        this.typ(*e.traitType)
        this.write(")")
      }
      is CastExpr       => {
        this.write("(as ")
        this.expr(*e.x)
        this.write(" ")
        this.typ(*e.typ)
        this.write(")")
      }
      is GatherExpr     => {
        this.open("gather")
        this.on(e.on, e.workers)
        this.nl()
        this.block(e.body)
        this.close()
      }
      is ScopeExpr      => {
        this.open("scope-value")
        this.on(e.on, e.workers)
        this.nl()
        this.block(e.body)
        this.close()
      }
      is RaceExpr       => {
        this.open("race")
        if (val body = e.body) {
          this.nl()
          this.block(body)
        }
        this.arms(e.arms)
        this.close()
      }
      is SelectExpr     => {
        this.open("select")
        this.arms(e.arms)
        this.close()
      }
      is UnsafeExpr     => {
        this.open("unsafe")
        this.nl()
        this.block(e.body)
        this.close()
      }
      is ControlExpr    => this.stmt(*e.stmt)
      is BadExpr        => this.write("(bad-expr)")
    }
  }

  fun stringLit(e: StringLit) {
    this.write("(str")
    loop (part in e.parts) {
      val x = part.expr
      if (x != null) {
        this.write(" \${")
        this.expr(*x)
        if (part.hasSpec) this.write(":${quote(part.spec)}")
        this.write("}")
      } else {
        this.write(" ${quote(part.text)}")
      }
    }
    this.write(")")
  }

  fun rangeExpr(e: RangeExpr) {
    this.write(if (e.inclusive) "(.. " else "(..< ")
    this.expr(*e.lo)
    this.write(" ")
    this.expr(*e.hi)
    this.write(")")
  }

  fun pair(op: string, left: Expr, right: Expr) {
    this.write("($op ")
    this.expr(left)
    this.write(" ")
    this.expr(right)
    this.write(")")
  }

  fun single(op: string, x: Expr) {
    this.write("($op ")
    this.expr(x)
    this.write(")")
  }

  fun list(head: string, elems: List<Expr>) {
    this.write(head)
    loop (el in elems) {
      this.write(" ")
      this.expr(el)
    }
    this.write(")")
  }

  fun whenExpr(e: WhenExpr) {
    this.open("when")
    if (val subject = e.subject) {
      this.write(" ")
      if (val bind = e.bind) this.write("val ${bind.name} = ")
      this.expr(*subject)
    }
    loop (arm in e.arms) {
      this.nl()
      this.write("(")
      val cond = arm.cond
      if (arm.isElse) {
        this.write("else")
      } else if (cond != null) {
        this.expr(*cond)
      } else {
        loop ((i, pat) in arm.patterns.enumerate()) {
          if (i > 0) this.write(", ")
          this.pattern(pat)
        }
      }
      if (val guard = arm.guard) {
        this.write(" if ")
        this.expr(*guard)
      }
      this.write(" => ")
      this.expr(*arm.body)
      this.write(")")
    }
    this.close()
  }
}
