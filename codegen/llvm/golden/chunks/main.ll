define i64 @v_main.sumSquares(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  %a7 = alloca { i64, i64, i1 }
  %a10 = alloca i64
  %a13 = alloca i64
  %a14 = alloca i1
  %a15 = alloca i64
  %a16 = alloca i1
  %a18 = alloca i1
  %a29 = alloca i64
  %a43 = alloca i1
  %a45 = alloca i1
  %a60 = alloca i64
  %a66 = alloca i1
  store i64 %p1, ptr %a1
  store i64 0, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t5 = insertvalue { i64, i64, i1 } %t4, i64 %t3, 1
  %t6 = insertvalue { i64, i64, i1 } %t5, i1 false, 2
  store { i64, i64, i1 } %t6, ptr %a7
  %t8 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 0
  %t9 = load i64, ptr %t8
  store i64 %t9, ptr %a10
  %t11 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 1
  %t12 = load i64, ptr %t11
  store i64 %t12, ptr %a13
  store i1 false, ptr %a14
  store i64 0, ptr %a15
  br label %chunk.cond.1
chunk.cond.1:
  %t17 = load i1, ptr %a14
  br i1 %t17, label %if.then.8, label %if.else.10
if.then.8:
  store i1 false, ptr %a16
  br label %if.end.9
if.else.10:
  %t19 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t20 = load i1, ptr %t19
  br i1 %t20, label %if.then.11, label %if.else.13
if.then.11:
  %t21 = load i64, ptr %a10
  %t22 = load i64, ptr %a13
  %t23 = icmp sle i64 %t21, %t22
  store i1 %t23, ptr %a18
  br label %if.end.12
if.else.13:
  %t24 = load i64, ptr %a10
  %t25 = load i64, ptr %a13
  %t26 = icmp slt i64 %t24, %t25
  store i1 %t26, ptr %a18
  br label %if.end.12
if.end.12:
  %t27 = load i1, ptr %a18
  store i1 %t27, ptr %a16
  br label %if.end.9
if.end.9:
  %t28 = load i1, ptr %a16
  br i1 %t28, label %chunk.start.7, label %chunk.end.3
chunk.start.7:
  %t30 = load i64, ptr %a10
  %t31 = icmp sgt i64 %t30, 9223372036854775408
  br i1 %t31, label %if.then.14, label %if.else.16
if.then.14:
  store i64 9223372036854775807, ptr %a29
  br label %if.end.15
if.else.16:
  %t32 = load i64, ptr %a10
  %t34 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t32, i64 399)
  %t35 = extractvalue { i64, i1 } %t34, 0
  %t36 = extractvalue { i64, i1 } %t34, 1
  br i1 %t36, label %overflow.17, label %arith.ok.18
overflow.17:
  %t37 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t38 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t39 = extractvalue %str { ptr @.str.2, i64 0 }, 0
  %t40 = extractvalue %str { ptr @.str.2, i64 0 }, 1
  call void @veles_panic_at(ptr %t37, i64 %t38, ptr %t39, i64 %t40)
  unreachable
arith.ok.18:
  store i64 %t35, ptr %a29
  br label %if.end.15
if.end.15:
  %t41 = load i64, ptr %a29
  store i64 %t41, ptr %a15
  br label %chunk.body.5
chunk.inner.4:
  %t44 = load i1, ptr %a14
  br i1 %t44, label %if.then.19, label %if.else.21
if.then.19:
  store i1 false, ptr %a43
  br label %if.end.20
if.else.21:
  %t46 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t47 = load i1, ptr %t46
  br i1 %t47, label %if.then.22, label %if.else.24
if.then.22:
  %t48 = load i64, ptr %a10
  %t49 = load i64, ptr %a13
  %t50 = icmp sle i64 %t48, %t49
  store i1 %t50, ptr %a45
  br label %if.end.23
if.else.24:
  %t51 = load i64, ptr %a10
  %t52 = load i64, ptr %a13
  %t53 = icmp slt i64 %t51, %t52
  store i1 %t53, ptr %a45
  br label %if.end.23
if.end.23:
  %t54 = load i1, ptr %a45
  store i1 %t54, ptr %a43
  br label %if.end.20
if.end.20:
  %t55 = load i1, ptr %a43
  %t56 = load i64, ptr %a10
  %t57 = load i64, ptr %a15
  %t58 = icmp slt i64 %t56, %t57
  %t42 = and i1 %t55, %t58
  br i1 %t42, label %chunk.body.5, label %chunk.looked.6
