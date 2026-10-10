%S.main.Resource = type { %str }
define ptr @v_main.slow(ptr %task, ptr %link, i64 %p1) presplitcoroutine {
entry:
  %a1 = alloca i64
  %a41 = alloca i64
  store i64 %p1, ptr %a1
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t29 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t29, label %coro.suspend [ i8 0, label %resume.10 i8 1, label %coro.cleanup ]
resume.10:
  %t30 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t31 = icmp ne i64 %t30, 0
  br i1 %t31, label %cancelled.11, label %cont.12
cancelled.11:
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  %t42 = load ptr, ptr %link
  %t43 = icmp eq ptr %t42, null
  br i1 %t43, label %ret.task.15, label %ret.call.16
ret.task.15:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a41, i64 8, i64 0, ptr null)
  br label %coro.final
ret.call.16:
  %t44 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %link, i32 0, i32 3
  store i64 %t35, ptr %t44
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

define ptr @v_main.produce(ptr %task, ptr %link, ptr %p1) presplitcoroutine {
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
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t33 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t33, label %coro.suspend [ i8 0, label %resume.14 i8 1, label %coro.cleanup ]
resume.14:
  %t34 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t35 = icmp ne i64 %t34, 0
  br i1 %t35, label %cancelled.15, label %cont.16
cancelled.15:
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  %t45 = load volatile i32, ptr @veles_attention_line, align 64
  %t46 = icmp ne i32 %t45, 0
  br i1 %t46, label %safepoint.22, label %safepoint.on.23, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.22:
  %t47 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t47, label %safepoint.on.23 [ i64 0, label %backedge.look.26 i64 1, label %backedge.yield.24 i64 2, label %backedge.cancel.25 ]
backedge.look.26:
  br label %safepoint.on.23
backedge.yield.24:
  br label %yield.27
yield.27:
  %t48 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t49 = icmp ne i64 %t48, 0
  br i1 %t49, label %safepoint.on.23, label %yield.susp.28
yield.susp.28:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t50 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t50, label %coro.suspend [ i8 0, label %resume.29 i8 1, label %coro.cleanup ]
resume.29:
  %t51 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t52 = icmp ne i64 %t51, 0
  br i1 %t52, label %cancelled.30, label %cont.31
cancelled.30:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.31:
  br label %yield.27
backedge.cancel.25:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.23:
  br label %loop.cond.1
loop.end.3:
  %t53 = load ptr, ptr %a1
  call void @veles_chan_close(ptr %t53)
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
  %a2 = alloca %S.main.Resource
  %a3 = alloca { ptr, ptr, ptr, ptr }
  %a5 = alloca ptr
  %a6 = alloca { ptr, ptr, ptr, ptr }
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
  %a99 = alloca { ptr, ptr, ptr, ptr }
  %a105 = alloca ptr
  %a107 = alloca i64
  %a125 = alloca i64
  %a130 = alloca i64
  %a181 = alloca [21 x i8]
  %a185 = alloca %str
  %a194 = alloca ptr
  %a196 = alloca %str
  %a199 = alloca %str
  %a202 = alloca %str
  %a205 = alloca %str
  %a208 = alloca %str
  %a211 = alloca %str
  %a214 = alloca %str
  %a220 = alloca %str
  %a226 = alloca %str
  %a232 = alloca %str
  %a238 = alloca %str
  %a244 = alloca %str
  %a249 = alloca %str
  %a250 = alloca [11 x %str]
  %a266 = alloca ptr
  %a267 = alloca i64
  %a269 = alloca ptr
  %a270 = alloca { ptr, ptr, ptr, ptr }
  %a276 = alloca ptr
  %a319 = alloca i64
  %a337 = alloca i64
  %a390 = alloca [21 x i8]
  %a394 = alloca %str
  %a403 = alloca ptr
  %a404 = alloca i64
  %a406 = alloca ptr
  %a407 = alloca { ptr, ptr, ptr, ptr }
  %a413 = alloca ptr
  %a417 = alloca i64
  %a423 = alloca { i1, i64 }
  %a515 = alloca [21 x i8]
  %a519 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t1 = insertvalue %S.main.Resource undef, %str { ptr @.str.6, i64 2 }, 0
  store %S.main.Resource %t1, ptr %a2
  call void @veles_cleanup_push(ptr %a3, ptr @with.close.1, ptr %a2, ptr @with.move.2)
  %t4 = call ptr @veles_scope_begin(ptr %task, i64 1, i64 %coro.depth)
  store ptr %t4, ptr %a5
  call void @veles_cleanup_push(ptr %a6, ptr @scope.cancel.thunk, ptr %a5, ptr @cleanup.move.word)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t22 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t22, label %coro.suspend [ i8 0, label %resume.7 i8 1, label %coro.cleanup ]
resume.7:
  %t23 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t28 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t28, label %coro.suspend [ i8 0, label %resume.13 i8 1, label %coro.cleanup ]
resume.13:
  br label %abandon.wait.10
abandon.done.11:
  call void @veles_cleanup_pop(ptr %a3)
  call void @veles_shield_enter()
  %t29 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t30 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t29)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_shield_leave()
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  call void @veles_scope_abandon(ptr %t34, ptr %task)
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
  %t37 = call ptr @veles_task_value(ptr %t19)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t46 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t46, label %coro.suspend [ i8 0, label %resume.22 i8 1, label %coro.cleanup ]
resume.22:
  %t47 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t52 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t52, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  br label %abandon.wait.25
