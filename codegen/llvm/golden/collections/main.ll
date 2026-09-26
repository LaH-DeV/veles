%S.main.Counter = type { i64 }
@adesc.main.Counter = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 1, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a5 = alloca ptr
  %a8 = alloca %S.main.Counter
  %a11 = alloca %S.main.Counter
  %a17 = alloca ptr
  %a19 = alloca i64
  %a38 = alloca ptr
  %a53 = alloca ptr
  %a56 = alloca i64
  %a76 = alloca ptr
  %a81 = alloca %str
  %a82 = alloca i64
  %a87 = alloca %str
  %a88 = alloca i64
  %a93 = alloca %str
  %a96 = alloca ptr
  %a99 = alloca ptr
  %a106 = alloca ptr
  %a122 = alloca %str
  %a125 = alloca ptr
  %a128 = alloca i64
  %a134 = alloca i64
  %a139 = alloca ptr
  %a140 = alloca i64
  %a148 = alloca { i1, i64 }
  %a149 = alloca i1
  %a174 = alloca i64
  %a179 = alloca %str
  %a185 = alloca %str
  %a188 = alloca ptr
  %a189 = alloca i64
  %a197 = alloca { i1, i64 }
  %a198 = alloca i1
  %a223 = alloca i64
  %a228 = alloca %str
  %a234 = alloca %str
  %a240 = alloca %str
  %a243 = alloca ptr
  %a245 = alloca ptr
  %a254 = alloca %str
  %a260 = alloca %str
  %a263 = alloca ptr
  %a264 = alloca i64
  %a272 = alloca { i1, %S.main.Counter }
  %a273 = alloca i1
  %a298 = alloca { i1, %S.main.Counter }
  %a299 = alloca { i1, i64 }
  %a314 = alloca %str
  %a320 = alloca %str
  %a323 = alloca ptr
  %a324 = alloca i64
  %a332 = alloca { i1, %S.main.Counter }
  %a333 = alloca i1
  %a358 = alloca { i1, %S.main.Counter }
  %a359 = alloca { i1, i64 }
  %a374 = alloca %str
  %a380 = alloca %str
  %a383 = alloca { i1, i64 }
  %a389 = alloca i64
  %a394 = alloca %str
  %a400 = alloca %str
  %a406 = alloca %str
  %a409 = alloca { i1, i64 }
  %a415 = alloca i64
  %a420 = alloca %str
  %a426 = alloca %str
  %a432 = alloca %str
  %a436 = alloca %str
  %a442 = alloca %str
  %a448 = alloca %str
  %a452 = alloca %str
  %a458 = alloca %str
  %a464 = alloca %str
  %a468 = alloca i64
  %a471 = alloca %str
  %a477 = alloca %str
  %t1 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 1, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 2, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store ptr %t1, ptr %a3
  %t4 = call ptr @veles_list_new(ptr @adesc.main.Counter, i64 0)
  store ptr %t4, ptr %a5
  %t6 = load ptr, ptr %a5
  %t7 = insertvalue %S.main.Counter undef, i64 1, 0
  store %S.main.Counter %t7, ptr %a8
  call void @veles_list_push(ptr %t6, ptr %a8)
  %t9 = load ptr, ptr %a5
  %t10 = insertvalue %S.main.Counter undef, i64 5, 0
  store %S.main.Counter %t10, ptr %a11
  call void @veles_list_push(ptr %t9, ptr %a11)
  %t12 = load ptr, ptr %a5
  %t13 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t12, i32 0, i32 1
  %t14 = load i64, ptr %t13
  %t15 = icmp eq i64 %t14, 2
  br i1 %t15, label %if.then.1, label %if.end.2
if.then.1:
  %t16 = load ptr, ptr %a5
  store ptr %t16, ptr %a17
  %t18 = load ptr, ptr %a17
  store i64 0, ptr %a19
  %t20 = load i64, ptr %a19
  %t21 = icmp slt i64 %t20, 0
  br i1 %t21, label %if.then.3, label %if.end.4
if.then.3:
  %t22 = load i64, ptr %a19
  %t23 = load ptr, ptr %a17
  %t24 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t23, i32 0, i32 1
  %t25 = load i64, ptr %t24
  %t26 = add i64 %t22, %t25
  store i64 %t26, ptr %a19
  br label %if.end.4
