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
  %a39 = alloca [21 x i8]
  %a63 = alloca [21 x i8]
  %a67 = alloca %str
  %a68 = alloca [3 x %str]
  %a95 = alloca ptr
  %a96 = alloca i64
  %a98 = alloca ptr
  %a99 = alloca { ptr, ptr, ptr }
  %a105 = alloca ptr
  %a107 = alloca i64
  %a125 = alloca i64
  %a130 = alloca i64
  %a159 = alloca [21 x i8]
  %a163 = alloca %str
  %a172 = alloca ptr
  %a174 = alloca %str
  %a177 = alloca %str
  %a180 = alloca %str
  %a183 = alloca %str
  %a186 = alloca %str
  %a189 = alloca %str
  %a192 = alloca %str
  %a198 = alloca %str
  %a204 = alloca %str
  %a210 = alloca %str
  %a216 = alloca %str
  %a222 = alloca %str
  %a227 = alloca %str
  %a228 = alloca [11 x %str]
  %a244 = alloca ptr
  %a245 = alloca i64
  %a247 = alloca ptr
  %a248 = alloca { ptr, ptr, ptr }
  %a254 = alloca ptr
  %a297 = alloca i64
  %a315 = alloca i64
  %a346 = alloca [21 x i8]
  %a350 = alloca %str
  %a359 = alloca ptr
  %a360 = alloca i64
  %a362 = alloca ptr
  %a363 = alloca { ptr, ptr, ptr }
  %a369 = alloca ptr
  %a373 = alloca i64
  %a379 = alloca { i1, i64 }
  %a449 = alloca [21 x i8]
  %a453 = alloca %str
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
  %t34 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t34)
  br label %scope.wait.1
cont.16:
  br label %await.4
await.got.5:
  %t35 = call i64 @veles_task_panicked(ptr %t19)
  %t36 = icmp ne i64 %t35, 0
  br i1 %t36, label %await.repanic.17, label %await.fine.18
await.repanic.17:
  call void @veles_task_repanic(ptr %t19)
  unreachable
await.fine.18:
  %t37 = call ptr @veles_task_result(ptr %t19)
  %t38 = load i64, ptr %t37
  %t40 = call i64 @veles_i64_format(ptr %a39, i64 %t38)
  %t41 = insertvalue %str undef, ptr %a39, 0
  %t42 = insertvalue %str %t41, i64 %t40, 1
  %t43 = load ptr, ptr %a18
  br label %await.19
await.19:
  %t44 = call i64 @veles_task_await(ptr %task, ptr %t43)
  %t45 = icmp ne i64 %t44, 0
  br i1 %t45, label %await.got.20, label %await.susp.21
await.susp.21:
  %t46 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t46, label %coro.suspend [ i8 0, label %resume.22 i8 1, label %coro.cleanup ]
resume.22:
  %t47 = call i64 @veles_task_cancelled(ptr %task)
  %t48 = icmp ne i64 %t47, 0
  br i1 %t48, label %cancelled.23, label %cont.24
cancelled.23:
  call void @veles_cleanup_pop(ptr %a6)
  %t49 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t49)
  br label %abandon.wait.25
abandon.wait.25:
  %t50 = call i64 @veles_scope_wait(ptr %task, ptr %t49)
  %t51 = icmp ne i64 %t50, 0
  br i1 %t51, label %abandon.done.26, label %abandon.susp.27
abandon.susp.27:
  %t52 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t52, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  br label %abandon.wait.25
abandon.done.26:
  call void @veles_cleanup_pop(ptr %a3)
  %t53 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t54 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t53)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.24:
  %t55 = load ptr, ptr %a5
  %t56 = call ptr @veles_scope_failed(ptr %t55)
  %t57 = icmp ne ptr %t56, null
  br i1 %t57, label %scope.abort.29, label %scope.ok.30
scope.ok.30:
  br label %cont.31
scope.abort.29:
  call void @veles_task_leave_waits(ptr %task)
  %t58 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t58)
  br label %scope.wait.1
cont.31:
  br label %await.19
await.got.20:
  %t59 = call i64 @veles_task_panicked(ptr %t43)
  %t60 = icmp ne i64 %t59, 0
  br i1 %t60, label %await.repanic.32, label %await.fine.33
await.repanic.32:
  call void @veles_task_repanic(ptr %t43)
  unreachable
await.fine.33:
  %t61 = call ptr @veles_task_result(ptr %t43)
  %t62 = load i64, ptr %t61
  %t64 = call i64 @veles_i64_format(ptr %a63, i64 %t62)
  %t65 = insertvalue %str undef, ptr %a63, 0
  %t66 = insertvalue %str %t65, i64 %t64, 1
  %t69 = getelementptr [3 x %str], ptr %a68, i64 0, i64 0
  store %str %t42, ptr %t69
  %t70 = getelementptr [3 x %str], ptr %a68, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t70
  %t71 = getelementptr [3 x %str], ptr %a68, i64 0, i64 2
  store %str %t66, ptr %t71
  call void @veles_string_concat_n(ptr %a67, ptr %a68, i64 3)
  %t72 = load %str, ptr %a67
  %t73 = extractvalue %str { ptr @.str.9, i64 20 }, 0
  %t74 = extractvalue %str { ptr @.str.9, i64 20 }, 1
  call void @veles_call_push(ptr %t73)
  call void @v_std.io.println(%str %t72)
  call void @veles_call_pop()
  br label %scope.wait.1
