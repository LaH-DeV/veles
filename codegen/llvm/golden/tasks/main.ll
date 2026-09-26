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
  br label %loop.cond.1
loop.end.3:
  %t34 = load ptr, ptr %a1
  call void @veles_chan_close(ptr %t34)
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
  %a4 = alloca ptr
  %a9 = alloca ptr
  %a10 = alloca ptr
  %a15 = alloca ptr
  %a16 = alloca ptr
  %a34 = alloca %str
  %a40 = alloca %str
  %a59 = alloca %str
  %a65 = alloca %str
  %a83 = alloca ptr
  %a84 = alloca i64
  %a86 = alloca ptr
  %a92 = alloca ptr
  %a94 = alloca i64
  %a111 = alloca i64
  %a116 = alloca i64
  %a143 = alloca %str
  %a149 = alloca %str
  %a152 = alloca ptr
  %a154 = alloca %str
  %a157 = alloca %str
  %a163 = alloca %str
  %a166 = alloca %str
  %a169 = alloca %str
  %a175 = alloca %str
  %a181 = alloca %str
  %a184 = alloca %str
  %a187 = alloca %str
  %a193 = alloca %str
  %a199 = alloca %str
  %a202 = alloca %str
  %a208 = alloca %str
  %a217 = alloca %str
  %a223 = alloca %str
  %a226 = alloca %str
  %a232 = alloca %str
  %a241 = alloca %str
  %a247 = alloca %str
  %a250 = alloca %str
  %a256 = alloca %str
  %a265 = alloca %str
  %a268 = alloca ptr
  %a269 = alloca i64
  %a271 = alloca ptr
  %a277 = alloca ptr
  %a317 = alloca i64
  %a334 = alloca i64
  %a363 = alloca %str
  %a369 = alloca %str
  %a372 = alloca ptr
  %a373 = alloca i64
  %a375 = alloca ptr
  %a381 = alloca ptr
  %a385 = alloca i64
  %a391 = alloca { i1, i64 }
  %a458 = alloca %str
  %a464 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 0, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles_alloc_words(i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %t1 = insertvalue %S.main.Resource undef, %str { ptr @.str.5, i64 2 }, 0
  store %S.main.Resource %t1, ptr %a2
  call void @veles_cleanup_push(ptr @with.close.1, ptr %a2)
  %t3 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t3, ptr %a4
  call void @veles_cleanup_push(ptr @scope.cancel.thunk, ptr %a4)
  %t5 = load ptr, ptr %a4
  %t6 = call ptr @veles_task_launch(ptr %t5)
  %t7 = call ptr @veles_alloc_words(i64 16)
  %t8 = getelementptr inbounds { i64 }, ptr %t7, i32 0, i32 0
  store i64 4, ptr %t8
  call void @veles_task_start(ptr %t6, ptr @entry.v_main.slow, ptr %t7)
  store ptr %t6, ptr %a9
  store ptr %t6, ptr %a10
  %t11 = load ptr, ptr %a4
  %t12 = call ptr @veles_task_launch(ptr %t11)
  %t13 = call ptr @veles_alloc_words(i64 16)
  %t14 = getelementptr inbounds { i64 }, ptr %t13, i32 0, i32 0
  store i64 5, ptr %t14
  call void @veles_task_start(ptr %t12, ptr @entry.v_main.slow, ptr %t13)
  store ptr %t12, ptr %a15
  store ptr %t12, ptr %a16
  %t17 = load ptr, ptr %a10
  br label %await.4
await.4:
  %t18 = call i64 @veles_task_await(ptr %task, ptr %t17)
  %t19 = icmp ne i64 %t18, 0
  br i1 %t19, label %await.got.5, label %await.susp.6
await.susp.6:
  %t20 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t20, label %coro.suspend [ i8 0, label %resume.7 i8 1, label %coro.cleanup ]
resume.7:
  %t21 = call i64 @veles_task_cancelled(ptr %task)
  %t22 = icmp ne i64 %t21, 0
  br i1 %t22, label %cancelled.8, label %cont.9
cancelled.8:
  call void @veles_cleanup_pop()
  %t23 = load ptr, ptr %a4
  call void @veles_scope_cancel(ptr %t23)
  br label %abandon.wait.10
abandon.wait.10:
  %t24 = call i64 @veles_scope_wait(ptr %task, ptr %t23)
  %t25 = icmp ne i64 %t24, 0
  br i1 %t25, label %abandon.done.11, label %abandon.susp.12
abandon.susp.12:
  %t26 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t26, label %coro.suspend [ i8 0, label %resume.13 i8 1, label %coro.cleanup ]
resume.13:
  br label %abandon.wait.10
abandon.done.11:
  call void @veles_cleanup_pop()
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.9:
  %t27 = load ptr, ptr %a4
  %t28 = call ptr @veles_scope_failed(ptr %t27)
  %t29 = icmp ne ptr %t28, null
  br i1 %t29, label %scope.abort.14, label %scope.ok.15
scope.ok.15:
  br label %cont.16
scope.abort.14:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.1
cont.16:
  br label %await.4
await.got.5:
  %t30 = call i64 @veles_task_panicked(ptr %t17)
  %t31 = icmp ne i64 %t30, 0
  br i1 %t31, label %await.repanic.17, label %await.fine.18
await.repanic.17:
  call void @veles_task_repanic(ptr %t17)
  unreachable
await.fine.18:
  %t32 = call ptr @veles_task_result(ptr %t17)
  %t33 = load i64, ptr %t32
  call void @veles_i64_to_string(ptr %a34, i64 %t33)
  %t35 = load %str, ptr %a34
  %t36 = extractvalue %str %t35, 0
  %t37 = extractvalue %str %t35, 1
  %t38 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t39 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a40, ptr %t36, i64 %t37, ptr %t38, i64 %t39)
  %t41 = load %str, ptr %a40
  %t42 = load ptr, ptr %a16
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
  call void @veles_cleanup_pop()
  %t48 = load ptr, ptr %a4
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
  call void @veles_cleanup_pop()
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.24:
  %t52 = load ptr, ptr %a4
  %t53 = call ptr @veles_scope_failed(ptr %t52)
  %t54 = icmp ne ptr %t53, null
  br i1 %t54, label %scope.abort.29, label %scope.ok.30
