define { ptr, ptr } @v_main.adder(i64 %p1) {
entry:
  %a1 = alloca ptr
  %cell1 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %cell1, ptr %a1
  store i64 %p1, ptr %cell1
  %t2 = call ptr @veles_alloc_words(i64 8)
  %t3 = load ptr, ptr %a1
  %t4 = getelementptr ptr, ptr %t2, i64 0
  store ptr %t3, ptr %t4
  %t5 = insertvalue { ptr, ptr } undef, ptr @v_main.adder.lambda1, 0
  %t6 = insertvalue { ptr, ptr } %t5, ptr %t2, 1
  ret { ptr, ptr } %t6
}

define i64 @v_main.apply({ ptr, ptr } %p1, i64 %p2) {
entry:
  %a1 = alloca { ptr, ptr }
  %a2 = alloca i64
  store { ptr, ptr } %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load { ptr, ptr }, ptr %a1
  %t4 = extractvalue { ptr, ptr } %t3, 0
  %t5 = extractvalue { ptr, ptr } %t3, 1
  %t6 = load i64, ptr %a2
  %t7 = call i64 %t4(ptr %t5, i64 %t6)
  ret i64 %t7
}

define void @v_main.main() {
entry:
  %a1 = alloca ptr
  %a8 = alloca { ptr, ptr }
  %a16 = alloca { ptr, ptr }
  %a18 = alloca i64
  %a19 = alloca ptr
  %a22 = alloca { ptr, ptr }
  %a24 = alloca ptr
  %a25 = alloca i64
  %a44 = alloca i64
  %a62 = alloca i64
  %a68 = alloca ptr
  %a70 = alloca ptr
  %a73 = alloca { ptr, ptr }
  %a74 = alloca i64
  %a75 = alloca i64
  %a94 = alloca i64
  %a106 = alloca i64
  %a109 = alloca [21 x i8]
  %a115 = alloca [21 x i8]
  %a122 = alloca [21 x i8]
  %a126 = alloca %str
  %a127 = alloca [7 x %str]
  %t2 = call ptr @veles_gc_alloc(ptr @desc.i64, i64 8)
  store ptr %t2, ptr %a1
  store i64 0, ptr %t2
  %t3 = call ptr @veles_alloc_words(i64 8)
  %t4 = load ptr, ptr %a1
  %t5 = getelementptr ptr, ptr %t3, i64 0
  store ptr %t4, ptr %t5
  %t6 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda2, 0
  %t7 = insertvalue { ptr, ptr } %t6, ptr %t3, 1
  store { ptr, ptr } %t7, ptr %a8
  %t9 = load { ptr, ptr }, ptr %a8
  %t10 = extractvalue { ptr, ptr } %t9, 0
  %t11 = extractvalue { ptr, ptr } %t9, 1
  call void %t10(ptr %t11)
  %t12 = load { ptr, ptr }, ptr %a8
  %t13 = extractvalue { ptr, ptr } %t12, 0
  %t14 = extractvalue { ptr, ptr } %t12, 1
  call void %t13(ptr %t14)
  %t15 = call { ptr, ptr } @v_main.adder(i64 5)
  store { ptr, ptr } %t15, ptr %a16
  %t17 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 1, ptr %a18
  call void @veles_list_push(ptr %t17, ptr %a18)
  store i64 2, ptr %a18
  call void @veles_list_push(ptr %t17, ptr %a18)
  store i64 3, ptr %a18
  call void @veles_list_push(ptr %t17, ptr %a18)
  store ptr %t17, ptr %a19
  %t20 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda3, 0
  %t21 = insertvalue { ptr, ptr } %t20, ptr null, 1
  store { ptr, ptr } %t21, ptr %a22
  %t23 = call ptr @veles_list_new(ptr @adesc.i64, i64 0)
  store ptr %t23, ptr %a24
  store i64 0, ptr %a25
  br label %loop.cond.1
loop.cond.1:
  %t26 = load i64, ptr %a25
  %t27 = load ptr, ptr %a19
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t27, i32 0, i32 1
  %t29 = load i64, ptr %t28
  %t30 = icmp slt i64 %t26, %t29
  br i1 %t30, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t31 = load ptr, ptr %a19
  %t32 = load i64, ptr %a25
  %t33 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t31, i32 0, i32 1
  %t34 = load i64, ptr %t33
  %t35 = icmp ult i64 %t32, %t34
  br i1 %t35, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t36 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t37 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t31, i64 %t32, ptr %t36, i64 %t37)
  unreachable
