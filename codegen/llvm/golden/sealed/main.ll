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
  %a35 = alloca [21 x i8]
  %a39 = alloca %str
  %a40 = alloca [1 x %str]
  %a48 = alloca i64
  %a53 = alloca i64
  %a57 = alloca { i1, i64 }
  %a58 = alloca { i1, %str }
  %a70 = alloca %str
  %a75 = alloca %str
  %a77 = alloca [21 x i8]
  %a82 = alloca %str
  %a83 = alloca [3 x %str]
  %a93 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a100 = alloca [21 x i8]
  %a104 = alloca %str
  %a119 = alloca [21 x i8]
  %a123 = alloca %str
  %a138 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a145 = alloca [21 x i8]
  %a149 = alloca %str
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
  %t32 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t33 = extractvalue %str { ptr @.str.5, i64 18 }, 1
  call void @veles_call_push(ptr %t32)
  %t34 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %a31)
  call void @veles_call_pop()
  %t36 = call i64 @veles_i64_format(ptr %a35, i64 %t34)
  %t37 = insertvalue %str undef, ptr %a35, 0
  %t38 = insertvalue %str %t37, i64 %t36, 1
  %t41 = getelementptr [1 x %str], ptr %a40, i64 0, i64 0
  store %str %t38, ptr %t41
  call void @veles_string_concat_n(ptr %a39, ptr %a40, i64 1)
  %t42 = load %str, ptr %a39
  %t43 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t44 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t43)
  call void @v_std.io.println(%str %t42)
  call void @veles_call_pop()
  %t45 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t46 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t45)
  %t47 = call { i1, i64 } @v_main.half(i64 10)
  call void @veles_call_pop()
  %t50 = extractvalue { i1, i64 } %t47, 0
  %t49 = xor i1 %t50, true
  br i1 %t49, label %elvis.default.1, label %elvis.some.2
elvis.some.2:
  %t51 = extractvalue { i1, i64 } %t47, 1
  store i64 %t51, ptr %a48
  br label %elvis.end.3
elvis.default.1:
  store i64 -1, ptr %a48
  br label %elvis.end.3
elvis.end.3:
  %t52 = load i64, ptr %a48
  store i64 %t52, ptr %a53
  %t54 = extractvalue %str { ptr @.str.8, i64 18 }, 0
  %t55 = extractvalue %str { ptr @.str.8, i64 18 }, 1
  call void @veles_call_push(ptr %t54)
  %t56 = call { i1, i64 } @v_main.half(i64 7)
  call void @veles_call_pop()
  store { i1, i64 } %t56, ptr %a57
  %t59 = load { i1, i64 }, ptr %a57
  %t61 = extractvalue { i1, i64 } %t59, 0
  %t60 = xor i1 %t61, true
  %t62 = xor i1 %t60, true
  br i1 %t62, label %if.then.4, label %if.else.6
if.then.4:
  %t63 = getelementptr inbounds { i1, i64 }, ptr %a57, i32 0, i32 1
  %t64 = extractvalue %str { ptr @.str.9, i64 22 }, 0
  %t65 = extractvalue %str { ptr @.str.9, i64 22 }, 1
  call void @veles_call_push(ptr %t64)
  %t66 = call %str @v_std.prelude.extend.i64.toString(ptr %t63, i64 10)
  call void @veles_call_pop()
  %t67 = insertvalue { i1, %str } undef, i1 true, 0
  %t68 = insertvalue { i1, %str } %t67, %str %t66, 1
  store { i1, %str } %t68, ptr %a58
  br label %if.end.5
if.else.6:
  store { i1, %str } zeroinitializer, ptr %a58
  br label %if.end.5
if.end.5:
  %t69 = load { i1, %str }, ptr %a58
  %t72 = extractvalue { i1, %str } %t69, 0
  %t71 = xor i1 %t72, true
  br i1 %t71, label %elvis.default.7, label %elvis.some.8
elvis.some.8:
  %t73 = extractvalue { i1, %str } %t69, 1
  store %str %t73, ptr %a70
  br label %elvis.end.9
elvis.default.7:
  store %str { ptr @.str.10, i64 3 }, ptr %a70
  br label %elvis.end.9