scope.ok.30:
  br label %cont.31
scope.abort.29:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.1
cont.31:
  br label %await.19
await.got.20:
  %t55 = call i64 @veles_task_panicked(ptr %t42)
  %t56 = icmp ne i64 %t55, 0
  br i1 %t56, label %await.repanic.32, label %await.fine.33
await.repanic.32:
  call void @veles_task_repanic(ptr %t42)
  unreachable
await.fine.33:
  %t57 = call ptr @veles_task_result(ptr %t42)
  %t58 = load i64, ptr %t57
  call void @veles_i64_to_string(ptr %a59, i64 %t58)
  %t60 = load %str, ptr %a59
  %t61 = extractvalue %str %t41, 0
  %t62 = extractvalue %str %t41, 1
  %t63 = extractvalue %str %t60, 0
  %t64 = extractvalue %str %t60, 1
  call void @veles_string_concat(ptr %a65, ptr %t61, i64 %t62, ptr %t63, i64 %t64)
  %t66 = load %str, ptr %a65
  call void @v_std.io.println(%str %t66)
  br label %scope.wait.1
scope.wait.1:
  %t67 = load ptr, ptr %a4
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
  call void @veles_cleanup_pop()
  %t73 = load ptr, ptr %a4
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
  call void @veles_cleanup_pop()
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.36:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop()
  %t77 = load ptr, ptr %a4
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
  call void @veles_cleanup_pop()
  call void @v_main.Closeable.Resource.close(ptr %a2)
  %t82 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t82, ptr %a83
  store i64 0, ptr %a84
  %t85 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t85, ptr %a86
  call void @veles_cleanup_push(ptr @scope.cancel.thunk, ptr %a86)
  %t87 = load ptr, ptr %a86
  %t88 = call ptr @veles_task_launch(ptr %t87)
  %t89 = load ptr, ptr %a83
  %t90 = call ptr @veles_alloc_words(i64 16)
  %t91 = getelementptr inbounds { ptr }, ptr %t90, i32 0, i32 0
  store ptr %t89, ptr %t91
  call void @veles_task_start(ptr %t88, ptr @entry.v_main.produce, ptr %t90)
  store ptr %t88, ptr %a92
  br label %loop.cond.48
loop.cond.48:
  br label %loop.body.51
loop.body.51:
  %t93 = load ptr, ptr %a83
  store i64 zeroinitializer, ptr %a94
  br label %recv.52
recv.52:
  %t95 = call i64 @veles_chan_recv(ptr %task, ptr %t93, ptr %a94)
  %t96 = icmp ne i64 %t95, 0
  br i1 %t96, label %recv.done.53, label %recv.susp.54
recv.susp.54:
  %t97 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t97, label %coro.suspend [ i8 0, label %resume.55 i8 1, label %coro.cleanup ]
resume.55:
  %t98 = call i64 @veles_task_cancelled(ptr %task)
  %t99 = icmp ne i64 %t98, 0
  br i1 %t99, label %cancelled.56, label %cont.57
cancelled.56:
  call void @veles_cleanup_pop()
  %t100 = load ptr, ptr %a86
  call void @veles_scope_cancel(ptr %t100)
  br label %abandon.wait.58
abandon.wait.58:
  %t101 = call i64 @veles_scope_wait(ptr %task, ptr %t100)
  %t102 = icmp ne i64 %t101, 0
  br i1 %t102, label %abandon.done.59, label %abandon.susp.60
abandon.susp.60:
  %t103 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t103, label %coro.suspend [ i8 0, label %resume.61 i8 1, label %coro.cleanup ]
resume.61:
  br label %abandon.wait.58
