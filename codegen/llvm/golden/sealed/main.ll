%S.main.TooBig = type { i64 }
%S._prelude_.Err_i64_main.TooBig_ = type { %S.main.TooBig }
%V._prelude_.Result_i64_main.TooBig_ = type { i32, [1 x i64] }
%S._prelude_.Ok_i64_main.TooBig_ = type { i64 }
%S.main.Num = type { i64 }
%V.main.Expr = type { i32, [1 x i64] }
%S.main.Add = type { ptr, ptr }
%S.main.Neg = type { ptr }
@desc.main.Expr = internal constant { i64, i64, i64, [1 x i64] } { i64 16, i64 0, i64 1, [1 x i64] [i64 8] }
@desc.main.Add = internal constant { i64, i64, i64, [2 x i64] } { i64 16, i64 0, i64 2, [2 x i64] [i64 0, i64 8] }
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
  %a20 = alloca %V.main.Expr
  %a24 = alloca ptr
  %a28 = alloca %V.main.Expr
  %a32 = alloca %V.main.Expr
  %a36 = alloca [21 x i8]
  %a40 = alloca %str
  %a41 = alloca [1 x %str]
  %a49 = alloca i64
  %a54 = alloca i64
  %a58 = alloca { i1, i64 }
  %a59 = alloca { i1, %str }
  %a71 = alloca %str
  %a76 = alloca %str
  %a78 = alloca [21 x i8]
  %a83 = alloca %str
  %a84 = alloca [3 x %str]
  %a94 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a101 = alloca [21 x i8]
  %a105 = alloca %str
  %a120 = alloca [21 x i8]
  %a124 = alloca %str
  %a139 = alloca %V._prelude_.Result_i64_main.TooBig_
  %a146 = alloca [21 x i8]
  %a150 = alloca %str
  %t1 = insertvalue %S.main.Num undef, i64 1, 0
  store %V.main.Expr zeroinitializer, ptr %a2
  %t3 = getelementptr inbounds %V.main.Expr, ptr %a2, i32 0, i32 0
  store i32 0, ptr %t3
  %t4 = getelementptr inbounds %V.main.Expr, ptr %a2, i32 0, i32 1
  store %S.main.Num %t1, ptr %t4
  %t5 = load %V.main.Expr, ptr %a2
  %t7 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 16)
  store ptr %t7, ptr %a6
  store %V.main.Expr %t5, ptr %t7
  %t8 = insertvalue %S.main.Num undef, i64 2, 0
  store %V.main.Expr zeroinitializer, ptr %a9
  %t10 = getelementptr inbounds %V.main.Expr, ptr %a9, i32 0, i32 0
  store i32 0, ptr %t10
  %t11 = getelementptr inbounds %V.main.Expr, ptr %a9, i32 0, i32 1
  store %S.main.Num %t8, ptr %t11
  %t12 = load %V.main.Expr, ptr %a9
  %t14 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 16)
  store ptr %t14, ptr %a13
  store %V.main.Expr %t12, ptr %t14
  %t15 = load ptr, ptr %a6
  %t16 = load ptr, ptr %a13
  %t17 = insertvalue %S.main.Add undef, ptr %t15, 0
  %t18 = insertvalue %S.main.Add %t17, ptr %t16, 1
  %t19 = call ptr @veles_gc_alloc(ptr @desc.main.Add, i64 16)
  store %S.main.Add %t18, ptr %t19
  store %V.main.Expr zeroinitializer, ptr %a20
  %t21 = getelementptr inbounds %V.main.Expr, ptr %a20, i32 0, i32 0
  store i32 1, ptr %t21
  %t22 = getelementptr inbounds %V.main.Expr, ptr %a20, i32 0, i32 1
  store ptr %t19, ptr %t22
  %t23 = load %V.main.Expr, ptr %a20
  %t25 = call ptr @veles_gc_alloc(ptr @desc.main.Expr, i64 16)
  store ptr %t25, ptr %a24
  store %V.main.Expr %t23, ptr %t25
  %t26 = load ptr, ptr %a24
  %t27 = insertvalue %S.main.Neg undef, ptr %t26, 0
  store %V.main.Expr zeroinitializer, ptr %a28
  %t29 = getelementptr inbounds %V.main.Expr, ptr %a28, i32 0, i32 0
  store i32 2, ptr %t29
  %t30 = getelementptr inbounds %V.main.Expr, ptr %a28, i32 0, i32 1
  store %S.main.Neg %t27, ptr %t30
  %t31 = load %V.main.Expr, ptr %a28
  store %V.main.Expr %t31, ptr %a32
  %t33 = extractvalue %str { ptr @.str.5, i64 18 }, 0
  %t34 = extractvalue %str { ptr @.str.5, i64 18 }, 1
  call void @veles_call_push(ptr %t33)
  %t35 = call i64 @v_main.Expr.eval_This_main.Expr_(ptr %a32)
  call void @veles_call_pop()
  %t37 = call i64 @veles_i64_format(ptr %a36, i64 %t35)
  %t38 = insertvalue %str undef, ptr %a36, 0
  %t39 = insertvalue %str %t38, i64 %t37, 1
  %t42 = getelementptr [1 x %str], ptr %a41, i64 0, i64 0
  store %str %t39, ptr %t42
  call void @veles_string_concat_n(ptr %a40, ptr %a41, i64 1)
  %t43 = load %str, ptr %a40
  %t44 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t45 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t44)
  call void @v_std.io.println(%str %t43)
  call void @veles_call_pop()
  %t46 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t47 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t46)
  %t48 = call { i1, i64 } @v_main.half(i64 10)
  call void @veles_call_pop()
  %t51 = extractvalue { i1, i64 } %t48, 0
  %t50 = xor i1 %t51, true
  br i1 %t50, label %elvis.default.1, label %elvis.some.2
