%V._prelude_.Result_main.Counter_main.Gone_ = type { [64 x i8], [8 x i64] }
%S.main.Gone = type { %str }
%S._prelude_.Err_main.Counter_main.Gone_ = type { %S.main.Gone }
%S.main.Counter = type { i64, [56 x i8] }
%S._prelude_.Ok_main.Counter_main.Gone_ = type { %S.main.Counter }
%S._prelude_.Err_i64_main.Gone_ = type { %S.main.Gone }
%V._prelude_.Result_i64_main.Gone_ = type { i32, [2 x i64] }
%S._prelude_.Ok_i64_main.Gone_ = type { i64 }
%S.main.Wide = type { i64, i64, i64, i64, i64, i64, i64, i64, i64, [56 x i8] }
%V.main.Slot = type { [64 x i8], [8 x i64] }
%S.main.Hot = type { %S.main.Counter }
%S.main.Cold = type { %str, [3 x %str] }
@desc.T_u8_main.Counter_ = internal constant { i64, i64, i64, [0 x i64] } { i64 128, i64 16384, i64 0, [0 x i64] [] }
@desc.main.Slot = internal constant { i64, i64, i64, [4 x i64] } { i64 128, i64 16384, i64 4, [4 x i64] [i64 64, i64 80, i64 96, i64 112] }
define void @v_main.count(ptr noalias sret(%V._prelude_.Result_main.Counter_main.Gone_) align 64 %sret, i64 %p1) {
entry:
  %a1 = alloca i64
  %a6 = alloca %V._prelude_.Result_main.Counter_main.Gone_, align 64
  %a12 = alloca %V._prelude_.Result_main.Counter_main.Gone_, align 64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = icmp slt i64 %t2, 0
  br i1 %t3, label %if.then.1, label %if.end.2
if.then.1:
  %t4 = insertvalue %S.main.Gone undef, %str { ptr @.str.1, i64 8 }, 0
  %t5 = insertvalue %S._prelude_.Err_main.Counter_main.Gone_ undef, %S.main.Gone %t4, 0
  %t7 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %a6, i32 0, i32 0
  store i32 1, ptr %t7
  %t8 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %a6, i32 0, i32 1
  store %S._prelude_.Err_main.Counter_main.Gone_ %t5, ptr %t8
  call void @llvm.memmove.p0.p0.i64(ptr %sret, ptr %a6, i64 128, i1 false)
  ret void
if.end.2:
  %t9 = load i64, ptr %a1
  %t10 = insertvalue %S.main.Counter undef, i64 %t9, 0
  %t11 = insertvalue %S._prelude_.Ok_main.Counter_main.Gone_ undef, %S.main.Counter %t10, 0
  %t13 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %a12, i32 0, i32 0
  store i32 0, ptr %t13
  %t14 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %a12, i32 0, i32 1
  store %S._prelude_.Ok_main.Counter_main.Gone_ %t11, ptr %t14
  call void @llvm.memmove.p0.p0.i64(ptr %sret, ptr %a12, i64 128, i1 false)
  ret void
}

define ptr @v_main.both(ptr %task, ptr %link, i64 %p1, i64 %p2) presplitcoroutine {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  %a3 = alloca i64
  %a5 = alloca ptr
  %a6 = alloca { ptr, ptr, ptr, ptr }
  %a12 = alloca ptr
  %a13 = alloca ptr
  %a19 = alloca ptr
  %a20 = alloca ptr
  %a89 = alloca %V._prelude_.Result_i64_main.Gone_
  %a93 = alloca %V._prelude_.Result_i64_main.Gone_
  %a106 = alloca %V._prelude_.Result_i64_main.Gone_
  %a110 = alloca %V._prelude_.Result_i64_main.Gone_
  %a119 = alloca %V._prelude_.Result_i64_main.Gone_
  %a123 = alloca %V._prelude_.Result_i64_main.Gone_
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  store i64 zeroinitializer, ptr %a3
  %t4 = call ptr @veles_scope_begin(ptr %task, i64 1, i64 %coro.depth)
  store ptr %t4, ptr %a5
  call void @veles_cleanup_push(ptr %a6, ptr @scope.cancel.thunk, ptr %a5, ptr @cleanup.move.word)
  %t7 = load ptr, ptr %a5
  %t8 = call ptr @veles_task_launch(ptr %t7, i64 0)
  call void @veles_task_set_unwrap(ptr %t8, i64 64)
  %t9 = load i64, ptr %a1
  %t10 = call ptr @veles_alloc_words(i64 16)
  %t11 = getelementptr inbounds { i64 }, ptr %t10, i32 0, i32 0
  store i64 %t9, ptr %t11
  call void @veles_task_spawn(ptr %t8, ptr @entry.ramp.v_main.count, ptr %t10)
  store ptr %t8, ptr %a12
  store ptr %t8, ptr %a13
  %t14 = load ptr, ptr %a5
  %t15 = call ptr @veles_task_launch(ptr %t14, i64 1)
  call void @veles_task_set_unwrap(ptr %t15, i64 64)
  %t16 = load i64, ptr %a2
  %t17 = call ptr @veles_alloc_words(i64 16)
  %t18 = getelementptr inbounds { i64 }, ptr %t17, i32 0, i32 0
  store i64 %t16, ptr %t18
  call void @veles_task_spawn(ptr %t15, ptr @entry.ramp.v_main.count, ptr %t17)
  store ptr %t15, ptr %a19
  store ptr %t15, ptr %a20
  %t21 = load ptr, ptr %a13
  br label %await.4
await.4:
  %t22 = call i64 @veles_task_await(ptr %task, ptr %t21)
  %t23 = icmp ne i64 %t22, 0
  br i1 %t23, label %await.got.5, label %await.susp.6
await.susp.6:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t24 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t24, label %coro.suspend [ i8 0, label %resume.7 i8 1, label %coro.cleanup ]
resume.7:
  %t25 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t26 = icmp ne i64 %t25, 0
  br i1 %t26, label %cancelled.8, label %cont.9
cancelled.8:
  call void @veles_cleanup_pop(ptr %a6)
  %t27 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t27)
  br label %abandon.wait.10
abandon.wait.10:
  %t28 = call i64 @veles_scope_wait(ptr %task, ptr %t27)
  %t29 = icmp ne i64 %t28, 0
  br i1 %t29, label %abandon.done.11, label %abandon.susp.12
