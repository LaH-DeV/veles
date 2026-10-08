define i64 @v_main.ints(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t6 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t3, i64 %t4)
  %t7 = extractvalue { i64, i1 } %t6, 0
  %t8 = extractvalue { i64, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t12 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  %t13 = load i64, ptr %a1
  %t14 = load i64, ptr %a2
  %t16 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t13, i64 %t14)
  %t17 = extractvalue { i64, i1 } %t16, 0
  %t18 = extractvalue { i64, i1 } %t16, 1
  br i1 %t18, label %overflow.3, label %arith.ok.4
overflow.3:
  %t19 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t20 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t21 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t22 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t19, i64 %t20, ptr %t21, i64 %t22)
  unreachable
arith.ok.4:
  %t23 = load i64, ptr %a2
  %t25 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t23, i64 1)
  %t26 = extractvalue { i64, i1 } %t25, 0
  %t27 = extractvalue { i64, i1 } %t25, 1
  br i1 %t27, label %overflow.5, label %arith.ok.6
overflow.5:
  %t28 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t29 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t30 = extractvalue %str { ptr @.str.4, i64 12 }, 0
  %t31 = extractvalue %str { ptr @.str.4, i64 12 }, 1
  call void @veles_panic_at(ptr %t28, i64 %t29, ptr %t30, i64 %t31)
  unreachable
arith.ok.6:
  %t33 = icmp eq i64 %t26, 0
  br i1 %t33, label %divzero.7, label %div.ok.8
divzero.7:
  %t34 = extractvalue %str { ptr @.str.5, i64 16 }, 0
  %t35 = extractvalue %str { ptr @.str.5, i64 16 }, 1
  %t36 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t37 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t34, i64 %t35, ptr %t36, i64 %t37)
  unreachable
div.ok.8:
  %t38 = icmp eq i64 %t17, -9223372036854775808
  %t39 = icmp eq i64 %t26, -1
  %t40 = and i1 %t38, %t39
  br i1 %t40, label %divof.9, label %div.ok.10
divof.9:
  %t41 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t42 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t43 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t44 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t41, i64 %t42, ptr %t43, i64 %t44)
  unreachable
div.ok.10:
  %t32 = sdiv i64 %t17, %t26
  %t46 = icmp eq i64 7, 0
  br i1 %t46, label %divzero.11, label %div.ok.12
divzero.11:
  %t47 = extractvalue %str { ptr @.str.5, i64 16 }, 0
  %t48 = extractvalue %str { ptr @.str.5, i64 16 }, 1
  %t49 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t50 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t47, i64 %t48, ptr %t49, i64 %t50)
  unreachable
div.ok.12:
  %t51 = icmp eq i64 %t32, -9223372036854775808
  %t52 = icmp eq i64 7, -1
  %t53 = and i1 %t51, %t52
  br i1 %t53, label %divof.13, label %div.ok.14
divof.13:
  %t54 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t55 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t56 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t57 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_panic_at(ptr %t54, i64 %t55, ptr %t56, i64 %t57)
  unreachable
div.ok.14:
  %t45 = srem i64 %t32, 7
  %t59 = call { i64, i1 } @llvm.ssub.with.overflow.i64(i64 %t7, i64 %t45)
  %t60 = extractvalue { i64, i1 } %t59, 0
  %t61 = extractvalue { i64, i1 } %t59, 1
  br i1 %t61, label %overflow.15, label %arith.ok.16
overflow.15:
  %t62 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t63 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t64 = extractvalue %str { ptr @.str.2, i64 12 }, 0
  %t65 = extractvalue %str { ptr @.str.2, i64 12 }, 1
  call void @veles_panic_at(ptr %t62, i64 %t63, ptr %t64, i64 %t65)
  unreachable
arith.ok.16:
  ret i64 %t60
}

define i64 @v_main.wrapping(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t5 = add i64 %t3, %t4
  %t6 = load i64, ptr %a1
  %t7 = load i64, ptr %a2
  %t8 = mul i64 %t6, %t7
  %t9 = sub i64 %t5, %t8
  ret i64 %t9
}

