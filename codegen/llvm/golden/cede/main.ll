define ptr @v_main.checksum(ptr %task, ptr %link, ptr %p1, i64 %p2) presplitcoroutine {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a3 = alloca i64
  %a10 = alloca { i64, i64, i1 }
  %a13 = alloca i64
  %a16 = alloca i64
  %a17 = alloca i1
  %a18 = alloca i1
  %a20 = alloca i1
  %a32 = alloca i64
  %a43 = alloca ptr
  %a45 = alloca i64
  %a114 = alloca i1
  %a132 = alloca i64
  store ptr %p1, ptr %a1
  store i64 %p2, ptr %a2
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  store i64 0, ptr %a3
  %t4 = load ptr, ptr %a1
  %t5 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t4, i32 0, i32 1
  %t6 = load i64, ptr %t5
  %t7 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t8 = insertvalue { i64, i64, i1 } %t7, i64 %t6, 1
  %t9 = insertvalue { i64, i64, i1 } %t8, i1 false, 2
  store { i64, i64, i1 } %t9, ptr %a10
  %t11 = getelementptr inbounds { i64, i64, i1 }, ptr %a10, i32 0, i32 0
  %t12 = load i64, ptr %t11
  store i64 %t12, ptr %a13
  %t14 = getelementptr inbounds { i64, i64, i1 }, ptr %a10, i32 0, i32 1
  %t15 = load i64, ptr %t14
  store i64 %t15, ptr %a16
  store i1 false, ptr %a17
  br label %loop.cond.1
loop.cond.1:
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a18)
  %t19 = load i1, ptr %a17
  br i1 %t19, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a18
  br label %if.end.6
if.else.7:
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a20)
  %t21 = getelementptr inbounds { i64, i64, i1 }, ptr %a10, i32 0, i32 2
  %t22 = load i1, ptr %t21
  br i1 %t22, label %if.then.8, label %if.else.10
if.then.8:
  %t23 = load i64, ptr %a13
  %t24 = load i64, ptr %a16
  %t25 = icmp sle i64 %t23, %t24
  store i1 %t25, ptr %a20
  br label %if.end.9
if.else.10:
  %t26 = load i64, ptr %a13
  %t27 = load i64, ptr %a16
  %t28 = icmp slt i64 %t26, %t27
  store i1 %t28, ptr %a20
  br label %if.end.9
if.end.9:
  %t29 = load i1, ptr %a20
  store i1 %t29, ptr %a18
  br label %if.end.6
if.end.6:
  %t30 = load i1, ptr %a18
  br i1 %t30, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t31 = load i64, ptr %a13
  store i64 %t31, ptr %a32
  %t33 = load i64, ptr %a3
  %t35 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t33, i64 31)
  %t36 = extractvalue { i64, i1 } %t35, 0
  %t37 = extractvalue { i64, i1 } %t35, 1
  br i1 %t37, label %overflow.11, label %arith.ok.12
overflow.11:
  %t38 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t39 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t40 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t41 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t38, i64 %t39, ptr %t40, i64 %t41)
  unreachable
arith.ok.12:
  %t42 = load ptr, ptr %a1
  store ptr %t42, ptr %a43
  %t44 = load i64, ptr %a32
  store i64 %t44, ptr %a45
  %t46 = load i64, ptr %a45
  %t47 = icmp slt i64 %t46, 0
  br i1 %t47, label %if.then.13, label %if.end.14
if.then.13:
  %t48 = load i64, ptr %a45
  %t49 = load ptr, ptr %a43
  %t50 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t49, i32 0, i32 1
  %t51 = load i64, ptr %t50
  %t52 = add i64 %t48, %t51
  store i64 %t52, ptr %a45
  br label %if.end.14
if.end.14:
  %t53 = load ptr, ptr %a43
  %t54 = load i64, ptr %a45
  %t55 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 1
  %t56 = load i64, ptr %t55
  %t57 = icmp ult i64 %t54, %t56
  br i1 %t57, label %idx.ok.15, label %idx.bad.16, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.16:
  %t58 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t59 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t53, i64 %t54, ptr %t58, i64 %t59)
  unreachable
idx.ok.15:
  %t60 = load ptr, ptr %t53
  %t61 = getelementptr inbounds [8 x i8], ptr %t60, i64 %t54
  %t62 = load i64, ptr %t61
  %t64 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t36, i64 %t62)
  %t65 = extractvalue { i64, i1 } %t64, 0
  %t66 = extractvalue { i64, i1 } %t64, 1
  br i1 %t66, label %overflow.17, label %arith.ok.18