if.end.4:
  %t27 = load i64, ptr %a19
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t18, i32 0, i32 1
  %t29 = load i64, ptr %t28
  %t30 = icmp ult i64 %t27, %t29
  br i1 %t30, label %idx.ok.5, label %idx.bad.6, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.6:
  %t31 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t32 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t18, i64 %t27, ptr %t31, i64 %t32)
  unreachable
idx.ok.5:
  %t33 = load ptr, ptr %t18
  %t34 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t18, i32 0, i32 3
  %t35 = load i64, ptr %t34
  %t36 = mul i64 %t35, %t27
  %t37 = getelementptr inbounds i8, ptr %t33, i64 %t36
  store ptr %t37, ptr %a38
  %t39 = load ptr, ptr %a38
  %t40 = getelementptr inbounds %S.main.Counter, ptr %t39, i32 0, i32 0
  %t41 = load i64, ptr %t40
  %t43 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t41, i64 10)
  %t44 = extractvalue { i64, i1 } %t43, 0
  %t45 = extractvalue { i64, i1 } %t43, 1
  br i1 %t45, label %overflow.7, label %arith.ok.8
overflow.7:
  %t46 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t47 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t48 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t49 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_panic_at(ptr %t46, i64 %t47, ptr %t48, i64 %t49)
  unreachable
arith.ok.8:
  %t50 = load ptr, ptr %a38
  %t51 = getelementptr inbounds %S.main.Counter, ptr %t50, i32 0, i32 0
  store i64 %t44, ptr %t51
  br label %if.end.2
if.end.2:
  %t52 = load ptr, ptr %a5
  store ptr %t52, ptr %a53
  %t54 = insertvalue %S.main.Counter undef, i64 7, 0
  %t55 = load ptr, ptr %a53
  store i64 1, ptr %a56
  %t57 = load i64, ptr %a56
  %t58 = icmp slt i64 %t57, 0
  br i1 %t58, label %if.then.9, label %if.end.10
if.then.9:
  %t59 = load i64, ptr %a56
  %t60 = load ptr, ptr %a53
  %t61 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t60, i32 0, i32 1
  %t62 = load i64, ptr %t61
  %t63 = add i64 %t59, %t62
  store i64 %t63, ptr %a56
  br label %if.end.10
if.end.10:
  %t64 = load i64, ptr %a56
  %t65 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t55, i32 0, i32 1
  %t66 = load i64, ptr %t65
  %t67 = icmp ult i64 %t64, %t66
  br i1 %t67, label %idx.ok.11, label %idx.bad.12, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.12:
  %t68 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t69 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t55, i64 %t64, ptr %t68, i64 %t69)
  unreachable
idx.ok.11:
  %t70 = load ptr, ptr %t55
  %t71 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t55, i32 0, i32 3
  %t72 = load i64, ptr %t71
  %t73 = mul i64 %t72, %t64
  %t74 = getelementptr inbounds i8, ptr %t70, i64 %t73
  store %S.main.Counter %t54, ptr %t74
  %t75 = call ptr @veles_map_new(ptr @adesc.string, ptr @adesc.i64)
  store ptr %t75, ptr %a76
  %t77 = load ptr, ptr %a76
  %t79 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t80 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t78 = call i64 @veles_hash_bytes(ptr %t79, i64 %t80)
  store %str { ptr @.str.4, i64 1 }, ptr %a81
  store i64 1, ptr %a82
  call i64 @veles_map_insert(ptr %t77, i64 %t78, ptr %a81, ptr %a82, ptr @eqp.string)
  %t83 = load ptr, ptr %a76
  %t85 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t86 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t84 = call i64 @veles_hash_bytes(ptr %t85, i64 %t86)
  store %str { ptr @.str.5, i64 1 }, ptr %a87
  store i64 2, ptr %a88
  call i64 @veles_map_insert(ptr %t83, i64 %t84, ptr %a87, ptr %a88, ptr @eqp.string)
  %t89 = load ptr, ptr %a76
  %t91 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t92 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t90 = call i64 @veles_hash_bytes(ptr %t91, i64 %t92)
  store %str { ptr @.str.4, i64 1 }, ptr %a93
  %t94 = call i64 @veles_map_find(ptr %t89, i64 %t90, ptr %a93, ptr @eqp.string)
  %t95 = icmp sge i64 %t94, 0
  store ptr null, ptr %a96
  br i1 %t95, label %map.hit.13, label %map.end.14
map.hit.13:
  %t97 = call ptr @veles_map_val_at(ptr %t89, i64 %t94)
  store ptr %t97, ptr %a96
  br label %map.end.14
