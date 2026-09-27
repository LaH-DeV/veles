%S.main.Counter = type { i64 }
@adesc.main.Counter = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 1, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a5 = alloca ptr
  %a19 = alloca %S.main.Counter
  %a33 = alloca %S.main.Counter
  %a39 = alloca ptr
  %a41 = alloca i64
  %a60 = alloca ptr
  %a75 = alloca ptr
  %a78 = alloca i64
  %a98 = alloca ptr
  %a103 = alloca %str
  %a104 = alloca i64
  %a109 = alloca %str
  %a110 = alloca i64
  %a115 = alloca %str
  %a118 = alloca ptr
  %a121 = alloca ptr
  %a128 = alloca ptr
  %a144 = alloca %str
  %a147 = alloca ptr
  %a150 = alloca i64
  %a156 = alloca i64
  %a161 = alloca ptr
  %a162 = alloca i64
  %a170 = alloca { i1, i64 }
  %a171 = alloca i1
  %a196 = alloca i64
  %a201 = alloca [21 x i8]
  %a206 = alloca ptr
  %a207 = alloca i64
  %a215 = alloca { i1, i64 }
  %a216 = alloca i1
  %a241 = alloca i64
  %a246 = alloca [21 x i8]
  %a251 = alloca ptr
  %a256 = alloca ptr
  %a257 = alloca i64
  %a265 = alloca { i1, %S.main.Counter }
  %a266 = alloca i1
  %a291 = alloca { i1, %S.main.Counter }
  %a292 = alloca { i1, i64 }
  %a304 = alloca ptr
  %a305 = alloca i64
  %a313 = alloca { i1, %S.main.Counter }
  %a314 = alloca i1
  %a339 = alloca { i1, %S.main.Counter }
  %a340 = alloca { i1, i64 }
  %a351 = alloca %str
  %a352 = alloca [9 x %str]
  %a367 = alloca %str
  %a370 = alloca { i1, i64 }
  %a376 = alloca i64
  %a381 = alloca [21 x i8]
  %a389 = alloca %str
  %a392 = alloca { i1, i64 }
  %a398 = alloca i64
  %a403 = alloca [21 x i8]
  %a409 = alloca [21 x i8]
  %a415 = alloca [21 x i8]
  %a421 = alloca i64
  %a424 = alloca %str
  %a426 = alloca %str
  %a427 = alloca [9 x %str]
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
  %t8 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t6, i32 0, i32 1
  %t9 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t6, i32 0, i32 2
  %t10 = load i64, ptr %t8
  %t11 = load i64, ptr %t9
  %t12 = icmp slt i64 %t10, %t11
  br i1 %t12, label %push.fast.1, label %push.grow.2, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.1:
  %t13 = load ptr, ptr %t6
  %t14 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t6, i32 0, i32 3
  %t15 = load i64, ptr %t14
  %t16 = mul i64 %t15, %t10
  %t17 = getelementptr inbounds i8, ptr %t13, i64 %t16
  store %S.main.Counter %t7, ptr %t17
  %t18 = add i64 %t10, 1
  store i64 %t18, ptr %t8
  br label %push.done.3
push.grow.2:
  store %S.main.Counter %t7, ptr %a19
  call void @veles_list_push(ptr %t6, ptr %a19)
  br label %push.done.3
push.done.3:
  %t20 = load ptr, ptr %a5
  %t21 = insertvalue %S.main.Counter undef, i64 5, 0
  %t22 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t20, i32 0, i32 1
  %t23 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t20, i32 0, i32 2
  %t24 = load i64, ptr %t22
  %t25 = load i64, ptr %t23
  %t26 = icmp slt i64 %t24, %t25
  br i1 %t26, label %push.fast.4, label %push.grow.5, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.4:
  %t27 = load ptr, ptr %t20
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t20, i32 0, i32 3
  %t29 = load i64, ptr %t28
  %t30 = mul i64 %t29, %t24
  %t31 = getelementptr inbounds i8, ptr %t27, i64 %t30
  store %S.main.Counter %t21, ptr %t31
  %t32 = add i64 %t24, 1
  store i64 %t32, ptr %t22
  br label %push.done.6