scope.wait.1:
  %t75 = load ptr, ptr %a5
  %t76 = call i64 @veles_scope_wait(ptr %task, ptr %t75)
  %t77 = icmp ne i64 %t76, 0
  br i1 %t77, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  %t78 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t78, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t79 = call i64 @veles_task_cancelled(ptr %task)
  %t80 = icmp ne i64 %t79, 0
  br i1 %t80, label %cancelled.35, label %cont.36
cancelled.35:
  call void @veles_cleanup_pop(ptr %a6)
  %t81 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t81)
  br label %abandon.wait.37
abandon.wait.37:
  %t82 = call i64 @veles_scope_wait(ptr %task, ptr %t81)
  %t83 = icmp ne i64 %t82, 0
  br i1 %t83, label %abandon.done.38, label %abandon.susp.39
abandon.susp.39:
  %t84 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t84, label %coro.suspend [ i8 0, label %resume.40 i8 1, label %coro.cleanup ]
resume.40:
  br label %abandon.wait.37
abandon.done.38:
  call void @veles_cleanup_pop(ptr %a3)
  %t85 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t86 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t85)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.36:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop(ptr %a6)
  %t87 = load ptr, ptr %a5
  %t88 = call ptr @veles_scope_failed(ptr %t87)
  %t89 = icmp ne ptr %t88, null
  br i1 %t89, label %scope.check.41, label %scope.after.42
scope.check.41:
  %t90 = call i64 @veles_task_panicked(ptr %t88)
  %t91 = icmp ne i64 %t90, 0
  br i1 %t91, label %scope.repanic.43, label %scope.errors.44
scope.repanic.43:
  call void @veles_task_repanic(ptr %t88)
  unreachable
scope.errors.44:
  br label %scope.after.42
scope.after.42:
  call void @veles_cleanup_pop(ptr %a3)
  %t92 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t93 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t92)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  %t94 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t94, ptr %a95
  store i64 0, ptr %a96
  %t97 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t97, ptr %a98
  call void @veles_cleanup_push(ptr %a99, ptr @scope.cancel.thunk, ptr %a98)
  %t100 = load ptr, ptr %a98
  %t101 = call ptr @veles_task_launch(ptr %t100, i64 0)
  %t102 = load ptr, ptr %a95
  %t103 = call ptr @veles_alloc_words(i64 16)
  %t104 = getelementptr inbounds { ptr }, ptr %t103, i32 0, i32 0
  store ptr %t102, ptr %t104
  call void @veles_task_spawn(ptr %t101, ptr @entry.v_main.produce, ptr %t103)
  store ptr %t101, ptr %a105
  br label %loop.cond.48
loop.cond.48:
  br label %loop.body.51
loop.body.51:
  %t106 = load ptr, ptr %a95
  store i64 zeroinitializer, ptr %a107
  br label %recv.52
recv.52:
  %t108 = call i64 @veles_chan_recv(ptr %task, ptr %t106, ptr %a107)
  %t109 = icmp ne i64 %t108, 0
  br i1 %t109, label %recv.done.53, label %recv.susp.54
recv.susp.54:
  %t110 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t110, label %coro.suspend [ i8 0, label %resume.55 i8 1, label %coro.cleanup ]
resume.55:
  %t111 = call i64 @veles_task_cancelled(ptr %task)
  %t112 = icmp ne i64 %t111, 0
  br i1 %t112, label %cancelled.56, label %cont.57
cancelled.56:
  call void @veles_cleanup_pop(ptr %a99)
  %t113 = load ptr, ptr %a98
  call void @veles_scope_cancel(ptr %t113)
  br label %abandon.wait.58
abandon.wait.58:
  %t114 = call i64 @veles_scope_wait(ptr %task, ptr %t113)
  %t115 = icmp ne i64 %t114, 0
  br i1 %t115, label %abandon.done.59, label %abandon.susp.60
abandon.susp.60:
  %t116 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t116, label %coro.suspend [ i8 0, label %resume.61 i8 1, label %coro.cleanup ]
resume.61:
  br label %abandon.wait.58
abandon.done.59:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.57:
  %t117 = load ptr, ptr %a98
  %t118 = call ptr @veles_scope_failed(ptr %t117)
  %t119 = icmp ne ptr %t118, null
  br i1 %t119, label %scope.abort.62, label %scope.ok.63
scope.ok.63:
  br label %cont.64
scope.abort.62:
  call void @veles_task_leave_waits(ptr %task)
  %t120 = load ptr, ptr %a98
  call void @veles_scope_cancel(ptr %t120)
  br label %scope.wait.45
cont.64:
  br label %recv.52
