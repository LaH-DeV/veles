%S.main.Resource = type { %str }
define ptr @v_main.slow(ptr %task, ptr %link, i64 %p1) presplitcoroutine {
entry:
  %a1 = alloca i64
  %a20 = alloca i64
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
  br label %sleep.1
sleep.1:
  %t6 = call i64 @veles_task_sleep(ptr %task, i64 %t5)
  %t7 = icmp ne i64 %t6, 0
  br i1 %t7, label %sleep.done.2, label %sleep.susp.3
sleep.susp.3:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t8 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t8, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t9 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t10 = icmp ne i64 %t9, 0
  br i1 %t10, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.6:
  br label %sleep.1
sleep.done.2:
  %t11 = load i64, ptr %a1
  %t13 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t11, i64 2)
  %t14 = extractvalue { i64, i1 } %t13, 0
  %t15 = extractvalue { i64, i1 } %t13, 1
  br i1 %t15, label %overflow.7, label %arith.ok.8
overflow.7:
  %t16 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t17 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t18 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t19 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_panic_at(ptr %t16, i64 %t17, ptr %t18, i64 %t19)
  unreachable
arith.ok.8:
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a20)
  store i64 %t14, ptr %a20
  %t21 = load ptr, ptr %link
  %t22 = icmp eq ptr %t21, null
  br i1 %t22, label %ret.task.9, label %ret.call.10
ret.task.9:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a20, i64 8, i64 0, ptr null)
  br label %coro.final
ret.call.10:
  %t23 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %link, i32 0, i32 3
  store i64 %t14, ptr %t23
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
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a13)
  %t14 = load i1, ptr %a12
  br i1 %t14, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a13
  br label %if.end.6
if.else.7:
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a15)
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
  %a298 = alloca i64
  %a316 = alloca i64
  %a369 = alloca [21 x i8]
  %a373 = alloca %str
  %a382 = alloca ptr
  %a383 = alloca i64
  %a385 = alloca ptr
  %a386 = alloca { ptr, ptr, ptr, ptr }
  %a392 = alloca ptr
  %a396 = alloca i64
  %a402 = alloca { i1, i64 }
  %a473 = alloca [21 x i8]
  %a477 = alloca %str
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t1 = insertvalue %S.main.Resource undef, %str { ptr @.str.4, i64 2 }, 0
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
  %t29 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t30 = extractvalue %str { ptr @.str.5, i64 18 }, 1
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
  %t53 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t54 = extractvalue %str { ptr @.str.5, i64 18 }, 1
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
  store %str { ptr @.str.6, i64 1 }, ptr %t70
  %t71 = getelementptr [3 x %str], ptr %a68, i64 0, i64 2
  store %str %t66, ptr %t71
  call void @veles_string_concat_n(ptr %a67, ptr %a68, i64 3)
  %t72 = load %str, ptr %a67
  %t73 = extractvalue %str { ptr @.str.7, i64 20 }, 0
  %t74 = extractvalue %str { ptr @.str.7, i64 20 }, 1
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
  %t85 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t86 = extractvalue %str { ptr @.str.5, i64 18 }, 1
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
  %t92 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t93 = extractvalue %str { ptr @.str.5, i64 18 }, 1
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
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a125)
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
  %t139 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t140 = extractvalue %str { ptr @.str.8, i64 12 }, 1
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
  %t186 = extractvalue %str { ptr @.str.9, i64 4 }, 0
  %t187 = extractvalue %str { ptr @.str.9, i64 4 }, 1
  %t188 = extractvalue %str %t184, 0
  %t189 = extractvalue %str %t184, 1
  call void @veles_string_concat(ptr %a185, ptr %t186, i64 %t187, ptr %t188, i64 %t189)
  %t190 = load %str, ptr %a185
  %t191 = extractvalue %str { ptr @.str.10, i64 20 }, 0
  %t192 = extractvalue %str { ptr @.str.10, i64 20 }, 1
  call void @veles_call_push(ptr %t191)
  call void @v_std.io.println(%str %t190)
  call void @veles_call_pop()
  %t193 = call ptr @veles_chan_new(ptr @adesc.string, i64 2)
  store ptr %t193, ptr %a194
  %t195 = load ptr, ptr %a194
  store %str { ptr @.str.11, i64 1 }, ptr %a196
  %t197 = call i64 @veles_chan_try_send(ptr %t195, ptr %a196)
  %t198 = icmp ne i64 %t197, 0
  call void @veles_bool_to_string(ptr %a199, i1 zeroext %t198)
  %t200 = load %str, ptr %a199
  %t201 = load ptr, ptr %a194
  store %str { ptr @.str.12, i64 1 }, ptr %a202
  %t203 = call i64 @veles_chan_try_send(ptr %t201, ptr %a202)
  %t204 = icmp ne i64 %t203, 0
  call void @veles_bool_to_string(ptr %a205, i1 zeroext %t204)
  %t206 = load %str, ptr %a205
  %t207 = load ptr, ptr %a194
  store %str { ptr @.str.13, i64 1 }, ptr %a208
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
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a220)
  %t222 = extractvalue { i1, %str } %t219, 0
  %t221 = xor i1 %t222, true
  br i1 %t221, label %elvis.default.105, label %elvis.some.106