abandon.done.59:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.57:
  %t104 = load ptr, ptr %a86
  %t105 = call ptr @veles_scope_failed(ptr %t104)
  %t106 = icmp ne ptr %t105, null
  br i1 %t106, label %scope.abort.62, label %scope.ok.63
scope.ok.63:
  br label %cont.64
scope.abort.62:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.45
cont.64:
  br label %recv.52
recv.done.53:
  %t107 = icmp eq i64 %t95, 1
  %t108 = load i64, ptr %a94
  %t109 = insertvalue { i1, i64 } undef, i1 %t107, 0
  %t110 = insertvalue { i1, i64 } %t109, i64 %t108, 1
  %t113 = extractvalue { i1, i64 } %t110, 0
  %t112 = xor i1 %t113, true
  br i1 %t112, label %elvis.default.65, label %elvis.some.66
elvis.some.66:
  %t114 = extractvalue { i1, i64 } %t110, 1
  store i64 %t114, ptr %a111
  br label %elvis.end.67
elvis.default.65:
  br label %loop.end.50
elvis.end.67:
  %t115 = load i64, ptr %a111
  store i64 %t115, ptr %a116
  %t117 = load i64, ptr %a84
  %t118 = load i64, ptr %a116
  %t120 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t117, i64 %t118)
  %t121 = extractvalue { i64, i1 } %t120, 0
  %t122 = extractvalue { i64, i1 } %t120, 1
  br i1 %t122, label %overflow.68, label %arith.ok.69
overflow.68:
  %t123 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t124 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t125 = extractvalue %str { ptr @.str.7, i64 12 }, 0
  %t126 = extractvalue %str { ptr @.str.7, i64 12 }, 1
  call void @veles_panic_at(ptr %t123, i64 %t124, ptr %t125, i64 %t126)
  unreachable
arith.ok.69:
  store i64 %t121, ptr %a84
  br label %loop.post.49
loop.post.49:
  br label %loop.cond.48
loop.end.50:
  br label %scope.wait.45
scope.wait.45:
  %t127 = load ptr, ptr %a86
  %t128 = call i64 @veles_scope_wait(ptr %task, ptr %t127)
  %t129 = icmp ne i64 %t128, 0
  br i1 %t129, label %scope.done.46, label %scope.susp.47
scope.susp.47:
  %t130 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t130, label %coro.suspend [ i8 0, label %resume.70 i8 1, label %coro.cleanup ]
resume.70:
  %t131 = call i64 @veles_task_cancelled(ptr %task)
  %t132 = icmp ne i64 %t131, 0
  br i1 %t132, label %cancelled.71, label %cont.72
cancelled.71:
  call void @veles_cleanup_pop()
  %t133 = load ptr, ptr %a86
  call void @veles_scope_cancel(ptr %t133)
  br label %abandon.wait.73
abandon.wait.73:
  %t134 = call i64 @veles_scope_wait(ptr %task, ptr %t133)
  %t135 = icmp ne i64 %t134, 0
  br i1 %t135, label %abandon.done.74, label %abandon.susp.75
abandon.susp.75:
  %t136 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t136, label %coro.suspend [ i8 0, label %resume.76 i8 1, label %coro.cleanup ]
resume.76:
  br label %abandon.wait.73
abandon.done.74:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.72:
  br label %scope.wait.45
scope.done.46:
  call void @veles_cleanup_pop()
  %t137 = load ptr, ptr %a86
  %t138 = call ptr @veles_scope_failed(ptr %t137)
  %t139 = icmp ne ptr %t138, null
  br i1 %t139, label %scope.check.77, label %scope.after.78
scope.check.77:
  %t140 = call i64 @veles_task_panicked(ptr %t138)
  %t141 = icmp ne i64 %t140, 0
  br i1 %t141, label %scope.repanic.79, label %scope.errors.80
scope.repanic.79:
  call void @veles_task_repanic(ptr %t138)
  unreachable
scope.errors.80:
  br label %scope.after.78
