// Generated from ast/ast.vs by `go test ./selfhost -run SpanWalker -update`; edit the source, not this file.

/// Every span of the tree, one line each: `Node.field start end` — what the
/// span gate compares with the bootstrap parser's tree.
fun spans(file: SourceFile): List<string> {
  val out: MutableList<string> = []
  spansOfSourceFile(file, out)
  out.toList()
}

fun spansOfType(node: Type, out: MutableList<string>) {
  when (node) {
    is NamedType      => spansOfNamedType(node, out)
    is NullableType   => spansOfNullableType(node, out)
    is PointerType    => spansOfPointerType(node, out)
    is TupleType      => spansOfTupleType(node, out)
    is FunType        => spansOfFunType(node, out)
    is ConstType      => spansOfConstType(node, out)
    is SelfType       => spansOfSelfType(node, out)
    is AssocType      => spansOfAssocType(node, out)
    is ErrorUnionType => spansOfErrorUnionType(node, out)
  }
}

fun spansOfDecl(node: Decl, out: MutableList<string>) {
  when (node) {
    is UseDecl          => spansOfUseDecl(node, out)
    is FunDeclNode      => spansOfFunDeclNode(node, out)
    is StructDecl       => spansOfStructDecl(node, out)
    is TraitDecl        => spansOfTraitDecl(node, out)
    is ImplDecl         => spansOfImplDecl(node, out)
    is ValDecl          => spansOfValDecl(node, out)
    is ExternBlock      => spansOfExternBlock(node, out)
    is ExecutorDecl     => spansOfExecutorDecl(node, out)
    is ErrorAliasDecl   => spansOfErrorAliasDecl(node, out)
    is TypeAliasDecl    => spansOfTypeAliasDecl(node, out)
    is EnumDecl         => spansOfEnumDecl(node, out)
    is TestDecl         => spansOfTestDecl(node, out)
    is SuiteDecl        => spansOfSuiteDecl(node, out)
    is SkipDecl         => spansOfSkipDecl(node, out)
    is StaticAssertDecl => spansOfStaticAssertDecl(node, out)
    is BadDecl          => spansOfBadDecl(node, out)
  }
}

fun spansOfStmt(node: Stmt, out: MutableList<string>) {
  when (node) {
    is BlockStmt        => spansOfBlockStmt(node, out)
    is ValStmt          => spansOfValStmt(node, out)
    is ExprStmt         => spansOfExprStmt(node, out)
    is AssignStmt       => spansOfAssignStmt(node, out)
    is ReturnStmt       => spansOfReturnStmt(node, out)
    is ThrowStmt        => spansOfThrowStmt(node, out)
    is BreakStmt        => spansOfBreakStmt(node, out)
    is ContinueStmt     => spansOfContinueStmt(node, out)
    is CedeStmt         => spansOfCedeStmt(node, out)
    is LoopStmt         => spansOfLoopStmt(node, out)
    is WithStmt         => spansOfWithStmt(node, out)
    is ScopeStmt        => spansOfScopeStmt(node, out)
    is FunStmt          => spansOfFunStmt(node, out)
    is StaticAssertStmt => spansOfStaticAssertStmt(node, out)
    is BadStmt          => spansOfBadStmt(node, out)
  }
}