recv.done.53:
  %t121 = icmp eq i64 %t108, 1
  %t122 = load i64, ptr %a107
  %t123 = insertvalue { i1, i64 } undef, i1 %t121, 0
  %t124 = insertvalue { i1, i64 } %t123, i64 %t122, 1
  %t127 = extractvalue { i1, i64 } %t124, 0
  %t126 = xor i1 %t127, true
  br i1 %t126, label %elvis.default.65, label %elvis.some.66
elvis.some.66:
  %t128 = extractvalue { i1, i64 } %t124, 1
  store i64 %t128, ptr %a125
  br label %elvis.end.67
elvis.default.65:
  br label %loop.end.50
elvis.end.67:
  %t129 = load i64, ptr %a125
  store i64 %t129, ptr %a130
  %t131 = load i64, ptr %a96
  %t132 = load i64, ptr %a130
  %t134 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t131, i64 %t132)
  %t135 = extractvalue { i64, i1 } %t134, 0
  %t136 = extractvalue { i64, i1 } %t134, 1
  br i1 %t136, label %overflow.68, label %arith.ok.69
overflow.68:
  %t137 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t138 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t139 = extractvalue %str { ptr @.str.10, i64 12 }, 0
  %t140 = extractvalue %str { ptr @.str.10, i64 12 }, 1
  call void @veles_panic_at(ptr %t137, i64 %t138, ptr %t139, i64 %t140)
  unreachable
arith.ok.69:
  store i64 %t135, ptr %a96
  br label %loop.post.49
loop.post.49:
  %t141 = load volatile i32, ptr @veles_stop_requested, align 4
  %t142 = icmp ne i32 %t141, 0
  br i1 %t142, label %safepoint.70, label %safepoint.on.71, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.70:
  call void @veles_gc_park()
  br label %safepoint.on.71
safepoint.on.71:
  br label %loop.cond.48
loop.end.50:
  br label %scope.wait.45
scope.wait.45:
  %t143 = load ptr, ptr %a98
  %t144 = call i64 @veles_scope_wait(ptr %task, ptr %t143)
  %t145 = icmp ne i64 %t144, 0
  br i1 %t145, label %scope.done.46, label %scope.susp.47
scope.susp.47:
  %t146 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t146, label %coro.suspend [ i8 0, label %resume.72 i8 1, label %coro.cleanup ]
resume.72:
  %t147 = call i64 @veles_task_cancelled(ptr %task)
  %t148 = icmp ne i64 %t147, 0
  br i1 %t148, label %cancelled.73, label %cont.74
cancelled.73:
  call void @veles_cleanup_pop(ptr %a99)
  %t149 = load ptr, ptr %a98
  call void @veles_scope_cancel(ptr %t149)
  br label %abandon.wait.75
abandon.wait.75:
  %t150 = call i64 @veles_scope_wait(ptr %task, ptr %t149)
  %t151 = icmp ne i64 %t150, 0
  br i1 %t151, label %abandon.done.76, label %abandon.susp.77
abandon.susp.77:
  %t152 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t152, label %coro.suspend [ i8 0, label %resume.78 i8 1, label %coro.cleanup ]
resume.78:
  br label %abandon.wait.75
abandon.done.76:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.74:
  br label %scope.wait.45
scope.done.46:
  call void @veles_cleanup_pop(ptr %a99)
  %t153 = load ptr, ptr %a98
  %t154 = call ptr @veles_scope_failed(ptr %t153)
  %t155 = icmp ne ptr %t154, null
  br i1 %t155, label %scope.check.79, label %scope.after.80
scope.check.79:
  %t156 = call i64 @veles_task_panicked(ptr %t154)
  %t157 = icmp ne i64 %t156, 0
  br i1 %t157, label %scope.repanic.81, label %scope.errors.82
scope.repanic.81:
  call void @veles_task_repanic(ptr %t154)
  unreachable
scope.errors.82:
  br label %scope.after.80