abandon.susp.12:
  call void @veles_frame_park_held(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t30 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t30, label %coro.suspend [ i8 0, label %resume.13 i8 1, label %coro.cleanup ]
resume.13:
  br label %abandon.wait.10
abandon.done.11:
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
  %t35 = call i64 @veles_task_panicked(ptr %t21)
  %t36 = icmp ne i64 %t35, 0
  br i1 %t36, label %await.repanic.17, label %await.fine.18
await.repanic.17:
  call void @veles_task_repanic(ptr %t21)
  unreachable
await.fine.18:
  %t37 = call ptr @veles_task_value(ptr %t21)
  %t38 = load %S.main.Counter, ptr %t37
  %t39 = extractvalue %S.main.Counter %t38, 0
  %t40 = load ptr, ptr %a20
  br label %await.19
await.19:
  %t41 = call i64 @veles_task_await(ptr %task, ptr %t40)
  %t42 = icmp ne i64 %t41, 0
  br i1 %t42, label %await.got.20, label %await.susp.21
await.susp.21:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t43 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t43, label %coro.suspend [ i8 0, label %resume.22 i8 1, label %coro.cleanup ]
resume.22:
  %t44 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
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
  call void @veles_frame_park_held(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t49 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t49, label %coro.suspend [ i8 0, label %resume.28 i8 1, label %coro.cleanup ]
resume.28:
  br label %abandon.wait.25
abandon.done.26:
  call void @veles_frame_unwound(ptr %task, ptr %link)
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
  %t53 = load ptr, ptr %a5
  call void @veles_scope_abandon(ptr %t53, ptr %task)
  br label %scope.wait.1
cont.31:
  br label %await.19
await.got.20:
  %t54 = call i64 @veles_task_panicked(ptr %t40)
  %t55 = icmp ne i64 %t54, 0
  br i1 %t55, label %await.repanic.32, label %await.fine.33
await.repanic.32:
  call void @veles_task_repanic(ptr %t40)
  unreachable
await.fine.33:
  %t56 = call ptr @veles_task_value(ptr %t40)
  %t57 = load %S.main.Counter, ptr %t56
  %t58 = extractvalue %S.main.Counter %t57, 0
  %t60 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t39, i64 %t58)
  %t61 = extractvalue { i64, i1 } %t60, 0
  %t62 = extractvalue { i64, i1 } %t60, 1
  br i1 %t62, label %overflow.34, label %arith.ok.35
overflow.34:
  %t63 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t64 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t65 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t66 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t63, i64 %t64, ptr %t65, i64 %t66)
  unreachable
arith.ok.35:
  store i64 %t61, ptr %a3
  br label %scope.wait.1
scope.wait.1:
  %t67 = load ptr, ptr %a5
  %t68 = call i64 @veles_scope_wait(ptr %task, ptr %t67)
  %t69 = icmp ne i64 %t68, 0
  br i1 %t69, label %scope.done.2, label %scope.susp.3
scope.susp.3:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t70 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t70, label %coro.suspend [ i8 0, label %resume.36 i8 1, label %coro.cleanup ]
resume.36:
  %t71 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t72 = icmp ne i64 %t71, 0
  br i1 %t72, label %cancelled.37, label %cont.38
cancelled.37:
  call void @veles_cleanup_pop(ptr %a6)
  %t73 = load ptr, ptr %a5
  call void @veles_scope_cancel(ptr %t73)
  br label %abandon.wait.39
abandon.wait.39:
  %t74 = call i64 @veles_scope_wait(ptr %task, ptr %t73)
  %t75 = icmp ne i64 %t74, 0
  br i1 %t75, label %abandon.done.40, label %abandon.susp.41
abandon.susp.41:
  call void @veles_frame_park_held(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t76 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t76, label %coro.suspend [ i8 0, label %resume.42 i8 1, label %coro.cleanup ]
resume.42:
  br label %abandon.wait.39
abandon.done.40:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.38:
  br label %scope.wait.1
scope.done.2:
  call void @veles_cleanup_pop(ptr %a6)
  %t77 = load ptr, ptr %a5
  %t78 = call ptr @veles_scope_failed(ptr %t77)
  %t79 = icmp ne ptr %t78, null
  br i1 %t79, label %scope.check.43, label %scope.after.44
scope.check.43:
  %t80 = call i64 @veles_task_panicked(ptr %t78)
  %t81 = icmp ne i64 %t80, 0
  br i1 %t81, label %scope.repanic.45, label %scope.errors.46
scope.repanic.45:
  call void @veles_task_repanic(ptr %t78)
  unreachable
scope.errors.46:
  %t82 = call i64 @veles_scope_failed_index(ptr %t77)
  switch i64 %t82, label %scope.end.47 [ i64 0, label %scope.fail.48 i64 1, label %scope.fail.49 ]
scope.fail.48:
  %t83 = call ptr @veles_scope_failed(ptr %t77)
  %t84 = call ptr @veles_task_result(ptr %t83)
  %t85 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %t84, i32 0, i32 1
  %t86 = load %S._prelude_.Err_main.Counter_main.Gone_, ptr %t85
  %t87 = extractvalue %S._prelude_.Err_main.Counter_main.Gone_ %t86, 0
  %t88 = insertvalue %S._prelude_.Err_i64_main.Gone_ undef, %S.main.Gone %t87, 0
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a89)
  store %V._prelude_.Result_i64_main.Gone_ zeroinitializer, ptr %a89
  %t90 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a89, i32 0, i32 0
  store i32 1, ptr %t90
  %t91 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a89, i32 0, i32 1
  store %S._prelude_.Err_i64_main.Gone_ %t88, ptr %t91
  %t92 = load %V._prelude_.Result_i64_main.Gone_, ptr %a89
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a93)
  store %V._prelude_.Result_i64_main.Gone_ %t92, ptr %a93
  %t94 = extractvalue %V._prelude_.Result_i64_main.Gone_ %t92, 0
  %t95 = icmp eq i32 %t94, 1
  %t96 = zext i1 %t95 to i64
  %t97 = load ptr, ptr %link
  %t98 = icmp eq ptr %t97, null
  br i1 %t98, label %ret.task.50, label %ret.call.51
ret.task.50:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a93, i64 24, i64 %t96, ptr null)
  br label %coro.final
ret.call.51:
  %t99 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %link, i32 0, i32 3
  store %V._prelude_.Result_i64_main.Gone_ %t92, ptr %t99
  call void @veles.frame.back(ptr %task, ptr %link)
  br label %coro.final
