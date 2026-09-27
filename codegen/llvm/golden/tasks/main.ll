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
  %a36 = alloca %str
  %a42 = alloca %str
  %a61 = alloca %str
  %a67 = alloca %str
  %a85 = alloca ptr
  %a86 = alloca i64
  %a88 = alloca ptr
  %a89 = alloca { ptr, ptr, ptr }
  %a95 = alloca ptr
  %a97 = alloca i64
  %a114 = alloca i64
  %a119 = alloca i64
  %a148 = alloca %str
  %a154 = alloca %str
  %a157 = alloca ptr
  %a159 = alloca %str
  %a162 = alloca %str
  %a168 = alloca %str
  %a171 = alloca %str
  %a174 = alloca %str
  %a180 = alloca %str
  %a186 = alloca %str
  %a189 = alloca %str
  %a192 = alloca %str
  %a198 = alloca %str
  %a204 = alloca %str
  %a207 = alloca %str
  %a213 = alloca %str
  %a222 = alloca %str
  %a228 = alloca %str
  %a231 = alloca %str
  %a237 = alloca %str
  %a246 = alloca %str
  %a252 = alloca %str
  %a255 = alloca %str
  %a261 = alloca %str
  %a270 = alloca %str
  %a273 = alloca ptr
  %a274 = alloca i64
  %a276 = alloca ptr
  %a277 = alloca { ptr, ptr, ptr }
  %a283 = alloca ptr
  %a323 = alloca i64
  %a340 = alloca i64
  %a371 = alloca %str
  %a377 = alloca %str
  %a380 = alloca ptr
  %a381 = alloca i64
  %a383 = alloca ptr
  %a384 = alloca { ptr, ptr, ptr }
  %a390 = alloca ptr
  %a394 = alloca i64
  %a400 = alloca { i1, i64 }
  %a469 = alloca %str
  %a475 = alloca %str
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
  call void @veles_i64_to_string(ptr %a36, i64 %t35)
  %t37 = load %str, ptr %a36
  %t38 = extractvalue %str %t37, 0
  %t39 = extractvalue %str %t37, 1
  %t40 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t41 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a42, ptr %t38, i64 %t39, ptr %t40, i64 %t41)
  %t43 = load %str, ptr %a42
  %t44 = load ptr, ptr %a18
  br label %await.19
await.19:
  %t45 = call i64 @veles_task_await(ptr %task, ptr %t44)
  %t46 = icmp ne i64 %t45, 0
  br i1 %t46, label %await.got.20, label %await.susp.21
await.susp.21:
  %t47 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t47, label %coro.suspend [ i8 0, label %resume.22 i8 1, label %coro.cleanup ]
resume.22:
  %t48 = call i64 @veles_task_cancelled(ptr %task)
  %t49 = icmp ne i64 %t48, 0
  br i1 %t49, label %cancelled.23, label %cont.24
cancelled.23:
  call void @veles_cleanup_pop(ptr %a6)
  %t50 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t50)
  br label %abandon.wait.25
abandon.wait.25:
  %t51 = call i64 @veles_scope_wait(ptr %task, ptr %t50)
  %t52 = icmp ne i64 %t51, 0
  br i1 %t52, label %abandon.done.26, label %abandon.susp.27
abandon.susp.27:
  %t53 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t53, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  br label %abandon.wait.25
abandon.done.26:
  call void @veles_cleanup_pop(ptr %a3)
  call void @v_main.Closeable.Resource.close(ptr %a2)
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
  %t57 = call i64 @veles_task_panicked(ptr %t44)
  %t58 = icmp ne i64 %t57, 0
  br i1 %t58, label %await.repanic.32, label %await.fine.33
await.repanic.32:
  call void @veles_task_repanic(ptr %t44)
  unreachable
await.fine.33:
  %t59 = call ptr @veles_task_result(ptr %t44)
  %t60 = load i64, ptr %t59
  call void @veles_i64_to_string(ptr %a61, i64 %t60)
  %t62 = load %str, ptr %a61
  %t63 = extractvalue %str %t43, 0
  %t64 = extractvalue %str %t43, 1
  %t65 = extractvalue %str %t62, 0
  %t66 = extractvalue %str %t62, 1
  call void @veles_string_concat(ptr %a67, ptr %t63, i64 %t64, ptr %t65, i64 %t66)
  %t68 = load %str, ptr %a67
  call void @v_std.io.println(%str %t68)
  br label %scope.wait.1
scope.wait.1:
  %t69 = load ptr, ptr %a5
  %t70 = call i64 @veles_scope_wait(ptr %task, ptr %t69)
  %t71 = icmp ne i64 %t70, 0
  br i1 %t71, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  %t72 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t72, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t73 = call i64 @veles_task_cancelled(ptr %task)
  %t74 = icmp ne i64 %t73, 0
  br i1 %t74, label %cancelled.35, label %cont.36
