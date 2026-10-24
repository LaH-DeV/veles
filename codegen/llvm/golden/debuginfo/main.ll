%S.main.Point = type { i64, i64 }
define i64 @v_main.distance(%S.main.Point %p1, %S.main.Point %p2) !dbg !0 {
entry:
  %a1 = alloca %S.main.Point
  %a2 = alloca %S.main.Point
  %a15 = alloca i64
  %a28 = alloca i64
  store %S.main.Point %p1, ptr %a1
  store %S.main.Point %p2, ptr %a2
  call void @llvm.dbg.declare(metadata ptr %a1, metadata !1, metadata !DIExpression()), !dbg !2
  call void @llvm.dbg.declare(metadata ptr %a2, metadata !3, metadata !DIExpression()), !dbg !2
  %t3 = getelementptr inbounds %S.main.Point, ptr %a1, i32 0, i32 0, !dbg !4
  %t4 = load i64, ptr %t3, !dbg !4
  %t5 = getelementptr inbounds %S.main.Point, ptr %a2, i32 0, i32 0, !dbg !4
  %t6 = load i64, ptr %t5, !dbg !4
  %t8 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t4, i64 %t6), !dbg !4
  %t9 = extractvalue { i64, i1 } %t8, 0, !dbg !4
  %t10 = extractvalue { i64, i1 } %t8, 1, !dbg !4
  br i1 %t10, label %overflow.1, label %arith.ok.2, !dbg !4
overflow.1:
  %t11 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !4
  %t12 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !4
  %t13 = extractvalue %str { ptr @.str.2, i64 13 }, 0, !dbg !4
  %t14 = extractvalue %str { ptr @.str.2, i64 13 }, 1, !dbg !4
  call void @veles_panic_at(ptr %t11, i64 %t12, ptr %t13, i64 %t14), !dbg !4
  unreachable, !dbg !4
arith.ok.2:
  call void @llvm.dbg.declare(metadata ptr %a15, metadata !5, metadata !DIExpression()), !dbg !4
  store i64 %t9, ptr %a15, !dbg !4
  %t16 = getelementptr inbounds %S.main.Point, ptr %a1, i32 0, i32 1, !dbg !6
  %t17 = load i64, ptr %t16, !dbg !6
  %t18 = getelementptr inbounds %S.main.Point, ptr %a2, i32 0, i32 1, !dbg !6
  %t19 = load i64, ptr %t18, !dbg !6
  %t21 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t17, i64 %t19), !dbg !6
  %t22 = extractvalue { i64, i1 } %t21, 0, !dbg !6
  %t23 = extractvalue { i64, i1 } %t21, 1, !dbg !6
  br i1 %t23, label %overflow.3, label %arith.ok.4, !dbg !6
overflow.3:
  %t24 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !6
  %t25 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !6
  %t26 = extractvalue %str { ptr @.str.3, i64 13 }, 0, !dbg !6
  %t27 = extractvalue %str { ptr @.str.3, i64 13 }, 1, !dbg !6
  call void @veles_panic_at(ptr %t24, i64 %t25, ptr %t26, i64 %t27), !dbg !6
  unreachable, !dbg !6
arith.ok.4:
  call void @llvm.dbg.declare(metadata ptr %a28, metadata !7, metadata !DIExpression()), !dbg !6
  store i64 %t22, ptr %a28, !dbg !6
  %t29 = load i64, ptr %a15, !dbg !8
  %t30 = load i64, ptr %a15, !dbg !8
  %t32 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t29, i64 %t30), !dbg !8
  %t33 = extractvalue { i64, i1 } %t32, 0, !dbg !8
  %t34 = extractvalue { i64, i1 } %t32, 1, !dbg !8
  br i1 %t34, label %overflow.5, label %arith.ok.6, !dbg !8
overflow.5:
  %t35 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !8
  %t36 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !8
  %t37 = extractvalue %str { ptr @.str.4, i64 12 }, 0, !dbg !8
  %t38 = extractvalue %str { ptr @.str.4, i64 12 }, 1, !dbg !8
  call void @veles_panic_at(ptr %t35, i64 %t36, ptr %t37, i64 %t38), !dbg !8
  unreachable, !dbg !8
