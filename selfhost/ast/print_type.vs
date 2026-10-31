extend Printer {
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
        loop ((i, elem) in t.elems.enumerate()) {
          if (i > 0) this.write(", ")
          if (val label = t.labels.at(i)) this.write("${label.name}: ")
          this.typ(elem)
        }
        this.write(")")
      }
      is FunType        => {
        if (t.sendable) this.write("sendable ")
        if (t.c) this.write("extern ")
        this.write("fun")
        if (val on = t.on) {
          this.write(" on ")
          this.typ(*on)
        }
        this.write("(")
        this.typeList(t.params)
        this.write(")")
        if (val ret = t.ret) {
          this.write(": ")
          this.typ(*ret)
        }
        this.effects(t.effects)
      }
      is SelfType       => this.write("This")
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
}
