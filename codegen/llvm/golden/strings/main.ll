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
  %a41 = alloca %str
  %a42 = alloca [15 x %str]
  %a64 = alloca %str
  %a75 = alloca [21 x i8]
  %a79 = alloca %str
  %a83 = alloca %str
  %a84 = alloca [7 x %str]
  %a97 = alloca %S.std.prelude.StringBuilder
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
  %t38 = extractvalue %str { ptr @.str.4, i64 21 }, 0
  %t39 = extractvalue %str { ptr @.str.4, i64 21 }, 1
  call void @veles_call_push(ptr %t38)
  %t40 = call %str @v_std.prelude.extend.string.toUpper(ptr %a1)
  call void @veles_call_pop()
  %t43 = getelementptr [15 x %str], ptr %a42, i64 0, i64 0
  store %str %t9, ptr %t43
  %t44 = getelementptr [15 x %str], ptr %a42, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t44
  %t45 = getelementptr [15 x %str], ptr %a42, i64 0, i64 2
  store %str %t14, ptr %t45
  %t46 = getelementptr [15 x %str], ptr %a42, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t46
  %t47 = getelementptr [15 x %str], ptr %a42, i64 0, i64 4
  store %str %t17, ptr %t47
  %t48 = getelementptr [15 x %str], ptr %a42, i64 0, i64 5
  store %str { ptr @.str.5, i64 1 }, ptr %t48
  %t49 = getelementptr [15 x %str], ptr %a42, i64 0, i64 6
  store %str %t20, ptr %t49
  %t50 = getelementptr [15 x %str], ptr %a42, i64 0, i64 7
  store %str { ptr @.str.5, i64 1 }, ptr %t50
  %t51 = getelementptr [15 x %str], ptr %a42, i64 0, i64 8
  store %str %t22, ptr %t51
  %t52 = getelementptr [15 x %str], ptr %a42, i64 0, i64 9
  store %str { ptr @.str.5, i64 1 }, ptr %t52
  %t53 = getelementptr [15 x %str], ptr %a42, i64 0, i64 10
  store %str %t28, ptr %t53
  %t54 = getelementptr [15 x %str], ptr %a42, i64 0, i64 11
  store %str { ptr @.str.5, i64 1 }, ptr %t54
  %t55 = getelementptr [15 x %str], ptr %a42, i64 0, i64 12
  store %str %t37, ptr %t55
  %t56 = getelementptr [15 x %str], ptr %a42, i64 0, i64 13
  store %str { ptr @.str.5, i64 1 }, ptr %t56
  %t57 = getelementptr [15 x %str], ptr %a42, i64 0, i64 14
  store %str %t40, ptr %t57
  call void @veles_string_concat_n(ptr %a41, ptr %a42, i64 15)
  %t58 = load %str, ptr %a41
  %t59 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t60 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t59)
  call void @v_std.io.println(%str %t58)
  call void @veles_call_pop()
  %t61 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t62 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t61)
  %t63 = call %str @v_std.prelude.extend.string.trim(ptr %a8)
  call void @veles_call_pop()
  store %str %t63, ptr %a64
  %t65 = extractvalue %str { ptr @.str.8, i64 19 }, 0
  %t66 = extractvalue %str { ptr @.str.8, i64 19 }, 1
  call void @veles_call_push(ptr %t65)
  %t67 = call ptr @v_std.prelude.extend.string.split(ptr %a64, %str { ptr @.str.9, i64 1 })
  call void @veles_call_pop()
  %t68 = call %str @show.List_string_(ptr %t67)
  %t69 = extractvalue %str { ptr @.str.10, i64 21 }, 0
  %t70 = extractvalue %str { ptr @.str.10, i64 21 }, 1
  call void @veles_call_push(ptr %t69)
  %t71 = call %str @v_std.prelude.extend.string.replace(ptr %a1, %str { ptr @.str.11, i64 1 }, %str { ptr @.str.12, i64 1 })
  call void @veles_call_pop()
  %t72 = extractvalue %str { ptr @.str.13, i64 21 }, 0
  %t73 = extractvalue %str { ptr @.str.13, i64 21 }, 1
  call void @veles_call_push(ptr %t72)
  %t74 = call i64 @v_std.prelude.extend.string.indexOf(ptr %a1, %str { ptr @.str.14, i64 1 }, i64 0)
  call void @veles_call_pop()
  %t76 = call i64 @veles_i64_format(ptr %a75, i64 %t74)
  %t77 = insertvalue %str undef, ptr %a75, 0
  %t78 = insertvalue %str %t77, i64 %t76, 1
  store %str { ptr @.str.15, i64 2 }, ptr %a79
  %t80 = extractvalue %str { ptr @.str.16, i64 20 }, 0
  %t81 = extractvalue %str { ptr @.str.16, i64 20 }, 1
  call void @veles_call_push(ptr %t80)
  %t82 = call %str @v_std.prelude.extend.string.repeat(ptr %a79, i64 3)
  call void @veles_call_pop()
  %t85 = getelementptr [7 x %str], ptr %a84, i64 0, i64 0
  store %str %t68, ptr %t85
  %t86 = getelementptr [7 x %str], ptr %a84, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t86
  %t87 = getelementptr [7 x %str], ptr %a84, i64 0, i64 2
  store %str %t71, ptr %t87
  %t88 = getelementptr [7 x %str], ptr %a84, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t88
  %t89 = getelementptr [7 x %str], ptr %a84, i64 0, i64 4
  store %str %t78, ptr %t89
  %t90 = getelementptr [7 x %str], ptr %a84, i64 0, i64 5
  store %str { ptr @.str.5, i64 1 }, ptr %t90
  %t91 = getelementptr [7 x %str], ptr %a84, i64 0, i64 6
  store %str %t82, ptr %t91
  call void @veles_string_concat_n(ptr %a83, ptr %a84, i64 7)
  %t92 = load %str, ptr %a83
  %t93 = extractvalue %str { ptr @.str.17, i64 20 }, 0
  %t94 = extractvalue %str { ptr @.str.17, i64 20 }, 1
  call void @veles_call_push(ptr %t93)
  call void @v_std.io.println(%str %t92)
  call void @veles_call_pop()
  %t95 = call ptr @veles_list_new(ptr @adesc.u8, i64 0)
  %t96 = insertvalue %S.std.prelude.StringBuilder undef, ptr %t95, 0
  store %S.std.prelude.StringBuilder %t96, ptr %a97
  %t98 = extractvalue %str { ptr @.str.18, i64 19 }, 0
  %t99 = extractvalue %str { ptr @.str.18, i64 19 }, 1
  call void @veles_call_push(ptr %t98)
  call void @v_std.prelude.StringBuilder.append(ptr %a97, %str { ptr @.str.19, i64 1 })
  call void @veles_call_pop()
  %t100 = extractvalue %str { ptr @.str.20, i64 23 }, 0
  %t101 = extractvalue %str { ptr @.str.20, i64 23 }, 1
  call void @veles_call_push(ptr %t100)
  call void @v_std.prelude.StringBuilder.appendByte(ptr %a97, i8 121)
  call void @veles_call_pop()
  %t102 = extractvalue %str { ptr @.str.21, i64 22 }, 0
  %t103 = extractvalue %str { ptr @.str.21, i64 22 }, 1
  call void @veles_call_push(ptr %t102)
  %t104 = call %str @v_std.prelude.StringBuilder.toString(ptr %a97)
  call void @veles_call_pop()
  %t105 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t106 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t105)
  call void @v_std.io.println(%str %t104)
  call void @veles_call_pop()
  ret void
}