cancelled.35:
  call void @veles_cleanup_pop(ptr %a6)
  %t75 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t75)
  br label %abandon.wait.37
abandon.wait.37:
  %t76 = call i64 @veles_scope_wait(ptr %task, ptr %t75)
  %t77 = icmp ne i64 %t76, 0
  br i1 %t77, label %abandon.done.38, label %abandon.susp.39
abandon.susp.39:
  %t78 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t78, label %coro.suspend [ i8 0, label %resume.40 i8 1, label %coro.cleanup ]
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
  %t79 = load ptr, ptr %a5
  %t80 = call ptr @veles_scope_failed(ptr %t79)
  %t81 = icmp ne ptr %t80, null
  br i1 %t81, label %scope.check.41, label %scope.after.42
scope.check.41:
  %t82 = call i64 @veles_task_panicked(ptr %t80)
  %t83 = icmp ne i64 %t82, 0
  br i1 %t83, label %scope.repanic.43, label %scope.errors.44
scope.repanic.43:
  call void @veles_task_repanic(ptr %t80)
  unreachable
scope.errors.44:
  br label %scope.after.42
scope.after.42:
  call void @veles_cleanup_pop(ptr %a3)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  %t84 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t84, ptr %a85
  store i64 0, ptr %a86
  %t87 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t87, ptr %a88
  call void @veles_cleanup_push(ptr %a89, ptr @scope.cancel.thunk, ptr %a88)
  %t90 = load ptr, ptr %a88
  %t91 = call ptr @veles_task_launch(ptr %t90, i64 0)
  %t92 = load ptr, ptr %a85
  %t93 = call ptr @veles_alloc_words(i64 16)
  %t94 = getelementptr inbounds { ptr }, ptr %t93, i32 0, i32 0
  store ptr %t92, ptr %t94
  call void @veles_task_spawn(ptr %t91, ptr @entry.v_main.produce, ptr %t93)
  store ptr %t91, ptr %a95
  br label %loop.cond.48
loop.cond.48:
  br label %loop.body.51
loop.body.51:
  %t96 = load ptr, ptr %a85
  store i64 zeroinitializer, ptr %a97
  br label %recv.52
recv.52:
  %t98 = call i64 @veles_chan_recv(ptr %task, ptr %t96, ptr %a97)
  %t99 = icmp ne i64 %t98, 0
  br i1 %t99, label %recv.done.53, label %recv.susp.54
recv.susp.54:
  %t100 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t100, label %coro.suspend [ i8 0, label %resume.55 i8 1, label %coro.cleanup ]
resume.55:
  %t101 = call i64 @veles_task_cancelled(ptr %task)
  %t102 = icmp ne i64 %t101, 0
  br i1 %t102, label %cancelled.56, label %cont.57
cancelled.56:
  call void @veles_cleanup_pop(ptr %a89)
  %t103 = load ptr, ptr %a88
  call void @veles_scope_cancel(ptr %t103)
  br label %abandon.wait.58
abandon.wait.58:
  %t104 = call i64 @veles_scope_wait(ptr %task, ptr %t103)
  %t105 = icmp ne i64 %t104, 0
  br i1 %t105, label %abandon.done.59, label %abandon.susp.60
abandon.susp.60:
  %t106 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t106, label %coro.suspend [ i8 0, label %resume.61 i8 1, label %coro.cleanup ]
resume.61:
  br label %abandon.wait.58
abandon.done.59:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.57:
  %t107 = load ptr, ptr %a88
  %t108 = call ptr @veles_scope_failed(ptr %t107)
  %t109 = icmp ne ptr %t108, null
  br i1 %t109, label %scope.abort.62, label %scope.ok.63
scope.ok.63:
  br label %cont.64
scope.abort.62:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.45
cont.64:
  br label %recv.52
recv.done.53:
  %t110 = icmp eq i64 %t98, 1
  %t111 = load i64, ptr %a97
  %t112 = insertvalue { i1, i64 } undef, i1 %t110, 0
  %t113 = insertvalue { i1, i64 } %t112, i64 %t111, 1
  %t116 = extractvalue { i1, i64 } %t113, 0
  %t115 = xor i1 %t116, true
  br i1 %t115, label %elvis.default.65, label %elvis.some.66
elvis.some.66:
  %t117 = extractvalue { i1, i64 } %t113, 1
  store i64 %t117, ptr %a114
  br label %elvis.end.67
elvis.default.65:
  br label %loop.end.50
elvis.end.67:
  %t118 = load i64, ptr %a114
  store i64 %t118, ptr %a119
  %t120 = load i64, ptr %a86
  %t121 = load i64, ptr %a119
  %t123 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t120, i64 %t121)
  %t124 = extractvalue { i64, i1 } %t123, 0
  %t125 = extractvalue { i64, i1 } %t123, 1
  br i1 %t125, label %overflow.68, label %arith.ok.69
