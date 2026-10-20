%S.main.Counter = type { i64 }
@adesc.main.Counter = internal constant { i64, i64, i64, [0 x i64] } { i64 8, i64 1, i64 0, [0 x i64] [] }
define void @v_main.main() {
entry:
  %a2 = alloca i64
  %a3 = alloca ptr
  %a4 = alloca { ptr, i64, i64, i64, ptr, i64, i64 }
  %a7 = alloca ptr
  %a24 = alloca %S.main.Counter
  %a41 = alloca %S.main.Counter
  %a47 = alloca ptr
  %a49 = alloca i64
  %a71 = alloca ptr
  %a86 = alloca ptr
  %a89 = alloca i64
  %a111 = alloca { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }
  %a116 = alloca ptr
  %a121 = alloca %str
  %a122 = alloca i64
  %a127 = alloca %str
  %a128 = alloca i64
  %a133 = alloca %str
  %a136 = alloca ptr
  %a139 = alloca ptr
  %a146 = alloca ptr
  %a162 = alloca %str
  %a164 = alloca { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }
  %a169 = alloca ptr
  %a172 = alloca i64
  %a178 = alloca i64
  %a183 = alloca ptr
  %a184 = alloca i64
  %a192 = alloca { i1, i64 }
  %a193 = alloca i1
  %a218 = alloca i64
  %a223 = alloca [21 x i8]
  %a228 = alloca ptr
  %a229 = alloca i64
  %a237 = alloca { i1, i64 }
  %a238 = alloca i1
  %a263 = alloca i64
  %a268 = alloca [21 x i8]
  %a273 = alloca ptr
  %a278 = alloca ptr
  %a279 = alloca i64
  %a287 = alloca { i1, %S.main.Counter }
  %a288 = alloca i1
  %a313 = alloca { i1, %S.main.Counter }
  %a314 = alloca { i1, i64 }
  %a326 = alloca ptr
  %a327 = alloca i64
  %a335 = alloca { i1, %S.main.Counter }
  %a336 = alloca i1
  %a361 = alloca { i1, %S.main.Counter }
  %a362 = alloca { i1, i64 }
  %a373 = alloca %str
  %a374 = alloca [9 x %str]
  %a391 = alloca %str
  %a394 = alloca { i1, i64 }
  %a400 = alloca i64
  %a405 = alloca [21 x i8]
  %a413 = alloca %str
  %a416 = alloca { i1, i64 }
  %a422 = alloca i64
  %a427 = alloca [21 x i8]
  %a433 = alloca [21 x i8]
  %a439 = alloca [21 x i8]
  %a445 = alloca i64
  %a448 = alloca %str
  %a450 = alloca %str
  %a451 = alloca [9 x %str]
  %t1 = call ptr @veles_list_new(ptr @adesc.i64, i64 3)
  store i64 3, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 1, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store i64 2, ptr %a2
  call void @veles_list_push(ptr %t1, ptr %a2)
  store ptr %t1, ptr %a3
  store { ptr, i64, i64, i64, ptr, i64, i64 } zeroinitializer, ptr %a4
  %t5 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a4, i32 0, i32 3
  store i64 8, ptr %t5
  %t6 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %a4, i32 0, i32 4
  store ptr @adesc.main.Counter, ptr %t6
  store ptr %a4, ptr %a7
  %t8 = load ptr, ptr %a7
  %t9 = insertvalue %S.main.Counter undef, i64 1, 0
  %t10 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 1
  %t11 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 2
  %t12 = load i64, ptr %t10
  %t13 = load i64, ptr %t11
  %t14 = icmp slt i64 %t12, %t13
  br i1 %t14, label %push.fast.1, label %push.grow.2, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.1:
  %t15 = load ptr, ptr %t8
  %t16 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 3
  %t17 = load i64, ptr %t16
  %t18 = mul i64 %t17, %t12
  %t19 = getelementptr inbounds i8, ptr %t15, i64 %t18
  store %S.main.Counter %t9, ptr %t19
  %t20 = add i64 %t12, 1
  store i64 %t20, ptr %t10
  %t21 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t8, i32 0, i32 5
  %t22 = load i64, ptr %t21
  %t23 = add i64 %t22, 1
  store i64 %t23, ptr %t21
  br label %push.done.3
push.grow.2:
  store %S.main.Counter %t9, ptr %a24
  call void @veles_list_push(ptr %t8, ptr %a24)
  br label %push.done.3
push.done.3:
  %t25 = load ptr, ptr %a7
  %t26 = insertvalue %S.main.Counter undef, i64 5, 0
  %t27 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 1
  %t28 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 2
  %t29 = load i64, ptr %t27
  %t30 = load i64, ptr %t28
  %t31 = icmp slt i64 %t29, %t30
  br i1 %t31, label %push.fast.4, label %push.grow.5, !prof !{!"branch_weights", i32 2000, i32 1}
push.fast.4:
  %t32 = load ptr, ptr %t25
  %t33 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 3
  %t34 = load i64, ptr %t33
  %t35 = mul i64 %t34, %t29
  %t36 = getelementptr inbounds i8, ptr %t32, i64 %t35
  store %S.main.Counter %t26, ptr %t36
  %t37 = add i64 %t29, 1
  store i64 %t37, ptr %t27
  %t38 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t25, i32 0, i32 5
  %t39 = load i64, ptr %t38
  %t40 = add i64 %t39, 1
  store i64 %t40, ptr %t38
  br label %push.done.6
push.grow.5:
  store %S.main.Counter %t26, ptr %a41
  call void @veles_list_push(ptr %t25, ptr %a41)
  br label %push.done.6
push.done.6:
  %t42 = load ptr, ptr %a7
  %t43 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t42, i32 0, i32 1
  %t44 = load i64, ptr %t43
  %t45 = icmp eq i64 %t44, 2
  br i1 %t45, label %if.then.7, label %if.end.8
if.then.7:
  %t46 = load ptr, ptr %a7
  store ptr %t46, ptr %a47
  %t48 = load ptr, ptr %a47
  store i64 0, ptr %a49
  %t50 = load i64, ptr %a49
  %t51 = icmp slt i64 %t50, 0
  br i1 %t51, label %if.then.9, label %if.end.10
if.then.9:
  %t52 = load i64, ptr %a49
  %t53 = load ptr, ptr %a47
  %t54 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t53, i32 0, i32 1
  %t55 = load i64, ptr %t54
  %t56 = add i64 %t52, %t55
  store i64 %t56, ptr %a49
  br label %if.end.10
if.end.10:
  %t57 = load i64, ptr %a49
  %t58 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t48, i32 0, i32 6
  %t59 = load i64, ptr %t58
  %t60 = icmp eq i64 %t59, 1
  br i1 %t60, label %flags.done.12, label %flags.slow.11, !prof !{!"branch_weights", i32 2000, i32 1}
flags.slow.11:
  call void @veles_list_expose(ptr %t48)
  br label %flags.done.12
flags.done.12:
  %t61 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t48, i32 0, i32 1
  %t62 = load i64, ptr %t61
  %t63 = icmp ult i64 %t57, %t62
  br i1 %t63, label %idx.ok.13, label %idx.bad.14, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.14:
  %t64 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t65 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t48, i64 %t57, ptr %t64, i64 %t65)
  unreachable