abandon.done.26:
  call void @veles_cleanup_pop(ptr %a3)
  call void @veles_shield_enter()
  %t53 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t54 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t53)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_shield_leave()
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  call void @veles_scope_abandon(ptr %t58, ptr %task)
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
  %t61 = call ptr @veles_task_value(ptr %t43)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t78 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t78, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t79 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t84 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t84, label %coro.suspend [ i8 0, label %resume.40 i8 1, label %coro.cleanup ]
resume.40:
  br label %abandon.wait.37
abandon.done.38:
  call void @veles_cleanup_pop(ptr %a3)
  call void @veles_shield_enter()
  %t85 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t86 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t85)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_shield_leave()
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  call void @veles_shield_enter()
  %t92 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t93 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t92)
  call void @v_main.Closeable.Resource.close(ptr %a2)
  call void @veles_call_pop()
  call void @veles_shield_leave()
  %t94 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t94, ptr %a95
  store i64 0, ptr %a96
  %t97 = call ptr @veles_scope_begin(ptr %task, i64 1, i64 %coro.depth)
  store ptr %t97, ptr %a98
  call void @veles_cleanup_push(ptr %a99, ptr @scope.cancel.thunk, ptr %a98, ptr @cleanup.move.word)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t110 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t110, label %coro.suspend [ i8 0, label %resume.55 i8 1, label %coro.cleanup ]
resume.55:
  %t111 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
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
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t116 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t116, label %coro.suspend [ i8 0, label %resume.61 i8 1, label %coro.cleanup ]
resume.61:
  br label %abandon.wait.58
abandon.done.59:
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  call void @veles_scope_abandon(ptr %t120, ptr %task)
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
  %t141 = load volatile i32, ptr @veles_attention_line, align 64
  %t142 = icmp ne i32 %t141, 0
  br i1 %t142, label %safepoint.70, label %safepoint.on.71, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.70:
  %t143 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t143, label %safepoint.on.71 [ i64 0, label %backedge.look.74 i64 1, label %backedge.yield.72 i64 2, label %backedge.cancel.73 ]
backedge.look.74:
  %t144 = load ptr, ptr %a98
  %t145 = call ptr @veles_scope_failed(ptr %t144)
  %t146 = icmp ne ptr %t145, null
  br i1 %t146, label %scope.abort.75, label %scope.ok.76
scope.ok.76:
  br label %cont.77
scope.abort.75:
  call void @veles_task_leave_waits(ptr %task)
  %t147 = load ptr, ptr %a98
  call void @veles_scope_abandon(ptr %t147, ptr %task)
  br label %scope.wait.45
cont.77:
  br label %safepoint.on.71
backedge.yield.72:
  br label %yield.78
yield.78:
  %t148 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t149 = icmp ne i64 %t148, 0
  br i1 %t149, label %safepoint.on.71, label %yield.susp.79
yield.susp.79:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t150 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t150, label %coro.suspend [ i8 0, label %resume.80 i8 1, label %coro.cleanup ]
resume.80:
  %t151 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t152 = icmp ne i64 %t151, 0
  br i1 %t152, label %cancelled.81, label %cont.82
cancelled.81:
  call void @veles_cleanup_pop(ptr %a99)
  %t153 = load ptr, ptr %a98
  call void @veles_scope_cancel(ptr %t153)
  br label %abandon.wait.83
abandon.wait.83:
  %t154 = call i64 @veles_scope_wait(ptr %task, ptr %t153)
  %t155 = icmp ne i64 %t154, 0
  br i1 %t155, label %abandon.done.84, label %abandon.susp.85
abandon.susp.85:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t156 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t156, label %coro.suspend [ i8 0, label %resume.86 i8 1, label %coro.cleanup ]
resume.86:
  br label %abandon.wait.83
abandon.done.84:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.82:
  %t157 = load ptr, ptr %a98
  %t158 = call ptr @veles_scope_failed(ptr %t157)
  %t159 = icmp ne ptr %t158, null
  br i1 %t159, label %scope.abort.87, label %scope.ok.88
scope.ok.88:
  br label %cont.89
scope.abort.87:
  call void @veles_task_leave_waits(ptr %task)
  %t160 = load ptr, ptr %a98
  call void @veles_scope_abandon(ptr %t160, ptr %task)
  br label %scope.wait.45
cont.89:
  br label %yield.78
backedge.cancel.73:
  call void @veles_cleanup_pop(ptr %a99)
  %t161 = load ptr, ptr %a98
  call void @veles_scope_cancel(ptr %t161)
  br label %abandon.wait.90
abandon.wait.90:
  %t162 = call i64 @veles_scope_wait(ptr %task, ptr %t161)
  %t163 = icmp ne i64 %t162, 0
  br i1 %t163, label %abandon.done.91, label %abandon.susp.92
abandon.susp.92:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t164 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t164, label %coro.suspend [ i8 0, label %resume.93 i8 1, label %coro.cleanup ]
resume.93:
  br label %abandon.wait.90
abandon.done.91:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.71:
  br label %loop.cond.48
loop.end.50:
  br label %scope.wait.45
scope.wait.45:
  %t165 = load ptr, ptr %a98
  %t166 = call i64 @veles_scope_wait(ptr %task, ptr %t165)
  %t167 = icmp ne i64 %t166, 0
  br i1 %t167, label %scope.done.46, label %scope.susp.47
scope.susp.47:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t168 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t168, label %coro.suspend [ i8 0, label %resume.94 i8 1, label %coro.cleanup ]
resume.94:
  %t169 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t170 = icmp ne i64 %t169, 0
  br i1 %t170, label %cancelled.95, label %cont.96
