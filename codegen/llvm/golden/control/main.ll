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
  %a40 = alloca i64
  %a41 = alloca i1
  %a43 = alloca i1
  %a54 = alloca i64
  %a68 = alloca i1
  %a70 = alloca i1
  %a85 = alloca i64
  %a118 = alloca i1
  %a129 = alloca i1
  %a140 = alloca i64
  %a154 = alloca i64
  %a171 = alloca { i64, i64, i1 }
  %a175 = alloca %S.std.prelude.RangeStepIter_i64_
  %a179 = alloca { i1, i64 }
  %a185 = alloca i64
  %a187 = alloca [21 x i8]
  %a191 = alloca %str
  %a202 = alloca [21 x i8]
  %a207 = alloca [21 x i8]
  %a212 = alloca [21 x i8]
  %a231 = alloca %str
  %a232 = alloca [15 x %str]
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
  store i64 0, ptr %a40
  br label %chunk.cond.11
chunk.cond.11:
  %t42 = load i1, ptr %a39
  br i1 %t42, label %if.then.18, label %if.else.20
if.then.18:
  store i1 false, ptr %a41
  br label %if.end.19
if.else.20:
  %t44 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 2
  %t45 = load i1, ptr %t44
  br i1 %t45, label %if.then.21, label %if.else.23
if.then.21:
  %t46 = load i64, ptr %a35
  %t47 = load i64, ptr %a38
  %t48 = icmp sle i64 %t46, %t47
  store i1 %t48, ptr %a43
  br label %if.end.22
if.else.23:
  %t49 = load i64, ptr %a35
  %t50 = load i64, ptr %a38
  %t51 = icmp slt i64 %t49, %t50
  store i1 %t51, ptr %a43
  br label %if.end.22
if.end.22:
  %t52 = load i1, ptr %a43
  store i1 %t52, ptr %a41
  br label %if.end.19
if.end.19:
  %t53 = load i1, ptr %a41
  br i1 %t53, label %chunk.start.17, label %chunk.end.13
chunk.start.17:
  %t55 = load i64, ptr %a35
  %t56 = icmp sgt i64 %t55, 9223372036854775510
  br i1 %t56, label %if.then.24, label %if.else.26
if.then.24:
  store i64 9223372036854775807, ptr %a54
  br label %if.end.25
if.else.26:
  %t57 = load i64, ptr %a35
  %t59 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t57, i64 297)
  %t60 = extractvalue { i64, i1 } %t59, 0
  %t61 = extractvalue { i64, i1 } %t59, 1
  br i1 %t61, label %overflow.27, label %arith.ok.28
overflow.27:
  %t62 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t63 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t64 = extractvalue %str { ptr @.str.12, i64 0 }, 0
  %t65 = extractvalue %str { ptr @.str.12, i64 0 }, 1
  call void @veles_panic_at(ptr %t62, i64 %t63, ptr %t64, i64 %t65)
  unreachable
arith.ok.28:
  store i64 %t60, ptr %a54
  br label %if.end.25
if.end.25:
  %t66 = load i64, ptr %a54
  store i64 %t66, ptr %a40
  br label %chunk.body.15
chunk.inner.14:
  %t69 = load i1, ptr %a39
  br i1 %t69, label %if.then.29, label %if.else.31
if.then.29:
  store i1 false, ptr %a68
  br label %if.end.30
if.else.31:
  %t71 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 2
  %t72 = load i1, ptr %t71
  br i1 %t72, label %if.then.32, label %if.else.34
if.then.32:
  %t73 = load i64, ptr %a35
  %t74 = load i64, ptr %a38
  %t75 = icmp sle i64 %t73, %t74
  store i1 %t75, ptr %a70
  br label %if.end.33
if.else.34:
  %t76 = load i64, ptr %a35
  %t77 = load i64, ptr %a38
  %t78 = icmp slt i64 %t76, %t77
  store i1 %t78, ptr %a70
  br label %if.end.33
if.end.33:
  %t79 = load i1, ptr %a70
  store i1 %t79, ptr %a68
  br label %if.end.30
if.end.30:
  %t80 = load i1, ptr %a68
  %t81 = load i64, ptr %a35
  %t82 = load i64, ptr %a40
  %t83 = icmp slt i64 %t81, %t82
  %t67 = and i1 %t80, %t83
  br i1 %t67, label %chunk.body.15, label %chunk.looked.16
