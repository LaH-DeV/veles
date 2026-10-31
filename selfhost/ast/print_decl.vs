use text { quote }

extend Printer {
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
          if (s.extend) {
            this.write(" extend")
            val extensions = s.extendNames ?: []
            if (!extensions.isEmpty()) {
              this.write(" {")
              loop ((i, n) in extensions.enumerate()) {
                if (i > 0) this.write(",")
                this.write(" ${n.name.name}")
              }
              this.write(" }")
            }
          }
        }
        this.write(")")
      }
      is FunDeclNode      => this.funDecl(d.decl)
      is SuiteDecl        => {
        this.attrs(d.attrs)
        this.open("suite ${quote(title(d.label, d.name))}")
        loop (inner in d.decls) {
          this.nl()
          this.decl(inner)
        }
        this.close()
      }
      is StaticAssertDecl => this.staticAssert(d.assert)
      is SkipDecl         => {
        this.open("skip")
        this.nl()
        this.expr(*d.call)
        this.close()
      }
      is TestDecl         => {
        this.attrs(d.attrs)
        this.open("test ${quote(title(d.label, d.name))}")
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
        if (d.indirect) this.write("indirect ")
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
        if (d.isLazy) this.write("lazy ")
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
      is ExecutorDecl     => {
        this.open("executor ")
        if (d.pub) this.write("public ")
        this.write(d.name.name)
        if (!d.args.isEmpty()) {
          this.write(" (")
          this.args(d.args)
          this.write(")")
        }
        if (d.hasBody) this.write(" {}")
        loop (s in d.state) {
          this.nl()
          this.decl(s)
        }
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
    this.open(when {
      d.isError            => "error "
      d.union              => "union "
      d.annotation != null => "annotation "
      else                 => "struct "
    })
    if (d.pub) this.write("public ")
    if (d.isConst) this.write("const ")
    if (d.isExtern) this.write("extern ")
    if (d.indirect) this.write("indirect ")
    if (d.isInline) this.write("inline ")
    if (d.compact) this.write("compact ")
    this.write(d.name.name)
    this.typeParams(d.typeParams)
    if (val variant = d.variant) {
      this.write(" : ")
      this.typ(*variant)
    }
    if (val info = d.annotation) {
      this.write(" on")
      loop (t in info.targets) this.write(" " + t.name)
      loop ((i, t) in info.readers.enumerate()) {
        this.write(if (i == 0) " for " else ", ")
        this.typ(t)
      }
      if (val wrapper = info.wrapper) {
        this.write(" = ")
        this.expr(*wrapper)
      }
    }
    loop (field in d.fields) {
      this.nl()
      this.write("(field ")
      this.attrs(field.attrs)
      if (field.pub) this.write("public ")
      if (field.isPrivate) this.write("private ")
      if (field.isInternal) this.write("internal ")
      when {
        field.isProtected => {
          this.write("protected var ")
        }
        field.isVar       => {
          this.write("var ")
        }
        field.isVal       => {
          this.write("val ")
        }
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
      this.write(if (s.isLazy) "(static lazy ${s.name.name} " else "(static ${s.name.name} ")
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
      if (d.pub) this.write(" public") else if (d.isInternal) this.write(" internal")
      if (d.withFun) this.write(" with")
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
    if (d.template) this.write("template ")
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
}