elvis.some.2:
  %t52 = extractvalue { i1, i64 } %t48, 1
  store i64 %t52, ptr %a49
  br label %elvis.end.3
elvis.default.1:
  store i64 -1, ptr %a49
  br label %elvis.end.3
elvis.end.3:
  %t53 = load i64, ptr %a49
  store i64 %t53, ptr %a54
  %t55 = extractvalue %str { ptr @.str.8, i64 18 }, 0
  %t56 = extractvalue %str { ptr @.str.8, i64 18 }, 1
  call void @veles_call_push(ptr %t55)
  %t57 = call { i1, i64 } @v_main.half(i64 7)
  call void @veles_call_pop()
  store { i1, i64 } %t57, ptr %a58
  %t60 = load { i1, i64 }, ptr %a58
  %t62 = extractvalue { i1, i64 } %t60, 0
  %t61 = xor i1 %t62, true
  %t63 = xor i1 %t61, true
  br i1 %t63, label %if.then.4, label %if.else.6
if.then.4:
  %t64 = getelementptr inbounds { i1, i64 }, ptr %a58, i32 0, i32 1
  %t65 = extractvalue %str { ptr @.str.9, i64 22 }, 0
  %t66 = extractvalue %str { ptr @.str.9, i64 22 }, 1
  call void @veles_call_push(ptr %t65)
  %t67 = call %str @v_std.prelude.extend.i64.toString(ptr %t64, i64 10, %str { ptr @.str.10, i64 13 })
  call void @veles_call_pop()
  %t68 = insertvalue { i1, %str } undef, i1 true, 0
  %t69 = insertvalue { i1, %str } %t68, %str %t67, 1
  store { i1, %str } %t69, ptr %a59
  br label %if.end.5
if.else.6:
  store { i1, %str } zeroinitializer, ptr %a59
  br label %if.end.5
if.end.5:
  %t70 = load { i1, %str }, ptr %a59
  %t73 = extractvalue { i1, %str } %t70, 0
  %t72 = xor i1 %t73, true
  br i1 %t72, label %elvis.default.7, label %elvis.some.8
elvis.some.8:
  %t74 = extractvalue { i1, %str } %t70, 1
  store %str %t74, ptr %a71
  br label %elvis.end.9
elvis.default.7:
  store %str { ptr @.str.11, i64 3 }, ptr %a71
  br label %elvis.end.9
