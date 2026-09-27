%S.main.TooBig = type { i64 }
%S._prelude_.Err_i64_main.TooBig_ = type { %S.main.TooBig }
%V._prelude_.Result_i64_main.TooBig_ = type { i32, [1 x i64] }
%S._prelude_.Ok_i64_main.TooBig_ = type { i64 }
%S.main.Num = type { i64 }
%V.main.Expr = type { i32, [2 x i64] }
%S.main.Add = type { ptr, ptr }
%S.main.Neg = type { ptr }
@desc.main.Expr = internal constant { i64, i64, i64, [2 x i64] } { i64 24, i64 0, i64 2, [2 x i64] [i64 8, i64 16] }
define %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  %a8 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a15 = alloca %V._prelude_.Result_i64_main.TooBig_
  store i64 %p1, ptr %a1
  %t3 = load i64, ptr %a1
  %t4 = icmp sgt i64 %t3, 100
  br i1 %t4, label %if.then.1, label %if.else.3
if.then.1:
  %t5 = load i64, ptr %a1
  %t6 = insertvalue %S.main.TooBig undef, i64 %t5, 0
  %t7 = insertvalue %S._prelude_.Err_i64_main.TooBig_ undef, %S.main.TooBig %t6, 0
  store %V._prelude_.Result_i64_main.TooBig_ zeroinitializer, ptr %a8
  %t9 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a8, i32 0, i32 0
  store i32 1, ptr %t9
  %t10 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a8, i32 0, i32 1
  store %S._prelude_.Err_i64_main.TooBig_ %t7, ptr %t10
  %t11 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a8
  ret %V._prelude_.Result_i64_main.TooBig_ %t11
if.else.3:
  %t12 = load i64, ptr %a1
  store i64 %t12, ptr %a2
  br label %if.end.2
if.end.2:
  %t13 = load i64, ptr %a2
  %t14 = insertvalue %S._prelude_.Ok_i64_main.TooBig_ undef, i64 %t13, 0
  store %V._prelude_.Result_i64_main.TooBig_ zeroinitializer, ptr %a15
  %t16 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a15, i32 0, i32 0
  store i32 0, ptr %t16
  %t17 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a15, i32 0, i32 1
  store %S._prelude_.Ok_i64_main.TooBig_ %t14, ptr %t17
  %t18 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a15
  ret %V._prelude_.Result_i64_main.TooBig_ %t18
}

define { i1, i64 } @v_main.half(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca { i1, i64 }
  store i64 %p1, ptr %a1
  %t3 = load i64, ptr %a1
  %t5 = icmp eq i64 2, 0
  br i1 %t5, label %divzero.1, label %div.ok.2
divzero.1:
  %t6 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t7 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t8 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t9 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t6, i64 %t7, ptr %t8, i64 %t9)
  unreachable
div.ok.2:
  %t10 = icmp eq i64 %t3, -9223372036854775808
  %t11 = icmp eq i64 2, -1
  %t12 = and i1 %t10, %t11
  br i1 %t12, label %divof.3, label %div.ok.4
divof.3:
  %t13 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t14 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t15 = extractvalue %str { ptr @.str.2, i64 13 }, 0
  %t16 = extractvalue %str { ptr @.str.2, i64 13 }, 1
  call void @veles_panic_at(ptr %t13, i64 %t14, ptr %t15, i64 %t16)
  unreachable
div.ok.4:
  %t4 = srem i64 %t3, 2
  %t17 = icmp eq i64 %t4, 0
  br i1 %t17, label %if.then.5, label %if.else.7
if.then.5:
  %t18 = load i64, ptr %a1
  %t20 = icmp eq i64 2, 0
  br i1 %t20, label %divzero.8, label %div.ok.9
divzero.8:
  %t21 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t22 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t23 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t24 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t21, i64 %t22, ptr %t23, i64 %t24)
  unreachable
div.ok.9:
  %t25 = icmp eq i64 %t18, -9223372036854775808
  %t26 = icmp eq i64 2, -1
  %t27 = and i1 %t25, %t26
  br i1 %t27, label %divof.10, label %div.ok.11
divof.10:
  %t28 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t29 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t30 = extractvalue %str { ptr @.str.4, i64 13 }, 0
  %t31 = extractvalue %str { ptr @.str.4, i64 13 }, 1
  call void @veles_panic_at(ptr %t28, i64 %t29, ptr %t30, i64 %t31)
  unreachable
