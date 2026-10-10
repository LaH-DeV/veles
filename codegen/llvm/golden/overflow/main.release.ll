define i8 @v_main.add(i8 %p1, i8 %p2) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = load i8, ptr %a2
  %t5 = add i8 %t3, %t4
  ret i8 %t5
}

define i8 @v_main.sub(i8 %p1, i8 %p2) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = load i8, ptr %a2
  %t5 = sub i8 %t3, %t4
  ret i8 %t5
}

define i64 @v_main.mul(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t5 = mul i64 %t3, %t4
  ret i64 %t5
}

define i32 @v_main.neg(i32 %p1) {
entry:
  %a1 = alloca i32
  store i32 %p1, ptr %a1
  %t2 = load i32, ptr %a1
  %t3 = sub i32 0, %t2
  ret i32 %t3
}

define i64 @v_main.quot(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t6 = icmp eq i64 %t4, 0
  br i1 %t6, label %divzero.1, label %div.ok.2
divzero.1:
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t10 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
div.ok.2:
  %t11 = icmp eq i64 %t4, -1
  %t12 = select i1 %t11, i64 1, i64 %t4
  %t13 = sdiv i64 %t3, %t12
  %t14 = sub i64 0, %t3
  %t5 = select i1 %t11, i64 %t14, i64 %t13
  ret i64 %t5
}

define i16 @v_main.absolute(i16 %p1) {
entry:
  %a1 = alloca i16
  store i16 %p1, ptr %a1
  %t2 = load i16, ptr %a1
  %t3 = call i16 @llvm.abs.i16(i16 %t2, i1 false)
  ret i16 %t3
}

define i8 @v_main.bump(i8 %p1) {
entry:
  %a1 = alloca i8
  %a3 = alloca i8
  store i8 %p1, ptr %a1
  %t2 = load i8, ptr %a1
  store i8 %t2, ptr %a3
  %t4 = load i8, ptr %a3
  %t5 = add i8 %t4, 1
  store i8 %t5, ptr %a3
  %t6 = load i8, ptr %a3
  ret i8 %t6
}

define i8 @v_main.wrapped(i8 %p1, i8 %p2) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = load i8, ptr %a2
  %t5 = add i8 %t3, %t4
  ret i8 %t5
}

define i8 @v_main.narrow(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = trunc i64 %t2 to i8
  ret i8 %t3
}

define i64 @v_main.sumUpTo(i8 %p1) {
entry:
  %a1 = alloca i8
  %a2 = alloca i64
  %a7 = alloca { i8, i8, i1 }
  %a10 = alloca i8
  %a13 = alloca i8
  %a14 = alloca i1
  %a15 = alloca i1
  %a17 = alloca i1
  %a29 = alloca i8
  %a34 = alloca i1
  store i8 %p1, ptr %a1
  store i64 0, ptr %a2
  %t3 = load i8, ptr %a1
  %t4 = insertvalue { i8, i8, i1 } undef, i8 250, 0
  %t5 = insertvalue { i8, i8, i1 } %t4, i8 %t3, 1
  %t6 = insertvalue { i8, i8, i1 } %t5, i1 true, 2
  store { i8, i8, i1 } %t6, ptr %a7
  %t8 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 0
  %t9 = load i8, ptr %t8
  store i8 %t9, ptr %a10
  %t11 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 1
  %t12 = load i8, ptr %t11
  store i8 %t12, ptr %a13
  store i1 false, ptr %a14
  br label %loop.cond.1
loop.cond.1:
  %t16 = load i1, ptr %a14
  br i1 %t16, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a15
  br label %if.end.6
if.else.7:
  %t18 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 2
  %t19 = load i1, ptr %t18
  br i1 %t19, label %if.then.8, label %if.else.10
if.then.8:
  %t20 = load i8, ptr %a10
  %t21 = load i8, ptr %a13
  %t22 = icmp ule i8 %t20, %t21
  store i1 %t22, ptr %a17
  br label %if.end.9
if.else.10:
  %t23 = load i8, ptr %a10
  %t24 = load i8, ptr %a13
  %t25 = icmp ult i8 %t23, %t24
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
  %t28 = load i8, ptr %a10
  store i8 %t28, ptr %a29
  %t30 = load i64, ptr %a2
  %t31 = load i8, ptr %a29
  %t32 = zext i8 %t31 to i64
  %t33 = add i64 %t30, %t32
  store i64 %t33, ptr %a2
  br label %loop.post.2
loop.post.2:
  %t35 = getelementptr inbounds { i8, i8, i1 }, ptr %a7, i32 0, i32 2
  %t36 = load i1, ptr %t35
  store i1 %t36, ptr %a34
  br i1 %t36, label %sc.rhs.11, label %sc.end.12
sc.rhs.11:
  %t37 = load i8, ptr %a10
  %t38 = load i8, ptr %a13
  %t39 = icmp eq i8 %t37, %t38
  store i1 %t39, ptr %a34
  br label %sc.end.12
sc.end.12:
  %t40 = load i1, ptr %a34
  br i1 %t40, label %if.then.13, label %if.else.15
if.then.13:
  store i1 true, ptr %a14
  br label %if.end.14
if.else.15:
  %t41 = load i8, ptr %a10
  %t42 = add i8 %t41, 1
  store i8 %t42, ptr %a10
  br label %if.end.14
if.end.14:
  %t43 = load volatile i32, ptr @veles_attention_line, align 64
  %t44 = icmp ne i32 %t43, 0
  br i1 %t44, label %safepoint.16, label %safepoint.on.17, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.16:
  call void @veles_backedge_plain()
  br label %safepoint.on.17
safepoint.on.17:
  br label %loop.cond.1
loop.end.3:
  %t45 = load i64, ptr %a2
  ret i64 %t45
}