chunk.body.5:
  %t59 = load i64, ptr %a10
  store i64 %t59, ptr %a60
  %t61 = load i64, ptr %a2
  %t62 = load i64, ptr %a60
  %t63 = load i64, ptr %a60
  %t64 = mul i64 %t62, %t63
  %t65 = add i64 %t61, %t64
  store i64 %t65, ptr %a2
  br label %chunk.post.2
chunk.post.2:
  %t67 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t68 = load i1, ptr %t67
  store i1 %t68, ptr %a66
  br i1 %t68, label %sc.rhs.25, label %sc.end.26
sc.rhs.25:
  %t69 = load i64, ptr %a10
  %t70 = load i64, ptr %a13
  %t71 = icmp eq i64 %t69, %t70
  store i1 %t71, ptr %a66
  br label %sc.end.26
sc.end.26:
  %t72 = load i1, ptr %a66
  br i1 %t72, label %if.then.27, label %if.else.29
if.then.27:
  store i1 true, ptr %a14
  br label %if.end.28
if.else.29:
  %t73 = load i64, ptr %a10
  %t74 = add i64 %t73, 1
  store i64 %t74, ptr %a10
  br label %if.end.28
if.end.28:
  br label %chunk.inner.4
chunk.looked.6:
  %t75 = load volatile i32, ptr @veles_attention_line, align 64
  %t76 = icmp ne i32 %t75, 0
  br i1 %t76, label %safepoint.30, label %safepoint.on.31, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.30:
  call void @veles_backedge_plain()
  br label %safepoint.on.31
safepoint.on.31:
  br label %chunk.cond.1
chunk.end.3:
  %t77 = load i64, ptr %a2
  ret i64 %t77
}

define i64 @v_main.firstNegative(ptr %p1) {
entry:
  %a1 = alloca ptr
  %a8 = alloca { i64, i64, i1 }
  %a11 = alloca i64
  %a14 = alloca i64
  %a15 = alloca i1
  %a16 = alloca i64
  %a17 = alloca i1
  %a19 = alloca i1
  %a30 = alloca i64
  %a44 = alloca i1
  %a46 = alloca i1
  %a61 = alloca i64
  %a63 = alloca ptr
  %a65 = alloca i64
  %a85 = alloca i1
  store ptr %p1, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t2, i32 0, i32 1
  %t4 = load i64, ptr %t3
  %t5 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t6 = insertvalue { i64, i64, i1 } %t5, i64 %t4, 1
  %t7 = insertvalue { i64, i64, i1 } %t6, i1 false, 2
  store { i64, i64, i1 } %t7, ptr %a8
  %t9 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 0
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a11
  %t12 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 1
  %t13 = load i64, ptr %t12
  store i64 %t13, ptr %a14
  store i1 false, ptr %a15
  store i64 0, ptr %a16
  br label %chunk.cond.1
chunk.cond.1:
  %t18 = load i1, ptr %a15
  br i1 %t18, label %if.then.8, label %if.else.10
if.then.8:
  store i1 false, ptr %a17
  br label %if.end.9
if.else.10:
  %t20 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t21 = load i1, ptr %t20
  br i1 %t21, label %if.then.11, label %if.else.13
if.then.11:
  %t22 = load i64, ptr %a11
  %t23 = load i64, ptr %a14
  %t24 = icmp sle i64 %t22, %t23
  store i1 %t24, ptr %a19
  br label %if.end.12
if.else.13:
  %t25 = load i64, ptr %a11
  %t26 = load i64, ptr %a14
  %t27 = icmp slt i64 %t25, %t26
  store i1 %t27, ptr %a19
  br label %if.end.12
if.end.12:
  %t28 = load i1, ptr %a19
  store i1 %t28, ptr %a17
  br label %if.end.9
if.end.9:
  %t29 = load i1, ptr %a17
  br i1 %t29, label %chunk.start.7, label %chunk.end.3
chunk.start.7:
  %t31 = load i64, ptr %a11
  %t32 = icmp sgt i64 %t31, 9223372036854775534
  br i1 %t32, label %if.then.14, label %if.else.16
if.then.14:
  store i64 9223372036854775807, ptr %a30
  br label %if.end.15
if.else.16:
  %t33 = load i64, ptr %a11
  %t35 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t33, i64 273)
  %t36 = extractvalue { i64, i1 } %t35, 0
  %t37 = extractvalue { i64, i1 } %t35, 1
  br i1 %t37, label %overflow.17, label %arith.ok.18
