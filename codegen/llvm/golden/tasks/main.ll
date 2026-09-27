%S.main.Resource = type { %str }
define ptr @v_main.slow(ptr %task, i64 %p1) presplitcoroutine {
entry:
  %a1 = alloca i64
  %a39 = alloca i64
  store i64 %p1, ptr %a1
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t2 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  %t3 = extractvalue %S.std.prelude.Duration %t2, 0
  %t5 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t3, i64 999999)
  %t6 = extractvalue { i64, i1 } %t5, 0
  %t7 = extractvalue { i64, i1 } %t5, 1
  br i1 %t7, label %overflow.1, label %arith.ok.2
overflow.1:
  %t8 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t10 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t11 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t8, i64 %t9, ptr %t10, i64 %t11)
  unreachable
arith.ok.2:
  %t13 = icmp eq i64 1000000, 0
  br i1 %t13, label %divzero.3, label %div.ok.4
divzero.3:
  %t14 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t15 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t16 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t17 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t14, i64 %t15, ptr %t16, i64 %t17)
  unreachable
div.ok.4:
  %t18 = icmp eq i64 %t6, -9223372036854775808
  %t19 = icmp eq i64 1000000, -1
  %t20 = and i1 %t18, %t19
  br i1 %t20, label %divof.5, label %div.ok.6
divof.5:
  %t21 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t22 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t23 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t24 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t21, i64 %t22, ptr %t23, i64 %t24)
  unreachable
div.ok.6:
  %t12 = sdiv i64 %t6, 1000000
  br label %sleep.7
sleep.7:
  %t25 = call i64 @veles_task_sleep(ptr %task, i64 %t12)
  %t26 = icmp ne i64 %t25, 0
  br i1 %t26, label %sleep.done.8, label %sleep.susp.9
sleep.susp.9:
  %t27 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t27, label %coro.suspend [ i8 0, label %resume.10 i8 1, label %coro.cleanup ]
resume.10:
  %t28 = call i64 @veles_task_cancelled(ptr %task)
  %t29 = icmp ne i64 %t28, 0
  br i1 %t29, label %cancelled.11, label %cont.12
cancelled.11:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.12:
  br label %sleep.7
sleep.done.8:
  %t30 = load i64, ptr %a1
  %t32 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t30, i64 2)
  %t33 = extractvalue { i64, i1 } %t32, 0
  %t34 = extractvalue { i64, i1 } %t32, 1
  br i1 %t34, label %overflow.13, label %arith.ok.14
overflow.13:
  %t35 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t36 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t37 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t38 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t35, i64 %t36, ptr %t37, i64 %t38)
  unreachable
arith.ok.14:
  store i64 %t33, ptr %a39
  call void @veles_task_finish(ptr %task, ptr %a39, i64 8, i64 0)
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
  %a23 = alloca i64
  %a26 = alloca i64
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
  br label %loop.cond.1
loop.cond.1:
  %t13 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t14 = load i1, ptr %t13
  br i1 %t14, label %if.then.5, label %if.else.7
if.then.5:
  %t15 = load i64, ptr %a8
  %t16 = load i64, ptr %a11
  %t17 = icmp sle i64 %t15, %t16
  store i1 %t17, ptr %a12
  br label %if.end.6
if.else.7:
  %t18 = load i64, ptr %a8
  %t19 = load i64, ptr %a11
  %t20 = icmp slt i64 %t18, %t19
  store i1 %t20, ptr %a12
  br label %if.end.6
if.end.6:
  %t21 = load i1, ptr %a12
  br i1 %t21, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t22 = load i64, ptr %a8
  store i64 %t22, ptr %a23
  %t24 = load ptr, ptr %a1
  %t25 = load i64, ptr %a23
  store i64 %t25, ptr %a26
  br label %send.8
send.8:
  %t27 = call i64 @veles_chan_send(ptr %task, ptr %t24, ptr %a26)
  %t28 = icmp ne i64 %t27, 0
  br i1 %t28, label %send.done.9, label %send.susp.10
send.susp.10:
  %t29 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t29, label %coro.suspend [ i8 0, label %resume.11 i8 1, label %coro.cleanup ]
resume.11:
  %t30 = call i64 @veles_task_cancelled(ptr %task)
  %t31 = icmp ne i64 %t30, 0
  br i1 %t31, label %cancelled.12, label %cont.13
cancelled.12:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.13:
  br label %send.8
send.done.9:
  br label %loop.post.2
loop.post.2:
  %t32 = load i64, ptr %a8
  %t33 = add i64 %t32, 1
  store i64 %t33, ptr %a8
  %t34 = load volatile i32, ptr @veles_stop_requested, align 4
  %t35 = icmp ne i32 %t34, 0
  br i1 %t35, label %safepoint.14, label %safepoint.on.15, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.14:
  call void @veles_gc_park()
  br label %safepoint.on.15
safepoint.on.15:
  br label %loop.cond.1