define void @v_main.edges(i8 %p1, i8 %p2, i8 %p3) {
entry:
  %a1 = alloca i8
  %a2 = alloca i8
  %a3 = alloca i8
  %a10 = alloca { i8, i8, i1 }
  %a12 = alloca %S.std.prelude.RangeIter_u8_
  %a14 = alloca [21 x i8]
  %a18 = alloca %str
  %a29 = alloca { i8, i8, i1 }
  %a31 = alloca [21 x i8]
  %a42 = alloca { i8, i8, i1 }
  %a44 = alloca [21 x i8]
  %a48 = alloca %str
  %a49 = alloca [4 x %str]
  %a60 = alloca { i8, i8, i1 }
  %a62 = alloca %S.std.prelude.RangeStepIter_i8_
  %a65 = alloca %str
  %a76 = alloca { i8, i8, i1 }
  %a78 = alloca %S.std.prelude.RangeStepIter_i8_
  %a80 = alloca %S.std.prelude.RangeStepIter_i8_
  %a88 = alloca { i8, i8, i1 }
  %a90 = alloca %S.std.prelude.RangeStepIter_i8_
  %a92 = alloca %S.std.prelude.RangeStepIter_i8_
  %a95 = alloca %str
  %a96 = alloca [4 x %str]
  store i8 %p1, ptr %a1
  store i8 %p2, ptr %a2
  store i8 %p3, ptr %a3
  %t4 = load i8, ptr %a3
  %t5 = sub i8 %t4, 5
  %t6 = load i8, ptr %a3
  %t7 = insertvalue { i8, i8, i1 } undef, i8 %t5, 0
  %t8 = insertvalue { i8, i8, i1 } %t7, i8 %t6, 1
  %t9 = insertvalue { i8, i8, i1 } %t8, i1 true, 2
  store { i8, i8, i1 } %t9, ptr %a10
  %t11 = call %S.std.prelude.RangeIter_u8_ @v_std.prelude.Iterable.iter_Self_Range_u8_T_u8_(ptr %a10)
  store %S.std.prelude.RangeIter_u8_ %t11, ptr %a12
  %t13 = call i64 @v_std.prelude.Iterator.count_Self_std.prelude.RangeIter_u8_T_u8_(ptr %a12)
  %t15 = call i64 @veles_i64_format(ptr %a14, i64 %t13)
  %t16 = insertvalue %str undef, ptr %a14, 0
  %t17 = insertvalue %str %t16, i64 %t15, 1
  %t19 = extractvalue %str { ptr @.str.3, i64 25 }, 0
  %t20 = extractvalue %str { ptr @.str.3, i64 25 }, 1
  %t21 = extractvalue %str %t17, 0
  %t22 = extractvalue %str %t17, 1
  call void @veles_string_concat(ptr %a18, ptr %t19, i64 %t20, ptr %t21, i64 %t22)
  %t23 = load %str, ptr %a18
  call void @v_std.io.println(%str %t23)
  %t24 = load i8, ptr %a1
  %t25 = load i8, ptr %a2
  %t26 = insertvalue { i8, i8, i1 } undef, i8 %t24, 0
  %t27 = insertvalue { i8, i8, i1 } %t26, i8 %t25, 1
  %t28 = insertvalue { i8, i8, i1 } %t27, i1 true, 2
  store { i8, i8, i1 } %t28, ptr %a29
  %t30 = call i64 @v_std.prelude.extend.Range_T.len_T_i8_(ptr %a29)
  %t32 = call i64 @veles_i64_format(ptr %a31, i64 %t30)
  %t33 = insertvalue %str undef, ptr %a31, 0
  %t34 = insertvalue %str %t33, i64 %t32, 1
  %t35 = load i8, ptr %a1
  %t36 = add i8 %t35, 127
  %t37 = add i8 %t36, 1
  %t38 = load i8, ptr %a1
  %t39 = insertvalue { i8, i8, i1 } undef, i8 %t37, 0
  %t40 = insertvalue { i8, i8, i1 } %t39, i8 %t38, 1
  %t41 = insertvalue { i8, i8, i1 } %t40, i1 false, 2
  store { i8, i8, i1 } %t41, ptr %a42
  %t43 = call i64 @v_std.prelude.extend.Range_T.len_T_i8_(ptr %a42)
  %t45 = call i64 @veles_i64_format(ptr %a44, i64 %t43)
  %t46 = insertvalue %str undef, ptr %a44, 0
  %t47 = insertvalue %str %t46, i64 %t45, 1
  %t50 = getelementptr [4 x %str], ptr %a49, i64 0, i64 0
  store %str { ptr @.str.4, i64 20 }, ptr %t50
  %t51 = getelementptr [4 x %str], ptr %a49, i64 0, i64 1
  store %str %t34, ptr %t51
  %t52 = getelementptr [4 x %str], ptr %a49, i64 0, i64 2
  store %str { ptr @.str.5, i64 21 }, ptr %t52
  %t53 = getelementptr [4 x %str], ptr %a49, i64 0, i64 3
  store %str %t47, ptr %t53
  call void @veles_string_concat_n(ptr %a48, ptr %a49, i64 4)
  %t54 = load %str, ptr %a48
  call void @v_std.io.println(%str %t54)
  %t55 = load i8, ptr %a1
  %t56 = load i8, ptr %a2
  %t57 = insertvalue { i8, i8, i1 } undef, i8 %t55, 0
  %t58 = insertvalue { i8, i8, i1 } %t57, i8 %t56, 1
  %t59 = insertvalue { i8, i8, i1 } %t58, i1 true, 2
  store { i8, i8, i1 } %t59, ptr %a60
  %t61 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.step_T_i8_(ptr %a60, i8 100, %str { ptr @.str.6, i64 13 })
  store %S.std.prelude.RangeStepIter_i8_ %t61, ptr %a62
  %t63 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a62)
  %t64 = call %str @show.List_i8_(ptr %t63)
  %t66 = extractvalue %str { ptr @.str.7, i64 24 }, 0
  %t67 = extractvalue %str { ptr @.str.7, i64 24 }, 1
  %t68 = extractvalue %str %t64, 0
  %t69 = extractvalue %str %t64, 1
  call void @veles_string_concat(ptr %a65, ptr %t66, i64 %t67, ptr %t68, i64 %t69)
  %t70 = load %str, ptr %a65
  call void @v_std.io.println(%str %t70)
  %t71 = load i8, ptr %a1
  %t72 = load i8, ptr %a2
  %t73 = insertvalue { i8, i8, i1 } undef, i8 %t71, 0
  %t74 = insertvalue { i8, i8, i1 } %t73, i8 %t72, 1
  %t75 = insertvalue { i8, i8, i1 } %t74, i1 true, 2
  store { i8, i8, i1 } %t75, ptr %a76
  %t77 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.reversed_T_i8_(ptr %a76)
  store %S.std.prelude.RangeStepIter_i8_ %t77, ptr %a78
  %t79 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.RangeStepIter.step_T_i8_(ptr %a78, i8 100, %str { ptr @.str.8, i64 13 })
  store %S.std.prelude.RangeStepIter_i8_ %t79, ptr %a80
  %t81 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a80)
  %t82 = call %str @show.List_i8_(ptr %t81)
  %t83 = load i8, ptr %a1
  %t84 = load i8, ptr %a2
  %t85 = insertvalue { i8, i8, i1 } undef, i8 %t83, 0
  %t86 = insertvalue { i8, i8, i1 } %t85, i8 %t84, 1
  %t87 = insertvalue { i8, i8, i1 } %t86, i1 true, 2
  store { i8, i8, i1 } %t87, ptr %a88
  %t89 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.step_T_i8_(ptr %a88, i8 100, %str { ptr @.str.9, i64 14 })
  store %S.std.prelude.RangeStepIter_i8_ %t89, ptr %a90
  %t91 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.RangeStepIter.reversed_T_i8_(ptr %a90)
  store %S.std.prelude.RangeStepIter_i8_ %t91, ptr %a92
  %t93 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a92)
  %t94 = call %str @show.List_i8_(ptr %t93)
  %t97 = getelementptr [4 x %str], ptr %a96, i64 0, i64 0
  store %str { ptr @.str.10, i64 21 }, ptr %t97
  %t98 = getelementptr [4 x %str], ptr %a96, i64 0, i64 1
  store %str %t82, ptr %t98
  %t99 = getelementptr [4 x %str], ptr %a96, i64 0, i64 2
  store %str { ptr @.str.11, i64 23 }, ptr %t99
  %t100 = getelementptr [4 x %str], ptr %a96, i64 0, i64 3
  store %str %t94, ptr %t100
  call void @veles_string_concat_n(ptr %a95, ptr %a96, i64 4)
  %t101 = load %str, ptr %a95
  call void @v_std.io.println(%str %t101)
  ret void
}

