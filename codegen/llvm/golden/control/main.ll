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
  %a23 = alloca i64
  %a28 = alloca { i64, i64, i1 }
  %a31 = alloca i64
  %a34 = alloca i64
  %a35 = alloca i1
  %a46 = alloca i64
  %a83 = alloca i64
  %a95 = alloca i64
  %a110 = alloca { i64, i64, i1 }
  %a112 = alloca %S.std.prelude.RangeStepIter_i64_
  %a114 = alloca { i1, i64 }
  %a120 = alloca i64
  %a122 = alloca %str
  %a128 = alloca %str
  %a131 = alloca %str
  %a137 = alloca %str
  %a140 = alloca %str
  %a146 = alloca %str
  %a152 = alloca %str
  %a155 = alloca %str
  %a161 = alloca %str
  %a167 = alloca %str
  %a174 = alloca %str
  %a180 = alloca %str
  %a187 = alloca %str
  %a193 = alloca %str
  %a200 = alloca %str
  %a206 = alloca %str
  %a213 = alloca %str
  %a219 = alloca %str
  %a226 = alloca %str
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
  br label %loop.cond.1
loop.cond.1:
  %t13 = getelementptr inbounds { i64, i64, i1 }, ptr %a5, i32 0, i32 2
  %t14 = load i1, ptr %t13
  br i1 %t14, label %if.then.5, label %if.else.7
if.then.5:
  %t15 = load i64, ptr %a8
  %t16 = load i64, ptr %a11
  %t17 = icmp sle i64 %t15, %t16
  store i1 %t17, ptr %a12
  br label %if.end.6
if.else.7:
  %t18 = load i64, ptr %a8
  %t19 = load i64, ptr %a11
  %t20 = icmp slt i64 %t18, %t19
  store i1 %t20, ptr %a12
  br label %if.end.6
if.end.6:
  %t21 = load i1, ptr %a12
  br i1 %t21, label %loop.body.4, label %loop.end.3
loop.body.4:
  %t22 = load i64, ptr %a8
  store i64 %t22, ptr %a23
  %t24 = load i64, ptr %a23
  %t25 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t26 = insertvalue { i64, i64, i1 } %t25, i64 %t24, 1
  %t27 = insertvalue { i64, i64, i1 } %t26, i1 false, 2
  store { i64, i64, i1 } %t27, ptr %a28
  %t29 = getelementptr inbounds { i64, i64, i1 }, ptr %a28, i32 0, i32 0
  %t30 = load i64, ptr %t29
  store i64 %t30, ptr %a31
  %t32 = getelementptr inbounds { i64, i64, i1 }, ptr %a28, i32 0, i32 1
  %t33 = load i64, ptr %t32
  store i64 %t33, ptr %a34
  br label %loop.cond.8
loop.cond.8:
  %t36 = getelementptr inbounds { i64, i64, i1 }, ptr %a28, i32 0, i32 2
  %t37 = load i1, ptr %t36
  br i1 %t37, label %if.then.12, label %if.else.14
if.then.12:
  %t38 = load i64, ptr %a31
  %t39 = load i64, ptr %a34
  %t40 = icmp sle i64 %t38, %t39
  store i1 %t40, ptr %a35
  br label %if.end.13
if.else.14:
  %t41 = load i64, ptr %a31
  %t42 = load i64, ptr %a34
  %t43 = icmp slt i64 %t41, %t42
  store i1 %t43, ptr %a35
  br label %if.end.13
if.end.13:
  %t44 = load i1, ptr %a35
  br i1 %t44, label %loop.body.11, label %loop.end.10
loop.body.11:
  %t45 = load i64, ptr %a31
  store i64 %t45, ptr %a46
  %t47 = load i64, ptr %a46
  %t48 = icmp eq i64 %t47, 3
  br i1 %t48, label %if.then.15, label %if.end.16
if.then.15:
  br label %loop.post.2
if.end.16:
  %t49 = load i64, ptr %a23
  %t50 = load i64, ptr %a46
  %t52 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t49, i64 %t50)
  %t53 = extractvalue { i64, i1 } %t52, 0
  %t54 = extractvalue { i64, i1 } %t52, 1
  br i1 %t54, label %overflow.17, label %arith.ok.18
