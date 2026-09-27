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
  %a87 = alloca i64
  %a101 = alloca i64
  %a118 = alloca { i64, i64, i1 }
  %a120 = alloca %S.std.prelude.RangeStepIter_i64_
  %a122 = alloca { i1, i64 }
  %a128 = alloca i64
  %a130 = alloca [21 x i8]
  %a134 = alloca %str
  %a143 = alloca [21 x i8]
  %a148 = alloca [21 x i8]
  %a153 = alloca [21 x i8]
  %a162 = alloca %str
  %a163 = alloca [15 x %str]
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
  %t81 = load volatile i32, ptr @veles_stop_requested, align 4
  %t82 = icmp ne i32 %t81, 0
  br i1 %t82, label %safepoint.25, label %safepoint.on.26, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.25:
  call void @veles_gc_park()
  br label %safepoint.on.26
safepoint.on.26:
  br label %loop.cond.8
loop.end.10:
  br label %loop.post.2
loop.post.2:
  %t83 = load i64, ptr %a8
  %t84 = add i64 %t83, 1
  store i64 %t84, ptr %a8
  %t85 = load volatile i32, ptr @veles_stop_requested, align 4
  %t86 = icmp ne i32 %t85, 0
  br i1 %t86, label %safepoint.27, label %safepoint.on.28, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.27:
  call void @veles_gc_park()
  br label %safepoint.on.28
safepoint.on.28:
  br label %loop.cond.1
loop.end.3:
  store i64 10, ptr %a87
  br label %loop.cond.29
loop.cond.29:
  %t88 = load i64, ptr %a87
  %t89 = icmp sgt i64 %t88, 0
  br i1 %t89, label %loop.body.32, label %loop.end.31
loop.body.32:
  %t90 = load i64, ptr %a87
  %t92 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t90, i64 3)
  %t93 = extractvalue { i64, i1 } %t92, 0
  %t94 = extractvalue { i64, i1 } %t92, 1
  br i1 %t94, label %overflow.33, label %arith.ok.34
overflow.33:
  %t95 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t96 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t97 = extractvalue %str { ptr @.str.15, i64 13 }, 0
  %t98 = extractvalue %str { ptr @.str.15, i64 13 }, 1
  call void @veles_panic_at(ptr %t95, i64 %t96, ptr %t97, i64 %t98)
  unreachable
arith.ok.34:
  store i64 %t93, ptr %a87
  br label %loop.post.30
loop.post.30:
  %t99 = load volatile i32, ptr @veles_stop_requested, align 4
  %t100 = icmp ne i32 %t99, 0
  br i1 %t100, label %safepoint.35, label %safepoint.on.36, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.35:
  call void @veles_gc_park()
  br label %safepoint.on.36
safepoint.on.36:
  br label %loop.cond.29
loop.end.31:
  store i64 0, ptr %a101
  br label %loop.cond.37
loop.cond.37:
  br label %loop.body.40
loop.body.40:
  %t102 = load i64, ptr %a101
  %t104 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t102, i64 1)
  %t105 = extractvalue { i64, i1 } %t104, 0
  %t106 = extractvalue { i64, i1 } %t104, 1
  br i1 %t106, label %overflow.41, label %arith.ok.42
overflow.41:
  %t107 = extractvalue %str { ptr @.str.11, i64 16 }, 0
  %t108 = extractvalue %str { ptr @.str.11, i64 16 }, 1
  %t109 = extractvalue %str { ptr @.str.16, i64 12 }, 0
  %t110 = extractvalue %str { ptr @.str.16, i64 12 }, 1
  call void @veles_panic_at(ptr %t107, i64 %t108, ptr %t109, i64 %t110)
  unreachable
arith.ok.42:
  store i64 %t105, ptr %a101
  %t111 = load i64, ptr %a101
  %t112 = icmp eq i64 %t111, 4
  br i1 %t112, label %if.then.43, label %if.end.44
if.then.43:
  br label %loop.end.39
if.end.44:
  br label %loop.post.38
loop.post.38:
  %t113 = load volatile i32, ptr @veles_stop_requested, align 4
  %t114 = icmp ne i32 %t113, 0
  br i1 %t114, label %safepoint.45, label %safepoint.on.46, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.45:
  call void @veles_gc_park()
  br label %safepoint.on.46
safepoint.on.46:
  br label %loop.cond.37
loop.end.39:
  %t115 = insertvalue { i64, i64, i1 } undef, i64 0, 0
  %t116 = insertvalue { i64, i64, i1 } %t115, i64 10, 1
  %t117 = insertvalue { i64, i64, i1 } %t116, i1 true, 2
  store { i64, i64, i1 } %t117, ptr %a118
  %t119 = call %S.std.prelude.RangeStepIter_i64_ @v_std.prelude.extend.Range_T.step_T_i64_(ptr %a118, i64 5)
  store %S.std.prelude.RangeStepIter_i64_ %t119, ptr %a120
  br label %loop.cond.47
loop.cond.47:
  br label %loop.body.50