define i64 @v_main.call({ ptr, ptr } %p1) {
entry:
  %a1 = alloca { ptr, ptr }
  store { ptr, ptr } %p1, ptr %a1
  %t2 = load { ptr, ptr }, ptr %a1
  %t3 = extractvalue { ptr, ptr } %t2, 0
  %t4 = extractvalue { ptr, ptr } %t2, 1
  %t5 = call i64 %t3(ptr %t4)
  ret i64 %t5
}

define ptr @v_main.attempt(ptr %task, ptr %link, %str %p1, { ptr, ptr } %p2) presplitcoroutine {
entry:
  %a1 = alloca %str
  %a2 = alloca { ptr, ptr }
  %a4 = alloca ptr
  %a5 = alloca { ptr, ptr, ptr, ptr }
  %a11 = alloca ptr
  %a24 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a27 = alloca i64
  %a32 = alloca i64
  %a40 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a46 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a53 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a55 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a62 = alloca i64
  %a65 = alloca [21 x i8]
  %a69 = alloca %str
  %a70 = alloca [3 x %str]
  %a81 = alloca %S.std.prelude.Panic
  %a85 = alloca %str
  %a86 = alloca [3 x %str]
  store %str %p1, ptr %a1
  store { ptr, ptr } %p2, ptr %a2
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t3 = call ptr @veles_scope_begin(ptr %task, i64 0, i64 %coro.depth)
  store ptr %t3, ptr %a4
  call void @veles_cleanup_push(ptr %a5, ptr @scope.cancel.thunk, ptr %a4, ptr @cleanup.move.word)
  %t6 = load ptr, ptr %a4
  %t7 = call ptr @veles_task_launch(ptr %t6, i64 0)
  %t8 = load { ptr, ptr }, ptr %a2
  %t9 = call ptr @veles_alloc_words(i64 24)
  %t10 = getelementptr inbounds { { ptr, ptr } }, ptr %t9, i32 0, i32 0
  store { ptr, ptr } %t8, ptr %t10
  call void @veles_task_spawn(ptr %t7, ptr @entry.ramp.v_main.call, ptr %t9)
  store ptr %t7, ptr %a11
  br label %scope.wait.1
scope.wait.1:
  %t12 = load ptr, ptr %a4
  %t13 = call i64 @veles_scope_wait(ptr %task, ptr %t12)
  %t14 = icmp ne i64 %t13, 0
  br i1 %t14, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t15 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t15, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t16 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t17 = icmp ne i64 %t16, 0
  br i1 %t17, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_cleanup_pop(ptr %a5)
  %t18 = load ptr, ptr %a4
  call void @veles_scope_cancel(ptr %t18)
  br label %abandon.wait.7
abandon.wait.7:
  %t19 = call i64 @veles_scope_wait(ptr %task, ptr %t18)
  %t20 = icmp ne i64 %t19, 0
  br i1 %t20, label %abandon.done.8, label %abandon.susp.9
abandon.susp.9:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t21 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t21, label %coro.suspend [ i8 0, label %resume.10 i8 1, label %coro.cleanup ]
resume.10:
  br label %abandon.wait.7
abandon.done.8:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.6:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop(ptr %a5)
  %t22 = load ptr, ptr %a11
  %t23 = call ptr @veles_task_result(ptr %t22)
  %t25 = call i64 @veles_task_panicked(ptr %t22)
  %t26 = icmp ne i64 %t25, 0
  br i1 %t26, label %gather.panic.11, label %gather.value.12
gather.panic.11:
  %t28 = call ptr @veles_task_panic_msg(ptr %t22, ptr %a27)
  %t29 = load i64, ptr %a27
  %t30 = insertvalue %str undef, ptr %t28, 0
  %t31 = insertvalue %str %t30, i64 %t29, 1
  %t33 = call ptr @veles_task_panic_loc(ptr %t22, ptr %a32)
  %t34 = load i64, ptr %a32
  %t35 = insertvalue %str undef, ptr %t33, 0
  %t36 = insertvalue %str %t35, i64 %t34, 1
  %t37 = insertvalue %S.std.prelude.Panic undef, %str %t31, 0
  %t38 = insertvalue %S.std.prelude.Panic %t37, %str %t36, 1
  %t39 = insertvalue %S._prelude_.Err_i64_std.prelude.Panic_ undef, %S.std.prelude.Panic %t38, 0
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a40)
  store %V._prelude_.Result_i64_std.prelude.Panic_ zeroinitializer, ptr %a40
  %t41 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a40, i32 0, i32 0
  store i32 1, ptr %t41
  %t42 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a40, i32 0, i32 1
  store %S._prelude_.Err_i64_std.prelude.Panic_ %t39, ptr %t42
  %t43 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a40
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t43, ptr %a24
  br label %gather.join.13