overflow.68:
  %t126 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t127 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t128 = extractvalue %str { ptr @.str.7, i64 12 }, 0
  %t129 = extractvalue %str { ptr @.str.7, i64 12 }, 1
  call void @veles_panic_at(ptr %t126, i64 %t127, ptr %t128, i64 %t129)
  unreachable
arith.ok.69:
  store i64 %t124, ptr %a86
  br label %loop.post.49
loop.post.49:
  %t130 = load volatile i32, ptr @veles_stop_requested, align 4
  %t131 = icmp ne i32 %t130, 0
  br i1 %t131, label %safepoint.70, label %safepoint.on.71, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.70:
  call void @veles_gc_park()
  br label %safepoint.on.71
safepoint.on.71:
  br label %loop.cond.48
loop.end.50:
  br label %scope.wait.45
scope.wait.45:
  %t132 = load ptr, ptr %a88
  %t133 = call i64 @veles_scope_wait(ptr %task, ptr %t132)
  %t134 = icmp ne i64 %t133, 0
  br i1 %t134, label %scope.done.46, label %scope.susp.47
scope.susp.47:
  %t135 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t135, label %coro.suspend [ i8 0, label %resume.72 i8 1, label %coro.cleanup ]
resume.72:
  %t136 = call i64 @veles_task_cancelled(ptr %task)
  %t137 = icmp ne i64 %t136, 0
  br i1 %t137, label %cancelled.73, label %cont.74
cancelled.73:
  call void @veles_cleanup_pop(ptr %a89)
  %t138 = load ptr, ptr %a88
  call void @veles_scope_cancel(ptr %t138)
  br label %abandon.wait.75
abandon.wait.75:
  %t139 = call i64 @veles_scope_wait(ptr %task, ptr %t138)
  %t140 = icmp ne i64 %t139, 0
  br i1 %t140, label %abandon.done.76, label %abandon.susp.77
abandon.susp.77:
  %t141 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t141, label %coro.suspend [ i8 0, label %resume.78 i8 1, label %coro.cleanup ]
resume.78:
  br label %abandon.wait.75
abandon.done.76:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.74:
  br label %scope.wait.45
scope.done.46:
  call void @veles_cleanup_pop(ptr %a89)
  %t142 = load ptr, ptr %a88
  %t143 = call ptr @veles_scope_failed(ptr %t142)
  %t144 = icmp ne ptr %t143, null
  br i1 %t144, label %scope.check.79, label %scope.after.80
scope.check.79:
  %t145 = call i64 @veles_task_panicked(ptr %t143)
  %t146 = icmp ne i64 %t145, 0
  br i1 %t146, label %scope.repanic.81, label %scope.errors.82
scope.repanic.81:
  call void @veles_task_repanic(ptr %t143)
  unreachable
scope.errors.82:
  br label %scope.after.80
scope.after.80:
  %t147 = load i64, ptr %a86
  call void @veles_i64_to_string(ptr %a148, i64 %t147)
  %t149 = load %str, ptr %a148
  %t150 = extractvalue %str { ptr @.str.8, i64 4 }, 0
  %t151 = extractvalue %str { ptr @.str.8, i64 4 }, 1
  %t152 = extractvalue %str %t149, 0
  %t153 = extractvalue %str %t149, 1
  call void @veles_string_concat(ptr %a154, ptr %t150, i64 %t151, ptr %t152, i64 %t153)
  %t155 = load %str, ptr %a154
  call void @v_std.io.println(%str %t155)
  %t156 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t156, ptr %a157
  %t158 = load ptr, ptr %a157
  store %str { ptr @.str.9, i64 1 }, ptr %a159
  %t160 = call i64 @veles_chan_try_send(ptr %t158, ptr %a159)
  %t161 = icmp ne i64 %t160, 0
  call void @veles_bool_to_string(ptr %a162, i1 zeroext %t161)
  %t163 = load %str, ptr %a162
  %t164 = extractvalue %str %t163, 0
  %t165 = extractvalue %str %t163, 1
  %t166 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t167 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a168, ptr %t164, i64 %t165, ptr %t166, i64 %t167)
  %t169 = load %str, ptr %a168
  %t170 = load ptr, ptr %a157
  store %str { ptr @.str.10, i64 1 }, ptr %a171
  %t172 = call i64 @veles_chan_try_send(ptr %t170, ptr %a171)
  %t173 = icmp ne i64 %t172, 0
  call void @veles_bool_to_string(ptr %a174, i1 zeroext %t173)
  %t175 = load %str, ptr %a174
  %t176 = extractvalue %str %t169, 0
  %t177 = extractvalue %str %t169, 1
  %t178 = extractvalue %str %t175, 0
  %t179 = extractvalue %str %t175, 1
  call void @veles_string_concat(ptr %a180, ptr %t176, i64 %t177, ptr %t178, i64 %t179)
  %t181 = load %str, ptr %a180
  %t182 = extractvalue %str %t181, 0
  %t183 = extractvalue %str %t181, 1
  %t184 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t185 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a186, ptr %t182, i64 %t183, ptr %t184, i64 %t185)
  %t187 = load %str, ptr %a186
  %t188 = load ptr, ptr %a157
  store %str { ptr @.str.11, i64 1 }, ptr %a189
  %t190 = call i64 @veles_chan_try_send(ptr %t188, ptr %a189)
  %t191 = icmp ne i64 %t190, 0
  call void @veles_bool_to_string(ptr %a192, i1 zeroext %t191)
  %t193 = load %str, ptr %a192
  %t194 = extractvalue %str %t187, 0
  %t195 = extractvalue %str %t187, 1
  %t196 = extractvalue %str %t193, 0
  %t197 = extractvalue %str %t193, 1
  call void @veles_string_concat(ptr %a198, ptr %t194, i64 %t195, ptr %t196, i64 %t197)
  %t199 = load %str, ptr %a198
  %t200 = extractvalue %str %t199, 0
  %t201 = extractvalue %str %t199, 1
  %t202 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t203 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a204, ptr %t200, i64 %t201, ptr %t202, i64 %t203)
  %t205 = load %str, ptr %a204
  %t206 = load ptr, ptr %a157
  store %str zeroinitializer, ptr %a207
  %t208 = call i64 @veles_chan_try_recv(ptr %t206, ptr %a207)
  %t209 = icmp ne i64 %t208, 0
  %t210 = load %str, ptr %a207
  %t211 = insertvalue { i1, %str } undef, i1 %t209, 0
  %t212 = insertvalue { i1, %str } %t211, %str %t210, 1
  %t215 = extractvalue { i1, %str } %t212, 0
  %t214 = xor i1 %t215, true
  br i1 %t214, label %elvis.default.83, label %elvis.some.84
