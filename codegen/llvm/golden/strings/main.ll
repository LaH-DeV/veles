define void @v_main.main() {
entry:
  %a1 = alloca %str
  %a2 = alloca i64
  %a3 = alloca double
  %a4 = alloca i1
  %a7 = alloca { i64, %str }
  %a8 = alloca %str
  %a11 = alloca [21 x i8]
  %a16 = alloca %str
  %a19 = alloca %str
  %a25 = alloca [21 x i8]
  %a34 = alloca [21 x i8]
  %a39 = alloca %str
  %a40 = alloca [15 x %str]
  %a58 = alloca %str
  %a63 = alloca [21 x i8]
  %a67 = alloca %str
  %a69 = alloca %str
  %a70 = alloca [7 x %str]
  %a81 = alloca %S.std.prelude.StringBuilder
  store %str { ptr @.str.1, i64 5 }, ptr %a1
  store i64 42, ptr %a2
  store double 0x4004000000000000, ptr %a3
  store i1 true, ptr %a4
  %t5 = insertvalue { i64, %str } undef, i64 1, 0
  %t6 = insertvalue { i64, %str } %t5, %str { ptr @.str.2, i64 3 }, 1
  store { i64, %str } %t6, ptr %a7
  store %str { ptr @.str.3, i64 9 }, ptr %a8
  %t9 = load %str, ptr %a1
  %t10 = load i64, ptr %a2
  %t12 = call i64 @veles_i64_format(ptr %a11, i64 %t10)
  %t13 = insertvalue %str undef, ptr %a11, 0
  %t14 = insertvalue %str %t13, i64 %t12, 1
  %t15 = load double, ptr %a3
  call void @veles_f64_to_string(ptr %a16, double %t15)
  %t17 = load %str, ptr %a16
  %t18 = load i1, ptr %a4
  call void @veles_bool_to_string(ptr %a19, i1 zeroext %t18)
  %t20 = load %str, ptr %a19
  %t21 = load { i64, %str }, ptr %a7
  %t22 = call %str @show.T_i64_string_({ i64, %str } %t21)
  %t23 = load %str, ptr %a1
  %t24 = extractvalue %str %t23, 1
  %t26 = call i64 @veles_i64_format(ptr %a25, i64 %t24)
  %t27 = insertvalue %str undef, ptr %a25, 0
  %t28 = insertvalue %str %t27, i64 %t26, 1
  %t29 = load %str, ptr %a1
  %t30 = extractvalue %str %t29, 0
  %t31 = extractvalue %str %t29, 1
  %t32 = call i8 @veles_string_byte_at(ptr %t30, i64 %t31, i64 0)
  %t33 = zext i8 %t32 to i64
  %t35 = call i64 @veles_u64_format(ptr %a34, i64 %t33)
  %t36 = insertvalue %str undef, ptr %a34, 0
  %t37 = insertvalue %str %t36, i64 %t35, 1
  %t38 = call %str @v_std.prelude.extend.string.toUpper(ptr %a1)
  %t41 = getelementptr [15 x %str], ptr %a40, i64 0, i64 0
  store %str %t9, ptr %t41
  %t42 = getelementptr [15 x %str], ptr %a40, i64 0, i64 1
  store %str { ptr @.str.4, i64 1 }, ptr %t42
  %t43 = getelementptr [15 x %str], ptr %a40, i64 0, i64 2
  store %str %t14, ptr %t43
  %t44 = getelementptr [15 x %str], ptr %a40, i64 0, i64 3
  store %str { ptr @.str.4, i64 1 }, ptr %t44
  %t45 = getelementptr [15 x %str], ptr %a40, i64 0, i64 4
  store %str %t17, ptr %t45
  %t46 = getelementptr [15 x %str], ptr %a40, i64 0, i64 5
  store %str { ptr @.str.4, i64 1 }, ptr %t46
  %t47 = getelementptr [15 x %str], ptr %a40, i64 0, i64 6
  store %str %t20, ptr %t47
  %t48 = getelementptr [15 x %str], ptr %a40, i64 0, i64 7
  store %str { ptr @.str.4, i64 1 }, ptr %t48
  %t49 = getelementptr [15 x %str], ptr %a40, i64 0, i64 8
  store %str %t22, ptr %t49
  %t50 = getelementptr [15 x %str], ptr %a40, i64 0, i64 9
  store %str { ptr @.str.4, i64 1 }, ptr %t50
  %t51 = getelementptr [15 x %str], ptr %a40, i64 0, i64 10
  store %str %t28, ptr %t51
  %t52 = getelementptr [15 x %str], ptr %a40, i64 0, i64 11
  store %str { ptr @.str.4, i64 1 }, ptr %t52
  %t53 = getelementptr [15 x %str], ptr %a40, i64 0, i64 12
  store %str %t37, ptr %t53
  %t54 = getelementptr [15 x %str], ptr %a40, i64 0, i64 13
  store %str { ptr @.str.4, i64 1 }, ptr %t54
  %t55 = getelementptr [15 x %str], ptr %a40, i64 0, i64 14
  store %str %t38, ptr %t55
  call void @veles_string_concat_n(ptr %a39, ptr %a40, i64 15)
  %t56 = load %str, ptr %a39
  call void @v_std.io.println(%str %t56)
  %t57 = call %str @v_std.prelude.extend.string.trim(ptr %a8)
  store %str %t57, ptr %a58
  %t59 = call ptr @v_std.prelude.extend.string.split(ptr %a58, %str { ptr @.str.5, i64 1 })
  %t60 = call %str @show.List_string_(ptr %t59)
  %t61 = call %str @v_std.prelude.extend.string.replace(ptr %a1, %str { ptr @.str.6, i64 1 }, %str { ptr @.str.7, i64 1 })
  %t62 = call i64 @v_std.prelude.extend.string.indexOf(ptr %a1, %str { ptr @.str.8, i64 1 }, i64 0)
  %t64 = call i64 @veles_i64_format(ptr %a63, i64 %t62)
  %t65 = insertvalue %str undef, ptr %a63, 0
  %t66 = insertvalue %str %t65, i64 %t64, 1
  store %str { ptr @.str.9, i64 2 }, ptr %a67
  %t68 = call %str @v_std.prelude.extend.string.repeat(ptr %a67, i64 3)
  %t71 = getelementptr [7 x %str], ptr %a70, i64 0, i64 0
  store %str %t60, ptr %t71
  %t72 = getelementptr [7 x %str], ptr %a70, i64 0, i64 1
  store %str { ptr @.str.4, i64 1 }, ptr %t72
  %t73 = getelementptr [7 x %str], ptr %a70, i64 0, i64 2
  store %str %t61, ptr %t73
  %t74 = getelementptr [7 x %str], ptr %a70, i64 0, i64 3
  store %str { ptr @.str.4, i64 1 }, ptr %t74
  %t75 = getelementptr [7 x %str], ptr %a70, i64 0, i64 4
  store %str %t66, ptr %t75
  %t76 = getelementptr [7 x %str], ptr %a70, i64 0, i64 5
  store %str { ptr @.str.4, i64 1 }, ptr %t76
  %t77 = getelementptr [7 x %str], ptr %a70, i64 0, i64 6
  store %str %t68, ptr %t77
  call void @veles_string_concat_n(ptr %a69, ptr %a70, i64 7)
  %t78 = load %str, ptr %a69
  call void @v_std.io.println(%str %t78)
  %t79 = call ptr @veles_list_new(ptr @adesc.u8, i64 0)
  %t80 = insertvalue %S.std.prelude.StringBuilder undef, ptr %t79, 0
  store %S.std.prelude.StringBuilder %t80, ptr %a81
  call void @v_std.prelude.StringBuilder.append(ptr %a81, %str { ptr @.str.10, i64 1 })
  call void @v_std.prelude.StringBuilder.appendByte(ptr %a81, i8 121)
  %t82 = call %str @v_std.prelude.StringBuilder.toString(ptr %a81)
  call void @v_std.io.println(%str %t82)
  ret void
}

@.str.1 = private unnamed_addr constant [6 x i8] c"veles\00"
@.str.2 = private unnamed_addr constant [4 x i8] c"two\00"
@.str.3 = private unnamed_addr constant [10 x i8] c"  a,b,c  \00"
@.str.4 = private unnamed_addr constant [2 x i8] c" \00"
@.str.5 = private unnamed_addr constant [2 x i8] c",\00"
@.str.6 = private unnamed_addr constant [2 x i8] c"e\00"
@.str.7 = private unnamed_addr constant [2 x i8] c"E\00"
@.str.8 = private unnamed_addr constant [2 x i8] c"l\00"
@.str.9 = private unnamed_addr constant [3 x i8] c"ab\00"
@.str.10 = private unnamed_addr constant [2 x i8] c"x\00"