cancelled.95:
  call void @veles_cleanup_pop(ptr %a99)
  %t171 = load ptr, ptr %a98
  call void @veles_scope_cancel(ptr %t171)
  br label %abandon.wait.97
abandon.wait.97:
  %t172 = call i64 @veles_scope_wait(ptr %task, ptr %t171)
  %t173 = icmp ne i64 %t172, 0
  br i1 %t173, label %abandon.done.98, label %abandon.susp.99
abandon.susp.99:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t174 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t174, label %coro.suspend [ i8 0, label %resume.100 i8 1, label %coro.cleanup ]
resume.100:
  br label %abandon.wait.97
abandon.done.98:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.96:
  br label %scope.wait.45
scope.done.46:
  call void @veles_cleanup_pop(ptr %a99)
  %t175 = load ptr, ptr %a98
  %t176 = call ptr @veles_scope_failed(ptr %t175)
  %t177 = icmp ne ptr %t176, null
  br i1 %t177, label %scope.check.101, label %scope.after.102
scope.check.101:
  %t178 = call i64 @veles_task_panicked(ptr %t176)
  %t179 = icmp ne i64 %t178, 0
  br i1 %t179, label %scope.repanic.103, label %scope.errors.104
scope.repanic.103:
  call void @veles_task_repanic(ptr %t176)
  unreachable
scope.errors.104:
  br label %scope.after.102
scope.after.102:
  %t180 = load i64, ptr %a96
  %t182 = call i64 @veles_i64_format(ptr %a181, i64 %t180)
  %t183 = insertvalue %str undef, ptr %a181, 0
  %t184 = insertvalue %str %t183, i64 %t182, 1
  %t186 = extractvalue %str { ptr @.str.11, i64 4 }, 0
  %t187 = extractvalue %str { ptr @.str.11, i64 4 }, 1
  %t188 = extractvalue %str %t184, 0
  %t189 = extractvalue %str %t184, 1
  call void @veles_string_concat(ptr %a185, ptr %t186, i64 %t187, ptr %t188, i64 %t189)
  %t190 = load %str, ptr %a185
  %t191 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t192 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t191)
  call void @v_std.io.println(%str %t190)
  call void @veles_call_pop()
  %t193 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t193, ptr %a194
  %t195 = load ptr, ptr %a194
  store %str { ptr @.str.13, i64 1 }, ptr %a196
  %t197 = call i64 @veles_chan_try_send(ptr %t195, ptr %a196)
  %t198 = icmp ne i64 %t197, 0
  call void @veles_bool_to_string(ptr %a199, i1 zeroext %t198)
  %t200 = load %str, ptr %a199
  %t201 = load ptr, ptr %a194
  store %str { ptr @.str.14, i64 1 }, ptr %a202
  %t203 = call i64 @veles_chan_try_send(ptr %t201, ptr %a202)
  %t204 = icmp ne i64 %t203, 0
  call void @veles_bool_to_string(ptr %a205, i1 zeroext %t204)
  %t206 = load %str, ptr %a205
  %t207 = load ptr, ptr %a194
  store %str { ptr @.str.15, i64 1 }, ptr %a208
  %t209 = call i64 @veles_chan_try_send(ptr %t207, ptr %a208)
  %t210 = icmp ne i64 %t209, 0
  call void @veles_bool_to_string(ptr %a211, i1 zeroext %t210)
  %t212 = load %str, ptr %a211
  %t213 = load ptr, ptr %a194
  store %str zeroinitializer, ptr %a214
  %t215 = call i64 @veles_chan_try_recv(ptr %t213, ptr %a214)
  %t216 = icmp ne i64 %t215, 0
  %t217 = load %str, ptr %a214
  %t218 = insertvalue { i1, %str } undef, i1 %t216, 0
  %t219 = insertvalue { i1, %str } %t218, %str %t217, 1
  %t222 = extractvalue { i1, %str } %t219, 0
  %t221 = xor i1 %t222, true
  br i1 %t221, label %elvis.default.105, label %elvis.some.106
elvis.some.106:
  %t223 = extractvalue { i1, %str } %t219, 1
  store %str %t223, ptr %a220
  br label %elvis.end.107
elvis.default.105:
  store %str { ptr @.str.16, i64 1 }, ptr %a220
  br label %elvis.end.107
elvis.end.107:
  %t224 = load %str, ptr %a220
  %t225 = load ptr, ptr %a194
  store %str zeroinitializer, ptr %a226
  %t227 = call i64 @veles_chan_try_recv(ptr %t225, ptr %a226)
  %t228 = icmp ne i64 %t227, 0
  %t229 = load %str, ptr %a226
  %t230 = insertvalue { i1, %str } undef, i1 %t228, 0
  %t231 = insertvalue { i1, %str } %t230, %str %t229, 1
  %t234 = extractvalue { i1, %str } %t231, 0
  %t233 = xor i1 %t234, true
  br i1 %t233, label %elvis.default.108, label %elvis.some.109
elvis.some.109:
  %t235 = extractvalue { i1, %str } %t231, 1
  store %str %t235, ptr %a232
  br label %elvis.end.110
elvis.default.108:
  store %str { ptr @.str.16, i64 1 }, ptr %a232
  br label %elvis.end.110
elvis.end.110:
  %t236 = load %str, ptr %a232
  %t237 = load ptr, ptr %a194
  store %str zeroinitializer, ptr %a238
  %t239 = call i64 @veles_chan_try_recv(ptr %t237, ptr %a238)
  %t240 = icmp ne i64 %t239, 0
  %t241 = load %str, ptr %a238
  %t242 = insertvalue { i1, %str } undef, i1 %t240, 0
  %t243 = insertvalue { i1, %str } %t242, %str %t241, 1
  %t246 = extractvalue { i1, %str } %t243, 0
  %t245 = xor i1 %t246, true
  br i1 %t245, label %elvis.default.111, label %elvis.some.112