overflow.17:
  %t67 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t68 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t69 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t70 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t67, i64 %t68, ptr %t69, i64 %t70)
  unreachable
arith.ok.18:
  %t72 = icmp eq i64 1000003, 0
  br i1 %t72, label %divzero.19, label %div.ok.20
divzero.19:
  %t73 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t74 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t75 = extractvalue %str { ptr @.str.5, i64 13 }, 0
  %t76 = extractvalue %str { ptr @.str.5, i64 13 }, 1
  call void @veles_panic_at(ptr %t73, i64 %t74, ptr %t75, i64 %t76)
  unreachable
div.ok.20:
  %t77 = icmp eq i64 %t65, -9223372036854775808
  %t78 = icmp eq i64 1000003, -1
  %t79 = and i1 %t77, %t78
  br i1 %t79, label %divof.21, label %div.ok.22
divof.21:
  %t80 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t81 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t82 = extractvalue %str { ptr @.str.5, i64 13 }, 0
  %t83 = extractvalue %str { ptr @.str.5, i64 13 }, 1
  call void @veles_panic_at(ptr %t80, i64 %t81, ptr %t82, i64 %t83)
  unreachable
div.ok.22:
  %t71 = srem i64 %t65, 1000003
  store i64 %t71, ptr %a3
  %t84 = load i64, ptr %a32
  %t85 = load i64, ptr %a2
  %t87 = icmp eq i64 %t85, 0
  br i1 %t87, label %divzero.23, label %div.ok.24
divzero.23:
  %t88 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t89 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t90 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t91 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t88, i64 %t89, ptr %t90, i64 %t91)
  unreachable
div.ok.24:
  %t92 = icmp eq i64 %t84, -9223372036854775808
  %t93 = icmp eq i64 %t85, -1
  %t94 = and i1 %t92, %t93
  br i1 %t94, label %divof.25, label %div.ok.26
divof.25:
  %t95 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t96 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t97 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t98 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t95, i64 %t96, ptr %t97, i64 %t98)
  unreachable
div.ok.26:
  %t86 = srem i64 %t84, %t85
  %t99 = load i64, ptr %a2
  %t101 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t99, i64 1)
  %t102 = extractvalue { i64, i1 } %t101, 0
  %t103 = extractvalue { i64, i1 } %t101, 1
  br i1 %t103, label %overflow.27, label %arith.ok.28
overflow.27:
  %t104 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t105 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t106 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t107 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t104, i64 %t105, ptr %t106, i64 %t107)
  unreachable
arith.ok.28:
  %t108 = icmp eq i64 %t86, %t102
  br i1 %t108, label %if.then.29, label %if.end.30
if.then.29:
  br label %cede.31
cede.31:
  %t109 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t110 = icmp ne i64 %t109, 0
  br i1 %t110, label %cede.done.32, label %cede.susp.33
cede.susp.33:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t111 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t111, label %coro.suspend [ i8 0, label %resume.34 i8 1, label %coro.cleanup ]
resume.34:
  %t112 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t113 = icmp ne i64 %t112, 0
  br i1 %t113, label %cancelled.35, label %cont.36
cancelled.35:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.36:
  br label %cede.31
cede.done.32:
  br label %if.end.30
if.end.30:
  br label %loop.post.2
loop.post.2:
  %t115 = getelementptr inbounds { i64, i64, i1 }, ptr %a10, i32 0, i32 2
  %t116 = load i1, ptr %t115
  store i1 %t116, ptr %a114
  br i1 %t116, label %sc.rhs.37, label %sc.end.38
sc.rhs.37:
  %t117 = load i64, ptr %a13
  %t118 = load i64, ptr %a16
  %t119 = icmp eq i64 %t117, %t118
  store i1 %t119, ptr %a114
  br label %sc.end.38
sc.end.38:
  %t120 = load i1, ptr %a114
  br i1 %t120, label %if.then.39, label %if.else.41
if.then.39:
  store i1 true, ptr %a17
  br label %if.end.40
if.else.41:
  %t121 = load i64, ptr %a13
  %t122 = add i64 %t121, 1
  store i64 %t122, ptr %a13
  br label %if.end.40