define i32 @v_main.narrow(i32 %p1, i32 %p2) {
entry:
  %a1 = alloca i32
  %a2 = alloca i32
  store i32 %p1, ptr %a1
  store i32 %p2, ptr %a2
  %t3 = load i32, ptr %a1
  %t4 = load i32, ptr %a2
  %t6 = call { i32, i1 } @llvm.smul.with.overflow.i32(i32 %t3, i32 %t4)
  %t7 = extractvalue { i32, i1 } %t6, 0
  %t8 = extractvalue { i32, i1 } %t6, 1
  br i1 %t8, label %overflow.1, label %arith.ok.2
overflow.1:
  %t9 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t10 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t11 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t12 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t9, i64 %t10, ptr %t11, i64 %t12)
  unreachable
arith.ok.2:
  %t13 = load i32, ptr %a1
  %t14 = load i32, ptr %a2
  %t16 = icmp eq i32 %t14, 0
  br i1 %t16, label %divzero.3, label %div.ok.4
divzero.3:
  %t17 = extractvalue %str { ptr @.str.5, i64 16 }, 0
  %t18 = extractvalue %str { ptr @.str.5, i64 16 }, 1
  %t19 = extractvalue %str { ptr @.str.7, i64 12 }, 0
  %t20 = extractvalue %str { ptr @.str.7, i64 12 }, 1
  call void @veles_panic_at(ptr %t17, i64 %t18, ptr %t19, i64 %t20)
  unreachable
div.ok.4:
  %t21 = icmp eq i32 %t13, -2147483648
  %t22 = icmp eq i32 %t14, -1
  %t23 = and i1 %t21, %t22
  br i1 %t23, label %divof.5, label %div.ok.6
divof.5:
  %t24 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t25 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t26 = extractvalue %str { ptr @.str.7, i64 12 }, 0
  %t27 = extractvalue %str { ptr @.str.7, i64 12 }, 1
  call void @veles_panic_at(ptr %t24, i64 %t25, ptr %t26, i64 %t27)
  unreachable
div.ok.6:
  %t15 = sdiv i32 %t13, %t14
  %t29 = call { i32, i1 } @llvm.ssub.with.overflow.i32(i32 %t7, i32 %t15)
  %t30 = extractvalue { i32, i1 } %t29, 0
  %t31 = extractvalue { i32, i1 } %t29, 1
  br i1 %t31, label %overflow.7, label %arith.ok.8
overflow.7:
  %t32 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t33 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t34 = extractvalue %str { ptr @.str.6, i64 12 }, 0
  %t35 = extractvalue %str { ptr @.str.6, i64 12 }, 1
  call void @veles_panic_at(ptr %t32, i64 %t33, ptr %t34, i64 %t35)
  unreachable
arith.ok.8:
  ret i32 %t30
}

define i32 @v_main.unsigned(i32 %p1, i32 %p2) {
entry:
  %a1 = alloca i32
  %a2 = alloca i32
  store i32 %p1, ptr %a1
  store i32 %p2, ptr %a2
  %t3 = load i32, ptr %a1
  %t4 = load i32, ptr %a2
  %t6 = icmp eq i32 %t4, 0
  br i1 %t6, label %divzero.1, label %div.ok.2
divzero.1:
  %t7 = extractvalue %str { ptr @.str.5, i64 16 }, 0
  %t8 = extractvalue %str { ptr @.str.5, i64 16 }, 1
  %t9 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t10 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t7, i64 %t8, ptr %t9, i64 %t10)
  unreachable
div.ok.2:
  %t5 = udiv i32 %t3, %t4
  %t11 = load i32, ptr %a1
  %t12 = load i32, ptr %a2
  %t14 = icmp eq i32 %t12, 0
  br i1 %t14, label %divzero.3, label %div.ok.4
divzero.3:
  %t15 = extractvalue %str { ptr @.str.5, i64 16 }, 0
  %t16 = extractvalue %str { ptr @.str.5, i64 16 }, 1
  %t17 = extractvalue %str { ptr @.str.9, i64 12 }, 0
  %t18 = extractvalue %str { ptr @.str.9, i64 12 }, 1
  call void @veles_panic_at(ptr %t15, i64 %t16, ptr %t17, i64 %t18)
  unreachable
div.ok.4:
  %t13 = urem i32 %t11, %t12
  %t20 = call { i32, i1 } @llvm.uadd.with.overflow.i32(i32 %t5, i32 %t13)
  %t21 = extractvalue { i32, i1 } %t20, 0
  %t22 = extractvalue { i32, i1 } %t20, 1
  br i1 %t22, label %overflow.5, label %arith.ok.6
overflow.5:
  %t23 = extractvalue %str { ptr @.str.1, i64 16 }, 0
  %t24 = extractvalue %str { ptr @.str.1, i64 16 }, 1
  %t25 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t26 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t23, i64 %t24, ptr %t25, i64 %t26)
  unreachable