elvis.end.9:
  %t74 = load %str, ptr %a70
  store %str %t74, ptr %a75
  %t76 = load i64, ptr %a53
  %t78 = call i64 @veles_i64_format(ptr %a77, i64 %t76)
  %t79 = insertvalue %str undef, ptr %a77, 0
  %t80 = insertvalue %str %t79, i64 %t78, 1
  %t81 = load %str, ptr %a75
  %t84 = getelementptr [3 x %str], ptr %a83, i64 0, i64 0
  store %str %t80, ptr %t84
  %t85 = getelementptr [3 x %str], ptr %a83, i64 0, i64 1
  store %str { ptr @.str.11, i64 1 }, ptr %t85
  %t86 = getelementptr [3 x %str], ptr %a83, i64 0, i64 2
  store %str %t81, ptr %t86
  call void @veles_string_concat_n(ptr %a82, ptr %a83, i64 3)
  %t87 = load %str, ptr %a82
  %t88 = extractvalue %str { ptr @.str.12, i64 20 }, 0
  %t89 = extractvalue %str { ptr @.str.12, i64 20 }, 1
  call void @veles_call_push(ptr %t88)
  call void @v_std.io.println(%str %t87)
  call void @veles_call_pop()
  %t90 = extractvalue %str { ptr @.str.13, i64 19 }, 0
  %t91 = extractvalue %str { ptr @.str.13, i64 19 }, 1
  call void @veles_call_push(ptr %t90)
  %t92 = call %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 500)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.TooBig_ %t92, ptr %a93
  br label %when.arm.11
when.arm.11:
  %t94 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a93
  %t95 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t94, 0
  %t96 = icmp eq i32 %t95, 0
  br i1 %t96, label %when.bind.14, label %when.arm.12
when.bind.14:
  br label %when.body.13
when.body.13:
  %t97 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a93, i32 0, i32 1
  %t98 = getelementptr inbounds %S._prelude_.Ok_i64_main.TooBig_, ptr %t97, i32 0, i32 0
  %t99 = load i64, ptr %t98
  %t101 = call i64 @veles_i64_format(ptr %a100, i64 %t99)
  %t102 = insertvalue %str undef, ptr %a100, 0
  %t103 = insertvalue %str %t102, i64 %t101, 1
  %t105 = extractvalue %str { ptr @.str.14, i64 3 }, 0
  %t106 = extractvalue %str { ptr @.str.14, i64 3 }, 1
  %t107 = extractvalue %str %t103, 0
  %t108 = extractvalue %str %t103, 1
  call void @veles_string_concat(ptr %a104, ptr %t105, i64 %t106, ptr %t107, i64 %t108)
  %t109 = load %str, ptr %a104
  %t110 = extractvalue %str { ptr @.str.15, i64 21 }, 0
  %t111 = extractvalue %str { ptr @.str.15, i64 21 }, 1
  call void @veles_call_push(ptr %t110)
  call void @v_std.io.println(%str %t109)
  call void @veles_call_pop()
  br label %when.end.10
when.arm.12:
  %t112 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a93
  %t113 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t112, 0
  %t114 = icmp eq i32 %t113, 1
  br i1 %t114, label %when.bind.17, label %when.arm.15
when.bind.17:
  br label %when.body.16
when.body.16:
  %t115 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a93, i32 0, i32 1
  %t116 = getelementptr inbounds %S._prelude_.Err_i64_main.TooBig_, ptr %t115, i32 0, i32 0
  %t117 = getelementptr inbounds %S.main.TooBig, ptr %t116, i32 0, i32 0
  %t118 = load i64, ptr %t117
  %t120 = call i64 @veles_i64_format(ptr %a119, i64 %t118)
  %t121 = insertvalue %str undef, ptr %a119, 0
  %t122 = insertvalue %str %t121, i64 %t120, 1
  %t124 = extractvalue %str { ptr @.str.16, i64 8 }, 0
  %t125 = extractvalue %str { ptr @.str.16, i64 8 }, 1
  %t126 = extractvalue %str %t122, 0
  %t127 = extractvalue %str %t122, 1
  call void @veles_string_concat(ptr %a123, ptr %t124, i64 %t125, ptr %t126, i64 %t127)
  %t128 = load %str, ptr %a123
  %t129 = extractvalue %str { ptr @.str.17, i64 21 }, 0
  %t130 = extractvalue %str { ptr @.str.17, i64 21 }, 1
  call void @veles_call_push(ptr %t129)
  call void @v_std.io.println(%str %t128)
  call void @veles_call_pop()
  br label %when.end.10