push.grow.5:
  store %S.main.Counter %t21, ptr %a33
  call void @veles_list_push(ptr %t20, ptr %a33)
  br label %push.done.6
push.done.6:
  %t34 = load ptr, ptr %a5
  %t35 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t34, i32 0, i32 1
  %t36 = load i64, ptr %t35
  %t37 = icmp eq i64 %t36, 2
  br i1 %t37, label %if.then.7, label %if.end.8
if.then.7:
  %t38 = load ptr, ptr %a5
  store ptr %t38, ptr %a39
  %t40 = load ptr, ptr %a39
  store i64 0, ptr %a41
  %t42 = load i64, ptr %a41
  %t43 = icmp slt i64 %t42, 0
  br i1 %t43, label %if.then.9, label %if.end.10
if.then.9:
  %t44 = load i64, ptr %a41
  %t45 = load ptr, ptr %a39
  %t46 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t45, i32 0, i32 1
  %t47 = load i64, ptr %t46
  %t48 = add i64 %t44, %t47
  store i64 %t48, ptr %a41
  br label %if.end.10
if.end.10:
  %t49 = load i64, ptr %a41
  %t50 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t40, i32 0, i32 1
  %t51 = load i64, ptr %t50
  %t52 = icmp ult i64 %t49, %t51
  br i1 %t52, label %idx.ok.11, label %idx.bad.12, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.12:
  %t53 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t54 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t40, i64 %t49, ptr %t53, i64 %t54)
  unreachable
idx.ok.11:
  %t55 = load ptr, ptr %t40
  %t56 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t40, i32 0, i32 3
  %t57 = load i64, ptr %t56
  %t58 = mul i64 %t57, %t49
  %t59 = getelementptr inbounds i8, ptr %t55, i64 %t58
  store ptr %t59, ptr %a60
  %t61 = load ptr, ptr %a60
  %t62 = getelementptr inbounds %S.main.Counter, ptr %t61, i32 0, i32 0
  %t63 = load i64, ptr %t62
  %t65 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t63, i64 10)
  %t66 = extractvalue { i64, i1 } %t65, 0
  %t67 = extractvalue { i64, i1 } %t65, 1
  br i1 %t67, label %overflow.13, label %arith.ok.14
overflow.13:
  %t68 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t69 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t70 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t71 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_panic_at(ptr %t68, i64 %t69, ptr %t70, i64 %t71)
  unreachable
arith.ok.14:
  %t72 = load ptr, ptr %a60
  %t73 = getelementptr inbounds %S.main.Counter, ptr %t72, i32 0, i32 0
  store i64 %t66, ptr %t73
  br label %if.end.8
if.end.8:
  %t74 = load ptr, ptr %a5
  store ptr %t74, ptr %a75
  %t76 = insertvalue %S.main.Counter undef, i64 7, 0
  %t77 = load ptr, ptr %a75
  store i64 1, ptr %a78
  %t79 = load i64, ptr %a78
  %t80 = icmp slt i64 %t79, 0
  br i1 %t80, label %if.then.15, label %if.end.16
if.then.15:
  %t81 = load i64, ptr %a78
  %t82 = load ptr, ptr %a75
  %t83 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t82, i32 0, i32 1
  %t84 = load i64, ptr %t83
  %t85 = add i64 %t81, %t84
  store i64 %t85, ptr %a78
  br label %if.end.16
if.end.16:
  %t86 = load i64, ptr %a78
  %t87 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t77, i32 0, i32 1
  %t88 = load i64, ptr %t87
  %t89 = icmp ult i64 %t86, %t88
  br i1 %t89, label %idx.ok.17, label %idx.bad.18, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.18:
  %t90 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t91 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t77, i64 %t86, ptr %t90, i64 %t91)
  unreachable
