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
  %a55 = alloca ptr
  %a57 = alloca ptr
  %a60 = alloca { ptr, ptr }
  %a61 = alloca i64
  %a62 = alloca i64
  %a81 = alloca i64
  %a91 = alloca i64
  %a94 = alloca %str
  %a100 = alloca %str
  %a104 = alloca %str
  %a110 = alloca %str
  %a116 = alloca %str
  %a124 = alloca %str
  %a130 = alloca %str
  %a133 = alloca %str
  %a139 = alloca %str
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
  br label %loop.cond.1
loop.end.3:
  %t54 = load ptr, ptr %a24
  store ptr %t54, ptr %a55
  %t56 = load ptr, ptr %a55
  store ptr %t56, ptr %a57
  %t58 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda4, 0
  %t59 = insertvalue { ptr, ptr } %t58, ptr null, 1
  store { ptr, ptr } %t59, ptr %a60
  store i64 0, ptr %a61
  store i64 0, ptr %a62
  br label %loop.cond.7
loop.cond.7:
  %t63 = load i64, ptr %a62
  %t64 = load ptr, ptr %a57
  %t65 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t64, i32 0, i32 1
  %t66 = load i64, ptr %t65
  %t67 = icmp slt i64 %t63, %t66
  br i1 %t67, label %loop.body.10, label %loop.end.9
loop.body.10:
  %t68 = load ptr, ptr %a57
  %t69 = load i64, ptr %a62
  %t70 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t68, i32 0, i32 1
  %t71 = load i64, ptr %t70
  %t72 = icmp ult i64 %t69, %t71
  br i1 %t72, label %idx.ok.11, label %idx.bad.12, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.12:
  %t73 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t74 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t68, i64 %t69, ptr %t73, i64 %t74)
  unreachable
idx.ok.11:
  %t75 = load ptr, ptr %t68
  %t76 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t68, i32 0, i32 3
  %t77 = load i64, ptr %t76
  %t78 = mul i64 %t77, %t69
  %t79 = getelementptr inbounds i8, ptr %t75, i64 %t78
  %t80 = load i64, ptr %t79
  store i64 %t80, ptr %a81
  %t82 = load { ptr, ptr }, ptr %a60
  %t83 = extractvalue { ptr, ptr } %t82, 0
  %t84 = extractvalue { ptr, ptr } %t82, 1
  %t85 = load i64, ptr %a61
  %t86 = load i64, ptr %a81
  %t87 = call i64 %t83(ptr %t84, i64 %t85, i64 %t86)
  store i64 %t87, ptr %a61
  br label %loop.post.8
loop.post.8:
  %t88 = load i64, ptr %a62
  %t89 = add i64 %t88, 1
  store i64 %t89, ptr %a62
  br label %loop.cond.7
loop.end.9:
  %t90 = load i64, ptr %a61
  store i64 %t90, ptr %a91
  %t92 = load ptr, ptr %a1
  %t93 = load i64, ptr %t92
  call void @veles_i64_to_string(ptr %a94, i64 %t93)
  %t95 = load %str, ptr %a94
  %t96 = extractvalue %str %t95, 0
  %t97 = extractvalue %str %t95, 1
  %t98 = extractvalue %str { ptr @.str.3, i64 1 }, 0
  %t99 = extractvalue %str { ptr @.str.3, i64 1 }, 1
  call void @veles_string_concat(ptr %a100, ptr %t96, i64 %t97, ptr %t98, i64 %t99)
  %t101 = load %str, ptr %a100
  %t102 = load { ptr, ptr }, ptr %a16
  %t103 = call i64 @v_main.apply({ ptr, ptr } %t102, i64 10)
  call void @veles_i64_to_string(ptr %a104, i64 %t103)
  %t105 = load %str, ptr %a104
  %t106 = extractvalue %str %t101, 0
  %t107 = extractvalue %str %t101, 1
  %t108 = extractvalue %str %t105, 0
  %t109 = extractvalue %str %t105, 1
  call void @veles_string_concat(ptr %a110, ptr %t106, i64 %t107, ptr %t108, i64 %t109)
  %t111 = load %str, ptr %a110
  %t112 = extractvalue %str %t111, 0
  %t113 = extractvalue %str %t111, 1
  %t114 = extractvalue %str { ptr @.str.3, i64 1 }, 0
  %t115 = extractvalue %str { ptr @.str.3, i64 1 }, 1
  call void @veles_string_concat(ptr %a116, ptr %t112, i64 %t113, ptr %t114, i64 %t115)
  %t117 = load %str, ptr %a116
  %t118 = load ptr, ptr %a55
  %t119 = call %str @show.List_i64_(ptr %t118)
  %t120 = extractvalue %str %t117, 0
  %t121 = extractvalue %str %t117, 1
  %t122 = extractvalue %str %t119, 0
  %t123 = extractvalue %str %t119, 1
  call void @veles_string_concat(ptr %a124, ptr %t120, i64 %t121, ptr %t122, i64 %t123)
  %t125 = load %str, ptr %a124
  %t126 = extractvalue %str %t125, 0
  %t127 = extractvalue %str %t125, 1
  %t128 = extractvalue %str { ptr @.str.3, i64 1 }, 0
  %t129 = extractvalue %str { ptr @.str.3, i64 1 }, 1
  call void @veles_string_concat(ptr %a130, ptr %t126, i64 %t127, ptr %t128, i64 %t129)
  %t131 = load %str, ptr %a130
  %t132 = load i64, ptr %a91
  call void @veles_i64_to_string(ptr %a133, i64 %t132)
  %t134 = load %str, ptr %a133
  %t135 = extractvalue %str %t131, 0
  %t136 = extractvalue %str %t131, 1
  %t137 = extractvalue %str %t134, 0
  %t138 = extractvalue %str %t134, 1
  call void @veles_string_concat(ptr %a139, ptr %t135, i64 %t136, ptr %t137, i64 %t138)
  %t140 = load %str, ptr %a139
  call void @v_std.io.println(%str %t140)
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