scope.after.80:
  %t158 = load i64, ptr %a96
  %t160 = call i64 @veles_i64_format(ptr %a159, i64 %t158)
  %t161 = insertvalue %str undef, ptr %a159, 0
  %t162 = insertvalue %str %t161, i64 %t160, 1
  %t164 = extractvalue %str { ptr @.str.11, i64 4 }, 0
  %t165 = extractvalue %str { ptr @.str.11, i64 4 }, 1
  %t166 = extractvalue %str %t162, 0
  %t167 = extractvalue %str %t162, 1
  call void @veles_string_concat(ptr %a163, ptr %t164, i64 %t165, ptr %t166, i64 %t167)
  %t168 = load %str, ptr %a163
  %t169 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t170 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t169)
  call void @v_std.io.println(%str %t168)
  call void @veles_call_pop()
  %t171 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t171, ptr %a172
  %t173 = load ptr, ptr %a172
  store %str { ptr @.str.13, i64 1 }, ptr %a174
  %t175 = call i64 @veles_chan_try_send(ptr %t173, ptr %a174)
  %t176 = icmp ne i64 %t175, 0
  call void @veles_bool_to_string(ptr %a177, i1 zeroext %t176)
  %t178 = load %str, ptr %a177
  %t179 = load ptr, ptr %a172
  store %str { ptr @.str.14, i64 1 }, ptr %a180
  %t181 = call i64 @veles_chan_try_send(ptr %t179, ptr %a180)
  %t182 = icmp ne i64 %t181, 0
  call void @veles_bool_to_string(ptr %a183, i1 zeroext %t182)
  %t184 = load %str, ptr %a183
  %t185 = load ptr, ptr %a172
  store %str { ptr @.str.15, i64 1 }, ptr %a186
  %t187 = call i64 @veles_chan_try_send(ptr %t185, ptr %a186)
  %t188 = icmp ne i64 %t187, 0
  call void @veles_bool_to_string(ptr %a189, i1 zeroext %t188)
  %t190 = load %str, ptr %a189
  %t191 = load ptr, ptr %a172
  store %str zeroinitializer, ptr %a192
  %t193 = call i64 @veles_chan_try_recv(ptr %t191, ptr %a192)
  %t194 = icmp ne i64 %t193, 0
  %t195 = load %str, ptr %a192
  %t196 = insertvalue { i1, %str } undef, i1 %t194, 0
  %t197 = insertvalue { i1, %str } %t196, %str %t195, 1
  %t200 = extractvalue { i1, %str } %t197, 0
  %t199 = xor i1 %t200, true
  br i1 %t199, label %elvis.default.83, label %elvis.some.84
elvis.some.84:
  %t201 = extractvalue { i1, %str } %t197, 1
  store %str %t201, ptr %a198
  br label %elvis.end.85
elvis.default.83:
  store %str { ptr @.str.16, i64 1 }, ptr %a198
  br label %elvis.end.85
elvis.end.85:
  %t202 = load %str, ptr %a198
  %t203 = load ptr, ptr %a172
  store %str zeroinitializer, ptr %a204
  %t205 = call i64 @veles_chan_try_recv(ptr %t203, ptr %a204)
  %t206 = icmp ne i64 %t205, 0
  %t207 = load %str, ptr %a204
  %t208 = insertvalue { i1, %str } undef, i1 %t206, 0
  %t209 = insertvalue { i1, %str } %t208, %str %t207, 1
  %t212 = extractvalue { i1, %str } %t209, 0
  %t211 = xor i1 %t212, true
  br i1 %t211, label %elvis.default.86, label %elvis.some.87
elvis.some.87:
  %t213 = extractvalue { i1, %str } %t209, 1
  store %str %t213, ptr %a210
  br label %elvis.end.88
elvis.default.86:
  store %str { ptr @.str.16, i64 1 }, ptr %a210
  br label %elvis.end.88
elvis.end.88:
  %t214 = load %str, ptr %a210
  %t215 = load ptr, ptr %a172
  store %str zeroinitializer, ptr %a216
  %t217 = call i64 @veles_chan_try_recv(ptr %t215, ptr %a216)
  %t218 = icmp ne i64 %t217, 0
  %t219 = load %str, ptr %a216
  %t220 = insertvalue { i1, %str } undef, i1 %t218, 0
  %t221 = insertvalue { i1, %str } %t220, %str %t219, 1
  %t224 = extractvalue { i1, %str } %t221, 0
  %t223 = xor i1 %t224, true
  br i1 %t223, label %elvis.default.89, label %elvis.some.90
elvis.some.90:
  %t225 = extractvalue { i1, %str } %t221, 1
  store %str %t225, ptr %a222
  br label %elvis.end.91
elvis.default.89:
  store %str { ptr @.str.16, i64 1 }, ptr %a222
  br label %elvis.end.91
elvis.end.91:
  %t226 = load %str, ptr %a222
  %t229 = getelementptr [11 x %str], ptr %a228, i64 0, i64 0
  store %str %t178, ptr %t229
  %t230 = getelementptr [11 x %str], ptr %a228, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t230
  %t231 = getelementptr [11 x %str], ptr %a228, i64 0, i64 2
  store %str %t184, ptr %t231
  %t232 = getelementptr [11 x %str], ptr %a228, i64 0, i64 3
  store %str { ptr @.str.8, i64 1 }, ptr %t232
  %t233 = getelementptr [11 x %str], ptr %a228, i64 0, i64 4
  store %str %t190, ptr %t233
  %t234 = getelementptr [11 x %str], ptr %a228, i64 0, i64 5
  store %str { ptr @.str.8, i64 1 }, ptr %t234
  %t235 = getelementptr [11 x %str], ptr %a228, i64 0, i64 6
  store %str %t202, ptr %t235
  %t236 = getelementptr [11 x %str], ptr %a228, i64 0, i64 7
  store %str { ptr @.str.8, i64 1 }, ptr %t236
  %t237 = getelementptr [11 x %str], ptr %a228, i64 0, i64 8
  store %str %t214, ptr %t237
  %t238 = getelementptr [11 x %str], ptr %a228, i64 0, i64 9
  store %str { ptr @.str.8, i64 1 }, ptr %t238
  %t239 = getelementptr [11 x %str], ptr %a228, i64 0, i64 10
  store %str %t226, ptr %t239
  call void @veles_string_concat_n(ptr %a227, ptr %a228, i64 11)
  %t240 = load %str, ptr %a227
  %t241 = extractvalue %str { ptr @.str.17, i64 20 }, 0
  %t242 = extractvalue %str { ptr @.str.17, i64 20 }, 1
  call void @veles_call_push(ptr %t241)
  call void @v_std.io.println(%str %t240)
  call void @veles_call_pop()
  %t243 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t243, ptr %a244
  store i64 0, ptr %a245
  %t246 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t246, ptr %a247
  call void @veles_cleanup_push(ptr %a248, ptr @scope.cancel.thunk, ptr %a247)
  %t249 = load ptr, ptr %a247
  %t250 = call ptr @veles_task_launch(ptr %t249, i64 0)
  %t251 = load ptr, ptr %a244
  %t252 = call ptr @veles_alloc_words(i64 16)
  %t253 = getelementptr inbounds { ptr }, ptr %t252, i32 0, i32 0
  store ptr %t251, ptr %t253
  call void @veles_task_spawn(ptr %t250, ptr @entry.v_main.produce, ptr %t252)
  store ptr %t250, ptr %a254
  br label %loop.cond.95