idx.ok.17:
  %t92 = load ptr, ptr %t77
  %t93 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t77, i32 0, i32 3
  %t94 = load i64, ptr %t93
  %t95 = mul i64 %t94, %t86
  %t96 = getelementptr inbounds i8, ptr %t92, i64 %t95
  store %S.main.Counter %t76, ptr %t96
  %t97 = call ptr @veles_map_new(ptr @adesc.string, ptr @adesc.i64)
  store ptr %t97, ptr %a98
  %t99 = load ptr, ptr %a98
  %t101 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t102 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t100 = call i64 @veles_hash_bytes(ptr %t101, i64 %t102)
  store %str { ptr @.str.4, i64 1 }, ptr %a103
  store i64 1, ptr %a104
  call i64 @veles_map_insert(ptr %t99, i64 %t100, ptr %a103, ptr %a104, ptr @eqp.string)
  %t105 = load ptr, ptr %a98
  %t107 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t108 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t106 = call i64 @veles_hash_bytes(ptr %t107, i64 %t108)
  store %str { ptr @.str.5, i64 1 }, ptr %a109
  store i64 2, ptr %a110
  call i64 @veles_map_insert(ptr %t105, i64 %t106, ptr %a109, ptr %a110, ptr @eqp.string)
  %t111 = load ptr, ptr %a98
  %t113 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t114 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t112 = call i64 @veles_hash_bytes(ptr %t113, i64 %t114)
  store %str { ptr @.str.4, i64 1 }, ptr %a115
  %t116 = call i64 @veles_map_find(ptr %t111, i64 %t112, ptr %a115, ptr @eqp.string)
  %t117 = icmp sge i64 %t116, 0
  store ptr null, ptr %a118
  br i1 %t117, label %map.hit.19, label %map.end.20
map.hit.19:
  %t119 = call ptr @veles_map_val_at(ptr %t111, i64 %t116)
  store ptr %t119, ptr %a118
  br label %map.end.20
map.end.20:
  %t120 = load ptr, ptr %a118
  %t122 = icmp eq ptr %t120, null
  br i1 %t122, label %elvis.default.21, label %elvis.some.22
elvis.some.22:
  store ptr %t120, ptr %a121
  br label %elvis.end.23
elvis.default.21:
  %t123 = extractvalue %str { ptr @.str.6, i64 15 }, 0
  %t124 = extractvalue %str { ptr @.str.6, i64 15 }, 1
  %t125 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t126 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t123, i64 %t124, ptr %t125, i64 %t126)
  unreachable
elvis.end.23:
  %t127 = load ptr, ptr %a121
  store ptr %t127, ptr %a128
  %t129 = load ptr, ptr %a128
  %t130 = load i64, ptr %t129
  %t132 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t130, i64 40)
  %t133 = extractvalue { i64, i1 } %t132, 0
  %t134 = extractvalue { i64, i1 } %t132, 1
  br i1 %t134, label %overflow.24, label %arith.ok.25
overflow.24:
  %t135 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t136 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t137 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t138 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t135, i64 %t136, ptr %t137, i64 %t138)
  unreachable
arith.ok.25:
  %t139 = load ptr, ptr %a128
  store i64 %t133, ptr %t139
  %t140 = load ptr, ptr %a98
  %t142 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t143 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t141 = call i64 @veles_hash_bytes(ptr %t142, i64 %t143)
  store %str { ptr @.str.5, i64 1 }, ptr %a144
  %t145 = call i1 @veles_map_remove(ptr %t140, i64 %t141, ptr %a144, ptr @eqp.string)
  %t146 = call ptr @veles_map_new(ptr @adesc.i64, ptr null)
  store ptr %t146, ptr %a147
  %t148 = load ptr, ptr %a147
  store i64 4, ptr %a150
  %t151 = call i64 @veles_map_len(ptr %t148)
  call i64 @veles_map_insert(ptr %t148, i64 4, ptr %a150, ptr null, ptr @eqp.i64)
  %t152 = call i64 @veles_map_len(ptr %t148)
  %t153 = icmp ne i64 %t151, %t152
  %t154 = load ptr, ptr %a147
  store i64 4, ptr %a156
  %t157 = call i64 @veles_map_len(ptr %t154)
  call i64 @veles_map_insert(ptr %t154, i64 4, ptr %a156, ptr null, ptr @eqp.i64)
  %t158 = call i64 @veles_map_len(ptr %t154)
  %t159 = icmp ne i64 %t157, %t158
  %t160 = load ptr, ptr %a3
  store ptr %t160, ptr %a161
  store i64 1, ptr %a162
  %t163 = load i64, ptr %a162
  %t164 = icmp slt i64 %t163, 0
  br i1 %t164, label %if.then.26, label %if.end.27