map.end.14:
  %t98 = load ptr, ptr %a96
  %t100 = icmp eq ptr %t98, null
  br i1 %t100, label %elvis.default.15, label %elvis.some.16
elvis.some.16:
  store ptr %t98, ptr %a99
  br label %elvis.end.17
elvis.default.15:
  %t101 = extractvalue %str { ptr @.str.6, i64 15 }, 0
  %t102 = extractvalue %str { ptr @.str.6, i64 15 }, 1
  %t103 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t104 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t101, i64 %t102, ptr %t103, i64 %t104)
  unreachable
elvis.end.17:
  %t105 = load ptr, ptr %a99
  store ptr %t105, ptr %a106
  %t107 = load ptr, ptr %a106
  %t108 = load i64, ptr %t107
  %t110 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t108, i64 40)
  %t111 = extractvalue { i64, i1 } %t110, 0
  %t112 = extractvalue { i64, i1 } %t110, 1
  br i1 %t112, label %overflow.18, label %arith.ok.19
overflow.18:
  %t113 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t114 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t115 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t116 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t113, i64 %t114, ptr %t115, i64 %t116)
  unreachable
arith.ok.19:
  %t117 = load ptr, ptr %a106
  store i64 %t111, ptr %t117
  %t118 = load ptr, ptr %a76
  %t120 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t121 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t119 = call i64 @veles_hash_bytes(ptr %t120, i64 %t121)
  store %str { ptr @.str.5, i64 1 }, ptr %a122
  %t123 = call i1 @veles_map_remove(ptr %t118, i64 %t119, ptr %a122, ptr @eqp.string)
  %t124 = call ptr @veles_map_new(ptr @adesc.i64, ptr null)
  store ptr %t124, ptr %a125
  %t126 = load ptr, ptr %a125
  store i64 4, ptr %a128
  %t129 = call i64 @veles_map_len(ptr %t126)
  call i64 @veles_map_insert(ptr %t126, i64 4, ptr %a128, ptr null, ptr @eqp.i64)
  %t130 = call i64 @veles_map_len(ptr %t126)
  %t131 = icmp ne i64 %t129, %t130
  %t132 = load ptr, ptr %a125
  store i64 4, ptr %a134
  %t135 = call i64 @veles_map_len(ptr %t132)
  call i64 @veles_map_insert(ptr %t132, i64 4, ptr %a134, ptr null, ptr @eqp.i64)
  %t136 = call i64 @veles_map_len(ptr %t132)
  %t137 = icmp ne i64 %t135, %t136
  %t138 = load ptr, ptr %a3
  store ptr %t138, ptr %a139
  store i64 1, ptr %a140
  %t141 = load i64, ptr %a140
  %t142 = icmp slt i64 %t141, 0
  br i1 %t142, label %if.then.20, label %if.end.21
if.then.20:
  %t143 = load i64, ptr %a140
  %t144 = load ptr, ptr %a139
  %t145 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t144, i32 0, i32 1
  %t146 = load i64, ptr %t145
  %t147 = add i64 %t143, %t146
  store i64 %t147, ptr %a140
  br label %if.end.21
if.end.21:
  %t150 = load i64, ptr %a140
  %t151 = icmp sge i64 %t150, 0
  store i1 %t151, ptr %a149
  br i1 %t151, label %sc.rhs.22, label %sc.end.23
sc.rhs.22:
  %t152 = load i64, ptr %a140
  %t153 = load ptr, ptr %a139
  %t154 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t153, i32 0, i32 1
  %t155 = load i64, ptr %t154
  %t156 = icmp slt i64 %t152, %t155
  store i1 %t156, ptr %a149
  br label %sc.end.23
sc.end.23:
  %t157 = load i1, ptr %a149
  br i1 %t157, label %if.then.24, label %if.else.26
if.then.24:
  %t158 = load ptr, ptr %a139
  %t159 = load i64, ptr %a140
  %t160 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t158, i32 0, i32 1
  %t161 = load i64, ptr %t160
  %t162 = icmp ult i64 %t159, %t161
  br i1 %t162, label %idx.ok.27, label %idx.bad.28, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.28:
  %t163 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t164 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t158, i64 %t159, ptr %t163, i64 %t164)
  unreachable
idx.ok.27:
  %t165 = load ptr, ptr %t158
  %t166 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t158, i32 0, i32 3
  %t167 = load i64, ptr %t166
  %t168 = mul i64 %t167, %t159
  %t169 = getelementptr inbounds i8, ptr %t165, i64 %t168
  %t170 = load i64, ptr %t169
  %t171 = insertvalue { i1, i64 } undef, i1 true, 0
  %t172 = insertvalue { i1, i64 } %t171, i64 %t170, 1
  store { i1, i64 } %t172, ptr %a148
  br label %if.end.25