gather.value.12:
  %t44 = load i64, ptr %t23
  %t45 = insertvalue %S._prelude_.Ok_i64_std.prelude.Panic_ undef, i64 %t44, 0
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a46)
  store %V._prelude_.Result_i64_std.prelude.Panic_ zeroinitializer, ptr %a46
  %t47 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a46, i32 0, i32 0
  store i32 0, ptr %t47
  %t48 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a46, i32 0, i32 1
  store %S._prelude_.Ok_i64_std.prelude.Panic_ %t45, ptr %t48
  %t49 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a46
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t49, ptr %a24
  br label %gather.join.13
gather.join.13:
  %t50 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a24
  %t51 = insertvalue { %V._prelude_.Result_i64_std.prelude.Panic_ } undef, %V._prelude_.Result_i64_std.prelude.Panic_ %t50, 0
  %t52 = extractvalue { %V._prelude_.Result_i64_std.prelude.Panic_ } %t51, 0
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t52, ptr %a53
  %t54 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a53
  store %V._prelude_.Result_i64_std.prelude.Panic_ %t54, ptr %a55
  br label %when.arm.15
when.arm.15:
  %t56 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55
  %t57 = extractvalue %V._prelude_.Result_i64_std.prelude.Panic_ %t56, 0
  %t58 = icmp eq i32 %t57, 0
  br i1 %t58, label %when.bind.18, label %when.arm.16
when.bind.18:
  %t59 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55, i32 0, i32 1
  %t60 = getelementptr inbounds %S._prelude_.Ok_i64_std.prelude.Panic_, ptr %t59, i32 0, i32 0
  %t61 = load i64, ptr %t60
  store i64 %t61, ptr %a62
  br label %when.body.17
when.body.17:
  %t63 = load %str, ptr %a1
  %t64 = load i64, ptr %a62
  %t66 = call i64 @veles_i64_format(ptr %a65, i64 %t64)
  %t67 = insertvalue %str undef, ptr %a65, 0
  %t68 = insertvalue %str %t67, i64 %t66, 1
  %t71 = getelementptr [3 x %str], ptr %a70, i64 0, i64 0
  store %str %t63, ptr %t71
  %t72 = getelementptr [3 x %str], ptr %a70, i64 0, i64 1
  store %str { ptr @.str.12, i64 3 }, ptr %t72
  %t73 = getelementptr [3 x %str], ptr %a70, i64 0, i64 2
  store %str %t68, ptr %t73
  call void @veles_string_concat_n(ptr %a69, ptr %a70, i64 3)
  %t74 = load %str, ptr %a69
  call void @v_std.io.println(%str %t74)
  br label %when.end.14
when.arm.16:
  %t75 = load %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55
  %t76 = extractvalue %V._prelude_.Result_i64_std.prelude.Panic_ %t75, 0
  %t77 = icmp eq i32 %t76, 1
  br i1 %t77, label %when.bind.21, label %when.arm.19
