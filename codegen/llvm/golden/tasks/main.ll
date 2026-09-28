%S.main.Resource = type { %str }
define ptr @v_main.slow(ptr %task, i64 %p1) presplitcoroutine {
entry:
  %a1 = alloca i64
  %a41 = alloca i64
  store i64 %p1, ptr %a1
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t2 = extractvalue %str { ptr @.str.1, i64 20 }, 0
  %t3 = extractvalue %str { ptr @.str.1, i64 20 }, 1
  call void @veles_call_push(ptr %t2)
  %t4 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t5 = extractvalue %S.std.prelude.Duration %t4, 0
  %t7 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t5, i64 999999)
  %t8 = extractvalue { i64, i1 } %t7, 0
  %t9 = extractvalue { i64, i1 } %t7, 1
  br i1 %t9, label %overflow.1, label %arith.ok.2
overflow.1:
  %t10 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t13 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
arith.ok.2:
  %t15 = icmp eq i64 1000000, 0
  br i1 %t15, label %divzero.3, label %div.ok.4
divzero.3:
  %t16 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t17 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t18 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t19 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t16, i64 %t17, ptr %t18, i64 %t19)
  unreachable
div.ok.4:
  %t20 = icmp eq i64 %t8, -9223372036854775808
  %t21 = icmp eq i64 1000000, -1
  %t22 = and i1 %t20, %t21
  br i1 %t22, label %divof.5, label %div.ok.6
divof.5:
  %t23 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t24 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t25 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t26 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t23, i64 %t24, ptr %t25, i64 %t26)
  unreachable
div.ok.6:
  %t14 = sdiv i64 %t8, 1000000
  br label %sleep.7
sleep.7:
  %t27 = call i64 @veles_task_sleep(ptr %task, i64 %t14)
  %t28 = icmp ne i64 %t27, 0
  br i1 %t28, label %sleep.done.8, label %sleep.susp.9
sleep.susp.9:
  %t29 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t29, label %coro.suspend [ i8 0, label %resume.10 i8 1, label %coro.cleanup ]
resume.10:
  %t30 = call i64 @veles_task_cancelled(ptr %task)
  %t31 = icmp ne i64 %t30, 0
  br i1 %t31, label %cancelled.11, label %cont.12
cancelled.11:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.12:
  br label %sleep.7
sleep.done.8:
  %t32 = load i64, ptr %a1
  %t34 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t32, i64 2)
  %t35 = extractvalue { i64, i1 } %t34, 0
  %t36 = extractvalue { i64, i1 } %t34, 1
  br i1 %t36, label %overflow.13, label %arith.ok.14
overflow.13:
  %t37 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t38 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t39 = extractvalue %str { ptr @.str.5, i64 13 }, 0
  %t40 = extractvalue %str { ptr @.str.5, i64 13 }, 1
  call void @veles_panic_at(ptr %t37, i64 %t38, ptr %t39, i64 %t40)
  unreachable
arith.ok.14:
  store i64 %t35, ptr %a41
  call void @veles_task_finish(ptr %task, ptr %a41, i64 8, i64 0)
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