elvis.end.9:
  %t75 = load %str, ptr %a71
  store %str %t75, ptr %a76
  %t77 = load i64, ptr %a54
  %t79 = call i64 @veles_i64_format(ptr %a78, i64 %t77)
  %t80 = insertvalue %str undef, ptr %a78, 0
  %t81 = insertvalue %str %t80, i64 %t79, 1
  %t82 = load %str, ptr %a76
  %t85 = getelementptr [3 x %str], ptr %a84, i64 0, i64 0
  store %str %t81, ptr %t85
  %t86 = getelementptr [3 x %str], ptr %a84, i64 0, i64 1
  store %str { ptr @.str.12, i64 1 }, ptr %t86
  %t87 = getelementptr [3 x %str], ptr %a84, i64 0, i64 2
  store %str %t82, ptr %t87
  call void @veles_string_concat_n(ptr %a83, ptr %a84, i64 3)
  %t88 = load %str, ptr %a83
  %t89 = extractvalue %str { ptr @.str.13, i64 20 }, 0
  %t90 = extractvalue %str { ptr @.str.13, i64 20 }, 1
  call void @veles_call_push(ptr %t89)
  call void @v_std.io.println(%str %t88)
  call void @veles_call_pop()
  %t91 = extractvalue %str { ptr @.str.14, i64 19 }, 0
  %t92 = extractvalue %str { ptr @.str.14, i64 19 }, 1
  call void @veles_call_push(ptr %t91)
  %t93 = call %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 500)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.TooBig_ %t93, ptr %a94
  br label %when.arm.11
when.arm.11:
  %t95 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a94
  %t96 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t95, 0
  %t97 = icmp eq i32 %t96, 0
  br i1 %t97, label %when.bind.14, label %when.arm.12
when.bind.14:
  br label %when.body.13
when.body.13:
  %t98 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a94, i32 0, i32 1
  %t99 = getelementptr inbounds %S._prelude_.Ok_i64_main.TooBig_, ptr %t98, i32 0, i32 0
  %t100 = load i64, ptr %t99
  %t102 = call i64 @veles_i64_format(ptr %a101, i64 %t100)
  %t103 = insertvalue %str undef, ptr %a101, 0
  %t104 = insertvalue %str %t103, i64 %t102, 1
  %t106 = extractvalue %str { ptr @.str.15, i64 3 }, 0
  %t107 = extractvalue %str { ptr @.str.15, i64 3 }, 1
  %t108 = extractvalue %str %t104, 0
  %t109 = extractvalue %str %t104, 1
  call void @veles_string_concat(ptr %a105, ptr %t106, i64 %t107, ptr %t108, i64 %t109)
  %t110 = load %str, ptr %a105
  %t111 = extractvalue %str { ptr @.str.16, i64 21 }, 0
  %t112 = extractvalue %str { ptr @.str.16, i64 21 }, 1
  call void @veles_call_push(ptr %t111)
  call void @v_std.io.println(%str %t110)
  call void @veles_call_pop()
  br label %when.end.10
when.arm.12:
  %t113 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a94
  %t114 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t113, 0
  %t115 = icmp eq i32 %t114, 1
  br i1 %t115, label %when.bind.17, label %when.arm.15
when.bind.17:
  br label %when.body.16
when.body.16:
  %t116 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a94, i32 0, i32 1
  %t117 = getelementptr inbounds %S._prelude_.Err_i64_main.TooBig_, ptr %t116, i32 0, i32 0
  %t118 = getelementptr inbounds %S.main.TooBig, ptr %t117, i32 0, i32 0
  %t119 = load i64, ptr %t118
  %t121 = call i64 @veles_i64_format(ptr %a120, i64 %t119)
  %t122 = insertvalue %str undef, ptr %a120, 0
  %t123 = insertvalue %str %t122, i64 %t121, 1
  %t125 = extractvalue %str { ptr @.str.17, i64 8 }, 0
  %t126 = extractvalue %str { ptr @.str.17, i64 8 }, 1
  %t127 = extractvalue %str %t123, 0
  %t128 = extractvalue %str %t123, 1
  call void @veles_string_concat(ptr %a124, ptr %t125, i64 %t126, ptr %t127, i64 %t128)
  %t129 = load %str, ptr %a124
  %t130 = extractvalue %str { ptr @.str.18, i64 21 }, 0
  %t131 = extractvalue %str { ptr @.str.18, i64 21 }, 1
  call void @veles_call_push(ptr %t130)
  call void @v_std.io.println(%str %t129)
  call void @veles_call_pop()
  br label %when.end.10
when.arm.15:
  %t132 = extractvalue %str { ptr @.str.19, i64 33 }, 0
  %t133 = extractvalue %str { ptr @.str.19, i64 33 }, 1
  %t134 = extractvalue %str { ptr @.str.20, i64 12 }, 0
  %t135 = extractvalue %str { ptr @.str.20, i64 12 }, 1
  call void @veles_panic_at(ptr %t132, i64 %t133, ptr %t134, i64 %t135)
  unreachable