if.else.26:
  store { i1, i64 } zeroinitializer, ptr %a148
  br label %if.end.25
if.end.25:
  %t173 = load { i1, i64 }, ptr %a148
  %t176 = extractvalue { i1, i64 } %t173, 0
  %t175 = xor i1 %t176, true
  br i1 %t175, label %elvis.default.29, label %elvis.some.30
elvis.some.30:
  %t177 = extractvalue { i1, i64 } %t173, 1
  store i64 %t177, ptr %a174
  br label %elvis.end.31
elvis.default.29:
  store i64 -1, ptr %a174
  br label %elvis.end.31
elvis.end.31:
  %t178 = load i64, ptr %a174
  call void @veles_i64_to_string(ptr %a179, i64 %t178)
  %t180 = load %str, ptr %a179
  %t181 = extractvalue %str %t180, 0
  %t182 = extractvalue %str %t180, 1
  %t183 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t184 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a185, ptr %t181, i64 %t182, ptr %t183, i64 %t184)
  %t186 = load %str, ptr %a185
  %t187 = load ptr, ptr %a3
  store ptr %t187, ptr %a188
  store i64 9, ptr %a189
  %t190 = load i64, ptr %a189
  %t191 = icmp slt i64 %t190, 0
  br i1 %t191, label %if.then.32, label %if.end.33
if.then.32:
  %t192 = load i64, ptr %a189
  %t193 = load ptr, ptr %a188
  %t194 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t193, i32 0, i32 1
  %t195 = load i64, ptr %t194
  %t196 = add i64 %t192, %t195
  store i64 %t196, ptr %a189
  br label %if.end.33
if.end.33:
  %t199 = load i64, ptr %a189
  %t200 = icmp sge i64 %t199, 0
  store i1 %t200, ptr %a198
  br i1 %t200, label %sc.rhs.34, label %sc.end.35
sc.rhs.34:
  %t201 = load i64, ptr %a189
  %t202 = load ptr, ptr %a188
  %t203 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t202, i32 0, i32 1
  %t204 = load i64, ptr %t203
  %t205 = icmp slt i64 %t201, %t204
  store i1 %t205, ptr %a198
  br label %sc.end.35
sc.end.35:
  %t206 = load i1, ptr %a198
  br i1 %t206, label %if.then.36, label %if.else.38
if.then.36:
  %t207 = load ptr, ptr %a188
  %t208 = load i64, ptr %a189
  %t209 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t207, i32 0, i32 1
  %t210 = load i64, ptr %t209
  %t211 = icmp ult i64 %t208, %t210
  br i1 %t211, label %idx.ok.39, label %idx.bad.40, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.40:
  %t212 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t213 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t207, i64 %t208, ptr %t212, i64 %t213)
  unreachable
idx.ok.39:
  %t214 = load ptr, ptr %t207
  %t215 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t207, i32 0, i32 3
  %t216 = load i64, ptr %t215
  %t217 = mul i64 %t216, %t208
  %t218 = getelementptr inbounds i8, ptr %t214, i64 %t217
  %t219 = load i64, ptr %t218
  %t220 = insertvalue { i1, i64 } undef, i1 true, 0
  %t221 = insertvalue { i1, i64 } %t220, i64 %t219, 1
  store { i1, i64 } %t221, ptr %a197
  br label %if.end.37
if.else.38:
  store { i1, i64 } zeroinitializer, ptr %a197
  br label %if.end.37
if.end.37:
  %t222 = load { i1, i64 }, ptr %a197
  %t225 = extractvalue { i1, i64 } %t222, 0
  %t224 = xor i1 %t225, true
  br i1 %t224, label %elvis.default.41, label %elvis.some.42
elvis.some.42:
  %t226 = extractvalue { i1, i64 } %t222, 1
  store i64 %t226, ptr %a223
  br label %elvis.end.43
elvis.default.41:
  store i64 -1, ptr %a223
  br label %elvis.end.43