idx.ok.13:
  %t66 = load ptr, ptr %t48
  %t67 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t48, i32 0, i32 3
  %t68 = load i64, ptr %t67
  %t69 = mul i64 %t68, %t57
  %t70 = getelementptr inbounds i8, ptr %t66, i64 %t69
  store ptr %t70, ptr %a71
  %t72 = load ptr, ptr %a71
  %t73 = getelementptr inbounds %S.main.Counter, ptr %t72, i32 0, i32 0
  %t74 = load i64, ptr %t73
  %t76 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t74, i64 10)
  %t77 = extractvalue { i64, i1 } %t76, 0
  %t78 = extractvalue { i64, i1 } %t76, 1
  br i1 %t78, label %overflow.15, label %arith.ok.16
overflow.15:
  %t79 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t80 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t81 = extractvalue %str { ptr @.str.1, i64 13 }, 0
  %t82 = extractvalue %str { ptr @.str.1, i64 13 }, 1
  call void @veles_panic_at(ptr %t79, i64 %t80, ptr %t81, i64 %t82)
  unreachable
arith.ok.16:
  %t83 = load ptr, ptr %a71
  %t84 = getelementptr inbounds %S.main.Counter, ptr %t83, i32 0, i32 0
  store i64 %t77, ptr %t84
  br label %if.end.8
if.end.8:
  %t85 = load ptr, ptr %a7
  store ptr %t85, ptr %a86
  %t87 = insertvalue %S.main.Counter undef, i64 7, 0
  %t88 = load ptr, ptr %a86
  store i64 1, ptr %a89
  %t90 = load i64, ptr %a89
  %t91 = icmp slt i64 %t90, 0
  br i1 %t91, label %if.then.17, label %if.end.18
if.then.17:
  %t92 = load i64, ptr %a89
  %t93 = load ptr, ptr %a86
  %t94 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t93, i32 0, i32 1
  %t95 = load i64, ptr %t94
  %t96 = add i64 %t92, %t95
  store i64 %t96, ptr %a89
  br label %if.end.18
if.end.18:
  %t97 = load i64, ptr %a89
  %t98 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t88, i32 0, i32 6
  %t99 = load i64, ptr %t98
  %t100 = icmp ult i64 %t99, 2
  br i1 %t100, label %flags.done.20, label %flags.slow.19, !prof !{!"branch_weights", i32 2000, i32 1}