elvis.some.112:
  %t247 = extractvalue { i1, %str } %t243, 1
  store %str %t247, ptr %a244
  br label %elvis.end.113
elvis.default.111:
  store %str { ptr @.str.16, i64 1 }, ptr %a244
  br label %elvis.end.113
elvis.end.113:
  %t248 = load %str, ptr %a244
  %t251 = getelementptr [11 x %str], ptr %a250, i64 0, i64 0
  store %str %t200, ptr %t251
  %t252 = getelementptr [11 x %str], ptr %a250, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t252
  %t253 = getelementptr [11 x %str], ptr %a250, i64 0, i64 2
  store %str %t206, ptr %t253
  %t254 = getelementptr [11 x %str], ptr %a250, i64 0, i64 3
  store %str { ptr @.str.8, i64 1 }, ptr %t254
  %t255 = getelementptr [11 x %str], ptr %a250, i64 0, i64 4
  store %str %t212, ptr %t255
  %t256 = getelementptr [11 x %str], ptr %a250, i64 0, i64 5
  store %str { ptr @.str.8, i64 1 }, ptr %t256
  %t257 = getelementptr [11 x %str], ptr %a250, i64 0, i64 6
  store %str %t224, ptr %t257
  %t258 = getelementptr [11 x %str], ptr %a250, i64 0, i64 7
  store %str { ptr @.str.8, i64 1 }, ptr %t258
  %t259 = getelementptr [11 x %str], ptr %a250, i64 0, i64 8
  store %str %t236, ptr %t259
  %t260 = getelementptr [11 x %str], ptr %a250, i64 0, i64 9
  store %str { ptr @.str.8, i64 1 }, ptr %t260
  %t261 = getelementptr [11 x %str], ptr %a250, i64 0, i64 10
  store %str %t248, ptr %t261
  call void @veles_string_concat_n(ptr %a249, ptr %a250, i64 11)
  %t262 = load %str, ptr %a249
  %t263 = extractvalue %str { ptr @.str.17, i64 20 }, 0
  %t264 = extractvalue %str { ptr @.str.17, i64 20 }, 1
  call void @veles_call_push(ptr %t263)
  call void @v_std.io.println(%str %t262)
  call void @veles_call_pop()
  %t265 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t265, ptr %a266
  store i64 0, ptr %a267
  %t268 = call ptr @veles_scope_begin(ptr %task, i64 1, i64 %coro.depth)
  store ptr %t268, ptr %a269
  call void @veles_cleanup_push(ptr %a270, ptr @scope.cancel.thunk, ptr %a269, ptr @cleanup.move.word)
  %t271 = load ptr, ptr %a269
  %t272 = call ptr @veles_task_launch(ptr %t271, i64 0)
  %t273 = load ptr, ptr %a266
  %t274 = call ptr @veles_alloc_words(i64 16)
  %t275 = getelementptr inbounds { ptr }, ptr %t274, i32 0, i32 0
  store ptr %t273, ptr %t275
  call void @veles_task_spawn(ptr %t272, ptr @entry.v_main.produce, ptr %t274)
  store ptr %t272, ptr %a276
  br label %loop.cond.117
loop.cond.117:
  %t277 = load i64, ptr %a267
  %t278 = icmp slt i64 %t277, 6
  br i1 %t278, label %loop.body.120, label %loop.end.119
loop.body.120:
  %t279 = extractvalue %str { ptr @.str.18, i64 20 }, 0
  %t280 = extractvalue %str { ptr @.str.18, i64 20 }, 1
  call void @veles_call_push(ptr %t279)
  %t281 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t282 = extractvalue %S.std.prelude.Duration %t281, 0
  %t284 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t282, i64 999999)
  %t285 = extractvalue { i64, i1 } %t284, 0
  %t286 = extractvalue { i64, i1 } %t284, 1
  br i1 %t286, label %overflow.121, label %arith.ok.122
overflow.121:
  %t287 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t288 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t289 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t290 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t287, i64 %t288, ptr %t289, i64 %t290)
  unreachable
arith.ok.122:
  %t292 = icmp eq i64 1000000, 0
  br i1 %t292, label %divzero.123, label %div.ok.124
divzero.123:
  %t293 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t294 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t295 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t296 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t293, i64 %t294, ptr %t295, i64 %t296)
  unreachable
div.ok.124:
  %t297 = icmp eq i64 %t285, -9223372036854775808
  %t298 = icmp eq i64 1000000, -1
  %t299 = and i1 %t297, %t298
  br i1 %t299, label %divof.125, label %div.ok.126
divof.125:
  %t300 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t301 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t302 = extractvalue %str { ptr @.str.19, i64 13 }, 0
  %t303 = extractvalue %str { ptr @.str.19, i64 13 }, 1
  call void @veles_panic_at(ptr %t300, i64 %t301, ptr %t302, i64 %t303)
  unreachable
div.ok.126:
  %t291 = sdiv i64 %t285, 1000000
  br label %sleep.127
sleep.127:
  %t304 = call i64 @veles_task_sleep(ptr %task, i64 %t291)
  %t305 = icmp ne i64 %t304, 0
  br i1 %t305, label %sleep.done.128, label %sleep.susp.129
sleep.susp.129:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t306 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t306, label %coro.suspend [ i8 0, label %resume.130 i8 1, label %coro.cleanup ]
resume.130:
  %t307 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t308 = icmp ne i64 %t307, 0
  br i1 %t308, label %cancelled.131, label %cont.132
