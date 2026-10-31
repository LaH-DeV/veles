// The tree as an indented S-expression, as `veles parse` prints it. The
// parser is checked against this text (gates G2 and G3), so every character
// of it matters, the quoting of strings included.

/// Every declaration of a file, one after another.
fun dump(file: SourceFile): string {
  var p = Printer()
  loop (d in file.decls) {
    p.decl(d)
    p.nl()
  }
  p.out.toString()
}

fun dumpDecl(d: Decl): string {
  var p = Printer()
  p.decl(d)
  p.out.toString()
}

fun dumpExpr(e: Expr): string {
  var p = Printer()
  p.expr(e)
  p.out.toString()
}

fun typeString(t: Type): string {
  var p = Printer()
  p.typ(t)
  p.out.toString()
}

struct Printer {
  out:        StringBuilder = StringBuilder()
  var indent: i64 = 0

  fun write(text: string) {
    this.out.append(text)
  }

  fun nl() {
    this.out.append("\n")
    this.out.append("  ".repeat(this.indent))
  }

  fun open(name: string) {
    this.write("(" + name)
    this.indent += 1
  }

  // the arms of a `select` (or of the removed `race` with arms, D169)
  fun arms(arms: List<RaceArm>) {
    loop (arm in arms) {
      this.nl()
      this.write("(")
      if (val binding = arm.binding) {
        this.write("val ")
        this.binding(binding)
        this.write(" = ")
      }
      this.expr(*arm.source)
      if (val name = arm.name) this.write(" as ${name.name}")
      if (val body = arm.body) {
        this.write(" => ")
        this.expr(*body)
      }
      this.write(")")
    }
  }

  // the executor of a `scope(on: e)` or `gather(on: e)` (D143)
  // the `on:` and `workers:` of a scope or gather (D143, D156)
  fun on(e: (*Expr)?, workers: (*Expr)?) {
    if (val x = e) {
      this.write(" (on ")
      this.expr(*x)
      this.write(")")
    }
    if (val n = workers) {
      this.write(" (workers ")
      this.expr(*n)
      this.write(")")
    }
  }

  fun close() {
    this.indent -= 1
    this.write(")")
  }

  fun attrs(attrs: List<Attribute>) {
    loop (a in attrs) {
      this.write(if (a.qual.name.isEmpty()) "@${a.name.name}" else "@${a.qual.name}.${a.name.name}")
      if (!a.args.isEmpty()) {
        this.write("(")
        this.args(a.args)
        this.write(")")
      }
      this.write(" ")
    }
  }

  fun typeParams(tps: List<TypeParam>) {
    if (tps.isEmpty()) return
    this.write("<")
    loop ((i, tp) in tps.enumerate()) {
      if (i > 0) this.write(", ")
      if (tp.isConst) this.write("const ")
      this.write(tp.name.name)
      if (tp.isConst) {
        this.write(": ")
        this.optType(tp.of)
      }
      loop ((j, bound) in tp.bounds.enumerate()) {
        this.write(if (j == 0) ": " else " + ")
        this.typ(bound)
      }
    }
    this.write(">")
  }

  fun effects(e: Effects) {
    if (e.suspending) this.write(" suspends")
    if (e.throwing) {
      this.write(" throws")
      val error = e.error ?: return
      this.write(" ")
      this.typ(*error)
    }
  }

  fun pattern(pat: Pattern) {
    when (pat) {
      is WildcardPat => this.write("_")
      is BindPat     => this.write(pat.name.name)
      is LiteralPat  => this.expr(*pat.value)
      is RangePat    => {
        this.write("in ")
        this.rangeExpr(pat.range)
      }
      is TypePat     => {
        this.write("is ")
        this.typ(*pat.typ)
        if (pat.hasArg) {
          this.write("(")
          loop ((i, fp) in pat.fields.enumerate()) {
            if (i > 0) this.write(", ")
            this.write(fp.name.name)
            if (val inner = fp.pat) {
              this.write(": ")
              this.pattern(*inner)
            }
          }
          this.write(")")
        }
      }
      is TuplePat    => {
        this.write("(")
        loop ((i, el) in pat.elems.enumerate()) {
          if (i > 0) this.write(", ")
          this.pattern(el)
        }
        this.write(")")
      }
      is ListPat     => {
        this.write("[")
        loop ((i, el) in pat.elems.enumerate()) {
          if (i > 0) this.write(", ")
          this.pattern(el)
        }
        this.write("]")
      }
      is RestPat     => {
        this.write("..")
        if (val name = pat.name) this.write(name.name)
      }
    }
  }

  fun handler(h: Handler) {
    if (val err = h.err) {
      this.write("(${err.name} => ")
      this.block(h.body)
      this.write(")")
      return
    }
    this.block(h.body)
  }

  fun paramList(params: List<Param>) {
    this.write("(")
    loop ((i, param) in params.enumerate()) {
      if (i > 0) this.write(", ")
      this.write(param.name.name)
      if (val typ = param.typ) {
        this.write(": ")
        this.typ(*typ)
        if (param.variadic) this.write("...")
      }
      if (val value = param.defaultValue) {
        this.write(" = ")
        this.expr(*value)
      }
    }
    this.write(")")
  }
}