overflow.17:
  %t38 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t39 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t40 = extractvalue %str { ptr @.str.2, i64 0 }, 0
  %t41 = extractvalue %str { ptr @.str.2, i64 0 }, 1
  call void @veles_panic_at(ptr %t38, i64 %t39, ptr %t40, i64 %t41)
  unreachable
arith.ok.18:
  store i64 %t36, ptr %a30
  br label %if.end.15
if.end.15:
  %t42 = load i64, ptr %a30
  store i64 %t42, ptr %a16
  br label %chunk.body.5
chunk.inner.4:
  %t45 = load i1, ptr %a15
  br i1 %t45, label %if.then.19, label %if.else.21
if.then.19:
  store i1 false, ptr %a44
  br label %if.end.20
if.else.21:
  %t47 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t48 = load i1, ptr %t47
  br i1 %t48, label %if.then.22, label %if.else.24
if.then.22:
  %t49 = load i64, ptr %a11
  %t50 = load i64, ptr %a14
  %t51 = icmp sle i64 %t49, %t50
  store i1 %t51, ptr %a46
  br label %if.end.23
if.else.24:
  %t52 = load i64, ptr %a11
  %t53 = load i64, ptr %a14
  %t54 = icmp slt i64 %t52, %t53
  store i1 %t54, ptr %a46
  br label %if.end.23
if.end.23:
  %t55 = load i1, ptr %a46
  store i1 %t55, ptr %a44
  br label %if.end.20
if.end.20:
  %t56 = load i1, ptr %a44
  %t57 = load i64, ptr %a11
  %t58 = load i64, ptr %a16
  %t59 = icmp slt i64 %t57, %t58
  %t43 = and i1 %t56, %t59
  br i1 %t43, label %chunk.body.5, label %chunk.looked.6
chunk.body.5:
  %t60 = load i64, ptr %a11
  store i64 %t60, ptr %a61
  %t62 = load ptr, ptr %a1
  store ptr %t62, ptr %a63
  %t64 = load i64, ptr %a61
  store i64 %t64, ptr %a65
  %t66 = load i64, ptr %a65
  %t67 = icmp slt i64 %t66, 0
  br i1 %t67, label %if.then.25, label %if.end.26
if.then.25:
  %t68 = load i64, ptr %a65
  %t69 = load ptr, ptr %a63
  %t70 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t69, i32 0, i32 1
  %t71 = load i64, ptr %t70
  %t72 = add i64 %t68, %t71
  store i64 %t72, ptr %a65
  br label %if.end.26
if.end.26:
  %t73 = load ptr, ptr %a63
  %t74 = load i64, ptr %a65
  %t75 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t73, i32 0, i32 1
  %t76 = load i64, ptr %t75
  %t77 = icmp ult i64 %t74, %t76
  br i1 %t77, label %idx.ok.27, label %idx.bad.28, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.28:
  %t78 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t79 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t73, i64 %t74, ptr %t78, i64 %t79)
  unreachable
idx.ok.27:
  %t80 = load ptr, ptr %t73
  %t81 = getelementptr inbounds [8 x i8], ptr %t80, i64 %t74
  %t82 = load i64, ptr %t81
  %t83 = icmp slt i64 %t82, 0
  br i1 %t83, label %if.then.29, label %if.end.30
if.then.29:
  %t84 = load i64, ptr %a61
  ret i64 %t84
if.end.30:
  br label %chunk.post.2
chunk.post.2:
  %t86 = getelementptr inbounds { i64, i64, i1 }, ptr %a8, i32 0, i32 2
  %t87 = load i1, ptr %t86
  store i1 %t87, ptr %a85
  br i1 %t87, label %sc.rhs.31, label %sc.end.32
sc.rhs.31:
  %t88 = load i64, ptr %a11
  %t89 = load i64, ptr %a14
  %t90 = icmp eq i64 %t88, %t89
  store i1 %t90, ptr %a85
  br label %sc.end.32
sc.end.32:
  %t91 = load i1, ptr %a85
  br i1 %t91, label %if.then.33, label %if.else.35
if.then.33:
  store i1 true, ptr %a15
  br label %if.end.34
if.else.35:
  %t92 = load i64, ptr %a11
  %t93 = add i64 %t92, 1
  store i64 %t93, ptr %a11
  br label %if.end.34
