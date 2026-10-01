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
  %a18 = alloca { ptr, ptr }
  %a20 = alloca i64
  %a21 = alloca ptr
  %a24 = alloca { ptr, ptr }
  %a26 = alloca ptr
  %a27 = alloca i64
  %a46 = alloca i64
  %a67 = alloca i64
  %a73 = alloca ptr
  %a75 = alloca ptr
  %a78 = alloca { ptr, ptr }
  %a79 = alloca i64
  %a80 = alloca i64
  %a99 = alloca i64
  %a111 = alloca i64
  %a114 = alloca [21 x i8]
  %a122 = alloca [21 x i8]
  %a129 = alloca [21 x i8]
  %a133 = alloca %str
  %a134 = alloca [7 x %str]
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
  %t15 = extractvalue %str { ptr @.str.1, i64 19 }, 0
  %t16 = extractvalue %str { ptr @.str.1, i64 19 }, 1
  call void @veles_call_push(ptr %t15)
  %t17 = call { ptr, ptr } @v_main.adder(i64 5)
  call void @veles_call_pop()
  store { ptr, ptr } %t17, ptr %a18
  %t19 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 1, ptr %a20
  call void @veles_list_push(ptr %t19, ptr %a20)
  store i64 2, ptr %a20
  call void @veles_list_push(ptr %t19, ptr %a20)
  store i64 3, ptr %a20
  call void @veles_list_push(ptr %t19, ptr %a20)
  store ptr %t19, ptr %a21
  %t22 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda3, 0
  %t23 = insertvalue { ptr, ptr } %t22, ptr null, 1
  store { ptr, ptr } %t23, ptr %a24
  %t25 = call ptr @veles_list_new(ptr @adesc.i64, i64 0)
  store ptr %t25, ptr %a26
  store i64 0, ptr %a27
  br label %loop.cond.1
loop.cond.1:
  %t28 = load i64, ptr %a27
  %t29 = load ptr, ptr %a21
  %t30 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t29, i32 0, i32 1
  %t31 = load i64, ptr %t30
  %t32 = icmp slt i64 %t28, %t31
  br i1 %t32, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t33 = load ptr, ptr %a21
  %t34 = load i64, ptr %a27
  %t35 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t33, i32 0, i32 1
  %t36 = load i64, ptr %t35
  %t37 = icmp ult i64 %t34, %t36
  br i1 %t37, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t38 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t39 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t33, i64 %t34, ptr %t38, i64 %t39)
  unreachable
idx.ok.5:
  %t40 = load ptr, ptr %t33
  %t41 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t33, i32 0, i32 3
  %t42 = load i64, ptr %t41
  %t43 = mul i64 %t42, %t34
  %t44 = getelementptr inbounds i8, ptr %t40, i64 %t43
  %t45 = load i64, ptr %t44
  store i64 %t45, ptr %a46
  %t47 = load ptr, ptr %a26
  %t48 = load { ptr, ptr }, ptr %a24
  %t49 = extractvalue { ptr, ptr } %t48, 0
  %t50 = extractvalue { ptr, ptr } %t48, 1
  %t51 = load i64, ptr %a46
  %t52 = call i64 %t49(ptr %t50, i64 %t51)
  %t53 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t47, i32 0, i32 1
  %t54 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t47, i32 0, i32 2
  %t55 = load i64, ptr %t53
  %t56 = load i64, ptr %t54
  %t57 = icmp slt i64 %t55, %t56
  br i1 %t57, label %push.fast.7, label %push.grow.8, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.7:
  %t58 = load ptr, ptr %t47
  %t59 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t47, i32 0, i32 3
  %t60 = load i64, ptr %t59
  %t61 = mul i64 %t60, %t55
  %t62 = getelementptr inbounds i8, ptr %t58, i64 %t61
  store i64 %t52, ptr %t62
  %t63 = add i64 %t55, 1
  store i64 %t63, ptr %t53
  %t64 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t47, i32 0, i32 5
  %t65 = load i64, ptr %t64
  %t66 = add i64 %t65, 1
  store i64 %t66, ptr %t64
  br label %push.done.9
push.grow.8:
  store i64 %t52, ptr %a67
  call void @veles_list_push(ptr %t47, ptr %a67)
  br label %push.done.9
push.done.9:
  br label %loop.post.2
loop.post.2:
  %t68 = load i64, ptr %a27
  %t69 = add i64 %t68, 1
  store i64 %t69, ptr %a27
  %t70 = load volatile i32, ptr @veles_stop_requested, align 4
  %t71 = icmp ne i32 %t70, 0
  br i1 %t71, label %safepoint.10, label %safepoint.on.11, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.10:
  call void @veles_gc_park()
  br label %safepoint.on.11
safepoint.on.11:
  br label %loop.cond.1
loop.end.3:
  %t72 = load ptr, ptr %a26
  store ptr %t72, ptr %a73
  %t74 = load ptr, ptr %a73
  store ptr %t74, ptr %a75
  %t76 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t77 = insertvalue { ptr, ptr } %t76, ptr null, 1
  store { ptr, ptr } %t77, ptr %a78
  store i64 0, ptr %a79
  store i64 0, ptr %a80
  br label %loop.cond.12
loop.cond.12:
  %t81 = load i64, ptr %a80
  %t82 = load ptr, ptr %a75
  %t83 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t82, i32 0, i32 1
  %t84 = load i64, ptr %t83
  %t85 = icmp slt i64 %t81, %t84
  br i1 %t85, label %loop.body.15, label %loop.end.14