elvis.end.43:
  %t227 = load i64, ptr %a223
  call void @veles_i64_to_string(ptr %a228, i64 %t227)
  %t229 = load %str, ptr %a228
  %t230 = extractvalue %str %t186, 0
  %t231 = extractvalue %str %t186, 1
  %t232 = extractvalue %str %t229, 0
  %t233 = extractvalue %str %t229, 1
  call void @veles_string_concat(ptr %a234, ptr %t230, i64 %t231, ptr %t232, i64 %t233)
  %t235 = load %str, ptr %a234
  %t236 = extractvalue %str %t235, 0
  %t237 = extractvalue %str %t235, 1
  %t238 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t239 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a240, ptr %t236, i64 %t237, ptr %t238, i64 %t239)
  %t241 = load %str, ptr %a240
  %t242 = load ptr, ptr %a3
  store ptr %t242, ptr %a243
  %t244 = load ptr, ptr %a243
  store ptr %t244, ptr %a245
  %t246 = insertvalue { ptr, ptr } undef, ptr @v_main.main.lambda1, 0
  %t247 = insertvalue { ptr, ptr } %t246, ptr null, 1
  %t248 = call ptr @v_std.prelude.extend.List_T.sortedWith_T_i64_(ptr %a245, { ptr, ptr } %t247)
  %t249 = call %str @show.List_i64_(ptr %t248)
  %t250 = extractvalue %str %t241, 0
  %t251 = extractvalue %str %t241, 1
  %t252 = extractvalue %str %t249, 0
  %t253 = extractvalue %str %t249, 1
  call void @veles_string_concat(ptr %a254, ptr %t250, i64 %t251, ptr %t252, i64 %t253)
  %t255 = load %str, ptr %a254
  %t256 = extractvalue %str %t255, 0
  %t257 = extractvalue %str %t255, 1
  %t258 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t259 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a260, ptr %t256, i64 %t257, ptr %t258, i64 %t259)
  %t261 = load %str, ptr %a260
  %t262 = load ptr, ptr %a5
  store ptr %t262, ptr %a263
  store i64 0, ptr %a264
  %t265 = load i64, ptr %a264
  %t266 = icmp slt i64 %t265, 0
  br i1 %t266, label %if.then.44, label %if.end.45
if.then.44:
  %t267 = load i64, ptr %a264
  %t268 = load ptr, ptr %a263
  %t269 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t268, i32 0, i32 1
  %t270 = load i64, ptr %t269
  %t271 = add i64 %t267, %t270
  store i64 %t271, ptr %a264
  br label %if.end.45
if.end.45:
  %t274 = load i64, ptr %a264
  %t275 = icmp sge i64 %t274, 0
  store i1 %t275, ptr %a273
  br i1 %t275, label %sc.rhs.46, label %sc.end.47
sc.rhs.46:
  %t276 = load i64, ptr %a264
  %t277 = load ptr, ptr %a263
  %t278 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t277, i32 0, i32 1
  %t279 = load i64, ptr %t278
  %t280 = icmp slt i64 %t276, %t279
  store i1 %t280, ptr %a273
  br label %sc.end.47
sc.end.47:
  %t281 = load i1, ptr %a273
  br i1 %t281, label %if.then.48, label %if.else.50
if.then.48:
  %t282 = load ptr, ptr %a263
  %t283 = load i64, ptr %a264
  %t284 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t282, i32 0, i32 1
  %t285 = load i64, ptr %t284
  %t286 = icmp ult i64 %t283, %t285
  br i1 %t286, label %idx.ok.51, label %idx.bad.52, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.52:
  %t287 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t288 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t282, i64 %t283, ptr %t287, i64 %t288)
  unreachable
idx.ok.51:
  %t289 = load ptr, ptr %t282
  %t290 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t282, i32 0, i32 3
  %t291 = load i64, ptr %t290
  %t292 = mul i64 %t291, %t283
  %t293 = getelementptr inbounds i8, ptr %t289, i64 %t292
  %t294 = load %S.main.Counter, ptr %t293
  %t295 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t296 = insertvalue { i1, %S.main.Counter } %t295, %S.main.Counter %t294, 1
  store { i1, %S.main.Counter } %t296, ptr %a272
  br label %if.end.49
if.else.50:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a272
  br label %if.end.49
if.end.49:
  %t297 = load { i1, %S.main.Counter }, ptr %a272
  store { i1, %S.main.Counter } %t297, ptr %a298
  %t300 = load { i1, %S.main.Counter }, ptr %a298
  %t302 = extractvalue { i1, %S.main.Counter } %t300, 0
  %t301 = xor i1 %t302, true
  br i1 %t301, label %if.then.53, label %if.else.55
if.then.53:
  store { i1, i64 } zeroinitializer, ptr %a299
  br label %if.end.54