arith.ok.6:
  ret i32 %t21
}

define i64 @v_main.bits(i64 %p1, i64 %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca i64
  store i64 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = load i64, ptr %a2
  %t5 = and i64 %t3, %t4
  %t6 = load i64, ptr %a1
  %t7 = load i64, ptr %a2
  %t8 = xor i64 %t6, %t7
  %t10 = icmp uge i64 3, 64
  %t11 = select i1 %t10, i64 0, i64 3
  %t12 = shl i64 %t8, %t11
  %t13 = select i1 %t10, i64 0, i64 %t12
  %t15 = icmp uge i64 1, 64
  %t16 = select i1 %t15, i64 0, i64 1
  %t17 = lshr i64 %t13, %t16
  %t18 = select i1 %t15, i64 0, i64 %t17
  %t19 = or i64 %t5, %t18
  ret i64 %t19
}

define i64 @v_main.negate(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = xor i64 %t2, -1
  ret i64 %t3
}

define double @v_main.floats(double %p1, double %p2) {
entry:
  %a1 = alloca double
  %a2 = alloca double
  store double %p1, ptr %a1
  store double %p2, ptr %a2
  %t3 = load double, ptr %a1
  %t4 = load double, ptr %a2
  %t5 = fmul double %t3, %t4
  %t6 = load double, ptr %a1
  %t7 = load double, ptr %a2
  %t8 = fdiv double %t6, %t7
  %t9 = fadd double %t5, %t8
  ret double %t9
}

define float @v_main.single(float %p1, float %p2) {
entry:
  %a1 = alloca float
  %a2 = alloca float
  store float %p1, ptr %a1
  store float %p2, ptr %a2
  %t3 = load float, ptr %a1
  %t4 = load float, ptr %a2
  %t5 = fmul float %t3, %t4
  ret float %t5
}