idx.ok.5:
  %t38 = load ptr, ptr %t31
  %t39 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t31, i32 0, i32 3
  %t40 = load i64, ptr %t39
  %t41 = mul i64 %t40, %t32
  %t42 = getelementptr inbounds i8, ptr %t38, i64 %t41
  %t43 = load i64, ptr %t42
  store i64 %t43, ptr %a44
  %t45 = load ptr, ptr %a24
  %t46 = load { ptr, ptr }, ptr %a22
  %t47 = extractvalue { ptr, ptr } %t46, 0
  %t48 = extractvalue { ptr, ptr } %t46, 1
  %t49 = load i64, ptr %a44
  %t50 = call i64 %t47(ptr %t48, i64 %t49)
  %t51 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t45, i32 0, i32 1
  %t52 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t45, i32 0, i32 2
  %t53 = load i64, ptr %t51
  %t54 = load i64, ptr %t52
  %t55 = icmp slt i64 %t53, %t54
  br i1 %t55, label %push.fast.7, label %push.grow.8, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.7:
  %t56 = load ptr, ptr %t45
  %t57 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t45, i32 0, i32 3
  %t58 = load i64, ptr %t57
  %t59 = mul i64 %t58, %t53
  %t60 = getelementptr inbounds i8, ptr %t56, i64 %t59
  store i64 %t50, ptr %t60
  %t61 = add i64 %t53, 1
  store i64 %t61, ptr %t51
  br label %push.done.9
push.grow.8:
  store i64 %t50, ptr %a62
  call void @veles_list_push(ptr %t45, ptr %a62)
  br label %push.done.9
push.done.9:
  br label %loop.post.2
loop.post.2:
  %t63 = load i64, ptr %a25
  %t64 = add i64 %t63, 1
  store i64 %t64, ptr %a25
  %t65 = load volatile i32, ptr @veles_stop_requested, align 4
  %t66 = icmp ne i32 %t65, 0
  br i1 %t66, label %safepoint.10, label %safepoint.on.11, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.10:
  call void @veles_gc_park()
  br label %safepoint.on.11
safepoint.on.11:
  br label %loop.cond.1
loop.end.3:
  %t67 = load ptr, ptr %a24
  store ptr %t67, ptr %a68
  %t69 = load ptr, ptr %a68
  store ptr %t69, ptr %a70
  %t71 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t72 = insertvalue { ptr, ptr } %t71, ptr null, 1
  store { ptr, ptr } %t72, ptr %a73
  store i64 0, ptr %a74
  store i64 0, ptr %a75
  br label %loop.cond.12
loop.cond.12:
  %t76 = load i64, ptr %a75
  %t77 = load ptr, ptr %a70
  %t78 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t77, i32 0, i32 1
  %t79 = load i64, ptr %t78
  %t80 = icmp slt i64 %t76, %t79
  br i1 %t80, label %loop.body.15, label %loop.end.14
loop.body.15:
  %t81 = load ptr, ptr %a70
  %t82 = load i64, ptr %a75
  %t83 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t81, i32 0, i32 1
  %t84 = load i64, ptr %t83
  %t85 = icmp ult i64 %t82, %t84
  br i1 %t85, label %idx.ok.16, label %idx.bad.17, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.17:
  %t86 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t87 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t81, i64 %t82, ptr %t86, i64 %t87)
  unreachable
idx.ok.16:
  %t88 = load ptr, ptr %t81
  %t89 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t81, i32 0, i32 3
  %t90 = load i64, ptr %t89
  %t91 = mul i64 %t90, %t82
  %t92 = getelementptr inbounds i8, ptr %t88, i64 %t91
  %t93 = load i64, ptr %t92
  store i64 %t93, ptr %a94
  %t95 = load { ptr, ptr }, ptr %a73
  %t96 = extractvalue { ptr, ptr } %t95, 0
  %t97 = extractvalue { ptr, ptr } %t95, 1
  %t98 = load i64, ptr %a74
  %t99 = load i64, ptr %a94
  %t100 = call i64 %t96(ptr %t97, i64 %t98, i64 %t99)
  store i64 %t100, ptr %a74
  br label %loop.post.13
loop.post.13:
  %t101 = load i64, ptr %a75
  %t102 = add i64 %t101, 1
  store i64 %t102, ptr %a75
  %t103 = load volatile i32, ptr @veles_stop_requested, align 4
  %t104 = icmp ne i32 %t103, 0
  br i1 %t104, label %safepoint.18, label %safepoint.on.19, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.18:
  call void @veles_gc_park()
  br label %safepoint.on.19
safepoint.on.19:
  br label %loop.cond.12