flags.slow.19:
  call void @veles_list_write_check(ptr %t88)
  br label %flags.done.20
flags.done.20:
  %t101 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t88, i32 0, i32 1
  %t102 = load i64, ptr %t101
  %t103 = icmp ult i64 %t97, %t102
  br i1 %t103, label %idx.ok.21, label %idx.bad.22, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.22:
  %t104 = extractvalue %str { ptr @.str.3, i64 12 }, 0
  %t105 = extractvalue %str { ptr @.str.3, i64 12 }, 1
  call void @veles_list_index_panic(ptr %t88, i64 %t97, ptr %t104, i64 %t105)
  unreachable
idx.ok.21:
  %t106 = load ptr, ptr %t88
  %t107 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t88, i32 0, i32 3
  %t108 = load i64, ptr %t107
  %t109 = mul i64 %t108, %t97
  %t110 = getelementptr inbounds i8, ptr %t106, i64 %t109
  store %S.main.Counter %t87, ptr %t110
  store { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 } zeroinitializer, ptr %a111
  %t112 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a111, i32 0, i32 3
  store ptr @adesc.string, ptr %t112
  %t113 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a111, i32 0, i32 4
  store ptr @adesc.i64, ptr %t113
  %t114 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a111, i32 0, i32 10
  store i64 16, ptr %t114
  %t115 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a111, i32 0, i32 11
  store i64 8, ptr %t115
  store ptr %a111, ptr %a116
  %t117 = load ptr, ptr %a116
  %t119 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t120 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t118 = call i64 @veles_hash_bytes(ptr %t119, i64 %t120)
  store %str { ptr @.str.4, i64 1 }, ptr %a121
  store i64 1, ptr %a122
  call i64 @veles_map_insert(ptr %t117, i64 %t118, ptr %a121, ptr %a122, ptr @eqp.string)
  %t123 = load ptr, ptr %a116
  %t125 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t126 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t124 = call i64 @veles_hash_bytes(ptr %t125, i64 %t126)
  store %str { ptr @.str.5, i64 1 }, ptr %a127
  store i64 2, ptr %a128
  call i64 @veles_map_insert(ptr %t123, i64 %t124, ptr %a127, ptr %a128, ptr @eqp.string)
  %t129 = load ptr, ptr %a116
  call void @veles_map_expose(ptr %t129)
  %t131 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t132 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t130 = call i64 @veles_hash_bytes(ptr %t131, i64 %t132)
  store %str { ptr @.str.4, i64 1 }, ptr %a133
  %t134 = call i64 @veles_map_find(ptr %t129, i64 %t130, ptr %a133, ptr @eqp.string)
  %t135 = icmp sge i64 %t134, 0
  store ptr null, ptr %a136
  br i1 %t135, label %map.hit.23, label %map.end.24
map.hit.23:
  %t137 = call ptr @veles_map_val_at(ptr %t129, i64 %t134)
  store ptr %t137, ptr %a136
  br label %map.end.24
map.end.24:
  %t138 = load ptr, ptr %a136
  %t140 = icmp eq ptr %t138, null
  br i1 %t140, label %elvis.default.25, label %elvis.some.26
elvis.some.26:
  store ptr %t138, ptr %a139
  br label %elvis.end.27
elvis.default.25:
  %t141 = extractvalue %str { ptr @.str.6, i64 15 }, 0
  %t142 = extractvalue %str { ptr @.str.6, i64 15 }, 1
  %t143 = extractvalue %str { ptr @.str.7, i64 13 }, 0
  %t144 = extractvalue %str { ptr @.str.7, i64 13 }, 1
  call void @veles_panic_at(ptr %t141, i64 %t142, ptr %t143, i64 %t144)
  unreachable
elvis.end.27:
  %t145 = load ptr, ptr %a139
  store ptr %t145, ptr %a146
  %t147 = load ptr, ptr %a146
  %t148 = load i64, ptr %t147
  %t150 = call { i64, i1 } @llvm.sadd.with.overflow.i64(i64 %t148, i64 40)
  %t151 = extractvalue { i64, i1 } %t150, 0
  %t152 = extractvalue { i64, i1 } %t150, 1
  br i1 %t152, label %overflow.28, label %arith.ok.29
overflow.28:
  %t153 = extractvalue %str { ptr @.str.2, i64 16 }, 0
  %t154 = extractvalue %str { ptr @.str.2, i64 16 }, 1
  %t155 = extractvalue %str { ptr @.str.8, i64 12 }, 0
  %t156 = extractvalue %str { ptr @.str.8, i64 12 }, 1
  call void @veles_panic_at(ptr %t153, i64 %t154, ptr %t155, i64 %t156)
  unreachable