elvis.some.106:
  %t223 = extractvalue { i1, %str } %t219, 1
  store %str %t223, ptr %a220
  br label %elvis.end.107
elvis.default.105:
  store %str { ptr @.str.14, i64 1 }, ptr %a220
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
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a232)
  %t234 = extractvalue { i1, %str } %t231, 0
  %t233 = xor i1 %t234, true
  br i1 %t233, label %elvis.default.108, label %elvis.some.109
elvis.some.109:
  %t235 = extractvalue { i1, %str } %t231, 1
  store %str %t235, ptr %a232
  br label %elvis.end.110
elvis.default.108:
  store %str { ptr @.str.14, i64 1 }, ptr %a232
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
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a244)
  %t246 = extractvalue { i1, %str } %t243, 0
  %t245 = xor i1 %t246, true
  br i1 %t245, label %elvis.default.111, label %elvis.some.112
elvis.some.112:
  %t247 = extractvalue { i1, %str } %t243, 1
  store %str %t247, ptr %a244
  br label %elvis.end.113
elvis.default.111:
  store %str { ptr @.str.14, i64 1 }, ptr %a244
  br label %elvis.end.113
elvis.end.113:
  %t248 = load %str, ptr %a244
  %t251 = getelementptr [11 x %str], ptr %a250, i64 0, i64 0
  store %str %t200, ptr %t251
  %t252 = getelementptr [11 x %str], ptr %a250, i64 0, i64 1
  store %str { ptr @.str.6, i64 1 }, ptr %t252
  %t253 = getelementptr [11 x %str], ptr %a250, i64 0, i64 2
  store %str %t206, ptr %t253
  %t254 = getelementptr [11 x %str], ptr %a250, i64 0, i64 3
  store %str { ptr @.str.6, i64 1 }, ptr %t254
  %t255 = getelementptr [11 x %str], ptr %a250, i64 0, i64 4
  store %str %t212, ptr %t255
  %t256 = getelementptr [11 x %str], ptr %a250, i64 0, i64 5
  store %str { ptr @.str.6, i64 1 }, ptr %t256
  %t257 = getelementptr [11 x %str], ptr %a250, i64 0, i64 6
  store %str %t224, ptr %t257
  %t258 = getelementptr [11 x %str], ptr %a250, i64 0, i64 7
  store %str { ptr @.str.6, i64 1 }, ptr %t258
  %t259 = getelementptr [11 x %str], ptr %a250, i64 0, i64 8
  store %str %t236, ptr %t259
  %t260 = getelementptr [11 x %str], ptr %a250, i64 0, i64 9
  store %str { ptr @.str.6, i64 1 }, ptr %t260
  %t261 = getelementptr [11 x %str], ptr %a250, i64 0, i64 10
  store %str %t248, ptr %t261
  call void @veles_string_concat_n(ptr %a249, ptr %a250, i64 11)
  %t262 = load %str, ptr %a249
  %t263 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t264 = extractvalue %str { ptr @.str.15, i64 20 }, 1
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
  %t279 = extractvalue %str { ptr @.str.16, i64 20 }, 0
  %t280 = extractvalue %str { ptr @.str.16, i64 20 }, 1
  call void @veles_call_push(ptr %t279)
  %t281 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t282 = extractvalue %S.std.prelude.Duration %t281, 0
  br label %sleep.121
