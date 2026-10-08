// The tree as an indented S-expression, as `veles parse` prints it. The
// parser is checked against this text (gates G2 and G3), so every character
// of it matters, the quoting of strings included.
use utf8
use lexer { PRINT, kindText }

/// Every declaration of a file, one after another.
public fun dump(file: SourceFile): string {
  var p = Printer()
  loop (d in file.decls) {
    p.decl(d)
    p.nl()
  }
  p.out.toString()
}

public fun dumpDecl(d: Decl): string {
  var p = Printer()
  p.decl(d)
  p.out.toString()
}

public fun dumpExpr(e: Expr): string {
  var p = Printer()
  p.expr(e)
  p.out.toString()
}

public fun typeString(t: Type): string {
  var p = Printer()
  p.typ(t)
  p.out.toString()
}

// whether r is printable: a letter, mark, number, punctuation or symbol, or
// the ASCII space
fun isPrint(r: i64): bool {
  if (r < 0x80) return r >= 0x20 && r < 0x7F
  val i = PRINT.partitionPoint(range => range.1 < r)
  val (first, _) = PRINT.at(i) ?: return false
  first <= r
}

fun hex(n: i64, width: i64): string => n.toString(radix: 16).padStart(width, "0")

/// `text` in double quotes, as the tree shows a string: with
/// `"` and `\` escaped, the C escapes for the control characters that have
/// one, `\xNN` for the other ASCII ones, and `\uNNNN` / `\UNNNNNNNN` for any
/// character above ASCII that is not printable.
public fun quote(text: string): string {
  val out = StringBuilder()
  out.append("\"")
  var i: i64 = 0
  loop (i < text.len()) {
    val rune = utf8.decode(text, i) ?: panic("quote: a string is valid UTF-8 at every character start")
    val r = rune.code
    i += rune.size
    if (r == '"'.toI64() || r == '\\'.toI64()) {
      out.append("\\")
      out.append(utf8.char(r))
      continue
    }
    if (isPrint(r)) {
      out.append(utf8.char(r))
      continue
    }
    when (r) {
      0x07 => out.append("\\a")
      0x08 => out.append("\\b")
      0x0C => out.append("\\f")
      0x0A => out.append("\\n")
      0x0D => out.append("\\r")
      0x09 => out.append("\\t")
      0x0B => out.append("\\v")
      else => {
        if (r < 0x20 || r == 0x7F) {
          out.append("\\x${hex(r, 2)}")
        } else if (r < 0x10000) {
          out.append("\\u${hex(r, 4)}")
        } else {
          out.append("\\U${hex(r, 8)}")
        }
      }
    }
  }
  out.append("\"")
  out.toString()
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

  fun close() {
    this.indent -= 1
    this.write(")")
  }

  fun attrs(attrs: List<Attribute>) {
    loop (a in attrs) {
      this.write("@${a.name.name}")
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

  fun decl(d: Decl) {
    when (d) {
      is UseDecl          => {
        this.write("(use")
        loop (s in d.specs) {
          this.write(" ")
          this.path(s.path)
          val alias = s.alias
          if (alias != null) this.write(" as ${alias.name}")
          val names = s.names ?: []
          if (!names.isEmpty()) {
            this.write(" {")
            loop ((i, n) in names.enumerate()) {
              if (i > 0) this.write(",")
              this.write(" ${n.name.name}")
              if (val nameAlias = n.alias) this.write(" as ${nameAlias.name}")
            }
            this.write(" }")
          }
        }
        this.write(")")
      }
      is FunDeclNode      => this.funDecl(d.decl)
      is SuiteDecl        => {
        this.open("suite ${quote(d.name)}")
        loop (inner in d.decls) {
          this.nl()
          this.decl(inner)
        }
        this.close()
      }
      is StaticAssertDecl => this.staticAssert(d.assert)
      is TestDecl         => {
        this.open("test ${quote(d.name)}")
        if (val body = d.body) {
          this.nl()
          this.block(body)
        }
        this.close()
      }
      is TypeAliasDecl    => {
        this.attrs(d.attrs)
        this.open("type ")
        if (d.pub) this.write("public ")
        this.write(d.name.name)
        this.typeParams(d.typeParams)
        this.write(" = ")
        this.optType(d.typ)
        this.close()
      }
      is ErrorAliasDecl   => {
        this.attrs(d.attrs)
        this.open("error ")
        if (d.pub) this.write("public ")
        this.write(d.name.name + " = ")
        this.optType(d.members)
        this.close()
      }
      is EnumDecl         => {
        this.attrs(d.attrs)
        this.open("enum ")
        if (d.pub) this.write("public ")
        this.write(d.name.name)
        if (val base = d.base) {
          this.write(" : ")
          this.typ(*base)
        }
        loop (m in d.members) {
          this.nl()
          this.write("(member ")
          this.attrs(m.attrs)
          this.write(m.name.name)
          if (val value = m.value) {
            this.write(" = ")
            this.expr(*value)
          }
          this.write(")")
        }
        this.close()
      }
      is StructDecl       => this.structDecl(d)
      is TraitDecl        => {
        this.attrs(d.attrs)
        this.open("trait ")
        if (d.pub) this.write("public ")
        if (d.isSealed) this.write("sealed ")
        this.write(d.name.name)
        this.typeParams(d.typeParams)
        loop ((i, s) in d.supers.enumerate()) {
          this.write(if (i == 0) " : " else " + ")
          this.typ(s)
        }
        loop (at in d.assocTypes) {
          this.nl()
          this.write("(type ${at.name.name}")
          loop ((i, b) in at.bounds.enumerate()) {
            this.write(if (i == 0) ": " else " + ")
            this.typ(b)
          }
          this.write(")")
        }
        loop (m in d.methods) {
          this.nl()
          this.funDecl(m)
        }
        this.close()
      }
      is ImplDecl         => this.implDecl(d)
      is ValDecl          => {
        this.attrs(d.attrs)
        this.write("(")
        if (d.pub) this.write("public ")
        this.write("${bindWord(d.kind)} ${d.name.name}")
        if (val typ = d.typ) {
          this.write(": ")
          this.typ(*typ)
        }
        if (val value = d.value) {
          this.write(" = ")
          this.expr(*value)
        }
        this.write(")")
      }
      is ExternBlock      => {
        this.open("extern ${quote(d.abi)}")
        loop (fn in d.funs) {
          this.nl()
          this.funDecl(fn)
        }
        this.close()
      }
      is BadDecl          => this.write("(bad-decl)")
    }
  }

  fun structDecl(d: StructDecl) {
    this.attrs(d.attrs)
    this.open(if (d.isError) "error " else if (d.union) "union " else "struct ")
    if (d.pub) this.write("public ")
    if (d.isExtern) this.write("extern ")
    this.write(d.name.name)
    this.typeParams(d.typeParams)
    if (val variant = d.variant) {
      this.write(" : ")
      this.typ(*variant)
    }
    loop (field in d.fields) {
      this.nl()
      this.write("(field ")
      this.attrs(field.attrs)
      if (field.pub) this.write("public ")
      if (field.isPrivate) this.write("private ")
      if (field.isInternal) this.write("internal ")
      if (field.isProtected) {
        this.write("protected var ")
      } else if (field.isVar) {
        this.write("var ")
      } else if (field.isVal) {
        this.write("val ")
      }
      this.write(field.name.name + ": ")
      this.optType(field.typ)
      if (val value = field.defaultValue) {
        this.write(" = ")
        this.expr(*value)
      }
      this.write(")")
    }
    loop (m in d.methods) {
      this.nl()
      this.funDecl(m)
    }
    loop (s in d.statics) {
      this.nl()
      this.write("(static ${s.name.name} ")
      this.optExpr(s.value)
      this.write(")")
    }
    if (val init = d.init) {
      this.nl()
      this.write("(init ")
      if (val params = d.initParams) {
        this.paramList(params)
        this.write(" ")
      }
      this.block(init)
      this.write(")")
    }
    this.close()
  }

  fun implDecl(d: ImplDecl) {
    this.attrs(d.attrs)
    if (d.extend) {
      this.open("extend")
      this.typeParams(d.typeParams)
      this.write(" ")
      this.optType(d.target)
    } else {
      this.open("impl")
      this.typeParams(d.typeParams)
      this.write(" ")
      this.optType(d.traitType)
      this.write(" for ")
      this.optType(d.target)
    }
    loop (at in d.assocTypes) {
      this.nl()
      this.write("(type ${at.name.name} = ")
      this.optType(at.typ)
      this.write(")")
    }
    loop (m in d.methods) {
      this.nl()
      this.funDecl(m)
    }
    this.close()
  }

  fun funDecl(d: FunDecl) {
    this.attrs(d.attrs)
    this.open("fun ")
    if (d.pub) this.write("public ")
    if (d.isPrivate) this.write("private ")
    if (d.isStatic) this.write("static ")
    if (d.isOverride) this.write("override ")
    if (d.exportC) this.write("extern-c ")
    if (d.isTest) this.write("test ")
    if (d.isUnsafe) this.write("unsafe ")
    if (d.isConst) this.write("const ")
    this.write(d.name.name)
    this.typeParams(d.typeParams)
    this.paramList(d.params)
    if (d.cVariadic) this.write(" ...")
    if (val ret = d.ret) {
      this.write(": ")
      this.typ(*ret)
    }
    this.effects(d.effects)
    val exprBody = d.exprBody
    val body = d.body
    if (exprBody != null) {
      this.nl()
      this.write("= ")
      this.expr(*exprBody)
    } else if (body != null) {
      this.nl()
      this.block(body)
    }
    this.close()
  }

  fun path(path: List<Ident>) {
    loop ((i, id) in path.enumerate()) {
      if (i > 0) this.write(".")
      this.write(id.name)
    }
  }

  // a missing type prints as `()`
  fun optType(t: (*Type)?) {
    val typ = t ?: return this.write("()")
    this.typ(*typ)
  }

  fun typeList(types: List<Type>) {
    loop ((i, t) in types.enumerate()) {
      if (i > 0) this.write(", ")
      this.typ(t)
    }
  }

  fun typ(t: Type) {
    when (t) {
      is NamedType      => {
        this.path(t.path)
        if (!t.args.isEmpty()) {
          this.write("<")
          this.typeList(t.args)
          this.write(">")
        }
      }
      is NullableType   => {
        val elem = *t.elem
        val wrap = elem is PointerType || elem is FunType
        if (wrap) this.write("(")
        this.typ(elem)
        if (wrap) this.write(")")
        this.write("?")
      }
      is ConstType      => this.expr(*t.x)
      is PointerType    => {
        this.write("*")
        if (t.isRaw) this.write("raw ")
        this.typ(*t.elem)
      }
      is TupleType      => {
        this.write("(")
        this.typeList(t.elems)
        this.write(")")
      }
      is FunType        => {
        if (t.sendable) this.write("sendable ")
        if (t.c) this.write("extern ")
        this.write("fun(")
        this.typeList(t.params)
        this.write(")")
        if (val ret = t.ret) {
          this.write(": ")
          this.typ(*ret)
        }
        this.effects(t.effects)
      }
      is SelfType       => this.write("Self")
      is AssocType      => {
        this.typ(*t.base)
        this.write("." + t.name.name)
      }
      is ErrorUnionType => {
        loop ((i, m) in t.members.enumerate()) {
          if (i > 0) this.write(" | ")
          this.typ(m)
        }
      }
    }
  }

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
      is LoopStmt         => {
        this.open("loop")
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
      is TupleExpr      => this.list("(tuple", e.elems)
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
        this.nl()
        this.block(e.body)
        this.close()
      }
      is RaceExpr       => {
        this.open("race")
        loop (arm in e.arms) {
          this.nl()
          this.write("(")
          if (val binding = arm.binding) {
            this.write("val ")
            this.binding(binding)
            this.write(" = ")
          }
          this.expr(*arm.source)
          this.write(" => ")
          this.expr(*arm.body)
          this.write(")")
        }
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