loop.body.15:
  %t86 = load ptr, ptr %a75
  %t87 = load i64, ptr %a80
  %t88 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t86, i32 0, i32 1
  %t89 = load i64, ptr %t88
  %t90 = icmp ult i64 %t87, %t89
  br i1 %t90, label %idx.ok.16, label %idx.bad.17, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.17:
  %t91 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t92 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t86, i64 %t87, ptr %t91, i64 %t92)
  unreachable
idx.ok.16:
  %t93 = load ptr, ptr %t86
  %t94 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t86, i32 0, i32 3
  %t95 = load i64, ptr %t94
  %t96 = mul i64 %t95, %t87
  %t97 = getelementptr inbounds i8, ptr %t93, i64 %t96
  %t98 = load i64, ptr %t97
  store i64 %t98, ptr %a99
  %t100 = load { ptr, ptr }, ptr %a78
  %t101 = extractvalue { ptr, ptr } %t100, 0
  %t102 = extractvalue { ptr, ptr } %t100, 1
  %t103 = load i64, ptr %a79
  %t104 = load i64, ptr %a99
  %t105 = call i64 %t101(ptr %t102, i64 %t103, i64 %t104)
  store i64 %t105, ptr %a79
  br label %loop.post.13
loop.post.13:
  %t106 = load i64, ptr %a80
  %t107 = add i64 %t106, 1
  store i64 %t107, ptr %a80
  %t108 = load volatile i32, ptr @veles_stop_requested, align 4
  %t109 = icmp ne i32 %t108, 0
  br i1 %t109, label %safepoint.18, label %safepoint.on.19, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.18:
  call void @veles_gc_park()
  br label %safepoint.on.19
safepoint.on.19:
  br label %loop.cond.12
loop.end.14:
  %t110 = load i64, ptr %a79
  store i64 %t110, ptr %a111
  %t112 = load ptr, ptr %a1
  %t113 = load i64, ptr %t112
  %t115 = call i64 @veles_i64_format(ptr %a114, i64 %t113)
  %t116 = insertvalue %str undef, ptr %a114, 0
  %t117 = insertvalue %str %t116, i64 %t115, 1
  %t118 = load { ptr, ptr }, ptr %a18
  %t119 = extractvalue %str { ptr @.str.4, i64 19 }, 0
  %t120 = extractvalue %str { ptr @.str.4, i64 19 }, 1
  call void @veles_call_push(ptr %t119)
  %t121 = call i64 @v_main.apply({ ptr, ptr } %t118, i64 10)
  call void @veles_call_pop()
  %t123 = call i64 @veles_i64_format(ptr %a122, i64 %t121)
  %t124 = insertvalue %str undef, ptr %a122, 0
  %t125 = insertvalue %str %t124, i64 %t123, 1
  %t126 = load ptr, ptr %a73
  %t127 = call %str @show.List_i64_(ptr %t126)
  %t128 = load i64, ptr %a111
  %t130 = call i64 @veles_i64_format(ptr %a129, i64 %t128)
  %t131 = insertvalue %str undef, ptr %a129, 0
  %t132 = insertvalue %str %t131, i64 %t130, 1
  %t135 = getelementptr [7 x %str], ptr %a134, i64 0, i64 0
  store %str %t117, ptr %t135
  %t136 = getelementptr [7 x %str], ptr %a134, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t136
  %t137 = getelementptr [7 x %str], ptr %a134, i64 0, i64 2
  store %str %t125, ptr %t137
  %t138 = getelementptr [7 x %str], ptr %a134, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t138
  %t139 = getelementptr [7 x %str], ptr %a134, i64 0, i64 4
  store %str %t127, ptr %t139
  %t140 = getelementptr [7 x %str], ptr %a134, i64 0, i64 5
  store %str { ptr @.str.5, i64 1 }, ptr %t140
  %t141 = getelementptr [7 x %str], ptr %a134, i64 0, i64 6
  store %str %t132, ptr %t141
  call void @veles_string_concat_n(ptr %a133, ptr %a134, i64 7)
  %t142 = load %str, ptr %a133
  %t143 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t144 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t143)
  call void @v_std.io.println(%str %t142)
  call void @veles_call_pop()
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
  %t12 = extractvalue %str { ptr @.str.7, i64 16 }, 0
  %t13 = extractvalue %str { ptr @.str.7, i64 16 }, 1
  %t14 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t15 = extractvalue %str { ptr @.str.8, i64 12 }, 1
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
  %t10 = extractvalue %str { ptr @.str.7, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.7, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t13 = extractvalue %str { ptr @.str.9, i64 13 }, 1
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
  %t8 = extractvalue %str { ptr @.str.7, i64 16 }, 0
  %t9 = extractvalue %str { ptr @.str.7, i64 16 }, 1
  %t10 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t11 = extractvalue %str { ptr @.str.10, i64 13 }, 1
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
  %t10 = extractvalue %str { ptr @.str.7, i64 16 }, 0
  %t11 = extractvalue %str { ptr @.str.7, i64 16 }, 1
  %t12 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t13 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_panic_at(ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  unreachable
arith.ok.2:
  ret i64 %t8
}

@.str.1 = private unnamed_addr constant [20 x i8] c"main.vs:13:14\00adder\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:14:17\00"
@.str.3 = private unnamed_addr constant [14 x i8] c"main.vs:15:15\00"
@.str.4 = private unnamed_addr constant [20 x i8] c"main.vs:16:24\00apply\00"
@.str.5 = private unnamed_addr constant [2 x i8] c" \00"
@.str.6 = private unnamed_addr constant [21 x i8] c"main.vs:16:3\00println\00"
@.str.7 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:4:41\00"
@.str.9 = private unnamed_addr constant [14 x i8] c"main.vs:10:22\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:14:36\00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:15:41\00"
