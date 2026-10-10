define %str @v_main.classify(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca %str
  store i64 %p1, ptr %a1
  br label %when.arm.2
when.arm.2:
  %t3 = load i64, ptr %a1
  %t4 = icmp slt i64 %t3, 0
  br i1 %t4, label %when.bind.5, label %when.arm.3
when.bind.5:
  br label %when.body.4
when.body.4:
  store %str { ptr @.str.1, i64 8 }, ptr %a2
  br label %when.end.1
when.arm.3:
  %t5 = load i64, ptr %a1
  %t6 = icmp eq i64 %t5, 0
  br i1 %t6, label %when.bind.8, label %when.arm.6
when.bind.8:
  br label %when.body.7
when.body.7:
  store %str { ptr @.str.2, i64 4 }, ptr %a2
  br label %when.end.1
when.arm.6:
  %t7 = load i64, ptr %a1
  %t8 = icmp slt i64 %t7, 10
  br i1 %t8, label %when.bind.11, label %when.arm.9
when.bind.11:
  br label %when.body.10
when.body.10:
  store %str { ptr @.str.3, i64 5 }, ptr %a2
  br label %when.end.1
when.arm.9:
  br label %when.body.13
when.body.13:
  store %str { ptr @.str.4, i64 5 }, ptr %a2
  br label %when.end.1
when.arm.12:
  %t9 = extractvalue %str { ptr @.str.5, i64 33 }, 0
  %t10 = extractvalue %str { ptr @.str.5, i64 33 }, 1
  %t11 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t12 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
when.end.1:
  %t13 = load %str, ptr %a2
  ret %str %t13
}

define %str @v_main.digits(i64 %p1) {
entry:
  %a1 = alloca i64
  %a2 = alloca %str
  %a4 = alloca i64
  %a7 = alloca i1
  store i64 %p1, ptr %a1
  %t3 = load i64, ptr %a1
  store i64 %t3, ptr %a4
  br label %when.arm.2
when.arm.2:
  %t5 = load i64, ptr %a4
  %t6 = icmp eq i64 %t5, 1
  br i1 %t6, label %when.bind.5, label %when.arm.3
when.bind.5:
  br label %when.body.4
when.body.4:
  store %str { ptr @.str.7, i64 3 }, ptr %a2
  br label %when.end.1
when.arm.3:
  %t8 = load i64, ptr %a4
  %t9 = icmp eq i64 %t8, 2
  store i1 %t9, ptr %a7
  br i1 %t9, label %sc.end.9, label %sc.rhs.8
sc.rhs.8:
  %t10 = load i64, ptr %a4
  %t11 = icmp eq i64 %t10, 3
  store i1 %t11, ptr %a7
  br label %sc.end.9
sc.end.9:
  %t12 = load i1, ptr %a7
  br i1 %t12, label %when.bind.10, label %when.arm.6
when.bind.10:
  br label %when.body.7
when.body.7:
  store %str { ptr @.str.8, i64 3 }, ptr %a2
  br label %when.end.1
when.arm.6:
  br label %when.body.12
when.body.12:
  store %str { ptr @.str.9, i64 4 }, ptr %a2
  br label %when.end.1
when.arm.11:
  %t13 = extractvalue %str { ptr @.str.5, i64 33 }, 0
  %t14 = extractvalue %str { ptr @.str.5, i64 33 }, 1
  %t15 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t16 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_panic_at(ptr %t13, i64 %t14, ptr %t15, i64 %t16)
  unreachable
when.end.1:
  %t17 = load %str, ptr %a2
  ret %str %t17
}

