%S.main.Counter = type { i64 }
@adesc.main.Counter = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 1, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a5 = alloca ptr
  %a22 = alloca %S.main.Counter
  %a39 = alloca %S.main.Counter
  %a45 = alloca ptr
  %a47 = alloca i64
  %a66 = alloca ptr
  %a81 = alloca ptr
  %a84 = alloca i64
  %a104 = alloca ptr
  %a109 = alloca %str
  %a110 = alloca i64
  %a115 = alloca %str
  %a116 = alloca i64
  %a121 = alloca %str
  %a124 = alloca ptr
  %a127 = alloca ptr
  %a134 = alloca ptr
  %a150 = alloca %str
  %a153 = alloca ptr
  %a156 = alloca i64
  %a162 = alloca i64
  %a167 = alloca ptr
  %a168 = alloca i64
  %a176 = alloca { i1, i64 }
  %a177 = alloca i1
  %a202 = alloca i64
  %a207 = alloca [21 x i8]
  %a212 = alloca ptr
  %a213 = alloca i64
  %a221 = alloca { i1, i64 }
  %a222 = alloca i1
  %a247 = alloca i64
  %a252 = alloca [21 x i8]
  %a257 = alloca ptr
  %a262 = alloca ptr
  %a263 = alloca i64
  %a271 = alloca { i1, %S.main.Counter }
  %a272 = alloca i1
  %a297 = alloca { i1, %S.main.Counter }
  %a298 = alloca { i1, i64 }
  %a310 = alloca ptr
  %a311 = alloca i64
  %a319 = alloca { i1, %S.main.Counter }
  %a320 = alloca i1
  %a345 = alloca { i1, %S.main.Counter }
  %a346 = alloca { i1, i64 }
  %a357 = alloca %str
  %a358 = alloca [9 x %str]
  %a375 = alloca %str
  %a378 = alloca { i1, i64 }
  %a384 = alloca i64
  %a389 = alloca [21 x i8]
  %a397 = alloca %str
  %a400 = alloca { i1, i64 }
  %a406 = alloca i64
  %a411 = alloca [21 x i8]
  %a417 = alloca [21 x i8]
  %a423 = alloca [21 x i8]
  %a429 = alloca i64
  %a432 = alloca %str
  %a434 = alloca %str
  %a435 = alloca [9 x %str]
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
  %t8 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t6, i32 0, i32 1
  %t9 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t6, i32 0, i32 2
  %t10 = load i64, ptr %t8
  %t11 = load i64, ptr %t9
  %t12 = icmp slt i64 %t10, %t11
  br i1 %t12, label %push.fast.1, label %push.grow.2, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.1:
  %t13 = load ptr, ptr %t6
  %t14 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t6, i32 0, i32 3
  %t15 = load i64, ptr %t14
  %t16 = mul i64 %t15, %t10
  %t17 = getelementptr inbounds i8, ptr %t13, i64 %t16
  store %S.main.Counter %t7, ptr %t17
  %t18 = add i64 %t10, 1
  store i64 %t18, ptr %t8
  %t19 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t6, i32 0, i32 5
  %t20 = load i64, ptr %t19
  %t21 = add i64 %t20, 1
  store i64 %t21, ptr %t19
  br label %push.done.3
push.grow.2:
  store %S.main.Counter %t7, ptr %a22
  call void @veles_list_push(ptr %t6, ptr %a22)
  br label %push.done.3
push.done.3:
  %t23 = load ptr, ptr %a5
  %t24 = insertvalue %S.main.Counter undef, i64 5, 0
  %t25 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t23, i32 0, i32 1
  %t26 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t23, i32 0, i32 2
  %t27 = load i64, ptr %t25
  %t28 = load i64, ptr %t26
  %t29 = icmp slt i64 %t27, %t28
  br i1 %t29, label %push.fast.4, label %push.grow.5, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.4:
  %t30 = load ptr, ptr %t23
  %t31 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t23, i32 0, i32 3
  %t32 = load i64, ptr %t31
  %t33 = mul i64 %t32, %t27
  %t34 = getelementptr inbounds i8, ptr %t30, i64 %t33
  store %S.main.Counter %t24, ptr %t34
  %t35 = add i64 %t27, 1
  store i64 %t35, ptr %t25
  %t36 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t23, i32 0, i32 5
  %t37 = load i64, ptr %t36
  %t38 = add i64 %t37, 1
  store i64 %t38, ptr %t36
  br label %push.done.6