define ptr @v_main.produce(ptr %task, ptr %p1) presplitcoroutine {
entry:
  %a1 = alloca ptr
  %a5 = alloca { i64, i64, i1 }
  %a8 = alloca i64
  %a11 = alloca i64
  %a12 = alloca i1
  %a13 = alloca i1
  %a15 = alloca i1
  %a27 = alloca i64
  %a30 = alloca i64
  %a36 = alloca i1
  store ptr %p1, ptr %a1
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t2 = insertvalue { i64, i64, i1 } undef, i64 1, 0
  %t3 = insertvalue { i64, i64, i1 } %t2, i64 3, 1
  %t4 = insertvalue { i64, i64, i1 } %t3, i1 true, 2
  store { i64, i64, i1 } %t4, ptr %a5
  %t6 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 0
  %t7 = load i64, ptr %t6
  store i64 %t7, ptr %a8
  %t9 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 1
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a11
  store i1 false, ptr %a12
  br label %loop.cond.1
loop.cond.1:
  %t14 = load i1, ptr %a12
  br i1 %t14, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a13
  br label %if.end.6
if.else.7:
  %t16 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t17 = load i1, ptr %t16
  br i1 %t17, label %if.then.8, label %if.else.10
if.then.8:
  %t18 = load i64, ptr %a8
  %t19 = load i64, ptr %a11
  %t20 = icmp sle i64 %t18, %t19
  store i1 %t20, ptr %a15
  br label %if.end.9
if.else.10:
  %t21 = load i64, ptr %a8
  %t22 = load i64, ptr %a11
  %t23 = icmp slt i64 %t21, %t22
  store i1 %t23, ptr %a15
  br label %if.end.9
if.end.9:
  %t24 = load i1, ptr %a15
  store i1 %t24, ptr %a13
  br label %if.end.6
if.end.6:
  %t25 = load i1, ptr %a13
  br i1 %t25, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t26 = load i64, ptr %a8
  store i64 %t26, ptr %a27
  %t28 = load ptr, ptr %a1
  %t29 = load i64, ptr %a27
  store i64 %t29, ptr %a30
  br label %send.11
send.11:
  %t31 = call i64 @veles_chan_send(ptr %task, ptr %t28, ptr %a30)
  %t32 = icmp ne i64 %t31, 0
  br i1 %t32, label %send.done.12, label %send.susp.13
send.susp.13:
  %t33 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t33, label %coro.suspend [ i8 0, label %resume.14 i8 1, label %coro.cleanup ]
resume.14:
  %t34 = call i64 @veles_task_cancelled(ptr %task)
  %t35 = icmp ne i64 %t34, 0
  br i1 %t35, label %cancelled.15, label %cont.16
cancelled.15:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.16:
  br label %send.11
send.done.12:
  br label %loop.post.2
loop.post.2:
  %t37 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t38 = load i1, ptr %t37
  store i1 %t38, ptr %a36
  br i1 %t38, label %sc.rhs.17, label %sc.end.18
sc.rhs.17:
  %t39 = load i64, ptr %a8
  %t40 = load i64, ptr %a11
  %t41 = icmp eq i64 %t39, %t40
  store i1 %t41, ptr %a36
  br label %sc.end.18
sc.end.18:
  %t42 = load i1, ptr %a36
  br i1 %t42, label %if.then.19, label %if.else.21
if.then.19:
  store i1 true, ptr %a12
  br label %if.end.20
if.else.21:
  %t43 = load i64, ptr %a8
  %t44 = add i64 %t43, 1
  store i64 %t44, ptr %a8
  br label %if.end.20
if.end.20:
  %t45 = load volatile i32, ptr @veles_stop_requested, align 4
  %t46 = icmp ne i32 %t45, 0
  br i1 %t46, label %safepoint.22, label %safepoint.on.23, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.22:
  call void @veles_gc_park()
  br label %safepoint.on.23
safepoint.on.23:
  br label %loop.cond.1
loop.end.3:
  %t47 = load ptr, ptr %a1
  call void @veles_chan_close(ptr %t47)
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
  %a2 = alloca %S.main.Resource
  %a3 = alloca { ptr, ptr, ptr }
  %a5 = alloca ptr
  %a6 = alloca { ptr, ptr, ptr }
  %a11 = alloca ptr
  %a12 = alloca ptr
  %a17 = alloca ptr
  %a18 = alloca ptr
  %a38 = alloca [21 x i8]
  %a61 = alloca [21 x i8]
  %a65 = alloca %str
  %a66 = alloca [3 x %str]
  %a93 = alloca ptr
  %a94 = alloca i64
  %a96 = alloca ptr
  %a97 = alloca { ptr, ptr, ptr }
  %a103 = alloca ptr
  %a105 = alloca i64
  %a122 = alloca i64
  %a127 = alloca i64
  %a156 = alloca [21 x i8]
  %a160 = alloca %str
  %a169 = alloca ptr
  %a171 = alloca %str
  %a174 = alloca %str
  %a177 = alloca %str
  %a180 = alloca %str
  %a183 = alloca %str
  %a186 = alloca %str
  %a189 = alloca %str
  %a195 = alloca %str
  %a201 = alloca %str
  %a207 = alloca %str
  %a213 = alloca %str
  %a219 = alloca %str
  %a224 = alloca %str
  %a225 = alloca [11 x %str]
  %a241 = alloca ptr
  %a242 = alloca i64
  %a244 = alloca ptr
  %a245 = alloca { ptr, ptr, ptr }
  %a251 = alloca ptr
  %a293 = alloca i64
  %a310 = alloca i64
  %a341 = alloca [21 x i8]
  %a345 = alloca %str
  %a354 = alloca ptr
  %a355 = alloca i64
  %a357 = alloca ptr
  %a358 = alloca { ptr, ptr, ptr }
  %a364 = alloca ptr
  %a368 = alloca i64
  %a374 = alloca { i1, i64 }
  %a443 = alloca [21 x i8]
  %a447 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = insertvalue %S.main.Resource undef, %str { ptr @.str.6, i64 2 }, 0
  store %S.main.Resource %t1, ptr %a2
  call void @veles_cleanup_push(ptr %a3, ptr @with.close.1, ptr %a2)
  %t4 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t4, ptr %a5
  call void @veles_cleanup_push(ptr %a6, ptr @scope.cancel.thunk, ptr %a5)
  %t7 = load ptr, ptr %a5
  %t8 = call ptr @veles_task_launch(ptr %t7, i64 0)
  %t9 = call ptr @veles_alloc_words(i64 16)
  %t10 = getelementptr inbounds { i64 }, ptr %t9, i32 0, i32 0
  store i64 4, ptr %t10
  call void @veles_task_spawn(ptr %t8, ptr @entry.v_main.slow, ptr %t9)
  store ptr %t8, ptr %a11
  store ptr %t8, ptr %a12
  %t13 = load ptr, ptr %a5
  %t14 = call ptr @veles_task_launch(ptr %t13, i64 1)
  %t15 = call ptr @veles_alloc_words(i64 16)
  %t16 = getelementptr inbounds { i64 }, ptr %t15, i32 0, i32 0
  store i64 5, ptr %t16
  call void @veles_task_spawn(ptr %t14, ptr @entry.v_main.slow, ptr %t15)
  store ptr %t14, ptr %a17
  store ptr %t14, ptr %a18
  %t19 = load ptr, ptr %a12
  br label %await.4
await.4:
  %t20 = call i64 @veles_task_await(ptr %task, ptr %t19)
  %t21 = icmp ne i64 %t20, 0
  br i1 %t21, label %await.got.5, label %await.susp.6
await.susp.6:
  %t22 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t22, label %coro.suspend [ i8 0, label %resume.7 i8 1, label %coro.cleanup ]
resume.7:
  %t23 = call i64 @veles_task_cancelled(ptr %task)
  %t24 = icmp ne i64 %t23, 0
  br i1 %t24, label %cancelled.8, label %cont.9
cancelled.8:
  call void @veles_cleanup_pop(ptr %a6)
  %t25 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t25)
  br label %abandon.wait.10
abandon.wait.10:
  %t26 = call i64 @veles_scope_wait(ptr %task, ptr %t25)
  %t27 = icmp ne i64 %t26, 0
  br i1 %t27, label %abandon.done.11, label %abandon.susp.12
abandon.susp.12:
  %t28 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t28, label %coro.suspend [ i8 0, label %resume.13 i8 1, label %coro.cleanup ]
resume.13:
  br label %abandon.wait.10
abandon.done.11:
  call void @veles_cleanup_pop(ptr %a3)
  %t29 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t30 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t29)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.9:
  %t31 = load ptr, ptr %a5
  %t32 = call ptr @veles_scope_failed(ptr %t31)
  %t33 = icmp ne ptr %t32, null
  br i1 %t33, label %scope.abort.14, label %scope.ok.15
scope.ok.15:
  br label %cont.16
scope.abort.14:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.1
cont.16:
  br label %await.4
await.got.5:
  %t34 = call i64 @veles_task_panicked(ptr %t19)
  %t35 = icmp ne i64 %t34, 0
  br i1 %t35, label %await.repanic.17, label %await.fine.18
await.repanic.17:
  call void @veles_task_repanic(ptr %t19)
  unreachable
await.fine.18:
  %t36 = call ptr @veles_task_result(ptr %t19)
  %t37 = load i64, ptr %t36
  %t39 = call i64 @veles_i64_format(ptr %a38, i64 %t37)
  %t40 = insertvalue %str undef, ptr %a38, 0
  %t41 = insertvalue %str %t40, i64 %t39, 1
  %t42 = load ptr, ptr %a18
  br label %await.19
await.19:
  %t43 = call i64 @veles_task_await(ptr %task, ptr %t42)
  %t44 = icmp ne i64 %t43, 0
  br i1 %t44, label %await.got.20, label %await.susp.21
await.susp.21:
  %t45 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t45, label %coro.suspend [ i8 0, label %resume.22 i8 1, label %coro.cleanup ]
resume.22:
  %t46 = call i64 @veles_task_cancelled(ptr %task)
  %t47 = icmp ne i64 %t46, 0
  br i1 %t47, label %cancelled.23, label %cont.24
cancelled.23:
  call void @veles_cleanup_pop(ptr %a6)
  %t48 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t48)
  br label %abandon.wait.25
