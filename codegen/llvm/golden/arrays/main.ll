%S.main.Page = type { [256 x i8], i64 }
@const.main.K = private unnamed_addr constant [4 x i32] [i32 10, i32 20, i32 30, i32 40]
define void @v_main.filled(ptr noalias sret([256 x i8]) align 1 %sret, i8 %p1) {
entry:
  %a1 = alloca i8
  %a3 = alloca [256 x i8]
  %a4 = alloca i64
  store i8 %p1, ptr %a1
  %t2 = load i8, ptr %a1
  store i64 0, ptr %a4
  br label %fill.head.1
fill.head.1:
  %t5 = load i64, ptr %a4
  %t6 = icmp slt i64 %t5, 256
  br i1 %t6, label %fill.body.2, label %fill.done.3
fill.body.2:
  %t7 = getelementptr inbounds [256 x i8], ptr %a3, i64 0, i64 %t5
  store i8 %t2, ptr %t7
  %t8 = add nuw nsw i64 %t5, 1
  store i64 %t8, ptr %a4
  br label %fill.head.1
fill.done.3:
  call void @llvm.memmove.p0.p0.i64(ptr %sret, ptr %a3, i64 256, i1 false)
  ret void
}

define i64 @v_main.weight(ptr %p1) {
entry:
  %a1 = alloca i64
  %a3 = alloca [256 x i8]
  %a4 = alloca i64
  %a13 = alloca i8
  store i64 0, ptr %a1
  %t2 = getelementptr inbounds %S.main.Page, ptr %p1, i32 0, i32 0
  call void @llvm.memmove.p0.p0.i64(ptr %a3, ptr %t2, i64 256, i1 false)
  store i64 0, ptr %a4
  br label %loop.cond.1
loop.cond.1:
  %t5 = load i64, ptr %a4
  %t6 = icmp slt i64 %t5, 256
  br i1 %t6, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t7 = load i64, ptr %a4
  %t8 = icmp ult i64 %t7, 256
  br i1 %t8, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t9 = extractvalue %str { ptr @.str.1, i64 12 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 12 }, 1
  call void @veles_array_index_panic(i64 %t7, i64 256, ptr %t9, i64 %t10)
  unreachable
idx.ok.5:
  %t11 = getelementptr inbounds [256 x i8], ptr %a3, i64 0, i64 %t7
  %t12 = load i8, ptr %t11
  store i8 %t12, ptr %a13
  %t14 = load i64, ptr %a1
  %t15 = load i8, ptr %a13
  %t16 = zext i8 %t15 to i64
  %t18 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t14, i64 %t16)
  %t19 = extractvalue { i64, i1 } %t18, 0
  %t20 = extractvalue { i64, i1 } %t18, 1
  br i1 %t20, label %overflow.7, label %arith.ok.8
overflow.7:
  %t21 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t22 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t23 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t24 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t21, i64 %t22, ptr %t23, i64 %t24)
  unreachable
arith.ok.8:
  store i64 %t19, ptr %a1
  br label %loop.post.2
loop.post.2:
  %t25 = load i64, ptr %a4
  %t26 = add i64 %t25, 1
  store i64 %t26, ptr %a4
  %t27 = load volatile i32, ptr @veles_stop_requested, align 4
  %t28 = icmp ne i32 %t27, 0
  br i1 %t28, label %safepoint.9, label %safepoint.on.10, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.9:
  call void @veles_gc_park()
  br label %safepoint.on.10
safepoint.on.10:
  br label %loop.cond.1
loop.end.3:
  %t29 = load i64, ptr %a1
  %t30 = getelementptr inbounds %S.main.Page, ptr %p1, i32 0, i32 1
  %t31 = load i64, ptr %t30
  %t33 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t29, i64 %t31)
  %t34 = extractvalue { i64, i1 } %t33, 0
  %t35 = extractvalue { i64, i1 } %t33, 1
  br i1 %t35, label %overflow.11, label %arith.ok.12