if.then.26:
  %t165 = load i64, ptr %a162
  %t166 = load ptr, ptr %a161
  %t167 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t166, i32 0, i32 1
  %t168 = load i64, ptr %t167
  %t169 = add i64 %t165, %t168
  store i64 %t169, ptr %a162
  br label %if.end.27
if.end.27:
  %t172 = load i64, ptr %a162
  %t173 = icmp sge i64 %t172, 0
  store i1 %t173, ptr %a171
  br i1 %t173, label %sc.rhs.28, label %sc.end.29
sc.rhs.28:
  %t174 = load i64, ptr %a162
  %t175 = load ptr, ptr %a161
  %t176 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t175, i32 0, i32 1
  %t177 = load i64, ptr %t176
  %t178 = icmp slt i64 %t174, %t177
  store i1 %t178, ptr %a171
  br label %sc.end.29
sc.end.29:
  %t179 = load i1, ptr %a171
  br i1 %t179, label %if.then.30, label %if.else.32
if.then.30:
  %t180 = load ptr, ptr %a161
  %t181 = load i64, ptr %a162
  %t182 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t180, i32 0, i32 1
  %t183 = load i64, ptr %t182
  %t184 = icmp ult i64 %t181, %t183
  br i1 %t184, label %idx.ok.33, label %idx.bad.34, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.34:
  %t185 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t186 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t180, i64 %t181, ptr %t185, i64 %t186)
  unreachable
idx.ok.33:
  %t187 = load ptr, ptr %t180
  %t188 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t180, i32 0, i32 3
  %t189 = load i64, ptr %t188
  %t190 = mul i64 %t189, %t181
  %t191 = getelementptr inbounds i8, ptr %t187, i64 %t190
  %t192 = load i64, ptr %t191
  %t193 = insertvalue { i1, i64 } undef, i1 true, 0
  %t194 = insertvalue { i1, i64 } %t193, i64 %t192, 1
  store { i1, i64 } %t194, ptr %a170
  br label %if.end.31
if.else.32:
  store { i1, i64 } zeroinitializer, ptr %a170
  br label %if.end.31
if.end.31:
  %t195 = load { i1, i64 }, ptr %a170
  %t198 = extractvalue { i1, i64 } %t195, 0
  %t197 = xor i1 %t198, true
  br i1 %t197, label %elvis.default.35, label %elvis.some.36
elvis.some.36:
  %t199 = extractvalue { i1, i64 } %t195, 1
  store i64 %t199, ptr %a196
  br label %elvis.end.37
elvis.default.35:
  store i64 -1, ptr %a196
  br label %elvis.end.37
elvis.end.37:
  %t200 = load i64, ptr %a196
  %t202 = call i64 @veles_i64_format(ptr %a201, i64 %t200)
  %t203 = insertvalue %str undef, ptr %a201, 0
  %t204 = insertvalue %str %t203, i64 %t202, 1
  %t205 = load ptr, ptr %a3
  store ptr %t205, ptr %a206
  store i64 9, ptr %a207
  %t208 = load i64, ptr %a207
  %t209 = icmp slt i64 %t208, 0
  br i1 %t209, label %if.then.38, label %if.end.39
if.then.38:
  %t210 = load i64, ptr %a207
  %t211 = load ptr, ptr %a206
  %t212 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t211, i32 0, i32 1
  %t213 = load i64, ptr %t212
  %t214 = add i64 %t210, %t213
  store i64 %t214, ptr %a207
  br label %if.end.39
if.end.39:
  %t217 = load i64, ptr %a207
  %t218 = icmp sge i64 %t217, 0
  store i1 %t218, ptr %a216
  br i1 %t218, label %sc.rhs.40, label %sc.end.41