if.end.34:
  br label %chunk.inner.4
chunk.looked.6:
  %t94 = load volatile i32, ptr @veles_attention_line, align 64
  %t95 = icmp ne i32 %t94, 0
  br i1 %t95, label %safepoint.36, label %safepoint.on.37, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.36:
  call void @veles_backedge_plain()
  br label %safepoint.on.37
safepoint.on.37:
  br label %chunk.cond.1
chunk.end.3:
  ret i64 -1
}

define i64 @v_main.twice(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t4 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t2, i64 2)
  %t5 = extractvalue { i64, i1 } %t4, 0
  %t6 = extractvalue { i64, i1 } %t4, 1
  br i1 %t6, label %overflow.1, label %arith.ok.2
overflow.1:
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t10 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
arith.ok.2:
  ret i64 %t5
}

define i64 @v_main.withCall(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  %a7 = alloca { i64, i64, i1 }
  %a10 = alloca i64
  %a13 = alloca i64
  %a14 = alloca i1
  %a15 = alloca i1
  %a17 = alloca i1
  %a29 = alloca i64
  %a43 = alloca i1
  store i64 %p1, ptr %a1
  store i64 0, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t5 = insertvalue { i64, i64, i1 } %t4, i64 %t3, 1
  %t6 = insertvalue { i64, i64, i1 } %t5, i1 false, 2
  store { i64, i64, i1 } %t6, ptr %a7
  %t8 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 0
  %t9 = load i64, ptr %t8
  store i64 %t9, ptr %a10
  %t11 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 1
  %t12 = load i64, ptr %t11
  store i64 %t12, ptr %a13
  store i1 false, ptr %a14
  br label %loop.cond.1
loop.cond.1:
  %t16 = load i1, ptr %a14
  br i1 %t16, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a15
  br label %if.end.6
if.else.7:
  %t18 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t19 = load i1, ptr %t18
  br i1 %t19, label %if.then.8, label %if.else.10
if.then.8:
  %t20 = load i64, ptr %a10
  %t21 = load i64, ptr %a13
  %t22 = icmp sle i64 %t20, %t21
  store i1 %t22, ptr %a17
  br label %if.end.9
if.else.10:
  %t23 = load i64, ptr %a10
  %t24 = load i64, ptr %a13
  %t25 = icmp slt i64 %t23, %t24
  store i1 %t25, ptr %a17
  br label %if.end.9
if.end.9:
  %t26 = load i1, ptr %a17
  store i1 %t26, ptr %a15
  br label %if.end.6
if.end.6:
  %t27 = load i1, ptr %a15
  br i1 %t27, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t28 = load i64, ptr %a10
  store i64 %t28, ptr %a29
  %t30 = load i64, ptr %a2
  %t31 = load i64, ptr %a29
  %t32 = extractvalue %str { ptr @.str.5, i64 19 }, 0
  %t33 = extractvalue %str { ptr @.str.5, i64 19 }, 1
  call void @veles_call_push(ptr %t32)
  %t34 = call i64 @v_main.twice(i64 %t31)
  call void @veles_call_pop()
  %t36 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t30, i64 %t34)
  %t37 = extractvalue { i64, i1 } %t36, 0
  %t38 = extractvalue { i64, i1 } %t36, 1
  br i1 %t38, label %overflow.11, label %arith.ok.12
overflow.11:
  %t39 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t40 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t41 = extractvalue %str { ptr @.str.6, i64 13 }, 0
  %t42 = extractvalue %str { ptr @.str.6, i64 13 }, 1
  call void @veles_panic_at(ptr %t39, i64 %t40, ptr %t41, i64 %t42)
  unreachable
arith.ok.12:
  store i64 %t37, ptr %a2
  br label %loop.post.2
loop.post.2:
  %t44 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t45 = load i1, ptr %t44
  store i1 %t45, ptr %a43
  br i1 %t45, label %sc.rhs.13, label %sc.end.14
sc.rhs.13:
  %t46 = load i64, ptr %a10
  %t47 = load i64, ptr %a13
  %t48 = icmp eq i64 %t46, %t47
  store i1 %t48, ptr %a43
  br label %sc.end.14
sc.end.14:
  %t49 = load i1, ptr %a43
  br i1 %t49, label %if.then.15, label %if.else.17
if.then.15:
  store i1 true, ptr %a14
  br label %if.end.16