overflow.11:
  %t36 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t37 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t38 = extractvalue %str { ptr @.str.4, i64 12 }, 0
  %t39 = extractvalue %str { ptr @.str.4, i64 12 }, 1
  call void @veles_panic_at(ptr %t36, i64 %t37, ptr %t38, i64 %t39)
  unreachable
arith.ok.12:
  ret i64 %t34
}

define void @v_main.main() {
entry:
  %a1 = alloca [4 x i64]
  %a7 = alloca [4 x i64]
  %a8 = alloca i64
  %a18 = alloca i64
  %a23 = alloca [21 x i8]
  %a28 = alloca i64
  %a33 = alloca { i1, i64 }
  %a34 = alloca i1
  %a51 = alloca [21 x i8]
  %a63 = alloca i64
  %a68 = alloca { i1, i32 }
  %a69 = alloca i1
  %a85 = alloca %str
  %a86 = alloca [7 x %str]
  %a97 = alloca %S.main.Page
  %a100 = alloca [256 x i8]
  %a103 = alloca %S.main.Page
  %a104 = alloca %S.main.Page
  %a105 = alloca i64
  %a116 = alloca %S.main.Page
  %a120 = alloca [21 x i8]
  %a124 = alloca %S.main.Page
  %a128 = alloca [21 x i8]
  %a132 = alloca %str
  %a133 = alloca [3 x %str]
  %t2 = getelementptr inbounds [4 x i64], ptr %a1, i64 0, i64 0
  store i64 1, ptr %t2
  %t3 = getelementptr inbounds [4 x i64], ptr %a1, i64 0, i64 1
  store i64 2, ptr %t3
  %t4 = getelementptr inbounds [4 x i64], ptr %a1, i64 0, i64 2
  store i64 3, ptr %t4
  %t5 = getelementptr inbounds [4 x i64], ptr %a1, i64 0, i64 3
  store i64 4, ptr %t5
  %t6 = load [4 x i64], ptr %a1
  store [4 x i64] %t6, ptr %a7
  store i64 1, ptr %a8
  %t9 = load i64, ptr %a8
  %t10 = icmp slt i64 %t9, 0
  br i1 %t10, label %if.then.1, label %if.end.2
if.then.1:
  %t11 = load i64, ptr %a8
  %t12 = add i64 %t11, 4
  store i64 %t12, ptr %a8
  br label %if.end.2
if.end.2:
  %t13 = load i64, ptr %a8
  %t14 = icmp ult i64 %t13, 4
  br i1 %t14, label %idx.ok.3, label %idx.bad.4, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.4:
  %t15 = extractvalue %str { ptr @.str.5, i64 12 }, 0
  %t16 = extractvalue %str { ptr @.str.5, i64 12 }, 1
  call void @veles_array_index_panic(i64 %t13, i64 4, ptr %t15, i64 %t16)
  unreachable
idx.ok.3:
  %t17 = getelementptr inbounds [4 x i64], ptr %a7, i64 0, i64 %t13
  store i64 20, ptr %t17
  store i64 7, ptr %a18
  %t19 = load [4 x i64], ptr %a7
  %t20 = extractvalue %str { ptr @.str.6, i64 17 }, 0
  %t21 = extractvalue %str { ptr @.str.6, i64 17 }, 1
  call void @veles_call_push(ptr %t20)
  %t22 = call i64 @v_main.sum__4([4 x i64] %t19)
  call void @veles_call_pop()
  %t24 = call i64 @veles_i64_format(ptr %a23, i64 %t22)
  %t25 = insertvalue %str undef, ptr %a23, 0
  %t26 = insertvalue %str %t25, i64 %t24, 1
  %t27 = load i64, ptr %a18
  store i64 %t27, ptr %a28
  %t29 = load i64, ptr %a28
  %t30 = icmp slt i64 %t29, 0
  br i1 %t30, label %if.then.5, label %if.end.6
if.then.5:
  %t31 = load i64, ptr %a28
  %t32 = add i64 %t31, 4
  store i64 %t32, ptr %a28
  br label %if.end.6
if.end.6:
  %t35 = load i64, ptr %a28
  %t36 = icmp sge i64 %t35, 0
  store i1 %t36, ptr %a34
  br i1 %t36, label %sc.rhs.7, label %sc.end.8
sc.rhs.7:
  %t37 = load i64, ptr %a28
  %t38 = icmp slt i64 %t37, 4
  store i1 %t38, ptr %a34
  br label %sc.end.8
sc.end.8:
  %t39 = load i1, ptr %a34
  br i1 %t39, label %if.then.9, label %if.else.11
if.then.9:
  %t40 = load i64, ptr %a28
  %t41 = icmp ult i64 %t40, 4
  br i1 %t41, label %idx.ok.12, label %idx.bad.13, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.13:
  %t42 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t43 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_array_index_panic(i64 %t40, i64 4, ptr %t42, i64 %t43)
  unreachable
idx.ok.12:
  %t44 = getelementptr inbounds [4 x i64], ptr %a7, i64 0, i64 %t40
  %t45 = load i64, ptr %t44
  %t46 = insertvalue { i1, i64 } undef, i1 true, 0
  %t47 = insertvalue { i1, i64 } %t46, i64 %t45, 1
  store { i1, i64 } %t47, ptr %a33
  br label %if.end.10
if.else.11:
  store { i1, i64 } zeroinitializer, ptr %a33
  br label %if.end.10
if.end.10:
  %t48 = load { i1, i64 }, ptr %a33
  %t49 = call %str @show.T_i64_N({ i1, i64 } %t48)
  %t50 = zext i32 30 to i64
  %t52 = call i64 @veles_u64_format(ptr %a51, i64 %t50)
  %t53 = insertvalue %str undef, ptr %a51, 0
  %t54 = insertvalue %str %t53, i64 %t52, 1
  %t56 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 4, i64 1)
  %t57 = extractvalue { i64, i1 } %t56, 0
  %t58 = extractvalue { i64, i1 } %t56, 1
  br i1 %t58, label %overflow.14, label %arith.ok.15