overflow.17:
  %t55 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t56 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t57 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t58 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_panic_at(ptr %t55, i64 %t56, ptr %t57, i64 %t58)
  unreachable
arith.ok.18:
  %t59 = icmp sgt i64 %t53, 8
  br i1 %t59, label %if.then.19, label %if.end.20
if.then.19:
  br label %loop.end.3
if.end.20:
  %t60 = load i64, ptr %a1
  %t61 = load i64, ptr %a23
  %t62 = load i64, ptr %a46
  %t64 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t61, i64 %t62)
  %t65 = extractvalue { i64, i1 } %t64, 0
  %t66 = extractvalue { i64, i1 } %t64, 1
  br i1 %t66, label %overflow.21, label %arith.ok.22
overflow.21:
  %t67 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t68 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t69 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t70 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_panic_at(ptr %t67, i64 %t68, ptr %t69, i64 %t70)
  unreachable
arith.ok.22:
  %t72 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t60, i64 %t65)
  %t73 = extractvalue { i64, i1 } %t72, 0
  %t74 = extractvalue { i64, i1 } %t72, 1
  br i1 %t74, label %overflow.23, label %arith.ok.24
overflow.23:
  %t75 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t76 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t77 = extractvalue %str { ptr @.str.14, i64 12 }, 0
  %t78 = extractvalue %str { ptr @.str.14, i64 12 }, 1
  call void @veles_panic_at(ptr %t75, i64 %t76, ptr %t77, i64 %t78)
  unreachable
arith.ok.24:
  store i64 %t73, ptr %a1
  br label %loop.post.9
loop.post.9:
  %t79 = load i64, ptr %a31
  %t80 = add i64 %t79, 1
  store i64 %t80, ptr %a31
  br label %loop.cond.8
loop.end.10:
  br label %loop.post.2
loop.post.2:
  %t81 = load i64, ptr %a8
  %t82 = add i64 %t81, 1
  store i64 %t82, ptr %a8
  br label %loop.cond.1
loop.end.3:
  store i64 10, ptr %a83
  br label %loop.cond.25
loop.cond.25:
  %t84 = load i64, ptr %a83
  %t85 = icmp sgt i64 %t84, 0
  br i1 %t85, label %loop.body.28, label %loop.end.27
loop.body.28:
  %t86 = load i64, ptr %a83
  %t88 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t86, i64 3)
  %t89 = extractvalue { i64, i1 } %t88, 0
  %t90 = extractvalue { i64, i1 } %t88, 1
  br i1 %t90, label %overflow.29, label %arith.ok.30
overflow.29:
  %t91 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t92 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t93 = extractvalue %str { ptr @.str.15, i64 13 }, 0
  %t94 = extractvalue %str { ptr @.str.15, i64 13 }, 1
  call void @veles_panic_at(ptr %t91, i64 %t92, ptr %t93, i64 %t94)
  unreachable
arith.ok.30:
  store i64 %t89, ptr %a83
  br label %loop.post.26
loop.post.26:
  br label %loop.cond.25
loop.end.27:
  store i64 0, ptr %a95
  br label %loop.cond.31
loop.cond.31:
  br label %loop.body.34
loop.body.34:
  %t96 = load i64, ptr %a95
  %t98 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t96, i64 1)
  %t99 = extractvalue { i64, i1 } %t98, 0
  %t100 = extractvalue { i64, i1 } %t98, 1
  br i1 %t100, label %overflow.35, label %arith.ok.36
overflow.35:
  %t101 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t102 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t103 = extractvalue %str { ptr @.str.16, i64 12 }, 0
  %t104 = extractvalue %str { ptr @.str.16, i64 12 }, 1
  call void @veles_panic_at(ptr %t101, i64 %t102, ptr %t103, i64 %t104)
  unreachable
arith.ok.36:
  store i64 %t99, ptr %a95
  %t105 = load i64, ptr %a95
  %t106 = icmp eq i64 %t105, 4
  br i1 %t106, label %if.then.37, label %if.end.38
if.then.37:
  br label %loop.end.33
if.end.38:
  br label %loop.post.32
loop.post.32:
  br label %loop.cond.31
