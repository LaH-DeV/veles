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
  %a20 = alloca i64
  %a23 = alloca i8
  %a26 = alloca float
  %a29 = alloca double
  %a32 = alloca float
  %a34 = alloca %str
  %a41 = alloca %str
  %a44 = alloca %str
  %a51 = alloca %str
  %a57 = alloca %str
  %a60 = alloca %str
  %a66 = alloca %str
  %a72 = alloca %str
  %a75 = alloca %str
  %a82 = alloca %str
  %a88 = alloca %str
  %a91 = alloca %str
  %a97 = alloca %str
  %a103 = alloca %str
  %a106 = alloca %str
  %a112 = alloca %str
  %a118 = alloca %str
  %a121 = alloca %str
  %a128 = alloca %str
  %a134 = alloca %str
  %a137 = alloca %str
  %a143 = alloca %str
  %a149 = alloca %str
  %a152 = alloca %str
  %a158 = alloca %str
  %a164 = alloca %str
  %a167 = alloca %str
  %a173 = alloca %str
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
  store i64 %t19, ptr %a20
  %t21 = load double, ptr %a2
  %t22 = call i8 @llvm.fptoui.sat.i8.f64(double %t21)
  store i8 %t22, ptr %a23
  %t24 = load double, ptr %a2
  %t25 = fptrunc double %t24 to float
  store float %t25, ptr %a26
  %t27 = load float, ptr %a26
  %t28 = fpext float %t27 to double
  store double %t28, ptr %a29
  %t30 = load i64, ptr %a1
  %t31 = uitofp i64 %t30 to float
  store float %t31, ptr %a32
  %t33 = load i8, ptr %a5
  %t35 = zext i8 %t33 to i64
  call void @veles_u64_to_string(ptr %a34, i64 %t35)
  %t36 = load %str, ptr %a34
  %t37 = extractvalue %str %t36, 0
  %t38 = extractvalue %str %t36, 1
  %t39 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t40 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a41, ptr %t37, i64 %t38, ptr %t39, i64 %t40)
  %t42 = load %str, ptr %a41
  %t43 = load i16, ptr %a8
  %t45 = sext i16 %t43 to i64
  call void @veles_i64_to_string(ptr %a44, i64 %t45)
  %t46 = load %str, ptr %a44
  %t47 = extractvalue %str %t42, 0
  %t48 = extractvalue %str %t42, 1
  %t49 = extractvalue %str %t46, 0
  %t50 = extractvalue %str %t46, 1
  call void @veles_string_concat(ptr %a51, ptr %t47, i64 %t48, ptr %t49, i64 %t50)
  %t52 = load %str, ptr %a51
  %t53 = extractvalue %str %t52, 0
  %t54 = extractvalue %str %t52, 1
  %t55 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t56 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a57, ptr %t53, i64 %t54, ptr %t55, i64 %t56)
  %t58 = load %str, ptr %a57
  %t59 = load i64, ptr %a11
  call void @veles_i64_to_string(ptr %a60, i64 %t59)
  %t61 = load %str, ptr %a60
  %t62 = extractvalue %str %t58, 0
  %t63 = extractvalue %str %t58, 1
  %t64 = extractvalue %str %t61, 0
  %t65 = extractvalue %str %t61, 1
  call void @veles_string_concat(ptr %a66, ptr %t62, i64 %t63, ptr %t64, i64 %t65)
  %t67 = load %str, ptr %a66
  %t68 = extractvalue %str %t67, 0
  %t69 = extractvalue %str %t67, 1
  %t70 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t71 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a72, ptr %t68, i64 %t69, ptr %t70, i64 %t71)
  %t73 = load %str, ptr %a72
  %t74 = load i32, ptr %a14
  %t76 = zext i32 %t74 to i64
  call void @veles_u64_to_string(ptr %a75, i64 %t76)
  %t77 = load %str, ptr %a75
  %t78 = extractvalue %str %t73, 0
  %t79 = extractvalue %str %t73, 1
  %t80 = extractvalue %str %t77, 0
  %t81 = extractvalue %str %t77, 1
  call void @veles_string_concat(ptr %a82, ptr %t78, i64 %t79, ptr %t80, i64 %t81)
  %t83 = load %str, ptr %a82
  %t84 = extractvalue %str %t83, 0
  %t85 = extractvalue %str %t83, 1
  %t86 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t87 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a88, ptr %t84, i64 %t85, ptr %t86, i64 %t87)
  %t89 = load %str, ptr %a88
  %t90 = load double, ptr %a17
  call void @veles_f64_to_string(ptr %a91, double %t90)
  %t92 = load %str, ptr %a91
  %t93 = extractvalue %str %t89, 0
  %t94 = extractvalue %str %t89, 1
  %t95 = extractvalue %str %t92, 0
  %t96 = extractvalue %str %t92, 1
  call void @veles_string_concat(ptr %a97, ptr %t93, i64 %t94, ptr %t95, i64 %t96)
  %t98 = load %str, ptr %a97
  %t99 = extractvalue %str %t98, 0
  %t100 = extractvalue %str %t98, 1
  %t101 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t102 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a103, ptr %t99, i64 %t100, ptr %t101, i64 %t102)
  %t104 = load %str, ptr %a103
  %t105 = load i64, ptr %a20
  call void @veles_i64_to_string(ptr %a106, i64 %t105)
  %t107 = load %str, ptr %a106
  %t108 = extractvalue %str %t104, 0
  %t109 = extractvalue %str %t104, 1
  %t110 = extractvalue %str %t107, 0
  %t111 = extractvalue %str %t107, 1
  call void @veles_string_concat(ptr %a112, ptr %t108, i64 %t109, ptr %t110, i64 %t111)
  %t113 = load %str, ptr %a112
  %t114 = extractvalue %str %t113, 0
  %t115 = extractvalue %str %t113, 1
  %t116 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t117 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a118, ptr %t114, i64 %t115, ptr %t116, i64 %t117)
  %t119 = load %str, ptr %a118
  %t120 = load i8, ptr %a23
  %t122 = zext i8 %t120 to i64
  call void @veles_u64_to_string(ptr %a121, i64 %t122)
  %t123 = load %str, ptr %a121
  %t124 = extractvalue %str %t119, 0
  %t125 = extractvalue %str %t119, 1
  %t126 = extractvalue %str %t123, 0
  %t127 = extractvalue %str %t123, 1
  call void @veles_string_concat(ptr %a128, ptr %t124, i64 %t125, ptr %t126, i64 %t127)
  %t129 = load %str, ptr %a128
  %t130 = extractvalue %str %t129, 0
  %t131 = extractvalue %str %t129, 1
  %t132 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t133 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a134, ptr %t130, i64 %t131, ptr %t132, i64 %t133)
  %t135 = load %str, ptr %a134
  %t136 = load float, ptr %a26
  call void @veles_f32_to_string(ptr %a137, float %t136)
  %t138 = load %str, ptr %a137
  %t139 = extractvalue %str %t135, 0
  %t140 = extractvalue %str %t135, 1
  %t141 = extractvalue %str %t138, 0
  %t142 = extractvalue %str %t138, 1
  call void @veles_string_concat(ptr %a143, ptr %t139, i64 %t140, ptr %t141, i64 %t142)
  %t144 = load %str, ptr %a143
  %t145 = extractvalue %str %t144, 0
  %t146 = extractvalue %str %t144, 1
  %t147 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t148 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a149, ptr %t145, i64 %t146, ptr %t147, i64 %t148)
  %t150 = load %str, ptr %a149
  %t151 = load double, ptr %a29
  call void @veles_f64_to_string(ptr %a152, double %t151)
  %t153 = load %str, ptr %a152
  %t154 = extractvalue %str %t150, 0
  %t155 = extractvalue %str %t150, 1
  %t156 = extractvalue %str %t153, 0
  %t157 = extractvalue %str %t153, 1
  call void @veles_string_concat(ptr %a158, ptr %t154, i64 %t155, ptr %t156, i64 %t157)
  %t159 = load %str, ptr %a158
  %t160 = extractvalue %str %t159, 0
  %t161 = extractvalue %str %t159, 1
  %t162 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t163 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a164, ptr %t160, i64 %t161, ptr %t162, i64 %t163)
  %t165 = load %str, ptr %a164
  %t166 = load float, ptr %a32
  call void @veles_f32_to_string(ptr %a167, float %t166)
  %t168 = load %str, ptr %a167
  %t169 = extractvalue %str %t165, 0
  %t170 = extractvalue %str %t165, 1
  %t171 = extractvalue %str %t168, 0
  %t172 = extractvalue %str %t168, 1
  call void @veles_string_concat(ptr %a173, ptr %t169, i64 %t170, ptr %t171, i64 %t172)
  %t174 = load %str, ptr %a173
  call void @v_std.io.println(%str %t174)
  ret void
}