scope.fail.49:
  %t100 = call ptr @veles_scope_failed(ptr %t77)
  %t101 = call ptr @veles_task_result(ptr %t100)
  %t102 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %t101, i32 0, i32 1
  %t103 = load %S._prelude_.Err_main.Counter_main.Gone_, ptr %t102
  %t104 = extractvalue %S._prelude_.Err_main.Counter_main.Gone_ %t103, 0
  %t105 = insertvalue %S._prelude_.Err_i64_main.Gone_ undef, %S.main.Gone %t104, 0
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a106)
  store %V._prelude_.Result_i64_main.Gone_ zeroinitializer, ptr %a106
  %t107 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a106, i32 0, i32 0
  store i32 1, ptr %t107
  %t108 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a106, i32 0, i32 1
  store %S._prelude_.Err_i64_main.Gone_ %t105, ptr %t108
  %t109 = load %V._prelude_.Result_i64_main.Gone_, ptr %a106
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a110)
  store %V._prelude_.Result_i64_main.Gone_ %t109, ptr %a110
  %t111 = extractvalue %V._prelude_.Result_i64_main.Gone_ %t109, 0
  %t112 = icmp eq i32 %t111, 1
  %t113 = zext i1 %t112 to i64
  %t114 = load ptr, ptr %link
  %t115 = icmp eq ptr %t114, null
  br i1 %t115, label %ret.task.52, label %ret.call.53
ret.task.52:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a110, i64 24, i64 %t113, ptr null)
  br label %coro.final
ret.call.53:
  %t116 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %link, i32 0, i32 3
  store %V._prelude_.Result_i64_main.Gone_ %t109, ptr %t116
  call void @veles.frame.back(ptr %task, ptr %link)
  br label %coro.final
scope.end.47:
  br label %scope.after.44
scope.after.44:
  %t117 = load i64, ptr %a3
  %t118 = insertvalue %S._prelude_.Ok_i64_main.Gone_ undef, i64 %t117, 0
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a119)
  store %V._prelude_.Result_i64_main.Gone_ zeroinitializer, ptr %a119
  %t120 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a119, i32 0, i32 0
  store i32 0, ptr %t120
  %t121 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a119, i32 0, i32 1
  store %S._prelude_.Ok_i64_main.Gone_ %t118, ptr %t121
  %t122 = load %V._prelude_.Result_i64_main.Gone_, ptr %a119
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a123)
  store %V._prelude_.Result_i64_main.Gone_ %t122, ptr %a123
  %t124 = extractvalue %V._prelude_.Result_i64_main.Gone_ %t122, 0
  %t125 = icmp eq i32 %t124, 1
  %t126 = zext i1 %t125 to i64
  %t127 = load ptr, ptr %link
  %t128 = icmp eq ptr %t127, null
  br i1 %t128, label %ret.task.54, label %ret.call.55
ret.task.54:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a123, i64 24, i64 %t126, ptr null)
  br label %coro.final
ret.call.55:
  %t129 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %link, i32 0, i32 3
  store %V._prelude_.Result_i64_main.Gone_ %t122, ptr %t129
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

define i64 @v_main.total(ptr noalias nocapture readonly %p1) noinline {
entry:
  %t1 = getelementptr inbounds %S.main.Wide, ptr %p1, i32 0, i32 0
  %t2 = load i64, ptr %t1
  %t3 = getelementptr inbounds %S.main.Wide, ptr %p1, i32 0, i32 1
  %t4 = load i64, ptr %t3
  %t6 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t2, i64 %t4)
  %t7 = extractvalue { i64, i1 } %t6, 0
  %t8 = extractvalue { i64, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t12 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  %t13 = getelementptr inbounds %S.main.Wide, ptr %p1, i32 0, i32 8
  %t14 = load i64, ptr %t13
  %t16 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t7, i64 %t14)
  %t17 = extractvalue { i64, i1 } %t16, 0
  %t18 = extractvalue { i64, i1 } %t16, 1
  br i1 %t18, label %overflow.3, label %arith.ok.4
overflow.3:
  %t19 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t20 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t21 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t22 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t19, i64 %t20, ptr %t21, i64 %t22)
  unreachable
arith.ok.4:
  ret i64 %t17
}

define %str @v_main.read(ptr noalias nocapture readonly %p1) {
entry:
  %a1 = alloca %str
  %a2 = alloca %V.main.Slot, align 64
  %a9 = alloca %S.main.Counter, align 64
  %a12 = alloca [21 x i8]
  %a16 = alloca %str
  %a28 = alloca %str
  %a30 = alloca %str
  call void @llvm.memmove.p0.p0.i64(ptr %a2, ptr %p1, i64 128, i1 false)
  br label %when.arm.2
when.arm.2:
  %t4 = getelementptr inbounds %V.main.Slot, ptr %a2, i32 0, i32 0
  %t3 = load i32, ptr %t4
  %t5 = icmp eq i32 %t3, 0
  br i1 %t5, label %when.bind.5, label %when.arm.3
when.bind.5:
  %t6 = getelementptr inbounds %V.main.Slot, ptr %a2, i32 0, i32 1
  %t7 = getelementptr inbounds %S.main.Hot, ptr %t6, i32 0, i32 0
  %t8 = load %S.main.Counter, ptr %t7
  store %S.main.Counter %t8, ptr %a9
  br label %when.body.4
when.body.4:
  %t10 = getelementptr inbounds %S.main.Counter, ptr %a9, i32 0, i32 0
  %t11 = load i64, ptr %t10
  %t13 = call i64 @veles_i64_format(ptr %a12, i64 %t11)
  %t14 = insertvalue %str undef, ptr %a12, 0
  %t15 = insertvalue %str %t14, i64 %t13, 1
  %t17 = extractvalue %str { ptr @.str.5, i64 4 }, 0
  %t18 = extractvalue %str { ptr @.str.5, i64 4 }, 1
  %t19 = extractvalue %str %t15, 0
  %t20 = extractvalue %str %t15, 1
  call void @veles_string_concat(ptr %a16, ptr %t17, i64 %t18, ptr %t19, i64 %t20)
  %t21 = load %str, ptr %a16
  store %str %t21, ptr %a1
  br label %when.end.1
when.arm.3:
  %t23 = getelementptr inbounds %V.main.Slot, ptr %a2, i32 0, i32 0
  %t22 = load i32, ptr %t23
  %t24 = icmp eq i32 %t22, 1
  br i1 %t24, label %when.bind.8, label %when.arm.6
when.bind.8:
  %t25 = getelementptr inbounds %V.main.Slot, ptr %a2, i32 0, i32 1
  %t26 = getelementptr inbounds %S.main.Cold, ptr %t25, i32 0, i32 0
  %t27 = load %str, ptr %t26
  store %str %t27, ptr %a28
  br label %when.body.7
when.body.7:
  %t29 = load %str, ptr %a28
  %t31 = extractvalue %str { ptr @.str.6, i64 5 }, 0
  %t32 = extractvalue %str { ptr @.str.6, i64 5 }, 1
  %t33 = extractvalue %str %t29, 0
  %t34 = extractvalue %str %t29, 1
  call void @veles_string_concat_shared(ptr %a30, ptr %t31, i64 %t32, ptr %t33, i64 %t34)
  %t35 = load %str, ptr %a30
  store %str %t35, ptr %a1
  br label %when.end.1
when.arm.6:
  %t36 = extractvalue %str { ptr @.str.7, i64 33 }, 0
  %t37 = extractvalue %str { ptr @.str.7, i64 33 }, 1
  %t38 = extractvalue %str { ptr @.str.8, i64 13 }, 0
  %t39 = extractvalue %str { ptr @.str.8, i64 13 }, 1
  call void @veles_panic_at(ptr %t36, i64 %t37, ptr %t38, i64 %t39)
  unreachable
when.end.1:
  %t40 = load %str, ptr %a1
  ret %str %t40
}