loop.cond.95:
  %t255 = load i64, ptr %a245
  %t256 = icmp slt i64 %t255, 6
  br i1 %t256, label %loop.body.98, label %loop.end.97
loop.body.98:
  %t257 = extractvalue %str { ptr @.str.18, i64 20 }, 0
  %t258 = extractvalue %str { ptr @.str.18, i64 20 }, 1
  call void @veles_call_push(ptr %t257)
  %t259 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t260 = extractvalue %S.std.prelude.Duration %t259, 0
  %t262 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t260, i64 999999)
  %t263 = extractvalue { i64, i1 } %t262, 0
  %t264 = extractvalue { i64, i1 } %t262, 1
  br i1 %t264, label %overflow.99, label %arith.ok.100
overflow.99:
  %t265 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t266 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t267 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t268 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t265, i64 %t266, ptr %t267, i64 %t268)
  unreachable
arith.ok.100:
  %t270 = icmp eq i64 1000000, 0
  br i1 %t270, label %divzero.101, label %div.ok.102
divzero.101:
  %t271 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t272 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t273 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t274 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t271, i64 %t272, ptr %t273, i64 %t274)
  unreachable
div.ok.102:
  %t275 = icmp eq i64 %t263, -9223372036854775808
  %t276 = icmp eq i64 1000000, -1
  %t277 = and i1 %t275, %t276
  br i1 %t277, label %divof.103, label %div.ok.104
divof.103:
  %t278 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t279 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t280 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t281 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t278, i64 %t279, ptr %t280, i64 %t281)
  unreachable
div.ok.104:
  %t269 = sdiv i64 %t263, 1000000
  br label %sleep.105
sleep.105:
  %t282 = call i64 @veles_task_sleep(ptr %task, i64 %t269)
  %t283 = icmp ne i64 %t282, 0
  br i1 %t283, label %sleep.done.106, label %sleep.susp.107
sleep.susp.107:
  %t284 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t284, label %coro.suspend [ i8 0, label %resume.108 i8 1, label %coro.cleanup ]
resume.108:
  %t285 = call i64 @veles_task_cancelled(ptr %task)
  %t286 = icmp ne i64 %t285, 0
  br i1 %t286, label %cancelled.109, label %cont.110
cancelled.109:
  call void @veles_cleanup_pop(ptr %a248)
  %t287 = load ptr, ptr %a247
  call void @veles_scope_cancel(ptr %t287)
  br label %abandon.wait.111
abandon.wait.111:
  %t288 = call i64 @veles_scope_wait(ptr %task, ptr %t287)
  %t289 = icmp ne i64 %t288, 0
  br i1 %t289, label %abandon.done.112, label %abandon.susp.113
abandon.susp.113:
  %t290 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t290, label %coro.suspend [ i8 0, label %resume.114 i8 1, label %coro.cleanup ]
resume.114:
  br label %abandon.wait.111
abandon.done.112:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.110:
  %t291 = load ptr, ptr %a247
  %t292 = call ptr @veles_scope_failed(ptr %t291)
  %t293 = icmp ne ptr %t292, null
  br i1 %t293, label %scope.abort.115, label %scope.ok.116
scope.ok.116:
  br label %cont.117
scope.abort.115:
  call void @veles_task_leave_waits(ptr %task)
  %t294 = load ptr, ptr %a247
  call void @veles_scope_cancel(ptr %t294)
  br label %scope.wait.92
cont.117:
  br label %sleep.105
sleep.done.106:
  %t295 = load i64, ptr %a245
  %t296 = load ptr, ptr %a244
  store i64 zeroinitializer, ptr %a297
  br label %recv.118