chunk.body.15:
  %t84 = load i64, ptr %a35
  store i64 %t84, ptr %a85
  %t86 = load i64, ptr %a85
  %t87 = icmp eq i64 %t86, 3
  br i1 %t87, label %if.then.35, label %if.end.36
if.then.35:
  br label %loop.post.2
if.end.36:
  %t88 = load i64, ptr %a27
  %t89 = load i64, ptr %a85
  %t91 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t88, i64 %t89)
  %t92 = extractvalue { i64, i1 } %t91, 0
  %t93 = extractvalue { i64, i1 } %t91, 1
  br i1 %t93, label %overflow.37, label %arith.ok.38
overflow.37:
  %t94 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t95 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t96 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t97 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t94, i64 %t95, ptr %t96, i64 %t97)
  unreachable
arith.ok.38:
  %t98 = icmp sgt i64 %t92, 8
  br i1 %t98, label %if.then.39, label %if.end.40
if.then.39:
  br label %loop.end.3
if.end.40:
  %t99 = load i64, ptr %a1
  %t100 = load i64, ptr %a27
  %t101 = load i64, ptr %a85
  %t103 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t100, i64 %t101)
  %t104 = extractvalue { i64, i1 } %t103, 0
  %t105 = extractvalue { i64, i1 } %t103, 1
  br i1 %t105, label %overflow.41, label %arith.ok.42
overflow.41:
  %t106 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t107 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t108 = extractvalue %str { ptr @.str.14, i64 13 }, 0
  %t109 = extractvalue %str { ptr @.str.14, i64 13 }, 1
  call void @veles_panic_at(ptr %t106, i64 %t107, ptr %t108, i64 %t109)
  unreachable
arith.ok.42:
  %t111 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t99, i64 %t104)
  %t112 = extractvalue { i64, i1 } %t111, 0
  %t113 = extractvalue { i64, i1 } %t111, 1
  br i1 %t113, label %overflow.43, label %arith.ok.44
overflow.43:
  %t114 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t115 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t116 = extractvalue %str { ptr @.str.15, i64 12 }, 0
  %t117 = extractvalue %str { ptr @.str.15, i64 12 }, 1
  call void @veles_panic_at(ptr %t114, i64 %t115, ptr %t116, i64 %t117)
  unreachable
arith.ok.44:
  store i64 %t112, ptr %a1
  br label %chunk.post.12
chunk.post.12:
  %t119 = getelementptr inbounds { i64, i64, i1 }, ptr %a32, i32 0, i32 2
  %t120 = load i1, ptr %t119
  store i1 %t120, ptr %a118
  br i1 %t120, label %sc.rhs.45, label %sc.end.46
sc.rhs.45:
  %t121 = load i64, ptr %a35
  %t122 = load i64, ptr %a38
  %t123 = icmp eq i64 %t121, %t122
  store i1 %t123, ptr %a118
  br label %sc.end.46
sc.end.46:
  %t124 = load i1, ptr %a118
  br i1 %t124, label %if.then.47, label %if.else.49
if.then.47:
  store i1 true, ptr %a39
  br label %if.end.48
if.else.49:
  %t125 = load i64, ptr %a35
  %t126 = add i64 %t125, 1
  store i64 %t126, ptr %a35
  br label %if.end.48
if.end.48:
  br label %chunk.inner.14
chunk.looked.16:
  %t127 = load volatile i32, ptr @veles_attention_line, align 64
  %t128 = icmp ne i32 %t127, 0
  br i1 %t128, label %safepoint.50, label %safepoint.on.51, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.50:
  call void @veles_backedge_plain()
  br label %safepoint.on.51
safepoint.on.51:
  br label %chunk.cond.11
chunk.end.13:
  br label %loop.post.2
loop.post.2:
  %t130 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t131 = load i1, ptr %t130
  store i1 %t131, ptr %a129
  br i1 %t131, label %sc.rhs.52, label %sc.end.53
sc.rhs.52:
  %t132 = load i64, ptr %a8
  %t133 = load i64, ptr %a11
  %t134 = icmp eq i64 %t132, %t133
  store i1 %t134, ptr %a129
  br label %sc.end.53
sc.end.53:
  %t135 = load i1, ptr %a129
  br i1 %t135, label %if.then.54, label %if.else.56