abandon.wait.25:
  %t49 = call i64 @veles_scope_wait(ptr %task, ptr %t48)
  %t50 = icmp ne i64 %t49, 0
  br i1 %t50, label %abandon.done.26, label %abandon.susp.27
abandon.susp.27:
  %t51 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t51, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  br label %abandon.wait.25
abandon.done.26:
  call void @veles_cleanup_pop(ptr %a3)
  %t52 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t53 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t52)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.24:
  %t54 = load ptr, ptr %a5
  %t55 = call ptr @veles_scope_failed(ptr %t54)
  %t56 = icmp ne ptr %t55, null
  br i1 %t56, label %scope.abort.29, label %scope.ok.30
scope.ok.30:
  br label %cont.31
scope.abort.29:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.1
cont.31:
  br label %await.19
await.got.20:
  %t57 = call i64 @veles_task_panicked(ptr %t42)
  %t58 = icmp ne i64 %t57, 0
  br i1 %t58, label %await.repanic.32, label %await.fine.33
await.repanic.32:
  call void @veles_task_repanic(ptr %t42)
  unreachable
await.fine.33:
  %t59 = call ptr @veles_task_result(ptr %t42)
  %t60 = load i64, ptr %t59
  %t62 = call i64 @veles_i64_format(ptr %a61, i64 %t60)
  %t63 = insertvalue %str undef, ptr %a61, 0
  %t64 = insertvalue %str %t63, i64 %t62, 1
  %t67 = getelementptr [3 x %str], ptr %a66, i64 0, i64 0
  store %str %t41, ptr %t67
  %t68 = getelementptr [3 x %str], ptr %a66, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t68
  %t69 = getelementptr [3 x %str], ptr %a66, i64 0, i64 2
  store %str %t64, ptr %t69
  call void @veles_string_concat_n(ptr %a65, ptr %a66, i64 3)
  %t70 = load %str, ptr %a65
  %t71 = extractvalue %str { ptr @.str.9, i64 20 }, 0
  %t72 = extractvalue %str { ptr @.str.9, i64 20 }, 1
  call void @veles_call_push(ptr %t71)
  call void @v_std.io.println(%str %t70)
  call void @veles_call_pop()
  br label %scope.wait.1
scope.wait.1:
  %t73 = load ptr, ptr %a5
  %t74 = call i64 @veles_scope_wait(ptr %task, ptr %t73)
  %t75 = icmp ne i64 %t74, 0
  br i1 %t75, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  %t76 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t76, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t77 = call i64 @veles_task_cancelled(ptr %task)
  %t78 = icmp ne i64 %t77, 0
  br i1 %t78, label %cancelled.35, label %cont.36
cancelled.35:
  call void @veles_cleanup_pop(ptr %a6)
  %t79 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t79)
  br label %abandon.wait.37
abandon.wait.37:
  %t80 = call i64 @veles_scope_wait(ptr %task, ptr %t79)
  %t81 = icmp ne i64 %t80, 0
  br i1 %t81, label %abandon.done.38, label %abandon.susp.39
abandon.susp.39:
  %t82 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t82, label %coro.suspend [ i8 0, label %resume.40 i8 1, label %coro.cleanup ]
resume.40:
  br label %abandon.wait.37
abandon.done.38:
  call void @veles_cleanup_pop(ptr %a3)
  %t83 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t84 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t83)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.36:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop(ptr %a6)
  %t85 = load ptr, ptr %a5
  %t86 = call ptr @veles_scope_failed(ptr %t85)
  %t87 = icmp ne ptr %t86, null
  br i1 %t87, label %scope.check.41, label %scope.after.42
scope.check.41:
  %t88 = call i64 @veles_task_panicked(ptr %t86)
  %t89 = icmp ne i64 %t88, 0
  br i1 %t89, label %scope.repanic.43, label %scope.errors.44
scope.repanic.43:
  call void @veles_task_repanic(ptr %t86)
  unreachable
scope.errors.44:
  br label %scope.after.42
scope.after.42:
  call void @veles_cleanup_pop(ptr %a3)
  %t90 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t91 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t90)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  %t92 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t92, ptr %a93
  store i64 0, ptr %a94
  %t95 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t95, ptr %a96
  call void @veles_cleanup_push(ptr %a97, ptr @scope.cancel.thunk, ptr %a96)
  %t98 = load ptr, ptr %a96
  %t99 = call ptr @veles_task_launch(ptr %t98, i64 0)
  %t100 = load ptr, ptr %a93
  %t101 = call ptr @veles_alloc_words(i64 16)
  %t102 = getelementptr inbounds { ptr }, ptr %t101, i32 0, i32 0
  store ptr %t100, ptr %t102
  call void @veles_task_spawn(ptr %t99, ptr @entry.v_main.produce, ptr %t101)
  store ptr %t99, ptr %a103
  br label %loop.cond.48
loop.cond.48:
  br label %loop.body.51
loop.body.51:
  %t104 = load ptr, ptr %a93
  store i64 zeroinitializer, ptr %a105
  br label %recv.52
recv.52:
  %t106 = call i64 @veles_chan_recv(ptr %task, ptr %t104, ptr %a105)
  %t107 = icmp ne i64 %t106, 0
  br i1 %t107, label %recv.done.53, label %recv.susp.54
recv.susp.54:
  %t108 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t108, label %coro.suspend [ i8 0, label %resume.55 i8 1, label %coro.cleanup ]
resume.55:
  %t109 = call i64 @veles_task_cancelled(ptr %task)
  %t110 = icmp ne i64 %t109, 0
  br i1 %t110, label %cancelled.56, label %cont.57
cancelled.56:
  call void @veles_cleanup_pop(ptr %a97)
  %t111 = load ptr, ptr %a96
  call void @veles_scope_cancel(ptr %t111)
  br label %abandon.wait.58
abandon.wait.58:
  %t112 = call i64 @veles_scope_wait(ptr %task, ptr %t111)
  %t113 = icmp ne i64 %t112, 0
  br i1 %t113, label %abandon.done.59, label %abandon.susp.60
abandon.susp.60:
  %t114 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t114, label %coro.suspend [ i8 0, label %resume.61 i8 1, label %coro.cleanup ]
resume.61:
  br label %abandon.wait.58
abandon.done.59:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.57:
  %t115 = load ptr, ptr %a96
  %t116 = call ptr @veles_scope_failed(ptr %t115)
  %t117 = icmp ne ptr %t116, null
  br i1 %t117, label %scope.abort.62, label %scope.ok.63
scope.ok.63:
  br label %cont.64
scope.abort.62:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.45
cont.64:
  br label %recv.52