scope.after.78:
  %t142 = load i64, ptr %a84
  call void @veles_i64_to_string(ptr %a143, i64 %t142)
  %t144 = load %str, ptr %a143
  %t145 = extractvalue %str { ptr @.str.8, i64 4 }, 0
  %t146 = extractvalue %str { ptr @.str.8, i64 4 }, 1
  %t147 = extractvalue %str %t144, 0
  %t148 = extractvalue %str %t144, 1
  call void @veles_string_concat(ptr %a149, ptr %t145, i64 %t146, ptr %t147, i64 %t148)
  %t150 = load %str, ptr %a149
  call void @v_std.io.println(%str %t150)
  %t151 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t151, ptr %a152
  %t153 = load ptr, ptr %a152
  store %str { ptr @.str.9, i64 1 }, ptr %a154
  %t155 = call i64 @veles_chan_try_send(ptr %t153, ptr %a154)
  %t156 = icmp ne i64 %t155, 0
  call void @veles_bool_to_string(ptr %a157, i1 %t156)
  %t158 = load %str, ptr %a157
  %t159 = extractvalue %str %t158, 0
  %t160 = extractvalue %str %t158, 1
  %t161 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t162 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a163, ptr %t159, i64 %t160, ptr %t161, i64 %t162)
  %t164 = load %str, ptr %a163
  %t165 = load ptr, ptr %a152
  store %str { ptr @.str.10, i64 1 }, ptr %a166
  %t167 = call i64 @veles_chan_try_send(ptr %t165, ptr %a166)
  %t168 = icmp ne i64 %t167, 0
  call void @veles_bool_to_string(ptr %a169, i1 %t168)
  %t170 = load %str, ptr %a169
  %t171 = extractvalue %str %t164, 0
  %t172 = extractvalue %str %t164, 1
  %t173 = extractvalue %str %t170, 0
  %t174 = extractvalue %str %t170, 1
  call void @veles_string_concat(ptr %a175, ptr %t171, i64 %t172, ptr %t173, i64 %t174)
  %t176 = load %str, ptr %a175
  %t177 = extractvalue %str %t176, 0
  %t178 = extractvalue %str %t176, 1
  %t179 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t180 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a181, ptr %t177, i64 %t178, ptr %t179, i64 %t180)
  %t182 = load %str, ptr %a181
  %t183 = load ptr, ptr %a152
  store %str { ptr @.str.11, i64 1 }, ptr %a184
  %t185 = call i64 @veles_chan_try_send(ptr %t183, ptr %a184)
  %t186 = icmp ne i64 %t185, 0
  call void @veles_bool_to_string(ptr %a187, i1 %t186)
  %t188 = load %str, ptr %a187
  %t189 = extractvalue %str %t182, 0
  %t190 = extractvalue %str %t182, 1
  %t191 = extractvalue %str %t188, 0
  %t192 = extractvalue %str %t188, 1
  call void @veles_string_concat(ptr %a193, ptr %t189, i64 %t190, ptr %t191, i64 %t192)
  %t194 = load %str, ptr %a193
  %t195 = extractvalue %str %t194, 0
  %t196 = extractvalue %str %t194, 1
  %t197 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t198 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a199, ptr %t195, i64 %t196, ptr %t197, i64 %t198)
  %t200 = load %str, ptr %a199
  %t201 = load ptr, ptr %a152
  store %str zeroinitializer, ptr %a202
  %t203 = call i64 @veles_chan_try_recv(ptr %t201, ptr %a202)
  %t204 = icmp ne i64 %t203, 0
  %t205 = load %str, ptr %a202
  %t206 = insertvalue { i1, %str } undef, i1 %t204, 0
  %t207 = insertvalue { i1, %str } %t206, %str %t205, 1
  %t210 = extractvalue { i1, %str } %t207, 0
  %t209 = xor i1 %t210, true
  br i1 %t209, label %elvis.default.81, label %elvis.some.82
elvis.some.82:
  %t211 = extractvalue { i1, %str } %t207, 1
  store %str %t211, ptr %a208
  br label %elvis.end.83
elvis.default.81:
  store %str { ptr @.str.12, i64 1 }, ptr %a208
  br label %elvis.end.83
elvis.end.83:
  %t212 = load %str, ptr %a208
  %t213 = extractvalue %str %t200, 0
  %t214 = extractvalue %str %t200, 1
  %t215 = extractvalue %str %t212, 0
  %t216 = extractvalue %str %t212, 1
  call void @veles_string_concat(ptr %a217, ptr %t213, i64 %t214, ptr %t215, i64 %t216)
  %t218 = load %str, ptr %a217
  %t219 = extractvalue %str %t218, 0
  %t220 = extractvalue %str %t218, 1
  %t221 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t222 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a223, ptr %t219, i64 %t220, ptr %t221, i64 %t222)
  %t224 = load %str, ptr %a223
  %t225 = load ptr, ptr %a152
  store %str zeroinitializer, ptr %a226
  %t227 = call i64 @veles_chan_try_recv(ptr %t225, ptr %a226)
  %t228 = icmp ne i64 %t227, 0
  %t229 = load %str, ptr %a226
  %t230 = insertvalue { i1, %str } undef, i1 %t228, 0
  %t231 = insertvalue { i1, %str } %t230, %str %t229, 1
  %t234 = extractvalue { i1, %str } %t231, 0
  %t233 = xor i1 %t234, true
  br i1 %t233, label %elvis.default.84, label %elvis.some.85
elvis.some.85:
  %t235 = extractvalue { i1, %str } %t231, 1
  store %str %t235, ptr %a232
  br label %elvis.end.86
elvis.default.84:
  store %str { ptr @.str.12, i64 1 }, ptr %a232
  br label %elvis.end.86