sc.rhs.40:
  %t219 = load i64, ptr %a207
  %t220 = load ptr, ptr %a206
  %t221 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t220, i32 0, i32 1
  %t222 = load i64, ptr %t221
  %t223 = icmp slt i64 %t219, %t222
  store i1 %t223, ptr %a216
  br label %sc.end.41
sc.end.41:
  %t224 = load i1, ptr %a216
  br i1 %t224, label %if.then.42, label %if.else.44
if.then.42:
  %t225 = load ptr, ptr %a206
  %t226 = load i64, ptr %a207
  %t227 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t225, i32 0, i32 1
  %t228 = load i64, ptr %t227
  %t229 = icmp ult i64 %t226, %t228
  br i1 %t229, label %idx.ok.45, label %idx.bad.46, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.46:
  %t230 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t231 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t225, i64 %t226, ptr %t230, i64 %t231)
  unreachable
idx.ok.45:
  %t232 = load ptr, ptr %t225
  %t233 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t225, i32 0, i32 3
  %t234 = load i64, ptr %t233
  %t235 = mul i64 %t234, %t226
  %t236 = getelementptr inbounds i8, ptr %t232, i64 %t235
  %t237 = load i64, ptr %t236
  %t238 = insertvalue { i1, i64 } undef, i1 true, 0
  %t239 = insertvalue { i1, i64 } %t238, i64 %t237, 1
  store { i1, i64 } %t239, ptr %a215
  br label %if.end.43
if.else.44:
  store { i1, i64 } zeroinitializer, ptr %a215
  br label %if.end.43
if.end.43:
  %t240 = load { i1, i64 }, ptr %a215
  %t243 = extractvalue { i1, i64 } %t240, 0
  %t242 = xor i1 %t243, true
  br i1 %t242, label %elvis.default.47, label %elvis.some.48
elvis.some.48:
  %t244 = extractvalue { i1, i64 } %t240, 1
  store i64 %t244, ptr %a241
  br label %elvis.end.49
elvis.default.47:
  store i64 -1, ptr %a241
  br label %elvis.end.49
elvis.end.49:
  %t245 = load i64, ptr %a241
  %t247 = call i64 @veles_i64_format(ptr %a246, i64 %t245)
  %t248 = insertvalue %str undef, ptr %a246, 0
  %t249 = insertvalue %str %t248, i64 %t247, 1
  %t250 = load ptr, ptr %a3
  store ptr %t250, ptr %a251
  %t252 = load ptr, ptr %a251
  %t253 = call ptr @veles_list_copy(ptr %t252)
  call void @veles_list_sort_native(ptr %t253, i32 4)
  %t254 = call %str @show.List_i64_(ptr %t253)
  %t255 = load ptr, ptr %a5
  store ptr %t255, ptr %a256
  store i64 0, ptr %a257
  %t258 = load i64, ptr %a257
  %t259 = icmp slt i64 %t258, 0
  br i1 %t259, label %if.then.50, label %if.end.51
if.then.50:
  %t260 = load i64, ptr %a257
  %t261 = load ptr, ptr %a256
  %t262 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t261, i32 0, i32 1
  %t263 = load i64, ptr %t262
  %t264 = add i64 %t260, %t263
  store i64 %t264, ptr %a257
  br label %if.end.51
if.end.51:
  %t267 = load i64, ptr %a257
  %t268 = icmp sge i64 %t267, 0
  store i1 %t268, ptr %a266
  br i1 %t268, label %sc.rhs.52, label %sc.end.53
sc.rhs.52:
  %t269 = load i64, ptr %a257
  %t270 = load ptr, ptr %a256
  %t271 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t270, i32 0, i32 1
  %t272 = load i64, ptr %t271
  %t273 = icmp slt i64 %t269, %t272
  store i1 %t273, ptr %a266
  br label %sc.end.53
sc.end.53:
  %t274 = load i1, ptr %a266
  br i1 %t274, label %if.then.54, label %if.else.56