define void @v_main.main() {
entry:
  %a1 = alloca i64
  %a5 = alloca { i64, i64, i1 }
  %a8 = alloca i64
  %a11 = alloca i64
  %a12 = alloca i1
  %a13 = alloca i1
  %a15 = alloca i1
  %a27 = alloca i64
  %a32 = alloca { i64, i64, i1 }
  %a35 = alloca i64
  %a38 = alloca i64
  %a39 = alloca i1
  %a40 = alloca i1
  %a42 = alloca i1
  %a54 = alloca i64
  %a87 = alloca i1
  %a98 = alloca i1
  %a109 = alloca i64
  %a123 = alloca i64
  %a140 = alloca { i64, i64, i1 }
  %a144 = alloca %S.std.prelude.RangeStepIter_i64_
  %a148 = alloca { i1, i64 }
  %a154 = alloca i64
  %a156 = alloca [21 x i8]
  %a160 = alloca %str
  %a171 = alloca [21 x i8]
  %a176 = alloca [21 x i8]
  %a181 = alloca [21 x i8]
  %a200 = alloca %str
  %a201 = alloca [15 x %str]
  store i64 0, ptr %a1
  %t2 = insertvalue { i64, i64, i1 } undef, i64 1, 0
  %t3 = insertvalue { i64, i64, i1 } %t2, i64 5, 1
  %t4 = insertvalue { i64, i64, i1 } %t3, i1 true, 2
  store { i64, i64, i1 } %t4, ptr %a5
  %t6 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 0
  %t7 = load i64, ptr %t6
  store i64 %t7, ptr %a8
  %t9 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 1
  %t10 = load i64, ptr %t9
  store i64 %t10, ptr %a11
  store i1 false, ptr %a12
  br label %loop.cond.1
loop.cond.1:
  %t14 = load i1, ptr %a12
  br i1 %t14, label %if.then.5, label %if.else.7
if.then.5:
  store i1 false, ptr %a13
  br label %if.end.6
if.else.7:
  %t16 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t17 = load i1, ptr %t16
  br i1 %t17, label %if.then.8, label %if.else.10
if.then.8:
  %t18 = load i64, ptr %a8
  %t19 = load i64, ptr %a11
  %t20 = icmp sle i64 %t18, %t19
  store i1 %t20, ptr %a15
  br label %if.end.9
if.else.10:
  %t21 = load i64, ptr %a8
  %t22 = load i64, ptr %a11
  %t23 = icmp slt i64 %t21, %t22
  store i1 %t23, ptr %a15
  br label %if.end.9
if.end.9:
  %t24 = load i1, ptr %a15
  store i1 %t24, ptr %a13
  br label %if.end.6
if.end.6:
  %t25 = load i1, ptr %a13
  br i1 %t25, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t26 = load i64, ptr %a8
  store i64 %t26, ptr %a27
  %t28 = load i64, ptr %a27
  %t29 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t30 = insertvalue { i64, i64, i1 } %t29, i64 %t28, 1
  %t31 = insertvalue { i64, i64, i1 } %t30, i1 false, 2
  store { i64, i64, i1 } %t31, ptr %a32
  %t33 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 0
  %t34 = load i64, ptr %t33
  store i64 %t34, ptr %a35
  %t36 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 1
  %t37 = load i64, ptr %t36
  store i64 %t37, ptr %a38
  store i1 false, ptr %a39
  br label %loop.cond.11
loop.cond.11:
  %t41 = load i1, ptr %a39
  br i1 %t41, label %if.then.15, label %if.else.17
if.then.15:
  store i1 false, ptr %a40
  br label %if.end.16
if.else.17:
  %t43 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 2
  %t44 = load i1, ptr %t43
  br i1 %t44, label %if.then.18, label %if.else.20
if.then.18:
  %t45 = load i64, ptr %a35
  %t46 = load i64, ptr %a38
  %t47 = icmp sle i64 %t45, %t46
  store i1 %t47, ptr %a42
  br label %if.end.19
if.else.20:
  %t48 = load i64, ptr %a35
  %t49 = load i64, ptr %a38
  %t50 = icmp slt i64 %t48, %t49
  store i1 %t50, ptr %a42
  br label %if.end.19
if.end.19:
  %t51 = load i1, ptr %a42
  store i1 %t51, ptr %a40
  br label %if.end.16
if.end.16:
  %t52 = load i1, ptr %a40
  br i1 %t52, label %loop.body.14, label %loop.end.13
loop.body.14:
  %t53 = load i64, ptr %a35
  store i64 %t53, ptr %a54
  %t55 = load i64, ptr %a54
  %t56 = icmp eq i64 %t55, 3
  br i1 %t56, label %if.then.21, label %if.end.22
if.then.21:
  br label %loop.post.2
if.end.22:
  %t57 = load i64, ptr %a27
  %t58 = load i64, ptr %a54
  %t60 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t57, i64 %t58)
  %t61 = extractvalue { i64, i1 } %t60, 0
  %t62 = extractvalue { i64, i1 } %t60, 1
  br i1 %t62, label %overflow.23, label %arith.ok.24