overflow.14:
  %t59 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t60 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t61 = extractvalue %str { ptr @.str.8, i64 13 }, 0
  %t62 = extractvalue %str { ptr @.str.8, i64 13 }, 1
  call void @veles_panic_at(ptr %t59, i64 %t60, ptr %t61, i64 %t62)
  unreachable
arith.ok.15:
  store i64 %t57, ptr %a63
  %t64 = load i64, ptr %a63
  %t65 = icmp slt i64 %t64, 0
  br i1 %t65, label %if.then.16, label %if.end.17
if.then.16:
  %t66 = load i64, ptr %a63
  %t67 = add i64 %t66, 4
  store i64 %t67, ptr %a63
  br label %if.end.17
if.end.17:
  %t70 = load i64, ptr %a63
  %t71 = icmp sge i64 %t70, 0
  store i1 %t71, ptr %a69
  br i1 %t71, label %sc.rhs.18, label %sc.end.19
sc.rhs.18:
  %t72 = load i64, ptr %a63
  %t73 = icmp slt i64 %t72, 4
  store i1 %t73, ptr %a69
  br label %sc.end.19
sc.end.19:
  %t74 = load i1, ptr %a69
  br i1 %t74, label %if.then.20, label %if.else.22
if.then.20:
  %t75 = load i64, ptr %a63
  %t76 = icmp ult i64 %t75, 4
  br i1 %t76, label %idx.ok.23, label %idx.bad.24, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.24:
  %t77 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t78 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_array_index_panic(i64 %t75, i64 4, ptr %t77, i64 %t78)
  unreachable
idx.ok.23:
  %t79 = getelementptr inbounds [4 x i32], ptr @const.main.K, i64 0, i64 %t75
  %t80 = load i32, ptr %t79
  %t81 = insertvalue { i1, i32 } undef, i1 true, 0
  %t82 = insertvalue { i1, i32 } %t81, i32 %t80, 1
  store { i1, i32 } %t82, ptr %a68
  br label %if.end.21