loop.end.3:
  %t36 = load ptr, ptr %a1
  call void @veles_chan_close(ptr %t36)
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
  %a36 = alloca [21 x i8]
  %a57 = alloca [21 x i8]
  %a61 = alloca %str
  %a62 = alloca [3 x %str]
  %a83 = alloca ptr
  %a84 = alloca i64
  %a86 = alloca ptr
  %a87 = alloca { ptr, ptr, ptr }
  %a93 = alloca ptr
  %a95 = alloca i64
  %a112 = alloca i64
  %a117 = alloca i64
  %a146 = alloca [21 x i8]
  %a150 = alloca %str
  %a157 = alloca ptr
  %a159 = alloca %str
  %a162 = alloca %str
  %a165 = alloca %str
  %a168 = alloca %str
  %a171 = alloca %str
  %a174 = alloca %str
  %a177 = alloca %str
  %a183 = alloca %str
  %a189 = alloca %str
  %a195 = alloca %str
  %a201 = alloca %str
  %a207 = alloca %str
  %a212 = alloca %str
  %a213 = alloca [11 x %str]
  %a227 = alloca ptr
  %a228 = alloca i64
  %a230 = alloca ptr
  %a231 = alloca { ptr, ptr, ptr }
  %a237 = alloca ptr
  %a277 = alloca i64
  %a294 = alloca i64
  %a325 = alloca [21 x i8]
  %a329 = alloca %str
  %a336 = alloca ptr
  %a337 = alloca i64
  %a339 = alloca ptr
  %a340 = alloca { ptr, ptr, ptr }
  %a346 = alloca ptr
  %a350 = alloca i64
  %a356 = alloca { i1, i64 }
  %a425 = alloca [21 x i8]
  %a429 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = insertvalue %S.main.Resource undef, %str { ptr @.str.5, i64 2 }, 0
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
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.9:
  %t29 = load ptr, ptr %a5
  %t30 = call ptr @veles_scope_failed(ptr %t29)
  %t31 = icmp ne ptr %t30, null
  br i1 %t31, label %scope.abort.14, label %scope.ok.15
scope.ok.15:
  br label %cont.16
scope.abort.14:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.1
cont.16:
  br label %await.4
await.got.5:
  %t32 = call i64 @veles_task_panicked(ptr %t19)
  %t33 = icmp ne i64 %t32, 0
  br i1 %t33, label %await.repanic.17, label %await.fine.18
await.repanic.17:
  call void @veles_task_repanic(ptr %t19)
  unreachable
await.fine.18:
  %t34 = call ptr @veles_task_result(ptr %t19)
  %t35 = load i64, ptr %t34
  %t37 = call i64 @veles_i64_format(ptr %a36, i64 %t35)
  %t38 = insertvalue %str undef, ptr %a36, 0
  %t39 = insertvalue %str %t38, i64 %t37, 1
  %t40 = load ptr, ptr %a18
  br label %await.19
await.19:
  %t41 = call i64 @veles_task_await(ptr %task, ptr %t40)
  %t42 = icmp ne i64 %t41, 0
  br i1 %t42, label %await.got.20, label %await.susp.21
await.susp.21:
  %t43 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t43, label %coro.suspend [ i8 0, label %resume.22 i8 1, label %coro.cleanup ]
resume.22:
  %t44 = call i64 @veles_task_cancelled(ptr %task)
  %t45 = icmp ne i64 %t44, 0
  br i1 %t45, label %cancelled.23, label %cont.24
cancelled.23:
  call void @veles_cleanup_pop(ptr %a6)
  %t46 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t46)
  br label %abandon.wait.25
abandon.wait.25:
  %t47 = call i64 @veles_scope_wait(ptr %task, ptr %t46)
  %t48 = icmp ne i64 %t47, 0
  br i1 %t48, label %abandon.done.26, label %abandon.susp.27
abandon.susp.27:
  %t49 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t49, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  br label %abandon.wait.25
abandon.done.26:
  call void @veles_cleanup_pop(ptr %a3)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.24:
  %t50 = load ptr, ptr %a5
  %t51 = call ptr @veles_scope_failed(ptr %t50)
  %t52 = icmp ne ptr %t51, null
  br i1 %t52, label %scope.abort.29, label %scope.ok.30
scope.ok.30:
  br label %cont.31
scope.abort.29:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.1
cont.31:
  br label %await.19
await.got.20:
  %t53 = call i64 @veles_task_panicked(ptr %t40)
  %t54 = icmp ne i64 %t53, 0
  br i1 %t54, label %await.repanic.32, label %await.fine.33
await.repanic.32:
  call void @veles_task_repanic(ptr %t40)
  unreachable
await.fine.33:
  %t55 = call ptr @veles_task_result(ptr %t40)
  %t56 = load i64, ptr %t55
  %t58 = call i64 @veles_i64_format(ptr %a57, i64 %t56)
  %t59 = insertvalue %str undef, ptr %a57, 0
  %t60 = insertvalue %str %t59, i64 %t58, 1
  %t63 = getelementptr [3 x %str], ptr %a62, i64 0, i64 0
  store %str %t39, ptr %t63
  %t64 = getelementptr [3 x %str], ptr %a62, i64 0, i64 1
  store %str { ptr @.str.6, i64 1 }, ptr %t64
  %t65 = getelementptr [3 x %str], ptr %a62, i64 0, i64 2
  store %str %t60, ptr %t65
  call void @veles_string_concat_n(ptr %a61, ptr %a62, i64 3)
  %t66 = load %str, ptr %a61
  call void @v_std.io.println(%str %t66)
  br label %scope.wait.1
scope.wait.1:
  %t67 = load ptr, ptr %a5
  %t68 = call i64 @veles_scope_wait(ptr %task, ptr %t67)
  %t69 = icmp ne i64 %t68, 0
  br i1 %t69, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  %t70 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t70, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t71 = call i64 @veles_task_cancelled(ptr %task)
  %t72 = icmp ne i64 %t71, 0
  br i1 %t72, label %cancelled.35, label %cont.36
cancelled.35:
  call void @veles_cleanup_pop(ptr %a6)
  %t73 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t73)
  br label %abandon.wait.37
abandon.wait.37:
  %t74 = call i64 @veles_scope_wait(ptr %task, ptr %t73)
  %t75 = icmp ne i64 %t74, 0
  br i1 %t75, label %abandon.done.38, label %abandon.susp.39
abandon.susp.39:
  %t76 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t76, label %coro.suspend [ i8 0, label %resume.40 i8 1, label %coro.cleanup ]