when.bind.21:
  %t78 = getelementptr inbounds %V._prelude_.Result_i64_std.prelude.Panic_, ptr %a55, i32 0, i32 1
  %t79 = getelementptr inbounds %S._prelude_.Err_i64_std.prelude.Panic_, ptr %t78, i32 0, i32 0
  %t80 = load %S.std.prelude.Panic, ptr %t79
  store %S.std.prelude.Panic %t80, ptr %a81
  br label %when.body.20
when.body.20:
  %t82 = load %str, ptr %a1
  %t83 = getelementptr inbounds %S.std.prelude.Panic, ptr %a81, i32 0, i32 0
  %t84 = load %str, ptr %t83
  %t87 = getelementptr [3 x %str], ptr %a86, i64 0, i64 0
  store %str %t82, ptr %t87
  %t88 = getelementptr [3 x %str], ptr %a86, i64 0, i64 1
  store %str { ptr @.str.13, i64 2 }, ptr %t88
  %t89 = getelementptr [3 x %str], ptr %a86, i64 0, i64 2
  store %str %t84, ptr %t89
  call void @veles_string_concat_n(ptr %a85, ptr %a86, i64 3)
  %t90 = load %str, ptr %a85
  call void @v_std.io.println(%str %t90)
  br label %when.end.14
when.arm.19:
  %t91 = extractvalue %str { ptr @.str.14, i64 33 }, 0
  %t92 = extractvalue %str { ptr @.str.14, i64 33 }, 1
  %t93 = extractvalue %str { ptr @.str.15, i64 12 }, 0
  %t94 = extractvalue %str { ptr @.str.15, i64 12 }, 1
  call void @veles_panic_at(ptr %t91, i64 %t92, ptr %t93, i64 %t94)
  unreachable
when.end.14:
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
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false, token none)
  ret ptr %coro.hdl
}