loop.end.14:
  %t105 = load i64, ptr %a74
  store i64 %t105, ptr %a106
  %t107 = load ptr, ptr %a1
  %t108 = load i64, ptr %t107
  %t110 = call i64 @veles_i64_format(ptr %a109, i64 %t108)
  %t111 = insertvalue %str undef, ptr %a109, 0
  %t112 = insertvalue %str %t111, i64 %t110, 1
  %t113 = load { ptr, ptr }, ptr %a16
  %t114 = call i64 @v_main.apply({ ptr, ptr } %t113, i64 10)
  %t116 = call i64 @veles_i64_format(ptr %a115, i64 %t114)
  %t117 = insertvalue %str undef, ptr %a115, 0
  %t118 = insertvalue %str %t117, i64 %t116, 1
  %t119 = load ptr, ptr %a68
  %t120 = call %str @show.List_i64_(ptr %t119)
  %t121 = load i64, ptr %a106
  %t123 = call i64 @veles_i64_format(ptr %a122, i64 %t121)
  %t124 = insertvalue %str undef, ptr %a122, 0
  %t125 = insertvalue %str %t124, i64 %t123, 1
  %t128 = getelementptr [7 x %str], ptr %a127, i64 0, i64 0
  store %str %t112, ptr %t128
  %t129 = getelementptr [7 x %str], ptr %a127, i64 0, i64 1
  store %str { ptr @.str.3, i64 1 }, ptr %t129
  %t130 = getelementptr [7 x %str], ptr %a127, i64 0, i64 2
  store %str %t118, ptr %t130
  %t131 = getelementptr [7 x %str], ptr %a127, i64 0, i64 3
  store %str { ptr @.str.3, i64 1 }, ptr %t131
  %t132 = getelementptr [7 x %str], ptr %a127, i64 0, i64 4
  store %str %t120, ptr %t132
  %t133 = getelementptr [7 x %str], ptr %a127, i64 0, i64 5
  store %str { ptr @.str.3, i64 1 }, ptr %t133
  %t134 = getelementptr [7 x %str], ptr %a127, i64 0, i64 6
  store %str %t125, ptr %t134
  call void @veles_string_concat_n(ptr %a126, ptr %a127, i64 7)
  %t135 = load %str, ptr %a126
  call void @v_std.io.println(%str %t135)
  ret void
}

define i64 @v_main.adder.lambda1(ptr %env, i64 %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  %t3 = load i64, ptr %a2
  %t4 = load ptr, ptr %a1
  %t5 = getelementptr ptr, ptr %t4, i64 0
  %t6 = load ptr, ptr %t5
  %t7 = load i64, ptr %t6
  %t9 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t3, i64 %t7)
  %t10 = extractvalue { i64, i1 } %t9, 0
  %t11 = extractvalue { i64, i1 } %t9, 1
  br i1 %t11, label %overflow.1, label %arith.ok.2
overflow.1:
  %t12 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t13 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t14 = extractvalue %str { ptr @.str.5, i64 12 }, 0
  %t15 = extractvalue %str { ptr @.str.5, i64 12 }, 1
  call void @veles_panic_at(ptr %t12, i64 %t13, ptr %t14, i64 %t15)
  unreachable
arith.ok.2:
  ret i64 %t10
}

define void @v_main.main.lambda2(ptr %env) {
entry:
  %a1 = alloca ptr
  store ptr %env, ptr %a1
  %t2 = load ptr, ptr %a1
  %t3 = getelementptr ptr, ptr %t2, i64 0
  %t4 = load ptr, ptr %t3
  %t5 = load i64, ptr %t4
  %t7 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t5, i64 1)
  %t8 = extractvalue { i64, i1 } %t7, 0
  %t9 = extractvalue { i64, i1 } %t7, 1
  br i1 %t9, label %overflow.1, label %arith.ok.2
overflow.1:
  %t10 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.6, i64 13 }, 0
  %t13 = extractvalue %str { ptr @.str.6, i64 13 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
arith.ok.2:
  %t14 = load ptr, ptr %a1
  %t15 = getelementptr ptr, ptr %t14, i64 0
  %t16 = load ptr, ptr %t15
  store i64 %t8, ptr %t16
  ret void
}

define i64 @v_main.main.lambda3(ptr %env, i64 %p1) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  %t3 = load i64, ptr %a2
  %t5 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t3, i64 2)
  %t6 = extractvalue { i64, i1 } %t5, 0
  %t7 = extractvalue { i64, i1 } %t5, 1
  br i1 %t7, label %overflow.1, label %arith.ok.2
overflow.1:
  %t8 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t9 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t10 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t11 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t8, i64 %t9, ptr %t10, i64 %t11)
  unreachable
arith.ok.2:
  ret i64 %t6
}

define i64 @v_main.main.lambda4(ptr %env, i64 %p1, i64 %p2) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a3 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  store i64 %p2, ptr %a3
  %t4 = load i64, ptr %a2
  %t5 = load i64, ptr %a3
  %t7 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t4, i64 %t5)
  %t8 = extractvalue { i64, i1 } %t7, 0
  %t9 = extractvalue { i64, i1 } %t7, 1
  br i1 %t9, label %overflow.1, label %arith.ok.2
overflow.1:
  %t10 = extractvalue %str { ptr @.str.4, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.4, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.8, i64 13 }, 0
  %t13 = extractvalue %str { ptr @.str.8, i64 13 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
arith.ok.2:
  ret i64 %t8
}

@.str.1 = private unnamed_addr constant [14 x i8] c"main.vs:14:17\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:15:15\00"
@.str.3 = private unnamed_addr constant [2 x i8] c" \00"
@.str.4 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.5 = private unnamed_addr constant [13 x i8] c"main.vs:4:41\00"
@.str.6 = private unnamed_addr constant [14 x i8] c"main.vs:10:22\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:14:36\00"
@.str.8 = private unnamed_addr constant [14 x i8] c"main.vs:15:41\00"