sleep.121:
  %t283 = call i64 @veles_task_sleep(ptr %task, i64 %t282)
  %t284 = icmp ne i64 %t283, 0
  br i1 %t284, label %sleep.done.122, label %sleep.susp.123
sleep.susp.123:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t285 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t285, label %coro.suspend [ i8 0, label %resume.124 i8 1, label %coro.cleanup ]
resume.124:
  %t286 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t287 = icmp ne i64 %t286, 0
  br i1 %t287, label %cancelled.125, label %cont.126
cancelled.125:
  call void @veles_cleanup_pop(ptr %a270)
  %t288 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t288)
  br label %abandon.wait.127
abandon.wait.127:
  %t289 = call i64 @veles_scope_wait(ptr %task, ptr %t288)
  %t290 = icmp ne i64 %t289, 0
  br i1 %t290, label %abandon.done.128, label %abandon.susp.129
abandon.susp.129:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t291 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t291, label %coro.suspend [ i8 0, label %resume.130 i8 1, label %coro.cleanup ]
resume.130:
  br label %abandon.wait.127
abandon.done.128:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.126:
  %t292 = load ptr, ptr %a269
  %t293 = call ptr @veles_scope_failed(ptr %t292)
  %t294 = icmp ne ptr %t293, null
  br i1 %t294, label %scope.abort.131, label %scope.ok.132
scope.ok.132:
  br label %cont.133
scope.abort.131:
  call void @veles_task_leave_waits(ptr %task)
  %t295 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t295, ptr %task)
  br label %scope.wait.114
cont.133:
  br label %sleep.121
sleep.done.122:
  %t296 = load i64, ptr %a267
  %t297 = load ptr, ptr %a266
  store i64 zeroinitializer, ptr %a298
  br label %recv.134
recv.134:
  %t299 = call i64 @veles_chan_recv(ptr %task, ptr %t297, ptr %a298)
  %t300 = icmp ne i64 %t299, 0
  br i1 %t300, label %recv.done.135, label %recv.susp.136
recv.susp.136:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t301 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t301, label %coro.suspend [ i8 0, label %resume.137 i8 1, label %coro.cleanup ]
resume.137:
  %t302 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t303 = icmp ne i64 %t302, 0
  br i1 %t303, label %cancelled.138, label %cont.139
cancelled.138:
  call void @veles_cleanup_pop(ptr %a270)
  %t304 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t304)
  br label %abandon.wait.140
abandon.wait.140:
  %t305 = call i64 @veles_scope_wait(ptr %task, ptr %t304)
  %t306 = icmp ne i64 %t305, 0
  br i1 %t306, label %abandon.done.141, label %abandon.susp.142
abandon.susp.142:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t307 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t307, label %coro.suspend [ i8 0, label %resume.143 i8 1, label %coro.cleanup ]
resume.143:
  br label %abandon.wait.140
abandon.done.141:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.139:
  %t308 = load ptr, ptr %a269
  %t309 = call ptr @veles_scope_failed(ptr %t308)
  %t310 = icmp ne ptr %t309, null
  br i1 %t310, label %scope.abort.144, label %scope.ok.145
scope.ok.145:
  br label %cont.146
scope.abort.144:
  call void @veles_task_leave_waits(ptr %task)
  %t311 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t311, ptr %task)
  br label %scope.wait.114
cont.146:
  br label %recv.134
recv.done.135:
  %t312 = icmp eq i64 %t299, 1
  %t313 = load i64, ptr %a298
  %t314 = insertvalue { i1, i64 } undef, i1 %t312, 0
  %t315 = insertvalue { i1, i64 } %t314, i64 %t313, 1
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a316)
  %t318 = extractvalue { i1, i64 } %t315, 0
  %t317 = xor i1 %t318, true
  br i1 %t317, label %elvis.default.147, label %elvis.some.148
elvis.some.148:
  %t319 = extractvalue { i1, i64 } %t315, 1
  store i64 %t319, ptr %a316
  br label %elvis.end.149
elvis.default.147:
  store i64 0, ptr %a316
  br label %elvis.end.149
elvis.end.149:
  %t320 = load i64, ptr %a316
  %t322 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t296, i64 %t320)
  %t323 = extractvalue { i64, i1 } %t322, 0
  %t324 = extractvalue { i64, i1 } %t322, 1
  br i1 %t324, label %overflow.150, label %arith.ok.151