arith.ok.29:
  %t157 = load ptr, ptr %a146
  store i64 %t151, ptr %t157
  %t158 = load ptr, ptr %a116
  %t160 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t161 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t159 = call i64 @veles_hash_bytes(ptr %t160, i64 %t161)
  store %str { ptr @.str.5, i64 1 }, ptr %a162
  %t163 = call i1 @veles_map_remove(ptr %t158, i64 %t159, ptr %a162, ptr @eqp.string)
  store { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 } zeroinitializer, ptr %a164
  %t165 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a164, i32 0, i32 3
  store ptr @adesc.i64, ptr %t165
  %t166 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a164, i32 0, i32 4
  store ptr null, ptr %t166
  %t167 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a164, i32 0, i32 10
  store i64 8, ptr %t167
  %t168 = getelementptr inbounds { ptr, ptr, ptr, ptr, ptr, ptr, i64, i64, i64, i64, i64, i64, i64, i64 }, ptr %a164, i32 0, i32 11
  store i64 0, ptr %t168
  store ptr %a164, ptr %a169
  %t170 = load ptr, ptr %a169
  store i64 4, ptr %a172
  %t173 = call i64 @veles_map_len(ptr %t170)
  call i64 @veles_map_insert(ptr %t170, i64 4, ptr %a172, ptr null, ptr @eqp.i64)
  %t174 = call i64 @veles_map_len(ptr %t170)
  %t175 = icmp ne i64 %t173, %t174
  %t176 = load ptr, ptr %a169
  store i64 4, ptr %a178
  %t179 = call i64 @veles_map_len(ptr %t176)
  call i64 @veles_map_insert(ptr %t176, i64 4, ptr %a178, ptr null, ptr @eqp.i64)
  %t180 = call i64 @veles_map_len(ptr %t176)
  %t181 = icmp ne i64 %t179, %t180
  %t182 = load ptr, ptr %a3
  store ptr %t182, ptr %a183
  store i64 1, ptr %a184
  %t185 = load i64, ptr %a184
  %t186 = icmp slt i64 %t185, 0
  br i1 %t186, label %if.then.30, label %if.end.31
if.then.30:
  %t187 = load i64, ptr %a184
  %t188 = load ptr, ptr %a183
  %t189 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t188, i32 0, i32 1
  %t190 = load i64, ptr %t189
  %t191 = add i64 %t187, %t190
  store i64 %t191, ptr %a184
  br label %if.end.31
if.end.31:
  %t194 = load i64, ptr %a184
  %t195 = icmp sge i64 %t194, 0
  store i1 %t195, ptr %a193
  br i1 %t195, label %sc.rhs.32, label %sc.end.33
sc.rhs.32:
  %t196 = load i64, ptr %a184
  %t197 = load ptr, ptr %a183
  %t198 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t197, i32 0, i32 1
  %t199 = load i64, ptr %t198
  %t200 = icmp slt i64 %t196, %t199
  store i1 %t200, ptr %a193
  br label %sc.end.33
sc.end.33:
  %t201 = load i1, ptr %a193
  br i1 %t201, label %if.then.34, label %if.else.36
if.then.34:
  %t202 = load ptr, ptr %a183
  %t203 = load i64, ptr %a184
  %t204 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t202, i32 0, i32 1
  %t205 = load i64, ptr %t204
  %t206 = icmp ult i64 %t203, %t205
  br i1 %t206, label %idx.ok.37, label %idx.bad.38, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.38:
  %t207 = extractvalue %str { ptr @.str.9, i64 13 }, 0
  %t208 = extractvalue %str { ptr @.str.9, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t202, i64 %t203, ptr %t207, i64 %t208)
  unreachable
idx.ok.37:
  %t209 = load ptr, ptr %t202
  %t210 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t202, i32 0, i32 3
  %t211 = load i64, ptr %t210
  %t212 = mul i64 %t211, %t203
  %t213 = getelementptr inbounds i8, ptr %t209, i64 %t212
  %t214 = load i64, ptr %t213
  %t215 = insertvalue { i1, i64 } undef, i1 true, 0
  %t216 = insertvalue { i1, i64 } %t215, i64 %t214, 1
  store { i1, i64 } %t216, ptr %a192
  br label %if.end.35
if.else.36:
  store { i1, i64 } zeroinitializer, ptr %a192
  br label %if.end.35
if.end.35:
  %t217 = load { i1, i64 }, ptr %a192
  %t220 = extractvalue { i1, i64 } %t217, 0
  %t219 = xor i1 %t220, true
  br i1 %t219, label %elvis.default.39, label %elvis.some.40
elvis.some.40:
  %t221 = extractvalue { i1, i64 } %t217, 1
  store i64 %t221, ptr %a218
  br label %elvis.end.41
elvis.default.39:
  store i64 -1, ptr %a218
  br label %elvis.end.41
