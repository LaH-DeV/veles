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
  %a9 = alloca [21 x i8]
  %a18 = alloca [21 x i8]
  %a25 = alloca [21 x i8]
  %a34 = alloca [21 x i8]
  %a41 = alloca [21 x i8]
  %a49 = alloca %str
  %a55 = alloca %str
  %a57 = alloca %str
  %a58 = alloca [13 x %str]
  store i32 2147483649, ptr %a1
  store i16 3, ptr %a2
  store i8 7, ptr %a3
  %t4 = load i32, ptr %a1
  %t5 = extractvalue %str { ptr @.str.1, i64 18 }, 0
  %t6 = extractvalue %str { ptr @.str.1, i64 18 }, 1
  call void @veles_call_push(ptr %t5)
  %t7 = call i32 @v_main.rotl(i32 %t4, i64 4)
  call void @veles_call_pop()
  %t8 = zext i32 %t7 to i64
  %t10 = call i64 @veles_u64_format(ptr %a9, i64 %t8)
  %t11 = insertvalue %str undef, ptr %a9, 0
  %t12 = insertvalue %str %t11, i64 %t10, 1
  %t13 = load i16, ptr %a2
  %t14 = extractvalue %str { ptr @.str.2, i64 18 }, 0
  %t15 = extractvalue %str { ptr @.str.2, i64 18 }, 1
  call void @veles_call_push(ptr %t14)
  %t16 = call i16 @v_main.rotr(i16 %t13, i64 1)
  call void @veles_call_pop()
  %t17 = sext i16 %t16 to i64
  %t19 = call i64 @veles_i64_format(ptr %a18, i64 %t17)
  %t20 = insertvalue %str undef, ptr %a18, 0
  %t21 = insertvalue %str %t20, i64 %t19, 1
  %t22 = extractvalue %str { ptr @.str.3, i64 18 }, 0
  %t23 = extractvalue %str { ptr @.str.3, i64 18 }, 1
  call void @veles_call_push(ptr %t22)
  %t24 = call i64 @v_main.swap(i64 1)
  call void @veles_call_pop()
  %t26 = call i64 @veles_u64_format(ptr %a25, i64 %t24)
  %t27 = insertvalue %str undef, ptr %a25, 0
  %t28 = insertvalue %str %t27, i64 %t26, 1
  %t29 = load i8, ptr %a3
  %t30 = extractvalue %str { ptr @.str.4, i64 19 }, 0
  %t31 = extractvalue %str { ptr @.str.4, i64 19 }, 1
  call void @veles_call_push(ptr %t30)
  %t32 = call i8 @v_main.swap8(i8 %t29)
  call void @veles_call_pop()
  %t33 = zext i8 %t32 to i64
  %t35 = call i64 @veles_u64_format(ptr %a34, i64 %t33)
  %t36 = insertvalue %str undef, ptr %a34, 0
  %t37 = insertvalue %str %t36, i64 %t35, 1
  %t38 = extractvalue %str { ptr @.str.5, i64 17 }, 0
  %t39 = extractvalue %str { ptr @.str.5, i64 17 }, 1
  call void @veles_call_push(ptr %t38)
  %t40 = call i64 @v_main.rev(i64 1)
  call void @veles_call_pop()
  %t42 = call i64 @veles_i64_format(ptr %a41, i64 %t40)
  %t43 = insertvalue %str undef, ptr %a41, 0
  %t44 = insertvalue %str %t43, i64 %t42, 1
  %t45 = fneg double 0x0000000000000000
  %t46 = extractvalue %str { ptr @.str.6, i64 18 }, 0
  %t47 = extractvalue %str { ptr @.str.6, i64 18 }, 1
  call void @veles_call_push(ptr %t46)
  %t48 = call double @v_main.sign(double 0x4000000000000000, double %t45)
  call void @veles_call_pop()
  call void @veles_f64_to_string(ptr %a49, double %t48)
  %t50 = load %str, ptr %a49
  %t51 = fneg float 0x0000000000000000
  %t52 = extractvalue %str { ptr @.str.7, i64 22 }, 0
  %t53 = extractvalue %str { ptr @.str.7, i64 22 }, 1
  call void @veles_call_push(ptr %t52)
  %t54 = call i1 @v_main.negative(float %t51)
  call void @veles_call_pop()
  call void @veles_bool_to_string(ptr %a55, i1 zeroext %t54)
  %t56 = load %str, ptr %a55
  %t59 = getelementptr [13 x %str], ptr %a58, i64 0, i64 0
  store %str %t12, ptr %t59
  %t60 = getelementptr [13 x %str], ptr %a58, i64 0, i64 1
  store %str { ptr @.str.8, i64 1 }, ptr %t60
  %t61 = getelementptr [13 x %str], ptr %a58, i64 0, i64 2
  store %str %t21, ptr %t61
  %t62 = getelementptr [13 x %str], ptr %a58, i64 0, i64 3
  store %str { ptr @.str.8, i64 1 }, ptr %t62
  %t63 = getelementptr [13 x %str], ptr %a58, i64 0, i64 4
  store %str %t28, ptr %t63
  %t64 = getelementptr [13 x %str], ptr %a58, i64 0, i64 5
  store %str { ptr @.str.8, i64 1 }, ptr %t64
  %t65 = getelementptr [13 x %str], ptr %a58, i64 0, i64 6
  store %str %t37, ptr %t65
  %t66 = getelementptr [13 x %str], ptr %a58, i64 0, i64 7
  store %str { ptr @.str.8, i64 1 }, ptr %t66
  %t67 = getelementptr [13 x %str], ptr %a58, i64 0, i64 8
  store %str %t44, ptr %t67
  %t68 = getelementptr [13 x %str], ptr %a58, i64 0, i64 9
  store %str { ptr @.str.8, i64 1 }, ptr %t68
  %t69 = getelementptr [13 x %str], ptr %a58, i64 0, i64 10
  store %str %t50, ptr %t69
  %t70 = getelementptr [13 x %str], ptr %a58, i64 0, i64 11
  store %str { ptr @.str.8, i64 1 }, ptr %t70
  %t71 = getelementptr [13 x %str], ptr %a58, i64 0, i64 12
  store %str %t56, ptr %t71
  call void @veles_string_concat_n(ptr %a57, ptr %a58, i64 13)
  %t72 = load %str, ptr %a57
  %t73 = extractvalue %str { ptr @.str.9, i64 20 }, 0
  %t74 = extractvalue %str { ptr @.str.9, i64 20 }, 1
  call void @veles_call_push(ptr %t73)
  call void @v_std.io.println(%str %t72)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [19 x i8] c"main.vs:17:17\00rotl\00"
@.str.2 = private unnamed_addr constant [19 x i8] c"main.vs:17:31\00rotr\00"
@.str.3 = private unnamed_addr constant [19 x i8] c"main.vs:17:45\00swap\00"
@.str.4 = private unnamed_addr constant [20 x i8] c"main.vs:17:56\00swap8\00"
@.str.5 = private unnamed_addr constant [18 x i8] c"main.vs:17:68\00rev\00"
@.str.6 = private unnamed_addr constant [19 x i8] c"main.vs:17:78\00sign\00"
@.str.7 = private unnamed_addr constant [23 x i8] c"main.vs:17:97\00negative\00"
@.str.8 = private unnamed_addr constant [2 x i8] c" \00"
@.str.9 = private unnamed_addr constant [21 x i8] c"main.vs:17:3\00println\00"