if.else.22:
  store { i1, i32 } zeroinitializer, ptr %a68
  br label %if.end.21
if.end.21:
  %t83 = load { i1, i32 }, ptr %a68
  %t84 = call %str @show.T_u32_N({ i1, i32 } %t83)
  %t87 = getelementptr [7 x %str], ptr %a86, i64 0, i64 0
  store %str %t26, ptr %t87
  %t88 = getelementptr [7 x %str], ptr %a86, i64 0, i64 1
  store %str { ptr @.str.10, i64 1 }, ptr %t88
  %t89 = getelementptr [7 x %str], ptr %a86, i64 0, i64 2
  store %str %t49, ptr %t89
  %t90 = getelementptr [7 x %str], ptr %a86, i64 0, i64 3
  store %str { ptr @.str.10, i64 1 }, ptr %t90
  %t91 = getelementptr [7 x %str], ptr %a86, i64 0, i64 4
  store %str %t54, ptr %t91
  %t92 = getelementptr [7 x %str], ptr %a86, i64 0, i64 5
  store %str { ptr @.str.10, i64 1 }, ptr %t92
  %t93 = getelementptr [7 x %str], ptr %a86, i64 0, i64 6
  store %str %t84, ptr %t93
  call void @veles_string_concat_n(ptr %a85, ptr %a86, i64 7)
  %t94 = load %str, ptr %a85
  %t95 = extractvalue %str { ptr @.str.11, i64 20 }, 0
  %t96 = extractvalue %str { ptr @.str.11, i64 20 }, 1
  call void @veles_call_push(ptr %t95)
  call void @v_std.io.println(%str %t94)
  call void @veles_call_pop()
  %t98 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t99 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t98)
  call void @v_main.filled(ptr noalias sret([256 x i8]) align 1 %a100, i8 1)
  call void @veles_call_pop()
  %t101 = getelementptr inbounds %S.main.Page, ptr %a97, i32 0, i32 0
  call void @llvm.memmove.p0.p0.i64(ptr %t101, ptr %a100, i64 256, i1 false)
  %t102 = getelementptr inbounds %S.main.Page, ptr %a97, i32 0, i32 1
  store i64 3, ptr %t102
  call void @llvm.memmove.p0.p0.i64(ptr %a103, ptr %a97, i64 264, i1 false)
  call void @llvm.memmove.p0.p0.i64(ptr %a104, ptr %a103, i64 264, i1 false)
  store i64 0, ptr %a105
  %t106 = load i64, ptr %a105
  %t107 = icmp slt i64 %t106, 0
  br i1 %t107, label %if.then.25, label %if.end.26
if.then.25:
  %t108 = load i64, ptr %a105
  %t109 = add i64 %t108, 256
  store i64 %t109, ptr %a105
  br label %if.end.26
if.end.26:
  %t110 = load i64, ptr %a105
  %t111 = getelementptr inbounds %S.main.Page, ptr %a103, i32 0, i32 0
  %t112 = icmp ult i64 %t110, 256
  br i1 %t112, label %idx.ok.27, label %idx.bad.28, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.28:
  %t113 = extractvalue %str { ptr @.str.13, i64 12 }, 0
  %t114 = extractvalue %str { ptr @.str.13, i64 12 }, 1
  call void @veles_array_index_panic(i64 %t110, i64 256, ptr %t113, i64 %t114)
  unreachable