resume.40:
  br label %abandon.wait.37
abandon.done.38:
  call void @veles_cleanup_pop(ptr %a3)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.36:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop(ptr %a6)
  %t77 = load ptr, ptr %a5
  %t78 = call ptr @veles_scope_failed(ptr %t77)
  %t79 = icmp ne ptr %t78, null
  br i1 %t79, label %scope.check.41, label %scope.after.42
scope.check.41:
  %t80 = call i64 @veles_task_panicked(ptr %t78)
  %t81 = icmp ne i64 %t80, 0
  br i1 %t81, label %scope.repanic.43, label %scope.errors.44
scope.repanic.43:
  call void @veles_task_repanic(ptr %t78)
  unreachable
scope.errors.44:
  br label %scope.after.42
scope.after.42:
  call void @veles_cleanup_pop(ptr %a3)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  %t82 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t82, ptr %a83
  store i64 0, ptr %a84
  %t85 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t85, ptr %a86
  call void @veles_cleanup_push(ptr %a87, ptr @scope.cancel.thunk, ptr %a86)
  %t88 = load ptr, ptr %a86
  %t89 = call ptr @veles_task_launch(ptr %t88, i64 0)
  %t90 = load ptr, ptr %a83
  %t91 = call ptr @veles_alloc_words(i64 16)
  %t92 = getelementptr inbounds { ptr }, ptr %t91, i32 0, i32 0
  store ptr %t90, ptr %t92
  call void @veles_task_spawn(ptr %t89, ptr @entry.v_main.produce, ptr %t91)
  store ptr %t89, ptr %a93
  br label %loop.cond.48
loop.cond.48:
  br label %loop.body.51
loop.body.51:
  %t94 = load ptr, ptr %a83
  store i64 zeroinitializer, ptr %a95
  br label %recv.52
recv.52:
  %t96 = call i64 @veles_chan_recv(ptr %task, ptr %t94, ptr %a95)
  %t97 = icmp ne i64 %t96, 0
  br i1 %t97, label %recv.done.53, label %recv.susp.54
recv.susp.54:
  %t98 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t98, label %coro.suspend [ i8 0, label %resume.55 i8 1, label %coro.cleanup ]
resume.55:
  %t99 = call i64 @veles_task_cancelled(ptr %task)
  %t100 = icmp ne i64 %t99, 0
  br i1 %t100, label %cancelled.56, label %cont.57
cancelled.56:
  call void @veles_cleanup_pop(ptr %a87)
  %t101 = load ptr, ptr %a86
  call void @veles_scope_cancel(ptr %t101)
  br label %abandon.wait.58
abandon.wait.58:
  %t102 = call i64 @veles_scope_wait(ptr %task, ptr %t101)
  %t103 = icmp ne i64 %t102, 0
  br i1 %t103, label %abandon.done.59, label %abandon.susp.60
abandon.susp.60:
  %t104 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t104, label %coro.suspend [ i8 0, label %resume.61 i8 1, label %coro.cleanup ]
resume.61:
  br label %abandon.wait.58
abandon.done.59:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.57:
  %t105 = load ptr, ptr %a86
  %t106 = call ptr @veles_scope_failed(ptr %t105)
  %t107 = icmp ne ptr %t106, null
  br i1 %t107, label %scope.abort.62, label %scope.ok.63
scope.ok.63:
  br label %cont.64
scope.abort.62:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.45
cont.64:
  br label %recv.52
recv.done.53:
  %t108 = icmp eq i64 %t96, 1
  %t109 = load i64, ptr %a95
  %t110 = insertvalue { i1, i64 } undef, i1 %t108, 0
  %t111 = insertvalue { i1, i64 } %t110, i64 %t109, 1
  %t114 = extractvalue { i1, i64 } %t111, 0
  %t113 = xor i1 %t114, true
  br i1 %t113, label %elvis.default.65, label %elvis.some.66
elvis.some.66:
  %t115 = extractvalue { i1, i64 } %t111, 1
  store i64 %t115, ptr %a112
  br label %elvis.end.67
elvis.default.65:
  br label %loop.end.50
elvis.end.67:
  %t116 = load i64, ptr %a112
  store i64 %t116, ptr %a117
  %t118 = load i64, ptr %a84
  %t119 = load i64, ptr %a117
  %t121 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t118, i64 %t119)
  %t122 = extractvalue { i64, i1 } %t121, 0
  %t123 = extractvalue { i64, i1 } %t121, 1
  br i1 %t123, label %overflow.68, label %arith.ok.69
overflow.68:
  %t124 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t125 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t126 = extractvalue %str { ptr @.str.7, i64 12 }, 0
  %t127 = extractvalue %str { ptr @.str.7, i64 12 }, 1
  call void @veles_panic_at(ptr %t124, i64 %t125, ptr %t126, i64 %t127)
  unreachable
arith.ok.69:
  store i64 %t122, ptr %a84
  br label %loop.post.49
loop.post.49:
  %t128 = load volatile i32, ptr @veles_stop_requested, align 4
  %t129 = icmp ne i32 %t128, 0
  br i1 %t129, label %safepoint.70, label %safepoint.on.71, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.70:
  call void @veles_gc_park()
  br label %safepoint.on.71
safepoint.on.71:
  br label %loop.cond.48
loop.end.50:
  br label %scope.wait.45
scope.wait.45:
  %t130 = load ptr, ptr %a86
  %t131 = call i64 @veles_scope_wait(ptr %task, ptr %t130)
  %t132 = icmp ne i64 %t131, 0
  br i1 %t132, label %scope.done.46, label %scope.susp.47
scope.susp.47:
  %t133 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t133, label %coro.suspend [ i8 0, label %resume.72 i8 1, label %coro.cleanup ]