if.else.55:
  %t303 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a298, i32 0, i32 1
  %t304 = getelementptr inbounds %S.main.Counter, ptr %t303, i32 0, i32 0
  %t305 = load i64, ptr %t304
  %t306 = insertvalue { i1, i64 } undef, i1 true, 0
  %t307 = insertvalue { i1, i64 } %t306, i64 %t305, 1
  store { i1, i64 } %t307, ptr %a299
  br label %if.end.54
if.end.54:
  %t308 = load { i1, i64 }, ptr %a299
  %t309 = call %str @show.T_i64_N({ i1, i64 } %t308)
  %t310 = extractvalue %str %t261, 0
  %t311 = extractvalue %str %t261, 1
  %t312 = extractvalue %str %t309, 0
  %t313 = extractvalue %str %t309, 1
  call void @veles_string_concat(ptr %a314, ptr %t310, i64 %t311, ptr %t312, i64 %t313)
  %t315 = load %str, ptr %a314
  %t316 = extractvalue %str %t315, 0
  %t317 = extractvalue %str %t315, 1
  %t318 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t319 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a320, ptr %t316, i64 %t317, ptr %t318, i64 %t319)
  %t321 = load %str, ptr %a320
  %t322 = load ptr, ptr %a5
  store ptr %t322, ptr %a323
  store i64 1, ptr %a324
  %t325 = load i64, ptr %a324
  %t326 = icmp slt i64 %t325, 0
  br i1 %t326, label %if.then.56, label %if.end.57
if.then.56:
  %t327 = load i64, ptr %a324
  %t328 = load ptr, ptr %a323
  %t329 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t328, i32 0, i32 1
  %t330 = load i64, ptr %t329
  %t331 = add i64 %t327, %t330
  store i64 %t331, ptr %a324
  br label %if.end.57
if.end.57:
  %t334 = load i64, ptr %a324
  %t335 = icmp sge i64 %t334, 0
  store i1 %t335, ptr %a333
  br i1 %t335, label %sc.rhs.58, label %sc.end.59
sc.rhs.58:
  %t336 = load i64, ptr %a324
  %t337 = load ptr, ptr %a323
  %t338 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t337, i32 0, i32 1
  %t339 = load i64, ptr %t338
  %t340 = icmp slt i64 %t336, %t339
  store i1 %t340, ptr %a333
  br label %sc.end.59
sc.end.59:
  %t341 = load i1, ptr %a333
  br i1 %t341, label %if.then.60, label %if.else.62
if.then.60:
  %t342 = load ptr, ptr %a323
  %t343 = load i64, ptr %a324
  %t344 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t342, i32 0, i32 1
  %t345 = load i64, ptr %t344
  %t346 = icmp ult i64 %t343, %t345
  br i1 %t346, label %idx.ok.63, label %idx.bad.64, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.64:
  %t347 = extractvalue %str { ptr @.str.13, i64 13 }, 0
  %t348 = extractvalue %str { ptr @.str.13, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t342, i64 %t343, ptr %t347, i64 %t348)
  unreachable
idx.ok.63:
  %t349 = load ptr, ptr %t342
  %t350 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t342, i32 0, i32 3
  %t351 = load i64, ptr %t350
  %t352 = mul i64 %t351, %t343
  %t353 = getelementptr inbounds i8, ptr %t349, i64 %t352
  %t354 = load %S.main.Counter, ptr %t353
  %t355 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t356 = insertvalue { i1, %S.main.Counter } %t355, %S.main.Counter %t354, 1
  store { i1, %S.main.Counter } %t356, ptr %a332
  br label %if.end.61
if.else.62:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a332
  br label %if.end.61
if.end.61:
  %t357 = load { i1, %S.main.Counter }, ptr %a332
  store { i1, %S.main.Counter } %t357, ptr %a358
  %t360 = load { i1, %S.main.Counter }, ptr %a358
  %t362 = extractvalue { i1, %S.main.Counter } %t360, 0
  %t361 = xor i1 %t362, true
  br i1 %t361, label %if.then.65, label %if.else.67
if.then.65:
  store { i1, i64 } zeroinitializer, ptr %a359
  br label %if.end.66
if.else.67:
  %t363 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a358, i32 0, i32 1
  %t364 = getelementptr inbounds %S.main.Counter, ptr %t363, i32 0, i32 0
  %t365 = load i64, ptr %t364
  %t366 = insertvalue { i1, i64 } undef, i1 true, 0
  %t367 = insertvalue { i1, i64 } %t366, i64 %t365, 1
  store { i1, i64 } %t367, ptr %a359
  br label %if.end.66