elvis.some.84:
  %t216 = extractvalue { i1, %str } %t212, 1
  store %str %t216, ptr %a213
  br label %elvis.end.85
elvis.default.83:
  store %str { ptr @.str.12, i64 1 }, ptr %a213
  br label %elvis.end.85
elvis.end.85:
  %t217 = load %str, ptr %a213
  %t218 = extractvalue %str %t205, 0
  %t219 = extractvalue %str %t205, 1
  %t220 = extractvalue %str %t217, 0
  %t221 = extractvalue %str %t217, 1
  call void @veles_string_concat(ptr %a222, ptr %t218, i64 %t219, ptr %t220, i64 %t221)
  %t223 = load %str, ptr %a222
  %t224 = extractvalue %str %t223, 0
  %t225 = extractvalue %str %t223, 1
  %t226 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t227 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a228, ptr %t224, i64 %t225, ptr %t226, i64 %t227)
  %t229 = load %str, ptr %a228
  %t230 = load ptr, ptr %a157
  store %str zeroinitializer, ptr %a231
  %t232 = call i64 @veles_chan_try_recv(ptr %t230, ptr %a231)
  %t233 = icmp ne i64 %t232, 0
  %t234 = load %str, ptr %a231
  %t235 = insertvalue { i1, %str } undef, i1 %t233, 0
  %t236 = insertvalue { i1, %str } %t235, %str %t234, 1
  %t239 = extractvalue { i1, %str } %t236, 0
  %t238 = xor i1 %t239, true
  br i1 %t238, label %elvis.default.86, label %elvis.some.87
elvis.some.87:
  %t240 = extractvalue { i1, %str } %t236, 1
  store %str %t240, ptr %a237
  br label %elvis.end.88
elvis.default.86:
  store %str { ptr @.str.12, i64 1 }, ptr %a237
  br label %elvis.end.88
elvis.end.88:
  %t241 = load %str, ptr %a237
  %t242 = extractvalue %str %t229, 0
  %t243 = extractvalue %str %t229, 1
  %t244 = extractvalue %str %t241, 0
  %t245 = extractvalue %str %t241, 1
  call void @veles_string_concat(ptr %a246, ptr %t242, i64 %t243, ptr %t244, i64 %t245)
  %t247 = load %str, ptr %a246
  %t248 = extractvalue %str %t247, 0
  %t249 = extractvalue %str %t247, 1
  %t250 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t251 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a252, ptr %t248, i64 %t249, ptr %t250, i64 %t251)
  %t253 = load %str, ptr %a252
  %t254 = load ptr, ptr %a157
  store %str zeroinitializer, ptr %a255
  %t256 = call i64 @veles_chan_try_recv(ptr %t254, ptr %a255)
  %t257 = icmp ne i64 %t256, 0
  %t258 = load %str, ptr %a255
  %t259 = insertvalue { i1, %str } undef, i1 %t257, 0
  %t260 = insertvalue { i1, %str } %t259, %str %t258, 1
  %t263 = extractvalue { i1, %str } %t260, 0
  %t262 = xor i1 %t263, true
  br i1 %t262, label %elvis.default.89, label %elvis.some.90