elvis.end.86:
  %t236 = load %str, ptr %a232
  %t237 = extractvalue %str %t224, 0
  %t238 = extractvalue %str %t224, 1
  %t239 = extractvalue %str %t236, 0
  %t240 = extractvalue %str %t236, 1
  call void @veles_string_concat(ptr %a241, ptr %t237, i64 %t238, ptr %t239, i64 %t240)
  %t242 = load %str, ptr %a241
  %t243 = extractvalue %str %t242, 0
  %t244 = extractvalue %str %t242, 1
  %t245 = extractvalue %str { ptr @.str.6, i64 1 }, 0
  %t246 = extractvalue %str { ptr @.str.6, i64 1 }, 1
  call void @veles_string_concat(ptr %a247, ptr %t243, i64 %t244, ptr %t245, i64 %t246)
  %t248 = load %str, ptr %a247
  %t249 = load ptr, ptr %a152
  store %str zeroinitializer, ptr %a250
  %t251 = call i64 @veles_chan_try_recv(ptr %t249, ptr %a250)
  %t252 = icmp ne i64 %t251, 0
  %t253 = load %str, ptr %a250
  %t254 = insertvalue { i1, %str } undef, i1 %t252, 0
  %t255 = insertvalue { i1, %str } %t254, %str %t253, 1
  %t258 = extractvalue { i1, %str } %t255, 0
  %t257 = xor i1 %t258, true
  br i1 %t257, label %elvis.default.87, label %elvis.some.88
elvis.some.88:
  %t259 = extractvalue { i1, %str } %t255, 1
  store %str %t259, ptr %a256
  br label %elvis.end.89
elvis.default.87:
  store %str { ptr @.str.12, i64 1 }, ptr %a256
  br label %elvis.end.89
elvis.end.89:
  %t260 = load %str, ptr %a256
  %t261 = extractvalue %str %t248, 0
  %t262 = extractvalue %str %t248, 1
  %t263 = extractvalue %str %t260, 0
  %t264 = extractvalue %str %t260, 1
  call void @veles_string_concat(ptr %a265, ptr %t261, i64 %t262, ptr %t263, i64 %t264)
  %t266 = load %str, ptr %a265
  call void @v_std.io.println(%str %t266)
  %t267 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t267, ptr %a268
  store i64 0, ptr %a269
  %t270 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t270, ptr %a271
  call void @veles_cleanup_push(ptr @scope.cancel.thunk, ptr %a271)
  %t272 = load ptr, ptr %a271
  %t273 = call ptr @veles_task_launch(ptr %t272)
  %t274 = load ptr, ptr %a268
  %t275 = call ptr @veles_alloc_words(i64 16)
  %t276 = getelementptr inbounds { ptr }, ptr %t275, i32 0, i32 0
  store ptr %t274, ptr %t276
  call void @veles_task_start(ptr %t273, ptr @entry.v_main.produce, ptr %t275)
  store ptr %t273, ptr %a277
  br label %loop.cond.93
loop.cond.93:
  %t278 = load i64, ptr %a269
  %t279 = icmp slt i64 %t278, 6
  br i1 %t279, label %loop.body.96, label %loop.end.95
loop.body.96:
  %t280 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  %t281 = extractvalue %S.std.prelude.Duration %t280, 0
  %t283 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t281, i64 999999)
  %t284 = extractvalue { i64, i1 } %t283, 0
  %t285 = extractvalue { i64, i1 } %t283, 1
  br i1 %t285, label %overflow.97, label %arith.ok.98
overflow.97:
  %t286 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t287 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t288 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t289 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t286, i64 %t287, ptr %t288, i64 %t289)
  unreachable
arith.ok.98:
  %t291 = icmp eq i64 1000000, 0
  br i1 %t291, label %divzero.99, label %div.ok.100
divzero.99:
  %t292 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t293 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t294 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t295 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t292, i64 %t293, ptr %t294, i64 %t295)
  unreachable
div.ok.100:
  %t296 = icmp eq i64 %t284, -9223372036854775808
  %t297 = icmp eq i64 1000000, -1
  %t298 = and i1 %t296, %t297
  br i1 %t298, label %divof.101, label %div.ok.102
divof.101:
  %t299 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t300 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t301 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t302 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t299, i64 %t300, ptr %t301, i64 %t302)
  unreachable
div.ok.102:
  %t290 = sdiv i64 %t284, 1000000
  br label %sleep.103
sleep.103:
  %t303 = call i64 @veles_task_sleep(ptr %task, i64 %t290)
  %t304 = icmp ne i64 %t303, 0
  br i1 %t304, label %sleep.done.104, label %sleep.susp.105
sleep.susp.105:
  %t305 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t305, label %coro.suspend [ i8 0, label %resume.106 i8 1, label %coro.cleanup ]
resume.106:
  %t306 = call i64 @veles_task_cancelled(ptr %task)
  %t307 = icmp ne i64 %t306, 0
  br i1 %t307, label %cancelled.107, label %cont.108
cancelled.107:
  call void @veles_cleanup_pop()
  %t308 = load ptr, ptr %a271
  call void @veles_scope_cancel(ptr %t308)
  br label %abandon.wait.109