if.end.66:
  %t368 = load { i1, i64 }, ptr %a359
  %t369 = call %str @show.T_i64_N({ i1, i64 } %t368)
  %t370 = extractvalue %str %t321, 0
  %t371 = extractvalue %str %t321, 1
  %t372 = extractvalue %str %t369, 0
  %t373 = extractvalue %str %t369, 1
  call void @veles_string_concat(ptr %a374, ptr %t370, i64 %t371, ptr %t372, i64 %t373)
  %t375 = load %str, ptr %a374
  call void @v_std.io.println(%str %t375)
  %t376 = load ptr, ptr %a76
  %t378 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t379 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t377 = call i64 @veles_hash_bytes(ptr %t378, i64 %t379)
  store %str { ptr @.str.4, i64 1 }, ptr %a380
  %t381 = call i64 @veles_map_find(ptr %t376, i64 %t377, ptr %a380, ptr @eqp.string)
  %t382 = icmp sge i64 %t381, 0
  store { i1, i64 } zeroinitializer, ptr %a383
  br i1 %t382, label %map.hit.68, label %map.end.69
map.hit.68:
  %t384 = call ptr @veles_map_val_at(ptr %t376, i64 %t381)
  %t385 = load i64, ptr %t384
  %t386 = insertvalue { i1, i64 } undef, i1 true, 0
  %t387 = insertvalue { i1, i64 } %t386, i64 %t385, 1
  store { i1, i64 } %t387, ptr %a383
  br label %map.end.69
map.end.69:
  %t388 = load { i1, i64 }, ptr %a383
  %t391 = extractvalue { i1, i64 } %t388, 0
  %t390 = xor i1 %t391, true
  br i1 %t390, label %elvis.default.70, label %elvis.some.71
elvis.some.71:
  %t392 = extractvalue { i1, i64 } %t388, 1
  store i64 %t392, ptr %a389
  br label %elvis.end.72
elvis.default.70:
  store i64 0, ptr %a389
  br label %elvis.end.72
elvis.end.72:
  %t393 = load i64, ptr %a389
  call void @veles_i64_to_string(ptr %a394, i64 %t393)
  %t395 = load %str, ptr %a394
  %t396 = extractvalue %str %t395, 0
  %t397 = extractvalue %str %t395, 1
  %t398 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t399 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a400, ptr %t396, i64 %t397, ptr %t398, i64 %t399)
  %t401 = load %str, ptr %a400
  %t402 = load ptr, ptr %a76
  %t404 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t405 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t403 = call i64 @veles_hash_bytes(ptr %t404, i64 %t405)
  store %str { ptr @.str.5, i64 1 }, ptr %a406
  %t407 = call i64 @veles_map_find(ptr %t402, i64 %t403, ptr %a406, ptr @eqp.string)
  %t408 = icmp sge i64 %t407, 0
  store { i1, i64 } zeroinitializer, ptr %a409
  br i1 %t408, label %map.hit.73, label %map.end.74
map.hit.73:
  %t410 = call ptr @veles_map_val_at(ptr %t402, i64 %t407)
  %t411 = load i64, ptr %t410
  %t412 = insertvalue { i1, i64 } undef, i1 true, 0
  %t413 = insertvalue { i1, i64 } %t412, i64 %t411, 1
  store { i1, i64 } %t413, ptr %a409
  br label %map.end.74
map.end.74:
  %t414 = load { i1, i64 }, ptr %a409
  %t417 = extractvalue { i1, i64 } %t414, 0
  %t416 = xor i1 %t417, true
  br i1 %t416, label %elvis.default.75, label %elvis.some.76
elvis.some.76:
  %t418 = extractvalue { i1, i64 } %t414, 1
  store i64 %t418, ptr %a415
  br label %elvis.end.77
elvis.default.75:
  store i64 0, ptr %a415
  br label %elvis.end.77