overflow.23:
  %t63 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t64 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t65 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t66 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_panic_at(ptr %t63, i64 %t64, ptr %t65, i64 %t66)
  unreachable
arith.ok.24:
  %t67 = icmp sgt i64 %t61, 8
  br i1 %t67, label %if.then.25, label %if.end.26
if.then.25:
  br label %loop.end.3
if.end.26:
  %t68 = load i64, ptr %a1
  %t69 = load i64, ptr %a27
  %t70 = load i64, ptr %a54
  %t72 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t69, i64 %t70)
  %t73 = extractvalue { i64, i1 } %t72, 0
  %t74 = extractvalue { i64, i1 } %t72, 1
  br i1 %t74, label %overflow.27, label %arith.ok.28
overflow.27:
  %t75 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t76 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t77 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t78 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t75, i64 %t76, ptr %t77, i64 %t78)
  unreachable
arith.ok.28:
  %t80 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t68, i64 %t73)
  %t81 = extractvalue { i64, i1 } %t80, 0
  %t82 = extractvalue { i64, i1 } %t80, 1
  br i1 %t82, label %overflow.29, label %arith.ok.30
overflow.29:
  %t83 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t84 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t85 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t86 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_panic_at(ptr %t83, i64 %t84, ptr %t85, i64 %t86)
  unreachable
arith.ok.30:
  store i64 %t81, ptr %a1
  br label %loop.post.12
loop.post.12:
  %t88 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 2
  %t89 = load i1, ptr %t88
  store i1 %t89, ptr %a87
  br i1 %t89, label %sc.rhs.31, label %sc.end.32
sc.rhs.31:
  %t90 = load i64, ptr %a35
  %t91 = load i64, ptr %a38
  %t92 = icmp eq i64 %t90, %t91
  store i1 %t92, ptr %a87
  br label %sc.end.32
sc.end.32:
  %t93 = load i1, ptr %a87
  br i1 %t93, label %if.then.33, label %if.else.35
if.then.33:
  store i1 true, ptr %a39
  br label %if.end.34
if.else.35:
  %t94 = load i64, ptr %a35
  %t95 = add i64 %t94, 1
  store i64 %t95, ptr %a35
  br label %if.end.34
if.end.34:
  %t96 = load volatile i32, ptr @veles_attention_line, align 64
  %t97 = icmp ne i32 %t96, 0
  br i1 %t97, label %safepoint.36, label %safepoint.on.37, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.36:
  call void @veles_backedge_plain()
  br label %safepoint.on.37
safepoint.on.37:
  br label %loop.cond.11
loop.end.13:
  br label %loop.post.2
loop.post.2:
  %t99 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t100 = load i1, ptr %t99
  store i1 %t100, ptr %a98
  br i1 %t100, label %sc.rhs.38, label %sc.end.39
sc.rhs.38:
  %t101 = load i64, ptr %a8
  %t102 = load i64, ptr %a11
  %t103 = icmp eq i64 %t101, %t102
  store i1 %t103, ptr %a98
  br label %sc.end.39
sc.end.39:
  %t104 = load i1, ptr %a98
  br i1 %t104, label %if.then.40, label %if.else.42
if.then.40:
  store i1 true, ptr %a12
  br label %if.end.41
if.else.42:
  %t105 = load i64, ptr %a8
  %t106 = add i64 %t105, 1
  store i64 %t106, ptr %a8
  br label %if.end.41
if.end.41:
  %t107 = load volatile i32, ptr @veles_attention_line, align 64
  %t108 = icmp ne i32 %t107, 0
  br i1 %t108, label %safepoint.43, label %safepoint.on.44, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.43:
  call void @veles_backedge_plain()
  br label %safepoint.on.44
safepoint.on.44:
  br label %loop.cond.1
loop.end.3:
  store i64 10, ptr %a109
  br label %loop.cond.45
loop.cond.45:
  %t110 = load i64, ptr %a109
  %t111 = icmp sgt i64 %t110, 0
  br i1 %t111, label %loop.body.48, label %loop.end.47