loop.end.33:
  %t107 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t108 = insertvalue { i64, i64, i1 } %t107, i64 10, 1
  %t109 = insertvalue { i64, i64, i1 } %t108, i1 true, 2
  store { i64, i64, i1 } %t109, ptr %a110
  %t111 = call %S.std.prelude.RangeStepIter_i64_ @v_std.prelude.extend.Range_T.step_T_i64_(ptr %a110, i64 5)
  store %S.std.prelude.RangeStepIter_i64_ %t111, ptr %a112
  br label %loop.cond.39
loop.cond.39:
  br label %loop.body.42
loop.body.42:
  %t113 = call { i1, i64 } @v_std.prelude.Iterator.RangeStepIter_T.next_T_i64_(ptr %a112)
  store { i1, i64 } %t113, ptr %a114
  %t115 = load { i1, i64 }, ptr %a114
  %t117 = extractvalue { i1, i64 } %t115, 0
  %t116 = xor i1 %t117, true
  br i1 %t116, label %if.then.43, label %if.end.44
if.then.43:
  br label %loop.end.41
if.end.44:
  %t118 = load { i1, i64 }, ptr %a114
  %t119 = extractvalue { i1, i64 } %t118, 1
  store i64 %t119, ptr %a120
  %t121 = load i64, ptr %a120
  call void @veles_i64_to_string(ptr %a122, i64 %t121)
  %t123 = load %str, ptr %a122
  %t124 = extractvalue %str { ptr @.str.17, i64 5 }, 0
  %t125 = extractvalue %str { ptr @.str.17, i64 5 }, 1
  %t126 = extractvalue %str %t123, 0
  %t127 = extractvalue %str %t123, 1
  call void @veles_string_concat(ptr %a128, ptr %t124, i64 %t125, ptr %t126, i64 %t127)
  %t129 = load %str, ptr %a128
  call void @v_std.io.println(%str %t129)
  br label %loop.post.40
loop.post.40:
  br label %loop.cond.39