define void @v_main.casts(i64 %p1, double %p2) {
entry:
  %a1 = alloca i64
  %a2 = alloca double
  %a5 = alloca i8
  %a8 = alloca i16
  %a11 = alloca i64
  %a14 = alloca i32
  %a17 = alloca double
  %a25 = alloca { i1, i64 }
  %a33 = alloca { i1, i8 }
  %a36 = alloca float
  %a39 = alloca double
  %a42 = alloca float
  %a45 = alloca [21 x i8]
  %a51 = alloca [21 x i8]
  %a56 = alloca [21 x i8]
  %a62 = alloca [21 x i8]
  %a67 = alloca %str
  %a74 = alloca %str
  %a77 = alloca %str
  %a80 = alloca %str
  %a82 = alloca %str
  %a83 = alloca [19 x %str]
  store i64 %p1, ptr %a1
  store double %p2, ptr %a2
  %t3 = load i64, ptr %a1
  %t4 = trunc i64 %t3 to i8
  store i8 %t4, ptr %a5
  %t6 = load i64, ptr %a1
  %t7 = trunc i64 %t6 to i16
  store i16 %t7, ptr %a8
  %t9 = load i8, ptr %a5
  %t10 = zext i8 %t9 to i64
  store i64 %t10, ptr %a11
  %t12 = load i16, ptr %a8
  %t13 = sext i16 %t12 to i32
  store i32 %t13, ptr %a14
  %t15 = load i64, ptr %a1
  %t16 = sitofp i64 %t15 to double
  store double %t16, ptr %a17
  %t18 = load double, ptr %a2
  %t19 = call i64 @llvm.fptosi.sat.i64.f64(double %t18)
  %t20 = fcmp oge double %t18, -9223372036854775808.0
  %t21 = fcmp olt double %t18, 9223372036854775808.0
  %t22 = and i1 %t20, %t21
  %t23 = insertvalue { i1, i64 } undef, i1 %t22, 0
  %t24 = insertvalue { i1, i64 } %t23, i64 %t19, 1
  store { i1, i64 } %t24, ptr %a25
  %t26 = load double, ptr %a2
  %t27 = call i8 @llvm.fptoui.sat.i8.f64(double %t26)
  %t28 = fcmp ogt double %t26, -1.0
  %t29 = fcmp olt double %t26, 256.0
  %t30 = and i1 %t28, %t29
  %t31 = insertvalue { i1, i8 } undef, i1 %t30, 0
  %t32 = insertvalue { i1, i8 } %t31, i8 %t27, 1
  store { i1, i8 } %t32, ptr %a33
  %t34 = load double, ptr %a2
  %t35 = fptrunc double %t34 to float
  store float %t35, ptr %a36
  %t37 = load float, ptr %a36
  %t38 = fpext float %t37 to double
  store double %t38, ptr %a39
  %t40 = load i64, ptr %a1
  %t41 = uitofp i64 %t40 to float
  store float %t41, ptr %a42
  %t43 = load i8, ptr %a5
  %t44 = zext i8 %t43 to i64
  %t46 = call i64 @veles_u64_format(ptr %a45, i64 %t44)
  %t47 = insertvalue %str undef, ptr %a45, 0
  %t48 = insertvalue %str %t47, i64 %t46, 1
  %t49 = load i16, ptr %a8
  %t50 = sext i16 %t49 to i64
  %t52 = call i64 @veles_i64_format(ptr %a51, i64 %t50)
  %t53 = insertvalue %str undef, ptr %a51, 0
  %t54 = insertvalue %str %t53, i64 %t52, 1
  %t55 = load i64, ptr %a11
  %t57 = call i64 @veles_i64_format(ptr %a56, i64 %t55)
  %t58 = insertvalue %str undef, ptr %a56, 0
  %t59 = insertvalue %str %t58, i64 %t57, 1
  %t60 = load i32, ptr %a14
  %t61 = zext i32 %t60 to i64
  %t63 = call i64 @veles_u64_format(ptr %a62, i64 %t61)
  %t64 = insertvalue %str undef, ptr %a62, 0
  %t65 = insertvalue %str %t64, i64 %t63, 1
  %t66 = load double, ptr %a17
  call void @veles_f64_to_string(ptr %a67, double %t66)
  %t68 = load %str, ptr %a67
  %t69 = load { i1, i64 }, ptr %a25
  %t70 = call %str @show.T_i64_N({ i1, i64 } %t69)
  %t71 = load { i1, i8 }, ptr %a33
  %t72 = call %str @show.T_u8_N({ i1, i8 } %t71)
  %t73 = load float, ptr %a36
  call void @veles_f32_to_string(ptr %a74, float %t73)
  %t75 = load %str, ptr %a74
  %t76 = load double, ptr %a39
  call void @veles_f64_to_string(ptr %a77, double %t76)
  %t78 = load %str, ptr %a77
  %t79 = load float, ptr %a42
  call void @veles_f32_to_string(ptr %a80, float %t79)
  %t81 = load %str, ptr %a80
  %t84 = getelementptr [19 x %str], ptr %a83, i64 0, i64 0
  store %str %t48, ptr %t84
  %t85 = getelementptr [19 x %str], ptr %a83, i64 0, i64 1
  store %str { ptr @.str.10, i64 1 }, ptr %t85
  %t86 = getelementptr [19 x %str], ptr %a83, i64 0, i64 2
  store %str %t54, ptr %t86
  %t87 = getelementptr [19 x %str], ptr %a83, i64 0, i64 3
  store %str { ptr @.str.10, i64 1 }, ptr %t87
  %t88 = getelementptr [19 x %str], ptr %a83, i64 0, i64 4
  store %str %t59, ptr %t88
  %t89 = getelementptr [19 x %str], ptr %a83, i64 0, i64 5
  store %str { ptr @.str.10, i64 1 }, ptr %t89
  %t90 = getelementptr [19 x %str], ptr %a83, i64 0, i64 6
  store %str %t65, ptr %t90
  %t91 = getelementptr [19 x %str], ptr %a83, i64 0, i64 7
  store %str { ptr @.str.10, i64 1 }, ptr %t91
  %t92 = getelementptr [19 x %str], ptr %a83, i64 0, i64 8
  store %str %t68, ptr %t92
  %t93 = getelementptr [19 x %str], ptr %a83, i64 0, i64 9
  store %str { ptr @.str.10, i64 1 }, ptr %t93
  %t94 = getelementptr [19 x %str], ptr %a83, i64 0, i64 10
  store %str %t70, ptr %t94
  %t95 = getelementptr [19 x %str], ptr %a83, i64 0, i64 11
  store %str { ptr @.str.10, i64 1 }, ptr %t95
  %t96 = getelementptr [19 x %str], ptr %a83, i64 0, i64 12
  store %str %t72, ptr %t96
  %t97 = getelementptr [19 x %str], ptr %a83, i64 0, i64 13
  store %str { ptr @.str.10, i64 1 }, ptr %t97
  %t98 = getelementptr [19 x %str], ptr %a83, i64 0, i64 14
  store %str %t75, ptr %t98
  %t99 = getelementptr [19 x %str], ptr %a83, i64 0, i64 15
  store %str { ptr @.str.10, i64 1 }, ptr %t99
  %t100 = getelementptr [19 x %str], ptr %a83, i64 0, i64 16
  store %str %t78, ptr %t100
  %t101 = getelementptr [19 x %str], ptr %a83, i64 0, i64 17
  store %str { ptr @.str.10, i64 1 }, ptr %t101
  %t102 = getelementptr [19 x %str], ptr %a83, i64 0, i64 18
  store %str %t81, ptr %t102
  call void @veles_string_concat_n(ptr %a82, ptr %a83, i64 19)
  %t103 = load %str, ptr %a82
  %t104 = extractvalue %str { ptr @.str.11, i64 20 }, 0
  %t105 = extractvalue %str { ptr @.str.11, i64 20 }, 1
  call void @veles_call_push(ptr %t104)
  call void @v_std.io.println(%str %t103)
  call void @veles_call_pop()
  ret void
}