resume.72:
  %t134 = call i64 @veles_task_cancelled(ptr %task)
  %t135 = icmp ne i64 %t134, 0
  br i1 %t135, label %cancelled.73, label %cont.74
cancelled.73:
  call void @veles_cleanup_pop(ptr %a87)
  %t136 = load ptr, ptr %a86
  call void @veles_scope_cancel(ptr %t136)
  br label %abandon.wait.75
abandon.wait.75:
  %t137 = call i64 @veles_scope_wait(ptr %task, ptr %t136)
  %t138 = icmp ne i64 %t137, 0
  br i1 %t138, label %abandon.done.76, label %abandon.susp.77
abandon.susp.77:
  %t139 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t139, label %coro.suspend [ i8 0, label %resume.78 i8 1, label %coro.cleanup ]
resume.78:
  br label %abandon.wait.75
abandon.done.76:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.74:
  br label %scope.wait.45
scope.done.46:
  call void @veles_cleanup_pop(ptr %a87)
  %t140 = load ptr, ptr %a86
  %t141 = call ptr @veles_scope_failed(ptr %t140)
  %t142 = icmp ne ptr %t141, null
  br i1 %t142, label %scope.check.79, label %scope.after.80
scope.check.79:
  %t143 = call i64 @veles_task_panicked(ptr %t141)
  %t144 = icmp ne i64 %t143, 0
  br i1 %t144, label %scope.repanic.81, label %scope.errors.82
scope.repanic.81:
  call void @veles_task_repanic(ptr %t141)
  unreachable
scope.errors.82:
  br label %scope.after.80
scope.after.80:
  %t145 = load i64, ptr %a84
  %t147 = call i64 @veles_i64_format(ptr %a146, i64 %t145)
  %t148 = insertvalue %str undef, ptr %a146, 0
  %t149 = insertvalue %str %t148, i64 %t147, 1
  %t151 = extractvalue %str { ptr @.str.8, i64 4 }, 0
  %t152 = extractvalue %str { ptr @.str.8, i64 4 }, 1
  %t153 = extractvalue %str %t149, 0
  %t154 = extractvalue %str %t149, 1
  call void @veles_string_concat(ptr %a150, ptr %t151, i64 %t152, ptr %t153, i64 %t154)
  %t155 = load %str, ptr %a150
  call void @v_std.io.println(%str %t155)
  %t156 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t156, ptr %a157
  %t158 = load ptr, ptr %a157
  store %str { ptr @.str.9, i64 1 }, ptr %a159
  %t160 = call i64 @veles_chan_try_send(ptr %t158, ptr %a159)
  %t161 = icmp ne i64 %t160, 0
  call void @veles_bool_to_string(ptr %a162, i1 zeroext %t161)
  %t163 = load %str, ptr %a162
  %t164 = load ptr, ptr %a157
  store %str { ptr @.str.10, i64 1 }, ptr %a165
  %t166 = call i64 @veles_chan_try_send(ptr %t164, ptr %a165)
  %t167 = icmp ne i64 %t166, 0
  call void @veles_bool_to_string(ptr %a168, i1 zeroext %t167)
  %t169 = load %str, ptr %a168
  %t170 = load ptr, ptr %a157
  store %str { ptr @.str.11, i64 1 }, ptr %a171
  %t172 = call i64 @veles_chan_try_send(ptr %t170, ptr %a171)
  %t173 = icmp ne i64 %t172, 0
  call void @veles_bool_to_string(ptr %a174, i1 zeroext %t173)
  %t175 = load %str, ptr %a174
  %t176 = load ptr, ptr %a157
  store %str zeroinitializer, ptr %a177
  %t178 = call i64 @veles_chan_try_recv(ptr %t176, ptr %a177)
  %t179 = icmp ne i64 %t178, 0
  %t180 = load %str, ptr %a177
  %t181 = insertvalue { i1, %str } undef, i1 %t179, 0
  %t182 = insertvalue { i1, %str } %t181, %str %t180, 1
  %t185 = extractvalue { i1, %str } %t182, 0
  %t184 = xor i1 %t185, true
  br i1 %t184, label %elvis.default.83, label %elvis.some.84
elvis.some.84:
  %t186 = extractvalue { i1, %str } %t182, 1
  store %str %t186, ptr %a183
  br label %elvis.end.85
elvis.default.83:
  store %str { ptr @.str.12, i64 1 }, ptr %a183
  br label %elvis.end.85
elvis.end.85:
  %t187 = load %str, ptr %a183
  %t188 = load ptr, ptr %a157
  store %str zeroinitializer, ptr %a189
  %t190 = call i64 @veles_chan_try_recv(ptr %t188, ptr %a189)
  %t191 = icmp ne i64 %t190, 0
  %t192 = load %str, ptr %a189
  %t193 = insertvalue { i1, %str } undef, i1 %t191, 0
  %t194 = insertvalue { i1, %str } %t193, %str %t192, 1
  %t197 = extractvalue { i1, %str } %t194, 0
  %t196 = xor i1 %t197, true
  br i1 %t196, label %elvis.default.86, label %elvis.some.87
elvis.some.87:
  %t198 = extractvalue { i1, %str } %t194, 1
  store %str %t198, ptr %a195
  br label %elvis.end.88
elvis.default.86:
  store %str { ptr @.str.12, i64 1 }, ptr %a195
  br label %elvis.end.88