push.grow.5:
  store %S.main.Counter %t24, ptr %a39
  call void @veles_list_push(ptr %t23, ptr %a39)
  br label %push.done.6
push.done.6:
  %t40 = load ptr, ptr %a5
  %t41 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t40, i32 0, i32 1
  %t42 = load i64, ptr %t41
  %t43 = icmp eq i64 %t42, 2
  br i1 %t43, label %if.then.7, label %if.end.8
if.then.7:
  %t44 = load ptr, ptr %a5
  store ptr %t44, ptr %a45
  %t46 = load ptr, ptr %a45
  store i64 0, ptr %a47
  %t48 = load i64, ptr %a47
  %t49 = icmp slt i64 %t48, 0
  br i1 %t49, label %if.then.9, label %if.end.10
if.then.9:
  %t50 = load i64, ptr %a47
  %t51 = load ptr, ptr %a45
  %t52 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t51, i32 0, i32 1
  %t53 = load i64, ptr %t52
  %t54 = add i64 %t50, %t53
  store i64 %t54, ptr %a47
  br label %if.end.10
if.end.10:
  %t55 = load i64, ptr %a47
  %t56 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t46, i32 0, i32 1
  %t57 = load i64, ptr %t56
  %t58 = icmp ult i64 %t55, %t57
  br i1 %t58, label %idx.ok.11, label %idx.bad.12, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.12:
  %t59 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t60 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t46, i64 %t55, ptr %t59, i64 %t60)
  unreachable
idx.ok.11:
  %t61 = load ptr, ptr %t46
  %t62 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t46, i32 0, i32 3
  %t63 = load i64, ptr %t62
  %t64 = mul i64 %t63, %t55
  %t65 = getelementptr inbounds i8, ptr %t61, i64 %t64
  store ptr %t65, ptr %a66
  %t67 = load ptr, ptr %a66
  %t68 = getelementptr inbounds %S.main.Counter, ptr %t67, i32 0, i32 0
  %t69 = load i64, ptr %t68
  %t71 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t69, i64 10)
  %t72 = extractvalue { i64, i1 } %t71, 0
  %t73 = extractvalue { i64, i1 } %t71, 1
  br i1 %t73, label %overflow.13, label %arith.ok.14
overflow.13:
  %t74 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t75 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t76 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t77 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_panic_at(ptr %t74, i64 %t75, ptr %t76, i64 %t77)
  unreachable
arith.ok.14:
  %t78 = load ptr, ptr %a66
  %t79 = getelementptr inbounds %S.main.Counter, ptr %t78, i32 0, i32 0
  store i64 %t72, ptr %t79
  br label %if.end.8
if.end.8:
  %t80 = load ptr, ptr %a5
  store ptr %t80, ptr %a81
  %t82 = insertvalue %S.main.Counter undef, i64 7, 0
  %t83 = load ptr, ptr %a81
  store i64 1, ptr %a84
  %t85 = load i64, ptr %a84
  %t86 = icmp slt i64 %t85, 0
  br i1 %t86, label %if.then.15, label %if.end.16
if.then.15:
  %t87 = load i64, ptr %a84
  %t88 = load ptr, ptr %a81
  %t89 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t88, i32 0, i32 1
  %t90 = load i64, ptr %t89
  %t91 = add i64 %t87, %t90
  store i64 %t91, ptr %a84
  br label %if.end.16
if.end.16:
  %t92 = load i64, ptr %a84
  %t93 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t83, i32 0, i32 1
  %t94 = load i64, ptr %t93
  %t95 = icmp ult i64 %t92, %t94
  br i1 %t95, label %idx.ok.17, label %idx.bad.18, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.18:
  %t96 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t97 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t83, i64 %t92, ptr %t96, i64 %t97)
  unreachable