define ptr @v_main.main(ptr %task, ptr %link) presplitcoroutine {
entry:
  %a3 = alloca { ptr, i64, i64 }
  %a19 = alloca { ptr, i64, i64 }
  %a35 = alloca { ptr, i64, i64 }
  %a51 = alloca { ptr, i64, i64 }
  %a67 = alloca { ptr, i64, i64 }
  %a83 = alloca { ptr, i64, i64 }
  %a99 = alloca { ptr, i64, i64 }
  %a115 = alloca [21 x i8]
  %a119 = alloca %str
  %a127 = alloca [21 x i8]
  %a133 = alloca [21 x i8]
  %a137 = alloca %str
  %a138 = alloca [4 x %str]
  %a145 = alloca [21 x i8]
  %a149 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t1 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t2 = insertvalue { ptr, ptr } %t1, ptr null, 1
  %t4 = getelementptr inbounds { ptr, i64, i64 }, ptr %a3, i32 0, i32 0
  store ptr %coro.hdl, ptr %t4
  %t5 = add i64 %coro.depth, 1
  %t6 = getelementptr inbounds { ptr, i64, i64 }, ptr %a3, i32 0, i32 1
  store i64 %t5, ptr %t6
  %t7 = getelementptr inbounds { ptr, i64, i64 }, ptr %a3, i32 0, i32 2
  store i64 0, ptr %t7
  %t8 = call ptr @v_main.attempt(ptr %task, ptr %a3, %str { ptr @.str.16, i64 12 }, { ptr, ptr } %t2)
  br label %call.check.1
call.check.1:
  %t9 = load i64, ptr %t7
  %t10 = icmp ne i64 %t9, 0
  br i1 %t10, label %call.done.2, label %call.wait.3
call.wait.3:
  %t11 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t11, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t12 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t13 = icmp ne i64 %t12, 0
  br i1 %t13, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.6:
  br label %call.check.1
call.done.2:
  call void @veles.frame.free(ptr %task, ptr %t8)
  %t14 = icmp eq i64 %t9, 2
  br i1 %t14, label %call.unwound.7, label %call.ok.8, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.7:
  %t15 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t16 = icmp ne i64 %t15, 0
  br i1 %t16, label %call.cancelled.9, label %call.look.10
call.look.10:
  br label %call.cancelled.9
call.cancelled.9:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.8:
  %t17 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2, 0
  %t18 = insertvalue { ptr, ptr } %t17, ptr null, 1
  %t20 = getelementptr inbounds { ptr, i64, i64 }, ptr %a19, i32 0, i32 0
  store ptr %coro.hdl, ptr %t20
  %t21 = add i64 %coro.depth, 1
  %t22 = getelementptr inbounds { ptr, i64, i64 }, ptr %a19, i32 0, i32 1
  store i64 %t21, ptr %t22
  %t23 = getelementptr inbounds { ptr, i64, i64 }, ptr %a19, i32 0, i32 2
  store i64 0, ptr %t23
  %t24 = call ptr @v_main.attempt(ptr %task, ptr %a19, %str { ptr @.str.17, i64 10 }, { ptr, ptr } %t18)
  br label %call.check.11
call.check.11:
  %t25 = load i64, ptr %t23
  %t26 = icmp ne i64 %t25, 0
  br i1 %t26, label %call.done.12, label %call.wait.13
call.wait.13:
  %t27 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t27, label %coro.suspend [ i8 0, label %resume.14 i8 1, label %coro.cleanup ]
resume.14:
  %t28 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t29 = icmp ne i64 %t28, 0
  br i1 %t29, label %cancelled.15, label %cont.16
cancelled.15:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.16:
  br label %call.check.11
call.done.12:
  call void @veles.frame.free(ptr %task, ptr %t24)
  %t30 = icmp eq i64 %t25, 2
  br i1 %t30, label %call.unwound.17, label %call.ok.18, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.17:
  %t31 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t32 = icmp ne i64 %t31, 0
  br i1 %t32, label %call.cancelled.19, label %call.look.20
call.look.20:
  br label %call.cancelled.19
call.cancelled.19:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.18:
  %t33 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda3, 0
  %t34 = insertvalue { ptr, ptr } %t33, ptr null, 1
  %t36 = getelementptr inbounds { ptr, i64, i64 }, ptr %a35, i32 0, i32 0
  store ptr %coro.hdl, ptr %t36
  %t37 = add i64 %coro.depth, 1
  %t38 = getelementptr inbounds { ptr, i64, i64 }, ptr %a35, i32 0, i32 1
  store i64 %t37, ptr %t38
  %t39 = getelementptr inbounds { ptr, i64, i64 }, ptr %a35, i32 0, i32 2
  store i64 0, ptr %t39
  %t40 = call ptr @v_main.attempt(ptr %task, ptr %a35, %str { ptr @.str.18, i64 8 }, { ptr, ptr } %t34)
  br label %call.check.21
call.check.21:
  %t41 = load i64, ptr %t39
  %t42 = icmp ne i64 %t41, 0
  br i1 %t42, label %call.done.22, label %call.wait.23
call.wait.23:
  %t43 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t43, label %coro.suspend [ i8 0, label %resume.24 i8 1, label %coro.cleanup ]
resume.24:
  %t44 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t45 = icmp ne i64 %t44, 0
  br i1 %t45, label %cancelled.25, label %cont.26
cancelled.25:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.26:
  br label %call.check.21
call.done.22:
  call void @veles.frame.free(ptr %task, ptr %t40)
  %t46 = icmp eq i64 %t41, 2
  br i1 %t46, label %call.unwound.27, label %call.ok.28, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.27:
  %t47 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t48 = icmp ne i64 %t47, 0
  br i1 %t48, label %call.cancelled.29, label %call.look.30
call.look.30:
  br label %call.cancelled.29
call.cancelled.29:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.28:
  %t49 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t50 = insertvalue { ptr, ptr } %t49, ptr null, 1
  %t52 = getelementptr inbounds { ptr, i64, i64 }, ptr %a51, i32 0, i32 0
  store ptr %coro.hdl, ptr %t52
  %t53 = add i64 %coro.depth, 1
  %t54 = getelementptr inbounds { ptr, i64, i64 }, ptr %a51, i32 0, i32 1
  store i64 %t53, ptr %t54
  %t55 = getelementptr inbounds { ptr, i64, i64 }, ptr %a51, i32 0, i32 2
  store i64 0, ptr %t55
  %t56 = call ptr @v_main.attempt(ptr %task, ptr %a51, %str { ptr @.str.19, i64 10 }, { ptr, ptr } %t50)
  br label %call.check.31
call.check.31:
  %t57 = load i64, ptr %t55
  %t58 = icmp ne i64 %t57, 0
  br i1 %t58, label %call.done.32, label %call.wait.33
call.wait.33:
  %t59 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t59, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t60 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t61 = icmp ne i64 %t60, 0
  br i1 %t61, label %cancelled.35, label %cont.36
cancelled.35:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.36:
  br label %call.check.31
call.done.32:
  call void @veles.frame.free(ptr %task, ptr %t56)
  %t62 = icmp eq i64 %t57, 2
  br i1 %t62, label %call.unwound.37, label %call.ok.38, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.37:
  %t63 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t64 = icmp ne i64 %t63, 0
  br i1 %t64, label %call.cancelled.39, label %call.look.40
call.look.40:
  br label %call.cancelled.39
call.cancelled.39:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.38:
  %t65 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda5, 0
  %t66 = insertvalue { ptr, ptr } %t65, ptr null, 1
  %t68 = getelementptr inbounds { ptr, i64, i64 }, ptr %a67, i32 0, i32 0
  store ptr %coro.hdl, ptr %t68
  %t69 = add i64 %coro.depth, 1
  %t70 = getelementptr inbounds { ptr, i64, i64 }, ptr %a67, i32 0, i32 1
  store i64 %t69, ptr %t70
  %t71 = getelementptr inbounds { ptr, i64, i64 }, ptr %a67, i32 0, i32 2
  store i64 0, ptr %t71
  %t72 = call ptr @v_main.attempt(ptr %task, ptr %a67, %str { ptr @.str.20, i64 8 }, { ptr, ptr } %t66)
  br label %call.check.41
call.check.41:
  %t73 = load i64, ptr %t71
  %t74 = icmp ne i64 %t73, 0
  br i1 %t74, label %call.done.42, label %call.wait.43
call.wait.43:
  %t75 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t75, label %coro.suspend [ i8 0, label %resume.44 i8 1, label %coro.cleanup ]
resume.44:
  %t76 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t77 = icmp ne i64 %t76, 0
  br i1 %t77, label %cancelled.45, label %cont.46
cancelled.45:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.46:
  br label %call.check.41
call.done.42:
  call void @veles.frame.free(ptr %task, ptr %t72)
  %t78 = icmp eq i64 %t73, 2
  br i1 %t78, label %call.unwound.47, label %call.ok.48, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.47:
  %t79 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t80 = icmp ne i64 %t79, 0
  br i1 %t80, label %call.cancelled.49, label %call.look.50
call.look.50:
  br label %call.cancelled.49
call.cancelled.49:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.48:
  %t81 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda6, 0
  %t82 = insertvalue { ptr, ptr } %t81, ptr null, 1
  %t84 = getelementptr inbounds { ptr, i64, i64 }, ptr %a83, i32 0, i32 0
  store ptr %coro.hdl, ptr %t84
  %t85 = add i64 %coro.depth, 1
  %t86 = getelementptr inbounds { ptr, i64, i64 }, ptr %a83, i32 0, i32 1
  store i64 %t85, ptr %t86
  %t87 = getelementptr inbounds { ptr, i64, i64 }, ptr %a83, i32 0, i32 2
  store i64 0, ptr %t87
  %t88 = call ptr @v_main.attempt(ptr %task, ptr %a83, %str { ptr @.str.21, i64 15 }, { ptr, ptr } %t82)
  br label %call.check.51
call.check.51:
  %t89 = load i64, ptr %t87
  %t90 = icmp ne i64 %t89, 0
  br i1 %t90, label %call.done.52, label %call.wait.53
call.wait.53:
  %t91 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t91, label %coro.suspend [ i8 0, label %resume.54 i8 1, label %coro.cleanup ]
resume.54:
  %t92 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t93 = icmp ne i64 %t92, 0
  br i1 %t93, label %cancelled.55, label %cont.56
cancelled.55:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.56:
  br label %call.check.51
call.done.52:
  call void @veles.frame.free(ptr %task, ptr %t88)
  %t94 = icmp eq i64 %t89, 2
  br i1 %t94, label %call.unwound.57, label %call.ok.58, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.57:
  %t95 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t96 = icmp ne i64 %t95, 0
  br i1 %t96, label %call.cancelled.59, label %call.look.60
call.look.60:
  br label %call.cancelled.59
call.cancelled.59:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.58:
  %t97 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda7, 0
  %t98 = insertvalue { ptr, ptr } %t97, ptr null, 1
  %t100 = getelementptr inbounds { ptr, i64, i64 }, ptr %a99, i32 0, i32 0
  store ptr %coro.hdl, ptr %t100
  %t101 = add i64 %coro.depth, 1
  %t102 = getelementptr inbounds { ptr, i64, i64 }, ptr %a99, i32 0, i32 1
  store i64 %t101, ptr %t102
  %t103 = getelementptr inbounds { ptr, i64, i64 }, ptr %a99, i32 0, i32 2
  store i64 0, ptr %t103
  %t104 = call ptr @v_main.attempt(ptr %task, ptr %a99, %str { ptr @.str.22, i64 18 }, { ptr, ptr } %t98)
  br label %call.check.61
call.check.61:
  %t105 = load i64, ptr %t103
  %t106 = icmp ne i64 %t105, 0
  br i1 %t106, label %call.done.62, label %call.wait.63
call.wait.63:
  %t107 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t107, label %coro.suspend [ i8 0, label %resume.64 i8 1, label %coro.cleanup ]
resume.64:
  %t108 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t109 = icmp ne i64 %t108, 0
  br i1 %t109, label %cancelled.65, label %cont.66
cancelled.65:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.66:
  br label %call.check.61
call.done.62:
  call void @veles.frame.free(ptr %task, ptr %t104)
  %t110 = icmp eq i64 %t105, 2
  br i1 %t110, label %call.unwound.67, label %call.ok.68, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.67:
  %t111 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t112 = icmp ne i64 %t111, 0
  br i1 %t112, label %call.cancelled.69, label %call.look.70
call.look.70:
  br label %call.cancelled.69
call.cancelled.69:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.68:
  %t113 = call i8 @v_main.wrapped(i8 127, i8 1)
  %t114 = sext i8 %t113 to i64
  %t116 = call i64 @veles_i64_format(ptr %a115, i64 %t114)
  %t117 = insertvalue %str undef, ptr %a115, 0
  %t118 = insertvalue %str %t117, i64 %t116, 1
  %t120 = extractvalue %str { ptr @.str.23, i64 16 }, 0
  %t121 = extractvalue %str { ptr @.str.23, i64 16 }, 1
  %t122 = extractvalue %str %t118, 0
  %t123 = extractvalue %str %t118, 1
  call void @veles_string_concat(ptr %a119, ptr %t120, i64 %t121, ptr %t122, i64 %t123)
  %t124 = load %str, ptr %a119
  call void @v_std.io.println(%str %t124)
  %t125 = call i8 @v_main.narrow(i64 300)
  %t126 = zext i8 %t125 to i64
  %t128 = call i64 @veles_u64_format(ptr %a127, i64 %t126)
  %t129 = insertvalue %str undef, ptr %a127, 0
  %t130 = insertvalue %str %t129, i64 %t128, 1
  %t131 = call i8 @v_main.narrow(i64 -1)
  %t132 = zext i8 %t131 to i64
  %t134 = call i64 @veles_u64_format(ptr %a133, i64 %t132)
  %t135 = insertvalue %str undef, ptr %a133, 0
  %t136 = insertvalue %str %t135, i64 %t134, 1
  %t139 = getelementptr [4 x %str], ptr %a138, i64 0, i64 0
  store %str { ptr @.str.24, i64 12 }, ptr %t139
  %t140 = getelementptr [4 x %str], ptr %a138, i64 0, i64 1
  store %str %t130, ptr %t140
  %t141 = getelementptr [4 x %str], ptr %a138, i64 0, i64 2
  store %str { ptr @.str.25, i64 13 }, ptr %t141
  %t142 = getelementptr [4 x %str], ptr %a138, i64 0, i64 3
  store %str %t136, ptr %t142
  call void @veles_string_concat_n(ptr %a137, ptr %a138, i64 4)
  %t143 = load %str, ptr %a137
  call void @v_std.io.println(%str %t143)
  %t144 = call i64 @v_main.sumUpTo(i8 255)
  %t146 = call i64 @veles_i64_format(ptr %a145, i64 %t144)
  %t147 = insertvalue %str undef, ptr %a145, 0
  %t148 = insertvalue %str %t147, i64 %t146, 1
  %t150 = extractvalue %str { ptr @.str.26, i64 23 }, 0
  %t151 = extractvalue %str { ptr @.str.26, i64 23 }, 1
  %t152 = extractvalue %str %t148, 0
  %t153 = extractvalue %str %t148, 1
  call void @veles_string_concat(ptr %a149, ptr %t150, i64 %t151, ptr %t152, i64 %t153)
  %t154 = load %str, ptr %a149
  call void @v_std.io.println(%str %t154)
  call void @v_main.edges(i8 -128, i8 127, i8 255)
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
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false, token none)
  ret ptr %coro.hdl
}