overflow.150:
  %t325 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t326 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t327 = extractvalue %str { ptr @.str.17, i64 12 }, 0
  %t328 = extractvalue %str { ptr @.str.17, i64 12 }, 1
  call void @veles_panic_at(ptr %t325, i64 %t326, ptr %t327, i64 %t328)
  unreachable
arith.ok.151:
  store i64 %t323, ptr %a267
  br label %loop.post.118
loop.post.118:
  %t329 = load volatile i32, ptr @veles_attention_line, align 64
  %t330 = icmp ne i32 %t329, 0
  br i1 %t330, label %safepoint.152, label %safepoint.on.153, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.152:
  %t331 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t331, label %safepoint.on.153 [ i64 0, label %backedge.look.156 i64 1, label %backedge.yield.154 i64 2, label %backedge.cancel.155 ]
backedge.look.156:
  %t332 = load ptr, ptr %a269
  %t333 = call ptr @veles_scope_failed(ptr %t332)
  %t334 = icmp ne ptr %t333, null
  br i1 %t334, label %scope.abort.157, label %scope.ok.158
scope.ok.158:
  br label %cont.159
scope.abort.157:
  call void @veles_task_leave_waits(ptr %task)
  %t335 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t335, ptr %task)
  br label %scope.wait.114
cont.159:
  br label %safepoint.on.153
backedge.yield.154:
  br label %yield.160
yield.160:
  %t336 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t337 = icmp ne i64 %t336, 0
  br i1 %t337, label %safepoint.on.153, label %yield.susp.161
yield.susp.161:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t338 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t338, label %coro.suspend [ i8 0, label %resume.162 i8 1, label %coro.cleanup ]
resume.162:
  %t339 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t340 = icmp ne i64 %t339, 0
  br i1 %t340, label %cancelled.163, label %cont.164
cancelled.163:
  call void @veles_cleanup_pop(ptr %a270)
  %t341 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t341)
  br label %abandon.wait.165
abandon.wait.165:
  %t342 = call i64 @veles_scope_wait(ptr %task, ptr %t341)
  %t343 = icmp ne i64 %t342, 0
  br i1 %t343, label %abandon.done.166, label %abandon.susp.167
abandon.susp.167:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t344 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t344, label %coro.suspend [ i8 0, label %resume.168 i8 1, label %coro.cleanup ]
resume.168:
  br label %abandon.wait.165
abandon.done.166:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.164:
  %t345 = load ptr, ptr %a269
  %t346 = call ptr @veles_scope_failed(ptr %t345)
  %t347 = icmp ne ptr %t346, null
  br i1 %t347, label %scope.abort.169, label %scope.ok.170
scope.ok.170:
  br label %cont.171
scope.abort.169:
  call void @veles_task_leave_waits(ptr %task)
  %t348 = load ptr, ptr %a269
  call void @veles_scope_abandon(ptr %t348, ptr %task)
  br label %scope.wait.114
cont.171:
  br label %yield.160
backedge.cancel.155:
  call void @veles_cleanup_pop(ptr %a270)
  %t349 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t349)
  br label %abandon.wait.172
abandon.wait.172:
  %t350 = call i64 @veles_scope_wait(ptr %task, ptr %t349)
  %t351 = icmp ne i64 %t350, 0
  br i1 %t351, label %abandon.done.173, label %abandon.susp.174
abandon.susp.174:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t352 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t352, label %coro.suspend [ i8 0, label %resume.175 i8 1, label %coro.cleanup ]
resume.175:
  br label %abandon.wait.172
abandon.done.173:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.153:
  br label %loop.cond.117
loop.end.119:
  br label %scope.wait.114
scope.wait.114:
  %t353 = load ptr, ptr %a269
  %t354 = call i64 @veles_scope_wait(ptr %task, ptr %t353)
  %t355 = icmp ne i64 %t354, 0
  br i1 %t355, label %scope.done.115, label %scope.susp.116
scope.susp.116:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t356 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t356, label %coro.suspend [ i8 0, label %resume.176 i8 1, label %coro.cleanup ]
resume.176:
  %t357 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t358 = icmp ne i64 %t357, 0
  br i1 %t358, label %cancelled.177, label %cont.178