recv.118:
  %t298 = call i64 @veles_chan_recv(ptr %task, ptr %t296, ptr %a297)
  %t299 = icmp ne i64 %t298, 0
  br i1 %t299, label %recv.done.119, label %recv.susp.120
recv.susp.120:
  %t300 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t300, label %coro.suspend [ i8 0, label %resume.121 i8 1, label %coro.cleanup ]
resume.121:
  %t301 = call i64 @veles_task_cancelled(ptr %task)
  %t302 = icmp ne i64 %t301, 0
  br i1 %t302, label %cancelled.122, label %cont.123
cancelled.122:
  call void @veles_cleanup_pop(ptr %a248)
  %t303 = load ptr, ptr %a247
  call void @veles_scope_cancel(ptr %t303)
  br label %abandon.wait.124
abandon.wait.124:
  %t304 = call i64 @veles_scope_wait(ptr %task, ptr %t303)
  %t305 = icmp ne i64 %t304, 0
  br i1 %t305, label %abandon.done.125, label %abandon.susp.126
abandon.susp.126:
  %t306 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t306, label %coro.suspend [ i8 0, label %resume.127 i8 1, label %coro.cleanup ]
resume.127:
  br label %abandon.wait.124
abandon.done.125:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.123:
  %t307 = load ptr, ptr %a247
  %t308 = call ptr @veles_scope_failed(ptr %t307)
  %t309 = icmp ne ptr %t308, null
  br i1 %t309, label %scope.abort.128, label %scope.ok.129
scope.ok.129:
  br label %cont.130
scope.abort.128:
  call void @veles_task_leave_waits(ptr %task)
  %t310 = load ptr, ptr %a247
  call void @veles_scope_cancel(ptr %t310)
  br label %scope.wait.92
cont.130:
  br label %recv.118
recv.done.119:
  %t311 = icmp eq i64 %t298, 1
  %t312 = load i64, ptr %a297
  %t313 = insertvalue { i1, i64 } undef, i1 %t311, 0
  %t314 = insertvalue { i1, i64 } %t313, i64 %t312, 1
  %t317 = extractvalue { i1, i64 } %t314, 0
  %t316 = xor i1 %t317, true
  br i1 %t316, label %elvis.default.131, label %elvis.some.132
elvis.some.132:
  %t318 = extractvalue { i1, i64 } %t314, 1
  store i64 %t318, ptr %a315
  br label %elvis.end.133
elvis.default.131:
  store i64 0, ptr %a315
  br label %elvis.end.133
elvis.end.133:
  %t319 = load i64, ptr %a315
  %t321 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t295, i64 %t319)
  %t322 = extractvalue { i64, i1 } %t321, 0
  %t323 = extractvalue { i64, i1 } %t321, 1
  br i1 %t323, label %overflow.134, label %arith.ok.135
overflow.134:
  %t324 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t325 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t326 = extractvalue %str { ptr @.str.20, i64 12 }, 0
  %t327 = extractvalue %str { ptr @.str.20, i64 12 }, 1
  call void @veles_panic_at(ptr %t324, i64 %t325, ptr %t326, i64 %t327)
  unreachable
arith.ok.135:
  store i64 %t322, ptr %a245
  br label %loop.post.96
loop.post.96:
  %t328 = load volatile i32, ptr @veles_stop_requested, align 4
  %t329 = icmp ne i32 %t328, 0
  br i1 %t329, label %safepoint.136, label %safepoint.on.137, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.136:
  call void @veles_gc_park()
  br label %safepoint.on.137
safepoint.on.137:
  br label %loop.cond.95
loop.end.97:
  br label %scope.wait.92
scope.wait.92:
  %t330 = load ptr, ptr %a247
  %t331 = call i64 @veles_scope_wait(ptr %task, ptr %t330)
  %t332 = icmp ne i64 %t331, 0
  br i1 %t332, label %scope.done.93, label %scope.susp.94
scope.susp.94:
  %t333 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t333, label %coro.suspend [ i8 0, label %resume.138 i8 1, label %coro.cleanup ]
resume.138:
  %t334 = call i64 @veles_task_cancelled(ptr %task)
  %t335 = icmp ne i64 %t334, 0
  br i1 %t335, label %cancelled.139, label %cont.140
cancelled.139:
  call void @veles_cleanup_pop(ptr %a248)
  %t336 = load ptr, ptr %a247
  call void @veles_scope_cancel(ptr %t336)
  br label %abandon.wait.141
abandon.wait.141:
  %t337 = call i64 @veles_scope_wait(ptr %task, ptr %t336)
  %t338 = icmp ne i64 %t337, 0
  br i1 %t338, label %abandon.done.142, label %abandon.susp.143
abandon.susp.143:
  %t339 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t339, label %coro.suspend [ i8 0, label %resume.144 i8 1, label %coro.cleanup ]
resume.144:
  br label %abandon.wait.141
abandon.done.142:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.140:
  br label %scope.wait.92
scope.done.93:
  call void @veles_cleanup_pop(ptr %a248)
  %t340 = load ptr, ptr %a247
  %t341 = call ptr @veles_scope_failed(ptr %t340)
  %t342 = icmp ne ptr %t341, null
  br i1 %t342, label %scope.check.145, label %scope.after.146