if.then.54:
  %t275 = load ptr, ptr %a256
  %t276 = load i64, ptr %a257
  %t277 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t275, i32 0, i32 1
  %t278 = load i64, ptr %t277
  %t279 = icmp ult i64 %t276, %t278
  br i1 %t279, label %idx.ok.57, label %idx.bad.58, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.58:
  %t280 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t281 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t275, i64 %t276, ptr %t280, i64 %t281)
  unreachable
idx.ok.57:
  %t282 = load ptr, ptr %t275
  %t283 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t275, i32 0, i32 3
  %t284 = load i64, ptr %t283
  %t285 = mul i64 %t284, %t276
  %t286 = getelementptr inbounds i8, ptr %t282, i64 %t285
  %t287 = load %S.main.Counter, ptr %t286
  %t288 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t289 = insertvalue { i1, %S.main.Counter } %t288, %S.main.Counter %t287, 1
  store { i1, %S.main.Counter } %t289, ptr %a265
  br label %if.end.55
if.else.56:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a265
  br label %if.end.55
if.end.55:
  %t290 = load { i1, %S.main.Counter }, ptr %a265
  store { i1, %S.main.Counter } %t290, ptr %a291
  %t293 = load { i1, %S.main.Counter }, ptr %a291
  %t295 = extractvalue { i1, %S.main.Counter } %t293, 0
  %t294 = xor i1 %t295, true
  br i1 %t294, label %if.then.59, label %if.else.61
if.then.59:
  store { i1, i64 } zeroinitializer, ptr %a292
  br label %if.end.60
if.else.61:
  %t296 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a291, i32 0, i32 1
  %t297 = getelementptr inbounds %S.main.Counter, ptr %t296, i32 0, i32 0
  %t298 = load i64, ptr %t297
  %t299 = insertvalue { i1, i64 } undef, i1 true, 0
  %t300 = insertvalue { i1, i64 } %t299, i64 %t298, 1
  store { i1, i64 } %t300, ptr %a292
  br label %if.end.60
if.end.60:
  %t301 = load { i1, i64 }, ptr %a292
  %t302 = call %str @show.T_i64_N({ i1, i64 } %t301)
  %t303 = load ptr, ptr %a5
  store ptr %t303, ptr %a304
  store i64 1, ptr %a305
  %t306 = load i64, ptr %a305
  %t307 = icmp slt i64 %t306, 0
  br i1 %t307, label %if.then.62, label %if.end.63
if.then.62:
  %t308 = load i64, ptr %a305
  %t309 = load ptr, ptr %a304
  %t310 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t309, i32 0, i32 1
  %t311 = load i64, ptr %t310
  %t312 = add i64 %t308, %t311
  store i64 %t312, ptr %a305
  br label %if.end.63
if.end.63:
  %t315 = load i64, ptr %a305
  %t316 = icmp sge i64 %t315, 0
  store i1 %t316, ptr %a314
  br i1 %t316, label %sc.rhs.64, label %sc.end.65
sc.rhs.64:
  %t317 = load i64, ptr %a305
  %t318 = load ptr, ptr %a304
  %t319 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t318, i32 0, i32 1
  %t320 = load i64, ptr %t319
  %t321 = icmp slt i64 %t317, %t320
  store i1 %t321, ptr %a314
  br label %sc.end.65
sc.end.65:
  %t322 = load i1, ptr %a314
  br i1 %t322, label %if.then.66, label %if.else.68
if.then.66:
  %t323 = load ptr, ptr %a304
  %t324 = load i64, ptr %a305
  %t325 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t323, i32 0, i32 1
  %t326 = load i64, ptr %t325
  %t327 = icmp ult i64 %t324, %t326
  br i1 %t327, label %idx.ok.69, label %idx.bad.70, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.70:
  %t328 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t329 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t323, i64 %t324, ptr %t328, i64 %t329)
  unreachable