arith.ok.6:
  %t39 = load i64, ptr %a28, !dbg !8
  %t40 = load i64, ptr %a28, !dbg !8
  %t42 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t39, i64 %t40), !dbg !8
  %t43 = extractvalue { i64, i1 } %t42, 0, !dbg !8
  %t44 = extractvalue { i64, i1 } %t42, 1, !dbg !8
  br i1 %t44, label %overflow.7, label %arith.ok.8, !dbg !8
overflow.7:
  %t45 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !8
  %t46 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !8
  %t47 = extractvalue %str { ptr @.str.5, i64 13 }, 0, !dbg !8
  %t48 = extractvalue %str { ptr @.str.5, i64 13 }, 1, !dbg !8
  call void @veles_panic_at(ptr %t45, i64 %t46, ptr %t47, i64 %t48), !dbg !8
  unreachable, !dbg !8
arith.ok.8:
  %t50 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t33, i64 %t43), !dbg !8
  %t51 = extractvalue { i64, i1 } %t50, 0, !dbg !8
  %t52 = extractvalue { i64, i1 } %t50, 1, !dbg !8
  br i1 %t52, label %overflow.9, label %arith.ok.10, !dbg !8
overflow.9:
  %t53 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !8
  %t54 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !8
  %t55 = extractvalue %str { ptr @.str.4, i64 12 }, 0, !dbg !8
  %t56 = extractvalue %str { ptr @.str.4, i64 12 }, 1, !dbg !8
  call void @veles_panic_at(ptr %t53, i64 %t54, ptr %t55, i64 %t56), !dbg !8
  unreachable, !dbg !8
arith.ok.10:
  ret i64 %t51, !dbg !8
}