define i64 @v_main.main.lambda1(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = call i8 @v_main.add(i8 127, i8 1)
  %t3 = sext i8 %t2 to i64
  ret i64 %t3
}

define i64 @v_main.main.lambda2(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = call i8 @v_main.sub(i8 0, i8 1)
  %t3 = zext i8 %t2 to i64
  ret i64 %t3
}

define i64 @v_main.main.lambda3(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = sub i64 -9223372036854775807, 1
  %t3 = call i64 @v_main.mul(i64 %t2, i64 -1)
  ret i64 %t3
}

define i64 @v_main.main.lambda4(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = sub i32 -2147483647, 1
  %t3 = call i32 @v_main.neg(i32 %t2)
  %t4 = sext i32 %t3 to i64
  ret i64 %t4
}

define i64 @v_main.main.lambda5(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = sub i64 -9223372036854775807, 1
  %t3 = call i64 @v_main.quot(i64 %t2, i64 -1)
  ret i64 %t3
}

define i64 @v_main.main.lambda6(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = call i16 @v_main.absolute(i16 -32768)
  %t3 = sext i16 %t2 to i64
  ret i64 %t3
}

define i64 @v_main.main.lambda7(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = call i8 @v_main.bump(i8 127)
  %t3 = sext i8 %t2 to i64
  ret i64 %t3
}