idx.ok.69:
  %t330 = load ptr, ptr %t323
  %t331 = getelementptr inbounds { ptr, i64, i64, i64, ptr }, ptr %t323, i32 0, i32 3
  %t332 = load i64, ptr %t331
  %t333 = mul i64 %t332, %t324
  %t334 = getelementptr inbounds i8, ptr %t330, i64 %t333
  %t335 = load %S.main.Counter, ptr %t334
  %t336 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t337 = insertvalue { i1, %S.main.Counter } %t336, %S.main.Counter %t335, 1
  store { i1, %S.main.Counter } %t337, ptr %a313
  br label %if.end.67
if.else.68:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a313
  br label %if.end.67
if.end.67:
  %t338 = load { i1, %S.main.Counter }, ptr %a313
  store { i1, %S.main.Counter } %t338, ptr %a339
  %t341 = load { i1, %S.main.Counter }, ptr %a339
  %t343 = extractvalue { i1, %S.main.Counter } %t341, 0
  %t342 = xor i1 %t343, true
  br i1 %t342, label %if.then.71, label %if.else.73
if.then.71:
  store { i1, i64 } zeroinitializer, ptr %a340
  br label %if.end.72
if.else.73:
  %t344 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a339, i32 0, i32 1
  %t345 = getelementptr inbounds %S.main.Counter, ptr %t344, i32 0, i32 0
  %t346 = load i64, ptr %t345
  %t347 = insertvalue { i1, i64 } undef, i1 true, 0
  %t348 = insertvalue { i1, i64 } %t347, i64 %t346, 1
  store { i1, i64 } %t348, ptr %a340
  br label %if.end.72
if.end.72:
  %t349 = load { i1, i64 }, ptr %a340
  %t350 = call %str @show.T_i64_N({ i1, i64 } %t349)
  %t353 = getelementptr [9 x %str], ptr %a352, i64 0, i64 0
  store %str %t204, ptr %t353
  %t354 = getelementptr [9 x %str], ptr %a352, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t354
  %t355 = getelementptr [9 x %str], ptr %a352, i64 0, i64 2
  store %str %t249, ptr %t355
  %t356 = getelementptr [9 x %str], ptr %a352, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t356
  %t357 = getelementptr [9 x %str], ptr %a352, i64 0, i64 4
  store %str %t254, ptr %t357
  %t358 = getelementptr [9 x %str], ptr %a352, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t358
  %t359 = getelementptr [9 x %str], ptr %a352, i64 0, i64 6
  store %str %t302, ptr %t359
  %t360 = getelementptr [9 x %str], ptr %a352, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t360
  %t361 = getelementptr [9 x %str], ptr %a352, i64 0, i64 8
  store %str %t350, ptr %t361
  call void @veles_string_concat_n(ptr %a351, ptr %a352, i64 9)
  %t362 = load %str, ptr %a351
  call void @v_std.io.println(%str %t362)
  %t363 = load ptr, ptr %a98
  %t365 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t366 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t364 = call i64 @veles_hash_bytes(ptr %t365, i64 %t366)
  store %str { ptr @.str.4, i64 1 }, ptr %a367
  %t368 = call i64 @veles_map_find(ptr %t363, i64 %t364, ptr %a367, ptr @eqp.string)
  %t369 = icmp sge i64 %t368, 0
  store { i1, i64 } zeroinitializer, ptr %a370
  br i1 %t369, label %map.hit.74, label %map.end.75
map.hit.74:
  %t371 = call ptr @veles_map_val_at(ptr %t363, i64 %t368)
  %t372 = load i64, ptr %t371
  %t373 = insertvalue { i1, i64 } undef, i1 true, 0
  %t374 = insertvalue { i1, i64 } %t373, i64 %t372, 1
  store { i1, i64 } %t374, ptr %a370
  br label %map.end.75
map.end.75:
  %t375 = load { i1, i64 }, ptr %a370
  %t378 = extractvalue { i1, i64 } %t375, 0
  %t377 = xor i1 %t378, true
  br i1 %t377, label %elvis.default.76, label %elvis.some.77
elvis.some.77:
  %t379 = extractvalue { i1, i64 } %t375, 1
  store i64 %t379, ptr %a376
  br label %elvis.end.78
elvis.default.76:
  store i64 0, ptr %a376
  br label %elvis.end.78