if.else.17:
  %t50 = load i64, ptr %a10
  %t51 = add i64 %t50, 1
  store i64 %t51, ptr %a10
  br label %if.end.16
if.end.16:
  %t52 = load volatile i32, ptr @veles_attention_line, align 64
  %t53 = icmp ne i32 %t52, 0
  br i1 %t53, label %safepoint.18, label %safepoint.on.19, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.18:
  call void @veles_backedge_plain()
  br label %safepoint.on.19
safepoint.on.19:
  br label %loop.cond.1
loop.end.3:
  %t54 = load i64, ptr %a2
  ret i64 %t54
}

define ptr @v_main.ceding(ptr %task, ptr %link, i64 %p1) presplitcoroutine {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  %a7 = alloca { i64, i64, i1 }
  %a10 = alloca i64
  %a13 = alloca i64
  %a14 = alloca i1
  %a15 = alloca i1
  %a17 = alloca i1
  %a29 = alloca i64
  %a45 = alloca i1
  %a63 = alloca i64
  store i64 %p1, ptr %a1
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  store i64 0, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t5 = insertvalue { i64, i64, i1 } %t4, i64 %t3, 1
  %t6 = insertvalue { i64, i64, i1 } %t5, i1 false, 2
  store { i64, i64, i1 } %t6, ptr %a7
  %t8 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 0
  %t9 = load i64, ptr %t8
  store i64 %t9, ptr %a10
  %t11 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 1
  %t12 = load i64, ptr %t11
  store i64 %t12, ptr %a13
  store i1 false, ptr %a14
  br label %loop.cond.1
loop.cond.1:
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a15)
  %t16 = load i1, ptr %a14
  br i1 %t16, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a15
  br label %if.end.6
if.else.7:
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a17)
  %t18 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t19 = load i1, ptr %t18
  br i1 %t19, label %if.then.8, label %if.else.10
if.then.8:
  %t20 = load i64, ptr %a10
  %t21 = load i64, ptr %a13
  %t22 = icmp sle i64 %t20, %t21
  store i1 %t22, ptr %a17
  br label %if.end.9
if.else.10:
  %t23 = load i64, ptr %a10
  %t24 = load i64, ptr %a13
  %t25 = icmp slt i64 %t23, %t24
  store i1 %t25, ptr %a17
  br label %if.end.9
if.end.9:
  %t26 = load i1, ptr %a17
  store i1 %t26, ptr %a15
  br label %if.end.6
if.end.6:
  %t27 = load i1, ptr %a15
  br i1 %t27, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t28 = load i64, ptr %a10
  store i64 %t28, ptr %a29
  %t30 = load i64, ptr %a2
  %t31 = load i64, ptr %a29
  %t33 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t30, i64 %t31)
  %t34 = extractvalue { i64, i1 } %t33, 0
  %t35 = extractvalue { i64, i1 } %t33, 1
  br i1 %t35, label %overflow.11, label %arith.ok.12
overflow.11:
  %t36 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t37 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t38 = extractvalue %str { ptr @.str.7, i64 12 }, 0
  %t39 = extractvalue %str { ptr @.str.7, i64 12 }, 1
  call void @veles_panic_at(ptr %t36, i64 %t37, ptr %t38, i64 %t39)
  unreachable
arith.ok.12:
  store i64 %t34, ptr %a2
  br label %cede.13
cede.13:
  %t40 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t41 = icmp ne i64 %t40, 0
  br i1 %t41, label %cede.done.14, label %cede.susp.15
cede.susp.15:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t42 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t42, label %coro.suspend [ i8 0, label %resume.16 i8 1, label %coro.cleanup ]
resume.16:
  %t43 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t44 = icmp ne i64 %t43, 0
  br i1 %t44, label %cancelled.17, label %cont.18
cancelled.17:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.18:
  br label %cede.13
cede.done.14:
  br label %loop.post.2
loop.post.2:
  %t46 = getelementptr inbounds { i64, i64, i1 }, ptr %a7, i32 0, i32 2
  %t47 = load i1, ptr %t46
  store i1 %t47, ptr %a45
  br i1 %t47, label %sc.rhs.19, label %sc.end.20
sc.rhs.19:
  %t48 = load i64, ptr %a10
  %t49 = load i64, ptr %a13
  %t50 = icmp eq i64 %t48, %t49
  store i1 %t50, ptr %a45
  br label %sc.end.20
sc.end.20:
  %t51 = load i1, ptr %a45
  br i1 %t51, label %if.then.21, label %if.else.23