cancelled.177:
  call void @veles_cleanup_pop(ptr %a270)
  %t359 = load ptr, ptr %a269
  call void @veles_scope_cancel(ptr %t359)
  br label %abandon.wait.179
abandon.wait.179:
  %t360 = call i64 @veles_scope_wait(ptr %task, ptr %t359)
  %t361 = icmp ne i64 %t360, 0
  br i1 %t361, label %abandon.done.180, label %abandon.susp.181
abandon.susp.181:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t362 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t362, label %coro.suspend [ i8 0, label %resume.182 i8 1, label %coro.cleanup ]
resume.182:
  br label %abandon.wait.179
abandon.done.180:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.178:
  br label %scope.wait.114
scope.done.115:
  call void @veles_cleanup_pop(ptr %a270)
  %t363 = load ptr, ptr %a269
  %t364 = call ptr @veles_scope_failed(ptr %t363)
  %t365 = icmp ne ptr %t364, null
  br i1 %t365, label %scope.check.183, label %scope.after.184
scope.check.183:
  %t366 = call i64 @veles_task_panicked(ptr %t364)
  %t367 = icmp ne i64 %t366, 0
  br i1 %t367, label %scope.repanic.185, label %scope.errors.186
scope.repanic.185:
  call void @veles_task_repanic(ptr %t364)
  unreachable
scope.errors.186:
  br label %scope.after.184
scope.after.184:
  %t368 = load i64, ptr %a267
  %t370 = call i64 @veles_i64_format(ptr %a369, i64 %t368)
  %t371 = insertvalue %str undef, ptr %a369, 0
  %t372 = insertvalue %str %t371, i64 %t370, 1
  %t374 = extractvalue %str { ptr @.str.18, i64 6 }, 0
  %t375 = extractvalue %str { ptr @.str.18, i64 6 }, 1
  %t376 = extractvalue %str %t372, 0
  %t377 = extractvalue %str %t372, 1
  call void @veles_string_concat(ptr %a373, ptr %t374, i64 %t375, ptr %t376, i64 %t377)
  %t378 = load %str, ptr %a373
  %t379 = extractvalue %str { ptr @.str.19, i64 20 }, 0
  %t380 = extractvalue %str { ptr @.str.19, i64 20 }, 1
  call void @veles_call_push(ptr %t379)
  call void @v_std.io.println(%str %t378)
  call void @veles_call_pop()
  %t381 = call ptr @veles_chan_new(ptr @adesc.i64, i64 1)
  store ptr %t381, ptr %a382
  store i64 0, ptr %a383
  %t384 = call ptr @veles_scope_begin(ptr %task, i64 1, i64 %coro.depth)
  store ptr %t384, ptr %a385
  call void @veles_cleanup_push(ptr %a386, ptr @scope.cancel.thunk, ptr %a385, ptr @cleanup.move.word)
  %t387 = load ptr, ptr %a385
  %t388 = call ptr @veles_task_launch(ptr %t387, i64 0)
  %t389 = load ptr, ptr %a382
  %t390 = call ptr @veles_alloc_words(i64 16)
  %t391 = getelementptr inbounds { ptr }, ptr %t390, i32 0, i32 0
  store ptr %t389, ptr %t391
  call void @veles_task_spawn(ptr %t388, ptr @entry.v_main.produce, ptr %t390)
  store ptr %t388, ptr %a392
  br label %loop.cond.190
loop.cond.190:
  %t393 = load i64, ptr %a383
  %t394 = icmp slt i64 %t393, 6
  br i1 %t394, label %loop.body.193, label %loop.end.192
loop.body.193:
  %t395 = load ptr, ptr %a382
  store i64 zeroinitializer, ptr %a396
  %t397 = call i64 @veles_chan_try_recv(ptr %t395, ptr %a396)
  %t398 = icmp ne i64 %t397, 0
  %t399 = load i64, ptr %a396
  %t400 = insertvalue { i1, i64 } undef, i1 %t398, 0
  %t401 = insertvalue { i1, i64 } %t400, i64 %t399, 1
  store { i1, i64 } %t401, ptr %a402
  %t403 = load { i1, i64 }, ptr %a402
  %t405 = extractvalue { i1, i64 } %t403, 0
  %t404 = xor i1 %t405, true
  %t406 = xor i1 %t404, true
  br i1 %t406, label %if.then.194, label %if.else.196