idx.ok.17:
  %t98 = load ptr, ptr %t83
  %t99 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t83, i32 0, i32 3
  %t100 = load i64, ptr %t99
  %t101 = mul i64 %t100, %t92
  %t102 = getelementptr inbounds i8, ptr %t98, i64 %t101
  store %S.main.Counter %t82, ptr %t102
  %t103 = call ptr @veles_map_new(ptr @adesc.string, ptr @adesc.i64)
  store ptr %t103, ptr %a104
  %t105 = load ptr, ptr %a104
  %t107 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t108 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t106 = call i64 @veles_hash_bytes(ptr %t107, i64 %t108)
  store %str { ptr @.str.4, i64 1 }, ptr %a109
  store i64 1, ptr %a110
  call i64 @veles_map_insert(ptr %t105, i64 %t106, ptr %a109, ptr %a110, ptr @eqp.string)
  %t111 = load ptr, ptr %a104
  %t113 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t114 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t112 = call i64 @veles_hash_bytes(ptr %t113, i64 %t114)
  store %str { ptr @.str.5, i64 1 }, ptr %a115
  store i64 2, ptr %a116
  call i64 @veles_map_insert(ptr %t111, i64 %t112, ptr %a115, ptr %a116, ptr @eqp.string)
  %t117 = load ptr, ptr %a104
  %t119 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t120 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t118 = call i64 @veles_hash_bytes(ptr %t119, i64 %t120)
  store %str { ptr @.str.4, i64 1 }, ptr %a121
  %t122 = call i64 @veles_map_find(ptr %t117, i64 %t118, ptr %a121, ptr @eqp.string)
  %t123 = icmp sge i64 %t122, 0
  store ptr null, ptr %a124
  br i1 %t123, label %map.hit.19, label %map.end.20
map.hit.19:
  %t125 = call ptr @veles_map_val_at(ptr %t117, i64 %t122)
  store ptr %t125, ptr %a124
  br label %map.end.20
map.end.20:
  %t126 = load ptr, ptr %a124
  %t128 = icmp eq ptr %t126, null
  br i1 %t128, label %elvis.default.21, label %elvis.some.22
elvis.some.22:
  store ptr %t126, ptr %a127
  br label %elvis.end.23
elvis.default.21:
  %t129 = extractvalue %str { ptr @.str.6, i64 15 }, 0
  %t130 = extractvalue %str { ptr @.str.6, i64 15 }, 1
  %t131 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t132 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t129, i64 %t130, ptr %t131, i64 %t132)
  unreachable
elvis.end.23:
  %t133 = load ptr, ptr %a127
  store ptr %t133, ptr %a134
  %t135 = load ptr, ptr %a134
  %t136 = load i64, ptr %t135
  %t138 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t136, i64 40)
  %t139 = extractvalue { i64, i1 } %t138, 0
  %t140 = extractvalue { i64, i1 } %t138, 1
  br i1 %t140, label %overflow.24, label %arith.ok.25
overflow.24:
  %t141 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t142 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t143 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t144 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t141, i64 %t142, ptr %t143, i64 %t144)
  unreachable
arith.ok.25:
  %t145 = load ptr, ptr %a134
  store i64 %t139, ptr %t145
  %t146 = load ptr, ptr %a104
  %t148 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t149 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t147 = call i64 @veles_hash_bytes(ptr %t148, i64 %t149)
  store %str { ptr @.str.5, i64 1 }, ptr %a150
  %t151 = call i1 @veles_map_remove(ptr %t146, i64 %t147, ptr %a150, ptr @eqp.string)
  %t152 = call ptr @veles_map_new(ptr @adesc.i64, ptr null)
  store ptr %t152, ptr %a153
  %t154 = load ptr, ptr %a153
  store i64 4, ptr %a156
  %t157 = call i64 @veles_map_len(ptr %t154)
  call i64 @veles_map_insert(ptr %t154, i64 4, ptr %a156, ptr null, ptr @eqp.i64)
  %t158 = call i64 @veles_map_len(ptr %t154)
  %t159 = icmp ne i64 %t157, %t158
  %t160 = load ptr, ptr %a153
  store i64 4, ptr %a162
  %t163 = call i64 @veles_map_len(ptr %t160)
  call i64 @veles_map_insert(ptr %t160, i64 4, ptr %a162, ptr null, ptr @eqp.i64)
  %t164 = call i64 @veles_map_len(ptr %t160)
  %t165 = icmp ne i64 %t163, %t164
  %t166 = load ptr, ptr %a3
  store ptr %t166, ptr %a167
  store i64 1, ptr %a168
  %t169 = load i64, ptr %a168
  %t170 = icmp slt i64 %t169, 0
  br i1 %t170, label %if.then.26, label %if.end.27