div.ok.11:
  %t19 = sdiv i64 %t18, 2
  %t32 = insertvalue { i1, i64 } undef, i1 true, 0
  %t33 = insertvalue { i1, i64 } %t32, i64 %t19, 1
  store { i1, i64 } %t33, ptr %a2
  br label %if.end.6
if.else.7:
  store { i1, i64 } zeroinitializer, ptr %a2
  br label %if.end.6
if.end.6:
  %t34 = load { i1, i64 }, ptr %a2
  ret { i1, i64 } %t34
}

define void @v_main.main() {
entry:
  %a2 = alloca %V.main.Expr
  %a6 = alloca ptr
  %a9 = alloca %V.main.Expr
  %a13 = alloca ptr
  %a19 = alloca %V.main.Expr
  %a23 = alloca ptr
  %a27 = alloca %V.main.Expr
  %a31 = alloca %V.main.Expr
  %a33 = alloca [21 x i8]
  %a37 = alloca %str
  %a38 = alloca [1 x %str]
  %a42 = alloca i64
  %a47 = alloca i64
  %a49 = alloca { i1, i64 }
  %a50 = alloca { i1, %str }
  %a60 = alloca %str
  %a65 = alloca %str
  %a67 = alloca [21 x i8]
  %a72 = alloca %str
  %a73 = alloca [3 x %str]
  %a79 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a86 = alloca [21 x i8]
  %a90 = alloca %str
  %a103 = alloca [21 x i8]
  %a107 = alloca %str
  %a118 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a125 = alloca [21 x i8]
  %a129 = alloca %str
  %t1 = insertvalue %S.main.Num undef, i64 1, 0
  store %V.main.Expr zeroinitializer, ptr %a2
  %t3 = getelementptr inbounds %V.main.Expr, ptr %a2, i32 0, i32 0
  store i32 0, ptr %t3
  %t4 = getelementptr inbounds %V.main.Expr, ptr %a2, i32 0, i32 1
  store %S.main.Num %t1, ptr %t4
  %t5 = load %V.main.Expr, ptr %a2
  %t7 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 24)
  store ptr %t7, ptr %a6
  store %V.main.Expr %t5, ptr %t7
  %t8 = insertvalue %S.main.Num undef, i64 2, 0
  store %V.main.Expr zeroinitializer, ptr %a9
  %t10 = getelementptr inbounds %V.main.Expr, ptr %a9, i32 0, i32 0
  store i32 0, ptr %t10
  %t11 = getelementptr inbounds %V.main.Expr, ptr %a9, i32 0, i32 1
  store %S.main.Num %t8, ptr %t11
  %t12 = load %V.main.Expr, ptr %a9
  %t14 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 24)
  store ptr %t14, ptr %a13
  store %V.main.Expr %t12, ptr %t14
  %t15 = load ptr, ptr %a6
  %t16 = load ptr, ptr %a13
  %t17 = insertvalue %S.main.Add undef, ptr %t15, 0
  %t18 = insertvalue %S.main.Add %t17, ptr %t16, 1
  store %V.main.Expr zeroinitializer, ptr %a19
  %t20 = getelementptr inbounds %V.main.Expr, ptr %a19, i32 0, i32 0
  store i32 1, ptr %t20
  %t21 = getelementptr inbounds %V.main.Expr, ptr %a19, i32 0, i32 1
  store %S.main.Add %t18, ptr %t21
  %t22 = load %V.main.Expr, ptr %a19
  %t24 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 24)
  store ptr %t24, ptr %a23
  store %V.main.Expr %t22, ptr %t24
  %t25 = load ptr, ptr %a23
  %t26 = insertvalue %S.main.Neg undef, ptr %t25, 0
  store %V.main.Expr zeroinitializer, ptr %a27
  %t28 = getelementptr inbounds %V.main.Expr, ptr %a27, i32 0, i32 0
  store i32 2, ptr %t28
  %t29 = getelementptr inbounds %V.main.Expr, ptr %a27, i32 0, i32 1
  store %S.main.Neg %t26, ptr %t29
  %t30 = load %V.main.Expr, ptr %a27
  store %V.main.Expr %t30, ptr %a31
  %t32 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %a31)
  %t34 = call i64 @veles_i64_format(ptr %a33, i64 %t32)
  %t35 = insertvalue %str undef, ptr %a33, 0
  %t36 = insertvalue %str %t35, i64 %t34, 1
  %t39 = getelementptr [1 x %str], ptr %a38, i64 0, i64 0
  store %str %t36, ptr %t39
  call void @veles_string_concat_n(ptr %a37, ptr %a38, i64 1)
  %t40 = load %str, ptr %a37
  call void @v_std.io.println(%str %t40)
  %t41 = call { i1, i64 } @v_main.half(i64 10)
  %t44 = extractvalue { i1, i64 } %t41, 0
  %t43 = xor i1 %t44, true
  br i1 %t43, label %elvis.default.1, label %elvis.some.2