scope.check.145:
  %t343 = call i64 @veles_task_panicked(ptr %t341)
  %t344 = icmp ne i64 %t343, 0
  br i1 %t344, label %scope.repanic.147, label %scope.errors.148
scope.repanic.147:
  call void @veles_task_repanic(ptr %t341)
  unreachable
scope.errors.148:
  br label %scope.after.146
scope.after.146:
  %t345 = load i64, ptr %a245
  %t347 = call i64 @veles_i64_format(ptr %a346, i64 %t345)
  %t348 = insertvalue %str undef, ptr %a346, 0
  %t349 = insertvalue %str %t348, i64 %t347, 1
  %t351 = extractvalue %str { ptr @.str.21, i64 6 }, 0
  %t352 = extractvalue %str { ptr @.str.21, i64 6 }, 1
  %t353 = extractvalue %str %t349, 0
  %t354 = extractvalue %str %t349, 1
  call void @veles_string_concat(ptr %a350, ptr %t351, i64 %t352, ptr %t353, i64 %t354)
  %t355 = load %str, ptr %a350
  %t356 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t357 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t356)
  call void @v_std.io.println(%str %t355)
  call void @veles_call_pop()
  %t358 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t358, ptr %a359
  store i64 0, ptr %a360
  %t361 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t361, ptr %a362
  call void @veles_cleanup_push(ptr %a363, ptr @scope.cancel.thunk, ptr %a362)
  %t364 = load ptr, ptr %a362
  %t365 = call ptr @veles_task_launch(ptr %t364, i64 0)
  %t366 = load ptr, ptr %a359
  %t367 = call ptr @veles_alloc_words(i64 16)
  %t368 = getelementptr inbounds { ptr }, ptr %t367, i32 0, i32 0
  store ptr %t366, ptr %t368
  call void @veles_task_spawn(ptr %t365, ptr @entry.v_main.produce, ptr %t367)
  store ptr %t365, ptr %a369
  br label %loop.cond.152
loop.cond.152:
  %t370 = load i64, ptr %a360
  %t371 = icmp slt i64 %t370, 6
  br i1 %t371, label %loop.body.155, label %loop.end.154
loop.body.155:
  %t372 = load ptr, ptr %a359
  store i64 zeroinitializer, ptr %a373
  %t374 = call i64 @veles_chan_try_recv(ptr %t372, ptr %a373)
  %t375 = icmp ne i64 %t374, 0
  %t376 = load i64, ptr %a373
  %t377 = insertvalue { i1, i64 } undef, i1 %t375, 0
  %t378 = insertvalue { i1, i64 } %t377, i64 %t376, 1
  store { i1, i64 } %t378, ptr %a379
  %t380 = load { i1, i64 }, ptr %a379
  %t382 = extractvalue { i1, i64 } %t380, 0
  %t381 = xor i1 %t382, true
  %t383 = xor i1 %t381, true
  br i1 %t383, label %if.then.156, label %if.else.158
if.then.156:
  %t384 = load i64, ptr %a360
  %t385 = load { i1, i64 }, ptr %a379
  %t386 = extractvalue { i1, i64 } %t385, 1
  %t388 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t384, i64 %t386)
  %t389 = extractvalue { i64, i1 } %t388, 0
  %t390 = extractvalue { i64, i1 } %t388, 1
  br i1 %t390, label %overflow.159, label %arith.ok.160
overflow.159:
  %t391 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t392 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t393 = extractvalue %str { ptr @.str.23, i64 13 }, 0
  %t394 = extractvalue %str { ptr @.str.23, i64 13 }, 1
  call void @veles_panic_at(ptr %t391, i64 %t392, ptr %t393, i64 %t394)
  unreachable
arith.ok.160:
  store i64 %t389, ptr %a360
  br label %if.end.157
if.else.158:
  %t395 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t396 = load i64, ptr %t395
  %t398 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t396, i64 999999)
  %t399 = extractvalue { i64, i1 } %t398, 0
  %t400 = extractvalue { i64, i1 } %t398, 1
  br i1 %t400, label %overflow.161, label %arith.ok.162
overflow.161:
  %t401 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t402 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t403 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t404 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t401, i64 %t402, ptr %t403, i64 %t404)
  unreachable
arith.ok.162:
  %t406 = icmp eq i64 1000000, 0
  br i1 %t406, label %divzero.163, label %div.ok.164
divzero.163:
  %t407 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t408 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t409 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t410 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t407, i64 %t408, ptr %t409, i64 %t410)
  unreachable
div.ok.164:
  %t411 = icmp eq i64 %t399, -9223372036854775808
  %t412 = icmp eq i64 1000000, -1
  %t413 = and i1 %t411, %t412
  br i1 %t413, label %divof.165, label %div.ok.166
divof.165:
  %t414 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t415 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t416 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t417 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t414, i64 %t415, ptr %t416, i64 %t417)
  unreachable