fun spansOfExpr(node: Expr, out: MutableList<string>) {
  when (node) {
    is IntLit         => spansOfIntLit(node, out)
    is FloatLit       => spansOfFloatLit(node, out)
    is StringLit      => spansOfStringLit(node, out)
    is TemplateExpr   => spansOfTemplateExpr(node, out)
    is CharLit        => spansOfCharLit(node, out)
    is RegexLit       => spansOfRegexLit(node, out)
    is BoolLit        => spansOfBoolLit(node, out)
    is NullLit        => spansOfNullLit(node, out)
    is ThisExpr       => spansOfThisExpr(node, out)
    is NameExpr       => spansOfNameExpr(node, out)
    is MemberExpr     => spansOfMemberExpr(node, out)
    is IndexExpr      => spansOfIndexExpr(node, out)
    is CallExpr       => spansOfCallExpr(node, out)
    is UnaryExpr      => spansOfUnaryExpr(node, out)
    is BinaryExpr     => spansOfBinaryExpr(node, out)
    is OrFailExpr     => spansOfOrFailExpr(node, out)
    is CoalesceExpr   => spansOfCoalesceExpr(node, out)
    is CatchExpr      => spansOfCatchExpr(node, out)
    is ElvisExpr      => spansOfElvisExpr(node, out)
    is RangeExpr      => spansOfRangeExpr(node, out)
    is LambdaExpr     => spansOfLambdaExpr(node, out)
    is TupleExpr      => spansOfTupleExpr(node, out)
    is ListLit        => spansOfListLit(node, out)
    is MapLit         => spansOfMapLit(node, out)
    is IfExpr         => spansOfIfExpr(node, out)
    is LetCond        => spansOfLetCond(node, out)
    is WhenExpr       => spansOfWhenExpr(node, out)
    is BlockExpr      => spansOfBlockExpr(node, out)
    is TryExpr        => spansOfTryExpr(node, out)
    is AwaitExpr      => spansOfAwaitExpr(node, out)
    is IsExpr         => spansOfIsExpr(node, out)
    is InExpr         => spansOfInExpr(node, out)
    is ImplementsExpr => spansOfImplementsExpr(node, out)
    is CastExpr       => spansOfCastExpr(node, out)
    is GatherExpr     => spansOfGatherExpr(node, out)
    is ScopeExpr      => spansOfScopeExpr(node, out)
    is RaceExpr       => spansOfRaceExpr(node, out)
    is SelectExpr     => spansOfSelectExpr(node, out)
    is UnsafeExpr     => spansOfUnsafeExpr(node, out)
    is WithExpr       => spansOfWithExpr(node, out)
    is ControlExpr    => spansOfControlExpr(node, out)
    is BadExpr        => spansOfBadExpr(node, out)
  }
}

fun spansOfPattern(node: Pattern, out: MutableList<string>) {
  when (node) {
    is WildcardPat => spansOfWildcardPat(node, out)
    is BindPat     => spansOfBindPat(node, out)
    is LiteralPat  => spansOfLiteralPat(node, out)
    is RangePat    => spansOfRangePat(node, out)
    is TypePat     => spansOfTypePat(node, out)
    is TuplePat    => spansOfTuplePat(node, out)
    is ListPat     => spansOfListPat(node, out)
    is RestPat     => spansOfRestPat(node, out)
  }
}