if.then.26:
  %t171 = load i64, ptr %a168
  %t172 = load ptr, ptr %a167
  %t173 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t172, i32 0, i32 1
  %t174 = load i64, ptr %t173
  %t175 = add i64 %t171, %t174
  store i64 %t175, ptr %a168
  br label %if.end.27
if.end.27:
  %t178 = load i64, ptr %a168
  %t179 = icmp sge i64 %t178, 0
  store i1 %t179, ptr %a177
  br i1 %t179, label %sc.rhs.28, label %sc.end.29
sc.rhs.28:
  %t180 = load i64, ptr %a168
  %t181 = load ptr, ptr %a167
  %t182 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t181, i32 0, i32 1
  %t183 = load i64, ptr %t182
  %t184 = icmp slt i64 %t180, %t183
  store i1 %t184, ptr %a177
  br label %sc.end.29
sc.end.29:
  %t185 = load i1, ptr %a177
  br i1 %t185, label %if.then.30, label %if.else.32
if.then.30:
  %t186 = load ptr, ptr %a167
  %t187 = load i64, ptr %a168
  %t188 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t186, i32 0, i32 1
  %t189 = load i64, ptr %t188
  %t190 = icmp ult i64 %t187, %t189
  br i1 %t190, label %idx.ok.33, label %idx.bad.34, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.34:
  %t191 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t192 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t186, i64 %t187, ptr %t191, i64 %t192)
  unreachable
idx.ok.33:
  %t193 = load ptr, ptr %t186
  %t194 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t186, i32 0, i32 3
  %t195 = load i64, ptr %t194
  %t196 = mul i64 %t195, %t187
  %t197 = getelementptr inbounds i8, ptr %t193, i64 %t196
  %t198 = load i64, ptr %t197
  %t199 = insertvalue { i1, i64 } undef, i1 true, 0
  %t200 = insertvalue { i1, i64 } %t199, i64 %t198, 1
  store { i1, i64 } %t200, ptr %a176
  br label %if.end.31
if.else.32:
  store { i1, i64 } zeroinitializer, ptr %a176
  br label %if.end.31
if.end.31:
  %t201 = load { i1, i64 }, ptr %a176
  %t204 = extractvalue { i1, i64 } %t201, 0
  %t203 = xor i1 %t204, true
  br i1 %t203, label %elvis.default.35, label %elvis.some.36
elvis.some.36:
  %t205 = extractvalue { i1, i64 } %t201, 1
  store i64 %t205, ptr %a202
  br label %elvis.end.37
elvis.default.35:
  store i64 -1, ptr %a202
  br label %elvis.end.37
elvis.end.37:
  %t206 = load i64, ptr %a202
  %t208 = call i64 @veles_i64_format(ptr %a207, i64 %t206)
  %t209 = insertvalue %str undef, ptr %a207, 0
  %t210 = insertvalue %str %t209, i64 %t208, 1
  %t211 = load ptr, ptr %a3
  store ptr %t211, ptr %a212
  store i64 9, ptr %a213
  %t214 = load i64, ptr %a213
  %t215 = icmp slt i64 %t214, 0
  br i1 %t215, label %if.then.38, label %if.end.39
if.then.38:
  %t216 = load i64, ptr %a213
  %t217 = load ptr, ptr %a212
  %t218 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t217, i32 0, i32 1
  %t219 = load i64, ptr %t218
  %t220 = add i64 %t216, %t219
  store i64 %t220, ptr %a213
  br label %if.end.39
if.end.39:
  %t223 = load i64, ptr %a213
  %t224 = icmp sge i64 %t223, 0
  store i1 %t224, ptr %a222
  br i1 %t224, label %sc.rhs.40, label %sc.end.41
sc.rhs.40:
  %t225 = load i64, ptr %a213
  %t226 = load ptr, ptr %a212
  %t227 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t226, i32 0, i32 1
  %t228 = load i64, ptr %t227
  %t229 = icmp slt i64 %t225, %t228
  store i1 %t229, ptr %a222
  br label %sc.end.41