define void @v_main.main() {
entry:
  %a4 = alloca [21 x i8]
  %a11 = alloca [21 x i8]
  %a19 = alloca [21 x i8]
  %a27 = alloca [21 x i8]
  %a31 = alloca %str
  %a32 = alloca [7 x %str]
  %a46 = alloca [21 x i8]
  %a53 = alloca [21 x i8]
  %a60 = alloca %str
  %a65 = alloca %str
  %a67 = alloca %str
  %a68 = alloca [7 x %str]
  %t1 = extractvalue %str { ptr @.str.12, i64 18 }, 0
  %t2 = extractvalue %str { ptr @.str.12, i64 18 }, 1
  call void @veles_call_push(ptr %t1)
  %t3 = call i64 @v_main.ints(i64 40, i64 3)
  call void @veles_call_pop()
  %t5 = call i64 @veles_i64_format(ptr %a4, i64 %t3)
  %t6 = insertvalue %str undef, ptr %a4, 0
  %t7 = insertvalue %str %t6, i64 %t5, 1
  %t8 = extractvalue %str { ptr @.str.13, i64 22 }, 0
  %t9 = extractvalue %str { ptr @.str.13, i64 22 }, 1
  call void @veles_call_push(ptr %t8)
  %t10 = call i64 @v_main.wrapping(i64 9, i64 4)
  call void @veles_call_pop()
  %t12 = call i64 @veles_i64_format(ptr %a11, i64 %t10)
  %t13 = insertvalue %str undef, ptr %a11, 0
  %t14 = insertvalue %str %t13, i64 %t12, 1
  %t15 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t16 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t15)
  %t17 = call i32 @v_main.narrow(i32 7, i32 2)
  call void @veles_call_pop()
  %t18 = sext i32 %t17 to i64
  %t20 = call i64 @veles_i64_format(ptr %a19, i64 %t18)
  %t21 = insertvalue %str undef, ptr %a19, 0
  %t22 = insertvalue %str %t21, i64 %t20, 1
  %t23 = extractvalue %str { ptr @.str.15, i64 22 }, 0
  %t24 = extractvalue %str { ptr @.str.15, i64 22 }, 1
  call void @veles_call_push(ptr %t23)
  %t25 = call i32 @v_main.unsigned(i32 17, i32 5)
  call void @veles_call_pop()
  %t26 = zext i32 %t25 to i64
  %t28 = call i64 @veles_u64_format(ptr %a27, i64 %t26)
  %t29 = insertvalue %str undef, ptr %a27, 0
  %t30 = insertvalue %str %t29, i64 %t28, 1
  %t33 = getelementptr [7 x %str], ptr %a32, i64 0, i64 0
  store %str %t7, ptr %t33
  %t34 = getelementptr [7 x %str], ptr %a32, i64 0, i64 1
  store %str { ptr @.str.10, i64 1 }, ptr %t34
  %t35 = getelementptr [7 x %str], ptr %a32, i64 0, i64 2
  store %str %t14, ptr %t35
  %t36 = getelementptr [7 x %str], ptr %a32, i64 0, i64 3
  store %str { ptr @.str.10, i64 1 }, ptr %t36
  %t37 = getelementptr [7 x %str], ptr %a32, i64 0, i64 4
  store %str %t22, ptr %t37
  %t38 = getelementptr [7 x %str], ptr %a32, i64 0, i64 5
  store %str { ptr @.str.10, i64 1 }, ptr %t38
  %t39 = getelementptr [7 x %str], ptr %a32, i64 0, i64 6
  store %str %t30, ptr %t39
  call void @veles_string_concat_n(ptr %a31, ptr %a32, i64 7)
  %t40 = load %str, ptr %a31
  %t41 = extractvalue %str { ptr @.str.16, i64 20 }, 0
  %t42 = extractvalue %str { ptr @.str.16, i64 20 }, 1
  call void @veles_call_push(ptr %t41)
  call void @v_std.io.println(%str %t40)
  call void @veles_call_pop()
  %t43 = extractvalue %str { ptr @.str.17, i64 18 }, 0
  %t44 = extractvalue %str { ptr @.str.17, i64 18 }, 1
  call void @veles_call_push(ptr %t43)
  %t45 = call i64 @v_main.bits(i64 12, i64 10)
  call void @veles_call_pop()
  %t47 = call i64 @veles_u64_format(ptr %a46, i64 %t45)
  %t48 = insertvalue %str undef, ptr %a46, 0
  %t49 = insertvalue %str %t48, i64 %t47, 1
  %t50 = extractvalue %str { ptr @.str.18, i64 20 }, 0
  %t51 = extractvalue %str { ptr @.str.18, i64 20 }, 1
  call void @veles_call_push(ptr %t50)
  %t52 = call i64 @v_main.negate(i64 5)
  call void @veles_call_pop()
  %t54 = call i64 @veles_i64_format(ptr %a53, i64 %t52)
  %t55 = insertvalue %str undef, ptr %a53, 0
  %t56 = insertvalue %str %t55, i64 %t54, 1
  %t57 = extractvalue %str { ptr @.str.19, i64 20 }, 0
  %t58 = extractvalue %str { ptr @.str.19, i64 20 }, 1
  call void @veles_call_push(ptr %t57)
  %t59 = call double @v_main.floats(double 0x401E000000000000, double 0x4000000000000000)
  call void @veles_call_pop()
  call void @veles_f64_to_string(ptr %a60, double %t59)
  %t61 = load %str, ptr %a60
  %t62 = extractvalue %str { ptr @.str.20, i64 20 }, 0
  %t63 = extractvalue %str { ptr @.str.20, i64 20 }, 1
  call void @veles_call_push(ptr %t62)
  %t64 = call float @v_main.single(float 0x3FF8000000000000, float 0x4000000000000000)
  call void @veles_call_pop()
  call void @veles_f32_to_string(ptr %a65, float %t64)
  %t66 = load %str, ptr %a65
  %t69 = getelementptr [7 x %str], ptr %a68, i64 0, i64 0
  store %str %t49, ptr %t69
  %t70 = getelementptr [7 x %str], ptr %a68, i64 0, i64 1
  store %str { ptr @.str.10, i64 1 }, ptr %t70
  %t71 = getelementptr [7 x %str], ptr %a68, i64 0, i64 2
  store %str %t56, ptr %t71
  %t72 = getelementptr [7 x %str], ptr %a68, i64 0, i64 3
  store %str { ptr @.str.10, i64 1 }, ptr %t72
  %t73 = getelementptr [7 x %str], ptr %a68, i64 0, i64 4
  store %str %t61, ptr %t73
  %t74 = getelementptr [7 x %str], ptr %a68, i64 0, i64 5
  store %str { ptr @.str.10, i64 1 }, ptr %t74
  %t75 = getelementptr [7 x %str], ptr %a68, i64 0, i64 6
  store %str %t66, ptr %t75
  call void @veles_string_concat_n(ptr %a67, ptr %a68, i64 7)
  %t76 = load %str, ptr %a67
  %t77 = extractvalue %str { ptr @.str.21, i64 20 }, 0
  %t78 = extractvalue %str { ptr @.str.21, i64 20 }, 1
  call void @veles_call_push(ptr %t77)
  call void @v_std.io.println(%str %t76)
  call void @veles_call_pop()
  %t79 = extractvalue %str { ptr @.str.22, i64 18 }, 0
  %t80 = extractvalue %str { ptr @.str.22, i64 18 }, 1
  call void @veles_call_push(ptr %t79)
  call void @v_main.casts(i64 300, double 0x4415AF1D78B58C40)
  call void @veles_call_pop()
  %t81 = fneg double 0x400D99999999999A
  %t82 = extractvalue %str { ptr @.str.23, i64 18 }, 0
  %t83 = extractvalue %str { ptr @.str.23, i64 18 }, 1
  call void @veles_call_push(ptr %t82)
  call void @v_main.casts(i64 -1, double %t81)
  call void @veles_call_pop()
  %t84 = extractvalue %str { ptr @.str.24, i64 23 }, 0
  %t85 = extractvalue %str { ptr @.str.24, i64 23 }, 1
  call void @veles_call_push(ptr %t84)
  call void @v_main.saturating(i64 4611686018427387904, i8 100, i8 200, i32 -2147483648)
  call void @veles_call_pop()
  %t86 = extractvalue %str { ptr @.str.25, i64 23 }, 0
  %t87 = extractvalue %str { ptr @.str.25, i64 23 }, 1
  call void @veles_call_push(ptr %t86)
  call void @v_main.saturating(i64 3, i8 -5, i8 7, i32 11)
  call void @veles_call_pop()
  ret void
}

