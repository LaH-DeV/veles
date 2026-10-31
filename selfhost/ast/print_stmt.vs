use lexer { kindText }

extend Printer {
  fun block(b: Block) {
    this.open("block")
    loop (s in b.stmts) {
      this.nl()
      this.stmt(s)
    }
    this.close()
  }

  fun binding(b: Binding) {
    if (val name = b.name) {
      if (b.ref) this.write("&")
      this.write(name.name)
      if (val typ = b.typ) {
        this.write(": ")
        this.typ(*typ)
      }
      return
    }
    this.write("(")
    loop ((i, e) in b.tuple.enumerate()) {
      if (i > 0) this.write(", ")
      this.binding(e)
    }
    this.write(")")
  }

  fun staticAssert(a: StaticAssert) {
    this.write("(static-assert ")
    this.expr(*a.cond)
    if (val reason = a.reason) {
      this.write(" ")
      this.expr(*reason)
    }
    this.write(")")
  }

  fun stmt(s: Stmt) {
    when (s) {
      is StaticAssertStmt => this.staticAssert(s.assert)
      is BlockStmt        => this.block(s.block)
      is ValStmt          => {
        this.write("(${bindWord(s.kind)} ")
        val pattern = s.pattern
        if (pattern != null) this.pattern(*pattern) else this.binding(s.binding)
        if (val value = s.value) {
          this.write(" = ")
          this.expr(*value)
        }
        if (val orElse = s.orElse) {
          this.write(" else ")
          this.handler(orElse)
        }
        this.write(")")
      }
      is ExprStmt         => this.expr(*s.x)
      is AssignStmt       => {
        this.write("(${kindText(s.op)} ")
        this.expr(*s.target)
        this.write(" ")
        this.expr(*s.value)
        this.write(")")
      }
      is ReturnStmt       => {
        this.write("(return")
        if (val value = s.value) {
          this.write(" ")
          this.expr(*value)
        }
        this.write(")")
      }
      is ThrowStmt        => {
        this.write("(throw ")
        this.optExpr(s.value)
        this.write(")")
      }
      is BreakStmt        => {
        this.write("(break")
        if (val label = s.label) this.write(" ${label.name}")
        this.write(")")
      }
      is ContinueStmt     => {
        this.write("(continue")
        if (val label = s.label) this.write(" ${label.name}")
        this.write(")")
      }
      is CedeStmt         => this.write("(cede)")
      is LoopStmt         => {
        this.open(if (s.isInline) "inline-loop" else "loop")
        if (val label = s.label) this.write(" @${label.name}")
        val bound = s.bound
        val cond = s.cond
        if (bound != null) {
          this.write(" (")
          this.binding(bound)
          this.write(" in ")
          this.optExpr(s.iter)
          this.write(")")
        } else if (cond != null) {
          this.write(" (")
          this.expr(*cond)
          this.write(")")
        }
        this.nl()
        this.block(s.body)
        this.close()
      }
      is ScopeStmt        => {
        this.open("scope")
        this.on(s.on, s.workers)
        this.nl()
        this.block(s.body)
        this.close()
      }
      is WithStmt         => {
        this.write("(with-stmt ")
        if (!s.binding.name.name.isEmpty()) this.write(s.binding.name.name + " ")
        this.expr(*s.binding.value)
        this.write(")")
      }
      is FunStmt          => this.funDecl(*s.decl)
      is BadStmt          => this.write("(bad-stmt)")
    }
  }
}
