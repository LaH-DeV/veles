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
  %a51 = alloca i64
  %a57 = alloca ptr
  %a59 = alloca ptr
  %a62 = alloca { ptr, ptr }
  %a63 = alloca i64
  %a64 = alloca i64
  %a83 = alloca i64
  %a95 = alloca i64
  %a98 = alloca %str
  %a104 = alloca %str
  %a108 = alloca %str
  %a114 = alloca %str
  %a120 = alloca %str
  %a128 = alloca %str
  %a134 = alloca %str
  %a137 = alloca %str
  %a143 = alloca %str
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
  store i64 %t50, ptr %a51
  call void @veles_list_push(ptr %t45, ptr %a51)
  br label %loop.post.2
loop.post.2:
  %t52 = load i64, ptr %a25
  %t53 = add i64 %t52, 1
  store i64 %t53, ptr %a25
  %t54 = load volatile i32, ptr @veles_stop_requested, align 4
  %t55 = icmp ne i32 %t54, 0
  br i1 %t55, label %safepoint.7, label %safepoint.on.8, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.7:
  call void @veles_gc_park()
  br label %safepoint.on.8
safepoint.on.8:
  br label %loop.cond.1
loop.end.3:
  %t56 = load ptr, ptr %a24
  store ptr %t56, ptr %a57
  %t58 = load ptr, ptr %a57
  store ptr %t58, ptr %a59
  %t60 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t61 = insertvalue { ptr, ptr } %t60, ptr null, 1
  store { ptr, ptr } %t61, ptr %a62
  store i64 0, ptr %a63
  store i64 0, ptr %a64
  br label %loop.cond.9
loop.cond.9:
  %t65 = load i64, ptr %a64
  %t66 = load ptr, ptr %a59
  %t67 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t66, i32 0, i32 1
  %t68 = load i64, ptr %t67
  %t69 = icmp slt i64 %t65, %t68
  br i1 %t69, label %loop.body.12, label %loop.end.11
loop.body.12:
  %t70 = load ptr, ptr %a59
  %t71 = load i64, ptr %a64
  %t72 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t70, i32 0, i32 1
  %t73 = load i64, ptr %t72
  %t74 = icmp ult i64 %t71, %t73
  br i1 %t74, label %idx.ok.13, label %idx.bad.14, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.14:
  %t75 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t76 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t70, i64 %t71, ptr %t75, i64 %t76)
  unreachable
idx.ok.13:
  %t77 = load ptr, ptr %t70
  %t78 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t70, i32 0, i32 3
  %t79 = load i64, ptr %t78
  %t80 = mul i64 %t79, %t71
  %t81 = getelementptr inbounds i8, ptr %t77, i64 %t80
  %t82 = load i64, ptr %t81
  store i64 %t82, ptr %a83
  %t84 = load { ptr, ptr }, ptr %a62
  %t85 = extractvalue { ptr, ptr } %t84, 0
  %t86 = extractvalue { ptr, ptr } %t84, 1
  %t87 = load i64, ptr %a63
  %t88 = load i64, ptr %a83
  %t89 = call i64 %t85(ptr %t86, i64 %t87, i64 %t88)
  store i64 %t89, ptr %a63
  br label %loop.post.10
loop.post.10:
  %t90 = load i64, ptr %a64
  %t91 = add i64 %t90, 1
  store i64 %t91, ptr %a64
  %t92 = load volatile i32, ptr @veles_stop_requested, align 4
  %t93 = icmp ne i32 %t92, 0
  br i1 %t93, label %safepoint.15, label %safepoint.on.16, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.15:
  call void @veles_gc_park()
  br label %safepoint.on.16
safepoint.on.16:
  br label %loop.cond.9
loop.end.11:
  %t94 = load i64, ptr %a63
  store i64 %t94, ptr %a95
  %t96 = load ptr, ptr %a1
  %t97 = load i64, ptr %t96
  call void @veles_i64_to_string(ptr %a98, i64 %t97)
  %t99 = load %str, ptr %a98
  %t100 = extractvalue %str %t99, 0
  %t101 = extractvalue %str %t99, 1
  %t102 = extractvalue %str { ptr @.str.3, i64 1 }, 0
  %t103 = extractvalue %str { ptr @.str.3, i64 1 }, 1
  call void @veles_string_concat(ptr %a104, ptr %t100, i64 %t101, ptr %t102, i64 %t103)
  %t105 = load %str, ptr %a104
  %t106 = load { ptr, ptr }, ptr %a16
  %t107 = call i64 @v_main.apply({ ptr, ptr } %t106, i64 10)
  call void @veles_i64_to_string(ptr %a108, i64 %t107)
  %t109 = load %str, ptr %a108
  %t110 = extractvalue %str %t105, 0
  %t111 = extractvalue %str %t105, 1
  %t112 = extractvalue %str %t109, 0
  %t113 = extractvalue %str %t109, 1
  call void @veles_string_concat(ptr %a114, ptr %t110, i64 %t111, ptr %t112, i64 %t113)
  %t115 = load %str, ptr %a114
  %t116 = extractvalue %str %t115, 0
  %t117 = extractvalue %str %t115, 1
  %t118 = extractvalue %str { ptr @.str.3, i64 1 }, 0
  %t119 = extractvalue %str { ptr @.str.3, i64 1 }, 1
  call void @veles_string_concat(ptr %a120, ptr %t116, i64 %t117, ptr %t118, i64 %t119)
  %t121 = load %str, ptr %a120
  %t122 = load ptr, ptr %a57
  %t123 = call %str @show.List_i64_(ptr %t122)
  %t124 = extractvalue %str %t121, 0
  %t125 = extractvalue %str %t121, 1
  %t126 = extractvalue %str %t123, 0
  %t127 = extractvalue %str %t123, 1
  call void @veles_string_concat(ptr %a128, ptr %t124, i64 %t125, ptr %t126, i64 %t127)
  %t129 = load %str, ptr %a128
  %t130 = extractvalue %str %t129, 0
  %t131 = extractvalue %str %t129, 1
  %t132 = extractvalue %str { ptr @.str.3, i64 1 }, 0
  %t133 = extractvalue %str { ptr @.str.3, i64 1 }, 1
  call void @veles_string_concat(ptr %a134, ptr %t130, i64 %t131, ptr %t132, i64 %t133)
  %t135 = load %str, ptr %a134
  %t136 = load i64, ptr %a95
  call void @veles_i64_to_string(ptr %a137, i64 %t136)
  %t138 = load %str, ptr %a137
  %t139 = extractvalue %str %t135, 0
  %t140 = extractvalue %str %t135, 1
  %t141 = extractvalue %str %t138, 0
  %t142 = extractvalue %str %t138, 1
  call void @veles_string_concat(ptr %a143, ptr %t139, i64 %t140, ptr %t141, i64 %t142)
  %t144 = load %str, ptr %a143
  call void @v_std.io.println(%str %t144)
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