loop.body.48:
  %t112 = load i64, ptr %a109
  %t114 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t112, i64 3)
  %t115 = extractvalue { i64, i1 } %t114, 0
  %t116 = extractvalue { i64, i1 } %t114, 1
  br i1 %t116, label %overflow.49, label %arith.ok.50
overflow.49:
  %t117 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t118 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t119 = extractvalue %str { ptr @.str.15, i64 13 }, 0
  %t120 = extractvalue %str { ptr @.str.15, i64 13 }, 1
  call void @veles_panic_at(ptr %t117, i64 %t118, ptr %t119, i64 %t120)
  unreachable
arith.ok.50:
  store i64 %t115, ptr %a109
  br label %loop.post.46
loop.post.46:
  %t121 = load volatile i32, ptr @veles_attention_line, align 64
  %t122 = icmp ne i32 %t121, 0
  br i1 %t122, label %safepoint.51, label %safepoint.on.52, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.51:
  call void @veles_backedge_plain()
  br label %safepoint.on.52
safepoint.on.52:
  br label %loop.cond.45
loop.end.47:
  store i64 0, ptr %a123
  br label %loop.cond.53
loop.cond.53:
  br label %loop.body.56
loop.body.56:
  %t124 = load i64, ptr %a123
  %t126 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t124, i64 1)
  %t127 = extractvalue { i64, i1 } %t126, 0
  %t128 = extractvalue { i64, i1 } %t126, 1
  br i1 %t128, label %overflow.57, label %arith.ok.58
overflow.57:
  %t129 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t130 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t131 = extractvalue %str { ptr @.str.16, i64 12 }, 0
  %t132 = extractvalue %str { ptr @.str.16, i64 12 }, 1
  call void @veles_panic_at(ptr %t129, i64 %t130, ptr %t131, i64 %t132)
  unreachable
arith.ok.58:
  store i64 %t127, ptr %a123
  %t133 = load i64, ptr %a123
  %t134 = icmp eq i64 %t133, 4
  br i1 %t134, label %if.then.59, label %if.end.60
if.then.59:
  br label %loop.end.55
if.end.60:
  br label %loop.post.54
loop.post.54:
  %t135 = load volatile i32, ptr @veles_attention_line, align 64
  %t136 = icmp ne i32 %t135, 0
  br i1 %t136, label %safepoint.61, label %safepoint.on.62, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.61:
  call void @veles_backedge_plain()
  br label %safepoint.on.62
safepoint.on.62:
  br label %loop.cond.53
loop.end.55:
  %t137 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t138 = insertvalue { i64, i64, i1 } %t137, i64 10, 1
  %t139 = insertvalue { i64, i64, i1 } %t138, i1 true, 2
  store { i64, i64, i1 } %t139, ptr %a140
  %t141 = extractvalue %str { ptr @.str.17, i64 18 }, 0
  %t142 = extractvalue %str { ptr @.str.17, i64 18 }, 1
  call void @veles_call_push(ptr %t141)
  %t143 = call %S.std.prelude.RangeStepIter_i64_ @v_std.prelude.extend.Range_T.step_T_i64_(ptr %a140, i64 5, %str { ptr @.str.18, i64 13 })
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i64_ %t143, ptr %a144
  br label %loop.cond.63
loop.cond.63:
  br label %loop.body.66
loop.body.66:
  %t145 = extractvalue %str { ptr @.str.19, i64 18 }, 0
  %t146 = extractvalue %str { ptr @.str.19, i64 18 }, 1
  call void @veles_call_push(ptr %t145)
  %t147 = call { i1, i64 } @v_std.prelude.Iterator.RangeStepIter_T.next_T_i64_(ptr %a144)
  call void @veles_call_pop()
  store { i1, i64 } %t147, ptr %a148
  %t149 = load { i1, i64 }, ptr %a148
  %t151 = extractvalue { i1, i64 } %t149, 0
  %t150 = xor i1 %t151, true
  br i1 %t150, label %if.then.67, label %if.end.68
if.then.67:
  br label %loop.end.65