define void @v_main.main() !dbg !9 {
entry:
  %a3 = alloca %S.main.Point
  %a4 = alloca %str
  %a5 = alloca i64
  %a9 = alloca { i64, i64, i1 }
  %a12 = alloca i64
  %a15 = alloca i64
  %a16 = alloca i1
  %a17 = alloca i1
  %a19 = alloca i1
  %a31 = alloca i64
  %a56 = alloca i1
  %a69 = alloca i1
  %a72 = alloca [21 x i8]
  %a77 = alloca %str
  %a79 = alloca %str
  %a80 = alloca [5 x %str]
  %t1 = insertvalue %S.main.Point undef, i64 3, 0, !dbg !10
  %t2 = insertvalue %S.main.Point %t1, i64 4, 1, !dbg !10
  call void @llvm.dbg.declare(metadata ptr %a3, metadata !11, metadata !DIExpression()), !dbg !10
  store %S.main.Point %t2, ptr %a3, !dbg !10
  call void @llvm.dbg.declare(metadata ptr %a4, metadata !12, metadata !DIExpression()), !dbg !13
  store %str { ptr @.str.6, i64 3 }, ptr %a4, !dbg !13
  call void @llvm.dbg.declare(metadata ptr %a5, metadata !14, metadata !DIExpression()), !dbg !15
  store i64 0, ptr %a5, !dbg !15
  %t6 = insertvalue { i64, i64, i1 } undef, i64 0, 0, !dbg !16
  %t7 = insertvalue { i64, i64, i1 } %t6, i64 2, 1, !dbg !16
  %t8 = insertvalue { i64, i64, i1 } %t7, i1 false, 2, !dbg !16
  store { i64, i64, i1 } %t8, ptr %a9, !dbg !16
  %t10 = getelementptr inbounds { i64, i64, i1 }, ptr %a9, i32 0, i32 0, !dbg !16
  %t11 = load i64, ptr %t10, !dbg !16
  store i64 %t11, ptr %a12, !dbg !16
  %t13 = getelementptr inbounds { i64, i64, i1 }, ptr %a9, i32 0, i32 1, !dbg !16
  %t14 = load i64, ptr %t13, !dbg !16
  store i64 %t14, ptr %a15, !dbg !16
  store i1 false, ptr %a16, !dbg !16
  br label %loop.cond.1
loop.cond.1:
  %t18 = load i1, ptr %a16, !dbg !16
  br i1 %t18, label %if.then.5, label %if.else.7, !dbg !16
if.then.5:
  store i1 false, ptr %a17, !dbg !16
  br label %if.end.6, !dbg !16
if.else.7:
  %t20 = getelementptr inbounds { i64, i64, i1 }, ptr %a9, i32 0, i32 2, !dbg !16
  %t21 = load i1, ptr %t20, !dbg !16
  br i1 %t21, label %if.then.8, label %if.else.10, !dbg !16
if.then.8:
  %t22 = load i64, ptr %a12, !dbg !16
  %t23 = load i64, ptr %a15, !dbg !16
  %t24 = icmp sle i64 %t22, %t23, !dbg !16
  store i1 %t24, ptr %a19, !dbg !16
  br label %if.end.9, !dbg !16
if.else.10:
  %t25 = load i64, ptr %a12, !dbg !16
  %t26 = load i64, ptr %a15, !dbg !16
  %t27 = icmp slt i64 %t25, %t26, !dbg !16
  store i1 %t27, ptr %a19, !dbg !16
  br label %if.end.9, !dbg !16
if.end.9:
  %t28 = load i1, ptr %a19, !dbg !16
  store i1 %t28, ptr %a17, !dbg !16
  br label %if.end.6, !dbg !16
if.end.6:
  %t29 = load i1, ptr %a17, !dbg !16
  br i1 %t29, label %loop.body.4, label %loop.end.3, !dbg !16
loop.body.4:
  %t30 = load i64, ptr %a12, !dbg !16
  call void @llvm.dbg.declare(metadata ptr %a31, metadata !17, metadata !DIExpression()), !dbg !16
  store i64 %t30, ptr %a31, !dbg !16
  %t32 = load i64, ptr %a5, !dbg !18
  %t33 = insertvalue %S.main.Point undef, i64 0, 0, !dbg !18
  %t34 = insertvalue %S.main.Point %t33, i64 0, 1, !dbg !18
  %t35 = load %S.main.Point, ptr %a3, !dbg !18
  %t36 = extractvalue %str { ptr @.str.7, i64 22 }, 0, !dbg !18
  %t37 = extractvalue %str { ptr @.str.7, i64 22 }, 1, !dbg !18
  call void @veles_call_push(ptr %t36), !dbg !18
  %t38 = call i64 @v_main.distance(%S.main.Point %t34, %S.main.Point %t35), !dbg !18
  call void @veles_call_pop(), !dbg !18
  %t39 = load i64, ptr %a31, !dbg !18
  %t41 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t38, i64 %t39), !dbg !18
  %t42 = extractvalue { i64, i1 } %t41, 0, !dbg !18
  %t43 = extractvalue { i64, i1 } %t41, 1, !dbg !18
  br i1 %t43, label %overflow.11, label %arith.ok.12, !dbg !18
overflow.11:
  %t44 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !18
  %t45 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !18
  %t46 = extractvalue %str { ptr @.str.8, i64 13 }, 0, !dbg !18
  %t47 = extractvalue %str { ptr @.str.8, i64 13 }, 1, !dbg !18
  call void @veles_panic_at(ptr %t44, i64 %t45, ptr %t46, i64 %t47), !dbg !18
  unreachable, !dbg !18
arith.ok.12:
  %t49 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t32, i64 %t42), !dbg !18
  %t50 = extractvalue { i64, i1 } %t49, 0, !dbg !18
  %t51 = extractvalue { i64, i1 } %t49, 1, !dbg !18
  br i1 %t51, label %overflow.13, label %arith.ok.14, !dbg !18
overflow.13:
  %t52 = extractvalue %str { ptr @.str.1, i64 16 }, 0, !dbg !18
  %t53 = extractvalue %str { ptr @.str.1, i64 16 }, 1, !dbg !18
  %t54 = extractvalue %str { ptr @.str.9, i64 12 }, 0, !dbg !18
  %t55 = extractvalue %str { ptr @.str.9, i64 12 }, 1, !dbg !18
  call void @veles_panic_at(ptr %t52, i64 %t53, ptr %t54, i64 %t55), !dbg !18
  unreachable, !dbg !18
arith.ok.14:
  store i64 %t50, ptr %a5, !dbg !18
  br label %loop.post.2, !dbg !18