elvis.some.2:
  %t45 = extractvalue { i1, i64 } %t41, 1
  store i64 %t45, ptr %a42
  br label %elvis.end.3
elvis.default.1:
  store i64 -1, ptr %a42
  br label %elvis.end.3
elvis.end.3:
  %t46 = load i64, ptr %a42
  store i64 %t46, ptr %a47
  %t48 = call { i1, i64 } @v_main.half(i64 7)
  store { i1, i64 } %t48, ptr %a49
  %t51 = load { i1, i64 }, ptr %a49
  %t53 = extractvalue { i1, i64 } %t51, 0
  %t52 = xor i1 %t53, true
  %t54 = xor i1 %t52, true
  br i1 %t54, label %if.then.4, label %if.else.6
if.then.4:
  %t55 = getelementptr inbounds { i1, i64 }, ptr %a49, i32 0, i32 1
  %t56 = call %str @v_std.prelude.extend.i64.toString(ptr %t55, i64 10)
  %t57 = insertvalue { i1, %str } undef, i1 true, 0
  %t58 = insertvalue { i1, %str } %t57, %str %t56, 1
  store { i1, %str } %t58, ptr %a50
  br label %if.end.5
if.else.6:
  store { i1, %str } zeroinitializer, ptr %a50
  br label %if.end.5
if.end.5:
  %t59 = load { i1, %str }, ptr %a50
  %t62 = extractvalue { i1, %str } %t59, 0
  %t61 = xor i1 %t62, true
  br i1 %t61, label %elvis.default.7, label %elvis.some.8
elvis.some.8:
  %t63 = extractvalue { i1, %str } %t59, 1
  store %str %t63, ptr %a60
  br label %elvis.end.9
elvis.default.7:
  store %str { ptr @.str.5, i64 3 }, ptr %a60
  br label %elvis.end.9
elvis.end.9:
  %t64 = load %str, ptr %a60
  store %str %t64, ptr %a65
  %t66 = load i64, ptr %a47
  %t68 = call i64 @veles_i64_format(ptr %a67, i64 %t66)
  %t69 = insertvalue %str undef, ptr %a67, 0
  %t70 = insertvalue %str %t69, i64 %t68, 1
  %t71 = load %str, ptr %a65
  %t74 = getelementptr [3 x %str], ptr %a73, i64 0, i64 0
  store %str %t70, ptr %t74
  %t75 = getelementptr [3 x %str], ptr %a73, i64 0, i64 1
  store %str { ptr @.str.6, i64 1 }, ptr %t75
  %t76 = getelementptr [3 x %str], ptr %a73, i64 0, i64 2
  store %str %t71, ptr %t76
  call void @veles_string_concat_n(ptr %a72, ptr %a73, i64 3)
  %t77 = load %str, ptr %a72
  call void @v_std.io.println(%str %t77)
  %t78 = call %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 500)
  store %V._prelude_.Result_i64_main.TooBig_ %t78, ptr %a79
  br label %when.arm.11
when.arm.11:
  %t80 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a79
  %t81 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t80, 0
  %t82 = icmp eq i32 %t81, 0
  br i1 %t82, label %when.bind.14, label %when.arm.12
when.bind.14:
  br label %when.body.13