when.end.10:
  %t136 = extractvalue %str { ptr @.str.21, i64 19 }, 0
  %t137 = extractvalue %str { ptr @.str.21, i64 19 }, 1
  call void @veles_call_push(ptr %t136)
  %t138 = call %V._prelude_.Result_i64_main.TooBig_ @v_main.check(i64 5)
  call void @veles_call_pop()
  store %V._prelude_.Result_i64_main.TooBig_ %t138, ptr %a139
  %t140 = load %V._prelude_.Result_i64_main.TooBig_, ptr %a139
  %t141 = extractvalue %V._prelude_.Result_i64_main.TooBig_ %t140, 0
  %t142 = icmp eq i32 %t141, 0
  br i1 %t142, label %if.then.18, label %if.end.19
if.then.18:
  %t143 = getelementptr inbounds %V._prelude_.Result_i64_main.TooBig_, ptr %a139, i32 0, i32 1
  %t144 = getelementptr inbounds %S._prelude_.Ok_i64_main.TooBig_, ptr %t143, i32 0, i32 0
  %t145 = load i64, ptr %t144
  %t147 = call i64 @veles_i64_format(ptr %a146, i64 %t145)
  %t148 = insertvalue %str undef, ptr %a146, 0
  %t149 = insertvalue %str %t148, i64 %t147, 1
  %t151 = extractvalue %str { ptr @.str.22, i64 5 }, 0
  %t152 = extractvalue %str { ptr @.str.22, i64 5 }, 1
  %t153 = extractvalue %str %t149, 0
  %t154 = extractvalue %str %t149, 1
  call void @veles_string_concat(ptr %a150, ptr %t151, i64 %t152, ptr %t153, i64 %t154)
  %t155 = load %str, ptr %a150
  %t156 = extractvalue %str { ptr @.str.23, i64 21 }, 0
  %t157 = extractvalue %str { ptr @.str.23, i64 21 }, 1
  call void @veles_call_push(ptr %t156)
  call void @v_std.io.println(%str %t155)
  call void @veles_call_pop()
  br label %if.end.19
if.end.19:
  ret void
}

define i64 @v_main.Expr.eval_This_main.Expr_(ptr %p0) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a5 = alloca %V.main.Expr
  %a12 = alloca i64
  %a21 = alloca ptr
  %a26 = alloca ptr
  %a49 = alloca ptr
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
  %t18 = load ptr, ptr %t17
  %t19 = getelementptr inbounds %S.main.Add, ptr %t18, i32 0, i32 0
  %t20 = load ptr, ptr %t19
  store ptr %t20, ptr %a21
  %t22 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t23 = load ptr, ptr %t22
  %t24 = getelementptr inbounds %S.main.Add, ptr %t23, i32 0, i32 1
  %t25 = load ptr, ptr %t24
  store ptr %t25, ptr %a26
  br label %when.body.7
when.body.7:
  %t27 = load ptr, ptr %a21
  %t28 = extractvalue %str { ptr @.str.24, i64 17 }, 0
  %t29 = extractvalue %str { ptr @.str.24, i64 17 }, 1
  call void @veles_call_push(ptr %t28)
  %t30 = call i64 @v_main.Expr.eval_This_main.Expr_(ptr %t27)
  call void @veles_call_pop()
  %t31 = load ptr, ptr %a26
  %t32 = extractvalue %str { ptr @.str.25, i64 17 }, 0
  %t33 = extractvalue %str { ptr @.str.25, i64 17 }, 1
  call void @veles_call_push(ptr %t32)
  %t34 = call i64 @v_main.Expr.eval_This_main.Expr_(ptr %t31)
  call void @veles_call_pop()
  %t36 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t30, i64 %t34)
  %t37 = extractvalue { i64, i1 } %t36, 0
  %t38 = extractvalue { i64, i1 } %t36, 1
  br i1 %t38, label %overflow.9, label %arith.ok.10
overflow.9:
  %t39 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t40 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t41 = extractvalue %str { ptr @.str.26, i64 12 }, 0
  %t42 = extractvalue %str { ptr @.str.26, i64 12 }, 1
  call void @veles_panic_at(ptr %t39, i64 %t40, ptr %t41, i64 %t42)
  unreachable
arith.ok.10:
  store i64 %t37, ptr %a2
  br label %when.end.1
when.arm.6:
  %t43 = load %V.main.Expr, ptr %a5
  %t44 = extractvalue %V.main.Expr %t43, 0
  %t45 = icmp eq i32 %t44, 2
  br i1 %t45, label %when.bind.13, label %when.arm.11