loop.post.2:
  %t57 = getelementptr inbounds { i64, i64, i1 }, ptr %a9, i32 0, i32 2, !dbg !18
  %t58 = load i1, ptr %t57, !dbg !18
  store i1 %t58, ptr %a56, !dbg !18
  br i1 %t58, label %sc.rhs.15, label %sc.end.16, !dbg !18
sc.rhs.15:
  %t59 = load i64, ptr %a12, !dbg !18
  %t60 = load i64, ptr %a15, !dbg !18
  %t61 = icmp eq i64 %t59, %t60, !dbg !18
  store i1 %t61, ptr %a56, !dbg !18
  br label %sc.end.16
sc.end.16:
  %t62 = load i1, ptr %a56, !dbg !18
  br i1 %t62, label %if.then.17, label %if.else.19, !dbg !18
if.then.17:
  store i1 true, ptr %a16, !dbg !18
  br label %if.end.18, !dbg !18
if.else.19:
  %t63 = load i64, ptr %a12, !dbg !18
  %t64 = add i64 %t63, 1, !dbg !18
  store i64 %t64, ptr %a12, !dbg !18
  br label %if.end.18, !dbg !18
if.end.18:
  %t65 = load volatile i32, ptr @veles_attention_line, align 64, !dbg !18
  %t66 = icmp ne i32 %t65, 0, !dbg !18
  br i1 %t66, label %safepoint.20, label %safepoint.on.21, !prof !{!"branch_weights", i32 1, i32 100000}, !dbg !18
safepoint.20:
  call void @veles_backedge_plain(), !dbg !18
  br label %safepoint.on.21, !dbg !18
safepoint.on.21:
  br label %loop.cond.1, !dbg !18
loop.end.3:
  %t67 = load i64, ptr %a5, !dbg !19
  %t68 = icmp sgt i64 %t67, 0, !dbg !19
  call void @llvm.dbg.declare(metadata ptr %a69, metadata !20, metadata !DIExpression()), !dbg !19
  store i1 %t68, ptr %a69, !dbg !19
  %t70 = load %str, ptr %a4, !dbg !21
  %t71 = load i64, ptr %a5, !dbg !21
  %t73 = call i64 @veles_i64_format(ptr %a72, i64 %t71), !dbg !21
  %t74 = insertvalue %str undef, ptr %a72, 0, !dbg !21
  %t75 = insertvalue %str %t74, i64 %t73, 1, !dbg !21
  %t76 = load i1, ptr %a69, !dbg !21
  call void @veles_bool_to_string(ptr %a77, i1 zeroext %t76), !dbg !21
  %t78 = load %str, ptr %a77, !dbg !21
  %t81 = getelementptr [5 x %str], ptr %a80, i64 0, i64 0, !dbg !21
  store %str %t70, ptr %t81, !dbg !21
  %t82 = getelementptr [5 x %str], ptr %a80, i64 0, i64 1, !dbg !21
  store %str { ptr @.str.10, i64 1 }, ptr %t82, !dbg !21
  %t83 = getelementptr [5 x %str], ptr %a80, i64 0, i64 2, !dbg !21
  store %str %t75, ptr %t83, !dbg !21
  %t84 = getelementptr [5 x %str], ptr %a80, i64 0, i64 3, !dbg !21
  store %str { ptr @.str.10, i64 1 }, ptr %t84, !dbg !21
  %t85 = getelementptr [5 x %str], ptr %a80, i64 0, i64 4, !dbg !21
  store %str %t78, ptr %t85, !dbg !21
  call void @veles_string_concat_n(ptr %a79, ptr %a80, i64 5), !dbg !21
  %t86 = load %str, ptr %a79, !dbg !21
  %t87 = extractvalue %str { ptr @.str.11, i64 20 }, 0, !dbg !21
  %t88 = extractvalue %str { ptr @.str.11, i64 20 }, 1, !dbg !21
  call void @veles_call_push(ptr %t87), !dbg !21
  call void @v_std.io.println(%str %t86), !dbg !21
  call void @veles_call_pop(), !dbg !21
  ret void, !dbg !21
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:14:12\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:15:12\00"
@.str.4 = private unnamed_addr constant [13 x i8] c"main.vs:16:3\00"
@.str.5 = private unnamed_addr constant [14 x i8] c"main.vs:16:13\00"
@.str.6 = private unnamed_addr constant [4 x i8] c"far\00"
@.str.7 = private unnamed_addr constant [23 x i8] c"main.vs:24:14\00distance\00"
@.str.8 = private unnamed_addr constant [14 x i8] c"main.vs:24:14\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:24:5\00"
@.str.10 = private unnamed_addr constant [2 x i8] c" \00"
@.str.11 = private unnamed_addr constant [21 x i8] c"main.vs:27:3\00println\00"