if.then.194:
  %t407 = load i64, ptr %a383
  %t408 = load { i1, i64 }, ptr %a402
  %t409 = extractvalue { i1, i64 } %t408, 1
  %t411 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t407, i64 %t409)
  %t412 = extractvalue { i64, i1 } %t411, 0
  %t413 = extractvalue { i64, i1 } %t411, 1
  br i1 %t413, label %overflow.197, label %arith.ok.198
overflow.197:
  %t414 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t415 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t416 = extractvalue %str { ptr @.str.20, i64 13 }, 0
  %t417 = extractvalue %str { ptr @.str.20, i64 13 }, 1
  call void @veles_panic_at(ptr %t414, i64 %t415, ptr %t416, i64 %t417)
  unreachable
arith.ok.198:
  store i64 %t412, ptr %a383
  br label %if.end.195
if.else.196:
  %t418 = getelementptr inbounds %S.std.prelude.Duration, ptr @g_std_prelude_Duration_zero, i32 0, i32 0
  %t419 = load i64, ptr %t418
  br label %sleep.199
sleep.199:
  %t420 = call i64 @veles_task_sleep(ptr %task, i64 %t419)
  %t421 = icmp ne i64 %t420, 0
  br i1 %t421, label %sleep.done.200, label %sleep.susp.201
sleep.susp.201:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t422 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t422, label %coro.suspend [ i8 0, label %resume.202 i8 1, label %coro.cleanup ]
resume.202:
  %t423 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t424 = icmp ne i64 %t423, 0
  br i1 %t424, label %cancelled.203, label %cont.204
cancelled.203:
  call void @veles_cleanup_pop(ptr %a386)
  %t425 = load ptr, ptr %a385
  call void @veles_scope_cancel(ptr %t425)
  br label %abandon.wait.205
abandon.wait.205:
  %t426 = call i64 @veles_scope_wait(ptr %task, ptr %t425)
  %t427 = icmp ne i64 %t426, 0
  br i1 %t427, label %abandon.done.206, label %abandon.susp.207
abandon.susp.207:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t428 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t428, label %coro.suspend [ i8 0, label %resume.208 i8 1, label %coro.cleanup ]
resume.208:
  br label %abandon.wait.205
abandon.done.206:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.204:
  %t429 = load ptr, ptr %a385
  %t430 = call ptr @veles_scope_failed(ptr %t429)
  %t431 = icmp ne ptr %t430, null
  br i1 %t431, label %scope.abort.209, label %scope.ok.210
scope.ok.210:
  br label %cont.211
scope.abort.209:
  call void @veles_task_leave_waits(ptr %task)
  %t432 = load ptr, ptr %a385
  call void @veles_scope_abandon(ptr %t432, ptr %task)
  br label %scope.wait.187
cont.211:
  br label %sleep.199
sleep.done.200:
  br label %if.end.195
if.end.195:
  br label %loop.post.191
loop.post.191:
  %t433 = load volatile i32, ptr @veles_attention_line, align 64
  %t434 = icmp ne i32 %t433, 0
  br i1 %t434, label %safepoint.212, label %safepoint.on.213, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.212:
  %t435 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t435, label %safepoint.on.213 [ i64 0, label %backedge.look.216 i64 1, label %backedge.yield.214 i64 2, label %backedge.cancel.215 ]
backedge.look.216:
  %t436 = load ptr, ptr %a385
  %t437 = call ptr @veles_scope_failed(ptr %t436)
  %t438 = icmp ne ptr %t437, null
  br i1 %t438, label %scope.abort.217, label %scope.ok.218
scope.ok.218:
  br label %cont.219
scope.abort.217:
  call void @veles_task_leave_waits(ptr %task)
  %t439 = load ptr, ptr %a385
  call void @veles_scope_abandon(ptr %t439, ptr %task)
  br label %scope.wait.187
cont.219:
  br label %safepoint.on.213
backedge.yield.214:
  br label %yield.220
yield.220:
  %t440 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t441 = icmp ne i64 %t440, 0
  br i1 %t441, label %safepoint.on.213, label %yield.susp.221
