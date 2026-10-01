define i32 @v_main.rotl(i32 %p1, i64 %p2) {
entry:
  %a1 = alloca i32
  %a2 = alloca i64
  store i32 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i32, ptr %a1
  %t4 = load i64, ptr %a2
  %t5 = trunc i64 %t4 to i32
  %t6 = call i32 @llvm.fshl.i32(i32 %t3, i32 %t3, i32 %t5)
  ret i32 %t6
}

define i16 @v_main.rotr(i16 %p1, i64 %p2) {
entry:
  %a1 = alloca i16
  %a2 = alloca i64
  store i16 %p1, ptr %a1
  store i64 %p2, ptr %a2
  %t3 = load i16, ptr %a1
  %t4 = load i64, ptr %a2
  %t5 = trunc i64 %t4 to i16
  %t6 = call i16 @llvm.fshr.i16(i16 %t3, i16 %t3, i16 %t5)
  ret i16 %t6
}

define i64 @v_main.swap(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = call i64 @llvm.bswap.i64(i64 %t2)
  ret i64 %t3
}

define i8 @v_main.swap8(i8 %p1) {
entry:
  %a1 = alloca i8
  store i8 %p1, ptr %a1
  %t2 = load i8, ptr %a1
  ret i8 %t2
}

define i64 @v_main.rev(i64 %p1) {
entry:
  %a1 = alloca i64
  store i64 %p1, ptr %a1
  %t2 = load i64, ptr %a1
  %t3 = call i64 @llvm.bitreverse.i64(i64 %t2)
  ret i64 %t3
}

define double @v_main.sign(double %p1, double %p2) {
entry:
  %a1 = alloca double
  %a2 = alloca double
  store double %p1, ptr %a1
  store double %p2, ptr %a2
  %t3 = load double, ptr %a1
  %t4 = load double, ptr %a2
  %t5 = call double @llvm.copysign.f64(double %t3, double %t4)
  ret double %t5
}

define i1 @v_main.negative(float %p1) {
entry:
  %a1 = alloca float
  store float %p1, ptr %a1
  %t2 = load float, ptr %a1
  %t3 = bitcast float %t2 to i32
  %t4 = icmp slt i32 %t3, 0
  ret i1 %t4
}

define void @v_main.main() {
entry:
  %a1 = alloca i32
  %a2 = alloca i16
  %a3 = alloca i8
  %a7 = alloca [21 x i8]
  %a14 = alloca [21 x i8]
  %a19 = alloca [21 x i8]
  %a26 = alloca [21 x i8]
  %a31 = alloca [21 x i8]
  %a37 = alloca %str
  %a41 = alloca %str
  %a43 = alloca %str
  %a44 = alloca [13 x %str]
  store i32 2147483649, ptr %a1
  store i16 3, ptr %a2
  store i8 7, ptr %a3
  %t4 = load i32, ptr %a1
  %t5 = call i32 @v_main.rotl(i32 %t4, i64 4)
  %t6 = zext i32 %t5 to i64
  %t8 = call i64 @veles_u64_format(ptr %a7, i64 %t6)
  %t9 = insertvalue %str undef, ptr %a7, 0
  %t10 = insertvalue %str %t9, i64 %t8, 1
  %t11 = load i16, ptr %a2
  %t12 = call i16 @v_main.rotr(i16 %t11, i64 1)
  %t13 = sext i16 %t12 to i64
  %t15 = call i64 @veles_i64_format(ptr %a14, i64 %t13)
  %t16 = insertvalue %str undef, ptr %a14, 0
  %t17 = insertvalue %str %t16, i64 %t15, 1
  %t18 = call i64 @v_main.swap(i64 1)
  %t20 = call i64 @veles_u64_format(ptr %a19, i64 %t18)
  %t21 = insertvalue %str undef, ptr %a19, 0
  %t22 = insertvalue %str %t21, i64 %t20, 1
  %t23 = load i8, ptr %a3
  %t24 = call i8 @v_main.swap8(i8 %t23)
  %t25 = zext i8 %t24 to i64
  %t27 = call i64 @veles_u64_format(ptr %a26, i64 %t25)
  %t28 = insertvalue %str undef, ptr %a26, 0
  %t29 = insertvalue %str %t28, i64 %t27, 1
  %t30 = call i64 @v_main.rev(i64 1)
  %t32 = call i64 @veles_i64_format(ptr %a31, i64 %t30)
  %t33 = insertvalue %str undef, ptr %a31, 0
  %t34 = insertvalue %str %t33, i64 %t32, 1
  %t35 = fneg double 0x0000000000000000
  %t36 = call double @v_main.sign(double 0x4000000000000000, double %t35)
  call void @veles_f64_to_string(ptr %a37, double %t36)
  %t38 = load %str, ptr %a37
  %t39 = fneg float 0x0000000000000000
  %t40 = call i1 @v_main.negative(float %t39)
  call void @veles_bool_to_string(ptr %a41, i1 zeroext %t40)
  %t42 = load %str, ptr %a41
  %t45 = getelementptr [13 x %str], ptr %a44, i64 0, i64 0
  store %str %t10, ptr %t45
  %t46 = getelementptr [13 x %str], ptr %a44, i64 0, i64 1
  store %str { ptr @.str.1, i64 1 }, ptr %t46
  %t47 = getelementptr [13 x %str], ptr %a44, i64 0, i64 2
  store %str %t17, ptr %t47
  %t48 = getelementptr [13 x %str], ptr %a44, i64 0, i64 3
  store %str { ptr @.str.1, i64 1 }, ptr %t48
  %t49 = getelementptr [13 x %str], ptr %a44, i64 0, i64 4
  store %str %t22, ptr %t49
  %t50 = getelementptr [13 x %str], ptr %a44, i64 0, i64 5
  store %str { ptr @.str.1, i64 1 }, ptr %t50
  %t51 = getelementptr [13 x %str], ptr %a44, i64 0, i64 6
  store %str %t29, ptr %t51
  %t52 = getelementptr [13 x %str], ptr %a44, i64 0, i64 7
  store %str { ptr @.str.1, i64 1 }, ptr %t52
  %t53 = getelementptr [13 x %str], ptr %a44, i64 0, i64 8
  store %str %t34, ptr %t53
  %t54 = getelementptr [13 x %str], ptr %a44, i64 0, i64 9
  store %str { ptr @.str.1, i64 1 }, ptr %t54
  %t55 = getelementptr [13 x %str], ptr %a44, i64 0, i64 10
  store %str %t38, ptr %t55
  %t56 = getelementptr [13 x %str], ptr %a44, i64 0, i64 11
  store %str { ptr @.str.1, i64 1 }, ptr %t56
  %t57 = getelementptr [13 x %str], ptr %a44, i64 0, i64 12
  store %str %t42, ptr %t57
  call void @veles_string_concat_n(ptr %a43, ptr %a44, i64 13)
  %t58 = load %str, ptr %a43
  call void @v_std.io.println(%str %t58)
  ret void
}

@.str.1 = private unnamed_addr constant [2 x i8] c" \00"