cancelled.131:
  call void @veles_cleanup_pop(ptr %a270)
  %t309 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t309)
  br label %abandon.wait.133
abandon.wait.133:
  %t310 = call i64 @veles_scope_wait(ptr %task, ptr %t309)
  %t311 = icmp ne i64 %t310, 0
  br i1 %t311, label %abandon.done.134, label %abandon.susp.135
abandon.susp.135:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t312 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t312, label %coro.suspend [ i8 0, label %resume.136 i8 1, label %coro.cleanup ]
resume.136:
  br label %abandon.wait.133
abandon.done.134:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.132:
  %t313 = load ptr, ptr %a269
  %t314 = call ptr @veles_scope_failed(ptr %t313)
  %t315 = icmp ne ptr %t314, null
  br i1 %t315, label %scope.abort.137, label %scope.ok.138
scope.ok.138:
  br label %cont.139
scope.abort.137:
  call void @veles_task_leave_waits(ptr %task)
  %t316 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t316, ptr %task)
  br label %scope.wait.114
cont.139:
  br label %sleep.127
sleep.done.128:
  %t317 = load i64, ptr %a267
  %t318 = load ptr, ptr %a266
  store i64 zeroinitializer, ptr %a319
  br label %recv.140
recv.140:
  %t320 = call i64 @veles_chan_recv(ptr %task, ptr %t318, ptr %a319)
  %t321 = icmp ne i64 %t320, 0
  br i1 %t321, label %recv.done.141, label %recv.susp.142
recv.susp.142:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t322 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t322, label %coro.suspend [ i8 0, label %resume.143 i8 1, label %coro.cleanup ]
resume.143:
  %t323 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t324 = icmp ne i64 %t323, 0
  br i1 %t324, label %cancelled.144, label %cont.145
cancelled.144:
  call void @veles_cleanup_pop(ptr %a270)
  %t325 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t325)
  br label %abandon.wait.146
abandon.wait.146:
  %t326 = call i64 @veles_scope_wait(ptr %task, ptr %t325)
  %t327 = icmp ne i64 %t326, 0
  br i1 %t327, label %abandon.done.147, label %abandon.susp.148
abandon.susp.148:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t328 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t328, label %coro.suspend [ i8 0, label %resume.149 i8 1, label %coro.cleanup ]
resume.149:
  br label %abandon.wait.146
abandon.done.147:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.145:
  %t329 = load ptr, ptr %a269
  %t330 = call ptr @veles_scope_failed(ptr %t329)
  %t331 = icmp ne ptr %t330, null
  br i1 %t331, label %scope.abort.150, label %scope.ok.151
scope.ok.151:
  br label %cont.152
scope.abort.150:
  call void @veles_task_leave_waits(ptr %task)
  %t332 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t332, ptr %task)
  br label %scope.wait.114
cont.152:
  br label %recv.140
recv.done.141:
  %t333 = icmp eq i64 %t320, 1
  %t334 = load i64, ptr %a319
  %t335 = insertvalue { i1, i64 } undef, i1 %t333, 0
  %t336 = insertvalue { i1, i64 } %t335, i64 %t334, 1
  %t339 = extractvalue { i1, i64 } %t336, 0
  %t338 = xor i1 %t339, true
  br i1 %t338, label %elvis.default.153, label %elvis.some.154
elvis.some.154:
  %t340 = extractvalue { i1, i64 } %t336, 1
  store i64 %t340, ptr %a337
  br label %elvis.end.155
elvis.default.153:
  store i64 0, ptr %a337
  br label %elvis.end.155
elvis.end.155:
  %t341 = load i64, ptr %a337
  %t343 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t317, i64 %t341)
  %t344 = extractvalue { i64, i1 } %t343, 0
  %t345 = extractvalue { i64, i1 } %t343, 1
  br i1 %t345, label %overflow.156, label %arith.ok.157
overflow.156:
  %t346 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t347 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t348 = extractvalue %str { ptr @.str.20, i64 12 }, 0
  %t349 = extractvalue %str { ptr @.str.20, i64 12 }, 1
  call void @veles_panic_at(ptr %t346, i64 %t347, ptr %t348, i64 %t349)
  unreachable
arith.ok.157:
  store i64 %t344, ptr %a267
  br label %loop.post.118
loop.post.118:
  %t350 = load volatile i32, ptr @veles_attention_line, align 64
  %t351 = icmp ne i32 %t350, 0
  br i1 %t351, label %safepoint.158, label %safepoint.on.159, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.158:
  %t352 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t352, label %safepoint.on.159 [ i64 0, label %backedge.look.162 i64 1, label %backedge.yield.160 i64 2, label %backedge.cancel.161 ]
backedge.look.162:
  %t353 = load ptr, ptr %a269
  %t354 = call ptr @veles_scope_failed(ptr %t353)
  %t355 = icmp ne ptr %t354, null
  br i1 %t355, label %scope.abort.163, label %scope.ok.164
scope.ok.164:
  br label %cont.165
scope.abort.163:
  call void @veles_task_leave_waits(ptr %task)
  %t356 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t356, ptr %task)
  br label %scope.wait.114
cont.165:
  br label %safepoint.on.159
backedge.yield.160:
  br label %yield.166
yield.166:
  %t357 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t358 = icmp ne i64 %t357, 0
  br i1 %t358, label %safepoint.on.159, label %yield.susp.167
yield.susp.167:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t359 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t359, label %coro.suspend [ i8 0, label %resume.168 i8 1, label %coro.cleanup ]
resume.168:
  %t360 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t361 = icmp ne i64 %t360, 0
  br i1 %t361, label %cancelled.169, label %cont.170