define void @v_main.main() {
entry:
  %a2 = alloca %str
  %a8 = alloca %str
  %a11 = alloca %str
  %a17 = alloca %str
  %a23 = alloca %str
  %a26 = alloca %str
  %a33 = alloca %str
  %a39 = alloca %str
  %a42 = alloca %str
  %a49 = alloca %str
  %a52 = alloca %str
  %a58 = alloca %str
  %a61 = alloca %str
  %a67 = alloca %str
  %a73 = alloca %str
  %a76 = alloca %str
  %a82 = alloca %str
  %a88 = alloca %str
  %a91 = alloca %str
  %a97 = alloca %str
  %t1 = call i64 @v_main.ints(i64 40, i64 3)
  call void @veles_i64_to_string(ptr %a2, i64 %t1)
  %t3 = load %str, ptr %a2
  %t4 = extractvalue %str %t3, 0
  %t5 = extractvalue %str %t3, 1
  %t6 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t7 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a8, ptr %t4, i64 %t5, ptr %t6, i64 %t7)
  %t9 = load %str, ptr %a8
  %t10 = call i64 @v_main.wrapping(i64 9, i64 4)
  call void @veles_i64_to_string(ptr %a11, i64 %t10)
  %t12 = load %str, ptr %a11
  %t13 = extractvalue %str %t9, 0
  %t14 = extractvalue %str %t9, 1
  %t15 = extractvalue %str %t12, 0
  %t16 = extractvalue %str %t12, 1
  call void @veles_string_concat(ptr %a17, ptr %t13, i64 %t14, ptr %t15, i64 %t16)
  %t18 = load %str, ptr %a17
  %t19 = extractvalue %str %t18, 0
  %t20 = extractvalue %str %t18, 1
  %t21 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t22 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a23, ptr %t19, i64 %t20, ptr %t21, i64 %t22)
  %t24 = load %str, ptr %a23
  %t25 = call i32 @v_main.narrow(i32 7, i32 2)
  %t27 = sext i32 %t25 to i64
  call void @veles_i64_to_string(ptr %a26, i64 %t27)
  %t28 = load %str, ptr %a26
  %t29 = extractvalue %str %t24, 0
  %t30 = extractvalue %str %t24, 1
  %t31 = extractvalue %str %t28, 0
  %t32 = extractvalue %str %t28, 1
  call void @veles_string_concat(ptr %a33, ptr %t29, i64 %t30, ptr %t31, i64 %t32)
  %t34 = load %str, ptr %a33
  %t35 = extractvalue %str %t34, 0
  %t36 = extractvalue %str %t34, 1
  %t37 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t38 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a39, ptr %t35, i64 %t36, ptr %t37, i64 %t38)
  %t40 = load %str, ptr %a39
  %t41 = call i32 @v_main.unsigned(i32 17, i32 5)
  %t43 = zext i32 %t41 to i64
  call void @veles_u64_to_string(ptr %a42, i64 %t43)
  %t44 = load %str, ptr %a42
  %t45 = extractvalue %str %t40, 0
  %t46 = extractvalue %str %t40, 1
  %t47 = extractvalue %str %t44, 0
  %t48 = extractvalue %str %t44, 1
  call void @veles_string_concat(ptr %a49, ptr %t45, i64 %t46, ptr %t47, i64 %t48)
  %t50 = load %str, ptr %a49
  call void @v_std.io.println(%str %t50)
  %t51 = call i64 @v_main.bits(i64 12, i64 10)
  call void @veles_u64_to_string(ptr %a52, i64 %t51)
  %t53 = load %str, ptr %a52
  %t54 = extractvalue %str %t53, 0
  %t55 = extractvalue %str %t53, 1
  %t56 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t57 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a58, ptr %t54, i64 %t55, ptr %t56, i64 %t57)
  %t59 = load %str, ptr %a58
  %t60 = call i64 @v_main.negate(i64 5)
  call void @veles_i64_to_string(ptr %a61, i64 %t60)
  %t62 = load %str, ptr %a61
  %t63 = extractvalue %str %t59, 0
  %t64 = extractvalue %str %t59, 1
  %t65 = extractvalue %str %t62, 0
  %t66 = extractvalue %str %t62, 1
  call void @veles_string_concat(ptr %a67, ptr %t63, i64 %t64, ptr %t65, i64 %t66)
  %t68 = load %str, ptr %a67
  %t69 = extractvalue %str %t68, 0
  %t70 = extractvalue %str %t68, 1
  %t71 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t72 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a73, ptr %t69, i64 %t70, ptr %t71, i64 %t72)
  %t74 = load %str, ptr %a73
  %t75 = call double @v_main.floats(double 0x401E000000000000, double 0x4000000000000000)
  call void @veles_f64_to_string(ptr %a76, double %t75)
  %t77 = load %str, ptr %a76
  %t78 = extractvalue %str %t74, 0
  %t79 = extractvalue %str %t74, 1
  %t80 = extractvalue %str %t77, 0
  %t81 = extractvalue %str %t77, 1
  call void @veles_string_concat(ptr %a82, ptr %t78, i64 %t79, ptr %t80, i64 %t81)
  %t83 = load %str, ptr %a82
  %t84 = extractvalue %str %t83, 0
  %t85 = extractvalue %str %t83, 1
  %t86 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t87 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a88, ptr %t84, i64 %t85, ptr %t86, i64 %t87)
  %t89 = load %str, ptr %a88
  %t90 = call float @v_main.single(float 0x3FF8000000000000, float 0x4000000000000000)
  call void @veles_f32_to_string(ptr %a91, float %t90)
  %t92 = load %str, ptr %a91
  %t93 = extractvalue %str %t89, 0
  %t94 = extractvalue %str %t89, 1
  %t95 = extractvalue %str %t92, 0
  %t96 = extractvalue %str %t92, 1
  call void @veles_string_concat(ptr %a97, ptr %t93, i64 %t94, ptr %t95, i64 %t96)
  %t98 = load %str, ptr %a97
  call void @v_std.io.println(%str %t98)
  call void @v_main.casts(i64 300, double 0x4415AF1D78B58C40)
  %t99 = fneg double 0x400D99999999999A
  call void @v_main.casts(i64 -1, double %t99)
  call void @v_main.saturating(i64 4611686018427387904, i8 100, i8 200, i32 -2147483648)
  call void @v_main.saturating(i64 3, i8 -5, i8 7, i32 11)
  ret void
}

