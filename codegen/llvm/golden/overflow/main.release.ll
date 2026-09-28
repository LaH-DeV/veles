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
  %t43 = load volatile i32, ptr @veles_stop_requested, align 4
  %t44 = icmp ne i32 %t43, 0
  br i1 %t44, label %safepoint.16, label %safepoint.on.17, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.16:
  call void @veles_gc_park()
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
  %t61 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.step_T_i8_(ptr %a60, i8 100)
  store %S.std.prelude.RangeStepIter_i8_ %t61, ptr %a62
  %t63 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a62)
  %t64 = call %str @show.List_i8_(ptr %t63)
  %t66 = extractvalue %str { ptr @.str.6, i64 24 }, 0
  %t67 = extractvalue %str { ptr @.str.6, i64 24 }, 1
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
  %t79 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.RangeStepIter.step_T_i8_(ptr %a78, i8 100)
  store %S.std.prelude.RangeStepIter_i8_ %t79, ptr %a80
  %t81 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a80)
  %t82 = call %str @show.List_i8_(ptr %t81)
  %t83 = load i8, ptr %a1
  %t84 = load i8, ptr %a2
  %t85 = insertvalue { i8, i8, i1 } undef, i8 %t83, 0
  %t86 = insertvalue { i8, i8, i1 } %t85, i8 %t84, 1
  %t87 = insertvalue { i8, i8, i1 } %t86, i1 true, 2
  store { i8, i8, i1 } %t87, ptr %a88
  %t89 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.extend.Range_T.step_T_i8_(ptr %a88, i8 100)
  store %S.std.prelude.RangeStepIter_i8_ %t89, ptr %a90
  %t91 = call %S.std.prelude.RangeStepIter_i8_ @v_std.prelude.RangeStepIter.reversed_T_i8_(ptr %a90)
  store %S.std.prelude.RangeStepIter_i8_ %t91, ptr %a92
  %t93 = call ptr @v_std.prelude.Iterator.toList_Self_std.prelude.RangeStepIter_i8_T_i8_(ptr %a92)
  %t94 = call %str @show.List_i8_(ptr %t93)
  %t97 = getelementptr [4 x %str], ptr %a96, i64 0, i64 0
  store %str { ptr @.str.7, i64 21 }, ptr %t97
  %t98 = getelementptr [4 x %str], ptr %a96, i64 0, i64 1
  store %str %t82, ptr %t98
  %t99 = getelementptr [4 x %str], ptr %a96, i64 0, i64 2
  store %str { ptr @.str.8, i64 23 }, ptr %t99
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