idx.ok.27:
  %t115 = getelementptr inbounds [256 x i8], ptr %t111, i64 0, i64 %t110
  store i8 9, ptr %t115
  call void @llvm.memmove.p0.p0.i64(ptr %a116, ptr %a103, i64 264, i1 false)
  %t117 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t118 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t117)
  %t119 = call i64 @v_main.weight(ptr %a116)
  call void @veles_call_pop()
  %t121 = call i64 @veles_i64_format(ptr %a120, i64 %t119)
  %t122 = insertvalue %str undef, ptr %a120, 0
  %t123 = insertvalue %str %t122, i64 %t121, 1
  call void @llvm.memmove.p0.p0.i64(ptr %a124, ptr %a104, i64 264, i1 false)
  %t125 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t126 = extractvalue %str { ptr @.str.15, i64 20 }, 1
  call void @veles_call_push(ptr %t125)
  %t127 = call i64 @v_main.weight(ptr %a124)
  call void @veles_call_pop()
  %t129 = call i64 @veles_i64_format(ptr %a128, i64 %t127)
  %t130 = insertvalue %str undef, ptr %a128, 0
  %t131 = insertvalue %str %t130, i64 %t129, 1
  %t134 = getelementptr [3 x %str], ptr %a133, i64 0, i64 0
  store %str %t123, ptr %t134
  %t135 = getelementptr [3 x %str], ptr %a133, i64 0, i64 1
  store %str { ptr @.str.10, i64 1 }, ptr %t135
  %t136 = getelementptr [3 x %str], ptr %a133, i64 0, i64 2
  store %str %t131, ptr %t136
  call void @veles_string_concat_n(ptr %a132, ptr %a133, i64 3)
  %t137 = load %str, ptr %a132
  %t138 = extractvalue %str { ptr @.str.16, i64 20 }, 0
  %t139 = extractvalue %str { ptr @.str.16, i64 20 }, 1
  call void @veles_call_push(ptr %t138)
  call void @v_std.io.println(%str %t137)
  call void @veles_call_pop()
  ret void
}

define i64 @v_main.sum__4([4 x i64] %p1) {
entry:
  %a1 = alloca [4 x i64]
  %a2 = alloca i64
  %a6 = alloca { i64, i64, i1 }
  %a9 = alloca i64
  %a12 = alloca i64
  %a13 = alloca i1
  %a14 = alloca i1
  %a16 = alloca i1
  %a28 = alloca i64
  %a31 = alloca i64
  %a50 = alloca i1
  store [4 x i64] %p1, ptr %a1
  store i64 0, ptr %a2
  %t3 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t4 = insertvalue { i64, i64, i1 } %t3, i64 4, 1
  %t5 = insertvalue { i64, i64, i1 } %t4, i1 false, 2
  store { i64, i64, i1 } %t5, ptr %a6
  %t7 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 0
  %t8 = load i64, ptr %t7
  store i64 %t8, ptr %a9
  %t10 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 1
  %t11 = load i64, ptr %t10
  store i64 %t11, ptr %a12
  store i1 false, ptr %a13
  br label %loop.cond.1
loop.cond.1:
  %t15 = load i1, ptr %a13
  br i1 %t15, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a14
  br label %if.end.6
if.else.7:
  %t17 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t18 = load i1, ptr %t17
  br i1 %t18, label %if.then.8, label %if.else.10
if.then.8:
  %t19 = load i64, ptr %a9
  %t20 = load i64, ptr %a12
  %t21 = icmp sle i64 %t19, %t20
  store i1 %t21, ptr %a16
  br label %if.end.9
if.else.10:
  %t22 = load i64, ptr %a9
  %t23 = load i64, ptr %a12
  %t24 = icmp slt i64 %t22, %t23
  store i1 %t24, ptr %a16
  br label %if.end.9
if.end.9:
  %t25 = load i1, ptr %a16
  store i1 %t25, ptr %a14
  br label %if.end.6
if.end.6:
  %t26 = load i1, ptr %a14
  br i1 %t26, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t27 = load i64, ptr %a9
  store i64 %t27, ptr %a28
  %t29 = load i64, ptr %a2
  %t30 = load i64, ptr %a28
  store i64 %t30, ptr %a31
  %t32 = load i64, ptr %a31
  %t33 = icmp slt i64 %t32, 0
  br i1 %t33, label %if.then.11, label %if.end.12
if.then.11:
  %t34 = load i64, ptr %a31
  %t35 = add i64 %t34, 4
  store i64 %t35, ptr %a31
  br label %if.end.12
if.end.12:
  %t36 = load i64, ptr %a31
  %t37 = icmp ult i64 %t36, 4
  br i1 %t37, label %idx.ok.13, label %idx.bad.14, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.14:
  %t38 = extractvalue %str { ptr @.str.17, i64 13 }, 0
  %t39 = extractvalue %str { ptr @.str.17, i64 13 }, 1
  call void @veles_array_index_panic(i64 %t36, i64 4, ptr %t38, i64 %t39)
  unreachable
idx.ok.13:
  %t40 = getelementptr inbounds [4 x i64], ptr %a1, i64 0, i64 %t36
  %t41 = load i64, ptr %t40
  %t43 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t29, i64 %t41)
  %t44 = extractvalue { i64, i1 } %t43, 0
  %t45 = extractvalue { i64, i1 } %t43, 1
  br i1 %t45, label %overflow.15, label %arith.ok.16