elvis.some.90:
  %t264 = extractvalue { i1, %str } %t260, 1
  store %str %t264, ptr %a261
  br label %elvis.end.91
elvis.default.89:
  store %str { ptr @.str.12, i64 1 }, ptr %a261
  br label %elvis.end.91
elvis.end.91:
  %t265 = load %str, ptr %a261
  %t266 = extractvalue %str %t253, 0
  %t267 = extractvalue %str %t253, 1
  %t268 = extractvalue %str %t265, 0
  %t269 = extractvalue %str %t265, 1
  call void @veles_string_concat(ptr %a270, ptr %t266, i64 %t267, ptr %t268, i64 %t269)
  %t271 = load %str, ptr %a270
  call void @v_std.io.println(%str %t271)
  %t272 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t272, ptr %a273
  store i64 0, ptr %a274
  %t275 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t275, ptr %a276
  call void @veles_cleanup_push(ptr %a277, ptr @scope.cancel.thunk, ptr %a276)
  %t278 = load ptr, ptr %a276
  %t279 = call ptr @veles_task_launch(ptr %t278, i64 0)
  %t280 = load ptr, ptr %a273
  %t281 = call ptr @veles_alloc_words(i64 16)
  %t282 = getelementptr inbounds { ptr }, ptr %t281, i32 0, i32 0
  store ptr %t280, ptr %t282
  call void @veles_task_spawn(ptr %t279, ptr @entry.v_main.produce, ptr %t281)
  store ptr %t279, ptr %a283
  br label %loop.cond.95
loop.cond.95:
  %t284 = load i64, ptr %a274
  %t285 = icmp slt i64 %t284, 6
  br i1 %t285, label %loop.body.98, label %loop.end.97
loop.body.98:
  %t286 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  %t287 = extractvalue %S.std.prelude.Duration %t286, 0
  %t289 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t287, i64 999999)
  %t290 = extractvalue { i64, i1 } %t289, 0
  %t291 = extractvalue { i64, i1 } %t289, 1
  br i1 %t291, label %overflow.99, label %arith.ok.100
overflow.99:
  %t292 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t293 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t294 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t295 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t292, i64 %t293, ptr %t294, i64 %t295)
  unreachable
arith.ok.100:
  %t297 = icmp eq i64 1000000, 0
  br i1 %t297, label %divzero.101, label %div.ok.102
divzero.101:
  %t298 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t299 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t300 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t301 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t298, i64 %t299, ptr %t300, i64 %t301)
  unreachable
div.ok.102:
  %t302 = icmp eq i64 %t290, -9223372036854775808
  %t303 = icmp eq i64 1000000, -1
  %t304 = and i1 %t302, %t303
  br i1 %t304, label %divof.103, label %div.ok.104
divof.103:
  %t305 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t306 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t307 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t308 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t305, i64 %t306, ptr %t307, i64 %t308)
  unreachable
div.ok.104:
  %t296 = sdiv i64 %t290, 1000000
  br label %sleep.105
sleep.105:
  %t309 = call i64 @veles_task_sleep(ptr %task, i64 %t296)
  %t310 = icmp ne i64 %t309, 0
  br i1 %t310, label %sleep.done.106, label %sleep.susp.107
sleep.susp.107:
  %t311 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t311, label %coro.suspend [ i8 0, label %resume.108 i8 1, label %coro.cleanup ]
resume.108:
  %t312 = call i64 @veles_task_cancelled(ptr %task)
  %t313 = icmp ne i64 %t312, 0
  br i1 %t313, label %cancelled.109, label %cont.110
cancelled.109:
  call void @veles_cleanup_pop(ptr %a277)
  %t314 = load ptr, ptr %a276
  call void @veles_scope_cancel(ptr %t314)
  br label %abandon.wait.111
abandon.wait.111:
  %t315 = call i64 @veles_scope_wait(ptr %task, ptr %t314)
  %t316 = icmp ne i64 %t315, 0
  br i1 %t316, label %abandon.done.112, label %abandon.susp.113
abandon.susp.113:
  %t317 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t317, label %coro.suspend [ i8 0, label %resume.114 i8 1, label %coro.cleanup ]
resume.114:
  br label %abandon.wait.111
abandon.done.112:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.110:
  %t318 = load ptr, ptr %a276
  %t319 = call ptr @veles_scope_failed(ptr %t318)
  %t320 = icmp ne ptr %t319, null
  br i1 %t320, label %scope.abort.115, label %scope.ok.116
scope.ok.116:
  br label %cont.117
scope.abort.115:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.92
cont.117:
  br label %sleep.105
sleep.done.106:
  %t321 = load i64, ptr %a274
  %t322 = load ptr, ptr %a273
  store i64 zeroinitializer, ptr %a323
  br label %recv.118