elvis.end.88:
  %t199 = load %str, ptr %a195
  %t200 = load ptr, ptr %a157
  store %str zeroinitializer, ptr %a201
  %t202 = call i64 @veles_chan_try_recv(ptr %t200, ptr %a201)
  %t203 = icmp ne i64 %t202, 0
  %t204 = load %str, ptr %a201
  %t205 = insertvalue { i1, %str } undef, i1 %t203, 0
  %t206 = insertvalue { i1, %str } %t205, %str %t204, 1
  %t209 = extractvalue { i1, %str } %t206, 0
  %t208 = xor i1 %t209, true
  br i1 %t208, label %elvis.default.89, label %elvis.some.90
elvis.some.90:
  %t210 = extractvalue { i1, %str } %t206, 1
  store %str %t210, ptr %a207
  br label %elvis.end.91
elvis.default.89:
  store %str { ptr @.str.12, i64 1 }, ptr %a207
  br label %elvis.end.91
elvis.end.91:
  %t211 = load %str, ptr %a207
  %t214 = getelementptr [11 x %str], ptr %a213, i64 0, i64 0
  store %str %t163, ptr %t214
  %t215 = getelementptr [11 x %str], ptr %a213, i64 0, i64 1
  store %str { ptr @.str.6, i64 1 }, ptr %t215
  %t216 = getelementptr [11 x %str], ptr %a213, i64 0, i64 2
  store %str %t169, ptr %t216
  %t217 = getelementptr [11 x %str], ptr %a213, i64 0, i64 3
  store %str { ptr @.str.6, i64 1 }, ptr %t217
  %t218 = getelementptr [11 x %str], ptr %a213, i64 0, i64 4
  store %str %t175, ptr %t218
  %t219 = getelementptr [11 x %str], ptr %a213, i64 0, i64 5
  store %str { ptr @.str.6, i64 1 }, ptr %t219
  %t220 = getelementptr [11 x %str], ptr %a213, i64 0, i64 6
  store %str %t187, ptr %t220
  %t221 = getelementptr [11 x %str], ptr %a213, i64 0, i64 7
  store %str { ptr @.str.6, i64 1 }, ptr %t221
  %t222 = getelementptr [11 x %str], ptr %a213, i64 0, i64 8
  store %str %t199, ptr %t222
  %t223 = getelementptr [11 x %str], ptr %a213, i64 0, i64 9
  store %str { ptr @.str.6, i64 1 }, ptr %t223
  %t224 = getelementptr [11 x %str], ptr %a213, i64 0, i64 10
  store %str %t211, ptr %t224
  call void @veles_string_concat_n(ptr %a212, ptr %a213, i64 11)
  %t225 = load %str, ptr %a212
  call void @v_std.io.println(%str %t225)
  %t226 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t226, ptr %a227
  store i64 0, ptr %a228
  %t229 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t229, ptr %a230
  call void @veles_cleanup_push(ptr %a231, ptr @scope.cancel.thunk, ptr %a230)
  %t232 = load ptr, ptr %a230
  %t233 = call ptr @veles_task_launch(ptr %t232, i64 0)
  %t234 = load ptr, ptr %a227
  %t235 = call ptr @veles_alloc_words(i64 16)
  %t236 = getelementptr inbounds { ptr }, ptr %t235, i32 0, i32 0
  store ptr %t234, ptr %t236
  call void @veles_task_spawn(ptr %t233, ptr @entry.v_main.produce, ptr %t235)
  store ptr %t233, ptr %a237
  br label %loop.cond.95
loop.cond.95:
  %t238 = load i64, ptr %a228
  %t239 = icmp slt i64 %t238, 6
  br i1 %t239, label %loop.body.98, label %loop.end.97
loop.body.98:
  %t240 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  %t241 = extractvalue %S.std.prelude.Duration %t240, 0
  %t243 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t241, i64 999999)
  %t244 = extractvalue { i64, i1 } %t243, 0
  %t245 = extractvalue { i64, i1 } %t243, 1
  br i1 %t245, label %overflow.99, label %arith.ok.100
overflow.99:
  %t246 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t247 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t248 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t249 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t246, i64 %t247, ptr %t248, i64 %t249)
  unreachable
arith.ok.100:
  %t251 = icmp eq i64 1000000, 0
  br i1 %t251, label %divzero.101, label %div.ok.102
divzero.101:
  %t252 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t253 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t254 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t255 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t252, i64 %t253, ptr %t254, i64 %t255)
  unreachable
div.ok.102:
  %t256 = icmp eq i64 %t244, -9223372036854775808
  %t257 = icmp eq i64 1000000, -1
  %t258 = and i1 %t256, %t257
  br i1 %t258, label %divof.103, label %div.ok.104
divof.103:
  %t259 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t260 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t261 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t262 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t259, i64 %t260, ptr %t261, i64 %t262)
  unreachable
div.ok.104:
  %t250 = sdiv i64 %t244, 1000000
  br label %sleep.105
sleep.105:
  %t263 = call i64 @veles_task_sleep(ptr %task, i64 %t250)
  %t264 = icmp ne i64 %t263, 0
  br i1 %t264, label %sleep.done.106, label %sleep.susp.107
sleep.susp.107:
  %t265 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t265, label %coro.suspend [ i8 0, label %resume.108 i8 1, label %coro.cleanup ]
resume.108:
  %t266 = call i64 @veles_task_cancelled(ptr %task)
  %t267 = icmp ne i64 %t266, 0
  br i1 %t267, label %cancelled.109, label %cont.110
cancelled.109:
  call void @veles_cleanup_pop(ptr %a231)
  %t268 = load ptr, ptr %a230
  call void @veles_scope_cancel(ptr %t268)
  br label %abandon.wait.111
abandon.wait.111:
  %t269 = call i64 @veles_scope_wait(ptr %task, ptr %t268)
  %t270 = icmp ne i64 %t269, 0
  br i1 %t270, label %abandon.done.112, label %abandon.susp.113
abandon.susp.113:
  %t271 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t271, label %coro.suspend [ i8 0, label %resume.114 i8 1, label %coro.cleanup ]