define i64 @v_main.offset(ptr %p1, ptr %p2) {
entry:
  %a1 = alloca ptr
  %a2 = alloca ptr
  store ptr %p1, ptr %a1
  store ptr %p2, ptr %a2
  %t3 = load ptr, ptr %a2
  %t4 = load ptr, ptr %a1
  %t5 = ptrtoint ptr %t3 to i64
  %t6 = ptrtoint ptr %t4 to i64
  %t7 = sub i64 %t5, %t6
  %t8 = ptrtoint ptr getelementptr (i8, ptr null, i64 1) to i64
  %t9 = sdiv i64 %t7, %t8
  ret i64 %t9
}

define ptr @v_main.main(ptr %task, ptr %link) presplitcoroutine {
entry:
  %a1 = alloca { i8, [63 x i8], %S.main.Counter }, align 64
  %a5 = alloca ptr
  %a23 = alloca { i1, [63 x i8], %S.main.Counter }, align 64
  %a26 = alloca { i1, [63 x i8], %S.main.Counter }, align 64
  %a29 = alloca %V.main.Slot, align 64
  %a32 = alloca ptr
  %a40 = alloca i64
  %a41 = alloca i64
  %a54 = alloca i64
  %a56 = alloca [21 x i8]
  %a61 = alloca [21 x i8]
  %a65 = alloca %str
  %a66 = alloca [4 x %str]
  %a76 = alloca { i1, [63 x i8], %S.main.Counter }, align 64
  %a77 = alloca { i1, i64 }
  %a89 = alloca %V.main.Slot, align 64
  %a93 = alloca [3 x %str]
  %a100 = alloca %V.main.Slot, align 64
  %a106 = alloca %str
  %a107 = alloca [7 x %str]
  %a118 = alloca %S.main.Wide, align 64
  %a128 = alloca %S.main.Wide, align 64
  %a163 = alloca [21 x i8]
  %a170 = alloca [21 x i8]
  %a174 = alloca %str
  %a175 = alloca [4 x %str]
  %a185 = alloca { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }
  %a201 = alloca %V._prelude_.Result_i64_main.Gone_
  %a202 = alloca i64
  %a210 = alloca [21 x i8]
  %a216 = alloca { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }
  %a232 = alloca %V._prelude_.Result_i64_main.Gone_
  %a233 = alloca i64
  %a241 = alloca [21 x i8]
  %a245 = alloca %str
  %a246 = alloca [4 x %str]
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t2 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %a1, i32 0, i32 0
  store i8 1, ptr %t2
  %t3 = insertvalue %S.main.Counter undef, i64 2, 0
  %t4 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %a1, i32 0, i32 2
  store %S.main.Counter %t3, ptr %t4
  %t6 = call ptr @veles_gc_alloc(ptr @desc.T_u8_main.Counter_, i64 128)
  store ptr %t6, ptr %a5
  call void @llvm.memmove.p0.p0.i64(ptr %t6, ptr %a1, i64 128, i1 false)
  %t7 = load ptr, ptr %a5
  %t8 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %t7, i32 0, i32 2
  %t9 = getelementptr inbounds %S.main.Counter, ptr %t8, i32 0, i32 0
  %t10 = load i64, ptr %t9
  %t12 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t10, i64 1)
  %t13 = extractvalue { i64, i1 } %t12, 0
  %t14 = extractvalue { i64, i1 } %t12, 1
  br i1 %t14, label %overflow.1, label %arith.ok.2
overflow.1:
  %t15 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t16 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t17 = extractvalue %str { ptr @.str.9, i64 12 }, 0
  %t18 = extractvalue %str { ptr @.str.9, i64 12 }, 1
  call void @veles_panic_at(ptr %t15, i64 %t16, ptr %t17, i64 %t18)
  unreachable
arith.ok.2:
  %t19 = load ptr, ptr %a5
  %t20 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %t19, i32 0, i32 2
  %t21 = getelementptr inbounds %S.main.Counter, ptr %t20, i32 0, i32 0
  store i64 %t13, ptr %t21
  %t22 = insertvalue %S.main.Counter undef, i64 4, 0
  %t24 = getelementptr inbounds { i1, [63 x i8], %S.main.Counter }, ptr %a23, i32 0, i32 0
  store i1 true, ptr %t24
  %t25 = getelementptr inbounds { i1, [63 x i8], %S.main.Counter }, ptr %a23, i32 0, i32 2
  store %S.main.Counter %t22, ptr %t25
  call void @llvm.memmove.p0.p0.i64(ptr %a26, ptr %a23, i64 128, i1 false)
  %t27 = insertvalue %S.main.Counter undef, i64 5, 0
  %t28 = insertvalue %S.main.Hot undef, %S.main.Counter %t27, 0
  %t30 = getelementptr inbounds %V.main.Slot, ptr %a29, i32 0, i32 0
  store i32 0, ptr %t30
  %t31 = getelementptr inbounds %V.main.Slot, ptr %a29, i32 0, i32 1
  store %S.main.Hot %t28, ptr %t31
  %t33 = call ptr @veles_gc_alloc(ptr @desc.main.Slot, i64 128)
  store ptr %t33, ptr %a32
  call void @llvm.memmove.p0.p0.i64(ptr %t33, ptr %a29, i64 128, i1 false)
  %t34 = load ptr, ptr %a5
  %t35 = load ptr, ptr %a5
  %t36 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %t35, i32 0, i32 2
  %t37 = extractvalue %str { ptr @.str.10, i64 20 }, 0
  %t38 = extractvalue %str { ptr @.str.10, i64 20 }, 1
  call void @veles_call_push(ptr %t37)
  %t39 = call i64 @v_main.offset(ptr %t34, ptr %t36)
  call void @veles_call_pop()
  store i64 %t39, ptr %a40
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a41)
  %t42 = load ptr, ptr %a32
  %t44 = getelementptr inbounds %V.main.Slot, ptr %t42, i32 0, i32 0
  %t43 = load i32, ptr %t44
  %t45 = icmp eq i32 %t43, 0
  br i1 %t45, label %if.then.3, label %if.else.5