elvis.end.41:
  %t222 = load i64, ptr %a218
  %t224 = call i64 @veles_i64_format(ptr %a223, i64 %t222)
  %t225 = insertvalue %str undef, ptr %a223, 0
  %t226 = insertvalue %str %t225, i64 %t224, 1
  %t227 = load ptr, ptr %a3
  store ptr %t227, ptr %a228
  store i64 9, ptr %a229
  %t230 = load i64, ptr %a229
  %t231 = icmp slt i64 %t230, 0
  br i1 %t231, label %if.then.42, label %if.end.43
if.then.42:
  %t232 = load i64, ptr %a229
  %t233 = load ptr, ptr %a228
  %t234 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t233, i32 0, i32 1
  %t235 = load i64, ptr %t234
  %t236 = add i64 %t232, %t235
  store i64 %t236, ptr %a229
  br label %if.end.43
if.end.43:
  %t239 = load i64, ptr %a229
  %t240 = icmp sge i64 %t239, 0
  store i1 %t240, ptr %a238
  br i1 %t240, label %sc.rhs.44, label %sc.end.45
sc.rhs.44:
  %t241 = load i64, ptr %a229
  %t242 = load ptr, ptr %a228
  %t243 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t242, i32 0, i32 1
  %t244 = load i64, ptr %t243
  %t245 = icmp slt i64 %t241, %t244
  store i1 %t245, ptr %a238
  br label %sc.end.45
sc.end.45:
  %t246 = load i1, ptr %a238
  br i1 %t246, label %if.then.46, label %if.else.48
if.then.46:
  %t247 = load ptr, ptr %a228
  %t248 = load i64, ptr %a229
  %t249 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t247, i32 0, i32 1
  %t250 = load i64, ptr %t249
  %t251 = icmp ult i64 %t248, %t250
  br i1 %t251, label %idx.ok.49, label %idx.bad.50, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.50:
  %t252 = extractvalue %str { ptr @.str.10, i64 13 }, 0
  %t253 = extractvalue %str { ptr @.str.10, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t247, i64 %t248, ptr %t252, i64 %t253)
  unreachable
idx.ok.49:
  %t254 = load ptr, ptr %t247
  %t255 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t247, i32 0, i32 3
  %t256 = load i64, ptr %t255
  %t257 = mul i64 %t256, %t248
  %t258 = getelementptr inbounds i8, ptr %t254, i64 %t257
  %t259 = load i64, ptr %t258
  %t260 = insertvalue { i1, i64 } undef, i1 true, 0
  %t261 = insertvalue { i1, i64 } %t260, i64 %t259, 1
  store { i1, i64 } %t261, ptr %a237
  br label %if.end.47
if.else.48:
  store { i1, i64 } zeroinitializer, ptr %a237
  br label %if.end.47
if.end.47:
  %t262 = load { i1, i64 }, ptr %a237
  %t265 = extractvalue { i1, i64 } %t262, 0
  %t264 = xor i1 %t265, true
  br i1 %t264, label %elvis.default.51, label %elvis.some.52
elvis.some.52:
  %t266 = extractvalue { i1, i64 } %t262, 1
  store i64 %t266, ptr %a263
  br label %elvis.end.53
elvis.default.51:
  store i64 -1, ptr %a263
  br label %elvis.end.53
elvis.end.53:
  %t267 = load i64, ptr %a263
  %t269 = call i64 @veles_i64_format(ptr %a268, i64 %t267)
  %t270 = insertvalue %str undef, ptr %a268, 0
  %t271 = insertvalue %str %t270, i64 %t269, 1
  %t272 = load ptr, ptr %a3
  store ptr %t272, ptr %a273
  %t274 = load ptr, ptr %a273
  %t275 = call ptr @veles_list_copy(ptr %t274)
  call void @veles_list_sort_native(ptr %t275, i32 4)
  %t276 = call %str @show.List_i64_(ptr %t275)
  %t277 = load ptr, ptr %a7
  store ptr %t277, ptr %a278
  store i64 0, ptr %a279
  %t280 = load i64, ptr %a279
  %t281 = icmp slt i64 %t280, 0
  br i1 %t281, label %if.then.54, label %if.end.55
if.then.54:
  %t282 = load i64, ptr %a279
  %t283 = load ptr, ptr %a278
  %t284 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t283, i32 0, i32 1
  %t285 = load i64, ptr %t284
  %t286 = add i64 %t282, %t285
  store i64 %t286, ptr %a279
  br label %if.end.55
if.end.55:
  %t289 = load i64, ptr %a279
  %t290 = icmp sge i64 %t289, 0
  store i1 %t290, ptr %a288
  br i1 %t290, label %sc.rhs.56, label %sc.end.57