resume.114:
  br label %abandon.wait.111
abandon.done.112:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.110:
  %t272 = load ptr, ptr %a230
  %t273 = call ptr @veles_scope_failed(ptr %t272)
  %t274 = icmp ne ptr %t273, null
  br i1 %t274, label %scope.abort.115, label %scope.ok.116
scope.ok.116:
  br label %cont.117
scope.abort.115:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.92
cont.117:
  br label %sleep.105
sleep.done.106:
  %t275 = load i64, ptr %a228
  %t276 = load ptr, ptr %a227
  store i64 zeroinitializer, ptr %a277
  br label %recv.118
recv.118:
  %t278 = call i64 @veles_chan_recv(ptr %task, ptr %t276, ptr %a277)
  %t279 = icmp ne i64 %t278, 0
  br i1 %t279, label %recv.done.119, label %recv.susp.120
recv.susp.120:
  %t280 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t280, label %coro.suspend [ i8 0, label %resume.121 i8 1, label %coro.cleanup ]
resume.121:
  %t281 = call i64 @veles_task_cancelled(ptr %task)
  %t282 = icmp ne i64 %t281, 0
  br i1 %t282, label %cancelled.122, label %cont.123
cancelled.122:
  call void @veles_cleanup_pop(ptr %a231)
  %t283 = load ptr, ptr %a230
  call void @veles_scope_cancel(ptr %t283)
  br label %abandon.wait.124
abandon.wait.124:
  %t284 = call i64 @veles_scope_wait(ptr %task, ptr %t283)
  %t285 = icmp ne i64 %t284, 0
  br i1 %t285, label %abandon.done.125, label %abandon.susp.126
abandon.susp.126:
  %t286 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t286, label %coro.suspend [ i8 0, label %resume.127 i8 1, label %coro.cleanup ]
resume.127:
  br label %abandon.wait.124
abandon.done.125:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.123:
  %t287 = load ptr, ptr %a230
  %t288 = call ptr @veles_scope_failed(ptr %t287)
  %t289 = icmp ne ptr %t288, null
  br i1 %t289, label %scope.abort.128, label %scope.ok.129
scope.ok.129:
  br label %cont.130
scope.abort.128:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.92
cont.130:
  br label %recv.118
recv.done.119:
  %t290 = icmp eq i64 %t278, 1
  %t291 = load i64, ptr %a277
  %t292 = insertvalue { i1, i64 } undef, i1 %t290, 0
  %t293 = insertvalue { i1, i64 } %t292, i64 %t291, 1
  %t296 = extractvalue { i1, i64 } %t293, 0
  %t295 = xor i1 %t296, true
  br i1 %t295, label %elvis.default.131, label %elvis.some.132
elvis.some.132:
  %t297 = extractvalue { i1, i64 } %t293, 1
  store i64 %t297, ptr %a294
  br label %elvis.end.133
elvis.default.131:
  store i64 0, ptr %a294
  br label %elvis.end.133
elvis.end.133:
  %t298 = load i64, ptr %a294
  %t300 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t275, i64 %t298)
  %t301 = extractvalue { i64, i1 } %t300, 0
  %t302 = extractvalue { i64, i1 } %t300, 1
  br i1 %t302, label %overflow.134, label %arith.ok.135
overflow.134:
  %t303 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t304 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t305 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t306 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_panic_at(ptr %t303, i64 %t304, ptr %t305, i64 %t306)
  unreachable
arith.ok.135:
  store i64 %t301, ptr %a228
  br label %loop.post.96
loop.post.96:
  %t307 = load volatile i32, ptr @veles_stop_requested, align 4
  %t308 = icmp ne i32 %t307, 0
  br i1 %t308, label %safepoint.136, label %safepoint.on.137, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.136:
  call void @veles_gc_park()
  br label %safepoint.on.137
safepoint.on.137:
  br label %loop.cond.95
loop.end.97:
  br label %scope.wait.92
scope.wait.92:
  %t309 = load ptr, ptr %a230
  %t310 = call i64 @veles_scope_wait(ptr %task, ptr %t309)
  %t311 = icmp ne i64 %t310, 0
  br i1 %t311, label %scope.done.93, label %scope.susp.94
scope.susp.94:
  %t312 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t312, label %coro.suspend [ i8 0, label %resume.138 i8 1, label %coro.cleanup ]
resume.138:
  %t313 = call i64 @veles_task_cancelled(ptr %task)
  %t314 = icmp ne i64 %t313, 0
  br i1 %t314, label %cancelled.139, label %cont.140
cancelled.139:
  call void @veles_cleanup_pop(ptr %a231)
  %t315 = load ptr, ptr %a230
  call void @veles_scope_cancel(ptr %t315)
  br label %abandon.wait.141
abandon.wait.141:
  %t316 = call i64 @veles_scope_wait(ptr %task, ptr %t315)
  %t317 = icmp ne i64 %t316, 0
  br i1 %t317, label %abandon.done.142, label %abandon.susp.143
abandon.susp.143:
  %t318 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t318, label %coro.suspend [ i8 0, label %resume.144 i8 1, label %coro.cleanup ]
resume.144:
  br label %abandon.wait.141
abandon.done.142:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.140:
  br label %scope.wait.92
scope.done.93:
  call void @veles_cleanup_pop(ptr %a231)
  %t319 = load ptr, ptr %a230
  %t320 = call ptr @veles_scope_failed(ptr %t319)
  %t321 = icmp ne ptr %t320, null
  br i1 %t321, label %scope.check.145, label %scope.after.146
scope.check.145:
  %t322 = call i64 @veles_task_panicked(ptr %t320)
  %t323 = icmp ne i64 %t322, 0
  br i1 %t323, label %scope.repanic.147, label %scope.errors.148