if.then.3:
  %t46 = load ptr, ptr %a32
  %t47 = load ptr, ptr %a32
  %t48 = getelementptr inbounds %V.main.Slot, ptr %t47, i32 0, i32 1
  %t49 = getelementptr inbounds %S.main.Hot, ptr %t48, i32 0, i32 0
  %t50 = extractvalue %str { ptr @.str.11, i64 20 }, 0
  %t51 = extractvalue %str { ptr @.str.11, i64 20 }, 1
  call void @veles_call_push(ptr %t50)
  %t52 = call i64 @v_main.offset(ptr %t46, ptr %t49)
  call void @veles_call_pop()
  store i64 %t52, ptr %a41
  br label %if.end.4
if.else.5:
  store i64 -1, ptr %a41
  br label %if.end.4
if.end.4:
  %t53 = load i64, ptr %a41
  store i64 %t53, ptr %a54
  %t55 = load i64, ptr %a40
  %t57 = call i64 @veles_i64_format(ptr %a56, i64 %t55)
  %t58 = insertvalue %str undef, ptr %a56, 0
  %t59 = insertvalue %str %t58, i64 %t57, 1
  %t60 = load i64, ptr %a54
  %t62 = call i64 @veles_i64_format(ptr %a61, i64 %t60)
  %t63 = insertvalue %str undef, ptr %a61, 0
  %t64 = insertvalue %str %t63, i64 %t62, 1
  %t67 = getelementptr [4 x %str], ptr %a66, i64 0, i64 0
  store %str { ptr @.str.12, i64 8 }, ptr %t67
  %t68 = getelementptr [4 x %str], ptr %a66, i64 0, i64 1
  store %str %t59, ptr %t68
  %t69 = getelementptr [4 x %str], ptr %a66, i64 0, i64 2
  store %str { ptr @.str.13, i64 1 }, ptr %t69
  %t70 = getelementptr [4 x %str], ptr %a66, i64 0, i64 3
  store %str %t64, ptr %t70
  call void @veles_string_concat_n(ptr %a65, ptr %a66, i64 4)
  %t71 = load %str, ptr %a65
  %t72 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t73 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t72)
  call void @v_std.io.println(%str %t71)
  call void @veles_call_pop()
  %t74 = load ptr, ptr %a5
  %t75 = call %str @show.T_u8_main.Counter_(ptr %t74)
  call void @llvm.memmove.p0.p0.i64(ptr %a76, ptr %a26, i64 128, i1 false)
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a77)
  %t80 = getelementptr inbounds { i1, [63 x i8], %S.main.Counter }, ptr %a76, i32 0, i32 0
  %t79 = load i1, ptr %t80
  %t78 = xor i1 %t79, true
  br i1 %t78, label %if.then.6, label %if.else.8
if.then.6:
  store { i1, i64 } zeroinitializer, ptr %a77
  br label %if.end.7
if.else.8:
  %t81 = getelementptr inbounds { i1, [63 x i8], %S.main.Counter }, ptr %a76, i32 0, i32 2
  %t82 = getelementptr inbounds %S.main.Counter, ptr %t81, i32 0, i32 0
  %t83 = load i64, ptr %t82
  %t84 = insertvalue { i1, i64 } undef, i1 true, 0
  %t85 = insertvalue { i1, i64 } %t84, i64 %t83, 1
  store { i1, i64 } %t85, ptr %a77
  br label %if.end.7
if.end.7:
  %t86 = load { i1, i64 }, ptr %a77
  %t87 = call %str @show.T_i64_N({ i1, i64 } %t86)
  %t88 = load ptr, ptr %a32
  call void @llvm.memmove.p0.p0.i64(ptr %a89, ptr %t88, i64 128, i1 false)
  %t90 = extractvalue %str { ptr @.str.15, i64 18 }, 0
  %t91 = extractvalue %str { ptr @.str.15, i64 18 }, 1
  call void @veles_call_push(ptr %t90)
  %t92 = call %str @v_main.read(ptr %a89)
  call void @veles_call_pop()
  %t94 = getelementptr inbounds [3 x %str], ptr %a93, i64 0, i64 0
  store %str { ptr @.str.16, i64 0 }, ptr %t94
  %t95 = getelementptr inbounds [3 x %str], ptr %a93, i64 0, i64 1
  store %str { ptr @.str.16, i64 0 }, ptr %t95
  %t96 = getelementptr inbounds [3 x %str], ptr %a93, i64 0, i64 2
  store %str { ptr @.str.16, i64 0 }, ptr %t96
  %t97 = load [3 x %str], ptr %a93
  %t98 = insertvalue %S.main.Cold undef, %str { ptr @.str.17, i64 1 }, 0
  %t99 = insertvalue %S.main.Cold %t98, [3 x %str] %t97, 1
  %t101 = getelementptr inbounds %V.main.Slot, ptr %a100, i32 0, i32 0
  store i32 1, ptr %t101
  %t102 = getelementptr inbounds %V.main.Slot, ptr %a100, i32 0, i32 1
  store %S.main.Cold %t99, ptr %t102
  %t103 = extractvalue %str { ptr @.str.18, i64 18 }, 0
  %t104 = extractvalue %str { ptr @.str.18, i64 18 }, 1
  call void @veles_call_push(ptr %t103)
  %t105 = call %str @v_main.read(ptr %a100)
  call void @veles_call_pop()
  %t108 = getelementptr [7 x %str], ptr %a107, i64 0, i64 0
  store %str %t75, ptr %t108
  %t109 = getelementptr [7 x %str], ptr %a107, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t109
  %t110 = getelementptr [7 x %str], ptr %a107, i64 0, i64 2
  store %str %t87, ptr %t110
  %t111 = getelementptr [7 x %str], ptr %a107, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t111
  %t112 = getelementptr [7 x %str], ptr %a107, i64 0, i64 4
  store %str %t92, ptr %t112
  %t113 = getelementptr [7 x %str], ptr %a107, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t113
  %t114 = getelementptr [7 x %str], ptr %a107, i64 0, i64 6
  store %str %t105, ptr %t114
  call void @veles_string_concat_n_shared(ptr %a106, ptr %a107, i64 7)
  %t115 = load %str, ptr %a106
  %t116 = extractvalue %str { ptr @.str.19, i64 20 }, 0
  %t117 = extractvalue %str { ptr @.str.19, i64 20 }, 1
  call void @veles_call_push(ptr %t116)
  call void @v_std.io.println(%str %t115)
  call void @veles_call_pop()
  %t119 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 0
  store i64 1, ptr %t119
  %t120 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 1
  store i64 2, ptr %t120
  %t121 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 2
  store i64 3, ptr %t121
  %t122 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 3
  store i64 4, ptr %t122
  %t123 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 4
  store i64 5, ptr %t123
  %t124 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 5
  store i64 6, ptr %t124
  %t125 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 6
  store i64 7, ptr %t125
  %t126 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 7
  store i64 8, ptr %t126
  %t127 = getelementptr inbounds %S.main.Wide, ptr %a118, i32 0, i32 8
  store i64 9, ptr %t127
  call void @llvm.memmove.p0.p0.i64(ptr %a128, ptr %a118, i64 128, i1 false)
  %t129 = extractvalue %str { ptr @.str.20, i64 20 }, 0
  %t130 = extractvalue %str { ptr @.str.20, i64 20 }, 1
  call void @veles_call_push(ptr %t129)
  %t131 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t132 = extractvalue %S.std.prelude.Duration %t131, 0
  br label %sleep.9