if.end.40:
  %t123 = load volatile i32, ptr @veles_attention_line, align 64
  %t124 = icmp ne i32 %t123, 0
  br i1 %t124, label %safepoint.42, label %safepoint.on.43, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.42:
  %t125 = call i64 @veles_backedge(ptr %task, i64 %coro.depth)
  switch i64 %t125, label %safepoint.on.43 [ i64 0, label %backedge.look.46 i64 1, label %backedge.yield.44 i64 2, label %backedge.cancel.45 ]
backedge.look.46:
  br label %safepoint.on.43
backedge.yield.44:
  br label %yield.47
yield.47:
  %t126 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t127 = icmp ne i64 %t126, 0
  br i1 %t127, label %safepoint.on.43, label %yield.susp.48
yield.susp.48:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t128 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t128, label %coro.suspend [ i8 0, label %resume.49 i8 1, label %coro.cleanup ]
resume.49:
  %t129 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t130 = icmp ne i64 %t129, 0
  br i1 %t130, label %cancelled.50, label %cont.51
cancelled.50:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.51:
  br label %yield.47
backedge.cancel.45:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
safepoint.on.43:
  br label %loop.cond.1
loop.end.3:
  %t131 = load i64, ptr %a3
  call void @llvm.lifetime.start.p0(i64 -1, ptr %a132)
  store i64 %t131, ptr %a132
  %t133 = load ptr, ptr %link
  %t134 = icmp eq ptr %t133, null
  br i1 %t134, label %ret.task.52, label %ret.call.53
ret.task.52:
  call void @veles_frame_return(ptr %task, ptr %link, ptr %a132, i64 8, i64 0, ptr null)
  br label %coro.final
ret.call.53:
  %t135 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %link, i32 0, i32 3
  store i64 %t131, ptr %t135
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