cancelled.169:
  call void @veles_cleanup_pop(ptr %a270)
  %t362 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t362)
  br label %abandon.wait.171
abandon.wait.171:
  %t363 = call i64 @veles_scope_wait(ptr %task, ptr %t362)
  %t364 = icmp ne i64 %t363, 0
  br i1 %t364, label %abandon.done.172, label %abandon.susp.173
abandon.susp.173:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t365 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t365, label %coro.suspend [ i8 0, label %resume.174 i8 1, label %coro.cleanup ]
resume.174:
  br label %abandon.wait.171
abandon.done.172:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.170:
  %t366 = load ptr, ptr %a269
  %t367 = call ptr @veles_scope_failed(ptr %t366)
  %t368 = icmp ne ptr %t367, null
  br i1 %t368, label %scope.abort.175, label %scope.ok.176
scope.ok.176:
  br label %cont.177
scope.abort.175:
  call void @veles_task_leave_waits(ptr %task)
  %t369 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t369, ptr %task)
  br label %scope.wait.114
cont.177:
  br label %yield.166
backedge.cancel.161:
  call void @veles_cleanup_pop(ptr %a270)
  %t370 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t370)
  br label %abandon.wait.178
abandon.wait.178:
  %t371 = call i64 @veles_scope_wait(ptr %task, ptr %t370)
  %t372 = icmp ne i64 %t371, 0
  br i1 %t372, label %abandon.done.179, label %abandon.susp.180
abandon.susp.180:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t373 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t373, label %coro.suspend [ i8 0, label %resume.181 i8 1, label %coro.cleanup ]
resume.181:
  br label %abandon.wait.178
abandon.done.179:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.159:
  br label %loop.cond.117
loop.end.119:
  br label %scope.wait.114
scope.wait.114:
  %t374 = load ptr, ptr %a269
  %t375 = call i64 @veles_scope_wait(ptr %task, ptr %t374)
  %t376 = icmp ne i64 %t375, 0
  br i1 %t376, label %scope.done.115, label %scope.susp.116
scope.susp.116:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t377 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t377, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  %t378 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t379 = icmp ne i64 %t378, 0
  br i1 %t379, label %cancelled.183, label %cont.184
cancelled.183:
  call void @veles_cleanup_pop(ptr %a270)
  %t380 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t380)
  br label %abandon.wait.185
abandon.wait.185:
  %t381 = call i64 @veles_scope_wait(ptr %task, ptr %t380)
  %t382 = icmp ne i64 %t381, 0
  br i1 %t382, label %abandon.done.186, label %abandon.susp.187
abandon.susp.187:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t383 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t383, label %coro.suspend [ i8 0, label %resume.188 i8 1, label %coro.cleanup ]
resume.188:
  br label %abandon.wait.185
abandon.done.186:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.184:
  br label %scope.wait.114
scope.done.115:
  call void @veles_cleanup_pop(ptr %a270)
  %t384 = load ptr, ptr %a269
  %t385 = call ptr @veles_scope_failed(ptr %t384)
  %t386 = icmp ne ptr %t385, null
  br i1 %t386, label %scope.check.189, label %scope.after.190
scope.check.189:
  %t387 = call i64 @veles_task_panicked(ptr %t385)
  %t388 = icmp ne i64 %t387, 0
  br i1 %t388, label %scope.repanic.191, label %scope.errors.192
scope.repanic.191:
  call void @veles_task_repanic(ptr %t385)
  unreachable
scope.errors.192:
  br label %scope.after.190
scope.after.190:
  %t389 = load i64, ptr %a267
  %t391 = call i64 @veles_i64_format(ptr %a390, i64 %t389)
  %t392 = insertvalue %str undef, ptr %a390, 0
  %t393 = insertvalue %str %t392, i64 %t391, 1
  %t395 = extractvalue %str { ptr @.str.21, i64 6 }, 0
  %t396 = extractvalue %str { ptr @.str.21, i64 6 }, 1
  %t397 = extractvalue %str %t393, 0
  %t398 = extractvalue %str %t393, 1
  call void @veles_string_concat(ptr %a394, ptr %t395, i64 %t396, ptr %t397, i64 %t398)
  %t399 = load %str, ptr %a394
  %t400 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t401 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t400)
  call void @v_std.io.println(%str %t399)
  call void @veles_call_pop()
  %t402 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t402, ptr %a403
  store i64 0, ptr %a404
  %t405 = call ptr @veles_scope_begin(ptr %task, i64 1, i64 %coro.depth)
  store ptr %t405, ptr %a406
  call void @veles_cleanup_push(ptr %a407, ptr @scope.cancel.thunk, ptr %a406, ptr @cleanup.move.word)
  %t408 = load ptr, ptr %a406
  %t409 = call ptr @veles_task_launch(ptr %t408, i64 0)
  %t410 = load ptr, ptr %a403
  %t411 = call ptr @veles_alloc_words(i64 16)
  %t412 = getelementptr inbounds { ptr }, ptr %t411, i32 0, i32 0
  store ptr %t410, ptr %t412
  call void @veles_task_spawn(ptr %t409, ptr @entry.v_main.produce, ptr %t411)
  store ptr %t409, ptr %a413
  br label %loop.cond.196
loop.cond.196:
  %t414 = load i64, ptr %a404
  %t415 = icmp slt i64 %t414, 6
  br i1 %t415, label %loop.body.199, label %loop.end.198