loop.end.41:
  %t130 = load i64, ptr %a1
  call void @veles_i64_to_string(ptr %a131, i64 %t130)
  %t132 = load %str, ptr %a131
  %t133 = extractvalue %str %t132, 0
  %t134 = extractvalue %str %t132, 1
  %t135 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t136 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a137, ptr %t133, i64 %t134, ptr %t135, i64 %t136)
  %t138 = load %str, ptr %a137
  %t139 = load i64, ptr %a83
  call void @veles_i64_to_string(ptr %a140, i64 %t139)
  %t141 = load %str, ptr %a140
  %t142 = extractvalue %str %t138, 0
  %t143 = extractvalue %str %t138, 1
  %t144 = extractvalue %str %t141, 0
  %t145 = extractvalue %str %t141, 1
  call void @veles_string_concat(ptr %a146, ptr %t142, i64 %t143, ptr %t144, i64 %t145)
  %t147 = load %str, ptr %a146
  %t148 = extractvalue %str %t147, 0
  %t149 = extractvalue %str %t147, 1
  %t150 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t151 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a152, ptr %t148, i64 %t149, ptr %t150, i64 %t151)
  %t153 = load %str, ptr %a152
  %t154 = load i64, ptr %a95
  call void @veles_i64_to_string(ptr %a155, i64 %t154)
  %t156 = load %str, ptr %a155
  %t157 = extractvalue %str %t153, 0
  %t158 = extractvalue %str %t153, 1
  %t159 = extractvalue %str %t156, 0
  %t160 = extractvalue %str %t156, 1
  call void @veles_string_concat(ptr %a161, ptr %t157, i64 %t158, ptr %t159, i64 %t160)
  %t162 = load %str, ptr %a161
  %t163 = extractvalue %str %t162, 0
  %t164 = extractvalue %str %t162, 1
  %t165 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t166 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a167, ptr %t163, i64 %t164, ptr %t165, i64 %t166)
  %t168 = load %str, ptr %a167
  %t169 = call %str @v_main.classify(i64 -2)
  %t170 = extractvalue %str %t168, 0
  %t171 = extractvalue %str %t168, 1
  %t172 = extractvalue %str %t169, 0
  %t173 = extractvalue %str %t169, 1
  call void @veles_string_concat(ptr %a174, ptr %t170, i64 %t171, ptr %t172, i64 %t173)
  %t175 = load %str, ptr %a174
  %t176 = extractvalue %str %t175, 0
  %t177 = extractvalue %str %t175, 1
  %t178 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t179 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a180, ptr %t176, i64 %t177, ptr %t178, i64 %t179)
  %t181 = load %str, ptr %a180
  %t182 = call %str @v_main.classify(i64 0)
  %t183 = extractvalue %str %t181, 0
  %t184 = extractvalue %str %t181, 1
  %t185 = extractvalue %str %t182, 0
  %t186 = extractvalue %str %t182, 1
  call void @veles_string_concat(ptr %a187, ptr %t183, i64 %t184, ptr %t185, i64 %t186)
  %t188 = load %str, ptr %a187
  %t189 = extractvalue %str %t188, 0
  %t190 = extractvalue %str %t188, 1
  %t191 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t192 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a193, ptr %t189, i64 %t190, ptr %t191, i64 %t192)
  %t194 = load %str, ptr %a193
  %t195 = call %str @v_main.classify(i64 4)
  %t196 = extractvalue %str %t194, 0
  %t197 = extractvalue %str %t194, 1
  %t198 = extractvalue %str %t195, 0
  %t199 = extractvalue %str %t195, 1
  call void @veles_string_concat(ptr %a200, ptr %t196, i64 %t197, ptr %t198, i64 %t199)
  %t201 = load %str, ptr %a200
  %t202 = extractvalue %str %t201, 0
  %t203 = extractvalue %str %t201, 1
  %t204 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t205 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a206, ptr %t202, i64 %t203, ptr %t204, i64 %t205)
  %t207 = load %str, ptr %a206
  %t208 = call %str @v_main.digits(i64 3)
  %t209 = extractvalue %str %t207, 0
  %t210 = extractvalue %str %t207, 1
  %t211 = extractvalue %str %t208, 0
  %t212 = extractvalue %str %t208, 1
  call void @veles_string_concat(ptr %a213, ptr %t209, i64 %t210, ptr %t211, i64 %t212)
  %t214 = load %str, ptr %a213
  %t215 = extractvalue %str %t214, 0
  %t216 = extractvalue %str %t214, 1
  %t217 = extractvalue %str { ptr @.str.18, i64 1 }, 0
  %t218 = extractvalue %str { ptr @.str.18, i64 1 }, 1
  call void @veles_string_concat(ptr %a219, ptr %t215, i64 %t216, ptr %t217, i64 %t218)
  %t220 = load %str, ptr %a219
  %t221 = call %str @v_main.digits(i64 9)
  %t222 = extractvalue %str %t220, 0
  %t223 = extractvalue %str %t220, 1
  %t224 = extractvalue %str %t221, 0
  %t225 = extractvalue %str %t221, 1
  call void @veles_string_concat(ptr %a226, ptr %t222, i64 %t223, ptr %t224, i64 %t225)
  %t227 = load %str, ptr %a226
  call void @v_std.io.println(%str %t227)
  ret void
}

@.str.1 = private unnamed_addr constant [9 x i8] c"negative\00"
@.str.2 = private unnamed_addr constant [5 x i8] c"zero\00"
@.str.3 = private unnamed_addr constant [6 x i8] c"small\00"
@.str.4 = private unnamed_addr constant [6 x i8] c"large\00"
@.str.5 = private unnamed_addr constant [34 x i8] c"unreachable: non-exhaustive match\00"
@.str.6 = private unnamed_addr constant [13 x i8] c"main.vs:4:32\00"
@.str.7 = private unnamed_addr constant [4 x i8] c"one\00"
@.str.8 = private unnamed_addr constant [4 x i8] c"few\00"
@.str.9 = private unnamed_addr constant [5 x i8] c"many\00"
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:11:30\00"
@.str.11 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.12 = private unnamed_addr constant [14 x i8] c"main.vs:22:11\00"
@.str.13 = private unnamed_addr constant [14 x i8] c"main.vs:23:16\00"
@.str.14 = private unnamed_addr constant [13 x i8] c"main.vs:23:7\00"
@.str.15 = private unnamed_addr constant [14 x i8] c"main.vs:27:16\00"
@.str.16 = private unnamed_addr constant [13 x i8] c"main.vs:30:5\00"
@.str.17 = private unnamed_addr constant [6 x i8] c"step \00"
@.str.18 = private unnamed_addr constant [2 x i8] c" \00"