fun spansOfIdent(node: Ident, out: MutableList<string>) {
  out.push("Ident.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfSourceFile(node: SourceFile, out: MutableList<string>) {
  loop (x2 in node.decls) {
    spansOfDecl(x2, out)
  }
}

fun spansOfNamedType(node: NamedType, out: MutableList<string>) {
  loop (x1 in node.path) {
    spansOfIdent(x1, out)
  }
  loop (x3 in node.args) {
    spansOfType(x3, out)
  }
  out.push("NamedType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfNullableType(node: NullableType, out: MutableList<string>) {
  spansOfType(*node.elem, out)
  out.push("NullableType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfPointerType(node: PointerType, out: MutableList<string>) {
  spansOfType(*node.elem, out)
  out.push("PointerType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTupleType(node: TupleType, out: MutableList<string>) {
  loop (x1 in node.elems) {
    spansOfType(x1, out)
  }
  loop (x3 in node.labels) {
    spansOfIdent(x3, out)
  }
  out.push("TupleType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfFunType(node: FunType, out: MutableList<string>) {
  if (val x1 = node.on) spansOfType(*x1, out)
  loop (x2 in node.params) {
    spansOfType(x2, out)
  }
  if (val x4 = node.ret) spansOfType(*x4, out)
  spansOfEffects(node.effects, out)
  out.push("FunType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfConstType(node: ConstType, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  out.push("ConstType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfSelfType(node: SelfType, out: MutableList<string>) {
  out.push("SelfType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfAssocType(node: AssocType, out: MutableList<string>) {
  spansOfType(*node.base, out)
  spansOfIdent(node.name, out)
  out.push("AssocType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfErrorUnionType(node: ErrorUnionType, out: MutableList<string>) {
  loop (x1 in node.members) {
    spansOfType(x1, out)
  }
  out.push("ErrorUnionType.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfEffects(node: Effects, out: MutableList<string>) {
  out.push("Effects.suspendsSpan ${node.suspendsSpan.start} ${node.suspendsSpan.end}")
  out.push("Effects.throwsSpan ${node.throwsSpan.start} ${node.throwsSpan.end}")
  if (val x5 = node.error) spansOfType(*x5, out)
}

fun spansOfUseDecl(node: UseDecl, out: MutableList<string>) {
  loop (x1 in node.specs) {
    spansOfUseSpec(x1, out)
  }
  out.push("UseDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfFunDeclNode(node: FunDeclNode, out: MutableList<string>) {
  spansOfFunDecl(node.decl, out)
}

fun spansOfStructDecl(node: StructDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  if (val x9 = node.errorImpl) spansOfImplDecl(*x9, out)
  if (val x10 = node.annotation) {
    spansOfAnnotationInfo(x10, out)
  }
  spansOfIdent(node.name, out)
  loop (x13 in node.typeParams) {
    spansOfTypeParam(x13, out)
  }
  if (val x15 = node.variant) spansOfType(*x15, out)
  loop (x16 in node.fields) {
    spansOfField(x16, out)
  }
  loop (x18 in node.methods) {
    spansOfFunDecl(x18, out)
  }
  loop (x20 in node.statics) {
    spansOfValDecl(x20, out)
  }
  loop (x22 in node.impls) {
    spansOfImplDecl(x22, out)
  }
  if (val x24 = node.init) {
    spansOfBlock(x24, out)
  }
  out.push("StructDecl.initPos ${node.initPos.start} ${node.initPos.end}")
  if (val x27 = node.initParams) {
    loop (x28 in x27) {
      spansOfParam(x28, out)
    }
  }
  out.push("StructDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTraitDecl(node: TraitDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  loop (x8 in node.typeParams) {
    spansOfTypeParam(x8, out)
  }
  loop (x10 in node.supers) {
    spansOfType(x10, out)
  }
  loop (x12 in node.assocTypes) {
    spansOfAssocTypeDecl(x12, out)
  }
  loop (x14 in node.methods) {
    spansOfFunDecl(x14, out)
  }
  loop (x16 in node.variants) {
    spansOfStructDecl(x16, out)
  }
  out.push("TraitDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfImplDecl(node: ImplDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  loop (x7 in node.typeParams) {
    spansOfTypeParam(x7, out)
  }
  if (val x9 = node.traitType) spansOfType(*x9, out)
  if (val x10 = node.target) spansOfType(*x10, out)
  loop (x11 in node.assocTypes) {
    spansOfAssocTypeBinding(x11, out)
  }
  loop (x13 in node.methods) {
    spansOfFunDecl(x13, out)
  }
  out.push("ImplDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfValDecl(node: ValDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  if (val x9 = node.typ) spansOfType(*x9, out)
  if (val x10 = node.value) spansOfExpr(*x10, out)
  out.push("ValDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfExternBlock(node: ExternBlock, out: MutableList<string>) {
  loop (x2 in node.funs) {
    spansOfFunDecl(x2, out)
  }
  out.push("ExternBlock.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfExecutorDecl(node: ExecutorDecl, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  loop (x5 in node.args) {
    spansOfArg(x5, out)
  }
  out.push("ExecutorDecl.argsPos ${node.argsPos.start} ${node.argsPos.end}")
  loop (x8 in node.state) {
    spansOfValDecl(x8, out)
  }
  loop (x10 in node.funs) {
    spansOfFunDecl(x10, out)
  }
  out.push("ExecutorDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfErrorAliasDecl(node: ErrorAliasDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  if (val x7 = node.members) spansOfType(*x7, out)
  out.push("ErrorAliasDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTypeAliasDecl(node: TypeAliasDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  loop (x7 in node.typeParams) {
    spansOfTypeParam(x7, out)
  }
  if (val x9 = node.typ) spansOfType(*x9, out)
  out.push("TypeAliasDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfEnumDecl(node: EnumDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  if (val x7 = node.base) spansOfType(*x7, out)
  loop (x8 in node.members) {
    spansOfEnumMember(x8, out)
  }
  out.push("EnumDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTestDecl(node: TestDecl, out: MutableList<string>) {
  loop (x2 in node.attrs) {
    spansOfAttribute(x2, out)
  }
  out.push("TestDecl.labelAt ${node.labelAt.start} ${node.labelAt.end}")
  out.push("TestDecl.at ${node.at.start} ${node.at.end}")
  if (val x8 = node.body) {
    spansOfBlock(x8, out)
  }
  out.push("TestDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfSuiteDecl(node: SuiteDecl, out: MutableList<string>) {
  loop (x2 in node.attrs) {
    spansOfAttribute(x2, out)
  }
  out.push("SuiteDecl.labelAt ${node.labelAt.start} ${node.labelAt.end}")
  out.push("SuiteDecl.at ${node.at.start} ${node.at.end}")
  loop (x8 in node.decls) {
    spansOfDecl(x8, out)
  }
  out.push("SuiteDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfSkipDecl(node: SkipDecl, out: MutableList<string>) {
  spansOfExpr(*node.call, out)
  out.push("SkipDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfStaticAssertDecl(node: StaticAssertDecl, out: MutableList<string>) {
  spansOfStaticAssert(node.assert, out)
}

fun spansOfBadDecl(node: BadDecl, out: MutableList<string>) {
  out.push("BadDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfAttribute(node: Attribute, out: MutableList<string>) {
  spansOfIdent(node.qual, out)
  spansOfIdent(node.name, out)
  loop (x3 in node.args) {
    spansOfArg(x3, out)
  }
  out.push("Attribute.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfAnnotationInfo(node: AnnotationInfo, out: MutableList<string>) {
  loop (x1 in node.targets) {
    spansOfIdent(x1, out)
  }
  loop (x3 in node.readers) {
    spansOfType(x3, out)
  }
  out.push("AnnotationInfo.onPos ${node.onPos.start} ${node.onPos.end}")
  out.push("AnnotationInfo.readersPos ${node.readersPos.start} ${node.readersPos.end}")
  if (val x7 = node.wrapper) spansOfExpr(*x7, out)
  out.push("AnnotationInfo.wrapperPos ${node.wrapperPos.start} ${node.wrapperPos.end}")
}

fun spansOfUseSpec(node: UseSpec, out: MutableList<string>) {
  loop (x1 in node.path) {
    spansOfIdent(x1, out)
  }
  if (val x3 = node.alias) {
    spansOfIdent(x3, out)
  }
  if (val x5 = node.names) {
    loop (x6 in x5) {
      spansOfUseName(x6, out)
    }
  }
  if (val x9 = node.extendNames) {
    loop (x10 in x9) {
      spansOfUseName(x10, out)
    }
  }
  out.push("UseSpec.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfUseName(node: UseName, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  if (val x2 = node.alias) {
    spansOfIdent(x2, out)
  }
  out.push("UseName.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTypeParam(node: TypeParam, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  loop (x2 in node.bounds) {
    spansOfType(x2, out)
  }
  out.push("TypeParam.at ${node.at.start} ${node.at.end}")
  if (val x6 = node.of) spansOfType(*x6, out)
}

fun spansOfParam(node: Param, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  if (val x2 = node.typ) spansOfType(*x2, out)
  if (val x3 = node.defaultValue) spansOfExpr(*x3, out)
  out.push("Param.lazyPos ${node.lazyPos.start} ${node.lazyPos.end}")
  if (val x7 = node.pattern) {
    spansOfBinding(x7, out)
  }
  out.push("Param.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfFunDecl(node: FunDecl, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  loop (x16 in node.typeParams) {
    spansOfTypeParam(x16, out)
  }
  loop (x18 in node.params) {
    spansOfParam(x18, out)
  }
  out.push("FunDecl.variadicAt ${node.variadicAt.start} ${node.variadicAt.end}")
  if (val x22 = node.ret) spansOfType(*x22, out)
  spansOfEffects(node.effects, out)
  if (val x24 = node.body) {
    spansOfBlock(x24, out)
  }
  if (val x26 = node.exprBody) spansOfExpr(*x26, out)
  out.push("FunDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfField(node: Field, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  if (val x11 = node.typ) spansOfType(*x11, out)
  if (val x12 = node.defaultValue) spansOfExpr(*x12, out)
  out.push("Field.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfAssocTypeDecl(node: AssocTypeDecl, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  loop (x2 in node.bounds) {
    spansOfType(x2, out)
  }
  out.push("AssocTypeDecl.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfAssocTypeBinding(node: AssocTypeBinding, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  if (val x2 = node.typ) spansOfType(*x2, out)
}

fun spansOfEnumMember(node: EnumMember, out: MutableList<string>) {
  loop (x1 in node.attrs) {
    spansOfAttribute(x1, out)
  }
  spansOfIdent(node.name, out)
  if (val x5 = node.value) spansOfExpr(*x5, out)
  out.push("EnumMember.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfStaticAssert(node: StaticAssert, out: MutableList<string>) {
  spansOfExpr(*node.cond, out)
  if (val x2 = node.reason) spansOfExpr(*x2, out)
  out.push("StaticAssert.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBlockStmt(node: BlockStmt, out: MutableList<string>) {
  spansOfBlock(node.block, out)
}

fun spansOfValStmt(node: ValStmt, out: MutableList<string>) {
  spansOfBinding(node.binding, out)
  if (val x3 = node.pattern) spansOfPattern(*x3, out)
  if (val x4 = node.value) spansOfExpr(*x4, out)
  if (val x5 = node.orElse) {
    spansOfHandler(x5, out)
  }
  out.push("ValStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfExprStmt(node: ExprStmt, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
}

fun spansOfAssignStmt(node: AssignStmt, out: MutableList<string>) {
  spansOfExpr(*node.target, out)
  spansOfExpr(*node.value, out)
  out.push("AssignStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfReturnStmt(node: ReturnStmt, out: MutableList<string>) {
  if (val x1 = node.value) spansOfExpr(*x1, out)
  out.push("ReturnStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfThrowStmt(node: ThrowStmt, out: MutableList<string>) {
  if (val x1 = node.value) spansOfExpr(*x1, out)
  out.push("ThrowStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBreakStmt(node: BreakStmt, out: MutableList<string>) {
  if (val x1 = node.label) {
    spansOfIdent(x1, out)
  }
  out.push("BreakStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfContinueStmt(node: ContinueStmt, out: MutableList<string>) {
  if (val x1 = node.label) {
    spansOfIdent(x1, out)
  }
  out.push("ContinueStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfCedeStmt(node: CedeStmt, out: MutableList<string>) {
  out.push("CedeStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfLoopStmt(node: LoopStmt, out: MutableList<string>) {
  if (val x2 = node.label) {
    spansOfIdent(x2, out)
  }
  if (val x4 = node.cond) spansOfExpr(*x4, out)
  if (val x5 = node.bound) {
    spansOfBinding(x5, out)
  }
  if (val x7 = node.iter) spansOfExpr(*x7, out)
  spansOfBlock(node.body, out)
  out.push("LoopStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfWithStmt(node: WithStmt, out: MutableList<string>) {
  spansOfWithBinding(node.binding, out)
  out.push("WithStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfScopeStmt(node: ScopeStmt, out: MutableList<string>) {
  if (val x1 = node.on) spansOfExpr(*x1, out)
  if (val x2 = node.workers) spansOfExpr(*x2, out)
  spansOfBlock(node.body, out)
  out.push("ScopeStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfFunStmt(node: FunStmt, out: MutableList<string>) {
  spansOfFunDecl(*node.decl, out)
}

fun spansOfStaticAssertStmt(node: StaticAssertStmt, out: MutableList<string>) {
  spansOfStaticAssert(node.assert, out)
}

fun spansOfBadStmt(node: BadStmt, out: MutableList<string>) {
  out.push("BadStmt.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBlock(node: Block, out: MutableList<string>) {
  loop (x1 in node.stmts) {
    spansOfStmt(x1, out)
  }
  out.push("Block.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBinding(node: Binding, out: MutableList<string>) {
  if (val x1 = node.name) {
    spansOfIdent(x1, out)
  }
  loop (x3 in node.tuple) {
    spansOfBinding(x3, out)
  }
  if (val x5 = node.typ) spansOfType(*x5, out)
  out.push("Binding.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfHandler(node: Handler, out: MutableList<string>) {
  if (val x1 = node.err) {
    spansOfIdent(x1, out)
  }
  spansOfBlock(node.body, out)
  out.push("Handler.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfWithBinding(node: WithBinding, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  spansOfExpr(*node.value, out)
}

fun spansOfIntLit(node: IntLit, out: MutableList<string>) {
  out.push("IntLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfFloatLit(node: FloatLit, out: MutableList<string>) {
  out.push("FloatLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfStringLit(node: StringLit, out: MutableList<string>) {
  loop (x1 in node.parts) {
    spansOfStringPart(x1, out)
  }
  out.push("StringLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTemplateExpr(node: TemplateExpr, out: MutableList<string>) {
  spansOfExpr(*node.tag, out)
  spansOfStringLit(node.lit, out)
  out.push("TemplateExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfCharLit(node: CharLit, out: MutableList<string>) {
  out.push("CharLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfRegexLit(node: RegexLit, out: MutableList<string>) {
  out.push("RegexLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBoolLit(node: BoolLit, out: MutableList<string>) {
  out.push("BoolLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfNullLit(node: NullLit, out: MutableList<string>) {
  out.push("NullLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfThisExpr(node: ThisExpr, out: MutableList<string>) {
  out.push("ThisExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfNameExpr(node: NameExpr, out: MutableList<string>) {
  loop (x2 in node.typeArgs) {
    spansOfType(x2, out)
  }
  out.push("NameExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfMemberExpr(node: MemberExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  spansOfIdent(node.name, out)
  loop (x4 in node.typeArgs) {
    spansOfType(x4, out)
  }
  out.push("MemberExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfIndexExpr(node: IndexExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  spansOfExpr(*node.index, out)
  out.push("IndexExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfCallExpr(node: CallExpr, out: MutableList<string>) {
  spansOfExpr(*node.callee, out)
  loop (x2 in node.typeArgs) {
    spansOfType(x2, out)
  }
  loop (x4 in node.args) {
    spansOfArg(x4, out)
  }
  out.push("CallExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfUnaryExpr(node: UnaryExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  out.push("UnaryExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBinaryExpr(node: BinaryExpr, out: MutableList<string>) {
  spansOfExpr(*node.left, out)
  spansOfExpr(*node.right, out)
  out.push("BinaryExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfOrFailExpr(node: OrFailExpr, out: MutableList<string>) {
  spansOfExpr(*node.left, out)
  spansOfExpr(*node.right, out)
  out.push("OrFailExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfCoalesceExpr(node: CoalesceExpr, out: MutableList<string>) {
  spansOfExpr(*node.left, out)
  if (val x2 = node.right) spansOfExpr(*x2, out)
  if (val x3 = node.handler) {
    spansOfHandler(x3, out)
  }
  out.push("CoalesceExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfCatchExpr(node: CatchExpr, out: MutableList<string>) {
  if (val x1 = node.body) {
    spansOfBlock(x1, out)
  }
  if (val x3 = node.x) spansOfExpr(*x3, out)
  spansOfHandler(node.handler, out)
  out.push("CatchExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfElvisExpr(node: ElvisExpr, out: MutableList<string>) {
  spansOfExpr(*node.left, out)
  spansOfExpr(*node.right, out)
  out.push("ElvisExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfRangeExpr(node: RangeExpr, out: MutableList<string>) {
  spansOfExpr(*node.lo, out)
  spansOfExpr(*node.hi, out)
  out.push("RangeExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfLambdaExpr(node: LambdaExpr, out: MutableList<string>) {
  loop (x1 in node.params) {
    spansOfParam(x1, out)
  }
  if (val x3 = node.ret) spansOfType(*x3, out)
  spansOfExpr(*node.body, out)
  out.push("LambdaExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTupleExpr(node: TupleExpr, out: MutableList<string>) {
  loop (x1 in node.elems) {
    spansOfExpr(x1, out)
  }
  loop (x3 in node.labels) {
    spansOfIdent(x3, out)
  }
  out.push("TupleExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfListLit(node: ListLit, out: MutableList<string>) {
  loop (x1 in node.elems) {
    spansOfExpr(x1, out)
  }
  out.push("ListLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfMapLit(node: MapLit, out: MutableList<string>) {
  loop (x1 in node.entries) {
    spansOfMapEntry(x1, out)
  }
  out.push("MapLit.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfIfExpr(node: IfExpr, out: MutableList<string>) {
  spansOfExpr(*node.cond, out)
  spansOfBlock(node.then, out)
  if (val x3 = node.orElse) {
    spansOfBlock(x3, out)
  }
  out.push("IfExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfLetCond(node: LetCond, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  spansOfExpr(*node.value, out)
  out.push("LetCond.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfWhenExpr(node: WhenExpr, out: MutableList<string>) {
  if (val x1 = node.subject) spansOfExpr(*x1, out)
  if (val x2 = node.bind) {
    spansOfIdent(x2, out)
  }
  loop (x4 in node.arms) {
    spansOfWhenArm(x4, out)
  }
  out.push("WhenExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBlockExpr(node: BlockExpr, out: MutableList<string>) {
  spansOfBlock(node.block, out)
}

fun spansOfTryExpr(node: TryExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  out.push("TryExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfAwaitExpr(node: AwaitExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  out.push("AwaitExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfIsExpr(node: IsExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  spansOfTypePat(node.pat, out)
  out.push("IsExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfInExpr(node: InExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  spansOfExpr(*node.c, out)
  out.push("InExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfImplementsExpr(node: ImplementsExpr, out: MutableList<string>) {
  spansOfType(*node.typ, out)
  spansOfType(*node.traitType, out)
  out.push("ImplementsExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfCastExpr(node: CastExpr, out: MutableList<string>) {
  spansOfExpr(*node.x, out)
  spansOfType(*node.typ, out)
  out.push("CastExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfGatherExpr(node: GatherExpr, out: MutableList<string>) {
  if (val x1 = node.on) spansOfExpr(*x1, out)
  if (val x2 = node.workers) spansOfExpr(*x2, out)
  spansOfBlock(node.body, out)
  out.push("GatherExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfScopeExpr(node: ScopeExpr, out: MutableList<string>) {
  if (val x1 = node.on) spansOfExpr(*x1, out)
  if (val x2 = node.workers) spansOfExpr(*x2, out)
  spansOfBlock(node.body, out)
  out.push("ScopeExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfRaceExpr(node: RaceExpr, out: MutableList<string>) {
  if (val x1 = node.body) {
    spansOfBlock(x1, out)
  }
  loop (x3 in node.arms) {
    spansOfRaceArm(x3, out)
  }
  out.push("RaceExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfSelectExpr(node: SelectExpr, out: MutableList<string>) {
  loop (x1 in node.arms) {
    spansOfRaceArm(x1, out)
  }
  out.push("SelectExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfUnsafeExpr(node: UnsafeExpr, out: MutableList<string>) {
  spansOfBlock(node.body, out)
  out.push("UnsafeExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfWithExpr(node: WithExpr, out: MutableList<string>) {
  loop (x1 in node.bindings) {
    spansOfWithBinding(x1, out)
  }
  spansOfBlock(node.body, out)
  out.push("WithExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfControlExpr(node: ControlExpr, out: MutableList<string>) {
  spansOfStmt(*node.stmt, out)
}

fun spansOfBadExpr(node: BadExpr, out: MutableList<string>) {
  out.push("BadExpr.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfStringPart(node: StringPart, out: MutableList<string>) {
  if (val x2 = node.expr) spansOfExpr(*x2, out)
  out.push("StringPart.specPos ${node.specPos.start} ${node.specPos.end}")
}

fun spansOfArg(node: Arg, out: MutableList<string>) {
  if (val x1 = node.name) {
    spansOfIdent(x1, out)
  }
  spansOfExpr(*node.value, out)
}

fun spansOfMapEntry(node: MapEntry, out: MutableList<string>) {
  spansOfExpr(*node.key, out)
  spansOfExpr(*node.value, out)
}

fun spansOfWhenArm(node: WhenArm, out: MutableList<string>) {
  loop (x1 in node.patterns) {
    spansOfPattern(x1, out)
  }
  if (val x3 = node.cond) spansOfExpr(*x3, out)
  if (val x4 = node.guard) spansOfExpr(*x4, out)
  spansOfExpr(*node.body, out)
  out.push("WhenArm.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfRaceArm(node: RaceArm, out: MutableList<string>) {
  if (val x1 = node.binding) {
    spansOfBinding(x1, out)
  }
  spansOfExpr(*node.source, out)
  if (val x4 = node.name) {
    spansOfIdent(x4, out)
  }
  if (val x6 = node.body) spansOfExpr(*x6, out)
  out.push("RaceArm.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfWildcardPat(node: WildcardPat, out: MutableList<string>) {
  out.push("WildcardPat.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfBindPat(node: BindPat, out: MutableList<string>) {
  spansOfIdent(node.name, out)
}

fun spansOfLiteralPat(node: LiteralPat, out: MutableList<string>) {
  spansOfExpr(*node.value, out)
}

fun spansOfRangePat(node: RangePat, out: MutableList<string>) {
  spansOfRangeExpr(node.range, out)
  out.push("RangePat.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTypePat(node: TypePat, out: MutableList<string>) {
  spansOfType(*node.typ, out)
  loop (x2 in node.fields) {
    spansOfFieldPat(x2, out)
  }
  out.push("TypePat.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfTuplePat(node: TuplePat, out: MutableList<string>) {
  loop (x1 in node.elems) {
    spansOfPattern(x1, out)
  }
  out.push("TuplePat.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfListPat(node: ListPat, out: MutableList<string>) {
  loop (x1 in node.elems) {
    spansOfPattern(x1, out)
  }
  out.push("ListPat.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfRestPat(node: RestPat, out: MutableList<string>) {
  if (val x1 = node.name) {
    spansOfIdent(x1, out)
  }
  out.push("RestPat.pos ${node.pos.start} ${node.pos.end}")
}

fun spansOfFieldPat(node: FieldPat, out: MutableList<string>) {
  spansOfIdent(node.name, out)
  if (val x2 = node.pat) spansOfPattern(*x2, out)
}