!0 = distinct !DISubprogram(name: "distance", scope: !22, file: !22, line: 13, type: !23, scopeLine: 13, spFlags: DISPFlagDefinition, unit: !24)
!1 = !DILocalVariable(name: "a", arg: 1, scope: !0, file: !22, line: 13, type: !25)
!2 = !DILocation(line: 13, column: 5, scope: !0)
!3 = !DILocalVariable(name: "b", arg: 2, scope: !0, file: !22, line: 13, type: !25)
!4 = !DILocation(line: 14, column: 3, scope: !0)
!5 = !DILocalVariable(name: "dx", scope: !0, file: !22, line: 14, type: !26)
!6 = !DILocation(line: 15, column: 3, scope: !0)
!7 = !DILocalVariable(name: "dy", scope: !0, file: !22, line: 15, type: !26)
!8 = !DILocation(line: 16, column: 3, scope: !0)
!9 = distinct !DISubprogram(name: "main", scope: !22, file: !22, line: 19, type: !27, scopeLine: 19, spFlags: DISPFlagDefinition, unit: !24)
!10 = !DILocation(line: 20, column: 3, scope: !9)
!11 = !DILocalVariable(name: "far", scope: !9, file: !22, line: 20, type: !25)
!12 = !DILocalVariable(name: "label", scope: !9, file: !22, line: 21, type: !28)
!13 = !DILocation(line: 21, column: 3, scope: !9)
!14 = !DILocalVariable(name: "total", scope: !9, file: !22, line: 22, type: !26)
!15 = !DILocation(line: 22, column: 3, scope: !9)
!16 = !DILocation(line: 23, column: 3, scope: !9)
!17 = !DILocalVariable(name: "i", scope: !9, file: !22, line: 23, type: !26)
!18 = !DILocation(line: 24, column: 5, scope: !9)
!19 = !DILocation(line: 26, column: 3, scope: !9)
!20 = !DILocalVariable(name: "ok", scope: !9, file: !22, line: 26, type: !29)
!21 = !DILocation(line: 27, column: 3, scope: !9)
!22 = !DIFile(filename: "main.vs", directory: "<dir>")
!23 = !DISubroutineType(types: !{})
!24 = distinct !DICompileUnit(language: DW_LANG_C, file: !22, producer: "veles", isOptimized: false, runtimeVersion: 0, emissionKind: FullDebug)
!25 = !DICompositeType(tag: DW_TAG_structure_type, name: "Point", size: 128, align: 64, elements: !{!30, !31})
!26 = !DIBasicType(name: "i64", size: 64, encoding: DW_ATE_signed)
!27 = !DISubroutineType(types: !{})
!28 = !DICompositeType(tag: DW_TAG_structure_type, name: "string", size: 128, elements: !{!32, !33})
!29 = !DIBasicType(name: "bool", size: 8, encoding: DW_ATE_boolean)
!30 = !DIDerivedType(tag: DW_TAG_member, name: "x", scope: !25, baseType: !26, size: 64, offset: 0)
!31 = !DIDerivedType(tag: DW_TAG_member, name: "y", scope: !25, baseType: !26, size: 64, offset: 64)
!32 = !DIDerivedType(tag: DW_TAG_member, name: "data", scope: !28, baseType: !34, size: 64, offset: 0)
!33 = !DIDerivedType(tag: DW_TAG_member, name: "len", scope: !28, baseType: !26, size: 64, offset: 64)
!34 = !DIDerivedType(tag: DW_TAG_pointer_type, baseType: !35, size: 64)
!35 = !DIBasicType(name: "u8", size: 8, encoding: DW_ATE_unsigned_char)