define void @v_main.saturating(i64 %p1, i8 %p2, i8 %p3, i32 %p4) {
entry:
  %a1 = alloca i64
  %a2 = alloca i8
  %a3 = alloca i8
  %a4 = alloca i32
  %a13 = alloca [21 x i8]
  %a25 = alloca [21 x i8]
  %a38 = alloca [21 x i8]
  %a48 = alloca [21 x i8]
  %a61 = alloca [21 x i8]
  %a65 = alloca %str
  %a66 = alloca [9 x %str]
  store i64 %p1, ptr %a1
  store i8 %p2, ptr %a2
  store i8 %p3, ptr %a3
  store i32 %p4, ptr %a4
  %t5 = load i64, ptr %a1
  %t6 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t5, i64 2)
  %t7 = extractvalue { i64, i1 } %t6, 0
  %t8 = extractvalue { i64, i1 } %t6, 1
  %t9 = xor i64 %t5, 2
  %t10 = icmp slt i64 %t9, 0
  %t11 = select i1 %t10, i64 -9223372036854775808, i64 9223372036854775807
  %t12 = select i1 %t8, i64 %t11, i64 %t7
  %t14 = call i64 @veles_i64_format(ptr %a13, i64 %t12)
  %t15 = insertvalue %str undef, ptr %a13, 0
  %t16 = insertvalue %str %t15, i64 %t14, 1
  %t17 = load i64, ptr %a1
  %t18 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t17, i64 -3)
  %t19 = extractvalue { i64, i1 } %t18, 0
  %t20 = extractvalue { i64, i1 } %t18, 1
  %t21 = xor i64 %t17, -3
  %t22 = icmp slt i64 %t21, 0
  %t23 = select i1 %t22, i64 -9223372036854775808, i64 9223372036854775807
  %t24 = select i1 %t20, i64 %t23, i64 %t19
  %t26 = call i64 @veles_i64_format(ptr %a25, i64 %t24)
  %t27 = insertvalue %str undef, ptr %a25, 0
  %t28 = insertvalue %str %t27, i64 %t26, 1
  %t29 = load i8, ptr %a2
  %t30 = call { i8, i1 } @llvm.smul.with.overflow.i8(i8 %t29, i8 -2)
  %t31 = extractvalue { i8, i1 } %t30, 0
  %t32 = extractvalue { i8, i1 } %t30, 1
  %t33 = xor i8 %t29, -2
  %t34 = icmp slt i8 %t33, 0
  %t35 = select i1 %t34, i8 -128, i8 127
  %t36 = select i1 %t32, i8 %t35, i8 %t31
  %t37 = sext i8 %t36 to i64
  %t39 = call i64 @veles_i64_format(ptr %a38, i64 %t37)
  %t40 = insertvalue %str undef, ptr %a38, 0
  %t41 = insertvalue %str %t40, i64 %t39, 1
  %t42 = load i8, ptr %a3
  %t43 = call { i8, i1 } @llvm.umul.with.overflow.i8(i8 %t42, i8 2)
  %t44 = extractvalue { i8, i1 } %t43, 0
  %t45 = extractvalue { i8, i1 } %t43, 1
  %t46 = select i1 %t45, i8 -1, i8 %t44
  %t47 = zext i8 %t46 to i64
  %t49 = call i64 @veles_u64_format(ptr %a48, i64 %t47)
  %t50 = insertvalue %str undef, ptr %a48, 0
  %t51 = insertvalue %str %t50, i64 %t49, 1
  %t52 = load i32, ptr %a4
  %t53 = call { i32, i1 } @llvm.smul.with.overflow.i32(i32 %t52, i32 -1)
  %t54 = extractvalue { i32, i1 } %t53, 0
  %t55 = extractvalue { i32, i1 } %t53, 1
  %t56 = xor i32 %t52, -1
  %t57 = icmp slt i32 %t56, 0
  %t58 = select i1 %t57, i32 -2147483648, i32 2147483647
  %t59 = select i1 %t55, i32 %t58, i32 %t54
  %t60 = sext i32 %t59 to i64
  %t62 = call i64 @veles_i64_format(ptr %a61, i64 %t60)
  %t63 = insertvalue %str undef, ptr %a61, 0
  %t64 = insertvalue %str %t63, i64 %t62, 1
  %t67 = getelementptr [9 x %str], ptr %a66, i64 0, i64 0
  store %str %t16, ptr %t67
  %t68 = getelementptr [9 x %str], ptr %a66, i64 0, i64 1
  store %str { ptr @.str.10, i64 1 }, ptr %t68
  %t69 = getelementptr [9 x %str], ptr %a66, i64 0, i64 2
  store %str %t28, ptr %t69
  %t70 = getelementptr [9 x %str], ptr %a66, i64 0, i64 3
  store %str { ptr @.str.10, i64 1 }, ptr %t70
  %t71 = getelementptr [9 x %str], ptr %a66, i64 0, i64 4
  store %str %t41, ptr %t71
  %t72 = getelementptr [9 x %str], ptr %a66, i64 0, i64 5
  store %str { ptr @.str.10, i64 1 }, ptr %t72
  %t73 = getelementptr [9 x %str], ptr %a66, i64 0, i64 6
  store %str %t51, ptr %t73
  %t74 = getelementptr [9 x %str], ptr %a66, i64 0, i64 7
  store %str { ptr @.str.10, i64 1 }, ptr %t74
  %t75 = getelementptr [9 x %str], ptr %a66, i64 0, i64 8
  store %str %t64, ptr %t75
  call void @veles_string_concat_n(ptr %a65, ptr %a66, i64 9)
  %t76 = load %str, ptr %a65
  %t77 = extractvalue %str { ptr @.str.26, i64 20 }, 0
  %t78 = extractvalue %str { ptr @.str.26, i64 20 }, 1
  call void @veles_call_push(ptr %t77)
  call void @v_std.io.println(%str %t76)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:5:34\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:5:42\00"