loop.body.50:
  %t121 = call { i1, i64 } @v_std.prelude.Iterator.RangeStepIter_T.next_T_i64_(ptr %a120)
  store { i1, i64 } %t121, ptr %a122
  %t123 = load { i1, i64 }, ptr %a122
  %t125 = extractvalue { i1, i64 } %t123, 0
  %t124 = xor i1 %t125, true
  br i1 %t124, label %if.then.51, label %if.end.52
if.then.51:
  br label %loop.end.49
if.end.52:
  %t126 = load { i1, i64 }, ptr %a122
  %t127 = extractvalue { i1, i64 } %t126, 1
  store i64 %t127, ptr %a128
  %t129 = load i64, ptr %a128
  %t131 = call i64 @veles_i64_format(ptr %a130, i64 %t129)
  %t132 = insertvalue %str undef, ptr %a130, 0
  %t133 = insertvalue %str %t132, i64 %t131, 1
  %t135 = extractvalue %str { ptr @.str.17, i64 5 }, 0
  %t136 = extractvalue %str { ptr @.str.17, i64 5 }, 1
  %t137 = extractvalue %str %t133, 0
  %t138 = extractvalue %str %t133, 1
  call void @veles_string_concat(ptr %a134, ptr %t135, i64 %t136, ptr %t137, i64 %t138)
  %t139 = load %str, ptr %a134
  call void @v_std.io.println(%str %t139)
  br label %loop.post.48
loop.post.48:
  %t140 = load volatile i32, ptr @veles_stop_requested, align 4
  %t141 = icmp ne i32 %t140, 0
  br i1 %t141, label %safepoint.53, label %safepoint.on.54, !prof !{!"branch_weights", i32 1, i32 100000}
safepoint.53:
  call void @veles_gc_park()
  br label %safepoint.on.54
safepoint.on.54:
  br label %loop.cond.47
loop.end.49:
  %t142 = load i64, ptr %a1
  %t144 = call i64 @veles_i64_format(ptr %a143, i64 %t142)
  %t145 = insertvalue %str undef, ptr %a143, 0
  %t146 = insertvalue %str %t145, i64 %t144, 1
  %t147 = load i64, ptr %a87
  %t149 = call i64 @veles_i64_format(ptr %a148, i64 %t147)
  %t150 = insertvalue %str undef, ptr %a148, 0
  %t151 = insertvalue %str %t150, i64 %t149, 1
  %t152 = load i64, ptr %a101
  %t154 = call i64 @veles_i64_format(ptr %a153, i64 %t152)
  %t155 = insertvalue %str undef, ptr %a153, 0
  %t156 = insertvalue %str %t155, i64 %t154, 1
  %t157 = call %str @v_main.classify(i64 -2)
  %t158 = call %str @v_main.classify(i64 0)
  %t159 = call %str @v_main.classify(i64 4)
  %t160 = call %str @v_main.digits(i64 3)
  %t161 = call %str @v_main.digits(i64 9)
  %t164 = getelementptr [15 x %str], ptr %a163, i64 0, i64 0
  store %str %t146, ptr %t164
  %t165 = getelementptr [15 x %str], ptr %a163, i64 0, i64 1
  store %str { ptr @.str.18, i64 1 }, ptr %t165
  %t166 = getelementptr [15 x %str], ptr %a163, i64 0, i64 2
  store %str %t151, ptr %t166
  %t167 = getelementptr [15 x %str], ptr %a163, i64 0, i64 3
  store %str { ptr @.str.18, i64 1 }, ptr %t167
  %t168 = getelementptr [15 x %str], ptr %a163, i64 0, i64 4
  store %str %t156, ptr %t168
  %t169 = getelementptr [15 x %str], ptr %a163, i64 0, i64 5
  store %str { ptr @.str.18, i64 1 }, ptr %t169
  %t170 = getelementptr [15 x %str], ptr %a163, i64 0, i64 6
  store %str %t157, ptr %t170
  %t171 = getelementptr [15 x %str], ptr %a163, i64 0, i64 7
  store %str { ptr @.str.18, i64 1 }, ptr %t171
  %t172 = getelementptr [15 x %str], ptr %a163, i64 0, i64 8
  store %str %t158, ptr %t172
  %t173 = getelementptr [15 x %str], ptr %a163, i64 0, i64 9
  store %str { ptr @.str.18, i64 1 }, ptr %t173
  %t174 = getelementptr [15 x %str], ptr %a163, i64 0, i64 10
  store %str %t159, ptr %t174
  %t175 = getelementptr [15 x %str], ptr %a163, i64 0, i64 11
  store %str { ptr @.str.18, i64 1 }, ptr %t175
  %t176 = getelementptr [15 x %str], ptr %a163, i64 0, i64 12
  store %str %t160, ptr %t176
  %t177 = getelementptr [15 x %str], ptr %a163, i64 0, i64 13
  store %str { ptr @.str.18, i64 1 }, ptr %t177
  %t178 = getelementptr [15 x %str], ptr %a163, i64 0, i64 14
  store %str %t161, ptr %t178
  call void @veles_string_concat_n(ptr %a162, ptr %a163, i64 15)
  %t179 = load %str, ptr %a162
  call void @v_std.io.println(%str %t179)
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