if.then.21:
  store i1 true, ptr %a14
  br label %if.end.22
if.else.23:
  %t52 = load i64, ptr %a10
  %t53 = add i64 %t52, 1
  store i64 %t53, ptr %a10
  br label %if.end.22
if.end.22:
  %t54 = load volatile i32, ptr @veles_attention_line, align 64
  %t55 = icmp ne i32 %t54, 0
  br i1 %t55, label %safepoint.24, label %safepoint.on.25, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.24:
  %t56 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t56, label %safepoint.on.25 [ i64 0, label %backedge.look.28 i64 1, label %backedge.yield.26 i64 2, label %backedge.cancel.27 ]
backedge.look.28:
  br label %safepoint.on.25
backedge.yield.26:
  br label %yield.29
yield.29:
  %t57 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t58 = icmp ne i64 %t57, 0
  br i1 %t58, label %safepoint.on.25, label %yield.susp.30
yield.susp.30:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t59 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t59, label %coro.suspend [ i8 0, label %resume.31 i8 1, label %coro.cleanup ]
resume.31:
  %t60 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t61 = icmp ne i64 %t60, 0
  br i1 %t61, label %cancelled.32, label %cont.33
cancelled.32:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.33:
  br label %yield.29
backedge.cancel.27:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.25:
  br label %loop.cond.1
loop.end.3:
  %t62 = load i64, ptr %a2
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a63)
  store i64 %t62, ptr %a63
  %t64 = load ptr, ptr %link
  %t65 = icmp eq ptr %t64, null
  br i1 %t65, label %ret.task.34, label %ret.call.35
ret.task.34:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a63, i64 8, i64 0, ptr null)
  br label %coro.final
ret.call.35:
  %t66 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %link, i32 0, i32 3
  store i64 %t62, ptr %t66
  call void @veles.frame.back(ptr %task, ptr %link)
  br label %coro.final
coro.final:
  %coro.fs = call i8 @llvm.coro.suspend(token none, i1 true)
  switch i8 %coro.fs, label %coro.suspend [ i8 0, label %coro.trap i8 1, label %coro.cleanup ]
coro.trap:
  unreachable
coro.cleanup:
  %coro.freed = call ptr @llvm.coro.free(token %coro.id, ptr %coro.hdl)
  br label %coro.suspend
coro.suspend:
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false)
  ret ptr %coro.hdl
}

define i64 @v_main.halvings(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  %a4 = alloca i64
  store i64 %p1, ptr %a1
  store i64 0, ptr %a2
  %t3 = load i64, ptr %a1
  store i64 %t3, ptr %a4
  br label %loop.cond.1
loop.cond.1:
  %t5 = load i64, ptr %a4
  %t6 = icmp sgt i64 %t5, 1
  br i1 %t6, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t7 = load i64, ptr %a4
  %t9 = icmp eq i64 2, 0
  br i1 %t9, label %divzero.5, label %div.ok.6
divzero.5:
  %t10 = extractvalue %str { ptr @.str.8, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.8, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.9, i64 12 }, 0
  %t13 = extractvalue %str { ptr @.str.9, i64 12 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
div.ok.6:
  %t14 = icmp eq i64 %t7, -9223372036854775808
  %t15 = icmp eq i64 2, -1
  %t16 = and i1 %t14, %t15
  br i1 %t16, label %divof.7, label %div.ok.8
divof.7:
  %t17 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t18 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t19 = extractvalue %str { ptr @.str.9, i64 12 }, 0
  %t20 = extractvalue %str { ptr @.str.9, i64 12 }, 1
  call void @veles_panic_at(ptr %t17, i64 %t18, ptr %t19, i64 %t20)
  unreachable
div.ok.8:
  %t8 = sdiv i64 %t7, 2
  store i64 %t8, ptr %a4
  %t21 = load i64, ptr %a2
  %t23 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t21, i64 1)
  %t24 = extractvalue { i64, i1 } %t23, 0
  %t25 = extractvalue { i64, i1 } %t23, 1
  br i1 %t25, label %overflow.9, label %arith.ok.10
overflow.9:
  %t26 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t27 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t28 = extractvalue %str { ptr @.str.10, i64 12 }, 0
  %t29 = extractvalue %str { ptr @.str.10, i64 12 }, 1
  call void @veles_panic_at(ptr %t26, i64 %t27, ptr %t28, i64 %t29)
  unreachable
arith.ok.10:
  store i64 %t24, ptr %a2
  br label %loop.post.2
loop.post.2:
  %t30 = load volatile i32, ptr @veles_attention_line, align 64
  %t31 = icmp ne i32 %t30, 0
  br i1 %t31, label %safepoint.11, label %safepoint.on.12, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.11:
  call void @veles_backedge_plain()
  br label %safepoint.on.12
safepoint.on.12:
  br label %loop.cond.1
loop.end.3:
  %t32 = load i64, ptr %a2
  ret i64 %t32
}