recv.done.53:
  %t118 = icmp eq i64 %t106, 1
  %t119 = load i64, ptr %a105
  %t120 = insertvalue { i1, i64 } undef, i1 %t118, 0
  %t121 = insertvalue { i1, i64 } %t120, i64 %t119, 1
  %t124 = extractvalue { i1, i64 } %t121, 0
  %t123 = xor i1 %t124, true
  br i1 %t123, label %elvis.default.65, label %elvis.some.66
elvis.some.66:
  %t125 = extractvalue { i1, i64 } %t121, 1
  store i64 %t125, ptr %a122
  br label %elvis.end.67
elvis.default.65:
  br label %loop.end.50
elvis.end.67:
  %t126 = load i64, ptr %a122
  store i64 %t126, ptr %a127
  %t128 = load i64, ptr %a94
  %t129 = load i64, ptr %a127
  %t131 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t128, i64 %t129)
  %t132 = extractvalue { i64, i1 } %t131, 0
  %t133 = extractvalue { i64, i1 } %t131, 1
  br i1 %t133, label %overflow.68, label %arith.ok.69
overflow.68:
  %t134 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t135 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t136 = extractvalue %str { ptr @.str.10, i64 12 }, 0
  %t137 = extractvalue %str { ptr @.str.10, i64 12 }, 1
  call void @veles_panic_at(ptr %t134, i64 %t135, ptr %t136, i64 %t137)
  unreachable
arith.ok.69:
  store i64 %t132, ptr %a94
  br label %loop.post.49
loop.post.49:
  %t138 = load volatile i32, ptr @veles_stop_requested, align 4
  %t139 = icmp ne i32 %t138, 0
  br i1 %t139, label %safepoint.70, label %safepoint.on.71, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.70:
  call void @veles_gc_park()
  br label %safepoint.on.71
safepoint.on.71:
  br label %loop.cond.48
loop.end.50:
  br label %scope.wait.45
scope.wait.45:
  %t140 = load ptr, ptr %a96
  %t141 = call i64 @veles_scope_wait(ptr %task, ptr %t140)
  %t142 = icmp ne i64 %t141, 0
  br i1 %t142, label %scope.done.46, label %scope.susp.47
scope.susp.47:
  %t143 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t143, label %coro.suspend [ i8 0, label %resume.72 i8 1, label %coro.cleanup ]
resume.72:
  %t144 = call i64 @veles_task_cancelled(ptr %task)
  %t145 = icmp ne i64 %t144, 0
  br i1 %t145, label %cancelled.73, label %cont.74
cancelled.73:
  call void @veles_cleanup_pop(ptr %a97)
  %t146 = load ptr, ptr %a96
  call void @veles_scope_cancel(ptr %t146)
  br label %abandon.wait.75
abandon.wait.75:
  %t147 = call i64 @veles_scope_wait(ptr %task, ptr %t146)
  %t148 = icmp ne i64 %t147, 0
  br i1 %t148, label %abandon.done.76, label %abandon.susp.77
abandon.susp.77:
  %t149 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t149, label %coro.suspend [ i8 0, label %resume.78 i8 1, label %coro.cleanup ]
resume.78:
  br label %abandon.wait.75
abandon.done.76:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.74:
  br label %scope.wait.45
scope.done.46:
  call void @veles_cleanup_pop(ptr %a97)
  %t150 = load ptr, ptr %a96
  %t151 = call ptr @veles_scope_failed(ptr %t150)
  %t152 = icmp ne ptr %t151, null
  br i1 %t152, label %scope.check.79, label %scope.after.80
scope.check.79:
  %t153 = call i64 @veles_task_panicked(ptr %t151)
  %t154 = icmp ne i64 %t153, 0
  br i1 %t154, label %scope.repanic.81, label %scope.errors.82
scope.repanic.81:
  call void @veles_task_repanic(ptr %t151)
  unreachable
scope.errors.82:
  br label %scope.after.80
scope.after.80:
  %t155 = load i64, ptr %a94
  %t157 = call i64 @veles_i64_format(ptr %a156, i64 %t155)
  %t158 = insertvalue %str undef, ptr %a156, 0
  %t159 = insertvalue %str %t158, i64 %t157, 1
  %t161 = extractvalue %str { ptr @.str.11, i64 4 }, 0
  %t162 = extractvalue %str { ptr @.str.11, i64 4 }, 1
  %t163 = extractvalue %str %t159, 0
  %t164 = extractvalue %str %t159, 1
  call void @veles_string_concat(ptr %a160, ptr %t161, i64 %t162, ptr %t163, i64 %t164)
  %t165 = load %str, ptr %a160
  %t166 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t167 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t166)
  call void @v_std.io.println(%str %t165)
  call void @veles_call_pop()
  %t168 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t168, ptr %a169
  %t170 = load ptr, ptr %a169
  store %str { ptr @.str.13, i64 1 }, ptr %a171
  %t172 = call i64 @veles_chan_try_send(ptr %t170, ptr %a171)
  %t173 = icmp ne i64 %t172, 0
  call void @veles_bool_to_string(ptr %a174, i1 zeroext %t173)
  %t175 = load %str, ptr %a174
  %t176 = load ptr, ptr %a169
  store %str { ptr @.str.14, i64 1 }, ptr %a177
  %t178 = call i64 @veles_chan_try_send(ptr %t176, ptr %a177)
  %t179 = icmp ne i64 %t178, 0
  call void @veles_bool_to_string(ptr %a180, i1 zeroext %t179)
  %t181 = load %str, ptr %a180
  %t182 = load ptr, ptr %a169
  store %str { ptr @.str.15, i64 1 }, ptr %a183
  %t184 = call i64 @veles_chan_try_send(ptr %t182, ptr %a183)
  %t185 = icmp ne i64 %t184, 0
  call void @veles_bool_to_string(ptr %a186, i1 zeroext %t185)
  %t187 = load %str, ptr %a186
  %t188 = load ptr, ptr %a169
  store %str zeroinitializer, ptr %a189
  %t190 = call i64 @veles_chan_try_recv(ptr %t188, ptr %a189)
  %t191 = icmp ne i64 %t190, 0
  %t192 = load %str, ptr %a189
  %t193 = insertvalue { i1, %str } undef, i1 %t191, 0
  %t194 = insertvalue { i1, %str } %t193, %str %t192, 1
  %t197 = extractvalue { i1, %str } %t194, 0
  %t196 = xor i1 %t197, true
  br i1 %t196, label %elvis.default.83, label %elvis.some.84
elvis.some.84:
  %t198 = extractvalue { i1, %str } %t194, 1
  store %str %t198, ptr %a195
  br label %elvis.end.85
elvis.default.83:
  store %str { ptr @.str.16, i64 1 }, ptr %a195
  br label %elvis.end.85