when.arm.15:
  %t131 = extractvalue %str { ptr @.str.18, i64 33 }, 0
  %t132 = extractvalue %str { ptr @.str.18, i64 33 }, 1
  %t133 = extractvalue %str { ptr @.str.19, i64 12 }, 0
  %t134 = extractvalue %str { ptr @.str.19, i64 12 }, 1
  call void @veles_panic_at(ptr %t131, i64 %t132, ptr %t133, i64 %t134)
  unreachable
when.end.10:
  %t135 = extractvalue %str { ptr @.str.20, i64 19 }, 0
  %t136 = extractvalue %str { ptr @.str.20, i64 19 }, 1
  call void @veles_call_push(ptr %t135)
  %t137 = call %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 5)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.TooBig_ %t137, ptr %a138
  %t139 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a138
  %t140 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t139, 0
  %t141 = icmp eq i32 %t140, 0
  br i1 %t141, label %if.then.18, label %if.end.19
if.then.18:
  %t142 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a138, i32 0, i32 1
  %t143 = getelementptr inbounds %S._prelude_.Ok_i64_main.TooBig_, ptr %t142, i32 0, i32 0
  %t144 = load i64, ptr %t143
  %t146 = call i64 @veles_i64_format(ptr %a145, i64 %t144)
  %t147 = insertvalue %str undef, ptr %a145, 0
  %t148 = insertvalue %str %t147, i64 %t146, 1
  %t150 = extractvalue %str { ptr @.str.21, i64 5 }, 0
  %t151 = extractvalue %str { ptr @.str.21, i64 5 }, 1
  %t152 = extractvalue %str %t148, 0
  %t153 = extractvalue %str %t148, 1
  call void @veles_string_concat(ptr %a149, ptr %t150, i64 %t151, ptr %t152, i64 %t153)
  %t154 = load %str, ptr %a149
  %t155 = extractvalue %str { ptr @.str.22, i64 21 }, 0
  %t156 = extractvalue %str { ptr @.str.22, i64 21 }, 1
  call void @veles_call_push(ptr %t155)
  call void @v_std.io.println(%str %t154)
  call void @veles_call_pop()
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
  %a47 = alloca ptr
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
  %t26 = extractvalue %str { ptr @.str.23, i64 17 }, 0
  %t27 = extractvalue %str { ptr @.str.23, i64 17 }, 1
  call void @veles_call_push(ptr %t26)
  %t28 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %t25)
  call void @veles_call_pop()
  %t29 = load ptr, ptr %a24
  %t30 = extractvalue %str { ptr @.str.24, i64 17 }, 0
  %t31 = extractvalue %str { ptr @.str.24, i64 17 }, 1
  call void @veles_call_push(ptr %t30)
  %t32 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %t29)
  call void @veles_call_pop()
  %t34 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t28, i64 %t32)
  %t35 = extractvalue { i64, i1 } %t34, 0
  %t36 = extractvalue { i64, i1 } %t34, 1
  br i1 %t36, label %overflow.9, label %arith.ok.10
overflow.9:
  %t37 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t38 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t39 = extractvalue %str { ptr @.str.25, i64 12 }, 0
  %t40 = extractvalue %str { ptr @.str.25, i64 12 }, 1
  call void @veles_panic_at(ptr %t37, i64 %t38, ptr %t39, i64 %t40)
  unreachable
arith.ok.10:
  store i64 %t35, ptr %a2
  br label %when.end.1
when.arm.6:
  %t41 = load %V.main.Expr, ptr %a5
  %t42 = extractvalue %V.main.Expr %t41, 0
  %t43 = icmp eq i32 %t42, 2
  br i1 %t43, label %when.bind.13, label %when.arm.11