sc.end.41:
  %t230 = load i1, ptr %a222
  br i1 %t230, label %if.then.42, label %if.else.44
if.then.42:
  %t231 = load ptr, ptr %a212
  %t232 = load i64, ptr %a213
  %t233 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t231, i32 0, i32 1
  %t234 = load i64, ptr %t233
  %t235 = icmp ult i64 %t232, %t234
  br i1 %t235, label %idx.ok.45, label %idx.bad.46, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.46:
  %t236 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t237 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t231, i64 %t232, ptr %t236, i64 %t237)
  unreachable
idx.ok.45:
  %t238 = load ptr, ptr %t231
  %t239 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t231, i32 0, i32 3
  %t240 = load i64, ptr %t239
  %t241 = mul i64 %t240, %t232
  %t242 = getelementptr inbounds i8, ptr %t238, i64 %t241
  %t243 = load i64, ptr %t242
  %t244 = insertvalue { i1, i64 } undef, i1 true, 0
  %t245 = insertvalue { i1, i64 } %t244, i64 %t243, 1
  store { i1, i64 } %t245, ptr %a221
  br label %if.end.43
if.else.44:
  store { i1, i64 } zeroinitializer, ptr %a221
  br label %if.end.43
if.end.43:
  %t246 = load { i1, i64 }, ptr %a221
  %t249 = extractvalue { i1, i64 } %t246, 0
  %t248 = xor i1 %t249, true
  br i1 %t248, label %elvis.default.47, label %elvis.some.48
elvis.some.48:
  %t250 = extractvalue { i1, i64 } %t246, 1
  store i64 %t250, ptr %a247
  br label %elvis.end.49
elvis.default.47:
  store i64 -1, ptr %a247
  br label %elvis.end.49
elvis.end.49:
  %t251 = load i64, ptr %a247
  %t253 = call i64 @veles_i64_format(ptr %a252, i64 %t251)
  %t254 = insertvalue %str undef, ptr %a252, 0
  %t255 = insertvalue %str %t254, i64 %t253, 1
  %t256 = load ptr, ptr %a3
  store ptr %t256, ptr %a257
  %t258 = load ptr, ptr %a257
  %t259 = call ptr @veles_list_copy(ptr %t258)
  call void @veles_list_sort_native(ptr %t259, i32 4)
  %t260 = call %str @show.List_i64_(ptr %t259)
  %t261 = load ptr, ptr %a5
  store ptr %t261, ptr %a262
  store i64 0, ptr %a263
  %t264 = load i64, ptr %a263
  %t265 = icmp slt i64 %t264, 0
  br i1 %t265, label %if.then.50, label %if.end.51
if.then.50:
  %t266 = load i64, ptr %a263
  %t267 = load ptr, ptr %a262
  %t268 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t267, i32 0, i32 1
  %t269 = load i64, ptr %t268
  %t270 = add i64 %t266, %t269
  store i64 %t270, ptr %a263
  br label %if.end.51
if.end.51:
  %t273 = load i64, ptr %a263
  %t274 = icmp sge i64 %t273, 0
  store i1 %t274, ptr %a272
  br i1 %t274, label %sc.rhs.52, label %sc.end.53
sc.rhs.52:
  %t275 = load i64, ptr %a263
  %t276 = load ptr, ptr %a262
  %t277 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t276, i32 0, i32 1
  %t278 = load i64, ptr %t277
  %t279 = icmp slt i64 %t275, %t278
  store i1 %t279, ptr %a272
  br label %sc.end.53
sc.end.53:
  %t280 = load i1, ptr %a272
  br i1 %t280, label %if.then.54, label %if.else.56
if.then.54:
  %t281 = load ptr, ptr %a262
  %t282 = load i64, ptr %a263
  %t283 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t281, i32 0, i32 1
  %t284 = load i64, ptr %t283
  %t285 = icmp ult i64 %t282, %t284
  br i1 %t285, label %idx.ok.57, label %idx.bad.58, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.58:
  %t286 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t287 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t281, i64 %t282, ptr %t286, i64 %t287)
  unreachable