abandon.wait.109:
  %t309 = call i64 @veles_scope_wait(ptr %task, ptr %t308)
  %t310 = icmp ne i64 %t309, 0
  br i1 %t310, label %abandon.done.110, label %abandon.susp.111
abandon.susp.111:
  %t311 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t311, label %coro.suspend [ i8 0, label %resume.112 i8 1, label %coro.cleanup ]
resume.112:
  br label %abandon.wait.109
abandon.done.110:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.108:
  %t312 = load ptr, ptr %a271
  %t313 = call ptr @veles_scope_failed(ptr %t312)
  %t314 = icmp ne ptr %t313, null
  br i1 %t314, label %scope.abort.113, label %scope.ok.114
scope.ok.114:
  br label %cont.115
scope.abort.113:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.90
cont.115:
  br label %sleep.103
sleep.done.104:
  %t315 = load i64, ptr %a269
  %t316 = load ptr, ptr %a268
  store i64 zeroinitializer, ptr %a317
  br label %recv.116
recv.116:
  %t318 = call i64 @veles_chan_recv(ptr %task, ptr %t316, ptr %a317)
  %t319 = icmp ne i64 %t318, 0
  br i1 %t319, label %recv.done.117, label %recv.susp.118
recv.susp.118:
  %t320 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t320, label %coro.suspend [ i8 0, label %resume.119 i8 1, label %coro.cleanup ]
resume.119:
  %t321 = call i64 @veles_task_cancelled(ptr %task)
  %t322 = icmp ne i64 %t321, 0
  br i1 %t322, label %cancelled.120, label %cont.121
cancelled.120:
  call void @veles_cleanup_pop()
  %t323 = load ptr, ptr %a271
  call void @veles_scope_cancel(ptr %t323)
  br label %abandon.wait.122
abandon.wait.122:
  %t324 = call i64 @veles_scope_wait(ptr %task, ptr %t323)
  %t325 = icmp ne i64 %t324, 0
  br i1 %t325, label %abandon.done.123, label %abandon.susp.124
abandon.susp.124:
  %t326 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t326, label %coro.suspend [ i8 0, label %resume.125 i8 1, label %coro.cleanup ]
resume.125:
  br label %abandon.wait.122
abandon.done.123:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.121:
  %t327 = load ptr, ptr %a271
  %t328 = call ptr @veles_scope_failed(ptr %t327)
  %t329 = icmp ne ptr %t328, null
  br i1 %t329, label %scope.abort.126, label %scope.ok.127
scope.ok.127:
  br label %cont.128
scope.abort.126:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.90
cont.128:
  br label %recv.116
recv.done.117:
  %t330 = icmp eq i64 %t318, 1
  %t331 = load i64, ptr %a317
  %t332 = insertvalue { i1, i64 } undef, i1 %t330, 0
  %t333 = insertvalue { i1, i64 } %t332, i64 %t331, 1
  %t336 = extractvalue { i1, i64 } %t333, 0
  %t335 = xor i1 %t336, true
  br i1 %t335, label %elvis.default.129, label %elvis.some.130
elvis.some.130:
  %t337 = extractvalue { i1, i64 } %t333, 1
  store i64 %t337, ptr %a334
  br label %elvis.end.131
elvis.default.129:
  store i64 0, ptr %a334
  br label %elvis.end.131
elvis.end.131:
  %t338 = load i64, ptr %a334
  %t340 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t315, i64 %t338)
  %t341 = extractvalue { i64, i1 } %t340, 0
  %t342 = extractvalue { i64, i1 } %t340, 1
  br i1 %t342, label %overflow.132, label %arith.ok.133
overflow.132:
  %t343 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t344 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t345 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t346 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_panic_at(ptr %t343, i64 %t344, ptr %t345, i64 %t346)
  unreachable
arith.ok.133:
  store i64 %t341, ptr %a269
  br label %loop.post.94
loop.post.94:
  br label %loop.cond.93
loop.end.95:
  br label %scope.wait.90
scope.wait.90:
  %t347 = load ptr, ptr %a271
  %t348 = call i64 @veles_scope_wait(ptr %task, ptr %t347)
  %t349 = icmp ne i64 %t348, 0
  br i1 %t349, label %scope.done.91, label %scope.susp.92
scope.susp.92:
  %t350 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t350, label %coro.suspend [ i8 0, label %resume.134 i8 1, label %coro.cleanup ]
resume.134:
  %t351 = call i64 @veles_task_cancelled(ptr %task)
  %t352 = icmp ne i64 %t351, 0
  br i1 %t352, label %cancelled.135, label %cont.136
cancelled.135:
  call void @veles_cleanup_pop()
  %t353 = load ptr, ptr %a271
  call void @veles_scope_cancel(ptr %t353)
  br label %abandon.wait.137
abandon.wait.137:
  %t354 = call i64 @veles_scope_wait(ptr %task, ptr %t353)
  %t355 = icmp ne i64 %t354, 0
  br i1 %t355, label %abandon.done.138, label %abandon.susp.139