if.end.68:
  %t152 = load { i1, i64 }, ptr %a148
  %t153 = extractvalue { i1, i64 } %t152, 1
  store i64 %t153, ptr %a154
  %t155 = load i64, ptr %a154
  %t157 = call i64 @veles_i64_format(ptr %a156, i64 %t155)
  %t158 = insertvalue %str undef, ptr %a156, 0
  %t159 = insertvalue %str %t158, i64 %t157, 1
  %t161 = extractvalue %str { ptr @.str.20, i64 5 }, 0
  %t162 = extractvalue %str { ptr @.str.20, i64 5 }, 1
  %t163 = extractvalue %str %t159, 0
  %t164 = extractvalue %str %t159, 1
  call void @veles_string_concat(ptr %a160, ptr %t161, i64 %t162, ptr %t163, i64 %t164)
  %t165 = load %str, ptr %a160
  %t166 = extractvalue %str { ptr @.str.21, i64 21 }, 0
  %t167 = extractvalue %str { ptr @.str.21, i64 21 }, 1
  call void @veles_call_push(ptr %t166)
  call void @v_std.io.println(%str %t165)
  call void @veles_call_pop()
  br label %loop.post.64
loop.post.64:
  %t168 = load volatile i32, ptr @veles_attention_line, align 64
  %t169 = icmp ne i32 %t168, 0
  br i1 %t169, label %safepoint.69, label %safepoint.on.70, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.69:
  call void @veles_backedge_plain()
  br label %safepoint.on.70
safepoint.on.70:
  br label %loop.cond.63
loop.end.65:
  %t170 = load i64, ptr %a1
  %t172 = call i64 @veles_i64_format(ptr %a171, i64 %t170)
  %t173 = insertvalue %str undef, ptr %a171, 0
  %t174 = insertvalue %str %t173, i64 %t172, 1
  %t175 = load i64, ptr %a109
  %t177 = call i64 @veles_i64_format(ptr %a176, i64 %t175)
  %t178 = insertvalue %str undef, ptr %a176, 0
  %t179 = insertvalue %str %t178, i64 %t177, 1
  %t180 = load i64, ptr %a123
  %t182 = call i64 @veles_i64_format(ptr %a181, i64 %t180)
  %t183 = insertvalue %str undef, ptr %a181, 0
  %t184 = insertvalue %str %t183, i64 %t182, 1
  %t185 = extractvalue %str { ptr @.str.22, i64 22 }, 0
  %t186 = extractvalue %str { ptr @.str.22, i64 22 }, 1
  call void @veles_call_push(ptr %t185)
  %t187 = call %str @v_main.classify(i64 -2)
  call void @veles_call_pop()
  %t188 = extractvalue %str { ptr @.str.23, i64 22 }, 0
  %t189 = extractvalue %str { ptr @.str.23, i64 22 }, 1
  call void @veles_call_push(ptr %t188)
  %t190 = call %str @v_main.classify(i64 0)
  call void @veles_call_pop()
  %t191 = extractvalue %str { ptr @.str.24, i64 22 }, 0
  %t192 = extractvalue %str { ptr @.str.24, i64 22 }, 1
  call void @veles_call_push(ptr %t191)
  %t193 = call %str @v_main.classify(i64 4)
  call void @veles_call_pop()
  %t194 = extractvalue %str { ptr @.str.25, i64 20 }, 0
  %t195 = extractvalue %str { ptr @.str.25, i64 20 }, 1
  call void @veles_call_push(ptr %t194)
  %t196 = call %str @v_main.digits(i64 3)
  call void @veles_call_pop()
  %t197 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t198 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t197)
  %t199 = call %str @v_main.digits(i64 9)
  call void @veles_call_pop()
  %t202 = getelementptr [15 x %str], ptr %a201, i64 0, i64 0
  store %str %t174, ptr %t202
  %t203 = getelementptr [15 x %str], ptr %a201, i64 0, i64 1
  store %str { ptr @.str.27, i64 1 }, ptr %t203
  %t204 = getelementptr [15 x %str], ptr %a201, i64 0, i64 2
  store %str %t179, ptr %t204
  %t205 = getelementptr [15 x %str], ptr %a201, i64 0, i64 3
  store %str { ptr @.str.27, i64 1 }, ptr %t205
  %t206 = getelementptr [15 x %str], ptr %a201, i64 0, i64 4
  store %str %t184, ptr %t206
  %t207 = getelementptr [15 x %str], ptr %a201, i64 0, i64 5
  store %str { ptr @.str.27, i64 1 }, ptr %t207
  %t208 = getelementptr [15 x %str], ptr %a201, i64 0, i64 6
  store %str %t187, ptr %t208
  %t209 = getelementptr [15 x %str], ptr %a201, i64 0, i64 7
  store %str { ptr @.str.27, i64 1 }, ptr %t209
  %t210 = getelementptr [15 x %str], ptr %a201, i64 0, i64 8
  store %str %t190, ptr %t210
  %t211 = getelementptr [15 x %str], ptr %a201, i64 0, i64 9
  store %str { ptr @.str.27, i64 1 }, ptr %t211
  %t212 = getelementptr [15 x %str], ptr %a201, i64 0, i64 10
  store %str %t193, ptr %t212
  %t213 = getelementptr [15 x %str], ptr %a201, i64 0, i64 11
  store %str { ptr @.str.27, i64 1 }, ptr %t213
  %t214 = getelementptr [15 x %str], ptr %a201, i64 0, i64 12
  store %str %t196, ptr %t214
  %t215 = getelementptr [15 x %str], ptr %a201, i64 0, i64 13
  store %str { ptr @.str.27, i64 1 }, ptr %t215
  %t216 = getelementptr [15 x %str], ptr %a201, i64 0, i64 14
  store %str %t199, ptr %t216
  call void @veles_string_concat_n(ptr %a200, ptr %a201, i64 15)
  %t217 = load %str, ptr %a200
  %t218 = extractvalue %str { ptr @.str.28, i64 20 }, 0
  %t219 = extractvalue %str { ptr @.str.28, i64 20 }, 1
  call void @veles_call_push(ptr %t218)
  call void @v_std.io.println(%str %t217)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [9 x i8] c"negative\00"