idx.ok.57:
  %t288 = load ptr, ptr %t281
  %t289 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t281, i32 0, i32 3
  %t290 = load i64, ptr %t289
  %t291 = mul i64 %t290, %t282
  %t292 = getelementptr inbounds i8, ptr %t288, i64 %t291
  %t293 = load %S.main.Counter, ptr %t292
  %t294 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t295 = insertvalue { i1, %S.main.Counter } %t294, %S.main.Counter %t293, 1
  store { i1, %S.main.Counter } %t295, ptr %a271
  br label %if.end.55
if.else.56:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a271
  br label %if.end.55
if.end.55:
  %t296 = load { i1, %S.main.Counter }, ptr %a271
  store { i1, %S.main.Counter } %t296, ptr %a297
  %t299 = load { i1, %S.main.Counter }, ptr %a297
  %t301 = extractvalue { i1, %S.main.Counter } %t299, 0
  %t300 = xor i1 %t301, true
  br i1 %t300, label %if.then.59, label %if.else.61
if.then.59:
  store { i1, i64 } zeroinitializer, ptr %a298
  br label %if.end.60
if.else.61:
  %t302 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a297, i32 0, i32 1
  %t303 = getelementptr inbounds %S.main.Counter, ptr %t302, i32 0, i32 0
  %t304 = load i64, ptr %t303
  %t305 = insertvalue { i1, i64 } undef, i1 true, 0
  %t306 = insertvalue { i1, i64 } %t305, i64 %t304, 1
  store { i1, i64 } %t306, ptr %a298
  br label %if.end.60
if.end.60:
  %t307 = load { i1, i64 }, ptr %a298
  %t308 = call %str @show.T_i64_N({ i1, i64 } %t307)
  %t309 = load ptr, ptr %a5
  store ptr %t309, ptr %a310
  store i64 1, ptr %a311
  %t312 = load i64, ptr %a311
  %t313 = icmp slt i64 %t312, 0
  br i1 %t313, label %if.then.62, label %if.end.63
if.then.62:
  %t314 = load i64, ptr %a311
  %t315 = load ptr, ptr %a310
  %t316 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t315, i32 0, i32 1
  %t317 = load i64, ptr %t316
  %t318 = add i64 %t314, %t317
  store i64 %t318, ptr %a311
  br label %if.end.63
if.end.63:
  %t321 = load i64, ptr %a311
  %t322 = icmp sge i64 %t321, 0
  store i1 %t322, ptr %a320
  br i1 %t322, label %sc.rhs.64, label %sc.end.65
sc.rhs.64:
  %t323 = load i64, ptr %a311
  %t324 = load ptr, ptr %a310
  %t325 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t324, i32 0, i32 1
  %t326 = load i64, ptr %t325
  %t327 = icmp slt i64 %t323, %t326
  store i1 %t327, ptr %a320
  br label %sc.end.65
sc.end.65:
  %t328 = load i1, ptr %a320
  br i1 %t328, label %if.then.66, label %if.else.68
if.then.66:
  %t329 = load ptr, ptr %a310
  %t330 = load i64, ptr %a311
  %t331 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t329, i32 0, i32 1
  %t332 = load i64, ptr %t331
  %t333 = icmp ult i64 %t330, %t332
  br i1 %t333, label %idx.ok.69, label %idx.bad.70, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.70:
  %t334 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t335 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t329, i64 %t330, ptr %t334, i64 %t335)
  unreachable
idx.ok.69:
  %t336 = load ptr, ptr %t329
  %t337 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64 }, ptr %t329, i32 0, i32 3
  %t338 = load i64, ptr %t337
  %t339 = mul i64 %t338, %t330
  %t340 = getelementptr inbounds i8, ptr %t336, i64 %t339
  %t341 = load %S.main.Counter, ptr %t340
  %t342 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t343 = insertvalue { i1, %S.main.Counter } %t342, %S.main.Counter %t341, 1
  store { i1, %S.main.Counter } %t343, ptr %a319
  br label %if.end.67
if.else.68:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a319
  br label %if.end.67