abandon.susp.139:
  %t356 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t356, label %coro.suspend [ i8 0, label %resume.140 i8 1, label %coro.cleanup ]
resume.140:
  br label %abandon.wait.137
abandon.done.138:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.136:
  br label %scope.wait.90
scope.done.91:
  call void @veles_cleanup_pop()
  %t357 = load ptr, ptr %a271
  %t358 = call ptr @veles_scope_failed(ptr %t357)
  %t359 = icmp ne ptr %t358, null
  br i1 %t359, label %scope.check.141, label %scope.after.142
scope.check.141:
  %t360 = call i64 @veles_task_panicked(ptr %t358)
  %t361 = icmp ne i64 %t360, 0
  br i1 %t361, label %scope.repanic.143, label %scope.errors.144
scope.repanic.143:
  call void @veles_task_repanic(ptr %t358)
  unreachable
scope.errors.144:
  br label %scope.after.142
scope.after.142:
  %t362 = load i64, ptr %a269
  call void @veles_i64_to_string(ptr %a363, i64 %t362)
  %t364 = load %str, ptr %a363
  %t365 = extractvalue %str { ptr @.str.15, i64 6 }, 0
  %t366 = extractvalue %str { ptr @.str.15, i64 6 }, 1
  %t367 = extractvalue %str %t364, 0
  %t368 = extractvalue %str %t364, 1
  call void @veles_string_concat(ptr %a369, ptr %t365, i64 %t366, ptr %t367, i64 %t368)
  %t370 = load %str, ptr %a369
  call void @v_std.io.println(%str %t370)
  %t371 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t371, ptr %a372
  store i64 0, ptr %a373
  %t374 = call ptr @veles_scope_begin(ptr %task, i64 1)
  store ptr %t374, ptr %a375
  call void @veles_cleanup_push(ptr @scope.cancel.thunk, ptr %a375)
  %t376 = load ptr, ptr %a375
  %t377 = call ptr @veles_task_launch(ptr %t376)
  %t378 = load ptr, ptr %a372
  %t379 = call ptr @veles_alloc_words(i64 16)
  %t380 = getelementptr inbounds { ptr }, ptr %t379, i32 0, i32 0
  store ptr %t378, ptr %t380
  call void @veles_task_start(ptr %t377, ptr @entry.v_main.produce, ptr %t379)
  store ptr %t377, ptr %a381
  br label %loop.cond.148
loop.cond.148:
  %t382 = load i64, ptr %a373
  %t383 = icmp slt i64 %t382, 6
  br i1 %t383, label %loop.body.151, label %loop.end.150
loop.body.151:
  %t384 = load ptr, ptr %a372
  store i64 zeroinitializer, ptr %a385
  %t386 = call i64 @veles_chan_try_recv(ptr %t384, ptr %a385)
  %t387 = icmp ne i64 %t386, 0
  %t388 = load i64, ptr %a385
  %t389 = insertvalue { i1, i64 } undef, i1 %t387, 0
  %t390 = insertvalue { i1, i64 } %t389, i64 %t388, 1
  store { i1, i64 } %t390, ptr %a391
  %t392 = load { i1, i64 }, ptr %a391
  %t394 = extractvalue { i1, i64 } %t392, 0
  %t393 = xor i1 %t394, true
  %t395 = xor i1 %t393, true
  br i1 %t395, label %if.then.152, label %if.else.154
if.then.152:
  %t396 = load i64, ptr %a373
  %t397 = load { i1, i64 }, ptr %a391
  %t398 = extractvalue { i1, i64 } %t397, 1
  %t400 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t396, i64 %t398)
  %t401 = extractvalue { i64, i1 } %t400, 0
  %t402 = extractvalue { i64, i1 } %t400, 1
  br i1 %t402, label %overflow.155, label %arith.ok.156
overflow.155:
  %t403 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t404 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t405 = extractvalue %str { ptr @.str.16, i64 13 }, 0
  %t406 = extractvalue %str { ptr @.str.16, i64 13 }, 1
  call void @veles_panic_at(ptr %t403, i64 %t404, ptr %t405, i64 %t406)
  unreachable
arith.ok.156:
  store i64 %t401, ptr %a373
  br label %if.end.153
if.else.154:
  %t407 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t408 = load i64, ptr %t407
  %t410 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t408, i64 999999)
  %t411 = extractvalue { i64, i1 } %t410, 0
  %t412 = extractvalue { i64, i1 } %t410, 1
  br i1 %t412, label %overflow.157, label %arith.ok.158
overflow.157:
  %t413 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t414 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t415 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t416 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t413, i64 %t414, ptr %t415, i64 %t416)
  unreachable
arith.ok.158:
  %t418 = icmp eq i64 1000000, 0
  br i1 %t418, label %divzero.159, label %div.ok.160
divzero.159:
  %t419 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t420 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t421 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t422 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t419, i64 %t420, ptr %t421, i64 %t422)
  unreachable
