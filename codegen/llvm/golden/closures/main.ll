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
  %a64 = alloca i64
  %a70 = alloca ptr
  %a72 = alloca ptr
  %a75 = alloca { ptr, ptr }
  %a76 = alloca i64
  %a77 = alloca i64
  %a96 = alloca i64
  %a108 = alloca i64
  %a111 = alloca [21 x i8]
  %a119 = alloca [21 x i8]
  %a126 = alloca [21 x i8]
  %a130 = alloca %str
  %a131 = alloca [7 x %str]
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
  %t30 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t29, i32 0, i32 1
  %t31 = load i64, ptr %t30
  %t32 = icmp slt i64 %t28, %t31
  br i1 %t32, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t33 = load ptr, ptr %a21
  %t34 = load i64, ptr %a27
  %t35 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t33, i32 0, i32 1
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
  %t41 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t33, i32 0, i32 3
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
  %t53 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t47, i32 0, i32 1
  %t54 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t47, i32 0, i32 2
  %t55 = load i64, ptr %t53
  %t56 = load i64, ptr %t54
  %t57 = icmp slt i64 %t55, %t56
  br i1 %t57, label %push.fast.7, label %push.grow.8, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.7:
  %t58 = load ptr, ptr %t47
  %t59 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t47, i32 0, i32 3
  %t60 = load i64, ptr %t59
  %t61 = mul i64 %t60, %t55
  %t62 = getelementptr inbounds i8, ptr %t58, i64 %t61
  store i64 %t52, ptr %t62
  %t63 = add i64 %t55, 1
  store i64 %t63, ptr %t53
  br label %push.done.9
push.grow.8:
  store i64 %t52, ptr %a64
  call void @veles_list_push(ptr %t47, ptr %a64)
  br label %push.done.9
push.done.9:
  br label %loop.post.2
loop.post.2:
  %t65 = load i64, ptr %a27
  %t66 = add i64 %t65, 1
  store i64 %t66, ptr %a27
  %t67 = load volatile i32, ptr @veles_stop_requested, align 4
  %t68 = icmp ne i32 %t67, 0
  br i1 %t68, label %safepoint.10, label %safepoint.on.11, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.10:
  call void @veles_gc_park()
  br label %safepoint.on.11
safepoint.on.11:
  br label %loop.cond.1
loop.end.3:
  %t69 = load ptr, ptr %a26
  store ptr %t69, ptr %a70
  %t71 = load ptr, ptr %a70
  store ptr %t71, ptr %a72
  %t73 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t74 = insertvalue { ptr, ptr } %t73, ptr null, 1
  store { ptr, ptr } %t74, ptr %a75
  store i64 0, ptr %a76
  store i64 0, ptr %a77
  br label %loop.cond.12
loop.cond.12:
  %t78 = load i64, ptr %a77
  %t79 = load ptr, ptr %a72
  %t80 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t79, i32 0, i32 1
  %t81 = load i64, ptr %t80
  %t82 = icmp slt i64 %t78, %t81
  br i1 %t82, label %loop.body.15, label %loop.end.14
loop.body.15:
  %t83 = load ptr, ptr %a72
  %t84 = load i64, ptr %a77
  %t85 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t83, i32 0, i32 1
  %t86 = load i64, ptr %t85
  %t87 = icmp ult i64 %t84, %t86
  br i1 %t87, label %idx.ok.16, label %idx.bad.17, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.17:
  %t88 = extractvalue %str { ptr @.str.3, i64 13 }, 0
  %t89 = extractvalue %str { ptr @.str.3, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t83, i64 %t84, ptr %t88, i64 %t89)
  unreachable
idx.ok.16:
  %t90 = load ptr, ptr %t83
  %t91 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t83, i32 0, i32 3
  %t92 = load i64, ptr %t91
  %t93 = mul i64 %t92, %t84
  %t94 = getelementptr inbounds i8, ptr %t90, i64 %t93
  %t95 = load i64, ptr %t94
  store i64 %t95, ptr %a96
  %t97 = load { ptr, ptr }, ptr %a75
  %t98 = extractvalue { ptr, ptr } %t97, 0
  %t99 = extractvalue { ptr, ptr } %t97, 1
  %t100 = load i64, ptr %a76
  %t101 = load i64, ptr %a96
  %t102 = call i64 %t98(ptr %t99, i64 %t100, i64 %t101)
  store i64 %t102, ptr %a76
  br label %loop.post.13
loop.post.13:
  %t103 = load i64, ptr %a77
  %t104 = add i64 %t103, 1
  store i64 %t104, ptr %a77
  %t105 = load volatile i32, ptr @veles_stop_requested, align 4
  %t106 = icmp ne i32 %t105, 0
  br i1 %t106, label %safepoint.18, label %safepoint.on.19, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.18:
  call void @veles_gc_park()
  br label %safepoint.on.19
safepoint.on.19:
  br label %loop.cond.12
loop.end.14:
  %t107 = load i64, ptr %a76
  store i64 %t107, ptr %a108
  %t109 = load ptr, ptr %a1
  %t110 = load i64, ptr %t109
  %t112 = call i64 @veles_i64_format(ptr %a111, i64 %t110)
  %t113 = insertvalue %str undef, ptr %a111, 0
  %t114 = insertvalue %str %t113, i64 %t112, 1
  %t115 = load { ptr, ptr }, ptr %a18
  %t116 = extractvalue %str { ptr @.str.4, i64 19 }, 0
  %t117 = extractvalue %str { ptr @.str.4, i64 19 }, 1
  call void @veles_call_push(ptr %t116)
  %t118 = call i64 @v_main.apply({ ptr, ptr } %t115, i64 10)
  call void @veles_call_pop()
  %t120 = call i64 @veles_i64_format(ptr %a119, i64 %t118)
  %t121 = insertvalue %str undef, ptr %a119, 0
  %t122 = insertvalue %str %t121, i64 %t120, 1
  %t123 = load ptr, ptr %a70
  %t124 = call %str @show.List_i64_(ptr %t123)
  %t125 = load i64, ptr %a108
  %t127 = call i64 @veles_i64_format(ptr %a126, i64 %t125)
  %t128 = insertvalue %str undef, ptr %a126, 0
  %t129 = insertvalue %str %t128, i64 %t127, 1
  %t132 = getelementptr [7 x %str], ptr %a131, i64 0, i64 0
  store %str %t114, ptr %t132
  %t133 = getelementptr [7 x %str], ptr %a131, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t133
  %t134 = getelementptr [7 x %str], ptr %a131, i64 0, i64 2
  store %str %t122, ptr %t134
  %t135 = getelementptr [7 x %str], ptr %a131, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t135
  %t136 = getelementptr [7 x %str], ptr %a131, i64 0, i64 4
  store %str %t124, ptr %t136
  %t137 = getelementptr [7 x %str], ptr %a131, i64 0, i64 5
  store %str { ptr @.str.5, i64 1 }, ptr %t137
  %t138 = getelementptr [7 x %str], ptr %a131, i64 0, i64 6
  store %str %t129, ptr %t138
  call void @veles_string_concat_n(ptr %a130, ptr %a131, i64 7)
  %t139 = load %str, ptr %a130
  %t140 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t141 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t140)
  call void @v_std.io.println(%str %t139)
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