div.ok.166:
  %t405 = sdiv i64 %t399, 1000000
  br label %sleep.167
sleep.167:
  %t418 = call i64 @veles_task_sleep(ptr %task, i64 %t405)
  %t419 = icmp ne i64 %t418, 0
  br i1 %t419, label %sleep.done.168, label %sleep.susp.169
sleep.susp.169:
  %t420 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t420, label %coro.suspend [ i8 0, label %resume.170 i8 1, label %coro.cleanup ]
resume.170:
  %t421 = call i64 @veles_task_cancelled(ptr %task)
  %t422 = icmp ne i64 %t421, 0
  br i1 %t422, label %cancelled.171, label %cont.172
cancelled.171:
  call void @veles_cleanup_pop(ptr %a363)
  %t423 = load ptr, ptr %a362
  call void @veles_scope_cancel(ptr %t423)
  br label %abandon.wait.173
abandon.wait.173:
  %t424 = call i64 @veles_scope_wait(ptr %task, ptr %t423)
  %t425 = icmp ne i64 %t424, 0
  br i1 %t425, label %abandon.done.174, label %abandon.susp.175
abandon.susp.175:
  %t426 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t426, label %coro.suspend [ i8 0, label %resume.176 i8 1, label %coro.cleanup ]
resume.176:
  br label %abandon.wait.173
abandon.done.174:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.172:
  %t427 = load ptr, ptr %a362
  %t428 = call ptr @veles_scope_failed(ptr %t427)
  %t429 = icmp ne ptr %t428, null
  br i1 %t429, label %scope.abort.177, label %scope.ok.178
scope.ok.178:
  br label %cont.179
scope.abort.177:
  call void @veles_task_leave_waits(ptr %task)
  %t430 = load ptr, ptr %a362
  call void @veles_scope_cancel(ptr %t430)
  br label %scope.wait.149
cont.179:
  br label %sleep.167
sleep.done.168:
  br label %if.end.157
if.end.157:
  br label %loop.post.153
loop.post.153:
  %t431 = load volatile i32, ptr @veles_stop_requested, align 4
  %t432 = icmp ne i32 %t431, 0
  br i1 %t432, label %safepoint.180, label %safepoint.on.181, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.180:
  call void @veles_gc_park()
  br label %safepoint.on.181
safepoint.on.181:
  br label %loop.cond.152
loop.end.154:
  br label %scope.wait.149
scope.wait.149:
  %t433 = load ptr, ptr %a362
  %t434 = call i64 @veles_scope_wait(ptr %task, ptr %t433)
  %t435 = icmp ne i64 %t434, 0
  br i1 %t435, label %scope.done.150, label %scope.susp.151
scope.susp.151:
  %t436 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t436, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  %t437 = call i64 @veles_task_cancelled(ptr %task)
  %t438 = icmp ne i64 %t437, 0
  br i1 %t438, label %cancelled.183, label %cont.184
cancelled.183:
  call void @veles_cleanup_pop(ptr %a363)
  %t439 = load ptr, ptr %a362
  call void @veles_scope_cancel(ptr %t439)
  br label %abandon.wait.185
abandon.wait.185:
  %t440 = call i64 @veles_scope_wait(ptr %task, ptr %t439)
  %t441 = icmp ne i64 %t440, 0
  br i1 %t441, label %abandon.done.186, label %abandon.susp.187
abandon.susp.187:
  %t442 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t442, label %coro.suspend [ i8 0, label %resume.188 i8 1, label %coro.cleanup ]
resume.188:
  br label %abandon.wait.185
abandon.done.186:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.184:
  br label %scope.wait.149
scope.done.150:
  call void @veles_cleanup_pop(ptr %a363)
  %t443 = load ptr, ptr %a362
  %t444 = call ptr @veles_scope_failed(ptr %t443)
  %t445 = icmp ne ptr %t444, null
  br i1 %t445, label %scope.check.189, label %scope.after.190
scope.check.189:
  %t446 = call i64 @veles_task_panicked(ptr %t444)
  %t447 = icmp ne i64 %t446, 0
  br i1 %t447, label %scope.repanic.191, label %scope.errors.192
scope.repanic.191:
  call void @veles_task_repanic(ptr %t444)
  unreachable
scope.errors.192:
  br label %scope.after.190
scope.after.190:
  %t448 = load i64, ptr %a360
  %t450 = call i64 @veles_i64_format(ptr %a449, i64 %t448)
  %t451 = insertvalue %str undef, ptr %a449, 0
  %t452 = insertvalue %str %t451, i64 %t450, 1
  %t454 = extractvalue %str { ptr @.str.25, i64 7 }, 0
  %t455 = extractvalue %str { ptr @.str.25, i64 7 }, 1
  %t456 = extractvalue %str %t452, 0
  %t457 = extractvalue %str %t452, 1
  call void @veles_string_concat(ptr %a453, ptr %t454, i64 %t455, ptr %t456, i64 %t457)
  %t458 = load %str, ptr %a453
  %t459 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t460 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t459)
  call void @v_std.io.println(%str %t458)
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