elvis.end.85:
  %t199 = load %str, ptr %a195
  %t200 = load ptr, ptr %a169
  store %str zeroinitializer, ptr %a201
  %t202 = call i64 @veles_chan_try_recv(ptr %t200, ptr %a201)
  %t203 = icmp ne i64 %t202, 0
  %t204 = load %str, ptr %a201
  %t205 = insertvalue { i1, %str } undef, i1 %t203, 0
  %t206 = insertvalue { i1, %str } %t205, %str %t204, 1
  %t209 = extractvalue { i1, %str } %t206, 0
  %t208 = xor i1 %t209, true
  br i1 %t208, label %elvis.default.86, label %elvis.some.87
elvis.some.87:
  %t210 = extractvalue { i1, %str } %t206, 1
  store %str %t210, ptr %a207
  br label %elvis.end.88
elvis.default.86:
  store %str { ptr @.str.16, i64 1 }, ptr %a207
  br label %elvis.end.88
elvis.end.88:
  %t211 = load %str, ptr %a207
  %t212 = load ptr, ptr %a169
  store %str zeroinitializer, ptr %a213
  %t214 = call i64 @veles_chan_try_recv(ptr %t212, ptr %a213)
  %t215 = icmp ne i64 %t214, 0
  %t216 = load %str, ptr %a213
  %t217 = insertvalue { i1, %str } undef, i1 %t215, 0
  %t218 = insertvalue { i1, %str } %t217, %str %t216, 1
  %t221 = extractvalue { i1, %str } %t218, 0
  %t220 = xor i1 %t221, true
  br i1 %t220, label %elvis.default.89, label %elvis.some.90
elvis.some.90:
  %t222 = extractvalue { i1, %str } %t218, 1
  store %str %t222, ptr %a219
  br label %elvis.end.91
elvis.default.89:
  store %str { ptr @.str.16, i64 1 }, ptr %a219
  br label %elvis.end.91
elvis.end.91:
  %t223 = load %str, ptr %a219
  %t226 = getelementptr [11 x %str], ptr %a225, i64 0, i64 0
  store %str %t175, ptr %t226
  %t227 = getelementptr [11 x %str], ptr %a225, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t227
  %t228 = getelementptr [11 x %str], ptr %a225, i64 0, i64 2
  store %str %t181, ptr %t228
  %t229 = getelementptr [11 x %str], ptr %a225, i64 0, i64 3
  store %str { ptr @.str.8, i64 1 }, ptr %t229
  %t230 = getelementptr [11 x %str], ptr %a225, i64 0, i64 4
  store %str %t187, ptr %t230
  %t231 = getelementptr [11 x %str], ptr %a225, i64 0, i64 5
  store %str { ptr @.str.8, i64 1 }, ptr %t231
  %t232 = getelementptr [11 x %str], ptr %a225, i64 0, i64 6
  store %str %t199, ptr %t232
  %t233 = getelementptr [11 x %str], ptr %a225, i64 0, i64 7
  store %str { ptr @.str.8, i64 1 }, ptr %t233
  %t234 = getelementptr [11 x %str], ptr %a225, i64 0, i64 8
  store %str %t211, ptr %t234
  %t235 = getelementptr [11 x %str], ptr %a225, i64 0, i64 9
  store %str { ptr @.str.8, i64 1 }, ptr %t235
  %t236 = getelementptr [11 x %str], ptr %a225, i64 0, i64 10
  store %str %t223, ptr %t236
  call void @veles_string_concat_n(ptr %a224, ptr %a225, i64 11)
  %t237 = load %str, ptr %a224
  %t238 = extractvalue %str { ptr @.str.17, i64 20 }, 0
  %t239 = extractvalue %str { ptr @.str.17, i64 20 }, 1
  call void @veles_call_push(ptr %t238)
  call void @v_std.io.println(%str %t237)
  call void @veles_call_pop()
  %t240 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t240, ptr %a241
  store i64 0, ptr %a242
  %t243 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t243, ptr %a244
  call void @veles_cleanup_push(ptr %a245, ptr @scope.cancel.thunk, ptr %a244)
  %t246 = load ptr, ptr %a244
  %t247 = call ptr @veles_task_launch(ptr %t246, i64 0)
  %t248 = load ptr, ptr %a241
  %t249 = call ptr @veles_alloc_words(i64 16)
  %t250 = getelementptr inbounds { ptr }, ptr %t249, i32 0, i32 0
  store ptr %t248, ptr %t250
  call void @veles_task_spawn(ptr %t247, ptr @entry.v_main.produce, ptr %t249)
  store ptr %t247, ptr %a251
  br label %loop.cond.95
loop.cond.95:
  %t252 = load i64, ptr %a242
  %t253 = icmp slt i64 %t252, 6
  br i1 %t253, label %loop.body.98, label %loop.end.97
loop.body.98:
  %t254 = extractvalue %str { ptr @.str.18, i64 20 }, 0
  %t255 = extractvalue %str { ptr @.str.18, i64 20 }, 1
  call void @veles_call_push(ptr %t254)
  %t256 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t257 = extractvalue %S.std.prelude.Duration %t256, 0
  %t259 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t257, i64 999999)
  %t260 = extractvalue { i64, i1 } %t259, 0
  %t261 = extractvalue { i64, i1 } %t259, 1
  br i1 %t261, label %overflow.99, label %arith.ok.100
overflow.99:
  %t262 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t263 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t264 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t265 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t262, i64 %t263, ptr %t264, i64 %t265)
  unreachable
arith.ok.100:
  %t267 = icmp eq i64 1000000, 0
  br i1 %t267, label %divzero.101, label %div.ok.102
divzero.101:
  %t268 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t269 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t270 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t271 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t268, i64 %t269, ptr %t270, i64 %t271)
  unreachable
div.ok.102:
  %t272 = icmp eq i64 %t260, -9223372036854775808
  %t273 = icmp eq i64 1000000, -1
  %t274 = and i1 %t272, %t273
  br i1 %t274, label %divof.103, label %div.ok.104
divof.103:
  %t275 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t276 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t277 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t278 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t275, i64 %t276, ptr %t277, i64 %t278)
  unreachable
div.ok.104:
  %t266 = sdiv i64 %t260, 1000000
  br label %sleep.105
sleep.105:
  %t279 = call i64 @veles_task_sleep(ptr %task, i64 %t266)
  %t280 = icmp ne i64 %t279, 0
  br i1 %t280, label %sleep.done.106, label %sleep.susp.107
sleep.susp.107:
  %t281 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t281, label %coro.suspend [ i8 0, label %resume.108 i8 1, label %coro.cleanup ]
resume.108:
  %t282 = call i64 @veles_task_cancelled(ptr %task)
  %t283 = icmp ne i64 %t282, 0
  br i1 %t283, label %cancelled.109, label %cont.110