define ptr @v_main.main(ptr %task, ptr %link) presplitcoroutine {
entry:
  %a6 = alloca ptr
  %a10 = alloca { ptr, i64, i64, i64 }
  %a26 = alloca [21 x i8]
  %a30 = alloca %str
  %a31 = alloca [1 x %str]
  %coro.id = call token @llvm.coro.id(i32 8, ptr null, ptr null, ptr null)
  %coro.size = call i64 @llvm.coro.size.i64()
  %coro.align = call i64 @llvm.coro.align.i64()
  %coro.mem = call ptr @veles.frame.alloc(ptr %task, ptr %link, i64 %coro.size, i64 %coro.align)
  %coro.hdl = call ptr @llvm.coro.begin(token %coro.id, ptr %coro.mem)
  %coro.depthp = getelementptr inbounds { ptr, i64, i64 }, ptr %link, i32 0, i32 1
  %coro.depth = load i64, ptr %coro.depthp
  %t1 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t2 = insertvalue { ptr, ptr } %t1, ptr null, 1
  %t3 = extractvalue %str { ptr @.str.8, i64 18 }, 0
  %t4 = extractvalue %str { ptr @.str.8, i64 18 }, 1
  call void @veles_call_push(ptr %t3)
  %t5 = call ptr @v_std.prelude.extend.MutableList_T.make_T_i64_(i64 100, { ptr, ptr } %t2)
  call void @veles_call_pop()
  store ptr %t5, ptr %a6
  %t7 = load ptr, ptr %a6
  %t8 = extractvalue %str { ptr @.str.9, i64 22 }, 0
  %t9 = extractvalue %str { ptr @.str.9, i64 22 }, 1
  call void @veles_call_push(ptr %t8)
  %t11 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a10, i32 0, i32 0
  store ptr %coro.hdl, ptr %t11
  %t12 = add i64 %coro.depth, 1
  %t13 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a10, i32 0, i32 1
  store i64 %t12, ptr %t13
  %t14 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a10, i32 0, i32 2
  store i64 0, ptr %t14
  %t15 = call ptr @v_main.checksum(ptr %task, ptr %a10, ptr %t7, i64 16)
  br label %call.check.1
call.check.1:
  %t16 = load i64, ptr %t14
  %t17 = icmp ne i64 %t16, 0
  br i1 %t17, label %call.done.2, label %call.wait.3
call.wait.3:
  %t18 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t18, label %coro.suspend [ i8 0, label %resume.4 i8 1, label %coro.cleanup ]
resume.4:
  %t19 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t20 = icmp ne i64 %t19, 0
  br i1 %t20, label %cancelled.5, label %cont.6
cancelled.5:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.6:
  br label %call.check.1
call.done.2:
  call void @veles.frame.free(ptr %task, ptr %t15)
  %t21 = icmp eq i64 %t16, 2
  br i1 %t21, label %call.unwound.7, label %call.ok.8, !prof !{!"branch_weights", i32 1, i32 100000}
call.unwound.7:
  %t22 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t23 = icmp ne i64 %t22, 0
  br i1 %t23, label %call.cancelled.9, label %call.look.10
call.look.10:
  br label %call.cancelled.9
call.cancelled.9:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
call.ok.8:
  %t24 = getelementptr inbounds { ptr, i64, i64, i64 }, ptr %a10, i32 0, i32 3
  %t25 = load i64, ptr %t24
  call void @veles_call_pop()
  %t27 = call i64 @veles_i64_format(ptr %a26, i64 %t25)
  %t28 = insertvalue %str undef, ptr %a26, 0
  %t29 = insertvalue %str %t28, i64 %t27, 1
  %t32 = getelementptr [1 x %str], ptr %a31, i64 0, i64 0
  store %str %t29, ptr %t32
  call void @veles_string_concat_n(ptr %a30, ptr %a31, i64 1)
  %t33 = load %str, ptr %a30
  %t34 = extractvalue %str { ptr @.str.10, i64 20 }, 0
  %t35 = extractvalue %str { ptr @.str.10, i64 20 }, 1
  call void @veles_call_push(ptr %t34)
  call void @v_std.io.println(%str %t33)
  call void @veles_call_pop()
  br label %cede.11
cede.11:
  %t36 = call i64 @veles_task_sleep(ptr %task, i64 0)
  %t37 = icmp ne i64 %t36, 0
  br i1 %t37, label %cede.done.12, label %cede.susp.13
cede.susp.13:
  call void @veles_frame_park(ptr %task, ptr %coro.hdl, i64 %coro.depth)
  %t38 = call i8 @llvm.coro.suspend(token none, i1 false)
  switch i8 %t38, label %coro.suspend [ i8 0, label %resume.14 i8 1, label %coro.cleanup ]
resume.14:
  %t39 = call i64 @veles_task_cancelled(ptr %task, i64 %coro.depth)
  %t40 = icmp ne i64 %t39, 0
  br i1 %t40, label %cancelled.15, label %cont.16
cancelled.15:
  call void @veles_frame_unwound(ptr %task, ptr %link)
  br label %coro.final
cont.16:
  br label %cede.11
cede.done.12:
  %t41 = extractvalue %str { ptr @.str.11, i64 20 }, 0
  %t42 = extractvalue %str { ptr @.str.11, i64 20 }, 1
  call void @veles_call_push(ptr %t41)
  call void @v_std.io.println(%str { ptr @.str.12, i64 7 })
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

define i64 @v_main.main.lambda1(ptr %env, i64 %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  %t3 = load i64, ptr %a2
  %t4 = load i64, ptr %a2
  %t6 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t3, i64 %t4)
  %t7 = extractvalue { i64, i1 } %t6, 0
  %t8 = extractvalue { i64, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t12 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  ret i64 %t7
}

define internal void @entry.v_main.main(ptr %task, ptr %args) {
entry:
  %t1 = extractvalue %str { ptr @.str.14, i64 5 }, 0
  %t2 = extractvalue %str { ptr @.str.14, i64 5 }, 1
  call void @veles_call_base(ptr %task, ptr %t1)
  %t3 = call ptr @v_main.main(ptr %task, ptr @veles.root.link)
  call void @veles_task_started(ptr %task, ptr %t3)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:10:12\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:10:23\00"
@.str.4 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.5 = private unnamed_addr constant [14 x i8] c"main.vs:10:11\00"
@.str.6 = private unnamed_addr constant [13 x i8] c"main.vs:11:9\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:11:22\00"
@.str.8 = private unnamed_addr constant [19 x i8] c"main.vs:17:14\00make\00"
@.str.9 = private unnamed_addr constant [23 x i8] c"main.vs:18:17\00checksum\00"
@.str.10 = private unnamed_addr constant [21 x i8] c"main.vs:18:3\00println\00"
@.str.11 = private unnamed_addr constant [21 x i8] c"main.vs:20:3\00println\00"
@.str.12 = private unnamed_addr constant [8 x i8] c"resumed\00"
@.str.13 = private unnamed_addr constant [14 x i8] c"main.vs:17:46\00"
@.str.14 = private unnamed_addr constant [6 x i8] c"\00main\00"