yield.susp.221:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t442 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t442, label %coro.suspend [ i8 0, label %resume.222 i8 1, label %coro.cleanup ]
resume.222:
  %t443 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t444 = icmp ne i64 %t443, 0
  br i1 %t444, label %cancelled.223, label %cont.224
cancelled.223:
  call void @veles_cleanup_pop(ptr %a386)
  %t445 = load ptr, ptr %a385
  call void @veles_scope_cancel(ptr %t445)
  br label %abandon.wait.225
abandon.wait.225:
  %t446 = call i64 @veles_scope_wait(ptr %task, ptr %t445)
  %t447 = icmp ne i64 %t446, 0
  br i1 %t447, label %abandon.done.226, label %abandon.susp.227
abandon.susp.227:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t448 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t448, label %coro.suspend [ i8 0, label %resume.228 i8 1, label %coro.cleanup ]
resume.228:
  br label %abandon.wait.225
abandon.done.226:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.224:
  %t449 = load ptr, ptr %a385
  %t450 = call ptr @veles_scope_failed(ptr %t449)
  %t451 = icmp ne ptr %t450, null
  br i1 %t451, label %scope.abort.229, label %scope.ok.230
scope.ok.230:
  br label %cont.231
scope.abort.229:
  call void @veles_task_leave_waits(ptr %task)
  %t452 = load ptr, ptr %a385
  call void @veles_scope_abandon(ptr %t452, ptr %task)
  br label %scope.wait.187
cont.231:
  br label %yield.220
backedge.cancel.215:
  call void @veles_cleanup_pop(ptr %a386)
  %t453 = load ptr, ptr %a385
  call void @veles_scope_cancel(ptr %t453)
  br label %abandon.wait.232
abandon.wait.232:
  %t454 = call i64 @veles_scope_wait(ptr %task, ptr %t453)
  %t455 = icmp ne i64 %t454, 0
  br i1 %t455, label %abandon.done.233, label %abandon.susp.234
abandon.susp.234:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t456 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t456, label %coro.suspend [ i8 0, label %resume.235 i8 1, label %coro.cleanup ]
resume.235:
  br label %abandon.wait.232
abandon.done.233:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.213:
  br label %loop.cond.190
loop.end.192:
  br label %scope.wait.187
scope.wait.187:
  %t457 = load ptr, ptr %a385
  %t458 = call i64 @veles_scope_wait(ptr %task, ptr %t457)
  %t459 = icmp ne i64 %t458, 0
  br i1 %t459, label %scope.done.188, label %scope.susp.189
scope.susp.189:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t460 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t460, label %coro.suspend [ i8 0, label %resume.236 i8 1, label %coro.cleanup ]
resume.236:
  %t461 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t462 = icmp ne i64 %t461, 0
  br i1 %t462, label %cancelled.237, label %cont.238
cancelled.237:
  call void @veles_cleanup_pop(ptr %a386)
  %t463 = load ptr, ptr %a385
  call void @veles_scope_cancel(ptr %t463)
  br label %abandon.wait.239
abandon.wait.239:
  %t464 = call i64 @veles_scope_wait(ptr %task, ptr %t463)
  %t465 = icmp ne i64 %t464, 0
  br i1 %t465, label %abandon.done.240, label %abandon.susp.241
abandon.susp.241:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t466 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t466, label %coro.suspend [ i8 0, label %resume.242 i8 1, label %coro.cleanup ]
resume.242:
  br label %abandon.wait.239
abandon.done.240:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.238:
  br label %scope.wait.187
scope.done.188:
  call void @veles_cleanup_pop(ptr %a386)
  %t467 = load ptr, ptr %a385
  %t468 = call ptr @veles_scope_failed(ptr %t467)
  %t469 = icmp ne ptr %t468, null
  br i1 %t469, label %scope.check.243, label %scope.after.244
scope.check.243:
  %t470 = call i64 @veles_task_panicked(ptr %t468)
  %t471 = icmp ne i64 %t470, 0
  br i1 %t471, label %scope.repanic.245, label %scope.errors.246
scope.repanic.245:
  call void @veles_task_repanic(ptr %t468)
  unreachable
scope.errors.246:
  br label %scope.after.244