when.bind.13:
  %t44 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t45 = getelementptr inbounds %S.main.Neg, ptr %t44, i32 0, i32 0
  %t46 = load ptr, ptr %t45
  store ptr %t46, ptr %a47
  br label %when.body.12
when.body.12:
  %t48 = load ptr, ptr %a47
  %t49 = extractvalue %str { ptr @.str.26, i64 17 }, 0
  %t50 = extractvalue %str { ptr @.str.26, i64 17 }, 1
  call void @veles_call_push(ptr %t49)
  %t51 = call i64 @v_main.Expr.eval_Self_main.Expr_(ptr %t48)
  call void @veles_call_pop()
  %t53 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 0, i64 %t51)
  %t54 = extractvalue { i64, i1 } %t53, 0
  %t55 = extractvalue { i64, i1 } %t53, 1
  br i1 %t55, label %overflow.14, label %arith.ok.15
overflow.14:
  %t56 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t57 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t58 = extractvalue %str { ptr @.str.27, i64 12 }, 0
  %t59 = extractvalue %str { ptr @.str.27, i64 12 }, 1
  call void @veles_panic_at(ptr %t56, i64 %t57, ptr %t58, i64 %t59)
  unreachable
arith.ok.15:
  store i64 %t54, ptr %a2
  br label %when.end.1
when.arm.11:
  %t60 = extractvalue %str { ptr @.str.18, i64 33 }, 0
  %t61 = extractvalue %str { ptr @.str.18, i64 33 }, 1
  %t62 = extractvalue %str { ptr @.str.28, i64 12 }, 0
  %t63 = extractvalue %str { ptr @.str.28, i64 12 }, 1
  call void @veles_panic_at(ptr %t60, i64 %t61, ptr %t62, i64 %t63)
  unreachable
when.end.1:
  %t64 = load i64, ptr %a2
  ret i64 %t64
}

@.str.1 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:20:30\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:20:42\00"
@.str.5 = private unnamed_addr constant [19 x i8] c"main.vs:27:17\00eval\00"
@.str.6 = private unnamed_addr constant [21 x i8] c"main.vs:27:3\00println\00"
@.str.7 = private unnamed_addr constant [19 x i8] c"main.vs:28:11\00half\00"
@.str.8 = private unnamed_addr constant [19 x i8] c"main.vs:29:11\00half\00"
@.str.9 = private unnamed_addr constant [23 x i8] c"main.vs:29:11\00toString\00"
@.str.10 = private unnamed_addr constant [4 x i8] c"odd\00"
@.str.11 = private unnamed_addr constant [2 x i8] c" \00"
@.str.12 = private unnamed_addr constant [21 x i8] c"main.vs:30:3\00println\00"
@.str.13 = private unnamed_addr constant [20 x i8] c"main.vs:31:17\00check\00"
@.str.14 = private unnamed_addr constant [4 x i8] c"ok \00"
@.str.15 = private unnamed_addr constant [22 x i8] c"main.vs:32:14\00println\00"
@.str.16 = private unnamed_addr constant [9 x i8] c"too big \00"
@.str.17 = private unnamed_addr constant [22 x i8] c"main.vs:33:15\00println\00"
@.str.18 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.19 = private unnamed_addr constant [13 x i8] c"main.vs:31:3\00"
@.str.20 = private unnamed_addr constant [20 x i8] c"main.vs:35:14\00check\00"
@.str.21 = private unnamed_addr constant [6 x i8] c"fine \00"
@.str.22 = private unnamed_addr constant [22 x i8] c"main.vs:36:16\00println\00"
@.str.23 = private unnamed_addr constant [18 x i8] c"main.vs:8:21\00eval\00"
@.str.24 = private unnamed_addr constant [18 x i8] c"main.vs:8:32\00eval\00"
@.str.25 = private unnamed_addr constant [13 x i8] c"main.vs:8:21\00"
@.str.26 = private unnamed_addr constant [18 x i8] c"main.vs:9:19\00eval\00"
@.str.27 = private unnamed_addr constant [13 x i8] c"main.vs:9:18\00"
@.str.28 = private unnamed_addr constant [13 x i8] c"main.vs:6:21\00"