cancelled.109:
  call void @veles_cleanup_pop(ptr %a245)
  %t284 = load ptr, ptr %a244
  call void @veles_scope_cancel(ptr %t284)
  br label %abandon.wait.111
abandon.wait.111:
  %t285 = call i64 @veles_scope_wait(ptr %task, ptr %t284)
  %t286 = icmp ne i64 %t285, 0
  br i1 %t286, label %abandon.done.112, label %abandon.susp.113
abandon.susp.113:
  %t287 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t287, label %coro.suspend [ i8 0, label %resume.114 i8 1, label %coro.cleanup ]
resume.114:
  br label %abandon.wait.111
abandon.done.112:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.110:
  %t288 = load ptr, ptr %a244
  %t289 = call ptr @veles_scope_failed(ptr %t288)
  %t290 = icmp ne ptr %t289, null
  br i1 %t290, label %scope.abort.115, label %scope.ok.116
scope.ok.116:
  br label %cont.117
scope.abort.115:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.92
cont.117:
  br label %sleep.105
sleep.done.106:
  %t291 = load i64, ptr %a242
  %t292 = load ptr, ptr %a241
  store i64 zeroinitializer, ptr %a293
  br label %recv.118
recv.118:
  %t294 = call i64 @veles_chan_recv(ptr %task, ptr %t292, ptr %a293)
  %t295 = icmp ne i64 %t294, 0
  br i1 %t295, label %recv.done.119, label %recv.susp.120
recv.susp.120:
  %t296 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t296, label %coro.suspend [ i8 0, label %resume.121 i8 1, label %coro.cleanup ]
resume.121:
  %t297 = call i64 @veles_task_cancelled(ptr %task)
  %t298 = icmp ne i64 %t297, 0
  br i1 %t298, label %cancelled.122, label %cont.123
cancelled.122:
  call void @veles_cleanup_pop(ptr %a245)
  %t299 = load ptr, ptr %a244
  call void @veles_scope_cancel(ptr %t299)
  br label %abandon.wait.124
abandon.wait.124:
  %t300 = call i64 @veles_scope_wait(ptr %task, ptr %t299)
  %t301 = icmp ne i64 %t300, 0
  br i1 %t301, label %abandon.done.125, label %abandon.susp.126
abandon.susp.126:
  %t302 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t302, label %coro.suspend [ i8 0, label %resume.127 i8 1, label %coro.cleanup ]
resume.127:
  br label %abandon.wait.124
abandon.done.125:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.123:
  %t303 = load ptr, ptr %a244
  %t304 = call ptr @veles_scope_failed(ptr %t303)
  %t305 = icmp ne ptr %t304, null
  br i1 %t305, label %scope.abort.128, label %scope.ok.129
scope.ok.129:
  br label %cont.130
scope.abort.128:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.92
cont.130:
  br label %recv.118
recv.done.119:
  %t306 = icmp eq i64 %t294, 1
  %t307 = load i64, ptr %a293
  %t308 = insertvalue { i1, i64 } undef, i1 %t306, 0
  %t309 = insertvalue { i1, i64 } %t308, i64 %t307, 1
  %t312 = extractvalue { i1, i64 } %t309, 0
  %t311 = xor i1 %t312, true
  br i1 %t311, label %elvis.default.131, label %elvis.some.132
elvis.some.132:
  %t313 = extractvalue { i1, i64 } %t309, 1
  store i64 %t313, ptr %a310
  br label %elvis.end.133
elvis.default.131:
  store i64 0, ptr %a310
  br label %elvis.end.133
elvis.end.133:
  %t314 = load i64, ptr %a310
  %t316 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t291, i64 %t314)
  %t317 = extractvalue { i64, i1 } %t316, 0
  %t318 = extractvalue { i64, i1 } %t316, 1
  br i1 %t318, label %overflow.134, label %arith.ok.135
overflow.134:
  %t319 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t320 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t321 = extractvalue %str { ptr @.str.20, i64 12 }, 0
  %t322 = extractvalue %str { ptr @.str.20, i64 12 }, 1
  call void @veles_panic_at(ptr %t319, i64 %t320, ptr %t321, i64 %t322)
  unreachable
arith.ok.135:
  store i64 %t317, ptr %a242
  br label %loop.post.96
loop.post.96:
  %t323 = load volatile i32, ptr @veles_stop_requested, align 4
  %t324 = icmp ne i32 %t323, 0
  br i1 %t324, label %safepoint.136, label %safepoint.on.137, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.136:
  call void @veles_gc_park()
  br label %safepoint.on.137
safepoint.on.137:
  br label %loop.cond.95
loop.end.97:
  br label %scope.wait.92
scope.wait.92:
  %t325 = load ptr, ptr %a244
  %t326 = call i64 @veles_scope_wait(ptr %task, ptr %t325)
  %t327 = icmp ne i64 %t326, 0
  br i1 %t327, label %scope.done.93, label %scope.susp.94
scope.susp.94:
  %t328 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t328, label %coro.suspend [ i8 0, label %resume.138 i8 1, label %coro.cleanup ]
resume.138:
  %t329 = call i64 @veles_task_cancelled(ptr %task)
  %t330 = icmp ne i64 %t329, 0
  br i1 %t330, label %cancelled.139, label %cont.140
cancelled.139:
  call void @veles_cleanup_pop(ptr %a245)
  %t331 = load ptr, ptr %a244
  call void @veles_scope_cancel(ptr %t331)
  br label %abandon.wait.141
abandon.wait.141:
  %t332 = call i64 @veles_scope_wait(ptr %task, ptr %t331)
  %t333 = icmp ne i64 %t332, 0
  br i1 %t333, label %abandon.done.142, label %abandon.susp.143
abandon.susp.143:
  %t334 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t334, label %coro.suspend [ i8 0, label %resume.144 i8 1, label %coro.cleanup ]
resume.144:
  br label %abandon.wait.141
abandon.done.142:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.140:
  br label %scope.wait.92
scope.done.93:
  call void @veles_cleanup_pop(ptr %a245)
  %t335 = load ptr, ptr %a244
  %t336 = call ptr @veles_scope_failed(ptr %t335)
  %t337 = icmp ne ptr %t336, null
  br i1 %t337, label %scope.check.145, label %scope.after.146
scope.check.145:
  %t338 = call i64 @veles_task_panicked(ptr %t336)
  %t339 = icmp ne i64 %t338, 0
  br i1 %t339, label %scope.repanic.147, label %scope.errors.148
scope.repanic.147:
  call void @veles_task_repanic(ptr %t336)
  unreachable
scope.errors.148:
  br label %scope.after.146