sc.rhs.56:
  %t291 = load i64, ptr %a279
  %t292 = load ptr, ptr %a278
  %t293 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t292, i32 0, i32 1
  %t294 = load i64, ptr %t293
  %t295 = icmp slt i64 %t291, %t294
  store i1 %t295, ptr %a288
  br label %sc.end.57
sc.end.57:
  %t296 = load i1, ptr %a288
  br i1 %t296, label %if.then.58, label %if.else.60
if.then.58:
  %t297 = load ptr, ptr %a278
  %t298 = load i64, ptr %a279
  %t299 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t297, i32 0, i32 1
  %t300 = load i64, ptr %t299
  %t301 = icmp ult i64 %t298, %t300
  br i1 %t301, label %idx.ok.61, label %idx.bad.62, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.62:
  %t302 = extractvalue %str { ptr @.str.11, i64 13 }, 0
  %t303 = extractvalue %str { ptr @.str.11, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t297, i64 %t298, ptr %t302, i64 %t303)
  unreachable
idx.ok.61:
  %t304 = load ptr, ptr %t297
  %t305 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t297, i32 0, i32 3
  %t306 = load i64, ptr %t305
  %t307 = mul i64 %t306, %t298
  %t308 = getelementptr inbounds i8, ptr %t304, i64 %t307
  %t309 = load %S.main.Counter, ptr %t308
  %t310 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t311 = insertvalue { i1, %S.main.Counter } %t310, %S.main.Counter %t309, 1
  store { i1, %S.main.Counter } %t311, ptr %a287
  br label %if.end.59
if.else.60:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a287
  br label %if.end.59
if.end.59:
  %t312 = load { i1, %S.main.Counter }, ptr %a287
  store { i1, %S.main.Counter } %t312, ptr %a313
  %t315 = load { i1, %S.main.Counter }, ptr %a313
  %t317 = extractvalue { i1, %S.main.Counter } %t315, 0
  %t316 = xor i1 %t317, true
  br i1 %t316, label %if.then.63, label %if.else.65
if.then.63:
  store { i1, i64 } zeroinitializer, ptr %a314
  br label %if.end.64
if.else.65:
  %t318 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a313, i32 0, i32 1
  %t319 = getelementptr inbounds %S.main.Counter, ptr %t318, i32 0, i32 0
  %t320 = load i64, ptr %t319
  %t321 = insertvalue { i1, i64 } undef, i1 true, 0
  %t322 = insertvalue { i1, i64 } %t321, i64 %t320, 1
  store { i1, i64 } %t322, ptr %a314
  br label %if.end.64
if.end.64:
  %t323 = load { i1, i64 }, ptr %a314
  %t324 = call %str @show.T_i64_N({ i1, i64 } %t323)
  %t325 = load ptr, ptr %a7
  store ptr %t325, ptr %a326
  store i64 1, ptr %a327
  %t328 = load i64, ptr %a327
  %t329 = icmp slt i64 %t328, 0
  br i1 %t329, label %if.then.66, label %if.end.67
if.then.66:
  %t330 = load i64, ptr %a327
  %t331 = load ptr, ptr %a326
  %t332 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t331, i32 0, i32 1
  %t333 = load i64, ptr %t332
  %t334 = add i64 %t330, %t333
  store i64 %t334, ptr %a327
  br label %if.end.67
if.end.67:
  %t337 = load i64, ptr %a327
  %t338 = icmp sge i64 %t337, 0
  store i1 %t338, ptr %a336
  br i1 %t338, label %sc.rhs.68, label %sc.end.69
sc.rhs.68:
  %t339 = load i64, ptr %a327
  %t340 = load ptr, ptr %a326
  %t341 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t340, i32 0, i32 1
  %t342 = load i64, ptr %t341
  %t343 = icmp slt i64 %t339, %t342
  store i1 %t343, ptr %a336
  br label %sc.end.69
sc.end.69:
  %t344 = load i1, ptr %a336
  br i1 %t344, label %if.then.70, label %if.else.72
if.then.70:
  %t345 = load ptr, ptr %a326
  %t346 = load i64, ptr %a327
  %t347 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t345, i32 0, i32 1
  %t348 = load i64, ptr %t347
  %t349 = icmp ult i64 %t346, %t348
  br i1 %t349, label %idx.ok.73, label %idx.bad.74, !prof !{!"branch_weights", i32 2000, i32 1}
idx.bad.74:
  %t350 = extractvalue %str { ptr @.str.12, i64 13 }, 0
  %t351 = extractvalue %str { ptr @.str.12, i64 13 }, 1
  call void @veles_list_index_panic(ptr %t345, i64 %t346, ptr %t350, i64 %t351)
  unreachable