when.body.13:
  %t83 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a79, i32 0, i32 1
  %t84 = getelementptr inbounds %S._prelude_.Ok_i64_main.TooBig_, ptr %t83, i32 0, i32 0
  %t85 = load i64, ptr %t84
  %t87 = call i64 @veles_i64_format(ptr %a86, i64 %t85)
  %t88 = insertvalue %str undef, ptr %a86, 0
  %t89 = insertvalue %str %t88, i64 %t87, 1
  %t91 = extractvalue %str { ptr @.str.7, i64 3 }, 0
  %t92 = extractvalue %str { ptr @.str.7, i64 3 }, 1
  %t93 = extractvalue %str %t89, 0
  %t94 = extractvalue %str %t89, 1
  call void @veles_string_concat(ptr %a90, ptr %t91, i64 %t92, ptr %t93, i64 %t94)
  %t95 = load %str, ptr %a90
  call void @v_std.io.println(%str %t95)
  br label %when.end.10
when.arm.12:
  %t96 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a79
  %t97 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t96, 0
  %t98 = icmp eq i32 %t97, 1
  br i1 %t98, label %when.bind.17, label %when.arm.15
when.bind.17:
  br label %when.body.16
when.body.16:
  %t99 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a79, i32 0, i32 1
  %t100 = getelementptr inbounds %S._prelude_.Err_i64_main.TooBig_, ptr %t99, i32 0, i32 0
  %t101 = getelementptr inbounds %S.main.TooBig, ptr %t100, i32 0, i32 0
  %t102 = load i64, ptr %t101
  %t104 = call i64 @veles_i64_format(ptr %a103, i64 %t102)
  %t105 = insertvalue %str undef, ptr %a103, 0
  %t106 = insertvalue %str %t105, i64 %t104, 1
  %t108 = extractvalue %str { ptr @.str.8, i64 8 }, 0
  %t109 = extractvalue %str { ptr @.str.8, i64 8 }, 1
  %t110 = extractvalue %str %t106, 0
  %t111 = extractvalue %str %t106, 1
  call void @veles_string_concat(ptr %a107, ptr %t108, i64 %t109, ptr %t110, i64 %t111)
  %t112 = load %str, ptr %a107
  call void @v_std.io.println(%str %t112)
  br label %when.end.10
when.arm.15:
  %t113 = extractvalue %str { ptr @.str.9, i64 33 }, 0
  %t114 = extractvalue %str { ptr @.str.9, i64 33 }, 1
  %t115 = extractvalue %str { ptr @.str.10, i64 12 }, 0
  %t116 = extractvalue %str { ptr @.str.10, i64 12 }, 1
  call void @veles_panic_at(ptr %t113, i64 %t114, ptr %t115, i64 %t116)
  unreachable
when.end.10:
  %t117 = call %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 5)
  store %V._prelude_.Result_i64_main.TooBig_ %t117, ptr %a118
  %t119 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a118
  %t120 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t119, 0
  %t121 = icmp eq i32 %t120, 0
  br i1 %t121, label %if.then.18, label %if.end.19
if.then.18:
  %t122 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a118, i32 0, i32 1
  %t123 = getelementptr inbounds %S._prelude_.Ok_i64_main.TooBig_, ptr %t122, i32 0, i32 0
  %t124 = load i64, ptr %t123
  %t126 = call i64 @veles_i64_format(ptr %a125, i64 %t124)
  %t127 = insertvalue %str undef, ptr %a125, 0
  %t128 = insertvalue %str %t127, i64 %t126, 1
  %t130 = extractvalue %str { ptr @.str.11, i64 5 }, 0
  %t131 = extractvalue %str { ptr @.str.11, i64 5 }, 1
  %t132 = extractvalue %str %t128, 0
  %t133 = extractvalue %str %t128, 1
  call void @veles_string_concat(ptr %a129, ptr %t130, i64 %t131, ptr %t132, i64 %t133)
  %t134 = load %str, ptr %a129
  call void @v_std.io.println(%str %t134)
  br label %if.end.19
if.end.19:
  ret void
}