div.ok.160:
  %t423 = icmp eq i64 %t411, -9223372036854775808
  %t424 = icmp eq i64 1000000, -1
  %t425 = and i1 %t423, %t424
  br i1 %t425, label %divof.161, label %div.ok.162
divof.161:
  %t426 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t427 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t428 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t429 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_panic_at(ptr %t426, i64 %t427, ptr %t428, i64 %t429)
  unreachable
div.ok.162:
  %t417 = sdiv i64 %t411, 1000000
  br label %sleep.163
sleep.163:
  %t430 = call i64 @veles_task_sleep(ptr %task, i64 %t417)
  %t431 = icmp ne i64 %t430, 0
  br i1 %t431, label %sleep.done.164, label %sleep.susp.165
sleep.susp.165:
  %t432 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t432, label %coro.suspend [ i8 0, label %resume.166 i8 1, label %coro.cleanup ]
resume.166:
  %t433 = call i64 @veles_task_cancelled(ptr %task)
  %t434 = icmp ne i64 %t433, 0
  br i1 %t434, label %cancelled.167, label %cont.168
cancelled.167:
  call void @veles_cleanup_pop()
  %t435 = load ptr, ptr %a375
  call void @veles_scope_cancel(ptr %t435)
  br label %abandon.wait.169
abandon.wait.169:
  %t436 = call i64 @veles_scope_wait(ptr %task, ptr %t435)
  %t437 = icmp ne i64 %t436, 0
  br i1 %t437, label %abandon.done.170, label %abandon.susp.171
abandon.susp.171:
  %t438 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t438, label %coro.suspend [ i8 0, label %resume.172 i8 1, label %coro.cleanup ]
resume.172:
  br label %abandon.wait.169
abandon.done.170:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.168:
  %t439 = load ptr, ptr %a375
  %t440 = call ptr @veles_scope_failed(ptr %t439)
  %t441 = icmp ne ptr %t440, null
  br i1 %t441, label %scope.abort.173, label %scope.ok.174
scope.ok.174:
  br label %cont.175
scope.abort.173:
  call void @veles_task_leave_waits(ptr %task)
  br label %scope.wait.145
cont.175:
  br label %sleep.163
sleep.done.164:
  br label %if.end.153
if.end.153:
  br label %loop.post.149
loop.post.149:
  br label %loop.cond.148
loop.end.150:
  br label %scope.wait.145
scope.wait.145:
  %t442 = load ptr, ptr %a375
  %t443 = call i64 @veles_scope_wait(ptr %task, ptr %t442)
  %t444 = icmp ne i64 %t443, 0
  br i1 %t444, label %scope.done.146, label %scope.susp.147
scope.susp.147:
  %t445 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t445, label %coro.suspend [ i8 0, label %resume.176 i8 1, label %coro.cleanup ]
resume.176:
  %t446 = call i64 @veles_task_cancelled(ptr %task)
  %t447 = icmp ne i64 %t446, 0
  br i1 %t447, label %cancelled.177, label %cont.178
cancelled.177:
  call void @veles_cleanup_pop()
  %t448 = load ptr, ptr %a375
  call void @veles_scope_cancel(ptr %t448)
  br label %abandon.wait.179
abandon.wait.179:
  %t449 = call i64 @veles_scope_wait(ptr %task, ptr %t448)
  %t450 = icmp ne i64 %t449, 0
  br i1 %t450, label %abandon.done.180, label %abandon.susp.181
abandon.susp.181:
  %t451 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t451, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  br label %abandon.wait.179
abandon.done.180:
  call void @veles_task_finish_cancelled(ptr %task)
  br label %coro.final
cont.178:
  br label %scope.wait.145
scope.done.146:
  call void @veles_cleanup_pop()
  %t452 = load ptr, ptr %a375
  %t453 = call ptr @veles_scope_failed(ptr %t452)
  %t454 = icmp ne ptr %t453, null
  br i1 %t454, label %scope.check.183, label %scope.after.184
scope.check.183:
  %t455 = call i64 @veles_task_panicked(ptr %t453)
  %t456 = icmp ne i64 %t455, 0
  br i1 %t456, label %scope.repanic.185, label %scope.errors.186
scope.repanic.185:
  call void @veles_task_repanic(ptr %t453)
  unreachable
scope.errors.186:
  br label %scope.after.184
scope.after.184:
  %t457 = load i64, ptr %a373
  call void @veles_i64_to_string(ptr %a458, i64 %t457)
  %t459 = load %str, ptr %a458
  %t460 = extractvalue %str { ptr @.str.18, i64 7 }, 0
  %t461 = extractvalue %str { ptr @.str.18, i64 7 }, 1
  %t462 = extractvalue %str %t459, 0
  %t463 = extractvalue %str %t459, 1
  call void @veles_string_concat(ptr %a464, ptr %t460, i64 %t461, ptr %t462, i64 %t463)
  %t465 = load %str, ptr %a464
  call void @v_std.io.println(%str %t465)
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