loop.body.199:
  %t416 = load ptr, ptr %a403
  store i64 zeroinitializer, ptr %a417
  %t418 = call i64 @veles_chan_try_recv(ptr %t416, ptr %a417)
  %t419 = icmp ne i64 %t418, 0
  %t420 = load i64, ptr %a417
  %t421 = insertvalue { i1, i64 } undef, i1 %t419, 0
  %t422 = insertvalue { i1, i64 } %t421, i64 %t420, 1
  store { i1, i64 } %t422, ptr %a423
  %t424 = load { i1, i64 }, ptr %a423
  %t426 = extractvalue { i1, i64 } %t424, 0
  %t425 = xor i1 %t426, true
  %t427 = xor i1 %t425, true
  br i1 %t427, label %if.then.200, label %if.else.202
if.then.200:
  %t428 = load i64, ptr %a404
  %t429 = load { i1, i64 }, ptr %a423
  %t430 = extractvalue { i1, i64 } %t429, 1
  %t432 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t428, i64 %t430)
  %t433 = extractvalue { i64, i1 } %t432, 0
  %t434 = extractvalue { i64, i1 } %t432, 1
  br i1 %t434, label %overflow.203, label %arith.ok.204
overflow.203:
  %t435 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t436 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t437 = extractvalue %str { ptr @.str.23, i64 13 }, 0
  %t438 = extractvalue %str { ptr @.str.23, i64 13 }, 1
  call void @veles_panic_at(ptr %t435, i64 %t436, ptr %t437, i64 %t438)
  unreachable
arith.ok.204:
  store i64 %t433, ptr %a404
  br label %if.end.201
if.else.202:
  %t439 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t440 = load i64, ptr %t439
  %t442 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t440, i64 999999)
  %t443 = extractvalue { i64, i1 } %t442, 0
  %t444 = extractvalue { i64, i1 } %t442, 1
  br i1 %t444, label %overflow.205, label %arith.ok.206
overflow.205:
  %t445 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t446 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t447 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t448 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t445, i64 %t446, ptr %t447, i64 %t448)
  unreachable
arith.ok.206:
  %t450 = icmp eq i64 1000000, 0
  br i1 %t450, label %divzero.207, label %div.ok.208
divzero.207:
  %t451 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t452 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t453 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t454 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t451, i64 %t452, ptr %t453, i64 %t454)
  unreachable
div.ok.208:
  %t455 = icmp eq i64 %t443, -9223372036854775808
  %t456 = icmp eq i64 1000000, -1
  %t457 = and i1 %t455, %t456
  br i1 %t457, label %divof.209, label %div.ok.210
divof.209:
  %t458 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t459 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t460 = extractvalue %str { ptr @.str.24, i64 13 }, 0
  %t461 = extractvalue %str { ptr @.str.24, i64 13 }, 1
  call void @veles_panic_at(ptr %t458, i64 %t459, ptr %t460, i64 %t461)
  unreachable
div.ok.210:
  %t449 = sdiv i64 %t443, 1000000
  br label %sleep.211
sleep.211:
  %t462 = call i64 @veles_task_sleep(ptr %task, i64 %t449)
  %t463 = icmp ne i64 %t462, 0
  br i1 %t463, label %sleep.done.212, label %sleep.susp.213
sleep.susp.213:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t464 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t464, label %coro.suspend [ i8 0, label %resume.214 i8 1, label %coro.cleanup ]
resume.214:
  %t465 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t466 = icmp ne i64 %t465, 0
  br i1 %t466, label %cancelled.215, label %cont.216
cancelled.215:
  call void @veles_cleanup_pop(ptr %a407)
  %t467 = load ptr, ptr %a406
  call void @veles_scope_cancel(ptr %t467)
  br label %abandon.wait.217
abandon.wait.217:
  %t468 = call i64 @veles_scope_wait(ptr %task, ptr %t467)
  %t469 = icmp ne i64 %t468, 0
  br i1 %t469, label %abandon.done.218, label %abandon.susp.219
abandon.susp.219:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t470 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t470, label %coro.suspend [ i8 0, label %resume.220 i8 1, label %coro.cleanup ]
resume.220:
  br label %abandon.wait.217
abandon.done.218:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.216:
  %t471 = load ptr, ptr %a406
  %t472 = call ptr @veles_scope_failed(ptr %t471)
  %t473 = icmp ne ptr %t472, null
  br i1 %t473, label %scope.abort.221, label %scope.ok.222
scope.ok.222:
  br label %cont.223
scope.abort.221:
  call void @veles_task_leave_waits(ptr %task)
  %t474 = load ptr, ptr %a406
  call void @veles_scope_abandon(ptr %t474, ptr %task)
  br label %scope.wait.193
cont.223:
  br label %sleep.211
sleep.done.212:
  br label %if.end.201
if.end.201:
  br label %loop.post.197
loop.post.197:
  %t475 = load volatile i32, ptr @veles_attention_line, align 64
  %t476 = icmp ne i32 %t475, 0
  br i1 %t476, label %safepoint.224, label %safepoint.on.225, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.224:
  %t477 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t477, label %safepoint.on.225 [ i64 0, label %backedge.look.228 i64 1, label %backedge.yield.226 i64 2, label %backedge.cancel.227 ]
backedge.look.228:
  %t478 = load ptr, ptr %a406
  %t479 = call ptr @veles_scope_failed(ptr %t478)
  %t480 = icmp ne ptr %t479, null
  br i1 %t480, label %scope.abort.229, label %scope.ok.230
scope.ok.230:
  br label %cont.231
scope.abort.229:
  call void @veles_task_leave_waits(ptr %task)
  %t481 = load ptr, ptr %a406
  call void @veles_scope_abandon(ptr %t481, ptr %task)
  br label %scope.wait.193