define ptr @v_main.attempt(ptr %task, %str %p1, { ptr, ptr } %p2) presplitcoroutine {
entry:
  %a1 = alloca %str
  %a2 = alloca { ptr, ptr }
  %a4 = alloca ptr
  %a5 = alloca { ptr, ptr, ptr }
  %a11 = alloca ptr
  %a24 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a27 = alloca i64
  %a32 = alloca i64
  %a40 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a46 = alloca %V._prelude_.Result_i64_std.prelude.Panic_
  %a52 = alloca { %V._prelude_.Result_i64_std.prelude.Panic_ }
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
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t3 = call ptr @veles_scope_begin(ptr %task, i64 0)
  store ptr %t3, ptr %a4
  call void @veles_cleanup_push(ptr %a5, ptr @scope.cancel.thunk, ptr %a4)
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
  %t15 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t15, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t16 = call i64 @veles_task_cancelled(ptr %task)
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
  %t21 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t21, label %coro.suspend [ i8 0, label %resume.10 i8 1, label %coro.cleanup ]
resume.10:
  br label %abandon.wait.7
abandon.done.8:
  call void @veles_task_finish_cancelled(ptr %task)
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
  store { %V._prelude_.Result_i64_std.prelude.Panic_ } %t51, ptr %a52
  %t53 = load { %V._prelude_.Result_i64_std.prelude.Panic_ }, ptr %a52
  %t54 = extractvalue { %V._prelude_.Result_i64_std.prelude.Panic_ } %t53, 0
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
  store %str { ptr @.str.9, i64 3 }, ptr %t72
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
  store %str { ptr @.str.10, i64 2 }, ptr %t88
  %t89 = getelementptr [3 x %str], ptr %a86, i64 0, i64 2
  store %str %t84, ptr %t89
  call void @veles_string_concat_n(ptr %a85, ptr %a86, i64 3)
  %t90 = load %str, ptr %a85
  call void @v_std.io.println(%str %t90)
  br label %when.end.14
when.arm.19:
  %t91 = extractvalue %str { ptr @.str.11, i64 33 }, 0
  %t92 = extractvalue %str { ptr @.str.11, i64 33 }, 1
  %t93 = extractvalue %str { ptr @.str.12, i64 12 }, 0
  %t94 = extractvalue %str { ptr @.str.12, i64 12 }, 1
  call void @veles_panic_at(ptr %t91, i64 %t92, ptr %t93, i64 %t94)
  unreachable
when.end.14:
  call void @veles_task_finish(ptr %task, ptr null, i64 0, i64 0)
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

define ptr @v_main.main(ptr %task) presplitcoroutine {
entry:
  %a94 = alloca [21 x i8]
  %a98 = alloca %str
  %a106 = alloca [21 x i8]
  %a112 = alloca [21 x i8]
  %a116 = alloca %str
  %a117 = alloca [4 x %str]
  %a124 = alloca [21 x i8]
  %a128 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t2 = insertvalue { ptr, ptr } %t1, ptr null, 1
  %t3 = call ptr @veles_task_new()
  %t4 = call ptr @veles_alloc_words(i64 40)
  %t5 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t4, i32 0, i32 0
  store %str { ptr @.str.13, i64 12 }, ptr %t5
  %t6 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t4, i32 0, i32 1
  store { ptr, ptr } %t2, ptr %t6
  call void @veles_task_start(ptr %t3, ptr @entry.v_main.attempt, ptr %t4)
  br label %await.1
await.1:
  %t7 = call i64 @veles_task_await(ptr %task, ptr %t3)
  %t8 = icmp ne i64 %t7, 0
  br i1 %t8, label %await.got.2, label %await.susp.3
await.susp.3:
  %t9 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t9, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t10 = call i64 @veles_task_cancelled(ptr %task)
  %t11 = icmp ne i64 %t10, 0
  br i1 %t11, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.6:
  br label %await.1
await.got.2:
  %t12 = call i64 @veles_task_panicked(ptr %t3)
  %t13 = icmp ne i64 %t12, 0
  br i1 %t13, label %await.repanic.7, label %await.fine.8
await.repanic.7:
  call void @veles_task_repanic(ptr %t3)
  unreachable
await.fine.8:
  %t14 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2, 0
  %t15 = insertvalue { ptr, ptr } %t14, ptr null, 1
  %t16 = call ptr @veles_task_new()
  %t17 = call ptr @veles_alloc_words(i64 40)
  %t18 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t17, i32 0, i32 0
  store %str { ptr @.str.14, i64 10 }, ptr %t18
  %t19 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t17, i32 0, i32 1
  store { ptr, ptr } %t15, ptr %t19
  call void @veles_task_start(ptr %t16, ptr @entry.v_main.attempt, ptr %t17)
  br label %await.9
await.9:
  %t20 = call i64 @veles_task_await(ptr %task, ptr %t16)
  %t21 = icmp ne i64 %t20, 0
  br i1 %t21, label %await.got.10, label %await.susp.11
await.susp.11:
  %t22 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t22, label %coro.suspend [ i8 0, label %resume.12 i8 1, label %coro.cleanup ]
resume.12:
  %t23 = call i64 @veles_task_cancelled(ptr %task)
  %t24 = icmp ne i64 %t23, 0
  br i1 %t24, label %cancelled.13, label %cont.14
cancelled.13:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.14:
  br label %await.9
await.got.10:
  %t25 = call i64 @veles_task_panicked(ptr %t16)
  %t26 = icmp ne i64 %t25, 0
  br i1 %t26, label %await.repanic.15, label %await.fine.16
await.repanic.15:
  call void @veles_task_repanic(ptr %t16)
  unreachable
await.fine.16:
  %t27 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda3, 0
  %t28 = insertvalue { ptr, ptr } %t27, ptr null, 1
  %t29 = call ptr @veles_task_new()
  %t30 = call ptr @veles_alloc_words(i64 40)
  %t31 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t30, i32 0, i32 0
  store %str { ptr @.str.15, i64 8 }, ptr %t31
  %t32 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t30, i32 0, i32 1
  store { ptr, ptr } %t28, ptr %t32
  call void @veles_task_start(ptr %t29, ptr @entry.v_main.attempt, ptr %t30)
  br label %await.17
await.17:
  %t33 = call i64 @veles_task_await(ptr %task, ptr %t29)
  %t34 = icmp ne i64 %t33, 0
  br i1 %t34, label %await.got.18, label %await.susp.19
await.susp.19:
  %t35 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t35, label %coro.suspend [ i8 0, label %resume.20 i8 1, label %coro.cleanup ]
resume.20:
  %t36 = call i64 @veles_task_cancelled(ptr %task)
  %t37 = icmp ne i64 %t36, 0
  br i1 %t37, label %cancelled.21, label %cont.22
cancelled.21:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.22:
  br label %await.17
await.got.18:
  %t38 = call i64 @veles_task_panicked(ptr %t29)
  %t39 = icmp ne i64 %t38, 0
  br i1 %t39, label %await.repanic.23, label %await.fine.24
await.repanic.23:
  call void @veles_task_repanic(ptr %t29)
  unreachable
await.fine.24:
  %t40 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t41 = insertvalue { ptr, ptr } %t40, ptr null, 1
  %t42 = call ptr @veles_task_new()
  %t43 = call ptr @veles_alloc_words(i64 40)
  %t44 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t43, i32 0, i32 0
  store %str { ptr @.str.16, i64 10 }, ptr %t44
  %t45 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t43, i32 0, i32 1
  store { ptr, ptr } %t41, ptr %t45
  call void @veles_task_start(ptr %t42, ptr @entry.v_main.attempt, ptr %t43)
  br label %await.25
await.25:
  %t46 = call i64 @veles_task_await(ptr %task, ptr %t42)
  %t47 = icmp ne i64 %t46, 0
  br i1 %t47, label %await.got.26, label %await.susp.27
await.susp.27:
  %t48 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t48, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  %t49 = call i64 @veles_task_cancelled(ptr %task)
  %t50 = icmp ne i64 %t49, 0
  br i1 %t50, label %cancelled.29, label %cont.30
cancelled.29:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.30:
  br label %await.25
await.got.26:
  %t51 = call i64 @veles_task_panicked(ptr %t42)
  %t52 = icmp ne i64 %t51, 0
  br i1 %t52, label %await.repanic.31, label %await.fine.32
await.repanic.31:
  call void @veles_task_repanic(ptr %t42)
  unreachable
await.fine.32:
  %t53 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda5, 0
  %t54 = insertvalue { ptr, ptr } %t53, ptr null, 1
  %t55 = call ptr @veles_task_new()
  %t56 = call ptr @veles_alloc_words(i64 40)
  %t57 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t56, i32 0, i32 0
  store %str { ptr @.str.17, i64 8 }, ptr %t57
  %t58 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t56, i32 0, i32 1
  store { ptr, ptr } %t54, ptr %t58
  call void @veles_task_start(ptr %t55, ptr @entry.v_main.attempt, ptr %t56)
  br label %await.33
await.33:
  %t59 = call i64 @veles_task_await(ptr %task, ptr %t55)
  %t60 = icmp ne i64 %t59, 0
  br i1 %t60, label %await.got.34, label %await.susp.35
await.susp.35:
  %t61 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t61, label %coro.suspend [ i8 0, label %resume.36 i8 1, label %coro.cleanup ]
resume.36:
  %t62 = call i64 @veles_task_cancelled(ptr %task)
  %t63 = icmp ne i64 %t62, 0
  br i1 %t63, label %cancelled.37, label %cont.38
cancelled.37:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.38:
  br label %await.33
await.got.34:
  %t64 = call i64 @veles_task_panicked(ptr %t55)
  %t65 = icmp ne i64 %t64, 0
  br i1 %t65, label %await.repanic.39, label %await.fine.40
await.repanic.39:
  call void @veles_task_repanic(ptr %t55)
  unreachable
await.fine.40:
  %t66 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda6, 0
  %t67 = insertvalue { ptr, ptr } %t66, ptr null, 1
  %t68 = call ptr @veles_task_new()
  %t69 = call ptr @veles_alloc_words(i64 40)
  %t70 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t69, i32 0, i32 0
  store %str { ptr @.str.18, i64 15 }, ptr %t70
  %t71 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t69, i32 0, i32 1
  store { ptr, ptr } %t67, ptr %t71
  call void @veles_task_start(ptr %t68, ptr @entry.v_main.attempt, ptr %t69)
  br label %await.41
await.41:
  %t72 = call i64 @veles_task_await(ptr %task, ptr %t68)
  %t73 = icmp ne i64 %t72, 0
  br i1 %t73, label %await.got.42, label %await.susp.43
await.susp.43:
  %t74 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t74, label %coro.suspend [ i8 0, label %resume.44 i8 1, label %coro.cleanup ]
resume.44:
  %t75 = call i64 @veles_task_cancelled(ptr %task)
  %t76 = icmp ne i64 %t75, 0
  br i1 %t76, label %cancelled.45, label %cont.46
cancelled.45:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.46:
  br label %await.41
await.got.42:
  %t77 = call i64 @veles_task_panicked(ptr %t68)
  %t78 = icmp ne i64 %t77, 0
  br i1 %t78, label %await.repanic.47, label %await.fine.48
await.repanic.47:
  call void @veles_task_repanic(ptr %t68)
  unreachable
await.fine.48:
  %t79 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda7, 0
  %t80 = insertvalue { ptr, ptr } %t79, ptr null, 1
  %t81 = call ptr @veles_task_new()
  %t82 = call ptr @veles_alloc_words(i64 40)
  %t83 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t82, i32 0, i32 0
  store %str { ptr @.str.19, i64 18 }, ptr %t83
  %t84 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %t82, i32 0, i32 1
  store { ptr, ptr } %t80, ptr %t84
  call void @veles_task_start(ptr %t81, ptr @entry.v_main.attempt, ptr %t82)
  br label %await.49
await.49:
  %t85 = call i64 @veles_task_await(ptr %task, ptr %t81)
  %t86 = icmp ne i64 %t85, 0
  br i1 %t86, label %await.got.50, label %await.susp.51
await.susp.51:
  %t87 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t87, label %coro.suspend [ i8 0, label %resume.52 i8 1, label %coro.cleanup ]
resume.52:
  %t88 = call i64 @veles_task_cancelled(ptr %task)
  %t89 = icmp ne i64 %t88, 0
  br i1 %t89, label %cancelled.53, label %cont.54
cancelled.53:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.54:
  br label %await.49
await.got.50:
  %t90 = call i64 @veles_task_panicked(ptr %t81)
  %t91 = icmp ne i64 %t90, 0
  br i1 %t91, label %await.repanic.55, label %await.fine.56
await.repanic.55:
  call void @veles_task_repanic(ptr %t81)
  unreachable
await.fine.56:
  %t92 = call i8 @v_main.wrapped(i8 127, i8 1)
  %t93 = sext i8 %t92 to i64
  %t95 = call i64 @veles_i64_format(ptr %a94, i64 %t93)
  %t96 = insertvalue %str undef, ptr %a94, 0
  %t97 = insertvalue %str %t96, i64 %t95, 1
  %t99 = extractvalue %str { ptr @.str.20, i64 16 }, 0
  %t100 = extractvalue %str { ptr @.str.20, i64 16 }, 1
  %t101 = extractvalue %str %t97, 0
  %t102 = extractvalue %str %t97, 1
  call void @veles_string_concat(ptr %a98, ptr %t99, i64 %t100, ptr %t101, i64 %t102)
  %t103 = load %str, ptr %a98
  call void @v_std.io.println(%str %t103)
  %t104 = call i8 @v_main.narrow(i64 300)
  %t105 = zext i8 %t104 to i64
  %t107 = call i64 @veles_u64_format(ptr %a106, i64 %t105)
  %t108 = insertvalue %str undef, ptr %a106, 0
  %t109 = insertvalue %str %t108, i64 %t107, 1
  %t110 = call i8 @v_main.narrow(i64 -1)
  %t111 = zext i8 %t110 to i64
  %t113 = call i64 @veles_u64_format(ptr %a112, i64 %t111)
  %t114 = insertvalue %str undef, ptr %a112, 0
  %t115 = insertvalue %str %t114, i64 %t113, 1
  %t118 = getelementptr [4 x %str], ptr %a117, i64 0, i64 0
  store %str { ptr @.str.21, i64 12 }, ptr %t118
  %t119 = getelementptr [4 x %str], ptr %a117, i64 0, i64 1
  store %str %t109, ptr %t119
  %t120 = getelementptr [4 x %str], ptr %a117, i64 0, i64 2
  store %str { ptr @.str.22, i64 13 }, ptr %t120
  %t121 = getelementptr [4 x %str], ptr %a117, i64 0, i64 3
  store %str %t115, ptr %t121
  call void @veles_string_concat_n(ptr %a116, ptr %a117, i64 4)
  %t122 = load %str, ptr %a116
  call void @v_std.io.println(%str %t122)
  %t123 = call i64 @v_main.sumUpTo(i8 255)
  %t125 = call i64 @veles_i64_format(ptr %a124, i64 %t123)
  %t126 = insertvalue %str undef, ptr %a124, 0
  %t127 = insertvalue %str %t126, i64 %t125, 1
  %t129 = extractvalue %str { ptr @.str.23, i64 23 }, 0
  %t130 = extractvalue %str { ptr @.str.23, i64 23 }, 1
  %t131 = extractvalue %str %t127, 0
  %t132 = extractvalue %str %t127, 1
  call void @veles_string_concat(ptr %a128, ptr %t129, i64 %t130, ptr %t131, i64 %t132)
  %t133 = load %str, ptr %a128
  call void @v_std.io.println(%str %t133)
  call void @v_main.edges(i8 -128, i8 127, i8 255)
  call void @veles_task_finish(ptr %task, ptr null, i64 0, i64 0)
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

define internal ptr @ramp.v_main.call(ptr %task, { ptr, ptr } %p0) presplitcoroutine {
entry:
  %a2 = alloca i64
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = call i64 @v_main.call({ ptr, ptr } %p0)
  store i64 %t1, ptr %a2
  call void @veles_task_finish(ptr %task, ptr %a2, i64 8, i64 0)
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
  %t3 = call ptr @ramp.v_main.call(ptr %task, { ptr, ptr } %t2)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

define internal void @entry.v_main.attempt(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %args, i32 0, i32 0
  %t2 = load %str, ptr %t1
  %t3 = getelementptr inbounds { %str, { ptr, ptr } }, ptr %args, i32 0, i32 1
  %t4 = load { ptr, ptr }, ptr %t3
  %t5 = call ptr @v_main.attempt(ptr %task, %str %t2, { ptr, ptr } %t4)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = call ptr @v_main.main(ptr %task)
  call void @veles_task_started(ptr %task, ptr %t1)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:12:33\00"
@.str.3 = private unnamed_addr constant [26 x i8] c"(250..255).iter() yields \00"
@.str.4 = private unnamed_addr constant [21 x i8] c"(-128..127).len() = \00"
@.str.5 = private unnamed_addr constant [22 x i8] c", (0..<-128).len() = \00"
@.str.6 = private unnamed_addr constant [25 x i8] c"(-128..127).step(100) = \00"
@.str.7 = private unnamed_addr constant [22 x i8] c"reversed then step = \00"
@.str.8 = private unnamed_addr constant [24 x i8] c", step then reversed = \00"
@.str.9 = private unnamed_addr constant [4 x i8] c" = \00"
@.str.10 = private unnamed_addr constant [3 x i8] c": \00"
@.str.11 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.12 = private unnamed_addr constant [13 x i8] c"main.vs:45:3\00"
@.str.13 = private unnamed_addr constant [13 x i8] c"127 + 1 (i8)\00"
@.str.14 = private unnamed_addr constant [11 x i8] c"0 - 1 (u8)\00"
@.str.15 = private unnamed_addr constant [9 x i8] c"MIN * -1\00"
@.str.16 = private unnamed_addr constant [11 x i8] c"-MIN (i32)\00"
@.str.17 = private unnamed_addr constant [9 x i8] c"MIN / -1\00"
@.str.18 = private unnamed_addr constant [16 x i8] c"MIN.abs() (i16)\00"
@.str.19 = private unnamed_addr constant [19 x i8] c"x += 1 at 127 (i8)\00"
@.str.20 = private unnamed_addr constant [17 x i8] c"127 +% 1 (i8) = \00"
@.str.21 = private unnamed_addr constant [13 x i8] c"300 as u8 = \00"
@.str.22 = private unnamed_addr constant [14 x i8] c", -1 as u8 = \00"
@.str.23 = private unnamed_addr constant [24 x i8] c"sum of 250..255 (u8) = \00"