idx.ok.73:
  %t352 = load ptr, ptr %t345
  %t353 = getelementptr inbounds { ptr, i64, i64, i64, ptr, i64, i64 }, ptr %t345, i32 0, i32 3
  %t354 = load i64, ptr %t353
  %t355 = mul i64 %t354, %t346
  %t356 = getelementptr inbounds i8, ptr %t352, i64 %t355
  %t357 = load %S.main.Counter, ptr %t356
  %t358 = insertvalue { i1, %S.main.Counter } undef, i1 true, 0
  %t359 = insertvalue { i1, %S.main.Counter } %t358, %S.main.Counter %t357, 1
  store { i1, %S.main.Counter } %t359, ptr %a335
  br label %if.end.71
if.else.72:
  store { i1, %S.main.Counter } zeroinitializer, ptr %a335
  br label %if.end.71
if.end.71:
  %t360 = load { i1, %S.main.Counter }, ptr %a335
  store { i1, %S.main.Counter } %t360, ptr %a361
  %t363 = load { i1, %S.main.Counter }, ptr %a361
  %t365 = extractvalue { i1, %S.main.Counter } %t363, 0
  %t364 = xor i1 %t365, true
  br i1 %t364, label %if.then.75, label %if.else.77
if.then.75:
  store { i1, i64 } zeroinitializer, ptr %a362
  br label %if.end.76
if.else.77:
  %t366 = getelementptr inbounds { i1, %S.main.Counter }, ptr %a361, i32 0, i32 1
  %t367 = getelementptr inbounds %S.main.Counter, ptr %t366, i32 0, i32 0
  %t368 = load i64, ptr %t367
  %t369 = insertvalue { i1, i64 } undef, i1 true, 0
  %t370 = insertvalue { i1, i64 } %t369, i64 %t368, 1
  store { i1, i64 } %t370, ptr %a362
  br label %if.end.76
if.end.76:
  %t371 = load { i1, i64 }, ptr %a362
  %t372 = call %str @show.T_i64_N({ i1, i64 } %t371)
  %t375 = getelementptr [9 x %str], ptr %a374, i64 0, i64 0
  store %str %t226, ptr %t375
  %t376 = getelementptr [9 x %str], ptr %a374, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t376
  %t377 = getelementptr [9 x %str], ptr %a374, i64 0, i64 2
  store %str %t271, ptr %t377
  %t378 = getelementptr [9 x %str], ptr %a374, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t378
  %t379 = getelementptr [9 x %str], ptr %a374, i64 0, i64 4
  store %str %t276, ptr %t379
  %t380 = getelementptr [9 x %str], ptr %a374, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t380
  %t381 = getelementptr [9 x %str], ptr %a374, i64 0, i64 6
  store %str %t324, ptr %t381
  %t382 = getelementptr [9 x %str], ptr %a374, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t382
  %t383 = getelementptr [9 x %str], ptr %a374, i64 0, i64 8
  store %str %t372, ptr %t383
  call void @veles_string_concat_n(ptr %a373, ptr %a374, i64 9)
  %t384 = load %str, ptr %a373
  %t385 = extractvalue %str { ptr @.str.14, i64 20 }, 0
  %t386 = extractvalue %str { ptr @.str.14, i64 20 }, 1
  call void @veles_call_push(ptr %t385)
  call void @v_std.io.println(%str %t384)
  call void @veles_call_pop()
  %t387 = load ptr, ptr %a116
  %t389 = extractvalue %str { ptr @.str.4, i64 1 }, 0
  %t390 = extractvalue %str { ptr @.str.4, i64 1 }, 1
  %t388 = call i64 @veles_hash_bytes(ptr %t389, i64 %t390)
  store %str { ptr @.str.4, i64 1 }, ptr %a391
  %t392 = call i64 @veles_map_find(ptr %t387, i64 %t388, ptr %a391, ptr @eqp.string)
  %t393 = icmp sge i64 %t392, 0
  store { i1, i64 } zeroinitializer, ptr %a394
  br i1 %t393, label %map.hit.78, label %map.end.79
map.hit.78:
  %t395 = call ptr @veles_map_val_at(ptr %t387, i64 %t392)
  %t396 = load i64, ptr %t395
  %t397 = insertvalue { i1, i64 } undef, i1 true, 0
  %t398 = insertvalue { i1, i64 } %t397, i64 %t396, 1
  store { i1, i64 } %t398, ptr %a394
  br label %map.end.79
map.end.79:
  %t399 = load { i1, i64 }, ptr %a394
  %t402 = extractvalue { i1, i64 } %t399, 0
  %t401 = xor i1 %t402, true
  br i1 %t401, label %elvis.default.80, label %elvis.some.81
elvis.some.81:
  %t403 = extractvalue { i1, i64 } %t399, 1
  store i64 %t403, ptr %a400
  br label %elvis.end.82
elvis.default.80:
  store i64 0, ptr %a400
  br label %elvis.end.82