define i64 @v_main.Expr.eval_Self_main.Expr_(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a5 = alloca %V.main.Expr
  %a12 = alloca i64
  %a20 = alloca ptr
  %a24 = alloca ptr
  %a43 = alloca ptr
  store ptr %p0, ptr %a1
  %t3 = load ptr, ptr %a1
  %t4 = load %V.main.Expr, ptr %t3
  store %V.main.Expr %t4, ptr %a5
  br label %when.arm.2
when.arm.2:
  %t6 = load %V.main.Expr, ptr %a5
  %t7 = extractvalue %V.main.Expr %t6, 0
  %t8 = icmp eq i32 %t7, 0
  br i1 %t8, label %when.bind.5, label %when.arm.3
when.bind.5:
  %t9 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t10 = getelementptr inbounds %S.main.Num, ptr %t9, i32 0, i32 0
  %t11 = load i64, ptr %t10
  store i64 %t11, ptr %a12
  br label %when.body.4
when.body.4:
  %t13 = load i64, ptr %a12
  store i64 %t13, ptr %a2
  br label %when.end.1
when.arm.3:
  %t14 = load %V.main.Expr, ptr %a5
  %t15 = extractvalue %V.main.Expr %t14, 0
  %t16 = icmp eq i32 %t15, 1
  br i1 %t16, label %when.bind.8, label %when.arm.6
when.bind.8:
  %t17 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t18 = getelementptr inbounds %S.main.Add, ptr %t17, i32 0, i32 0
  %t19 = load ptr, ptr %t18
  store ptr %t19, ptr %a20
  %t21 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t22 = getelementptr inbounds %S.main.Add, ptr %t21, i32 0, i32 1
  %t23 = load ptr, ptr %t22
  store ptr %t23, ptr %a24
  br label %when.body.7
when.body.7:
  %t25 = load ptr, ptr %a20
  %t26 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %t25)
  %t27 = load ptr, ptr %a24
  %t28 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %t27)
  %t30 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t26, i64 %t28)
  %t31 = extractvalue { i64, i1 } %t30, 0
  %t32 = extractvalue { i64, i1 } %t30, 1
  br i1 %t32, label %overflow.9, label %arith.ok.10
overflow.9:
  %t33 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t34 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t35 = extractvalue %str { ptr @.str.12, i64 12 }, 0
  %t36 = extractvalue %str { ptr @.str.12, i64 12 }, 1
  call void @veles_panic_at(ptr %t33, i64 %t34, ptr %t35, i64 %t36)
  unreachable
arith.ok.10:
  store i64 %t31, ptr %a2
  br label %when.end.1
when.arm.6:
  %t37 = load %V.main.Expr, ptr %a5
  %t38 = extractvalue %V.main.Expr %t37, 0
  %t39 = icmp eq i32 %t38, 2
  br i1 %t39, label %when.bind.13, label %when.arm.11
when.bind.13:
  %t40 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t41 = getelementptr inbounds %S.main.Neg, ptr %t40, i32 0, i32 0
  %t42 = load ptr, ptr %t41
  store ptr %t42, ptr %a43
  br label %when.body.12
when.body.12:
  %t44 = load ptr, ptr %a43
  %t45 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %t44)
  %t47 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 0, i64 %t45)
  %t48 = extractvalue { i64, i1 } %t47, 0
  %t49 = extractvalue { i64, i1 } %t47, 1
  br i1 %t49, label %overflow.14, label %arith.ok.15
overflow.14:
  %t50 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t51 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t52 = extractvalue %str { ptr @.str.13, i64 12 }, 0
  %t53 = extractvalue %str { ptr @.str.13, i64 12 }, 1
  call void @veles_panic_at(ptr %t50, i64 %t51, ptr %t52, i64 %t53)
  unreachable
arith.ok.15:
  store i64 %t48, ptr %a2
  br label %when.end.1
when.arm.11:
  %t54 = extractvalue %str { ptr @.str.9, i64 33 }, 0
  %t55 = extractvalue %str { ptr @.str.9, i64 33 }, 1
  %t56 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t57 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_panic_at(ptr %t54, i64 %t55, ptr %t56, i64 %t57)
  unreachable
when.end.1:
  %t58 = load i64, ptr %a2
  ret i64 %t58
}

@.str.1 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:20:30\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:20:42\00"
@.str.5 = private unnamed_addr constant [4 x i8] c"odd\00"
@.str.6 = private unnamed_addr constant [2 x i8] c" \00"
@.str.7 = private unnamed_addr constant [4 x i8] c"ok \00"
@.str.8 = private unnamed_addr constant [9 x i8] c"too big \00"
@.str.9 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.10 = private unnamed_addr constant [13 x i8] c"main.vs:31:3\00"
@.str.11 = private unnamed_addr constant [6 x i8] c"fine \00"
@.str.12 = private unnamed_addr constant [13 x i8] c"main.vs:8:21\00"
@.str.13 = private unnamed_addr constant [13 x i8] c"main.vs:9:18\00"
@.str.14 = private unnamed_addr constant [13 x i8] c"main.vs:6:21\00"