when.bind.13:
  %t46 = getelementptr inbounds %V.main.Expr, ptr %a5, i32 0, i32 1
  %t47 = getelementptr inbounds %S.main.Neg, ptr %t46, i32 0, i32 0
  %t48 = load ptr, ptr %t47
  store ptr %t48, ptr %a49
  br label %when.body.12
when.body.12:
  %t50 = load ptr, ptr %a49
  %t51 = extractvalue %str { ptr @.str.27, i64 18 }, 0
  %t52 = extractvalue %str { ptr @.str.27, i64 18 }, 1
  call void @veles_call_push(ptr %t51)
  %t53 = call i64 @v_main.Expr.eval_This_main.Expr_(ptr %t50)
  call void @veles_call_pop()
  %t55 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 0, i64 %t53)
  %t56 = extractvalue { i64, i1 } %t55, 0
  %t57 = extractvalue { i64, i1 } %t55, 1
  br i1 %t57, label %overflow.14, label %arith.ok.15
overflow.14:
  %t58 = extractvalue %str { ptr @.str.3, i64 16 }, 0
  %t59 = extractvalue %str { ptr @.str.3, i64 16 }, 1
  %t60 = extractvalue %str { ptr @.str.28, i64 13 }, 0
  %t61 = extractvalue %str { ptr @.str.28, i64 13 }, 1
  call void @veles_panic_at(ptr %t58, i64 %t59, ptr %t60, i64 %t61)
  unreachable
arith.ok.15:
  store i64 %t56, ptr %a2
  br label %when.end.1
when.arm.11:
  %t62 = extractvalue %str { ptr @.str.19, i64 33 }, 0
  %t63 = extractvalue %str { ptr @.str.19, i64 33 }, 1
  %t64 = extractvalue %str { ptr @.str.29, i64 12 }, 0
  %t65 = extractvalue %str { ptr @.str.29, i64 12 }, 1
  call void @veles_panic_at(ptr %t62, i64 %t63, ptr %t64, i64 %t65)
  unreachable
when.end.1:
  %t66 = load i64, ptr %a2
  ret i64 %t66
}

@.str.1 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.2 = private unnamed_addr constant [14 x i8] c"main.vs:21:31\00"
@.str.3 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.4 = private unnamed_addr constant [14 x i8] c"main.vs:21:43\00"
@.str.5 = private unnamed_addr constant [19 x i8] c"main.vs:28:17\00eval\00"
@.str.6 = private unnamed_addr constant [21 x i8] c"main.vs:28:3\00println\00"
@.str.7 = private unnamed_addr constant [19 x i8] c"main.vs:29:11\00half\00"
@.str.8 = private unnamed_addr constant [19 x i8] c"main.vs:30:11\00half\00"
@.str.9 = private unnamed_addr constant [23 x i8] c"main.vs:30:11\00toString\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:30:11\00"
@.str.11 = private unnamed_addr constant [4 x i8] c"odd\00"
@.str.12 = private unnamed_addr constant [2 x i8] c" \00"
@.str.13 = private unnamed_addr constant [21 x i8] c"main.vs:31:3\00println\00"
@.str.14 = private unnamed_addr constant [20 x i8] c"main.vs:32:17\00check\00"
@.str.15 = private unnamed_addr constant [4 x i8] c"ok \00"
@.str.16 = private unnamed_addr constant [22 x i8] c"main.vs:33:14\00println\00"
@.str.17 = private unnamed_addr constant [9 x i8] c"too big \00"
@.str.18 = private unnamed_addr constant [22 x i8] c"main.vs:34:15\00println\00"
@.str.19 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.20 = private unnamed_addr constant [13 x i8] c"main.vs:32:3\00"
@.str.21 = private unnamed_addr constant [20 x i8] c"main.vs:36:14\00check\00"
@.str.22 = private unnamed_addr constant [6 x i8] c"fine \00"
@.str.23 = private unnamed_addr constant [22 x i8] c"main.vs:37:16\00println\00"
@.str.24 = private unnamed_addr constant [18 x i8] c"main.vs:9:21\00eval\00"
@.str.25 = private unnamed_addr constant [18 x i8] c"main.vs:9:32\00eval\00"
@.str.26 = private unnamed_addr constant [13 x i8] c"main.vs:9:21\00"
@.str.27 = private unnamed_addr constant [19 x i8] c"main.vs:10:19\00eval\00"
@.str.28 = private unnamed_addr constant [14 x i8] c"main.vs:10:18\00"
@.str.29 = private unnamed_addr constant [13 x i8] c"main.vs:7:22\00"