define ptr @v_main.main(ptr %task, ptr %link) presplitcoroutine {
entry:
  %a4 = alloca [21 x i8]
  %a9 = alloca i64
  %a13 = alloca [21 x i8]
  %a20 = alloca [21 x i8]
  %a26 = alloca { ptr, i64, i64, i64 }
  %a42 = alloca [21 x i8]
  %a49 = alloca [21 x i8]
  %a53 = alloca %str
  %a54 = alloca [9 x %str]
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t1 = extractvalue %str { ptr @.str.11, i64 24 }, 0
  %t2 = extractvalue %str { ptr @.str.11, i64 24 }, 1
  call void @veles_call_push(ptr %t1)
  %t3 = call i64 @v_main.sumSquares(i64 100)
  call void @veles_call_pop()
  %t5 = call i64 @veles_i64_format(ptr %a4, i64 %t3)
  %t6 = insertvalue %str undef, ptr %a4, 0
  %t7 = insertvalue %str %t6, i64 %t5, 1
  %t8 = call ptr @veles_list_new(ptr @adesc.i64, i64 4)
  store i64 3, ptr %a9
  call void @veles_list_push(ptr %t8, ptr %a9)
  store i64 1, ptr %a9
  call void @veles_list_push(ptr %t8, ptr %a9)
  store i64 -4, ptr %a9
  call void @veles_list_push(ptr %t8, ptr %a9)
  store i64 1, ptr %a9
  call void @veles_list_push(ptr %t8, ptr %a9)
  %t10 = extractvalue %str { ptr @.str.12, i64 27 }, 0
  %t11 = extractvalue %str { ptr @.str.12, i64 27 }, 1
  call void @veles_call_push(ptr %t10)
  %t12 = call i64 @v_main.firstNegative(ptr %t8)
  call void @veles_call_pop()
  %t14 = call i64 @veles_i64_format(ptr %a13, i64 %t12)
  %t15 = insertvalue %str undef, ptr %a13, 0
  %t16 = insertvalue %str %t15, i64 %t14, 1
  %t17 = extractvalue %str { ptr @.str.13, i64 22 }, 0
  %t18 = extractvalue %str { ptr @.str.13, i64 22 }, 1
  call void @veles_call_push(ptr %t17)
  %t19 = call i64 @v_main.withCall(i64 10)
  call void @veles_call_pop()
  %t21 = call i64 @veles_i64_format(ptr %a20, i64 %t19)
  %t22 = insertvalue %str undef, ptr %a20, 0
  %t23 = insertvalue %str %t22, i64 %t21, 1
  %t24 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t25 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t24)
  %t27 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a26, i32 0, i32 0
  store ptr %coro.hdl, ptr %t27
  %t28 = add i64 %coro.depth, 1
  %t29 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a26, i32 0, i32 1
  store i64 %t28, ptr %t29
  %t30 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a26, i32 0, i32 2
  store i64 0, ptr %t30
  %t31 = call ptr @v_main.ceding(ptr %task, ptr %a26, i64 10)
  br label %call.check.1
call.check.1:
  %t32 = load i64, ptr %t30
  %t33 = icmp ne i64 %t32, 0
  br i1 %t33, label %call.done.2, label %call.wait.3
call.wait.3:
  %t34 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t34, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t35 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t36 = icmp ne i64 %t35, 0
  br i1 %t36, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.6:
  br label %call.check.1
call.done.2:
  call void @veles.frame.free(ptr %task, ptr %t31)
  %t37 = icmp eq i64 %t32, 2
  br i1 %t37, label %call.unwound.7, label %call.ok.8, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.7:
  %t38 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t39 = icmp ne i64 %t38, 0
  br i1 %t39, label %call.cancelled.9, label %call.look.10
call.look.10:
  br label %call.cancelled.9