scope.after.146:
  %t340 = load i64, ptr %a242
  %t342 = call i64 @veles_i64_format(ptr %a341, i64 %t340)
  %t343 = insertvalue %str undef, ptr %a341, 0
  %t344 = insertvalue %str %t343, i64 %t342, 1
  %t346 = extractvalue %str { ptr @.str.21, i64 6 }, 0
  %t347 = extractvalue %str { ptr @.str.21, i64 6 }, 1
  %t348 = extractvalue %str %t344, 0
  %t349 = extractvalue %str %t344, 1
  call void @veles_string_concat(ptr %a345, ptr %t346, i64 %t347, ptr %t348, i64 %t349)
  %t350 = load %str, ptr %a345
  %t351 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t352 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t351)
  call void @v_std.io.println(%str %t350)
  call void @veles_call_pop()
  %t353 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t353, ptr %a354
  store i64 0, ptr %a355
  %t356 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t356, ptr %a357
  call void @veles_cleanup_push(ptr %a358, ptr @scope.cancel.thunk, ptr %a357)
  %t359 = load ptr, ptr %a357
  %t360 = call ptr @veles_task_launch(ptr %t359, i64 0)
  %t361 = load ptr, ptr %a354
  %t362 = call ptr @veles_alloc_words(i64 16)
  %t363 = getelementptr inbounds { ptr }, ptr %t362, i32 0, i32 0
  store ptr %t361, ptr %t363
  call void @veles_task_spawn(ptr %t360, ptr @entry.v_main.produce, ptr %t362)
  store ptr %t360, ptr %a364
  br label %loop.cond.152
loop.cond.152:
  %t365 = load i64, ptr %a355
  %t366 = icmp slt i64 %t365, 6
  br i1 %t366, label %loop.body.155, label %loop.end.154
loop.body.155:
  %t367 = load ptr, ptr %a354
  store i64 zeroinitializer, ptr %a368
  %t369 = call i64 @veles_chan_try_recv(ptr %t367, ptr %a368)
  %t370 = icmp ne i64 %t369, 0
  %t371 = load i64, ptr %a368
  %t372 = insertvalue { i1, i64 } undef, i1 %t370, 0
  %t373 = insertvalue { i1, i64 } %t372, i64 %t371, 1
  store { i1, i64 } %t373, ptr %a374
  %t375 = load { i1, i64 }, ptr %a374
  %t377 = extractvalue { i1, i64 } %t375, 0
  %t376 = xor i1 %t377, true
  %t378 = xor i1 %t376, true
  br i1 %t378, label %if.then.156, label %if.else.158
if.then.156:
  %t379 = load i64, ptr %a355
  %t380 = load { i1, i64 }, ptr %a374
  %t381 = extractvalue { i1, i64 } %t380, 1
  %t383 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t379, i64 %t381)
  %t384 = extractvalue { i64, i1 } %t383, 0
  %t385 = extractvalue { i64, i1 } %t383, 1
  br i1 %t385, label %overflow.159, label %arith.ok.160
overflow.159:
  %t386 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t387 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t388 = extractvalue %str { ptr @.str.23, i64 13 }, 0
  %t389 = extractvalue %str { ptr @.str.23, i64 13 }, 1
  call void @veles_panic_at(ptr %t386, i64 %t387, ptr %t388, i64 %t389)
  unreachable
arith.ok.160:
  store i64 %t384, ptr %a355
  br label %if.end.157
if.else.158:
  %t390 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t391 = load i64, ptr %t390
  %t393 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t391, i64 999999)
  %t394 = extractvalue { i64, i1 } %t393, 0
  %t395 = extractvalue { i64, i1 } %t393, 1
  br i1 %t395, label %overflow.161, label %arith.ok.162
overflow.161:
  %t396 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t397 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t398 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t399 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t396, i64 %t397, ptr %t398, i64 %t399)
  unreachable
arith.ok.162:
  %t401 = icmp eq i64 1000000, 0
  br i1 %t401, label %divzero.163, label %div.ok.164
divzero.163:
  %t402 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t403 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t404 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t405 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t402, i64 %t403, ptr %t404, i64 %t405)
  unreachable
div.ok.164:
  %t406 = icmp eq i64 %t394, -9223372036854775808
  %t407 = icmp eq i64 1000000, -1
  %t408 = and i1 %t406, %t407
  br i1 %t408, label %divof.165, label %div.ok.166
divof.165:
  %t409 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t410 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t411 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t412 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t409, i64 %t410, ptr %t411, i64 %t412)
  unreachable
div.ok.166:
  %t400 = sdiv i64 %t394, 1000000
  br label %sleep.167
sleep.167:
  %t413 = call i64 @veles_task_sleep(ptr %task, i64 %t400)
  %t414 = icmp ne i64 %t413, 0
  br i1 %t414, label %sleep.done.168, label %sleep.susp.169
sleep.susp.169:
  %t415 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t415, label %coro.suspend [ i8 0, label %resume.170 i8 1, label %coro.cleanup ]
resume.170:
  %t416 = call i64 @veles_task_cancelled(ptr %task)
  %t417 = icmp ne i64 %t416, 0
  br i1 %t417, label %cancelled.171, label %cont.172
cancelled.171:
  call void @veles_cleanup_pop(ptr %a358)
  %t418 = load ptr, ptr %a357
  call void @veles_scope_cancel(ptr %t418)
  br label %abandon.wait.173
abandon.wait.173:
  %t419 = call i64 @veles_scope_wait(ptr %task, ptr %t418)
  %t420 = icmp ne i64 %t419, 0
  br i1 %t420, label %abandon.done.174, label %abandon.susp.175
abandon.susp.175:
  %t421 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t421, label %coro.suspend [ i8 0, label %resume.176 i8 1, label %coro.cleanup ]
resume.176:
  br label %abandon.wait.173
abandon.done.174:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.172:
  %t422 = load ptr, ptr %a357
  %t423 = call ptr @veles_scope_failed(ptr %t422)
  %t424 = icmp ne ptr %t423, null
  br i1 %t424, label %scope.abort.177, label %scope.ok.178
scope.ok.178:
  br label %cont.179
scope.abort.177:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.149
cont.179:
  br label %sleep.167
sleep.done.168:
  br label %if.end.157
if.end.157:
  br label %loop.post.153
loop.post.153:
  %t425 = load volatile i32, ptr @veles_stop_requested, align 4
  %t426 = icmp ne i32 %t425, 0
  br i1 %t426, label %safepoint.180, label %safepoint.on.181, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.180:
  call void @veles_gc_park()
  br label %safepoint.on.181
safepoint.on.181:
  br label %loop.cond.152
loop.end.154:
  br label %scope.wait.149