scope.after.244:
  %t472 = load i64, ptr %a383
  %t474 = call i64 @veles_i64_format(ptr %a473, i64 %t472)
  %t475 = insertvalue %str undef, ptr %a473, 0
  %t476 = insertvalue %str %t475, i64 %t474, 1
  %t478 = extractvalue %str { ptr @.str.21, i64 7 }, 0
  %t479 = extractvalue %str { ptr @.str.21, i64 7 }, 1
  %t480 = extractvalue %str %t476, 0
  %t481 = extractvalue %str %t476, 1
  call void @veles_string_concat(ptr %a477, ptr %t478, i64 %t479, ptr %t480, i64 %t481)
  %t482 = load %str, ptr %a477
  %t483 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t484 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t483)
  call void @v_std.io.println(%str %t482)
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
  %t6 = extractvalue %str { ptr @.str.23, i64 7 }, 0
  %t7 = extractvalue %str { ptr @.str.23, i64 7 }, 1
  %t8 = extractvalue %str %t4, 0
  %t9 = extractvalue %str %t4, 1
  call void @veles_string_concat(ptr %a5, ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  %t10 = load %str, ptr %a5
  %t11 = extractvalue %str { ptr @.str.24, i64 20 }, 0
  %t12 = extractvalue %str { ptr @.str.24, i64 20 }, 1
  call void @veles_call_push(ptr %t11)
  call void @v_std.io.println(%str %t10)
  call void @veles_call_pop()
  ret void
}

define internal void @entry.v_main.slow(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { i64 }, ptr %args, i32 0, i32 0
  %t2 = load i64, ptr %t1
  %t3 = extractvalue %str { ptr @.str.25, i64 5 }, 0
  %t4 = extractvalue %str { ptr @.str.25, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t3)
  %t5 = call ptr @v_main.slow(ptr %task, ptr @veles.root.link, i64 %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.produce(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { ptr }, ptr %args, i32 0, i32 0
  %t2 = load ptr, ptr %t1
  %t3 = extractvalue %str { ptr @.str.26, i64 8 }, 0
  %t4 = extractvalue %str { ptr @.str.26, i64 8 }, 1
  call void @veles_call_base(ptr %task, ptr %t3)
  %t5 = call ptr @v_main.produce(ptr %task, ptr @veles.root.link, ptr %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.27, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.27, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task, ptr @veles.root.link)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

@.str.1 = private unnamed_addr constant [21 x i8] c"main.vs:12:15\00millis\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:13:10\00"
@.str.4 = private unnamed_addr constant [3 x i8] c"db\00"
@.str.5 = private unnamed_addr constant [19 x i8] c"main.vs:22:9\00close\00"
@.str.6 = private unnamed_addr constant [2 x i8] c" \00"
@.str.7 = private unnamed_addr constant [21 x i8] c"main.vs:26:7\00println\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:35:7\00"
@.str.9 = private unnamed_addr constant [5 x i8] c"sum \00"
@.str.10 = private unnamed_addr constant [21 x i8] c"main.vs:38:3\00println\00"
@.str.11 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.12 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.13 = private unnamed_addr constant [2 x i8] c"c\00"
@.str.14 = private unnamed_addr constant [2 x i8] c"-\00"
@.str.15 = private unnamed_addr constant [21 x i8] c"main.vs:42:3\00println\00"
@.str.16 = private unnamed_addr constant [21 x i8] c"main.vs:51:19\00millis\00"
@.str.17 = private unnamed_addr constant [13 x i8] c"main.vs:52:7\00"
@.str.18 = private unnamed_addr constant [7 x i8] c"slept \00"
@.str.19 = private unnamed_addr constant [21 x i8] c"main.vs:55:3\00println\00"
@.str.20 = private unnamed_addr constant [14 x i8] c"main.vs:64:22\00"
@.str.21 = private unnamed_addr constant [8 x i8] c"polled \00"
@.str.22 = private unnamed_addr constant [21 x i8] c"main.vs:67:3\00println\00"
@.str.23 = private unnamed_addr constant [8 x i8] c"closed \00"
@.str.24 = private unnamed_addr constant [21 x i8] c"main.vs:8:39\00println\00"
@.str.25 = private unnamed_addr constant [6 x i8] c"\00slow\00"
@.str.26 = private unnamed_addr constant [9 x i8] c"\00produce\00"
@.str.27 = private unnamed_addr constant [6 x i8] c"\00main\00"