sleep.9:
  %t133 = call i64 @veles_task_sleep(ptr %task, i64 %t132)
  %t134 = icmp ne i64 %t133, 0
  br i1 %t134, label %sleep.done.10, label %sleep.susp.11
sleep.susp.11:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t135 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t135, label %coro.suspend [ i8 0, label %resume.12 i8 1, label %coro.cleanup ]
resume.12:
  %t136 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t137 = icmp ne i64 %t136, 0
  br i1 %t137, label %cancelled.13, label %cont.14
cancelled.13:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.14:
  br label %sleep.9
sleep.done.10:
  %t138 = getelementptr inbounds %S.main.Wide, ptr %a128, i32 0, i32 0
  %t139 = load i64, ptr %t138
  %t140 = extractvalue %str { ptr @.str.21, i64 19 }, 0
  %t141 = extractvalue %str { ptr @.str.21, i64 19 }, 1
  call void @veles_call_push(ptr %t140)
  %t142 = call i64 @v_main.total(ptr %a128)
  call void @veles_call_pop()
  %t144 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t139, i64 %t142)
  %t145 = extractvalue { i64, i1 } %t144, 0
  %t146 = extractvalue { i64, i1 } %t144, 1
  br i1 %t146, label %overflow.15, label %arith.ok.16
overflow.15:
  %t147 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t148 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t149 = extractvalue %str { ptr @.str.22, i64 12 }, 0
  %t150 = extractvalue %str { ptr @.str.22, i64 12 }, 1
  call void @veles_panic_at(ptr %t147, i64 %t148, ptr %t149, i64 %t150)
  unreachable
arith.ok.16:
  %t151 = getelementptr inbounds %S.main.Wide, ptr %a128, i32 0, i32 0
  store i64 %t145, ptr %t151
  %t152 = extractvalue %str { ptr @.str.23, i64 20 }, 0
  %t153 = extractvalue %str { ptr @.str.23, i64 20 }, 1
  call void @veles_call_push(ptr %t152)
  %t154 = call %S.std.prelude.Duration @v_std.prelude.Duration.millis(i64 1)
  call void @veles_call_pop()
  %t155 = extractvalue %S.std.prelude.Duration %t154, 0
  br label %sleep.17
sleep.17:
  %t156 = call i64 @veles_task_sleep(ptr %task, i64 %t155)
  %t157 = icmp ne i64 %t156, 0
  br i1 %t157, label %sleep.done.18, label %sleep.susp.19
sleep.susp.19:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t158 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t158, label %coro.suspend [ i8 0, label %resume.20 i8 1, label %coro.cleanup ]
resume.20:
  %t159 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t160 = icmp ne i64 %t159, 0
  br i1 %t160, label %cancelled.21, label %cont.22
cancelled.21:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.22:
  br label %sleep.17
sleep.done.18:
  %t161 = getelementptr inbounds %S.main.Wide, ptr %a128, i32 0, i32 0
  %t162 = load i64, ptr %t161
  %t164 = call i64 @veles_i64_format(ptr %a163, i64 %t162)
  %t165 = insertvalue %str undef, ptr %a163, 0
  %t166 = insertvalue %str %t165, i64 %t164, 1
  %t167 = extractvalue %str { ptr @.str.24, i64 19 }, 0
  %t168 = extractvalue %str { ptr @.str.24, i64 19 }, 1
  call void @veles_call_push(ptr %t167)
  %t169 = call i64 @v_main.total(ptr %a128)
  call void @veles_call_pop()
  %t171 = call i64 @veles_i64_format(ptr %a170, i64 %t169)
  %t172 = insertvalue %str undef, ptr %a170, 0
  %t173 = insertvalue %str %t172, i64 %t171, 1
  %t176 = getelementptr [4 x %str], ptr %a175, i64 0, i64 0
  store %str { ptr @.str.25, i64 6 }, ptr %t176
  %t177 = getelementptr [4 x %str], ptr %a175, i64 0, i64 1
  store %str %t166, ptr %t177
  %t178 = getelementptr [4 x %str], ptr %a175, i64 0, i64 2
  store %str { ptr @.str.13, i64 1 }, ptr %t178
  %t179 = getelementptr [4 x %str], ptr %a175, i64 0, i64 3
  store %str %t173, ptr %t179
  call void @veles_string_concat_n(ptr %a174, ptr %a175, i64 4)
  %t180 = load %str, ptr %a174
  %t181 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t182 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t181)
  call void @v_std.io.println(%str %t180)
  call void @veles_call_pop()
  %t183 = extractvalue %str { ptr @.str.27, i64 18 }, 0
  %t184 = extractvalue %str { ptr @.str.27, i64 18 }, 1
  call void @veles_call_push(ptr %t183)
  %t186 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a185, i32 0, i32 0
  store ptr %coro.hdl, ptr %t186
  %t187 = add i64 %coro.depth, 1
  %t188 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a185, i32 0, i32 1
  store i64 %t187, ptr %t188
  %t189 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a185, i32 0, i32 2
  store i64 0, ptr %t189
  %t190 = call ptr @v_main.both(ptr %task, ptr %a185, i64 7, i64 8)
  br label %call.check.23
call.check.23:
  %t191 = load i64, ptr %t189
  %t192 = icmp ne i64 %t191, 0
  br i1 %t192, label %call.done.24, label %call.wait.25
call.wait.25:
  %t193 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t193, label %coro.suspend [ i8 0, label %resume.26 i8 1, label %coro.cleanup ]
resume.26:
  %t194 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t195 = icmp ne i64 %t194, 0
  br i1 %t195, label %cancelled.27, label %cont.28
cancelled.27:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.28:
  br label %call.check.23
call.done.24:
  call void @veles.frame.free(ptr %task, ptr %t190)
  %t196 = icmp eq i64 %t191, 2
  br i1 %t196, label %call.unwound.29, label %call.ok.30, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.29:
  %t197 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t198 = icmp ne i64 %t197, 0
  br i1 %t198, label %call.cancelled.31, label %call.look.32
call.look.32:
  br label %call.cancelled.31