scope.repanic.147:
  call void @veles_task_repanic(ptr %t320)
  unreachable
scope.errors.148:
  br label %scope.after.146
scope.after.146:
  %t324 = load i64, ptr %a228
  %t326 = call i64 @veles_i64_format(ptr %a325, i64 %t324)
  %t327 = insertvalue %str undef, ptr %a325, 0
  %t328 = insertvalue %str %t327, i64 %t326, 1
  %t330 = extractvalue %str { ptr @.str.15, i64 6 }, 0
  %t331 = extractvalue %str { ptr @.str.15, i64 6 }, 1
  %t332 = extractvalue %str %t328, 0
  %t333 = extractvalue %str %t328, 1
  call void @veles_string_concat(ptr %a329, ptr %t330, i64 %t331, ptr %t332, i64 %t333)
  %t334 = load %str, ptr %a329
  call void @v_std.io.println(%str %t334)
  %t335 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t335, ptr %a336
  store i64 0, ptr %a337
  %t338 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t338, ptr %a339
  call void @veles_cleanup_push(ptr %a340, ptr @scope.cancel.thunk, ptr %a339)
  %t341 = load ptr, ptr %a339
  %t342 = call ptr @veles_task_launch(ptr %t341, i64 0)
  %t343 = load ptr, ptr %a336
  %t344 = call ptr @veles_alloc_words(i64 16)
  %t345 = getelementptr inbounds { ptr }, ptr %t344, i32 0, i32 0
  store ptr %t343, ptr %t345
  call void @veles_task_spawn(ptr %t342, ptr @entry.v_main.produce, ptr %t344)
  store ptr %t342, ptr %a346
  br label %loop.cond.152
loop.cond.152:
  %t347 = load i64, ptr %a337
  %t348 = icmp slt i64 %t347, 6
  br i1 %t348, label %loop.body.155, label %loop.end.154
loop.body.155:
  %t349 = load ptr, ptr %a336
  store i64 zeroinitializer, ptr %a350
  %t351 = call i64 @veles_chan_try_recv(ptr %t349, ptr %a350)
  %t352 = icmp ne i64 %t351, 0
  %t353 = load i64, ptr %a350
  %t354 = insertvalue { i1, i64 } undef, i1 %t352, 0
  %t355 = insertvalue { i1, i64 } %t354, i64 %t353, 1
  store { i1, i64 } %t355, ptr %a356
  %t357 = load { i1, i64 }, ptr %a356
  %t359 = extractvalue { i1, i64 } %t357, 0
  %t358 = xor i1 %t359, true
  %t360 = xor i1 %t358, true
  br i1 %t360, label %if.then.156, label %if.else.158
if.then.156:
  %t361 = load i64, ptr %a337
  %t362 = load { i1, i64 }, ptr %a356
  %t363 = extractvalue { i1, i64 } %t362, 1
  %t365 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t361, i64 %t363)
  %t366 = extractvalue { i64, i1 } %t365, 0
  %t367 = extractvalue { i64, i1 } %t365, 1
  br i1 %t367, label %overflow.159, label %arith.ok.160
overflow.159:
  %t368 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t369 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t370 = extractvalue %str { ptr @.str.16, i64 13 }, 0
  %t371 = extractvalue %str { ptr @.str.16, i64 13 }, 1
  call void @veles_panic_at(ptr %t368, i64 %t369, ptr %t370, i64 %t371)
  unreachable
arith.ok.160:
  store i64 %t366, ptr %a337
  br label %if.end.157
if.else.158:
  %t372 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t373 = load i64, ptr %t372
  %t375 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t373, i64 999999)
  %t376 = extractvalue { i64, i1 } %t375, 0
  %t377 = extractvalue { i64, i1 } %t375, 1
  br i1 %t377, label %overflow.161, label %arith.ok.162
overflow.161:
  %t378 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t379 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t380 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t381 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t378, i64 %t379, ptr %t380, i64 %t381)
  unreachable
arith.ok.162:
  %t383 = icmp eq i64 1000000, 0
  br i1 %t383, label %divzero.163, label %div.ok.164
divzero.163:
  %t384 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t385 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t386 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t387 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t384, i64 %t385, ptr %t386, i64 %t387)
  unreachable
div.ok.164:
  %t388 = icmp eq i64 %t376, -9223372036854775808
  %t389 = icmp eq i64 1000000, -1
  %t390 = and i1 %t388, %t389
  br i1 %t390, label %divof.165, label %div.ok.166
divof.165:
  %t391 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t392 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t393 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t394 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t391, i64 %t392, ptr %t393, i64 %t394)
  unreachable
div.ok.166:
  %t382 = sdiv i64 %t376, 1000000
  br label %sleep.167
sleep.167:
  %t395 = call i64 @veles_task_sleep(ptr %task, i64 %t382)
  %t396 = icmp ne i64 %t395, 0
  br i1 %t396, label %sleep.done.168, label %sleep.susp.169
sleep.susp.169:
  %t397 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t397, label %coro.suspend [ i8 0, label %resume.170 i8 1, label %coro.cleanup ]
resume.170:
  %t398 = call i64 @veles_task_cancelled(ptr %task)
  %t399 = icmp ne i64 %t398, 0
  br i1 %t399, label %cancelled.171, label %cont.172
cancelled.171:
  call void @veles_cleanup_pop(ptr %a340)
  %t400 = load ptr, ptr %a339
  call void @veles_scope_cancel(ptr %t400)
  br label %abandon.wait.173
abandon.wait.173:
  %t401 = call i64 @veles_scope_wait(ptr %task, ptr %t400)
  %t402 = icmp ne i64 %t401, 0
  br i1 %t402, label %abandon.done.174, label %abandon.susp.175