if.end.67:
  %t344 = load { i1, %S.main.Counter }, ptr %a319
  store { i1, %S.main.Counter } %t344, ptr %a345
  %t347 = load { i1, %S.main.Counter }, ptr %a345
  %t349 = extractvalue { i1, %S.main.Counter } %t347, 0
  %t348 = xor i1 %t349, true
  br i1 %t348, label %if.then.71, label %if.else.73
if.then.71:
  store { i1, i64 } zeroinitializer, ptr %a346
  br label %if.end.72
if.else.73:
  %t350 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a345, i32 0, i32 1
  %t351 = getelementptr inbounds %S.main.Counter, ptr %t350, i32 0, i32 0
  %t352 = load i64, ptr %t351
  %t353 = insertvalue { i1, i64 } undef, i1 true, 0
  %t354 = insertvalue { i1, i64 } %t353, i64 %t352, 1
  store { i1, i64 } %t354, ptr %a346
  br label %if.end.72
if.end.72:
  %t355 = load { i1, i64 }, ptr %a346
  %t356 = call %str @show.T_i64_N({ i1, i64 } %t355)
  %t359 = getelementptr [9 x %str], ptr %a358, i64 0, i64 0
  store %str %t210, ptr %t359
  %t360 = getelementptr [9 x %str], ptr %a358, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t360
  %t361 = getelementptr [9 x %str], ptr %a358, i64 0, i64 2
  store %str %t255, ptr %t361
  %t362 = getelementptr [9 x %str], ptr %a358, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t362
  %t363 = getelementptr [9 x %str], ptr %a358, i64 0, i64 4
  store %str %t260, ptr %t363
  %t364 = getelementptr [9 x %str], ptr %a358, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t364
  %t365 = getelementptr [9 x %str], ptr %a358, i64 0, i64 6
  store %str %t308, ptr %t365
  %t366 = getelementptr [9 x %str], ptr %a358, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t366
  %t367 = getelementptr [9 x %str], ptr %a358, i64 0, i64 8
  store %str %t356, ptr %t367
  call void @veles_string_concat_n(ptr %a357, ptr %a358, i64 9)
  %t368 = load %str, ptr %a357
  %t369 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t370 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t369)
  call void @v_std.io.println(%str %t368)
  call void @veles_call_pop()
  %t371 = load ptr, ptr %a104
  %t373 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t374 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t372 = call i64 @veles_hash_bytes(ptr %t373, i64 %t374)
  store %str { ptr @.str.4, i64 1 }, ptr %a375
  %t376 = call i64 @veles_map_find(ptr %t371, i64 %t372, ptr %a375, ptr @eqp.string)
  %t377 = icmp sge i64 %t376, 0
  store { i1, i64 } zeroinitializer, ptr %a378
  br i1 %t377, label %map.hit.74, label %map.end.75
map.hit.74:
  %t379 = call ptr @veles_map_val_at(ptr %t371, i64 %t376)
  %t380 = load i64, ptr %t379
  %t381 = insertvalue { i1, i64 } undef, i1 true, 0
  %t382 = insertvalue { i1, i64 } %t381, i64 %t380, 1
  store { i1, i64 } %t382, ptr %a378
  br label %map.end.75
map.end.75:
  %t383 = load { i1, i64 }, ptr %a378
  %t386 = extractvalue { i1, i64 } %t383, 0
  %t385 = xor i1 %t386, true
  br i1 %t385, label %elvis.default.76, label %elvis.some.77
elvis.some.77:
  %t387 = extractvalue { i1, i64 } %t383, 1
  store i64 %t387, ptr %a384
  br label %elvis.end.78
elvis.default.76:
  store i64 0, ptr %a384
  br label %elvis.end.78
elvis.end.78:
  %t388 = load i64, ptr %a384
  %t390 = call i64 @veles_i64_format(ptr %a389, i64 %t388)
  %t391 = insertvalue %str undef, ptr %a389, 0
  %t392 = insertvalue %str %t391, i64 %t390, 1
  %t393 = load ptr, ptr %a104
  %t395 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t396 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t394 = call i64 @veles_hash_bytes(ptr %t395, i64 %t396)
  store %str { ptr @.str.5, i64 1 }, ptr %a397
  %t398 = call i64 @veles_map_find(ptr %t393, i64 %t394, ptr %a397, ptr @eqp.string)
  %t399 = icmp sge i64 %t398, 0
  store { i1, i64 } zeroinitializer, ptr %a400
  br i1 %t399, label %map.hit.79, label %map.end.80
