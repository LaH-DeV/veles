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
  %a36 = alloca [21 x i8]
  %a43 = alloca %str
  %a44 = alloca [15 x %str]
  %a66 = alloca %str
  %a78 = alloca %str
  %a82 = alloca %str
  %a83 = alloca [7 x %str]
  %a96 = alloca %S.std.prelude.StringBuilder
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
  %t32 = icmp ult i64 0, %t31
  br i1 %t32, label %byte.ok.1, label %byte.bad.2, !prof !{!"branch_weights", i32 2000, i32 1}
byte.bad.2:
  call i8 @veles_string_byte_at(ptr %t30, i64 %t31, i64 0)
  unreachable
byte.ok.1:
  %t33 = getelementptr inbounds i8, ptr %t30, i64 0
  %t34 = load i8, ptr %t33
  %t35 = zext i8 %t34 to i64
  %t37 = call i64 @veles_u64_format(ptr %a36, i64 %t35)
  %t38 = insertvalue %str undef, ptr %a36, 0
  %t39 = insertvalue %str %t38, i64 %t37, 1
  %t40 = extractvalue %str { ptr @.str.4, i64 21 }, 0
  %t41 = extractvalue %str { ptr @.str.4, i64 21 }, 1
  call void @veles_call_push(ptr %t40)
  %t42 = call %str @v_std.prelude.extend.string.toUpper(ptr %a1)
  call void @veles_call_pop()
  %t45 = getelementptr [15 x %str], ptr %a44, i64 0, i64 0
  store %str %t9, ptr %t45
  %t46 = getelementptr [15 x %str], ptr %a44, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t46
  %t47 = getelementptr [15 x %str], ptr %a44, i64 0, i64 2
  store %str %t14, ptr %t47
  %t48 = getelementptr [15 x %str], ptr %a44, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t48
  %t49 = getelementptr [15 x %str], ptr %a44, i64 0, i64 4
  store %str %t17, ptr %t49
  %t50 = getelementptr [15 x %str], ptr %a44, i64 0, i64 5
  store %str { ptr @.str.5, i64 1 }, ptr %t50
  %t51 = getelementptr [15 x %str], ptr %a44, i64 0, i64 6
  store %str %t20, ptr %t51
  %t52 = getelementptr [15 x %str], ptr %a44, i64 0, i64 7
  store %str { ptr @.str.5, i64 1 }, ptr %t52
  %t53 = getelementptr [15 x %str], ptr %a44, i64 0, i64 8
  store %str %t22, ptr %t53
  %t54 = getelementptr [15 x %str], ptr %a44, i64 0, i64 9
  store %str { ptr @.str.5, i64 1 }, ptr %t54
  %t55 = getelementptr [15 x %str], ptr %a44, i64 0, i64 10
  store %str %t28, ptr %t55
  %t56 = getelementptr [15 x %str], ptr %a44, i64 0, i64 11
  store %str { ptr @.str.5, i64 1 }, ptr %t56
  %t57 = getelementptr [15 x %str], ptr %a44, i64 0, i64 12
  store %str %t39, ptr %t57
  %t58 = getelementptr [15 x %str], ptr %a44, i64 0, i64 13
  store %str { ptr @.str.5, i64 1 }, ptr %t58
  %t59 = getelementptr [15 x %str], ptr %a44, i64 0, i64 14
  store %str %t42, ptr %t59
  call void @veles_string_concat_n(ptr %a43, ptr %a44, i64 15)
  %t60 = load %str, ptr %a43
  %t61 = extractvalue %str { ptr @.str.6, i64 20 }, 0
  %t62 = extractvalue %str { ptr @.str.6, i64 20 }, 1
  call void @veles_call_push(ptr %t61)
  call void @v_std.io.println(%str %t60)
  call void @veles_call_pop()
  %t63 = extractvalue %str { ptr @.str.7, i64 18 }, 0
  %t64 = extractvalue %str { ptr @.str.7, i64 18 }, 1
  call void @veles_call_push(ptr %t63)
  %t65 = call %str @v_std.prelude.extend.string.trim(ptr %a8)
  call void @veles_call_pop()
  store %str %t65, ptr %a66
  %t67 = extractvalue %str { ptr @.str.8, i64 19 }, 0
  %t68 = extractvalue %str { ptr @.str.8, i64 19 }, 1
  call void @veles_call_push(ptr %t67)
  %t69 = call ptr @v_std.prelude.extend.string.split(ptr %a66, %str { ptr @.str.9, i64 1 })
  call void @veles_call_pop()
  %t70 = call %str @show.List_string_(ptr %t69)
  %t71 = extractvalue %str { ptr @.str.10, i64 21 }, 0
  %t72 = extractvalue %str { ptr @.str.10, i64 21 }, 1
  call void @veles_call_push(ptr %t71)
  %t73 = call %str @v_std.prelude.extend.string.replace(ptr %a1, %str { ptr @.str.11, i64 1 }, %str { ptr @.str.12, i64 1 })
  call void @veles_call_pop()
  %t74 = extractvalue %str { ptr @.str.13, i64 21 }, 0
  %t75 = extractvalue %str { ptr @.str.13, i64 21 }, 1
  call void @veles_call_push(ptr %t74)
  %t76 = call { i1, i64 } @v_std.prelude.extend.string.indexOf(ptr %a1, %str { ptr @.str.14, i64 1 }, i64 0)
  call void @veles_call_pop()
  %t77 = call %str @show.T_i64_N({ i1, i64 } %t76)
  store %str { ptr @.str.15, i64 2 }, ptr %a78
  %t79 = extractvalue %str { ptr @.str.16, i64 20 }, 0
  %t80 = extractvalue %str { ptr @.str.16, i64 20 }, 1
  call void @veles_call_push(ptr %t79)
  %t81 = call %str @v_std.prelude.extend.string.repeat(ptr %a78, i64 3)
  call void @veles_call_pop()
  %t84 = getelementptr [7 x %str], ptr %a83, i64 0, i64 0
  store %str %t70, ptr %t84
  %t85 = getelementptr [7 x %str], ptr %a83, i64 0, i64 1
  store %str { ptr @.str.5, i64 1 }, ptr %t85
  %t86 = getelementptr [7 x %str], ptr %a83, i64 0, i64 2
  store %str %t73, ptr %t86
  %t87 = getelementptr [7 x %str], ptr %a83, i64 0, i64 3
  store %str { ptr @.str.5, i64 1 }, ptr %t87
  %t88 = getelementptr [7 x %str], ptr %a83, i64 0, i64 4
  store %str %t77, ptr %t88
  %t89 = getelementptr [7 x %str], ptr %a83, i64 0, i64 5
  store %str { ptr @.str.5, i64 1 }, ptr %t89
  %t90 = getelementptr [7 x %str], ptr %a83, i64 0, i64 6
  store %str %t81, ptr %t90
  call void @veles_string_concat_n_shared(ptr %a82, ptr %a83, i64 7)
  %t91 = load %str, ptr %a82
  %t92 = extractvalue %str { ptr @.str.17, i64 20 }, 0
  %t93 = extractvalue %str { ptr @.str.17, i64 20 }, 1
  call void @veles_call_push(ptr %t92)
  call void @v_std.io.println(%str %t91)
  call void @veles_call_pop()
  %t94 = call ptr @veles_list_new(ptr @adesc.u8, i64 0)
  %t95 = insertvalue %S.std.prelude.StringBuilder undef, ptr %t94, 0
  store %S.std.prelude.StringBuilder %t95, ptr %a96
  %t97 = extractvalue %str { ptr @.str.18, i64 19 }, 0
  %t98 = extractvalue %str { ptr @.str.18, i64 19 }, 1
  call void @veles_call_push(ptr %t97)
  call void @v_std.prelude.StringBuilder.append(ptr %a96, %str { ptr @.str.19, i64 1 })
  call void @veles_call_pop()
  %t99 = extractvalue %str { ptr @.str.20, i64 23 }, 0
  %t100 = extractvalue %str { ptr @.str.20, i64 23 }, 1
  call void @veles_call_push(ptr %t99)
  call void @v_std.prelude.StringBuilder.appendByte(ptr %a96, i8 121)
  call void @veles_call_pop()
  %t101 = extractvalue %str { ptr @.str.21, i64 22 }, 0
  %t102 = extractvalue %str { ptr @.str.21, i64 22 }, 1
  call void @veles_call_push(ptr %t101)
  %t103 = call %str @v_std.prelude.StringBuilder.toString(ptr %a96)
  call void @veles_call_pop()
  %t104 = extractvalue %str { ptr @.str.22, i64 20 }, 0
  %t105 = extractvalue %str { ptr @.str.22, i64 20 }, 1
  call void @veles_call_push(ptr %t104)
  call void @v_std.io.println(%str %t103)
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
