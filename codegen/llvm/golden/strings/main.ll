define void @v_main.main() {
entry:
  %a1 = alloca %str
  %a2 = alloca i64
  %a3 = alloca double
  %a4 = alloca i1
  %a7 = alloca { i64, %str }
  %a8 = alloca %str
  %a14 = alloca %str
  %a17 = alloca %str
  %a23 = alloca %str
  %a29 = alloca %str
  %a32 = alloca %str
  %a38 = alloca %str
  %a44 = alloca %str
  %a47 = alloca %str
  %a53 = alloca %str
  %a59 = alloca %str
  %a67 = alloca %str
  %a73 = alloca %str
  %a77 = alloca %str
  %a83 = alloca %str
  %a89 = alloca %str
  %a95 = alloca %str
  %a102 = alloca %str
  %a108 = alloca %str
  %a115 = alloca %str
  %a118 = alloca %str
  %a125 = alloca %str
  %a132 = alloca %str
  %a138 = alloca %str
  %a141 = alloca %str
  %a147 = alloca %str
  %a153 = alloca %str
  %a155 = alloca %str
  %a161 = alloca %str
  %a164 = alloca %S.std.prelude.StringBuilder
  store %str { ptr @.str.1, i64 5 }, ptr %a1
  store i64 42, ptr %a2
  store double 0x4004000000000000, ptr %a3
  store i1 true, ptr %a4
  %t5 = insertvalue { i64, %str } undef, i64 1, 0
  %t6 = insertvalue { i64, %str } %t5, %str { ptr @.str.2, i64 3 }, 1
  store { i64, %str } %t6, ptr %a7
  store %str { ptr @.str.3, i64 9 }, ptr %a8
  %t9 = load %str, ptr %a1
  %t10 = extractvalue %str %t9, 0
  %t11 = extractvalue %str %t9, 1
  %t12 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t13 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a14, ptr %t10, i64 %t11, ptr %t12, i64 %t13)
  %t15 = load %str, ptr %a14
  %t16 = load i64, ptr %a2
  call void @veles_i64_to_string(ptr %a17, i64 %t16)
  %t18 = load %str, ptr %a17
  %t19 = extractvalue %str %t15, 0
  %t20 = extractvalue %str %t15, 1
  %t21 = extractvalue %str %t18, 0
  %t22 = extractvalue %str %t18, 1
  call void @veles_string_concat(ptr %a23, ptr %t19, i64 %t20, ptr %t21, i64 %t22)
  %t24 = load %str, ptr %a23
  %t25 = extractvalue %str %t24, 0
  %t26 = extractvalue %str %t24, 1
  %t27 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t28 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a29, ptr %t25, i64 %t26, ptr %t27, i64 %t28)
  %t30 = load %str, ptr %a29
  %t31 = load double, ptr %a3
  call void @veles_f64_to_string(ptr %a32, double %t31)
  %t33 = load %str, ptr %a32
  %t34 = extractvalue %str %t30, 0
  %t35 = extractvalue %str %t30, 1
  %t36 = extractvalue %str %t33, 0
  %t37 = extractvalue %str %t33, 1
  call void @veles_string_concat(ptr %a38, ptr %t34, i64 %t35, ptr %t36, i64 %t37)
  %t39 = load %str, ptr %a38
  %t40 = extractvalue %str %t39, 0
  %t41 = extractvalue %str %t39, 1
  %t42 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t43 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a44, ptr %t40, i64 %t41, ptr %t42, i64 %t43)
  %t45 = load %str, ptr %a44
  %t46 = load i1, ptr %a4
  call void @veles_bool_to_string(ptr %a47, i1 %t46)
  %t48 = load %str, ptr %a47
  %t49 = extractvalue %str %t45, 0
  %t50 = extractvalue %str %t45, 1
  %t51 = extractvalue %str %t48, 0
  %t52 = extractvalue %str %t48, 1
  call void @veles_string_concat(ptr %a53, ptr %t49, i64 %t50, ptr %t51, i64 %t52)
  %t54 = load %str, ptr %a53
  %t55 = extractvalue %str %t54, 0
  %t56 = extractvalue %str %t54, 1
  %t57 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t58 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a59, ptr %t55, i64 %t56, ptr %t57, i64 %t58)
  %t60 = load %str, ptr %a59
  %t61 = load { i64, %str }, ptr %a7
  %t62 = call %str @show.T_i64_string_({ i64, %str } %t61)
  %t63 = extractvalue %str %t60, 0
  %t64 = extractvalue %str %t60, 1
  %t65 = extractvalue %str %t62, 0
  %t66 = extractvalue %str %t62, 1
  call void @veles_string_concat(ptr %a67, ptr %t63, i64 %t64, ptr %t65, i64 %t66)
  %t68 = load %str, ptr %a67
  %t69 = extractvalue %str %t68, 0
  %t70 = extractvalue %str %t68, 1
  %t71 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t72 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a73, ptr %t69, i64 %t70, ptr %t71, i64 %t72)
  %t74 = load %str, ptr %a73
  %t75 = load %str, ptr %a1
  %t76 = extractvalue %str %t75, 1
  call void @veles_i64_to_string(ptr %a77, i64 %t76)
  %t78 = load %str, ptr %a77
  %t79 = extractvalue %str %t74, 0
  %t80 = extractvalue %str %t74, 1
  %t81 = extractvalue %str %t78, 0
  %t82 = extractvalue %str %t78, 1
  call void @veles_string_concat(ptr %a83, ptr %t79, i64 %t80, ptr %t81, i64 %t82)
  %t84 = load %str, ptr %a83
  %t85 = extractvalue %str %t84, 0
  %t86 = extractvalue %str %t84, 1
  %t87 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t88 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a89, ptr %t85, i64 %t86, ptr %t87, i64 %t88)
  %t90 = load %str, ptr %a89
  %t91 = load %str, ptr %a1
  %t92 = extractvalue %str %t91, 0
  %t93 = extractvalue %str %t91, 1
  %t94 = call i8 @veles_string_byte_at(ptr %t92, i64 %t93, i64 0)
  %t96 = zext i8 %t94 to i64
  call void @veles_u64_to_string(ptr %a95, i64 %t96)
  %t97 = load %str, ptr %a95
  %t98 = extractvalue %str %t90, 0
  %t99 = extractvalue %str %t90, 1
  %t100 = extractvalue %str %t97, 0
  %t101 = extractvalue %str %t97, 1
  call void @veles_string_concat(ptr %a102, ptr %t98, i64 %t99, ptr %t100, i64 %t101)
  %t103 = load %str, ptr %a102
  %t104 = extractvalue %str %t103, 0
  %t105 = extractvalue %str %t103, 1
  %t106 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t107 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a108, ptr %t104, i64 %t105, ptr %t106, i64 %t107)
  %t109 = load %str, ptr %a108
  %t110 = call %str @v_std.prelude.extend.string.toUpper(ptr %a1)
  %t111 = extractvalue %str %t109, 0
  %t112 = extractvalue %str %t109, 1
  %t113 = extractvalue %str %t110, 0
  %t114 = extractvalue %str %t110, 1
  call void @veles_string_concat(ptr %a115, ptr %t111, i64 %t112, ptr %t113, i64 %t114)
  %t116 = load %str, ptr %a115
  call void @v_std.io.println(%str %t116)
  %t117 = call %str @v_std.prelude.extend.string.trim(ptr %a8)
  store %str %t117, ptr %a118
  %t119 = call ptr @v_std.prelude.extend.string.split(ptr %a118, %str { ptr @.str.5, i64 1 })
  %t120 = call %str @show.List_string_(ptr %t119)
  %t121 = extractvalue %str %t120, 0
  %t122 = extractvalue %str %t120, 1
  %t123 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t124 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a125, ptr %t121, i64 %t122, ptr %t123, i64 %t124)
  %t126 = load %str, ptr %a125
  %t127 = call %str @v_std.prelude.extend.string.replace(ptr %a1, %str { ptr @.str.6, i64 1 }, %str { ptr @.str.7, i64 1 })
  %t128 = extractvalue %str %t126, 0
  %t129 = extractvalue %str %t126, 1
  %t130 = extractvalue %str %t127, 0
  %t131 = extractvalue %str %t127, 1
  call void @veles_string_concat(ptr %a132, ptr %t128, i64 %t129, ptr %t130, i64 %t131)
  %t133 = load %str, ptr %a132
  %t134 = extractvalue %str %t133, 0
  %t135 = extractvalue %str %t133, 1
  %t136 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t137 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a138, ptr %t134, i64 %t135, ptr %t136, i64 %t137)
  %t139 = load %str, ptr %a138
  %t140 = call i64 @v_std.prelude.extend.string.indexOf(ptr %a1, %str { ptr @.str.8, i64 1 }, i64 0)
  call void @veles_i64_to_string(ptr %a141, i64 %t140)
  %t142 = load %str, ptr %a141
  %t143 = extractvalue %str %t139, 0
  %t144 = extractvalue %str %t139, 1
  %t145 = extractvalue %str %t142, 0
  %t146 = extractvalue %str %t142, 1
  call void @veles_string_concat(ptr %a147, ptr %t143, i64 %t144, ptr %t145, i64 %t146)
  %t148 = load %str, ptr %a147
  %t149 = extractvalue %str %t148, 0
  %t150 = extractvalue %str %t148, 1
  %t151 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t152 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  call void @veles_string_concat(ptr %a153, ptr %t149, i64 %t150, ptr %t151, i64 %t152)
  %t154 = load %str, ptr %a153
  store %str { ptr @.str.9, i64 2 }, ptr %a155
  %t156 = call %str @v_std.prelude.extend.string.repeat(ptr %a155, i64 3)
  %t157 = extractvalue %str %t154, 0
  %t158 = extractvalue %str %t154, 1
  %t159 = extractvalue %str %t156, 0
  %t160 = extractvalue %str %t156, 1
  call void @veles_string_concat(ptr %a161, ptr %t157, i64 %t158, ptr %t159, i64 %t160)
  %t162 = load %str, ptr %a161
  call void @v_std.io.println(%str %t162)
  %t163 = call %S.std.prelude.StringBuilder @v_std.prelude.stringBuilder()
  store %S.std.prelude.StringBuilder %t163, ptr %a164
  call void @v_std.prelude.StringBuilder.append(ptr %a164, %str { ptr @.str.10, i64 1 })
  call void @v_std.prelude.StringBuilder.appendByte(ptr %a164, i8 121)
  %t165 = call %str @v_std.prelude.StringBuilder.toString(ptr %a164)
  call void @v_std.io.println(%str %t165)
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