if.then.54:
  store i1 true, ptr %a12
  br label %if.end.55
if.else.56:
  %t136 = load i64, ptr %a8
  %t137 = add i64 %t136, 1
  store i64 %t137, ptr %a8
  br label %if.end.55
if.end.55:
  %t138 = load volatile i32, ptr @veles_attention_line, align 64
  %t139 = icmp ne i32 %t138, 0
  br i1 %t139, label %safepoint.57, label %safepoint.on.58, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.57:
  call void @veles_backedge_plain()
  br label %safepoint.on.58
safepoint.on.58:
  br label %loop.cond.1
loop.end.3:
  store i64 10, ptr %a140
  br label %loop.cond.59
loop.cond.59:
  %t141 = load i64, ptr %a140
  %t142 = icmp sgt i64 %t141, 0
  br i1 %t142, label %loop.body.62, label %loop.end.61
loop.body.62:
  %t143 = load i64, ptr %a140
  %t145 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t143, i64 3)
  %t146 = extractvalue { i64, i1 } %t145, 0
  %t147 = extractvalue { i64, i1 } %t145, 1
  br i1 %t147, label %overflow.63, label %arith.ok.64
overflow.63:
  %t148 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t149 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t150 = extractvalue %str { ptr @.str.16, i64 13 }, 0
  %t151 = extractvalue %str { ptr @.str.16, i64 13 }, 1
  call void @veles_panic_at(ptr %t148, i64 %t149, ptr %t150, i64 %t151)
  unreachable
arith.ok.64:
  store i64 %t146, ptr %a140
  br label %loop.post.60
loop.post.60:
  %t152 = load volatile i32, ptr @veles_attention_line, align 64
  %t153 = icmp ne i32 %t152, 0
  br i1 %t153, label %safepoint.65, label %safepoint.on.66, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.65:
  call void @veles_backedge_plain()
  br label %safepoint.on.66
safepoint.on.66:
  br label %loop.cond.59
loop.end.61:
  store i64 0, ptr %a154
  br label %loop.cond.67
loop.cond.67:
  br label %loop.body.70
loop.body.70:
  %t155 = load i64, ptr %a154
  %t157 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t155, i64 1)
  %t158 = extractvalue { i64, i1 } %t157, 0
  %t159 = extractvalue { i64, i1 } %t157, 1
  br i1 %t159, label %overflow.71, label %arith.ok.72
overflow.71:
  %t160 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t161 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t162 = extractvalue %str { ptr @.str.17, i64 12 }, 0
  %t163 = extractvalue %str { ptr @.str.17, i64 12 }, 1
  call void @veles_panic_at(ptr %t160, i64 %t161, ptr %t162, i64 %t163)
  unreachable
arith.ok.72:
  store i64 %t158, ptr %a154
  %t164 = load i64, ptr %a154
  %t165 = icmp eq i64 %t164, 4
  br i1 %t165, label %if.then.73, label %if.end.74
if.then.73:
  br label %loop.end.69
if.end.74:
  br label %loop.post.68
loop.post.68:
  %t166 = load volatile i32, ptr @veles_attention_line, align 64
  %t167 = icmp ne i32 %t166, 0
  br i1 %t167, label %safepoint.75, label %safepoint.on.76, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.75:
  call void @veles_backedge_plain()
  br label %safepoint.on.76
safepoint.on.76:
  br label %loop.cond.67
loop.end.69:
  %t168 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t169 = insertvalue { i64, i64, i1 } %t168, i64 10, 1
  %t170 = insertvalue { i64, i64, i1 } %t169, i1 true, 2
  store { i64, i64, i1 } %t170, ptr %a171
  %t172 = extractvalue %str { ptr @.str.18, i64 18 }, 0
  %t173 = extractvalue %str { ptr @.str.18, i64 18 }, 1
  call void @veles_call_push(ptr %t172)
  %t174 = call %S.std.prelude.RangeStepIter_i64_ @v_std.prelude.extend.Range_T.step_T_i64_(ptr %a171, i64 5, %str { ptr @.str.19, i64 13 })
  call void @veles_call_pop()
  store %S.std.prelude.RangeStepIter_i64_ %t174, ptr %a175
  br label %loop.cond.77
loop.cond.77:
  br label %loop.body.80