elvis.end.82:
  %t404 = load i64, ptr %a400
  %t406 = call i64 @veles_i64_format(ptr %a405, i64 %t404)
  %t407 = insertvalue %str undef, ptr %a405, 0
  %t408 = insertvalue %str %t407, i64 %t406, 1
  %t409 = load ptr, ptr %a116
  %t411 = extractvalue %str { ptr @.str.5, i64 1 }, 0
  %t412 = extractvalue %str { ptr @.str.5, i64 1 }, 1
  %t410 = call i64 @veles_hash_bytes(ptr %t411, i64 %t412)
  store %str { ptr @.str.5, i64 1 }, ptr %a413
  %t414 = call i64 @veles_map_find(ptr %t409, i64 %t410, ptr %a413, ptr @eqp.string)
  %t415 = icmp sge i64 %t414, 0
  store { i1, i64 } zeroinitializer, ptr %a416
  br i1 %t415, label %map.hit.83, label %map.end.84
map.hit.83:
  %t417 = call ptr @veles_map_val_at(ptr %t409, i64 %t414)
  %t418 = load i64, ptr %t417
  %t419 = insertvalue { i1, i64 } undef, i1 true, 0
  %t420 = insertvalue { i1, i64 } %t419, i64 %t418, 1
  store { i1, i64 } %t420, ptr %a416
  br label %map.end.84
map.end.84:
  %t421 = load { i1, i64 }, ptr %a416
  %t424 = extractvalue { i1, i64 } %t421, 0
  %t423 = xor i1 %t424, true
  br i1 %t423, label %elvis.default.85, label %elvis.some.86
elvis.some.86:
  %t425 = extractvalue { i1, i64 } %t421, 1
  store i64 %t425, ptr %a422
  br label %elvis.end.87
elvis.default.85:
  store i64 0, ptr %a422
  br label %elvis.end.87
elvis.end.87:
  %t426 = load i64, ptr %a422
  %t428 = call i64 @veles_i64_format(ptr %a427, i64 %t426)
  %t429 = insertvalue %str undef, ptr %a427, 0
  %t430 = insertvalue %str %t429, i64 %t428, 1
  %t431 = load ptr, ptr %a116
  %t432 = call i64 @veles_map_len(ptr %t431)
  %t434 = call i64 @veles_i64_format(ptr %a433, i64 %t432)
  %t435 = insertvalue %str undef, ptr %a433, 0
  %t436 = insertvalue %str %t435, i64 %t434, 1
  %t437 = load ptr, ptr %a169
  %t438 = call i64 @veles_map_len(ptr %t437)
  %t440 = call i64 @veles_i64_format(ptr %a439, i64 %t438)
  %t441 = insertvalue %str undef, ptr %a439, 0
  %t442 = insertvalue %str %t441, i64 %t440, 1
  %t443 = load ptr, ptr %a169
  store i64 4, ptr %a445
  %t446 = call i64 @veles_map_find(ptr %t443, i64 4, ptr %a445, ptr @eqp.i64)
  %t447 = icmp sge i64 %t446, 0
  call void @veles_bool_to_string(ptr %a448, i1 zeroext %t447)
  %t449 = load %str, ptr %a448
  %t452 = getelementptr [9 x %str], ptr %a451, i64 0, i64 0
  store %str %t408, ptr %t452
  %t453 = getelementptr [9 x %str], ptr %a451, i64 0, i64 1
  store %str { ptr @.str.13, i64 1 }, ptr %t453
  %t454 = getelementptr [9 x %str], ptr %a451, i64 0, i64 2
  store %str %t430, ptr %t454
  %t455 = getelementptr [9 x %str], ptr %a451, i64 0, i64 3
  store %str { ptr @.str.13, i64 1 }, ptr %t455
  %t456 = getelementptr [9 x %str], ptr %a451, i64 0, i64 4
  store %str %t436, ptr %t456
  %t457 = getelementptr [9 x %str], ptr %a451, i64 0, i64 5
  store %str { ptr @.str.13, i64 1 }, ptr %t457
  %t458 = getelementptr [9 x %str], ptr %a451, i64 0, i64 6
  store %str %t442, ptr %t458
  %t459 = getelementptr [9 x %str], ptr %a451, i64 0, i64 7
  store %str { ptr @.str.13, i64 1 }, ptr %t459
  %t460 = getelementptr [9 x %str], ptr %a451, i64 0, i64 8
  store %str %t449, ptr %t460
  call void @veles_string_concat_n(ptr %a450, ptr %a451, i64 9)
  %t461 = load %str, ptr %a450
  %t462 = extractvalue %str { ptr @.str.15, i64 20 }, 0
  %t463 = extractvalue %str { ptr @.str.15, i64 20 }, 1
  call void @veles_call_push(ptr %t462)
  call void @v_std.io.println(%str %t461)
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