define internal ptr @ramp.v_main.call(ptr %task, ptr %link, { ptr, ptr } %p0) presplitcoroutine {
entry:
  %a2 = alloca i64
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t1 = call i64 @v_main.call({ ptr, ptr } %p0)
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a2)
  store i64 %t1, ptr %a2
  %t3 = load ptr, ptr %link
  %t4 = icmp eq ptr %t3, null
  br i1 %t4, label %ret.task.1, label %ret.call.2
ret.task.1:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a2, i64 8, i64 0, ptr null)
  br label %coro.final
ret.call.2:
  %t5 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %link, i32 0, i32 3
  store i64 %t1, ptr %t5
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
  %coro.ended = call i1 @llvm.coro.end(ptr %coro.hdl, i1 false, token none)
  ret ptr %coro.hdl
}

define internal void @entry.ramp.v_main.call(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { { ptr, ptr } }, ptr %args, i32 0, i32 0
  %t2 = load { ptr, ptr }, ptr %t1
  %t3 = call ptr @ramp.v_main.call(ptr %task, ptr @veles.root.link, { ptr, ptr } %t2)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = call ptr @v_main.main(ptr %task, ptr @veles.root.link)
  call void @veles_task_started(ptr %task, ptr %t1)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:12:34\00"
@.str.3 = private unnamed_addr constant [26 x i8] c"(250..255).iter() yields \00"
@.str.4 = private unnamed_addr constant [21 x i8] c"(-128..127).len() = \00"
@.str.5 = private unnamed_addr constant [22 x i8] c", (0..<-128).len() = \00"
@.str.6 = private unnamed_addr constant [14 x i8] c"main.vs:35:41\00"
@.str.7 = private unnamed_addr constant [25 x i8] c"(-128..127).step(100) = \00"
@.str.8 = private unnamed_addr constant [14 x i8] c"main.vs:36:38\00"
@.str.9 = private unnamed_addr constant [15 x i8] c"main.vs:36:102\00"
@.str.10 = private unnamed_addr constant [22 x i8] c"reversed then step = \00"
@.str.11 = private unnamed_addr constant [24 x i8] c", step then reversed = \00"
@.str.12 = private unnamed_addr constant [4 x i8] c" = \00"
@.str.13 = private unnamed_addr constant [3 x i8] c": \00"
@.str.14 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.15 = private unnamed_addr constant [13 x i8] c"main.vs:45:3\00"
@.str.16 = private unnamed_addr constant [13 x i8] c"127 + 1 (i8)\00"
@.str.17 = private unnamed_addr constant [11 x i8] c"0 - 1 (u8)\00"
@.str.18 = private unnamed_addr constant [9 x i8] c"MIN * -1\00"
@.str.19 = private unnamed_addr constant [11 x i8] c"-MIN (i32)\00"
@.str.20 = private unnamed_addr constant [9 x i8] c"MIN / -1\00"
@.str.21 = private unnamed_addr constant [16 x i8] c"MIN.abs() (i16)\00"
@.str.22 = private unnamed_addr constant [19 x i8] c"x += 1 at 127 (i8)\00"
@.str.23 = private unnamed_addr constant [17 x i8] c"127 +% 1 (i8) = \00"
@.str.24 = private unnamed_addr constant [13 x i8] c"300 as u8 = \00"
@.str.25 = private unnamed_addr constant [14 x i8] c", -1 as u8 = \00"
@.str.26 = private unnamed_addr constant [24 x i8] c"sum of 250..255 (u8) = \00"