cont.231:
  br label %safepoint.on.225
backedge.yield.226:
  br label %yield.232
yield.232:
  %t482 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t483 = icmp ne i64 %t482, 0
  br i1 %t483, label %safepoint.on.225, label %yield.susp.233
yield.susp.233:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t484 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t484, label %coro.suspend [ i8 0, label %resume.234 i8 1, label %coro.cleanup ]
resume.234:
  %t485 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t486 = icmp ne i64 %t485, 0
  br i1 %t486, label %cancelled.235, label %cont.236
cancelled.235:
  call void @veles_cleanup_pop(ptr %a407)
  %t487 = load ptr, ptr %a406
  call void @veles_scope_cancel(ptr %t487)
  br label %abandon.wait.237
abandon.wait.237:
  %t488 = call i64 @veles_scope_wait(ptr %task, ptr %t487)
  %t489 = icmp ne i64 %t488, 0
  br i1 %t489, label %abandon.done.238, label %abandon.susp.239
abandon.susp.239:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t490 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t490, label %coro.suspend [ i8 0, label %resume.240 i8 1, label %coro.cleanup ]
resume.240:
  br label %abandon.wait.237
abandon.done.238:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.236:
  %t491 = load ptr, ptr %a406
  %t492 = call ptr @veles_scope_failed(ptr %t491)
  %t493 = icmp ne ptr %t492, null
  br i1 %t493, label %scope.abort.241, label %scope.ok.242
scope.ok.242:
  br label %cont.243
scope.abort.241:
  call void @veles_task_leave_waits(ptr %task)
  %t494 = load ptr, ptr %a406
  call void @veles_scope_abandon(ptr %t494, ptr %task)
  br label %scope.wait.193
cont.243:
  br label %yield.232
backedge.cancel.227:
  call void @veles_cleanup_pop(ptr %a407)
  %t495 = load ptr, ptr %a406
  call void @veles_scope_cancel(ptr %t495)
  br label %abandon.wait.244
abandon.wait.244:
  %t496 = call i64 @veles_scope_wait(ptr %task, ptr %t495)
  %t497 = icmp ne i64 %t496, 0
  br i1 %t497, label %abandon.done.245, label %abandon.susp.246
abandon.susp.246:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t498 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t498, label %coro.suspend [ i8 0, label %resume.247 i8 1, label %coro.cleanup ]
resume.247:
  br label %abandon.wait.244
abandon.done.245:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.225:
  br label %loop.cond.196
loop.end.198:
  br label %scope.wait.193
scope.wait.193:
  %t499 = load ptr, ptr %a406
  %t500 = call i64 @veles_scope_wait(ptr %task, ptr %t499)
  %t501 = icmp ne i64 %t500, 0
  br i1 %t501, label %scope.done.194, label %scope.susp.195
scope.susp.195:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t502 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t502, label %coro.suspend [ i8 0, label %resume.248 i8 1, label %coro.cleanup ]
resume.248:
  %t503 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t504 = icmp ne i64 %t503, 0
  br i1 %t504, label %cancelled.249, label %cont.250
cancelled.249:
  call void @veles_cleanup_pop(ptr %a407)
  %t505 = load ptr, ptr %a406
  call void @veles_scope_cancel(ptr %t505)
  br label %abandon.wait.251
abandon.wait.251:
  %t506 = call i64 @veles_scope_wait(ptr %task, ptr %t505)
  %t507 = icmp ne i64 %t506, 0
  br i1 %t507, label %abandon.done.252, label %abandon.susp.253
abandon.susp.253:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t508 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t508, label %coro.suspend [ i8 0, label %resume.254 i8 1, label %coro.cleanup ]
resume.254:
  br label %abandon.wait.251
abandon.done.252:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.250:
  br label %scope.wait.193
scope.done.194:
  call void @veles_cleanup_pop(ptr %a407)
  %t509 = load ptr, ptr %a406
  %t510 = call ptr @veles_scope_failed(ptr %t509)
  %t511 = icmp ne ptr %t510, null
  br i1 %t511, label %scope.check.255, label %scope.after.256
scope.check.255:
  %t512 = call i64 @veles_task_panicked(ptr %t510)
  %t513 = icmp ne i64 %t512, 0
  br i1 %t513, label %scope.repanic.257, label %scope.errors.258
scope.repanic.257:
  call void @veles_task_repanic(ptr %t510)
  unreachable
scope.errors.258:
  br label %scope.after.256
scope.after.256:
  %t514 = load i64, ptr %a404
  %t516 = call i64 @veles_i64_format(ptr %a515, i64 %t514)
  %t517 = insertvalue %str undef, ptr %a515, 0
  %t518 = insertvalue %str %t517, i64 %t516, 1
  %t520 = extractvalue %str { ptr @.str.25, i64 7 }, 0
  %t521 = extractvalue %str { ptr @.str.25, i64 7 }, 1
  %t522 = extractvalue %str %t518, 0
  %t523 = extractvalue %str %t518, 1
  call void @veles_string_concat(ptr %a519, ptr %t520, i64 %t521, ptr %t522, i64 %t523)
  %t524 = load %str, ptr %a519
  %t525 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t526 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t525)
  call void @v_std.io.println(%str %t524)
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
  %t5 = call ptr @v_main.slow(ptr %task, ptr @veles.root.link, i64 %t2)
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
  %t5 = call ptr @v_main.produce(ptr %task, ptr @veles.root.link, ptr %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.31, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.31, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task, ptr @veles.root.link)
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