loop.body.80:
  %t176 = extractvalue %str { ptr @.str.20, i64 18 }, 0
  %t177 = extractvalue %str { ptr @.str.20, i64 18 }, 1
  call void @veles_call_push(ptr %t176)
  %t178 = call { i1, i64 } @v_std.prelude.Iterator.RangeStepIter_T.next_T_i64_(ptr %a175)
  call void @veles_call_pop()
  store { i1, i64 } %t178, ptr %a179
  %t180 = load { i1, i64 }, ptr %a179
  %t182 = extractvalue { i1, i64 } %t180, 0
  %t181 = xor i1 %t182, true
  br i1 %t181, label %if.then.81, label %if.end.82
if.then.81:
  br label %loop.end.79
if.end.82:
  %t183 = load { i1, i64 }, ptr %a179
  %t184 = extractvalue { i1, i64 } %t183, 1
  store i64 %t184, ptr %a185
  %t186 = load i64, ptr %a185
  %t188 = call i64 @veles_i64_format(ptr %a187, i64 %t186)
  %t189 = insertvalue %str undef, ptr %a187, 0
  %t190 = insertvalue %str %t189, i64 %t188, 1
  %t192 = extractvalue %str { ptr @.str.21, i64 5 }, 0
  %t193 = extractvalue %str { ptr @.str.21, i64 5 }, 1
  %t194 = extractvalue %str %t190, 0
  %t195 = extractvalue %str %t190, 1
  call void @veles_string_concat(ptr %a191, ptr %t192, i64 %t193, ptr %t194, i64 %t195)
  %t196 = load %str, ptr %a191
  %t197 = extractvalue %str { ptr @.str.22, i64 21 }, 0
  %t198 = extractvalue %str { ptr @.str.22, i64 21 }, 1
  call void @veles_call_push(ptr %t197)
  call void @v_std.io.println(%str %t196)
  call void @veles_call_pop()
  br label %loop.post.78
loop.post.78:
  %t199 = load volatile i32, ptr @veles_attention_line, align 64
  %t200 = icmp ne i32 %t199, 0
  br i1 %t200, label %safepoint.83, label %safepoint.on.84, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.83:
  call void @veles_backedge_plain()
  br label %safepoint.on.84
safepoint.on.84:
  br label %loop.cond.77