recv.118:
  %t324 = call i64 @veles_chan_recv(ptr %task, ptr %t322, ptr %a323)
  %t325 = icmp ne i64 %t324, 0
  br i1 %t325, label %recv.done.119, label %recv.susp.120
recv.susp.120:
  %t326 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t326, label %coro.suspend [ i8 0, label %resume.121 i8 1, label %coro.cleanup ]
resume.121:
  %t327 = call i64 @veles_task_cancelled(ptr %task)
  %t328 = icmp ne i64 %t327, 0
  br i1 %t328, label %cancelled.122, label %cont.123
cancelled.122:
  call void @veles_cleanup_pop(ptr %a277)
  %t329 = load ptr, ptr %a276
  call void @veles_scope_cancel(ptr %t329)
  br label %abandon.wait.124
abandon.wait.124:
  %t330 = call i64 @veles_scope_wait(ptr %task, ptr %t329)
  %t331 = icmp ne i64 %t330, 0
  br i1 %t331, label %abandon.done.125, label %abandon.susp.126
abandon.susp.126:
  %t332 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t332, label %coro.suspend [ i8 0, label %resume.127 i8 1, label %coro.cleanup ]
resume.127:
  br label %abandon.wait.124
abandon.done.125:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.123:
  %t333 = load ptr, ptr %a276
  %t334 = call ptr @veles_scope_failed(ptr %t333)
  %t335 = icmp ne ptr %t334, null
  br i1 %t335, label %scope.abort.128, label %scope.ok.129
scope.ok.129:
  br label %cont.130
scope.abort.128:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.92
cont.130:
  br label %recv.118
recv.done.119:
  %t336 = icmp eq i64 %t324, 1
  %t337 = load i64, ptr %a323
  %t338 = insertvalue { i1, i64 } undef, i1 %t336, 0
  %t339 = insertvalue { i1, i64 } %t338, i64 %t337, 1
  %t342 = extractvalue { i1, i64 } %t339, 0
  %t341 = xor i1 %t342, true
  br i1 %t341, label %elvis.default.131, label %elvis.some.132
elvis.some.132:
  %t343 = extractvalue { i1, i64 } %t339, 1
  store i64 %t343, ptr %a340
  br label %elvis.end.133
elvis.default.131:
  store i64 0, ptr %a340
  br label %elvis.end.133
elvis.end.133:
  %t344 = load i64, ptr %a340
  %t346 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t321, i64 %t344)
  %t347 = extractvalue { i64, i1 } %t346, 0
  %t348 = extractvalue { i64, i1 } %t346, 1
  br i1 %t348, label %overflow.134, label %arith.ok.135
overflow.134:
  %t349 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t350 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t351 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t352 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_panic_at(ptr %t349, i64 %t350, ptr %t351, i64 %t352)
  unreachable
arith.ok.135:
  store i64 %t347, ptr %a274
  br label %loop.post.96
loop.post.96:
  %t353 = load volatile i32, ptr @veles_stop_requested, align 4
  %t354 = icmp ne i32 %t353, 0
  br i1 %t354, label %safepoint.136, label %safepoint.on.137, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.136:
  call void @veles_gc_park()
  br label %safepoint.on.137
safepoint.on.137:
  br label %loop.cond.95
loop.end.97:
  br label %scope.wait.92
scope.wait.92:
  %t355 = load ptr, ptr %a276
  %t356 = call i64 @veles_scope_wait(ptr %task, ptr %t355)
  %t357 = icmp ne i64 %t356, 0
  br i1 %t357, label %scope.done.93, label %scope.susp.94
scope.susp.94:
  %t358 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t358, label %coro.suspend [ i8 0, label %resume.138 i8 1, label %coro.cleanup ]
resume.138:
  %t359 = call i64 @veles_task_cancelled(ptr %task)
  %t360 = icmp ne i64 %t359, 0
  br i1 %t360, label %cancelled.139, label %cont.140
cancelled.139:
  call void @veles_cleanup_pop(ptr %a277)
  %t361 = load ptr, ptr %a276
  call void @veles_scope_cancel(ptr %t361)
  br label %abandon.wait.141
abandon.wait.141:
  %t362 = call i64 @veles_scope_wait(ptr %task, ptr %t361)
  %t363 = icmp ne i64 %t362, 0
  br i1 %t363, label %abandon.done.142, label %abandon.susp.143
abandon.susp.143:
  %t364 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t364, label %coro.suspend [ i8 0, label %resume.144 i8 1, label %coro.cleanup ]
resume.144:
  br label %abandon.wait.141
abandon.done.142:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.140:
  br label %scope.wait.92
scope.done.93:
  call void @veles_cleanup_pop(ptr %a277)
  %t365 = load ptr, ptr %a276
  %t366 = call ptr @veles_scope_failed(ptr %t365)
  %t367 = icmp ne ptr %t366, null
  br i1 %t367, label %scope.check.145, label %scope.after.146