call.cancelled.9:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.8:
  %t40 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a26, i32 0, i32 3
  %t41 = load i64, ptr %t40
  call void @veles_call_pop()
  %t43 = call i64 @veles_i64_format(ptr %a42, i64 %t41)
  %t44 = insertvalue %str undef, ptr %a42, 0
  %t45 = insertvalue %str %t44, i64 %t43, 1
  %t46 = extractvalue %str { ptr @.str.15, i64 22 }, 0
  %t47 = extractvalue %str { ptr @.str.15, i64 22 }, 1
  call void @veles_call_push(ptr %t46)
  %t48 = call i64 @v_main.halvings(i64 1024)
  call void @veles_call_pop()
  %t50 = call i64 @veles_i64_format(ptr %a49, i64 %t48)
  %t51 = insertvalue %str undef, ptr %a49, 0
  %t52 = insertvalue %str %t51, i64 %t50, 1
  %t55 = getelementptr [9 x %str], ptr %a54, i64 0, i64 0
  store %str %t7, ptr %t55
  %t56 = getelementptr [9 x %str], ptr %a54, i64 0, i64 1
  store %str { ptr @.str.16, i64 1 }, ptr %t56
  %t57 = getelementptr [9 x %str], ptr %a54, i64 0, i64 2
  store %str %t16, ptr %t57
  %t58 = getelementptr [9 x %str], ptr %a54, i64 0, i64 3
  store %str { ptr @.str.16, i64 1 }, ptr %t58
  %t59 = getelementptr [9 x %str], ptr %a54, i64 0, i64 4
  store %str %t23, ptr %t59
  %t60 = getelementptr [9 x %str], ptr %a54, i64 0, i64 5
  store %str { ptr @.str.16, i64 1 }, ptr %t60
  %t61 = getelementptr [9 x %str], ptr %a54, i64 0, i64 6
  store %str %t45, ptr %t61
  %t62 = getelementptr [9 x %str], ptr %a54, i64 0, i64 7
  store %str { ptr @.str.16, i64 1 }, ptr %t62
  %t63 = getelementptr [9 x %str], ptr %a54, i64 0, i64 8
  store %str %t52, ptr %t63
  call void @veles_string_concat_n(ptr %a53, ptr %a54, i64 9)
  %t64 = load %str, ptr %a53
  %t65 = extractvalue %str { ptr @.str.17, i64 20 }, 0
  %t66 = extractvalue %str { ptr @.str.17, i64 20 }, 1
  call void @veles_call_push(ptr %t65)
  call void @v_std.io.println(%str %t64)
  call void @veles_call_pop()
  call void @veles_frame_return(ptr %task, ptr %link, ptr null, i64 0, i64 0, ptr null)
  br label %coro.final
coro.final:
  %coro.fs = call i8 @llvm.coro.suspend(token none, i1 true)
  switch i8 %coro.fs, label %coro.suspend [ i8 0, label %coro.trap i8 1, label %coro.cleanup ]
coro.trap:
  unreachable
coro.cleanup:
  %coro.freed = call ptr @llvm.coro.free(token %coro.id, ptr %coro.hdl)
  br label %coro.suspend
coro.suspend:
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false)
  ret ptr %coro.hdl
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.18, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.18, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task, ptr @veles.root.link)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [1 x i8] c"\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:15:9\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:20:27\00"
@.str.5 = private unnamed_addr constant [20 x i8] c"main.vs:24:30\00twice\00"
@.str.6 = private unnamed_addr constant [14 x i8] c"main.vs:24:21\00"
@.str.7 = private unnamed_addr constant [13 x i8] c"main.vs:31:5\00"
@.str.8 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:41:5\00"
@.str.10 = private unnamed_addr constant [13 x i8] c"main.vs:42:5\00"
@.str.11 = private unnamed_addr constant [25 x i8] c"main.vs:48:17\00sumSquares\00"
@.str.12 = private unnamed_addr constant [28 x i8] c"main.vs:48:36\00firstNegative\00"
@.str.13 = private unnamed_addr constant [23 x i8] c"main.vs:48:68\00withCall\00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:48:84\00ceding\00"
@.str.15 = private unnamed_addr constant [23 x i8] c"main.vs:48:98\00halvings\00"
@.str.16 = private unnamed_addr constant [2 x i8] c" \00"
@.str.17 = private unnamed_addr constant [21 x i8] c"main.vs:48:3\00println\00"
@.str.18 = private unnamed_addr constant [6 x i8] c"\00main\00"