map.hit.79:
  %t401 = call ptr @veles_map_val_at(ptr %t393, i64 %t398)
  %t402 = load i64, ptr %t401
  %t403 = insertvalue { i1, i64 } undef, i1 true, 0
  %t404 = insertvalue { i1, i64 } %t403, i64 %t402, 1
  store { i1, i64 } %t404, ptr %a400
  br label %map.end.80
map.end.80:
  %t405 = load { i1, i64 }, ptr %a400
  %t408 = extractvalue { i1, i64 } %t405, 0
  %t407 = xor i1 %t408, true
  br i1 %t407, label %elvis.default.81, label %elvis.some.82
elvis.some.82:
  %t409 = extractvalue { i1, i64 } %t405, 1
  store i64 %t409, ptr %a406
  br label %elvis.end.83
elvis.default.81:
  store i64 0, ptr %a406
  br label %elvis.end.83
elvis.end.83:
  %t410 = load i64, ptr %a406
  %t412 = call i64 @veles_i64_format(ptr %a411, i64 %t410)
  %t413 = insertvalue %str undef, ptr %a411, 0
  %t414 = insertvalue %str %t413, i64 %t412, 1
  %t415 = load ptr, ptr %a104
  %t416 = call i64 @veles_map_len(ptr %t415)
  %t418 = call i64 @veles_i64_format(ptr %a417, i64 %t416)
  %t419 = insertvalue %str undef, ptr %a417, 0
  %t420 = insertvalue %str %t419, i64 %t418, 1
  %t421 = load ptr, ptr %a153
  %t422 = call i64 @veles_map_len(ptr %t421)
  %t424 = call i64 @veles_i64_format(ptr %a423, i64 %t422)
  %t425 = insertvalue %str undef, ptr %a423, 0
  %t426 = insertvalue %str %t425, i64 %t424, 1
  %t427 = load ptr, ptr %a153
  store i64 4, ptr %a429
  %t430 = call i64 @veles_map_find(ptr %t427, i64 4, ptr %a429, ptr @eqp.i64)
  %t431 = icmp sge i64 %t430, 0
  call void @veles_bool_to_string(ptr %a432, i1 zeroext %t431)
  %t433 = load %str, ptr %a432
  %t436 = getelementptr [9 x %str], ptr %a435, i64 0, i64 0
  store %str %t392, ptr %t436
  %t437 = getelementptr [9 x %str], ptr %a435, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t437
  %t438 = getelementptr [9 x %str], ptr %a435, i64 0, i64 2
  store %str %t414, ptr %t438
  %t439 = getelementptr [9 x %str], ptr %a435, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t439
  %t440 = getelementptr [9 x %str], ptr %a435, i64 0, i64 4
  store %str %t420, ptr %t440
  %t441 = getelementptr [9 x %str], ptr %a435, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t441
  %t442 = getelementptr [9 x %str], ptr %a435, i64 0, i64 6
  store %str %t426, ptr %t442
  %t443 = getelementptr [9 x %str], ptr %a435, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t443
  %t444 = getelementptr [9 x %str], ptr %a435, i64 0, i64 8
  store %str %t433, ptr %t444
  call void @veles_string_concat_n(ptr %a434, ptr %a435, i64 9)
  %t445 = load %str, ptr %a434
  %t446 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t447 = extractvalue %str { ptr @.str.15, i64 20 }, 1
  call void @veles_call_push(ptr %t446)
  call void @v_std.io.println(%str %t445)
  call void @veles_call_pop()
  ret void
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
@.str.10 = private unnamed_addr constant [14 x i8] c"main.vs:21:35\00"
@.str.11 = private unnamed_addr constant [14 x i8] c"main.vs:21:68\00"
@.str.12 = private unnamed_addr constant [14 x i8] c"main.vs:21:83\00"
@.str.13 = private unnamed_addr constant [2 x i8] c" \00"
@.str.14 = private unnamed_addr constant [21 x i8] c"main.vs:21:3\00println\00"
@.str.15 = private unnamed_addr constant [21 x i8] c"main.vs:22:3\00println\00"