loop.end.79:
  %t201 = load i64, ptr %a1
  %t203 = call i64 @veles_i64_format(ptr %a202, i64 %t201)
  %t204 = insertvalue %str undef, ptr %a202, 0
  %t205 = insertvalue %str %t204, i64 %t203, 1
  %t206 = load i64, ptr %a140
  %t208 = call i64 @veles_i64_format(ptr %a207, i64 %t206)
  %t209 = insertvalue %str undef, ptr %a207, 0
  %t210 = insertvalue %str %t209, i64 %t208, 1
  %t211 = load i64, ptr %a154
  %t213 = call i64 @veles_i64_format(ptr %a212, i64 %t211)
  %t214 = insertvalue %str undef, ptr %a212, 0
  %t215 = insertvalue %str %t214, i64 %t213, 1
  %t216 = extractvalue %str { ptr @.str.23, i64 22 }, 0
  %t217 = extractvalue %str { ptr @.str.23, i64 22 }, 1
  call void @veles_call_push(ptr %t216)
  %t218 = call %str @v_main.classify(i64 -2)
  call void @veles_call_pop()
  %t219 = extractvalue %str { ptr @.str.24, i64 22 }, 0
  %t220 = extractvalue %str { ptr @.str.24, i64 22 }, 1
  call void @veles_call_push(ptr %t219)
  %t221 = call %str @v_main.classify(i64 0)
  call void @veles_call_pop()
  %t222 = extractvalue %str { ptr @.str.25, i64 22 }, 0
  %t223 = extractvalue %str { ptr @.str.25, i64 22 }, 1
  call void @veles_call_push(ptr %t222)
  %t224 = call %str @v_main.classify(i64 4)
  call void @veles_call_pop()
  %t225 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t226 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t225)
  %t227 = call %str @v_main.digits(i64 3)
  call void @veles_call_pop()
  %t228 = extractvalue %str { ptr @.str.27, i64 20 }, 0
  %t229 = extractvalue %str { ptr @.str.27, i64 20 }, 1
  call void @veles_call_push(ptr %t228)
  %t230 = call %str @v_main.digits(i64 9)
  call void @veles_call_pop()
  %t233 = getelementptr [15 x %str], ptr %a232, i64 0, i64 0
  store %str %t205, ptr %t233
  %t234 = getelementptr [15 x %str], ptr %a232, i64 0, i64 1
  store %str { ptr @.str.28, i64 1 }, ptr %t234
  %t235 = getelementptr [15 x %str], ptr %a232, i64 0, i64 2
  store %str %t210, ptr %t235
  %t236 = getelementptr [15 x %str], ptr %a232, i64 0, i64 3
  store %str { ptr @.str.28, i64 1 }, ptr %t236
  %t237 = getelementptr [15 x %str], ptr %a232, i64 0, i64 4
  store %str %t215, ptr %t237
  %t238 = getelementptr [15 x %str], ptr %a232, i64 0, i64 5
  store %str { ptr @.str.28, i64 1 }, ptr %t238
  %t239 = getelementptr [15 x %str], ptr %a232, i64 0, i64 6
  store %str %t218, ptr %t239
  %t240 = getelementptr [15 x %str], ptr %a232, i64 0, i64 7
  store %str { ptr @.str.28, i64 1 }, ptr %t240
  %t241 = getelementptr [15 x %str], ptr %a232, i64 0, i64 8
  store %str %t221, ptr %t241
  %t242 = getelementptr [15 x %str], ptr %a232, i64 0, i64 9
  store %str { ptr @.str.28, i64 1 }, ptr %t242
  %t243 = getelementptr [15 x %str], ptr %a232, i64 0, i64 10
  store %str %t224, ptr %t243
  %t244 = getelementptr [15 x %str], ptr %a232, i64 0, i64 11
  store %str { ptr @.str.28, i64 1 }, ptr %t244
  %t245 = getelementptr [15 x %str], ptr %a232, i64 0, i64 12
  store %str %t227, ptr %t245
  %t246 = getelementptr [15 x %str], ptr %a232, i64 0, i64 13
  store %str { ptr @.str.28, i64 1 }, ptr %t246
  %t247 = getelementptr [15 x %str], ptr %a232, i64 0, i64 14
  store %str %t230, ptr %t247
  call void @veles_string_concat_n(ptr %a231, ptr %a232, i64 15)
  %t248 = load %str, ptr %a231
  %t249 = extractvalue %str { ptr @.str.29, i64 20 }, 0
  %t250 = extractvalue %str { ptr @.str.29, i64 20 }, 1
  call void @veles_call_push(ptr %t249)
  call void @v_std.io.println(%str %t248)
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
@.str.12 = private unnamed_addr constant [1 x i8] c"\00"
@.str.13 = private unnamed_addr constant [14 x i8] c"main.vs:22:11\00"
@.str.14 = private unnamed_addr constant [14 x i8] c"main.vs:23:16\00"
@.str.15 = private unnamed_addr constant [13 x i8] c"main.vs:23:7\00"
@.str.16 = private unnamed_addr constant [14 x i8] c"main.vs:27:16\00"
@.str.17 = private unnamed_addr constant [13 x i8] c"main.vs:30:5\00"
@.str.18 = private unnamed_addr constant [19 x i8] c"main.vs:33:14\00step\00"
@.str.19 = private unnamed_addr constant [14 x i8] c"main.vs:33:14\00"
@.str.20 = private unnamed_addr constant [19 x i8] c"main.vs:33:14\00next\00"
@.str.21 = private unnamed_addr constant [6 x i8] c"step \00"
@.str.22 = private unnamed_addr constant [22 x i8] c"main.vs:33:31\00println\00"
@.str.23 = private unnamed_addr constant [23 x i8] c"main.vs:34:34\00classify\00"
@.str.24 = private unnamed_addr constant [23 x i8] c"main.vs:34:50\00classify\00"
@.str.25 = private unnamed_addr constant [23 x i8] c"main.vs:34:65\00classify\00"
@.str.26 = private unnamed_addr constant [21 x i8] c"main.vs:34:80\00digits\00"
@.str.27 = private unnamed_addr constant [21 x i8] c"main.vs:34:93\00digits\00"
@.str.28 = private unnamed_addr constant [2 x i8] c" \00"
@.str.29 = private unnamed_addr constant [21 x i8] c"main.vs:34:3\00println\00"