@.str.2 = private unnamed_addr constant [5 x i8] c"zero\00"
@.str.3 = private unnamed_addr constant [6 x i8] c"small\00"
@.str.4 = private unnamed_addr constant [6 x i8] c"large\00"
@.str.5 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.6 = private unnamed_addr constant [13 x i8] c"main.vs:4:33\00"
@.str.7 = private unnamed_addr constant [4 x i8] c"one\00"
@.str.8 = private unnamed_addr constant [4 x i8] c"few\00"
@.str.9 = private unnamed_addr constant [5 x i8] c"many\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:11:31\00"
@.str.11 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.12 = private unnamed_addr constant [14 x i8] c"main.vs:22:11\00"
@.str.13 = private unnamed_addr constant [14 x i8] c"main.vs:23:16\00"
@.str.14 = private unnamed_addr constant [13 x i8] c"main.vs:23:7\00"
@.str.15 = private unnamed_addr constant [14 x i8] c"main.vs:27:16\00"
@.str.16 = private unnamed_addr constant [13 x i8] c"main.vs:30:5\00"
@.str.17 = private unnamed_addr constant [19 x i8] c"main.vs:33:14\00step\00"
@.str.18 = private unnamed_addr constant [14 x i8] c"main.vs:33:14\00"
@.str.19 = private unnamed_addr constant [19 x i8] c"main.vs:33:14\00next\00"
@.str.20 = private unnamed_addr constant [6 x i8] c"step \00"
@.str.21 = private unnamed_addr constant [22 x i8] c"main.vs:33:31\00println\00"
@.str.22 = private unnamed_addr constant [23 x i8] c"main.vs:34:34\00classify\00"
@.str.23 = private unnamed_addr constant [23 x i8] c"main.vs:34:50\00classify\00"
@.str.24 = private unnamed_addr constant [23 x i8] c"main.vs:34:65\00classify\00"
@.str.25 = private unnamed_addr constant [21 x i8] c"main.vs:34:80\00digits\00"
@.str.26 = private unnamed_addr constant [21 x i8] c"main.vs:34:93\00digits\00"
@.str.27 = private unnamed_addr constant [2 x i8] c" \00"
@.str.28 = private unnamed_addr constant [21 x i8] c"main.vs:34:3\00println\00"