@.str.4 = private unnamed_addr constant [13 x i8] c"main.vs:5:51\00"
@.str.5 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.6 = private unnamed_addr constant [13 x i8] c"main.vs:7:36\00"
@.str.7 = private unnamed_addr constant [13 x i8] c"main.vs:7:44\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:8:38\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:8:46\00"
@.str.10 = private unnamed_addr constant [2 x i8] c" \00"
@.str.11 = private unnamed_addr constant [21 x i8] c"main.vs:25:3\00println\00"
@.str.12 = private unnamed_addr constant [19 x i8] c"main.vs:29:17\00ints\00"
@.str.13 = private unnamed_addr constant [23 x i8] c"main.vs:29:32\00wrapping\00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:29:50\00narrow\00"
@.str.15 = private unnamed_addr constant [23 x i8] c"main.vs:29:66\00unsigned\00"
@.str.16 = private unnamed_addr constant [21 x i8] c"main.vs:29:3\00println\00"
@.str.17 = private unnamed_addr constant [19 x i8] c"main.vs:30:17\00bits\00"
@.str.18 = private unnamed_addr constant [21 x i8] c"main.vs:30:33\00negate\00"
@.str.19 = private unnamed_addr constant [21 x i8] c"main.vs:30:46\00floats\00"
@.str.20 = private unnamed_addr constant [21 x i8] c"main.vs:30:66\00single\00"
@.str.21 = private unnamed_addr constant [21 x i8] c"main.vs:30:3\00println\00"
@.str.22 = private unnamed_addr constant [19 x i8] c"main.vs:31:3\00casts\00"
@.str.23 = private unnamed_addr constant [19 x i8] c"main.vs:32:3\00casts\00"
@.str.24 = private unnamed_addr constant [24 x i8] c"main.vs:33:3\00saturating\00"
@.str.25 = private unnamed_addr constant [24 x i8] c"main.vs:34:3\00saturating\00"
@.str.26 = private unnamed_addr constant [21 x i8] c"main.vs:38:3\00println\00"