scope.check.145:
  %t368 = call i64 @veles_task_panicked(ptr %t366)
  %t369 = icmp ne i64 %t368, 0
  br i1 %t369, label %scope.repanic.147, label %scope.errors.148
scope.repanic.147:
  call void @veles_task_repanic(ptr %t366)
  unreachable
scope.errors.148:
  br label %scope.after.146
scope.after.146:
  %t370 = load i64, ptr %a274
  call void @veles_i64_to_string(ptr %a371, i64 %t370)
  %t372 = load %str, ptr %a371
  %t373 = extractvalue %str { ptr @.str.15, i64 6 }, 0
  %t374 = extractvalue %str { ptr @.str.15, i64 6 }, 1
  %t375 = extractvalue %str %t372, 0
  %t376 = extractvalue %str %t372, 1
  call void @veles_string_concat(ptr %a377, ptr %t373, i64 %t374, ptr %t375, i64 %t376)
  %t378 = load %str, ptr %a377
  call void @v_std.io.println(%str %t378)
  %t379 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t379, ptr %a380
  store i64 0, ptr %a381
  %t382 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t382, ptr %a383
  call void @veles_cleanup_push(ptr %a384, ptr @scope.cancel.thunk, ptr %a383)
  %t385 = load ptr, ptr %a383
  %t386 = call ptr @veles_task_launch(ptr %t385, i64 0)
  %t387 = load ptr, ptr %a380
  %t388 = call ptr @veles_alloc_words(i64 16)
  %t389 = getelementptr inbounds { ptr }, ptr %t388, i32 0, i32 0
  store ptr %t387, ptr %t389
  call void @veles_task_spawn(ptr %t386, ptr @entry.v_main.produce, ptr %t388)
  store ptr %t386, ptr %a390
  br label %loop.cond.152
loop.cond.152:
  %t391 = load i64, ptr %a381
  %t392 = icmp slt i64 %t391, 6
  br i1 %t392, label %loop.body.155, label %loop.end.154
loop.body.155:
  %t393 = load ptr, ptr %a380
  store i64 zeroinitializer, ptr %a394
  %t395 = call i64 @veles_chan_try_recv(ptr %t393, ptr %a394)
  %t396 = icmp ne i64 %t395, 0
  %t397 = load i64, ptr %a394
  %t398 = insertvalue { i1, i64 } undef, i1 %t396, 0
  %t399 = insertvalue { i1, i64 } %t398, i64 %t397, 1
  store { i1, i64 } %t399, ptr %a400
  %t401 = load { i1, i64 }, ptr %a400
  %t403 = extractvalue { i1, i64 } %t401, 0
  %t402 = xor i1 %t403, true
  %t404 = xor i1 %t402, true
  br i1 %t404, label %if.then.156, label %if.else.158
if.then.156:
  %t405 = load i64, ptr %a381
  %t406 = load { i1, i64 }, ptr %a400
  %t407 = extractvalue { i1, i64 } %t406, 1
  %t409 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t405, i64 %t407)
  %t410 = extractvalue { i64, i1 } %t409, 0
  %t411 = extractvalue { i64, i1 } %t409, 1
  br i1 %t411, label %overflow.159, label %arith.ok.160
overflow.159:
  %t412 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t413 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t414 = extractvalue %str { ptr @.str.16, i64 13 }, 0
  %t415 = extractvalue %str { ptr @.str.16, i64 13 }, 1
  call void @veles_panic_at(ptr %t412, i64 %t413, ptr %t414, i64 %t415)
  unreachable
arith.ok.160:
  store i64 %t410, ptr %a381
  br label %if.end.157
if.else.158:
  %t416 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t417 = load i64, ptr %t416
  %t419 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t417, i64 999999)
  %t420 = extractvalue { i64, i1 } %t419, 0
  %t421 = extractvalue { i64, i1 } %t419, 1
  br i1 %t421, label %overflow.161, label %arith.ok.162
overflow.161:
  %t422 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t423 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t424 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t425 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t422, i64 %t423, ptr %t424, i64 %t425)
  unreachable
arith.ok.162:
  %t427 = icmp eq i64 1000000, 0
  br i1 %t427, label %divzero.163, label %div.ok.164
divzero.163:
  %t428 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t429 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t430 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t431 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t428, i64 %t429, ptr %t430, i64 %t431)
  unreachable
div.ok.164:
  %t432 = icmp eq i64 %t420, -9223372036854775808
  %t433 = icmp eq i64 1000000, -1
  %t434 = and i1 %t432, %t433
  br i1 %t434, label %divof.165, label %div.ok.166
divof.165:
  %t435 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t436 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t437 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t438 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t435, i64 %t436, ptr %t437, i64 %t438)
  unreachable
div.ok.166:
  %t426 = sdiv i64 %t420, 1000000
  br label %sleep.167
sleep.167:
  %t439 = call i64 @veles_task_sleep(ptr %task, i64 %t426)
  %t440 = icmp ne i64 %t439, 0
  br i1 %t440, label %sleep.done.168, label %sleep.susp.169