define void @v_main.saturating(i64 %p1, i8 %p2, i8 %p3, i32 %p4) {
entry:
  %a1 = alloca i64
  %a2 = alloca i8
  %a3 = alloca i8
  %a4 = alloca i32
  %a13 = alloca %str
  %a19 = alloca %str
  %a29 = alloca %str
  %a35 = alloca %str
  %a41 = alloca %str
  %a51 = alloca %str
  %a58 = alloca %str
  %a64 = alloca %str
  %a71 = alloca %str
  %a78 = alloca %str
  %a84 = alloca %str
  %a94 = alloca %str
  %a101 = alloca %str
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
  call void @veles_i64_to_string(ptr %a13, i64 %t12)
  %t14 = load %str, ptr %a13
  %t15 = extractvalue %str %t14, 0
  %t16 = extractvalue %str %t14, 1
  %t17 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t18 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a19, ptr %t15, i64 %t16, ptr %t17, i64 %t18)
  %t20 = load %str, ptr %a19
  %t21 = load i64, ptr %a1
  %t22 = call { i64, i1 } @llvm.smul.with.overflow.i64(i64 %t21, i64 -3)
  %t23 = extractvalue { i64, i1 } %t22, 0
  %t24 = extractvalue { i64, i1 } %t22, 1
  %t25 = xor i64 %t21, -3
  %t26 = icmp slt i64 %t25, 0
  %t27 = select i1 %t26, i64 -9223372036854775808, i64 9223372036854775807
  %t28 = select i1 %t24, i64 %t27, i64 %t23
  call void @veles_i64_to_string(ptr %a29, i64 %t28)
  %t30 = load %str, ptr %a29
  %t31 = extractvalue %str %t20, 0
  %t32 = extractvalue %str %t20, 1
  %t33 = extractvalue %str %t30, 0
  %t34 = extractvalue %str %t30, 1
  call void @veles_string_concat(ptr %a35, ptr %t31, i64 %t32, ptr %t33, i64 %t34)
  %t36 = load %str, ptr %a35
  %t37 = extractvalue %str %t36, 0
  %t38 = extractvalue %str %t36, 1
  %t39 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t40 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a41, ptr %t37, i64 %t38, ptr %t39, i64 %t40)
  %t42 = load %str, ptr %a41
  %t43 = load i8, ptr %a2
  %t44 = call { i8, i1 } @llvm.smul.with.overflow.i8(i8 %t43, i8 -2)
  %t45 = extractvalue { i8, i1 } %t44, 0
  %t46 = extractvalue { i8, i1 } %t44, 1
  %t47 = xor i8 %t43, -2
  %t48 = icmp slt i8 %t47, 0
  %t49 = select i1 %t48, i8 -128, i8 127
  %t50 = select i1 %t46, i8 %t49, i8 %t45
  %t52 = sext i8 %t50 to i64
  call void @veles_i64_to_string(ptr %a51, i64 %t52)
  %t53 = load %str, ptr %a51
  %t54 = extractvalue %str %t42, 0
  %t55 = extractvalue %str %t42, 1
  %t56 = extractvalue %str %t53, 0
  %t57 = extractvalue %str %t53, 1
  call void @veles_string_concat(ptr %a58, ptr %t54, i64 %t55, ptr %t56, i64 %t57)
  %t59 = load %str, ptr %a58
  %t60 = extractvalue %str %t59, 0
  %t61 = extractvalue %str %t59, 1
  %t62 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t63 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a64, ptr %t60, i64 %t61, ptr %t62, i64 %t63)
  %t65 = load %str, ptr %a64
  %t66 = load i8, ptr %a3
  %t67 = call { i8, i1 } @llvm.umul.with.overflow.i8(i8 %t66, i8 2)
  %t68 = extractvalue { i8, i1 } %t67, 0
  %t69 = extractvalue { i8, i1 } %t67, 1
  %t70 = select i1 %t69, i8 -1, i8 %t68
  %t72 = zext i8 %t70 to i64
  call void @veles_u64_to_string(ptr %a71, i64 %t72)
  %t73 = load %str, ptr %a71
  %t74 = extractvalue %str %t65, 0
  %t75 = extractvalue %str %t65, 1
  %t76 = extractvalue %str %t73, 0
  %t77 = extractvalue %str %t73, 1
  call void @veles_string_concat(ptr %a78, ptr %t74, i64 %t75, ptr %t76, i64 %t77)
  %t79 = load %str, ptr %a78
  %t80 = extractvalue %str %t79, 0
  %t81 = extractvalue %str %t79, 1
  %t82 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t83 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a84, ptr %t80, i64 %t81, ptr %t82, i64 %t83)
  %t85 = load %str, ptr %a84
  %t86 = load i32, ptr %a4
  %t87 = call { i32, i1 } @llvm.smul.with.overflow.i32(i32 %t86, i32 -1)
  %t88 = extractvalue { i32, i1 } %t87, 0
  %t89 = extractvalue { i32, i1 } %t87, 1
  %t90 = xor i32 %t86, -1
  %t91 = icmp slt i32 %t90, 0
  %t92 = select i1 %t91, i32 -2147483648, i32 2147483647
  %t93 = select i1 %t89, i32 %t92, i32 %t88
  %t95 = sext i32 %t93 to i64
  call void @veles_i64_to_string(ptr %a94, i64 %t95)
  %t96 = load %str, ptr %a94
  %t97 = extractvalue %str %t85, 0
  %t98 = extractvalue %str %t85, 1
  %t99 = extractvalue %str %t96, 0
  %t100 = extractvalue %str %t96, 1
  call void @veles_string_concat(ptr %a101, ptr %t97, i64 %t98, ptr %t99, i64 %t100)
  %t102 = load %str, ptr %a101
  call void @v_std.io.println(%str %t102)
  ret void
}

@.str.1 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.2 = private unnamed_addr constant [13 x i8] c"main.vs:5:33\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:5:41\00"
@.str.4 = private unnamed_addr constant [13 x i8] c"main.vs:5:50\00"
@.str.5 = private unnamed_addr constant [17 x i8] c"division by zero\00"
@.str.6 = private unnamed_addr constant [13 x i8] c"main.vs:7:35\00"
@.str.7 = private unnamed_addr constant [13 x i8] c"main.vs:7:43\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:8:37\00"
@.str.9 = private unnamed_addr constant [13 x i8] c"main.vs:8:45\00"
@.str.10 = private unnamed_addr constant [2 x i8] c" \00"