scope.wait.149:
  %t427 = load ptr, ptr %a357
  %t428 = call i64 @veles_scope_wait(ptr %task, ptr %t427)
  %t429 = icmp ne i64 %t428, 0
  br i1 %t429, label %scope.done.150, label %scope.susp.151
scope.susp.151:
  %t430 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t430, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  %t431 = call i64 @veles_task_cancelled(ptr %task)
  %t432 = icmp ne i64 %t431, 0
  br i1 %t432, label %cancelled.183, label %cont.184
cancelled.183:
  call void @veles_cleanup_pop(ptr %a358)
  %t433 = load ptr, ptr %a357
  call void @veles_scope_cancel(ptr %t433)
  br label %abandon.wait.185
abandon.wait.185:
  %t434 = call i64 @veles_scope_wait(ptr %task, ptr %t433)
  %t435 = icmp ne i64 %t434, 0
  br i1 %t435, label %abandon.done.186, label %abandon.susp.187
abandon.susp.187:
  %t436 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t436, label %coro.suspend [ i8 0, label %resume.188 i8 1, label %coro.cleanup ]
resume.188:
  br label %abandon.wait.185
abandon.done.186:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.184:
  br label %scope.wait.149
scope.done.150:
  call void @veles_cleanup_pop(ptr %a358)
  %t437 = load ptr, ptr %a357
  %t438 = call ptr @veles_scope_failed(ptr %t437)
  %t439 = icmp ne ptr %t438, null
  br i1 %t439, label %scope.check.189, label %scope.after.190
scope.check.189:
  %t440 = call i64 @veles_task_panicked(ptr %t438)
  %t441 = icmp ne i64 %t440, 0
  br i1 %t441, label %scope.repanic.191, label %scope.errors.192
scope.repanic.191:
  call void @veles_task_repanic(ptr %t438)
  unreachable
scope.errors.192:
  br label %scope.after.190
scope.after.190:
  %t442 = load i64, ptr %a355
  %t444 = call i64 @veles_i64_format(ptr %a443, i64 %t442)
  %t445 = insertvalue %str undef, ptr %a443, 0
  %t446 = insertvalue %str %t445, i64 %t444, 1
  %t448 = extractvalue %str { ptr @.str.25, i64 7 }, 0
  %t449 = extractvalue %str { ptr @.str.25, i64 7 }, 1
  %t450 = extractvalue %str %t446, 0
  %t451 = extractvalue %str %t446, 1
  call void @veles_string_concat(ptr %a447, ptr %t448, i64 %t449, ptr %t450, i64 %t451)
  %t452 = load %str, ptr %a447
  %t453 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t454 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t453)
  call void @v_std.io.println(%str %t452)
  call void @veles_call_pop()
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

define void @v_main.Closeable.Resource.close(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a5 = alloca %str
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds %S.main.Resource, ptr %t2, i32 0, i32 0
  %t4 = load %str, ptr %t3
  %t6 = extractvalue %str { ptr @.str.27, i64 7 }, 0
  %t7 = extractvalue %str { ptr @.str.27, i64 7 }, 1
  %t8 = extractvalue %str %t4, 0
  %t9 = extractvalue %str %t4, 1
  call void @veles_string_concat(ptr %a5, ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  %t10 = load %str, ptr %a5
  %t11 = extractvalue %str { ptr @.str.28, i64 20 }, 0
  %t12 = extractvalue %str { ptr @.str.28, i64 20 }, 1
  call void @veles_call_push(ptr %t11)
  call void @v_std.io.println(%str %t10)
  call void @veles_call_pop()
  ret void
}

define internal void @entry.v_main.slow(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { i64 }, ptr %args, i32 0, i32 0
  %t2 = load i64, ptr %t1
  %t3 = extractvalue %str { ptr @.str.29, i64 5 }, 0
  %t4 = extractvalue %str { ptr @.str.29, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t3)
  %t5 = call ptr @v_main.slow(ptr %task, i64 %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.produce(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { ptr }, ptr %args, i32 0, i32 0
  %t2 = load ptr, ptr %t1
  %t3 = extractvalue %str { ptr @.str.30, i64 8 }, 0
  %t4 = extractvalue %str { ptr @.str.30, i64 8 }, 1
  call void @veles_call_base(ptr %task, ptr %t3)
  %t5 = call ptr @v_main.produce(ptr %task, ptr %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.31, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.31, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

@.str.1 = private unnamed_addr constant [21 x i8] c"main.vs:12:15\00millis\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:12:9\00"
@.str.4 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.5 = private unnamed_addr constant [14 x i8] c"main.vs:13:10\00"
@.str.6 = private unnamed_addr constant [3 x i8] c"db\00"
@.str.7 = private unnamed_addr constant [19 x i8] c"main.vs:22:9\00close\00"
@.str.8 = private unnamed_addr constant [2 x i8] c" \00"
@.str.9 = private unnamed_addr constant [21 x i8] c"main.vs:26:7\00println\00"
@.str.10 = private unnamed_addr constant [13 x i8] c"main.vs:35:7\00"
@.str.11 = private unnamed_addr constant [5 x i8] c"sum \00"
@.str.12 = private unnamed_addr constant [21 x i8] c"main.vs:38:3\00println\00"
@.str.13 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.14 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.15 = private unnamed_addr constant [2 x i8] c"c\00"
@.str.16 = private unnamed_addr constant [2 x i8] c"-\00"
@.str.17 = private unnamed_addr constant [21 x i8] c"main.vs:42:3\00println\00"
@.str.18 = private unnamed_addr constant [21 x i8] c"main.vs:51:19\00millis\00"
@.str.19 = private unnamed_addr constant [14 x i8] c"main.vs:51:13\00"
@.str.20 = private unnamed_addr constant [13 x i8] c"main.vs:52:7\00"
@.str.21 = private unnamed_addr constant [7 x i8] c"slept \00"
@.str.22 = private unnamed_addr constant [21 x i8] c"main.vs:55:3\00println\00"
@.str.23 = private unnamed_addr constant [14 x i8] c"main.vs:64:22\00"
@.str.24 = private unnamed_addr constant [14 x i8] c"main.vs:64:44\00"
@.str.25 = private unnamed_addr constant [8 x i8] c"polled \00"
@.str.26 = private unnamed_addr constant [21 x i8] c"main.vs:67:3\00println\00"
@.str.27 = private unnamed_addr constant [8 x i8] c"closed \00"
@.str.28 = private unnamed_addr constant [21 x i8] c"main.vs:8:39\00println\00"
@.str.29 = private unnamed_addr constant [6 x i8] c"\00slow\00"
@.str.30 = private unnamed_addr constant [9 x i8] c"\00produce\00"
@.str.31 = private unnamed_addr constant [6 x i8] c"\00main\00"