elvis.end.78:
  %t380 = load i64, ptr %a376
  %t382 = call i64 @veles_i64_format(ptr %a381, i64 %t380)
  %t383 = insertvalue %str undef, ptr %a381, 0
  %t384 = insertvalue %str %t383, i64 %t382, 1
  %t385 = load ptr, ptr %a98
  %t387 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t388 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t386 = call i64 @veles_hash_bytes(ptr %t387, i64 %t388)
  store %str { ptr @.str.5, i64 1 }, ptr %a389
  %t390 = call i64 @veles_map_find(ptr %t385, i64 %t386, ptr %a389, ptr @eqp.string)
  %t391 = icmp sge i64 %t390, 0
  store { i1, i64 } zeroinitializer, ptr %a392
  br i1 %t391, label %map.hit.79, label %map.end.80
map.hit.79:
  %t393 = call ptr @veles_map_val_at(ptr %t385, i64 %t390)
  %t394 = load i64, ptr %t393
  %t395 = insertvalue { i1, i64 } undef, i1 true, 0
  %t396 = insertvalue { i1, i64 } %t395, i64 %t394, 1
  store { i1, i64 } %t396, ptr %a392
  br label %map.end.80
map.end.80:
  %t397 = load { i1, i64 }, ptr %a392
  %t400 = extractvalue { i1, i64 } %t397, 0
  %t399 = xor i1 %t400, true
  br i1 %t399, label %elvis.default.81, label %elvis.some.82
elvis.some.82:
  %t401 = extractvalue { i1, i64 } %t397, 1
  store i64 %t401, ptr %a398
  br label %elvis.end.83
elvis.default.81:
  store i64 0, ptr %a398
  br label %elvis.end.83
elvis.end.83:
  %t402 = load i64, ptr %a398
  %t404 = call i64 @veles_i64_format(ptr %a403, i64 %t402)
  %t405 = insertvalue %str undef, ptr %a403, 0
  %t406 = insertvalue %str %t405, i64 %t404, 1
  %t407 = load ptr, ptr %a98
  %t408 = call i64 @veles_map_len(ptr %t407)
  %t410 = call i64 @veles_i64_format(ptr %a409, i64 %t408)
  %t411 = insertvalue %str undef, ptr %a409, 0
  %t412 = insertvalue %str %t411, i64 %t410, 1
  %t413 = load ptr, ptr %a147
  %t414 = call i64 @veles_map_len(ptr %t413)
  %t416 = call i64 @veles_i64_format(ptr %a415, i64 %t414)
  %t417 = insertvalue %str undef, ptr %a415, 0
  %t418 = insertvalue %str %t417, i64 %t416, 1
  %t419 = load ptr, ptr %a147
  store i64 4, ptr %a421
  %t422 = call i64 @veles_map_find(ptr %t419, i64 4, ptr %a421, ptr @eqp.i64)
  %t423 = icmp sge i64 %t422, 0
  call void @veles_bool_to_string(ptr %a424, i1 zeroext %t423)
  %t425 = load %str, ptr %a424
  %t428 = getelementptr [9 x %str], ptr %a427, i64 0, i64 0
  store %str %t384, ptr %t428
  %t429 = getelementptr [9 x %str], ptr %a427, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t429
  %t430 = getelementptr [9 x %str], ptr %a427, i64 0, i64 2
  store %str %t406, ptr %t430
  %t431 = getelementptr [9 x %str], ptr %a427, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t431
  %t432 = getelementptr [9 x %str], ptr %a427, i64 0, i64 4
  store %str %t412, ptr %t432
  %t433 = getelementptr [9 x %str], ptr %a427, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t433
  %t434 = getelementptr [9 x %str], ptr %a427, i64 0, i64 6
  store %str %t418, ptr %t434
  %t435 = getelementptr [9 x %str], ptr %a427, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t435
  %t436 = getelementptr [9 x %str], ptr %a427, i64 0, i64 8
  store %str %t425, ptr %t436
  call void @veles_string_concat_n(ptr %a426, ptr %a427, i64 9)
  %t437 = load %str, ptr %a426
  call void @v_std.io.println(%str %t437)
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