@.str.1 = private unnamed_addr constant [6 x i8] c"veles\00"
@.str.2 = private unnamed_addr constant [4 x i8] c"two\00"
@.str.3 = private unnamed_addr constant [10 x i8] c"  a,b,c  \00"
@.str.4 = private unnamed_addr constant [22 x i8] c"main.vs:12:68\00toUpper\00"
@.str.5 = private unnamed_addr constant [2 x i8] c" \00"
@.str.6 = private unnamed_addr constant [21 x i8] c"main.vs:12:3\00println\00"
@.str.7 = private unnamed_addr constant [19 x i8] c"main.vs:13:17\00trim\00"
@.str.8 = private unnamed_addr constant [20 x i8] c"main.vs:13:17\00split\00"
@.str.9 = private unnamed_addr constant [2 x i8] c",\00"
@.str.10 = private unnamed_addr constant [22 x i8] c"main.vs:13:40\00replace\00"
@.str.11 = private unnamed_addr constant [2 x i8] c"e\00"
@.str.12 = private unnamed_addr constant [2 x i8] c"E\00"
@.str.13 = private unnamed_addr constant [22 x i8] c"main.vs:13:66\00indexOf\00"
@.str.14 = private unnamed_addr constant [2 x i8] c"l\00"
@.str.15 = private unnamed_addr constant [3 x i8] c"ab\00"
@.str.16 = private unnamed_addr constant [21 x i8] c"main.vs:13:87\00repeat\00"
@.str.17 = private unnamed_addr constant [21 x i8] c"main.vs:13:3\00println\00"
@.str.18 = private unnamed_addr constant [20 x i8] c"main.vs:15:3\00append\00"
@.str.19 = private unnamed_addr constant [2 x i8] c"x\00"
@.str.20 = private unnamed_addr constant [24 x i8] c"main.vs:16:3\00appendByte\00"
@.str.21 = private unnamed_addr constant [23 x i8] c"main.vs:17:14\00toString\00"
@.str.22 = private unnamed_addr constant [21 x i8] c"main.vs:17:3\00println\00"