call.cancelled.31:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.30:
  %t199 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a185, i32 0, i32 3
  %t200 = load %V._prelude_.Result_i64_main.Gone_, ptr %t199
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.Gone_ %t200, ptr %a201
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a202)
  %t203 = load %V._prelude_.Result_i64_main.Gone_, ptr %a201
  %t204 = extractvalue %V._prelude_.Result_i64_main.Gone_ %t203, 0
  %t205 = icmp eq i32 %t204, 1
  br i1 %t205, label %if.then.33, label %if.else.35
if.then.33:
  store i64 -1, ptr %a202
  br label %if.end.34
if.else.35:
  %t206 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a201, i32 0, i32 1
  %t207 = getelementptr inbounds %S._prelude_.Ok_i64_main.Gone_, ptr %t206, i32 0, i32 0
  %t208 = load i64, ptr %t207
  store i64 %t208, ptr %a202
  br label %if.end.34
if.end.34:
  %t209 = load i64, ptr %a202
  %t211 = call i64 @veles_i64_format(ptr %a210, i64 %t209)
  %t212 = insertvalue %str undef, ptr %a210, 0
  %t213 = insertvalue %str %t212, i64 %t211, 1
  %t214 = extractvalue %str { ptr @.str.28, i64 18 }, 0
  %t215 = extractvalue %str { ptr @.str.28, i64 18 }, 1
  call void @veles_call_push(ptr %t214)
  %t217 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a216, i32 0, i32 0
  store ptr %coro.hdl, ptr %t217
  %t218 = add i64 %coro.depth, 1
  %t219 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a216, i32 0, i32 1
  store i64 %t218, ptr %t219
  %t220 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a216, i32 0, i32 2
  store i64 0, ptr %t220
  %t221 = call ptr @v_main.both(ptr %task, ptr %a216, i64 7, i64 -8)
  br label %call.check.36
call.check.36:
  %t222 = load i64, ptr %t220
  %t223 = icmp ne i64 %t222, 0
  br i1 %t223, label %call.done.37, label %call.wait.38
call.wait.38:
  %t224 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t224, label %coro.suspend [ i8 0, label %resume.39 i8 1, label %coro.cleanup ]
resume.39:
  %t225 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t226 = icmp ne i64 %t225, 0
  br i1 %t226, label %cancelled.40, label %cont.41
cancelled.40:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.41:
  br label %call.check.36
call.done.37:
  call void @veles.frame.free(ptr %task, ptr %t221)
  %t227 = icmp eq i64 %t222, 2
  br i1 %t227, label %call.unwound.42, label %call.ok.43, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.42:
  %t228 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t229 = icmp ne i64 %t228, 0
  br i1 %t229, label %call.cancelled.44, label %call.look.45
call.look.45:
  br label %call.cancelled.44
call.cancelled.44:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.43:
  %t230 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_i64_main.Gone_ }, ptr %a216, i32 0, i32 3
  %t231 = load %V._prelude_.Result_i64_main.Gone_, ptr %t230
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.Gone_ %t231, ptr %a232
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a233)
  %t234 = load %V._prelude_.Result_i64_main.Gone_, ptr %a232
  %t235 = extractvalue %V._prelude_.Result_i64_main.Gone_ %t234, 0
  %t236 = icmp eq i32 %t235, 1
  br i1 %t236, label %if.then.46, label %if.else.48
if.then.46:
  store i64 -1, ptr %a233
  br label %if.end.47
if.else.48:
  %t237 = getelementptr inbounds %V._prelude_.Result_i64_main.Gone_, ptr %a232, i32 0, i32 1
  %t238 = getelementptr inbounds %S._prelude_.Ok_i64_main.Gone_, ptr %t237, i32 0, i32 0
  %t239 = load i64, ptr %t238
  store i64 %t239, ptr %a233
  br label %if.end.47
if.end.47:
  %t240 = load i64, ptr %a233
  %t242 = call i64 @veles_i64_format(ptr %a241, i64 %t240)
  %t243 = insertvalue %str undef, ptr %a241, 0
  %t244 = insertvalue %str %t243, i64 %t242, 1
  %t247 = getelementptr [4 x %str], ptr %a246, i64 0, i64 0
  store %str { ptr @.str.29, i64 6 }, ptr %t247
  %t248 = getelementptr [4 x %str], ptr %a246, i64 0, i64 1
  store %str %t213, ptr %t248
  %t249 = getelementptr [4 x %str], ptr %a246, i64 0, i64 2
  store %str { ptr @.str.13, i64 1 }, ptr %t249
  %t250 = getelementptr [4 x %str], ptr %a246, i64 0, i64 3
  store %str %t244, ptr %t250
  call void @veles_string_concat_n(ptr %a245, ptr %a246, i64 4)
  %t251 = load %str, ptr %a245
  %t252 = extractvalue %str { ptr @.str.30, i64 20 }, 0
  %t253 = extractvalue %str { ptr @.str.30, i64 20 }, 1
  call void @veles_call_push(ptr %t252)
  call void @v_std.io.println(%str %t251)
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

define internal ptr @ramp.v_main.count(ptr %task, ptr %link, i64 %p0) presplitcoroutine {
entry:
  %a1 = alloca %V._prelude_.Result_main.Counter_main.Gone_, align 64
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  call void @v_main.count(ptr noalias sret(%V._prelude_.Result_main.Counter_main.Gone_) align 64 %a1, i64 %p0)
  %t3 = getelementptr inbounds %V._prelude_.Result_main.Counter_main.Gone_, ptr %a1, i32 0, i32 0
  %t2 = load i32, ptr %t3
  %t4 = icmp eq i32 %t2, 1
  %t5 = zext i1 %t4 to i64
  %t6 = load ptr, ptr %link
  %t7 = icmp eq ptr %t6, null
  br i1 %t7, label %ret.task.1, label %ret.call.2
ret.task.1:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a1, i64 128, i64 %t5, ptr null)
  br label %coro.final
ret.call.2:
  %t8 = getelementptr inbounds { ptr, i64, i64, %V._prelude_.Result_main.Counter_main.Gone_ }, ptr %link, i32 0, i32 3
  call void @llvm.memmove.p0.p0.i64(ptr %t8, ptr %a1, i64 128, i1 false)
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

define internal void @entry.ramp.v_main.count(ptr %task, ptr %args) {
entry:
  %t1 = getelementptr inbounds { i64 }, ptr %args, i32 0, i32 0
  %t2 = load i64, ptr %t1
  %t3 = extractvalue %str { ptr @.str.31, i64 6 }, 0
  %t4 = extractvalue %str { ptr @.str.31, i64 6 }, 1
  call void @veles_call_base(ptr %task, ptr %t3)
  %t5 = call ptr @ramp.v_main.count(ptr %task, ptr @veles.root.link, i64 %t2)
  call void @veles_task_started(ptr %task, ptr %t5)
  ret void
}