elvis.end.77:
  %t419 = load i64, ptr %a415
  call void @veles_i64_to_string(ptr %a420, i64 %t419)
  %t421 = load %str, ptr %a420
  %t422 = extractvalue %str %t401, 0
  %t423 = extractvalue %str %t401, 1
  %t424 = extractvalue %str %t421, 0
  %t425 = extractvalue %str %t421, 1
  call void @veles_string_concat(ptr %a426, ptr %t422, i64 %t423, ptr %t424, i64 %t425)
  %t427 = load %str, ptr %a426
  %t428 = extractvalue %str %t427, 0
  %t429 = extractvalue %str %t427, 1
  %t430 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t431 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a432, ptr %t428, i64 %t429, ptr %t430, i64 %t431)
  %t433 = load %str, ptr %a432
  %t434 = load ptr, ptr %a76
  %t435 = call i64 @veles_map_len(ptr %t434)
  call void @veles_i64_to_string(ptr %a436, i64 %t435)
  %t437 = load %str, ptr %a436
  %t438 = extractvalue %str %t433, 0
  %t439 = extractvalue %str %t433, 1
  %t440 = extractvalue %str %t437, 0
  %t441 = extractvalue %str %t437, 1
  call void @veles_string_concat(ptr %a442, ptr %t438, i64 %t439, ptr %t440, i64 %t441)
  %t443 = load %str, ptr %a442
  %t444 = extractvalue %str %t443, 0
  %t445 = extractvalue %str %t443, 1
  %t446 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t447 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a448, ptr %t444, i64 %t445, ptr %t446, i64 %t447)
  %t449 = load %str, ptr %a448
  %t450 = load ptr, ptr %a125
  %t451 = call i64 @veles_map_len(ptr %t450)
  call void @veles_i64_to_string(ptr %a452, i64 %t451)
  %t453 = load %str, ptr %a452
  %t454 = extractvalue %str %t449, 0
  %t455 = extractvalue %str %t449, 1
  %t456 = extractvalue %str %t453, 0
  %t457 = extractvalue %str %t453, 1
  call void @veles_string_concat(ptr %a458, ptr %t454, i64 %t455, ptr %t456, i64 %t457)
  %t459 = load %str, ptr %a458
  %t460 = extractvalue %str %t459, 0
  %t461 = extractvalue %str %t459, 1
  %t462 = extractvalue %str { ptr @.str.10, i64 1 }, 0
  %t463 = extractvalue %str { ptr @.str.10, i64 1 }, 1
  call void @veles_string_concat(ptr %a464, ptr %t460, i64 %t461, ptr %t462, i64 %t463)
  %t465 = load %str, ptr %a464
  %t466 = load ptr, ptr %a125
  store i64 4, ptr %a468
  %t469 = call i64 @veles_map_find(ptr %t466, i64 4, ptr %a468, ptr @eqp.i64)
  %t470 = icmp sge i64 %t469, 0
  call void @veles_bool_to_string(ptr %a471, i1 %t470)
  %t472 = load %str, ptr %a471
  %t473 = extractvalue %str %t465, 0
  %t474 = extractvalue %str %t465, 1
  %t475 = extractvalue %str %t472, 0
  %t476 = extractvalue %str %t472, 1
  call void @veles_string_concat(ptr %a477, ptr %t473, i64 %t474, ptr %t475, i64 %t476)
  %t478 = load %str, ptr %a477
  call void @v_std.io.println(%str %t478)
  ret void
}

define i64 @v_main.main.lambda1(ptr %env, i64 %p1, i64 %p2) {
entry:
  %a1 = alloca ptr
  %a2 = alloca i64
  %a3 = alloca i64
  store ptr %env, ptr %a1
  store i64 %p1, ptr %a2
  store i64 %p2, ptr %a3
  %t4 = load i64, ptr %a3
  %t5 = call i64 @v_std.prelude.Comparable.i64.compareTo(ptr %a2, i64 %t4)
  ret i64 %t5
}

@.str.1 = private unnamed_addr constant [14 x i8] c"main.vs:11:22\00"
@.str.2 = private unnamed_addr constant [17 x i8] c"integer overflow\00"
@.str.3 = private unnamed_addr constant [13 x i8] c"main.vs:12:3\00"
@.str.4 = private unnamed_addr constant [2 x i8] c"a\00"
@.str.5 = private unnamed_addr constant [2 x i8] c"b\00"
@.str.6 = private unnamed_addr constant [16 x i8] c"a was set above\00"
@.str.7 = private unnamed_addr constant [14 x i8] c"main.vs:16:19\00"
@.str.8 = private unnamed_addr constant [13 x i8] c"main.vs:16:3\00"
@.str.9 = private unnamed_addr constant [14 x i8] c"main.vs:21:17\00"
@.str.10 = private unnamed_addr constant [2 x i8] c" \00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:21:35\00"
@.str.12 = private unnamed_addr constant [14 x i8] c"main.vs:21:68\00"
@.str.13 = private unnamed_addr constant [14 x i8] c"main.vs:21:83\00"