abandon.susp.175:
  %t403 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t403, label %coro.suspend [ i8 0, label %resume.176 i8 1, label %coro.cleanup ]
resume.176:
  br label %abandon.wait.173
abandon.done.174:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.172:
  %t404 = load ptr, ptr %a339
  %t405 = call ptr @veles_scope_failed(ptr %t404)
  %t406 = icmp ne ptr %t405, null
  br i1 %t406, label %scope.abort.177, label %scope.ok.178
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
  %t407 = load volatile i32, ptr @veles_stop_requested, align 4
  %t408 = icmp ne i32 %t407, 0
  br i1 %t408, label %safepoint.180, label %safepoint.on.181, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.180:
  call void @veles_gc_park()
  br label %safepoint.on.181
safepoint.on.181:
  br label %loop.cond.152
loop.end.154:
  br label %scope.wait.149
scope.wait.149:
  %t409 = load ptr, ptr %a339
  %t410 = call i64 @veles_scope_wait(ptr %task, ptr %t409)
  %t411 = icmp ne i64 %t410, 0
  br i1 %t411, label %scope.done.150, label %scope.susp.151
scope.susp.151:
  %t412 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t412, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  %t413 = call i64 @veles_task_cancelled(ptr %task)
  %t414 = icmp ne i64 %t413, 0
  br i1 %t414, label %cancelled.183, label %cont.184
cancelled.183:
  call void @veles_cleanup_pop(ptr %a340)
  %t415 = load ptr, ptr %a339
  call void @veles_scope_cancel(ptr %t415)
  br label %abandon.wait.185
abandon.wait.185:
  %t416 = call i64 @veles_scope_wait(ptr %task, ptr %t415)
  %t417 = icmp ne i64 %t416, 0
  br i1 %t417, label %abandon.done.186, label %abandon.susp.187
abandon.susp.187:
  %t418 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t418, label %coro.suspend [ i8 0, label %resume.188 i8 1, label %coro.cleanup ]
resume.188:
  br label %abandon.wait.185
abandon.done.186:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.184:
  br label %scope.wait.149
scope.done.150:
  call void @veles_cleanup_pop(ptr %a340)
  %t419 = load ptr, ptr %a339
  %t420 = call ptr @veles_scope_failed(ptr %t419)
  %t421 = icmp ne ptr %t420, null
  br i1 %t421, label %scope.check.189, label %scope.after.190
scope.check.189:
  %t422 = call i64 @veles_task_panicked(ptr %t420)
  %t423 = icmp ne i64 %t422, 0
  br i1 %t423, label %scope.repanic.191, label %scope.errors.192
scope.repanic.191:
  call void @veles_task_repanic(ptr %t420)
  unreachable
scope.errors.192:
  br label %scope.after.190
scope.after.190:
  %t424 = load i64, ptr %a337
  %t426 = call i64 @veles_i64_format(ptr %a425, i64 %t424)
  %t427 = insertvalue %str undef, ptr %a425, 0
  %t428 = insertvalue %str %t427, i64 %t426, 1
  %t430 = extractvalue %str { ptr @.str.18, i64 7 }, 0
  %t431 = extractvalue %str { ptr @.str.18, i64 7 }, 1
  %t432 = extractvalue %str %t428, 0
  %t433 = extractvalue %str %t428, 1
  call void @veles_string_concat(ptr %a429, ptr %t430, i64 %t431, ptr %t432, i64 %t433)
  %t434 = load %str, ptr %a429
  call void @v_std.io.println(%str %t434)
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
  %t6 = extractvalue %str { ptr @.str.19, i64 7 }, 0
  %t7 = extractvalue %str { ptr @.str.19, i64 7 }, 1
  %t8 = extractvalue %str %t4, 0
  %t9 = extractvalue %str %t4, 1
  call void @veles_string_concat(ptr %a5, ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  %t10 = load %str, ptr %a5
  call void @v_std.io.println(%str %t10)
  ret void
}

define internal void @entry.v_main.slow(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { i64 }, ptr %args, i32 0, i32 0
  %t2 = load i64, ptr %t1
  %t3 = call ptr @v_main.slow(ptr %task, i64 %t2)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

define internal void @entry.v_main.produce(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { ptr }, ptr %args, i32 0, i32 0
  %t2 = load ptr, ptr %t1
  %t3 = call ptr @v_main.produce(ptr %task, ptr %t2)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = call ptr @v_main.main(ptr %task)
  call void @veles_task_started(ptr %task, ptr %t1)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:12:9\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:13:10\00"
@.str.5 = private unnamed_addr constant [3 x i8] c"db\00"
@.str.6 = private unnamed_addr constant [2 x i8] c" \00"
@.str.7 = private unnamed_addr constant [13 x i8] c"main.vs:35:7\00"
@.str.8 = private unnamed_addr constant [5 x i8] c"sum \00"
@.str.9 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.10 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.11 = private unnamed_addr constant [2 x i8] c"c\00"
@.str.12 = private unnamed_addr constant [2 x i8] c"-\00"
@.str.13 = private unnamed_addr constant [14 x i8] c"main.vs:51:13\00"
@.str.14 = private unnamed_addr constant [13 x i8] c"main.vs:52:7\00"
@.str.15 = private unnamed_addr constant [7 x i8] c"slept \00"
@.str.16 = private unnamed_addr constant [14 x i8] c"main.vs:64:22\00"
@.str.17 = private unnamed_addr constant [14 x i8] c"main.vs:64:44\00"
@.str.18 = private unnamed_addr constant [8 x i8] c"polled \00"
@.str.19 = private unnamed_addr constant [8 x i8] c"closed \00"