overflow.15:
  %t46 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t47 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t48 = extractvalue %str { ptr @.str.18, i64 12 }, 0
  %t49 = extractvalue %str { ptr @.str.18, i64 12 }, 1
  call void @veles_panic_at(ptr %t46, i64 %t47, ptr %t48, i64 %t49)
  unreachable
arith.ok.16:
  store i64 %t44, ptr %a2
  br label %loop.post.2
loop.post.2:
  %t51 = getelementptr inbounds { i64, i64, i1 }, ptr %a6, i32 0, i32 2
  %t52 = load i1, ptr %t51
  store i1 %t52, ptr %a50
  br i1 %t52, label %sc.rhs.17, label %sc.end.18
sc.rhs.17:
  %t53 = load i64, ptr %a9
  %t54 = load i64, ptr %a12
  %t55 = icmp eq i64 %t53, %t54
  store i1 %t55, ptr %a50
  br label %sc.end.18
sc.end.18:
  %t56 = load i1, ptr %a50
  br i1 %t56, label %if.then.19, label %if.else.21
if.then.19:
  store i1 true, ptr %a13
  br label %if.end.20
if.else.21:
  %t57 = load i64, ptr %a9
  %t58 = add i64 %t57, 1
  store i64 %t58, ptr %a9
  br label %if.end.20
if.end.20:
  %t59 = load volatile i32, ptr @veles_stop_requested, align 4
  %t60 = icmp ne i32 %t59, 0
  br i1 %t60, label %safepoint.22, label %safepoint.on.23, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.22:
  call void @veles_gc_park()
  br label %safepoint.on.23
safepoint.on.23:
  br label %loop.cond.1
loop.end.3:
  %t61 = load i64, ptr %a2
  ret i64 %t61
}

@.str.1 = private unnamed_addr constant [13 x i8] c"main.vs:26:3\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:27:5\00"
@.str.4 = private unnamed_addr constant [13 x i8] c"main.vs:29:3\00"
@.str.5 = private unnamed_addr constant [13 x i8] c"main.vs:34:3\00"
@.str.6 = private unnamed_addr constant [18 x i8] c"main.vs:36:14\00sum\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:36:28\00"
@.str.8 = private unnamed_addr constant [14 x i8] c"main.vs:36:61\00"
@.str.9 = private unnamed_addr constant [14 x i8] c"main.vs:36:56\00"
@.str.10 = private unnamed_addr constant [2 x i8] c" \00"
@.str.11 = private unnamed_addr constant [21 x i8] c"main.vs:36:3\00println\00"
@.str.12 = private unnamed_addr constant [21 x i8] c"main.vs:37:26\00filled\00"
@.str.13 = private unnamed_addr constant [13 x i8] c"main.vs:39:3\00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:40:14\00weight\00"
@.str.15 = private unnamed_addr constant [21 x i8] c"main.vs:40:30\00weight\00"
@.str.16 = private unnamed_addr constant [21 x i8] c"main.vs:40:3\00println\00"
@.str.17 = private unnamed_addr constant [14 x i8] c"main.vs:17:10\00"
@.str.18 = private unnamed_addr constant [13 x i8] c"main.vs:17:5\00"