define internal %str @show.T_u8_main.Counter_(ptr %v) {
entry:
  %a3 = alloca %str
  %a10 = alloca %str
  %a16 = alloca %str
  %a25 = alloca %str
  %a31 = alloca %str
  %t1 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %v, i32 0, i32 0
  %t2 = load i8, ptr %t1
  %t4 = zext i8 %t2 to i64
  call void @veles_u64_to_string(ptr %a3, i64 %t4)
  %t5 = load %str, ptr %a3
  %t6 = extractvalue %str { ptr @.str.32, i64 1 }, 0
  %t7 = extractvalue %str { ptr @.str.32, i64 1 }, 1
  %t8 = extractvalue %str %t5, 0
  %t9 = extractvalue %str %t5, 1
  call void @veles_string_concat(ptr %a10, ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  %t11 = load %str, ptr %a10
  %t12 = extractvalue %str %t11, 0
  %t13 = extractvalue %str %t11, 1
  %t14 = extractvalue %str { ptr @.str.33, i64 2 }, 0
  %t15 = extractvalue %str { ptr @.str.33, i64 2 }, 1
  call void @veles_string_concat(ptr %a16, ptr %t12, i64 %t13, ptr %t14, i64 %t15)
  %t17 = load %str, ptr %a16
  %t18 = getelementptr inbounds { i8, [63 x i8], %S.main.Counter }, ptr %v, i32 0, i32 2
  %t19 = load %S.main.Counter, ptr %t18
  %t20 = call %str @show.main.Counter(%S.main.Counter %t19)
  %t21 = extractvalue %str %t17, 0
  %t22 = extractvalue %str %t17, 1
  %t23 = extractvalue %str %t20, 0
  %t24 = extractvalue %str %t20, 1
  call void @veles_string_concat(ptr %a25, ptr %t21, i64 %t22, ptr %t23, i64 %t24)
  %t26 = load %str, ptr %a25
  %t27 = extractvalue %str %t26, 0
  %t28 = extractvalue %str %t26, 1
  %t29 = extractvalue %str { ptr @.str.34, i64 1 }, 0
  %t30 = extractvalue %str { ptr @.str.34, i64 1 }, 1
  call void @veles_string_concat(ptr %a31, ptr %t27, i64 %t28, ptr %t29, i64 %t30)
  %t32 = load %str, ptr %a31
  ret %str %t32
}

define internal %str @show.main.Counter(%S.main.Counter %v) {
entry:
  %a5 = alloca %str
  %a8 = alloca %str
  %a14 = alloca %str
  %a20 = alloca %str
  %t1 = extractvalue %str { ptr @.str.35, i64 8 }, 0
  %t2 = extractvalue %str { ptr @.str.35, i64 8 }, 1
  %t3 = extractvalue %str { ptr @.str.36, i64 6 }, 0
  %t4 = extractvalue %str { ptr @.str.36, i64 6 }, 1
  call void @veles_string_concat(ptr %a5, ptr %t1, i64 %t2, ptr %t3, i64 %t4)
  %t6 = load %str, ptr %a5
  %t7 = extractvalue %S.main.Counter %v, 0
  call void @veles_i64_to_string(ptr %a8, i64 %t7)
  %t9 = load %str, ptr %a8
  %t10 = extractvalue %str %t6, 0
  %t11 = extractvalue %str %t6, 1
  %t12 = extractvalue %str %t9, 0
  %t13 = extractvalue %str %t9, 1
  call void @veles_string_concat(ptr %a14, ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  %t15 = load %str, ptr %a14
  %t16 = extractvalue %str %t15, 0
  %t17 = extractvalue %str %t15, 1
  %t18 = extractvalue %str { ptr @.str.34, i64 1 }, 0
  %t19 = extractvalue %str { ptr @.str.34, i64 1 }, 1
  call void @veles_string_concat(ptr %a20, ptr %t16, i64 %t17, ptr %t18, i64 %t19)
  %t21 = load %str, ptr %a20
  ret %str %t21
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.37, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.37, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task, ptr @veles.root.link)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

@.str.1 = private unnamed_addr constant [9 x i8] c"negative\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:47:3\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:51:28\00"
@.str.5 = private unnamed_addr constant [5 x i8] c"hot \00"
@.str.6 = private unnamed_addr constant [6 x i8] c"cold \00"
@.str.7 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.8 = private unnamed_addr constant [14 x i8] c"main.vs:53:30\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:62:3\00"
@.str.10 = private unnamed_addr constant [21 x i8] c"main.vs:67:18\00offset\00"
@.str.11 = private unnamed_addr constant [21 x i8] c"main.vs:68:35\00offset\00"
@.str.12 = private unnamed_addr constant [9 x i8] c"offsets \00"
@.str.13 = private unnamed_addr constant [2 x i8] c" \00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:69:5\00println\00"
@.str.15 = private unnamed_addr constant [19 x i8] c"main.vs:71:39\00read\00"
@.str.16 = private unnamed_addr constant [1 x i8] c"\00"
@.str.17 = private unnamed_addr constant [2 x i8] c"c\00"
@.str.18 = private unnamed_addr constant [19 x i8] c"main.vs:71:53\00read\00"
@.str.19 = private unnamed_addr constant [21 x i8] c"main.vs:71:3\00println\00"
@.str.20 = private unnamed_addr constant [21 x i8] c"main.vs:74:15\00millis\00"
@.str.21 = private unnamed_addr constant [20 x i8] c"main.vs:75:10\00total\00"
@.str.22 = private unnamed_addr constant [13 x i8] c"main.vs:75:3\00"
@.str.23 = private unnamed_addr constant [21 x i8] c"main.vs:76:15\00millis\00"
@.str.24 = private unnamed_addr constant [20 x i8] c"main.vs:77:30\00total\00"
@.str.25 = private unnamed_addr constant [7 x i8] c"frame \00"
@.str.26 = private unnamed_addr constant [21 x i8] c"main.vs:77:3\00println\00"
@.str.27 = private unnamed_addr constant [19 x i8] c"main.vs:78:23\00both\00"
@.str.28 = private unnamed_addr constant [19 x i8] c"main.vs:78:43\00both\00"
@.str.29 = private unnamed_addr constant [7 x i8] c"tasks \00"
@.str.30 = private unnamed_addr constant [21 x i8] c"main.vs:78:3\00println\00"
@.str.31 = private unnamed_addr constant [7 x i8] c"\00count\00"
@.str.32 = private unnamed_addr constant [2 x i8] c"(\00"
@.str.33 = private unnamed_addr constant [3 x i8] c", \00"
@.str.34 = private unnamed_addr constant [2 x i8] c")\00"
@.str.35 = private unnamed_addr constant [9 x i8] c"Counter(\00"
@.str.36 = private unnamed_addr constant [7 x i8] c"hits: \00"
@.str.37 = private unnamed_addr constant [6 x i8] c"\00main\00"