sleep.susp.169:
  %t441 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t441, label %coro.suspend [ i8 0, label %resume.170 i8 1, label %coro.cleanup ]
resume.170:
  %t442 = call i64 @veles_task_cancelled(ptr %task)
  %t443 = icmp ne i64 %t442, 0
  br i1 %t443, label %cancelled.171, label %cont.172
cancelled.171:
  call void @veles_cleanup_pop(ptr %a384)
  %t444 = load ptr, ptr %a383
  call void @veles_scope_cancel(ptr %t444)
  br label %abandon.wait.173
abandon.wait.173:
  %t445 = call i64 @veles_scope_wait(ptr %task, ptr %t444)
  %t446 = icmp ne i64 %t445, 0
  br i1 %t446, label %abandon.done.174, label %abandon.susp.175
abandon.susp.175:
  %t447 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t447, label %coro.suspend [ i8 0, label %resume.176 i8 1, label %coro.cleanup ]
resume.176:
  br label %abandon.wait.173
abandon.done.174:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.172:
  %t448 = load ptr, ptr %a383
  %t449 = call ptr @veles_scope_failed(ptr %t448)
  %t450 = icmp ne ptr %t449, null
  br i1 %t450, label %scope.abort.177, label %scope.ok.178
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
  %t451 = load volatile i32, ptr @veles_stop_requested, align 4
  %t452 = icmp ne i32 %t451, 0
  br i1 %t452, label %safepoint.180, label %safepoint.on.181, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.180:
  call void @veles_gc_park()
  br label %safepoint.on.181
safepoint.on.181:
  br label %loop.cond.152
loop.end.154:
  br label %scope.wait.149
scope.wait.149:
  %t453 = load ptr, ptr %a383
  %t454 = call i64 @veles_scope_wait(ptr %task, ptr %t453)
  %t455 = icmp ne i64 %t454, 0
  br i1 %t455, label %scope.done.150, label %scope.susp.151
scope.susp.151:
  %t456 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t456, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  %t457 = call i64 @veles_task_cancelled(ptr %task)
  %t458 = icmp ne i64 %t457, 0
  br i1 %t458, label %cancelled.183, label %cont.184
cancelled.183:
  call void @veles_cleanup_pop(ptr %a384)
  %t459 = load ptr, ptr %a383
  call void @veles_scope_cancel(ptr %t459)
  br label %abandon.wait.185
abandon.wait.185:
  %t460 = call i64 @veles_scope_wait(ptr %task, ptr %t459)
  %t461 = icmp ne i64 %t460, 0
  br i1 %t461, label %abandon.done.186, label %abandon.susp.187
abandon.susp.187:
  %t462 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t462, label %coro.suspend [ i8 0, label %resume.188 i8 1, label %coro.cleanup ]
resume.188:
  br label %abandon.wait.185
abandon.done.186:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.184:
  br label %scope.wait.149
scope.done.150:
  call void @veles_cleanup_pop(ptr %a384)
  %t463 = load ptr, ptr %a383
  %t464 = call ptr @veles_scope_failed(ptr %t463)
  %t465 = icmp ne ptr %t464, null
  br i1 %t465, label %scope.check.189, label %scope.after.190
scope.check.189:
  %t466 = call i64 @veles_task_panicked(ptr %t464)
  %t467 = icmp ne i64 %t466, 0
  br i1 %t467, label %scope.repanic.191, label %scope.errors.192
scope.repanic.191:
  call void @veles_task_repanic(ptr %t464)
  unreachable
scope.errors.192:
  br label %scope.after.190
scope.after.190:
  %t468 = load i64, ptr %a381
  call void @veles_i64_to_string(ptr %a469, i64 %t468)
  %t470 = load %str, ptr %a469
  %t471 = extractvalue %str { ptr @.str.18, i64 7 }, 0
  %t472 = extractvalue %str { ptr @.str.18, i64 7 }, 1
  %t473 = extractvalue %str %t470, 0
  %t474 = extractvalue %str %t470, 1
  call void @veles_string_concat(ptr %a475, ptr %t471, i64 %t472, ptr %t473, i64 %t474)
  %t476 = load %str, ptr %a475
  call void @v_std.io.println(%str %t476)
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
  %a9 = alloca %str
  store ptr %p0, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr inbounds %S.main.Resource, ptr %t2, i32 0, i32 0
  %t4 = load %str, ptr %t3
  %t5 = extractvalue %str { ptr @.str.19, i64 7 }, 0
  %t6 = extractvalue %str { ptr @.str.19, i64 7 }, 1
  %t7 = extractvalue %str %t4, 0
  %t8 = extractvalue %str %t4, 1
  call void @veles_string_concat(ptr %a9, ptr %t5, i64 %t6, ptr %t7, i64 %t8)
  %t10 = load %str, ptr %a9
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
